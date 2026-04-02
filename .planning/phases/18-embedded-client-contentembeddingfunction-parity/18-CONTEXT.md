# Phase 18: Embedded Client contentEmbeddingFunction Parity - Context

**Gathered:** 2026-04-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Add `contentEmbeddingFunction` support to `embeddedCollection` so the embedded client has feature parity with the HTTP client for content embedding lifecycle, auto-wiring, and Close handling. Issue #472.

</domain>

<decisions>
## Implementation Decisions

### Fork stub handling
- **D-01:** Skip contentEF propagation in Fork(). Fork returns an unsupported error in embedded mode — adding contentEF close-once wrapping to the stub would be dead code. If Fork ever becomes supported, contentEF wiring would be part of that feature work.

### Auto-wiring scope
- **D-02:** Full auto-wiring via `BuildContentEFFromConfig` in embedded `GetCollection`, matching HTTP behavior. When no explicit `WithContentEmbeddingFunctionGet` is passed, attempt to build contentEF from stored collection configuration. This ensures users get the same behavior regardless of client type.

### Close() sharing detection
- **D-03:** Full mirror of HTTP sharing detection in `embeddedCollection.Close()`. Close contentEF first, then check if denseEF shares the same resource (via `EmbeddingFunctionUnwrapper` unwrapper check + identity check) before closing denseEF. This prevents double-close when auto-wiring produces a contentEF that wraps the denseEF (adapter case from Phase 3).

### Claude's Discretion
- `embeddedCollectionState` struct field naming and initialization details
- Test structure and file organization
- Whether to add `contentEmbeddingFunction` to `CreateCollectionOp` or keep it GetCollection-only (matching current HTTP behavior)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Issue
- GitHub Issue #472 — Gap analysis and implementation checklist

### HTTP reference implementation
- `pkg/api/v2/collection_http.go:51-66` — CollectionImpl struct with contentEmbeddingFunction field
- `pkg/api/v2/collection_http.go:412-427` — Fork() contentEF close-once wrapping (reference pattern, not needed for embedded)
- `pkg/api/v2/collection_http.go:709-746` — Close() with full sharing detection logic to mirror
- `pkg/api/v2/client_http.go:421-462` — GetCollection auto-wiring via BuildContentEFFromConfig

### Embedded client (files to modify)
- `pkg/api/v2/client_local_embedded.go:21-27` — embeddedCollectionState struct (add contentEF field)
- `pkg/api/v2/client_local_embedded.go:697-774` — buildEmbeddedCollection (wire contentEF)
- `pkg/api/v2/client_local_embedded.go:776-793` — embeddedCollection struct (add contentEF field)
- `pkg/api/v2/client_local_embedded.go:495-527` — GetCollection (add auto-wiring + explicit option)
- `pkg/api/v2/client_local_embedded.go:1405-1423` — Close() (add sharing detection)

### Close-once infrastructure (reuse)
- `pkg/api/v2/ef_close_once.go` — wrapContentEFCloseOnce, closeOnceContentEF, unwrapCloseOnceContentEF, unwrapCloseOnceEF
- `pkg/api/v2/ef_close_once_test.go` — existing tests for close-once wrappers

### Get option
- `pkg/api/v2/client.go:162-207` — GetCollectionOp with contentEmbeddingFunction field and WithContentEmbeddingFunctionGet option

### Auto-wiring
- `pkg/api/v2/configuration.go` — BuildContentEFFromConfig function

### Prior phase context
- `.planning/phases/11-fork-double-close-bug/11-CONTEXT.md` — ownsEF flag pattern, close-once wrapper decisions

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `wrapContentEFCloseOnce()` / `closeOnceContentEF`: Ready-to-use close-once wrapper for contentEF (ef_close_once.go)
- `wrapEFCloseOnce()` / `unwrapCloseOnceEF()` / `unwrapCloseOnceContentEF()`: Unwrap helpers for sharing detection
- `BuildContentEFFromConfig()`: Auto-wiring function that builds contentEF from collection configuration
- `safeCloseEF()` / `reportClosePanic()`: Safe close helpers already used in HTTP path
- `EmbeddingFunctionUnwrapper`: Interface for detecting when contentEF wraps denseEF

### Established Patterns
- `ownsEF` atomic.Bool flag gates Close() — already implemented in embeddedCollection for dense EF
- `closeOnce` sync.Once already in embeddedCollection struct — needs to encompass both EFs
- `embeddedCollectionState` centralizes mutable collection state — contentEF should be stored here
- `upsertCollectionState` callback pattern for updating state — add contentEF updates

### Integration Points
- `buildEmbeddedCollection`: Add contentEF parameter and wire to struct
- `embeddedCollectionState`: Add contentEmbeddingFunction field
- `GetCollection`: Add auto-wiring fallback and explicit option wiring
- `GetOrCreateCollection`: Propagate contentEF through GetCollection fallback
- `Close()`: Expand to close both EFs with sharing detection

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches matching HTTP client patterns.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 18-embedded-client-contentembeddingfunction-parity*
*Context gathered: 2026-04-02*
