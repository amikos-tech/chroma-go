---
phase: 25-error-body-truncation
reviewed: 2026-04-13T13:33:10Z
depth: standard
files_reviewed: 27
files_reviewed_list:
  - pkg/commons/http/utils.go
  - pkg/commons/http/utils_test.go
  - pkg/embeddings/baseten/baseten.go
  - pkg/embeddings/baseten/baseten_test.go
  - pkg/embeddings/bedrock/bedrock.go
  - pkg/embeddings/chromacloud/chromacloud.go
  - pkg/embeddings/chromacloudsplade/chromacloudsplade.go
  - pkg/embeddings/cloudflare/cloudflare.go
  - pkg/embeddings/cloudflare/cloudflare_error_test.go
  - pkg/embeddings/cohere/cohere.go
  - pkg/embeddings/hf/hf.go
  - pkg/embeddings/jina/jina.go
  - pkg/embeddings/mistral/mistral.go
  - pkg/embeddings/morph/morph.go
  - pkg/embeddings/nomic/nomic.go
  - pkg/embeddings/ollama/ollama.go
  - pkg/embeddings/openai/openai.go
  - pkg/embeddings/openai/openai_test.go
  - pkg/embeddings/openrouter/openrouter.go
  - pkg/embeddings/openrouter/openrouter_test.go
  - pkg/embeddings/perplexity/perplexity.go
  - pkg/embeddings/perplexity/perplexity_test.go
  - pkg/embeddings/roboflow/roboflow.go
  - pkg/embeddings/together/together.go
  - pkg/embeddings/twelvelabs/twelvelabs.go
  - pkg/embeddings/twelvelabs/twelvelabs_test.go
  - pkg/embeddings/voyage/voyage.go
findings:
  critical: 0
  warning: 3
  info: 0
  total: 3
status: issues_found
---

# Phase 25: Code Review Report

**Reviewed:** 2026-04-13T13:33:10Z
**Depth:** standard
**Files Reviewed:** 27
**Status:** issues_found

## Summary

Reviewed the Phase 25 error-body truncation rollout across the shared HTTP helper and the embedding providers in scope. The `[truncated]` contract is wired through most raw-body paths and the targeted regression tests for the touched providers pass, but three issues remain: the shared sanitizer is still unsafe for max-sized bodies, Cloudflare can still emit oversized structured error content, and Cohere's runtime default model changed as a side effect of this phase.

Verification run during review:

- `go test ./pkg/commons/http ./pkg/embeddings/baseten ./pkg/embeddings/bedrock ./pkg/embeddings/chromacloud ./pkg/embeddings/chromacloudsplade ./pkg/embeddings/cloudflare ./pkg/embeddings/cohere ./pkg/embeddings/hf ./pkg/embeddings/jina ./pkg/embeddings/mistral ./pkg/embeddings/morph ./pkg/embeddings/nomic ./pkg/embeddings/ollama ./pkg/embeddings/openai ./pkg/embeddings/openrouter ./pkg/embeddings/perplexity ./pkg/embeddings/roboflow ./pkg/embeddings/together ./pkg/embeddings/twelvelabs ./pkg/embeddings/voyage`
- `go test -tags=ef ./pkg/commons/http ./pkg/embeddings/openrouter ./pkg/embeddings/perplexity ./pkg/embeddings/openai ./pkg/embeddings/baseten ./pkg/embeddings/cloudflare ./pkg/embeddings/twelvelabs -run 'Test(SanitizeErrorBody|APIErrorResponseParsing|ParseAPIErrorTruncatesLargeBody|PerplexityEmbeddingFunction_HTTPErrorResponse_TruncatedBody|OpenAIEmbeddingFunction_APIErrorTruncatesLongBody|BasetenEmbeddingFunction_APIErrorTruncatesLongBody|CreateEmbeddingPreservesStructuredErrorsWhileSanitizingRawTail|CreateEmbeddingSanitizesNonJSONErrorBody|TwelveLabsAPIErrorSanitizesStructuredMessage|TwelveLabsAPIErrorSanitizesRawFallbackBody)$'`

Residual gap: I did not rerun the full live-provider `make test-ef` sweep, so provider behavior that depends on real upstream services was not revalidated in this review pass.

## Warnings

### WR-01: Shared sanitizer still scales with full response size

**File:** `pkg/commons/http/utils.go:31-58`
**Issue:** `sanitizeErrorBodyString` converts the entire trimmed body to `[]rune` before applying the 512-rune cap, and `SanitizeErrorBody`'s deferred recovery retries the same `string(body)`/`sanitizeErrorBodyString(...)` work on the same input.
**Impact:** A provider error body near the existing `MaxResponseBodySize` limit can still trigger very large allocations or an unrecoverable OOM/panic, so the helper does not actually deliver the advertised "never panics" behavior under worst-case inputs.
**Fix:** Truncate incrementally instead of materializing the full rune slice. Scan only until `maxSanitizedErrorBodyRunes+1` runes with `utf8.DecodeRune`, and make the recover path return a fixed placeholder or an already-built prefix rather than retrying the same high-allocation conversion.

### WR-02: Cloudflare structured errors bypass the truncation contract

**File:** `pkg/embeddings/cloudflare/cloudflare.go:153-171`
**Issue:** The new Cloudflare path sanitizes `respData`, but it still formats `embeddings.Errors` directly with `%v`. Any long message inside the parsed `errors` array is emitted verbatim before the sanitized raw-body tail.
**Impact:** Cloudflare responses can still produce oversized body-derived error strings, so this provider does not fully honor the shared 512-rune display contract introduced by Phase 25.
**Fix:** Sanitize the structured segment too. A simple fix is to marshal `embeddings.Errors` back to JSON and pass that string through `chttp.SanitizeErrorBody(...)`, or sanitize individual parsed `message` fields before formatting.

### WR-03: Cohere default model changed in a non-compatibility phase

**File:** `pkg/embeddings/cohere/cohere.go:36-42`
**Issue:** `DefaultEmbedModel` now points to `ModelEmbedEnglishV30` instead of `ModelEmbedEnglishV20`. Every caller that relies on the implicit default model now sends a different request than before, even though this phase is scoped to error-body formatting.
**Impact:** Existing applications can receive different embedding vectors and dimensions, and newly persisted default configs will serialize a different `model_name`. That is a runtime behavior regression unrelated to the truncation rollout.
**Fix:** Keep the old runtime default in Phase 25 and make any Cohere verification/tests request `embed-english-v3.0` explicitly. If the default must change, ship it as a separate compatibility update with release notes and dedicated regression coverage.

---

_Reviewed: 2026-04-13T13:33:10Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
