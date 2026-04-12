---
phase: 24-getorcreatecollection-ef-safety
plan: 01
subsystem: api
tags: [go, v2, embedded, getorcreatecollection, lifecycle]
requires:
  - phase: 23-ort-ef-leak-fix
    provides: tracked SDK-owned default dense EF provenance and embedded reuse guards
provides:
  - preserves caller-provided dense/content EFs across provisional embedded GetCollection failures before GetOrCreate fallback
  - gates collection-state cleanup on explicit dense/content ownership instead of unconditionally closing provisional wrappers
  - adds deterministic fallback and concurrent -race regressions for embedded GetOrCreateCollection EF lifecycle safety
affects: [embedded, getcollection, getorcreatecollection, lifecycle]
tech-stack:
  added: []
  patterns: [ownership-aware provisional state, conditional convergence on embedded reuse, focused race regressions]
key-files:
  created: []
  modified:
    - pkg/api/v2/client_local_embedded.go
    - pkg/api/v2/close_logging.go
    - pkg/api/v2/client_local_embedded_test.go
key-decisions:
  - "Borrowed-vs-owned EF provenance is recorded inside the GetCollection state lock so failure cleanup can distinguish caller EFs from SDK-owned auto-wired/default EFs."
  - "Concurrent GetOrCreateCollection losers reload authoritative winner state when available, but still keep the temporary fallback EF on the accepted empty/no-config ambiguity branch."
  - "State-layer cleanup ownership remains separate from embeddedCollection.ownsEF so provisional cleanup and normal Close lifecycles can evolve independently."
patterns-established:
  - "Embedded provisional state stores close-once wrappers plus explicit ownership bits before revalidation."
  - "Fallback/race bug fixes stay narrow: targeted state cleanup changes, colocated lifecycle regressions, and repo-wide verification."
requirements-completed: [EFL-02, EFL-03]
duration: 13m
completed: 2026-04-12
---

# Phase 24 Plan 01: GetOrCreateCollection EF Safety Summary

**Embedded `GetOrCreateCollection(...)` now keeps caller-provided dense/content EFs usable across provisional get failures and returns usable collections under concurrent miss/create races.**

## Performance

- **Duration:** 13m
- **Started:** 2026-04-12T17:31:16+03:00
- **Completed:** 2026-04-12T17:44:16+03:00
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Added ownership-aware provisional embedded state so failed `GetCollection(...)` cleanup closes only SDK-owned dense/content EF paths.
- Preserved caller EF inputs through the `GetOrCreateCollection(...)` fallback and embedded reuse/reload branches, including shared dual-interface content EF handling.
- Added deterministic fallback and concurrent `-race` regressions, then reran the repo-wide test and lint gates successfully.

## Task Commits

Each task was committed atomically:

1. **Task 1: Make provisional embedded state ownership-aware and keep caller EFs open across fallback** - `ac96ab9` (test), `7fbb8fa` (fix)
2. **Task 2: Cover concurrent GetOrCreateCollection race convergence and cleanup regressions** - `8779913` (test), `80703c8` (test)

## Files Created/Modified

- `pkg/api/v2/client_local_embedded.go` - tracks dense/content ownership on provisional state, forwards explicit caller EFs through fallback/reuse branches, and promotes successful handoffs into owned lifecycle state.
- `pkg/api/v2/close_logging.go` - adds ownership-gated dense/content cleanup while preserving the shared dense/content single-close rule.
- `pkg/api/v2/client_local_embedded_test.go` - adds fallback and concurrent race regressions plus ownership-flag updates for existing cleanup tests.

## Decisions Made

- Kept the fix at the embedded provisional-state boundary instead of patching only `GetOrCreateCollection(...)`.
- Promoted successful revalidated handoffs back to owned state so client shutdown and collection close semantics remain unchanged after the provisional window.
- Reused the existing close-once wrappers and shared dense/content detection instead of introducing new lifecycle wrapper types.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The initial `gsd-executor` handoff stalled without producing artifacts, so the plan was executed inline against the approved Phase 24 plan and existing branch commits.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 24 satisfies `EFL-02` and `EFL-03` and leaves the embedded fallback/race path covered by deterministic regressions.
- `24-REVIEW.md` records one advisory follow-up about restoring prior owned state when a revalidation failure happens after an explicit override on an already-known collection.

## Verification Evidence

- `go test -race -tags=basicv2 -run 'TestEmbeddedLocalClientGetOrCreateCollection_ConcurrentRaceReturnsUsableCollection|TestEmbeddedLocalClientGetOrCreateCollection_FallbackAfterProvisionalGetFailureKeepsCallerEFOpen' ./pkg/api/v2/...` -> passed (`ok github.com/amikos-tech/chroma-go/pkg/api/v2 1.689s`).
- `make test` -> passed (`DONE 1783 tests, 7 skipped`).
- `make lint` -> passed (`0 issues.`).

## Self-Check: PASSED

- Verified `.planning/phases/24-getorcreatecollection-ef-safety/24-01-SUMMARY.md` exists on disk.
- Verified task commits `ac96ab9`, `7fbb8fa`, `8779913`, and `80703c8` exist in git history.
