---
phase: 16
slug: twelve-labs-embedding-function
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-01
---

# Phase 16 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify |
| **Config file** | none (build tags in file headers) |
| **Quick run command** | `go test -tags=ef -run TestTwelveLabs -count=1 ./pkg/embeddings/twelvelabs/...` |
| **Full suite command** | `go test -tags=ef -count=1 ./pkg/embeddings/twelvelabs/...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=ef -run TestTwelveLabs -count=1 ./pkg/embeddings/twelvelabs/...`
- **After every plan wave:** Run `go test -tags=ef -count=1 ./pkg/embeddings/twelvelabs/...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 16-01-01 | 01 | 1 | SC-1 | unit | `go test -tags=ef -run TestTwelveLabsEmbed -count=1 ./pkg/embeddings/twelvelabs/...` | ❌ W0 | ⬜ pending |
| 16-01-02 | 01 | 1 | SC-2 | unit | `go test -tags=ef -run TestTwelveLabsModality -count=1 ./pkg/embeddings/twelvelabs/...` | ❌ W0 | ⬜ pending |
| 16-01-03 | 01 | 1 | SC-3 | unit | `go test -tags=ef -run TestTwelveLabsRegistry -count=1 ./pkg/embeddings/twelvelabs/...` | ❌ W0 | ⬜ pending |
| 16-01-04 | 01 | 1 | SC-4 | unit | `go test -tags=ef -count=1 ./pkg/embeddings/twelvelabs/...` | ❌ W0 | ⬜ pending |
| 16-01-05 | 01 | 1 | SC-5 | manual | N/A | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `pkg/embeddings/twelvelabs/twelvelabs_test.go` — httptest unit tests for text embedding, registry, config round-trip
- [ ] `pkg/embeddings/twelvelabs/twelvelabs_content_test.go` — Content API unit tests for multimodal (image, audio, video)
- [ ] All test files need `//go:build ef` build tag

*Existing infrastructure covers test framework — only test files needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Docs and examples | SC-5 | Documentation correctness requires human review | Verify `docs/docs/embeddings.md` has Twelve Labs section, example in `examples/v2/` runs |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
