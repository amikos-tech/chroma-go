---
phase: 23
slug: ort-ef-leak-fix
status: draft
nyquist_compliant: false
wave_0_complete: true
created: 2026-04-10
---

# Phase 23 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` |
| **Config file** | `Makefile` / existing `basicv2` build tag |
| **Quick run command** | `go test -tags=basicv2 -run 'TestEmbeddedLocalClientCreateCollection_IfNotExistsExistingDoesNotOverrideState|TestEmbeddedCreateCollection_DefaultORT.*' ./pkg/api/v2/...` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~10s focused / longer for full V2 suite |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=basicv2 -run 'TestEmbeddedLocalClientCreateCollection_IfNotExistsExistingDoesNotOverrideState|TestEmbeddedCreateCollection_DefaultORT.*' ./pkg/api/v2/...`
- **After every plan wave:** Run `make test`
- **Before `/gsd-verify-work`:** `make test` and `make lint` must be green
- **Max feedback latency:** 45 seconds for focused feedback

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 23-01-01 | 01 | 1 | EFL-01 | T-23-01 / ownership confusion | Embedded `CreateCollection` closes the SDK-owned default ORT EF exactly once on the existing-collection path and returns an error if that cleanup fails | unit | `go test -tags=basicv2 -run 'TestEmbeddedCreateCollection_DefaultORT.*' ./pkg/api/v2/...` | ✅ | ⬜ pending |
| 23-01-02 | 01 | 1 | EFL-01 | T-23-01 / resource leak | Existing embedded collection state still wins after `WithIfNotExistsCreate()`; no temporary default EF is adopted into the returned collection | unit | `go test -tags=basicv2 -run 'TestEmbeddedLocalClientCreateCollection_IfNotExistsExistingDoesNotOverrideState' ./pkg/api/v2/...` | ✅ | ⬜ pending |
| 23-01-03 | 01 | 2 | EFL-01 | T-23-01 / regression safety | Embedded V2 lifecycle tests remain green after the leak fix | unit | `make test` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

All phase behaviors can be validated with automated tests in this phase.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 45s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
