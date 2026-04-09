---
phase: 21-rrfrank-arithmetic-fix
plan: 01
subsystem: rank-expressions
tags: [bugfix, rank, rrf, arithmetic, expression-tree]
dependency_graph:
  requires: []
  provides: [rrf-arithmetic-parity]
  affects: [pkg/api/v2/rank.go]
tech_stack:
  added: []
  patterns: [expression-tree-delegation]
key_files:
  created: []
  modified:
    - pkg/api/v2/rank.go
    - pkg/api/v2/rank_test.go
decisions:
  - Follow identical pattern to KnnRank/ValRank for all 10 arithmetic methods
metrics:
  duration: 2m
  completed: "2026-04-09T06:27:19Z"
  tasks_completed: 2
  tasks_total: 2
---

# Phase 21 Plan 01: RrfRank Arithmetic Fix Summary

Fixed all 10 RrfRank arithmetic methods to build expression trees instead of silently returning the receiver, with comprehensive TDD test coverage using exact JSON assertions.

## What Changed

All 10 RrfRank arithmetic/math methods (Multiply, Sub, Add, Div, Negate, Abs, Exp, Log, Max, Min) were no-ops that returned the receiver `r`, making any rank expression composition with RRF silently produce incorrect results. Each method now delegates to the same expression node types used by KnnRank and ValRank (MulRank, SubRank, SumRank, DivRank, AbsRank, ExpRank, LogRank, MaxRank, MinRank).

## Task Results

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 (RED) | Add RrfRank arithmetic tests | a40c363 | pkg/api/v2/rank_test.go |
| 1 (GREEN) | Fix RrfRank arithmetic methods | 9313179 | pkg/api/v2/rank.go |
| 2 | Full test suite + lint verification | (no changes) | -- |

## Verification Results

- `go test -tags=basicv2 -run TestRrfRankArithmetic ./pkg/api/v2/...` -- PASS (11/11 subtests)
- `go test -tags=basicv2 -run "Test(Val|Arithmetic|Math|Division|Rrf|Rank|Operand|Knn|Complex|Unknown)" ./pkg/api/v2/...` -- PASS (all existing tests, zero regressions)
- `make lint` -- 0 issues
- No `return r` remains in RrfRank arithmetic methods
- No `// no-op` comment remains

## Test Coverage

TestRrfRankArithmetic includes 11 subtests:
1. **10 arithmetic method tests** -- each asserts exact JSON structure via `require.JSONEq`, verifies the result is a different pointer from the receiver, and confirms receiver immutability by comparing MarshalJSON output before/after
2. **1 chained composition test** -- `rrf.Add(FloatOperand(1)).Log()` produces correct nested JSON, proving returned Ranks are composable

## Deviations from Plan

None -- plan executed exactly as written.

## Known Stubs

None.

## Self-Check: PASSED

- FOUND: pkg/api/v2/rank.go
- FOUND: pkg/api/v2/rank_test.go
- FOUND: .planning/phases/21-rrfrank-arithmetic-fix/21-01-SUMMARY.md
- FOUND: commit a40c363 (RED: failing tests)
- FOUND: commit 9313179 (GREEN: fix methods)
