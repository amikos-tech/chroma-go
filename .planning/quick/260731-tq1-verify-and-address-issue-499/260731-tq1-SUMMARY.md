---
phase: quick-260731-tq1
plan: 01
status: complete
subsystem: api
tags: [go, rank, rrf, nil-handling]

requires: []
provides:
  - "Fail-closed conversion of untyped nil Rank operands through ErrNilRank"
  - "Regression coverage for all six RrfRank binary methods and four self-flattening methods"
affects: [v2-rank-api, GH-499]

tech-stack:
  added: []
  patterns: ["Untyped and typed nil Rank operands share the ErrNilRank path"]

key-files:
  created: []
  modified:
    - CHANGELOG.md
    - pkg/api/v2/rank.go
    - pkg/api/v2/rank_test.go

key-decisions:
  - "Return a nil Rank from operandToRank for both untyped and typed nil so callers receive the existing matchable ErrNilRank; reserve UnknownRank for unsupported operand implementations."

patterns-established:
  - "Explicit nil Operand values fail during composite marshaling with ErrNilRank regardless of whether the nil is typed."

requirements-completed: [GH-499]

duration: 3 min
completed: 2026-07-31
---

# Quick Task 260731-tq1: Verify and Address Issue 499 Summary

**Untyped and typed nil rank operands now share the matchable `ErrNilRank` path, so arithmetic expressions fail visibly during marshaling without panics, misleading errors, or silent zero substitution.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-07-31T18:29:42Z
- **Completed:** 2026-07-31T18:32:54Z
- **Tasks:** 1
- **Implementation files modified:** 3

## Accomplishments

- Changed only the shared `operandToRank(nil)` branch to return a nil Rank, reusing the composite marshal guard and `ErrNilRank`.
- Added no-panic marshal regressions for all six RRF binary methods and the self-flattening `SumRank.Add`, `MulRank.Multiply`, `MaxRank.Max`, and `MinRank.Min` paths.
- Preserved existing integer, float, non-nil Rank, and unsupported-operand conversion behavior; `UnknownRank` remains the sentinel for unsupported implementations only.
- Updated the public Rank and conversion comments to describe the fail-closed contract.
- Documented the user-visible nil arithmetic behavior in the v0.4.2 changelog.

## Task Commits

The initial isolated RED/GREEN execution was squash-integrated into the PR branch, followed by the review correction:

- **Fail closed for nil rank operands and add regressions:** `84797b1`
- **Unify untyped and typed nil errors and close review gaps:** `96fade3`

## Files Created/Modified

- `CHANGELOG.md` - Documents the v0.4.2 Rank API behavior change.
- `pkg/api/v2/rank.go` - Routes untyped nil operands to the existing `ErrNilRank` composite path and documents the unified contract.
- `pkg/api/v2/rank_test.go` - Pins direct conversion, all six public RRF binary operations, and all four self-flattening branches.

## Decisions Made

- Reused the existing nil-child marshal guard rather than adding per-method checks or a new public error.
- Unified untyped and typed nil operands under `ErrNilRank`, which is both matchable with `errors.Is` and accurately describes an absent operand.
- Left `UnknownRank.UnmarshalJSON` unchanged because that cosmetic promoted-method behavior is not reachable through the public conversion API.

## TDD Evidence

- **RED:** Before the production change, direct conversion was not `*UnknownRank`; five RRF operations emitted JSON containing zero; `Div(nil)` returned the unrelated division-by-zero error.
- **INITIAL GREEN:** The first implementation made nil fail through `UnknownRank`.
- **REVIEW RED:** Ten focused cases proved that six RRF operations and four flattening paths returned the non-matchable unknown-operand error instead of `ErrNilRank`.
- **FINAL GREEN:** The same focused cases passed after untyped nil joined the typed-nil path.

## Verification

Fresh checks on PR-branch commit `96fade3`:

- `go test -tags=basicv2 ./pkg/api/v2 -run '^(TestOperandConversion|TestRankUntypedNilOperandMarshal|TestRankArithmeticTypedNilOperandMarshal|TestUnknownRankError)$' -count=1 -timeout=30s` - PASS
- `go test -tags=basicv2 ./pkg/api/v2/... -count=1 -timeout=120s` - PASS (`23.859s`)
- `make lint` - PASS (`0 issues`)
- `git diff --check` - PASS

## Deviations from Plan

Post-execution review changed the error sentinel from `UnknownRank` to a nil Rank. The existing composite marshal guard safely converts both untyped and typed nil to `ErrNilRank`, giving callers one accurate, `errors.Is`-matchable contract. Review also added the v0.4.2 changelog entry and four self-flattening regression cases.

## Issues Encountered

None. The initial failing test output was the required TDD RED gate.

## User Setup Required

None.

## Next Phase Readiness

- GH-499 is implemented and locally verified on `fix/issue-499`.
- The branch is ready for PR publication; the eventual PR must use squash merge.

## Self-Check: PASSED

- Confirmed all three implementation/documentation files and this summary exist.
- Confirmed review commit `96fade3` exists on the PR branch.

---
*Phase: quick-260731-tq1*
*Completed: 2026-07-31*
