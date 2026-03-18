---
phase: 1
slug: shared-multimodal-contract
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-18
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + github.com/stretchr/testify |
| **Config file** | none |
| **Quick run command** | `go test ./pkg/embeddings ./pkg/api/v2` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./pkg/embeddings ./pkg/api/v2`
- **After every plan wave:** Run `make test`
- **Before `$gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 01-01-01 | 01 | 1 | MMOD-01 | unit | `go test ./pkg/embeddings -run '^TestMultimodalContentSupportsAllModalities$'` | ❌ W0 | ⬜ pending |
| 01-01-02 | 01 | 1 | MMOD-02 | unit | `go test ./pkg/embeddings -run '^TestMultimodalContentPreservesOrder$'` | ❌ W0 | ⬜ pending |
| 01-02-01 | 02 | 1 | MMOD-03 | unit | `go test ./pkg/embeddings -run '^TestMultimodalIntentValidation$'` | ❌ W0 | ⬜ pending |
| 01-02-02 | 02 | 1 | MMOD-04 | unit | `go test ./pkg/embeddings -run '^TestMultimodalRequestOptions$'` | ❌ W0 | ⬜ pending |
| 01-02-03 | 02 | 1 | MMOD-05 | unit | `go test ./pkg/embeddings -run '^TestMultimodalValidationErrors$'` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `pkg/embeddings/multimodal_test.go` — stubs for MMOD-01 and MMOD-02
- [ ] `pkg/embeddings/multimodal_validation_test.go` — stubs for MMOD-03, MMOD-04, and MMOD-05
- [ ] Typed validation error assertions — verify issue paths and codes, not only error strings

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
