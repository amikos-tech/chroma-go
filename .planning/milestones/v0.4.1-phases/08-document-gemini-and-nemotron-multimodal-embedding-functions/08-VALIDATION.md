---
phase: 8
slug: document-gemini-and-voyageai-multimodal-embedding-functions
status: draft
nyquist_compliant: true
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

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | Status |
|---------|------|------|-------------|-----------|-------------------|--------|
| 08-01-01 | 01 | 1 | D-03,D-04,D-05,D-06,D-07 | lint | `make lint` | pending |
| 08-01-02 | 01 | 1 | D-05,D-08,D-09 | build | `go build ./examples/v2/gemini_multimodal/ && go build ./examples/v2/voyage_multimodal/` | pending |
| 08-02-01 | 02 | 1 | D-10 | lint | `make lint` | pending |
| 08-02-02 | 02 | 1 | D-01,D-02,D-11 | lint+verify | `make lint && ! grep -q 'Nemotron' .planning/ROADMAP.md && test -f CHANGELOG.md` | pending |

*Status: pending / green / red / flaky*

### Task-to-Decision Traceability

| Decision | Plan | Task | Description |
|----------|------|------|-------------|
| D-01 | 02 | 2 | Phase covers Gemini + VoyageAI (not Nemotron) |
| D-02 | 02 | 2 | Correct ROADMAP phase name to VoyageAI |
| D-03 | 01 | 1 | Add Multimodal (Content API) subsection under both providers |
| D-04 | 01 | 1 | Keep existing text-only EmbedDocuments examples intact |
| D-05 | 01 | 1,2 | Show image AND video embedding examples in multimodal subsections and runnable examples |
| D-06 | 01 | 1 | Update Gemini default model to gemini-embedding-2-preview |
| D-07 | 01 | 1 | Update VoyageAI section with full option functions list |
| D-08 | 01 | 2 | Add examples/v2/gemini_multimodal/ runnable program |
| D-09 | 01 | 2 | Add examples/v2/voyage_multimodal/ runnable program |
| D-10 | 02 | 1 | Update README with multimodal mentions and example rows |
| D-11 | 02 | 2 | Create CHANGELOG.md with v0.4.1 release notes |

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. No new test framework needed -- this is a documentation phase.

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

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
