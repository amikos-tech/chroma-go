---
phase: 15-openrouter-embeddings-compatibility
verified: 2026-03-30T22:00:00Z
status: passed
score: 7/7 must-haves verified
gaps: []
---

# Phase 15: OpenRouter Embeddings Compatibility Verification Report

**Phase Goal:** Add OpenRouter embedding compatibility via standalone provider and OpenAI WithModelString for proxy endpoints
**Verified:** 2026-03-30
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #  | Truth                                                                              | Status     | Evidence                                                                                     |
|----|------------------------------------------------------------------------------------|------------|----------------------------------------------------------------------------------------------|
| 1  | OpenAI WithModelString accepts any non-empty string without validation             | VERIFIED | `func WithModelString` in `openai/options.go:100`, rejects empty, sets `c.Model = model` directly |
| 2  | OpenAI WithModel still rejects non-standard models                                 | VERIFIED | `openai/options.go:56` still validates against 3 known constants; unchanged                  |
| 3  | OpenRouter provider sends encoding_format, input_type, and provider fields in JSON | VERIFIED | `CreateEmbeddingRequest` has all three fields with correct JSON tags; `TestRequestSerialization` passes |
| 4  | OpenRouter provider registers as 'openrouter' in dense registry                    | VERIFIED | `init()` in `openrouter/openrouter.go:288` calls `RegisterDense("openrouter", ...)`; `TestRegistryRegistration` passes |
| 5  | OpenRouter config round-trip preserves all fields including provider preferences   | VERIFIED | `GetConfig`/`NewOpenRouterEmbeddingFunctionFromConfig` tested in `TestConfigRoundTrip`; passes |
| 6  | WithModelString tested: accept, reject empty, config round-trip                   | VERIFIED | 3 subtests in `openai_test.go` all pass                                                      |
| 7  | Existing OpenAI tests still pass unchanged                                         | VERIFIED | `go test -tags=ef ./pkg/embeddings/openai/...` exits 0 (5.65s)                              |

**Score:** 7/7 truths verified

### Required Artifacts

| Artifact                                          | Expected                                            | Status     | Details                                                              |
|---------------------------------------------------|-----------------------------------------------------|------------|----------------------------------------------------------------------|
| `pkg/embeddings/openai/options.go`                | WithModelString option function                     | VERIFIED   | `func WithModelString(model string) Option` at line 100              |
| `pkg/embeddings/openai/openai.go`                 | FromConfig using WithModelString for non-standard models | VERIFIED | `switch EmbeddingModel(model)` at line 335; default branch uses `WithModelString(model)` |
| `pkg/embeddings/openrouter/openrouter.go`         | Client, request/response types, EmbeddingFunction   | VERIFIED   | All types present; exports `NewOpenRouterEmbeddingFunction`, `NewOpenRouterEmbeddingFunctionFromConfig` |
| `pkg/embeddings/openrouter/options.go`            | With* functional options including WithModel        | VERIFIED   | `func WithModel`, `WithEncodingFormat`, `WithInputType`, `WithProviderPreferences` all present |
| `pkg/embeddings/openrouter/provider.go`           | ProviderPreferences struct with custom MarshalJSON  | VERIFIED   | 13 typed fields + `Extras map[string]any` with `json:"-"`; `MarshalJSON` at line 23 |
| `pkg/embeddings/openai/openai_test.go`            | WithModelString and config round-trip tests         | VERIFIED   | `Test WithModelString accepts arbitrary model`, `Test WithModelString rejects empty`, `Test config round-trip with non-standard model` |
| `pkg/embeddings/openrouter/openrouter_test.go`    | OpenRouter provider unit tests                      | VERIFIED   | 7 test functions including `TestRequestSerialization`, `TestProviderPreferences`, `TestConfigRoundTrip`, `TestRegistryRegistration` |

### Key Link Verification

| From                                 | To                                     | Via                                        | Status   | Details                                                                 |
|--------------------------------------|----------------------------------------|--------------------------------------------|----------|-------------------------------------------------------------------------|
| `openrouter/openrouter.go`           | `embeddings/registry.go`               | `init()` calling `RegisterDense`           | WIRED    | Line 288: `RegisterDense("openrouter", ...)` verified; `HasDense` test passes |
| `openai/openai.go`                   | `openai/options.go`                    | `FromConfig` using `WithModelString`       | WIRED    | Line 339: `opts = append(opts, WithModelString(model))` in default case |
| `openrouter/openrouter_test.go`      | `openrouter/openrouter.go`             | test calling `NewOpenRouterEmbeddingFunction` | WIRED  | Used in `TestRequestSerialization`, `TestEmbedQuerySingleInput`, `TestNameReturnsOpenRouter` |
| `openai/openai_test.go`              | `openai/options.go`                    | test calling `WithModelString`             | WIRED    | Used at lines 178, 190, 200 in test subtests                            |

### Data-Flow Trace (Level 4)

| Artifact                          | Data Variable    | Source                              | Produces Real Data | Status   |
|-----------------------------------|------------------|-------------------------------------|--------------------|----------|
| `OpenRouterEmbeddingFunction`     | `resp.Data`      | `CreateEmbedding` HTTP POST         | Yes — real HTTP call, response unmarshaled into `CreateEmbeddingResponse` | FLOWING |
| `EmbedDocuments` / `EmbedQuery`   | `Embedding`      | `resp.Data[i].Embedding []float32`  | Yes — `embeddings.NewEmbeddingFromFloat32(d.Embedding)` | FLOWING |
| `GetConfig`                       | `cfg`            | `e.apiClient.*` fields              | Yes — all populated fields serialized; provider marshaled via JSON round-trip | FLOWING |
| `FromConfig`                      | rebuilt client   | `cfg["api_key_env_var"]`, `cfg["model_name"]`, etc. | Yes — all keys extracted and applied via options | FLOWING |

### Behavioral Spot-Checks

| Behavior                                           | Command                                                              | Result                  | Status   |
|----------------------------------------------------|----------------------------------------------------------------------|-------------------------|----------|
| OpenRouter tests pass (7 tests)                    | `go test -tags=ef -count=1 ./pkg/embeddings/openrouter/...`         | PASS — 0.380s           | PASS     |
| OpenAI WithModelString tests pass (3 subtests)     | `go test -tags=ef -run "Test_openai_client/Test_WithModelString\|Test_config_round" ./pkg/embeddings/openai/...` | PASS | PASS |
| Full OpenAI test suite still passes                | `go test -tags=ef -count=1 ./pkg/embeddings/openai/...`             | PASS — 5.650s           | PASS     |
| Both packages build cleanly                        | `go build ./pkg/embeddings/openai/... ./pkg/embeddings/openrouter/...` | No output (success)  | PASS     |

### Requirements Coverage

| Requirement | Source Plan | Description                                                                                                    | Status    | Evidence                                                              |
|-------------|-------------|----------------------------------------------------------------------------------------------------------------|-----------|-----------------------------------------------------------------------|
| OR-01       | 15-01, 15-02 | `CreateEmbeddingRequest` supports `encoding_format`, `input_type`, `provider` fields                         | SATISFIED | `openrouter/openrouter.go:46-54` — all three fields with correct JSON tags; `TestRequestSerialization` asserts all three in captured body |
| OR-02       | 15-01, 15-02 | OpenAI `WithModelString` accepts any non-empty string for compatible proxies                                  | SATISFIED | `openai/options.go:100-108`; test `Test_WithModelString_accepts_arbitrary_model` passes |
| OR-03       | 15-01, 15-02 | `ProviderPreferences` typed struct with all documented fields + `Extras map[string]any` + custom `MarshalJSON` | SATISFIED | `openrouter/provider.go:6-42`; 13 typed fields + Extras; `TestProviderPreferences` with 3 subcases passes |
| OR-04       | 15-01, 15-02 | Existing OpenAI behavior and tests unchanged                                                                   | SATISFIED | `WithModel` validation unchanged; full `go test -tags=ef ./pkg/embeddings/openai/...` passes |
| OR-05       | 15-01, 15-02 | OpenRouter registered as `"openrouter"` in dense registry with full `GetConfig`/`FromConfig` round-trip       | SATISFIED | `init()` at line 288 registers; `TestRegistryRegistration` and `TestConfigRoundTrip` both pass |

All 5 requirements (OR-01 through OR-05) are satisfied. No orphaned requirements found — REQUIREMENTS.md maps all five IDs to Phase 15, and both plans claim all five.

### Anti-Patterns Found

No anti-patterns detected. Scanned all five implementation files for TODO/FIXME/HACK/PLACEHOLDER markers, stub returns, and empty implementations. None found.

The only "openai" references in the openrouter package are model name strings in test data (e.g., `"openai/text-embedding-3-small"`), not package imports. The no-cross-dependency constraint is satisfied.

### Human Verification Required

None. All observable truths are verifiable programmatically via the test suite and build checks.

### Gaps Summary

No gaps. All must-haves from both PLAN files are verified:

- `WithModelString` is implemented, wired into `FromConfig`, and tested.
- The standalone OpenRouter provider package exists with all required types (`Client`, `Input`, `CreateEmbeddingRequest`, `CreateEmbeddingResponse`, `ProviderPreferences`), functional options, HTTP dispatch, config round-trip, and `init()` registry entry.
- All 7 OpenRouter unit tests pass. All 3 new OpenAI tests pass. The full existing OpenAI test suite is unbroken.
- No dependency from `pkg/embeddings/openrouter` on `pkg/embeddings/openai`.

---

_Verified: 2026-03-30_
_Verifier: Claude (gsd-verifier)_
