---
phase: 07-voyage-multimodal-adoption
plan: 01
subsystem: embeddings
tags: [voyageai, multimodal, content-api, intent-mapping, capabilities]

# Dependency graph
requires:
  - phase: 06-gemini-multimodal-adoption
    provides: "Reference pattern for ContentEmbeddingFunction + CapabilityAware + IntentMapper native implementation"
  - phase: 04-provider-mapping-and-explicit-failures
    provides: "IntentMapper contract, ValidateContentSupport, explicit failure paths"
  - phase: 01-shared-multimodal-contract
    provides: "Content, Part, BinarySource, Modality, Intent types"
provides:
  - "VoyageAI ContentEmbeddingFunction implementation for text, image, and video modalities"
  - "VoyageAI CapabilityAware with model-based capability derivation"
  - "VoyageAI IntentMapper mapping retrieval_query/retrieval_document to Voyage input_type"
  - "CreateMultimodalEmbedding client method targeting /v1/multimodalembeddings"
  - "RegisterContent('voyageai') alongside existing RegisterDense"
affects: [07-02, 08-documentation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Dual-endpoint provider pattern (separate text and multimodal API endpoints)"
    - "Model-based capability derivation (capabilitiesForModel)"
    - "ProviderHints escape hatch for input_type passthrough"

key-files:
  created:
    - "pkg/embeddings/voyage/content.go"
  modified:
    - "pkg/embeddings/voyage/voyage.go"

key-decisions:
  - "Copied resolveBytes/resolveMIME/containsDotDot helpers from Gemini rather than extracting to shared (scoped refactor)"
  - "Batch requests reject per-item Intent, Dimension, and ProviderHints['input_type'] with explicit errors"
  - "multimodalURL derives endpoint from BaseAPI by replacing path suffix, falling back to constant"

patterns-established:
  - "Dual-endpoint provider: separate /v1/embeddings and /v1/multimodalembeddings with shared response type"
  - "Sparse intent mapping: only 2 of 5 neutral intents supported with explicit rejection of remaining 3"

requirements-completed: [VOY-01, VOY-02, VOY-03]

# Metrics
duration: 3min
completed: 2026-03-22
---

# Phase 7 Plan 1: VoyageAI Multimodal Content Embedding Summary

**VoyageAI ContentEmbeddingFunction with text/image/video multimodal support, dual-endpoint pattern, and intent mapping to Voyage input_type via shared Content API**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-22T16:04:45Z
- **Completed:** 2026-03-22T16:08:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- VoyageAI implements ContentEmbeddingFunction, CapabilityAware, and IntentMapper at compile time
- capabilitiesForModel derives text/image/video modalities for voyage-multimodal-3.5 with dimension support
- MapIntent maps retrieval_query/retrieval_document and explicitly rejects classification, clustering, semantic_similarity
- EmbedContent/EmbedContents validate, convert to Voyage multimodal format, and delegate to CreateMultimodalEmbedding
- Dual registration (RegisterDense + RegisterContent) enables config round-trip via content factory

## Task Commits

Each task was committed atomically:

1. **Task 1: Create content.go with multimodal types, conversion helpers, capabilities, and intent mapping** - `f96dfb9` (feat)
2. **Task 2: Add CreateMultimodalEmbedding, interface assertions, and RegisterContent to voyage.go** - `7669992` (feat)

## Files Created/Modified
- `pkg/embeddings/voyage/content.go` - Multimodal types, conversion pipeline, capabilities, intent mapping, EmbedContent/EmbedContents
- `pkg/embeddings/voyage/voyage.go` - Interface assertions, CreateMultimodalEmbedding client method, RegisterContent in init()

## Decisions Made
- Copied binary resolution helpers from Gemini content.go rather than extracting to shared package (out of scope refactor)
- Batch requests (len > 1) reject per-item Intent/Dimension/ProviderHints to match Voyage's request-level configuration
- Multimodal URL derivation replaces /v1/embeddings suffix with /v1/multimodalembeddings, falling back to hardcoded constant for custom base URLs

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- VoyageAI content implementation complete, ready for Phase 7 Plan 2 (unit tests)
- Integration tests require VOYAGE_API_KEY env var

## Self-Check: PASSED

All created files verified present. All commit hashes found in git log.

---
*Phase: 07-voyage-multimodal-adoption*
*Completed: 2026-03-22*
