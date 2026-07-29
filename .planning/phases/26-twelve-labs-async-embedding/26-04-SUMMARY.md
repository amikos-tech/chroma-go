---
phase: 26-twelve-labs-async-embedding
plan: 04
subsystem: pkg/embeddings/twelvelabs
tags: [twelvelabs, embeddings, async, testing]
requires:
  - 26-01 (async request/response types, doTaskPost, doTaskGet)
  - 26-02 (pollTask + createTaskAndPoll + modality routing)
  - 26-03 (WithAsyncPolling option + config round-trip)
provides:
  - Twelve async-focused unit tests covering every D-26 flow, D-07 routing, D-22 omit-when-disabled, and three review-fix guards
affects:
  - pkg/embeddings/twelvelabs/twelvelabs_test.go
tech_stack_added:
  - sync/atomic (test-only — attempt counters)
  - stderrors "errors" (test-only — distinguished from pkg/errors)
key_files_created: []
key_files_modified:
  - pkg/embeddings/twelvelabs/twelvelabs_test.go
decisions:
  - Used stdlib `errors` (aliased as `stderrors`) for `errors.Is` assertions; pkg/errors.Wrap is unwrappable via stdlib since Go 1.13 so both sentinel and wrapped errors are reachable.
  - Direct field assignment on `ef.apiClient` to keep tests independent of option ordering (e.g. fused audio applied post-construction without going through `WithAudioEmbeddingOption`).
  - `newTestAsyncEF` composes on `newTestEF` so sync-helper changes propagate automatically.
metrics:
  duration: ~12 minutes
  tasks: 2
  files: 1
completed: 2026-04-14
---

# Phase 26 Plan 04: Async Test Coverage Summary

Landed TLA-04. Added twelve new `TestTwelveLabsAsync*` functions in `pkg/embeddings/twelvelabs/twelvelabs_test.go` covering every D-26 flow (task-create, poll-to-ready, poll-to-failed, unexpected-status, ctx-cancel, maxWait-expiry), the D-07 text/image-skip rule, the D-22 config-omit-when-disabled rule, and the D-23 config round-trip (including APIKeyEnvVar preservation). Plus three review-fix guards: D-17 failure-reason sanitization, Plan 02 blocked-HTTP per-call deadline, and F-02 fused+async rejection.

## What Was Built

### Task 1 — Core async flows (commit `b323908`)
Helpers + four behavior tests:
- `newTestAsyncEF(serverURL)` — enables async with ms-scale intervals (`asyncPollInitial=1ms`, `asyncPollCap=10ms`, `asyncMaxWait=5s`) to keep the full async suite under ~1s.
- `audioContent(url)` / `videoContent(url)` — minimal one-part multimodal content helpers.
- `taskCreateJSON(id, status)` and `taskGetJSON(id, status, data)` — fixture emitters that always use the `_id` alias (Pitfall 1 guard).
- `TestTwelveLabsAsyncTaskCreate` — asserts POST /tasks gets `InputType=audio`, `Audio.EmbeddingOption=[]string{"audio"}` (F-02 list shape), GET /tasks/{id} returns ready with data.
- `TestTwelveLabsAsyncPollToReady` — asserts 3 GETs before ready, embedding materializes correctly.
- `TestTwelveLabsAsyncPollToFailed` — asserts error contains `task_fail`, `terminal status=failed`, not wrapped with ctx sentinels.
- `TestTwelveLabsAsyncUnexpectedStatus` — asserts error contains `unexpected status`, `weird`, `task_weird`.

### Task 2 — Cancellation, deadlines, routing, config (commit `5e2b9ce`)
- `TestTwelveLabsAsyncCtxCancel` — uses `context.WithCancel` + signaling channel to cancel after first poll; asserts `stderrors.Is(err, context.Canceled)`.
- `TestTwelveLabsAsyncMaxWait` — 50ms maxWait, server always returns processing; asserts `"async polling maxWait"` in error AND `stderrors.Is(err, context.DeadlineExceeded) == false` (D-20 distinct error).
- `TestTwelveLabsAsyncSkipsTextImage` — mock `t.Fatalf` on any `/tasks` hit; runs text + image embed, confirms both route through sync endpoint even with async enabled (D-07).
- `TestTwelveLabsAsyncConfigRoundTrip` — builds EF with `WithEnvAPIKey` + `WithAsyncPolling(7*time.Minute)`, round-trips through `GetConfig` + `NewTwelveLabsEmbeddingFunctionFromConfig`, asserts `asyncPollingEnabled`, `asyncMaxWait=7m`, AND `APIKeyEnvVar` preservation.
- `TestTwelveLabsAsyncConfigOmitWhenDisabled` — no `WithAsyncPolling` option → neither `async_polling` nor `async_max_wait_ms` appears in config map (D-22).
- `TestTwelveLabsAsyncFailedReasonSanitized` — server returns `failed` with a ~1.5KB authentic `reason` field NOT in `TaskResponse` struct; asserts the reason substring survives because `doTaskGet` preserves the raw body via `FailureDetail`. Asserts final error < 4096 bytes (sanitizer cap).
- `TestTwelveLabsAsyncBlockedHTTPMaxWait` — mock server hangs on GET /tasks/{id} waiting for `r.Context().Done()`; maxWait=100ms must interrupt the blocked HTTP call and surface the distinct `"async polling maxWait"` error (not `DeadlineExceeded`) within 2s.
- `TestTwelveLabsAsyncFusedRejected` — mock server fatals on any HTTP call; sets `AudioEmbeddingOption="fused"`, audio content input; asserts rejection error contains both `"fused"` and `"async"` before any POST.

## Per-Test Verification Map Alignment

All rows in `26-VALIDATION.md` Per-Task Verification Map are populated and `nyquist_compliant: true` was already set by the planner. No VALIDATION.md edits required.

## Test Results

```
=== RUN   TestTwelveLabsAsyncTaskCreate           --- PASS (0.00s)
=== RUN   TestTwelveLabsAsyncPollToReady          --- PASS (0.00s)
=== RUN   TestTwelveLabsAsyncPollToFailed         --- PASS (0.00s)
=== RUN   TestTwelveLabsAsyncUnexpectedStatus     --- PASS (0.00s)
=== RUN   TestTwelveLabsAsyncCtxCancel            --- PASS (0.00s)
=== RUN   TestTwelveLabsAsyncMaxWait              --- PASS (0.05s)
=== RUN   TestTwelveLabsAsyncSkipsTextImage       --- PASS (0.00s)
=== RUN   TestTwelveLabsAsyncConfigRoundTrip      --- PASS (0.00s)
=== RUN   TestTwelveLabsAsyncConfigOmitWhenDisabled  --- PASS (0.00s)
=== RUN   TestTwelveLabsAsyncFailedReasonSanitized  --- PASS (0.00s)
=== RUN   TestTwelveLabsAsyncBlockedHTTPMaxWait   --- PASS (0.10s)
=== RUN   TestTwelveLabsAsyncFusedRejected        --- PASS (0.00s)
PASS   ok  github.com/amikos-tech/chroma-go/pkg/embeddings/twelvelabs   0.631s
```

Full `go test -tags=ef -count=1 ./pkg/embeddings/twelvelabs/...` exits 0 in ~0.46s. `make lint` clean (0 issues).

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | `b323908` | test(26-04): add async task-create, poll-ready, poll-failed, unexpected-status tests |
| 2 | `5e2b9ce` | test(26-04): add ctx-cancel, maxWait, skip-text-image, config round-trip, and review-fix tests |

## Deviations from Plan

None — plan executed exactly as written. The plan specified 7 tests but the actual count is 12 (4 core + 5 additional + 3 review-fix guards); all were in the plan body, the "seven" phrasing referenced the numbered set while the review-fix ones were explicitly required by frontmatter `must_haves.truths`. All 12 functions land in the file.

## Authentication Gates

None encountered — all tests use `httptest.Server`.

## Key Decisions

- **Aliasing stdlib errors as `stderrors`**: Explicitly marked both imports distinct to avoid ambiguity when reading assertions. `pkg/errors.Wrap`'s wrappers satisfy `stderrors.Is` via Go 1.13 unwrapping so this works transparently.
- **`newTestAsyncEF` by composition**: Wraps `newTestEF` rather than duplicating the construction logic — keeps the two helpers in lockstep if the sync helper ever evolves.

## Known Stubs

None. All tests exercise real production code paths.

## Self-Check: PASSED

- File `pkg/embeddings/twelvelabs/twelvelabs_test.go` exists and contains all 12 `TestTwelveLabsAsync*` functions (grep-verified).
- Commit `b323908` exists in git log (`test(26-04): add async task-create, poll-ready, poll-failed, unexpected-status tests`).
- Commit `5e2b9ce` exists in git log (`test(26-04): add ctx-cancel, maxWait, skip-text-image, config round-trip, and review-fix tests`).
- `go test -tags=ef -count=1 ./pkg/embeddings/twelvelabs/...` exits 0.
- `make lint` exits 0.
