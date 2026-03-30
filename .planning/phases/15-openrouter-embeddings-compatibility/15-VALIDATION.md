---
phase: 15
slug: openrouter-embeddings-compatibility
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-30
---

# Phase 15 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify |
| **Config file** | Makefile targets with build tags |
| **Quick run command** | `go test -tags=ef -run TestOpenRouter -count=1 ./pkg/embeddings/openrouter/...` |
| **Full suite command** | `make test-ef` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=ef -count=1 ./pkg/embeddings/openrouter/... ./pkg/embeddings/openai/...`
- **After every plan wave:** Run `make test-ef`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 15-01-01 | 01 | 1 | SC-1 | unit | `go test -tags=ef -run TestRequestSerialization ./pkg/embeddings/openrouter/...` | ❌ W0 | ⬜ pending |
| 15-01-02 | 01 | 1 | SC-3 | unit | `go test -tags=ef -run TestProviderPreferences ./pkg/embeddings/openrouter/...` | ❌ W0 | ⬜ pending |
| 15-01-03 | 01 | 1 | SC-5 | unit | `go test -tags=ef -run TestConfigRoundTrip ./pkg/embeddings/openrouter/...` | ❌ W0 | ⬜ pending |
| 15-02-01 | 02 | 1 | SC-2 | unit | `go test -tags=ef -run TestModelString ./pkg/embeddings/openai/...` | ❌ W0 | ⬜ pending |
| 15-xx-xx | all | all | SC-4 | regression | `go test -tags=ef ./pkg/embeddings/openai/...` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `pkg/embeddings/openrouter/openrouter_test.go` — stubs for SC-1, SC-3, SC-5
- [ ] OpenAI test for `WithModelString` in `pkg/embeddings/openai/` — covers SC-2
- [ ] Framework install: none needed (Go testing + testify already in go.mod)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| OpenRouter live embedding call | SC-1 | Requires API key and network | Set `OPENROUTER_API_KEY`, run integration test |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
