# Phase 15: OpenRouter Embeddings Compatibility - Context

**Gathered:** 2026-03-30
**Status:** Ready for planning

<domain>
## Phase Boundary

Extend the OpenAI embedding function with a `WithModelString` bypass for arbitrary model IDs, and create a new standalone OpenRouter embedding provider (`pkg/embeddings/openrouter/`) that supports `encoding_format`, `input_type`, and `ProviderPreferences` fields. The OpenRouter provider is independent from the OpenAI package — own HTTP logic, own types, own registry entry.

</domain>

<decisions>
## Implementation Decisions

### Architecture
- **D-01:** Create a new standalone `pkg/embeddings/openrouter/` provider package. Do NOT bloat the OpenAI provider with OpenRouter-specific fields.
- **D-02:** The OpenRouter provider has its own HTTP client, request/response types, and options — no dependency on `pkg/embeddings/openai`.
- **D-03:** The OpenAI provider gets only a `WithModelString(string)` option that accepts any non-empty string without validation, for use with any OpenAI-compatible endpoint (Azure, LiteLLM, vLLM, etc.).

### Model Validation
- **D-04:** `WithModel` on the OpenAI provider keeps strict validation against the 3 known OpenAI model constants when using the default OpenAI base URL.
- **D-05:** New `WithModelString(string)` on the OpenAI provider accepts any non-empty string unconditionally.
- **D-06:** The OpenRouter provider accepts any model string (no validation) — provider-prefixed IDs like `openai/text-embedding-3-small` are the norm.

### Provider Preferences
- **D-07:** `ProviderPreferences` is a typed struct with all documented OpenRouter fields plus an `Extras map[string]any` with custom MarshalJSON for forward-compatibility.
- **D-08:** Typed fields cover: `allow_fallbacks`, `require_parameters`, `data_collection`, `zdr`, `enforce_distillable_text`, `order`, `only`, `ignore`, `quantizations`, `sort`, `max_price`, `preferred_min_throughput`, `preferred_max_latency`.

### Request Fields
- **D-09:** `CreateEmbeddingRequest` in the OpenRouter package includes `encoding_format`, `input_type`, and `provider` fields alongside standard OpenAI-compatible fields (`model`, `input`, `dimensions`, `user`).
- **D-10:** All OpenRouter-specific fields are wired through `With*` functional options at constructor time.

### Config Round-Trip
- **D-11:** Register as `"openrouter"` in the dense registry with full `GetConfig`/`FromConfig` support.
- **D-12:** Config serializes all OpenRouter fields including provider preferences (as nested map). FromConfig rebuilds the full client from stored collection metadata.

### Claude's Discretion
- HTTP client setup, error handling patterns, and internal helper structure
- Test organization within the new package

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### OpenRouter API
- Issue #438 — full problem statement and acceptance criteria
- `https://openrouter.ai/docs/api/reference/embeddings` — OpenRouter embeddings API reference
- `https://openrouter.ai/docs/api/api-reference/embeddings/create-embeddings` — request schema
- `https://openrouter.ai/openapi.json` — POST /embeddings and ProviderPreferences schema

### Existing Provider Patterns
- `pkg/embeddings/openai/openai.go` — OpenAI provider structure to reference (but NOT depend on)
- `pkg/embeddings/openai/options.go` — Where `WithModelString` will be added
- `pkg/embeddings/together/together.go` — Example of standalone provider with own HTTP client
- `pkg/embeddings/registry.go` — Dense registry registration pattern
- `pkg/embeddings/embedding.go` — `EmbeddingFunction` interface and `EmbeddingFunctionConfig`

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/commons/http` — `ReadLimitedBody`, `ChromaGoClientUserAgent` for HTTP utilities
- `pkg/embeddings.Secret` — API key wrapper with env var support
- `pkg/embeddings.NewValidator()` — Struct validator
- `pkg/embeddings.ConfigInt()` — Config map type helpers
- `pkg/embeddings.AllowInsecureFromEnv()` / `LogInsecureEnvVarWarning()` — Insecure mode utilities

### Established Patterns
- Functional options (`With*` returning `func(*Client) error`)
- `init()` registration with `embeddings.RegisterDense(name, factory)`
- `GetConfig()` returns `EmbeddingFunctionConfig` (map[string]any) for persistence
- `NewXFromConfig(cfg)` factory for registry rebuild
- `DefaultSpace()` and `SupportedSpaces()` for distance metric metadata
- `Name()` returns provider name string

### Integration Points
- Dense registry in `pkg/embeddings/registry.go` — register as `"openrouter"`
- Config persistence through collection metadata auto-wiring

</code_context>

<specifics>
## Specific Ideas

- OpenRouter base URL defaults to `https://openrouter.ai/api/v1/`
- The `ProviderPreferences.Extras` map uses custom `MarshalJSON` to merge typed fields with extras
- `WithModelString` on the OpenAI provider is useful beyond OpenRouter — any OpenAI-compatible proxy benefits

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 15-openrouter-embeddings-compatibility*
*Context gathered: 2026-03-30*
