---
phase: 6
slug: gemini-multimodal-adoption
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-20
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | Makefile (test-ef target) |
| **Quick run command** | `go test -tags=ef -run TestGemini -count=1 ./pkg/embeddings/gemini/...` |
| **Full suite command** | `go test -tags=ef -count=1 ./pkg/embeddings/gemini/...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=ef -run TestGemini -count=1 ./pkg/embeddings/gemini/...`
- **After every plan wave:** Run `go test -tags=ef -count=1 ./pkg/embeddings/gemini/...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 06-01-01 | 01 | 1 | GEM-01 | unit | `go test -tags=ef -run TestContentConversion ./pkg/embeddings/gemini/...` | ❌ W0 | ⬜ pending |
| 06-01-02 | 01 | 1 | GEM-01 | unit | `go test -tags=ef -run TestCapabilities ./pkg/embeddings/gemini/...` | ❌ W0 | ⬜ pending |
| 06-02-01 | 02 | 1 | GEM-02 | unit | `go test -tags=ef -run TestIntentMapping ./pkg/embeddings/gemini/...` | ❌ W0 | ⬜ pending |
| 06-03-01 | 03 | 2 | GEM-03 | unit | `go test -tags=ef -run TestRegistry ./pkg/embeddings/gemini/...` | ❌ W0 | ⬜ pending |
| 06-03-02 | 03 | 2 | GEM-03 | unit | `go test -tags=ef -run TestConfigRoundTrip ./pkg/embeddings/gemini/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `pkg/embeddings/gemini/content_test.go` — stubs for GEM-01 (content conversion, capabilities)
- [ ] `pkg/embeddings/gemini/intent_test.go` — stubs for GEM-02 (intent mapping)
- [ ] `pkg/embeddings/gemini/registry_test.go` — stubs for GEM-03 (registry, config round-trip)

*Existing go test infrastructure covers framework requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Backward compat: existing EmbedDocuments/EmbedQuery unchanged | GEM-01 | Requires running against live Gemini API | Run existing ef tests with GEMINI_API_KEY set |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
