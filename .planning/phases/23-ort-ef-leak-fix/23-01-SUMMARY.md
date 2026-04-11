---
phase: 23-ort-ef-leak-fix
plan: 01
subsystem: api
tags: [go, v2, embedded, ort, lifecycle]
requires: []
provides:
  - closes SDK-owned temporary default ORT dense embedding functions on embedded existing-collection create
  - preserves Phase 20 existing-state EF precedence while surfacing cleanup failures synchronously
  - adds focused basicv2 regressions for cleanup success, cleanup failure, and new-collection ownership
affects: [embedded, create-collection, default-ef, lifecycle]
tech-stack:
  added: []
  patterns: [per-op default EF factory seam, tracked SDK-owned EF provenance, focused basicv2 lifecycle regressions]
key-files:
  created: []
  modified:
    - pkg/api/v2/client.go
    - pkg/api/v2/client_local_embedded.go
    - pkg/api/v2/client_local_embedded_test.go
key-decisions:
  - "Used a per-op defaultDenseEFFactory seam instead of a package-global override so basicv2 tests can inject a mock default EF without parallel-test races."
  - "Tracked the current SDK-owned default dense EF instance directly and only cleaned it up when the embedded existing-path still held that exact instance."
patterns-established:
  - "Embedded existing-collection cleanup gates compare the live runtime dense EF against tracked SDK-owned provenance before closing."
  - "Default EF lifecycle bug fixes stay narrow: per-op seams in production, colocated basicv2 regressions, and synchronous error returns on owned cleanup paths."
requirements-completed: [EFL-01]
duration: 3m
completed: 2026-04-11
---

# Phase 23 Plan 01: ORT EF Leak Fix Summary

**Embedded `CreateCollection(..., WithIfNotExistsCreate())` now closes the temporary SDK-owned default ORT EF on the existing-collection path without disturbing stored collection state.**

## Performance

- **Duration:** 3m
- **Started:** 2026-04-11T12:42:24+03:00
- **Completed:** 2026-04-11T12:45:23+03:00
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Added a per-op `defaultDenseEFFactory` seam plus `sdkOwnedDefaultDenseEF` tracking on `CreateCollectionOp` so tests can inject a temporary default EF without global mutable state.
- Closed the tracked SDK-owned default EF on the embedded existing-collection path and returned the exact wrapped cleanup error when `Close()` fails.
- Added focused `basicv2` regressions proving cleanup on existing collections, cleanup-error propagation, and ownership transfer on the new-collection path.

## Task Commits

Each task was committed atomically:

1. **Task 1: Replace the global seam draft with per-op default-EF provenance and focused lifecycle regressions** - `2b98280` (test), `8c6904c` (feat), `5358dcf` (chore)
2. **Task 2: Run full V2 and lint verification without widening the phase** - verification rerun during phase execution; no additional source commit required

## Files Created/Modified
- `pkg/api/v2/client.go` - adds the per-op default dense EF factory seam and tracks the current SDK-owned default EF instance.
- `pkg/api/v2/client_local_embedded.go` - closes the tracked SDK-owned default EF only on the embedded existing-collection path before discarding overrides.
- `pkg/api/v2/client_local_embedded_test.go` - pins cleanup success, cleanup failure, and new-collection ownership behavior in the standard `basicv2` suite.

## Decisions Made
- Kept the fix narrow in the embedded existing-path instead of deferring default EF creation or refactoring shared create-flow semantics.
- Returned cleanup failures synchronously with `error closing default embedding function for existing collection` rather than logging and continuing.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The initial `gsd-executor` handoff stalled without producing artifacts, so phase bookkeeping was completed manually against the already-landed Phase 23 commits and fresh verification output.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 23 satisfies `EFL-01` and keeps the existing embedded collection state authoritative on idempotent create calls.
- Phase 24 can now build on the explicit SDK-owned EF provenance without revisiting the existing-collection leak path.

## Verification Evidence

- `go test -tags=basicv2 -run 'TestEmbeddedLocalClientCreateCollection_IfNotExistsExistingDoesNotOverrideState|TestEmbeddedCreateCollection_DefaultORT.*' ./pkg/api/v2/...` -> passed (`ok github.com/amikos-tech/chroma-go/pkg/api/v2`).
- `make test` -> passed (`DONE 1732 tests, 7 skipped`).
- `make lint` -> passed (`0 issues.`).

## Self-Check: PASSED

- Verified `.planning/phases/23-ort-ef-leak-fix/23-01-SUMMARY.md` exists on disk.
- Verified task commits `2b98280`, `8c6904c`, and `5358dcf` exist in git history.
