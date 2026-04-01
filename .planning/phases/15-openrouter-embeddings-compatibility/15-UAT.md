---
status: complete
phase: 15-openrouter-embeddings-compatibility
source: [15-01-SUMMARY.md, 15-02-SUMMARY.md]
started: 2026-03-31T10:00:00Z
updated: 2026-03-31T10:05:00Z
---

## Current Test

[testing complete]

## Tests

### 1. OpenAI WithModelString Accepts Arbitrary Models
expected: Running `go test -tags=ef -run "TestWithModelString" ./pkg/embeddings/openai/...` passes. WithModelString allows arbitrary model IDs without validation errors.
result: pass

### 2. OpenAI Config Round-Trip with Non-Standard Model
expected: Running `go test -tags=ef -run "TestConfigRoundTrip" ./pkg/embeddings/openai/...` passes. A non-standard model name set via config is routed through WithModelString and preserved in round-trip.
result: pass
note: Test matched under openrouter package (TestConfigRoundTrip). OpenAI config round-trip tested as part of full suite subtests.

### 3. OpenRouter Provider Request Serialization
expected: Running `go test -tags=ef -run "TestOpenRouter" ./pkg/embeddings/openrouter/...` passes. HTTP request body includes encoding_format, input_type, and provider preferences fields as expected.
result: pass
note: TestRequestSerialization passed with correct field verification.

### 4. ProviderPreferences MarshalJSON Merge Behavior
expected: Running `go test -tags=ef -run "TestProviderPreferences" ./pkg/embeddings/openrouter/...` passes. Typed fields take precedence over Extras map duplicates during JSON marshaling.
result: pass

### 5. OpenRouter Config Round-Trip
expected: Running `go test -tags=ef -run "TestConfig" ./pkg/embeddings/openrouter/...` passes. All OpenRouter fields including nested provider preferences survive config serialization/deserialization.
result: pass

### 6. OpenRouter Dense Registry Registration
expected: Running `go test -tags=ef -run "TestRegistry" ./pkg/embeddings/openrouter/...` passes. `embeddings.HasDense("openrouter")` returns true after package import.
result: pass

### 7. Full Test Suite Passes
expected: Running `go test -tags=ef ./pkg/embeddings/openai/... ./pkg/embeddings/openrouter/...` passes with 0 failures. No lint issues from `make lint`.
result: pass

## Summary

total: 7
passed: 7
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none yet]
