---
phase: quick-260730-fzu
plan: 01
subsystem: v2-search
tags: [panic-prevention, nil-handling, godoc, sentinels, wire-payload]
requires:
  - pkg/api/v2 Search option pipeline (SearchRequestOption)
  - github.com/pkg/errors (Unwrap support for errors.Is through errors.Wrap)
provides:
  - isNilInterface shared typed-nil helper
  - ErrNilFilter / ErrNilRank / ErrNilGroupBy exported sentinels
  - typed-nil-safe searchWhereOption and WhereClauseWhereClauses.Validate
  - omission-equivalent WithFilter(nil) wire payload
affects:
  - pkg/api/v2/search.go
  - pkg/api/v2/options.go
  - pkg/api/v2/where.go
tech-stack:
  added: []
  patterns:
    - reflect-based typed-nil detection at interface boundaries
    - exported sentinel errors returned unwrapped for errors.Is identity
    - lazy allocation of optional request sub-structs to preserve payload fidelity
key-files:
  created: []
  modified:
    - pkg/api/v2/search.go
    - pkg/api/v2/options.go
    - pkg/api/v2/where.go
    - pkg/api/v2/search_test.go
    - pkg/api/v2/options_test.go
    - pkg/api/v2/groupby_test.go
    - CHANGELOG.md
decisions:
  - isNilRank kept as a thin wrapper over isNilInterface so Rank reject-on-nil semantics are provably unchanged
  - typed nil normalized to a true nil at the option boundary rather than at marshal time
  - sentinels returned bare (not wrapped) at the return site to avoid per-call stack traces
metrics:
  duration: ~25m
  completed: 2026-07-30
---

# Quick Task 260730-fzu: Typed-nil WhereClause Panic and Doc Accuracy Summary

Fixed a typed-nil `WhereClause` panic in the V2 Search option pipeline, made `WithFilter(nil)` byte-identical to omitting the option, exported three nil-validation sentinels, and removed unverifiable cross-SDK claims from godoc.

## What Was Built

### Task 1 — Typed-nil safety and lazy filter allocation (`9450706` RED, `331fe30` GREEN)

`WhereClause` is an interface, so a typed nil (`var w *WhereClauseString`) is a non-nil interface value that slipped through every `!= nil` guard and crashed the host application on the subsequent `Validate()` call. Three sites were involved:

- **`pkg/api/v2/search.go`** — generalized the existing `isNilRank` into a shared `isNilInterface(v any) bool`. `isNilRank` is now a thin wrapper, so `WithRank`'s reject-on-nil semantics are unchanged by construction. Scope kept to `reflect.Pointer` as planned.
- **`pkg/api/v2/options.go`** — `searchWhereOption.ApplyToSearchRequest` now branches on `isNilInterface(o.where)`. On the nil path it returns no error (nil still means "no filter"), sets `req.Filter.Where = nil` when a filter already exists so the typed nil is normalized away, and returns without allocating a `&SearchFilter{}` when it does not.
- **`pkg/api/v2/where.go`** — `WhereClauseWhereClauses.Validate` guards nested clauses with `isNilInterface`, so `And(EqString(...), w)` yields `nil clause in $and expression` instead of dereferencing nil.

The normalization step matters: without it a typed nil survives into `SearchFilter.MarshalJSON`'s `if f.Where != nil` check and panics at marshal time rather than at option-apply time.

### Task 2 — Exported nil sentinels (`f6dddf3` RED, `c332d45` GREEN)

Added `ErrNilFilter`, `ErrNilRank` and `ErrNilGroupBy` to the existing `// Option validation errors.` block in `options.go` and returned them directly from `searchFilterOption`, `rankOption` and `groupByOption`. Message text is unchanged, so existing string-based assertions elsewhere still hold. Because the package uses `github.com/pkg/errors`, whose `Wrap` result implements `Unwrap()`, the sentinels survive the `errors.Wrap(err, "error applying search option")` performed at `collection_http.go:430` — asserted explicitly in `TestNilOptionSentinels`.

### Task 3 — Doc accuracy and CHANGELOG (`eccd363`)

- Deleted the "intentional, permanent divergence from the Python and TypeScript SDKs" clause from `WithSearchFilter`, `WithRank` and `WithGroupBy`, and "matching Python/TypeScript SDK nil semantics" from `WithSearchWhere`. Actionable guidance ("simply not call this option") was kept.
- Reworded "Passing nil returns a validation error" to "Passing nil causes the enclosing search request to fail with `[ErrNilX]`", since these constructors have no error return.
- Replaced the inaccurate "treated as 'no filter'" claim on `WithFilter`/`WithSearchWhere` with the verified contract: a nil clause clears any clause set earlier on the same request (ordinary last-write-wins, not nil-specific) while `WithIDs` IDs are preserved. Dropped "tested" from "intentional, tested asymmetry".
- CHANGELOG `v0.4.2 - Unreleased` gained an `### Added` entry for the sentinels and two `### Fixed` entries (panic, empty-filter payload). The pre-existing SDK-comparison rationale stays in the CHANGELOG, which is now its only home.

## Behavioral Contract Preserved

Per the plan's critical requirements, none of the following were changed:

- `WithFilter`/`WithSearchWhere` still **accept** nil as "no filter" — they do not reject it. The asymmetry with the struct-based options is deliberate and intact.
- `isNilRank`'s reject-on-nil semantics for `Rank` are unchanged.
- `WithIDs`/`WithFilter` compose in either order (both directions covered by `TestWithFilterNilComposition`).

## Tests Added

| Test | File | Covers |
|------|------|--------|
| `TestTypedNilWhereClause` | options_test.go | typed nil inert via WithSearchWhere/WithFilter; nested And/Or errors; normalization + marshal safety |
| `TestWithFilterNilComposition` | options_test.go | WithIDs↔WithFilter(nil) in both orders; nil clears a previously set clause |
| `TestEarlyValidationInvalidSearchWhereFilter/nil search where filter is allowed` | options_test.go | strengthened from bare `require.NoError` to assert `req.Filter` stays nil |
| `TestNilOptionSentinels` | search_test.go | `errors.Is` for all three sentinels, plus identity through `errors.Wrap` |
| `TestWithFilterNilPayloadEqualsOmission` | search_test.go | marshalled JSON equality against the omit-the-option baseline, untyped and typed nil |

RED was confirmed empirically before each implementation commit: the first run produced `SIGSEGV` at `where.go:65`, `where.go:471` and `options.go:870`; the second failed to compile on the undefined sentinels.

## Verification Gate

`go build ./...` — no output (success).

```
$ make lint
golangci-lint run
0 issues.
```

```
$ go test -tags=basicv2 ./pkg/api/v2/...
ok  	github.com/amikos-tech/chroma-go/pkg/api/v2	22.878s
```

Godoc scrub check (`grep -n "Python and TypeScript SDKs\|Python/TypeScript SDK nil semantics\|intentional, tested asymmetry" pkg/api/v2/search.go pkg/api/v2/options.go`) returns exit 1 / no matches, as required by the plan's automated verification.

## Deviations from Plan

None — plan executed as written.

## Out-of-Scope Items Confirmed Untouched

`git diff f9cac29..HEAD -- pkg/api/v2/collection_http.go` is empty, confirming `cloneRank`'s nested typed-nil gap (line 471-473, still `rank == nil`) was left for its follow-up issue. Also untouched: missing `Validate()` calls in `rankOption`/`searchFilterOption`, non-pointer reflect kinds in the nil helper, and `WithSearchFilter`'s wholesale `req.Filter` overwrite.

## Known Stubs

None.

## Threat Flags

None. The changes close T-fzu-01 and T-fzu-02 (denial of service via panic) and T-fzu-03 (misleading cross-SDK doc claims). No new network endpoints, auth paths, file access patterns, or schema changes were introduced.

## Commits

| Commit | Type | Description |
|--------|------|-------------|
| `9450706` | test | failing tests for typed-nil WhereClause panic (RED) |
| `331fe30` | fix | isNilInterface, typed-nil normalization, lazy filter alloc (GREEN) |
| `f6dddf3` | test | errors.Is sentinel assertions (RED) |
| `c332d45` | feat | ErrNilFilter / ErrNilRank / ErrNilGroupBy (GREEN) |
| `eccd363` | docs | godoc corrections and CHANGELOG entries |

## TDD Gate Compliance

Both TDD tasks show the full RED→GREEN sequence in git log: `test(...)` before `fix(...)` for Task 1, and `test(...)` before `feat(...)` for Task 2. No REFACTOR commits were needed — the implementations were already minimal.

## Self-Check: PASSED

Files verified present: `pkg/api/v2/search.go`, `pkg/api/v2/options.go`, `pkg/api/v2/where.go`, `pkg/api/v2/search_test.go`, `pkg/api/v2/options_test.go`, `pkg/api/v2/groupby_test.go`, `CHANGELOG.md`.

Commits verified in git log: `9450706`, `331fe30`, `f6dddf3`, `c332d45`, `eccd363`.

Plan key_links verified in source: `isNilInterface(o.where)` at options.go:873, `isNilInterface(clause)` at where.go:468, `return ErrNilFilter` at search.go:399, `return ErrNilRank` at search.go:631, `return ErrNilGroupBy` at search.go:679.
