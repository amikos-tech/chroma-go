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
| **Quick run command** | `go test ./pkg/embeddings -run 'TestMultimodalContentSupportsAllModalities|TestMultimodalContentPreservesOrder|TestMultimodalRequestOptions|TestMultimodalIntentValidation|TestMultimodalValidationErrors|TestNewImagePartFromImageInput' && go test -tags=basicv2 ./pkg/api/v2 -run '^TestBuildEmbeddingFunctionFromConfig$'` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~20 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./pkg/embeddings -run 'TestMultimodalContentSupportsAllModalities|TestMultimodalContentPreservesOrder|TestMultimodalRequestOptions|TestMultimodalIntentValidation|TestMultimodalValidationErrors|TestNewImagePartFromImageInput' && go test -tags=basicv2 ./pkg/api/v2 -run '^TestBuildEmbeddingFunctionFromConfig$'`
- **After every plan wave:** Run `go test ./pkg/embeddings -run 'TestMultimodalContentSupportsAllModalities|TestMultimodalContentPreservesOrder|TestMultimodalRequestOptions|TestMultimodalIntentValidation|TestMultimodalValidationErrors|TestNewImagePartFromImageInput' && go test -tags=basicv2 ./pkg/api/v2 -run '^TestBuildEmbeddingFunctionFromConfig$' && make test`
- **Before `$gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 20 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 01-00-01 | 00 | 0 | MMOD-01, MMOD-02, MMOD-04 | unit stub | `go test ./pkg/embeddings -run 'TestMultimodalContentSupportsAllModalities|TestMultimodalContentPreservesOrder|TestMultimodalRequestOptions'` | ❌ planned in 01-00 | ⬜ pending |
| 01-00-02 | 00 | 0 | MMOD-03, MMOD-05 | unit stub | `go test ./pkg/embeddings -run 'TestMultimodalIntentValidation|TestMultimodalValidationErrors|TestNewImagePartFromImageInput'` | ❌ planned in 01-00 | ⬜ pending |
| 01-03-01 | 03 | 3 | MMOD-01, MMOD-02, MMOD-04 | unit | `rg -n "func TestMultimodalContentSupportsAllModalities|func TestMultimodalContentPreservesOrder|func TestMultimodalRequestOptions" pkg/embeddings/multimodal_test.go && go test ./pkg/embeddings -run 'TestMultimodalContentSupportsAllModalities|TestMultimodalContentPreservesOrder|TestMultimodalRequestOptions'` | ❌ planned in 01-03 | ⬜ pending |
| 01-03-02 | 03 | 3 | MMOD-03, MMOD-05 | unit | `rg -n "func TestMultimodalIntentValidation|func TestMultimodalValidationErrors|func TestNewImagePartFromImageInput|require\\.ErrorAs" pkg/embeddings/multimodal_validation_test.go && go test ./pkg/embeddings -run 'TestMultimodalIntentValidation|TestMultimodalValidationErrors|TestNewImagePartFromImageInput'` | ❌ planned in 01-03 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `pkg/embeddings/multimodal_test.go` — compileable Wave 0 stubs created by plan `01-00`
- [ ] `pkg/embeddings/multimodal_validation_test.go` — compileable Wave 0 stubs created by plan `01-00`
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
- [ ] Feedback latency < 20s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
