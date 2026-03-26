---
phase: 10
slug: code-cleanups
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-26
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | Makefile (build tags) |
| **Quick run command** | `go test -tags=basicv2 -count=1 -short ./...` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -tags=basicv2 -count=1 -short ./...`
- **After every plan wave:** Run `make test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 10-01-01 | 01 | 1 | SC-1 | unit | `go test ./pkg/internal/pathutil/...` | ❌ W0 | ⬜ pending |
| 10-01-02 | 01 | 1 | SC-2 | unit | `go test -tags=ef ./pkg/embeddings/gemini/... ./pkg/embeddings/voyage/... ./pkg/embeddings/default_ef/...` | ✅ | ⬜ pending |
| 10-01-03 | 01 | 1 | SC-3 | build | `go build ./pkg/embeddings/gemini/... ./pkg/embeddings/nomic/... ./pkg/embeddings/mistral/...` | ✅ | ⬜ pending |
| 10-02-01 | 02 | 2 | SC-4 | unit | `go test -count=1 -race ./pkg/embeddings/registry_test.go` | ✅ | ⬜ pending |
| 10-02-02 | 02 | 2 | SC-6 | unit | `go test -tags=ef ./pkg/embeddings/gemini/... ./pkg/embeddings/voyage/...` | ✅ | ⬜ pending |
| 10-XX-XX | ALL | ALL | SC-5 | integration | `make test` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `pkg/internal/pathutil/pathutil_test.go` — unit tests for ContainsDotDot, ValidateFilePath, SafePath

*Existing test infrastructure covers all other phase requirements.*

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
