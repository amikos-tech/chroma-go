# Phase 13: Collection.ForkCount - Context

**Gathered:** 2026-03-28
**Status:** Ready for planning

<domain>
## Phase Boundary

Add `ForkCount(ctx context.Context) (int, error)` to the V2 Collection interface with HTTP transport support, matching upstream Chroma's `/fork_count` endpoint. Embedded client returns explicit unsupported error. Issue #460.

</domain>

<decisions>
## Implementation Decisions

### Response shape
- **D-01:** Decode the upstream `{"count": n}` JSON response using a strict struct (`struct{ Count int }` with `json:"count"` tag) and `json.Unmarshal`. No flexible map decoding.
- **D-02:** Return `int` (not `int32`), matching the existing `Count(ctx) (int, error)` signature for interface consistency.

### Error semantics
- **D-03:** Embedded client returns `errors.New("fork count is not supported in embedded local mode")` — same pattern as `Fork()`, `Search()`, and other unsupported embedded operations.
- **D-04:** HTTP errors follow existing Fork/Count patterns: `errors.Wrap(err, "error getting fork count")` for URL composition and request failures.

### Documentation
- **D-05:** Add godoc comment on the `ForkCount` interface method and both implementations.
- **D-06:** Update existing forking docs at `docs/go-examples/cloud/features/collection-forking.md` with a ForkCount section.
- **D-07:** Add a Fork + ForkCount example under `examples/v2/` (or extend existing forking example if appropriate).

### Claude's Discretion
- HTTP method (GET expected, but verify against upstream)
- Response struct naming and placement (inline anonymous struct vs named type)
- Test structure and naming conventions
- Example structure and naming

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Issue
- GitHub Issue #460 — ForkCount endpoint request

### Existing Fork implementation (pattern to follow)
- `pkg/api/v2/collection.go` line ~209 — `Fork()` interface definition
- `pkg/api/v2/collection.go` line ~141 — `Count()` interface definition (return type precedent)
- `pkg/api/v2/collection_http.go` line ~394 — `CollectionImpl.Fork()` HTTP implementation (URL pattern)
- `pkg/api/v2/collection_http.go` line ~276 — `CollectionImpl.Count()` HTTP implementation (simple GET precedent)
- `pkg/api/v2/client_local_embedded.go` line ~1366 — `embeddedCollection.Fork()` unsupported error pattern

### Documentation
- `docs/go-examples/cloud/features/collection-forking.md` — Existing forking docs to update

### Examples directory
- `examples/v2/` — Location for new ForkCount example

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `CollectionImpl.Count()`: Nearly identical pattern — GET request, decode response, return int. ForkCount is the same shape but with JSON decode instead of `strconv.Atoi`.
- `CollectionImpl.Fork()`: URL composition pattern (`tenants/{t}/databases/{d}/collections/{id}/fork`) — ForkCount appends `fork_count` instead.

### Established Patterns
- URL composition via `url.JoinPath("tenants", t, "databases", d, "collections", id, "fork_count")`
- Error wrapping with `errors.Wrap(err, "error ...")`
- Embedded unsupported ops return `errors.New("... is not supported in embedded local mode")`
- Interface method with `(ctx context.Context)` parameter and `(int, error)` return

### Integration Points
- `pkg/api/v2/collection.go` — Add `ForkCount` to Collection interface
- `pkg/api/v2/collection_http.go` — Add HTTP implementation
- `pkg/api/v2/client_local_embedded.go` — Add embedded unsupported stub
- `docs/go-examples/cloud/features/collection-forking.md` — Update docs
- `examples/v2/` — New example

</code_context>

<specifics>
## Specific Ideas

- User wants all three documentation touchpoints: godoc, forking docs page, and a runnable example
- Follow existing Count() and Fork() as structural templates

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 13-collection-forkcount*
*Context gathered: 2026-03-28*
