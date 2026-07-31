---
phase: quick-260731-ewm
plan: 01
status: complete
subsystem: api
tags: [go, rank, typed-nil, rrf, serialization, testing]

requires:
  - phase: quick-260731-e0k
    provides: Initial typed-nil arithmetic guard and search-path regressions
provides:
  - ErrNilRank failures instead of panics at every composite rank marshal boundary
  - Nested typed-nil RRF identity preservation through cloning and rejection before HTTP transmission
  - Bounded HTTP request-body regression synchronization
  - Corrected historical receiver and nested-RRF contract records
affects: [rank arithmetic, Search API, RRF validation, rank cloning]

tech-stack:
  added: []
  patterns:
    - Route composite child serialization through one nil-aware marshalRank helper
    - Preserve nested invalid interface identity until validation while omitting direct optional nil ranks

key-files:
  created: []
  modified:
    - pkg/api/v2/rank.go
    - pkg/api/v2/rank_test.go
    - pkg/api/v2/options.go
    - pkg/api/v2/search.go
    - pkg/api/v2/collection_http.go
    - pkg/api/v2/collection_http_test.go
    - pkg/api/v2/search_test.go
    - .planning/quick/260731-e0k-address-typed-nil-rank-arithmetic-and-re/260731-e0k-PLAN.md
    - .planning/quick/260731-e0k-address-typed-nil-rank-arithmetic-and-re/260731-e0k-SUMMARY.md

key-decisions:
  - "Use one marshalRank helper at the nine composite serialization boundaries instead of adding guards to every fluent method."
  - "Keep untyped nil Operand conversion as Val(0), return ErrNilRank for typed-nil Rank operands, and retain UnknownRank only for unsupported Operand implementations."
  - "Keep direct SearchRequest.Rank nil or typed nil as optional omission, while preserving nested typed nils so RRF validation rejects them."
  - "Keep isNilRank and isNilInterface in search.go to avoid unrelated helper relocation."

patterns-established:
  - "Composite serialization boundary: validate a child interface before invoking its MarshalJSON method."
  - "Clone semantics: preserve nested invalid dynamic types when later validation must distinguish them from omitted top-level fields."

requirements-completed:
  - EWM-01
  - EWM-02
  - EWM-03
  - EWM-04
  - EWM-05
  - EWM-06
  - EWM-07
  - EWM-08

duration: 9 min
completed: 2026-07-31
---

# Quick Task 260731-ewm: Typed-Nil Composite Rank and Nested RRF Safety Summary

**Nil-aware composite serialization now returns `ErrNilRank` without panics or nested RRF score substitution, while direct optional rank omission remains intact.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-07-31T07:55:19Z
- **Completed:** 2026-07-31T08:04:07Z
- **Tasks:** 3
- **Files modified:** 9

## Accomplishments

- Routed all nine composite rank serializers through one typed-nil-aware child marshal helper.
- Preserved nested typed-nil rank identity through recursive cloning so RRF validation returns `ErrNilRank` before any HTTP request is sent.
- Kept untyped nil operands as `Val(0)`, unsupported operands as `UnknownRank`, and direct typed-nil `SearchRequest.Rank` values as omission.
- Replaced blocking HTTP capture receives with bounded assertions and corrected the earlier quick-task receiver and RRF contract record.

## TDD Execution

- **Task 1 RED:** The focused suite failed with `UnknownRank` instead of `ErrNilRank`, panicked when composite serializers invoked typed-nil children, and accepted nil RRF children.
- **Task 1 GREEN:** `marshalRank`, accurate operand conversion, and RRF validation made all receiver, composite, conversion, and RRF regressions pass.
- **Task 2 RED:** Clone regressions failed because typed-nil `*KnnRank` and `*ValRank` values were converted to untyped nil.
- **Task 2 GREEN:** Returning the incoming nil interface from `cloneRank` preserved concrete type identity and made the focused search/clone suite pass.
- **REFACTOR:** HTTP capture setup was isolated per subtest and all waits were bounded with `time.After(time.Second)`.

## Task Commits

1. **Task 1: Reject nil children at every composite marshal boundary** — `584d073` (`fix`, isolated execution)
2. **Task 2: Preserve nested typed nils and fail RRF searches cleanly** — `58f3b04` (`fix`, isolated execution)
3. **Task 3: Correct the earlier GSD receiver claim and mark the RRF contract superseded** — included in the final GSD documentation commit

The two isolated implementation commits were squash-integrated as `b0f52ed`.

## Files Created/Modified

- `pkg/api/v2/rank.go` — Adds the shared marshal guard, accurate operand conversion, and nil-aware RRF validation.
- `pkg/api/v2/rank_test.go` — Covers all typed-nil receiver compositions, every composite shape, RRF validation, and distinct operand errors.
- `pkg/api/v2/options.go` — Broadens the public `ErrNilRank` contract comment.
- `pkg/api/v2/search.go` — Documents option rejection versus direct optional-field omission.
- `pkg/api/v2/collection_http.go` — Preserves nested typed-nil dynamic types during deep cloning.
- `pkg/api/v2/collection_http_test.go` — Proves clean nested-RRF failure, exact successful payloads, and bounded request capture.
- `pkg/api/v2/search_test.go` — Proves typed-nil clone identity and direct omission behavior.
- Earlier `260731-e0k` PLAN/SUMMARY — Corrects the valid receiver description and marks the former nested-RRF substitution contract superseded.

## Decisions Made

- Kept the fix at serialization and validation boundaries, which covers all composite shapes without duplicating guards across fluent methods.
- Preserved the intentional asymmetry: `WithRank` rejects nil input, but direct assignment to the optional request field is omitted.
- Kept historical changes factual: the earlier commit remains recorded as-is, while the later contract change is explicitly attributed to this task.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Verification

- Task 1 focused suite — passed: `ok github.com/amikos-tech/chroma-go/pkg/api/v2 0.447s`.
- Task 2 focused suite — passed: `ok github.com/amikos-tech/chroma-go/pkg/api/v2 1.364s`.
- Full basicv2 package suite — passed: `ok github.com/amikos-tech/chroma-go/pkg/api/v2 23.156s`.
- `make lint` — passed with `0 issues`.
- Historical record phrase and supersession checks — passed.
- `git diff --check` for committed code and uncommitted planning changes — passed.

## User Setup Required

None.

## Next Phase Readiness

- The implementation was squash-integrated as `b0f52ed`, and the quick-task ledger records that reachable commit.
- No follow-up implementation or user setup is required.

## Self-Check: PASSED

All nine planned modified files and the new summary exist, the isolated execution commits were squash-integrated as `b0f52ed`, no tracked files were deleted, no stub markers were introduced, and all diffs pass whitespace validation.
