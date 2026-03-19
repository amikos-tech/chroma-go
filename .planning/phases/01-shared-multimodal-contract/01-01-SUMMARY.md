---
phase: 01-shared-multimodal-contract
plan: "01"
subsystem: api
tags: [embeddings, multimodal, go, shared-contract]
requires:
  - phase: 01-00
    provides: Compileable multimodal test targets for task verification
provides:
  - "Canonical Content, Part, BinarySource, Modality, Intent, and SourceKind types in pkg/embeddings"
  - "Additive ContentEmbeddingFunction interface for batch and single-item multimodal content embedding"
affects: [phase-2-compatibility, phase-3-registry, multimodal-embeddings]
tech-stack:
  added: []
  patterns: [ordered-content-contract, additive-interface-expansion]
key-files:
  created:
    - pkg/embeddings/multimodal.go
  modified:
    - pkg/embeddings/embedding.go
key-decisions:
  - "Keep the shared multimodal request model in a dedicated file so later validation and compatibility work can layer on without disturbing legacy APIs."
  - "Add ContentEmbeddingFunction beside MultimodalEmbeddingFunction instead of widening the legacy image-only interface in place."
patterns-established:
  - "Ordered mixed-part content is represented as Content with []Part and explicit Modality values."
  - "Richer multimodal APIs are introduced additively through new interfaces rather than by changing existing public contracts."
requirements-completed: [MMOD-01, MMOD-02, MMOD-03, MMOD-04]
duration: 4min
completed: 2026-03-18
---

# Phase 1 Plan 01: Shared Multimodal Contract Summary

**Portable multimodal request types and an additive content embedding interface now exist in `pkg/embeddings` without changing the legacy text-only or image-only APIs.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-18T19:34:30Z
- **Completed:** 2026-03-18T19:38:52Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added `pkg/embeddings/multimodal.go` with the canonical `Content`, `Part`, `BinarySource`, `Modality`, `Intent`, and `SourceKind` contract types for ordered multimodal requests.
- Added portable request-time override fields on `Content` via `Intent`, `Dimension`, and `ProviderHints`.
- Added `ContentEmbeddingFunction` in `pkg/embeddings/embedding.go` while preserving the existing `EmbeddingFunction` and `MultimodalEmbeddingFunction` interfaces unchanged.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add canonical multimodal request, source, and intent types** - `271c4ca` (`feat`)
2. **Task 2: Add richer content embedding interface without changing legacy multimodal APIs** - `7e7d6de` (`feat`)

**Plan metadata:** recorded in the dedicated docs wrap-up commit for this plan

## Files Created/Modified

- `pkg/embeddings/multimodal.go` - shared caller-facing multimodal request, source, modality, intent, and option types
- `pkg/embeddings/embedding.go` - additive `ContentEmbeddingFunction` interface beside the existing legacy interfaces

## Decisions Made

- Kept the multimodal contract in a separate file to make later validation and compatibility work additive and localized.
- Preserved the existing public interfaces exactly and introduced richer multimodal embedding as a new interface instead of altering the image-only one.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Phase 1 now has the shared request surface needed for validation helpers and compatibility-safe constructors in `01-02`.
The current package and `basicv2` config reconstruction tests remain green after the additive API expansion.

## Self-Check: PASSED

- FOUND: `.planning/phases/01-shared-multimodal-contract/01-01-SUMMARY.md`
- FOUND: `271c4ca`
- FOUND: `7e7d6de`

---
*Phase: 01-shared-multimodal-contract*
*Completed: 2026-03-18*
