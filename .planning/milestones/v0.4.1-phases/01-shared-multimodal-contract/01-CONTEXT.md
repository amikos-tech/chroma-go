# Phase 1: Shared Multimodal Contract - Context

**Gathered:** 2026-03-18
**Status:** Ready for planning

<domain>
## Phase Boundary

Define the shared caller-facing multimodal request contract for Chroma Go: canonical content/item shape, ordered multimodal parts, neutral intent vocabulary, request-time override fields, and strict shared-shape validation. This phase does not decide provider capability introspection or provider-specific mapping behavior beyond preserving room for those later phases.

</domain>

<decisions>
## Implementation Decisions

### Public request shape
- The core embeddable unit is a single canonical `Content` item containing an ordered list of `Parts`.
- A mixed-part `Content` produces one aggregated embedding.
- Part order must be preserved exactly as provided by the caller.
- Phase 1 should expose both a single-item API and a batch API over the same canonical item shape:
  - `EmbedContent(content)`
  - `EmbedContents(contents)`
- Batch support is a companion API over repeated `Content` items, not the primary shape of one multimodal item.

### Media source model
- Binary modalities should share one common source pattern across image, audio, video, and PDF.
- The public contract must carry explicit modality, not infer it from MIME or source details.
- URL is a first-class source alongside file path, base64/inline bytes, and similar direct data forms.
- Source kind must remain explicit in the public model so provenance is preserved.
- Canonical execution may normalize to bytes when necessary, but the shared contract must preserve source provenance first.

### Intent ergonomics
- `Intent` should be a string-backed custom type, not a closed enum.
- The SDK should provide a small shared core of neutral intent constants, starting with:
  - `retrieval_query`
  - `retrieval_document`
  - `classification`
  - `clustering`
  - `semantic_similarity`
- Intent is optional. Omission means no explicit intent was set.
- Neutral intent names should be user-facing and domain-style, not provider-style spellings.
- Callers may still pass raw/custom string values in the same `Intent` field as an escape hatch.
- Neutral constants are the portable contract; raw/custom values are allowed but may be provider-specific.
- `retrieval_query` and `retrieval_document` remain distinct neutral intents, even for mixed multimodal content.
- Unsupported intents must fail explicitly.

### Request-time overrides
- Portable request-time overrides belong on the `Content` request object itself, not hidden in `context.Context`.
- Phase 1 portable overrides are limited to:
  - `intent`
  - output `dimension`
- Provider-specific request hints should live in a separate optional provider-hints map on the request.
- If a provider-hint conflicts with a portable field, the portable field wins.

### Validation and loading semantics
- Text parts stay minimal in Phase 1 and should only carry the text payload.
- Shared contract validation should be strict before provider I/O.
- Invalid shared shape should fail early, including empty content, empty parts, and invalid source combinations.
- File and URL sources remain lazy references until embedding time.
- For this version, URLs should be passed through directly to providers when they support URL-based inputs.
- Introducing an SDK-managed security boundary that fetches remote content into bytes is a later explicit decision, not implicit in Phase 1.
- `dimension` is an optional shared request hint.
- If a provider cannot support the requested dimension, the call should hard fail with a clear unsupported-dimension error rather than warn and ignore it.

### Claude's Discretion
- Concrete type names and helper constructor names, as long as they preserve the decisions above.
- Exact field naming for provider-hints and source-kind fields.
- Internal adapter layout for turning file/base64/URL helpers into the canonical public part model.
- Whether compatibility shims rely on helper constructors, wrapper methods, or small adapter types, as long as the Phase 1 public contract stays consistent with these decisions.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Planning Scope
- `.planning/ROADMAP.md` — Phase 1 goal, success criteria, and explicit phase boundary
- `.planning/REQUIREMENTS.md` — `MMOD-01` through `MMOD-05`, which define the required shared contract behavior
- `.planning/PROJECT.md` — project-level compatibility, portability, persistence, and validation constraints
- `.planning/research/SUMMARY.md` — recommended phase ordering and the specific risks around compatibility and semantic drift

### Existing Shared Contract
- `pkg/embeddings/embedding.go` — current `EmbeddingFunction`, `ImageInput`, and `MultimodalEmbeddingFunction` definitions that Phase 1 evolves
- `pkg/embeddings/registry.go` — current dense/sparse/multimodal registry structure that the new contract must remain compatible with conceptually
- `pkg/api/v2/configuration.go` — current config reconstruction constraints that the shared contract must not preclude

### Current Public Behavior
- `docs/docs/embeddings.md` — current public embedding docs, including Roboflow text/image multimodal examples and current provider override behavior
- `.planning/codebase/CONCERNS.md` — known contract fragmentation, docs drift, and the explicit note that provider-neutral multimodal foundations are missing
- `.planning/codebase/ARCHITECTURE.md` — architectural layering and shared-contract placement inside the repo

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/embeddings/embedding.go` `ImageInput`: existing helper pattern for file, URL, and base64-backed sources that can inform the generalized binary source model
- `pkg/embeddings/openai`, `pkg/embeddings/gemini`, `pkg/embeddings/nomic`: existing request-time override behavior for dimension/task via `context.Context`, which Phase 1 should make explicit in the request model
- `pkg/embeddings/registry.go`: shared registry split between dense, sparse, and multimodal builders; later phases will build on this structure
- `docs/docs/embeddings.md`: current examples and option documentation that must be updated once the new contract exists

### Established Patterns
- Shared public contracts live in `pkg/embeddings` and use explicit exported types and interfaces
- Provider configuration persists through serializable `EmbeddingFunctionConfig` maps with env-var indirection for secrets
- Validation is expected to happen early and return wrapped, explicit errors rather than silently degrading
- Functional options are used for provider defaults, but request-time behavior is currently inconsistently threaded through `context.Context`

### Integration Points
- `pkg/embeddings/embedding.go` is the primary integration point for the new shared item/part/intent/override types
- Existing provider packages already distinguish batch vs single-item methods (`EmbedDocuments` / `EmbedQuery`), which maps well to the planned `EmbedContents` / `EmbedContent` split
- `pkg/api/v2/configuration.go` and provider `GetConfig()` implementations constrain how far Phase 1 can drift from the current compatibility story, even though full config integration is Phase 3
- Public docs in `docs/docs/embeddings.md` currently reflect image-specific multimodal behavior and should be treated as a user-facing constraint on naming and migration clarity

</code_context>

<specifics>
## Specific Ideas

- Align the shared item model with the newer Gemini embedding pattern: one semantic unit with ordered parts yields one aggregated embedding, while batching lives outside that core item shape.
- Do not overfit the shared contract to the older Vertex multimodal embedding API shape that uses separate top-level modality fields and modality-specific outputs.
- Preserve URL as a first-class source so providers that safely accept URLs can consume them without the SDK fetching untrusted remote content itself.
- Keep the `Intent` field flexible via a string-backed custom type: neutral constants for portability, raw strings for advanced provider-specific escape hatches.

</specifics>

<deferred>
## Deferred Ideas

- Broader neutral intent catalog beyond the initial shared core — future expansion once more provider mappings are understood
- Additional text-part metadata such as language, MIME, labels, or annotations — defer until a concrete provider need emerges
- SDK-managed remote fetching/security boundary for URL sources — separate future decision once provider behavior and threat boundaries are clearer
- Stronger universal guarantees around dimension support or a wider shared override surface — revisit after provider capability work

</deferred>

---
*Phase: 01-shared-multimodal-contract*
*Context gathered: 2026-03-18*
