---
phase: 26-twelve-labs-async-embedding
plan: 02
subsystem: embeddings
tags: [twelvelabs, embeddings, async, polling]

# Dependency graph
requires:
  - phase: 26-twelve-labs-async-embedding
    plan: 01
    provides: AsyncEmbedV2Request/AsyncAudioInput/AsyncVideoInput types, TaskCreateResponse, TaskResponse with FailureDetail json.RawMessage, doTaskPost, doTaskGet, polling fields + backoff defaults
provides:
  - pollTask polling loop with capped exponential backoff (time.NewTimer, not time.After)
  - Per-HTTP-call deadline = min(parent ctx, SDK maxWait) bounds blocked doTaskGet calls (D-09 hard bound)
  - Distinct error messages for ctx.Canceled, ctx.DeadlineExceeded, and SDK maxWait expiry (D-20); errors.Is still unwraps to stdlib sentinels via pkg/errors
  - Terminal status=failed uses raw server body from TaskResponse.FailureDetail → chttp.SanitizeErrorBody (D-17)
  - D-16 default branch rejects unknown status values with descriptive error
  - contentToAsyncRequest rejects audioOpt="fused" on the async path per RESEARCH F-02 / A5 — no silent drop, no silent map
  - createTaskAndPoll orchestrates build → POST → optional early-ready short-circuit → poll → extract
  - embedContent routes audio/video through async path when asyncPollingEnabled=true; text/image and flag-off callers unchanged
affects: [26-03-option-and-config, 26-04-tests]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-call context.WithDeadline wrapping derived from min(parentCtxDeadline, sdkMaxWaitDeadline) so a blocked HTTP request cannot outlive the SDK-side bound"
    - "Two-source deadline reconciliation — when a derived-deadline error fires, inspect sdkMaxWaitDeadline vs parent ctx to translate back into the distinct caller-facing error (D-20)"
    - "Separate file for async-specific code (twelvelabs_async.go) to keep Plan 03's option/config churn on twelvelabs.go conflict-free in the same wave"

key-files:
  created:
    - pkg/embeddings/twelvelabs/twelvelabs_async.go
  modified:
    - pkg/embeddings/twelvelabs/content.go

key-decisions:
  - "Renamed the nextBackoff cap parameter to backoffCap to avoid shadowing the Go builtin cap; semantics unchanged"
  - "Applied //nolint:unused annotations to the async symbols in Task 1 (mirroring Plan 01's precedent) and removed them in Task 2 once content.go routing made them reachable. Zero out-and-back churn — Task 2 is a two-line delete of pragmas plus the routing switch"
  - "Kept the fused-on-async rejection as an explicit error at contentToAsyncRequest rather than silently mapping it to 'audio' or 'transcription'; silent mapping would violate F-02 and produce results callers did not request"
  - "Used two distinct error message substrings ('async polling canceled' vs 'async polling deadline exceeded' vs 'async polling maxWait %s exceeded') so callers can introspect via substring or errors.Is without collapsing the three timeout sources"

patterns-established:
  - "Polling loops use a time.NewTimer + timer.Stop() on ctx branch instead of time.After (Pitfall 2)"
  - "SDK-side maxWait lives as a time.Time deadline variable, never as context.WithTimeout(ctx, maxWait) — keeps it distinguishable from caller ctx deadlines (Pitfall 3)"
  - "Failure reasons sanitize the raw server body (json.RawMessage captured on success parse) rather than re-marshaling a known-fields struct subset"

requirements-completed: [TLA-01, TLA-02, TLA-03]

# Metrics
duration: ~4min
completed: 2026-04-14
---

# Phase 26 Plan 02: Async Polling Loop and Modality Routing Summary

**Wired the Twelve Labs async polling loop and modality-based routing: audio/video + `asyncPollingEnabled=true` now flows through POST /tasks → poll GET /tasks/{id} → extract embedding, with three distinguishable timeout sources, per-HTTP-call deadline bounding, and raw-body failure sanitization. Text/image and non-opt-in callers see zero behavior change.**

## Performance

- **Duration:** ~4 min
- **Started:** 2026-04-14T09:29:56Z
- **Completed:** 2026-04-14T09:33:37Z
- **Tasks:** 2
- **Files created:** 1
- **Files modified:** 1

## Accomplishments

- New file `pkg/embeddings/twelvelabs/twelvelabs_async.go` lands `contentToAsyncRequest`, `pollTask`, `nextBackoff`, `createTaskAndPoll`, and `buildEmbeddingFromData` — all deviation-rule compliant with D-09, D-11, D-14, D-16, D-17, D-20 and RESEARCH Pitfall 2/3/4/5 + F-02.
- `pollTask` uses `time.NewTimer` with explicit `timer.Stop()` on the ctx branch (Pitfall 2) and clamps the per-iteration wait to the remaining `sdkMaxWaitDeadline` so the final timer fires at the deadline, not past it.
- Per-HTTP-call deadline = `min(parentCtxDeadline, sdkMaxWaitDeadline)` means a blocked `doTaskGet` cannot outlive `maxWait` — D-09 is a hard bound, not a between-polls best-effort check.
- Three timeout sources surface with distinct wording: `async polling canceled` (ctx.Canceled), `async polling deadline exceeded` (ctx.DeadlineExceeded), `async polling maxWait %s exceeded` (SDK-side). All three still unwrap to the correct stdlib sentinel via pkg/errors.
- Terminal `status=failed` sanitizes `resp.FailureDetail` (the raw server body preserved in Plan 01) through `chttp.SanitizeErrorBody` — D-17 compliance, no re-marshaled subset.
- `contentToAsyncRequest` rejects `audioOpt="fused"` with an explicit validation error pointing callers at `WithAsyncPolling` disabling for fused-audio calls (F-02 / A5).
- `embedContent` in `content.go` now branches on `asyncPollingEnabled && len(content.Parts)==1 && (Modality in {Audio, Video})` → `createTaskAndPoll`; every other case preserves the existing `contentToRequest → doPost → embeddingFromResponse` sequence verbatim.
- Existing `TestTwelveLabs*` suite exits 0 — flag-off is the default, and no test sets `asyncPollingEnabled=true` yet (Plan 04 will).

## Task Commits

1. **Task 1: Implement pollTask, contentToAsyncRequest, and createTaskAndPoll** — `ef31d06` (feat)
2. **Task 2: Route audio/video through async path when asyncPollingEnabled** — `d6e0222` (feat)

## Files Created/Modified

- `pkg/embeddings/twelvelabs/twelvelabs_async.go` — new file; async request builder, polling loop, orchestrator, and embedding extractor
- `pkg/embeddings/twelvelabs/content.go` — `embedContent` now dispatches audio/video to async path when the flag is on

## Decisions Made

- Kept the per-HTTP-call deadline derivation inline inside the loop rather than hoisting into a helper — the logic is ~5 lines and hoisting would require passing `sdkMaxWaitDeadline` and `ctx` across a boundary, obscuring the invariant that `callDeadline ≤ sdkMaxWaitDeadline` always.
- Chose error translation *after* `doTaskGet` returns rather than wrapping the call in a goroutine-with-cancel pattern — simpler, preserves existing HTTP client timeouts, and keeps `errors.Is(err, context.DeadlineExceeded)` as the source of truth for which source fired.
- Placed the fused-rejection inside `contentToAsyncRequest` (not at `embedContent` dispatch) so that any callable that builds an async request — including future test fixtures — hits the same validation gate.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Shadowed Go builtin `cap` in nextBackoff parameter**
- **Found during:** Task 1 implementation (before first build)
- **Issue:** The plan's reference code used `cap time.Duration` as the `nextBackoff` parameter name, which shadows Go's builtin `cap(...)`. Lint would flag this under `predeclared` / `shadow` rules and it is a known footgun even when lint doesn't catch it.
- **Fix:** Renamed the parameter to `backoffCap`. Behavior unchanged; the comparison `next > backoffCap` is semantically identical.
- **Files modified:** `pkg/embeddings/twelvelabs/twelvelabs_async.go`
- **Committed in:** `ef31d06` (Task 1)

**2. [Rule 3 - Blocking] Transient `unused` lint on async symbols between Task 1 commit and Task 2 commit**
- **Found during:** Task 1 post-implementation `make lint`
- **Issue:** After Task 1 landed the async symbols without a consumer, `unused` fired on `contentToAsyncRequest`, `pollTask`, `nextBackoff`, `createTaskAndPoll`, and `buildEmbeddingFromData`. The plan's Task 1 acceptance requires `make lint` to exit 0.
- **Fix:** Added `//nolint:unused` with comments citing Task 2 as the consumer. Removed all five annotations in Task 2's edits once content.go routing made the symbols reachable. This mirrors Plan 26-01's `//nolint:unused` pattern for the same intra-plan scaffolding concern.
- **Files modified:** `pkg/embeddings/twelvelabs/twelvelabs_async.go`
- **Committed in:** `ef31d06` (add), `d6e0222` (remove)

---

**Total deviations:** 2 auto-fixed (both blocking)
**Impact on plan:** No scope change. The `cap` rename is a builtin-shadow defense; the nolint add/remove is a transient scaffolding concern that the plan itself anticipated via the sequential task split.

## Issues Encountered

- `PreToolUse:Edit` read-before-edit hook fires on every Edit operation even within a single session after the file has been read. Each Edit required re-reading a slice of the file; the edits themselves all succeeded.

## User Setup Required

None — no environment variables, no external service configuration. Plan 03 will add the `WithAsyncPolling(maxWait)` option that flips `asyncPollingEnabled` on.

## Next Phase Readiness

- **Plan 26-03 (option + config)** can now safely add `WithAsyncPolling(maxWait)`, wire the config round-trip for `async_polling_enabled` + `async_max_wait`, and the routing + polling loop will activate immediately for audio/video callers who opt in.
- **Plan 26-04 (tests)** can now mock the task endpoints, construct a client with `asyncPollingEnabled=true` + `asyncMaxWait=...` directly (or via Plan 03's option once landed), and exercise: happy path, ready-first-poll, failed with sanitized reason, maxWait expiry, ctx.Canceled vs ctx.DeadlineExceeded distinction, unknown-status rejection, fused-rejection, and the modality routing gate (text/image stay on sync even with the flag on).

## Verification Log

- `go build -tags=ef ./pkg/embeddings/twelvelabs/...` — exit 0
- `make lint` — 0 issues
- `go test -tags=ef -count=1 -run TestTwelveLabs ./pkg/embeddings/twelvelabs/...` — ok (flag-off default preserved)
- Task 1 grep acceptance (15 assertions): all pass
- Task 2 grep acceptance (3 assertions): all pass
- No `time.After(` in the polling loop
- No `context.WithTimeout(ctx, maxWait)` at loop level (Pitfall 3 compliance)
- `context.WithDeadline(ctx, callDeadline)` present per-call (D-09 hard bound)
- `resp.FailureDetail` used on failed branch (D-17 authenticity)

## Self-Check: PASSED

- File `pkg/embeddings/twelvelabs/twelvelabs_async.go` exists and contains `contentToAsyncRequest`, `pollTask`, `nextBackoff`, `createTaskAndPoll`, `buildEmbeddingFromData`.
- File `pkg/embeddings/twelvelabs/content.go` modified `embedContent` contains `asyncPollingEnabled`, `ModalityAudio, embeddings.ModalityVideo`, `createTaskAndPoll(ctx, content)`.
- Commit `ef31d06` present in `git log`.
- Commit `d6e0222` present in `git log`.

---
*Phase: 26-twelve-labs-async-embedding*
*Completed: 2026-04-14*
