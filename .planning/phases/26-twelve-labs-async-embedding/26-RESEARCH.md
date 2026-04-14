# Phase 26: Twelve Labs Async Embedding - Research

**Researched:** 2026-04-14
**Domain:** Twelve Labs embed-v2 async task API integration into an existing Go embedding provider
**Confidence:** HIGH

## Summary

CONTEXT.md locks 27 implementation decisions before research. This document validates those decisions against the live Fern-generated official Twelve Labs Python SDK (the authoritative source for the API contract) and against the existing `pkg/embeddings/twelvelabs` code. Almost every decision is confirmed; two material deltas require the planner's attention and one has been corrected against stale documentation URLs.

**Material flags for the planner (detail in "Flags for Planner" below):**

1. **Only TWO task endpoints exist, not three.** CONTEXT.md D-01 lists `POST /embed-v2/tasks`, `GET /embed-v2/tasks/{id}/status`, and `GET /embed-v2/tasks/{id}`. The official Python SDK only uses `POST /embed-v2/tasks` and `GET /embed-v2/tasks/{id}` — the retrieve endpoint already returns `status` plus `data` in a single response, so polling and retrieval collapse to one GET. A `/status` sub-path is not exposed in the SDK reference or the generated raw client.
2. **The async `AudioInputRequest` body shape differs from the sync body shape.** Sync `/embed-v2` uses `embedding_option: string` (single value). Async `/embed-v2/tasks` uses `embedding_option: list[string]` plus `embedding_scope: list[string]` plus `embedding_type: list[string]`. The provider's existing `AudioInput.EmbeddingOption string` field cannot be reused as-is for the task endpoint.
3. **CONTEXT.md doc URLs 404.** The `docs.twelvelabs.io/api-reference/embed-v2/*` paths in the Canonical References block return "Page Not Found". The correct path prefix is `docs.twelvelabs.io/v1.3/api-reference/create-embeddings-v2/*` (verified via the Python SDK's embedded doc links).

**Primary recommendation:** Implement two new internal helpers (`createTask` + `retrieveTask`), poll via the retrieve endpoint alone, and add a small async-specific request type for the tasks body. Otherwise, follow CONTEXT.md verbatim.

## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Two-endpoint model (Framing A). Sync `/embed-v2` usage is unchanged. The new code path calls `POST /embed-v2/tasks` to create a task, `GET /embed-v2/tasks/{id}/status` to poll, and `GET /embed-v2/tasks/{id}` to retrieve the final embedding. The roadmap phrase "sync endpoint returns an async task response" was a premise drift — verified against Twelve Labs docs and the Fern-generated official Python SDK; the sync endpoint always returns `EmbeddingSuccessResponse` and never a task response.
  > **Research flag:** the `/status` sub-endpoint does not exist in the Python SDK — see Flag F-01. The planner should collapse poll + retrieve into a single GET unless a deeper doc source contradicts the SDK.
- **D-02:** Existing `POST /v1.3/embed-v2` behavior is not modified. Callers who do not opt in see zero behavioral change.
- **D-03:** One new functional option: `WithAsyncPolling(maxWait time.Duration) Option`. Passing `0` selects the default `maxWait` of 30 minutes. Presence of this option is the sole trigger for the async code path.
- **D-04:** No other async-related options are exported in this phase.
- **D-05:** `EmbeddingFunction` and `ContentEmbeddingFunction` interface signatures are unchanged.
- **D-06:** `Collection.Add` / `Collection.Query` semantics are unchanged.
- **D-07:** When `WithAsyncPolling` is present, audio and video modalities route through the tasks endpoint; text and image always use sync.
- **D-08:** When `WithAsyncPolling` is absent, all modalities continue to use the sync endpoint.
- **D-09:** Total polling time is bounded by `min(ctx.Deadline(), maxWait)`.
- **D-10:** `ctx` cancellation mid-poll terminates the loop immediately and surfaces `context.Canceled`.
- **D-11:** Polling uses capped exponential backoff: `initial=2s`, `multiplier=1.5`, `cap=60s`, no jitter. Unexported fields on `TwelveLabsClient` with defaults in `applyDefaults`.
- **D-12:** No external polling/backoff dependency.
- **D-13:** No jitter in v1.
- **D-14:** Task status is determined by the `status` enum on the task retrieval response: `processing` → continue; `ready` → extract embedding; `failed` → terminate with error.
- **D-15:** HTTP status code (200 vs. 202) is NOT used as the async discriminator.
- **D-16:** Any unexpected `status` value produces a descriptive error.
- **D-17:** Terminal task failures surface as `errors.Errorf(...)` including task ID, status, and `chttp.SanitizeErrorBody`-truncated reason. No exported error type.
- **D-18:** HTTP/transport errors flow through the existing `errors.Wrap` / `errors.Errorf` pattern; error bodies sanitized via `chttp.SanitizeErrorBody`.
- **D-19:** `context.Canceled` and `context.DeadlineExceeded` propagate unchanged.
- **D-20:** `maxWait` expiration surfaces as a distinct error (not stdlib `context.DeadlineExceeded`).
- **D-21:** When `WithAsyncPolling` is enabled, `GetConfig` includes `async_polling: true` and `async_max_wait_ms: <int64 ms>`.
- **D-22:** When absent, both keys are omitted.
- **D-23:** `NewTwelveLabsEmbeddingFunctionFromConfig` reads these keys and reconstructs the option.
- **D-24:** Tests use `httptest.Server` with attempt counters; polling fields set directly in test construction to ms-scale values.
- **D-25:** No clock abstraction.
- **D-26:** Required test flows (TLA-04 coverage): task-create, poll-to-ready, poll-to-failed, ctx-cancellation, maxWait expiration, config round-trip.
- **D-27:** Tests live under the `ef` build tag; prefer extending `twelvelabs_test.go`.

### Claude's Discretion

None called out explicitly in CONTEXT.md — every meaningful design choice is locked. Claude's remaining discretion is naming of internal helpers, internal struct layouts, and fixture shape in tests.

### Deferred Ideas (OUT OF SCOPE)

- Callback-based async surface
- Path 2 / `CreateEmbeddingTask` + `TaskHandle` public API
- Path 3 / architecture ADR before code
- Parallel task dispatch / batch throughput
- Exposed polling knobs (`WithAsyncPollInterval`, `WithAsyncPollBackoff`)
- Structured `*TaskFailedError` exported type
- Jitter in backoff
- Clock abstraction for tests

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TLA-01 | Twelve Labs provider detects async task responses and enters a polling loop | The *trigger* is opt-in via `WithAsyncPolling` + audio/video modality, not a sync-endpoint response shape. See "Routing decision point" below. |
| TLA-02 | Async polling respects caller context for cancellation and timeout | Polling helper receives `ctx` and uses `select { case <-ctx.Done(): ... case <-time.After(...): ... }`. `maxWait` bound is tracked via `time.Now()` arithmetic, not a derived ctx (D-20 requires distinct maxWait error). |
| TLA-03 | Async polling handles terminal states (ready, failed) with appropriate error messages | The `EmbeddingTaskResponse.status` enum is `"processing" | "ready" | "failed"` (verified in Python SDK `embedding_task_response_status.py`). |
| TLA-04 | Tests cover async task creation, polling, completion, failure, and context cancellation | D-26 enumerates six flows; Validation Architecture section below maps each to a concrete httptest server + attempt-counter assertion. |

## Standard Stack

### Core (no new dependencies)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `net/http` | go 1.22+ | HTTP client for task endpoints | Already used by `doPost` in twelvelabs.go — extend, don't add |
| Go stdlib `context` | go 1.22+ | Ctx threading + cancellation | Standard Go pattern; `select { case <-ctx.Done(): }` loop |
| Go stdlib `time` | go 1.22+ | Polling timers, maxWait bound | `time.NewTimer` (not `time.After` inside loops to avoid leak on ctx cancel) |
| `github.com/pkg/errors` | existing | Error wrapping | Existing package convention; don't mix with stdlib `%w` in this package |
| `chttp` (`pkg/commons/http`) | internal | `SanitizeErrorBody`, `ReadLimitedBody`, `ChromaGoClientUserAgent` | Phase 25 convention — reuse verbatim |

### Supporting (existing code to extend)

| File | Touch type | Reason |
|------|------------|--------|
| `pkg/embeddings/twelvelabs/twelvelabs.go` | MODIFY | Add polling fields on `TwelveLabsClient`, defaults in `applyDefaults`, task-endpoint helpers, config round-trip |
| `pkg/embeddings/twelvelabs/content.go` | MODIFY | Branch `embedContent` on `asyncPollingEnabled && (audio|video)` to route to task helper |
| `pkg/embeddings/twelvelabs/option.go` | MODIFY | Add `WithAsyncPolling(maxWait time.Duration) Option` |
| `pkg/embeddings/twelvelabs/twelvelabs_test.go` | MODIFY | Extend `newTestEF` construction pattern for ms-scale polling; add six flows from D-26 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff | Decision |
|------------|-----------|----------|----------|
| Hand-rolled backoff loop | `github.com/cenkalti/backoff/v4` | Proven library, but adds dep | D-12 rejects — ~30 LoC stdlib is fine |
| Clock interface for tests | `github.com/benbjohnson/clock` | Deterministic fake-clock tests | D-25 rejects — direct field override on `TwelveLabsClient` is sufficient |
| Separate `/status` + `/retrieve` polling | One GET per poll that carries both | Two round-trips per poll vs one | **Research finding:** the SDK uses one GET — collapse to one. See F-01 below. |
| Structured `*TaskFailedError` export | Sentinel-style `errors.Is`-compatible | Cleaner caller code | D-17 rejects for minimal surface |

**Installation:** none — zero new deps (verified against CONTEXT.md D-12 and the existing `go.mod`; nothing added).

**Version verification:**
- Twelve Labs base path prefix: `https://api.twelvelabs.io/v1.3/embed-v2` (existing `defaultBaseAPI` in `twelvelabs.go:18`). [VERIFIED: codebase read]
- Python SDK HTTP paths: `embed-v2/tasks` (POST, GET list) and `embed-v2/tasks/{task_id}` (GET). [VERIFIED: raw_client.py L82, L198, L280]
- No registry version check needed; no new package.

## Architecture Patterns

### Recommended File Layout

No new files required. Prefer extending existing files:

```
pkg/embeddings/twelvelabs/
├── twelvelabs.go         # TwelveLabsClient + polling fields + doTaskPost/doTaskGet + pollTask
├── content.go            # routing branch in embedContent
├── option.go             # WithAsyncPolling
├── twelvelabs_test.go    # extended with six new flows under `ef` build tag
```

If `twelvelabs.go` crosses ~500 LoC, split the async-specific helpers into `twelvelabs_async.go` (same package, same build tag story). This is a judgment call for the planner — D-27 prefers minimal file growth.

### Pattern 1: Async request body shape

The async `POST /embed-v2/tasks` body uses a RICHER shape than the sync body. The existing `AudioInput` / `VideoInput` types in `twelvelabs.go` are scoped to sync.

**Sync `AudioInput` shape today** (`twelvelabs.go:113-116`):
```go
type AudioInput struct {
    MediaSource     MediaSource `json:"media_source"`
    EmbeddingOption string      `json:"embedding_option,omitempty"` // single string
}
```

**Async body shape required by `POST /embed-v2/tasks`** [VERIFIED: Python SDK `audio_input_request.py`]:
```go
// Required for the tasks endpoint only.
type AsyncAudioInput struct {
    MediaSource     MediaSource `json:"media_source"`
    StartSec        *float64    `json:"start_sec,omitempty"`
    EndSec          *float64    `json:"end_sec,omitempty"`
    EmbeddingOption []string    `json:"embedding_option,omitempty"` // list: "audio" | "transcription"
    EmbeddingScope  []string    `json:"embedding_scope,omitempty"` // list: "clip" | "asset"
    EmbeddingType   []string    `json:"embedding_type,omitempty"` // list: "separate_embedding" | "fused_embedding"
}
type AsyncVideoInput struct {
    MediaSource     MediaSource `json:"media_source"`
    StartSec        *float64    `json:"start_sec,omitempty"`
    EndSec          *float64    `json:"end_sec,omitempty"`
    EmbeddingOption []string    `json:"embedding_option,omitempty"`
    EmbeddingScope  []string    `json:"embedding_scope,omitempty"`
    EmbeddingType   []string    `json:"embedding_type,omitempty"`
}
type AsyncEmbedV2Request struct {
    InputType string           `json:"input_type"`             // "audio" | "video"
    ModelName string           `json:"model_name"`             // "marengo3.0"
    Audio     *AsyncAudioInput `json:"audio,omitempty"`
    Video     *AsyncVideoInput `json:"video,omitempty"`
}
```

**For Phase 26's radical-simplicity brief:** a minimal body (only `media_source`, plus `embedding_option: ["audio"]` mapped from the existing string `AudioEmbeddingOption`) is sufficient to satisfy TLA-01..04. The planner should decide whether to expose the extra fields now or defer them. Recommendation: defer — emit only `media_source` for video and `media_source + embedding_option: [<string>]` (wrapping the existing single-value option into a one-element list) for audio. This keeps the phase scoped to the bug fix and avoids widening the public option surface. [ASSUMED] that Twelve Labs accepts a minimal async body — verify by hitting the endpoint once in a spike, or defer verification to integration testing.

### Pattern 2: Task-retrieve response shape

**Response shape of `GET /embed-v2/tasks/{task_id}`** [VERIFIED: Python SDK `embedding_task_response.py`]:

```go
type TaskResponse struct {
    ID       string            `json:"_id"`              // Pydantic alias: id -> _id
    Status   string            `json:"status"`           // "processing" | "ready" | "failed"
    Data     []EmbedV2DataItem `json:"data,omitempty"`   // null when processing/failed
    Metadata json.RawMessage   `json:"metadata,omitempty"`
    // created_at / updated_at are present but not needed for this phase
}
```

Reusing the existing `EmbedV2DataItem` type works because `EmbeddingData.embedding` is `list[float]` in both sync and async responses.

**Response shape of `POST /embed-v2/tasks`** [VERIFIED: Python SDK `tasks_create_response.py`]:

```go
type TaskCreateResponse struct {
    ID     string `json:"_id"`     // task id (Pydantic alias id -> _id)
    Status string `json:"status"`  // always "processing" on create
    Data   []EmbedV2DataItem `json:"data,omitempty"` // almost always null on create
}
```

The `_id` JSON alias is non-obvious — miss it and the task ID comes back empty.

### Pattern 3: Polling loop

**Recommended skeleton** (pseudocode, not prescriptive layout):

```go
func (e *TwelveLabsEmbeddingFunction) pollTask(ctx context.Context, taskID string, maxWait time.Duration) (*TaskResponse, error) {
    deadline := time.Now().Add(maxWait)
    interval := e.apiClient.asyncPollInitial // e.g., 2s
    for {
        resp, err := e.doTaskGet(ctx, taskID)
        if err != nil {
            return nil, err // HTTP / transport error, already wrapped
        }
        switch resp.Status {
        case "ready":
            return resp, nil
        case "failed":
            return nil, errors.Errorf("Twelve Labs task [%s] terminal status=failed: %s", taskID, chttp.SanitizeErrorBody(...reasonBytes...))
        case "processing":
            // fall through to sleep
        default:
            return nil, errors.Errorf("Twelve Labs task [%s] unexpected status %q", taskID, resp.Status)
        }
        // sleep with ctx + maxWait awareness
        remaining := time.Until(deadline)
        if remaining <= 0 {
            return nil, errors.Errorf("Twelve Labs task [%s] maxWait %s exceeded", taskID, maxWait)
        }
        wait := interval
        if wait > remaining {
            wait = remaining
        }
        timer := time.NewTimer(wait)
        select {
        case <-ctx.Done():
            timer.Stop()
            return nil, errors.Wrap(ctx.Err(), "Twelve Labs async polling canceled")
        case <-timer.C:
        }
        interval = nextBackoff(interval, e.apiClient.asyncPollMultiplier, e.apiClient.asyncPollCap)
    }
}
```

Key correctness points the planner must enforce:

- **`time.NewTimer` not `time.After`** inside a loop — `time.After` leaks a timer per iteration if `ctx.Done()` wins the `select`. Stop the timer on the ctx branch.
- **Poll first, sleep second.** `POST /tasks` itself returns `status: "processing"`; the first poll after create should not sleep 2s before the first check. Either (a) return early if the `TaskCreateResponse.status == "ready"` (extremely unlikely but documented as possible in the SDK type), or (b) start the polling loop immediately.
- **`maxWait` is absolute, not per-poll.** Compute `deadline := time.Now().Add(maxWait)` once, before the loop.
- **Two distinct timeout errors** — ctx deadline exceeded vs. maxWait exceeded (D-20) — must surface distinguishable messages. Ctx fires through stdlib wrapping; maxWait fires through `errors.Errorf` with a distinct message (e.g., "maxWait 30m0s exceeded").

### Pattern 4: Modality-based routing branch

The natural decision point is the `embedContent` method in `content.go:130-140`:

```go
func (e *TwelveLabsEmbeddingFunction) embedContent(ctx context.Context, content embeddings.Content) (embeddings.Embedding, error) {
    // existing: contentToRequest -> doPost -> embeddingFromResponse
    //
    // new branch: if e.apiClient.asyncPollingEnabled && (modality == audio || video),
    //             use the async path.
}
```

Rather than branching inside `embedContent`, a cleaner split is:

- `contentToRequest` keeps building the sync request (unchanged).
- A new `contentToAsyncRequest` builds the tasks-endpoint body for audio/video.
- `embedContent` checks `e.apiClient.asyncPollingEnabled` + `part.Modality` once, then dispatches to either `doPost` or `createTaskAndPoll`.

This keeps the modality switch in one place and preserves the existing code for non-audio/video paths.

### Anti-Patterns to Avoid

- **Do not call `/embed-v2` first and inspect the response for async hints.** D-01 explicitly rejects this. The sync endpoint will reject audio/video > 10 min with a 400; do not try to recover.
- **Do not use `time.After` in the poll loop.** Classic Go timer-leak pitfall.
- **Do not create a new HTTP client** — reuse `e.apiClient.Client` so the caller's `WithHTTPClient` continues to work.
- **Do not panic on polling misconfiguration.** Fall back to defaults in `applyDefaults` silently.
- **Do not set `Retry-After` handling.** The docs do not specify a `Retry-After` header on 429, and polling cadence is governed by the backoff schedule. [VERIFIED: Twelve Labs rate-limits docs page via WebFetch]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Error-body truncation | Custom bytes.Truncate | `chttp.SanitizeErrorBody` | Phase 25 convention, handles utf8 boundaries, panic recovery |
| Reading HTTP body safely | `io.ReadAll` | `chttp.ReadLimitedBody` | Enforces the 200 MB cap |
| User-Agent string | Raw string literal | `chttp.ChromaGoClientUserAgent` | Single source of truth |
| Struct validation | Custom `reflect` checks | `embeddings.NewValidator()` | Existing pattern in `validate()` |
| Secret handling | Raw `string` | `embeddings.Secret` via `NewSecret` | Existing pattern on `TwelveLabsClient.APIKey` |

**Key insight:** Phase 25 already did the work of making error surfaces safe. Do not reintroduce raw-body error messages. Every path that includes body text in an error MUST route through `chttp.SanitizeErrorBody`.

## Common Pitfalls

### Pitfall 1: Missing `_id` JSON alias on task responses

**What goes wrong:** Parsing the task-create or task-retrieve response into a struct with `ID string \`json:"id"\`` silently decodes an empty string because the Twelve Labs API uses `_id` (Mongo-style) — the Python SDK aliases it in Pydantic. Tests that mock with `{"id": "..."}` instead of `{"_id": "..."}` will pass while production polls against `""`.

**Why it happens:** The public Twelve Labs docs examples tend to show `id` in prose but the wire format uses `_id`. [VERIFIED: Python SDK `embedding_task_response.py` L20, `tasks_create_response.py` L14]

**How to avoid:** Use `json:"_id"` in Go structs. Mirror this in test fixtures — every mock response for the create and retrieve endpoints MUST emit `_id`.

**Warning signs:** Polling immediately loops forever against `GET /embed-v2/tasks//` (note the empty path segment) and the server 404s.

### Pitfall 2: `time.After` timer leak in polling loop

**What goes wrong:** Every `time.After(interval)` allocates a fresh timer that lives until the interval expires. In a loop with ctx cancellation winning the `select`, those timers accumulate until GC at interval-completion time. Not a crash, but a leak detectable under `-race` + long cancellation tests.

**How to avoid:** Use `time.NewTimer(interval)` and call `timer.Stop()` on the ctx-done branch.

**Warning signs:** `go vet` does not catch this. A slow `TestCtxCancel` run is the tell.

### Pitfall 3: `ctx.Deadline()` interaction with `maxWait`

**What goes wrong:** D-09 requires bounding by `min(ctx.Deadline(), maxWait)`. A naive implementation uses `ctx.WithTimeout(maxWait)` which ALSO reduces `ctx.Deadline()`, making D-20 (distinct maxWait error) impossible because both fire as `context.DeadlineExceeded`.

**How to avoid:** Do NOT derive a child ctx from `maxWait`. Track `maxWaitDeadline := time.Now().Add(maxWait)` independently; check it in the poll loop via `time.Now().After(maxWaitDeadline)` and emit a distinct error. Let the caller's original `ctx` carry only the caller's deadline.

### Pitfall 4: First-poll pause

**What goes wrong:** A naive `for { sleep; poll }` loop waits 2 seconds before the first status check. This is annoying in tests (hard to set millisecond-scale polling) and adds 2 seconds of latency for small tasks that finished quickly on the server.

**How to avoid:** `for { poll; if terminal return; sleep }` — poll, then sleep.

### Pitfall 5: Different `embedding_option` shape between sync and async

**What goes wrong:** The existing `AudioInput.EmbeddingOption` is a single string. Copying that struct into the tasks body produces an invalid request shape and a 400 from the server with a message about "expected array, got string".

**How to avoid:** Introduce a dedicated `AsyncAudioInput` / `AsyncVideoInput` or at minimum send `embedding_option: []string{apiClient.AudioEmbeddingOption}` (wrap in a slice). Tests should assert the wire shape explicitly. [VERIFIED: Python SDK `audio_input_request.py` vs `audio_input.py` would show the diff — the async variant expects a list.]

### Pitfall 6: Config round-trip skipping `false`

**What goes wrong:** When `WithAsyncPolling` is absent, emitting `"async_polling": false` in `GetConfig` round-trips cleanly but violates D-22. When present, omitting `async_max_wait_ms` drops the caller's tuning. When `maxWait == 0` (default sentinel), the saved config loses the information that the option was even enabled.

**How to avoid:** Track `asyncPollingEnabled bool` on `TwelveLabsClient` separately from `asyncMaxWait time.Duration`. `GetConfig` emits both keys iff `asyncPollingEnabled == true`. `NewTwelveLabsEmbeddingFunctionFromConfig` treats missing keys as "off" (D-23). For round-trip idempotence: if `asyncPollingEnabled && asyncMaxWait == 0`, emit `async_max_wait_ms: 1800000` (30 min default) — do not emit `0`, which would round-trip back to "use default" and produce the same end state but make diffs noisy.

## Runtime State Inventory

Not a rename/refactor/migration phase. Section omitted.

## Environment Availability

This is a code/test-only phase. No external CLI, runtime, service, or tool is invoked during execution beyond what already powers `go test`. Existing test infrastructure (`httptest.Server`, `testify/require`, `testify/assert`) is already in use in `twelvelabs_test.go`. No fallback analysis needed.

**Skip rationale:** Phase is purely Go package code + tests; no external dependencies introduced (D-12 prohibits new deps).

## Code Examples

### Adding the option (verified pattern from `option.go`)

```go
// Source: existing pattern in pkg/embeddings/twelvelabs/option.go
func WithAsyncPolling(maxWait time.Duration) Option {
    return func(p *TwelveLabsClient) error {
        p.asyncPollingEnabled = true
        if maxWait == 0 {
            p.asyncMaxWait = 30 * time.Minute
        } else if maxWait < 0 {
            return errors.New("maxWait cannot be negative")
        } else {
            p.asyncMaxWait = maxWait
        }
        return nil
    }
}
```

### Extending `applyDefaults` (verified against `twelvelabs.go:44-57`)

```go
// Source: extend existing applyDefaults
func applyDefaults(c *TwelveLabsClient) {
    // ... existing defaults ...
    if c.asyncPollInitial == 0 {
        c.asyncPollInitial = 2 * time.Second
    }
    if c.asyncPollMultiplier == 0 {
        c.asyncPollMultiplier = 1.5
    }
    if c.asyncPollCap == 0 {
        c.asyncPollCap = 60 * time.Second
    }
}
```

### doTaskPost pattern (mirror of existing `doPost` in `twelvelabs.go:157-197`)

```go
// Source: mirror pkg/embeddings/twelvelabs/twelvelabs.go doPost
func (e *TwelveLabsEmbeddingFunction) doTaskPost(ctx context.Context, req AsyncEmbedV2Request) (*TaskCreateResponse, error) {
    reqJSON, err := json.Marshal(req)
    if err != nil {
        return nil, errors.Wrap(err, "failed to marshal async task request")
    }
    url := strings.TrimRight(e.apiClient.BaseAPI, "/") + "/tasks"
    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqJSON))
    if err != nil {
        return nil, errors.Wrap(err, "failed to create HTTP request")
    }
    httpReq.Header.Set("x-api-key", e.apiClient.APIKey.Value())
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Accept", "application/json")
    httpReq.Header.Set("User-Agent", chttp.ChromaGoClientUserAgent)

    resp, err := e.apiClient.Client.Do(httpReq)
    if err != nil {
        return nil, errors.Wrapf(err, "failed to send task request to %s", url)
    }
    defer resp.Body.Close()

    respData, err := chttp.ReadLimitedBody(resp.Body)
    if err != nil {
        return nil, errors.Wrap(err, "failed to read response body")
    }
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        // reuse the parsed-then-raw fallback pattern from doPost
        var apiErr EmbedV2ErrorResponse
        if jsonErr := json.Unmarshal(respData, &apiErr); jsonErr == nil && apiErr.Message != "" {
            return nil, errors.Errorf("Twelve Labs task create error [%s]: %s", resp.Status, chttp.SanitizeErrorBody([]byte(apiErr.Message)))
        }
        return nil, errors.Errorf("unexpected status [%s] from %s: %s", resp.Status, url, chttp.SanitizeErrorBody(respData))
    }
    var taskResp TaskCreateResponse
    if err := json.Unmarshal(respData, &taskResp); err != nil {
        return nil, errors.Wrap(err, "failed to unmarshal task create response")
    }
    return &taskResp, nil
}
```

URL construction note: the existing `BaseAPI` default is `https://api.twelvelabs.io/v1.3/embed-v2`. Appending `/tasks` yields `https://api.twelvelabs.io/v1.3/embed-v2/tasks` — the SDK-verified path. `GET /tasks/{id}` similarly becomes `BaseAPI + "/tasks/" + url.PathEscape(taskID)`.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Sync-endpoint-returns-task-ID hypothesis | Dedicated `/embed-v2/tasks` POST + retrieve | Always was the contract; roadmap had a premise drift | D-01 corrected the research premise before research ran |
| Poll via `/status` + retrieve via `/tasks/{id}` | Single `GET /tasks/{id}` returns both | Per SDK generated 2024+ | Collapses polling to one endpoint (Flag F-01) |
| `Retry-After` header for 429 | Not specified by Twelve Labs | n/a | No special retry logic needed for rate limits in this phase |

**Deprecated/outdated:**
- None.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | A minimal async request body (`media_source` only for video; `media_source + embedding_option: ["audio"]` for audio) is accepted by `POST /embed-v2/tasks`. | Pattern 1 | If rejected, planner must widen the async body struct to include `embedding_scope`/`embedding_type` defaults. Verifiable with a one-shot manual curl against the live endpoint. |
| A2 | The `GET /embed-v2/tasks/{id}` response's `data` field is always populated when `status == "ready"` and always `null` when `status == "processing"` or `"failed"`. | Pattern 2 | If partially populated on `processing`, the naive "return on ready" check still works, but tests should assert explicitly. SDK docstrings support this behavior. |
| A3 | A failed task's reason text lives in the response body (likely as a top-level `error`/`message` field on `EmbeddingTaskResponse` due to Pydantic `extra="allow"`). | Pattern 3, Pitfall 1 | The SDK doesn't name a specific failure-message field, so the provider should extract `status` + whatever extra string fields are present and sanitize them. Minimally, include the raw response body via `SanitizeErrorBody(respData)` when `status == "failed"`. |
| A4 | Two-endpoint model (create + retrieve) is sufficient; the `/status` endpoint listed in D-01 is not required. | Summary, Flag F-01 | If Twelve Labs exposes `/status` as a lighter-weight poll target (smaller response), using `/tasks/{id}` wastes bandwidth by carrying the full `data` array on every poll. For task polling where `data` is `null` until `ready`, the difference is negligible. |
| A5 | `marengo3.0` accepts the existing `AudioEmbeddingOption` values (`audio`, `transcription`, `fused`) when wrapped in a list for the async endpoint. Note: `fused` is NOT a valid async `embedding_option` — the async SDK type lists only `audio` and `transcription`; `fused` is represented via `embedding_type: ["fused_embedding"]`. | Pitfall 5 | If the existing option value is `"fused"` and the user enables async polling for audio, the naive wrap-in-list produces an invalid async request. Planner must either reject `fused` when async is enabled for audio, or transform `fused` into the correct async representation. |

**All five assumptions are verifiable** with a single authenticated curl against the live API or by inspecting a richer slice of the Python SDK source. They do not block planning, but they are the claims most likely to bite at integration time.

## Flags for Planner

These are premise-breaking or scope-adjacent discoveries. They are raised for planner review without reopening locked decisions.

### F-01 (HIGH priority): D-01 lists three endpoints; only two exist

**Evidence:** The Fern-generated Python SDK raw client ([twelvelabs-python/src/twelvelabs/embed/v_2/tasks/raw_client.py](https://github.com/twelvelabs-io/twelvelabs-python/blob/main/src/twelvelabs/embed/v_2/tasks/raw_client.py)) defines only three methods: `list`, `create`, and `retrieve`. The `retrieve` method calls `GET embed-v2/tasks/{task_id}` and parses the response as `EmbeddingTaskResponse`, which carries the `status` discriminator AND the `data` array. There is no method hitting `/embed-v2/tasks/{id}/status`.

**Ground truth (Python SDK, `reference.md` lines 4294-4298):**
> 1. Create a task using this endpoint. The platform returns a task ID.
> 2. Poll for the status of the task using the GET method of the `/embed-v2/tasks/{task_id}` endpoint. Wait until the status is `ready`.
> 3. Retrieve the embeddings from the response when the status is `ready` using the GET method of the `/embed-v2/tasks/{task_id}` endpoint.

Both steps 2 and 3 use the same endpoint.

**Impact:** The implementation needs only `doTaskPost` + `doTaskGet`. No separate `/status` helper. This SIMPLIFIES the phase.

**Recommendation:** Planner should relax D-01 to reflect the two-endpoint reality. This does not reopen any decision beyond the endpoint list — all other invariants in D-01 (sync endpoint untouched, async opt-in via separate endpoint, no sync-endpoint detection) hold.

### F-02 (MEDIUM priority): Async request body shape differs from sync

**Evidence:** See Pattern 1 above; `audio_input_request.py` in the async tasks SDK uses `embedding_option: list[str]` vs the sync `audio_input.py` using a single string. The existing Go `AudioInput` type cannot be reused verbatim.

**Impact:** A minimum of one new request struct (`AsyncEmbedV2Request`) is required, plus either separate `AsyncAudioInput`/`AsyncVideoInput` types or on-the-fly wrapping of `[]string{apiClient.AudioEmbeddingOption}`. Small but not zero.

**Recommendation:** Planner should include a task step for introducing the async request types explicitly and mapping the existing `AudioEmbeddingOption` into the list shape. Document the mapping of `fused` (Assumption A5) in code or reject it with a validation error when async is enabled.

### F-03 (LOW priority): Canonical References doc URLs are 404

**Evidence:** Every URL in CONTEXT.md's `<canonical_refs>` block under "Twelve Labs API references" that matches `docs.twelvelabs.io/api-reference/embed-v2/*` returns "Page Not Found" when fetched. The live prefix appears to be `docs.twelvelabs.io/v1.3/api-reference/create-embeddings-v2/*` (the Python SDK's doc links use this form).

**Impact:** Future researchers or reviewers clicking those links won't reach content. This is a doc-rot issue, not a design issue.

**Recommendation:** When the planner writes plan docs, prefer the SDK source (`github.com/twelvelabs-io/twelvelabs-python/src/twelvelabs/...`) over the `docs.twelvelabs.io` URLs for API contracts. The SDK is generated from the OpenAPI spec and is the stable ground truth.

### F-04 (LOW priority): `Retry-After` not documented

**Evidence:** Twelve Labs rate-limits docs page lists `X-RateLimit-Request-Remaining` and `X-RateLimit-Duration-Remaining` headers but does not document a `Retry-After` header on 429 responses. [VERIFIED: docs.twelvelabs.io/docs/get-started/rate-limits via WebFetch]

**Impact:** The polling loop should NOT attempt to honor `Retry-After`. If a 429 fires during polling, the existing backoff (which already includes the `cap=60s` ceiling) is the correct response. Add a single inline comment documenting this.

**Recommendation:** Treat 429 during polling as a transient error, wrap with `errors.Errorf` like other HTTP failures, and continue the loop. The capped exponential backoff naturally limits burst traffic.

## Open Questions

None of these block planning — the planner can proceed and have each answered during implementation or UAT.

1. **Does `POST /embed-v2/tasks` accept a minimal body (`media_source` only for video)?**
   - What we know: Python SDK exposes many optional fields; required fields are `input_type`, `model_name`, and `audio` or `video` with `media_source`.
   - What's unclear: whether the server rejects when optional embedding/scope/type arrays are empty.
   - Recommendation: Start with minimal body; if the live endpoint rejects, widen in a follow-up before merge.

2. **What does a `failed` task response body contain beyond `status`?**
   - What we know: Pydantic v2 `extra="allow"` on `EmbeddingTaskResponse` means the shape can include undeclared fields.
   - What's unclear: the exact field name for the failure reason (`error`, `message`, `failure_reason`?).
   - Recommendation: In the failed-state error message, include the sanitized full response body via `chttp.SanitizeErrorBody(respData)`. This handles any field name without guessing.

3. **Does `_id` appear consistently across both create and retrieve responses?**
   - What we know: Python SDK aliases `id -> _id` in both `EmbeddingTaskResponse` and `TasksCreateResponse`.
   - What's unclear: whether Mongo-to-wire translation changes across API versions.
   - Recommendation: Always use `json:"_id"` in Go struct tags; test fixtures use `_id`.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | `go test` via `gotestsum`, with `github.com/stretchr/testify/{require,assert}` assertions |
| Config file | No config — build tags gate which suites run. See `CLAUDE.md` commands. |
| Quick run command | `go test -tags=ef -count=1 -run TestTwelveLabsAsync ./pkg/embeddings/twelvelabs/...` |
| Full suite command | `make test-ef` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TLA-01 | Task creation path — `POST /tasks` succeeds, task ID captured, polling loop entered for audio/video when opt-in is on | unit (httptest) | `go test -tags=ef -count=1 -run TestTwelveLabsAsyncTaskCreate ./pkg/embeddings/twelvelabs/...` | EXISTS (extend `twelvelabs_test.go`) |
| TLA-01 | Text/image modalities skip the async path even when `WithAsyncPolling` is set | unit (httptest) | `go test -tags=ef -count=1 -run TestTwelveLabsAsyncSkipsTextImage ./pkg/embeddings/twelvelabs/...` | EXISTS (extend `twelvelabs_test.go`) |
| TLA-02 | Polling respects `ctx.Cancel()` mid-poll and returns `context.Canceled` | unit (httptest) | `go test -tags=ef -count=1 -run TestTwelveLabsAsyncCtxCancel ./pkg/embeddings/twelvelabs/...` | EXISTS (extend `twelvelabs_test.go`) |
| TLA-02 | `maxWait` bound fires with a distinct (non-`context.DeadlineExceeded`) error message | unit (httptest) | `go test -tags=ef -count=1 -run TestTwelveLabsAsyncMaxWait ./pkg/embeddings/twelvelabs/...` | EXISTS (extend `twelvelabs_test.go`) |
| TLA-03 | N `processing` responses followed by `ready` returns the expected embedding | unit (httptest, attempt counter) | `go test -tags=ef -count=1 -run TestTwelveLabsAsyncPollToReady ./pkg/embeddings/twelvelabs/...` | EXISTS (extend `twelvelabs_test.go`) |
| TLA-03 | `processing` → `failed` produces an error containing task ID, "failed", and a sanitized reason | unit (httptest) | `go test -tags=ef -count=1 -run TestTwelveLabsAsyncPollToFailed ./pkg/embeddings/twelvelabs/...` | EXISTS (extend `twelvelabs_test.go`) |
| TLA-03 | Unexpected status value produces a clear error (D-16) | unit (httptest) | `go test -tags=ef -count=1 -run TestTwelveLabsAsyncUnexpectedStatus ./pkg/embeddings/twelvelabs/...` | EXISTS (extend `twelvelabs_test.go`) |
| TLA-04 | All of the above pass as a group | test group | `go test -tags=ef -count=1 -run TestTwelveLabsAsync ./pkg/embeddings/twelvelabs/...` | EXISTS |
| (D-21/D-22/D-23) | `GetConfig` emits `async_polling` + `async_max_wait_ms` when enabled, omits when disabled; rebuild applies option | unit (no HTTP) | `go test -tags=ef -count=1 -run TestTwelveLabsAsyncConfigRoundTrip ./pkg/embeddings/twelvelabs/...` | EXISTS (extend `twelvelabs_test.go`) |

### Sample Fixture Sketches (one per D-26 flow)

**Flow 1: Task creation + poll-to-ready**

```go
func TestTwelveLabsAsyncPollToReady(t *testing.T) {
    vec := make512DimVector()
    var attempts atomic.Int32
    srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        switch {
        case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/tasks"):
            fmt.Fprint(w, `{"_id":"task_abc","status":"processing"}`)
        case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/tasks/task_abc"):
            n := attempts.Add(1)
            if n < 3 {
                fmt.Fprint(w, `{"_id":"task_abc","status":"processing"}`)
                return
            }
            resp := map[string]any{
                "_id":    "task_abc",
                "status": "ready",
                "data":   []map[string]any{{"embedding": vec}},
            }
            _ = json.NewEncoder(w).Encode(resp)
        default:
            http.Error(w, "unexpected", http.StatusBadRequest)
        }
    })

    ef := newTestEF(srv.URL)
    ef.apiClient.asyncPollingEnabled = true
    ef.apiClient.asyncMaxWait = 5 * time.Second
    ef.apiClient.asyncPollInitial = 1 * time.Millisecond
    ef.apiClient.asyncPollMultiplier = 1.0
    ef.apiClient.asyncPollCap = 5 * time.Millisecond

    emb, err := ef.EmbedContent(context.Background(), videoContent("https://example.com/v.mp4"))
    require.NoError(t, err)
    assert.Equal(t, 512, emb.Len())
    assert.Equal(t, int32(3), attempts.Load())
}
```

**Flow 2: Poll-to-failed** — same structure, third response is `{"_id":"task_abc","status":"failed","error":"unsupported codec"}`. Assertion: `err.Error()` contains `task_abc`, `failed`, and `unsupported codec` (post-sanitization).

**Flow 3: ctx cancellation** — handler always returns `processing`. Test calls `ctx, cancel := context.WithCancel(ctx)` and `cancel()` from a goroutine after attempts reaches 1. Assertion: `errors.Is(err, context.Canceled)`.

**Flow 4: maxWait expiration** — handler always returns `processing`. `asyncMaxWait` set to 50 ms with 10 ms polling. Assertion: `err.Error()` contains `maxWait`, does NOT contain `context.DeadlineExceeded`.

**Flow 5: unexpected status** — handler returns `{"_id":"task_abc","status":"zombie"}`. Assertion: `err.Error()` contains `unexpected status`, `zombie`, and `task_abc`.

**Flow 6: config round-trip** — construct an EF with `WithAsyncPolling(45*time.Minute)`, call `GetConfig`, assert `cfg["async_polling"] == true && cfg["async_max_wait_ms"] == int64(2700000)`. Rebuild via `NewTwelveLabsEmbeddingFunctionFromConfig`, assert `restored.apiClient.asyncPollingEnabled == true` and `restored.apiClient.asyncMaxWait == 45*time.Minute`. Additionally: when option is absent, assert `_, ok := cfg["async_polling"]; !ok` and `_, ok := cfg["async_max_wait_ms"]; !ok` (D-22).

**Flow 7 (bonus, optional): sync-path regression** — assert that text + image with `WithAsyncPolling` set still hit `/embed-v2` (not `/embed-v2/tasks`), enforced by a test server that returns 400 on any `/tasks` path.

### Observable Assertions Per Flow

| Flow | HTTP counter | Error shape | Response shape |
|------|-------------|-------------|----------------|
| 1 poll-to-ready | attempt count > 1 (proves polling happened) | `err == nil` | `emb.Len() == expected dim` |
| 2 poll-to-failed | attempt count > 1 | `err` contains task ID + `failed` + reason | n/a |
| 3 ctx cancel | attempt count >= 1 | `errors.Is(err, context.Canceled)` == true | n/a |
| 4 maxWait | attempt count >= 1 | `err` contains `maxWait`, does not match `context.DeadlineExceeded` via `errors.Is` | n/a |
| 5 unexpected status | attempt count == 1 | `err` contains `unexpected status` + value + task ID | n/a |
| 6 config round-trip | zero HTTP (no server) | n/a | map keys match D-21 / D-22 |

### Sampling Rate

- **Per task commit:** `go test -tags=ef -count=1 -run TestTwelveLabsAsync ./pkg/embeddings/twelvelabs/... -timeout 30s` (expected < 5 s wall clock given ms-scale polling)
- **Per wave merge:** `make test-ef` (full provider suite)
- **Phase gate:** `make test-ef && make lint` green before `/gsd-verify-work`

### Wave 0 Gaps

None. The existing test infrastructure covers all phase requirements:
- `newTestEF` helper exists in `twelvelabs_test.go` (lines 26-36). Extend to inject polling fields.
- `newMockServer` helper exists.
- `embedV2Response` fixture helper exists.
- `atomic.Int32` counters are idiomatic for attempt tracking (stdlib — no new import required for tests that already use `sync/atomic`).

New helpers to ADD (small, inline in `twelvelabs_test.go`):
- `videoContent(url string) embeddings.Content` — builds a one-part video Content with URL source.
- `audioContent(url string) embeddings.Content` — same for audio.
- `taskResponseJSON(status, err string, data []float64) string` — builds a task-response fixture with correct `_id` alias.

These are all ~5-line helpers that can live at the top of `twelvelabs_test.go`.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | Reuse existing `x-api-key` header pattern from `doPost`; never log the key. `embeddings.Secret` already wraps the key. |
| V3 Session Management | no | No session state — every call carries the API key header. |
| V4 Access Control | no | Authorization is enforced server-side by Twelve Labs. |
| V5 Input Validation | yes | Validate `maxWait >= 0` in `WithAsyncPolling`. Validate async request body types before send. |
| V6 Cryptography | yes | HTTPS required (existing `validate()` check at `twelvelabs.go:67-70` — do not weaken for task endpoints). |
| V8 Data Protection | yes | Task IDs and media URLs may be sensitive; do not include full response bodies in error logs beyond `SanitizeErrorBody` truncation (512 runes). |
| V9 Communications | yes | TLS-only via existing `Insecure` gate; polling reuses same `*http.Client`. |
| V11 Business Logic | yes | `maxWait` protects against unbounded blocking — explicit footgun mitigation. |

### Known Threat Patterns for chroma-go + Twelve Labs async

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| API key leakage via error message on failed poll | Information Disclosure | Already mitigated — `doPost` never includes headers in error strings; task-endpoint helpers inherit this via the `doPost` mirror pattern. |
| Oversized response body (malicious server) | Denial of Service | Reuse `chttp.ReadLimitedBody` (200 MB cap) |
| Unbounded polling causing goroutine pile-up | Denial of Service | `maxWait` + ctx deadline both bound polling time (D-09) |
| Timer leaks under ctx cancellation | Denial of Service (slow) | Use `time.NewTimer` + `timer.Stop()` (Pitfall 2) |
| Error body containing user-controlled content echoed to caller logs | Information Disclosure | `chttp.SanitizeErrorBody` truncates to 512 runes with recovery (Phase 25) |
| Poll loop panicking on malformed JSON | Availability | `json.Unmarshal` returns error; the existing non-panic pattern (`CLAUDE.md` Panic Prevention) is preserved. |
| Task ID treated as trusted (path injection) | Tampering | `url.PathEscape(taskID)` when constructing `GET /tasks/{id}` — the task ID comes from a Twelve Labs response but defense-in-depth is cheap. |

## Project Constraints (from CLAUDE.md)

- **V2-first:** This phase touches `pkg/embeddings/twelvelabs/` which is shared infra consumed by V2. No V1 impact. Compliant.
- **No panics in production code:** The new polling helper MUST use error returns, not panics. Timer-based code uses `time.NewTimer` (no panic path). `json.Unmarshal` returns errors. Compliant.
- **No `Must*` functions:** None required. Compliant.
- **`errors.Wrap` / `errors.Errorf` from `pkg/errors`:** All new error paths use this (consistent with existing `twelvelabs.go`). Do NOT introduce stdlib `%w` wrapping in this package. Compliant.
- **Conventional commits:** Commit prefix `feat(twelvelabs): ...` or `fix(twelvelabs): ...`. Phase 26 aligns with a `feat` since it adds a new capability. Applicable.
- **Build tags:** New tests live under `ef` build tag (D-27). Compliant.
- **Minimal dependencies:** Zero new deps (D-12). Compliant.
- **Lint before commit:** `make lint` is phase-gate. Already in Validation Architecture.
- **Cloud test bar (from MEMORY):** The completion bar is that cloud integration tests exercising the new code path end-to-end pass. Phase 26 is provider-internal and does not touch Cloud API, BUT the `ef`-gated live Twelve Labs test in `twelvelabs_live_test.go` (if it exists) should be considered for a live-key smoke. Planner should check whether a live-async test is feasible given free-tier rate limits (RPM 8; 4-hour task duration makes even one live test expensive). Acceptable to defer live-cloud verification for TLA-* given the httptest-based unit coverage.
- **Radical simplicity (CLAUDE.md):** Two-endpoint model (not three), minimal async body, extend existing files rather than creating new ones. Compliant.

## Sources

### Primary (HIGH confidence)

- [twelvelabs-python SDK — tasks raw client](https://github.com/twelvelabs-io/twelvelabs-python/blob/main/src/twelvelabs/embed/v_2/tasks/raw_client.py) — endpoint paths, methods, status code handling
- [twelvelabs-python SDK — EmbeddingTaskResponse](https://github.com/twelvelabs-io/twelvelabs-python/blob/main/src/twelvelabs/types/embedding_task_response.py) — response shape, `_id` alias, `data` nullability
- [twelvelabs-python SDK — TasksCreateResponse](https://github.com/twelvelabs-io/twelvelabs-python/blob/main/src/twelvelabs/embed/v_2/tasks/types/tasks_create_response.py) — create-response shape, status enum = `"processing"`
- [twelvelabs-python SDK — embedding_task_response_status](https://github.com/twelvelabs-io/twelvelabs-python/blob/main/src/twelvelabs/types/embedding_task_response_status.py) — full status enum `"processing" | "ready" | "failed"`
- [twelvelabs-python SDK — EmbeddingData](https://github.com/twelvelabs-io/twelvelabs-python/blob/main/src/twelvelabs/types/embedding_data.py) — `data[].embedding: list[float]`
- [twelvelabs-python SDK — AudioInputRequest](https://github.com/twelvelabs-io/twelvelabs-python/blob/main/src/twelvelabs/types/audio_input_request.py) — async body shape with list-typed fields
- [twelvelabs-python SDK — reference.md](https://github.com/twelvelabs-io/twelvelabs-python/blob/main/reference.md) — `reference.md` lines 3984, 4260, 4294-4298, 4399-4474 confirm two-endpoint polling flow
- Repo codebase reads — existing patterns, file locations, helper signatures
  - `pkg/embeddings/twelvelabs/twelvelabs.go`
  - `pkg/embeddings/twelvelabs/content.go`
  - `pkg/embeddings/twelvelabs/option.go`
  - `pkg/embeddings/twelvelabs/twelvelabs_test.go`
  - `pkg/commons/http/utils.go`, `constants.go`, `errors.go`
  - `.planning/codebase/TESTING.md`, `CONVENTIONS.md`

### Secondary (MEDIUM confidence)

- [Twelve Labs rate-limits docs](https://docs.twelvelabs.io/docs/get-started/rate-limits) — verified no `Retry-After` mention, confirmed `X-RateLimit-*` header names (via WebFetch summary)
- [Twelve Labs create-embeddings-v2 sync doc](https://docs.twelvelabs.io/api-reference/create-embeddings-v2) — verified sync endpoint description distinguishes 10-min limit → tasks endpoint for longer media (via WebSearch citation)

### Tertiary (LOW confidence)

- None retained. Earlier WebFetch attempts against `docs.twelvelabs.io/api-reference/embed-v2/*` returned 404 and are excluded from this research.

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — zero new deps, all from stdlib + existing packages.
- Architecture (two-endpoint polling loop): HIGH — verified in generated SDK code.
- Pitfalls (timer leak, `_id` alias, shape divergence, maxWait/ctx split): HIGH — verified from SDK source + direct codebase read.
- Error semantics (sanitization, status enum, unexpected-value handling): HIGH — Phase 25 convention + SDK enum.
- Request body for tasks endpoint (minimal-body hypothesis): MEDIUM — Assumption A1 not verified with a live server call; SDK shows many optional fields, server behavior for omitted fields is inferred not confirmed.
- Failed-task reason field name: LOW — not declared in SDK types (Pydantic `extra="allow"`); mitigation is to sanitize the full response body (Assumption A3).

**Research date:** 2026-04-14
**Valid until:** 2026-05-14 (30 days — Twelve Labs API is stable; Marengo 3.0 reindex was announced for mid-March 2026 and is already in effect).

## RESEARCH COMPLETE

**Phase:** 26 - twelve-labs-async-embedding
**Confidence:** HIGH

### Key Findings

- Only two task endpoints exist, not three. `GET /embed-v2/tasks/{task_id}` returns both status AND data; no `/status` sub-path is used by the official Python SDK. This SIMPLIFIES the implementation relative to CONTEXT.md D-01. (Flag F-01)
- Async request body differs from sync body: `embedding_option` is `list[string]` (async) vs `string` (sync). A dedicated `AsyncEmbedV2Request` / `AsyncAudioInput` / `AsyncVideoInput` type is needed, or the existing single-value option must be wrapped in a list. (Flag F-02)
- Task response uses `_id` (Mongo-style) not `id` as the JSON field for the task identifier. Miss this and polling hits `GET /tasks//` (empty segment). Tests must emit `_id` in fixtures.
- Status enum is exactly `"processing" | "ready" | "failed"` (D-14 is correct). Unexpected values should be a distinct error (D-16 is correct).
- `maxWait` must be tracked independently from the caller's `ctx` (do NOT derive a child ctx from `maxWait`), otherwise D-20's distinct-error requirement cannot be satisfied.

### File Created

`/Users/tazarov/GolandProjects/chroma-go/.planning/phases/26-twelve-labs-async-embedding/26-RESEARCH.md`

### Confidence Assessment

| Area | Level | Reason |
|------|-------|--------|
| Standard Stack | HIGH | Zero new deps; all existing packages. |
| Architecture | HIGH | Verified against Fern-generated official SDK source. |
| Pitfalls | HIGH | `_id` alias, timer leaks, and body-shape divergence are concrete and sourced. |
| Minimal async body acceptance | MEDIUM | Assumption A1 requires a live call or spike to prove. |
| Failed-task reason field | LOW | Pydantic `extra="allow"` hides exact field name; mitigation via full-body sanitization. |

### Open Questions

- Minimum required fields for `POST /embed-v2/tasks` (can we send only `media_source`?).
- Exact JSON field name of the failure reason on a `failed` task response.
- Whether the live Twelve Labs environment accepts `fused` as an async audio `embedding_option` or whether we must reject it at option time.

### Ready for Planning

Research complete. Planner can now create PLAN.md files. Recommended first plan step: address Flag F-01 (collapse to two endpoints) and Flag F-02 (introduce async-specific request types) in the shared task-endpoint helper, then layer polling + routing + tests. Planner should treat the five assumptions in the Assumptions Log as items to verify via code review or optional UAT — none block implementation.
