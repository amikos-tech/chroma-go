# Project Research Summary

**Project:** Chroma Go
**Domain:** Brownfield Go SDK milestone for provider-neutral multimodal embeddings
**Researched:** 2026-03-18
**Confidence:** HIGH

## Executive Summary

This milestone is best approached as an additive shared-contract expansion, not a rewrite. The repo already has strong seams for this work: shared embedding interfaces in `pkg/embeddings`, registry/build-from-config flows in `pkg/embeddings/registry.go` and `pkg/api/v2/configuration.go`, one current multimodal provider (`roboflow`), and good existing test patterns for config round-trips and provider behavior. The core recommendation is to stabilize a portable multimodal request model first, then layer in capability metadata, registry/config integration, provider mapping, and finally docs/tests.

The main risk is semantic drift. If provider-native task names or first-provider behaviors leak into the shared interface, the new foundation will not stay provider-neutral. The mitigation is to keep the neutral intent set small, capability-aware, and explicitly mapped to providers with clear unsupported-combination errors. The second major risk is regression in legacy callers and config auto-wiring; additive compatibility adapters and round-trip tests should be treated as first-class deliverables, not cleanup.

## Key Findings

### Recommended Stack

No new framework is needed. The recommended stack is the current repo stack: Go 1.24.11, shared contract work in `pkg/embeddings`, config and auto-wiring updates in `pkg/api/v2/configuration.go`, and regression coverage in existing `*_test.go` suites. This keeps the milestone aligned with current architecture and avoids parallel abstractions that would increase maintenance cost.

**Core technologies:**
- Go 1.24.11: implement the shared types, capability metadata, and adapters in the existing language and toolchain
- `pkg/embeddings`: own portable request types, validation, registry contracts, and compatibility shims
- `pkg/api/v2/configuration.go`: preserve collection configuration and rebuild-from-config behavior
- `testify` plus repo test patterns: verify compatibility, config round-trips, and unsupported-combination errors

### Expected Features

The must-have set is narrow and structural: richer multimodal requests, provider-neutral intents, capability introspection, explicit unsupported-combination failures, registry/config integration, and docs that explain portable usage vs provider hints.

**Must have (table stakes):**
- Rich multimodal request model — users need a portable way to express mixed-modality inputs
- Neutral intent semantics — users should not need provider-native task names to use the shared API
- Capability metadata — callers need to discover supported modalities and intents safely
- Backward compatibility — existing text-only and image-only code must keep working
- Config/registry support — richer multimodal functions must still rebuild from stored configuration

**Should have (competitive):**
- Mixed-part request support — positions the library for future provider adoption beyond image-only multimodal flows
- Portable docs and examples — makes the new contract adoptable instead of code-only

**Defer (v2+):**
- Full provider migration across the entire embedding provider matrix
- Remote or dynamic provider capability discovery

### Architecture Approach

Use the existing layer boundaries: callers enter through shared request and compatibility APIs, shared validation and capability metadata sit in `pkg/embeddings`, providers map neutral intents to native semantics within their packages, and configuration/registry code rebuilds implementations from persisted config. This ordering keeps the milestone additive and compatible with current auto-wiring behavior.

**Major components:**
1. Shared multimodal request and option types — represent portable user intent
2. Capability metadata and compatibility adapters — preserve legacy callers while enabling richer APIs
3. Registry/config integration — rebuild richer multimodal functions from stored config
4. Provider mapping helpers and explicit unsupported errors — keep the portable contract honest
5. Docs and tests — verify and explain the final public behavior

### Critical Pitfalls

1. **Breaking existing callers** — avoid by keeping legacy interfaces stable and regression-tested
2. **Neutral intents that cannot map cleanly** — avoid by keeping the intent set small and capability-aware
3. **Config round-trip drift** — avoid by updating persistence and auto-wiring tests with the contract change
4. **Overfitting to the first provider** — avoid by separating portable semantics from provider-specific escape hatches
5. **Docs drift** — avoid by treating docs/examples as acceptance criteria, not polish

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Shared Multimodal Contract
**Rationale:** The rest of the milestone depends on a stable additive request, intent, option, and validation model.  
**Delivers:** Rich shared multimodal types and validation helpers.  
**Addresses:** Portable request shape and explicit invalid-request failures.  
**Avoids:** Overfitting later phases to provider-specific APIs.

### Phase 2: Capability Metadata and Compatibility
**Rationale:** Once the shared request exists, the next highest-risk work is preserving existing callers and exposing what providers can actually support.  
**Delivers:** Capability metadata and compatibility adapters.  
**Uses:** Shared contract from Phase 1.  
**Implements:** The public compatibility story required by issue `#442`.

### Phase 3: Registry and Config Integration
**Rationale:** Shared interfaces are not shippable until persisted collection config can rebuild them.  
**Delivers:** Updated registry and build-from-config paths with regression tests.  
**Uses:** Capability and compatibility behavior from earlier phases.  
**Implements:** Collection auto-wiring continuity.

### Phase 4: Provider Mapping and Explicit Failures
**Rationale:** The foundation is only trustworthy once neutral intents and modalities map cleanly to provider-native semantics or fail explicitly.  
**Delivers:** Mapping helpers, unsupported-combination errors, and current-provider adaptations.

### Phase 5: Documentation and Verification
**Rationale:** Public adoption requires guidance and proof that the compatibility and failure behavior actually holds.  
**Delivers:** Updated docs/examples plus focused acceptance-criteria coverage.

### Phase Ordering Rationale

- Shared semantics must stabilize before persistence and provider mapping can be correct
- Compatibility should land early because it constrains every later implementation choice
- Config integration comes before provider rollout so richer implementations remain reconstructable
- Docs and verification belong last because they should describe the final contract, not a draft

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 4:** provider-native intent mapping details may vary more than the shared contract suggests
- **Phase 5:** user-facing docs should reconcile current docs drift around multimodal support

Phases with standard patterns (skip research-phase if needed):
- **Phase 1:** additive shared types and validation follow established repo patterns
- **Phase 2:** compatibility adapters and shared capability metadata are primarily internal design work

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Driven by current repo architecture and toolchain, not speculative new dependencies |
| Features | HIGH | Directly grounded in issue `#442` and current contract gaps |
| Architecture | HIGH | Existing codebase map and source files make the integration points clear |
| Pitfalls | HIGH | Risks are visible in current docs, registry, and provider abstractions |

**Overall confidence:** HIGH

### Gaps to Address

- Provider-specific intent mapping details may need refinement once Phase 4 planning reads the concrete provider implementations more deeply
- The `v0.5` milestone label is a planning placeholder and may need renaming if maintainers choose a different release line

## Sources

### Primary (HIGH confidence)
- `.planning/codebase/ARCHITECTURE.md` — current layering and data flow
- `.planning/codebase/CONCERNS.md` — existing multimodal contract gaps and risks
- `pkg/embeddings/embedding.go` — current shared embedding abstractions
- `pkg/embeddings/registry.go` — current builder structure
- `pkg/api/v2/configuration.go` — collection config and auto-wiring path
- GitHub issue `#442` — milestone scope and acceptance criteria

### Secondary (MEDIUM confidence)
- `docs/docs/embeddings.md` — existing user-facing provider guidance and multimodal docs
- `pkg/embeddings/roboflow/roboflow.go` — current multimodal provider behavior

### Tertiary (LOW confidence)
- None

---
*Research completed: 2026-03-18*
*Ready for roadmap: yes*
