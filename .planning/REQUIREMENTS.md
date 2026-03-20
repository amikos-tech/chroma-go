# Requirements: Chroma Go

**Defined:** 2026-03-18
**Core Value:** Go applications can use Chroma and embedding providers through a stable, portable API that minimizes provider-specific friction.

## v1 Requirements

### Multimodal Contract

- [x] **MMOD-01**: Caller can describe a multimodal embedding request as an ordered set of parts containing text, image, audio, video, or PDF content
- [x] **MMOD-02**: Caller can submit mixed-part multimodal requests without losing the original part ordering
- [x] **MMOD-03**: Caller can set a provider-neutral intent for a multimodal request using shared semantics such as retrieval query, retrieval document, classification, clustering, or semantic similarity
- [x] **MMOD-04**: Caller can set per-request options such as target output dimensionality and provider-specific hints without mutating provider-wide configuration
- [x] **MMOD-05**: Invalid request shapes are rejected before provider I/O with explicit validation errors

### Capabilities and Compatibility

- [x] **CAPS-01**: A provider can declare which modalities, intents, and request options it supports through shared capability metadata
- [x] **CAPS-02**: Caller can inspect shared capability metadata without depending on provider-specific concrete types
- [x] **COMP-01**: Existing `EmbeddingFunction` text-only callers continue to compile and behave the same without adopting the new multimodal request API
- [x] **COMP-02**: Existing image-only multimodal callers continue to compile and interoperate with the new shared multimodal foundations

### Registry and Mapping

- [x] **REG-01**: Factory and registry code can build richer multimodal embedding functions from stored config using additive shared interfaces
- [x] **REG-02**: Collection configuration auto-wiring keeps working for existing dense and multimodal providers after the richer interfaces are introduced
- [x] **MAP-01**: Neutral intents are mapped to provider-native task and input semantics through a defined contract with tests
- [x] **MAP-02**: Unsupported modality or intent combinations fail explicitly instead of silently downgrading or guessing

### Docs and Verification

- [ ] **DOCS-01**: Public docs explain portable intent usage, provider-specific escape hatches, and compatibility expectations for multimodal callers
- [ ] **DOCS-02**: Tests cover shared type validation, compatibility adapters, registry/config round-trips, and unsupported-combination failures

### Gemini Multimodal Adoption

- [ ] **GEM-01**: Gemini implements `SharedContentEmbeddingFunction` and `CapabilityAware` for text, image, audio, video, and PDF modalities
- [ ] **GEM-02**: Neutral intents map to Gemini task types with explicit errors for unsupported combinations
- [ ] **GEM-03**: Gemini is registered in the multimodal factory/registry path with config round-trip support

### vLLM/Nemotron Provider Validation

- [ ] **VLLM-01**: A vLLM/OpenAI-compatible provider implements `SharedContentEmbeddingFunction` and `CapabilityAware` for modalities supported by omni-embed-nemotron-3b
- [ ] **VLLM-02**: Integration tests validate multimodal embedding through a live vLLM endpoint without provider-specific contract hacks

## v2 Requirements

### Provider Adoption

- **PROV-01**: Additional providers beyond the current multimodal baseline adopt the richer shared multimodal contract
- **PROV-02**: End-to-end examples cover concrete audio, video, or PDF provider implementations once they exist

### Advanced Discovery

- **DISC-01**: Capability discovery can be derived from remote or generated provider metadata instead of only static code declarations
- **DISC-02**: Shared batching helpers optimize multimodal requests where providers expose compatible batch semantics

## Out of Scope

| Feature | Reason |
|---------|--------|
| Migrate every embedding provider to the richer multimodal contract immediately | Too large for the foundation milestone; ship the shared contract first |
| Replace or remove existing text-only and image-only APIs | Contradicts the no-breaking-change acceptance criteria |
| Change collection query semantics outside embedding abstractions | Not necessary to deliver provider-neutral multimodal foundations |
| Add server-side capability negotiation in this milestone | Valuable later, but not required for the additive client-side contract |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| MMOD-01 | Phase 1 | Complete |
| MMOD-02 | Phase 1 | Complete |
| MMOD-03 | Phase 1 | Complete |
| MMOD-04 | Phase 1 | Complete |
| MMOD-05 | Phase 1 | Complete |
| CAPS-01 | Phase 2 | Complete |
| CAPS-02 | Phase 2 | Complete |
| COMP-01 | Phase 2 | Complete |
| COMP-02 | Phase 2 | Complete |
| REG-01 | Phase 3 | Complete |
| REG-02 | Phase 3 | Complete |
| MAP-01 | Phase 4 | Complete |
| MAP-02 | Phase 4 | Complete |
| DOCS-01 | Phase 5 | Pending |
| DOCS-02 | Phase 5 | Pending |
| GEM-01 | Phase 6 | Pending |
| GEM-02 | Phase 6 | Pending |
| GEM-03 | Phase 6 | Pending |
| VLLM-01 | Phase 7 | Pending |
| VLLM-02 | Phase 7 | Pending |

**Coverage:**
- v1 requirements: 20 total
- Mapped to phases: 20
- Unmapped: 0 ✓

---
*Requirements defined: 2026-03-18*
*Last updated: 2026-03-20 — added GEM-01/02/03 and VLLM-01/02 for phases 6-7*
