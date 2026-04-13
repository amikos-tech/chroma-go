---
phase: 25
slug: error-body-truncation
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-12
updated: 2026-04-13
---

# Phase 25 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Synced to the revised 4-plan / 7-task layout on 2026-04-13.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` |
| **Config file** | `none` |
| **Quick run command** | `go test ./pkg/commons/http ./pkg/embeddings/openrouter ./pkg/embeddings/perplexity ./pkg/embeddings/openai ./pkg/embeddings/baseten ./pkg/embeddings/cloudflare ./pkg/embeddings/twelvelabs` |
| **Full suite command** | `go test ./pkg/commons/http ./pkg/embeddings/... && make lint` |
| **Estimated runtime** | ~20-45s depending on package cache |

---

## Final Plan Graph

| Wave | Plans | Notes |
|------|-------|-------|
| 1 | `25-01` | Shared helper contract and Perplexity/OpenRouter normalization |
| 2 | `25-02`, `25-03` | Parallel provider slices with zero file overlap |
| 3 | `25-04` | Remaining provider batch, Twelve Labs regression, final sweep/lint |

---

## Sampling Rate

- **After every task commit:** run the task-specific automated command from the verification map below
- **After Wave 1:** run `go test ./pkg/commons/http ./pkg/embeddings/openrouter ./pkg/embeddings/perplexity`
- **After Wave 2:** run `go test ./pkg/commons/http ./pkg/embeddings/openai ./pkg/embeddings/baseten ./pkg/embeddings/bedrock ./pkg/embeddings/chromacloud ./pkg/embeddings/chromacloudsplade ./pkg/embeddings/cloudflare ./pkg/embeddings/cohere ./pkg/embeddings/hf ./pkg/embeddings/jina`
- **Before `/gsd-verify-work`:** `go test ./pkg/commons/http ./pkg/embeddings/... && make lint` must be green
- **Max feedback latency:** 60 seconds for focused feedback

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement IDs | Threat Ref | Secure Behavior | Automated Command | File Exists | Status |
|---------|------|------|-----------------|------------|-----------------|-------------------|-------------|--------|
| 25-01-01 | 01 | 1 | ERR-01, ERR-02 | T-25-03 / T-25-04 | Shared sanitizer tests pin trimming, rune-safe truncation, exact `[truncated]` suffix, and the compile-fail RED entry point for the new helper contract | `cd /Users/tazarov/GolandProjects/chroma-go && go test ./pkg/commons/http ./pkg/embeddings/openrouter ./pkg/embeddings/perplexity` | ✅ | ⬜ pending |
| 25-01-02 | 01 | 1 | ERR-01, ERR-02 | T-25-01 / T-25-02 / T-25-04 | Shared helper implements panic-safe best-effort sanitization and Perplexity/OpenRouter stop owning local truncation behavior | `cd /Users/tazarov/GolandProjects/chroma-go && go test ./pkg/commons/http ./pkg/embeddings/openrouter ./pkg/embeddings/perplexity` | ✅ | ⬜ pending |
| 25-02-01 | 02 | 2 | ERR-02 | T-25-05 / T-25-06 | Representative raw-body providers sanitize body text and OpenAI/Baseten regressions prove large payloads collapse to `[truncated]` output | `cd /Users/tazarov/GolandProjects/chroma-go && go test ./pkg/commons/http ./pkg/embeddings/openrouter ./pkg/embeddings/perplexity ./pkg/embeddings/openai ./pkg/embeddings/baseten ./pkg/embeddings/bedrock ./pkg/embeddings/chromacloud ./pkg/embeddings/chromacloudsplade` | ✅ | ⬜ pending |
| 25-03-01 | 03 | 2 | ERR-02 | T-25-08 / T-25-09 | Cloudflare preserves `embeddings.Errors` while sanitizing only the appended raw-body tail; Cohere/HF/Jina sanitize raw provider text | `cd /Users/tazarov/GolandProjects/chroma-go && go test ./pkg/commons/http ./pkg/embeddings/cloudflare ./pkg/embeddings/cohere ./pkg/embeddings/hf ./pkg/embeddings/jina && rg -n 'embeddings.Errors|SanitizeErrorBody\\(respData\\)' pkg/embeddings/cloudflare/cloudflare.go` | ✅ | ⬜ pending |
| 25-04-01 | 04 | 3 | ERR-02 | T-25-10 | Remaining batch-B raw-body providers use the shared sanitizer without changing provider-specific status and endpoint wording | `cd /Users/tazarov/GolandProjects/chroma-go && go test ./pkg/commons/http ./pkg/embeddings/mistral ./pkg/embeddings/morph ./pkg/embeddings/nomic ./pkg/embeddings/ollama ./pkg/embeddings/roboflow ./pkg/embeddings/together ./pkg/embeddings/voyage` | ✅ | ⬜ pending |
| 25-04-02 | 04 | 3 | ERR-02 | T-25-11 | Twelve Labs treats parsed `message` fields as body-derived text, sanitizes them, and proves the structured-message path truncates safely | `cd /Users/tazarov/GolandProjects/chroma-go && go test ./pkg/commons/http ./pkg/embeddings/twelvelabs` | ✅ | ⬜ pending |
| 25-04-03 | 04 | 3 | ERR-01, ERR-02 | T-25-12 | Final phase gate proves the shared helper and all provider migrations survive the full embedding-package sweep and lint | `cd /Users/tazarov/GolandProjects/chroma-go && go test ./pkg/commons/http ./pkg/embeddings/... && make lint` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. No extra Wave 0 scaffolding is needed.

---

## Manual-Only Verifications

All Phase 25 behaviors are automatable in shared helper tests, provider unit tests, source-level grep spot-checks, and the final suite/lint gate.

---

## Validation Sign-Off

- [x] All 7 current tasks have automated verification
- [x] Every task is mapped with task ID, plan, wave, requirement IDs, threat refs, and exact command
- [x] Wave structure matches the revised 4-plan graph (`25-01` -> `25-02`/`25-03` -> `25-04`)
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all missing references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** ready for `/gsd-execute-phase 25`
