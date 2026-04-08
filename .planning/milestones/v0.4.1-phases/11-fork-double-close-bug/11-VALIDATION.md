---
phase: 11
slug: fork-double-close-bug
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-26
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify v1 |
| **Config file** | none (standard go test) |
| **Quick run command** | `go test -tags=basicv2 -run "TestFork\|TestClose" -count=1 ./pkg/api/v2/...` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=basicv2 -run "TestFork|TestClose" -count=1 ./pkg/api/v2/...`
- **After every plan wave:** Run `make test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 11-01-01 | 01 | 1 | SC-01 | unit | `go test -tags=basicv2 -run TestForkCloseDoesNotDoubleClose -count=1 ./pkg/api/v2/...` | No -- Wave 0 | ⬜ pending |
| 11-01-02 | 01 | 1 | SC-02 | unit | `go test -tags=basicv2 -run TestForkOwnership -count=1 ./pkg/api/v2/...` | No -- Wave 0 | ⬜ pending |
| 11-01-03 | 01 | 1 | SC-03 | unit | `go test -tags=basicv2 -run TestForkClose -count=1 ./pkg/api/v2/...` | No -- Wave 0 | ⬜ pending |
| 11-01-04 | 01 | 1 | SC-04 | integration | `make test` | Yes (existing) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Fork + Close double-close test (mock closeable EF, verify Close() called exactly once)
- [ ] Fork + Close with contentEmbeddingFunction test (both EF types, owner flag gating)
- [ ] Close-once wrapper unit tests (idempotent close, use-after-close returns error)
- [ ] Embedded collection fork + close test (same pattern, embedded path)

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
