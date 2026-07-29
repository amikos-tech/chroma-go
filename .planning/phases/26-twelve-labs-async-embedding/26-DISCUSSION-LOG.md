# Phase 26: Twelve Labs Async Embedding - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-14
**Phase:** 26-twelve-labs-async-embedding
**Areas discussed:** Framing (premise check), API surface, Polling schedule, Async status detection, Opt-in vs auto polling, Error semantics, Test strategy

**Advisor mode:** active (USER-PROFILE.md present; calibration tier = full_maturity / thorough-evaluator)

---

## 0. Framing / Premise Check (not in original gray-area list)

Surfaced by advisor research: the phase goal in ROADMAP.md ("sync endpoint returns an async task response") did not match the actual Twelve Labs API contract. The `POST /v1.3/embed-v2` endpoint always returns `EmbeddingSuccessResponse` — there is no sync-to-async fallback. Async work happens on the distinct `POST /v1.3/embed-v2/tasks` endpoint.

| Option | Description | Selected |
|--------|-------------|----------|
| A. Two-endpoint model | Sync `/embed-v2` unchanged; new code path calls `/embed-v2/tasks` + poll | ✓ |
| B. "Try sync, fall back to async" | Detect a specific 4xx on sync and retry via tasks endpoint | |

**User's choice:** A (Framing A — two-endpoint model).
**Notes:** User questioned whether the audio/video modality split made sense for large text/image batches; this surfaced that async isn't a batching primitive but a latency-decoupling one. Led to a deeper discussion of what async should actually deliver at the DX level.

---

## 1. Path framing (what does Phase 26 actually deliver?)

After the modality discussion, it became clear that "hidden polling in EmbedContent" doesn't deliver true async DX — it just extends the blocking wait. Three honest paths were laid out.

| Option | Description | Selected |
|--------|-------------|----------|
| Path 1 — timeout bug-fix | Hidden polling inside existing EmbedContent; long media just works; no new public async surface | ✓ |
| Path 2 — expose task lifecycle | Add package-level `CreateEmbeddingTask` + `TaskHandle.Wait/Status/Result`; real async for direct users; Collection.Add still sync-only | |
| Path 3 — defer and write ADR | Write design doc first, ship no code this phase | |

**User's choice:** Path 1.
**Notes:** User wanted to unlock TwelveLabs' long-media capability this milestone; Path 2 is deferred pending evidence that callers need fire-and-forget / parallel dispatch. Callback-pattern idea floated by user during discussion was explicitly deferred (cross-cutting interface change).

---

## 2. Async routing trigger (under Path 1 + Framing A)

Once Path 1 was locked, the routing trigger question simplified.

| Option | Description | Selected |
|--------|-------------|----------|
| A2a. Modality-based | Audio+video always async when opt-in enabled; text/image always sync | ✓ |
| A2b. Sync-first, async fallback | Try `/embed-v2`, fall back to `/tasks` on duration error | |
| A2c. Per-request context key | `ContextWithAsyncPolling(ctx, maxWait)` on any call | |
| A2d. Always-on when opt-in | All modalities route through `/tasks` when enabled | |

**User's choice:** A2a.
**Notes:** Text/image have no duration concept on Twelve Labs; routing them through `/tasks` would add a task-create round-trip for zero benefit. A2b depends on undocumented error-code sniffing.

---

## 3. Public surface / API delta

| Option | Description | Selected |
|--------|-------------|----------|
| Minimal: single `WithAsyncPolling(maxWait)` option, everything else hidden | One knob; polling intervals, detection, error shape all internal | ✓ |
| Rich: expose polling interval/backoff options + structured error type | More knobs, more programmatic access | |

**User's choice:** Minimal.
**Notes:** User explicitly locked this: "I want users to explicitly use WithAsyncPolling flag to trigger the async flow and all the rest of the details are hidden." This drove D-07 (plain `errors.Errorf` instead of exported `*TaskFailedError`) and kept polling fields unexported.

---

## 4. Polling schedule

| Option | Description | Selected |
|--------|-------------|----------|
| Fixed interval (e.g., 5s) | Matches TwelveLabs Python SDK; simplest | |
| Hand-rolled capped exponential (2s → 60s, mult 1.5, no jitter) | Matches AssemblyAI shape; no deps; ~30 LoC | ✓ |
| `cenkalti/backoff` v4 | Battle-tested but adds dep | |
| Linear ramp | Non-standard | |
| Server-hinted (Retry-After) | Not supported by TwelveLabs | |

**User's choice:** Hand-rolled capped exponential; internal constants, not exposed as options.
**Notes:** Aligns with codebase's stdlib-first ethos; ctx.Deadline + maxWait are the only user-visible kill switches.

---

## 5. Async status detection (inside polling loop)

| Option | Description | Selected |
|--------|-------------|----------|
| `status` enum field on task response | `processing` / `ready` / `failed` — matches official SDK Fern discriminator | ✓ |
| HTTP status code (200 vs 202) | Not documented by TwelveLabs; speculative | |
| `id` field presence | Weaker discriminator than `status` | |
| Separate Go methods per endpoint (no detection) | Bigger surface; unnecessary under Path 1 | |

**User's choice:** `status` enum.
**Notes:** Unknown status values treated as malformed (D-16), not silently coerced.

---

## 6. Error semantics (terminal task failures)

| Option | Description | Selected |
|--------|-------------|----------|
| Plain `errors.Errorf` with task ID + status + sanitized reason in message | No new exported surface; matches existing `twelvelabs.go:185` style | ✓ |
| Sentinel `ErrTwelveLabsTaskFailed` + `%w` wrap | Enables `errors.Is` but conflicts with package's pkg/errors.Wrap style | |
| Structured `*TaskFailedError{TaskID, Status, Reason}` + `errors.As` | Programmatic access but exports a new type | |
| Hybrid sentinel + struct | Covers both patterns but largest surface | |

**User's choice:** Plain `errors.Errorf`.
**Notes:** Changed from initial structured-type recommendation after user's "hide all details except WithAsyncPolling" instruction. If callers later need programmatic access, add the type in a follow-up.

---

## 7. Test strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Short intervals via direct field override (ms-scale in tests, seconds in prod) | Matches existing `newTestEF` pattern; no new deps | ✓ |
| Internal clock interface with fake/real impls | Fully deterministic but adds abstraction for one loop | |
| `benbjohnson/clock` external dep | Overkill; dep rot risk | |
| Package-var sleep seam | Anti-pattern; blocks `t.Parallel` | |

**User's choice:** Short intervals via direct field override.
**Notes:** Polling fields added as unexported struct fields on `TwelveLabsClient`, set directly in test construction like existing `BaseAPI` / `APIKey`.

---

## Deferred Ideas

Captured in CONTEXT.md `<deferred>` section:
- Callback-based async surface (user-floated, cross-cuts interface contract)
- Path 2 (exposed `TaskHandle` lifecycle) — revisit if "blocking for 45 minutes" becomes a complaint
- Path 3 (design ADR before code) — rejected in favor of shipping Path 1 now
- Parallel task dispatch / batch throughput (separate concern, async doesn't solve it)
- Exposed polling knobs (internal in v1)
- Structured `*TaskFailedError` exported type
- Jitter in backoff
- Clock abstraction for tests
