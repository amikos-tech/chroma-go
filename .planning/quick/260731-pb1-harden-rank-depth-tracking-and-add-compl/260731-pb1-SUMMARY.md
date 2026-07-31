---
phase: quick-260731-pb1
plan: "01"
status: complete
subsystem: api
tags: [go, json, rank, recursion, depth-guard]

requires:
  - phase: quick-260731-nj1
    provides: "Initial bounded recursive marshaling for built-in Rank expressions"
provides:
  - "Shared depth-aware marshaling for every built-in Rank implementation"
  - "Boundary regressions for all ten composite Rank serializers"
  - "RRF depth failures with RrfRank-specific error context"
  - "Compatibility for external Rank implementations that embed built-in composites"
affects: [v2-rank-expressions, v2-search]

tech-stack:
  added: []
  patterns:
    - "Exact built-in composite dispatch keeps depth accounting private without sealing Rank"
    - "Leaf and caller-defined Rank implementations retain the plain MarshalJSON fallback"

key-files:
  created:
    - pkg/api/v2/rank_external_test.go
  modified:
    - pkg/api/v2/rank.go
    - pkg/api/v2/rank_test.go
    - pkg/api/v2/search_test.go

key-decisions:
  - "Kept depth-aware marshaling private and dispatches exact built-in composite types so external Rank implementations retain their MarshalJSON behavior"
  - "Kept leaves on their existing MarshalJSON methods to preserve encoding and validation behavior"
  - "Kept RRF's generated expression at depth+1 and wrapped only its final marshaling error"

requirements-completed: [GH-528]

duration: 7 min
completed: 2026-07-31
---

# Quick Task 260731-pb1: Rank Depth Contract Hardening Summary

**Every built-in Rank serializer now participates in one shared depth limit, while external Rank implementations retain their existing marshaling contract.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-07-31T15:22:49Z
- **Completed:** 2026-07-31T15:29:22Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Added private depth-aware marshaling for every built-in composite with exact-type dispatch and an external implementation fallback.
- Exercised Sum, Sub, Mul, Div, Abs, Exp, Log, Max, Min, and RRF through public construction paths at their exact accepted and rejected child-depth boundaries.
- Preserved RRF's hidden four-level expansion accounting while adding cannot marshal RrfRank context to final generated-expression failures.
- Added an external-package regression proving that embedding a built-in composite does not override a caller's custom MarshalJSON method.

## TDD Cycle

- **RED:** The expanded focused test failed only for rrf/rejects_next_child_depth: the shared depth error did not yet contain cannot marshal RrfRank.
- **GREEN:** Wrapping the error from marshalRank(result, depth+1) made all composite boundaries pass while preserving successful RRF bytes.
- **COMPATIBILITY:** Review exposed that putting a package-private method on Rank prevented external implementations and that interface-based dispatch could capture embedded built-ins. Exact concrete-type dispatch fixes both cases.

## Task Commits

1. **Harden built-in Rank depth tracking** - 3243aa4 (fix)
2. **Restore the private depth-aware marshal interface** - d150679 (fix)
3. **Tighten depth-guard docs, tests, and consistency** - 0316b27 (fix)
4. **Preserve embedded custom marshaling** - 21cc467 (fix)

## Files Created/Modified

- pkg/api/v2/rank.go - Guards every exact built-in composite, preserves the leaf/external fallback, and contextualizes final RRF depth failures.
- pkg/api/v2/rank_test.go - Covers both sides of every composite boundary, including RRF inner depths 96 and 97.
- pkg/api/v2/search_test.go - Documents and tests the package-local caller-defined Rank fallback.
- pkg/api/v2/rank_external_test.go - Proves an external type embedding SumRank keeps its custom MarshalJSON output.

## Decisions Made

- Kept nil-rank validation ahead of the depth check, retaining ErrNilRank precedence.
- Kept zero-based root accounting and MaxExpressionDepth == 100.
- Kept existing leaf encoders rather than duplicating JSON or validation logic.
- Kept Rank externally implementable and matched exact built-in composite types before falling back to caller-defined MarshalJSON methods.

## Deviations from Plan

Post-plan compatibility review found that the exhaustive private method on Rank sealed the public interface and that interface-based dispatch could intercept external types embedding built-in composites. The implementation now uses exact built-in type dispatch and preserves the documented external fallback.

## Issues Encountered

None. The expected RED failure confirmed that the new RRF context assertion detected the intended missing behavior.

## Verification

- Focused depth, fallback, external-embedding, and nil-sentinel regressions passed (ok, 0.534s).
- Full basicv2 suite passed: 1,981 tests, 7 expected skips, in 34.213s.
- make lint passed with 0 issues.
- git diff --check passed with no whitespace errors.
- pkg/api/v2/collection_http.go, go.mod, and go.sum are unchanged; no external issue was created.

## User Setup Required

None - no dependency or external service changes.

## Next Phase Readiness

- Expressions deeper than 100 now return an error even if the former fallback path allowed them to marshal.
- External Rank implementations, including types embedding built-in composites, keep control of their MarshalJSON output.
- cloneRank still performs independent recursive traversal; it remains a follow-up concern only, with no code TODO or external issue created in this task.
- Ready for final planning metadata to be committed.

## Self-Check: PASSED

- All four code/test files and this summary exist.
- Compatibility commit 21cc467 follows the original hardening and review-fix commits on the PR branch.
- STATE.md, ROADMAP.md, REQUIREMENTS.md, the plan, and collection_http.go remain unchanged by the executor.

---
*Quick task: 260731-pb1*
*Completed: 2026-07-31*
