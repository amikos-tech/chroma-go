---
phase: 04-provider-mapping-and-explicit-failures
plan: 01
subsystem: embeddings
tags: [go, interfaces, validation, multimodal, embeddings]

requires:
  - phase: 03-collection-wiring
    provides: ContentEmbeddingFunction, CapabilityAware, CapabilityMetadata — shared multimodal contract this plan extends
provides:
  - IntentMapper opt-in interface for providers to translate neutral intents to native strings
  - IsNeutralIntent helper identifying the 5 shared neutral intent constants
  - ValidateContentSupport single-item capability pre-check helper
  - ValidateContentsSupport batch capability pre-check helper
  - Three new validation codes: unsupported_modality, unsupported_intent, unsupported_dimension
affects:
  - 04-02
  - 05-gemini-provider
  - 06-vllm-nemotron

tech-stack:
  added: []
  patterns:
    - "Opt-in interface: IntentMapper follows same pattern as CapabilityAware and Closeable — type-assert before use"
    - "Pass-through guard: ValidateContentSupport skips modality check when caps.Modalities is empty (non-CapabilityAware providers)"
    - "Neutral-only intent enforcement: only neutral intents are checked against caps; custom provider-native strings bypass enforcement"
    - "Fail-on-first batch: ValidateContentsSupport returns on first unsupported item, consistent with existing batch validation"

key-files:
  created: []
  modified:
    - pkg/embeddings/embedding.go
    - pkg/embeddings/multimodal.go
    - pkg/embeddings/multimodal_validate.go

key-decisions:
  - "IntentMapper is an opt-in interface; callers type-assert rather than requiring all providers to implement it"
  - "IsNeutralIntent uses a switch over exact constants, so any future provider-native intent string not in the 5 constants returns false automatically"
  - "ValidateContentSupport passes through (returns nil) when caps.Modalities is empty — preserves backward compatibility for providers that do not implement CapabilityAware"
  - "Custom intents bypass capability intent enforcement — only neutral intents are checked against declared caps.Intents"

patterns-established:
  - "Provider mapping contract: providers opt into IntentMapper to translate before dispatch"
  - "Pre-check helpers: ValidateContentSupport/ValidateContentsSupport called before provider I/O to fail fast"

requirements-completed:
  - MAP-01
  - MAP-02

duration: 8min
completed: 2026-03-20
---

# Phase 4 Plan 1: Provider Mapping Contract and Explicit Failure Helpers Summary

**IntentMapper opt-in interface, IsNeutralIntent switch helper, and ValidateContentSupport/ValidateContentsSupport pre-check helpers with 3 new validation codes added to pkg/embeddings**

## Performance

- **Duration:** 8 min
- **Started:** 2026-03-20T12:10:00Z
- **Completed:** 2026-03-20T12:18:00Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Defined IntentMapper opt-in interface following the CapabilityAware/Closeable pattern, enabling providers to translate neutral intents to native strings via type-assertion
- Added IsNeutralIntent helper in multimodal.go that identifies the 5 shared neutral intent constants via a switch, returning false for any custom provider-native strings
- Added ValidateContentSupport and ValidateContentsSupport pre-check helpers that validate modality, intent, and dimension against CapabilityMetadata, with empty-slice guards for backward compatibility
- Added three new unexported validation codes: unsupported_modality, unsupported_intent, unsupported_dimension

## Task Commits

Each task was committed atomically:

1. **Task 1: Add IntentMapper interface and IsNeutralIntent helper** - `435e4a8` (feat)
2. **Task 2: Add ValidateContentSupport, ValidateContentsSupport, and 3 validation codes** - `66418ac` (feat)
3. **Deviation auto-fix: gci const formatting** - `9f0095b` (chore)

## Files Created/Modified
- `pkg/embeddings/embedding.go` - Added IntentMapper opt-in interface alongside CapabilityAware and Closeable
- `pkg/embeddings/multimodal.go` - Added IsNeutralIntent helper after the 5 intent constants
- `pkg/embeddings/multimodal_validate.go` - Added 3 new validation codes and ValidateContentSupport/ValidateContentsSupport functions

## Decisions Made
- IntentMapper is an opt-in interface (type-assert pattern) rather than widening ContentEmbeddingFunction — keeps contract minimal, lets non-mapping providers remain unaffected
- IsNeutralIntent uses exhaustive switch so new provider-native strings automatically return false without code changes
- ValidateContentSupport passes through when caps.Modalities is empty to preserve backward compatibility with non-CapabilityAware providers

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed gci formatting violation in validation code const block**
- **Found during:** Post-task lint check (Task 2)
- **Issue:** Extra spaces in const alignment caused gci linter to report formatting violation
- **Fix:** Ran `make lint-fix` to auto-format the const block to proper alignment
- **Files modified:** pkg/embeddings/multimodal_validate.go
- **Verification:** `make lint` reports 0 issues
- **Committed in:** `9f0095b` (separate chore commit)

---

**Total deviations:** 1 auto-fixed (lint formatting)
**Impact on plan:** Minor formatting fix. No behavior change, no scope creep.

## Issues Encountered
None - all planned changes applied cleanly and existing tests passed.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- IntentMapper, IsNeutralIntent, ValidateContentSupport, and ValidateContentsSupport are production-ready in pkg/embeddings
- Phase 4 Plan 2 can now add tests for these new helpers
- Phases 5-7 (provider implementations) can implement IntentMapper and call ValidateContentSupport before dispatch

---
*Phase: 04-provider-mapping-and-explicit-failures*
*Completed: 2026-03-20*
