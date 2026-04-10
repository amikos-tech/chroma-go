---
phase: 22
slug: withgroupby-validation
status: draft
nyquist_compliant: false
wave_0_complete: true
created: 2026-04-09
---

# Phase 22 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` |
| **Config file** | `Makefile` / existing `basicv2` build tag |
| **Quick run command** | `go test -tags=basicv2 -run 'TestWithGroupBy|TestSearchRequestWithGroupBy' ./pkg/api/v2/...` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~5s focused / longer for full V2 suite |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=basicv2 -run 'TestWithGroupBy|TestSearchRequestWithGroupBy' ./pkg/api/v2/...`
- **After every plan wave:** Run `make test`
- **Before `/gsd-verify-work`:** `make test` and `make lint` must be green
- **Max feedback latency:** 30 seconds for focused feedback

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 22-01-01 | 01 | 1 | GRP-01 | T-22-01 / — | Explicit `WithGroupBy(nil)` is rejected before request mutation or send | unit | `go test -tags=basicv2 -run TestWithGroupBy ./pkg/api/v2/...` | ✅ | ⬜ pending |
| 22-01-02 | 01 | 1 | GRP-01 | T-22-01 / — | `NewSearchRequest(..., WithGroupBy(nil))` returns the same validation error and appends no search | unit | `go test -tags=basicv2 -run TestSearchRequestWithGroupBy ./pkg/api/v2/...` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
