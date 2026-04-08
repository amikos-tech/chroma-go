# Phase 6: Gemini Multimodal Adoption - Context

**Gathered:** 2026-03-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire Gemini into the shared multimodal contract so it supports text, image, audio, video, and PDF embeddings through the portable `ContentEmbeddingFunction` and `CapabilityAware` interfaces, while keeping existing text-only `EmbedDocuments`/`EmbedQuery` behavior unchanged. Gemini becomes the first provider to natively implement the shared content interface (Roboflow delegates through the compatibility adapter).

</domain>

<decisions>
## Implementation Decisions

### Model selection
- **D-01:** Default model changes from `gemini-embedding-001` to `gemini-embedding-2-preview` for new instances
- **D-02:** `gemini-embedding-2-preview` supports text, image, audio, video, and PDF; `gemini-embedding-001` is text-only
- **D-03:** If user explicitly selects a legacy model and sends multimodal content, fail with explicit error
- **D-04:** Add a negative test case demonstrating the failure mode when legacy model receives multimodal content

### Content part handling
- **D-05:** Non-text parts use inline blobs via `genai.NewPartFromBytes(data, mimeType)` — no URI references for the embedding endpoint
- **D-06:** MIME types resolved from `BinarySource.MIMEType` field first; file-backed sources infer from extension as fallback; fail explicitly if MIME type is empty and can't be inferred
- **D-07:** Add MIME-modality consistency validation before sending (e.g., image modality must have image/* MIME prefix) as a security pre-flight check
- **D-08:** Mixed-part Content items are supported natively — multiple Parts in one Content produce one aggregated embedding. Advertise `SupportsMixedPart: true` in capabilities
- **D-09:** Provider-side byte resolution: Gemini's `EmbedContent` impl resolves all `BinarySource` kinds (file → `os.ReadFile`, base64 → decode, bytes → pass through, URL → HTTP fetch client-side)

### Gemini API modality limits (from docs)
- **D-10:** Images: PNG/JPEG only, max 6 per request
- **D-11:** Audio: MP3/WAV, max 80 seconds
- **D-12:** Video: MP4/MOV, max 120 seconds
- **D-13:** PDF: max 6 pages

### Intent mapping
- **D-14:** Implement `IntentMapper` interface with direct 1:1 mapping of 5 neutral intents to Gemini task types:
  - `retrieval_query` → `RETRIEVAL_QUERY`
  - `retrieval_document` → `RETRIEVAL_DOCUMENT`
  - `classification` → `CLASSIFICATION`
  - `clustering` → `CLUSTERING`
  - `semantic_similarity` → `SEMANTIC_SIMILARITY`
- **D-15:** Gemini-only task types (CODE_RETRIEVAL_QUERY, QUESTION_ANSWERING, FACT_VERIFICATION) are accessed via `ProviderHints["task_type"]` escape hatch, not neutral intents
- **D-16:** `MapIntent` checks `ProviderHints["task_type"]` first (override), then maps neutral intent, then passes empty string for no-intent (API uses its own default behavior)
- **D-17:** Declare supported intents in `CapabilityMetadata.Intents` for pre-check validation

### Backward compatibility
- **D-18:** Shared helpers, separate entry points — `CreateEmbedding` (text path) stays unchanged, new `CreateContentEmbedding` (multimodal path) added
- **D-19:** Both paths share config/model/taskType/dimension resolution and response parsing helpers
- **D-20:** Both paths converge at `client.Models.EmbedContent()` SDK call but construct `genai.Content` differently
- **D-21:** Existing `EmbedDocuments`/`EmbedQuery` delegate through unchanged `CreateEmbedding`
- **D-22:** New `EmbedContent`/`EmbedContents` delegate through `CreateContentEmbedding`

### Registry and config
- **D-23:** Dual registration: keep existing `RegisterDense("google_genai")` and add `RegisterContent("google_genai")`
- **D-24:** Content factory builds from the same config schema — no new config fields (constrained by `google_genai.json` schema with `additionalProperties: false`)
- **D-25:** Allowed config fields per upstream schema: `model_name`, `task_type`, `dimension`, `api_key_env_var`, `vertexai`, `project`, `location`
- **D-26:** Capabilities are runtime-derived from model name, not persisted in config
- **D-27:** Config round-trip shape stays unchanged: `{type: "known", name: "google_genai", config: {...}}`

### Claude's Discretion
- Internal helper function names and organization (e.g., `resolveBytes`, `resolveMIME`, `convertToGenaiContent`)
- Exact placement of `ValidateContentSupport` call within `EmbedContent`/`EmbedContents`
- Test scaffolding structure and assertion patterns
- Error message wording for unsupported modality/model combinations
- Whether to add Vertex AI support in this phase or defer

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Gemini API documentation
- https://ai.google.dev/gemini-api/docs/embeddings — EmbedContent API, multimodal examples, task types, dimensionality
- https://ai.google.dev/gemini-api/docs/models/gemini-embedding-2-preview — Gemini Embedding 2 model capabilities, modality limits, MIME types

### Cross-language config schema
- https://github.com/chroma-core/chroma/blob/main/schemas/embedding_functions/google_genai.json — Canonical config fields with `additionalProperties: false`

### Shared contract types
- `pkg/embeddings/multimodal.go` — `Intent`, `Modality`, `Content`, `Part`, `BinarySource`, `SourceKind` types
- `pkg/embeddings/embedding.go` — `ContentEmbeddingFunction`, `CapabilityAware`, `IntentMapper`, `EmbeddingFunction` interfaces
- `pkg/embeddings/capabilities.go` — `CapabilityMetadata` struct and support queries
- `pkg/embeddings/multimodal_validate.go` — `ValidationError`, `ValidateContentSupport`, validation codes
- `pkg/embeddings/multimodal_compat.go` — Compatibility adapters (reference for how Roboflow delegates)

### Registry
- `pkg/embeddings/registry.go` — `RegisterDense`, `RegisterContent`, `BuildContent` fallback chain

### Existing Gemini provider
- `pkg/embeddings/gemini/gemini.go` — Current text-only implementation, `Client` struct, `CreateEmbedding`, `GeminiEmbeddingFunction`
- `pkg/embeddings/gemini/task_type.go` — 8 Gemini task types
- `pkg/embeddings/gemini/option.go` — Functional options pattern

### Prior phase decisions
- `.planning/phases/03-registry-and-config-integration/03-CONTEXT.md` — BuildContent fallback chain, content factory registration, config round-trip
- `.planning/phases/04-provider-mapping-and-explicit-failures/04-CONTEXT.md` — IntentMapper contract, ValidateContentSupport, escape hatch behavior

### Requirements
- `.planning/ROADMAP.md` — Phase 6 goal, success criteria [GEM-01, GEM-02, GEM-03]
- `.planning/REQUIREMENTS.md` — GEM-01 (SharedContentEF + CapabilityAware), GEM-02 (intent mapping), GEM-03 (registry + config round-trip)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `Client` struct in `gemini.go`: API key, model, task type, dimension, genai client — extends naturally with content embedding methods
- `buildEmbedContentConfig`: Constructs `genai.EmbedContentConfig` from task type and dimension — reusable for both text and content paths
- `taskTypeFromContext` / `modelFromContext` / `outputDimensionalityFromContext`: Context-based override helpers — shared between paths
- `ValidateContentSupport(content, caps)`: Pre-flight validation from Phase 4 — called before API dispatch
- `IsNeutralIntent(intent)`: Distinguishes neutral vs custom intents for mapping logic
- genai SDK constructors: `NewPartFromBytes(data, mime)`, `NewPartFromText(text)`, `NewContentFromParts(parts, role)`

### Established Patterns
- Functional options: `WithDefaultModel`, `WithTaskType`, `WithDimension` — same pattern for any new options
- `init()` registration with `RegisterDense` — extend with `RegisterContent` in same `init()` block
- Interface compile-time assertions: `var _ embeddings.ContentEmbeddingFunction = (*GeminiEmbeddingFunction)(nil)`
- Config round-trip via `Name()` + `GetConfig()` → `NewGeminiEmbeddingFunctionFromConfig(cfg)`
- Roboflow as reference: implements `ContentEmbeddingFunction` + `CapabilityAware` directly (though delegates via adapter)

### Integration Points
- `GeminiEmbeddingFunction` struct gains `ContentEmbeddingFunction`, `CapabilityAware`, and `IntentMapper` interface implementations
- `Client` gains `CreateContentEmbedding` method alongside existing `CreateEmbedding`
- `init()` block adds `RegisterContent("google_genai", ...)` call
- `NewGeminiContentEFFromConfig` factory function for the content registry path

</code_context>

<specifics>
## Specific Ideas

- Gemini is the first provider to natively support mixed-part content (SupportsMixedPart: true) — Roboflow delegates through the adapter which requires single-part Content items
- The model-based capability derivation pattern (embedding-2 → full multimodal, embedding-001 → text only) could serve as a reference for future providers with multiple model tiers
- Negative test case for legacy model + multimodal content validates the explicit failure path from Phase 4's ValidateContentSupport

</specifics>

<deferred>
## Deferred Ideas

- Vertex AI backend support (vertexai, project, location config fields) — schema supports it but implementation deferred
- Gemini Files API integration for large media (alternative to inline blobs for files exceeding size limits) — future enhancement

</deferred>

---

*Phase: 06-gemini-multimodal-adoption*
*Context gathered: 2026-03-20*
