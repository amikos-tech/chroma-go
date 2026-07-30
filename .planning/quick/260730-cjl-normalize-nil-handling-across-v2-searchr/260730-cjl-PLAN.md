---
phase: 260730-cjl
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - pkg/api/v2/search.go
  - pkg/api/v2/search_test.go
  - CHANGELOG.md
autonomous: true
requirements: [ISSUE-503]

must_haves:
  truths:
    - "WithSearchFilter(nil).ApplyToSearchRequest(req) returns an error and leaves req.Filter nil"
    - "WithRank(nil).ApplyToSearchRequest(req) returns an error and leaves req.Rank nil"
    - "NewSearchRequest(WithSearchFilter(nil)) returns an error and does not append to SearchQuery.Searches"
    - "NewSearchRequest(WithRank(nil)) returns an error and does not append to SearchQuery.Searches"
    - "Existing valid WithSearchFilter(...) and WithRank(...) calls behave exactly as before (no regression)"
    - "WithSearchFilter, WithRank, and WithGroupBy doc comments state the nil-rejection contract and the intentional Python/TS divergence"
    - "CHANGELOG.md documents the extended nil-validation behavior for all three options"
  artifacts:
    - path: "pkg/api/v2/search.go"
      provides: "Nil-rejecting searchFilterOption.ApplyToSearchRequest and rankOption.ApplyToSearchRequest, plus updated doc comments"
      contains: "filter cannot be nil"
    - path: "pkg/api/v2/search_test.go"
      provides: "Nil-validation test coverage for WithSearchFilter and WithRank, direct and composed via NewSearchRequest"
      contains: "rank cannot be nil"
    - path: "CHANGELOG.md"
      provides: "Documented behavior change entry covering WithSearchFilter, WithRank, and WithGroupBy nil handling plus Python/TS divergence"
      contains: "WithSearchFilter(nil)"
  key_links:
    - from: "pkg/api/v2/search.go (searchFilterOption.ApplyToSearchRequest)"
      to: "pkg/api/v2/search_test.go (TestSearchFilter nil subtest)"
      via: "exact error string match"
      pattern: "filter cannot be nil"
    - from: "pkg/api/v2/search.go (rankOption.ApplyToSearchRequest)"
      to: "pkg/api/v2/search_test.go (TestWithRank nil subtest)"
      via: "exact error string match"
      pattern: "rank cannot be nil"
---

<objective>
Close GH #503: `WithGroupBy(nil)` (Phase 22) fails fast with a validation error, but sibling
`SearchRequestOption` helpers `WithSearchFilter` and `WithRank` still silently treat an explicit
`nil` as omission. Normalize both to the same fail-fast contract as `WithGroupBy`, and document
the resulting (intentional) divergence from the Python/TypeScript SDKs, which treat
`None`/`null`/`undefined` as a no-op for filter/rank/group_by.

Purpose: present a uniform, loud-failure contract for explicit nil across all three
`SearchRequestOption` helpers in `pkg/api/v2/search.go`, instead of two silently swallowing
caller bugs while the third (already fixed) does not.

Output:
1. `searchFilterOption.ApplyToSearchRequest` and `rankOption.ApplyToSearchRequest` reject nil
   with `errors.New("filter cannot be nil")` / `errors.New("rank cannot be nil")`, matching
   `groupByOption.ApplyToSearchRequest`'s existing pattern exactly.
2. Doc comments on `WithSearchFilter`, `WithRank`, and `WithGroupBy` state the nil-rejection
   contract and the intentional Python/TS divergence.
3. Test coverage mirroring `groupby_test.go`'s `TestWithGroupBy`/`TestSearchRequestWithGroupBy`
   pattern, pinning both the direct option path and the composed `NewSearchRequest` path.
4. CHANGELOG.md entry extended to cover all three options.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/quick/260730-cjl-normalize-nil-handling-across-v2-searchr/260730-cjl-CONTEXT.md
@.planning/quick/260730-cjl-normalize-nil-handling-across-v2-searchr/260730-cjl-RESEARCH.md
@CLAUDE.md

**Working directory:** `/Users/tazarov/GolandProjects/chroma-go`

**Scope boundary (locked, per CONTEXT.md):** `WithSelect()` with zero keys stays a valid
no-op — do NOT add a nil/empty check there. Only `WithSearchFilter` and `WithRank` change
behavior; `WithGroupBy` behavior is already correct (Phase 22) and only its doc comment changes
in this plan.

**No existing call sites break:** repo-wide grep for `WithSearchFilter(` and `WithRank(` (incl.
`examples/` and `test/`) found zero nil-literal or conditionally-nil call sites — see
RESEARCH.md "No existing nil-call breakage".
</context>

<interfaces>
Reference implementation already in `pkg/api/v2/search.go:635-644` — copy this exact shape for
the other two options (same file, same `github.com/pkg/errors` import, already in scope):

```go
func (o *groupByOption) ApplyToSearchRequest(req *SearchRequest) error {
	if o.groupBy == nil {
		return errors.New("groupBy cannot be nil")
	}
	if err := o.groupBy.Validate(); err != nil {
		return err
	}
	req.GroupBy = o.groupBy
	return nil
}
```

Target functions to modify (unchanged signatures, only bodies gain a leading nil check):
- `searchFilterOption.ApplyToSearchRequest(req *SearchRequest) error` — search.go:392-395
- `rankOption.ApplyToSearchRequest(req *SearchRequest) error` — search.go:610-613

Test pattern to mirror — `pkg/api/v2/groupby_test.go:182-216` (`TestWithGroupBy`, per-case
`t.Run` table: happy path + `"nil <x> returns exact validation error"` using
`require.EqualError` + `require.Nil`) and `groupby_test.go:218-251`
(`TestSearchRequestWithGroupBy`, composed `NewSearchRequest(...)` path).
</interfaces>

<tasks>

<task type="auto">
  <name>Task 1: Reject nil in WithSearchFilter/WithRank and document the divergence</name>
  <files>pkg/api/v2/search.go</files>
  <action>
    In `searchFilterOption.ApplyToSearchRequest` (search.go:392-395), add a leading nil check
    before the existing assignment: if `o.filter == nil`, return `errors.New("filter cannot be
    nil")`. Keep the existing `req.Filter = o.filter; return nil` for the non-nil path
    unchanged.

    In `rankOption.ApplyToSearchRequest` (search.go:610-613), add the same pattern: if
    `o.rank == nil`, return `errors.New("rank cannot be nil")`, before the existing
    `req.Rank = o.rank; return nil`.

    Both checks use the file's existing `github.com/pkg/errors` import (already used by
    `groupByOption.ApplyToSearchRequest` and `WithReadLevel`) — no new import needed.

    Add a short (1-2 line) doc-comment addendum to `WithSearchFilter` (search.go:385-387),
    `WithRank` (search.go:591-605), and `WithGroupBy` (search.go:620-630) stating: passing nil
    returns a validation error instead of being treated as omission, and this is an intentional,
    permanent divergence from the Python and TypeScript SDKs (which treat
    `None`/`null`/`undefined` as a no-op for filter/rank/group_by) — callers who want to omit
    the option should simply not call it. Keep each addendum inline in the existing doc block,
    no shared doc helper.

    Do NOT touch `selectOption`/`WithSelect` — zero-key selection stays a valid no-op
    (out of scope per CONTEXT.md).
  </action>
  <verify>
    <automated>cd /Users/tazarov/GolandProjects/chroma-go && go build ./pkg/api/v2/... && grep -c '"filter cannot be nil"' pkg/api/v2/search.go && grep -c '"rank cannot be nil"' pkg/api/v2/search.go</automated>
  </verify>
  <done>
    `searchFilterOption.ApplyToSearchRequest` and `rankOption.ApplyToSearchRequest` return the
    exact error strings for nil input without mutating `req.Filter`/`req.Rank`; `WithSearchFilter`,
    `WithRank`, and `WithGroupBy` doc comments document the nil-rejection contract and the
    Python/TS divergence; package builds clean.
  </done>
</task>

<task type="auto">
  <name>Task 2: Add nil-validation tests and extend the CHANGELOG entry</name>
  <files>pkg/api/v2/search_test.go, CHANGELOG.md</files>
  <action>
    In `search_test.go`, extend `TestSearchFilter` (search_test.go:183-207) with two new
    `t.Run` subtests mirroring `TestWithGroupBy` (groupby_test.go:182-198) and
    `TestSearchRequestWithGroupBy` (groupby_test.go:218-251):
    - `"nil filter returns exact validation error"`: `WithSearchFilter(nil).ApplyToSearchRequest(req)`
      on a fresh `&SearchRequest{}`; assert `require.EqualError(t, err, "filter cannot be nil")`
      and `require.Nil(t, req.Filter)`.
    - `"composed NewSearchRequest with nil filter fails before append"`: build a fresh
      `&SearchQuery{}`, call `NewSearchRequest(WithSearchFilter(nil))(sq)`, assert the returned
      error is non-nil and `require.Empty(t, sq.Searches)`.

    Add a new `TestWithRank` test function near `TestWithKnnRank`/`TestWithRrfRank`
    (search_test.go:300-372) — there is currently no dedicated test for the base `WithRank`
    option (only its `WithKnnRank`/`WithRrfRank` convenience wrappers are covered). Include:
    - a happy-path subtest: apply `WithRank(validRank)` (use `mustKnnRank(t, KnnQueryText(...))`
      as the concrete `Rank` value, matching the helper already used elsewhere in this file) to
      a fresh `&SearchRequest{}` and assert `req.Rank` is set to that value.
    - `"nil rank returns exact validation error"`: `WithRank(nil).ApplyToSearchRequest(req)`;
      assert `require.EqualError(t, err, "rank cannot be nil")` and `require.Nil(t, req.Rank)`.
    - `"composed NewSearchRequest with nil rank fails before append"`: mirror the filter case
      above using `NewSearchRequest(WithRank(nil))(sq)` and `require.Empty(t, sq.Searches)`.

    In `CHANGELOG.md`, replace the single-line `WithGroupBy(nil)` entry (currently line 11,
    under `## [v0.4.2] - Unreleased` → `### Changed`) with the extended entry covering all three
    options plus the Python/TS divergence note (wording per RESEARCH.md "CHANGELOG format"
    section) — one `### Changed` bullet, not three separate bullets.
  </action>
  <verify>
    <automated>cd /Users/tazarov/GolandProjects/chroma-go && go test -tags=basicv2 ./pkg/api/v2/... -run 'TestSearchFilter|TestWithRank|TestWithGroupBy|TestSearchRequestWithGroupBy' -v</automated>
  </verify>
  <done>
    New subtests pass for both the direct `ApplyToSearchRequest` path and the composed
    `NewSearchRequest` path, for both filter and rank; full `go test -tags=basicv2 ./pkg/api/v2/...`
    passes; `CHANGELOG.md` documents the extended nil-validation behavior and the Python/TS
    divergence in a single `### Changed` bullet under `## [v0.4.2] - Unreleased`.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| caller application code -> chroma-go V2 Search API options (`WithSearchFilter`/`WithRank`/`WithGroupBy`) | Caller-supplied option values (possibly nil due to caller bugs, e.g. a conditionally-nil variable threaded through) cross into SDK request construction |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-cjl-01 | Tampering | `searchFilterOption.ApplyToSearchRequest` / `rankOption.ApplyToSearchRequest` | mitigate | Explicit nil-check returns a loud error instead of silently constructing an unintentionally unfiltered/unranked search request — the exact failure mode this task closes |
| T-cjl-02 | Denial of Service (nil-pointer panic) | Downstream consumers of `req.Filter`/`req.Rank` (e.g. `MarshalJSON`) | mitigate | Per CLAUDE.md "never panic in production code" — rejecting nil at the option boundary prevents any future nil-dereference if a caller further down the chain assumes non-nil |
| T-cjl-03 | Information Disclosure | Error message content | accept | Error strings (`"filter cannot be nil"`, `"rank cannot be nil"`) contain no caller data, no PII, no internal state |
| T-cjl-SC | Tampering | npm/pip/cargo installs | accept | No new package dependencies introduced; only the already-imported `github.com/pkg/errors` is used — no Package Legitimacy Gate applicable |
</threat_model>

<verification>
1. `cd /Users/tazarov/GolandProjects/chroma-go && go build ./...` — full repo builds clean.
2. `go test -tags=basicv2 ./pkg/api/v2/...` — full V2 test suite passes, including the new
   nil-validation subtests.
3. `make lint` — clean (per CLAUDE.md: always lint before committing).
4. Manual review of `CHANGELOG.md` — the `## [v0.4.2] - Unreleased` → `### Changed` section has
   exactly one bullet covering `WithSearchFilter`, `WithRank`, and `WithGroupBy` nil behavior
   plus the Python/TS divergence note (no leftover duplicate `WithGroupBy(nil)`-only line).
</verification>

<source_coverage_audit>
| Source | Item | Covered by |
|--------|------|------------|
| GOAL | Normalize nil-handling across V2 SearchRequestOption helpers (GH #503) | Task 1 |
| REQ | ISSUE-503 | Task 1 (implementation), Task 2 (tests + CHANGELOG) |
| RESEARCH | No existing nil call sites — safe to reject nil with zero breakage | Task 1 (informs safety, cited in context) |
| RESEARCH | Error style: `github.com/pkg/errors`, `errors.New("<field> cannot be nil")` matching `WithGroupBy` | Task 1 |
| RESEARCH | Test structure to mirror (`groupby_test.go` `t.Run` table + composed-path pin) | Task 2 |
| RESEARCH | CHANGELOG format (single extended `### Changed` bullet) | Task 2 |
| RESEARCH | Doc comment placement (near `WithSearchFilter`/`WithRank`/`WithGroupBy`, 1-2 lines) | Task 1 |
| CONTEXT D | `WithSearchFilter(nil)` rejects with error matching `WithGroupBy(nil)` precedent | Task 1 |
| CONTEXT D | `WithRank(nil)` rejects with error matching `WithGroupBy(nil)` precedent | Task 1 |
| CONTEXT D | `WithSelect()` with zero keys stays a valid no-op — explicitly out of scope | Excluded (locked scope boundary, noted in `<context>`) |
| CONTEXT D | Document intentional, permanent divergence from Python/TS nil semantics | Task 1 (doc comments), Task 2 (CHANGELOG) |
| CONTEXT (discretion) | Exact error message wording — follow `WithGroupBy` style/tone | Task 1 |
| CONTEXT (discretion) | Inline nil-check pattern, no shared helper (duplication is minor) | Task 1 |
| CONTEXT (discretion) | Test structure/naming for new validation error cases | Task 2 |
| CONTEXT (discretion) | Doc note placement (doc comment + CHANGELOG, both) | Task 1 + Task 2 |
</source_coverage_audit>

<success_criteria>
- `WithSearchFilter(nil)` and `WithRank(nil)` both return validation errors identical in style
  to `WithGroupBy(nil)`, leaving the corresponding `SearchRequest` field nil.
- The composed `NewSearchRequest(...)` path fails before appending to `SearchQuery.Searches`
  for both new nil cases, matching the existing `WithGroupBy(nil)` composed-path behavior.
- No existing test, example, or call site breaks (confirmed zero nil-literal call sites in
  RESEARCH.md).
- `WithSearchFilter`, `WithRank`, and `WithGroupBy` doc comments and `CHANGELOG.md` both state
  the nil-rejection contract and the intentional, permanent Python/TS divergence.
- `go build ./...`, `go test -tags=basicv2 ./pkg/api/v2/...`, and `make lint` all pass clean.
</success_criteria>

<output>
Create `.planning/quick/260730-cjl-normalize-nil-handling-across-v2-searchr/260730-cjl-SUMMARY.md` when done
</output>
