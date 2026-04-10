---
phase: 22-withgroupby-validation
plan: 01
subsystem: api
tags: [go, v2, search, groupby, validation]
requires: []
provides:
  - fail-fast nil validation for WithGroupBy during search request construction
  - exact regression coverage for direct and NewSearchRequest WithGroupBy(nil) paths
affects: [search, groupby, validation]
tech-stack:
  added: []
  patterns: [fail-fast option validation, colocated v2 regression tests]
key-files:
  created: []
  modified:
    - pkg/api/v2/search.go
    - pkg/api/v2/groupby_test.go
key-decisions:
  - "Kept nil handling in groupByOption.ApplyToSearchRequest with a direct errors.New(\"groupBy cannot be nil\") message."
  - "Pinned both the direct option path and the composed NewSearchRequest path without changing non-nil GroupBy validation."
patterns-established:
  - "Explicit nil option input is rejected at ApplyToSearchRequest before request mutation or append."
  - "Nil-contract bug fixes in pkg/api/v2 keep valid and invalid non-nil coverage in the same colocated test file."
requirements-completed: [GRP-01]
duration: 5m
completed: 2026-04-10
---

# Phase 22 Plan 01: WithGroupBy Validation Summary

**WithGroupBy(nil) now fails fast with a stable validation error before search requests are appended or sent.**

## Performance

- **Duration:** 5m
- **Started:** 2026-04-10T04:35:20Z
- **Completed:** 2026-04-10T04:40:17Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Replaced the silent nil no-op in `groupByOption.ApplyToSearchRequest` with the exact error `groupBy cannot be nil`.
- Added regression coverage for both direct `WithGroupBy(nil)` application and `NewSearchRequest(..., WithGroupBy(nil))` fail-before-append behavior.
- Preserved existing valid and invalid non-nil `GroupBy` behavior and cleared the focused, full, and lint verification commands.

## Task Commits

Each task was committed atomically:

1. **Task 1: Replace nil-no-op behavior with fail-fast validation and pin it with direct + request-construction tests** - `40db705` (test), `508c19a` (feat)
2. **Task 2: Run broader V2 regressions and lint** - `3525b36` (test)

## Files Created/Modified
- `pkg/api/v2/search.go` - rejects explicit nil `WithGroupBy` input before request mutation.
- `pkg/api/v2/groupby_test.go` - asserts the exact nil error and the no-append regression while retaining non-nil coverage.

## Decisions Made
- Kept the nil error as a direct string in `pkg/api/v2/search.go` instead of introducing a new exported sentinel, matching existing repo validation style.
- Left `pkg/api/v2/groupby.go` unchanged so non-nil validation continues to flow through `(*GroupBy).Validate()` exactly as before.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 22's `GRP-01` contract is implemented and regression-covered.
- No blockers were found for subsequent v0.4.2 bug-fix phases.

## Verification Evidence

- `go test -tags=basicv2 -run 'TestWithGroupBy|TestSearchRequestWithGroupBy' ./pkg/api/v2/...` -> failed in RED, then passed after the implementation change.
- `make test` -> passed (`DONE 1729 tests, 7 skipped in 30.110s`).
- `make lint` -> passed (`0 issues.`).

## Self-Check: PASSED

- Verified `.planning/phases/22-withgroupby-validation/22-01-SUMMARY.md` exists on disk.
- Verified task commits `40db705`, `508c19a`, and `3525b36` exist in git history.
