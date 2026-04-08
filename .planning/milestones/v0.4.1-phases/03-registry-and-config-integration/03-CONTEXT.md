# Phase 3: Registry and Config Integration - Context

**Gathered:** 2026-03-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Extend registry and config-persistence flows so richer multimodal functions (ContentEmbeddingFunction) can be rebuilt from stored configuration without regressing existing dense and multimodal auto-wiring. This phase does not add new provider implementations or change intent/modality mapping behavior.

</domain>

<decisions>
## Implementation Decisions

### Registry extension
- Add a dedicated 4th factory map (`contentFactories`) with `RegisterContent`, `BuildContent`, `ListContent`, `HasContent`, `BuildContentCloseable` — full parity with the existing dense/sparse/multimodal pattern
- `BuildContent` uses a fallback chain: content factory (native) → multimodal factory + `AdaptMultimodalEmbeddingFunctionToContent` → dense factory + `AdaptEmbeddingFunctionToContent` → error
- Existing providers get content support automatically through the adapter fallback — no changes to existing provider `init()` functions
- When auto-adapting, if the built provider implements `CapabilityAware`, use its actual metadata; otherwise infer minimal capabilities from the interface type:
  - Dense: `{Modalities: [text], SupportsBatch: true}`
  - Multimodal: `{Modalities: [text, image], SupportsBatch: true}`
- New content-native providers (Gemini Phase 6, vLLM Phase 7) will register directly in the content map later

### Config build chain
- `BuildEmbeddingFunctionFromConfig` keeps its existing return type (`EmbeddingFunction`) but gains a multimodal fallback: dense → multimodal → schema (existing)
- Add a new `BuildContentEFFromConfig` function returning `ContentEmbeddingFunction` with fallback chain: content → multimodal+adapt → dense+adapt
- Unknown provider names return `nil, nil` (matching existing auto-wiring contract where missing providers are silently skipped)

### Collection auto-wiring
- Collections gain an optional `contentEF` field alongside the existing `ef` field
- Auto-wiring populates both: `BuildEmbeddingFunctionFromConfig` for `ef`, `BuildContentEFFromConfig` for `contentEF`
- Add `WithContentEmbeddingFunction(ContentEmbeddingFunction)` as a new collection option alongside existing `WithEmbeddingFunction`
- If both are set, content takes priority; dense EF is derived from the content one
- Add `SetContentEmbeddingFunction` on `CollectionConfigurationImpl` for config persistence

### Config round-trip
- Config persistence shape remains unchanged: `{type, name, config}` — no capability metadata stored
- Content-native providers implement both `EmbeddingFunction` and `ContentEmbeddingFunction` (same pattern as `MultimodalEmbeddingFunction` embedding `EmbeddingFunction`)
- `SetContentEmbeddingFunction` type-asserts to `EmbeddingFunction` to extract `Name()`/`GetConfig()` for persistence; if assertion fails, skip persistence (runtime-only content EF)
- Capability metadata is purely a runtime concern — discovered at build time from the provider impl, not persisted

### Claude's Discretion
- Internal helper names for the adaptation logic in the build chain
- Exact placement of the auto-wiring content EF population within the HTTP client flow
- Whether `BuildContentCloseable` wraps the adapted closer or chains closers
- Test scaffolding organization and specific assertion patterns

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Registry and factory contracts
- `pkg/embeddings/registry.go` — Current 3-map registry (dense/sparse/multimodal) that Phase 3 extends with a 4th content map
- `pkg/embeddings/embedding.go` — `EmbeddingFunction`, `ContentEmbeddingFunction`, `CapabilityAware`, and config helper functions
- `pkg/embeddings/capabilities.go` — `CapabilityMetadata` struct and support queries used by auto-adapt fallback

### Compatibility adapters
- `pkg/embeddings/multimodal_compat.go` — `AdaptEmbeddingFunctionToContent` and `AdaptMultimodalEmbeddingFunctionToContent` used by the BuildContent fallback chain

### Configuration and auto-wiring
- `pkg/api/v2/configuration.go` — `BuildEmbeddingFunctionFromConfig`, `CollectionConfigurationImpl`, `EmbeddingFunctionInfo`, and `SetEmbeddingFunction`
- `pkg/api/v2/client_http.go` §421-425 — Auto-wiring callsite where `BuildEmbeddingFunctionFromConfig` is invoked during collection fetch

### Phase 1 context (prior decisions)
- `.planning/phases/01-shared-multimodal-contract/01-CONTEXT.md` — Shared request shape, intent ergonomics, and validation decisions that constrain registry behavior

### Project constraints
- `.planning/ROADMAP.md` — Phase 3 goal, success criteria, and requirements [REG-01, REG-02]
- `.planning/REQUIREMENTS.md` — REG-01 (richer multimodal builders) and REG-02 (auto-wiring stability)
- `.planning/PROJECT.md` — Compatibility, persistence, and validation constraints

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `AdaptEmbeddingFunctionToContent` / `AdaptMultimodalEmbeddingFunctionToContent`: Phase 2 adapters that the BuildContent fallback chain wraps around dense/multimodal factories
- `CapabilityMetadata` struct with `SupportsModality`, `SupportsIntent`, `SupportsRequestOption` query methods
- `ConfigInt`, `ConfigFloat64`, `ConfigStringSlice`: Config extraction helpers reusable by any new config path
- `BuildDenseCloseable` / `BuildMultimodalCloseable`: Closeable wrapper pattern to replicate for `BuildContentCloseable`

### Established Patterns
- Factory maps use `sync.RWMutex` for concurrent access — 4th map must follow same pattern
- `init()` registration with error-checked `Register*` calls and panic on duplicate
- `BuildEmbeddingFunctionFromConfig` returns `nil, nil` for unknown providers (auto-wiring contract)
- `SetEmbeddingFunction` stores `{type: "known", name: ef.Name(), config: ef.GetConfig()}`

### Integration Points
- `client_http.go:424` — Primary auto-wiring callsite; needs parallel `BuildContentEFFromConfig` call
- `collection.go` — Collection struct needs optional `contentEF` field and `WithContentEmbeddingFunction` option
- `configuration.go` — `SetContentEmbeddingFunction` setter and `BuildContentEFFromConfig` function
- Provider `init()` functions — Existing providers unchanged; new providers (Phases 6-7) will add `RegisterContent` calls

</code_context>

<specifics>
## Specific Ideas

- The fallback chain (content → multimodal+adapt → dense+adapt) mirrors how the Phase 2 compatibility adapters were designed — reuse them directly rather than creating new adapter logic
- Roboflow currently dual-registers as both dense and multimodal; the content fallback chain will find it via the multimodal path and auto-adapt
- The config persistence shape staying unchanged means existing collection configs from before Phase 3 will seamlessly gain content support when the build chain tries richer interfaces at load time

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 03-registry-and-config-integration*
*Context gathered: 2026-03-20*
