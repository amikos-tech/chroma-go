---
phase: 17
slug: cloud-rrf-and-groupby-test-coverage
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-02
---

# Phase 17 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify v1.x |
| **Config file** | Makefile targets + build tags (`basicv2 && cloud`) |
| **Quick run command** | `go test -tags=basicv2,cloud -run "TestCloudClientSearch(RRF\|GroupBy)" -v -timeout=5m ./pkg/api/v2/...` |
| **Full suite command** | `make test-cloud` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=basicv2,cloud -run "TestCloudClientSearch(RRF|GroupBy)" -v -timeout=5m ./pkg/api/v2/...`
- **After every plan wave:** Run `make test-cloud`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 17-01-01 | 01 | 1 | SC-01 | cloud integration | `go test -tags=basicv2,cloud -run TestCloudClientSearchRRF/smoke -v -timeout=5m ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 17-01-02 | 01 | 1 | SC-02 | cloud integration | `go test -tags=basicv2,cloud -run TestCloudClientSearchRRF/weighted -v -timeout=5m ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 17-01-03 | 01 | 1 | SC-03 | cloud integration | `go test -tags=basicv2,cloud -run TestCloudClientSearchGroupBy/MinK -v -timeout=5m ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 17-01-04 | 01 | 1 | SC-04 | cloud integration | `go test -tags=basicv2,cloud -run TestCloudClientSearchGroupBy/MaxK -v -timeout=5m ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

*Existing infrastructure covers all phase requirements. No new test framework setup needed. Tests are purely additive to `client_cloud_test.go`.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
