---
phase: 9
slug: convenience-constructors-and-documentation-polish
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-25
---

# Phase 9 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go test tooling |
| **Quick run command** | `go test -tags=basicv2 -run TestContentConstructors ./pkg/api/v2/...` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=basicv2 -run TestContentConstructors ./pkg/api/v2/...`
- **After every plan wave:** Run `make test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 09-01-01 | 01 | 1 | SC-1 | unit | `go test -tags=basicv2 -run TestNewImageURL ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 09-01-02 | 01 | 1 | SC-1 | unit | `go test -tags=basicv2 -run TestNewImageFile ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 09-01-03 | 01 | 1 | SC-1 | unit | `go test -tags=basicv2 -run TestNewVideoURL ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 09-01-04 | 01 | 1 | SC-1 | unit | `go test -tags=basicv2 -run TestNewVideoFile ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 09-01-05 | 01 | 1 | SC-1 | unit | `go test -tags=basicv2 -run TestNewAudioFile ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 09-01-06 | 01 | 1 | SC-1 | unit | `go test -tags=basicv2 -run TestNewPDFFile ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 09-01-07 | 01 | 1 | SC-1 | unit | `go test -tags=basicv2 -run TestNewContent ./pkg/api/v2/...` | ❌ W0 | ⬜ pending |
| 09-02-01 | 02 | 2 | SC-2 | integration | `make test` | ✅ | ⬜ pending |
| 09-03-01 | 03 | 2 | SC-3 | manual | review docs for shorthand usage | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `pkg/api/v2/content_constructors_test.go` — unit tests for all convenience constructors
- [ ] Ensure `go test -tags=basicv2 ./pkg/api/v2/...` runs cleanly before starting

*Existing infrastructure covers framework needs — Go test tooling is already in place.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Doc examples show shorthand forms | SC-3 | Documentation review | Read `embeddings.md` Gemini/Voyage sections and example files; verify shorthand constructors appear alongside verbose forms |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
