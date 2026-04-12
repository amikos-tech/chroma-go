---
phase: 24
slug: getorcreatecollection-ef-safety
status: draft
nyquist_compliant: false
wave_0_complete: true
created: 2026-04-12
---

# Phase 24 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` |
| **Config file** | `Makefile` / existing `basicv2` build tag |
| **Quick run command** | `go test -tags=basicv2 -run 'TestEmbedded(LocalClient)?GetOrCreateCollection.*|TestEmbeddedGetCollection_Race.*' ./pkg/api/v2/...` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~15s focused / longer for full V2 suite |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=basicv2 -run 'TestEmbedded(LocalClient)?GetOrCreateCollection.*|TestEmbeddedGetCollection_Race.*' ./pkg/api/v2/...`
- **After every plan wave:** Run `make test`
- **Before `/gsd-verify-work`:** `make test` and `make lint` must be green
- **Max feedback latency:** 60 seconds for focused feedback

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 24-01-01 | 01 | 1 | EFL-02 | T-24-01 | Provisional embedded `GetCollection` cleanup never closes caller-provided dense/content EFs before `GetOrCreateCollection` fallback completes | unit | `go test -tags=basicv2 -run 'TestEmbedded(LocalClient)?GetOrCreateCollection.*' ./pkg/api/v2/...` | ✅ | ⬜ pending |
| 24-01-02 | 01 | 1 | EFL-02 | T-24-03 | Shared dense/content and dual-interface ownership paths remain single-close after the ownership-aware cleanup change | unit | `go test -tags=basicv2 -run 'TestEmbedded(LocalClient)?GetOrCreateCollection.*|TestEmbeddedGetCollection_Race.*|TestCollectionImpl_Close_SkipsSharedDenseClose.*' ./pkg/api/v2/...` | ✅ | ⬜ pending |
| 24-01-03 | 01 | 2 | EFL-03 | T-24-02 | Concurrent `GetOrCreateCollection` miss/create races pass under `-race` and return usable collections without double-close or panic | unit | `go test -race -tags=basicv2 -run 'TestEmbedded(LocalClient)?GetOrCreateCollection.*Race.*' ./pkg/api/v2/...` | ✅ | ⬜ pending |
| 24-01-04 | 01 | 3 | EFL-02, EFL-03 | T-24-01 / T-24-02 / T-24-03 | Embedded V2 lifecycle suite remains green after the ownership and convergence fix | unit | `make test` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

All Phase 24 behaviors should be automatable in the colocated `basicv2` suite.

---

## Validation Sign-Off

- [x] All tasks have automated verification
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all missing references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
