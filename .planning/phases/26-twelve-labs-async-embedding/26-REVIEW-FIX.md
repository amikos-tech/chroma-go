---
phase: 26-twelve-labs-async-embedding
fixed_at: 2026-04-14T00:00:00Z
review_path: .planning/phases/26-twelve-labs-async-embedding/26-REVIEW.md
iteration: 1
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 26: Code Review Fix Report

**Fixed at:** 2026-04-14
**Source review:** .planning/phases/26-twelve-labs-async-embedding/26-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 2 (Critical=0, Warning=2)
- Fixed: 2
- Skipped: 0

Info-level findings (IN-01 .. IN-04) were intentionally out of scope per the
`critical_warning` fix scope.

## Fixed Issues

### WR-01: Task-create call is not bounded by `maxWait`

**Files modified:** `pkg/embeddings/twelvelabs/twelvelabs_async.go`, `pkg/embeddings/twelvelabs/option.go`
**Commit:** 598f823
**Applied fix:**
- In `createTaskAndPoll`, wrap `e.doTaskPost` in a derived context bounded by
  `min(parent ctx deadline, sdk maxWait deadline)` — same pattern used per-call
  inside `pollTask`. The `cancel()` is invoked unconditionally after the call.
- On error, translate `context.DeadlineExceeded` into the distinct SDK message
  `Twelve Labs async task create maxWait %s exceeded` when `sdkMaxWaitDeadline`
  has elapsed, and into `Twelve Labs async task create deadline exceeded` when
  the parent ctx fired first. `context.Canceled` translates to a distinct
  cancel message. Other errors retain the original `failed to create Twelve
  Labs async task` wrap to preserve the prior surface for non-deadline
  failures.
- Updated the `WithAsyncPolling` godoc to state explicitly that `maxWait` is a
  hard upper bound on the **whole async operation** (create + polling), not
  just polling, matching the new behavior and the D-09 design note.

### WR-02: No regression test for `doTaskPost` non-2xx error path

**Files modified:** `pkg/embeddings/twelvelabs/twelvelabs_test.go`
**Commit:** 7322514
**Applied fix:** Added three regression tests modeled on the existing sync
`doPost` error-path coverage:
- `TestTwelveLabsAsyncTaskCreateError` — structured `{message,code}` 4xx body;
  asserts the wrap text `task create error` and message contents survive.
- `TestTwelveLabsAsyncTaskCreateErrorSanitizesStructuredMessage` — long
  structured message body; asserts the `[truncated]` suffix appears and the
  full long body does not.
- `TestTwelveLabsAsyncTaskCreateErrorRawFallback` — non-JSON 5xx body;
  asserts the `unexpected status` raw-fallback branch fires with truncation.

All three pass under `go test -tags=ef`.

---

_Fixed: 2026-04-14_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
