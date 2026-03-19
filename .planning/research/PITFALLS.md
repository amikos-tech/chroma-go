# Pitfalls Research

**Domain:** Brownfield Go SDK milestone for provider-neutral multimodal embeddings
**Researched:** 2026-03-18
**Confidence:** HIGH

## Critical Pitfalls

### Pitfall 1: Breaking Existing Callers While “Cleaning Up” the Interface

**What goes wrong:** existing text-only or image-only consumers stop compiling or change behavior after the richer multimodal contract lands.  
**Why it happens:** developers try to replace the old API with the new one instead of making it additive.  
**How to avoid:** keep legacy interfaces stable, add adapters, and write explicit compatibility regression tests.  
**Warning signs:** type signatures change in public interfaces, examples need rewrites for existing flows, or current provider tests start failing for unchanged callers.  
**Phase to address:** Phase 2

---

### Pitfall 2: Neutral Intents That Do Not Map Cleanly to Providers

**What goes wrong:** a neutral intent seems portable in theory but means different things across providers, leading to ambiguous or surprising behavior.  
**Why it happens:** shared terminology is chosen before capability and mapping constraints are made explicit.  
**How to avoid:** define a small neutral intent set, require provider-specific mapping code, and fail unsupported combinations explicitly.  
**Warning signs:** mapping logic starts guessing, provider adapters special-case the same intent repeatedly, or docs need caveats like “usually means.”  
**Phase to address:** Phase 4

---

### Pitfall 3: Config Round-Trip Drift

**What goes wrong:** richer multimodal functions work when created directly but fail when rebuilt from stored collection configuration.  
**Why it happens:** shared types evolve without keeping `GetConfig()`, registry builders, and `BuildEmbeddingFunctionFromConfig` aligned.  
**How to avoid:** update persistence and reconstruction tests alongside contract changes, and keep new config keys additive.  
**Warning signs:** provider constructors gain new required options that never appear in config, or collection auto-wiring fails after contract changes.  
**Phase to address:** Phase 3

---

### Pitfall 4: Overfitting the Foundation to the First Provider

**What goes wrong:** the shared multimodal model mostly mirrors Roboflow or another first adopter instead of remaining provider-neutral.  
**Why it happens:** the first working provider shapes the abstraction more than the long-term portability goals do.  
**How to avoid:** separate portable request semantics from provider-specific escape hatches and validate the contract against multiple provider patterns in tests and docs.  
**Warning signs:** shared types mention provider-native fields by name or capability metadata only makes sense for one provider.  
**Phase to address:** Phase 1

---

### Pitfall 5: Docs Drift Behind the New Contract

**What goes wrong:** the code supports richer multimodal behavior, but docs still describe outdated limitations or omit compatibility guidance.  
**Why it happens:** shared-contract work lands without bundling public documentation updates.  
**How to avoid:** treat docs/examples as part of the milestone acceptance criteria and update them in the final phase before calling the work done.  
**Warning signs:** old docs still say multimodal is unsupported, or new shared types have no user-facing guidance.  
**Phase to address:** Phase 5

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Hard-code provider-specific task strings in shared helpers | Faster first implementation | Shared API portability erodes quickly | Never for the primary contract |
| Skip capability metadata and let providers fail deep in request execution | Less up-front design work | Errors become late, inconsistent, and hard to explain | Only for temporary internal spikes, not shipped code |
| Add config keys without round-trip tests | Faster merge | Auto-wiring regressions surface later and are hard to diagnose | Never for shipped provider/config changes |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Provider config builders | Require new fields but do not persist them in `GetConfig()` | Keep config additive and test direct-construction vs rebuild parity |
| Collection auto-wiring | Assume only dense builders matter | Extend multimodal builder paths and verify existing dense paths still work |
| Docs/examples | Update provider code but not docs pages | Ship docs/examples in the same milestone phase as the API change |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Per-item fallback loops remain hidden in multimodal flows | Large mixed batches are slow or provider rate limits spike | Keep batching semantics explicit and document when a provider is sequential | Noticeable once callers batch many inputs |
| Excessively large provider-specific hint payloads | Requests become hard to validate and persist | Keep a small portable core plus clearly scoped escape hatch fields | As soon as multiple providers add custom knobs |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Persist provider secrets directly in config | Secret leakage in collection config or logs | Continue storing env-var names rather than secret values |
| Allow insecure provider transport silently | Credential exposure in non-HTTPS flows | Keep secure defaults and document any insecure escape hatch clearly |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Silent fallback on unsupported modality or intent | Users cannot trust what was actually embedded | Return explicit capability-aware errors |
| Portable API without guidance on escape hatches | Users either avoid the feature or overfit to one provider | Document the portable path first, then explain provider-specific hints |

## "Looks Done But Isn't" Checklist

- [ ] **Shared request model:** Supports ordered mixed parts and validates malformed combinations
- [ ] **Compatibility story:** Existing text-only and image-only callers still compile and pass regression tests
- [ ] **Config integration:** Direct construction and build-from-config behave the same
- [ ] **Provider mapping:** Unsupported combinations fail explicitly instead of guessing
- [ ] **Docs:** Portable intent usage and provider-specific hints are both documented

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Broken legacy callers | HIGH | Reintroduce additive adapters, restore prior signatures, and add missing compatibility tests |
| Bad neutral-intent mapping | MEDIUM | Narrow the intent set, add capability metadata, and codify explicit unsupported errors |
| Config drift | MEDIUM | Patch persistence keys, extend round-trip coverage, and verify collection auto-wiring |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Breaking existing callers | Phase 2 | Compatibility tests for current dense and image-only multimodal APIs pass |
| Bad neutral-intent mapping | Phase 4 | Shared mapping tests and unsupported-error cases are explicit |
| Config round-trip drift | Phase 3 | Rebuild-from-config and collection auto-wiring tests pass |
| Overfitting to first provider | Phase 1 | Shared types stay provider-neutral and escape hatches remain additive |
| Docs drift | Phase 5 | Docs/examples reflect the final public contract and compatibility guidance |

## Sources

- `.planning/codebase/CONCERNS.md` — known multimodal contract fragmentation and docs drift
- `pkg/embeddings/embedding.go` — current public interface constraints
- `pkg/embeddings/registry.go` — current builder entry points
- `pkg/embeddings/roboflow/roboflow.go` — current provider-specific multimodal behavior
- `docs/docs/embeddings.md` and `docs/go-examples/docs/embeddings/multimodal.md` — current documentation state
- GitHub issue `#442` — target failures and acceptance criteria to guard

---
*Pitfalls research for: brownfield Go SDK multimodal foundations*
*Researched: 2026-03-18*
