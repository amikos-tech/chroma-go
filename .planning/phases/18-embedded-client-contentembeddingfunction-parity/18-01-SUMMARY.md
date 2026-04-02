---
phase: 18-embedded-client-contentembeddingfunction-parity
plan: 01
subsystem: api
tags: [embedded-client, content-embedding, close-lifecycle, auto-wiring]

requires:
  - phase: 11-fork-double-close-bug
    provides: close-once wrappers and ownsEF flag pattern
  - phase: 03-registry-and-config-integration
    provides: BuildContentEFFromConfig and auto-wiring infrastructure
provides:
  - contentEmbeddingFunction field in embeddedCollectionState and embeddedCollection
  - GetCollection auto-wiring for contentEF in embedded client
  - Close() sharing detection mirroring HTTP path for embedded collections
affects: [18-02, embedded-client-tests]

tech-stack:
  added: []
  patterns: [state-aware auto-wiring that checks existing state before building new instances]

key-files:
  created: []
  modified: [pkg/api/v2/client_local_embedded.go]

key-decisions:
  - "Check existing state before auto-wiring to preserve prior EF pointer identity across GetCollection calls"
  - "Silent error discard for auto-wiring failures matching embedded client's minimal logging approach"

patterns-established:
  - "State-aware auto-wiring: embedded client checks collectionState before calling BuildContentEFFromConfig to avoid replacing existing EF instances with new ones from config"

requirements-completed: [SC-1, SC-2, SC-3, SC-5]

duration: 10min
completed: 2026-04-02
---

# Phase 18 Plan 01: Embedded Client contentEmbeddingFunction Parity Summary

**contentEmbeddingFunction wired into embedded collection structs, state management, GetCollection auto-wiring, and Close() sharing detection mirroring HTTP client path**

## Performance

- **Duration:** 10 min
- **Started:** 2026-04-02T11:04:25Z
- **Completed:** 2026-04-02T11:15:06Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- Added contentEmbeddingFunction field to both embeddedCollectionState and embeddedCollection structs
- Extended buildEmbeddedCollection to accept and wire contentEF parameter through state and struct
- Implemented GetCollection auto-wiring via BuildContentEFFromConfig with state-aware guard to preserve existing EF pointers
- Replaced single-EF Close() with dual-EF sharing detection using safeCloseEF and unwrapCloseOnceEF

## Task Commits

Each task was committed atomically:

1. **Task 1: Add contentEF field to structs and state management** - `783efcf` (feat)
2. **Task 2: Wire GetCollection auto-wiring and Close() sharing detection** - `c468a12` (feat)

## Files Created/Modified
- `pkg/api/v2/client_local_embedded.go` - Added contentEF to structs, state management, GetCollection auto-wiring, and Close() sharing detection

## Decisions Made
- Check existing state before auto-wiring: the embedded client preserves EF state across GetCollection calls via the collectionState map, so auto-wiring from config must only fire when state does not already hold an EF. This differs from the HTTP client which always creates fresh instances.
- Silent error discard for auto-wiring failures: the embedded client has no logger, so BuildContentEFFromConfig and BuildEmbeddingFunctionFromConfig errors are discarded via `_`, matching the embedded client's minimal approach.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] State-aware auto-wiring guard to preserve EF pointer identity**
- **Found during:** Task 2 (GetCollection auto-wiring)
- **Issue:** Direct translation of HTTP auto-wiring pattern overwrote existing EF state with new instances from BuildEmbeddingFunctionFromConfig, breaking pointer identity assertions in TestEmbeddedLocalClientGetOrCreateCollection_ExistingWithoutEFPreservesLocalState and ExistingWithEFUpdatesLocalState
- **Fix:** Added RLock check of existing collectionState before auto-wiring; only call BuildContentEFFromConfig / BuildEmbeddingFunctionFromConfig when state does not already hold the respective EF
- **Files modified:** pkg/api/v2/client_local_embedded.go
- **Verification:** Both previously failing tests pass; full suite 1651 tests pass
- **Committed in:** c468a12

---

**Total deviations:** 1 auto-fixed (1 bug fix)
**Impact on plan:** Essential correctness fix for embedded client state preservation semantics. No scope creep.

## Issues Encountered
None beyond the auto-fixed deviation above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- ContentEF wiring is complete in production code
- Plan 02 (tests) can verify all scenarios: explicit option, auto-wiring, Close() sharing detection, and state preservation

## Self-Check: PASSED

- FOUND: pkg/api/v2/client_local_embedded.go
- FOUND: 18-01-SUMMARY.md
- FOUND: commit 783efcf (Task 1)
- FOUND: commit c468a12 (Task 2)

---
*Phase: 18-embedded-client-contentembeddingfunction-parity*
*Completed: 2026-04-02*
