---
phase: 20
slug: getorcreatecollection-contentef-support
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-07
---

# Phase 20 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | Makefile (build tags) |
| **Quick run command** | `go test -tags=basicv2 -run TestGetOrCreate ./test/client_v2/...` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=basicv2 -run TestGetOrCreate ./test/client_v2/...`
- **After every plan wave:** Run `make test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 20-01-01 | 01 | 1 | SC-1 | — | N/A | unit | `go test -tags=basicv2 -run TestCreateCollectionOp ./test/client_v2/...` | ❌ W0 | ⬜ pending |
| 20-01-02 | 01 | 1 | SC-2 | — | N/A | unit | `go test -tags=basicv2 -run TestWithContentEmbeddingFunctionCreate ./test/client_v2/...` | ❌ W0 | ⬜ pending |
| 20-01-03 | 01 | 1 | SC-3 | — | N/A | integration | `go test -tags=basicv2 -run TestGetOrCreateCollectionContentEF ./test/client_v2/...` | ❌ W0 | ⬜ pending |
| 20-01-04 | 01 | 1 | SC-4 | — | N/A | integration | `go test -tags=basicv2 -run TestGetOrCreateCollectionContentEF ./test/client_v2/...` | ❌ W0 | ⬜ pending |
| 20-01-05 | 01 | 1 | SC-5 | — | N/A | integration | `go test -tags=basicv2 -run TestGetOrCreateCollectionContentEF ./test/client_v2/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Test stubs for GetOrCreateCollection with contentEF in `test/client_v2/`
- [ ] Existing test infrastructure (testcontainers) covers all phase requirements

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
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
