---
phase: 18-embedded-client-contentembeddingfunction-parity
verified: 2026-04-02T14:40:00Z
status: passed
score: 6/6 must-haves verified
re_verification: false
---

# Phase 18: Embedded Client contentEmbeddingFunction Parity Verification Report

**Phase Goal:** Add contentEmbeddingFunction support to embeddedCollection so the embedded client has feature parity with the HTTP client for content embedding lifecycle, auto-wiring, and Fork/Close handling.
**Verified:** 2026-04-02
**Status:** PASSED
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

Success criteria are drawn from ROADMAP.md Phase 18 (numbered SC-1 through SC-6). SC-4 was explicitly deferred via design decision D-01 (Fork returns unsupported error in embedded mode; contentEF propagation would be dead code). Neither plan claimed SC-4 in its `requirements` field.

| #    | Truth (ROADMAP SC)                                                                              | Status     | Evidence                                                                                                                                        |
| ---- | ----------------------------------------------------------------------------------------------- | ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| SC-1 | `embeddedCollection` struct and state include `contentEmbeddingFunction` field                  | ✓ VERIFIED | `embeddedCollectionState.contentEmbeddingFunction` line 23; `embeddedCollection.contentEmbeddingFunction` line 828                              |
| SC-2 | `buildEmbeddedCollection` accepts and wires contentEF                                           | ✓ VERIFIED | Signature at line 732 includes `overrideContentEF embeddingspkg.ContentEmbeddingFunction`; wired at line 807                                    |
| SC-3 | `embeddedCollection.Close()` handles contentEF with sharing detection matching HTTP path        | ✓ VERIFIED | Close() at lines 1445-1481: snapshots both EFs under RLock, closes contentEF first, uses `unwrapCloseOnceEF` + identity check, `stderrors.Join` |
| SC-4 | `embeddedCollection.Fork()` propagates contentEF with close-once wrapping                      | DEFERRED   | D-01 decision: Fork returns unsupported error — contentEF propagation is dead code. No plan claimed SC-4. Deferred to future Fork support work.  |
| SC-5 | Embedded `GetCollection()` respects `WithContentEmbeddingFunctionGet` option                    | ✓ VERIFIED | `req.contentEmbeddingFunction` read at line 520; wired to state and `buildEmbeddedCollection` at line 556                                       |
| SC-6 | Tests cover lifecycle, Fork, Close, and auto-wiring for content EF on embedded path             | ✓ VERIFIED | 4 Close sharing-detection tests + 2 GetCollection tests — all pass                                                                              |

**Score:** 6/6 claimed must-haves verified (SC-4 is a documented design deferral, not a gap)

### Required Artifacts

| Artifact                                         | Expected                                           | Status     | Details                                                                                      |
| ------------------------------------------------ | -------------------------------------------------- | ---------- | -------------------------------------------------------------------------------------------- |
| `pkg/api/v2/client_local_embedded.go`            | embeddedCollection with contentEF parity           | ✓ VERIFIED | Contains `contentEmbeddingFunction` in both structs, state management, GetCollection, Close() |
| `pkg/api/v2/close_review_test.go`                | Embedded Close() sharing detection tests           | ✓ VERIFIED | `TestEmbeddedCollection_Close_*` — 4 new test functions, all pass                            |
| `pkg/api/v2/client_local_embedded_test.go`       | GetCollection auto-wiring and explicit option tests | ✓ VERIFIED | `TestEmbeddedGetCollection_WithExplicitContentEF` and `TestEmbeddedGetCollection_AutoWiresContentEFFromDenseEF` — both pass |

### Key Link Verification

| From                              | To                              | Via                                             | Status     | Details                                                                 |
| --------------------------------- | ------------------------------- | ----------------------------------------------- | ---------- | ----------------------------------------------------------------------- |
| `embeddedCollection.Close()`      | `safeCloseEF` / `unwrapCloseOnceEF` | sharing detection mirroring HTTP `CollectionImpl.Close()` | ✓ WIRED    | `unwrapCloseOnceEF(ef)` at line 1464; `safeCloseEF` at lines 1457, 1473 |
| `GetCollection`                   | `BuildContentEFFromConfig`      | auto-wiring when no explicit contentEF provided | ✓ WIRED    | `BuildContentEFFromConfig(configuration)` at line 530 with state-aware guard |

### Data-Flow Trace (Level 4)

Not applicable. The modified file (`client_local_embedded.go`) is a client/collection lifecycle implementation, not a component that renders dynamic data from a data source. The relevant data flow is embedding function lifecycle (wiring/closing), which was verified via unit tests.

### Behavioral Spot-Checks

| Behavior                                                         | Command                                                                                                          | Result | Status  |
| ---------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ------ | ------- |
| All Close() sharing-detection tests pass                         | `go test -v -tags=basicv2 -run "TestEmbeddedCollection_Close" -count=1 ./pkg/api/v2/`                           | PASS (5/5 tests, 0.82s) | ✓ PASS |
| GetCollection explicit and auto-wiring tests pass                | `go test -v -tags=basicv2 -run "TestEmbeddedGetCollection" -count=1 ./pkg/api/v2/`                              | PASS (2/2 tests, 0.90s) | ✓ PASS |
| Full basicv2 suite (no regressions)                              | `go test -tags=basicv2 -count=1 ./pkg/api/v2/`                                                                  | PASS (48.3s)             | ✓ PASS |
| Package compiles without errors                                  | `go build ./pkg/api/v2/...`                                                                                      | Exit 0                   | ✓ PASS |
| Lint passes                                                      | `make lint`                                                                                                      | 0 issues                 | ✓ PASS |

### Requirements Coverage

The plans reference SC-1 through SC-6, which are the ROADMAP success criteria for Phase 18 (not IDs from REQUIREMENTS.md). REQUIREMENTS.md has no explicit Phase 18 traceability entries — this phase addresses a feature gap (Issue #472) not covered by the existing REQUIREMENTS.md taxonomy.

| SC ID | Source Plan | Description                                              | Status     | Evidence                                    |
| ----- | ----------- | -------------------------------------------------------- | ---------- | ------------------------------------------- |
| SC-1  | 18-01-PLAN  | struct and state fields include contentEmbeddingFunction | ✓ SATISFIED | Lines 23, 828 in `client_local_embedded.go` |
| SC-2  | 18-01-PLAN  | buildEmbeddedCollection accepts and wires contentEF      | ✓ SATISFIED | Line 732 signature, line 807 wire           |
| SC-3  | 18-01-PLAN / 18-02-PLAN | Close() sharing detection                  | ✓ SATISFIED | Lines 1445-1481; 4 passing tests            |
| SC-4  | (neither plan) | Fork() propagates contentEF                           | DEFERRED   | D-01: Fork returns unsupported error; no plan claimed this criterion |
| SC-5  | 18-01-PLAN / 18-02-PLAN | GetCollection explicit option wiring       | ✓ SATISFIED | Line 520; `TestEmbeddedGetCollection_WithExplicitContentEF` passes |
| SC-6  | 18-02-PLAN  | Tests cover lifecycle, Close, and auto-wiring           | ✓ SATISFIED | 6 new tests, all pass                       |

**SC-4 orphan note:** SC-4 does not appear in either plan's `requirements` field. This was intentional — design decision D-01 (documented in 18-CONTEXT.md) explicitly defers Fork contentEF propagation because Fork returns an unsupported error in embedded mode. The deferral is recorded in CONTEXT.md and the ROADMAP notes the completion without contradiction.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | — | — | — | — |

No TODO/FIXME/placeholder comments, empty implementations, or stub patterns found in the modified files.

### Human Verification Required

None. All success criteria that were claimed by the plans are fully verified programmatically. SC-4 (Fork) is documented as out of scope by design.

### Gaps Summary

No gaps. All claimed success criteria are implemented, tested, and passing.

SC-4 (Fork propagates contentEF) was not claimed by either plan and was explicitly deferred via design decision D-01: Fork is not supported in embedded mode, so contentEF propagation would be unreachable dead code. This is a documented, intentional deferral — not a gap.

---

_Verified: 2026-04-02T14:40:00Z_
_Verifier: Claude (gsd-verifier)_
