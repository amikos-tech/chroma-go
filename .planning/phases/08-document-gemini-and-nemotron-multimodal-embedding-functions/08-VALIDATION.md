---
phase: 8
slug: document-gemini-and-nemotron-multimodal-embedding-functions
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-23
---

# Phase 8 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + make lint |
| **Config file** | Makefile |
| **Quick run command** | `make lint` |
| **Full suite command** | `make lint && make build` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `make lint`
- **After every plan wave:** Run `make lint && make build`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 08-01-01 | 01 | 1 | D-02 | lint | `make lint` | N/A | ⬜ pending |
| 08-01-02 | 01 | 1 | D-03,D-05,D-07 | lint | `make lint` | N/A | ⬜ pending |
| 08-01-03 | 01 | 1 | D-03,D-05,D-06 | lint | `make lint` | N/A | ⬜ pending |
| 08-02-01 | 02 | 1 | D-08 | build | `go build ./examples/v2/gemini_multimodal/` | N/A | ⬜ pending |
| 08-02-02 | 02 | 1 | D-09 | build | `go build ./examples/v2/voyage_multimodal/` | N/A | ⬜ pending |
| 08-03-01 | 03 | 2 | D-10 | lint | `make lint` | N/A | ⬜ pending |
| 08-03-02 | 03 | 2 | D-11 | lint | `make lint` | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. No new test framework needed — this is a documentation phase.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Doc links work | D-03, D-05 | Cross-references in markdown require rendering | Review multimodal.md links in provider sections |
| Code examples match API | D-05, D-08, D-09 | Requires comparing doc snippets to source | Diff example code against content.go constructors |
| ROADMAP consistency | D-02 | Semantic check | Verify no "Nemotron" references remain |
| Visual doc quality | All | Layout and readability | Review rendered embeddings.md page |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
