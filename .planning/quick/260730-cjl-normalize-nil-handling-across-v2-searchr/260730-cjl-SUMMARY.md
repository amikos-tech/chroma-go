---
phase: 260730-cjl
plan: 01
subsystem: api
tags: [v2-search, validation, nil-handling, functional-options]

# Dependency graph
requires:
  - phase: 22-withgroupby-validation
    provides: "WithGroupBy(nil) fail-fast precedent (errors.New pattern) that this task extends to WithSearchFilter/WithRank"
provides:
  - "WithSearchFilter(nil) and WithRank(nil) both reject with validation errors matching WithGroupBy(nil)"
  - "Documented, intentional divergence from Python/TS SDK nil semantics (doc comments + CHANGELOG)"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: ["Fail-fast nil validation on functional SearchRequestOption helpers (github.com/pkg/errors)"]

key-files:
  created: []
  modified: [pkg/api/v2/search.go, pkg/api/v2/search_test.go, CHANGELOG.md]

key-decisions:
  - "Matched WithGroupBy's exact error style: errors.New(\"<field> cannot be nil\") via github.com/pkg/errors, no new dependency"
  - "Kept WithSelect() with zero keys as a valid no-op (out of scope, locked by CONTEXT.md)"
  - "Documented the Python/TS divergence in both doc comments (go doc discoverable) and CHANGELOG"
  - "Inlined nil checks in each ApplyToSearchRequest rather than adding a shared helper (duplication is minor)"

patterns-established:
  - "SearchRequestOption nil-rejection: if o.field == nil { return errors.New(\"field cannot be nil\") } before assignment"

requirements-completed: [ISSUE-503]

# Metrics
duration: 15min
completed: 2026-07-30
---

# Quick Task 260730-cjl: Normalize nil-handling across V2 SearchRequestOption helpers Summary

**WithSearchFilter(nil) and WithRank(nil) now fail fast with validation errors, matching WithGroupBy(nil)'s existing Phase 22 contract, closing GH #503.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-07-30T06:53:00Z
- **Completed:** 2026-07-30T06:55:23Z
- **Tasks:** 2 completed
- **Files modified:** 3

## Accomplishments
- `searchFilterOption.ApplyToSearchRequest` and `rankOption.ApplyToSearchRequest` now reject nil with `"filter cannot be nil"` / `"rank cannot be nil"`, identical in style to the existing `groupByOption` check.
- `WithSearchFilter`, `WithRank`, and `WithGroupBy` doc comments now state the nil-rejection contract and the intentional, permanent divergence from Python/TS SDK nil semantics.
- New test coverage pins both the direct `ApplyToSearchRequest` path and the composed `NewSearchRequest` path for both new nil cases, mirroring `TestWithGroupBy`/`TestSearchRequestWithGroupBy`.
- `CHANGELOG.md` extended to a single `### Changed` bullet covering all three options plus the divergence note.

## Task Commits

Each task was committed atomically:

1. **Task 1: Reject nil in WithSearchFilter/WithRank and document the divergence** - `8b9c1b3` (fix)
2. **Task 2: Add nil-validation tests and extend the CHANGELOG entry** - `7c63fec` (test)

## Files Created/Modified
- `pkg/api/v2/search.go` - Added nil checks to `searchFilterOption.ApplyToSearchRequest` and `rankOption.ApplyToSearchRequest`; added divergence doc-comment addendum to `WithSearchFilter`, `WithRank`, `WithGroupBy`
- `pkg/api/v2/search_test.go` - Added nil-validation subtests to `TestSearchFilter` (direct + composed path); added new `TestWithRank` (happy path + nil direct + nil composed)
- `CHANGELOG.md` - Replaced the single `WithGroupBy(nil)` bullet with an extended entry covering all three options and the Python/TS divergence

## Decisions Made
- Matched `WithGroupBy`'s exact error style (`github.com/pkg/errors`, `"<field> cannot be nil"`) for consistency — no new import.
- `WithSelect()` with zero keys stays a valid no-op per the locked scope boundary in CONTEXT.md — not touched.
- Doc comments and CHANGELOG both document the divergence (not just one or the other), per RESEARCH.md guidance, so it's discoverable via `go doc` as well as release notes.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

All three `SearchRequestOption` helpers (`WithSearchFilter`, `WithRank`, `WithGroupBy`) now share a uniform, loud-failure contract for explicit nil input. No further follow-up work identified for this issue.

---
*Phase: 260730-cjl*
*Completed: 2026-07-30*

## Self-Check: PASSED

All modified files (`pkg/api/v2/search.go`, `pkg/api/v2/search_test.go`, `CHANGELOG.md`) confirmed present. Both task commits (`8b9c1b3`, `7c63fec`) confirmed in git log.
