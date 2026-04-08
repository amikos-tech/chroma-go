---
phase: 15-openrouter-embeddings-compatibility
plan: 01
subsystem: embeddings
tags: [openrouter, openai, embedding-function, provider-preferences, registry]

requires:
  - phase: 04-provider-mapping-and-explicit-failures
    provides: Dense registry pattern and EmbeddingFunction interface
provides:
  - WithModelString option on OpenAI provider for arbitrary model names
  - Standalone OpenRouter embedding provider with ProviderPreferences
  - Dense registry entry "openrouter" with config round-trip
affects: [15-02-PLAN, openai-compatible-proxies]

tech-stack:
  added: []
  patterns: [standalone-provider-no-openai-dependency, withmodelstring-for-proxy-compatibility]

key-files:
  created:
    - pkg/embeddings/openrouter/openrouter.go
    - pkg/embeddings/openrouter/options.go
    - pkg/embeddings/openrouter/provider.go
  modified:
    - pkg/embeddings/openai/options.go
    - pkg/embeddings/openai/openai.go

key-decisions:
  - "Follow Together provider pattern for standalone OpenRouter package - no dependency on openai package"
  - "WithModelString bypasses validation for proxy-compatible model names while WithModel retains strict validation"

patterns-established:
  - "WithModelString pattern: allow arbitrary model IDs on OpenAI-compatible endpoints without enum validation"
  - "ProviderPreferences with Extras map and custom MarshalJSON for forward-compatible provider routing"

requirements-completed: [OR-01, OR-02, OR-03, OR-04, OR-05]

duration: 2min
completed: 2026-03-30
---

# Phase 15 Plan 01: OpenRouter Provider and OpenAI WithModelString Summary

**Standalone OpenRouter embedding provider with ProviderPreferences routing and WithModelString for OpenAI-compatible proxy model names**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-30T18:28:32Z
- **Completed:** 2026-03-30T18:30:54Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Added WithModelString option to OpenAI provider for arbitrary model names on compatible proxies
- Updated OpenAI FromConfig to route non-standard model names through WithModelString
- Created standalone OpenRouter provider with full request/response types, ProviderPreferences, functional options, config round-trip, and dense registry registration

## Task Commits

Each task was committed atomically:

1. **Task 1: Add WithModelString to OpenAI provider and update FromConfig** - `4f4933d` (feat)
2. **Task 2: Create standalone OpenRouter provider package** - `6d827d5` (feat)

## Files Created/Modified
- `pkg/embeddings/openai/options.go` - Added WithModelString option function
- `pkg/embeddings/openai/openai.go` - Updated FromConfig to use WithModelString for non-standard models
- `pkg/embeddings/openrouter/openrouter.go` - Client, HTTP, EmbeddingFunction, config round-trip, registry
- `pkg/embeddings/openrouter/options.go` - Functional options (WithModel, WithEncodingFormat, WithInputType, WithProviderPreferences, etc.)
- `pkg/embeddings/openrouter/provider.go` - ProviderPreferences struct with custom MarshalJSON and Extras

## Decisions Made
- Follow Together provider pattern for standalone OpenRouter package - no dependency on openai package
- WithModelString bypasses validation for proxy-compatible model names while WithModel retains strict validation

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed gci import ordering lint error**
- **Found during:** Task 2 (OpenRouter package creation)
- **Issue:** Import grouping did not match gci linter expectations
- **Fix:** Ran `make lint-fix` to auto-format imports
- **Files modified:** pkg/embeddings/openrouter/openrouter.go
- **Verification:** `make lint` passes with 0 issues
- **Committed in:** 6d827d5 (part of Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Minor formatting fix, no scope change.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- OpenRouter provider package ready for unit and integration tests in Plan 02
- WithModelString available for any OpenAI-compatible proxy usage

## Self-Check: PASSED

All 5 files verified present. Both task commits (4f4933d, 6d827d5) verified in git log.

---
*Phase: 15-openrouter-embeddings-compatibility*
*Completed: 2026-03-30*
