# Roadmap: Chroma Go

## Overview

This roadmap initializes GSD planning for the current brownfield milestone focused on provider-neutral multimodal embedding foundations. The work is sequenced to stabilize additive shared types first, then expose capabilities and preserve compatibility, then wire richer multimodal support through config and registry flows, and only then lock down provider mapping behavior, docs, and verification.

## Milestones

- 🚧 **v0.4.1 Provider-Neutral Multimodal Foundations** - Phases 1-7 (current planning milestone)

## v0.4.1 Provider-Neutral Multimodal Foundations

**Milestone Goal:** Add provider-neutral multimodal embedding foundations that support richer modalities and portable intents while preserving existing text-only and image-only APIs, then validate with Gemini and vLLM/Nemotron provider adoptions.

## Phases

- [x] **Phase 1: Shared Multimodal Contract** - Add additive request and part types, neutral intents, per-request options, and validation primitives. (completed 2026-03-18)
- [x] **Phase 2: Capability Metadata and Compatibility** - Expose provider capabilities and keep legacy callers working unchanged. (completed 2026-03-19)
- [x] **Phase 3: Registry and Config Integration** - Extend registry/build-from-config and collection auto-wiring for richer multimodal interfaces. (completed 2026-03-20)
- [ ] **Phase 4: Provider Mapping and Explicit Failures** - Define neutral intent mapping and surface unsupported combinations explicitly.
- [ ] **Phase 5: Documentation and Verification** - Update docs, examples, and tests around portable multimodal usage and compatibility.
- [ ] **Phase 6: Gemini Multimodal Adoption** - Wire Gemini into the shared multimodal contract with full modality support. (issue #443)
- [ ] **Phase 7: vLLM/Nemotron Provider Validation** - Add vLLM OpenAI-compatible provider targeting nvidia/omni-embed-nemotron-3b to validate the foundation end-to-end.

## Phase Details

### Phase 1: Shared Multimodal Contract
**Goal:** Introduce additive shared multimodal types that can represent ordered mixed-part requests, neutral intents, per-request options, and explicit validation results.
**Depends on**: Nothing (first phase)
**Requirements**: [MMOD-01, MMOD-02, MMOD-03, MMOD-04, MMOD-05]
**Success Criteria** (what must be TRUE):
  1. Callers can construct a validated multimodal request using text, image, audio, video, or PDF parts.
  2. Mixed-part request ordering is preserved in the shared API surface.
  3. Per-request intent, dimensionality, and provider-hint fields are represented without mutating provider-wide config.
  4. Invalid request shapes fail before provider I/O with clear errors.
**Plans**: 4/4 plans executed

Plans:
- [x] 01-00: Add Wave 0 multimodal test scaffolding and Nyquist verification targets
- [x] 01-01: Define additive multimodal request, part, intent, and option types in `pkg/embeddings`
- [x] 01-02: Implement validation helpers and compatibility-safe constructors
- [x] 01-03: Add unit tests for request construction, ordering, and validation

### Phase 2: Capability Metadata and Compatibility
**Goal:** Add capability introspection and compatibility adapters so existing text-only and image-only callers continue to work while new multimodal APIs become available.
**Depends on**: Phase 1
**Requirements**: [CAPS-01, CAPS-02, COMP-01, COMP-02]
**Success Criteria** (what must be TRUE):
  1. Providers can expose supported modalities, intents, and option support through shared capability metadata.
  2. Callers can inspect capability metadata without type-asserting provider implementations.
  3. Existing `EmbeddingFunction` and image-only `MultimodalEmbeddingFunction` callers continue to compile and pass compatibility tests.
**Plans**: 3/3 plans executed

Plans:
- [x] 02-01: Add shared capability metadata types and interfaces
- [x] 02-02: Introduce compatibility adapters or delegation paths between legacy and richer multimodal contracts
- [x] 02-03: Add regression tests for text-only and image-only callers

### Phase 3: Registry and Config Integration
**Goal:** Extend registry and config-persistence flows so richer multimodal functions can be rebuilt from stored configuration without regressing existing auto-wiring.
**Depends on**: Phase 2
**Requirements**: [REG-01, REG-02]
**Success Criteria** (what must be TRUE):
  1. Factory and registry paths can build richer multimodal implementations from additive shared interfaces.
  2. Existing dense and multimodal config round-trips remain stable.
  3. Collection configuration auto-wiring continues to work for existing providers after the new types are introduced.
**Plans**: 3 plans

Plans:
- [ ] 03-01-PLAN.md — Add 4th content factory map to registry with fallback chain and inferCaps
- [ ] 03-02-PLAN.md — Extend config build chain, collection contentEF field, and auto-wiring
- [ ] 03-03-PLAN.md — Add config round-trip, build chain, and auto-wiring tests

### Phase 4: Provider Mapping and Explicit Failures
**Goal:** Define how provider-neutral intents and modalities map to provider-native semantics and fail clearly when a provider cannot support the request.
**Depends on**: Phase 3
**Requirements**: [MAP-01, MAP-02]
**Success Criteria** (what must be TRUE):
  1. The shared contract defines a neutral intent-to-provider mapping strategy with test coverage.
  2. Current multimodal providers can advertise what they support and reject unsupported combinations explicitly.
  3. No request silently degrades from a requested modality or intent to a different provider behavior.
**Plans**: 2 plans

Plans:
- [ ] 04-01: Implement provider mapping helpers and explicit unsupported error paths
- [ ] 04-02: Adapt current multimodal provider behavior and add mapping tests

### Phase 5: Documentation and Verification
**Goal:** Document the portable multimodal API and verify the foundation through docs, examples, and focused tests before follow-on provider adoption.
**Depends on**: Phase 4
**Requirements**: [DOCS-01, DOCS-02]
**Success Criteria** (what must be TRUE):
  1. Docs explain portable intents, provider-specific escape hatches, and compatibility expectations.
  2. Tests cover validation, compatibility, registry/config persistence, and unsupported-combination failures.
  3. Example guidance reflects current multimodal support and the new shared foundations.
**Plans**: 3 plans

Plans:
- [ ] 05-01: Update docs and migration guidance for multimodal embeddings
- [ ] 05-02: Add focused examples or snippets for portable multimodal requests
- [ ] 05-03: Audit and extend tests for acceptance-criteria coverage

### Phase 6: Gemini Multimodal Adoption
**Goal:** Wire Gemini into the shared multimodal contract so it supports text, image, audio, video, and PDF embeddings through the portable interface while keeping existing text-only APIs as backward-compatible wrappers.
**Depends on**: Phase 5
**Requirements**: [GEM-01, GEM-02, GEM-03]
**Issue**: #443
**Success Criteria** (what must be TRUE):
  1. Gemini implements `SharedContentEmbeddingFunction` and `CapabilityAware` interfaces for text, image, audio, video, and PDF modalities.
  2. Neutral intents map to Gemini task types with explicit errors for unsupported combinations.
  3. Existing `EmbedDocuments`/`EmbedQuery` behavior remains unchanged.
  4. Gemini is registered in the multimodal factory/registry path with config round-trip support.
  5. Unit tests cover request construction, intent mapping, and backward-compatible wrappers.

Plans: TBD during planning

### Phase 7: vLLM/Nemotron Provider Validation
**Goal:** Add a vLLM OpenAI-compatible embedding provider targeting nvidia/omni-embed-nemotron-3b to validate the shared multimodal contract against a second real multimodal model beyond Gemini.
**Depends on**: Phase 6
**Requirements**: [VLLM-01, VLLM-02]
**Success Criteria** (what must be TRUE):
  1. A vLLM/OpenAI-compatible provider implements `SharedContentEmbeddingFunction` and `CapabilityAware` for the modalities supported by omni-embed-nemotron-3b.
  2. The provider works against a live vLLM API endpoint with multimodal inputs.
  3. The shared contract, registry, and intent mapping work without provider-specific hacks — validating the foundation is truly portable.
  4. Integration tests cover at least text + image multimodal embedding through the vLLM endpoint.

Plans: TBD during planning

## Progress

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Shared Multimodal Contract | 4/4 | Complete | 2026-03-18 |
| 2. Capability Metadata and Compatibility | 3/3 | Complete | 2026-03-19 |
| 3. Registry and Config Integration | 3/3 | Complete   | 2026-03-20 |
| 4. Provider Mapping and Explicit Failures | 0/2 | Not started | - |
| 5. Documentation and Verification | 0/3 | Not started | - |
| 6. Gemini Multimodal Adoption | - | Not started | - |
| 7. vLLM/Nemotron Provider Validation | - | Not started | - |
