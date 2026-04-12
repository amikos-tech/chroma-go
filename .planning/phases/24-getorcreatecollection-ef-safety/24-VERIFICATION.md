---
phase: 24-getorcreatecollection-ef-safety
verified: 2026-04-12T17:44:16+03:00
status: passed
score: 9/9 must-haves verified
overrides_applied: 0
---

# Phase 24: GetOrCreateCollection EF Safety Verification Report

**Phase Goal:** `GetOrCreateCollection` never passes a closed EF to `CreateCollection` fallback  
**Verified:** 2026-04-12T17:44:16+03:00  
**Status:** passed  
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Embedded provisional `GetCollection` state distinguishes borrowed caller EFs from SDK-owned auto-wired/default EFs instead of treating every stored wrapper as cleanup-owned | ✓ VERIFIED | `pkg/api/v2/client_local_embedded.go:706-715` stores close-once wrappers and sets `ownsContentEmbeddingFunction` / `ownsEmbeddingFunction` from whether the request supplied those EFs versus the SDK resolving them. |
| 2 | A failed or revalidation-broken provisional `GetCollection` path never closes caller-provided dense EF, `contentEF`, or dual-interface content EF before `GetOrCreateCollection` fallback completes | ✓ VERIFIED | `pkg/api/v2/client_local_embedded.go:730-744` routes failure cleanup through `deleteCollectionState(...)`, which now uses `closeOwnedEmbeddingFunctions(...)` at `pkg/api/v2/client_local_embedded.go:977-995` and `pkg/api/v2/close_logging.go:53-75`. `pkg/api/v2/client_local_embedded_test.go:1989-2083` proves dense, content-only, and dual-interface caller EFs remain usable until the returned collection is closed. |
| 3 | Borrowed-vs-owned provenance is assigned inside the existing `GetCollection(...)` lock scope after auto-wiring resolves, so `deleteCollectionState(...)` reads the correct cleanup policy if revalidation fails immediately afterward | ✓ VERIFIED | `pkg/api/v2/client_local_embedded.go:666-718` resolves auto-wiring and ownership bits while `collectionStateMu` is held, then snapshots that state before the revalidation call at `pkg/api/v2/client_local_embedded.go:725-744`. |
| 4 | A successful verified handoff still promotes adopted caller-provided EFs into owned state lifecycle while `embeddedCollection.ownsEF` remains the separate guard for normal `Close()` on returned handles | ✓ VERIFIED | `pkg/api/v2/client_local_embedded.go:746-753` promotes successful state back to owned lifecycle, while `pkg/api/v2/client_local_embedded.go:1130` and `pkg/api/v2/client_local_embedded.go:1767-1778` keep returned-collection `Close()` ownership on `embeddedCollection.ownsEF`. |
| 5 | `deleteCollectionState` closes only owned EF paths and still preserves the existing shared-resource rule that dual-interface content EF is the single close owner | ✓ VERIFIED | `pkg/api/v2/client_local_embedded.go:977-995` passes the ownership bits into `closeOwnedEmbeddingFunctions(...)`, and `pkg/api/v2/close_logging.go:53-75` still skips dense close when `isDenseEFSharedWithContent(...)` is true. |
| 6 | Concurrent `GetOrCreateCollection` miss/create races return usable collections under `-race`, converging to authoritative winner state when available and keeping the temporary fallback EF only for the locked empty/no-config ambiguity branch | ✓ VERIFIED | `pkg/api/v2/client_local_embedded.go:463-485` reloads authoritative winner state when reuse is detected and no state/cache exists, while `pkg/api/v2/client_local_embedded_test.go:2085-2165` exercises the shared-client miss/create race and proves both returned handles remain usable under `-race`. |
| 7 | `ListCollections` and other `buildEmbeddedCollection(..., nil, nil, ...)` rebuild paths keep sourcing EF handles from stored state/config without accidentally promoting new cleanup ownership on nil-override paths | ✓ VERIFIED | `pkg/api/v2/client_local_embedded.go:821-828` calls `buildEmbeddedCollection(...)` with nil overrides, and `pkg/api/v2/client_local_embedded.go:1097-1133` only mutates state ownership when explicit overrides are provided. `pkg/api/v2/client_local_embedded_test.go:3975-3981` confirms `ListCollections` still rebuilds content EF from stored state. |
| 8 | Focused deterministic and concurrent regressions cover dense EF, `contentEF`, and dual-interface ownership paths without introducing soak/stress infrastructure | ✓ VERIFIED | The deterministic revalidation-failure helper at `pkg/api/v2/client_local_embedded_test.go:362-375` and the create-blocking race helper at `pkg/api/v2/client_local_embedded_test.go:350-360` support `pkg/api/v2/client_local_embedded_test.go:1989-2165`, which covers all three EF shapes plus the concurrent winner/loser path. |
| 9 | `make test` and `make lint` remain green after the Phase 24 fix | ✓ VERIFIED | Fresh verification on this tree passed: `go test -race -tags=basicv2 -run 'TestEmbeddedLocalClientGetOrCreateCollection_ConcurrentRaceReturnsUsableCollection|TestEmbeddedLocalClientGetOrCreateCollection_FallbackAfterProvisionalGetFailureKeepsCallerEFOpen' ./pkg/api/v2/...` -> `ok github.com/amikos-tech/chroma-go/pkg/api/v2 1.689s`; `make test` -> `DONE 1783 tests, 7 skipped`; `make lint` -> `0 issues.` |

**Score:** 9/9 truths verified

### Roadmap Success Criteria

| # | Success Criterion | Status | Evidence |
| --- | --- | --- | --- |
| 1 | When `GetCollection` fails and `GetOrCreateCollection` falls back to `CreateCollection`, the EF passed is still open and usable | ✓ VERIFIED | `pkg/api/v2/client_local_embedded_test.go:1989-2083` validates this for dense, content-only, and dual-interface caller EFs after the injected revalidation failure. |
| 2 | Concurrent `GetOrCreateCollection` calls under `-race` do not trigger data races or double-close panics | ✓ VERIFIED | The focused `-race` run passed on `TestEmbeddedLocalClientGetOrCreateCollection_ConcurrentRaceReturnsUsableCollection` and `...FallbackAfterProvisionalGetFailureKeepsCallerEFOpen`. |
| 3 | Tests demonstrate the EF lifecycle under concurrent access | ✓ VERIFIED | `pkg/api/v2/client_local_embedded_test.go:2085-2165` proves both concurrent callers receive usable collections and the shared dual-interface EF closes exactly once. |

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `pkg/api/v2/client_local_embedded.go` | Ownership-aware provisional state and narrow fallback/race convergence | ✓ VERIFIED | `pkg/api/v2/client_local_embedded.go:463-518`, `639-753`, and `977-995` contain the ownership tracking, reuse reload, and conditional cleanup logic. |
| `pkg/api/v2/close_logging.go` | Ownership-gated cleanup helper preserving shared dense/content close rules | ✓ VERIFIED | `pkg/api/v2/close_logging.go:49-75` defines `closeOwnedEmbeddingFunctions(...)` and routes `closeEmbeddingFunctions(...)` through the shared helper. |
| `pkg/api/v2/client_local_embedded_test.go` | Deterministic fallback regression and concurrent `-race` coverage | ✓ VERIFIED | `pkg/api/v2/client_local_embedded_test.go:350-375` plus `1989-2165` contain the new helper seams and both required regressions. |
| `.planning/phases/24-getorcreatecollection-ef-safety/24-01-SUMMARY.md` | Execution summary tied to the phase commits and verification output | ✓ VERIFIED | Summary exists on disk and records the four Phase 24 task commits plus the fresh test/lint evidence. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `pkg/api/v2/client_local_embedded.go:GetCollection` | `pkg/api/v2/client_local_embedded.go:deleteCollectionState` | Provisional ownership bits determine which EF paths failure cleanup may close | WIRED | The lock-scoped ownership assignment at `pkg/api/v2/client_local_embedded.go:706-715` feeds `deleteCollectionState(...)` at `pkg/api/v2/client_local_embedded.go:977-995`. |
| `pkg/api/v2/client_local_embedded.go:deleteCollectionState` | `pkg/api/v2/close_logging.go:closeOwnedEmbeddingFunctions` | State cleanup closes only owned dense/content paths and preserves shared-resource single-close behavior | WIRED | `deleteCollectionState(...)` delegates directly to the ownership-aware cleanup helper. |
| `pkg/api/v2/client_local_embedded.go:GetOrCreateCollection` | `pkg/api/v2/client_local_embedded_test.go:TestEmbeddedLocalClientGetOrCreateCollection_ConcurrentRaceReturnsUsableCollection` | Concurrent loser path reloads authoritative winner state when available | WIRED | The reuse reload at `pkg/api/v2/client_local_embedded.go:463-485` is exercised by the shared-client race test at `pkg/api/v2/client_local_embedded_test.go:2085-2165`. |
| `pkg/api/v2/client_local_embedded.go:ListCollections` | `pkg/api/v2/client_local_embedded.go:buildEmbeddedCollection` | Nil-override rebuild paths continue sourcing EF handles from stored state/config | WIRED | `ListCollections(...)` rebuilds via `buildEmbeddedCollection(..., nil, nil, ...)`, and the stored-state regression remains green. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Fallback caller-EF survival plus concurrent race usability under the race detector | `go test -race -tags=basicv2 -run 'TestEmbeddedLocalClientGetOrCreateCollection_ConcurrentRaceReturnsUsableCollection\|TestEmbeddedLocalClientGetOrCreateCollection_FallbackAfterProvisionalGetFailureKeepsCallerEFOpen' ./pkg/api/v2/...` | `ok github.com/amikos-tech/chroma-go/pkg/api/v2 1.689s` | ✓ PASS |
| Full repo regression suite | `make test` | `DONE 1783 tests, 7 skipped` | ✓ PASS |
| Lint gate | `make lint` | `0 issues.` | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| `EFL-02` | `24-01-PLAN.md` | `GetOrCreateCollection` does not pass closed EFs to `CreateCollection` fallback when `GetCollection` fails mid-build | ✓ SATISFIED | `REQUIREMENTS.md:18` defines the requirement. The fallback regression at `pkg/api/v2/client_local_embedded_test.go:1989-2083` proves dense/content/dual-interface caller EFs remain open and usable through the fallback path. |
| `EFL-03` | `24-01-PLAN.md` | Tests cover EF lifecycle under `-race` flag for concurrent `GetOrCreateCollection` calls | ✓ SATISFIED | `REQUIREMENTS.md:19` defines the requirement. The `-race` regression at `pkg/api/v2/client_local_embedded_test.go:2085-2165` plus the passing focused race command above satisfy the concurrency coverage requirement. |

Orphaned requirements for Phase 24: none. `24-01-PLAN.md` declares only `EFL-02` and `EFL-03`, and `REQUIREMENTS.md` maps only those IDs to Phase 24.

### Anti-Patterns Found

One advisory follow-up remains in `24-REVIEW.md`: `GetCollection(...)` still overwrites an already-owned state EF before revalidation, so a later revalidation failure can drop the previous owner wrapper instead of restoring it. This does not block `EFL-02` / `EFL-03` because the caller-EF fallback bug and the concurrent race behavior are both fixed and verified, but it is a narrow embedded lifecycle follow-up worth tracking.

### Human Verification Required

None. The phase goal and all declared must-haves are verifiable programmatically through direct code inspection and automated test evidence.

### Gaps Summary

No gaps found for the Phase 24 goal. The must-haves, roadmap success criteria, and both mapped requirements are satisfied, and the focused race/test/lint gates are green. The remaining code-review warning is advisory and outside the locked phase success criteria.

---

_Verified: 2026-04-12T17:44:16+03:00_  
_Verifier: Codex manual verification_
