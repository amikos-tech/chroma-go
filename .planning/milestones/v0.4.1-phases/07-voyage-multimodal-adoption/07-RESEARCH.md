# Phase 7: Voyage Multimodal Adoption - Research

**Researched:** 2026-03-22
**Domain:** VoyageAI multimodal embedding provider integration into the shared Content API
**Confidence:** HIGH

## Summary

Phase 7 extends the existing VoyageAI text-only provider (`pkg/embeddings/voyage/`) to implement the shared `ContentEmbeddingFunction`, `CapabilityAware`, and `IntentMapper` interfaces established in Phases 1-5 and battle-tested by Gemini in Phase 6. The implementation follows the identical dual-registration + dual-endpoint pattern that Gemini established, adapted for Voyage's different API shape: Voyage uses a separate `/v1/multimodalembeddings` endpoint (vs Gemini's single `EmbedContent` call), has only 2 input_type values (vs Gemini's 8 task types), and natively supports text + image + video (no audio or PDF).

The existing `VoyageAIClient` struct, functional options pattern, response unmarshaling (`EmbeddingTypeResult.UnmarshalJSON`), and context-override helpers (`getModel`, `getTruncation`, etc.) are directly reusable. The new work is: (1) a `CreateMultimodalEmbedding` method on `VoyageAIClient` that talks to `/v1/multimodalembeddings`, (2) `EmbedContent`/`EmbedContents` methods on `VoyageAIEmbeddingFunction` that validate, convert, and delegate, (3) `Capabilities()` and `MapIntent()` implementations, and (4) `RegisterContent("voyageai", ...)` in `init()`.

A critical finding during research: the Voyage multimodal REST API does NOT list `output_dimension` as a parameter, even though the model page says voyage-multimodal-3.5 supports 256/512/1024/2048 dimensions. The Python SDK does pass `output_dimension` through. The implementation should include `output_dimension` in the request body (following the Python SDK pattern) but should NOT advertise `RequestOptionDimension` in capabilities until runtime verification confirms the API accepts it. This is flagged as a validation item.

**Primary recommendation:** Follow the Gemini Phase 6 pattern exactly (content.go + interface implementations + dual registration + unit tests), adapted for Voyage's separate multimodal endpoint and simpler intent model.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Extend existing `pkg/embeddings/voyage/` package -- no new package needed
- **D-02:** Add `CreateMultimodalEmbedding()` to existing `VoyageAIClient`, targeting `/v1/multimodalembeddings` endpoint
- **D-03:** Keep existing `VoyageAIEmbeddingFunction` unchanged; add `ContentEmbeddingFunction`, `CapabilityAware`, `IntentMapper` interface implementations
- **D-04:** Dual registration: keep existing `RegisterDense("voyageai")` and add `RegisterContent("voyageai")`
- **D-05:** Map `SourceKindURL` to Voyage `image_url`/`video_url` type; `SourceKindBase64`/`SourceKindBytes`/`SourceKindFile` resolve to bytes then `image_base64`/`video_base64` with `data:<mimetype>;base64,<data>` URI prefix
- **D-06:** Provider-side byte resolution: Voyage's `EmbedContent` impl resolves all `BinarySource` kinds
- **D-07:** MIME type resolved from `BinarySource.MIMEType` first; file-backed sources infer from extension as fallback; fail explicitly if MIME type is empty and can't be inferred
- **D-08:** Mixed-part Content items supported natively -- `SupportsMixedPart: true`
- **D-09:** Advertise text, image, and video in `CapabilityMetadata.Modalities`
- **D-10:** Default model: `voyage-multimodal-3.5` for multimodal instances
- **D-11:** Supported image formats: PNG, JPEG, WEBP, GIF
- **D-12:** Video format: MP4 only
- **D-13:** IntentMapper mapping: `retrieval_query` -> `"query"`, `retrieval_document` -> `"document"`
- **D-14:** `MapIntent` rejects `classification`, `clustering`, `semantic_similarity` with explicit errors
- **D-15:** `MapIntent` checks `ProviderHints["input_type"]` first (override), then maps neutral intent, then passes null for no-intent
- **D-16:** Batch requests reject per-item Intent fields; single-item requests allow per-item ProviderHints override
- **D-17:** Declare only `retrieval_query` + `retrieval_document` in `CapabilityMetadata.Intents`
- **D-18:** `voyage-multimodal-3.5` supports output dimensions: 256, 512, 1024 (default), 2048
- **D-19:** Advertise `RequestOptionDimension` in `CapabilityMetadata.RequestOptions`
- **D-20:** Per-request `Dimension` field maps to Voyage's `output_dimension` parameter (needs runtime verification)
- **D-21:** Content factory builds from the same config schema -- no new config fields
- **D-22:** Allowed config fields per upstream schema: `model_name`, `api_key_env_var`, `input_type`, `truncation`
- **D-23:** Capabilities are runtime-derived from model name, not persisted in config
- **D-24:** Config round-trip shape stays unchanged: `{type: "known", name: "voyageai", config: {...}}`
- **D-25:** Shared helpers, separate entry points -- `CreateEmbedding` stays unchanged, new `CreateMultimodalEmbedding` added
- **D-26:** Existing `EmbedDocuments`/`EmbedQuery` delegate through unchanged `CreateEmbedding` on `/v1/embeddings`
- **D-27:** New `EmbedContent`/`EmbedContents` delegate through `CreateMultimodalEmbedding` on `/v1/multimodalembeddings`

### Claude's Discretion
- Internal helper function names and organization (e.g., `resolveSource`, `buildContentBlock`, `resolveMIME`)
- Exact placement of `ValidateContentSupport` call within `EmbedContent`/`EmbedContents`
- Test scaffolding structure and assertion patterns
- Error message wording for unsupported modality/intent combinations
- Whether to add `output_encoding` (base64 response optimization) support in this phase or defer

### Deferred Ideas (OUT OF SCOPE)
- vLLM/Nemotron provider -- blocked on vLLM adding `NVOmniEmbedModel` architecture support
- Voyage `output_encoding: "base64"` response optimization -- future enhancement
- Video-specific integration tests -- start with text + image, add video tests later
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| VOY-01 | VoyageAI implements `ContentEmbeddingFunction`, `CapabilityAware`, and `IntentMapper` for text, image, and video modalities | Gemini Phase 6 provides the exact pattern: compile-time interface assertions, `capabilitiesForModel()` derivation, `MapIntent()` with neutral-only mapping, `EmbedContent`/`EmbedContents` delegation through a new `CreateMultimodalEmbedding` method |
| VOY-02 | Neutral intents map to Voyage input types with explicit errors for unsupported combinations | Voyage only supports `"query"` and `"document"` input_type values; the 3 other neutral intents (classification, clustering, semantic_similarity) must return explicit errors. ProviderHints["input_type"] escape hatch for direct pass-through |
| VOY-03 | VoyageAI is registered in the multimodal factory/registry path with config round-trip support | `RegisterContent("voyageai", ...)` in `init()` alongside existing `RegisterDense`. Config round-trip uses same `GetConfig()` and `NewVoyageAIEmbeddingFunctionFromConfig` -- no new config fields needed per upstream schema constraint |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/pkg/errors` | (existing dep) | Error wrapping | Already used throughout codebase |
| `encoding/json` | stdlib | JSON marshal/unmarshal for API requests | Standard HTTP API client |
| `encoding/base64` | stdlib | Binary-to-base64 conversion for image/video payloads | Voyage requires data URI format |
| `net/http` | stdlib | HTTP client for API calls | Already used by VoyageAIClient |
| `github.com/stretchr/testify` | (existing dep) | Test assertions | Project standard per CLAUDE.md |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `io` / `os` | stdlib | File reading for SourceKindFile | resolveBytes helper |
| `path/filepath` | stdlib | Extension extraction for MIME inference | resolveMIME helper |
| `strings` | stdlib | MIME type prefix checking | validateMIMEModality |

No new dependencies required. Everything uses existing project dependencies.

## Architecture Patterns

### Recommended File Structure
```
pkg/embeddings/voyage/
  option.go              # existing - functional options (unchanged)
  voyage.go              # existing - VoyageAIClient, text API, EmbeddingFunction (minimal additions)
  content.go             # NEW - multimodal types, Content interface impls, helpers
  voyage_test.go         # existing - integration tests (unchanged)
  voyage_content_test.go # NEW - unit tests for content functionality
```

### Pattern 1: Dual-Endpoint Provider (Gemini Reference)
**What:** Provider has separate text and multimodal API endpoints but exposes a single `VoyageAIEmbeddingFunction` that implements both `EmbeddingFunction` and `ContentEmbeddingFunction`.
**When to use:** When the provider's multimodal API is a separate endpoint from the text API.
**Example:**
```go
// Compile-time interface assertions (same pattern as Gemini)
var _ embeddings.EmbeddingFunction = (*VoyageAIEmbeddingFunction)(nil)
var _ embeddings.ContentEmbeddingFunction = (*VoyageAIEmbeddingFunction)(nil)
var _ embeddings.CapabilityAware = (*VoyageAIEmbeddingFunction)(nil)
var _ embeddings.IntentMapper = (*VoyageAIEmbeddingFunction)(nil)

// EmbedContent delegates to CreateMultimodalEmbedding (new endpoint)
func (e *VoyageAIEmbeddingFunction) EmbedContent(ctx context.Context, content embeddings.Content) (embeddings.Embedding, error) {
    if err := content.Validate(); err != nil {
        return nil, err
    }
    caps := e.Capabilities()
    if err := embeddings.ValidateContentSupport(content, caps); err != nil {
        return nil, err
    }
    result, err := e.apiClient.CreateMultimodalEmbedding(ctx, []embeddings.Content{content}, e)
    if err != nil {
        return nil, err
    }
    if len(result) == 0 {
        return nil, errors.New("no embedding returned")
    }
    return result[0], nil
}

// EmbedDocuments continues to use CreateEmbedding (text endpoint, unchanged)
```

### Pattern 2: Model-Based Capability Derivation
**What:** Capabilities are determined at runtime from the configured model name, not hardcoded or persisted.
**When to use:** When different models within the same provider have different capability sets.
**Example:**
```go
func capabilitiesForModel(model string) embeddings.CapabilityMetadata {
    switch model {
    case "voyage-multimodal-3.5":
        return embeddings.CapabilityMetadata{
            Modalities:     []embeddings.Modality{embeddings.ModalityText, embeddings.ModalityImage, embeddings.ModalityVideo},
            Intents:        []embeddings.Intent{embeddings.IntentRetrievalQuery, embeddings.IntentRetrievalDocument},
            RequestOptions: []embeddings.RequestOption{embeddings.RequestOptionDimension},
            SupportsBatch:     true,
            SupportsMixedPart: true,
        }
    case "voyage-multimodal-3":
        return embeddings.CapabilityMetadata{
            Modalities:     []embeddings.Modality{embeddings.ModalityText, embeddings.ModalityImage},
            Intents:        []embeddings.Intent{embeddings.IntentRetrievalQuery, embeddings.IntentRetrievalDocument},
            SupportsBatch:     true,
            SupportsMixedPart: true,
        }
    default:
        // Non-multimodal models: text-only
        return embeddings.CapabilityMetadata{
            Modalities:    []embeddings.Modality{embeddings.ModalityText},
            SupportsBatch: true,
        }
    }
}
```

### Pattern 3: Voyage Multimodal Request Format
**What:** Voyage multimodal API expects inputs as arrays of objects with "content" arrays containing typed content blocks.
**When to use:** Converting shared `Content` items to Voyage API format.
**Example:**
```go
// Voyage API expects this JSON shape:
// {
//   "inputs": [
//     {"content": [{"type": "text", "text": "..."}, {"type": "image_base64", "image_base64": "data:image/png;base64,..."}]},
//     {"content": [{"type": "image_url", "image_url": "https://..."}]}
//   ],
//   "model": "voyage-multimodal-3.5",
//   "input_type": "query",
//   "truncation": true,
//   "output_dimension": 1024
// }

type MultimodalContentBlock struct {
    Type        string `json:"type"`
    Text        string `json:"text,omitempty"`
    ImageBase64 string `json:"image_base64,omitempty"`
    ImageURL    string `json:"image_url,omitempty"`
    VideoBase64 string `json:"video_base64,omitempty"`
    VideoURL    string `json:"video_url,omitempty"`
}

type MultimodalInput struct {
    Content []MultimodalContentBlock `json:"content"`
}

type CreateMultimodalEmbeddingRequest struct {
    Model           string            `json:"model"`
    Inputs          []MultimodalInput `json:"inputs"`
    InputType       *InputType        `json:"input_type"`
    Truncation      *bool             `json:"truncation"`
    OutputDimension *int              `json:"output_dimension,omitempty"`
}
```

### Pattern 4: Intent Mapping with ProviderHints Escape Hatch
**What:** MapIntent only handles the 2 supported neutral intents; ProviderHints["input_type"] provides a direct override.
**When to use:** In resolveInputTypeForContent before sending API request.
**Example:**
```go
func (e *VoyageAIEmbeddingFunction) MapIntent(intent embeddings.Intent) (string, error) {
    if !embeddings.IsNeutralIntent(intent) {
        return "", errors.Errorf("unsupported intent %q: use ProviderHints[\"input_type\"] for Voyage-native input types", intent)
    }
    switch intent {
    case embeddings.IntentRetrievalQuery:
        return string(InputTypeQuery), nil
    case embeddings.IntentRetrievalDocument:
        return string(InputTypeDocument), nil
    default:
        return "", errors.Errorf("intent %q is not supported by Voyage; only retrieval_query and retrieval_document are available", intent)
    }
}
```

### Anti-Patterns to Avoid
- **Adding multimodal logic into voyage.go:** Keep new content code in content.go. The existing voyage.go should only gain the `CreateMultimodalEmbedding` client method and interface assertion lines.
- **Modifying existing EmbedDocuments/EmbedQuery signatures:** These must remain unchanged per D-26.
- **Persisting capability info in config:** Capabilities derive from model name at runtime per D-23.
- **Silent degradation for unsupported intents:** Must return explicit errors per D-14.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Content validation | Custom validation | `content.Validate()` + `ValidateContentSupport(content, caps)` | Already built in Phases 1/4, handles all edge cases |
| Batch validation | Custom batch checks | `ValidateContents(contents)` + `ValidateContentsSupport(contents, caps)` | Consistent error format with batch indexing |
| MIME resolution | Custom MIME lookup | Reuse Gemini's `resolveMIME` pattern (or extract to shared) | Same logic needed; extension-to-MIME map |
| Byte resolution | Custom file/base64 reading | Reuse Gemini's `resolveBytes` pattern (or extract to shared) | File reading, base64 decoding, size limits already handled |
| Intent neutrality check | Custom intent string matching | `embeddings.IsNeutralIntent(intent)` | Already built in Phase 1 |
| Config helpers | Custom type coercion | `embeddings.ConfigInt()`, `ConfigFloat64()` | Already built for registry config parsing |

**Key insight:** The resolveBytes and resolveMIME functions from Gemini's content.go can be copied into Voyage's content.go (they don't depend on genai). Alternatively, they could be extracted to a shared helper, but that's a refactor beyond this phase's scope. Copying is acceptable since they are small, self-contained functions.

## Common Pitfalls

### Pitfall 1: Voyage data URI format for base64 payloads
**What goes wrong:** Sending raw base64 data instead of data URI format to Voyage API.
**Why it happens:** Gemini uses `genai.NewPartFromBytes(data, mimeType)` which handles encoding. Voyage expects `data:<mimetype>;base64,<encoded_data>` as the value of `image_base64` or `video_base64`.
**How to avoid:** The content block builder must always prepend `data:<mimetype>;base64,` before the base64-encoded bytes.
**Warning signs:** API returns 400 errors about invalid image/video format.

### Pitfall 2: URL vs Base64 type selection for Voyage content blocks
**What goes wrong:** Using `image_base64` type when the source is a URL, or vice versa.
**Why it happens:** Voyage has 5 distinct content block types: `text`, `image_url`, `image_base64`, `video_url`, `video_base64`. The type must match the payload format.
**How to avoid:** Map `SourceKindURL` to `image_url`/`video_url` types. All other source kinds resolve to bytes, encode as base64, and use `image_base64`/`video_base64` types.
**Warning signs:** API rejects requests with type/payload mismatches.

### Pitfall 3: output_dimension API support uncertainty
**What goes wrong:** Advertising dimension support but the API silently ignores the parameter, or rejects it.
**Why it happens:** The Voyage REST API docs don't list `output_dimension` for the multimodal endpoint, but the Python SDK passes it and the model page lists dimension options.
**How to avoid:** Include `output_dimension` in the request (matching Python SDK), advertise `RequestOptionDimension` in capabilities, but add a code comment noting the documentation gap. Integration tests should verify dimension output actually changes.
**Warning signs:** Embeddings always return 1024-dimensional vectors regardless of requested dimension.

### Pitfall 4: BaseAPI URL divergence between text and multimodal
**What goes wrong:** Using the same base URL for both endpoints when Voyage uses different paths.
**Why it happens:** The existing `VoyageAIClient.BaseAPI` defaults to `https://api.voyageai.com/v1/embeddings`. The multimodal endpoint is `https://api.voyageai.com/v1/multimodalembeddings`.
**How to avoid:** The `CreateMultimodalEmbedding` method must derive the multimodal URL, not use `c.BaseAPI` directly. Either store a separate multimodal base URL or derive it by replacing the path component.
**Warning signs:** 404 errors when sending multimodal requests to the text embeddings endpoint.

### Pitfall 5: Per-item overrides in batch requests
**What goes wrong:** Allowing per-item Intent or ProviderHints in batch requests, which Voyage silently applies to the entire batch.
**Why it happens:** The Voyage API applies `input_type` at the request level, not per-input.
**How to avoid:** Reject per-item Intent/ProviderHints/Dimension in batch requests (len > 1) with explicit errors, exactly as Gemini does. Single-item requests allow per-item overrides.
**Warning signs:** Silent incorrect embeddings when different items have different intents.

### Pitfall 6: Multimodal endpoint URL derivation with custom BaseAPI
**What goes wrong:** When a user sets a custom `BaseAPI` (e.g., for proxy or local testing), the multimodal URL derivation breaks.
**Why it happens:** Naive string replacement or path manipulation fails for non-standard base URLs.
**How to avoid:** Use the base host from `BaseAPI` but construct the multimodal path independently. The simplest approach: store a `MultimodalBaseAPI` field that defaults to the standard URL and derives from `BaseAPI` when custom.
**Warning signs:** Tests with custom base URLs fail or route to wrong endpoint.

## Code Examples

### Voyage Multimodal Request/Response Types
```go
// Source: https://docs.voyageai.com/reference/multimodal-embeddings-api

type MultimodalContentBlock struct {
    Type        string `json:"type"`
    Text        string `json:"text,omitempty"`
    ImageBase64 string `json:"image_base64,omitempty"`
    ImageURL    string `json:"image_url,omitempty"`
    VideoBase64 string `json:"video_base64,omitempty"`
    VideoURL    string `json:"video_url,omitempty"`
}

type MultimodalInput struct {
    Content []MultimodalContentBlock `json:"content"`
}

type CreateMultimodalEmbeddingRequest struct {
    Model           string            `json:"model"`
    Inputs          []MultimodalInput `json:"inputs"`
    InputType       *InputType        `json:"input_type"`
    Truncation      *bool             `json:"truncation"`
    OutputDimension *int              `json:"output_dimension,omitempty"`
}

// Response reuses existing CreateEmbeddingResponse (same shape: object, data, model, usage)
// The multimodal response adds text_tokens, image_pixels, video_pixels to usage,
// but the embedding data structure is identical.
```

### Content-to-VoyageInput Conversion
```go
// Convert a shared Content item to a Voyage MultimodalInput
func convertToVoyageInput(ctx context.Context, content embeddings.Content, maxFileSize int64) (*MultimodalInput, error) {
    blocks := make([]MultimodalContentBlock, 0, len(content.Parts))
    for i, part := range content.Parts {
        block, err := buildContentBlock(ctx, part, maxFileSize)
        if err != nil {
            return nil, errors.Wrapf(err, "part[%d]", i)
        }
        blocks = append(blocks, block)
    }
    return &MultimodalInput{Content: blocks}, nil
}

func buildContentBlock(ctx context.Context, part embeddings.Part, maxFileSize int64) (MultimodalContentBlock, error) {
    switch part.Modality {
    case embeddings.ModalityText:
        return MultimodalContentBlock{Type: "text", Text: part.Text}, nil
    case embeddings.ModalityImage:
        return buildBinaryBlock(ctx, part.Source, "image", maxFileSize)
    case embeddings.ModalityVideo:
        return buildBinaryBlock(ctx, part.Source, "video", maxFileSize)
    default:
        return MultimodalContentBlock{}, errors.Errorf("unsupported modality %q", part.Modality)
    }
}

func buildBinaryBlock(ctx context.Context, source *embeddings.BinarySource, mediaType string, maxFileSize int64) (MultimodalContentBlock, error) {
    if source.Kind == embeddings.SourceKindURL {
        // URL passthrough
        block := MultimodalContentBlock{Type: mediaType + "_url"}
        switch mediaType {
        case "image":
            block.ImageURL = source.URL
        case "video":
            block.VideoURL = source.URL
        }
        return block, nil
    }
    // All other kinds: resolve to bytes, encode as base64 data URI
    mimeType, err := resolveMIME(source)
    if err != nil {
        return MultimodalContentBlock{}, err
    }
    data, err := resolveBytes(ctx, source, maxFileSize)
    if err != nil {
        return MultimodalContentBlock{}, err
    }
    dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data))
    block := MultimodalContentBlock{Type: mediaType + "_base64"}
    switch mediaType {
    case "image":
        block.ImageBase64 = dataURI
    case "video":
        block.VideoBase64 = dataURI
    }
    return block, nil
}
```

### Dual Registration in init()
```go
func init() {
    if err := embeddings.RegisterDense("voyageai", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.EmbeddingFunction, error) {
        return NewVoyageAIEmbeddingFunctionFromConfig(cfg)
    }); err != nil {
        panic(err)
    }
    if err := embeddings.RegisterContent("voyageai", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.ContentEmbeddingFunction, error) {
        return NewVoyageAIEmbeddingFunctionFromConfig(cfg)
    }); err != nil {
        panic(err)
    }
}
```

### Intent Resolution for Content Requests
```go
// resolveInputTypeForContent determines the effective Voyage input_type for a content request.
// Priority: ProviderHints["input_type"] > intent via mapper > context input_type > nil.
func resolveInputTypeForContent(content embeddings.Content, defaultInputType *InputType, mapper embeddings.IntentMapper) (*InputType, error) {
    if hints := content.ProviderHints; hints != nil {
        if raw, ok := hints["input_type"]; ok {
            hintStr, ok := raw.(string)
            if !ok || hintStr == "" {
                return nil, errors.New("ProviderHints[\"input_type\"] must be a non-empty string")
            }
            it := InputType(hintStr)
            return &it, nil
        }
    }
    if content.Intent != "" && mapper != nil {
        mapped, err := mapper.MapIntent(content.Intent)
        if err != nil {
            return nil, errors.Wrap(err, "failed to map intent to Voyage input_type")
        }
        it := InputType(mapped)
        return &it, nil
    }
    return defaultInputType, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| VoyageAI text-only (`/v1/embeddings`) | VoyageAI multimodal (`/v1/multimodalembeddings`) added | voyage-multimodal-3 model launch | Separate endpoint, different request format |
| voyage-multimodal-3 (image only, 1024d fixed) | voyage-multimodal-3.5 (image+video, 256-2048d) | 2024-2025 | Added video support, flexible dimensions |
| Text API `output_dimension` | Multimodal API dimension support unclear in docs | Current | Python SDK passes it; REST docs omit it |

**Deprecated/outdated:**
- `voyage-2` is the current text-only default model in the codebase, but multimodal instances should default to `voyage-multimodal-3.5` per D-10.

## Open Questions

1. **Does the multimodal REST API accept `output_dimension`?**
   - What we know: The Python SDK passes it. The model page lists 256/512/1024/2048 as supported. The REST API reference page does not list it as a parameter.
   - What's unclear: Whether the REST API silently ignores it, accepts it, or returns an error.
   - Recommendation: Include it in the request body (following Python SDK). Advertise `RequestOptionDimension` in capabilities. Integration tests should verify dimension output changes. If the API rejects it, remove from capabilities and request in a follow-up.

2. **Multimodal endpoint URL derivation with custom BaseAPI**
   - What we know: `VoyageAIClient.BaseAPI` defaults to `https://api.voyageai.com/v1/embeddings`. The multimodal endpoint is at `/v1/multimodalembeddings`.
   - What's unclear: Best approach when user overrides BaseAPI.
   - Recommendation: Derive multimodal URL by replacing the path segment. If BaseAPI ends with `/v1/embeddings`, replace with `/v1/multimodalembeddings`. Otherwise, construct from the URL's host. Store as a computed field, not a new option.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | go test + testify |
| Config file | Makefile (`make test-ef` with `ef` build tag) |
| Quick run command | `go test -tags=ef -run TestVoyage -count=1 ./pkg/embeddings/voyage/...` |
| Full suite command | `make test-ef` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| VOY-01 | Capabilities returns text/image/video modalities | unit | `go test -tags=ef -run TestVoyageCapabilities -count=1 ./pkg/embeddings/voyage/...` | Wave 0 |
| VOY-01 | EmbedContent validates and delegates to CreateMultimodalEmbedding | unit | `go test -tags=ef -run TestVoyageEmbedContent -count=1 ./pkg/embeddings/voyage/...` | Wave 0 |
| VOY-01 | Compile-time interface assertions | unit | `go build ./pkg/embeddings/voyage/...` | Wave 0 |
| VOY-02 | MapIntent maps retrieval_query/retrieval_document correctly | unit | `go test -tags=ef -run TestVoyageMapIntent -count=1 ./pkg/embeddings/voyage/...` | Wave 0 |
| VOY-02 | MapIntent rejects classification/clustering/semantic_similarity | unit | `go test -tags=ef -run TestVoyageMapIntentRejects -count=1 ./pkg/embeddings/voyage/...` | Wave 0 |
| VOY-02 | ProviderHints input_type escape hatch works | unit | `go test -tags=ef -run TestVoyageResolveInputType -count=1 ./pkg/embeddings/voyage/...` | Wave 0 |
| VOY-03 | RegisterContent("voyageai") registered in init | unit | `go test -tags=ef -run TestVoyageContentRegistration -count=1 ./pkg/embeddings/voyage/...` | Wave 0 |
| VOY-03 | Config round-trip: GetConfig -> FromConfig -> GetConfig | unit | `go test -tags=ef -run TestVoyageConfigRoundTrip -count=1 ./pkg/embeddings/voyage/...` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -tags=ef -run TestVoyage -count=1 ./pkg/embeddings/voyage/...`
- **Per wave merge:** `make test-ef && make lint`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `pkg/embeddings/voyage/voyage_content_test.go` -- covers VOY-01, VOY-02, VOY-03 (unit tests)
- [ ] No framework install needed -- existing test infrastructure sufficient

## Sources

### Primary (HIGH confidence)
- [VoyageAI Multimodal Embeddings API Reference](https://docs.voyageai.com/reference/multimodal-embeddings-api) -- endpoint, request/response format, input types, constraints
- [VoyageAI Multimodal Embeddings Overview](https://docs.voyageai.com/docs/multimodal-embeddings) -- models, modalities, dimensions, truncation
- [VoyageAI Text Embeddings API](https://docs.voyageai.com/docs/embeddings) -- output_dimension support for text models
- [Chroma voyageai.json schema](https://github.com/chroma-core/chroma/blob/main/schemas/embedding_functions/voyageai.json) -- config field constraints
- Gemini Phase 6 implementation (`pkg/embeddings/gemini/content.go`, `gemini.go`) -- reference pattern for ContentEmbeddingFunction
- Existing Voyage provider (`pkg/embeddings/voyage/voyage.go`) -- current implementation to extend

### Secondary (MEDIUM confidence)
- [VoyageAI Python SDK client.py](https://github.com/voyage-ai/voyageai-python/blob/main/voyageai/client.py) -- multimodal_embed method signature, output_dimension passthrough

### Tertiary (LOW confidence)
- output_dimension support on multimodal REST endpoint -- Python SDK passes it but REST docs don't list it; needs runtime verification

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies, all existing
- Architecture: HIGH -- directly follows Phase 6 Gemini pattern with clear 1:1 mapping
- Pitfalls: HIGH -- documented from API docs and cross-referencing with Python SDK
- output_dimension support: LOW -- conflicting signals between REST docs and Python SDK

**Research date:** 2026-03-22
**Valid until:** 2026-04-22 (stable -- VoyageAI API versioned, unlikely to change)
