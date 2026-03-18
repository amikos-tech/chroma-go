# Architecture Research

**Domain:** Brownfield Go SDK milestone for provider-neutral multimodal embeddings
**Researched:** 2026-03-18
**Confidence:** HIGH

## Standard Architecture

### System Overview

```text
┌────────────────────────────────────────────────────────────────────┐
│                    Public SDK Layer (`pkg/api/v2`)                │
├────────────────────────────────────────────────────────────────────┤
│ Client / Collection APIs │ Schema / Config │ Query / Search APIs │
└───────────────┬────────────────────────────┬──────────────────────┘
                │                            │
┌───────────────▼────────────────────────────▼──────────────────────┐
│              Shared Embedding Contract Layer (`pkg/embeddings`)   │
├────────────────────────────────────────────────────────────────────┤
│ Dense EF │ Sparse EF │ Rich multimodal types │ Capabilities │      │
│ Intent mapping helpers │ Validation │ Registry contracts            │
└───────────────┬────────────────────────────┬──────────────────────┘
                │                            │
┌───────────────▼────────────────────────────▼──────────────────────┐
│                   Provider Implementation Layer                   │
├────────────────────────────────────────────────────────────────────┤
│ roboflow │ openai │ jina │ gemini │ voyage │ ...                  │
│ provider config builders │ provider-native task mapping            │
└───────────────┬────────────────────────────┬──────────────────────┘
                │                            │
┌───────────────▼────────────────────────────▼──────────────────────┐
│                Docs, Examples, and Test Coverage                  │
├────────────────────────────────────────────────────────────────────┤
│ docs/docs/embeddings.md │ *_test.go │ config round-trip tests      │
└────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| Shared multimodal types | Represent ordered modality parts, intents, options, and validation | `pkg/embeddings/embedding.go` plus helper types |
| Capability metadata | Describe supported modalities, intents, and request options per provider | Additive shared interfaces or structs returned by providers |
| Provider mapping layer | Translate neutral intents and request shapes into provider-native semantics | Shared helpers plus provider-specific adapters |
| Registry/config layer | Persist and rebuild multimodal functions from config safely | `pkg/embeddings/registry.go` and `pkg/api/v2/configuration.go` |
| Compatibility adapters | Preserve existing text-only and image-only APIs | Adapters or wrapper methods that delegate into the richer contract |

## Recommended Project Structure

```text
pkg/
├── api/v2/
│   └── configuration.go      # auto-wiring and build-from-config integration
├── embeddings/
│   ├── embedding.go          # shared contracts, multimodal types, validation
│   ├── registry.go           # factory and capability-aware builder hooks
│   ├── registry_test.go      # shared registry tests
│   ├── persistence_test.go   # config round-trip coverage
│   └── roboflow/             # first multimodal provider adaptation target
docs/
└── docs/embeddings.md        # portable intent and compatibility guidance
```

### Structure Rationale

- **`pkg/embeddings/`**: the shared contract already lives here, so richer multimodal foundations should stay close to existing interfaces and registry code
- **`pkg/api/v2/configuration.go`**: collection auto-wiring rebuilds embedding functions here, so new config keys must integrate instead of side-stepping this file
- **Provider packages**: provider-native mapping belongs with provider code, but only after the shared neutral contract is established
- **Docs/tests**: public behavior changes must land with docs and regression coverage, not as a follow-up

## Architectural Patterns

### Pattern 1: Additive Shared Contract

**What:** extend the existing shared embedding abstractions with new multimodal request, intent, option, and capability types instead of replacing current interfaces.  
**When to use:** whenever a new capability can be expressed without breaking current callers.  
**Trade-offs:** compatibility is preserved, but the shared surface area becomes broader and needs disciplined documentation.

### Pattern 2: Provider Capability Gate

**What:** check declared provider capabilities before turning a neutral multimodal request into provider-native API calls.  
**When to use:** before mapping modality combinations, intents, or per-request options.  
**Trade-offs:** more metadata to maintain, but explicit unsupported errors are much safer than silent fallback behavior.

### Pattern 3: Compatibility Adapter

**What:** keep legacy `EmbedDocuments`, `EmbedQuery`, `EmbedImage`, and `EmbedImages` paths working by translating them into the richer request model internally.  
**When to use:** when introducing new shared behavior into an existing public SDK.  
**Trade-offs:** some duplication remains temporarily, but migration risk drops materially.

## Data Flow

### Request Flow

```text
[Caller]
    ↓
[Shared multimodal request]
    ↓ validate
[Capability check]
    ↓ map neutral intent/options
[Provider adapter]
    ↓
[Provider HTTP/API call]
    ↓
[Embedding result(s)]
```

### State Management

```text
[Provider config map]
    ↓ persist
[Collection configuration]
    ↓ rebuild
[Registry/factory]
    ↓
[Concrete embedding function]
```

### Key Data Flows

1. **Portable request execution:** caller constructs a rich multimodal request, validation and capability checks run, and provider adapters map it into provider-native calls.
2. **Compatibility execution:** legacy text-only or image-only methods delegate into the richer contract so behavior stays stable while shared semantics improve.
3. **Config reconstruction:** persisted config selects the correct factory path and rebuilds a richer multimodal implementation without leaking secrets.

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| Current provider surface | Keep additive shared types small, explicit, and well-tested |
| More multimodal providers | Centralize neutral intent and capability semantics to avoid per-provider drift |
| Large provider matrix | Invest in shared conformance-style tests for capability declarations and unsupported-error behavior |

### Scaling Priorities

1. **First bottleneck:** semantic drift between shared intents and provider-native tasks — fix with explicit mapping helpers and shared tests
2. **Second bottleneck:** config and docs drift as more providers adopt the contract — fix with round-trip tests and docs updates per provider

## Anti-Patterns

### Anti-Pattern 1: Provider Terms in the Shared API

**What people do:** expose raw provider task strings as the primary cross-provider interface.  
**Why it's wrong:** the strings differ by provider and force callers to learn implementation details.  
**Do this instead:** use neutral intents in shared types and keep provider strings behind adapters or escape hatches.

### Anti-Pattern 2: Parallel Multimodal Stack Outside Existing Config Paths

**What people do:** add a new request path that never participates in registry or collection config reconstruction.  
**Why it's wrong:** it breaks auto-wiring expectations and creates two incompatible ways to configure embeddings.  
**Do this instead:** extend existing config and registry flows additively.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| Provider APIs | Provider package adapters | Neutral intents/options should be translated as close to provider code as possible |
| Chroma collection config | Existing `EmbeddingFunctionConfig` persistence | New multimodal keys must remain serializable and safe to persist |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| `pkg/api/v2` ↔ `pkg/embeddings` | direct interfaces and config maps | Keep the richer contract discoverable without leaking provider-specific details upward |
| `pkg/embeddings` shared ↔ provider packages | interfaces, helper types, config builders | Shared types should own portability; provider packages should own mapping specifics |
| code ↔ docs/tests | examples, docs pages, regression tests | Public contract changes are incomplete until these artifacts move together |

## Sources

- `.planning/codebase/ARCHITECTURE.md` — existing layering and data flow
- `.planning/codebase/STRUCTURE.md` — where new shared and provider changes belong
- `pkg/embeddings/embedding.go` — current shared embedding abstractions
- `pkg/embeddings/registry.go` — current builder and registry structure
- `pkg/embeddings/roboflow/roboflow.go` — existing multimodal provider implementation
- `pkg/api/v2/configuration.go` — current embedding reconstruction flow
- GitHub issue `#442` — milestone architecture goals

---
*Architecture research for: brownfield Go SDK multimodal foundations*
*Researched: 2026-03-18*
