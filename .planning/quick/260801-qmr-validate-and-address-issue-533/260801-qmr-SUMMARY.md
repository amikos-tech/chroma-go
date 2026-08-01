---
phase: quick-260801-qmr
plan: 01
status: complete
subsystem: api
tags: [go, api-v2, validation, json, recursion-safety]
requires:
  - phase: prior-rank-depth-guard
    provides: MaxExpressionDepth boundary convention
provides:
  - Bounded validation for nested built-in Where compound clauses
  - Regression coverage for validation and JSON depth boundaries
affects: [where-clauses, search-filter-validation]
tech-stack:
  added: []
  patterns:
    - Reuse shared expression-depth limits with a private recursive validator
key-files:
  created: []
  modified:
    - pkg/api/v2/where.go
    - pkg/api/v2/where_test.go
key-decisions:
  - "Use the package-level MaxExpressionDepth with root depth zero to match rank expressions."
  - "Validate nested built-in compound clauses through a private helper while preserving external WhereClause validation behavior."
patterns-established:
  - "Recursive built-in Where clauses carry depth explicitly; public Validate starts at depth zero."
requirements-completed: [GH-533]
duration: 5min
completed: 2026-08-01
---

# Quick Task 260801-qmr Summary

**Nested `$and` and `$or` Where clauses now stop at the shared expression-depth boundary before validation or JSON encoding can recurse indefinitely.**

## Performance

- **Duration:** 5 min
- **Tasks:** 1/1
- **Files modified:** 2

## Accomplishments

- Added a root-zero, private depth-aware validation path for built-in compound Where clauses.
- Reused `MaxExpressionDepth` and return a stable error for the first over-limit child.
- Covered the permitted boundary, rejection from `Validate`, rejection from `json.Marshal`, and unchanged shallow JSON output.

## Task Commit

1. **Task 1: Guard recursive compound Where validation at the shared expression limit** - `bb3ef57` (squash commit)

## Files Created/Modified

- `pkg/api/v2/where.go` - Delegates compound child validation through a depth-aware private helper.
- `pkg/api/v2/where_test.go` - Adds depth-boundary and JSON-marshalling regressions.

## Verification

- `go test -tags=basicv2 ./pkg/api/v2 -run '^TestWhereClauseExpressionDepthGuard$' -count=1 -timeout=60s` — passed.
- `go test -tags=basicv2 ./pkg/api/v2 -count=1 -timeout=60s` — passed.
- `git diff --check` — passed.

## Decisions Made

- Root compound expressions start at depth `0`; the final leaf at depth `MaxExpressionDepth` remains valid, matching rank's boundary semantics.
- Only exact built-in `*WhereClauseWhereClauses` children use the private recursive path; other `WhereClause` implementations keep their existing `Validate` behavior.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

The depth guard was squash-merged into the PR branch and is ready for verification.

## Self-Check: PASSED

- Implementation files exist and commit `bb3ef57` is present.

---
*Quick task: 260801-qmr*
*Completed: 2026-08-01*
