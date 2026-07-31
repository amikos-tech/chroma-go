---
phase: quick-260731-pb1
plan: "01"
status: complete
subsystem: api
tags: [go, json, rank, recursion, depth-guard]

requires:
  - phase: quick-260731-nj1
    provides: "Initial bounded recursive marshaling for built-in Rank expressions"
provides:
  - "Exhaustive depth-aware marshaling contract for every Rank implementation"
  - "Boundary regressions for all ten composite Rank serializers"
  - "RRF depth failures with RrfRank-specific error context"
affects: [v2-rank-expressions, v2-search]

tech-stack:
  added: []
  patterns:
    - "Rank implementations must provide package-controlled depth-aware JSON marshaling"
    - "Leaf depth adapters delegate to existing non-recursive encoders"

key-files:
  created: []
  modified:
    - pkg/api/v2/rank.go
    - pkg/api/v2/rank_test.go
    - pkg/api/v2/search_test.go

key-decisions:
  - "Made depth-aware marshaling mandatory on Rank so no implementation can bypass or reset the shared recursion guard"
  - "Delegated leaf depth adapters to existing MarshalJSON methods to preserve encoding and validation behavior"
  - "Kept RRF's generated expression at depth+1 and wrapped only its final marshaling error"

requirements-completed: [GH-528]

duration: 7 min
completed: 2026-07-31
---

# Quick Task 260731-pb1: Rank Depth Contract Hardening Summary

**Every Rank serializer now participates in one shared depth limit, with complete composite boundary coverage and contextual RRF failures.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-07-31T15:22:49Z
- **Completed:** 2026-07-31T15:29:22Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Added depth-aware marshaling directly to Rank, removed fallback dispatch, and adapted all production leaves plus the package-local map-backed test double.
- Exercised Sum, Sub, Mul, Div, Abs, Exp, Log, Max, Min, and RRF through public construction paths at their exact accepted and rejected child-depth boundaries.
- Preserved RRF's hidden four-level expansion accounting while adding cannot marshal RrfRank context to final generated-expression failures.

## TDD Cycle

- **RED:** The expanded focused test failed only for rrf/rejects_next_child_depth: the shared depth error did not yet contain cannot marshal RrfRank.
- **GREEN:** Wrapping the error from marshalRank(result, depth+1) made all composite boundaries pass while preserving successful RRF bytes.
- **REFACTOR:** No separate refactor was needed; the production change is one guarded call plus contextual error propagation.

## Task Commits

1. **Task 1: Make depth-aware marshaling an exhaustive Rank contract** - 62a6688 (fix)
2. **Task 2 RED: Cover every composite Rank depth boundary** - ec756f1 (test)
3. **Task 2 GREEN: Retain context for RRF depth errors** - c4a1592 (feat)

The task commits were integrated with a squash merge as required by the repository workflow.

## Files Created/Modified

- pkg/api/v2/rank.go - Makes depth-aware marshaling exhaustive, delegates leaf encoders, and contextualizes final RRF depth failures.
- pkg/api/v2/rank_test.go - Covers both sides of every composite boundary, including RRF inner depths 96 and 97.
- pkg/api/v2/search_test.go - Adapts the package-local map-backed Rank test double to the exhaustive contract.

## Decisions Made

- Kept nil-rank validation ahead of the depth check, retaining ErrNilRank precedence.
- Kept zero-based root accounting and MaxExpressionDepth == 100.
- Used existing leaf encoders rather than duplicating JSON or validation logic.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. The expected RED failure confirmed that the new RRF context assertion detected the intended missing behavior.

## Verification

- Task 1 focused test command passed (ok, 0.553s).
- RED focused depth test failed as expected only because the RRF depth error lacked cannot marshal RrfRank.
- Final focused depth test passed (ok, 0.366s).
- Full tagged V2 package suite passed (ok, 22.914s).
- make lint passed with 0 issues.
- git diff --check and the base-to-HEAD diff check passed with no whitespace errors.
- Base-to-HEAD scope inspection found only pkg/api/v2/rank.go, pkg/api/v2/rank_test.go, and pkg/api/v2/search_test.go.
- pkg/api/v2/collection_http.go, go.mod, and go.sum are unchanged; no external issue was created.

## User Setup Required

None - no dependency or external service changes.

## Next Phase Readiness

- Expressions deeper than 100 now return an error even if the former fallback path allowed them to marshal.
- cloneRank still performs independent recursive traversal; it remains a follow-up concern only, with no code TODO or external issue created in this task.
- Ready for final planning metadata to be committed.

## Self-Check: PASSED

- All three modified code files and this summary exist.
- Code commits 62a6688, ec756f1, and c4a1592 were created in order from the expected base and squash-integrated.
- STATE.md, ROADMAP.md, REQUIREMENTS.md, the plan, and collection_http.go remain unchanged by the executor.

---
*Quick task: 260731-pb1*
*Completed: 2026-07-31*
