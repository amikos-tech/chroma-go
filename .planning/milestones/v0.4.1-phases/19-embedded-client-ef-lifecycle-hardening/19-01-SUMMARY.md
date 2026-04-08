---
phase: 19-embedded-client-ef-lifecycle-hardening
plan: 01
subsystem: embedded-client-ef-lifecycle
tags: [concurrency, close-once, TOCTOU, resource-leak, embedded-client]
dependency_graph:
  requires: []
  provides: [hardened-ef-lifecycle, toctou-fix, close-once-canonical-state]
  affects: [pkg/api/v2/client_local_embedded.go, pkg/api/v2/close_logging.go, pkg/api/v2/client_http.go]
tech_stack:
  added: []
  patterns: [copy-under-lock-then-close, canonical-wrapper-in-state, symmetric-unwrap]
key_files:
  created: []
  modified:
    - pkg/api/v2/client_local_embedded.go
    - pkg/api/v2/close_logging.go
    - pkg/api/v2/client_http.go
    - pkg/api/v2/client_local_embedded_test.go
    - pkg/api/v2/close_review_test.go
decisions:
  - Store close-once WRAPPED EFs in collectionState as canonical wrapper location
  - Write lock spans only check-nil+build+assign, not buildEmbeddedCollection (avoids deadlock)
  - Delete/close removes map entry under lock, closes EFs outside lock via shared sync.Once
  - Build error guards applied to all three auto-wire paths (embedded, HTTP GetCollection, HTTP ListCollections)
metrics:
  duration: 13min
  completed: "2026-04-06T09:47:00Z"
  tasks: 2
  files: 5
---

# Phase 19 Plan 01: Core EF Lifecycle Fixes Summary

Write-locked TOCTOU elimination, canonical close-once wrapper storage in collectionState, symmetric unwrapping, delete/close ordering with copy-under-lock pattern, build error guards across all auto-wire paths.

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | 6abeeb5 | fix(19-01): TOCTOU race, close-once wrapping, symmetric unwrap, delete/close cleanup, build error guards |
| 2 | c211fbd | test(19-01): 9 tests covering all fixed paths with race detector |

## Changes Made

### Task 1: Eight Production Code Fixes

**Fix 1 - TOCTOU race (D-01/D-02/D-09):** Replaced RLock+conditional-build with write-locked check-nil+build-EF+assign in embedded `GetCollection`. Lock spans only the atomic check-and-set cycle, NOT `buildEmbeddedCollection` (which calls `upsertCollectionState` and would deadlock).

**Fix 2 - Close-once wrapping in buildEmbeddedCollection (D-03):** `buildEmbeddedCollection` now wraps both EFs in close-once at collection creation. Idempotent since state already stores wrapped EFs from Fix 1.

**Fix 3 - deleteCollectionState closes EFs (D-06):** Removes map entry under lock, then closes copied state outside lock. Close-once wrappers shared with live `embeddedCollection` instances prevent double-close via `sync.Once`.

**Fix 4 - embeddedLocalClient.Close() (D-05):** Copies all state entries under lock, clears map, releases lock, then closes all EFs outside lock before shutting down runtime. Continues closing all entries even after individual failures using `errors.Join`.

**Fix 5 - localDeleteCollectionFromCache (D-07):** Added `*embeddedCollection` type assertion in the HTTP client's cache cleanup, alongside the existing `*CollectionImpl` handler.

**Fix 6 - Symmetric unwrapping (D-08):** `isDenseEFSharedWithContent` now unwraps both sides via `unwrapCloseOnceEF`, so identity comparison works when both dense and content EFs are wrapped in close-once.

**Fix 7 - HTTP GetCollection build error guard (D-09):** Added `else` guards so failed `BuildContentEFFromConfig` and `BuildEmbeddingFunctionFromConfig` calls do not assign error results to `contentEF`/`ef`.

**Fix 8 - HTTP ListCollections build error guard (SC-07):** Same pattern applied to `ListCollections` auto-wire at `BuildEmbeddingFunctionFromConfig`.

### Task 2: Nine New Tests

All tests pass with `-race` flag.

1. **TestEmbeddedGetCollection_ConcurrentAutoWire** - 10 goroutines behind barrier; atomic counter proves factory invoked exactly once
2. **TestEmbeddedDeleteCollectionState_ClosesEFs** - Verifies EFs closed and map entry removed
3. **TestEmbeddedLocalClient_Close_CleansUpCollectionState** - 2 collection entries, one failing close; all 4 EFs closed, errors aggregated
4. **TestEmbeddedBuildCollection_CloseOnceWrapping** - Verifies `*closeOnceEF` and `*closeOnceContentEF` type assertions
5. **TestEmbeddedGetCollection_BuildErrorGuard** - Nonexistent provider; nil EFs in state and collection; subsequent explicit EF works
6. **TestEmbeddedDeleteAndCloseShareWrapper** - State delete + collection close share same `sync.Once`; close count == 1
7. **TestEmbeddedDeleteAndCloseRace** - Concurrent delete + close behind barrier; no panic or deadlock
8. **TestIsDenseEFSharedWithContent_SymmetricUnwrap** - Wrapped shared, wrapped different, unwrapped shared, unwrapped different
9. **TestDeleteCollectionFromCache_EmbeddedCollection** - `*embeddedCollection` type switch closes EFs and removes cache entry

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Existing test assertions broken by close-once wrapping**
- **Found during:** Task 1 verification
- **Issue:** Seven existing tests used `require.Same` for pointer identity comparison, but EFs are now wrapped in close-once wrappers at state and collection level
- **Fix:** Updated assertions to use `unwrapCloseOnceEF()` / `unwrapCloseOnceContentEF()` before pointer comparison
- **Files modified:** pkg/api/v2/client_local_embedded_test.go
- **Commit:** 6abeeb5

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| Store close-once WRAPPED EFs in collectionState | Canonical wrapper location ensures all close paths (delete, client.Close, collection.Close) share the same sync.Once instance |
| Write lock spans only check+build+assign | buildEmbeddedCollection internally calls upsertCollectionState which would deadlock if called under the same lock |
| Delete removes map entry under lock, closes outside | Prevents concurrent GetCollection from seeing stale EFs while avoiding blocking during slow EF teardown |
| Build error guards on all 3 auto-wire paths | Consistency: embedded GetCollection, HTTP GetCollection, and HTTP ListCollections all guard against nil-from-error assignment |

## Verification

- `go build ./pkg/api/v2/...` - PASS
- `go test -tags=basicv2 -race -count=1 ./pkg/api/v2/...` - PASS (27s)
- `make lint` - PASS (0 issues)

## Self-Check: PASSED

- All 6 modified/created files exist on disk
- Both commits (6abeeb5, c211fbd) found in git log
