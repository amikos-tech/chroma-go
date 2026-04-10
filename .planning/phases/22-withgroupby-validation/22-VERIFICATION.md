---
phase: 22-withgroupby-validation
verified: 2026-04-10T04:51:04Z
status: passed
score: 6/6 must-haves verified
overrides_applied: 0
---

# Phase 22: WithGroupBy Validation Verification Report

**Phase Goal:** WithGroupBy rejects nil input with a clear error  
**Verified:** 2026-04-10T04:51:04Z  
**Status:** passed  
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Passing nil to `WithGroupBy` returns a validation error before the request is sent | ✓ VERIFIED | `pkg/api/v2/search.go:635-643` now rejects `nil` immediately; uncached `go test -count=1 -tags=basicv2 -run 'TestWithGroupBy\|TestSearchRequestWithGroupBy' ./pkg/api/v2/...` passed. |
| 2 | Non-nil `WithGroupBy` calls continue to work as before | ✓ VERIFIED | Valid non-nil values still flow through `o.groupBy.Validate()` and `req.GroupBy = o.groupBy` in `pkg/api/v2/search.go:639-642`; valid direct and full-search coverage pass in `pkg/api/v2/groupby_test.go:183-190` and `pkg/api/v2/groupby_test.go:240-275`. |
| 3 | Explicit `WithGroupBy(nil)` returns the exact error `groupBy cannot be nil` before any request append | ✓ VERIFIED | Exact string is emitted at `pkg/api/v2/search.go:636-638` and asserted directly at `pkg/api/v2/groupby_test.go:193-197` and `pkg/api/v2/groupby_test.go:290-300`. |
| 4 | Omitting `WithGroupBy(...)` remains the way to request no grouping | ✓ VERIFIED | `NewSearchRequest` only applies provided options in `pkg/api/v2/search.go:658-666`; if `WithGroupBy` is omitted, no `groupByOption` runs and `SearchRequest.GroupBy` remains nil. |
| 5 | `NewSearchRequest(..., WithGroupBy(nil))` fails before appending a partial request to `SearchQuery.Searches` | ✓ VERIFIED | `NewSearchRequest` returns on option error before `append` in `pkg/api/v2/search.go:661-666`; `pkg/api/v2/groupby_test.go:290-300` asserts the exact nil error and `require.Len(t, sq.Searches, 0)`. |
| 6 | `pkg/api/v2/groupby_test.go` pins the nil-error contract while keeping valid non-nil coverage in place | ✓ VERIFIED | `pkg/api/v2/groupby_test.go:182-215` covers direct valid/nil/invalid option application, and `pkg/api/v2/groupby_test.go:240-300` covers valid request construction plus nil fail-before-append. |

**Score:** 6/6 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `pkg/api/v2/search.go` | Fail-fast nil validation for `WithGroupBy` | ✓ VERIFIED | File exists and is substantive (940 lines). `groupByOption.ApplyToSearchRequest` rejects `nil` with `groupBy cannot be nil` at `pkg/api/v2/search.go:635-638`, preserves non-nil validation at `pkg/api/v2/search.go:639-640`, and appends only after successful option application at `pkg/api/v2/search.go:658-666`. |
| `pkg/api/v2/groupby_test.go` | Exact-error regression coverage for nil and non-nil `WithGroupBy` behavior | ✓ VERIFIED | File exists and is substantive (302 lines). Direct nil-error coverage is at `pkg/api/v2/groupby_test.go:193-197`; request-construction nil/no-append coverage is at `pkg/api/v2/groupby_test.go:290-300`; valid non-nil coverage remains at `pkg/api/v2/groupby_test.go:183-190` and `pkg/api/v2/groupby_test.go:240-275`. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `pkg/api/v2/search.go` `groupByOption.ApplyToSearchRequest` | `pkg/api/v2/groupby_test.go` `TestWithGroupBy` | Exact nil-error contract for explicit `WithGroupBy(nil)` | WIRED | `pkg/api/v2/groupby_test.go:195-197` calls `WithGroupBy(nil).ApplyToSearchRequest(req)` and asserts `require.EqualError(t, err, "groupBy cannot be nil")`; adjacent subtests cover valid and invalid non-nil inputs. |
| `pkg/api/v2/search.go` `NewSearchRequest` option application | `pkg/api/v2/groupby_test.go` `TestSearchRequestWithGroupBy` | Nil groupby request-construction regression | WIRED | `pkg/api/v2/groupby_test.go:293-300` constructs `NewSearchRequest(..., WithGroupBy(nil))` and asserts the exact error and zero appended searches; the valid path at `pkg/api/v2/groupby_test.go:243-275` proves successful append when `GroupBy` is non-nil and valid. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| `pkg/api/v2/search.go:635-643` | `req.GroupBy` | Caller-provided `o.groupBy`, guarded by `(*GroupBy).Validate()` in `pkg/api/v2/groupby.go:30-40` | Yes — valid non-nil `GroupBy` values are assigned unchanged; explicit nil is rejected before mutation | ✓ FLOWING |
| `pkg/api/v2/search.go:658-666` | `SearchQuery.Searches` | Local `search` built by applying each `SearchRequestOption` in order | Yes — successful options append one request, and failing `WithGroupBy(nil)` short-circuits before append | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Nil `WithGroupBy` is rejected in both direct and request-construction paths | `go test -count=1 -tags=basicv2 -run 'TestWithGroupBy\|TestSearchRequestWithGroupBy' ./pkg/api/v2/...` | `ok github.com/amikos-tech/chroma-go/pkg/api/v2 0.448s` | ✓ PASS |
| `GroupBy` validation plus `WithGroupBy` regressions still pass | `go test -count=1 -tags=basicv2 -run 'TestGroupBy\|TestWithGroupBy\|TestSearchRequestWithGroupBy' ./pkg/api/v2/...` | `ok github.com/amikos-tech/chroma-go/pkg/api/v2 0.628s` | ✓ PASS |

Additional regression evidence: `go test -count=1 -tags=basicv2 ./pkg/api/v2/...` → `ok github.com/amikos-tech/chroma-go/pkg/api/v2 23.011s`.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| `GRP-01` | `22-01-PLAN.md` | `WithGroupBy(nil)` returns a validation error instead of silently skipping grouping | ✓ SATISFIED | The nil branch in `pkg/api/v2/search.go:636-638` now returns `groupBy cannot be nil`, and `pkg/api/v2/groupby_test.go:193-197` plus `pkg/api/v2/groupby_test.go:290-300` assert both the direct and composed error/no-append behavior. |

Orphaned requirements for Phase 22: none. `REQUIREMENTS.md` maps only `GRP-01` to Phase 22, and `22-01-PLAN.md` declares the same requirement ID.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `pkg/api/v2/groupby_test.go` | 278 | Invalid request-path regression uses generic `require.Error` instead of pinning the exact passthrough error | ℹ️ Info | This does not block the phase because `pkg/api/v2/search.go:639-640` still returns `(*GroupBy).Validate()` errors unchanged, but the request-path test is broader than the direct nil-error contract. |

### Gaps Summary

No blocking gaps found. The roadmap success criteria and the plan’s must-haves are implemented in `pkg/api/v2/search.go`, exercised directly in `pkg/api/v2/groupby_test.go`, and backed by fresh uncached `basicv2` test runs. Requirement coverage is complete for `GRP-01`, and there are no orphaned requirement IDs for this phase.

---

_Verified: 2026-04-10T04:51:04Z_  
_Verifier: Claude (gsd-verifier)_
