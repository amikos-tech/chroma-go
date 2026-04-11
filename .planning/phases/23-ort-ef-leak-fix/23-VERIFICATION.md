---
phase: 23-ort-ef-leak-fix
verified: 2026-04-11T09:48:23Z
status: passed
score: 8/8 must-haves verified
overrides_applied: 0
---

# Phase 23: ORT EF Leak Fix Verification Report

**Phase Goal:** Default ORT embedding function is properly cleaned up when `CreateCollection` encounters an existing collection  
**Verified:** 2026-04-11T09:48:23Z  
**Status:** passed  
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | `CreateCollectionOp` exposes a per-op default dense EF factory seam instead of a package-global override | ✓ VERIFIED | `pkg/api/v2/client.go:245-258` defines `defaultDenseEFFactory` and stores it on `CreateCollectionOp`; `pkg/api/v2/client.go:523-528` adds the package-local `withDefaultDenseEFFactoryCreate(...)` helper. |
| 2 | Request validation resets default-EF provenance on every run so stale ownership cannot leak across reuse | ✓ VERIFIED | `pkg/api/v2/client.go:277-283` clears `op.sdkOwnedDefaultDenseEF = nil` before any default creation logic. |
| 3 | When validation auto-creates the default dense EF, it tracks the exact SDK-owned instance that is currently active | ✓ VERIFIED | `pkg/api/v2/client.go:288-293` calls `op.defaultDenseEFFactory()`, assigns the EF to `op.embeddingFunction`, and stores the same instance in `op.sdkOwnedDefaultDenseEF`. |
| 4 | Dual-interface content-EF promotion clears the tracked SDK-owned default before replacing the runtime dense EF | ✓ VERIFIED | `pkg/api/v2/client.go:330-337` closes the temporary default during promotion, nils `op.sdkOwnedDefaultDenseEF`, and then swaps in the promoted dense EF. |
| 5 | Embedded existing-collection create closes only the tracked SDK-owned default EF when it is still the live dense EF | ✓ VERIFIED | `pkg/api/v2/client_local_embedded.go:409-418` gates cleanup on `req.sdkOwnedDefaultDenseEF != nil && req.embeddingFunction == req.sdkOwnedDefaultDenseEF` before discarding overrides. |
| 6 | Cleanup failures surface synchronously with the exact wrapped error required by the plan | ✓ VERIFIED | `pkg/api/v2/client_local_embedded.go:411-417` returns `errors.Wrap(err, "error closing default embedding function for existing collection")` if `Close()` fails. |
| 7 | Existing embedded collection state still wins on the idempotent create path while the temporary default is closed exactly once | ✓ VERIFIED | `pkg/api/v2/client_local_embedded_test.go:1661-1697` proves the temporary default close count reaches `1` while `gotCollection.embeddingFunctionSnapshot()` still unwraps to `initialEF` and metadata remains `"initial"`. |
| 8 | New embedded collection creation does not eagerly close the temporary default; ownership transfers to collection close | ✓ VERIFIED | `pkg/api/v2/client_local_embedded_test.go:1727-1748` asserts close count `0` before `got.Close()` and `1` after the collection is closed. |

**Score:** 8/8 truths verified

### Roadmap Success Criteria

| # | Success Criterion | Status | Evidence |
| --- | --- | --- | --- |
| 1 | When `CreateCollection` finds an existing collection, any default ORT EF created by `PrepareAndValidateCollectionRequest` is closed | ✓ VERIFIED | Proven structurally by `pkg/api/v2/client.go:288-293` plus `pkg/api/v2/client_local_embedded.go:409-418`, and behaviorally by `pkg/api/v2/client_local_embedded_test.go:1661-1697` and the uncached focused regression run below. |
| 2 | No ORT runtime resources remain open after `CreateCollection` returns in the existing-collection path | ✓ VERIFIED | The phase uses close-counting lifecycle regressions rather than native leak instrumentation. `pkg/api/v2/client_local_embedded_test.go:1661-1697` verifies the SDK-owned temporary default is closed exactly once before return, which is the phase’s explicit cleanup contract for releasing the owned ORT resource. |

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `pkg/api/v2/client.go` | Per-op seam, explicit SDK-owned default EF provenance, and no package-global test seam | ✓ VERIFIED | `pkg/api/v2/client.go:245-258`, `261-293`, `330-337`, and `523-528` contain the new seam/provenance logic. `rg -n "var newDefaultDenseEF = ort.NewDefaultEmbeddingFunction" pkg/api/v2/client.go` returned `absent`. |
| `pkg/api/v2/client_local_embedded.go` | Existing-path cleanup gate with exact closability and wrapped-error behavior | ✓ VERIFIED | `pkg/api/v2/client_local_embedded.go:409-418` contains the exact cleanup gate, closability error, and wrapped cleanup error string. |
| `pkg/api/v2/client_local_embedded_test.go` | Focused `basicv2` regressions for existing-path cleanup, cleanup failure, and new-collection ownership | ✓ VERIFIED | `pkg/api/v2/client_local_embedded_test.go:1661-1748` contains all three required tests with close-count and state-preservation assertions. |
| `.planning/phases/23-ort-ef-leak-fix/23-01-SUMMARY.md` | Execution summary tied to the phase commits and verification output | ✓ VERIFIED | Summary exists on disk and was committed in `62c4351`. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `pkg/api/v2/client.go:261-293` | `pkg/api/v2/client_local_embedded.go:409-418` | Tracked `sdkOwnedDefaultDenseEF` instance gates existing-path cleanup | WIRED | Validation records the exact SDK-owned default EF instance and the embedded existing-path cleanup only runs when the live dense EF is still that same instance. |
| `pkg/api/v2/client.go:330-337` | `pkg/api/v2/client_local_embedded.go:409-418` | Promotion clears stale provenance before cleanup can run | WIRED | Content-EF promotion nils `sdkOwnedDefaultDenseEF` before swapping the dense EF, so the embedded cleanup gate cannot close a replaced runtime EF. |
| `pkg/api/v2/client_local_embedded.go:409-418` | `pkg/api/v2/client_local_embedded_test.go:1661-1748` | Regression tests prove success, failure, and new-collection ownership behavior | WIRED | The three focused tests exercise the cleanup success path, cleanup-error propagation, and non-eager-close behavior for new collections. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Existing-path cleanup, cleanup-error propagation, and Phase 20 precedence guard | `go test -count=1 -tags=basicv2 -run 'TestEmbeddedLocalClientCreateCollection_IfNotExistsExistingDoesNotOverrideState\|TestEmbeddedCreateCollection_DefaultORT.*' ./pkg/api/v2/...` | `ok github.com/amikos-tech/chroma-go/pkg/api/v2 0.483s` | ✓ PASS |
| Full V2 regression suite | `make test` | `DONE 1732 tests, 7 skipped` | ✓ PASS |
| Lint gate | `make lint` | `0 issues.` | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| `EFL-01` | `23-01-PLAN.md` | Default ORT EF created by `PrepareAndValidateCollectionRequest` is closed when `CreateCollection` finds an existing collection | ✓ SATISFIED | `REQUIREMENTS.md:17` defines the requirement and `REQUIREMENTS.md:67` maps it to Phase 23. The implementation and focused regressions above verify the SDK-owned temporary default EF is closed on the embedded existing path and that cleanup errors are returned synchronously. |

Orphaned requirements for Phase 23: none. `23-01-PLAN.md` declares only `EFL-01`, and `REQUIREMENTS.md` maps only `EFL-01` to Phase 23.

### Anti-Patterns Found

None. The rejected package-global seam is absent, ownership tracking uses the actual EF instance rather than a boolean-only marker, and the phase stays within the narrow embedded lifecycle scope defined by the plan and context.

### Human Verification Required

None. The phase goal and all must-haves are verifiable programmatically through direct code inspection and automated test evidence.

### Gaps Summary

No gaps found. The phase goal is achieved, `EFL-01` is fully satisfied, the focused uncached regression slice passes, the repo-wide `make test` and `make lint` gates are green, and the implementation matches the plan’s must-haves without widening into Phase 24 or a broader create-flow refactor.

---

_Verified: 2026-04-11T09:48:23Z_  
_Verifier: Codex manual verification_
