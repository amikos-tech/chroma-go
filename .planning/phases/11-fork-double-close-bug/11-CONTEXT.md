# Phase 11: Fork Double-Close Bug - Context

**Gathered:** 2026-03-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Fix EF pointer sharing in Fork() that causes the same underlying embedding function resource to be closed twice when client.Close() iterates cached collections. Both `embeddingFunction` and `contentEmbeddingFunction` ownership must be handled correctly. Issue #454.

</domain>

<decisions>
## Implementation Decisions

### Ownership model
- **D-01:** Use an **owner flag + close-once wrapper** combo. Add `ownsEF bool` field to `CollectionImpl` and `embeddedCollection`. Fork() sets it `false`; constructors (CreateCollection, GetCollection, GetOrCreateCollection) set it `true`. Close() gates EF teardown on `ownsEF`.
- **D-02:** Wrap shared EFs in a thin `sync.Once`-based close wrapper that makes Close() idempotent. This is a defence-in-depth layer: if a user manually closes the original collection while a fork is still live, the fork won't panic on subsequent operations — the EF returns a clean error instead of undefined behavior.
- **D-03:** The `ownsEF` flag is set once at creation time and never changes. Forked collections can use the EF for all embedding operations (Add, Query, Search) normally — the flag only gates the Close() teardown path.

### Scope of fix
- **D-04:** Fix both HTTP client (`CollectionImpl.Fork` + `CollectionImpl.Close`) and embedded client (`embeddedCollection.Fork` + `embeddedCollection.Close`). The bug is structurally identical in both paths. The embedded path is arguably worse because `embeddedCollection.Close()` has zero sharing guard today.

### Close semantics
- **D-05:** Forked collection's Close() skips EF teardown (owner flag) but still runs any other cleanup the collection might need. It is not a full no-op — only the EF close is gated.
- **D-06:** The close-once wrapper on the EF is the safety net for edge cases where the original collection is closed before the fork.

### Claude's Discretion
- Close-once wrapper implementation details (struct shape, interface delegation)
- Test structure and naming
- Whether the owner flag field is named `ownsEF`, `isOwner`, or similar

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Bug report
- GitHub Issue #454 — Fork double-close bug report and discussion

### Fork implementation
- `pkg/api/v2/collection_http.go` lines 388-420 — HTTP CollectionImpl.Fork() copies EF pointers directly
- `pkg/api/v2/collection_http.go` lines 684-710 — CollectionImpl.Close() with intra-collection dedup logic
- `pkg/api/v2/client_local_embedded.go` lines 1360-1420 — embeddedCollection.Fork() copies EF pointer
- `pkg/api/v2/client_local_embedded.go` lines 1421-1428 — embeddedCollection.Close() with no sharing guard

### Client close path
- `pkg/api/v2/client_http.go` lines 693-720 — APIClientV2.Close() iterates collectionCache calling collection.Close()

### Prior decisions (Phase 3)
- Phase 03-02 decision: Close contentEF first in CollectionImpl.Close() to avoid double-close when contentEF wraps denseEF (adapter case)

### Existing close-once pattern
- `pkg/embeddings/default_ef.go` — DefaultEmbeddingFunction already uses atomic closed guard internally

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/embeddings/default_ef.go`: Has an existing `atomic` closed guard pattern that could inform the close-once wrapper design
- `CollectionImpl.Close()`: Already has intra-collection sharing detection (EmbeddingFunctionUnwrapper + interface identity check) — the owner flag extends this to cross-collection sharing

### Established Patterns
- Functional options pattern for client/collection construction — owner flag should be set in constructor paths, not via option
- `io.Closer` interface: Collections implement Close() and EFs optionally implement io.Closer — the close-once wrapper must preserve these interface contracts
- `EmbeddingFunctionUnwrapper` interface: Used in Close() to detect content-adapter wrapping — must still work with the close-once wrapper

### Integration Points
- `CollectionImpl` struct: Add `ownsEF bool` field
- `embeddedCollection` struct: Add `ownsEF bool` field
- `Fork()` in both implementations: Set `ownsEF = false` on forked collection
- `CreateCollection`, `GetCollection`, `GetOrCreateCollection` in both clients: Ensure `ownsEF = true`
- `CollectionImpl.Close()` and `embeddedCollection.Close()`: Gate EF close on `ownsEF`

</code_context>

<specifics>
## Specific Ideas

- User wants defensive protection against manual close of original while fork is live — not just fixing the client.Close() iteration path
- The close-once wrapper should make the EF return a clean error on second close, never panic (library must never panic)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 11-fork-double-close-bug*
*Context gathered: 2026-03-26*
