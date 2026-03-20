---
phase: 4
slug: provider-mapping-and-explicit-failures
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-20
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (standard library + testify) |
| **Config file** | none — existing infrastructure |
| **Quick run command** | `go test -tags=basicv2 -run TestIntentMapper ./pkg/embeddings/...` |
| **Full suite command** | `go test -tags=basicv2 ./pkg/embeddings/...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=basicv2 -run TestIntentMapper ./pkg/embeddings/...`
- **After every plan wave:** Run `go test -tags=basicv2 ./pkg/embeddings/...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | MAP-01 | unit | `go test -tags=basicv2 -run TestIsNeutralIntent ./pkg/embeddings/...` | ❌ W0 | ⬜ pending |
| 04-01-02 | 01 | 1 | MAP-01 | unit | `go test -tags=basicv2 -run TestIntentMapper ./pkg/embeddings/...` | ❌ W0 | ⬜ pending |
| 04-01-03 | 01 | 1 | MAP-02 | unit | `go test -tags=basicv2 -run TestValidateContentSupport ./pkg/embeddings/...` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 2 | MAP-02 | unit | `go test -tags=basicv2 -run TestCompatAdapter ./pkg/embeddings/...` | ❌ W0 | ⬜ pending |
| 04-02-02 | 02 | 2 | MAP-01, MAP-02 | integration | `go test -tags=basicv2 -run TestMappingIntegration ./pkg/embeddings/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `pkg/embeddings/intent_mapper_test.go` — stubs for MAP-01 (IntentMapper, IsNeutralIntent)
- [ ] `pkg/embeddings/content_validate_test.go` — stubs for MAP-02 (ValidateContentSupport)

*Existing test infrastructure (testify, go test with build tags) covers all framework needs.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
