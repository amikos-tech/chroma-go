---
phase: 13
slug: collection-forkcount
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-28
---

# Phase 13 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify |
| **Config file** | Makefile (build tags: basicv2) |
| **Quick run command** | `go test -tags=basicv2 -run "ForkCount" ./pkg/api/v2/...` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=basicv2 -run "ForkCount" ./pkg/api/v2/...`
- **After every plan wave:** Run `make test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 13-01-01 | 01 | 1 | SC-1 | compile | `go build -tags=basicv2 ./pkg/api/v2/...` | N/A (compiler) | ⬜ pending |
| 13-01-02 | 01 | 1 | SC-2 | unit | `go test -tags=basicv2 -run TestCollectionForkCount ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 13-01-03 | 01 | 1 | SC-3 | unit | `go test -tags=basicv2 -run TestEmbeddedCollection_ForkCountNotSupported ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 13-01-04 | 01 | 1 | SC-4 | unit | `go test -tags=basicv2 -run TestCollectionForkCount ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 13-02-01 | 02 | 2 | SC-5 | manual | Visual review | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. No new test frameworks, fixtures, or configuration needed. Tests are additions to existing test files that already use testify + httptest.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Forking docs updated with ForkCount section | SC-5 | Documentation content review | Check `docs/go-examples/cloud/features/collection-forking.md` contains ForkCount section with code examples |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
