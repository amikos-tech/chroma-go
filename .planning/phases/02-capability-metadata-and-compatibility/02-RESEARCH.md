# Phase 2: Capability Metadata and Compatibility - Research

**Researched:** 2026-03-19
**Domain:** Go embedding capability introspection, additive compatibility adapters, and regression-safe multimodal expansion
**Confidence:** HIGH

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CAPS-01 | A provider can declare which modalities, intents, and request options it supports through shared capability metadata | Add shared capability structs and a provider-facing capability interface in `pkg/embeddings`, with explicit modality, intent, and option fields instead of provider-specific concrete types. |
| CAPS-02 | Caller can inspect shared capability metadata without depending on provider-specific concrete types | Expose capability discovery through a shared interface implemented by providers or compatibility wrappers, so callers can type-assert only the shared interface. |
| COMP-01 | Existing `EmbeddingFunction` text-only callers continue to compile and behave the same without adopting the new multimodal request API | Keep `EmbeddingFunction` unchanged and provide additive wrappers/helpers that can project text-only providers into the richer capability surface without changing call sites. |
| COMP-02 | Existing image-only multimodal callers continue to compile and interoperate with the new shared multimodal foundations | Keep `MultimodalEmbeddingFunction` unchanged and add compatibility delegation between legacy image methods and the shared `Content` contract for supported single-image and single-text cases. |
</phase_requirements>

## Summary

Phase 2 should stay inside `pkg/embeddings` and the existing provider packages. The repo already has the additive shared multimodal request model from Phase 1, but there is still no shared way to ask "what can this provider do?" and no shared bridge between the legacy text-only/image-only interfaces and the richer `Content` contract. That makes capability discovery and compatibility the highest-risk seam before registry/config work in Phase 3.

The main design constraint is backward compatibility. `EmbeddingFunction`, `MultimodalEmbeddingFunction`, `BuildDense`, `BuildMultimodal`, and `pkg/api/v2/configuration.go` still assume the current public interfaces. Phase 2 should therefore add capability metadata and compatibility adapters without changing existing method signatures, config keys, or registration behavior. Any richer surface must be additive and optional.

The safest decomposition is:
1. Define shared capability metadata and capability-reporting interfaces in `pkg/embeddings`.
2. Add compatibility adapters and delegation helpers that bridge legacy text/image providers to `Content` for the supported single-part cases.
3. Lock the behavior down with regression tests that prove legacy callers, capability inspection, and supported compatibility paths work unchanged.

**Primary recommendation:** introduce shared capability metadata plus additive adapter interfaces in `pkg/embeddings`, implement the first concrete capability provider on `roboflow`, and keep unsupported mixed-part or unsupported-modality requests as explicit errors rather than implicit downgrades.

## Current Code Reality

### What already exists

- `pkg/embeddings/embedding.go` now defines `Content`, `Part`, `BinarySource`, `Intent`, and additive `ContentEmbeddingFunction`.
- `pkg/embeddings/registry.go` still has separate dense and image-only multimodal registries.
- `pkg/api/v2/configuration.go` only rebuilds dense `EmbeddingFunction` instances today.
- `pkg/embeddings/roboflow/roboflow.go` implements both `EmbeddingFunction` and `MultimodalEmbeddingFunction`, making it the best first target for capability metadata and compatibility delegation.
- Several dense providers already expose task or dimension-like knobs through config or context (`gemini`, `nomic`, `openai`, `bedrock`, `perplexity`, `jina`, `chromacloud`), which means the capability model must not be hard-coded to Roboflow's simpler shape.

### What is still missing

- No shared capability metadata type for modalities, intents, or request options.
- No shared capability interface callers can inspect safely.
- No shared adapter that turns legacy text-only or image-only provider behavior into `Content`-level behavior for the supported cases.
- No tests that prove the capability surface is provider-neutral and does not break current caller entry points.

## Recommended Structure

```text
pkg/embeddings/
├── embedding.go                  # keep legacy interfaces; add shared capability interfaces if kept central
├── multimodal.go                 # existing Content/Part/Intent types
├── multimodal_compat.go          # extend with compatibility adapters/helpers
├── capabilities.go               # shared capability metadata, option enums, helper predicates
├── capabilities_test.go          # shared capability and adapter tests
└── registry.go                   # unchanged in Phase 2 unless tests need helper accessors only

pkg/embeddings/roboflow/
├── roboflow.go                   # implement shared capability reporting and supported compatibility path
└── roboflow_test.go              # regression coverage for capabilities and legacy interfaces
```

## Architecture Patterns

### Pattern 1: Shared Capability Metadata, Not Provider Concrete Types

**What:** Define a small shared metadata model that callers can inspect through one shared interface.

**Recommended shape (HIGH confidence on intent, MEDIUM on exact names):**

```go
type RequestOption string

const (
	RequestOptionDimension     RequestOption = "dimension"
	RequestOptionProviderHints RequestOption = "provider_hints"
)

type CapabilityMetadata struct {
	Modalities        []Modality
	Intents           []Intent
	RequestOptions    []RequestOption
	SupportsBatch     bool
	SupportsMixedPart bool
}

type CapabilityAware interface {
	Capabilities() CapabilityMetadata
}
```

**Why this fits the repo:** callers can type-assert only `CapabilityAware`, providers remain free to implement the interface directly, and future phases can extend this metadata without breaking legacy embedding method signatures.

### Pattern 2: Additive Compatibility Wrappers Around Legacy Interfaces

**What:** Add helpers or wrapper types that expose limited `ContentEmbeddingFunction` behavior on top of existing interfaces.

**Recommended supported cases:**
- `EmbeddingFunction` -> `ContentEmbeddingFunction` only for `Content` containing exactly one text part and no unsupported request options.
- `MultimodalEmbeddingFunction` -> `ContentEmbeddingFunction` for exactly one text part or one image part.
- Single image `Content` should reuse `NewImagePartFromImageInput`/`ImageInput` bridging logic rather than inventing a second source model.
- Mixed-part content, audio/video/PDF on legacy providers, or unsupported request options should return explicit unsupported errors.

**Why this fits the repo:** it preserves COMP-01 and COMP-02 without pretending legacy providers already understand the full shared multimodal contract.

### Pattern 3: Capability Metadata Must Reflect Current Provider Behavior, Not Future Aspirations

**What:** A provider should advertise only the modalities, intents, and options it actually supports today.

**Concrete example:**
- A text-only provider may advertise `Modalities: [ModalityText]`.
- Roboflow should likely advertise `Modalities: [ModalityText, ModalityImage]`, limited/no portable intent support, and no mixed-part support yet.
- Providers like Gemini/OpenAI/Nomic can eventually advertise dimension or intent support, but Phase 2 only needs the shared metadata seam and one or two implementations to prove it works.

**Why this fits the repo:** the roadmap intentionally stages provider mapping and explicit unsupported behavior later. Capability metadata should therefore be descriptive and conservative, not an abstraction that invents support the implementation does not have.

### Pattern 4: Keep Config and Registry Behavior Stable in This Phase

**What:** Do not widen `BuildEmbeddingFunctionFromConfig` or collection auto-wiring yet.

**Why:** `pkg/api/v2/configuration.go` still rebuilds dense embedding functions only, and roadmap Phase 3 explicitly owns richer registry/config integration. Phase 2 should only ensure the new capability and compatibility surface does not regress existing config round-trips.

## Concrete Planning Implications

### Plan 02-01: Shared capability metadata types and interfaces

Best scope:
- Add `CapabilityMetadata` plus any supporting option enums/helpers in `pkg/embeddings`.
- Add one shared interface for capability inspection.
- Add tests that prove capability inspection does not require provider concrete types.

Likely files:
- `pkg/embeddings/capabilities.go`
- `pkg/embeddings/embedding.go`
- `pkg/embeddings/capabilities_test.go`

### Plan 02-02: Compatibility adapters and delegation paths

Best scope:
- Extend `pkg/embeddings/multimodal_compat.go` or add a sibling adapter file with text/image compatibility wrappers.
- Add explicit unsupported errors for mixed-part, unsupported modality, or unsupported option cases on legacy paths.
- Implement capability reporting and supported delegation on `roboflow`.

Likely files:
- `pkg/embeddings/multimodal_compat.go`
- `pkg/embeddings/embedding.go`
- `pkg/embeddings/roboflow/roboflow.go`
- `pkg/embeddings/roboflow/roboflow_test.go`

### Plan 02-03: Regression tests for legacy callers

Best scope:
- Add shared tests for text-only compatibility, image-only compatibility, and capability inspection through shared interfaces.
- Re-run the current config reconstruction tests in `pkg/api/v2/configuration_test.go` to prove there is no regression.
- Verify registry presence tests remain unchanged.

Likely files:
- `pkg/embeddings/capabilities_test.go`
- `pkg/embeddings/roboflow/roboflow_test.go`
- `pkg/api/v2/configuration_test.go`

## Anti-Patterns to Avoid

- **Do not replace legacy interfaces or config objects.**
  `EmbeddingFunction`, `MultimodalEmbeddingFunction`, and `EmbeddingFunctionInfo` are active compatibility surface.

- **Do not make capability metadata provider-specific.**
  Fields like `task_type`, `clip_version`, or provider model names belong in provider config, not the shared capability contract.

- **Do not silently coerce mixed-part content into one legacy path.**
  If a provider cannot support mixed parts, audio/video/PDF, or request-time options, return explicit unsupported errors.

- **Do not pull registry/config reconstruction forward.**
  Any plan that changes `BuildEmbeddingFunctionFromConfig` to rebuild richer content interfaces is leaking Phase 3 into Phase 2.

- **Do not overfit capability metadata to Roboflow.**
  The contract must leave room for text-only providers with task/dimension options and for later richer multimodal providers.

## Common Pitfalls

### Pitfall 1: Capability metadata that mixes "supported now" with "planned later"

**Why it happens:** teams try to encode roadmap aspirations into the interface.
**Avoidance:** capability metadata should only describe current runtime behavior. Future mapping support belongs to later phases.

### Pitfall 2: Adapters that accept unsupported request options and ignore them

**Why it happens:** it is tempting to reuse legacy providers by dropping `Intent`, `Dimension`, or extra parts.
**Avoidance:** adapters must fail explicitly when the content request cannot be represented safely in the legacy interface.

### Pitfall 3: Shared tests that only exercise concrete provider types

**Why it happens:** the first implementation target is Roboflow.
**Avoidance:** tests must type-assert only the shared capability interface and shared compatibility adapter behavior, then separately cover Roboflow as a concrete implementation.

### Pitfall 4: Config regression hidden behind additive interface work

**Why it happens:** Phase 2 is mostly in `pkg/embeddings`, so it is easy to forget that `pkg/api/v2/configuration.go` depends on those interfaces remaining stable.
**Avoidance:** include `go test -tags=basicv2 ./pkg/api/v2 -run '^TestBuildEmbeddingFunctionFromConfig$'` in every phase verification path.

## Validation Architecture

### Test Infrastructure

- Framework: `go test` + `github.com/stretchr/testify/require`
- Quick feedback loop: targeted `pkg/embeddings` and `pkg/api/v2/configuration_test.go` runs
- Full suite: `make test`
- Existing infrastructure is sufficient; no Wave 0 test scaffolding is required for this phase

### Recommended quick commands

```bash
go test ./pkg/embeddings -run 'TestCapabilityMetadata|TestLegacyTextCompatibility|TestLegacyImageCompatibility|TestCompatibilityAdapterRejectsUnsupportedContent|TestMultimodalInterface' && \
go test -tags=basicv2 ./pkg/api/v2 -run '^TestBuildEmbeddingFunctionFromConfig$|^TestCollectionConfiguration_(Get|Set)EmbeddingFunction'
```

### Recommended full command

```bash
make test
```

### Validation focus

- Shared capability metadata exposes modalities, intents, and request-option support through shared interfaces only
- Legacy text-only and image-only paths still compile and behave as before
- Shared compatibility adapters only accept representable `Content` shapes and fail explicitly for unsupported cases
- Existing dense config reconstruction in `pkg/api/v2/configuration.go` remains green

## Sources

- `CLAUDE.md`
- `.planning/PROJECT.md`
- `.planning/ROADMAP.md`
- `.planning/REQUIREMENTS.md`
- `.planning/STATE.md`
- `.planning/codebase/ARCHITECTURE.md`
- `.planning/codebase/CONCERNS.md`
- `.planning/codebase/TESTING.md`
- `.planning/research/SUMMARY.md`
- `.planning/phases/01-shared-multimodal-contract/01-RESEARCH.md`
- `pkg/embeddings/embedding.go`
- `pkg/embeddings/multimodal_compat.go`
- `pkg/embeddings/registry.go`
- `pkg/embeddings/roboflow/roboflow.go`
- `pkg/embeddings/roboflow/roboflow_test.go`
- `pkg/api/v2/configuration.go`
- `pkg/api/v2/configuration_test.go`

---
*Research completed: 2026-03-19*
*Ready for planning: yes*
