# Phase 14: Delete with Limit - Research

**Researched:** 2026-03-29
**Domain:** Chroma Go SDK - Collection Delete API extension
**Confidence:** HIGH

## Summary

This phase adds an optional `limit` parameter to the collection `Delete` operation, matching upstream Chroma PRs #6573/#6582. The implementation is straightforward: extend the existing `limitOption` type with an `ApplyToDelete` method, add a `Limit` field to `CollectionDeleteOp`, wire it through HTTP serialization and the embedded path, and add client-side validation matching upstream rules.

The `chroma-go-local` dependency is already at v0.3.4, which has `Limit *uint32` on `EmbeddedDeleteRecordsRequest`. No dependency bump is needed. The `limitOption` type already stores `limit int` and validates `> 0`. The only design decision is the limit field type on `CollectionDeleteOp` -- using `*int32` (pointer for nil-ability, matching JSON `omitempty` semantics) is the right choice since the limit is optional and `0` would be ambiguous with `int`.

**Primary recommendation:** Add `Limit *int32` field directly to `CollectionDeleteOp`, add `ApplyToDelete` to `limitOption`, add validation in `PrepareAndValidate`, and convert to `*uint32` at the embedded boundary.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Extend the existing `WithLimit` option by adding `ApplyToDelete(*CollectionDeleteOp) error` to `limitOption`. No new public function -- `WithLimit(n)` works for Get, Search, AND Delete.
- **D-02:** Add a `Limit *int32` field to `CollectionDeleteOp` (or embed via a new `LimitOp` if cleaner).
- **D-03:** Bump `chroma-go-local` dependency from v0.1.0 to v0.3.4. (NOTE: Already at v0.3.4 -- no action needed.)
- **D-04:** Pass `deleteObject.Limit` through to `EmbeddedDeleteRecordsRequest.Limit` in `embeddedCollection.Delete()`.
- **D-05:** Client-side validation matching upstream Chroma exactly:
  - `limit` can only be specified when a `where` or `where_document` clause is provided
  - `limit` must be greater than 0
  - Error messages: `"limit can only be specified when a where or where_document clause is provided"` and `"limit must be greater than 0"`
- **D-06:** Validation runs in `CollectionDeleteOp.PrepareAndValidate()` before any HTTP/embedded call.

### Claude's Discretion
- Whether to add `Limit` directly to `CollectionDeleteOp` or to the embedded `FilterOp`
- Whether to use `*int32` or `*uint32` for the limit field type (should align with what `limitOption` already stores)
- Test structure and specific test case names

### Deferred Ideas (OUT OF SCOPE)
- Return deleted count from `Delete()` using `DeleteRecordsWithResponse()` -- future phase
- Pagination/cursor-based deletion for very large filter matches -- out of scope
</user_constraints>

## Project Constraints (from CLAUDE.md)

- New features target V2 API (`/pkg/api/v2/`)
- Tests use `testify` for assertions
- Tests use build tags (`basicv2` for V2 tests)
- Integration tests use testcontainers
- Run `make lint` before committing
- Never panic in production code
- Keep things radically simple
- Use conventional commits

## Architecture Patterns

### Existing Option Pattern (to follow exactly)

The `limitOption` type at `pkg/api/v2/options.go:521-574` stores `limit int` and implements `ApplyToGet` and `ApplyToSearchRequest`. Adding `ApplyToDelete` follows the identical pattern:

```go
func (o *limitOption) ApplyToDelete(op *CollectionDeleteOp) error {
    if o.limit <= 0 {
        return ErrInvalidLimit
    }
    limit := int32(o.limit)
    op.Limit = &limit
    return nil
}
```

### CollectionDeleteOp Structure

Current struct at `collection.go:1095-1098`:
```go
type CollectionDeleteOp struct {
    FilterOp   // Where and WhereDocument filters
    FilterIDOp // ID filter
}
```

The `Limit` field should be added directly to `CollectionDeleteOp` (not to `FilterOp`) because:
1. `FilterOp` is shared across Get, Query, and Delete -- adding Limit there would expose it to Query which uses `NResults` instead
2. Direct field is simpler than creating a new embedded struct for a single field
3. Matches the pattern: `CollectionGetOp` embeds `LimitAndOffsetOp` but Delete only needs limit (no offset)

Recommended change:
```go
type CollectionDeleteOp struct {
    FilterOp   // Where and WhereDocument filters
    FilterIDOp // ID filter
    Limit      *int32 `json:"limit,omitempty"`
}
```

### Field Type Decision: `*int32`

Use `*int32` because:
- `limitOption` stores `int` and validates `> 0`; `int32` is the natural narrowing (matches existing `int32` casts elsewhere)
- Pointer enables `nil` vs zero distinction -- `nil` means "no limit specified", which serializes correctly with `omitempty`
- `EmbeddedDeleteRecordsRequest.Limit` is `*uint32` -- conversion at the boundary is a simple cast since validation guarantees `> 0`

### Validation in PrepareAndValidate

Add to `CollectionDeleteOp.PrepareAndValidate()`:
```go
if c.Limit != nil {
    if *c.Limit <= 0 {
        return errors.New("limit must be greater than 0")
    }
    if c.Where == nil && c.WhereDocument == nil {
        return errors.New("limit can only be specified when a where or where_document clause is provided")
    }
}
```

Note: The `> 0` check in `PrepareAndValidate` is defense-in-depth since `ApplyToDelete` already validates via `ErrInvalidLimit`. The filter requirement check is the key new validation.

### Embedded Path Wiring

In `client_local_embedded.go:994-1001`, add `Limit` to the request:
```go
var limit *uint32
if deleteObject.Limit != nil {
    l := uint32(*deleteObject.Limit)
    limit = &l
}
return c.client.embedded.DeleteRecords(localchroma.EmbeddedDeleteRecordsRequest{
    // ... existing fields ...
    Limit: limit,
})
```

### HTTP Path

No changes needed to `collection_http.go` -- the `deleteObject` is already serialized as the request body via `ExecuteRequest`. Adding `Limit *int32 json:"limit,omitempty"` to the struct means it auto-serializes when set and is omitted when nil.

### Options Table Update

The comment table in `options.go` line 25 shows `WithLimit` working with Get and Search. Update to include Delete:
```
WithLimit           |  ✓  |       |   ✓    |     |        |   ✓
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Limit option type | New `deleteLimitOption` | Extend existing `limitOption` with `ApplyToDelete` | Reuses established validation and API surface |
| Limit validation | Custom validation logic | `ErrInvalidLimit` sentinel + filter check in `PrepareAndValidate` | Consistent with existing error patterns |

## Common Pitfalls

### Pitfall 1: JSON serialization of zero limit
**What goes wrong:** Using `int` instead of `*int32` causes `limit: 0` to appear in JSON when not specified (since Go zero-values serialize even with `omitempty` for non-pointer types... actually `int` with `omitempty` omits zero, but semantically `0` and "not set" should be distinguishable).
**How to avoid:** Use `*int32` with `json:"limit,omitempty"`. Nil pointer omits the field entirely.

### Pitfall 2: int32 to uint32 conversion
**What goes wrong:** Negative values passed to `uint32()` wrap around to large positive numbers.
**How to avoid:** Validation in `ApplyToDelete` ensures `limit > 0` before the value reaches the embedded path. The `PrepareAndValidate` check is defense-in-depth.

### Pitfall 3: Limit without filter
**What goes wrong:** Upstream Chroma rejects delete with limit but no where/where_document clause. If client doesn't validate, error surfaces as an opaque server error.
**How to avoid:** Client-side validation in `PrepareAndValidate` catches this with a clear error message before any network call.

## Code Examples

### Usage Pattern (what users will write)
```go
// Delete at most 100 documents matching a metadata filter
err := collection.Delete(ctx,
    WithWhere(EqString("status", "archived")),
    WithLimit(100),
)

// Delete at most 50 documents matching a document content filter
err := collection.Delete(ctx,
    WithWhereDocument(Contains("DRAFT:")),
    WithLimit(50),
)
```

### Error Cases
```go
// Error: limit without filter
err := collection.Delete(ctx, WithLimit(100))
// => "limit can only be specified when a where or where_document clause is provided"

// Error: limit <= 0
err := collection.Delete(ctx,
    WithWhere(EqString("status", "old")),
    WithLimit(0),
)
// => "limit must be greater than 0"
```

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify |
| Config file | Makefile (build tags) |
| Quick run command | `go test -tags=basicv2 -run TestDelete ./pkg/api/v2/...` |
| Full suite command | `make test` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| D-01 | `WithLimit(n)` applies to Delete op | unit | `go test -tags=basicv2 -run TestLimitOption.*Delete ./pkg/api/v2/...` | Wave 0 |
| D-05a | limit rejected without where/where_document | unit | `go test -tags=basicv2 -run TestDeleteLimitWithoutFilter ./pkg/api/v2/...` | Wave 0 |
| D-05b | limit must be > 0 | unit | `go test -tags=basicv2 -run TestDeleteLimitZero ./pkg/api/v2/...` | Wave 0 |
| D-02 | Limit field serializes in JSON | unit | `go test -tags=basicv2 -run TestCollectionDelete.*limit ./pkg/api/v2/...` | Wave 0 |
| D-04 | Embedded path passes limit through | unit | existing embedded test pattern | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -tags=basicv2 -run TestDelete ./pkg/api/v2/... && make lint`
- **Per wave merge:** `make test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- None -- existing test infrastructure (`collection_http_test.go`, `options_test.go`) covers the patterns needed. New test cases follow established table-driven patterns.

## Files to Modify

| File | Change | Scope |
|------|--------|-------|
| `pkg/api/v2/collection.go` | Add `Limit *int32` field to `CollectionDeleteOp`, update `PrepareAndValidate` | 10 lines |
| `pkg/api/v2/options.go` | Add `ApplyToDelete` method to `limitOption`, update option matrix comment | 15 lines |
| `pkg/api/v2/client_local_embedded.go` | Pass `Limit` to `EmbeddedDeleteRecordsRequest` | 6 lines |
| `pkg/api/v2/options_test.go` | Add delete+limit unit tests | 30 lines |
| `pkg/api/v2/collection_http_test.go` | Add HTTP serialization test for delete with limit | 25 lines |

**Total estimated change:** ~85 lines across 5 files.

## Sources

### Primary (HIGH confidence)
- Direct code inspection of `pkg/api/v2/collection.go`, `options.go`, `collection_http.go`, `client_local_embedded.go`
- `chroma-go-local@v0.3.4` module cache -- `EmbeddedDeleteRecordsRequest` struct verified
- Upstream Chroma PR #6573 -- confirmed `limit: Option<u32>` in delete route

### Secondary (MEDIUM confidence)
- go.mod confirms `chroma-go-local` already at v0.3.4 -- D-03 bump is a no-op

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - no new dependencies, all existing patterns
- Architecture: HIGH - direct code inspection of all integration points
- Pitfalls: HIGH - straightforward change with well-understood edge cases

**Research date:** 2026-03-29
**Valid until:** 2026-04-28 (stable codebase, no upstream churn expected)
