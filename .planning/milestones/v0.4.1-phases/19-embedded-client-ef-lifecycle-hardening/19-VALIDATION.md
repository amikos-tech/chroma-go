---
phase: 19
slug: embedded-client-ef-lifecycle-hardening
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-06
---

# Phase 19 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify (assert/require) |
| **Config file** | Makefile (build tags: basicv2) |
| **Quick run command** | `go test -tags=basicv2 -run TestEmbedded -count=1 ./pkg/api/v2/...` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=basicv2 -count=1 ./pkg/api/v2/...`
- **After every plan wave:** Run `make test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 19-01-01 | 01 | 1 | SC-01 | — | N/A | unit | `go test -tags=basicv2 -run TestEmbeddedGetCollection_ConcurrentAutoWire -count=1 ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 19-01-02 | 01 | 1 | SC-02 | — | N/A | unit | `go test -tags=basicv2 -run TestEmbeddedDeleteCollectionState_ClosesEFs -count=1 ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 19-01-03 | 01 | 1 | SC-03 | — | N/A | unit | `go test -tags=basicv2 -run TestEmbeddedLocalClient_Close_CleansUpCollectionState -count=1 ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 19-01-04 | 01 | 1 | SC-04 | — | N/A | unit | `go test -tags=basicv2 -run TestDeleteCollectionFromCache_EmbeddedCollection -count=1 ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 19-01-05 | 01 | 1 | SC-05 | — | N/A | unit | `go test -tags=basicv2 -run TestEmbeddedBuildCollection_CloseOnceWrapping -count=1 ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 19-01-06 | 01 | 1 | SC-06 | — | N/A | unit | `go test -tags=basicv2 -run TestIsDenseEFSharedWithContent_SymmetricUnwrap -count=1 ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 19-01-07 | 01 | 1 | SC-07 | — | N/A | unit | `go test -tags=basicv2 -run TestEmbeddedGetCollection_BuildErrorGuard -count=1 ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 19-01-08 | 01 | 1 | SC-08 | — | N/A | unit | `go test -tags=basicv2 -run TestEmbeddedClient_LoggerReceivesErrors -count=1 ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 19-01-09 | 01 | 1 | SC-09 | — | N/A | regression | `make test` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `pkg/api/v2/client_local_embedded_test.go` — test stubs for SC-01 through SC-08
- [ ] Existing mock types (`mockCloseableEF`, `mockCloseableContentEF`, etc.) are sufficient

*Existing infrastructure covers most requirements. New test functions are created during implementation.*

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
