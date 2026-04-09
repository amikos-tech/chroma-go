---
phase: 21-rrfrank-arithmetic-fix
reviewed: 2026-04-09T09:45:00Z
depth: standard
files_reviewed: 2
files_reviewed_list:
  - pkg/api/v2/rank.go
  - pkg/api/v2/rank_test.go
findings:
  critical: 0
  warning: 0
  info: 1
  total: 1
status: issues_found
---

# Phase 21: Code Review Report

**Reviewed:** 2026-04-09T09:45:00Z
**Depth:** standard
**Files Reviewed:** 2
**Status:** issues_found

## Summary

Phase 21 fixes all 10 `RrfRank` arithmetic methods (`Multiply`, `Sub`, `Add`, `Div`, `Negate`, `Abs`, `Exp`, `Log`, `Max`, `Min`) that were previously returning the receiver `r` (no-op stubs). Each method now constructs the correct expression tree node, exactly matching the corresponding `KnnRank` implementation. I verified all 10 methods line-by-line against `KnnRank` and they are structurally identical (modulo receiver variable name).

The fix is correct and complete. The test coverage is thorough: every method is tested individually with exact JSON assertions, a chaining test is included (`Add` then `Log`), receiver immutability is verified, and identity checks confirm a new `Rank` object is returned rather than the receiver.

One pre-existing info-level issue was found in non-changed code.

## Info

### IN-01: LogRank.Log() incorrectly returns receiver (pre-existing, not in diff)

**File:** `pkg/api/v2/rank.go:606-607`
**Issue:** `LogRank.Log()` returns `l` (the receiver), which means `val.Log().Log()` evaluates to `log(x)` instead of `log(log(x))`. Unlike `AbsRank.Abs()` (line 478-479) where `abs(abs(x)) == abs(x)` is mathematically correct, `log(log(x)) != log(x)`. This is not part of the phase 21 changes but was noticed during review. It follows the same no-op pattern that was just fixed in `RrfRank`.
**Fix:** Return a new `LogRank` wrapping the receiver, consistent with how `ExpRank.Exp()` works (line 542-544):
```go
func (l *LogRank) Log() Rank {
    return &LogRank{rank: l}
}
```

---

_Reviewed: 2026-04-09T09:45:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
