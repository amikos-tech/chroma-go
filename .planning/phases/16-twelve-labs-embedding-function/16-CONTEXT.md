# Phase 16: Twelve Labs Embedding Function - Context

**Gathered:** 2026-04-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Add a Twelve Labs multimodal embedding provider (`pkg/embeddings/twelvelabs`) that implements the shared Content API interfaces, supports text, image, audio, and video modalities via the Embed API v2 sync endpoint, registers in the factory/registry with config round-trip support, and includes docs and examples.

</domain>

<decisions>
## Implementation Decisions

### API endpoint strategy
- **D-01:** Use the Embed API v2 sync endpoint (`POST /v1.3/embed-v2`) for all modalities. Single unified request path — no separate text vs multimodal code paths.
- **D-02:** Default model: `marengo3.0` (Marengo 2.7 was sunset March 30, 2026). Model is configurable via `WithModel` option.
- **D-03:** Sync endpoint only in this phase. Async support (`/v1.3/embed-v2/tasks` for audio/video up to 4 hours) deferred to a separate GitHub issue.

### Modality & content handling
- **D-04:** Advertise text, image, audio, and video in `CapabilityMetadata.Modalities`. Sync endpoint handles all four (audio/video limited to < 10 minutes).
- **D-05:** Single-modality per Content item only. `SupportsMixedPart: false`. Each Content item maps to one API call with one `input_type`.
- **D-06:** Input mapping: `SourceKindURL` → `media_source.url`, `SourceKindBase64`/`SourceKindBytes`/`SourceKindFile` → `media_source.base_64_string`. No `asset_id` support (Twelve Labs-specific concept).
- **D-07:** MIME type resolution follows established pattern: `BinarySource.MIMEType` first → infer from extension → fail explicitly.

### Authentication & config
- **D-08:** Environment variable: `TWELVE_LABS_API_KEY`.
- **D-09:** Audio `embedding_option` exposed as `WithAudioEmbeddingOption("audio"|"transcription"|"fused")` functional option. Default: `"audio"`.
- **D-10:** Embedding dimensions are fixed at 512 (Marengo 3.0 spec). No dimensionality option needed.

### Test approach
- **D-11:** httptest-based unit tests with mocked Twelve Labs API responses, using `ef` build tag.
- **D-12:** Separate integration test file with dedicated build tag that hits real Twelve Labs API (requires `TWELVE_LABS_API_KEY`).
- **D-13:** Test both `ContentEmbeddingFunction` (multimodal) and `EmbeddingFunction` (text-only) interfaces via dual registration pattern.

### Claude's Discretion
- Request body construction details and error mapping
- Config round-trip key naming
- Provider-side byte resolution implementation details (file → read, base64 → pass through, bytes → encode, URL → pass through)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Twelve Labs API docs
- https://docs.twelvelabs.io/api-reference/create-embeddings-v2 — Embed API v2 endpoint reference
- https://docs.twelvelabs.io/docs/guides/create-embeddings/text — Text embedding guide (model_name: `marengo3.0`, input_type: `text`)
- https://docs.twelvelabs.io/v1.3/docs/guides/create-embeddings/image — Image embedding guide (input_type: `image`, media_source with url/base_64_string)
- https://docs.twelvelabs.io/docs/guides/create-embeddings/audio — Audio embedding guide (input_type: `audio`, embedding_option: audio/transcription/fused)
- https://docs.twelvelabs.io/docs/concepts/models/marengo — Marengo 3.0 model specs (512d, supported formats)

### Codebase patterns
- `pkg/embeddings/voyage/voyage.go` — Reference provider: dual registration, ContentEmbeddingFunction, CapabilityAware, IntentMapper
- `pkg/embeddings/voyage/content.go` — Reference: Content API implementation, media_source mapping, byte resolution
- `pkg/embeddings/gemini/gemini.go` — Reference provider: unified endpoint pattern
- `pkg/embeddings/embedding.go` — Shared interfaces: ContentEmbeddingFunction, CapabilityAware, IntentMapper
- `pkg/embeddings/registry.go` — RegisterDense, RegisterContent functions
- `pkg/embeddings/openai/openai_test.go` — Reference: httptest mock pattern for embedding providers

### Project issue
- GitHub issue #190 — Twelve Labs embedding function feature request

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `ContentEmbeddingFunction` interface in `pkg/embeddings/embedding.go` — implement for multimodal support
- `CapabilityAware` interface — advertise modalities, intents, mixed-part support
- `IntentMapper` interface — translate neutral intents to provider-native strings
- `RegisterDense` + `RegisterContent` in `pkg/embeddings/registry.go` — dual registration
- Content constructors in `pkg/embeddings/content_constructors.go` — `NewTextContent`, `NewImageURL`, etc.
- `pkg/internal/pathutil` — shared path safety utilities
- `pkg/commons/http` — shared HTTP client utilities

### Established Patterns
- Functional options pattern (`WithModel`, `WithAPIKey`, etc.) for provider configuration
- Provider-side byte resolution: `EmbedContent` resolves `BinarySource` kinds before API call
- MIME type inference from extension as fallback via `resolveMIME`
- Config round-trip: `GetConfig()`/`BuildFromConfig()` for registry persistence
- Context-based per-request options (e.g., `ContextWithModel`)

### Integration Points
- `pkg/embeddings/registry.go` — register as `"twelvelabs"` for both dense and content
- `pkg/embeddings/build_from_json_test.go` — config round-trip test patterns
- `docs/docs/embeddings.md` — provider documentation
- `examples/v2/` — usage examples

</code_context>

<specifics>
## Specific Ideas

### Twelve Labs API v2 Request Format
- Text: `{ input_type: "text", model_name: "marengo3.0", text: { input_text: "..." } }`
- Image: `{ input_type: "image", model_name: "marengo3.0", image: { media_source: { url | base_64_string } } }`
- Audio: `{ input_type: "audio", model_name: "marengo3.0", audio: { media_source: { url | base_64_string }, embedding_option: "audio" } }`
- Video: `{ input_type: "video", model_name: "marengo3.0", video: { media_source: { url | base_64_string } } }`
- Response: `{ data: [{ embedding: [float...] }] }` — 512-dimensional vectors

### API Details
- Base URL: `https://api.twelvelabs.io`
- API version: v1.3
- Sync endpoint: `POST /v1.3/embed-v2` (content < 10 min)
- Max images per request: 10
- Max text length: 500 tokens
- Auth: API key in header

</specifics>

<deferred>
## Deferred Ideas

- **Async embedding support** — `POST /v1.3/embed-v2/tasks` for audio/video up to 4 hours. Requires task creation + polling loop. Will create a GitHub issue for this.

</deferred>

---

*Phase: 16-twelve-labs-embedding-function*
*Context gathered: 2026-04-01*
