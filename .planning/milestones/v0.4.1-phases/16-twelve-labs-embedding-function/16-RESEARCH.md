# Phase 16: Twelve Labs Embedding Function - Research

**Researched:** 2026-04-01
**Domain:** Twelve Labs Embed API v2 + Go embedding provider implementation
**Confidence:** HIGH

## Summary

This phase adds a new Twelve Labs multimodal embedding provider to `pkg/embeddings/twelvelabs`. The implementation follows the well-established VoyageAI provider pattern: a client struct with functional options, dual registration (dense + content), and the full `ContentEmbeddingFunction` + `CapabilityAware` + `IntentMapper` interface set.

The Twelve Labs Embed API v2 uses a single sync endpoint (`POST https://api.twelvelabs.io/v1.3/embed-v2`) for all modalities. Each request specifies `input_type` (text/image/audio/video) with modality-specific payload fields. Authentication uses `x-api-key` header (not Bearer token). The response returns 512-dimensional float vectors.

**Primary recommendation:** Mirror the VoyageAI package structure but with a simpler API surface since Twelve Labs uses a single endpoint for all modalities and does not support batch or mixed-part requests. Each Content item maps to exactly one API call.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Use the Embed API v2 sync endpoint (`POST /v1.3/embed-v2`) for all modalities. Single unified request path.
- **D-02:** Default model: `marengo3.0`. Model is configurable via `WithModel` option.
- **D-03:** Sync endpoint only. Async support deferred to separate GitHub issue.
- **D-04:** Advertise text, image, audio, and video in `CapabilityMetadata.Modalities`.
- **D-05:** Single-modality per Content item only. `SupportsMixedPart: false`.
- **D-06:** Input mapping: `SourceKindURL` -> `media_source.url`, `SourceKindBase64`/`SourceKindBytes`/`SourceKindFile` -> `media_source.base_64_string`. No `asset_id` support.
- **D-07:** MIME type resolution: `BinarySource.MIMEType` first -> infer from extension -> fail explicitly.
- **D-08:** Environment variable: `TWELVE_LABS_API_KEY`.
- **D-09:** Audio `embedding_option` exposed as `WithAudioEmbeddingOption("audio"|"transcription"|"fused")` functional option. Default: `"audio"`.
- **D-10:** Embedding dimensions fixed at 512 (Marengo 3.0 spec). No dimensionality option needed.
- **D-11:** httptest-based unit tests with `ef` build tag.
- **D-12:** Separate integration test file with dedicated build tag that hits real API.
- **D-13:** Test both `ContentEmbeddingFunction` and `EmbeddingFunction` interfaces via dual registration.

### Claude's Discretion
- Request body construction details and error mapping
- Config round-trip key naming
- Provider-side byte resolution implementation details

### Deferred Ideas (OUT OF SCOPE)
- Async embedding support (`POST /v1.3/embed-v2/tasks` for audio/video up to 4 hours)
</user_constraints>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `net/http` | stdlib | HTTP client for API calls | All existing providers use stdlib HTTP |
| `encoding/json` | stdlib | JSON marshal/unmarshal | Standard for API request/response |
| `github.com/pkg/errors` | existing dep | Error wrapping | Project-wide convention |
| `github.com/amikos-tech/chroma-go/pkg/embeddings` | local | Shared interfaces | ContentEmbeddingFunction, CapabilityAware, IntentMapper |
| `github.com/amikos-tech/chroma-go/pkg/commons/http` | local | HTTP utilities | ChromaGoClientUserAgent, ReadLimitedBody |
| `github.com/amikos-tech/chroma-go/pkg/internal/pathutil` | local | File path safety | ValidateFilePath for SourceKindFile |

### Supporting (test only)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `net/http/httptest` | stdlib | Mock HTTP server | Unit tests with mocked API responses |
| `github.com/stretchr/testify` | existing dep | Test assertions | All test files |

No new dependencies required.

## Architecture Patterns

### Recommended Project Structure
```
pkg/embeddings/twelvelabs/
    twelvelabs.go           # Client, EF struct, EmbedDocuments/EmbedQuery, GetConfig, init()
    content.go              # EmbedContent/EmbedContents, resolveBytes, resolveMIME, capabilities
    option.go               # Functional options (WithModel, WithAPIKey, etc.)
    twelvelabs_test.go      # httptest unit tests (build tag: ef)
    twelvelabs_content_test.go  # Content API unit tests (build tag: ef)
    twelvelabs_integration_test.go  # Real API integration tests (build tag: twelvelabs)
```

### Pattern 1: Unified Single-Endpoint Client
**What:** Unlike VoyageAI which has separate text and multimodal endpoints, Twelve Labs uses one endpoint for all modalities. The request body structure changes per `input_type`.
**When to use:** Always. All four modalities route through `POST /v1.3/embed-v2`.
**Example:**
```go
// Text request
{
    "input_type": "text",
    "model_name": "marengo3.0",
    "text": {"input_text": "hello world"}
}

// Image request
{
    "input_type": "image",
    "model_name": "marengo3.0",
    "image": {"media_source": {"url": "https://example.com/photo.png"}}
}

// Audio request
{
    "input_type": "audio",
    "model_name": "marengo3.0",
    "audio": {
        "media_source": {"base_64_string": "..."},
        "embedding_option": "audio"
    }
}

// Video request
{
    "input_type": "video",
    "model_name": "marengo3.0",
    "video": {"media_source": {"url": "https://example.com/clip.mp4"}}
}
```

### Pattern 2: Struct-Literal Test Construction
**What:** Unit tests construct `TwelveLabsEmbeddingFunction` via struct literal + injected `*TwelveLabsClient` (with overridden BaseAPI pointing to httptest server), bypassing real API key validation.
**When to use:** All httptest-based tests.
**Example (from VoyageAI pattern):**
```go
ef := &TwelveLabsEmbeddingFunction{
    apiClient: &TwelveLabsClient{
        BaseAPI:      server.URL + "/v1.3/embed-v2",
        APIKey:       embeddings.NewSecret("test-key"),
        DefaultModel: defaultModel,
    },
}
```

### Pattern 3: Dual Registration in init()
**What:** Register as `"twelvelabs"` in both `RegisterDense` and `RegisterContent`.
**When to use:** Always. Required for config round-trip and auto-wiring.
**Example (from VoyageAI):**
```go
func init() {
    embeddings.RegisterDense("twelvelabs", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.EmbeddingFunction, error) {
        return NewTwelveLabsEmbeddingFunctionFromConfig(cfg)
    })
    embeddings.RegisterContent("twelvelabs", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.ContentEmbeddingFunction, error) {
        return NewTwelveLabsEmbeddingFunctionFromConfig(cfg)
    })
}
```

### Pattern 4: One API Call per Content Item
**What:** Twelve Labs does not support batching multiple items in a single API call. `EmbedContents` loops over items, calling the API once per Content.
**When to use:** Always. `SupportsBatch: false` in capabilities.
**Important:** This differs from VoyageAI which sends batch requests. The loop is in `EmbedContents`.

### Anti-Patterns to Avoid
- **Separate code paths per modality at the client level:** Use a single `doEmbed` method that accepts the modality-specific request body. The endpoint URL is the same for all modalities.
- **Batching Content items into one API call:** Twelve Labs does not support batching. Each Content item is one API call.
- **Using `Authorization: Bearer` header:** Twelve Labs uses `x-api-key` header, not Bearer token.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTTP POST with JSON | Custom HTTP wrapper | `VoyageAI.doPost` pattern with `chttp.ReadLimitedBody` | Consistent error handling, body limits |
| File path validation | Path traversal checks | `pathutil.ValidateFilePath` | Shared, tested utility |
| Secret handling | String API key field | `embeddings.Secret` type | Prevents accidental logging |
| Config validation | Manual nil/empty checks | `embeddings.NewValidator()` struct validation | Consistent with all providers |

## Common Pitfalls

### Pitfall 1: Auth Header Format
**What goes wrong:** Using `Authorization: Bearer <key>` instead of `x-api-key: <key>`.
**Why it happens:** Most embedding APIs (OpenAI, Voyage, etc.) use Bearer auth. Twelve Labs is different.
**How to avoid:** Hardcode `x-api-key` header in the `doPost` method. Do not reuse Voyage's Bearer pattern.
**Warning signs:** 401 responses from API.

### Pitfall 2: Request Body Structure per Modality
**What goes wrong:** Using the same field names across modalities (e.g., `text` field for all types).
**Why it happens:** Each modality has its own top-level key: `text`, `image`, `audio`, `video`.
**How to avoid:** Use `json:"...,omitempty"` on all modality-specific fields so only the relevant one is serialized.
**Warning signs:** API returning "invalid input_type" or "missing field" errors.

### Pitfall 3: Audio embedding_option Serialization
**What goes wrong:** Sending `embedding_option` as an array instead of a string for the sync endpoint.
**Why it happens:** The async API documentation shows `embedding_option` as an array. The sync endpoint uses a string.
**How to avoid:** Per CONTEXT.md D-09, use a single string value: `"audio"`, `"transcription"`, or `"fused"`.
**Warning signs:** API returning type validation errors for audio requests.

### Pitfall 4: No Batch Support
**What goes wrong:** Trying to send multiple items in a single API call.
**Why it happens:** VoyageAI supports batching, tempting copy-paste.
**How to avoid:** `EmbedContents` must loop and call the API once per item. Set `SupportsBatch: false` in capabilities.
**Warning signs:** API errors about request format.

### Pitfall 5: Base64 vs URL Media Source
**What goes wrong:** Sending both `url` and `base_64_string` in `media_source`, or sending raw bytes without encoding.
**Why it happens:** The `media_source` object expects exactly one of `url` or `base_64_string`.
**How to avoid:** `resolveBytes` pattern: URL -> passthrough as `media_source.url`; all other SourceKinds -> resolve to bytes -> base64 encode -> `media_source.base_64_string`.
**Warning signs:** API returning "invalid media_source" errors.

## Code Examples

### Request Body Types
```go
// Source: Twelve Labs API docs + CONTEXT.md specifics section
type EmbedV2Request struct {
    InputType string          `json:"input_type"`
    ModelName string          `json:"model_name"`
    Text      *TextInput      `json:"text,omitempty"`
    Image     *ImageInput     `json:"image,omitempty"`
    Audio     *AudioInput     `json:"audio,omitempty"`
    Video     *VideoInput     `json:"video,omitempty"`
}

type TextInput struct {
    InputText string `json:"input_text"`
}

type MediaSource struct {
    URL          string `json:"url,omitempty"`
    Base64String string `json:"base_64_string,omitempty"`
}

type ImageInput struct {
    MediaSource MediaSource `json:"media_source"`
}

type AudioInput struct {
    MediaSource     MediaSource `json:"media_source"`
    EmbeddingOption string      `json:"embedding_option,omitempty"`
}

type VideoInput struct {
    MediaSource MediaSource `json:"media_source"`
}
```

### Response Body
```go
// Source: Twelve Labs API docs
type EmbedV2Response struct {
    Data []EmbedV2DataItem `json:"data"`
}

type EmbedV2DataItem struct {
    Embedding       []float64 `json:"embedding"`
    EmbeddingOption string    `json:"embedding_option,omitempty"`
}
```

### Auth Header Pattern
```go
// Source: Twelve Labs Authentication docs
httpReq.Header.Set("x-api-key", c.APIKey.Value())
httpReq.Header.Set("Content-Type", "application/json")
httpReq.Header.Set("Accept", "application/json")
httpReq.Header.Set("User-Agent", chttp.ChromaGoClientUserAgent)
```

### Capabilities Declaration
```go
func (e *TwelveLabsEmbeddingFunction) Capabilities() embeddings.CapabilityMetadata {
    return embeddings.CapabilityMetadata{
        Modalities: []embeddings.Modality{
            embeddings.ModalityText,
            embeddings.ModalityImage,
            embeddings.ModalityAudio,
            embeddings.ModalityVideo,
        },
        Intents: []embeddings.Intent{
            embeddings.IntentRetrievalQuery,
            embeddings.IntentRetrievalDocument,
        },
        SupportsBatch:     false,
        SupportsMixedPart: false,
    }
}
```

### Config Round-Trip
```go
func (e *TwelveLabsEmbeddingFunction) GetConfig() embeddings.EmbeddingFunctionConfig {
    cfg := embeddings.EmbeddingFunctionConfig{
        "api_key_env_var":        envVar,
        "model_name":             string(e.apiClient.DefaultModel),
        "audio_embedding_option": e.apiClient.AudioEmbeddingOption,
    }
    if e.apiClient.BaseAPI != defaultBaseAPI {
        cfg["base_url"] = e.apiClient.BaseAPI
    }
    if e.apiClient.Insecure {
        cfg["insecure"] = true
    }
    return cfg
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Marengo 2.7 (1024d) | Marengo 3.0 (512d) | March 30, 2026 (sunset) | Default model must be `marengo3.0` |
| Embed API v1 | Embed API v2 (`/v1.3/embed-v2`) | 2025 | New endpoint path and request format |

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify |
| Config file | none (build tags in file headers) |
| Quick run command | `go test -tags=ef -run TestTwelveLabs -count=1 ./pkg/embeddings/twelvelabs/...` |
| Full suite command | `go test -tags=ef -count=1 ./pkg/embeddings/twelvelabs/...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SC-1 | Implements dense + Content API interfaces | unit | `go test -tags=ef -run TestTwelveLabsEmbed -count=1 ./pkg/embeddings/twelvelabs/...` | Wave 0 |
| SC-2 | Supports text, image, audio, video modalities | unit | `go test -tags=ef -run TestTwelveLabsModality -count=1 ./pkg/embeddings/twelvelabs/...` | Wave 0 |
| SC-3 | Registered in factory/registry | unit | `go test -tags=ef -run TestTwelveLabsRegistry -count=1 ./pkg/embeddings/twelvelabs/...` | Wave 0 |
| SC-4 | Tests cover request construction, modality validation, config persistence | unit | `go test -tags=ef -count=1 ./pkg/embeddings/twelvelabs/...` | Wave 0 |
| SC-5 | Docs and examples | manual | N/A | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -tags=ef -count=1 ./pkg/embeddings/twelvelabs/...`
- **Per wave merge:** `make test-ef`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `pkg/embeddings/twelvelabs/twelvelabs_test.go` -- httptest unit tests for text embedding
- [ ] `pkg/embeddings/twelvelabs/twelvelabs_content_test.go` -- Content API unit tests
- [ ] All test files need `//go:build ef` build tag

## Sources

### Primary (HIGH confidence)
- Twelve Labs Authentication docs (https://docs.twelvelabs.io/api-reference/authentication) -- `x-api-key` header format confirmed
- Twelve Labs Marengo model specs (https://docs.twelvelabs.io/docs/concepts/models/marengo) -- 512d, supported formats
- Twelve Labs text embedding guide (https://docs.twelvelabs.io/docs/guides/create-embeddings/text) -- request body format
- Twelve Labs audio embedding guide (https://docs.twelvelabs.io/docs/guides/create-embeddings/audio) -- embedding_option values
- VoyageAI provider source (`pkg/embeddings/voyage/`) -- reference implementation pattern

### Secondary (MEDIUM confidence)
- Twelve Labs Embed API v2 reference (https://docs.twelvelabs.io/api-reference/create-embeddings-v2) -- endpoint path confirmed, full schema not fully extracted from docs page
- CONTEXT.md specifics section -- API request body JSON verified against Twelve Labs guides

### Tertiary (LOW confidence)
- Audio sync endpoint `embedding_option` as string vs array -- CONTEXT.md says string, API docs show array for async. Sync endpoint format needs validation during implementation.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new deps, follows established patterns exactly
- Architecture: HIGH -- CONTEXT.md decisions are comprehensive, VoyageAI reference pattern is clear
- Pitfalls: MEDIUM -- auth header and request body format verified, but sync audio embedding_option format has minor uncertainty

**Research date:** 2026-04-01
**Valid until:** 2026-05-01 (stable API, model already released)
