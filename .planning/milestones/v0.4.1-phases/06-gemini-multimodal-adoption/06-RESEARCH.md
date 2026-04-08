# Phase 6: Gemini Multimodal Adoption - Research

**Researched:** 2026-03-20
**Domain:** Go embedding provider adoption — Google Gemini multimodal contract
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Model selection**
- D-01: Default model changes from `gemini-embedding-001` to `gemini-embedding-2-preview` for new instances
- D-02: `gemini-embedding-2-preview` supports text, image, audio, video, and PDF; `gemini-embedding-001` is text-only
- D-03: If user explicitly selects a legacy model and sends multimodal content, fail with explicit error
- D-04: Add a negative test case demonstrating the failure mode when legacy model receives multimodal content

**Content part handling**
- D-05: Non-text parts use inline blobs via `genai.NewPartFromBytes(data, mimeType)` — no URI references for the embedding endpoint
- D-06: MIME types resolved from `BinarySource.MIMEType` field first; file-backed sources infer from extension as fallback; fail explicitly if MIME type is empty and can't be inferred
- D-07: Add MIME-modality consistency validation before sending (e.g., image modality must have image/* MIME prefix) as a security pre-flight check
- D-08: Mixed-part Content items are supported natively — multiple Parts in one Content produce one aggregated embedding. Advertise `SupportsMixedPart: true` in capabilities
- D-09: Provider-side byte resolution: Gemini's `EmbedContent` impl resolves all `BinarySource` kinds (file → `os.ReadFile`, base64 → decode, bytes → pass through, URL → HTTP fetch client-side)

**Gemini API modality limits (from docs)**
- D-10: Images: PNG/JPEG only, max 6 per request
- D-11: Audio: MP3/WAV, max 80 seconds
- D-12: Video: MP4/MOV, max 120 seconds
- D-13: PDF: max 6 pages

**Intent mapping**
- D-14: Implement `IntentMapper` interface with direct 1:1 mapping of 5 neutral intents to Gemini task types
- D-15: Gemini-only task types (CODE_RETRIEVAL_QUERY, QUESTION_ANSWERING, FACT_VERIFICATION) accessed via `ProviderHints["task_type"]` escape hatch only
- D-16: `MapIntent` checks `ProviderHints["task_type"]` first (override), then maps neutral intent, then passes empty string for no-intent
- D-17: Declare supported intents in `CapabilityMetadata.Intents` for pre-check validation

**Backward compatibility**
- D-18: Shared helpers, separate entry points — `CreateEmbedding` stays unchanged, new `CreateContentEmbedding` added
- D-19: Both paths share config/model/taskType/dimension resolution and response parsing helpers
- D-20: Both paths converge at `client.Models.EmbedContent()` SDK call but construct `genai.Content` differently
- D-21: Existing `EmbedDocuments`/`EmbedQuery` delegate through unchanged `CreateEmbedding`
- D-22: New `EmbedContent`/`EmbedContents` delegate through `CreateContentEmbedding`

**Registry and config**
- D-23: Dual registration: keep existing `RegisterDense("google_genai")` and add `RegisterContent("google_genai")`
- D-24: Content factory builds from the same config schema — no new config fields
- D-25: Allowed config fields per upstream schema: `model_name`, `task_type`, `dimension`, `api_key_env_var`, `vertexai`, `project`, `location`
- D-26: Capabilities are runtime-derived from model name, not persisted in config
- D-27: Config round-trip shape stays unchanged: `{type: "known", name: "google_genai", config: {...}}`

### Claude's Discretion
- Internal helper function names and organization (e.g., `resolveBytes`, `resolveMIME`, `convertToGenaiContent`)
- Exact placement of `ValidateContentSupport` call within `EmbedContent`/`EmbedContents`
- Test scaffolding structure and assertion patterns
- Error message wording for unsupported modality/model combinations
- Whether to add Vertex AI support in this phase or defer

### Deferred Ideas (OUT OF SCOPE)
- Vertex AI backend support (vertexai, project, location config fields)
- Gemini Files API integration for large media
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GEM-01 | Gemini implements `SharedContentEmbeddingFunction` and `CapabilityAware` for text, image, audio, video, and PDF modalities | genai SDK provides `NewPartFromBytes(data, mimeType)` + `NewPartFromText(text)` for all five modalities; `EmbedContent` API accepts `[]*genai.Content` with mixed parts |
| GEM-02 | Neutral intents map to Gemini task types with explicit errors for unsupported combinations | 5 neutral intents map 1:1 to the 5 shared Gemini task types; `IntentMapper` interface is already defined in `embedding.go`; `ValidateContentSupport` pre-checks declared intents |
| GEM-03 | Gemini is registered in the multimodal factory/registry path with config round-trip support | `RegisterContent` is available in `registry.go`; existing `NewGeminiEmbeddingFunctionFromConfig` pattern provides the config round-trip template |
</phase_requirements>

## Summary

Phase 6 wires the existing Gemini provider into the shared multimodal contract introduced in Phases 1-5. The foundation is complete: `ContentEmbeddingFunction`, `CapabilityAware`, `IntentMapper` interfaces are defined, `ValidateContentSupport` is implemented, and `RegisterContent` + `BuildContent` fallback chain are in the registry. Roboflow demonstrates the delegation pattern via adapter; Gemini will be the first provider to implement the interfaces directly (natively).

The work decomposes into three areas: (1) capability declaration and intent mapping, (2) content-to-genai conversion with byte resolution for all four source kinds, and (3) registration and config round-trip. All three areas have clear prior art in the codebase. The critical new logic is `convertToGenaiContent` — the helper that maps `embeddings.Content` (with its Parts, each potentially backed by a `BinarySource`) into `[]*genai.Content` using `genai.NewPartFromBytes` for non-text parts.

The `google.golang.org/genai v1.45.0` SDK already supports the full `EmbedContent` API for multimodal content. `genai.Models.EmbedContent(ctx, model, contents, config)` accepts `[]*genai.Content` where each Content may have multiple Parts mixing text and inline blob data. This is exactly what the phase needs.

**Primary recommendation:** Implement `GeminiEmbeddingFunction` as a native `ContentEmbeddingFunction` + `CapabilityAware` + `IntentMapper` by adding `CreateContentEmbedding` to `Client`, implementing the four interface methods on `GeminiEmbeddingFunction`, and adding `RegisterContent("google_genai", ...)` to the existing `init()` block — mirroring the Roboflow pattern without the adapter indirection.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `google.golang.org/genai` | v1.45.0 (project-pinned) | Gemini API SDK — `Models.EmbedContent`, `NewPartFromBytes`, `NewPartFromText`, `NewContentFromParts` | Already imported; `EmbedContent` API supports multimodal content natively |
| `github.com/pkg/errors` | (project-pinned) | Error wrapping with stack traces | Used throughout; consistent with codebase style |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `encoding/base64` | stdlib | Decode base64-backed `BinarySource` | In `resolveBytes` for `SourceKindBase64` |
| `os` | stdlib | Read file-backed `BinarySource` | In `resolveBytes` for `SourceKindFile` |
| `net/http` | stdlib | Fetch URL-backed `BinarySource` | In `resolveBytes` for `SourceKindURL` |
| `path/filepath` | stdlib | Infer MIME type from extension when `BinarySource.MIMEType` is empty | In `resolveMIME` fallback |

**Installation:** No new dependencies needed — all are already imported or are stdlib.

## Architecture Patterns

### Recommended File Layout

```
pkg/embeddings/gemini/
├── gemini.go            # Client.CreateContentEmbedding + interface impls on GeminiEmbeddingFunction
├── task_type.go         # Existing TaskType consts — no changes needed
├── option.go            # Existing functional options — no changes needed
├── content.go           # NEW: convertToGenaiContent, resolveBytes, resolveMIME, mimeModalityCheck, capabilitiesForModel
├── gemini_config_test.go  # Existing unit tests — extend with new tests
└── gemini_content_test.go # NEW: unit tests for content path (request construction, intent mapping, negative cases)
```

### Pattern 1: Client Method Separation (D-18, D-20)

`Client` gets a `CreateContentEmbedding` method that handles multimodal `Content` slices. Both `CreateEmbedding` and `CreateContentEmbedding` converge at `c.Client.Models.EmbedContent()` but prepare `[]*genai.Content` differently.

```go
// Source: existing gemini.go pattern + genai SDK
func (c *Client) CreateContentEmbedding(ctx context.Context, contents []embeddings.Content) ([]embeddings.Embedding, error) {
    model, err := modelFromContext(ctx, string(c.DefaultModel))
    // ... shared resolution helpers ...
    taskType, err := taskTypeFromContext(ctx, c.DefaultTaskType)
    outputDimensionality, err := outputDimensionalityFromContext(ctx, c.DefaultDimension)

    genaiContents, err := convertToGenaiContents(contents, taskType) // new helper
    if err != nil {
        return nil, errors.Wrap(err, "failed to convert contents")
    }

    res, err := c.Client.Models.EmbedContent(ctx, model, genaiContents, buildEmbedContentConfig(taskType, outputDimensionality))
    // ... parse response same as CreateEmbedding ...
}
```

### Pattern 2: Interface Implementation on GeminiEmbeddingFunction (GEM-01, GEM-02)

```go
// Source: roboflow.go interface assertion pattern
var _ embeddings.ContentEmbeddingFunction = (*GeminiEmbeddingFunction)(nil)
var _ embeddings.CapabilityAware          = (*GeminiEmbeddingFunction)(nil)
var _ embeddings.IntentMapper             = (*GeminiEmbeddingFunction)(nil)

func (e *GeminiEmbeddingFunction) EmbedContent(ctx context.Context, content embeddings.Content) (embeddings.Embedding, error) {
    caps := e.Capabilities()
    if err := embeddings.ValidateContentSupport(content, caps); err != nil {
        return nil, err
    }
    result, err := e.apiClient.CreateContentEmbedding(ctx, []embeddings.Content{content})
    if err != nil {
        return nil, err
    }
    if len(result) == 0 {
        return nil, errors.New("no embedding returned")
    }
    return result[0], nil
}

func (e *GeminiEmbeddingFunction) EmbedContents(ctx context.Context, contents []embeddings.Content) ([]embeddings.Embedding, error) {
    caps := e.Capabilities()
    if err := embeddings.ValidateContentsSupport(contents, caps); err != nil {
        return nil, err
    }
    return e.apiClient.CreateContentEmbedding(ctx, contents)
}
```

### Pattern 3: Capability Derivation from Model Name (D-26)

```go
// Source: CONTEXT.md D-02, D-26
func capabilitiesForModel(model string) embeddings.CapabilityMetadata {
    switch model {
    case "gemini-embedding-2-preview":
        return embeddings.CapabilityMetadata{
            Modalities:    []embeddings.Modality{
                embeddings.ModalityText, embeddings.ModalityImage,
                embeddings.ModalityAudio, embeddings.ModalityVideo, embeddings.ModalityPDF,
            },
            Intents: []embeddings.Intent{
                embeddings.IntentRetrievalQuery, embeddings.IntentRetrievalDocument,
                embeddings.IntentClassification, embeddings.IntentClustering,
                embeddings.IntentSemanticSimilarity,
            },
            RequestOptions: []embeddings.RequestOption{
                embeddings.RequestOptionDimension,
                embeddings.RequestOptionProviderHints,
            },
            SupportsBatch:     true,
            SupportsMixedPart: true,
        }
    default: // gemini-embedding-001 and any unknown models
        return embeddings.CapabilityMetadata{
            Modalities:    []embeddings.Modality{embeddings.ModalityText},
            Intents: []embeddings.Intent{
                embeddings.IntentRetrievalQuery, embeddings.IntentRetrievalDocument,
                embeddings.IntentClassification, embeddings.IntentClustering,
                embeddings.IntentSemanticSimilarity,
            },
            RequestOptions: []embeddings.RequestOption{embeddings.RequestOptionDimension},
            SupportsBatch:  true,
            SupportsMixedPart: false,
        }
    }
}

func (e *GeminiEmbeddingFunction) Capabilities() embeddings.CapabilityMetadata {
    return capabilitiesForModel(string(e.apiClient.DefaultModel))
}
```

### Pattern 4: Intent Mapping (D-14, D-15, D-16)

```go
// Source: CONTEXT.md D-14 through D-17
var neutralIntentToTaskType = map[embeddings.Intent]TaskType{
    embeddings.IntentRetrievalQuery:     TaskTypeRetrievalQuery,
    embeddings.IntentRetrievalDocument:  TaskTypeRetrievalDocument,
    embeddings.IntentClassification:     TaskTypeClassification,
    embeddings.IntentClustering:         TaskTypeClustering,
    embeddings.IntentSemanticSimilarity: TaskTypeSemanticSimilarity,
}

func (e *GeminiEmbeddingFunction) MapIntent(intent embeddings.Intent) (string, error) {
    // D-16: Check ProviderHints override is done at call site, not here
    if !embeddings.IsNeutralIntent(intent) {
        return "", errors.Errorf("unsupported intent %q: use ProviderHints[\"task_type\"] for Gemini-native task types", intent)
    }
    tt, ok := neutralIntentToTaskType[intent]
    if !ok {
        return "", errors.Errorf("no Gemini task type mapping for intent %q", intent)
    }
    return string(tt), nil
}
```

### Pattern 5: BinarySource Resolution (D-09)

```go
// Source: CONTEXT.md D-05, D-06, D-09
func resolveBytes(ctx context.Context, source *embeddings.BinarySource) ([]byte, error) {
    switch source.Kind {
    case embeddings.SourceKindBytes:
        return source.Bytes, nil
    case embeddings.SourceKindBase64:
        data, err := base64.StdEncoding.DecodeString(source.Base64)
        if err != nil {
            return nil, errors.Wrap(err, "failed to decode base64 source")
        }
        return data, nil
    case embeddings.SourceKindFile:
        data, err := os.ReadFile(source.FilePath)
        if err != nil {
            return nil, errors.Wrap(err, "failed to read file source")
        }
        return data, nil
    case embeddings.SourceKindURL:
        // client-side HTTP fetch
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
        if err != nil {
            return nil, errors.Wrap(err, "failed to create URL fetch request")
        }
        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            return nil, errors.Wrap(err, "failed to fetch URL source")
        }
        defer resp.Body.Close()
        data, err := io.ReadAll(resp.Body)
        if err != nil {
            return nil, errors.Wrap(err, "failed to read URL response body")
        }
        return data, nil
    default:
        return nil, errors.Errorf("unsupported source kind %q", source.Kind)
    }
}
```

### Pattern 6: MIME Resolution and Modality Consistency (D-06, D-07)

```go
// Source: CONTEXT.md D-06, D-07
var extToMIME = map[string]string{
    ".png":  "image/png",
    ".jpg":  "image/jpeg",
    ".jpeg": "image/jpeg",
    ".mp3":  "audio/mpeg",
    ".wav":  "audio/wav",
    ".mp4":  "video/mp4",
    ".mov":  "video/mov",
    ".pdf":  "application/pdf",
}

func resolveMIME(source *embeddings.BinarySource) (string, error) {
    if source.MIMEType != "" {
        return source.MIMEType, nil
    }
    if source.FilePath != "" {
        ext := strings.ToLower(filepath.Ext(source.FilePath))
        if mime, ok := extToMIME[ext]; ok {
            return mime, nil
        }
    }
    return "", errors.New("MIME type is required: set BinarySource.MIMEType or use a file with a known extension")
}

// validateMIMEModality checks that a MIME type is consistent with the declared modality (D-07)
func validateMIMEModality(modality embeddings.Modality, mimeType string) error {
    switch modality {
    case embeddings.ModalityImage:
        if !strings.HasPrefix(mimeType, "image/") {
            return errors.Errorf("image modality requires image/* MIME type, got %q", mimeType)
        }
    case embeddings.ModalityAudio:
        if !strings.HasPrefix(mimeType, "audio/") {
            return errors.Errorf("audio modality requires audio/* MIME type, got %q", mimeType)
        }
    case embeddings.ModalityVideo:
        if !strings.HasPrefix(mimeType, "video/") {
            return errors.Errorf("video modality requires video/* MIME type, got %q", mimeType)
        }
    case embeddings.ModalityPDF:
        if mimeType != "application/pdf" {
            return errors.Errorf("pdf modality requires application/pdf MIME type, got %q", mimeType)
        }
    }
    return nil
}
```

### Pattern 7: Dual Registration in init() (D-23)

```go
// Source: existing gemini.go init() block + registry.go RegisterContent
func init() {
    if err := embeddings.RegisterDense("google_genai", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.EmbeddingFunction, error) {
        return NewGeminiEmbeddingFunctionFromConfig(cfg)
    }); err != nil {
        panic(err)
    }

    if err := embeddings.RegisterContent("google_genai", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.ContentEmbeddingFunction, error) {
        return NewGeminiEmbeddingFunctionFromConfig(cfg)
    }); err != nil {
        panic(err)
    }
}
```

Note: `panic` in `init()` is the established pattern for registration errors across the codebase (consistent with Roboflow, ConsistentHash, etc.) — this is acceptable since it only fires if the registry has a duplicate name, which is a programming error.

### Pattern 8: ProviderHints Escape Hatch in CreateContentEmbedding (D-15, D-16)

The `CreateContentEmbedding` helper must check `content.ProviderHints["task_type"]` to allow caller override with Gemini-native task types (CODE_RETRIEVAL_QUERY, etc.) that are not in the neutral intent set. This override happens per-content, not per-batch.

```go
func resolveTaskTypeForContent(content embeddings.Content, defaultTaskType TaskType) (TaskType, error) {
    if hint, ok := content.ProviderHints["task_type"]; ok {
        if hintStr, ok := hint.(string); ok && hintStr != "" {
            tt := TaskType(hintStr)
            if !tt.IsValid() {
                return "", errors.Errorf("invalid task_type hint %q", hintStr)
            }
            return tt, nil
        }
    }
    // Use intent if set (already mapped by MapIntent at the EmbedContent level)
    // Fall through to default
    return defaultTaskType, nil
}
```

### Anti-Patterns to Avoid

- **Adapter delegation for Gemini:** Unlike Roboflow, Gemini should NOT delegate `EmbedContent`/`EmbedContents` through `AdaptMultimodalEmbeddingFunctionToContent`. Roboflow uses the adapter because it already implements the legacy image-only interface. Gemini bypasses the adapter and calls `CreateContentEmbedding` directly.
- **Persisting capabilities in config:** Capabilities are runtime-derived from model name (D-26). Do not add a `capabilities` field to `EmbeddingFunctionConfig` or `GetConfig()`.
- **Fetching URLs server-side:** The Gemini embedding API does not accept URI references for the `EmbedContent` endpoint. All binary content must be inline bytes (D-05).
- **Applying intent mapping inside `MapIntent` from ProviderHints:** The `MapIntent` method maps neutral intents only. The ProviderHints escape hatch is checked separately in the content path, not inside `MapIntent`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Structural content validation | Custom validation logic | `embeddings.ValidateContentSupport(content, caps)` | Already handles modality + intent + dimension checks against declared capabilities |
| Batch content validation | Loop over contents manually | `embeddings.ValidateContentsSupport(contents, caps)` | Consistent fail-on-first semantics established in Phase 4 |
| genai Content construction from text | Manual struct initialization | `genai.NewPartFromText(text)`, `genai.NewContentFromParts(parts, role)` | SDK constructors from `google.golang.org/genai v1.45.0` |
| genai Content construction from bytes | Manual Blob struct | `genai.NewPartFromBytes(data, mimeType)` | Correct field paths handled by SDK constructor |
| Intent neutrality check | String comparison set | `embeddings.IsNeutralIntent(intent)` | Defined in `multimodal.go`, covers all 5 cases |
| Response parsing | Custom response struct | Existing `res.Embeddings[i].Values` loop from `CreateEmbedding` | Already handles nil/empty cases |
| Config integer extraction | Type switch on interface{} | `embeddings.ConfigInt(cfg, key)` | Handles int, float64, int64 JSON unmarshaling variants |
| Embedding slice construction | Manual float32 copy | `embeddings.NewEmbeddingsFromFloat32(embs)` | Consistent with rest of codebase |

**Key insight:** Phases 1-5 built the entire validation, capability, and registry infrastructure so Phase 6 only needs to implement the provider-specific conversion logic. The plumbing is already there.

## Common Pitfalls

### Pitfall 1: Gemini embedding-001 with multimodal content passes `ValidateContentSupport` silently

**What goes wrong:** `ValidateContentSupport` only rejects unsupported modalities when `caps.Modalities` is non-empty. If `capabilitiesForModel("gemini-embedding-001")` is declared with `Modalities: []Modality{ModalityText}`, then sending an image Part will be correctly rejected. But if the model name was left as `DefaultEmbeddingModel = "gemini-embedding-001"` after changing the default to `gemini-embedding-2-preview`, the declared capability and actual model could diverge.

**Why it happens:** The capability table is derived from the model name at runtime. If the model name constant is changed correctly (D-01), the capabilities follow automatically.

**How to avoid:** `capabilitiesForModel` must check for the exact string `"gemini-embedding-2-preview"` — all other values fall back to text-only capabilities (D-26). Update `DefaultEmbeddingModel` constant to `"gemini-embedding-2-preview"` (D-01).

**Warning signs:** Test for negative case passes when it should fail; multimodal content reaches the API with `gemini-embedding-001` and returns a 400.

### Pitfall 2: `MapIntent` called with ProviderHints custom task type

**What goes wrong:** If the caller sets `content.ProviderHints["task_type"] = "CODE_RETRIEVAL_QUERY"` and the code path also calls `e.MapIntent(content.Intent)`, the `MapIntent` call may fail if `content.Intent` is empty or a custom string.

**Why it happens:** D-16 specifies that ProviderHints override is checked first, before intent mapping. The check must short-circuit and skip `MapIntent` entirely.

**How to avoid:** In `CreateContentEmbedding`, apply this order: (1) check `ProviderHints["task_type"]`, (2) if no hint and intent is set, call `MapIntent`, (3) if no intent, use default task type.

**Warning signs:** Error "unsupported intent" for a request that only sets `ProviderHints`, not `Intent`.

### Pitfall 3: Empty MIME type fails silently at the API

**What goes wrong:** `genai.NewPartFromBytes(data, "")` creates a valid Go struct. The Gemini API will return a 400 or misidentify the content type. The error message may be cryptic.

**Why it happens:** Go does not enforce non-empty strings in function arguments.

**How to avoid:** `resolveMIME` must return an explicit error if it cannot resolve a MIME type from `BinarySource.MIMEType` or file extension (D-06). Call `resolveMIME` before constructing the genai Part.

**Warning signs:** API error 400 with "invalid mime type" or "unsupported content type" message.

### Pitfall 4: URL source kind produces large binary data in memory for every request

**What goes wrong:** Client-side URL fetching (D-09) downloads the full media file into memory before sending it to the API as inline bytes. For large videos or audio files, this can exhaust memory.

**Why it happens:** The Gemini `EmbedContent` endpoint only accepts inline data; the Files API is deferred (CONTEXT.md deferred section).

**How to avoid:** Document the memory implication in code comments. No mitigation in this phase — the deferred Files API path is the proper solution.

**Warning signs:** OOM in production with large video/audio URLs.

### Pitfall 5: Batch with mixed per-content intents

**What goes wrong:** When `EmbedContents` processes a batch, each `Content` may have a different `ProviderHints["task_type"]`. The Gemini SDK's `EmbedContent` accepts a single config applied to all contents. If intents differ per content, the per-content task type override cannot be expressed in a single API call.

**Why it happens:** The Gemini batch embedding endpoint applies one `EmbedContentConfig` to all content items.

**How to avoid:** For the batch path, resolve the task type from the first content item (or the default), and validate that all items agree. Alternatively, fall back to one-at-a-time calling if task types differ. This is Claude's discretion — research finding to surface for the planner. The simplest approach (and consistent with existing `CreateEmbedding` behavior) is to use the client-level default task type and ignore per-content hints in batch calls, documenting this limitation.

**Warning signs:** First item's task type silently applied to all; caller confused why subsequent items don't reflect their ProviderHints.

### Pitfall 6: `RegisterContent` called after `RegisterDense` in init() — duplicate name panic

**What goes wrong:** `RegisterContent` uses a separate map from `RegisterDense`, so the same name "google_genai" can be registered in both. There is no conflict. However, calling `RegisterContent` a second time (e.g., from a test `init()` that imports the package) will panic.

**Why it happens:** Registry panics on duplicate registration. Test files that import the package will trigger the real `init()`.

**How to avoid:** Tests should not call `RegisterContent("google_genai", ...)` themselves. Use `BuildContent("google_genai", cfg)` to test the round-trip, which relies on the `init()` registration.

**Warning signs:** Test panic "content embedding function "google_genai" already registered".

## Code Examples

### genai SDK: Full multimodal content construction

```go
// Source: google.golang.org/genai v1.45.0 types.go
// Text part
textPart := genai.NewPartFromText("describe this image")

// Inline binary part (image, audio, video, PDF)
imagePart := genai.NewPartFromBytes(imageBytes, "image/jpeg")

// Mixed-part Content (produces one aggregated embedding)
content := genai.NewContentFromParts([]*genai.Part{textPart, imagePart}, genai.RoleUser)

// Submit to API
res, err := client.Models.EmbedContent(ctx, "gemini-embedding-2-preview", []*genai.Content{content}, config)
```

### ValidateContentSupport usage (from multimodal_validate.go)

```go
// Source: pkg/embeddings/multimodal_validate.go
caps := e.Capabilities() // derived from model name
if err := embeddings.ValidateContentSupport(content, caps); err != nil {
    return nil, err
}
// or for batch:
if err := embeddings.ValidateContentsSupport(contents, caps); err != nil {
    return nil, err
}
```

### RegisterContent factory (from registry.go)

```go
// Source: pkg/embeddings/registry.go
embeddings.RegisterContent("google_genai", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.ContentEmbeddingFunction, error) {
    return NewGeminiEmbeddingFunctionFromConfig(cfg)
})
```

### IntentMapper interface (from embedding.go)

```go
// Source: pkg/embeddings/embedding.go
// IntentMapper is implemented by providers that translate neutral intents to provider-native strings.
type IntentMapper interface {
    MapIntent(intent Intent) (string, error)
}
// Called by callers via type assertion:
if mapper, ok := ef.(embeddings.IntentMapper); ok {
    taskTypeStr, err := mapper.MapIntent(content.Intent)
}
```

### Compile-time interface assertions (Roboflow pattern)

```go
// Source: pkg/embeddings/roboflow/roboflow.go
var _ embeddings.ContentEmbeddingFunction = (*GeminiEmbeddingFunction)(nil)
var _ embeddings.CapabilityAware          = (*GeminiEmbeddingFunction)(nil)
var _ embeddings.IntentMapper             = (*GeminiEmbeddingFunction)(nil)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `gemini-embedding-001` default | `gemini-embedding-2-preview` default | Phase 6 (D-01) | New instances are multimodal by default; existing configs with explicit model name are unaffected |
| Text-only `CreateEmbedding` path | `CreateEmbedding` + `CreateContentEmbedding` coexisting | Phase 6 (D-18) | No breaking change; multimodal path is additive |
| Roboflow as only content-native provider | Gemini added as second native implementor | Phase 6 | Establishes the model-tier capability derivation pattern for future providers |

**Deprecated/outdated:**
- `DefaultEmbeddingModel = "gemini-embedding-001"` constant: renamed/updated to `"gemini-embedding-2-preview"` in D-01. Old string should remain accessible (e.g., as `LegacyEmbeddingModel`) for tests that verify the negative path (D-04).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | `testing` stdlib + `github.com/stretchr/testify` v1.x |
| Config file | None (build tags control inclusion) |
| Quick run command | `go test ./pkg/embeddings/gemini/... ` (no build tag = unit tests only) |
| Full suite command | `go test -tags=ef ./pkg/embeddings/gemini/...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GEM-01 | `GeminiEmbeddingFunction` satisfies `ContentEmbeddingFunction` + `CapabilityAware` at compile time | unit (compile assertion) | `go build ./pkg/embeddings/gemini/...` | ❌ Wave 0 |
| GEM-01 | `Capabilities()` returns correct modalities for `gemini-embedding-2-preview` | unit | `go test ./pkg/embeddings/gemini/... -run TestCapabilitiesForModel` | ❌ Wave 0 |
| GEM-01 | `Capabilities()` returns text-only for `gemini-embedding-001` | unit | `go test ./pkg/embeddings/gemini/... -run TestCapabilitiesForModel` | ❌ Wave 0 |
| GEM-01 | `convertToGenaiContent` produces correct `genai.Content` for text Part | unit | `go test ./pkg/embeddings/gemini/... -run TestConvertToGenaiContent` | ❌ Wave 0 |
| GEM-01 | `convertToGenaiContent` produces correct `genai.Content` for image/audio/video/PDF parts | unit | `go test ./pkg/embeddings/gemini/... -run TestConvertToGenaiContent` | ❌ Wave 0 |
| GEM-01 | `EmbedContent` with legacy model + non-text part returns `ValidationError` | unit | `go test ./pkg/embeddings/gemini/... -run TestEmbedContent_LegacyModelRejectsMultimodal` | ❌ Wave 0 |
| GEM-02 | `MapIntent` maps all 5 neutral intents to correct Gemini task type strings | unit | `go test ./pkg/embeddings/gemini/... -run TestMapIntent` | ❌ Wave 0 |
| GEM-02 | `MapIntent` returns error for non-neutral intents | unit | `go test ./pkg/embeddings/gemini/... -run TestMapIntent` | ❌ Wave 0 |
| GEM-02 | ProviderHints `task_type` override passes through to request config | unit | `go test ./pkg/embeddings/gemini/... -run TestProviderHintsTaskTypeOverride` | ❌ Wave 0 |
| GEM-03 | `BuildContent("google_genai", cfg)` constructs a `ContentEmbeddingFunction` | unit | `go test ./pkg/embeddings/gemini/... -run TestGeminiContentRegistration` | ❌ Wave 0 |
| GEM-03 | Config round-trip: `Name()` + `GetConfig()` → `BuildContent()` produces equivalent instance | unit | `go test ./pkg/embeddings/gemini/... -run TestGeminiContentConfigRoundTrip` | ❌ Wave 0 |
| GEM-01 | Live `EmbedContent` with image produces non-empty embedding (ef build tag) | integration | `go test -tags=ef ./pkg/embeddings/gemini/... -run TestGemini_EmbedContent_Image` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./pkg/embeddings/gemini/...` (unit tests, no API key needed)
- **Per wave merge:** `go test ./pkg/embeddings/gemini/... ./pkg/embeddings/...` (all unit tests)
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `pkg/embeddings/gemini/gemini_content_test.go` — covers GEM-01 (request construction, capability derivation, negative model test), GEM-02 (intent mapping), GEM-03 (registration, config round-trip)
- [ ] `pkg/embeddings/gemini/content.go` — new file for conversion helpers

*(Existing `gemini_config_test.go` has no build tag — new unit tests should follow the same pattern.)*

## Open Questions

1. **Per-content task type override in batch calls (Pitfall 5)**
   - What we know: The Gemini SDK applies one `EmbedContentConfig` to all contents in the batch call. Per-content `ProviderHints["task_type"]` cannot be expressed in a single API call.
   - What's unclear: Should `EmbedContents` (a) ignore per-content hints and use the default, (b) error if any content has a hint that differs from the first, or (c) fall back to serial one-at-a-time calls?
   - Recommendation: Use option (a) — use client-level default task type for batch calls, document the limitation in code comments. The simple approach aligns with the project's "radically simple" principle.

2. **`io.ReadAll` for URL fetching — no size limit**
   - What we know: D-09 specifies client-side URL fetch for `SourceKindURL`. No size limit is defined.
   - What's unclear: Should there be a max-bytes safeguard (e.g., 100MB) to prevent OOM for very large URLs?
   - Recommendation: Add a `io.LimitReader` with a generous limit (e.g., 200MB) and document it. Consistent with `MaxImageFileSize` pattern in `embedding.go`.

## Sources

### Primary (HIGH confidence)
- `pkg/embeddings/gemini/gemini.go` — Existing Client structure, patterns, and helpers (read directly)
- `pkg/embeddings/embedding.go` — `ContentEmbeddingFunction`, `CapabilityAware`, `IntentMapper` interface definitions (read directly)
- `pkg/embeddings/capabilities.go` — `CapabilityMetadata` struct (read directly)
- `pkg/embeddings/multimodal.go` — `Content`, `Part`, `BinarySource`, `Intent`, `Modality` types (read directly)
- `pkg/embeddings/multimodal_validate.go` — `ValidateContentSupport`, `ValidationError` (read directly)
- `pkg/embeddings/multimodal_compat.go` — `AdaptMultimodalEmbeddingFunctionToContent`, helper constructors (read directly)
- `pkg/embeddings/registry.go` — `RegisterContent`, `BuildContent`, `inferCaps` (read directly)
- `pkg/embeddings/roboflow/roboflow.go` — Reference `ContentEmbeddingFunction` implementation (read directly)
- `google.golang.org/genai v1.45.0/types.go` — `NewPartFromBytes`, `NewPartFromText`, `NewContentFromParts`, `EmbedContentConfig`, `EmbedContentResponse` (read from module cache)
- `google.golang.org/genai v1.45.0/models.go` — `Models.EmbedContent` signature (read from module cache)
- https://ai.google.dev/gemini-api/docs/embeddings — Modality limits, MIME types, task types (fetched)

### Secondary (MEDIUM confidence)
- `.planning/phases/06-gemini-multimodal-adoption/06-CONTEXT.md` — All implementation decisions (D-01 through D-27) confirmed by code inspection

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — genai SDK v1.45.0 is pinned in go.mod, API verified in module cache
- Architecture: HIGH — patterns derived directly from existing code (Roboflow, existing Gemini client)
- Pitfalls: HIGH for SDK/API behavior; MEDIUM for batch task-type handling (no official docs found on per-item config)

**Research date:** 2026-03-20
**Valid until:** 2026-04-20 (genai SDK API is stable; Gemini embedding-2-preview is in preview, check for GA status)
