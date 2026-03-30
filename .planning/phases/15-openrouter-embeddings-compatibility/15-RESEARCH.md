# Phase 15: OpenRouter Embeddings Compatibility - Research

**Researched:** 2026-03-30
**Domain:** Go embedding provider implementation, OpenRouter API compatibility
**Confidence:** HIGH

## Summary

This phase adds two things: (1) a `WithModelString` option to the existing OpenAI provider for arbitrary model IDs, and (2) a standalone `pkg/embeddings/openrouter/` provider with OpenRouter-specific fields (`encoding_format`, `input_type`, `provider` preferences). The OpenRouter embeddings API is OpenAI-compatible at its core (same request/response shape for `model`, `input`, `dimensions`, `user`) but adds `encoding_format`, `input_type`, and a `provider` object for routing preferences.

The codebase has 15+ embedding providers following a consistent pattern: own package, functional options, `init()` registry, `GetConfig`/`FromConfig` round-trip. The Together provider is the best structural template for a standalone provider with own HTTP client. The OpenAI provider shows the config round-trip and validation patterns to follow.

**Primary recommendation:** Create `pkg/embeddings/openrouter/` as a standalone package following the Together provider pattern, with its own HTTP client, request/response types, and `ProviderPreferences` struct. Add `WithModelString` to `pkg/embeddings/openai/options.go` as a simple bypass.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Create a new standalone `pkg/embeddings/openrouter/` provider package. Do NOT bloat the OpenAI provider with OpenRouter-specific fields.
- **D-02:** The OpenRouter provider has its own HTTP client, request/response types, and options -- no dependency on `pkg/embeddings/openai`.
- **D-03:** The OpenAI provider gets only a `WithModelString(string)` option that accepts any non-empty string without validation, for use with any OpenAI-compatible endpoint (Azure, LiteLLM, vLLM, etc.).
- **D-04:** `WithModel` on the OpenAI provider keeps strict validation against the 3 known OpenAI model constants when using the default OpenAI base URL.
- **D-05:** New `WithModelString(string)` on the OpenAI provider accepts any non-empty string unconditionally.
- **D-06:** The OpenRouter provider accepts any model string (no validation) -- provider-prefixed IDs like `openai/text-embedding-3-small` are the norm.
- **D-07:** `ProviderPreferences` is a typed struct with all documented OpenRouter fields plus an `Extras map[string]any` with custom MarshalJSON for forward-compatibility.
- **D-08:** Typed fields cover: `allow_fallbacks`, `require_parameters`, `data_collection`, `zdr`, `enforce_distillable_text`, `order`, `only`, `ignore`, `quantizations`, `sort`, `max_price`, `preferred_min_throughput`, `preferred_max_latency`.
- **D-09:** `CreateEmbeddingRequest` in the OpenRouter package includes `encoding_format`, `input_type`, and `provider` fields alongside standard OpenAI-compatible fields (`model`, `input`, `dimensions`, `user`).
- **D-10:** All OpenRouter-specific fields are wired through `With*` functional options at constructor time.
- **D-11:** Register as `"openrouter"` in the dense registry with full `GetConfig`/`FromConfig` support.
- **D-12:** Config serializes all OpenRouter fields including provider preferences (as nested map). FromConfig rebuilds the full client from stored collection metadata.

### Claude's Discretion
- HTTP client setup, error handling patterns, and internal helper structure
- Test organization within the new package

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

## Project Constraints (from CLAUDE.md)

- Use conventional commits
- Never panic in production code; no `Must*` functions
- Use `testify` for assertions, `testcontainers` for integration tests
- Build tags segregate test suites (`ef` tag for embedding function tests)
- Run `make lint` before committing
- Keep things radically simple
- Do not leave too many or verbose comments
- New features target V2 API
- Follow existing provider patterns in `pkg/embeddings/`

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/pkg/errors` | (in go.mod) | Error wrapping | Used by all existing providers |
| `github.com/creasty/defaults` | (in go.mod) | Struct default values | Used by OpenAI/morph providers |
| `github.com/go-playground/validator/v10` | (in go.mod) | Struct validation | Used via `embeddings.NewValidator()` |
| `github.com/stretchr/testify` | (in go.mod) | Test assertions | Project standard |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `pkg/commons/http` | internal | `ReadLimitedBody`, `ChromaGoClientUserAgent` | HTTP response reading, user-agent header |
| `pkg/embeddings` | internal | `Secret`, `NewValidator()`, `ConfigInt()`, registry, interfaces | API key handling, validation, registration |

No new external dependencies needed. All required libraries are already in go.mod.

## Architecture Patterns

### Recommended Project Structure
```
pkg/embeddings/openrouter/
  openrouter.go       # Client, request/response types, EmbeddingFunction impl
  options.go           # With* functional options
  provider.go          # ProviderPreferences struct with custom MarshalJSON
  openrouter_test.go   # Unit tests (build tag: ef)
```

### Pattern 1: Standalone Provider (Together Pattern)
**What:** Self-contained package with own HTTP client, request/response types, no cross-provider imports.
**When to use:** When the provider has fields incompatible with the base OpenAI request shape.
**Example from Together:**
```go
type TogetherAIClient struct {
    BaseAPI      string
    APIToken     embeddings.Secret `json:"-" validate:"required"`
    APIKeyEnvVar string
    DefaultModel embeddings.EmbeddingModel
    Client       *http.Client
}

func NewTogetherClient(opts ...Option) (*TogetherAIClient, error) {
    client := &TogetherAIClient{}
    for _, opt := range opts {
        err := opt(client)
        if err != nil { return nil, errors.Wrap(err, "...") }
    }
    applyDefaults(client)
    if err := validate(client); err != nil { return nil, errors.Wrap(err, "...") }
    return client, nil
}
```

### Pattern 2: Registry Registration (init pattern)
**What:** Register factory in `init()` via `embeddings.RegisterDense`.
**Example from OpenAI:**
```go
func init() {
    if err := embeddings.RegisterDense("openrouter", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.EmbeddingFunction, error) {
        return NewOpenRouterEmbeddingFunctionFromConfig(cfg)
    }); err != nil {
        panic(err)
    }
}
```

### Pattern 3: Config Round-Trip (GetConfig/FromConfig)
**What:** Serialize all constructor options to `EmbeddingFunctionConfig` map; reconstruct from map.
**Key details:**
- API keys stored as env var name, not value: `"api_key_env_var": envVar`
- Nested objects (provider preferences) stored as `map[string]any`
- Use `embeddings.ConfigInt()`, `embeddings.ConfigFloat64()`, `embeddings.ConfigStringSlice()` helpers for type-safe extraction

### Pattern 4: WithModelString (OpenAI addition)
**What:** Simple option that sets `c.Model` to any non-empty string without validation.
**Where:** `pkg/embeddings/openai/options.go`
```go
func WithModelString(model string) Option {
    return func(c *OpenAIClient) error {
        if model == "" {
            return errors.New("model cannot be empty")
        }
        c.Model = model
        return nil
    }
}
```
**Also update `NewOpenAIEmbeddingFunctionFromConfig`** to use `WithModelString` instead of `WithModel` when the model name doesn't match known constants, so config round-trip works with arbitrary model IDs.

### Anti-Patterns to Avoid
- **Importing `pkg/embeddings/openai`:** The OpenRouter provider must NOT depend on the OpenAI package (D-02).
- **Validating model strings:** OpenRouter uses provider-prefixed IDs; no validation (D-06).
- **Using `Must*` functions:** Library code must never panic (CLAUDE.md).
- **Storing API key values in config:** Always store env var name, never the secret value.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| API key handling | Custom secret type | `embeddings.Secret` / `embeddings.NewSecret()` | Handles redaction, validation, JSON safety |
| Struct validation | Manual field checks | `embeddings.NewValidator()` | Consistent with all providers |
| HTTP response reading | Manual `io.ReadAll` | `chttp.ReadLimitedBody()` | Bounded reads, prevents OOM |
| Config type coercion | Manual type switches | `embeddings.ConfigInt()`, `ConfigFloat64()`, `ConfigStringSlice()` | Handles JSON unmarshaling type quirks |
| User-Agent header | Hardcoded string | `chttp.ChromaGoClientUserAgent` | Consistent across all providers |

## Common Pitfalls

### Pitfall 1: ProviderPreferences MarshalJSON Merging
**What goes wrong:** Custom `MarshalJSON` on `ProviderPreferences` that either drops typed fields or drops extras.
**Why it happens:** The struct has both typed fields and an `Extras map[string]any`. Standard `json.Marshal` ignores `Extras`; a naive custom marshal ignores typed fields.
**How to avoid:** Marshal the struct normally (to get typed fields), unmarshal into a `map[string]any`, merge `Extras` keys, then re-marshal the merged map. Extras keys must NOT override typed field keys.
**Warning signs:** Provider preferences JSON missing fields, or extras silently dropped.

### Pitfall 2: Config Round-Trip Type Loss
**What goes wrong:** `GetConfig()` stores `int` but `FromConfig()` receives `float64` after JSON round-trip.
**Why it happens:** `json.Unmarshal` decodes numbers as `float64` by default.
**How to avoid:** Use `embeddings.ConfigInt()` and `embeddings.ConfigFloat64()` which handle both types. For nested maps (provider preferences), expect all numbers to be `float64`.
**Warning signs:** Config round-trip tests failing with type assertion panics.

### Pitfall 3: Nil Pointer in Optional Fields
**What goes wrong:** `json:"omitempty"` on pointer fields causes marshaling issues when nil.
**Why it happens:** Using `*int` or `*string` for optional fields but not checking nil before use.
**How to avoid:** Use `omitempty` JSON tags on pointer/slice/map fields. Check nil before dereference.
**Warning signs:** Unexpected `null` in JSON or nil pointer dereference.

### Pitfall 4: OpenAI FromConfig Breaking with WithModelString
**What goes wrong:** `NewOpenAIEmbeddingFunctionFromConfig` uses `WithModel(EmbeddingModel(model))` which rejects non-standard model names stored in config.
**Why it happens:** A user creates an OpenAI EF with `WithModelString("custom-model")`, config stores `"model_name": "custom-model"`, but `FromConfig` feeds it through `WithModel` which validates.
**How to avoid:** In `FromConfig`, check if model matches known constants; if not, use `WithModelString` instead.
**Warning signs:** Config round-trip test failures for non-standard model names.

### Pitfall 5: HTTPS Validation for OpenRouter
**What goes wrong:** OpenRouter uses `https://openrouter.ai/api/v1/` which is HTTPS, so no issue. But local test servers use HTTP.
**Why it happens:** The OpenAI provider rejects HTTP without `WithInsecure()`.
**How to avoid:** Include `WithInsecure()` option in the OpenRouter provider for test scenarios. Default base URL is HTTPS so production use is fine.
**Warning signs:** Test failures when using `httptest.NewServer`.

## Code Examples

### OpenRouter Client Structure
```go
// Source: derived from existing Together/OpenAI patterns
type Client struct {
    BaseURL          string            `default:"https://openrouter.ai/api/v1/" json:"base_url,omitempty"`
    APIKey           embeddings.Secret `json:"-" validate:"required"`
    APIKeyEnvVar     string            `json:"-"`
    Model            string            `json:"model,omitempty"`
    Dimensions       *int              `json:"dimensions,omitempty"`
    User             string            `json:"user,omitempty"`
    EncodingFormat   string            `json:"encoding_format,omitempty"`
    InputType        string            `json:"input_type,omitempty"`
    Provider         *ProviderPreferences `json:"provider,omitempty"`
    HTTPClient       *http.Client      `json:"-"`
    Insecure         bool              `json:"insecure,omitempty"`
}
```

### ProviderPreferences with Custom MarshalJSON
```go
// Source: OpenRouter API docs + D-07/D-08
type ProviderPreferences struct {
    AllowFallbacks         *bool    `json:"allow_fallbacks,omitempty"`
    RequireParameters      *bool    `json:"require_parameters,omitempty"`
    DataCollection         string   `json:"data_collection,omitempty"`  // "allow" | "deny"
    ZDR                    *bool    `json:"zdr,omitempty"`
    EnforceDistillableText *bool    `json:"enforce_distillable_text,omitempty"`
    Order                  []string `json:"order,omitempty"`
    Only                   []string `json:"only,omitempty"`
    Ignore                 []string `json:"ignore,omitempty"`
    Quantizations          []string `json:"quantizations,omitempty"`
    Sort                   map[string]any `json:"sort,omitempty"`
    MaxPrice               map[string]any `json:"max_price,omitempty"`
    PreferredMinThroughput any      `json:"preferred_min_throughput,omitempty"`
    PreferredMaxLatency    any      `json:"preferred_max_latency,omitempty"`
    Extras                 map[string]any `json:"-"`
}

func (p ProviderPreferences) MarshalJSON() ([]byte, error) {
    type Alias ProviderPreferences
    data, err := json.Marshal(Alias(p))
    if err != nil {
        return nil, err
    }
    if len(p.Extras) == 0 {
        return data, nil
    }
    var merged map[string]any
    if err := json.Unmarshal(data, &merged); err != nil {
        return nil, err
    }
    for k, v := range p.Extras {
        if _, exists := merged[k]; !exists {
            merged[k] = v
        }
    }
    return json.Marshal(merged)
}
```

### CreateEmbeddingRequest
```go
type CreateEmbeddingRequest struct {
    Model          string               `json:"model"`
    Input          *Input               `json:"input"`
    Dimensions     *int                 `json:"dimensions,omitempty"`
    User           string               `json:"user,omitempty"`
    EncodingFormat string               `json:"encoding_format,omitempty"`
    InputType      string               `json:"input_type,omitempty"`
    Provider       *ProviderPreferences `json:"provider,omitempty"`
}
```

### Config Round-Trip for Provider Preferences
```go
func (e *OpenRouterEmbeddingFunction) GetConfig() embeddings.EmbeddingFunctionConfig {
    cfg := embeddings.EmbeddingFunctionConfig{
        "api_key_env_var": e.client.APIKeyEnvVar,
        "model_name":      e.client.Model,
    }
    if e.client.Provider != nil {
        provMap := make(map[string]any)
        // Marshal ProviderPreferences to map for config storage
        data, err := json.Marshal(e.client.Provider)
        if err == nil {
            _ = json.Unmarshal(data, &provMap)
        }
        cfg["provider"] = provMap
    }
    // ... other fields
    return cfg
}
```

## OpenRouter API Reference

### Embeddings Endpoint
- **URL:** `POST https://openrouter.ai/api/v1/embeddings`
- **Auth:** `Authorization: Bearer <API_KEY>`
- **Content-Type:** `application/json`

### Request Fields (HIGH confidence)
| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `model` | string | Yes | Provider-prefixed, e.g. `openai/text-embedding-3-small` |
| `input` | string or string[] | Yes | Text to embed |
| `encoding_format` | `"float"` or `"base64"` | No | Output format |
| `dimensions` | int | No | Output dimensionality |
| `user` | string | No | End-user identifier |
| `input_type` | string | No | Hint for the provider |
| `provider` | object | No | ProviderPreferences routing object |

### Response Format
Same as OpenAI: `{ object, data: [{ object, index, embedding }], model, usage }`

### ProviderPreferences Fields (HIGH confidence)
| Field | Type | Notes |
|-------|------|-------|
| `allow_fallbacks` | bool or null | Allow fallback to other providers |
| `require_parameters` | bool or null | Require all params supported |
| `data_collection` | `"deny"` or `"allow"` | Provider data collection policy |
| `zdr` | bool or null | Zero data retention |
| `enforce_distillable_text` | bool or null | Text must be distillable |
| `order` | string[] | Provider order preference |
| `only` | string[] | Only use these providers |
| `ignore` | string[] | Ignore these providers |
| `quantizations` | string[] | Allowed quantizations: int4, int8, fp4, fp6, fp8, fp16, bf16, fp32, unknown |
| `sort` | object | Sorting preferences |
| `max_price` | object | Price caps: prompt, completion, image, audio, request |
| `preferred_min_throughput` | number or percentile object | Min throughput preference |
| `preferred_max_latency` | number or percentile object | Max latency preference |

### Environment Variable
- `OPENROUTER_API_KEY` -- standard env var name for OpenRouter API keys

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify |
| Config file | Makefile targets with build tags |
| Quick run command | `go test -tags=ef -run TestOpenRouter -count=1 ./pkg/embeddings/openrouter/...` |
| Full suite command | `make test-ef` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SC-1 | CreateEmbeddingRequest has encoding_format, input_type, provider fields | unit | `go test -tags=ef -run TestRequestSerialization ./pkg/embeddings/openrouter/...` | Wave 0 |
| SC-2 | WithModel accepts provider-prefixed model IDs | unit | `go test -tags=ef -run TestModelString ./pkg/embeddings/openai/...` | Wave 0 |
| SC-3 | ProviderPreferences covers documented fields + extensibility | unit | `go test -tags=ef -run TestProviderPreferences ./pkg/embeddings/openrouter/...` | Wave 0 |
| SC-4 | Existing OpenAI behavior unchanged | unit | `go test -tags=ef ./pkg/embeddings/openai/...` | Existing |
| SC-5 | Config round-trip for OpenRouter provider | unit | `go test -tags=ef -run TestConfigRoundTrip ./pkg/embeddings/openrouter/...` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -tags=ef -count=1 ./pkg/embeddings/openrouter/... ./pkg/embeddings/openai/...`
- **Per wave merge:** `make test-ef`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `pkg/embeddings/openrouter/openrouter_test.go` -- covers SC-1, SC-3, SC-5
- [ ] OpenAI test for `WithModelString` -- covers SC-2
- [ ] Framework install: none needed (Go testing + testify already in go.mod)

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| OpenAI-only model names | Provider-prefixed IDs (openai/text-embedding-3-small) | OpenRouter convention | Must relax validation |
| No routing preferences | ProviderPreferences object | OpenRouter-specific | New typed struct needed |

## Open Questions

1. **`sort` field schema**
   - What we know: OpenRouter docs show it as an object with "empty schema"
   - What's unclear: Exact structure of sort preferences
   - Recommendation: Use `map[string]any` for maximum flexibility, document as extensible

2. **`preferred_min_throughput` / `preferred_max_latency` dual types**
   - What we know: Can be a plain number OR a percentile object with p50/p75/p90/p99 fields
   - What's unclear: Whether both forms are commonly used
   - Recommendation: Use `any` type with documentation; users pass either `float64` or `map[string]float64`

## Sources

### Primary (HIGH confidence)
- OpenRouter embeddings API docs: https://openrouter.ai/docs/api/reference/embeddings
- OpenRouter API reference: https://openrouter.ai/docs/api/api-reference/embeddings/create-embeddings
- Issue #438 (gh issue view 438) -- full problem statement and acceptance criteria
- Existing codebase: `pkg/embeddings/openai/`, `pkg/embeddings/together/`, `pkg/embeddings/registry.go`

### Secondary (MEDIUM confidence)
- OpenRouter OpenAPI spec: https://openrouter.ai/openapi.json (embeddings section not fully accessible via fetch, but fields confirmed through API reference docs)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all libraries already in go.mod, patterns well-established in codebase
- Architecture: HIGH - follows existing Together/OpenAI provider patterns exactly
- Pitfalls: HIGH - based on direct code reading of existing providers and known JSON round-trip issues
- OpenRouter API fields: HIGH - confirmed via API reference docs and issue #438

**Research date:** 2026-03-30
**Valid until:** 2026-04-30 (stable domain, unlikely to change)
