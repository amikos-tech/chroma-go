# Feature Research

**Domain:** Brownfield Go SDK milestone for provider-neutral multimodal embeddings
**Researched:** 2026-03-18
**Confidence:** HIGH

## Feature Landscape

### Table Stakes (Users Expect These)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Rich multimodal request model | Portable multimodal support is impossible if the shared contract only speaks text and image separately | HIGH | Needs ordered parts, mixed-part support, and validation |
| Provider-neutral intent semantics | Provider task names already vary (`search_query`, `retrieval.query`, `query`) and users should not have to learn each one first | HIGH | Shared intents need a provider mapping layer and explicit unsupported errors |
| Capability introspection | Callers need to know what a provider can do before issuing mixed-modality requests | MEDIUM | Shared metadata should expose modalities, intents, and option support |
| Backward compatibility | This is a public SDK with existing dense and image-only callers | HIGH | Acceptance criteria require no breaking changes |
| Config/registry persistence | Collections already rely on stored embedding config for reconstruction | HIGH | New multimodal foundations must round-trip through config and registry code |

### Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Mixed-part request support | Enables future providers to embed compound prompts instead of forcing separate text/image APIs | HIGH | Foundation work only; provider adoption can follow later |
| Provider-specific escape hatches on top of a portable core | Lets advanced callers use provider knobs without giving up a neutral shared API | MEDIUM | Keep the escape hatch additive and clearly documented |
| Portable docs and examples | Makes the richer contract usable across providers instead of only understandable by reading code | MEDIUM | Important because docs drift already exists in multimodal areas |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Migrate every provider immediately | Feels comprehensive | Explodes scope before the shared contract stabilizes | Ship the foundation first, then migrate providers incrementally |
| Replace legacy interfaces with one new interface | Feels cleaner on paper | Breaks existing callers and contradicts the issue acceptance criteria | Keep legacy APIs and bridge them to the richer contract |
| Silent fallback from unsupported multimodal requests to text-only behavior | Feels convenient | Produces ambiguous behavior and cross-provider surprises | Fail explicitly with capability-aware errors |

## Feature Dependencies

```text
[Shared multimodal request model]
    └──requires──> [Validation primitives]
    └──enables───> [Capability introspection]
                         └──enables──> [Provider mapping contract]
                                             └──requires──> [Registry/config integration]

[Compatibility adapters] ──protect──> [Existing text-only and image-only callers]

[Docs and tests] ──validate──> [Portable contract adoption]
```

### Dependency Notes

- **Capability introspection requires the shared request model:** providers cannot declare modality or intent support until those concepts exist in shared types
- **Provider mapping requires capabilities:** unsupported combinations should be rejected based on declared support, not guessed
- **Registry/config integration depends on the final contract shape:** persistence should stabilize after the additive interfaces and options exist
- **Docs and tests depend on all earlier phases:** public guidance is only useful once the contract and compatibility behavior are settled

## MVP Definition

### Launch With (v1)

- [ ] Additive multimodal request and option types — needed to express the new contract
- [ ] Provider-neutral intents and capability metadata — needed for portability
- [ ] Compatibility-safe registry/config integration — needed to avoid breaking auto-wiring
- [ ] Explicit failure behavior and tests — needed to prevent silent regressions
- [ ] Docs for portable usage and escape hatches — needed for adoption

### Add After Validation (v1.x)

- [ ] Migrate additional providers beyond the current multimodal baseline — once the shared contract proves stable
- [ ] Add richer end-to-end examples for providers that adopt audio, video, or PDF inputs — once concrete providers exist

### Future Consideration (v2+)

- [ ] Provider capability discovery from remote metadata — defer until there is enough provider coverage to justify it
- [ ] Higher-level multimodal batching helpers — defer until multiple providers expose compatible batch semantics

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Shared multimodal request model | HIGH | HIGH | P1 |
| Neutral intents and options | HIGH | HIGH | P1 |
| Capability introspection | HIGH | MEDIUM | P1 |
| Compatibility adapters | HIGH | MEDIUM | P1 |
| Registry/config integration | HIGH | HIGH | P1 |
| Portable docs/examples | MEDIUM | MEDIUM | P2 |
| Broad provider migration | MEDIUM | HIGH | P3 |

**Priority key:**
- P1: Must have for launch
- P2: Should have, add when possible
- P3: Nice to have, future consideration

## Competitor Feature Analysis

| Feature | Current Chroma Go State | Provider-Native SDK State | Our Approach |
|---------|-------------------------|---------------------------|--------------|
| Multimodal request shape | Text-only dense plus image-only multimodal methods | Usually provider-specific request objects and task enums | Introduce a portable additive request model |
| Intent/task semantics | Inconsistent across providers and mostly implicit today | Exposed through provider-specific terminology | Use neutral intents with explicit provider mapping |
| Capability discovery | Limited to reading provider docs or code | Often implicit or doc-driven | Expose shared capability metadata in code |
| Compatibility story | Existing callers rely on current interfaces | Native SDKs do not need to preserve Chroma Go compatibility | Keep old APIs stable while adding richer shared foundations |

## Sources

- `.planning/codebase/CONCERNS.md` — current contract fragmentation and missing provider-neutral multimodal foundation
- `pkg/embeddings/embedding.go` — present shared interfaces and image-only multimodal shape
- `pkg/embeddings/registry.go` — current dense/sparse/multimodal registration model
- `pkg/embeddings/roboflow/roboflow.go` — current text+image multimodal implementation
- `docs/docs/embeddings.md` — existing provider docs and task-specific options
- GitHub issue `#442` — target feature set and acceptance criteria

---
*Feature research for: brownfield Go SDK multimodal foundations*
*Researched: 2026-03-18*
