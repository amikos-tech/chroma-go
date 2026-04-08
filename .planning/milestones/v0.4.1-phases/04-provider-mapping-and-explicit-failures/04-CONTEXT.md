# Phase 4: Provider Mapping and Explicit Failures - Context

**Gathered:** 2026-03-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Define how provider-neutral intents and modalities map to provider-native semantics and fail clearly when a provider cannot support the request. This phase delivers the mapping interface, shared pre-flight validation helpers, and new error codes. It does not wire real providers into the mapping layer — that happens in Phases 6-7.

</domain>

<decisions>
## Implementation Decisions

### Intent mapping contract
- Provider-owned mapping via a standalone `IntentMapper` interface (not embedded in `ContentEmbeddingFunction`)
- `IntentMapper` exposes `MapIntent(Intent) → (string, error)` — each provider owns its own mapping table
- When a provider does not implement `IntentMapper` and receives a non-empty intent, the intent string is passed through as-is to the provider API (not rejected)
- A shared `ValidateContentSupport(content, caps)` helper is available for providers to call before dispatch — validates modality, intent, and dimension against `CapabilityMetadata`

### Failure semantics
- Unsupported-combination errors reuse the existing `ValidationError` type with new validation codes (`unsupported_intent`, `unsupported_modality`, `unsupported_dimension`)
- Capability validation happens eagerly via the shared pre-check helper before provider I/O
- For batch requests, validation fails on the first unsupported item (consistent with existing `ValidateContents` behavior)
- The shared pre-check validates all three: modality, intent, AND dimension support against `CapabilityMetadata`

### Provider adoption path
- Phase 4 delivers the contract, helpers, and tests only — no real provider implements `IntentMapper` in this phase
- Real provider adoption happens in Phase 6 (Gemini) and Phase 7 (vLLM/Nemotron)
- Providers declare supported intents in `CapabilityMetadata.Intents`; the shared pre-check rejects unsupported neutral intents before `MapIntent` is called
- When a provider does not implement `CapabilityAware` (e.g., adapted legacy providers), the shared pre-check skips validation entirely and passes through

### Escape hatch behavior
- `MapIntent` is always called for all intents (both neutral and custom/raw strings) — provider decides its own policy for custom values
- When both a neutral intent and a conflicting provider hint are set, the intent (portable field) wins per Phase 1 decision
- The shared pre-check skips intent validation against `CapabilityMetadata` when the intent is a non-neutral (custom) string — escape hatches bypass capability enforcement
- A public `IsNeutralIntent(Intent) bool` helper identifies whether an intent is one of the 5 shared neutral constants — used by pre-check and available to providers in `MapIntent`

### Claude's Discretion
- Concrete function signatures and parameter ordering for `ValidateContentSupport`
- Exact validation code string values for new unsupported-* codes
- Internal organization of the `IsNeutralIntent` helper (set vs switch)
- Test scaffolding structure for mock IntentMapper implementations

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 1 decisions (prior context)
- `.planning/phases/01-shared-multimodal-contract/01-CONTEXT.md` — Intent ergonomics: string-backed type, 5 neutral constants, optional intent, raw/custom strings allowed as escape hatch, portable field wins over provider hints

### Phase 2-3 decisions (prior context)
- `.planning/phases/03-registry-and-config-integration/03-CONTEXT.md` — BuildContent fallback chain, adapter pattern, CapabilityMetadata inference for adapted providers

### Shared contract types
- `pkg/embeddings/multimodal.go` — `Intent`, `Modality`, `Content`, `Part` types and 5 neutral intent constants
- `pkg/embeddings/capabilities.go` — `CapabilityMetadata`, `SupportsModality()`, `SupportsIntent()`, `SupportsRequestOption()`, `RequestOptionDimension`
- `pkg/embeddings/multimodal_validate.go` — `ValidationError`, `ValidationIssue`, validation codes, `Content.Validate()`, `ValidateContents()`
- `pkg/embeddings/multimodal_compat.go` — Compatibility adapters, `validateCompatibleContent()`, existing modality/intent rejection in adapters

### Provider-specific task types (mapping targets for Phases 6-7)
- `pkg/embeddings/gemini/task_type.go` — 8 Gemini task types (RETRIEVAL_QUERY, SEMANTIC_SIMILARITY, etc.)
- `pkg/embeddings/nomic/nomic.go` §47-50 — 4 Nomic task types (search_query, search_document, etc.)
- `pkg/embeddings/chromacloud/chromacloud.go` §25-30 — 2 ChromaCloud tasks (default, nl_to_code)

### Interfaces
- `pkg/embeddings/embedding.go` — `ContentEmbeddingFunction`, `CapabilityAware`, `Closeable`, `EmbeddingFunction` interfaces

### Requirements
- `.planning/ROADMAP.md` — Phase 4 goal, success criteria [MAP-01, MAP-02]
- `.planning/REQUIREMENTS.md` — MAP-01 (neutral intent mapping with tests), MAP-02 (explicit unsupported-combination failures)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `ValidationError` / `ValidationIssue` with path, code, message: existing structured error pattern for capability validation errors
- `CapabilityMetadata.SupportsIntent()` / `SupportsModality()` / `SupportsRequestOption()`: query methods ready for pre-check validation
- `validateCompatibleContent()` in `multimodal_compat.go`: pattern for pre-dispatch validation that Phase 4 generalizes
- `compatibilityError()` helper: creates single-issue `ValidationError` — reusable pattern for unsupported-combination errors

### Established Patterns
- Opt-in interfaces via type assertion: `CapabilityAware`, `Closeable`, `EmbeddingFunctionUnwrapper` — `IntentMapper` follows the same pattern
- Validation codes as unexported string constants: `validationCodeForbidden`, `validationCodeRequired`, etc. — new codes follow same convention
- Shared types live in `pkg/embeddings/` as exported types and interfaces

### Integration Points
- `IntentMapper` interface definition goes in `pkg/embeddings/` alongside `CapabilityAware`
- `ValidateContentSupport` function goes in `pkg/embeddings/` (possibly `multimodal_validate.go` or a new `content_validate.go`)
- `IsNeutralIntent` helper goes in `pkg/embeddings/multimodal.go` alongside intent constants
- New validation codes added to `multimodal_validate.go` constants block
- Provider `EmbedContent`/`EmbedContents` implementations will call `ValidateContentSupport` then `MapIntent` in Phases 6-7

</code_context>

<specifics>
## Specific Ideas

- The `ValidateContentSupport` helper should be a building block, not a mandatory gate — providers opt in by calling it in their `EmbedContent` implementation. This keeps the shared layer non-invasive.
- `IsNeutralIntent` enables a clean pre-check flow: if neutral and capability-declared, validate; if neutral and no caps, pass through; if custom, always pass through.
- The test suite for Phase 4 should use a mock/test `IntentMapper` that maps neutral intents to predictable native strings, validates the pre-check helper catches unsupported combinations, and verifies escape hatch pass-through for custom intents.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 04-provider-mapping-and-explicit-failures*
*Context gathered: 2026-03-20*
