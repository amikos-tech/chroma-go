---
phase: 11-fork-double-close-bug
verified: 2026-03-26T18:15:00Z
status: passed
score: 10/10 must-haves verified
re_verification: false
---

# Phase 11: Fork Double-Close Bug Verification Report

**Phase Goal:** Fix EF pointer sharing in Fork() that causes the same underlying embedding function resource to be closed twice when client.Close() iterates cached collections.
**Verified:** 2026-03-26T18:15:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #  | Truth                                                                                              | Status     | Evidence                                                                                  |
|----|----------------------------------------------------------------------------------------------------|------------|-------------------------------------------------------------------------------------------|
| 1  | Forked collections do not double-close shared EF resources when client.Close() iterates the cache  | VERIFIED   | `ownsEF: false` in Fork(); `Close()` returns nil immediately when `!c.ownsEF`            |
| 2  | Both embeddingFunction and contentEmbeddingFunction ownership is tracked correctly                 | VERIFIED   | Both wrapped in collection_http.go Fork() via `wrapEFCloseOnce` / `wrapContentEFCloseOnce`|
| 3  | Closing the original collection before the fork does not panic the fork                            | VERIFIED   | close-once wrapper makes Close() idempotent via `sync.Once`; second call returns nil      |
| 4  | Existing intra-collection sharing detection (Unwrapper pattern) still works                        | VERIFIED   | `Close()` ownsEF guard wraps the existing sharing detection block; logic untouched        |
| 5  | Close-once wrapper makes Close() idempotent — second call returns nil, not panic                   | VERIFIED   | TestCloseOnceEF_IdempotentClose: closeCount==1 after two Close() calls (PASS)             |
| 6  | Close-once wrapper returns clean error on use-after-close                                          | VERIFIED   | TestCloseOnceEF_UseAfterClose / TestCloseOnceContentEF_UseAfterClose: errEFClosed (PASS)  |
| 7  | Forked collection Close() does not close the underlying EF                                         | VERIFIED   | TestCollectionImpl_ForkOwnsEF / TestEmbeddedCollection_ForkOwnsEF: closeCount==0 (PASS)   |
| 8  | Original collection Close() closes the underlying EF exactly once                                  | VERIFIED   | TestCollectionImpl_ForkOwnsEF: owner closeCount==1 after Close() (PASS)                  |
| 9  | EmbeddingFunctionUnwrapper delegation works through close-once wrapper                             | VERIFIED   | TestCloseOnceEF_UnwrapperDelegation / TestCloseOnceEF_UnwrapperNonUnwrapper (PASS)       |
| 10 | Nil EF wrapping returns nil                                                                        | VERIFIED   | TestWrapEFCloseOnce_NilReturnsNil (PASS)                                                 |

**Score:** 10/10 truths verified

### Required Artifacts

| Artifact                                   | Expected                                              | Status     | Details                                                                     |
|--------------------------------------------|-------------------------------------------------------|------------|-----------------------------------------------------------------------------|
| `pkg/api/v2/ef_close_once.go`              | Close-once wrappers for EF and ContentEF              | VERIFIED   | 132 lines; contains closeOnceEF, closeOnceContentEF, helper functions       |
| `pkg/api/v2/ef_close_once_test.go`         | Unit tests for close-once and ownership gating        | VERIFIED   | 274 lines; 11 test functions; build tag `//go:build basicv2`                |
| `pkg/api/v2/collection_http.go`            | ownsEF field and Close() gating for HTTP collections  | VERIFIED   | ownsEF at line 60; Fork() at lines 416-419; Close() guard at line 687      |
| `pkg/api/v2/client_local_embedded.go`      | ownsEF field and Close() gating for embedded          | VERIFIED   | ownsEF at line 789; buildEmbeddedCollection line 769; Fork() lines 1400-1401; Close() guard line 1426 |
| `pkg/api/v2/client_http.go`                | ownsEF: true in CreateCollection and GetCollection    | VERIFIED   | CreateCollection line 355; GetCollection line 460                           |

### Key Link Verification

| From                              | To                           | Via                                           | Status     | Details                                                                |
|-----------------------------------|------------------------------|-----------------------------------------------|------------|------------------------------------------------------------------------|
| `pkg/api/v2/collection_http.go`   | `pkg/api/v2/ef_close_once.go` | Fork() wraps EFs with wrapEFCloseOnce         | VERIFIED   | Lines 416-417 call wrapEFCloseOnce and wrapContentEFCloseOnce          |
| `pkg/api/v2/client_local_embedded.go` | `pkg/api/v2/ef_close_once.go` | Fork() wraps EF with wrapEFCloseOnce      | VERIFIED   | Line 1401 calls wrapEFCloseOnce after buildEmbeddedCollection          |
| `pkg/api/v2/collection_http.go`   | `CollectionImpl.Close()`     | ownsEF gates EF teardown                      | VERIFIED   | Lines 687-688: `if !c.ownsEF { return nil }`                           |
| `pkg/api/v2/client_local_embedded.go` | `embeddedCollection.Close()` | ownsEF gates EF teardown               | VERIFIED   | Lines 1426-1427: `if !c.ownsEF { return nil }`                        |
| `pkg/api/v2/ef_close_once_test.go` | `pkg/api/v2/ef_close_once.go` | Tests exercise all wrapper functions         | VERIFIED   | All 11 test functions reference closeOnceEF, wrapEFCloseOnce, etc.     |

### Data-Flow Trace (Level 4)

Not applicable — this phase adds ownership/lifecycle logic, not data-rendering components. No dynamic data flows to trace.

### Behavioral Spot-Checks

| Behavior                                                    | Command                                                                                                                              | Result                | Status  |
|-------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------|-----------------------|---------|
| 11 close-once and ownership tests pass                      | `go test -tags=basicv2 -run "TestCloseOnce\|TestWrapEF\|TestCollectionImpl_ForkOwnsEF\|TestEmbeddedCollection_ForkOwnsEF" ./pkg/api/v2/...` | 11/11 PASS       | PASS    |
| Package builds cleanly                                      | `go build ./pkg/api/v2/...`                                                                                                          | exit 0                | PASS    |
| Commits exist in git history                                | `git log --oneline bf0c5ab e68041a fa987e8`                                                                                          | All 3 commits found   | PASS    |

### Requirements Coverage

| Requirement | Source Plan | Description                                                                                      | Status    | Evidence                                                                        |
|-------------|-------------|--------------------------------------------------------------------------------------------------|-----------|---------------------------------------------------------------------------------|
| FORK-01     | 11-01, 11-02 | Forked collections do not double-close shared EF resources when client.Close() iterates the cache | SATISFIED | ownsEF=false on Fork(); Close() returns nil for non-owners; all ownership tests pass |
| FORK-02     | 11-01, 11-02 | Both embeddingFunction and contentEmbeddingFunction ownership tracked via ownsEF flag             | SATISFIED | ownsEF field on both CollectionImpl and embeddedCollection; both EFs wrapped at Fork() |
| FORK-03     | 11-01, 11-02 | Shared EFs wrapped in sync.Once-based close-once adapter as defence-in-depth                      | SATISFIED | closeOnceEF/closeOnceContentEF use sync.Once + atomic.Bool; idempotent close test passes |
| FORK-04     | 11-01, 11-02 | Tests cover Fork + Close lifecycle including idempotent close, use-after-close, ownership gating  | SATISFIED | 11 unit tests in ef_close_once_test.go; all pass; no panics possible           |

Note: REQUIREMENTS.md traceability table shows FORK-04 as "Planned" (not yet updated to "Complete"). This is a documentation lag — the tests exist and pass. No functional gap.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| None | — | — | — | — |

No TODOs, FIXMEs, placeholders, empty returns, or hardcoded stubs found in any phase-11 files.

### Human Verification Required

No items require human verification. The fix is purely mechanical (ownership flag + sync.Once wrapper) with full unit test coverage. No UI, real-time behavior, or external service integration involved.

### Gaps Summary

No gaps. All 10 observable truths verified, all 5 artifacts confirmed substantive and wired, all 5 key links confirmed, all 4 requirements satisfied with evidence.

The only minor discrepancy is cosmetic: REQUIREMENTS.md marks FORK-04 as "Planned" in the traceability table while the implementation and tests are complete. This does not affect functionality.

---

_Verified: 2026-03-26T18:15:00Z_
_Verifier: Claude (gsd-verifier)_
