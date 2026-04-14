---
phase: 26-twelve-labs-async-embedding
reviewed: 2026-04-14T00:00:00Z
depth: standard
files_reviewed: 5
files_reviewed_list:
  - pkg/embeddings/twelvelabs/content.go
  - pkg/embeddings/twelvelabs/option.go
  - pkg/embeddings/twelvelabs/twelvelabs.go
  - pkg/embeddings/twelvelabs/twelvelabs_async.go
  - pkg/embeddings/twelvelabs/twelvelabs_test.go
findings:
  critical: 0
  warning: 1
  info: 0
  total: 1
status: issues_found
---

# Phase 26: Code Review Report

**Reviewed:** 2026-04-14
**Depth:** standard
**Files Reviewed:** 5
**Status:** issues_found

## Summary

Phase 26's async Twelve Labs path is generally well-structured: audio/video routing is isolated, the create and poll requests are both bounded by derived deadlines, and the new tests cover the intended happy-path and cancellation behaviors. The one material regression is in the timeout translation logic: transport-level `context.DeadlineExceeded` errors from caller-supplied HTTP clients are treated as if they always came from either the SDK `maxWait` deadline or the parent context, which is not true.

## Warnings

### WR-01: HTTP client timeouts can erase the error or panic the async path

**Files:** `pkg/embeddings/twelvelabs/twelvelabs_async.go:77-82`, `pkg/embeddings/twelvelabs/twelvelabs_async.go:158-163`

**Issue:** Both timeout-translation branches assume every `context.DeadlineExceeded` returned from `doTaskGet` or `doTaskPost` came from either:

1. the SDK's derived `maxWait` deadline, or
2. the caller's parent `ctx`.

That assumption breaks as soon as the caller uses the supported `WithHTTPClient(...)` option with a client-level timeout or transport deadline. In that case `Client.Do(...)` returns an error that satisfies `errors.Is(err, context.DeadlineExceeded)`, but `ctx.Err()` is still `nil` and the SDK `maxWait` deadline has not elapsed yet.

The current code then executes `errors.Wrap(ctx.Err(), "...")`. With `github.com/pkg/errors`, `errors.Wrap(nil, "...")` returns `nil`, so the timeout is silently erased.

**Observed consequences (reproduced locally against the current code):**

- Slow `POST /tasks` with `WithHTTPClient(&http.Client{Timeout: 50 * time.Millisecond})` returns `emb=nil, err=nil`.
- Slow `GET /tasks/{id}` with the same client returns `(nil, nil)` from `pollTask`, after which `createTaskAndPoll` panics on `buildEmbeddingFromData(final.Data)` because `final == nil`.

**Why this matters:** `WithHTTPClient` is part of the public API, so this is a realistic integration path, not a synthetic edge case. Any consumer that adds a client timeout for safety can get silent success on task creation or a nil-pointer panic during polling.

**Fix:** Only wrap `ctx.Err()` when it is actually non-nil. Otherwise preserve the original transport error or return a dedicated timeout error for client-level deadlines. For example:

```go
if errors.Is(err, context.DeadlineExceeded) {
	if !time.Now().Before(sdkMaxWaitDeadline) {
		return nil, errors.Errorf("Twelve Labs task [%s] async polling maxWait %s exceeded", taskID, maxWait)
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, errors.Wrap(ctxErr, "Twelve Labs async polling deadline exceeded")
	}
	return nil, errors.Wrap(err, "Twelve Labs async polling request timed out")
}
```

Apply the same pattern to the task-create branch, then add two regressions:

- delayed `POST /tasks` with a short custom HTTP client timeout must return a non-nil error
- delayed `GET /tasks/{id}` with a short custom HTTP client timeout must return a non-nil error and must not panic

---

_Reviewed: 2026-04-14_
_Reviewer: Codex_
_Depth: standard_
