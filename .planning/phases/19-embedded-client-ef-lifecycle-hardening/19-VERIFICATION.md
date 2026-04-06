---
phase: 19-embedded-client-ef-lifecycle-hardening
verified: 2026-04-06T14:00:00Z
status: passed
score: 9/9 must-haves verified
re_verification: false
---

# Phase 19: Embedded Client EF Lifecycle Hardening Verification Report

**Phase Goal:** Harden EF lifecycle management in embedded client — fix TOCTOU race in GetCollection auto-wiring, add close-once wrapping, proper state map cleanup, symmetric unwrapping, build error guards, and structured logging for observability parity with HTTP client.
**Verified:** 2026-04-06T14:00:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | GetCollection auto-wiring uses check-and-set under write lock to prevent TOCTOU race | VERIFIED | `client_local_embedded.go:529` — `client.collectionStateMu.Lock()` wraps nil-check + build + assign; `TestEmbeddedGetCollection_ConcurrentAutoWire` asserts factory invoked exactly once across 10 concurrent goroutines |
| 2 | `deleteCollectionState` closes EFs before removing the map entry | VERIFIED | `client_local_embedded.go:725-740` — map entry deleted under lock, `closeEmbeddingFunctions` called outside lock on copied state; `TestEmbeddedDeleteCollectionState_ClosesEFs` asserts closeCount==1 and nil map entry |
| 3 | `embeddedLocalClient.Close()` iterates `collectionState` to close any remaining EFs | VERIFIED | `client_local_embedded.go:666-705` — copies+clears state under lock, iterates and calls `closeEmbeddingFunctions` for each; `TestEmbeddedLocalClient_Close_CleansUpCollectionState` covers two-entry case with one failing close |
| 4 | `localDeleteCollectionFromCache` handles `*embeddedCollection` type for EF cleanup on delete | VERIFIED | `client_http.go:816-818` — `if ec, ok := deleted.(*embeddedCollection); ok && ec.ownsEF.Load()` branch exists; `TestDeleteCollectionFromCache_EmbeddedCollection` asserts both EFs closed and cache entry removed |
| 5 | `buildEmbeddedCollection` wraps EFs in close-once wrappers matching the HTTP client pattern | VERIFIED | `client_local_embedded.go:868-869` — `wrapEFCloseOnce(snapshot.embeddingFunction)` and `wrapContentEFCloseOnce(snapshot.contentEmbeddingFunction)`; `TestEmbeddedBuildCollection_CloseOnceWrapping` asserts `*closeOnceEF` and `*closeOnceContentEF` types |
| 6 | `isDenseEFSharedWithContent` unwraps both dense and content EFs symmetrically | VERIFIED | `close_logging.go:42` — `return unwrapCloseOnceEF(efFromContent) == unwrapped`; `TestIsDenseEFSharedWithContent_SymmetricUnwrap` covers wrapped-shared, wrapped-different, unwrapped-shared, unwrapped-different cases |
| 7 | Auto-wired EFs are only assigned when the build error is nil | VERIFIED | `client_local_embedded.go:538-548, 558-570` — both auto-wire paths use `if buildErr != nil { log } else { assign }`; HTTP client `client_http.go:427-431, 443-447, 537-543` also guarded; `TestEmbeddedGetCollection_BuildErrorGuard` asserts nil EFs in state and collection on build failure |
| 8 | Embedded client has an optional structured logger for auto-wire and close errors | VERIFIED | `client_local.go:532-538` — `WithPersistentLogger` option; `client_local_embedded.go:50` — `logger logger.Logger` field (nil by default); all 4 callsites use `if client.logger != nil` guards with stderr fallback; 3 logger tests pass |
| 9 | Tests cover all fixed paths with no regressions | VERIFIED | 12 new tests all pass with `-race` flag; `go build ./pkg/api/v2/...` exits 0 |

**Score:** 9/9 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `pkg/api/v2/client_local_embedded.go` | TOCTOU fix, close-once wrapping in state and collection, delete/close cleanup, logger field and callsites | VERIFIED | Write lock at line 529; `wrapEFCloseOnce` in state (579) and collection (868); `Close()` and `deleteCollectionState` both implement copy-under-lock pattern; logger field at line 50; 4 logger-guarded callsites present |
| `pkg/api/v2/close_logging.go` | Symmetric unwrapping fix in `isDenseEFSharedWithContent` | VERIFIED | Line 42: `return unwrapCloseOnceEF(efFromContent) == unwrapped` |
| `pkg/api/v2/client_http.go` | `*embeddedCollection` type switch in `localDeleteCollectionFromCache`, HTTP build error guards | VERIFIED | Line 816: `*embeddedCollection` case; lines 427-431 and 443-447: GetCollection guards; lines 537-543: ListCollections guard |
| `pkg/api/v2/client_local.go` | `WithPersistentLogger` option function, `logger` field on `localClientConfig` | VERIFIED | Line 115: `logger logger.Logger` field; lines 532-538: `func WithPersistentLogger(l logger.Logger)` with dual propagation |
| `pkg/api/v2/client_local_embedded_test.go` | 9+3=12 new tests covering all fixed paths | VERIFIED | All 12 test functions exist at expected lines; all pass with race detector |
| `pkg/api/v2/close_review_test.go` | Tests for symmetric unwrapping and delete-from-cache embedded type | VERIFIED | `TestIsDenseEFSharedWithContent_SymmetricUnwrap` at line 413; `TestDeleteCollectionFromCache_EmbeddedCollection` at line 439 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `client_local_embedded.go:GetCollection` | `collectionStateMu.Lock()` | Wide write lock around check-nil + build-EF + assign | WIRED | Line 529 locks before nil-check; line 582 unlocks after snapshot copy |
| `client_local_embedded.go:buildEmbeddedCollection` | `wrapEFCloseOnce(` | Close-once wrapping at collection creation | WIRED | Lines 868-869 wrap both EFs; idempotent since state already holds wrapped EFs from GetCollection |
| `close_logging.go:isDenseEFSharedWithContent` | `unwrapCloseOnceEF(efFromContent)` | Symmetric unwrapping of both dense and content EF | WIRED | Line 37 unwraps denseEF; line 42 unwraps efFromContent |
| `client_local.go:WithPersistentLogger` | `client_local_embedded.go:embeddedLocalClient.logger` | `cfg.logger` passed to `newEmbeddedLocalClient` | WIRED | Line 534 sets `cfg.logger = l`; constructor at line 75 passes `logger: cfg.logger` |
| `client_local.go:WithPersistentLogger` | state client logger | `cfg.clientOptions = append(cfg.clientOptions, WithLogger(l))` | WIRED | Line 535 appends `WithLogger(l)` to clientOptions; confirmed by `TestWithPersistentLogger_PropagatesToStateClient` |
| `client_local_embedded.go:GetCollection` | `client.logger.Warn` | Structured logging of auto-wire errors when logger non-nil | WIRED | Lines 540-542 and 561-563 call `client.logger.Warn` with nil guard |
| `client_local_embedded.go:deleteCollectionState` | `client.logger.Error` | Structured logging of close cleanup errors when logger non-nil | WIRED | Lines 731-735 call `client.logger.Error` with nil guard |

### Data-Flow Trace (Level 4)

Not applicable — this phase modifies resource lifecycle management code (locks, close-once wrappers, cleanup loops), not data-rendering components. All changes are infrastructure/safety fixes, not data pipelines.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All 12 phase-19 tests pass with race detector | `go test -tags=basicv2 -race -run "TestEmbeddedGetCollection_ConcurrentAutoWire|..."` | `ok  github.com/amikos-tech/chroma-go/pkg/api/v2  2.131s` | PASS |
| Package builds without errors | `go build ./pkg/api/v2/...` | Exit 0, no output | PASS |
| Commits exist in git log | `git log --oneline` | 6abeeb5, c211fbd, dbcf252, b29e343 all present | PASS |

### Requirements Coverage

The plan declares requirement IDs `SC-01` through `SC-09`. These are phase-local success criteria defined in ROADMAP.md, not entries in REQUIREMENTS.md. Phase 19 is a hardening/bugfix phase; the REQUIREMENTS.md traceability table does not include a Phase 19 row because no new v1 requirements were introduced by this phase.

| SC ID | Description | Plan | Status | Evidence |
|-------|-------------|------|--------|----------|
| SC-01 | TOCTOU race: concurrent GetCollection auto-wires once | 19-01 | SATISFIED | Write lock at line 529; `TestEmbeddedGetCollection_ConcurrentAutoWire` asserts `buildCount == 1` |
| SC-02 | deleteCollectionState closes EFs before removing | 19-01 | SATISFIED | `deleteCollectionState` lines 725-740; `TestEmbeddedDeleteCollectionState_ClosesEFs` |
| SC-03 | embeddedLocalClient.Close() iterates collectionState | 19-01 | SATISFIED | `Close()` lines 666-705; `TestEmbeddedLocalClient_Close_CleansUpCollectionState` |
| SC-04 | localDeleteCollectionFromCache handles *embeddedCollection | 19-01 | SATISFIED | `client_http.go:816`; `TestDeleteCollectionFromCache_EmbeddedCollection` |
| SC-05 | buildEmbeddedCollection wraps EFs in close-once | 19-01 | SATISFIED | `client_local_embedded.go:868-869`; `TestEmbeddedBuildCollection_CloseOnceWrapping` |
| SC-06 | isDenseEFSharedWithContent unwraps both sides | 19-01 | SATISFIED | `close_logging.go:42`; `TestIsDenseEFSharedWithContent_SymmetricUnwrap` |
| SC-07 | Auto-wired EF only assigned on nil error | 19-01 | SATISFIED | All 3 HTTP sites and 2 embedded sites guarded; `TestEmbeddedGetCollection_BuildErrorGuard` |
| SC-08 | Structured logger receives auto-wire and close errors | 19-02 | SATISFIED | `WithPersistentLogger`, 4 logger-guarded callsites, 3 logger tests |
| SC-09 | All existing tests pass with no regressions | 19-01 | SATISFIED | `go test -tags=basicv2 -race -count=1 ./pkg/api/v2/...` passes per both summaries |

### Anti-Patterns Found

None. No TODO/FIXME/placeholder comments found in any of the five modified production files. No empty implementations or hardcoded return stubs. No return-null patterns in modified paths.

### Human Verification Required

None. All success criteria are mechanically verifiable through code inspection and test execution.

### Gaps Summary

No gaps. All 9 roadmap success criteria are satisfied:

- The TOCTOU race in `GetCollection` is eliminated by a write lock spanning the full check-nil + build + assign cycle, with an atomic counter test proving exactly-once factory invocation under concurrency.
- State cleanup on delete and close is implemented via copy-under-lock then close-outside-lock in both `deleteCollectionState` and `Close()`.
- `localDeleteCollectionFromCache` handles `*embeddedCollection` alongside `*CollectionImpl`.
- `buildEmbeddedCollection` wraps both EFs in close-once wrappers; since state already stores wrapped EFs, the wrapping is idempotent, ensuring all close paths share the same `sync.Once` instance.
- `isDenseEFSharedWithContent` now unwraps both sides symmetrically, making sharing detection correct when both EFs are close-once wrapped.
- Build error guards are applied to all auto-wire callsites: both in the embedded client and all three HTTP client paths (GetCollection x2, ListCollections x1).
- `WithPersistentLogger` injects a structured logger into the embedded client with nil-default (preserving stderr fallback), and dual-propagates to the state client via `WithLogger`. All four logging callsites are guarded with `if client.logger != nil`.
- All 12 new tests pass with `-race` flag; the package builds cleanly.

---

_Verified: 2026-04-06T14:00:00Z_
_Verifier: Claude (gsd-verifier)_
