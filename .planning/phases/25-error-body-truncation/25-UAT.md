---
status: complete
phase: 25-error-body-truncation
source: [25-01-SUMMARY.md, 25-02-SUMMARY.md, 25-03-SUMMARY.md, 25-04-SUMMARY.md]
started: 2026-04-13T12:17:55.622Z
updated: 2026-04-13T12:42:47.486Z
---

## Current Test

[testing complete]

## Tests

### 1. Shared sanitizer contract
expected: Run `go test -tags=ef ./pkg/commons/http -run TestSanitizeErrorBody -count=1` — it passes, proving whitespace trimming, 512-rune truncation, the exact `[truncated]` suffix, panic-safe fallback behavior, and UTF-8-safe handling.
result: pass

### 2. Perplexity and OpenRouter truncate oversized provider errors
expected: Run `go test -tags=ef ./pkg/embeddings/perplexity -run 'TestPerplexityEmbeddingFunction_HTTPErrorResponse_TruncatedBody(_UTF8Safe)?' -count=1 && go test -tags=ef ./pkg/embeddings/openrouter -run TestAPIErrorResponseParsing -count=1` — both pass, and oversized raw or structured provider messages surface `[truncated]` instead of the full payload.
result: pass

### 3. OpenAI and Baseten sanitize representative long raw bodies
expected: Run `go test -tags=ef ./pkg/embeddings/openai -run TestOpenAIEmbeddingFunction_APIErrorTruncatesLongBody -count=1 && go test -tags=ef ./pkg/embeddings/baseten -run TestBasetenEmbeddingFunction_APIErrorTruncatesLongBody -count=1` — both pass, and long raw provider bodies are shortened to the shared display contract.
result: pass

### 4. Cloudflare preserves structured errors while sanitizing the raw tail
expected: Run `go test -tags=ef ./pkg/embeddings/cloudflare -run 'TestCreateEmbedding(PreservesStructuredErrorsWhileSanitizingRawTail|SanitizesNonJSONErrorBody)' -count=1` — the tests pass, structured `embeddings.Errors` content stays intact, and only the appended raw-body segment is truncated.
result: pass

### 5. Cohere, Hugging Face, and Jina sanitize provider error bodies
expected: Run `go test -tags=ef ./pkg/embeddings/cohere ./pkg/embeddings/hf ./pkg/embeddings/jina -count=1` — the suite passes and these providers no longer leak full raw error bodies into returned errors.
result: pass

### 6. Cohere default path still works without an explicit model
expected: Exercise the default Cohere embedding path without setting a model, or run `go test -tags=ef ./pkg/embeddings/cohere -count=1` — callers that omit a model should still succeed with the current supported default instead of failing on the retired v2 default.
result: pass

### 7. Twelve Labs truncates parsed and raw fallback errors
expected: Run `go test -tags=ef ./pkg/embeddings/twelvelabs -run TestTwelveLabsAPIError -count=1` — both parsed `message` content and raw fallback bodies are sanitized with `[truncated]`.
result: pass

### 8. Full embedding sweep and lint stay green
expected: Run `go test -tags=ef ./pkg/commons/http ./pkg/embeddings/... && make lint` — the full embedding tree passes and lint reports `0 issues.`
result: pass

## Summary

total: 8
passed: 8
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none]
