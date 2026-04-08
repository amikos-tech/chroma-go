# Phase 20: GetOrCreateCollection contentEF Support - Context

**Gathered:** 2026-04-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Add contentEmbeddingFunction support to GetOrCreateCollection and CreateCollection by extending CreateCollectionOp with a contentEF field, adding WithContentEmbeddingFunctionCreate option, wiring contentEF through both HTTP and embedded client paths, and persisting contentEF config for future auto-wiring. Issue #486.

</domain>

<decisions>
## Implementation Decisions

### Auto-wiring behavior
- **D-01:** HTTP GetOrCreateCollection does NOT auto-wire contentEF from server config for existing collections. ContentEF comes only from the user-provided option (or nil if not given). This matches Python and JS SDK behavior — neither SDK auto-wires in getOrCreateCollection. Only GetCollection auto-wires from server config.

### Embedded GetOrCreateCollection forwarding
- **D-02:** Forward contentEF from CreateCollectionOp to GetCollection via WithContentEmbeddingFunctionGet, same pattern as existing denseEF forwarding. Keeps the embedded path internally consistent — the embedded GetOrCreateCollection already tries GetCollection first (which auto-wires), then falls back to CreateCollection.

### Embedded CreateCollection state storage for existing collections
- **D-03:** When embedded CreateCollection handles an existing collection (isNewCreation=false), ignore the user-provided contentEF and use existing state — same pattern as denseEF. Set overrideContentEF=nil for existing collections.

### Validation and conflict detection
- **D-04:** No EF conflict validation in this phase. The Go client currently has zero EF conflict detection across all operations. Adding it only for contentEF would be inconsistent. A separate GH issue should be created to track full conflict detection across all collection operations (parity with Python's validate_embedding_function_conflict_on_get/create).

### Config persistence for contentEF
- **D-05:** Persist contentEF config in PrepareAndValidateCollectionRequest when contentEF is provided. Call SetContentEmbeddingFunction on Configuration (or Schema). This enables future GetCollection calls to auto-wire the contentEF from server-side config. The infrastructure already exists — SetContentEmbeddingFunction delegates to SetEmbeddingFunction when the contentEF implements EmbeddingFunction.

### HTTP CreateCollection contentEF wiring
- **D-06:** Mirror the denseEF close-once + ownsEF pattern. Add `contentEmbeddingFunction: wrapContentEFCloseOnce(req.contentEmbeddingFunction)` to the CollectionImpl constructor in HTTP CreateCollection. The existing ownsEF flag already covers both EFs (established in Phase 11). Nil-safe — wrapContentEFCloseOnce returns nil for nil input.

### Claude's Discretion
- Test structure and file organization
- Whether to add contentEF wiring to embedded CreateCollection's isNewCreation=true path (storing in state via upsertCollectionState)
- Internal helper decomposition

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Issue
- GitHub Issue #486 — GetOrCreateCollection contentEF support

### CreateCollectionOp (add contentEF field)
- `pkg/api/v2/client.go:244-253` — CreateCollectionOp struct (add contentEmbeddingFunction field)
- `pkg/api/v2/client.go:268-301` — PrepareAndValidateCollectionRequest (add contentEF config persistence)
- `pkg/api/v2/client.go:466-474` — WithEmbeddingFunctionCreate pattern (model WithContentEmbeddingFunctionCreate after this)

### GetCollectionOp (existing contentEF support — reference pattern)
- `pkg/api/v2/client.go:162-207` — GetCollectionOp with contentEmbeddingFunction field and WithContentEmbeddingFunctionGet

### HTTP client (files to modify)
- `pkg/api/v2/client_http.go:313-359` — CreateCollection (add contentEF to CollectionImpl constructor at line 344)
- `pkg/api/v2/client_http.go:361-364` — GetOrCreateCollection (thin wrapper, no changes needed)

### Embedded client (files to modify)
- `pkg/api/v2/client_local_embedded.go:341-413` — CreateCollection (add contentEF to state storage and buildEmbeddedCollection call)
- `pkg/api/v2/client_local_embedded.go:415-453` — GetOrCreateCollection (add contentEF forwarding to GetCollection options)

### Close-once infrastructure (reuse)
- `pkg/api/v2/ef_close_once.go` — wrapContentEFCloseOnce, wrapEFCloseOnce

### Config persistence infrastructure (reuse)
- `pkg/api/v2/configuration.go:247-258` — SetContentEmbeddingFunction (delegates to SetEmbeddingFunction)
- `pkg/api/v2/configuration.go:225-245` — BuildContentEFFromConfig (used by GetCollection auto-wiring)

### SDK research (behavioral reference)
- Python: `get_or_create_collection` — single server call, user-provided EF, no auto-wiring from config
- JS: `getOrCreateCollection` — single server call with `get_or_create: true`, user-provided EF, no auto-wiring
- Both SDKs: only `getCollection` auto-wires from server-persisted config

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `wrapContentEFCloseOnce` / `wrapEFCloseOnce` — close-once wrappers ready to use
- `SetContentEmbeddingFunction` on `CollectionConfigurationImpl` — config persistence for contentEF
- `BuildContentEFFromConfig` — auto-wiring from config (used by GetCollection, not needed here)
- `WithContentEmbeddingFunctionGet` — existing option pattern to model the new Create option after

### Established Patterns
- `CreateCollectionOp` uses functional options pattern (`CreateCollectionOption func(*CreateCollectionOp) error`)
- HTTP CreateCollection constructs `CollectionImpl` with close-once wrapped EFs
- Embedded CreateCollection distinguishes `isNewCreation` vs existing via `get_or_create` flag
- Embedded GetOrCreateCollection tries GetCollection first, then falls back to CreateCollection
- `ownsEF` atomic flag covers both denseEF and contentEF lifecycle (Phase 11)

### Integration Points
- `PrepareAndValidateCollectionRequest` — where contentEF config persistence should be added
- `buildEmbeddedCollection` — already accepts `overrideContentEF` parameter
- `upsertCollectionState` — where embedded state is managed (already handles contentEF)

</code_context>

<specifics>
## Specific Ideas

- SDK behavioral consistency verified: Go should match Python/JS behavior where GetOrCreateCollection uses user-provided EF, not server-config auto-wiring

</specifics>

<deferred>
## Deferred Ideas

- **Full EF conflict detection** — Create a GH issue to track cross-operation EF conflict validation parity with Python's `validate_embedding_function_conflict_on_get` and `validate_embedding_function_conflict_on_create`. Should cover CreateCollection, GetCollection, and GetOrCreateCollection consistently.
- **Embedded GetOrCreateCollection restructuring** — The embedded path diverges from Python/JS by calling GetCollection first (which auto-wires). A future refactor could align it with the single-call pattern. Low priority since the current behavior is correct and internally consistent.

</deferred>

---

*Phase: 20-getorcreatecollection-contentef-support*
*Context gathered: 2026-04-07*
