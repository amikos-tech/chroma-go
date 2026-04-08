---
phase: 3
slug: registry-and-config-integration
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-20
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify (assert/require) |
| **Config file** | none — build tags control suite selection |
| **Quick run command** | `go test ./pkg/embeddings/ -run TestRegister` |
| **Full suite command** | `go test ./pkg/embeddings/... ./pkg/api/v2/...` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./pkg/embeddings/ -run TestRegister`
- **After every plan wave:** Run `go test ./pkg/embeddings/... ./pkg/api/v2/...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | REG-01 | unit | `go test ./pkg/embeddings/ -run TestRegisterAndBuildContent` | ❌ W0 | ⬜ pending |
| 03-01-02 | 01 | 1 | REG-01 | unit | `go test ./pkg/embeddings/ -run TestBuildContentFallback` | ❌ W0 | ⬜ pending |
| 03-02-01 | 02 | 2 | REG-01 | unit | `go test ./pkg/api/v2/ -run TestBuildContentEFFromConfig` | ❌ W0 | ⬜ pending |
| 03-02-02 | 02 | 2 | REG-01 | unit | `go test ./pkg/embeddings/ -run TestContentConfigRoundTrip` | ❌ W0 | ⬜ pending |
| 03-02-03 | 02 | 2 | REG-02 | unit | `go test ./pkg/api/v2/ -run TestBuildEmbeddingFunctionFromConfig` | ❌ W0 | ⬜ pending |
| 03-02-04 | 02 | 2 | REG-02 | unit | `go test ./pkg/api/v2/ -run TestBuildEFFromConfigMultimodalFallback` | ❌ W0 | ⬜ pending |
| 03-03-01 | 03 | 2 | REG-02 | unit | `go test ./pkg/api/v2/ -run TestAutoWiring` | ❌ W0 | ⬜ pending |
| 03-03-02 | 03 | 2 | REG-02 | unit | `go test ./pkg/api/v2/ -run TestWithContentEmbeddingFunction` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `pkg/embeddings/registry_test.go` — extend with content map tests (REG-01 registry functions)
- [ ] `pkg/api/v2/configuration_test.go` (new) — covers BuildContentEFFromConfig, BuildEmbeddingFunctionFromConfig multimodal fallback, SetContentEmbeddingFunction (REG-01, REG-02)
- [ ] `pkg/api/v2/collection_content_test.go` (new) — covers auto-wiring, WithContentEmbeddingFunction, priority logic (REG-02)

*Existing `go test` + `testify` infrastructure is sufficient.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
