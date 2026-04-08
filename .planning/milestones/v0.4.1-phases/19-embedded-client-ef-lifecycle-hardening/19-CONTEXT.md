# Phase 19: Embedded Client EF Lifecycle Hardening - Context

**Gathered:** 2026-04-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Fix all embedded client EF robustness gaps: TOCTOU race in GetCollection auto-wiring, state map cleanup on delete and close, close-once wrapping in buildEmbeddedCollection, symmetric unwrapping in isDenseEFSharedWithContent, guard auto-wired EF assignment against build errors, and add structured logger for observability parity. Issues: #484, #485, #488, #489.

</domain>

<decisions>
## Implementation Decisions

### TOCTOU race prevention
- **D-01:** GetCollection auto-wiring uses check-and-set under a full write lock (Lock(), not RLock()). The lock spans the entire check-nil + build + assign cycle (wide lock).
- **D-02:** Wide lock is acceptable because concurrent GetCollection calls for the same collection are not a real-world scenario — every collection operation requires a prior GetCollection call, making usage inherently sequential.

### Close-once wrapping
- **D-03:** buildEmbeddedCollection wraps both denseEF and contentEF in close-once wrappers (wrapEFCloseOnce / wrapContentEFCloseOnce), mirroring the HTTP client pattern in collection_http.go.

### Structured logger
- **D-04:** Add a WithLogger option to PersistentClient that accepts the existing pkg/logger interface. When set, auto-wire and close errors route through the injected logger (structured). When unset, fall back to stderr (current behavior).

### Cleanup on delete and close
- **D-05:** embeddedLocalClient.Close() iterates all collectionState entries and closes their EFs before clearing the map, mirroring HTTP client collection cache cleanup.
- **D-06:** deleteCollectionState closes EFs first (with sharing detection), then removes the map entry.
- **D-07:** localDeleteCollectionFromCache adds a type switch case for *embeddedCollection to close EFs via the same sharing detection logic before removing from cache.

### Symmetric unwrapping
- **D-08:** isDenseEFSharedWithContent unwraps both denseEF and contentEF before comparing. With close-once wrappers on both sides in the embedded path, symmetric unwrapping ensures the identity check works regardless of wrapping depth.

### Build error guard
- **D-09:** Auto-wired EF is only assigned when the build error is nil. A failed build means no EF — the collection proceeds without one. The error is logged for observability.

### Claude's Discretion
- Internal helper decomposition and method ordering
- Exact log message format and severity levels
- Whether to refactor shared cleanup logic into a common helper or keep it inline per type

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Issues
- GitHub Issue #484 — TOCTOU race in GetCollection auto-wiring
- GitHub Issue #485 — State map cleanup on delete and close
- GitHub Issue #488 — Close-once wrapping and symmetric unwrapping
- GitHub Issue #489 — Build error guard and structured logger

### HTTP reference implementation (pattern to mirror)
- `pkg/api/v2/collection_http.go:51-66` — CollectionImpl struct with close-once wrapped EFs
- `pkg/api/v2/collection_http.go:709-746` — Close() with full sharing detection logic
- `pkg/api/v2/client_http.go:421-462` — GetCollection auto-wiring via BuildContentEFFromConfig

### Embedded client (files to modify)
- `pkg/api/v2/client_local_embedded.go:21-27` — embeddedCollectionState struct
- `pkg/api/v2/client_local_embedded.go:495-527` — GetCollection (TOCTOU fix + error guard)
- `pkg/api/v2/client_local_embedded.go:648-665` — embeddedLocalClient.Close() (iterate collectionState)
- `pkg/api/v2/client_local_embedded.go:681-688` — deleteCollectionState (close EFs before delete)
- `pkg/api/v2/client_local_embedded.go:787-822` — buildEmbeddedCollection (add close-once wrapping)

### Close-once infrastructure (reuse)
- `pkg/api/v2/ef_close_once.go` — wrapEFCloseOnce, wrapContentEFCloseOnce, unwrap helpers
- `pkg/api/v2/close_logging.go` — isDenseEFSharedWithContent (symmetric unwrapping fix), stderr logging functions

### Logger infrastructure
- `pkg/logger/` — existing logger abstraction to inject

### Prior phase context
- `.planning/phases/18-embedded-client-contentembeddingfunction-parity/18-CONTEXT.md` — contentEF wiring decisions
- `.planning/phases/11-fork-double-close-bug/11-CONTEXT.md` — ownsEF flag pattern, close-once wrapper decisions

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `wrapEFCloseOnce()` / `wrapContentEFCloseOnce()`: Close-once wrappers ready to use in buildEmbeddedCollection
- `unwrapCloseOnceEF()` / `unwrapCloseOnceContentEF()`: Unwrap helpers for sharing detection
- `isDenseEFSharedWithContent()`: Sharing detection function (needs symmetric unwrapping fix)
- `BuildContentEFFromConfig()` / `BuildEmbeddingFunctionFromConfig()`: Auto-wiring builders
- `safeCloseEF()` / `reportClosePanic()`: Safe close helpers already used in HTTP path
- `pkg/logger`: Logger abstraction with Zap bridge support

### Established Patterns
- `ownsEF` atomic.Bool flag gates Close() — already in embeddedCollection for dense EF
- `closeOnce` sync.Once in embeddedCollection — needs to encompass both EFs
- `embeddedCollectionState` centralizes mutable collection state with mutex protection
- `collectionStateMu` sync.RWMutex guards collectionState map access
- HTTP client wraps EFs in close-once at collection build time — pattern to mirror

### Integration Points
- `buildEmbeddedCollection`: Add close-once wrapping for both EFs
- `GetCollection`: Upgrade to write lock for auto-wiring, add error guard
- `embeddedLocalClient.Close()`: Add collectionState iteration and EF cleanup
- `deleteCollectionState`: Add EF close before map entry removal
- `localDeleteCollectionFromCache`: Add *embeddedCollection type switch case
- `isDenseEFSharedWithContent`: Add contentEF unwrapping for symmetry
- `PersistentClient` options: Add WithLogger option

</code_context>

<specifics>
## Specific Ideas

No specific requirements — mirror HTTP client patterns throughout.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 19-embedded-client-ef-lifecycle-hardening*
*Context gathered: 2026-04-06*
