---
phase: 11-fork-double-close-bug
plan: 01
subsystem: api
tags: [fork, close, embedding-function, sync, concurrency]

requires:
  - phase: 03-registry-and-config-integration
    provides: contentEmbeddingFunction field and Close() sharing detection in CollectionImpl
provides:
  - closeOnceEF and closeOnceContentEF wrappers with sync.Once idempotent close
  - ownsEF ownership flag on CollectionImpl and embeddedCollection
  - Fork() produces non-owning collections with close-once wrapped EFs
  - Close() gates EF teardown on ownership flag
affects: [11-fork-double-close-bug, 13-collection-forkcount]

tech-stack:
  added: []
  patterns: [ownership-flag-gating, close-once-wrapper, sync.Once-with-atomic-closed-guard]

key-files:
  created:
    - pkg/api/v2/ef_close_once.go
  modified:
    - pkg/api/v2/collection_http.go
    - pkg/api/v2/client_local_embedded.go
    - pkg/api/v2/client_http.go

key-decisions:
  - "Use ownsEF bool flag as primary ownership guard with close-once wrapper as defence-in-depth"
  - "Forked collections get close-once wrapped copies of parent EFs, preventing double-close even if ownsEF check is bypassed"

patterns-established:
  - "Ownership flag pattern: ownsEF bool on collection structs gates Close() teardown"
  - "Close-once wrapper pattern: sync.Once + atomic.Bool guard for idempotent EF Close"

requirements-completed: [FORK-01, FORK-02, FORK-03, FORK-04]

duration: 3min
completed: 2026-03-26
---

# Phase 11 Plan 01: Fork Double-Close Bug Fix Summary

**ownsEF ownership flag and close-once EF wrappers prevent double-close of shared embedding functions in Fork() collections**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-26T17:49:10Z
- **Completed:** 2026-03-26T17:52:51Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Created close-once EF wrappers implementing EmbeddingFunction, ContentEmbeddingFunction, io.Closer, and EmbeddingFunctionUnwrapper interfaces
- Added ownsEF ownership flag to both HTTP and embedded collection structs
- Fork() now produces non-owning collections with close-once wrapped EFs
- Close() gates EF teardown on ownership, preventing double-close when client.Close() iterates cached collections

## Task Commits

Each task was committed atomically:

1. **Task 1: Create close-once EF wrappers** - `bf0c5ab` (feat)
2. **Task 2: Add ownsEF flag and gate Close()** - `e68041a` (fix)

## Files Created/Modified
- `pkg/api/v2/ef_close_once.go` - Close-once wrappers for EmbeddingFunction and ContentEmbeddingFunction with sync.Once and atomic closed guard
- `pkg/api/v2/collection_http.go` - Added ownsEF field to CollectionImpl, Fork() wraps EFs and sets ownsEF=false, Close() gates on ownsEF
- `pkg/api/v2/client_local_embedded.go` - Added ownsEF field to embeddedCollection, buildEmbeddedCollection sets ownsEF=true, Fork() overrides to false with wrapped EF, Close() gates on ownsEF
- `pkg/api/v2/client_http.go` - CreateCollection and GetCollection set ownsEF=true on CollectionImpl literals

## Decisions Made
- Use ownsEF bool flag as primary ownership guard with close-once wrapper as defence-in-depth
- Forked collections get close-once wrapped copies of parent EFs, preventing double-close even if ownsEF check is bypassed

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Plan 02 (tests) can proceed to verify Fork + Close lifecycle with the ownership and close-once patterns established here
- Phase 13 (Collection.ForkCount) will benefit from the fork bug fix

## Self-Check: PASSED

All artifacts verified:
- All 4 source files exist
- Both task commits (bf0c5ab, e68041a) found in git log
- closeOnceEF, closeOnceContentEF structs present
- ownsEF field present in collection_http.go, client_local_embedded.go, client_http.go
- Build passes, lint passes, 1582 tests pass

---
*Phase: 11-fork-double-close-bug*
*Completed: 2026-03-26*
