---
phase: 7
slug: voyage-multimodal-adoption
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-22
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test with build tags |
| **Config file** | Makefile (test-ef target) |
| **Quick run command** | `go test -tags=ef -run TestVoyageAI -v ./test/...` |
| **Full suite command** | `make test-ef` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=ef -run TestVoyageAI -v ./test/...`
- **After every plan wave:** Run `make test-ef`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 07-01-01 | 01 | 1 | VOY-01 | unit | `go test -tags=ef -run TestVoyageMultimodal -v ./test/...` | ❌ W0 | ⬜ pending |
| 07-01-02 | 01 | 1 | VOY-01 | unit | `go test -tags=ef -run TestVoyageContentBlock -v ./test/...` | ❌ W0 | ⬜ pending |
| 07-02-01 | 02 | 1 | VOY-02 | unit | `go test -tags=ef -run TestVoyageIntentMap -v ./test/...` | ❌ W0 | ⬜ pending |
| 07-02-02 | 02 | 1 | VOY-02 | unit | `go test -tags=ef -run TestVoyageCapabilities -v ./test/...` | ❌ W0 | ⬜ pending |
| 07-03-01 | 03 | 2 | VOY-03 | integration | `go test -tags=ef -run TestVoyageRegistry -v ./test/...` | ❌ W0 | ⬜ pending |
| 07-03-02 | 03 | 2 | VOY-03 | integration | `go test -tags=ef -run TestVoyageBackcompat -v ./test/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Test stubs for multimodal content embedding (VOY-01)
- [ ] Test stubs for intent mapping and capabilities (VOY-02)
- [ ] Test stubs for registry and backward compatibility (VOY-03)

*Existing test infrastructure (go test + testify + build tags) covers framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Voyage API multimodal endpoint responds correctly | VOY-01 | Requires VOYAGE_API_KEY | Set env var, run `go test -tags=ef -run TestVoyageMultimodal -v ./test/...` |

*Integration tests require API key — automated in CI with secrets, manual locally.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
