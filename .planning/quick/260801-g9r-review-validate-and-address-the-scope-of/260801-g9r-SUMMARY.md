---
phase: quick-260801-g9r
plan: 01
status: complete
subsystem: api
tags: [go, search, rank, validation, tdd]

requires:
  - phase: quick-260731-pb1
    provides: Recursive rank marshaling guards for nils, depth, expression terms, and RRF
provides:
  - Eager validation of exact built-in Rank trees before SearchRequest assignment
  - KNN query, key, and limit validation at the serialization boundary
  - Eager recursive validation of non-nil SearchFilter Where trees
  - Public Collection.Search proof that invalid options never reach HTTP transport
affects: [pkg/api/v2, search-options, GH-516]

tech-stack:
  added: []
  patterns: [validate-before-assignment, exact-built-in compatibility boundary, transport no-send regression]

key-files:
  created: []
  modified:
    - pkg/api/v2/rank.go
    - pkg/api/v2/search.go
    - pkg/api/v2/rank_test.go
    - pkg/api/v2/search_test.go
    - pkg/api/v2/collection_http_test.go

key-decisions:
  - "Validate exact SDK Rank types through marshalRank while leaving the public Rank interface unchanged."
  - "Validate only non-nil SearchFilter.Where trees so empty, IDs-only, nil, and typed-nil filters remain compatible."
  - "Keep Collection.Search and HTTP error wrapping unchanged; pin option-stage failure with a zero-request regression."

patterns-established:
  - "Option validation precedes request mutation."
  - "Exact concrete-type dispatch preserves caller-defined interface implementations."

requirements-completed: [GH-516]
integrated-commit: 5dff7609f5be6046fe206f5d6c0dd38671dc33fe

duration: 11 min
completed: 2026-08-01
---

# Quick Task 260801-g9r: Search Option Validation Summary

**Built-in rank trees and non-nil search-filter trees now fail at option application, before request mutation or HTTP transport, without changing custom Rank compatibility.**

## Status

Complete — both TDD tasks and all plan-level verification commands passed.

## Performance

- **Duration:** 11 min
- **Started:** 2026-08-01T09:10:35Z
- **Completed:** 2026-08-01T09:20:47Z
- **Tasks:** 2/2
- **Code files modified:** 5

## Accomplishments

- Added `KnnRank.Validate` for non-nil supported queries, non-empty keys, and positive limits while preserving valid text, dense-vector, and sparse-vector JSON.
- Added one exact-built-in `validateBuiltInRank` boundary backed by `marshalRank`, preserving the public `Rank` interface and valid caller-defined implementations.
- Made `WithRank` and `WithSearchFilter` validate before assignment, preserving existing request state after every validation error.
- Proved through `Collection.Search` that invalid rank and filter options return as option-application errors with zero HTTP requests.

## Task Commits

Each TDD gate was committed atomically on the isolated execution branch before squash integration:

1. **Task 1 RED: rank option regressions** — `07c9ef7` (test)
2. **Task 1 GREEN: built-in rank validation** — `57178ea` (fix)
3. **Task 2 RED: filter option regressions** — `fb3f5f3` (test)
4. **Task 2 GREEN: eager filter validation** — `b0c4a13` (fix)

The orchestrator squash-integrates these commits and owns all planning artifacts.

**Integrated squash commit:** `5dff7609f5be6046fe206f5d6c0dd38671dc33fe`

## Files Created/Modified

- `pkg/api/v2/rank.go` — KNN structural validation and exact-built-in recursive validation boundary.
- `pkg/api/v2/search.go` — validate-before-assignment behavior and public option documentation.
- `pkg/api/v2/rank_test.go` — invalid KNN cases and valid text/dense/sparse wire-format regressions.
- `pkg/api/v2/search_test.go` — built-in rank, custom rank, filter compatibility, mutation, and append timing regressions.
- `pkg/api/v2/collection_http_test.go` — public search no-send/error-attribution regression.

## Decisions Made

- Reused `marshalRank` instead of adding another recursive tree walker.
- Used an exact concrete-type switch so standalone caller-defined ranks are not eagerly marshaled and require no new method.
- Kept `ErrNilRank` and `ErrNilFilter` checks first and assigned fields only after validation succeeded.
- Left `CollectionImpl.Search`, `APIClientV2.ExecuteRequest`, and HTTP error wrapping untouched.

## TDD Gate Compliance

| Task | RED | GREEN | REFACTOR | Status |
|------|-----|-------|----------|--------|
| Built-in rank validation | `07c9ef7` | `57178ea` | Not needed | Pass |
| Search-filter validation and no-send proof | `fb3f5f3` | `b0c4a13` | Not needed | Pass |

## Verification

- `go test -tags=basicv2 ./pkg/api/v2 -run '^(TestKnnRankValidation|TestWithRank|TestWithRankNilNonPointerImplementation|TestSearchFilter|TestTypedNilWhereBypassesOptionPipeline|TestCollectionSearchRejectsInvalidOptionsBeforeSend)$' -count=1 -timeout=60s` — PASS (`ok`, 0.450s).
- `go test -tags=basicv2 ./pkg/api/v2 -count=1 -timeout=120s` — PASS (`ok`, 38.043s).
- `make lint` — PASS (`0 issues`).
- `git diff --check 2f73fc2f7f21259adeddc6d2321cdb76d4fb9129..HEAD` — PASS.
- Diff guard for `pkg/api/v2/collection_http.go` and `pkg/api/v2/client.go` — PASS; both remain unchanged.
- Scope audit — PASS; the code diff contains only the five planned files.

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

- The package exposes `embeddings.KnnVector` but has no concrete dense implementation available to this unit test. A minimal test-only `denseKnnVector` exercised the public constructor contract without production changes.

## Known Stubs

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

GH-516 is covered by focused option-boundary and public request-path regressions. No blockers remain.

## Self-Check: PASSED

All five modified code files and the summary exist, all four TDD commits were created on the isolated branch, and the worktree remained on the expected branch and base.

---
*Quick task: 260801-g9r*
*Completed: 2026-08-01*
