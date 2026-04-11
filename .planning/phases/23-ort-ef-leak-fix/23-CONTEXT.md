# Phase 23: ORT EF Leak Fix - Context

**Gathered:** 2026-04-10
**Status:** Ready for planning

<domain>
## Phase Boundary

Fix the embedded `CreateCollection(..., WithIfNotExistsCreate())` path so an auto-created default ORT embedding function does not leak when the target collection already exists. The phase is limited to cleanup and verification of that abandoned SDK-owned default EF on the existing-collection branch.

**In scope:**
- Detect the existing-collection path that currently drops an auto-created default ORT EF
- Ensure the SDK-owned default ORT EF is closed before the call returns on that path
- Add focused regression coverage proving both the close behavior and the preserved existing-collection state behavior
- Define the synchronous error contract if that cleanup fails

**Out of scope:**
- Redesigning shared `PrepareAndValidateCollectionRequest` semantics across all client backends
- Broader EF lifecycle hardening that belongs to Phase 24 (`GetOrCreateCollection` closed-EF safety)
- ORT-specific native leak harnesses or env-gated integration testing beyond the narrow regression bar needed for this bug fix

</domain>

<decisions>
## Implementation Decisions

### Fix shape
- **D-01:** Phase 23 uses a narrow embedded-path fix, not a shared create-flow refactor.
- **D-02:** The fix happens in embedded `CreateCollection` when `isNewCreation=false`: if the dense EF was auto-created by the SDK as the default ORT EF for this request, it must be explicitly closed before the temporary override is discarded.
- **D-03:** Existing-collection EF precedence does not change. Phase 20's decision stands: existing embedded collections keep their state-backed EF/contentEF; temporary override EFs on the existing path are not adopted.

### Verification style
- **D-04:** Verification uses a mixed approach:
  - one focused close-spy/unit regression with a seam around default EF creation
  - one broader existing-collection lifecycle regression proving the returned collection still preserves the original EF/state and does not adopt the temporary default
- **D-05:** Verification stays in the standard colocated `basicv2` test suite; no new ORT-specific integration or leak-detection harness is required for this phase.

### Cleanup failure contract
- **D-06:** If the SDK-owned default ORT EF cannot be closed on the existing-collection path, `CreateCollection` returns an error instead of logging and returning success.
- **D-07:** Log-only handling is reserved for asynchronous cleanup paths like cache/state shutdown cleanup, not for this synchronous request path where the SDK still owns the temporary EF and can surface failure directly.

### Scope guardrails
- **D-08:** Phase 23 does not defer default EF creation until after the existence check. That alternative is a larger shared-flow refactor and is intentionally deferred.
- **D-09:** Phase 23 does not introduce conditional or error-class-based cleanup behavior. A plain fail-on-cleanup-error contract is locked for this phase.

### the agent's Discretion
- Exact internal representation of the "auto-created default dense EF" marker, as long as it reliably distinguishes SDK-created default ORT EFs from caller-provided EFs and promoted dual-interface content EFs
- Exact helper or package-level seam shape used by tests to substitute a close-counting default EF
- Exact test naming and assertion structure, as long as the two locked verification goals above are both covered

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone and requirement context
- `.planning/ROADMAP.md` — Phase 23 goal, success criteria, and dependency boundary relative to Phase 24
- `.planning/REQUIREMENTS.md` — `EFL-01` requirement for closing the default ORT EF on the existing-collection path
- `.planning/PROJECT.md` — v0.4.2 milestone context and bug statement for issue `#494`
- `.planning/STATE.md` — current project position showing Phase 23 as the active focus

### Prior locked decisions to carry forward
- `.planning/milestones/v0.4.1-phases/20-getorcreatecollection-contentef-support/20-CONTEXT.md` — Phase 20 decision `D-03` that existing embedded collections ignore new override EFs/contentEFs and preserve state-backed ones
- `.planning/phases/22-withgroupby-validation/22-CONTEXT.md` — recent bug-fix precedent for narrow scope, stable error contract, and colocated regression coverage

### Research and risk framing
- `.planning/research/ARCHITECTURE.md` — issue `#494` root-cause analysis and the two viable fix directions
- `.planning/research/FEATURES.md` — milestone bug inventory classifying `#494` as a narrow lifecycle bug
- `.planning/research/PITFALLS.md` — warnings about wrapper layering, ownership, and avoiding accidental caller-EF closure during `#493/#494` work

### Implementation targets
- `pkg/api/v2/client.go` — `CreateCollectionOp` and `PrepareAndValidateCollectionRequest`, where the default ORT EF is currently auto-created
- `pkg/api/v2/client_local_embedded.go` — embedded `CreateCollection`, `buildEmbeddedCollection`, and collection-state cleanup behavior
- `pkg/embeddings/default_ef/default_ef.go` — default ORT EF close semantics and runtime teardown behavior
- `pkg/api/v2/close_logging.go` — existing distinction between returned cleanup errors and log-only cleanup paths
- `pkg/api/v2/ef_close_once.go` — close-once wrapping rules that must remain the only wrapping mechanism

### Test references
- `.planning/codebase/TESTING.md` — repo testing conventions and build-tag expectations
- `.planning/codebase/CONVENTIONS.md` — repo error-handling convention to surface explicit failures in synchronous validation/setup paths
- `pkg/api/v2/client_local_embedded_test.go` — existing embedded lifecycle/state tests to extend
- `pkg/api/v2/ef_close_once_test.go` — close-counting helpers and wrapper behavior references
- `pkg/api/v2/close_review_test.go` — cleanup error logging and ownership transfer test patterns

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `wrapEFCloseOnce` / `wrapContentEFCloseOnce` in `pkg/api/v2/ef_close_once.go`: idempotent wrapper helpers that must remain the only wrapping entry points
- `mockCloseableEF`, `mockFailingCloseEF`, and related close-counting helpers in `pkg/api/v2/ef_close_once_test.go`: reusable test doubles for close behavior
- Existing embedded lifecycle tests in `pkg/api/v2/client_local_embedded_test.go`: patterns for asserting preserved state, cleanup errors, and existing-collection behavior

### Established Patterns
- Embedded existing-collection paths preserve state-backed EFs instead of adopting new override EFs
- Synchronous setup/promotion cleanup failures are returned as errors; asynchronous cache/state cleanup failures are typically logged
- Colocated `basicv2` tests with lightweight mocks are preferred over env-gated heavyweight integration tests for narrow bug fixes

### Integration Points
- `pkg/api/v2/client.go`: mark when the dense EF was SDK-created as the temporary default ORT EF during request preparation
- `pkg/api/v2/client_local_embedded.go`: close that temporary SDK-owned default EF on the `isNewCreation=false` path before dropping the override
- `pkg/api/v2/client_local_embedded_test.go`: add one seam-based close regression and one existing-state regression

</code_context>

<specifics>
## Specific Ideas

- The returned existing collection must remain behaviorally identical to today except that the abandoned temporary default ORT EF is no longer leaked.
- The verification bar is intentionally split:
  - prove the temporary default EF is closed exactly once
  - prove the existing collection still uses its original state-backed EF rather than the temporary default
- The recommended contract is intentionally strict: if cleanup of the SDK-created temporary EF fails, surface that failure immediately rather than hiding it behind logging.

</specifics>

<deferred>
## Deferred Ideas

- Defer default EF creation until after the existence check — cleaner by construction, but wider shared-flow refactor than Phase 23 allows
- Unify Phase 23 and Phase 24 into one broader embedded EF lifecycle redesign — explicitly deferred because Phase 24 already owns the closed-EF fallback bug
- Add ORT-native or env-gated leak-detection/integration harnesses — not required for this narrow bug-fix phase
- Introduce typed cleanup-error classes or conditional best-effort cleanup behavior — unnecessary scope for Phase 23

</deferred>

---

*Phase: 23-ort-ef-leak-fix*
*Context gathered: 2026-04-10*
