---
phase: 03-registry-and-config-integration
plan: "01"
subsystem: embeddings
tags: [go, registry, content-embedding, factory, capabilities]

requires:
  - phase: 02-compatibility-layer
    provides: AdaptEmbeddingFunctionToContent, AdaptMultimodalEmbeddingFunctionToContent, CapabilityAware, ContentEmbeddingFunction

provides:
  - contentFactories map with RegisterContent, BuildContent, BuildContentCloseable, ListContent, HasContent
  - inferCaps helper for CapabilityMetadata inference from any embedding function
  - 3-step fallback chain: native content -> multimodal+adapt -> dense+adapt -> error

affects:
  - 03-02 (BuildContentEFFromConfig depends on BuildContent and RegisterContent)
  - 03-03 (collection auto-wiring delegates to content registry)

tech-stack:
  added: []
  patterns:
    - "4th factory map pattern: shares mu sync.RWMutex with dense/sparse/multimodal maps"
    - "Lock-release-before-call: mu.RLock released before each factory call to avoid deadlock in fallback chain"
    - "inferCaps: CapabilityAware first, then MultimodalEmbeddingFunction interface check, then text-only default"

key-files:
  created: []
  modified:
    - pkg/embeddings/registry.go
    - pkg/embeddings/registry_test.go

key-decisions:
  - "BuildContent fallback chain acquires and releases mu.RLock separately before each factory call to avoid recursive lock acquisition deadlock"
  - "inferCaps uses CapabilityAware metadata when available and falls back to interface-typed defaults (multimodal gets text+image, dense gets text-only)"
  - "ContentEmbeddingFunctionFactory follows the same factory function signature pattern as Dense/Sparse/Multimodal for consistency"

patterns-established:
  - "inferCaps: prefer CapabilityAware > MultimodalEmbeddingFunction interface check > text-only default"
  - "BuildContent fallback: each registry lookup is a separate lock+unlock pair before calling factory"

requirements-completed:
  - REG-01

duration: 3min
completed: "2026-03-20"
---

# Phase 03 Plan 01: Content Factory Registry Summary

**4th content factory map added to embedding registry with 3-step fallback chain (native -> multimodal+adapt -> dense+adapt) and inferCaps helper using CapabilityAware metadata**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-20T09:37:36Z
- **Completed:** 2026-03-20T09:40:18Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added `ContentEmbeddingFunctionFactory` type and `contentFactories` map to registry.go (4th map alongside dense/sparse/multimodal)
- Implemented `RegisterContent`, `BuildContent` (with 3-step fallback), `BuildContentCloseable`, `ListContent`, `HasContent`, and `inferCaps`
- Added 10 unit tests covering native registration, fallback chain, closeable delegation, capability inference, list/has, and duplicate rejection

## Task Commits

Each task was committed atomically:

1. **Task 1: Add 4th content factory map to registry.go** - `5837fb5` (feat)
2. **Task 2: Add content registry unit tests** - `c044a2e` (test)

**Plan metadata:** (docs commit follows)

_Note: TDD — tests written before implementation (RED verified via build failure, GREEN via all 10 tests passing)_

## Files Created/Modified

- `pkg/embeddings/registry.go` - Added ContentEmbeddingFunctionFactory type, contentFactories map, RegisterContent, BuildContent (with fallback), BuildContentCloseable, ListContent, HasContent, inferCaps
- `pkg/embeddings/registry_test.go` - Added 10 content registry tests plus 3 mock types (mockContentEmbeddingFunction, mockMultimodalEmbeddingFunction, mockCapabilityAwareEmbeddingFunction)

## Decisions Made

- BuildContent fallback chain releases the read lock before calling each factory to prevent deadlock, since BuildDense/BuildMultimodal also acquire mu.RLock internally
- inferCaps checks CapabilityAware first (provider-declared capabilities take precedence), then checks MultimodalEmbeddingFunction interface for {text, image} default, otherwise uses {text} default

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Content registry is fully operational; Plan 02 can implement `BuildContentEFFromConfig` by calling `BuildContent`
- `RegisterContent` is available for providers to self-register in their `init()` functions
- No blockers

---
*Phase: 03-registry-and-config-integration*
*Completed: 2026-03-20*
