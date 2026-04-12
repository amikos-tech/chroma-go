# Phase 24: GetOrCreateCollection EF Safety - Context

**Gathered:** 2026-04-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Harden the embedded `GetOrCreateCollection` fallback path so a failed or racing `GetCollection` attempt never leaves `CreateCollection` with a closed embedding function. Preserve the existing local-state precedence rules for reused collections, while making the returned handle usable under concurrent miss/create races.

**In scope:**
- Prevent SDK cleanup from closing caller-provided EFs before `GetOrCreateCollection` falls back to `CreateCollection`
- Define how concurrent miss/create races converge onto existing collection state versus keeping a temporary fallback EF
- Cover dense EF, `contentEF`, and dual-interface ownership paths where the same cleanup mechanism applies
- Add focused colocated `basicv2` regressions, including a concurrent `-race` path, proving EF lifecycle safety

**Out of scope:**
- A broader embedded EF lifecycle redesign with reference counting or lease tracking
- Changing steady-state embedded semantics beyond the narrow fallback/race window for this bug
- New stress/soak infrastructure beyond the minimum coverage needed for `EFL-03`

</domain>

<decisions>
## Implementation Decisions

### Cleanup ownership
- **D-01:** Caller-provided EFs are borrowed during the provisional `GetCollection` path and must never be closed by SDK cleanup when `GetCollection` fails or revalidation falls apart.
- **D-02:** Only SDK-owned EFs created or auto-wired by the SDK itself remain eligible for cleanup on those failure paths.
- **D-03:** Ownership may transfer only after the SDK has a verified collection/state handoff; Phase 24 does not redefine global EF ownership semantics across all paths.

### Concurrent convergence
- **D-04:** Phase 24 uses conditional convergence: when a concurrent winner's state is authoritatively observable, the loser converges onto that winner snapshot instead of keeping a transient override EF.
- **D-05:** The one exception is the already-accepted empty/no-config ambiguity branch: if forced convergence would return a nil or unusable EF handle, keep the temporary fallback EF so the returned collection still works.
- **D-06:** This conditional rule is intentionally narrow and exists to avoid reintroducing the Phase 23 "reloaded existing collection with no usable EF" failure shape.

### EF coverage breadth
- **D-07:** Phase 24 covers the shared ownership bug class, not just the dense-EF symptom. The fix and tests must cover dense EF, `contentEF`, and dual-interface content EFs that also expose dense behavior.
- **D-08:** Shared-resource close behavior must continue to treat dual-interface content EFs as the owning close path when dense and content wrappers resolve to the same underlying resource.

### Verification depth
- **D-09:** Verification stays narrow and colocated: one deterministic failure-path regression for the fallback bug and one orchestrated concurrent `GetOrCreateCollection` regression intended to pass under `go test -race`.
- **D-10:** Phase 24 does not add repeated stress loops or a new soak harness unless the deterministic concurrent path proves insufficient during implementation.

### the agent's Discretion
- Exact mechanism for marking provisional caller-borrowed EFs versus SDK-owned auto-wired/default EFs in embedded collection state
- Exact branch structure for detecting when winner state is "authoritatively observable" versus when the empty/no-config exception applies
- Exact test fixture shape and synchronization seam, as long as the locked lifecycle semantics and `-race` coverage goals above are both satisfied

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone and requirement context
- `.planning/ROADMAP.md` — Phase 24 goal, dependency on Phase 23, and success criteria for `EFL-02` / `EFL-03`
- `.planning/REQUIREMENTS.md` — `EFL-02` and `EFL-03` requirement text for fallback EF safety and concurrent race coverage
- `.planning/PROJECT.md` — v0.4.2 bug-fix milestone framing and issue `#493` scope
- `.planning/STATE.md` — current project position showing Phase 24 as the active focus

### Prior locked decisions to carry forward
- `.planning/phases/23-ort-ef-leak-fix/23-CONTEXT.md` — narrow embedded-path lifecycle precedent and the empty/no-config reload guard added in Phase 23
- `.planning/milestones/v0.4.1-phases/20-getorcreatecollection-contentef-support/20-CONTEXT.md` — existing-collection precedence and `contentEF` forwarding rules that still apply here
- `.planning/phases/22-withgroupby-validation/22-CONTEXT.md` — recent bug-fix precedent for stable contracts and narrow colocated regression coverage

### Research and risk framing
- `.planning/research/FEATURES.md` — issue `#493` root-cause note that `GetCollection` cleanup can close user EFs before fallback, plus milestone-level bug framing
- `.planning/research/ARCHITECTURE.md` — code-path analysis for embedded `GetOrCreateCollection` and Phase 24 integration boundaries
- `.planning/research/PITFALLS.md` — warnings about wrapper layering, ownership mistakes, and race-sensitive EF cleanup behavior

### Implementation targets
- `pkg/api/v2/client_local_embedded.go` — embedded `GetOrCreateCollection`, `CreateCollection`, `GetCollection`, `returnedExistingCollection`, `deleteCollectionState`, and `upsertCollectionState`
- `pkg/api/v2/client.go` — `CreateCollectionOp` ownership inputs and `PrepareAndValidateCollectionRequest`
- `pkg/api/v2/close_logging.go` — shared close behavior for dense/content EF cleanup and shared-resource detection
- `pkg/api/v2/ef_close_once.go` — close-once wrapper behavior and `errEFClosed` semantics that Phase 24 must preserve

### Test references
- `.planning/codebase/TESTING.md` — repo test conventions, `basicv2` expectations, and `go test` / `-race` fit
- `.planning/codebase/CONVENTIONS.md` — error-handling and narrow V2 bug-fix conventions
- `pkg/api/v2/client_local_embedded_test.go` — existing `GetOrCreateCollection`, existing-state, default-EF, and `contentEF` regression patterns to extend
- `pkg/api/v2/close_review_test.go` — shared dense/content close behavior and dual-interface ownership expectations
- `pkg/api/v2/ef_close_once_test.go` — use-after-close and wrapper identity/close-count test helpers

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `returnedExistingCollection` in `pkg/api/v2/client_local_embedded.go`: existing helper for deciding whether the embedded runtime already returned an authoritative existing collection
- `deleteCollectionState` / `upsertCollectionState` in `pkg/api/v2/client_local_embedded.go`: current lifecycle hooks that need ownership-aware cleanup instead of unconditional close behavior
- `closeEmbeddingFunctions` in `pkg/api/v2/close_logging.go`: shared dense/content cleanup helper that already knows how to avoid double-closing shared dense/content resources
- `wrapEFCloseOnce` / `wrapContentEFCloseOnce` in `pkg/api/v2/ef_close_once.go`: canonical idempotent wrapper entry points that must remain the only wrapping layer
- Existing embedded test runtimes such as `newCountingMemoryEmbeddedRuntime`, `newBlockingGetMemoryEmbeddedRuntime`, and the Phase 23 default-EF fixtures in `pkg/api/v2/client_local_embedded_test.go`

### Established Patterns
- Embedded `GetOrCreateCollection` first tries `GetCollection`, then falls back to `CreateCollection(..., WithIfNotExistsCreate())`
- Existing reused collections preserve authoritative state when that state is already known, but embedded local state can still be updated by explicit `GetCollection` / `GetOrCreateCollection` calls
- Shared dense/content cleanup is centralized, and dual-interface content EFs are treated as the primary close owner when both wrappers point at the same underlying resource
- Narrow bug fixes in this repo are usually locked down with colocated `basicv2` tests instead of new infrastructure

### Integration Points
- `pkg/api/v2/client_local_embedded.go`: teach provisional `GetCollection` state how to distinguish caller-borrowed EFs from SDK-owned temporary EFs
- `pkg/api/v2/client_local_embedded.go`: apply conditional convergence when `CreateCollection` discovers a concurrent winner and decides whether to reload winner state or keep the temporary fallback EF
- `pkg/api/v2/client_local_embedded_test.go`: add one deterministic fallback cleanup regression and one orchestrated concurrent `GetOrCreateCollection` regression under `-race`
- `pkg/api/v2/close_logging.go` and related tests: extend coverage so dense/content/dual-interface cleanup semantics stay aligned after the ownership fix

</code_context>

<specifics>
## Specific Ideas

- The bug is not just "avoid close" — the returned collection handle must still be usable after the race or fallback completes.
- Conditional convergence is preferred over unconditional convergence because the Phase 23 empty/no-config branch already proved that blindly reloading the winner can hand the caller a collection with no usable EF.
- The ownership fix should cover the same shared mechanism across dense, `contentEF`, and dual-interface content EFs rather than patching only the dense symptom.

</specifics>

<deferred>
## Deferred Ideas

- Reference counting / lease-based EF lifecycle management across shared state and returned collections — broader lifecycle work than Phase 24 allows
- Broader repeated stress or soak coverage for embedded EF lifecycle races — defer unless the deterministic `-race` regression proves insufficient
- Any cross-backend semantic rewrite of caller-owned EF lifecycle beyond the embedded fallback bug class addressed here

</deferred>

---

*Phase: 24-getorcreatecollection-ef-safety*
*Context gathered: 2026-04-12*
