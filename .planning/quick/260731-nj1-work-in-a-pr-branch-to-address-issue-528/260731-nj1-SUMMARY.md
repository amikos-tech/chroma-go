---
phase: quick-260731-nj1
plan: "01"
status: complete
subsystem: api
tags: [go, json, rank, recursion, depth-guard]

requires: []
provides:
  - "Bounded recursive Rank JSON marshaling at MaxExpressionDepth"
  - "Exact accepted and rejected depth-boundary regression coverage"
affects: [v2-rank-expressions, v2-search]

tech-stack:
  added: []
  patterns:
    - "Private depth-aware dispatch for recursive serializers with public zero-depth entry points"

key-files:
  created: []
  modified:
    - pkg/api/v2/rank.go
    - pkg/api/v2/rank_test.go

key-decisions:
  - "Kept Rank unchanged and used a private depth-aware interface so caller-defined Rank implementations retain the existing MarshalJSON fallback"
  - "Counted the public root as depth zero and rejected only depth values greater than MaxExpressionDepth"

requirements-completed: [GH-528]

duration: 5 min
completed: 2026-07-31
---

# Quick Task 260731-nj1: Rank Marshal Depth Guard Summary

**Rank JSON marshaling now stops recursive built-in expressions beyond the shared depth limit while preserving valid output and custom Rank compatibility.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-07-31T14:09:18Z
- **Completed:** 2026-07-31T14:13:55Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments

- Added zero-based depth tracking across all ten recursive built-in Rank serializers, including generated RRF expressions.
- Preserved the exported Rank interface, nil-rank handling, validation, valid JSON shapes, and caller-defined Rank fallback.
- Replaced loose deep-chain coverage with exact success and bounded-error checks at both sides of MaxExpressionDepth.

## TDD Cycle

- **RED:** `TestRankMarshalExpressionDepthGuard` failed because the over-limit expression returned no error.
- **GREEN:** Private depth-aware dispatch rejects depth above MaxExpressionDepth and passes the focused and full tagged suites.
- **REFACTOR:** No separate refactor was needed; the minimal implementation follows the existing serializer bodies.

## Task Commits

1. **RED: Add rank marshal depth boundary coverage** - `f6c4ddf` (test)
2. **GREEN: Bound recursive rank marshaling** - `304446b` (fix)

The summary and planning state remain uncommitted as required by the quick-task orchestrator.

## Files Created/Modified

- `pkg/api/v2/rank.go` - Adds the depth guard and propagates depth through recursive built-in serializers.
- `pkg/api/v2/rank_test.go` - Tests successful marshaling at the maximum and a no-panic error one level beyond it.

## Decisions Made

- Used an unexported depth-aware interface so no public API or caller implementation requirement changed.
- Routed RRF's generated expression through the same guarded helper rather than restarting at depth zero.
- Kept `ErrNilRank` precedence by checking nil ranks before checking depth.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Verification

- `go test -tags=basicv2 ./pkg/api/v2 -run '^TestRankMarshalExpressionDepthGuard$' -count=1 -timeout=30s` - passed.
- `go test -tags=basicv2 ./pkg/api/v2/... -count=1 -timeout=120s` - passed.
- `make lint` - passed with 0 issues.
- `git diff --check` - passed.
- Scope inspection confirmed changes are limited to `pkg/api/v2/rank.go` and `pkg/api/v2/rank_test.go`.

## User Setup Required

None - no external service configuration or dependency changes.

## Next Phase Readiness

- Ready for the orchestrator to integrate the two TDD commits into the final PR branch.
- No blockers or deferred issues.

## Self-Check: PASSED

- Both modified files and this summary exist.
- RED commit `f6c4ddf` and GREEN commit `304446b` exist in git history.
- `STATE.md` and the plan remain unchanged; only this summary is uncommitted.

---
*Quick task: 260731-nj1*
*Completed: 2026-07-31*
