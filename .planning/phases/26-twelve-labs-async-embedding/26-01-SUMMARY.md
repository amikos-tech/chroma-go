---
phase: 26-twelve-labs-async-embedding
plan: 01
subsystem: embeddings
tags: [twelvelabs, embeddings, async, http-client, tasks-endpoint]

# Dependency graph
requires:
  - phase: 25-error-body-truncation
    provides: chttp.SanitizeErrorBody and chttp.ReadLimitedBody conventions reused verbatim on new task helpers
provides:
  - AsyncEmbedV2Request / AsyncAudioInput / AsyncVideoInput types with `embedding_option: []string` list shape (RESEARCH F-02)
  - TaskCreateResponse and TaskResponse types with `_id` Mongo-style JSON alias (RESEARCH Pitfall 1)
  - TaskResponse.FailureDetail (json.RawMessage, `json:"-"`) for Plan 02 D-17 failure-reason sanitization
  - Unexported async polling fields on TwelveLabsClient: asyncPollingEnabled, asyncMaxWait, asyncPollInitial, asyncPollMultiplier, asyncPollCap
  - applyDefaults populates backoff defaults: initial=2s, multiplier=1.5, cap=60s (D-11)
  - doTaskPost HTTP helper (POST {BaseAPI}/tasks) mirroring doPost headers + error sanitization
  - doTaskGet HTTP helper (GET {BaseAPI}/tasks/{id}) with url.PathEscape, empty-ID guard, and raw-body preservation
affects: [26-02-polling-loop, 26-03-option-and-config, 26-04-content-routing-and-tests]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Async request type distinct from sync EmbedV2Request to model endpoint shape differences (embedding_option list vs string)"
    - "Raw-body preservation on success responses via json.RawMessage FailureDetail field so downstream callers sanitize the authentic server reason instead of re-marshaling"
    - "//nolint:unused annotations on feature-foundation symbols consumed by later plans in the same wave"

key-files:
  created: []
  modified:
    - pkg/embeddings/twelvelabs/twelvelabs.go

key-decisions:
  - "Introduced AsyncEmbedV2Request as a distinct type from EmbedV2Request because the async tasks endpoint takes embedding_option as []string, not a bare string (RESEARCH F-02). Reusing AudioInput would encode the wrong shape."
  - "Used json:\"_id\" aliases on both TaskCreateResponse.ID and TaskResponse.ID so the Mongo-style server response decodes correctly; silent empty-ID bugs are a documented pitfall."
  - "Added TaskResponse.FailureDetail as json.RawMessage with json:\"-\" tag; doTaskGet populates it directly from the raw body bytes. This lets Plan 02 sanitize the server's verbatim failure reason instead of re-marshaling a struct subset (D-17)."
  - "doTaskGet guards empty taskID with a descriptive error instead of silently building /tasks/ URLs (defends against the _id alias footgun from Pitfall 1)."
  - "Applied polling backoff defaults in applyDefaults (initial=2s, multiplier=1.5, cap=60s) but intentionally left asyncPollingEnabled and asyncMaxWait at zero-value — per D-22, opt-in is driven only by WithAsyncPolling from Plan 03."
  - "Used //nolint:unused annotations on async fields and helpers rather than temporary stubs, because Plans 02/03 wire them in the same wave and removing them after the fact would churn this file."

patterns-established:
  - "Async endpoint variant types live alongside sync types in the same file, with comments that name the RESEARCH finding they encode"
  - "HTTP task helpers mirror doPost structure byte-for-byte on headers, ReadLimitedBody, and SanitizeErrorBody"

requirements-completed: [TLA-01, TLA-02, TLA-03]

# Metrics
duration: ~20min
completed: 2026-04-14
---

# Phase 26 Plan 01: Async Embedding Foundation Summary

**Async tasks-endpoint plumbing for Twelve Labs: dedicated request/response types with `_id` alias + `embedding_option: []string` shape, client polling fields with backoff defaults, and two HTTP helpers (doTaskPost, doTaskGet) that mirror existing Phase 25 sanitization conventions.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-04-14T09:06:00Z
- **Completed:** 2026-04-14T09:26:00Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments

- New async request/response types (`AsyncEmbedV2Request`, `AsyncAudioInput`, `AsyncVideoInput`, `TaskCreateResponse`, `TaskResponse`) encode the exact tasks-endpoint contract — correct `_id` Mongo alias and `embedding_option` as a list.
- `TwelveLabsClient` gained five unexported async polling fields plus backoff defaults in `applyDefaults` without touching any existing defaults or public surface.
- `doTaskPost` and `doTaskGet` HTTP helpers land with the same headers, `ReadLimitedBody`, and `SanitizeErrorBody` posture as the existing `doPost`, plus URL-escaped task IDs and an empty-ID defensive guard.
- `TaskResponse.FailureDetail` preserves raw response bytes so Plan 02's polling loop can surface the authentic server-provided failure reason rather than a re-marshaled subset.

## Task Commits

1. **Task 1: Async types + polling fields + defaults** — `818c569` (feat)
2. **Task 2: doTaskPost and doTaskGet HTTP helpers** — `0566ec2` (feat)

## Files Created/Modified

- `pkg/embeddings/twelvelabs/twelvelabs.go` — added async request/response types, polling fields on TwelveLabsClient, backoff defaults in applyDefaults, and two HTTP helpers (doTaskPost, doTaskGet)

## Decisions Made

- Preserved raw response bytes for `TaskResponse.FailureDetail` by copying `respData` via `append(json.RawMessage(nil), respData...)` rather than aliasing; the underlying `ReadLimitedBody` buffer is not guaranteed stable across request lifetimes and `json.RawMessage` needs stable bytes for later sanitization.
- Left `asyncPollingEnabled` and `asyncMaxWait` at zero-value defaults. Per D-22 the option flag and max-wait are driven entirely by `WithAsyncPolling` (Plan 03). Defaulting them here would silently turn async on for callers who never opted in.
- Kept `AsyncEmbedV2Request` separate from `EmbedV2Request` instead of overloading with a polymorphic `embedding_option` shape. The two endpoints are distinct contracts; sharing a type would leak the async footgun into sync callers.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Lint failures on unused async symbols**
- **Found during:** Task 2 (post-implementation `make lint`)
- **Issue:** golangci-lint's `unused` check flagged the four symbols (`asyncPollingEnabled`, `asyncMaxWait`, `doTaskPost`, `doTaskGet`) that are foundation scaffolding for Plans 26-02 and 26-03 in the same wave. The plan's acceptance criteria required `make lint` to exit 0.
- **Fix:** Added `//nolint:unused` pragmas with comments citing the consumer plans. Chose this over removing the symbols because the symbols are load-bearing for the Wave 2 plans and churning them out-and-back-in is pure waste.
- **Files modified:** `pkg/embeddings/twelvelabs/twelvelabs.go`
- **Verification:** `make lint` exits 0; `go build -tags=ef` and `go test -tags=ef -run TestTwelveLabs` both pass.
- **Committed in:** `0566ec2` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** No scope change. The nolint annotations are a transient scaffolding concern that Plans 02/03 will eliminate naturally when they start consuming the symbols — the annotations can be removed at that point.

## Issues Encountered

- During Task 1 and Task 2 edits, the PreToolUse read-before-edit hook fired repeatedly even though the file had been read in-session. Working around this required re-reading slices of the file between edits. The edits themselves succeeded as expected.

## User Setup Required

None — no external service configuration or env-var changes in this plan.

## Next Phase Readiness

- Plan 02 (polling loop) can now consume `doTaskPost` / `doTaskGet` and the polling fields without any further type plumbing.
- Plan 03 (option + config) can now wire `WithAsyncPolling(maxWait)` to set `asyncPollingEnabled` and `asyncMaxWait` on the client, and can round-trip the config keys.
- Plans 02 and 03 can run in parallel in Wave 2 because they touch disjoint surfaces: Plan 02 adds a polling method on `TwelveLabsEmbeddingFunction`; Plan 03 adds the option and updates `GetConfig` / `NewTwelveLabsEmbeddingFunctionFromConfig`.

## Verification Log

- `go build -tags=ef ./pkg/embeddings/twelvelabs/...` — exit 0
- `go vet -tags=ef ./pkg/embeddings/twelvelabs/...` — exit 0
- `make lint` — 0 issues
- `go test -tags=ef -count=1 -run TestTwelveLabs ./pkg/embeddings/twelvelabs/...` — ok
- Sanity run: `json.Unmarshal({"_id":"task_abc",...}, &TaskCreateResponse{})` populates `ID="task_abc"` and `Status="processing"`; `json.Marshal(&AsyncAudioInput{EmbeddingOption: []string{"audio"}})` produces `"embedding_option":["audio"]` as expected.

## Self-Check: PASSED

- File `pkg/embeddings/twelvelabs/twelvelabs.go` exists and contains `AsyncEmbedV2Request`, `TaskCreateResponse`, `TaskResponse`, `doTaskPost`, `doTaskGet`, `asyncPollingEnabled`, `asyncMaxWait`, `asyncPollInitial`, `asyncPollMultiplier`, `asyncPollCap`, `FailureDetail json.RawMessage`.
- Commit `818c569` present in `git log`.
- Commit `0566ec2` present in `git log`.

---
*Phase: 26-twelve-labs-async-embedding*
*Completed: 2026-04-14*
