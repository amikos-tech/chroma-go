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

- [x] **DOCS-01**: Public docs explain portable intent usage, provider-specific escape hatches, and compatibility expectations for multimodal callers
- [x] **DOCS-02**: Tests cover shared type validation, compatibility adapters, registry/config round-trips, and unsupported-combination failures

### Gemini Multimodal Adoption

- [x] **GEM-01**: Gemini implements `SharedContentEmbeddingFunction` and `CapabilityAware` for text, image, audio, video, and PDF modalities
- [x] **GEM-02**: Neutral intents map to Gemini task types with explicit errors for unsupported combinations
- [x] **GEM-03**: Gemini is registered in the multimodal factory/registry path with config round-trip support

### Voyage Multimodal Adoption

- [x] **VOY-01**: VoyageAI implements `ContentEmbeddingFunction`, `CapabilityAware`, and `IntentMapper` for text, image, and video modalities
- [x] **VOY-02**: Neutral intents map to Voyage input types with explicit errors for unsupported combinations
- [x] **VOY-03**: VoyageAI is registered in the multimodal factory/registry path with config round-trip support

### Convenience Constructors

- [x] **CONV-01**: Caller can create single-modality Content with a single function call (NewTextContent, NewImageURL, NewImageFile, NewVideoURL, NewVideoFile, NewAudioURL, NewAudioFile, NewPDFURL, NewPDFFile) instead of nested struct literals
- [x] **CONV-02**: Caller can compose multi-part Content from Part helpers via NewContent with optional ContentOption configuration
- [x] **CONV-03**: All convenience constructors have unit tests and existing tests/examples remain green
- [x] **CONV-04**: Multimodal docs and provider examples show shorthand constructors as the primary examples with verbose forms linked from the generic Content API page

### Code Cleanups

- [x] **CLN-01**: Duplicated path safety functions (`containsDotDot`, `safePath`) are extracted into a shared `pkg/internal/pathutil` package with unit tests
- [x] **CLN-02**: Gemini, Voyage, and default_ef providers import path safety utilities from the shared package instead of maintaining local copies
- [x] **CLN-03**: Gemini, Nomic, and Mistral providers use `context.Context` (value type) for `DefaultContext` instead of `*context.Context` (pointer-to-interface anti-pattern)
- [x] **CLN-04**: Registry tests use `t.Cleanup` with unexported unregister helpers to prevent global state leaks between test runs
- [x] **CLN-05**: Gemini and VoyageAI `resolveMIME` functions infer MIME type from URL path extensions as a fallback, making URL constructors work end-to-end
- [x] **CLN-06**: All existing tests pass without modification after cleanup changes

### Fork Double-Close Bug

- [x] **FORK-01**: Forked collections do not double-close shared EF resources when `client.Close()` iterates the collection cache
- [x] **FORK-02**: Both `embeddingFunction` and `contentEmbeddingFunction` ownership is tracked via an `ownsEF` flag on collection structs
- [x] **FORK-03**: Shared EFs are wrapped in a `sync.Once`-based close-once adapter that makes `Close()` idempotent as defence-in-depth
- [x] **FORK-04**: Tests cover Fork + Close lifecycle including idempotent close, use-after-close errors, and ownership gating without panics

### Collection ForkCount

- [ ] **FC-01**: `pkg/api/v2.Collection` interface includes `ForkCount(ctx context.Context) (int, error)`
- [ ] **FC-02**: HTTP implementation issues `GET .../fork_count` and decodes `{"count": n}` using strict struct with `json:"count"` tag
- [ ] **FC-03**: Embedded/local implementation returns explicit unsupported error matching existing Fork/Search pattern
- [ ] **FC-04**: Tests cover HTTP happy path, HTTP failure path, and embedded unsupported path
- [x] **FC-05**: Forking docs page includes ForkCount section with Go and Python examples and API reference row
- [x] **FC-06**: Runnable Fork + ForkCount example exists under `examples/v2/`

### Delete with Limit

- [x] **DEL-01**: `WithLimit(n)` applies to `Collection.Delete` via `ApplyToDelete` method on `limitOption`, reusing the existing option function
- [x] **DEL-02**: `CollectionDeleteOp` has a `Limit *int32` field with `json:"limit,omitempty"` tag
- [x] **DEL-03**: `PrepareAndValidate` rejects limit without where/where_document filter and limit <= 0 with exact upstream error messages
- [x] **DEL-04**: Embedded path converts `*int32` limit to `*uint32` and passes to `EmbeddedDeleteRecordsRequest.Limit`
- [ ] **DEL-05**: Tests cover option application, validation edge cases, and HTTP serialization round-trip

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
| DOCS-01 | Phase 5 | Complete |
| DOCS-02 | Phase 5 | Complete |
| GEM-01 | Phase 6 | Complete |
| GEM-02 | Phase 6 | Complete |
| GEM-03 | Phase 6 | Complete |
| VOY-01 | Phase 7 | Complete |
| VOY-02 | Phase 7 | Complete |
| VOY-03 | Phase 7 | Complete |
| CONV-01 | Phase 9 | Planned |
| CONV-02 | Phase 9 | Planned |
| CONV-03 | Phase 9 | Planned |
| CONV-04 | Phase 9 | Planned |
| CLN-01 | Phase 10 | Planned |
| CLN-02 | Phase 10 | Planned |
| CLN-03 | Phase 10 | Planned |
| CLN-04 | Phase 10 | Planned |
| CLN-05 | Phase 10 | Planned |
| CLN-06 | Phase 10 | Planned |
| FORK-01 | Phase 11 | Complete |
| FORK-02 | Phase 11 | Complete |
| FORK-03 | Phase 11 | Complete |
| FORK-04 | Phase 11 | Planned |
| FC-01 | Phase 13 | Planned |
| FC-02 | Phase 13 | Planned |
| FC-03 | Phase 13 | Planned |
| FC-04 | Phase 13 | Planned |
| FC-05 | Phase 13 | Planned |
| FC-06 | Phase 13 | Planned |
| DEL-01 | Phase 14 | Complete |
| DEL-02 | Phase 14 | Complete |
| DEL-03 | Phase 14 | Complete |
| DEL-04 | Phase 14 | Complete |
| DEL-05 | Phase 14 | Planned |

**Coverage:**
- v1 requirements: 46 total
- Mapped to phases: 46
- Unmapped: 0

---
*Requirements defined: 2026-03-18*
*Last updated: 2026-03-29 -- added DEL-01/02/03/04/05 for phase 14*
