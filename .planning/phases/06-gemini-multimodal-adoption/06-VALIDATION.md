---
phase: 6
slug: gemini-multimodal-adoption
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-20
updated: 2026-03-21
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | Makefile (test-ef target) |
| **Quick run command** | `go test -run TestGemini -count=1 ./pkg/embeddings/gemini/...` |
| **Full suite command** | `go test -count=1 ./pkg/embeddings/gemini/...` |
| **Estimated runtime** | ~1 second (unit tests only) |

---

## Sampling Rate

- **After every task commit:** Run `go test -count=1 ./pkg/embeddings/gemini/...`
- **After every plan wave:** Run full suite
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 1 second

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | Tests | Status |
|---------|------|------|-------------|-----------|-------------------|-------|--------|
| 06-01-01 | 01 | 1 | GEM-01 | unit | `go test -run "TestConvertToGenaiContent" ./pkg/embeddings/gemini/...` | 9 tests | ✅ green |
| 06-01-02 | 01 | 1 | GEM-01 | unit | `go test -run "TestCapabilities\|TestEmbedContent.*Effective" ./pkg/embeddings/gemini/...` | 4 tests | ✅ green |
| 06-02-01 | 02 | 1 | GEM-02 | unit | `go test -run "TestMapIntent\|TestResolveTaskType" ./pkg/embeddings/gemini/...` | 4 tests | ✅ green |
| 06-03-01 | 03 | 2 | GEM-03 | unit | `go test -run "TestGeminiContentRegistration" ./pkg/embeddings/gemini/...` | 1 test | ✅ green |
| 06-03-02 | 03 | 2 | GEM-03 | unit | `go test -run "TestGeminiContentConfigRoundTrip" ./pkg/embeddings/gemini/...` | 1 test | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## PR Feedback Coverage (added post-review)

| Area | Tests | Status |
|------|-------|--------|
| Nil source guards | `TestResolveMIMENilSource`, `TestResolveBytesNilSource`, `TestConvertToGenaiContentNilSourceReturnsError`, `TestEmbedContentValidatesStructure`, `TestCreateContentEmbeddingValidatesContent` | ✅ green |
| File/payload size limits | `TestResolveBytesFileExceedsMaxSize`, `TestResolveBytesPayloadExceedsMaxSize`, `TestResolveBytesBase64PayloadExceedsMaxSize` | ✅ green |
| Path traversal | `TestResolveBytesFilePathTraversal`, `TestResolveBytesFilePathTraversalObfuscated` | ✅ green |
| URL passthrough | `TestConvertToGenaiContentURLPassthrough`, `TestConvertToGenaiContentURLMissingMIME`, `TestResolveBytesRejectsURLKind` | ✅ green |
| Content.Dimension | `TestCreateContentEmbeddingHonorsContentDimension`, `TestCreateContentEmbeddingContextDimensionOverridesContentDimension` | ✅ green |
| Effective model caps | `TestEmbedContentUsesEffectiveModelCapabilities`, `TestEmbedContentsUsesEffectiveModelCapabilities` | ✅ green |
| MaxBatchSize | `TestEmbedContentsEnforcesMaxBatchSize`, `TestDefaultMaxBatchSize` | ✅ green |
| Batch override rejection | `TestCreateContentEmbeddingRejectsBatchPerItemOverrides` (4 subtests) | ✅ green |
| IntentMapper validation | `TestResolveTaskTypeForContentRejectsInvalidMapperResult` | ✅ green |

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Backward compat: existing EmbedDocuments/EmbedQuery unchanged | GEM-01 | Requires running against live Gemini API | Run existing ef tests with GEMINI_API_KEY set |

---

## Validation Sign-Off

- [x] All tasks have automated verify commands
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s (actual: ~1s)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated

---

## Validation Audit 2026-03-21

| Metric | Count |
|--------|-------|
| Total test functions | 56 |
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
| Requirements covered | GEM-01, GEM-02, GEM-03 (all 3) |
