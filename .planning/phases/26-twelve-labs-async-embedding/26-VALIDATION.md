---
phase: 26
slug: twelve-labs-async-embedding
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-14
validated: 2026-04-14
---

# Phase 26 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) + `httptest.Server` + `testify` |
| **Config file** | none — uses existing `ef` build tag (`Makefile`: `make test-ef`) |
| **Quick run command** | `go test -tags=ef -run TestTwelveLabs -count=1 ./pkg/embeddings/twelvelabs/...` |
| **Full suite command** | `make test-ef` |
| **Estimated runtime** | ~2 seconds (ms-scale poll intervals in tests) |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=ef -run TestTwelveLabs -count=1 ./pkg/embeddings/twelvelabs/...`
- **After every plan wave:** Run `make test-ef`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

> Populated by planner. Each task from PLAN.md must map to a test command (or be covered by Wave 0 fixture/helper setup). Execution auditor fills Status column during `/gsd-execute-phase`.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 26-01-01 | 01 | 1 | TLA-01, TLA-02, TLA-03 | T-26-01..05 | `_id` alias + list-shape `embedding_option` + polling defaults | unit (build + grep) | `go build -tags=ef ./pkg/embeddings/twelvelabs/...` plus grep asserts `_id`, `[]string`, `asyncPollInitial` | EXISTS | green |
| 26-01-02 | 01 | 1 | TLA-01, TLA-02, TLA-03 | T-26-01..05 | body sanitization on non-2xx, URL escaping, empty-id guard | unit (build + grep) | grep asserts `doTaskPost`, `doTaskGet`, `url.PathEscape`, `SanitizeErrorBody` count >= 4 | EXISTS | green |
| 26-02-01 | 02 | 2 | TLA-01, TLA-02, TLA-03 | T-26-06..10 | status discriminator, ctx-cancel, maxWait distinct, no `time.After`, no `WithTimeout` on maxWait | unit (build + grep) | grep asserts `time.NewTimer`, `case <-ctx.Done`, `async polling maxWait`, `terminal status=failed`, absence of `time.After(` and `context.WithTimeout(ctx, maxWait)` | EXISTS | green |
| 26-02-02 | 02 | 2 | TLA-01, TLA-02, TLA-03 | T-26-06..10 | modality-gated routing; sync path unchanged for non-opt-in | unit (existing tests regression) | `go test -tags=ef -count=1 -run TestTwelveLabs ./pkg/embeddings/twelvelabs/...` | EXISTS | green |
| 26-03-01 | 03 | 2 | TLA-03 | T-26-11, T-26-12 | reject negative maxWait, 30m default on zero | unit (build + grep) | grep asserts `maxWait cannot be negative`, `30 * time.Minute`, exact signature `func WithAsyncPolling(maxWait time.Duration) Option` | EXISTS | green |
| 26-03-02 | 03 | 2 | TLA-03 | T-26-13, T-26-14 | config keys round-trip; omit when disabled | unit (build + grep + regression) | grep asserts emit/read of `async_polling` + `async_max_wait_ms` + `asyncMaxWait.Milliseconds()`; existing tests still pass | EXISTS | green |
| 26-04-01 | 04 | 3 | TLA-04 | T-26-15, T-26-16 | task-create, poll-to-ready, poll-to-failed, unexpected-status | unit (httptest) | `go test -tags=ef -count=1 -run TestTwelveLabsAsyncTaskCreate\|TestTwelveLabsAsyncPollToReady\|TestTwelveLabsAsyncPollToFailed\|TestTwelveLabsAsyncUnexpectedStatus ./pkg/embeddings/twelvelabs/...` | EXISTS | green |
| 26-04-02 | 04 | 3 | TLA-04 | T-26-17 | ctx-cancel, maxWait distinct from DeadlineExceeded, text/image skip async, config round-trip | unit (httptest) | `go test -tags=ef -count=1 -run TestTwelveLabsAsyncCtxCancel\|TestTwelveLabsAsyncMaxWait\|TestTwelveLabsAsyncSkipsTextImage\|TestTwelveLabsAsyncConfigRoundTrip\|TestTwelveLabsAsyncConfigOmitWhenDisabled ./pkg/embeddings/twelvelabs/...` | EXISTS | green |

*Status: pending · green · red · flaky*

---

## Wave 0 Requirements

- [x] `pkg/embeddings/twelvelabs/twelvelabs_test.go` — sibling helper `newTestAsyncEF` wraps `newTestEF` and sets ms-scale `asyncPollInitial=1ms`, `asyncPollMultiplier=1.5`, `asyncPollCap=10ms`, `asyncMaxWait=5s` (Plan 04 Task 1)
- [x] Fixture helpers `taskCreateJSON(id, status)` and `taskGetJSON(id, status, data)` emit the Mongo-style `_id` alias (Pitfall 1 guard) — Plan 04 Task 1
- [x] Seven tests cover all 6 D-26 flows plus the D-07 text/image-skip rule and D-22 config-omit-when-disabled — Plan 04 Task 2

*No framework install needed — stdlib `net/http/httptest` + existing testify.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live long-media (>5 min video) completes end-to-end via `WithAsyncPolling` | TLA-01..04 | Requires paid Twelve Labs API key and a real media URL; not runnable in CI without credentials | Set `TWELVE_LABS_API_KEY`, call `EmbedContent` with a 10-minute video URL and `WithAsyncPolling(30*time.Minute)`, confirm a non-empty embedding returns |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (helpers + fixtures land in Plan 04 Task 1, consumed by Task 2)
- [x] No watch-mode flags
- [x] Feedback latency < 5s (ms-scale polling keeps async test suite ~2s)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved by planner 2026-04-14

---

## Validation Audit 2026-04-14

| Metric | Count |
|--------|-------|
| Tasks audited | 8 |
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

**Evidence:**

- `go test -tags=ef -count=1 -run TestTwelveLabs ./pkg/embeddings/twelvelabs/...` → PASS (0.537s)
- All 9 async httptest cases (`TestTwelveLabsAsync{TaskCreate,PollToReady,PollToFailed,UnexpectedStatus,CtxCancel,MaxWait,SkipsTextImage,ConfigRoundTrip,ConfigOmitWhenDisabled}`) green; 3 extra error-sanitization regressions (WR-02) also green
- Grep assertions verified for 26-01-01, 26-01-02, 26-02-01, 26-03-01, 26-03-02 (positive matches present, negative matches absent — no `time.After(` and no `context.WithTimeout(ctx, maxWait)` in `twelvelabs_async.go`)
- Manual-only entry (live long-media) retained — requires paid API key, not CI-runnable

Phase 26 remains `nyquist_compliant: true`. No new test files generated.

