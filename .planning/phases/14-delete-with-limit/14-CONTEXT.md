# Phase 14: Delete with Limit - Context

**Gathered:** 2026-03-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Add an optional `limit` parameter to collection delete operations, matching upstream Chroma PRs #6573/#6582. Both HTTP and embedded paths must support limit. No new operations or return value changes.

</domain>

<decisions>
## Implementation Decisions

### Limit option design
- **D-01:** Extend the existing `WithLimit` option by adding `ApplyToDelete(*CollectionDeleteOp) error` to `limitOption`. No new public function — `WithLimit(n)` works for Get, Search, AND Delete.
- **D-02:** Add a `Limit *int32` field to `CollectionDeleteOp` (or embed via a new `LimitOp` if cleaner).

### Embedded path handling
- **D-03:** Bump `chroma-go-local` dependency from v0.1.0 to v0.3.4. This version has `Limit *uint32` on `EmbeddedDeleteRecordsRequest` and validates limit+filter constraints. Full parity on both HTTP and embedded paths.
- **D-04:** Pass `deleteObject.Limit` through to `EmbeddedDeleteRecordsRequest.Limit` in `embeddedCollection.Delete()`.

### Validation rules
- **D-05:** Client-side validation matching upstream Chroma exactly:
  - `limit` can only be specified when a `where` or `where_document` clause is provided
  - `limit` must be greater than 0
  - Error messages: `"limit can only be specified when a where or where_document clause is provided"` and `"limit must be greater than 0"`
- **D-06:** Validation runs in `CollectionDeleteOp.PrepareAndValidate()` before any HTTP/embedded call.

### Claude's Discretion
- Whether to add `Limit` directly to `CollectionDeleteOp` or to the embedded `FilterOp`
- Whether to use `*int32` or `*uint32` for the limit field type (should align with what `limitOption` already stores)
- Test structure and specific test case names

</decisions>

<specifics>
## Specific Ideas

- Error messages must match upstream Chroma verbatim for consistency across SDKs
- The `chroma-go-local` v0.3.4 also provides `DeleteRecordsWithResponse()` that returns deleted count — NOT in scope for this phase but worth noting for future work

</specifics>

<canonical_refs>
## Canonical References

### Upstream Chroma PRs
- chroma-core/chroma#6573 — Introduced `limit` parameter in delete route (Chroma ≥ 1.5.2)
- chroma-core/chroma#6582 — Backward compatibility for `deleted` response field

### Existing codebase
- `pkg/api/v2/collection.go` — `CollectionDeleteOp` struct, `Delete` interface method
- `pkg/api/v2/options.go` — `limitOption` type, `WithLimit()` constructor, `ApplyToGet`/`ApplyToSearchRequest` implementations
- `pkg/api/v2/collection_http.go` — HTTP delete implementation
- `pkg/api/v2/client_local_embedded.go` — Embedded delete implementation, `EmbeddedDeleteRecordsRequest` construction

### External dependency
- `github.com/amikos-tech/chroma-go-local@v0.3.4` — `EmbeddedDeleteRecordsRequest.Limit`, `validateDeleteRecordsRequest()`, `DeleteRecordsWithResponse()`

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `limitOption` (options.go): Already validates limit > 0 and stores as `int32`. Adding `ApplyToDelete` is ~5 lines.
- `CollectionDeleteOp.PrepareAndValidate()` (collection.go): Existing validation hook — add limit checks here.
- `CollectionDeleteOp.MarshalJSON()` (collection.go): Uses type alias pattern, new field auto-serializes.

### Established Patterns
- Option interface pattern: each option type implements `ApplyToX(*CollectionXOp) error` for each operation it supports
- `PrepareAndValidate()` runs before HTTP/embedded calls, returns typed errors
- Sentinel errors defined as `var Err... = errors.New(...)` in the package

### Integration Points
- `limitOption.ApplyToDelete()` — new method on existing type
- `CollectionDeleteOp` — new `Limit` field
- `embeddedCollection.Delete()` — pass limit to `EmbeddedDeleteRecordsRequest`
- `go.mod` — bump `chroma-go-local` to v0.3.4

</code_context>

<deferred>
## Deferred Ideas

- Return deleted count from `Delete()` using `DeleteRecordsWithResponse()` — future phase
- Pagination/cursor-based deletion for very large filter matches — out of scope

</deferred>

---

*Phase: 14-delete-with-limit*
*Context gathered: 2026-03-29*
