---
phase: 18
slug: embedded-client-contentembeddingfunction-parity
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-02
---

# Phase 18 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify v1.x |
| **Config file** | None (build tags in file headers) |
| **Quick run command** | `go test -v -tags=basicv2 -run TestEmbedded -count=1 ./pkg/api/v2/` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -v -tags=basicv2 -run TestEmbedded -count=1 ./pkg/api/v2/`
- **After every plan wave:** Run `make test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 18-01-01 | 01 | 1 | SC-1 | unit | `go test -v -tags=basicv2 -run TestEmbeddedCollection_ContentEF -count=1 ./pkg/api/v2/` | ❌ W0 | ⬜ pending |
| 18-01-02 | 01 | 1 | SC-2 | unit | `go test -v -tags=basicv2 -run TestEmbeddedBuild_ContentEF -count=1 ./pkg/api/v2/` | ❌ W0 | ⬜ pending |
| 18-01-03 | 01 | 1 | SC-3 | unit | `go test -v -tags=basicv2 -run TestEmbeddedCollection_Close.*Content -count=1 ./pkg/api/v2/` | ❌ W0 | ⬜ pending |
| 18-01-04 | 01 | 1 | SC-5 | unit | `go test -v -tags=basicv2 -run TestEmbeddedGetCollection.*ContentEF -count=1 ./pkg/api/v2/` | ❌ W0 | ⬜ pending |
| 18-01-05 | 01 | 1 | SC-6 | unit | `go test -v -tags=basicv2 -run TestEmbeddedGetCollection.*AutoWire -count=1 ./pkg/api/v2/` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Test functions for embedded contentEF lifecycle in `close_review_test.go`
- [ ] Tests for Close() sharing detection scenarios (unwrapper case, dual-interface case, independent case)
- [ ] Tests for GetCollection auto-wiring with contentEF in `client_local_embedded_test.go`
- [ ] Tests for GetCollection with explicit `WithContentEmbeddingFunctionGet`

*Existing infrastructure (testify, build tags, mock types) covers all framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Fork() does NOT propagate contentEF (D-01) | SC-4 | Fork returns unsupported error — no change needed, existing test covers | Verify Fork() still returns unsupported error |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
