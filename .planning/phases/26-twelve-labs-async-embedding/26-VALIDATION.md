---
phase: 26
slug: twelve-labs-async-embedding
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-14
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
| 26-01-01 | 01 | 1 | TLA-01..04 | — | N/A | — | (Wave 0 fixtures) | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `pkg/embeddings/twelvelabs/twelvelabs_test.go` — extend `newTestEF` to expose ms-scale polling fields (`pollInitial`, `pollMultiplier`, `pollCap`) for tests
- [ ] Shared task-mock helpers for `httptest.Server` covering: `POST /v1.3/embed-v2/tasks`, `GET /v1.3/embed-v2/tasks/{id}` (status discriminator via `status` field)
- [ ] Per D-26, fixtures for 6 flows: task-create, poll-to-ready, poll-to-failed, ctx-cancel mid-poll, maxWait expiry, config round-trip

*No framework install needed — stdlib `net/http/httptest` + existing testify.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live long-media (>5 min video) completes end-to-end via `WithAsyncPolling` | TLA-01..04 | Requires paid Twelve Labs API key and a real media URL; not runnable in CI without credentials | Set `TWELVE_LABS_API_KEY`, call `EmbedContent` with a 10-minute video URL and `WithAsyncPolling(30*time.Minute)`, confirm a non-empty embedding returns |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
