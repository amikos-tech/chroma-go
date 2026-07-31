---
quick_task: 260731-pb1
includes: [260731-nj1]
status: passed
verified: 2026-07-31
commit: 21cc467
requirements: [GH-528]
---

# Rank Marshal Depth Guard Verification

## Goal

Bound recursive JSON marshaling for built-in Rank expressions without changing accepted JSON or breaking caller-defined Rank implementations.

## Verdict

Passed. Built-in composite Rank values preserve zero-based depth accounting and reject expressions beyond MaxExpressionDepth. External Rank implementations continue to use their own MarshalJSON methods, including types that embed a built-in composite.

## Acceptance Evidence

- `TestRankMarshalExpressionDepthGuard` covers accepted and rejected boundaries for Sum, Sub, Mul, Div, Abs, Exp, Log, Max, Min, and RRF.
- `TestCustomRankEmbeddingCompositePreservesMarshalJSON` runs from the external `v2_test` package and proves an embedded SumRank does not capture custom JSON serialization.
- `TestMarshalRankFallbackForNestedNonCompositeChild` proves caller-defined ranks nested inside built-in expressions still use the fallback.
- `TestNilOptionSentinels` confirms nil Rank behavior and sentinels remain intact.

## Fresh Verification

- `go test -count=1 -tags=basicv2 -run 'Test(CustomRankEmbeddingCompositePreservesMarshalJSON|MarshalRankFallbackForNestedNonCompositeChild|RankMarshalExpressionDepthGuard|NilOptionSentinels)$' ./pkg/api/v2` — passed (`ok`, 0.534s).
- `make test` — passed: 1,981 tests, 7 expected skips, 34.213s.
- `make lint` — passed with 0 issues.
- `git diff --check` — passed.

## Scope Check

- Production behavior changes are limited to Rank JSON depth dispatch.
- Tests cover internal boundary behavior and the public external-implementation contract.
- No dependency changes were introduced.
- `pkg/api/v2/collection_http.go`, `go.mod`, and `go.sum` remain unchanged.

## Follow-up

`cloneRank` has an independent recursive traversal and remains outside this change. Expressions deeper than 100 now return a normal error instead of continuing recursive built-in JSON marshaling.
