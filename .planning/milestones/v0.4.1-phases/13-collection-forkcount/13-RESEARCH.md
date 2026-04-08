# Phase 13: Collection.ForkCount - Research

**Researched:** 2026-03-28
**Domain:** Go SDK method addition (HTTP transport, interface extension)
**Confidence:** HIGH

## Summary

Phase 13 adds a single new method `ForkCount(ctx context.Context) (int, error)` to the V2 Collection interface. This is a straightforward addition that follows well-established patterns already in the codebase: `Count()` (GET + parse response), `IndexingStatus()` (GET + JSON decode), and `Fork()` (URL composition pattern). The upstream Chroma endpoint is `GET /api/v2/tenants/{tenant}/databases/{database}/collections/{collection_id}/fork_count` returning `{"count": n}`.

The implementation touches exactly 5 files: the interface definition, HTTP implementation, embedded stub, tests, and documentation. Every pattern needed already exists in the codebase -- no new dependencies, no architectural decisions, no research gaps.

**Primary recommendation:** Follow the `IndexingStatus()` pattern exactly (GET + JSON unmarshal) with the `Fork()` URL composition pattern. This is the simplest phase in the project.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Decode the upstream `{"count": n}` JSON response using a strict struct (`struct{ Count int }` with `json:"count"` tag) and `json.Unmarshal`. No flexible map decoding.
- **D-02:** Return `int` (not `int32`), matching the existing `Count(ctx) (int, error)` signature for interface consistency.
- **D-03:** Embedded client returns `errors.New("fork count is not supported in embedded local mode")` -- same pattern as `Fork()`, `Search()`, and other unsupported embedded operations.
- **D-04:** HTTP errors follow existing Fork/Count patterns: `errors.Wrap(err, "error getting fork count")` for URL composition and request failures.
- **D-05:** Add godoc comment on the `ForkCount` interface method and both implementations.
- **D-06:** Update existing forking docs at `docs/go-examples/cloud/features/collection-forking.md` with a ForkCount section.
- **D-07:** Add a Fork + ForkCount example under `examples/v2/` (or extend existing forking example if appropriate).

### Claude's Discretion
- HTTP method (GET expected, but verify against upstream) -- **Verified: GET** (confirmed from issue #460 upstream references)
- Response struct naming and placement (inline anonymous struct vs named type)
- Test structure and naming conventions
- Example structure and naming

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

## Architecture Patterns

### Implementation Template

ForkCount follows a pattern identical to `IndexingStatus()` in `collection_http.go` (lines 675-689). The only differences are the URL path segment and the response struct shape.

### Pattern: GET + JSON Decode (IndexingStatus precedent)
**What:** Issue GET request to collection sub-endpoint, decode JSON response body
**When to use:** Any collection-level read-only endpoint returning structured JSON
**Example:**
```go
// Source: pkg/api/v2/collection_http.go:675-689 (IndexingStatus pattern)
func (c *CollectionImpl) ForkCount(ctx context.Context) (int, error) {
    reqURL, err := url.JoinPath("tenants", c.Tenant().Name(), "databases", c.Database().Name(), "collections", c.ID(), "fork_count")
    if err != nil {
        return 0, errors.Wrap(err, "error composing request URL")
    }
    respBody, err := c.client.ExecuteRequest(ctx, http.MethodGet, reqURL, nil)
    if err != nil {
        return 0, errors.Wrap(err, "error getting fork count")
    }
    var result struct {
        Count int `json:"count"`
    }
    if err := json.Unmarshal(respBody, &result); err != nil {
        return 0, errors.Wrap(err, "error decoding fork count response")
    }
    return result.Count, nil
}
```

### Pattern: Embedded Unsupported Stub
**What:** Return explicit error for operations not available in embedded/local mode
**When to use:** Cloud-only features
**Example:**
```go
// Source: pkg/api/v2/client_local_embedded.go:1365-1368 (Fork pattern)
func (c *embeddedCollection) ForkCount(_ context.Context) (int, error) {
    return 0, errors.New("fork count is not supported in embedded local mode")
}
```

### Pattern: HTTP Unit Test (httptest server)
**What:** Stand up httptest.NewServer, match URL regex, return canned response, verify decoded result
**When to use:** All collection HTTP method tests
**Example:**
```go
// Source: pkg/api/v2/collection_http_test.go:519-548 (IndexingStatus test pattern)
func TestCollectionForkCount(t *testing.T) {
    rx := regexp.MustCompile(`/api/v2/tenants/[^/]+/databases/[^/]+/collections/[^/]+/fork_count`)
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch {
        case r.Method == http.MethodGet && rx.MatchString(r.URL.Path):
            w.WriteHeader(http.StatusOK)
            _, _ = w.Write([]byte(`{"count":5}`))
        default:
            w.WriteHeader(http.StatusNotFound)
        }
    }))
    defer server.Close()
    client, err := NewHTTPClient(WithBaseURL(server.URL))
    // ... construct CollectionImpl, call ForkCount, assert result == 5
}
```

### Pattern: Embedded Unsupported Test
**What:** Construct zero-value embeddedCollection, call method, assert error message
**Source:** `pkg/api/v2/client_local_test.go:620-625`
```go
func TestEmbeddedCollection_ForkCountNotSupported(t *testing.T) {
    col := &embeddedCollection{}
    _, err := col.ForkCount(nil)
    require.Error(t, err)
    require.Contains(t, err.Error(), "not supported in embedded local mode")
}
```

### Files to Modify

| File | Change | Lines of Code |
|------|--------|---------------|
| `pkg/api/v2/collection.go` | Add `ForkCount` to `Collection` interface (after `Fork`, before `IndexingStatus`) | ~3 lines |
| `pkg/api/v2/collection_http.go` | Add `CollectionImpl.ForkCount()` HTTP implementation | ~15 lines |
| `pkg/api/v2/client_local_embedded.go` | Add `embeddedCollection.ForkCount()` unsupported stub | ~4 lines |
| `pkg/api/v2/collection_http_test.go` | Add happy path + failure path HTTP tests | ~40 lines |
| `pkg/api/v2/client_local_test.go` | Add embedded unsupported test | ~6 lines |
| `docs/go-examples/cloud/features/collection-forking.md` | Add ForkCount section to docs | ~30 lines |
| `examples/v2/` (new directory or file) | Add Fork + ForkCount example | ~40 lines |

### Anti-Patterns to Avoid
- **Using map[string]interface{} for JSON decode:** D-01 explicitly locks strict struct decode. Use `struct{ Count int \`json:"count"\` }`.
- **Returning int32/int64:** D-02 locks return type to `int` for consistency with `Count()`.
- **Named top-level type for trivial response:** An anonymous struct inline is sufficient for a single-field response. No need for a `ForkCountResponse` type unless reused.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTTP test server | Custom TCP listener | `net/http/httptest` | Already used everywhere in codebase |
| JSON response decode | Manual string parsing | `encoding/json` with struct tags | D-01 requirement |
| URL composition | String concatenation | `url.JoinPath` | Established codebase pattern, handles escaping |

## Common Pitfalls

### Pitfall 1: Forgetting pre-flight-checks in test server
**What goes wrong:** Some HTTP tests include a pre-flight-checks handler, others don't. The `IndexingStatus` test at line 519 does NOT include it but the `Count` test at line 482 does.
**How to avoid:** Check whether `NewHTTPClient` issues pre-flight on construction. If using `WithBaseURL` only (no logger), pre-flight may not trigger. Follow the `IndexingStatus` test pattern which is simpler and doesn't include pre-flight.

### Pitfall 2: Wrong HTTP method
**What goes wrong:** Using POST instead of GET for fork_count.
**How to avoid:** Upstream issue #460 confirms GET. The `Count()` and `IndexingStatus()` precedents also use GET. Fork (POST) is different because it creates a resource.

### Pitfall 3: Missing interface implementation
**What goes wrong:** Adding method to interface but forgetting one implementation causes compile error.
**How to avoid:** Both `CollectionImpl` (HTTP) and `embeddedCollection` (local) must implement the new method. The compiler will catch this, but plan both together.

## Upstream API Verification

From issue #460 (HIGH confidence -- detailed upstream references):

| Property | Value | Source |
|----------|-------|--------|
| HTTP method | GET | Issue #460, upstream commit `6c40da74b` |
| URL path | `/api/v2/tenants/{t}/databases/{d}/collections/{id}/fork_count` | Issue #460 |
| Request body | None (path params only) | Issue #460 |
| Response shape | `{"count": <non-negative integer>}` | Issue #460 |
| Semantics | Lineage-wide count (not just direct children) | Issue #460 upstream tests |
| Local support | Unsupported (Python raises NotImplementedError, Rust returns error) | Issue #460 |
| Introduced | 2026-03-17 upstream commit | Issue #460 |

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify |
| Config file | Makefile (build tags) |
| Quick run command | `go test -tags=basicv2 -run TestCollectionForkCount ./pkg/api/v2/...` |
| Full suite command | `make test` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SC-1 | Interface includes ForkCount | compile | `go build -tags=basicv2 ./pkg/api/v2/...` | N/A (compiler) |
| SC-2 | HTTP GET /fork_count decodes {"count": n} | unit | `go test -tags=basicv2 -run TestCollectionForkCount ./pkg/api/v2/...` | Wave 0 |
| SC-3 | Embedded returns unsupported error | unit | `go test -tags=basicv2 -run TestEmbeddedCollection_ForkCountNotSupported ./pkg/api/v2/...` | Wave 0 |
| SC-4 | HTTP failure path | unit | `go test -tags=basicv2 -run TestCollectionForkCount ./pkg/api/v2/...` | Wave 0 |
| SC-5 | Docs updated | manual | Visual review | N/A |

### Sampling Rate
- **Per task commit:** `go test -tags=basicv2 -run "ForkCount" ./pkg/api/v2/...`
- **Per wave merge:** `make test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
None -- test infrastructure (testify, httptest, build tags) fully exists. Tests are new files/additions to existing test files.

## Code Examples

### Interface Addition (collection.go)
```go
// Source: follows Fork() at line 209 and IndexingStatus() at line 212
// ForkCount returns the total number of forks in this collection's lineage.
// This count is lineage-wide: source and all forked descendants report the
// same value. Requires Chroma Cloud; embedded local mode returns an error.
ForkCount(ctx context.Context) (int, error)
```

### HTTP Implementation (collection_http.go)
```go
// Source: follows IndexingStatus pattern at line 675
func (c *CollectionImpl) ForkCount(ctx context.Context) (int, error) {
    reqURL, err := url.JoinPath("tenants", c.Tenant().Name(), "databases", c.Database().Name(), "collections", c.ID(), "fork_count")
    if err != nil {
        return 0, errors.Wrap(err, "error composing request URL")
    }
    respBody, err := c.client.ExecuteRequest(ctx, http.MethodGet, reqURL, nil)
    if err != nil {
        return 0, errors.Wrap(err, "error getting fork count")
    }
    var result struct {
        Count int `json:"count"`
    }
    if err := json.Unmarshal(respBody, &result); err != nil {
        return 0, errors.Wrap(err, "error decoding fork count response")
    }
    return result.Count, nil
}
```

### Embedded Stub (client_local_embedded.go)
```go
// Source: follows Fork pattern at line 1365
// ForkCount is not supported in embedded local mode.
func (c *embeddedCollection) ForkCount(_ context.Context) (int, error) {
    return 0, errors.New("fork count is not supported in embedded local mode")
}
```

## Documentation Update Pattern

The existing forking docs at `docs/go-examples/cloud/features/collection-forking.md` use `{% codetabs %}` blocks with Python and Go tabs. The ForkCount section should follow the same pattern. The API Reference table at line 410 should be extended with a ForkCount row.

The example under `examples/v2/` should demonstrate Fork followed by ForkCount, showing lineage-wide count semantics.

## Project Constraints (from CLAUDE.md)

- Use conventional commits
- Run `make lint` before committing
- New features target V2 API (`/pkg/api/v2/`)
- Use `testify` for assertions
- Never panic in production code (not relevant here -- no panic risk)
- Keep things radically simple
- Do not leave too many or verbose comments
- All API methods accept context for cancellation/timeout

## Sources

### Primary (HIGH confidence)
- GitHub Issue #460 -- complete upstream API specification with commit references
- `pkg/api/v2/collection_http.go:675-689` -- IndexingStatus pattern (identical structure)
- `pkg/api/v2/collection_http.go:276-286` -- Count pattern (GET + response parse)
- `pkg/api/v2/collection_http.go:394-427` -- Fork pattern (URL composition)
- `pkg/api/v2/client_local_embedded.go:1365-1368` -- Embedded unsupported pattern
- `pkg/api/v2/collection_http_test.go:482-548` -- Count and IndexingStatus test patterns
- `pkg/api/v2/client_local_test.go:620-625` -- Embedded unsupported test pattern

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies, pure Go stdlib + existing patterns
- Architecture: HIGH -- exact precedent exists (IndexingStatus is structurally identical)
- Pitfalls: HIGH -- minimal complexity, well-trodden patterns

**Research date:** 2026-03-28
**Valid until:** 2026-04-28 (stable -- upstream endpoint unlikely to change)
