# Roadmap: Chroma Go

## Overview

This roadmap initializes GSD planning for the current brownfield milestone focused on provider-neutral multimodal embedding foundations. The work is sequenced to stabilize additive shared types first, then expose capabilities and preserve compatibility, then wire richer multimodal support through config and registry flows, and only then lock down provider mapping behavior, docs, and verification.

## Milestones

- 🚧 **v0.4.1 Provider-Neutral Multimodal Foundations** - Phases 1-18 (current planning milestone)

## v0.4.1 Provider-Neutral Multimodal Foundations

**Milestone Goal:** Add provider-neutral multimodal embedding foundations that support richer modalities and portable intents while preserving existing text-only and image-only APIs, then validate with Gemini and VoyageAI provider adoptions.

## Phases

- [x] **Phase 1: Shared Multimodal Contract** - Add additive request and part types, neutral intents, per-request options, and validation primitives. (completed 2026-03-18)
- [x] **Phase 2: Capability Metadata and Compatibility** - Expose provider capabilities and keep legacy callers working unchanged. (completed 2026-03-19)
- [x] **Phase 3: Registry and Config Integration** - Extend registry/build-from-config and collection auto-wiring for richer multimodal interfaces. (completed 2026-03-20)
- [x] **Phase 4: Provider Mapping and Explicit Failures** - Define neutral intent mapping and surface unsupported combinations explicitly. (completed 2026-03-20)
- [x] **Phase 5: Documentation and Verification** - Update docs, examples, and tests around portable multimodal usage and compatibility. (completed 2026-03-20)
- [x] **Phase 6: Gemini Multimodal Adoption** - Wire Gemini into the shared multimodal contract with full modality support. (issue #443) (completed 2026-03-20)
- [x] **Phase 7: Voyage Multimodal Adoption** - Wire VoyageAI into the shared multimodal contract with text, image, and video support to validate the foundation end-to-end.
- [x] **Phase 8: Document Gemini and VoyageAI multimodal embedding functions** - Update provider docs, add runnable examples, update README, create changelog. (completed 2026-03-23)
- [ ] **Phase 9: Convenience Constructors and Documentation Polish** - Add shorthand constructors to reduce Content API verbosity and update docs.
- [x] **Phase 10: Code Cleanups** - Extract shared path safety utilities, fix *context.Context anti-pattern, add registry test cleanup, fix resolveMIME for URL-backed sources. (issues #456, #461, #466, #469) (completed 2026-03-26)
- [x] **Phase 11: Fork Double-Close Bug** - Fix EF pointer sharing in Fork() that causes double-close on client.Close(). (issue #454) (completed 2026-03-26)
- [x] **Phase 12: SDK Auto-Wiring Research** - Trace contentEmbeddingFunction auto-wiring behavior in official Chroma SDKs. (issue #455) (completed 2026-03-28)
- [x] **Phase 13: Collection.ForkCount** - Add ForkCount endpoint support for upstream /fork_count API. (issue #460) (completed 2026-03-28)
- [x] **Phase 14: Delete with Limit** - Add delete-with-limit support for upstream limit parameter. (issue #439) [1/2 plans complete] (completed 2026-03-29)
- [x] **Phase 15: OpenRouter Embeddings Compatibility** - Add first-class OpenRouter support via provider preferences and encoding_format. (issue #438) (completed 2026-03-30)
- [ ] **Phase 16: Twelve Labs Embedding Function** - Add Twelve Labs multimodal embedding provider. (issue #190)
- [ ] **Phase 17: Cloud RRF and GroupBy Test Coverage** - Add cloud integration tests for Search API RRF and GroupBy primitives. (issue #462)
- [ ] **Phase 18: Embedded Client contentEmbeddingFunction Parity** - Add contentEmbeddingFunction support to embeddedCollection for feature parity with HTTP client. (issue #472)

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
- [ ] 04-01-PLAN.md — Add IntentMapper interface, IsNeutralIntent helper, ValidateContentSupport pre-check, and 3 new validation codes
- [ ] 04-02-PLAN.md — Add mapping and validation test coverage for IntentMapper contract and pre-check helper

### Phase 5: Documentation and Verification
**Goal:** Document the portable multimodal API and verify the foundation through docs, examples, and focused tests before follow-on provider adoption.
**Depends on**: Phase 4
**Requirements**: [DOCS-01, DOCS-02]
**Success Criteria** (what must be TRUE):
  1. Docs explain portable intents, provider-specific escape hatches, and compatibility expectations.
  2. Tests cover validation, compatibility, registry/config persistence, and unsupported-combination failures.
  3. Example guidance reflects current multimodal support and the new shared foundations.
**Plans**: 2 plans

Plans:
- [ ] 05-01-PLAN.md — Rewrite multimodal.md as Go Content API page and add cross-link from embeddings.md
- [ ] 05-02-PLAN.md — Audit DOCS-02 test coverage and add registry round-trip EmbedContent dispatch test

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
**Plans**: 2 plans

Plans:
- [x] 06-01-PLAN.md — Implement content helpers, interface methods, CreateContentEmbedding, registration, and default model update
- [x] 06-02-PLAN.md — Add unit tests for capability derivation, intent mapping, MIME resolution, content conversion, negative cases, and config round-trip

### Phase 7: Voyage Multimodal Adoption
**Goal:** Wire VoyageAI into the shared multimodal contract so it supports text, image, and video embeddings through the portable interface, validating the foundation against a second real multimodal provider beyond Gemini.
**Depends on**: Phase 6
**Requirements**: [VOY-01, VOY-02, VOY-03]
**Success Criteria** (what must be TRUE):
  1. Voyage implements `ContentEmbeddingFunction`, `CapabilityAware`, and `IntentMapper` for text, image, and video modalities.
  2. Neutral intents map to Voyage input types with explicit errors for unsupported combinations.
  3. Existing `EmbedDocuments`/`EmbedQuery` behavior remains unchanged.
  4. Voyage is registered in the multimodal factory/registry path with config round-trip support.
  5. The shared contract, registry, and intent mapping work without provider-specific hacks — validating the foundation is truly portable.
**Plans**: 2 plans

Plans:
- [x] 07-01-PLAN.md — Implement content.go with multimodal types, conversion helpers, capabilities, intent mapping, and wire interface implementations + registration into voyage.go
- [x] 07-02-PLAN.md — Add unit tests for capability derivation, intent mapping, content conversion, batch rejection, config round-trip, and registration

### Phase 8: Document Gemini and VoyageAI multimodal embedding functions
**Goal:** Update provider-specific documentation for Gemini and VoyageAI to show Content API multimodal usage, add runnable examples, update README and changelog to close the v0.4.1 milestone.
**Depends on:** Phase 7
**Requirements**: [D-01, D-02, D-03, D-04, D-05, D-06, D-07, D-08, D-09, D-10, D-11]
**Success Criteria** (what must be TRUE):
  1. Gemini and VoyageAI sections in embeddings.md have "Multimodal (Content API)" subsections with EmbedContent examples.
  2. Gemini default model references updated to gemini-embedding-2-preview throughout docs.
  3. VoyageAI section lists all available option functions.
  4. Runnable multimodal examples exist for both Gemini and VoyageAI.
  5. README mentions multimodal Content API capabilities and lists new examples.
  6. CHANGELOG.md documents v0.4.1 release.
  7. ROADMAP.md references VoyageAI consistently throughout all phase headings and descriptions.
**Plans**: 2 plans

Plans:
- [ ] 08-01-PLAN.md — Update embeddings.md provider sections and add runnable multimodal examples
- [ ] 08-02-PLAN.md — Update README, create CHANGELOG, correct ROADMAP naming

## Progress

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Shared Multimodal Contract | 4/4 | Complete | 2026-03-18 |
| 2. Capability Metadata and Compatibility | 3/3 | Complete | 2026-03-19 |
| 3. Registry and Config Integration | 3/3 | Complete   | 2026-03-20 |
| 4. Provider Mapping and Explicit Failures | 2/2 | Complete   | 2026-03-20 |
| 5. Documentation and Verification | 2/2 | Complete   | 2026-03-20 |
| 6. Gemini Multimodal Adoption | 2/2 | Complete   | 2026-03-20 |
| 7. Voyage Multimodal Adoption | 2/2 | Complete | 2026-03-22 |
| 8. Document Gemini and VoyageAI | 2/2 | Complete | 2026-03-23 |
| 9. Convenience Constructors | 2/2 | Complete | - |
| 10. Code Cleanups | 2/2 | Complete    | 2026-03-26 |
| 11. Fork Double-Close Bug | 2/2 | Complete    | 2026-03-26 |
| 12. SDK Auto-Wiring Research | 1/1 | Complete    | 2026-03-28 |
| 13. Collection.ForkCount | 2/2 | Complete    | 2026-03-28 |
| 14. Delete with Limit | 2/2 | Complete    | 2026-03-29 |
| 15. OpenRouter Embeddings | 2/2 | Complete    | 2026-03-30 |
| 16. Twelve Labs EF | 1/2 | In progress | - |
| 17. Cloud RRF/GroupBy Tests | 0/0 | Not started | - |
| 18. Embedded contentEF Parity | 0/0 | Not started | - |

### Phase 9: Convenience Constructors and Documentation Polish

**Goal:** Add shorthand constructors (NewImageURL, NewImageFile, NewVideoURL, etc.) to reduce Content API verbosity, update multimodal docs and examples to use them, and verify the simplified surface end-to-end.
**Requirements**: [CONV-01, CONV-02, CONV-03, CONV-04]
**Depends on:** Phase 8
**Success Criteria** (what must be TRUE):
  1. Convenience constructors exist for common modality+source combinations (at minimum: NewImageURL, NewImageFile, NewVideoURL, NewVideoFile, NewAudioFile, NewPDFFile).
  2. Existing tests and examples continue to work — constructors are additive sugar, not replacements.
  3. Multimodal docs and provider examples are updated to show the shorthand forms alongside the verbose forms.
  4. All new constructors have unit tests.
**Plans:** 2 plans

Plans:
- [x] 09-01-PLAN.md — Implement convenience constructors, ContentOption, and unit tests
- [x] 09-02-PLAN.md — Update multimodal docs, provider sections, and rewrite runnable examples

### Phase 10: Code Cleanups
**Goal:** Consolidate duplicated path safety utilities into a shared internal package, fix the *context.Context pointer-to-interface anti-pattern across embedding providers, add registry test cleanup to prevent global state leaks, and fix resolveMIME for URL-backed sources.
**Depends on:** Phase 9
**Issues**: #456, #461, #466, #469
**Requirements**: [CLN-01, CLN-02, CLN-03, CLN-04, CLN-05, CLN-06]
**Success Criteria** (what must be TRUE):
  1. A shared `pkg/internal/pathutil` package provides `ContainsDotDot`, `ValidateFilePath`, and `SafePath` utilities.
  2. Gemini, Voyage, and default_ef use the shared path utilities instead of local duplicates.
  3. Gemini, Nomic, and Mistral use `context.Context` (not `*context.Context`) for DefaultContext.
  4. Registry tests use `t.Cleanup` with unregister helpers to prevent global state leaks.
  5. All existing tests pass without modification.
  6. Gemini and VoyageAI `resolveMIME` infer MIME type from URL path extensions, making URL constructors work end-to-end.
**Plans:** 2/2 plans complete

Plans:
- [x] 10-01-PLAN.md — Create shared pathutil package, replace local implementations, fix *context.Context anti-pattern
- [x] 10-02-PLAN.md — Add resolveMIME URL fallback, registry unregister helpers, and t.Cleanup to all tests

### Phase 11: Fork Double-Close Bug
**Goal:** Fix EF pointer sharing in Fork() that causes the same underlying embedding function resource to be closed twice when client.Close() iterates cached collections.
**Depends on:** None (independent, but should precede ForkCount work)
**Issues**: #454
**Success Criteria** (what must be TRUE):
  1. Forked collections do not double-close shared EF resources when client.Close() is called.
  2. Both `embeddingFunction` and `contentEmbeddingFunction` ownership is handled correctly.
  3. Tests cover Fork + Close lifecycle without panics or use-after-close errors.
  4. Existing fork tests continue to pass.
**Requirements**: [FORK-01, FORK-02, FORK-03, FORK-04]
**Plans:** 2/2 plans complete

Plans:
- [x] 11-01-PLAN.md — Create close-once EF wrappers, add ownsEF flag, gate Close() in HTTP and embedded paths
- [x] 11-02-PLAN.md — Add unit tests for close-once wrappers and ownership gating

### Phase 12: SDK Auto-Wiring Research
**Goal:** Trace contentEmbeddingFunction auto-wiring behavior in official Chroma SDKs (Python, JavaScript) to verify chroma-go's approach is consistent or document deliberate differences.
**Depends on:** None (research task, informs Phase 13)
**Issues**: #455
**Success Criteria** (what must be TRUE):
  1. Python SDK auto-wiring behavior documented for get_collection, list_collections, and create_collection.
  2. JavaScript SDK auto-wiring behavior documented for equivalent operations.
  3. Comparison with chroma-go behavior written up with any recommended changes or documented differences.
**Plans:** 1/1 plans complete

Plans:
- [x] 12-01-PLAN.md — Verify SDK source claims and finalize comparison document

### Phase 13: Collection.ForkCount
**Goal:** Add `ForkCount(ctx) (int, error)` to the V2 Collection interface with HTTP transport support, matching upstream Chroma's /fork_count endpoint.
**Depends on:** Phase 11, Phase 12 (benefits from fork bug fix and SDK research)
**Issues**: #460
**Success Criteria** (what must be TRUE):
  1. `pkg/api/v2.Collection` includes `ForkCount(ctx context.Context) (int, error)`.
  2. HTTP implementation issues `GET .../fork_count` and decodes `{"count": n}`.
  3. Embedded/local behavior returns an explicit unsupported error.
  4. Tests cover HTTP happy path, failure path, and embedded unsupported path.
  5. Forking docs mention the new method.
**Plans:** 2/2 plans complete

Plans:
- [x] TBD (run /gsd:plan-phase 13 to break down) (completed 2026-03-28)

### Phase 14: Delete with Limit
**Goal:** Add limit parameter support to collection delete operations, matching upstream Chroma PRs #6573/#6582.
**Depends on:** None (independent)
**Issues**: #439
**Requirements**: [DEL-01, DEL-02, DEL-03, DEL-04, DEL-05]
**Success Criteria** (what must be TRUE):
  1. Delete operations accept an optional limit parameter.
  2. HTTP transport sends the limit when specified.
  3. Tests cover delete-with-limit happy path and edge cases.
**Plans:** 2/2 plans complete

Plans:
- [x] 14-01-PLAN.md — Add Limit field to CollectionDeleteOp, ApplyToDelete to limitOption, validation, and embedded path wiring
- [x] 14-02-PLAN.md — Add unit tests for option application, validation, and HTTP serialization

### Phase 15: OpenRouter Embeddings Compatibility
**Goal:** Extend the OpenAI embedding function to support OpenRouter-specific fields (encoding_format, input_type, provider preferences) and relax model validation for provider-prefixed IDs.
**Depends on:** None (independent)
**Issues**: #438
**Success Criteria** (what must be TRUE):
  1. `CreateEmbeddingRequest` supports `encoding_format`, `input_type`, and `provider` fields.
  2. `WithModel` accepts provider-prefixed model IDs (e.g. `openai/text-embedding-3-small`).
  3. Provider preferences struct covers documented OpenRouter fields with extensibility.
  4. Existing OpenAI behavior and tests remain unchanged.
  5. Docs include OpenRouter usage example with `WithBaseURL`.
**Plans:** 2/2 plans complete

Plans:
- [x] 15-01-PLAN.md -- Add WithModelString to OpenAI provider and create standalone OpenRouter provider package
- [x] 15-02-PLAN.md -- Add unit tests and integration verification for OpenRouter provider

### Phase 16: Twelve Labs Embedding Function
**Goal:** Add a new Twelve Labs multimodal embedding provider supporting text, image, and audio embeddings via the Twelve Labs API.
**Depends on:** Phase 9 (benefits from Content API foundations)
**Issues**: #190
**Success Criteria** (what must be TRUE):
  1. `pkg/embeddings/twelvelabs` implements dense embedding and Content API interfaces.
  2. Supports text, image, and audio modalities per Twelve Labs API docs.
  3. Registered in factory/registry with config round-trip support.
  4. Tests cover request construction, modality validation, and config persistence.
  5. Docs and examples added for Twelve Labs provider.
**Plans:** 2 plans

Plans:
- [x] 16-01-PLAN.md -- Implement client struct, text embedding, Content API, config, and dual registration
- [ ] 16-02-PLAN.md -- Add unit tests for capability derivation, intent mapping, content conversion, and config round-trip

### Phase 17: Cloud RRF and GroupBy Test Coverage
**Goal:** Add end-to-end cloud integration tests that exercise Search API RRF and GroupBy primitives against live Chroma Cloud.
**Depends on:** None (independent, but best run last as test hardening)
**Issues**: #462
**Success Criteria** (what must be TRUE):
  1. RRF smoke test using dense + sparse KNN ranks with `WithKnnReturnRank`.
  2. RRF weighted/custom-k test proves request acceptance and ordering changes.
  3. GroupBy MinK/MaxK tests assert per-group caps and flattened limits.
  4. All tests tagged `cloud` and use existing cloud test infrastructure.
**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd:plan-phase 17 to break down)

### Phase 18: Embedded Client contentEmbeddingFunction Parity
**Goal:** Add contentEmbeddingFunction support to embeddedCollection so the embedded client has feature parity with the HTTP client for content embedding lifecycle, auto-wiring, and Fork/Close handling.
**Depends on:** Phase 11 (fork double-close fix provides the close-once infrastructure)
**Issues**: #472
**Success Criteria** (what must be TRUE):
  1. `embeddedCollection` struct and state include `contentEmbeddingFunction` field.
  2. `buildEmbeddedCollection` accepts and wires contentEF.
  3. `embeddedCollection.Close()` handles contentEF with sharing detection matching HTTP path.
  4. `embeddedCollection.Fork()` propagates contentEF with close-once wrapping.
  5. Embedded `GetCollection()` respects `WithContentEmbeddingFunctionGet` option.
  6. Tests cover lifecycle, Fork, Close, and auto-wiring for content EF on embedded path.
**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd:plan-phase 18 to break down)
