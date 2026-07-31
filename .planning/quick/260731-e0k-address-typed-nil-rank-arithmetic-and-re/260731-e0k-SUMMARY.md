---
phase: quick-260731-e0k
plan: 01
status: complete
subsystem: api
tags: [go, rank, typed-nil, search, rrf, testing]

requires:
  - phase: quick-260730-u7z
    provides: Typed-nil search normalization
provides:
  - Typed-nil Rank operands fail cleanly through the existing UnknownRank marshal error
  - Exact HTTP payload coverage for direct nil, nested RRF, and mixed-batch searches
  - Recursive pointer-independence and JSON-equivalence coverage for nested RRF clones
  - Corrected 260730-u7z quick-task ledger entry
affects: [rank arithmetic, Search API, RRF cloning]

tech-stack:
  added: []
  patterns:
    - Guard typed-nil interface values at the shared operand conversion boundary
    - Assert complete HTTP JSON payloads instead of shallow field presence

key-files:
  created: []
  modified:
    - pkg/api/v2/rank.go
    - pkg/api/v2/rank_test.go
    - pkg/api/v2/collection_http_test.go
    - .planning/STATE.md

key-decisions:
  - "Reused isNilRank in operandToRank so all six fluent binary operations share one typed-nil guard."
  - "Preserved operandToRank(nil) as Val(0); only typed-nil Rank values become UnknownRank."
  - "Kept cloneRank unchanged because focused recursive tests confirmed its existing deep-copy behavior."

requirements-completed:
  - REVIEW-01
  - REVIEW-02
  - REVIEW-03
  - REVIEW-04
  - REVIEW-05
  - REVIEW-06
  - REVIEW-07

duration: 7 min
completed: 2026-07-31
---

# Quick Task 260731-e0k: Typed-Nil Rank Arithmetic and Search Regressions

A shared `isNilRank` guard turns typed-nil arithmetic operands into stable marshal errors, backed by exact search payload and recursive RRF clone regressions.

## Accomplishments

- All six binary fluent rank operations reject a typed-nil Rank during marshal without panicking.
- Search regressions pin complete JSON for direct typed nil, nested RRF, and ordered mixed-batch requests.
- A three-level RRF tree clones into independent nodes and preserves complete marshal output.
- The `260730-u7z` ledger row now names reachable commit `b79ede4` with status `Verified`.

## TDD Execution

- **RED:** Added a six-operation regression and confirmed each case reached a nil `*KnnRank` receiver and panicked.
- **GREEN:** Added the shared `isNilRank` guard in `operandToRank`; the focused regression passed.
- **REFACTOR:** No further production change was needed.

## Commit

- `b3ef969` — `fix(api): handle typed-nil rank operands`

## Files Modified

- `pkg/api/v2/rank.go` — Detects typed-nil Rank operands at the shared conversion boundary.
- `pkg/api/v2/rank_test.go` — Covers Add, Multiply, Sub, Div, Max, and Min marshal behavior.
- `pkg/api/v2/collection_http_test.go` — Pins exact search payloads and recursive RRF clone behavior.
- `.planning/STATE.md` — Corrects the earlier ledger row and records this quick task.

## Decisions

- Reused the existing nillable-kind-aware helper instead of adding reflection or per-operation guards.
- Kept untyped nil's established `Val(0)` fluent-chain behavior unchanged.
- Added test-only recursive clone coverage because the production clone already performs the required deep copy.
- Left PR-branch filtering out of scope because no PR operation was requested.

## Verification

- `go test -tags=basicv2 ./pkg/api/v2 -run '^TestRankArithmeticTypedNilOperandMarshal$' -count=1` — passed.
- `go test -tags=basicv2 ./pkg/api/v2 -run '^(TestCollectionSearchTypedNilRank|TestCloneRank)$' -count=1` — passed.
- `go test -tags=basicv2 ./pkg/api/v2/... -count=1` — passed.
- `make lint` — passed with `0 issues`.
- `git diff --check` — passed.
- Ledger verification confirmed `b79ede4` is reachable, the earlier row is `Verified`, and the stale hash is absent.

## Deviations

None.

## User Setup Required

None.

## Self-Check

Passed. The implementation commit is reachable on the feature branch, the requested regressions pass, and no PR or merge operation outside the isolated squash integration was performed.
