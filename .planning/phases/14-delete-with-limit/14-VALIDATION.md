---
phase: 14
slug: delete-with-limit
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-29
---

# Phase 14 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify |
| **Config file** | Makefile (build tags: basicv2) |
| **Quick run command** | `go test -tags=basicv2 -run TestDelete ./pkg/api/v2/...` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=basicv2 -run TestDelete ./pkg/api/v2/... && make lint`
- **After every plan wave:** Run `make test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 14-01-01 | 01 | 1 | D-01 | unit | `go test -tags=basicv2 -run TestLimitOption.*Delete ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 14-01-02 | 01 | 1 | D-05a | unit | `go test -tags=basicv2 -run TestDeleteLimitWithoutFilter ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 14-01-03 | 01 | 1 | D-05b | unit | `go test -tags=basicv2 -run TestDeleteLimitZero ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 14-01-04 | 01 | 1 | D-02 | unit | `go test -tags=basicv2 -run TestCollectionDelete.*limit ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 14-01-05 | 01 | 1 | D-04 | unit | `go test -tags=basicv2 -run TestDeleteEmbedded.*limit ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- None — existing test infrastructure (`collection_http_test.go`, `options_test.go`) covers the patterns needed. New test cases follow established table-driven patterns.

*Existing infrastructure covers all phase requirements.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
