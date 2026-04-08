# Phase 7: Voyage Multimodal Adoption - Context

**Gathered:** 2026-03-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire VoyageAI into the shared multimodal contract so it supports text, image, and video embeddings through the portable `ContentEmbeddingFunction`, `CapabilityAware`, and `IntentMapper` interfaces, while keeping existing text-only `EmbedDocuments`/`EmbedQuery` behavior unchanged. Voyage becomes the second provider (after Gemini) to natively implement the shared content interface, validating that the foundation is truly portable across providers.

**Pivot note:** Originally scoped as vLLM/Nemotron provider validation. Pivoted because vLLM does not support `nvidia/omni-embed-nemotron-3b` (custom `NVOmniEmbedModel` architecture not in vLLM's registry), and Ollama's embedding endpoint is text-only. VoyageAI's multimodal API provides a clean validation target with an existing provider to extend.

</domain>

<decisions>
## Implementation Decisions

### Provider structure
- **D-01:** Extend existing `pkg/embeddings/voyage/` package — no new package needed
- **D-02:** Add `CreateMultimodalEmbedding()` to existing `VoyageAIClient`, targeting `/v1/multimodalembeddings` endpoint
- **D-03:** Keep existing `VoyageAIEmbeddingFunction` unchanged; add `ContentEmbeddingFunction`, `CapabilityAware`, `IntentMapper` interface implementations
- **D-04:** Dual registration: keep existing `RegisterDense("voyageai")` and add `RegisterContent("voyageai")`

### Content part handling
- **D-05:** Map `SourceKindURL` → Voyage `image_url`/`video_url` type; `SourceKindBase64`/`SourceKindBytes`/`SourceKindFile` → resolve to bytes then `image_base64`/`video_base64` with `data:<mimetype>;base64,<data>` URI prefix
- **D-06:** Provider-side byte resolution: Voyage's `EmbedContent` impl resolves all `BinarySource` kinds (file → read, base64 → decode, bytes → pass through, URL → pass through as `image_url`/`video_url`)
- **D-07:** MIME type resolved from `BinarySource.MIMEType` first; file-backed sources infer from extension as fallback; fail explicitly if MIME type is empty and can't be inferred
- **D-08:** Mixed-part Content items supported natively — multiple Parts in one Content produce one input object with multiple content blocks. Advertise `SupportsMixedPart: true` in capabilities

### Modality support
- **D-09:** Advertise text, image, and video in `CapabilityMetadata.Modalities` (all Voyage multimodal-supported modalities)
- **D-10:** Default model: `voyage-multimodal-3.5` for multimodal instances
- **D-11:** Supported image formats: PNG, JPEG, WEBP, GIF (per Voyage docs)
- **D-12:** Video format: MP4 only (per Voyage docs)

### Intent mapping
- **D-13:** Implement `IntentMapper` interface with mapping:
  - `retrieval_query` → `"query"`
  - `retrieval_document` → `"document"`
- **D-14:** `MapIntent` rejects `classification`, `clustering`, `semantic_similarity` with explicit errors — no silent degradation
- **D-15:** `MapIntent` checks `ProviderHints["input_type"]` first (override), then maps neutral intent, then passes null for no-intent
- **D-16:** Batch requests reject per-item Intent fields; single-item requests allow per-item ProviderHints override
- **D-17:** Declare only `retrieval_query` + `retrieval_document` in `CapabilityMetadata.Intents`

### Dimension support
- **D-18:** `voyage-multimodal-3.5` supports output dimensions: 256, 512, 1024 (default), 2048
- **D-19:** Advertise `RequestOptionDimension` in `CapabilityMetadata.RequestOptions`
- **D-20:** Per-request `Dimension` field maps to Voyage's `output_dimension` parameter (if the API supports it — verify during research)

### Registry and config
- **D-21:** Content factory builds from the same config schema — no new config fields (constrained by `voyageai.json` schema with `additionalProperties: false`)
- **D-22:** Allowed config fields per upstream schema: `model_name`, `api_key_env_var`, `input_type`, `truncation`
- **D-23:** Capabilities are runtime-derived from model name, not persisted in config
- **D-24:** Config round-trip shape stays unchanged: `{type: "known", name: "voyageai", config: {...}}`

### Backward compatibility
- **D-25:** Shared helpers, separate entry points — `CreateEmbedding` (text path) stays unchanged, new `CreateMultimodalEmbedding` (multimodal path) added
- **D-26:** Existing `EmbedDocuments`/`EmbedQuery` delegate through unchanged `CreateEmbedding` on `/v1/embeddings`
- **D-27:** New `EmbedContent`/`EmbedContents` delegate through `CreateMultimodalEmbedding` on `/v1/multimodalembeddings`

### Claude's Discretion
- Internal helper function names and organization (e.g., `resolveSource`, `buildContentBlock`, `resolveMIME`)
- Exact placement of `ValidateContentSupport` call within `EmbedContent`/`EmbedContents`
- Test scaffolding structure and assertion patterns
- Error message wording for unsupported modality/intent combinations
- Whether to add `output_encoding` (base64 response optimization) support in this phase or defer

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### VoyageAI API documentation
- https://docs.voyageai.com/reference/multimodal-embeddings-api — Multimodal embeddings endpoint, request/response format, constraints, supported input types
- https://docs.voyageai.com/docs/multimodal-embeddings — Multimodal capabilities overview, models, modalities, input_type parameter, dimension options

### Cross-language config schema
- https://github.com/chroma-core/chroma/blob/main/schemas/embedding_functions/voyageai.json — Canonical config fields with `additionalProperties: false`

### Shared contract types
- `pkg/embeddings/multimodal.go` — `Intent`, `Modality`, `Content`, `Part`, `BinarySource`, `SourceKind` types
- `pkg/embeddings/embedding.go` — `ContentEmbeddingFunction`, `CapabilityAware`, `IntentMapper`, `EmbeddingFunction` interfaces
- `pkg/embeddings/capabilities.go` — `CapabilityMetadata` struct and support queries
- `pkg/embeddings/multimodal_validate.go` — `ValidationError`, `ValidateContentSupport`, validation codes
- `pkg/embeddings/multimodal_compat.go` — Compatibility adapters (reference for delegation pattern)

### Registry
- `pkg/embeddings/registry.go` — `RegisterDense`, `RegisterContent`, `BuildContent` fallback chain

### Existing Voyage provider
- `pkg/embeddings/voyage/voyage.go` — Current text-only implementation, `VoyageAIClient` struct, `CreateEmbedding`, `VoyageAIEmbeddingFunction`
- `pkg/embeddings/voyage/options.go` — Functional options pattern

### Prior phase decisions (reference implementations)
- `.planning/phases/06-gemini-multimodal-adoption/06-CONTEXT.md` — Gemini as first native ContentEmbeddingFunction provider; model-based capability derivation, dual registration, provider-side byte resolution, intent mapping with escape hatch
- `.planning/phases/04-provider-mapping-and-explicit-failures/04-CONTEXT.md` — IntentMapper contract, ValidateContentSupport, escape hatch behavior
- `.planning/phases/03-registry-and-config-integration/03-CONTEXT.md` — BuildContent fallback chain, content factory registration, config round-trip

### Requirements
- `.planning/ROADMAP.md` — Phase 7 goal, success criteria
- `.planning/REQUIREMENTS.md` — VLLM-01 and VLLM-02 (to be reworded for Voyage pivot)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `VoyageAIClient` struct in `voyage.go`: API key, model, truncation, encoding format, HTTP client — extends naturally with multimodal embedding method
- `InputType` constants (`"query"`, `"document"`): Already defined, directly reusable for intent mapping
- Context override helpers (`getModel`, `getTruncation`, `getInputType`, `getEncodingFormat`): Shared between text and content paths
- `ValidateContentSupport(content, caps)`: Pre-flight validation from Phase 4 — called before API dispatch
- `IsNeutralIntent(intent)`: Distinguishes neutral vs custom intents for mapping logic
- `EmbeddingTypeResult.UnmarshalJSON`: Handles both float array and base64-encoded response formats — reusable for multimodal response parsing

### Established Patterns
- Functional options: `WithAPIKey`, `WithDefaultModel`, `WithBaseURL` — same pattern for any new options
- `init()` registration with `RegisterDense` — extend with `RegisterContent` in same `init()` block
- Interface compile-time assertions: `var _ embeddings.ContentEmbeddingFunction = (*VoyageAIEmbeddingFunction)(nil)`
- Config round-trip via `Name()` + `GetConfig()` → `NewVoyageAIEmbeddingFunctionFromConfig(cfg)`
- Gemini as reference: implements `ContentEmbeddingFunction` + `CapabilityAware` + `IntentMapper` natively

### Integration Points
- `VoyageAIEmbeddingFunction` gains `ContentEmbeddingFunction`, `CapabilityAware`, and `IntentMapper` interface implementations
- `VoyageAIClient` gains `CreateMultimodalEmbedding` method alongside existing `CreateEmbedding`
- `init()` block adds `RegisterContent("voyageai", ...)` call
- New multimodal request/response types for the `/v1/multimodalembeddings` endpoint

</code_context>

<specifics>
## Specific Ideas

- Voyage is the second provider to natively support mixed-part content (`SupportsMixedPart: true`) after Gemini — validates that the shared contract works across different API shapes
- The separate endpoint pattern (`/v1/embeddings` vs `/v1/multimodalembeddings`) differs from Gemini's single-endpoint approach — proves the contract accommodates both patterns
- Voyage's `input_type` has only 2 values vs Gemini's 8 task types — validates that IntentMapper works with sparse intent support and explicit rejection of unsupported intents
- Voyage's dimension flexibility (256/512/1024/2048) validates the `RequestOptionDimension` capability path

</specifics>

<deferred>
## Deferred Ideas

- vLLM/Nemotron provider — blocked on vLLM adding `NVOmniEmbedModel` architecture support; revisit when vLLM supports it
- Voyage `output_encoding: "base64"` response optimization — future enhancement for bandwidth reduction
- Video-specific integration tests — start with text + image, add video tests later if a test video fixture is available

</deferred>

---

*Phase: 07-voyage-multimodal-adoption*
*Context gathered: 2026-03-22*
