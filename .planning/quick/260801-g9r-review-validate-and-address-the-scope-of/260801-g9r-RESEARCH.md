# Quick Task 260801-g9r: Issue 516 Research

**Researched:** 2026-08-01
**Domain:** Go API v2 Search option validation
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Rank validation contract
- Validate built-in rank structures recursively when `WithRank` is applied.
- Preserve compatibility with caller-defined `Rank` implementations; do not add a required method to the public `Rank` interface.
- Invalid input must be rejected before it is assigned to the request.

#### Filter behavior
- Eagerly validate every non-nil `SearchFilter.Where` clause when `WithSearchFilter` is applied, including nested clauses.
- Keep an empty `SearchFilter` valid.
- Keep IDs-only filters valid.
- Preserve existing nil and typed-nil `Where` behavior unless current documented behavior requires a validation error.

#### Error attribution scope
- Make eager option validation and regression tests the required scope.
- During research, inspect the HTTP-layer error wrapping.
- Include a narrowly scoped wrapper improvement only if it cleanly distinguishes pre-send serialization failures without changing public behavior; otherwise record it as out of scope.

### Agent's Discretion
- Choose the smallest internal validation design that covers all built-in rank forms recursively and avoids duplicate validation logic.
- Select focused unit and request-path tests that prove failure timing and prevent request mutation on validation errors.
</user_constraints>

## Project Constraints

- Do not include private-repository or internal-company information in commits, pull requests, or related artifacts.
- Use squash merge.
- Prefer the least code necessary; do not add a dependency for this fix.
- Explain behavior in plain language with small examples.

## Summary

Issue 516 is still reproducible on current `main` (`da70881`): `WithRank` and `WithSearchFilter` reject only top-level nil, then assign without validating, while `WithGroupBy` and `WithSearchWhere` validate before mutation. The issue's cited client line has shifted by two lines, but the error flow is otherwise unchanged. [VERIFIED: `pkg/api/v2/search.go:408-413,642-647,697-705`; `pkg/api/v2/options.go:886-903`; `pkg/api/v2/client.go:1113-1115`; `pkg/api/v2/collection_http.go:426-449`]

**Primary recommendation:** add one unexported exact-built-in rank validation boundary that reuses the existing recursive marshal path, add structural validation to `KnnRank`, call that boundary before `req.Rank` assignment, and directly validate a non-nil `SearchFilter.Where` before `req.Filter` assignment. Leave HTTP error classification unchanged. [VERIFIED: `pkg/api/v2/rank.go:56-69,108-141,1081-1101,1240-1363`; `pkg/api/v2/search.go:408-413,642-647`]

No package, schema, persistence, runtime-state, or external-service change is needed. [VERIFIED: task scope and inspected integration points]

## Issue Claim Validation

| Issue 516 claim | Verdict | Current-main evidence |
|---|---|---|
| `groupByOption` nil-checks and validates before assignment. | Confirmed | `Validate` runs at lines 701-703 and assignment is line 704. [VERIFIED: `pkg/api/v2/search.go:697-705`] |
| `rankOption` only nil-checks. | Confirmed | A non-nil rank is assigned at line 646 with no intervening validation. [VERIFIED: `pkg/api/v2/search.go:642-647`] |
| `searchFilterOption` only nil-checks. | Confirmed | A non-nil filter is assigned at line 412 with no nested `Where` validation. [VERIFIED: `pkg/api/v2/search.go:408-413`] |
| `WithRank(&KnnRank{})` returns nil and the zero value serializes with empty key, zero limit, and null query. | Confirmed | The option accepts every non-nil rank; `KnnRank.MarshalJSON` currently accepts nil query and does not validate key or limit. [VERIFIED: `pkg/api/v2/search.go:642-647`; `pkg/api/v2/rank.go:996-1002,1081-1101`] |
| `WithSearchFilter(&SearchFilter{Where: EqString(K(""), "x")})` returns nil. | Confirmed | The option assigns immediately; the empty-key error is produced only when `SearchFilter.MarshalJSON` later calls `WhereClause.Validate`. [VERIFIED: `pkg/api/v2/search.go:225-256,408-413`; `pkg/api/v2/where.go:64-78`] |
| A nested invalid `Where` is already detected recursively once validation runs. | Confirmed | `WhereClauseWhereClauses.Validate` walks every child, rejects nil/typed-nil children, and returns the first child error. [VERIFIED: `pkg/api/v2/where.go:460-475`] |
| `WithRank(&RrfRank{})` accepts the option and later fails with `rrf k must be >= 1`. | Confirmed | `WithRank` does not validate; `RrfRank.MarshalJSON` calls `Validate`, whose first zero-value failure is `K < 1`. [VERIFIED: `pkg/api/v2/search.go:642-647`; `pkg/api/v2/rank.go:1240-1246,1306-1314`] |
| Serialization errors are wrapped as a send failure. | Confirmed | `ExecuteRequest` returns `error marshalling request JSON`, then `CollectionImpl.Search` wraps every executor error as `error sending search request`. [VERIFIED: `pkg/api/v2/client.go:1108-1116`; `pkg/api/v2/collection_http.go:446-449`] |
| `WithSearchWhere` is the eager-validation precedent. | Confirmed | It preserves nil/typed-nil omission, calls `Validate` for every non-nil clause, and assigns only after success. [VERIFIED: `pkg/api/v2/options.go:886-903`] |
| The suggested literal `rank.Validate()` call is sufficient. | Not literally implementable | Public `Rank` has no `Validate` method, and adding one would break every caller-defined implementation. Only `RrfRank` currently has `Validate`; `SearchFilter` also has no such method. [VERIFIED: `pkg/api/v2/rank.go:56-69,1240-1262`; `pkg/api/v2/search.go:200-208`] |

The public Chroma Search documentation also defines KNN `query` as required, with `key="#embedding"` and `limit=16` as defaults; the Go constructor supplies those defaults, while a direct zero-value struct bypasses them. [CITED: https://docs.trychroma.com/cloud/search-api/ranking]

## Implementation-Ready Design

### 1. Built-in rank validation

Keep `Rank` unchanged. Add an unexported helper in `rank.go`, adjacent to `marshalRank`, that exact-type-switches over every built-in rank type: `UnknownRank`, `ValRank`, `SumRank`, `SubRank`, `MulRank`, `DivRank`, `AbsRank`, `ExpRank`, `LogRank`, `MaxRank`, `MinRank`, `KnnRank`, and `RrfRank`. For an exact built-in, call `marshalRank(rank, 0)` and discard the bytes; for any other implementation, return nil. This reuses the existing nil, depth, term-count, division-by-zero, RRF, JSON-number, and recursive child checks instead of creating a second tree walker. [VERIFIED: `pkg/api/v2/rank.go:100-141,215-859,996-1101,1194-1363`]

Use exact concrete cases, matching `marshalRank`'s existing compatibility pattern. Do not use a generic `interface{ Validate() error }` assertion: a caller-defined rank may already expose a method with that name and must not acquire new validation behavior accidentally. An external type embedding a built-in must also remain under its own contract. [VERIFIED: `pkg/api/v2/rank.go:116-141`; `pkg/api/v2/rank_test.go:1028-1049`]

Add `(*KnnRank).Validate() error` and invoke it at the start of `KnnRank.MarshalJSON`. Move the existing query-type switch into it, reject nil and typed-nil queries, require a non-empty key, and require `Limit >= 1`. The constructor-created values remain valid because `NewKnnRank` sets `#embedding` and `16`; the direct zero value becomes invalid as required. Let JSON encoding continue to reject NaN/Inf values rather than duplicating encoder rules. [VERIFIED: `pkg/api/v2/rank.go:1017-1033,1081-1101`; query requirement cited at https://docs.trychroma.com/cloud/search-api/ranking]

Then make `rankOption.ApplyToSearchRequest` perform, in order:

```go
if isNilRank(o.rank) {
    return ErrNilRank
}
if err := validateBuiltInRank(o.rank); err != nil {
    return err
}
req.Rank = o.rank
return nil
```

This preserves the existing top-level `ErrNilRank` precedence and guarantees no request mutation on any validation error. [VERIFIED: `pkg/api/v2/search.go:642-647`; `pkg/api/v2/options.go:906-933`]

A valid custom rank used directly or inside a built-in expression must remain accepted; the existing `mapBackedRank` test double is suitable for both cases. [VERIFIED: `pkg/api/v2/search_test.go:1008-1027,1058-1075`; `pkg/api/v2/rank_test.go:1028-1049`]

### 2. Search filter validation

In `searchFilterOption.ApplyToSearchRequest`, keep `ErrNilFilter` first. If `o.filter.Where` is not nil according to `isNilInterface`, call its existing `Validate`; wrap the error with `invalid search filter` to preserve the context currently produced at marshal time. Assign `req.Filter` only after success. [VERIFIED: `pkg/api/v2/search.go:225-256,408-413,650-665`]

```go
if o.filter == nil {
    return ErrNilFilter
}
if !isNilInterface(o.filter.Where) {
    if err := o.filter.Where.Validate(); err != nil {
        return errors.Wrap(err, "invalid search filter")
    }
}
req.Filter = o.filter
return nil
```

One `Validate` call covers nested `And`/`Or` trees. Empty and IDs-only filters skip the call and remain valid. Untyped and typed-nil `Where` values also skip it, preserving their current empty/IDs-only serialization without mutating the caller-owned filter. [VERIFIED: `pkg/api/v2/search.go:225-256`; `pkg/api/v2/where.go:460-475`; `pkg/api/v2/search_test.go:1103-1143`]

### 3. HTTP error attribution

Do not change the HTTP wrapper in this task. With the option fixes, issue 516 inputs return from `CollectionImpl.Search` as `error applying search option: ...` before embedding, URL construction, serialization, or transport. [VERIFIED: `pkg/api/v2/collection_http.go:426-449`]

`ExecuteRequest` is the layer that knows whether failure happened during JSON encoding, request creation, or `http.Client.Do`, but it exposes those stages only through ordinary wrapped errors. A local `Search` fix would therefore require brittle string matching, incomplete checks for concrete JSON error types, duplicate pre-marshalling, or a wider shared error-classification contract. None is behavior-preserving or necessary to close the reported option paths. [VERIFIED: `pkg/api/v2/client.go:1108-1145`; `pkg/api/v2/collection_http.go:446-449`]

Direct `SearchRequest`/`SearchQuery` construction can still bypass option validation and retain the existing outer `error sending search request` wording; that is a separate shared-client error-taxonomy concern, not evidence for expanding this fix. [VERIFIED: exported fields at `pkg/api/v2/search.go:282-299`; custom collection options at `pkg/api/v2/search.go:359-363`; executor path at `pkg/api/v2/client.go:1108-1145`]

## Precise Test Plan

| File | Test | Required assertions |
|---|---|---|
| `pkg/api/v2/rank_test.go` | `TestKnnRankValidation` | Zero-value KNN fails; nil/typed-nil query, empty key, zero limit, and existing invalid query type fail; a constructor-built text rank and populated dense/sparse ranks still marshal. [VERIFIED: existing coverage patterns at `pkg/api/v2/rank_test.go:73-179,867-881`] |
| `pkg/api/v2/search_test.go` | Extend `TestWithRank` | `&KnnRank{}`, `&RrfRank{}`, a composite containing invalid `&KnnRank{}`, and a composite containing typed-nil rank fail before assignment; pre-seed `req.Rank` and assert it is unchanged; `errors.Is(err, ErrNilRank)` survives for nested nil. [VERIFIED: existing option tests at `pkg/api/v2/search_test.go:317-350`; nil guarantees at `pkg/api/v2/rank_test.go:342-500`] |
| `pkg/api/v2/search_test.go` | Extend custom-rank coverage | Non-nil `mapBackedRank` remains accepted directly and as a child of `Val(0).Add(custom)`. [VERIFIED: `pkg/api/v2/search_test.go:1008-1075`; `pkg/api/v2/rank_test.go:1028-1049`] |
| `pkg/api/v2/search_test.go` | Extend `TestSearchFilter` | Direct empty-key and nested empty-key `Where` fail; a pre-existing `req.Filter` is unchanged; empty, IDs-only, nil-Where, and typed-nil-Where filters are accepted and preserve current JSON. [VERIFIED: existing cases at `pkg/api/v2/search_test.go:185-224,441-530,1103-1143`] |
| `pkg/api/v2/search_test.go` | Extend `NewSearchRequest` tests | Invalid rank/filter options return an error and do not append to `SearchQuery.Searches`, matching existing nil-option behavior. [VERIFIED: `pkg/api/v2/search_test.go:217-223,334-340`; `pkg/api/v2/search.go:720-729`] |
| `pkg/api/v2/collection_http_test.go` | `TestCollectionSearchRejectsInvalidOptionsBeforeSend` | Table-test invalid rank and invalid filter through `Collection.Search`; assert `error applying search option`, assert it does not contain `error sending search request`, and assert an `httptest` request counter stays zero. [VERIFIED: request-path pattern at `pkg/api/v2/collection_http_test.go:706-803`; production order at `pkg/api/v2/collection_http.go:426-449`] |

Focused verification:

```bash
go test -tags=basicv2 ./pkg/api/v2 \
  -run '^(TestKnnRankValidation|TestWithRank|TestSearchFilter|TestCollectionSearchRejectsInvalidOptionsBeforeSend)$' \
  -count=1 -timeout=60s
```

Package gate:

```bash
go test -tags=basicv2 ./pkg/api/v2 -count=1 -timeout=120s
```

The relevant current-main baseline passed before implementation: `go test -tags=basicv2 ./pkg/api/v2` with the focused existing rank, filter, depth, and request-path tests completed successfully. [VERIFIED: local test run on 2026-08-01]

## Compatibility Hazards and Pitfalls

- **Do not add `Validate` to `Rank`.** It is a public, externally implementable interface; adding a method is a source break. [VERIFIED: `pkg/api/v2/rank.go:56-69`; caller implementation at `pkg/api/v2/search_test.go:1008-1027`]
- **Do not validate after assignment.** Returning an error with `req.Rank` or `req.Filter` already changed violates the locked mutation contract and differs from `groupByOption`. [VERIFIED: `pkg/api/v2/search.go:697-705`]
- **Do not use `where != nil`.** It misses typed-nil implementations; use `isNilInterface` and do not rewrite the caller's field. [VERIFIED: `pkg/api/v2/search.go:650-665`; `pkg/api/v2/search_test.go:1103-1143`]
- **Do not call only `RrfRank.Validate`.** It checks RRF's shallow fields and nil children but does not recursively validate a non-nil child KNN/composite; the recursive marshal path supplies that coverage. [VERIFIED: `pkg/api/v2/rank.go:1240-1262,1310-1363`]
- **Keep exact-type dispatch.** A type that embeds a built-in rank but provides its own marshaler must remain caller-defined; the existing `marshalRank` comment explicitly protects this case. [VERIFIED: `pkg/api/v2/rank.go:116-141`]
- **Expect eager validation to marshal a valid custom child inside a built-in expression.** Tests should use a deterministic, side-effect-free custom marshaler; this does not change the interface or acceptance of a valid implementation, but it can move its first marshal call earlier. [VERIFIED: recursive fallback at `pkg/api/v2/rank.go:118-141`; mixed custom-child test at `pkg/api/v2/rank_test.go:1028-1049`]
- **Do not string-match HTTP error text.** The current chain has no reliable stage sentinel, and matching `error marshalling request JSON` would couple collection code to presentation text. [VERIFIED: `pkg/api/v2/client.go:1108-1145`; `pkg/api/v2/collection_http.go:446-449`]

## Security and Validation Domain

ASVS V5 input validation applies: malformed built-in rank trees and `Where` clauses should be rejected at the public option boundary before serialization or I/O. Existing maximum-depth, maximum-term, nil-child, invalid-weight, and division-by-zero guards remain the standard controls and are reused by the recommended rank boundary. Authentication, session management, access control, and cryptography are not changed by this task. [VERIFIED: `pkg/api/v2/rank.go:108-141,274-286,425-437,502-520,1212-1262`; task scope]

## Assumptions Log

None. All behavioral findings were verified against the checked-out `main`, the public issue, existing tests, or official Chroma Search documentation.

## Sources

- [Public issue 516](https://github.com/amikos-tech/chroma-go/issues/516) — claims, reproductions, and proposed direction. [CITED: https://github.com/amikos-tech/chroma-go/issues/516]
- [Chroma ranking documentation](https://docs.trychroma.com/cloud/search-api/ranking) — KNN required query and defaults. [CITED: https://docs.trychroma.com/cloud/search-api/ranking]
- Current code and tests under `pkg/api/v2` at `da70881`. [VERIFIED: local codebase]

## Confidence Assessment

| Area | Level | Reason |
|---|---|---|
| Issue reproduction | HIGH | Every control-flow claim maps directly to current code; relevant baseline tests pass. [VERIFIED: local codebase and test run] |
| Rank design | HIGH | Reuses the established exact-type recursive marshal boundary and leaves `Rank` unchanged. [VERIFIED: `pkg/api/v2/rank.go:56-69,108-141`] |
| Filter design | HIGH | Mirrors existing `WithSearchWhere` nil and validation behavior with assignment after success. [VERIFIED: `pkg/api/v2/options.go:886-903`] |
| HTTP scope decision | HIGH | The reported paths are eliminated before `ExecuteRequest`, and current errors have no clean stage discriminator at the collection boundary. [VERIFIED: `pkg/api/v2/collection_http.go:426-449`; `pkg/api/v2/client.go:1108-1145`] |
