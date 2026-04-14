# Phase 26: Twelve Labs Async Embedding - Context

**Gathered:** 2026-04-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Enable the Twelve Labs provider to embed long-running audio and video content (up to the Twelve Labs 4-hour limit) through the existing blocking `EmbedContent` / `Collection.Add` path by routing audio/video through the dedicated async task endpoints and polling to completion.

**Framing:** timeout-bug-fix, not async-DX. The caller still blocks on a single call; the call now succeeds for media that would previously fail because the sync HTTP connection cannot be held open long enough server-side. No new async-shaped public surface, no interface changes, no impact on existing callers who do not opt in.

**In scope:**
- New internal code path hitting `POST /v1.3/embed-v2/tasks`, `GET /v1.3/embed-v2/tasks/{id}/status`, `GET /v1.3/embed-v2/tasks/{id}`
- Single new functional option `WithAsyncPolling(maxWait time.Duration)` that gates the async path
- Modality-based routing: when opt-in is present, audio and video go through the tasks endpoint; text and image stay on the sync `/embed-v2` path
- Polling loop that respects `ctx.Deadline()` and the `maxWait` bound
- Terminal state handling (ready, failed) with error messages that include the task ID and a sanitized reason
- Colocated tests under the `ef` build tag covering create, poll-to-ready, poll-to-failed, and ctx-cancellation
- Config round-trip keys for the new option so registry-rebuilt EFs behave identically

**Out of scope:**
- Any new public method, callback surface, task-handle type, or other shape that would expose async to callers beyond the single opt-in flag
- Exporting a structured `*TaskFailedError` type (plain `errors.Errorf` is sufficient for Path 1 framing)
- Changes to `EmbeddingFunction` or `ContentEmbeddingFunction` interface signatures
- Unblocking `Collection.Add` semantics (it stays a blocking call; long media now just succeeds where it previously failed)
- Parallel task dispatch, progress callbacks, fire-and-forget semantics, or batch throughput work
- Exposing polling interval / backoff knobs as public options in v1
- Using `POST /v1.3/embed-v2` for async detection (Twelve Labs does not document sync-to-async fallback on that endpoint; the tasks endpoint is a distinct contract)

</domain>

<decisions>
## Implementation Decisions

### Endpoint strategy
- **D-01:** Two-endpoint model (Framing A). Sync `/embed-v2` usage is unchanged. The new code path calls `POST /embed-v2/tasks` to create a task, `GET /embed-v2/tasks/{id}/status` to poll, and `GET /embed-v2/tasks/{id}` to retrieve the final embedding. The roadmap phrase "sync endpoint returns an async task response" was a premise drift — verified against Twelve Labs docs and the Fern-generated official Python SDK; the sync endpoint always returns `EmbeddingSuccessResponse` and never a task response.
- **D-02:** Existing `POST /v1.3/embed-v2` behavior is not modified. Callers who do not opt in see zero behavioral change.

### Public surface
- **D-03:** One new functional option: `WithAsyncPolling(maxWait time.Duration) Option`. Passing `0` selects the default `maxWait` of 30 minutes. Presence of this option is the sole trigger for the async code path.
- **D-04:** No other async-related options are exported in this phase. Polling interval, backoff multiplier, and cap are internal implementation details. If real-world usage later demonstrates the need for tuning, options can be added without breaking existing callers.
- **D-05:** `EmbeddingFunction` and `ContentEmbeddingFunction` interface signatures are unchanged. No new public methods, types, callbacks, or task handles.
- **D-06:** `Collection.Add` / `Collection.Query` semantics are unchanged. Callers experience the async path as a blocking `EmbedContent` call that may take longer than a single HTTP round-trip.

### Routing
- **D-07:** When `WithAsyncPolling` is present, audio and video modalities route through the tasks endpoint + polling. Text and image modalities always use the sync `/embed-v2` endpoint regardless of the option, because they have no duration concept and routing them through `/tasks` would add a task-create round-trip for zero benefit.
- **D-08:** When `WithAsyncPolling` is absent, all modalities continue to use the sync endpoint (today's behavior); long audio/video that exceeds the Twelve Labs sync limit will fail exactly as it does today.

### Bounds and cancellation
- **D-09:** Total polling time is bounded by the minimum of `ctx.Deadline()` and `maxWait`. Whichever fires first terminates the poll. `ctx.Deadline()` is the canonical Go kill switch; `maxWait` is a belt-and-suspenders bound to protect callers who forget to set a deadline (a real DX footgun since `Collection.Add` opaquely threads the caller's context into the EF).
- **D-10:** `ctx` cancellation mid-poll terminates the polling loop immediately and surfaces `context.Canceled` via the existing stdlib wrapping (no special handling required).

### Polling schedule
- **D-11:** Polling uses a hand-rolled capped exponential backoff: `initial=2s`, `multiplier=1.5`, `cap=60s`, no jitter. Values are unexported fields on `TwelveLabsClient` with defaults applied in `applyDefaults`.
- **D-12:** No external polling/backoff dependency (e.g., `cenkalti/backoff`) is introduced. This is ~30 LoC of stdlib logic; adding a dep for it violates the codebase's minimal-dep convention.
- **D-13:** No jitter in v1. Deterministic intervals make `ef`-tag tests straightforward; jitter can be added later if production load patterns warrant it.

### Async response detection (inside the polling loop)
- **D-14:** Task status is determined by the required `status` enum field on the task retrieval response: `processing` → continue polling with the next backoff step; `ready` → extract embedding from `data` and return; `failed` → terminate with a task-failure error. This matches the Fern-generated discriminator used by the official Twelve Labs Python/TypeScript SDKs.
- **D-15:** HTTP status code (200 vs. 202) is not used as the async discriminator. Twelve Labs does not document 202 semantics for these endpoints, and relying on it would bake in an unverified contract.
- **D-16:** Any unexpected `status` value (not one of the three known states) is treated as a malformed response and produces a descriptive error, not silently treated as either terminal state.

### Error semantics
- **D-17:** Terminal task failures surface as `errors.Errorf(...)` with a message that always includes the task ID, the terminal status, and a `chttp.SanitizeErrorBody`-truncated reason from the task response. No exported error type, no sentinel.
  - Example shape: `Twelve Labs task [task_abc123] terminal status=failed: <sanitized reason>`
- **D-18:** HTTP / transport errors during polling flow through the existing `errors.Wrap` / `errors.Errorf` pattern used elsewhere in `twelvelabs.go`. Error bodies from failed poll responses are sanitized via `chttp.SanitizeErrorBody` (Phase 25 convention).
- **D-19:** `context.Canceled` and `context.DeadlineExceeded` propagate unchanged via stdlib wrapping. Callers can distinguish ctx cancellation from task failure from HTTP error using standard Go patterns (`errors.Is(err, context.Canceled)` etc.); no sdk-specific sentinel is needed.
- **D-20:** `maxWait` expiration surfaces as a distinct error (not the stdlib `context.DeadlineExceeded`) so callers who hit the SDK-level bound rather than their own ctx deadline can tell the two apart from the error message.

### Config round-trip
- **D-21:** When `WithAsyncPolling` is enabled, `GetConfig` includes two new keys:
  - `async_polling: true`
  - `async_max_wait_ms: <int64 milliseconds>`
- **D-22:** When `WithAsyncPolling` is absent, both keys are omitted from the config map (no `async_polling: false` noise).
- **D-23:** `NewTwelveLabsEmbeddingFunctionFromConfig` reads these keys and reconstructs the option. Missing keys → opt-in is off, matching construction.

### Test strategy
- **D-24:** Tests use `net/http/httptest.Server` with an attempt counter, matching the existing `twelvelabs_test.go` conventions. The polling fields on `TwelveLabsClient` are set directly in test construction (like the existing `newTestEF` helper sets `BaseAPI` and `APIKey`) to millisecond-scale values so tests run in single-digit milliseconds.
- **D-25:** No clock abstraction is introduced. No new external dependency (`benbjohnson/clock` or otherwise) is added for tests. The codebase has no clock interface today and adding one for a single polling loop is overkill.
- **D-26:** Required test flows (TLA-04 coverage):
  1. Task creation path: first `POST /tasks` succeeds, returns task ID
  2. Poll-to-ready: N `processing` responses followed by `ready` + result retrieval returns the expected embedding
  3. Poll-to-failed: `processing` → `failed` produces the expected error shape
  4. `ctx` cancellation mid-poll: cancel after first `processing` response; loop exits with `context.Canceled`
  5. `maxWait` expiration: `maxWait` set shorter than ctx deadline triggers the SDK-level timeout error (distinct message)
  6. Config round-trip: `WithAsyncPolling` enabled → `GetConfig` emits the two new keys → rebuild → option reapplied
- **D-27:** Tests live under the existing `ef` build tag in `pkg/embeddings/twelvelabs/`. No new test file layout is introduced unless existing files become unwieldy; prefer extending `twelvelabs_test.go`.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone and requirement context
- `.planning/ROADMAP.md` — Phase 26 goal, dependency on Phase 25, and success criteria for TLA-01..04 (note premise-drift on "sync endpoint returns async task" — superseded by D-01)
- `.planning/REQUIREMENTS.md` — TLA-01 through TLA-04 requirement text
- `.planning/PROJECT.md` — v0.4.2 milestone framing and issue `#479` scope

### Twelve Labs API references
- https://docs.twelvelabs.io/api-reference/create-embeddings-v2 — sync embed-v2 endpoint (baseline, unchanged)
- https://docs.twelvelabs.io/api-reference/embed-v2/create-audio-video-embeddings — tasks endpoint create path
- https://docs.twelvelabs.io/api-reference/embed-v2/retrieve-embeddings-task — task retrieval (result + status)
- https://docs.twelvelabs.io/api-reference/embed-v2/retrieve-task-status — task status polling
- https://docs.twelvelabs.io/docs/concepts/tasks — task lifecycle concepts
- https://docs.twelvelabs.io/docs/guides/create-embeddings/audio — audio guide with duration notes
- https://github.com/twelvelabs-io/twelvelabs-python — reference for `EmbeddingTaskResponse` shape and `status` discriminator

### Prior locked decisions to carry forward
- `.planning/phases/25-error-body-truncation/25-CONTEXT.md` — `chttp.SanitizeErrorBody` usage convention for all error-body-derived text (applies to failure reasons and polling HTTP errors)
- `.planning/milestones/v0.4.1-phases/16-*` (if present) — original Twelve Labs provider context and capability model

### Implementation targets
- `pkg/embeddings/twelvelabs/twelvelabs.go` — `TwelveLabsClient`, `doPost`, and registration; polling fields and async detection live here
- `pkg/embeddings/twelvelabs/content.go` — `contentToRequest`, `embedContent`, `EmbedContent`, `EmbedContents`; modality-based routing decision point
- `pkg/embeddings/twelvelabs/option.go` — `Option` type and existing functional options; add `WithAsyncPolling` here
- `pkg/embeddings/twelvelabs/twelvelabs_test.go` — existing `ef`-tag mock server tests; extend for async flows
- `pkg/commons/http/` — `chttp.SanitizeErrorBody`, `chttp.ReadLimitedBody`, `chttp.ChromaGoClientUserAgent` (reuse, do not reinvent)

### Repo conventions
- `CLAUDE.md` — project conventions (radical simplicity, no panics in production, V2-first, build-tag segregation, minimal deps)
- `.planning/codebase/TESTING.md` — `ef` build-tag expectations and test layout

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable assets
- `chttp.ReadLimitedBody`, `chttp.SanitizeErrorBody`, `chttp.ChromaGoClientUserAgent` — reused verbatim for task endpoint HTTP handling and error-body sanitization
- `embeddings.NewValidator()` — struct validation pattern, reuse for any new validated fields on `TwelveLabsClient`
- `embeddingFromResponse` helper in `twelvelabs.go` — can be adapted (or a sibling helper added) for extracting the embedding from the task retrieval response body
- `float64sToFloat32s` — direct reuse for float64→float32 conversion on task results

### Established patterns
- Functional options via `Option func(p *TwelveLabsClient) error` with validation inside each option
- Config round-trip via `GetConfig()` + `NewTwelveLabsEmbeddingFunctionFromConfig` — new keys must be readable and writable on both sides and must omit defaults to keep config maps lean
- Test construction bypasses the public API: `newTestEF` builds a `TwelveLabsEmbeddingFunction` with a hand-crafted `TwelveLabsClient` — the same approach works for setting ms-scale polling intervals in tests
- Error messages wrap with `errors.Wrap` / `errors.Errorf` from `github.com/pkg/errors`; no stdlib `%w` wrapping in this package (don't mix)
- `resolveModel(ctx)` pattern: context-keyed per-request overrides — not needed for polling but good reference if per-call async overrides are ever considered in a future phase

### Integration points
- `content.go` modality switch in `contentToRequest` is the natural decision point for "sync vs. tasks-endpoint" routing — branch on `part.Modality` + `apiClient.asyncPollingEnabled`
- `doPost` currently handles sync POST; the async path needs a parallel `doTaskPost` + `doTaskGet` pair or a unified helper, as long as URL construction stays clear
- `Capabilities()` continues to report `SupportsBatch: false, SupportsMixedPart: false` — async doesn't change capability semantics

</code_context>

<specifics>
## Specific Ideas

- Async is framed as a **timeout-bug-fix**, not async-DX. This explicitly rejected framing is recorded because it shaped every other decision: the public surface stayed minimal, no task handles were added, polling remained hidden, and test coverage was scoped to the four required flows rather than DX richness.
- The `/embed-v2` sync endpoint is intentionally **not** altered to detect-and-redirect to the tasks endpoint. Twelve Labs does not document a sync-to-async fallback, and building one on top of 4xx error-code sniffing would tie the SDK to an undocumented server contract.
- `maxWait` existing alongside `ctx.Deadline` is intentional belt-and-suspenders. The common footgun is `Collection.Add(context.Background(), ...)`: without `maxWait`, that would silently block for up to 4 hours. With `maxWait`, the SDK can't accidentally hang for hours just because the caller forgot a deadline.
- The `status` field is the authoritative async discriminator (D-14). The HTTP-202 interpretation from ad-hoc web searches is speculative and not in the docs.

</specifics>

<deferred>
## Deferred Ideas

- **Callback-based async surface** — user floated during discussion ("allow users to pass a callback such that the add/embed calls return immediately and the callback gets the results when ready"). Explicitly deferred: changes the `EmbeddingFunction` contract, cross-cuts all providers, too large for a Twelve Labs phase. Future phase should approach this as an SDK-wide architecture question.
- **Path 2 — expose task lifecycle on the package (`CreateEmbeddingTask` + `TaskHandle`)** — would deliver real async DX for direct package users while leaving `Collection.Add` unchanged. Explicitly deferred in favor of Path 1. Worth revisiting if users start reporting that "blocking for 45 minutes on a single call" is the actual pain rather than "long media fails at all".
- **Path 3 — architecture ADR before any code** — rejected for this phase; the underlying server timeout pain is real enough to justify shipping Path 1 now.
- **Parallel task dispatch / batch throughput improvements** — async-task-endpoint does not help here; each task is still one input. Throughput work is its own concern (parallel goroutines dispatching sync calls) and belongs in a separate phase if requested.
- **Exposed polling knobs** (`WithAsyncPollInterval`, `WithAsyncPollBackoff`) — deliberately internal in v1. Add later only if users demonstrate a need.
- **Structured `*TaskFailedError` exported type** — considered and rejected (D-17) to keep the surface minimal. Revisit if callers need programmatic access to task ID / status / reason beyond log-message parsing.
- **Jitter in backoff** — deferred; deterministic polling makes tests simpler. Add later if server load patterns warrant it.
- **Clock abstraction for tests** — deferred. Short intervals via direct field override are sufficient; no package-wide clock interface is justified by this single polling loop.

</deferred>

---

*Phase: 26-twelve-labs-async-embedding*
*Context gathered: 2026-04-14*
