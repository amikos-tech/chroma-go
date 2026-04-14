---
phase: 26-twelve-labs-async-embedding
plan: 03
subsystem: embeddings
tags: [twelvelabs, embeddings, async, options, config-roundtrip]

# Dependency graph
requires:
  - phase: 26-twelve-labs-async-embedding
    provides: asyncPollingEnabled / asyncMaxWait fields on TwelveLabsClient (Plan 26-01)
provides:
  - WithAsyncPolling(maxWait time.Duration) Option — sole public async trigger (D-03, D-04)
  - Negative maxWait rejected with "maxWait cannot be negative" error
  - maxWait=0 selects 30-minute default
  - GetConfig emits async_polling + async_max_wait_ms iff asyncPollingEnabled (D-21, D-22)
  - NewTwelveLabsEmbeddingFunctionFromConfig round-trip via embeddings.ConfigInt (D-23)
  - Malformed async_max_wait_ms with async_polling=true → async stays off (no silent 30-min default)
affects: [26-04-content-routing-and-tests]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Functional Option that validates input and sets internal fields on the client (follows existing WithAudioEmbeddingOption pattern)"
    - "Conditional config emission — async keys are omitted entirely when the feature is off so default-config round-trips stay byte-identical"
    - "Missing-key-OR-malformed-value → do not enable feature (avoids silent 30-minute default when caller supplied broken input)"
    - "Reuse shared embeddings.ConfigInt helper for numeric-type coercion across int / int64 / float64"

key-files:
  created: []
  modified:
    - pkg/embeddings/twelvelabs/option.go
    - pkg/embeddings/twelvelabs/twelvelabs.go

key-decisions:
  - "Reject negative maxWait in WithAsyncPolling with errors.New instead of clamping — matches existing option validation style (WithAPIKey, WithModel) and prevents immediately-expired deadlines that would surface as confusing maxWait-exceeded errors on first poll."
  - "Emit async_max_wait_ms using time.Duration.Milliseconds() which returns int64 directly — cleaner than manual division and matches CONTEXT D-21."
  - "Treat malformed async_max_wait_ms with async_polling=true as a broken round-trip and leave async OFF rather than falling back to WithAsyncPolling(0) — the 30-minute default is a caller-opt-in value, not a fallback for bad data. Silent fallback would mask registry corruption."
  - "Did NOT remove //nolint:unused from asyncPollingEnabled / asyncMaxWait in twelvelabs.go — the linter accepts the current state; Plan 26-02's polling loop consumes these fields directly, so the annotation will naturally drop when Plan 26-02 lands rather than churn twice."
  - "Reused embeddings.ConfigInt at pkg/embeddings/embedding.go:674 instead of reimplementing a type switch — this helper already handles int/int64/float64 coercion for JSON round-trips (Pitfall 6)."

patterns-established:
  - "Option validates + sets client field directly; no allocation, no side-channel state"
  - "Config emit-iff-enabled, read-iff-both-keys-valid, skip on missing-or-malformed"

requirements-completed: [TLA-03]

# Metrics
duration: ~2min
completed: 2026-04-14
---

# Phase 26 Plan 03: Async Option + Config Round-Trip Summary

**Single public async trigger `WithAsyncPolling(maxWait)` plus lossless config round-trip via `async_polling` / `async_max_wait_ms` keys — the only user-visible surface Phase 26 exposes, wired without touching Plan 26-01's foundation fields or Plan 26-02's polling surface.**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-04-14T09:30:01Z
- **Completed:** 2026-04-14T09:31:35Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- `WithAsyncPolling(time.Duration)` added to `option.go` as the sole public trigger for async (D-03, D-04). Negative input → explicit validation error. Zero → 30-minute default. Positive → verbatim.
- `GetConfig` now emits `async_polling: true` and `async_max_wait_ms: <int64 ms>` together, but only when the feature is enabled — non-opt-in callers see byte-identical configs to the pre-Phase-26 state (D-22).
- `NewTwelveLabsEmbeddingFunctionFromConfig` reconstructs `WithAsyncPolling(maxWait)` from the two keys using the existing `embeddings.ConfigInt` helper so int / int64 / float64 all coerce correctly across a JSON round-trip (D-23).
- Malformed `async_max_wait_ms` with `async_polling=true` deliberately leaves async OFF instead of falling back to `WithAsyncPolling(0)` — the 30-minute default is an opt-in value, not a recovery path for broken config.

## Task Commits

1. **Task 1: Add WithAsyncPolling option** — `4f24085` (feat)
2. **Task 2: Wire async config keys into GetConfig and FromConfig** — `83e2e8e` (feat)

## Files Created/Modified

- `pkg/embeddings/twelvelabs/option.go` — added `"time"` import; appended `WithAsyncPolling` (20 lines).
- `pkg/embeddings/twelvelabs/twelvelabs.go` — `GetConfig` appends two keys when `asyncPollingEnabled`; `NewTwelveLabsEmbeddingFunctionFromConfig` appends `WithAsyncPolling(ms)` when both keys parse cleanly (14 lines).

## Decisions Made

- Validated the `embeddings.ConfigInt` helper already exists at `pkg/embeddings/embedding.go:674` before following the plan's instruction to use it — confirmed the signature `(int, bool)` and that it handles `int`, `int64`, and `float64` cases. No need to reimplement numeric parsing.
- Kept the `//nolint:unused` pragmas on the polling fields in Plan 26-01 because the linter is satisfied (0 issues). Plan 26-02's polling loop will consume the fields directly; removing the pragmas now would just re-churn them.

## Deviations from Plan

None — plan executed exactly as written. All grep acceptance criteria matched on first run, build green, lint green, existing `TestTwelveLabs*` suite green.

## Issues Encountered

- PreToolUse read-before-edit hook fired after each Edit call despite the file being read in-session. Edits succeeded regardless; no workaround required.

## User Setup Required

None — no env-var or external service changes in this plan.

## Next Phase Readiness

- **Plan 26-04 (tests + content routing)** can now:
  - Construct EFs with `WithAsyncPolling(5*time.Minute)` and assert `asyncPollingEnabled==true` / `asyncMaxWait==5m`.
  - Assert `WithAsyncPolling(-1*time.Second)` returns `"maxWait cannot be negative"`.
  - Assert `WithAsyncPolling(0)` sets `asyncMaxWait == 30*time.Minute`.
  - Round-trip a config map: build → `GetConfig()` → `NewTwelveLabsEmbeddingFunctionFromConfig()` → compare `asyncPollingEnabled` and `asyncMaxWait`.
  - Assert a config with `async_polling=true` but `async_max_wait_ms="not a number"` produces an EF with `asyncPollingEnabled==false`.
  - Assert `GetConfig()` on a default (non-async) EF emits neither key.
- No blocking dependency on Plan 26-02; these plans touched disjoint surfaces (option.go + GetConfig/FromConfig here vs. twelvelabs_async.go + content.go there).

## Verification Log

- `go build ./pkg/embeddings/twelvelabs/...` — exit 0
- `go build -tags=ef ./pkg/embeddings/twelvelabs/...` — exit 0
- `go vet -tags=ef ./pkg/embeddings/twelvelabs/...` — exit 0
- `make lint` — 0 issues
- `go test -tags=ef -count=1 -run TestTwelveLabs ./pkg/embeddings/twelvelabs/...` — ok (0.510s)
- `go test -tags=ef -count=1 ./pkg/embeddings/twelvelabs/...` — ok (0.309s)
- All 8 grep acceptance criteria (Task 1: 4, Task 2: 7 positive + 1 negative `!grep`) matched.

## Self-Check: PASSED

- File `pkg/embeddings/twelvelabs/option.go` exists and contains `func WithAsyncPolling(maxWait time.Duration) Option`, `"maxWait cannot be negative"`, `30 * time.Minute`, `asyncPollingEnabled = true`.
- File `pkg/embeddings/twelvelabs/twelvelabs.go` exists and contains `cfg["async_polling"] = true`, `asyncMaxWait.Milliseconds()`, `cfg["async_polling"].(bool)`, `embeddings.ConfigInt(cfg, "async_max_wait_ms")`, `WithAsyncPolling(time.Duration(ms)`.
- Commit `4f24085` present in `git log`.
- Commit `83e2e8e` present in `git log`.

---
*Phase: 26-twelve-labs-async-embedding*
*Completed: 2026-04-14*
