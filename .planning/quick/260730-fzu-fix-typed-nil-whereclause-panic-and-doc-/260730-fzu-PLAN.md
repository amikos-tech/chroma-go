---
phase: quick-260730-fzu
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - pkg/api/v2/search.go
  - pkg/api/v2/options.go
  - pkg/api/v2/where.go
  - pkg/api/v2/search_test.go
  - pkg/api/v2/options_test.go
  - pkg/api/v2/groupby_test.go
  - CHANGELOG.md
autonomous: true
requirements: [FZU-01, FZU-02, FZU-03, FZU-04, FZU-05]

must_haves:
  truths:
    - "A typed-nil WhereClause (e.g. var w *WhereClauseString) passed to WithFilter/WithSearchWhere does not panic and is treated as 'no filter'"
    - "A typed-nil WhereClause nested inside And(...)/Or(...) returns the 'nil clause in $and expression' error instead of panicking"
    - "WithFilter(nil) produces byte-identical JSON to omitting WithFilter entirely (no \"filter\":{} key)"
    - "WithFilter(nil) still clears a previously-set where clause while preserving IDs added by WithIDs"
    - "WithSearchFilter(nil), WithRank(nil), WithGroupBy(nil) errors are discriminable via errors.Is, including after collection Search wrapping"
    - "Godoc no longer asserts Python/TypeScript SDK parity, no longer says nil is 'treated as no filter', and no longer claims an untested asymmetry"
  artifacts:
    - path: "pkg/api/v2/search.go"
      provides: "isNilInterface shared typed-nil helper; ErrNilFilter/ErrNilRank/ErrNilGroupBy returns; rewritten godoc"
      contains: "isNilInterface"
    - path: "pkg/api/v2/options.go"
      provides: "typed-nil-safe searchWhereOption; ErrNilFilter/ErrNilRank/ErrNilGroupBy sentinel declarations"
      contains: "ErrNilFilter"
    - path: "pkg/api/v2/where.go"
      provides: "typed-nil-safe WhereClauseWhereClauses.Validate nested clause guard"
      contains: "isNilInterface"
    - path: "CHANGELOG.md"
      provides: "v0.4.2 Unreleased entries for the panic fix, payload fix, and sentinels; SDK-comparison rationale relocated here"
  key_links:
    - from: "pkg/api/v2/options.go searchWhereOption.ApplyToSearchRequest"
      to: "isNilInterface"
      via: "nil guard replacing plain `o.where != nil`"
      pattern: "isNilInterface\\(o\\.where\\)"
    - from: "pkg/api/v2/where.go WhereClauseWhereClauses.Validate"
      to: "isNilInterface"
      via: "nested clause nil guard replacing `clause == nil`"
      pattern: "isNilInterface\\(clause\\)"
    - from: "pkg/api/v2/search.go option Apply methods"
      to: "options.go sentinel block"
      via: "return ErrNilFilter / ErrNilRank / ErrNilGroupBy"
      pattern: "return Err(NilFilter|NilRank|NilGroupBy)"
---

<objective>
Close the five verified gaps left by the previous nil-handling PR (GH #503) on branch
`quick/260730-cjl-normalize-nil-handling-across-v2-searchr`: a typed-nil `WhereClause` panic,
two doc-accuracy defects, an empty `{"filter":{}}` wire payload, and three inline `errors.New`
values that cannot be discriminated with `errors.Is`.

Purpose: this library must never panic in production code (project CLAUDE.md), and its godoc
must describe behavior that was actually observed rather than asserted.
Output: typed-nil-safe option/validation paths, omission-equivalent `WithFilter(nil)` payload,
three exported sentinel errors, corrected godoc, CHANGELOG entry, and regression tests.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@CLAUDE.md

@pkg/api/v2/search.go
@pkg/api/v2/options.go
@pkg/api/v2/where.go
@CHANGELOG.md

<interfaces>
<!-- Contracts the executor needs. Extracted from the codebase — no exploration required. -->

pkg/api/v2/where.go — WhereClause is an INTERFACE, so a typed-nil pointer is a non-nil
interface value and passes a plain `!= nil` guard:

```go
type WhereClause interface {
	Operator() WhereFilterOperator
	Key() string
	Operand() interface{}
	String() string
	Validate() error
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(b []byte) error
}
```

pkg/api/v2/search.go:639 — existing helper to generalize (its reject-on-nil semantics for
Rank must stay UNCHANGED):

```go
func isNilRank(rank Rank) bool {
	if rank == nil {
		return true
	}
	v := reflect.ValueOf(rank)
	return v.Kind() == reflect.Pointer && v.IsNil()
}
```

pkg/api/v2/options.go:868 — the panicking guard:

```go
func (o *searchWhereOption) ApplyToSearchRequest(req *SearchRequest) error {
	if o.where != nil {            // typed nil slips through here
		if err := o.where.Validate(); err != nil {   // panics at where.go:65
			return err
		}
	}
	if req.Filter == nil {
		req.Filter = &SearchFilter{}   // unconditional alloc -> {"filter":{}}
	}
	req.Filter.Where = o.where
	return nil
}
```

pkg/api/v2/where.go:467 — the same blind spot for nested clauses:

```go
for _, clause := range w.operand {
	if clause == nil {   // typed nil slips through -> panic in clause.Validate()
		return errors.Errorf("nil clause in %s expression", w.operator)
	}
	if err := clause.Validate(); err != nil {
		return err
	}
}
```

pkg/api/v2/options.go:284 — WithIDs also lazily allocates req.Filter; the two options must
still compose in either order:

```go
func (o *idsOption) ApplyToSearchRequest(req *SearchRequest) error {
	if len(o.ids) == 0 { return errors.New("at least one id is required") }
	if req.Filter == nil { req.Filter = &SearchFilter{} }
	if err := checkDuplicateIDs(o.ids, req.Filter.IDs); err != nil { return err }
	req.Filter.IDs = append(req.Filter.IDs, o.ids...)
	return nil
}
```

pkg/api/v2/options.go:884 — the established sentinel block to extend:

```go
// Option validation errors.
var (
	ErrInvalidLimit    = errors.New("limit must be greater than 0")
	ErrLimitOverflow   = fmt.Errorf("limit cannot exceed %d", math.MaxInt32)
	ErrInvalidOffset   = errors.New("offset must be greater than or equal to 0")
	ErrInvalidNResults = errors.New("nResults must be greater than 0")
	ErrNoQueryTexts    = errors.New("at least one query text is required")
	ErrNoTexts         = errors.New("at least one text is required")
	ErrNoMetadatas     = errors.New("at least one metadata is required")
)
```

The package imports `github.com/pkg/errors` v0.9.1 — `errors.Wrap` results implement
`Unwrap()`, so `errors.Is` traverses the `collection_http.go:430` wrap
(`errors.Wrap(err, "error applying search option")`).

`reflect` is already imported in pkg/api/v2/search.go. `where.go` and `options.go` are in the
same package and can call the helper without new imports.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Make typed-nil WhereClause safe and stop allocating an empty filter</name>
  <files>pkg/api/v2/search.go, pkg/api/v2/options.go, pkg/api/v2/where.go, pkg/api/v2/options_test.go, pkg/api/v2/search_test.go</files>
  <behavior>
    - `WithSearchWhere(w)` where `var w *WhereClauseString` returns no error, leaves `req.Filter` nil, and does not panic
    - `WithFilter(w)` with the same typed nil behaves identically (it delegates to WithSearchWhere)
    - `And(EqString(K("a"), "b"), w)` with a typed-nil `w` returns error `nil clause in $and expression` from Validate instead of panicking; same for `Or(...)`
    - `json.Marshal` of `NewSearchRequest(WithFilter(nil), WithKnnRank(...))` output equals the output of `NewSearchRequest(WithKnnRank(...))` — no `"filter"` key at all
    - `WithIDs("a","b")` then `WithFilter(nil)` leaves `req.Filter.IDs` intact with `req.Filter.Where == nil`; the reverse order (`WithFilter(nil)` then `WithIDs`) yields the same state
    - `WithFilter(EqString(...))` then `WithFilter(nil)` clears the where clause (last-write-wins, existing behavior preserved)
    - Existing test `TestEarlyValidationInvalidSearchWhereFilter/nil search where filter is allowed` is strengthened to assert resulting request state, not just `require.NoError`
  </behavior>
  <action>
Generalize the existing `isNilRank` helper in pkg/api/v2/search.go into a shared
`isNilInterface(v any) bool` placed next to it (reflect is already imported). Keep `isNilRank`
as-is behaviorally — either leave it delegating to `isNilInterface` or keep it as a thin
wrapper; `WithRank`'s reject-on-nil semantics MUST NOT change. Keep the helper limited to the
`reflect.Pointer` kind already covered; broadening to other kinds is explicitly out of scope.

In pkg/api/v2/options.go `searchWhereOption.ApplyToSearchRequest`, replace the
`if o.where != nil` guard with `isNilInterface(o.where)`. On the nil path (untyped OR typed):
do NOT return an error — a nil where clause means "no filter", preserving the deliberate
asymmetry with the struct-based options. If `req.Filter` is already non-nil (e.g. WithIDs ran
first), set `req.Filter.Where = nil` so a typed nil is normalized away and any previously-set
clause is cleared; if `req.Filter` is nil, return without allocating a `&SearchFilter{}` so the
marshalled request omits the `filter` key entirely. On the non-nil path keep the existing
Validate-then-assign flow, allocating `req.Filter` lazily as today.

Do NOT make WithFilter/WithSearchWhere reject nil.

In pkg/api/v2/where.go `WhereClauseWhereClauses.Validate`, replace the nested `clause == nil`
guard with `isNilInterface(clause)` so a typed-nil clause produces the existing
`nil clause in %s expression` error rather than panicking in `clause.Validate()`.

Add regression tests. New/updated cases in pkg/api/v2/options_test.go and
pkg/api/v2/search_test.go (both already carry `//go:build basicv2 && !cloud`; any new file
needs that same tag). Cover every bullet in the behavior block above. For the wire-payload
test, marshal two `SearchQuery` values built via `NewSearchRequest` and compare the resulting
JSON bytes for equality rather than asserting on a hand-written string.
  </action>
  <verify>
    <automated>go test -tags=basicv2 -run 'TestEarlyValidationInvalidSearchWhereFilter|TestWithFilter|TestWithSearchWhere|TestWhereClause|TestSearchRequestJSON' ./pkg/api/v2/...</automated>
  </verify>
  <done>Typed-nil WhereClause is inert everywhere (option path and nested And/Or path) with zero panics; `WithFilter(nil)` payload is byte-identical to omitting the option; WithIDs composition works in either order; the weak options_test.go:488 assertion now checks request state.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Convert the three inline nil errors to exported sentinels</name>
  <files>pkg/api/v2/options.go, pkg/api/v2/search.go, pkg/api/v2/search_test.go, pkg/api/v2/groupby_test.go</files>
  <behavior>
    - `errors.Is(err, ErrNilFilter)` is true for `WithSearchFilter(nil).ApplyToSearchRequest(req)`
    - `errors.Is(err, ErrNilRank)` is true for both `WithRank(nil)` and a typed-nil `var kr *KnnRank`
    - `errors.Is(err, ErrNilGroupBy)` is true for `WithGroupBy(nil)`
    - Each sentinel survives the `errors.Wrap(err, "error applying search option")` wrap performed at collection_http.go:430 — assert this by wrapping with `errors.Wrap` in the test and re-checking `errors.Is`
    - Error message strings are unchanged, so `NewSearchRequest(...)` composition tests keep the same text
  </behavior>
  <action>
Add three sentinels to the existing `// Option validation errors.` var block in
pkg/api/v2/options.go, following the surrounding godoc style (one-line comment naming the
option that returns it):

- `ErrNilFilter` — message `filter cannot be nil`, returned by `WithSearchFilter`
- `ErrNilRank` — message `rank cannot be nil`, returned by `WithRank`
- `ErrNilGroupBy` — message `groupBy cannot be nil`, returned by `WithGroupBy`

Keep the exact message text so no existing string assertion elsewhere breaks.

Replace the inline `errors.New` calls with the sentinels: search.go:399
(`searchFilterOption.ApplyToSearchRequest`), search.go:630
(`rankOption.ApplyToSearchRequest`), search.go:673
(`groupByOption.ApplyToSearchRequest`). Return the sentinel value directly — do not wrap it
at the return site, or `errors.Is` identity is preserved but the stack-trace-per-call problem
returns.

Switch the affected assertions from `require.EqualError` to `require.ErrorIs`:
search_test.go:211 (filter), search_test.go:328 and :345 (rank, including the typed-nil case),
groupby_test.go:196 and :299 (groupBy). Add one test that wraps a sentinel with
`errors.Wrap(err, "error applying search option")` and asserts `require.ErrorIs` still matches,
mirroring the real `CollectionImpl.Search` path.
  </action>
  <verify>
    <automated>go test -tags=basicv2 -run 'TestWithSearchFilter|TestWithRank|TestWithGroupBy|TestSearchFilter|TestGroupBy' ./pkg/api/v2/...</automated>
  </verify>
  <done>ErrNilFilter/ErrNilRank/ErrNilGroupBy are exported in the options.go sentinel block, returned by the three Apply methods, matched via errors.Is by the updated tests, and still matched after an errors.Wrap.</done>
</task>

<task type="auto">
  <name>Task 3: Correct the godoc claims, update CHANGELOG, and run full verification</name>
  <files>pkg/api/v2/search.go, pkg/api/v2/options.go, CHANGELOG.md</files>
  <action>
Fix doc accuracy. Keep edits terse — no verbose comment blocks (project CLAUDE.md rule).

1. Remove the unverifiable cross-SDK assertions. Delete the "intentional, permanent divergence
   from the Python and TypeScript SDKs, which treat nil as a no-op" clause from search.go:390-392
   (`WithSearchFilter`), search.go:621-623 (`WithRank`), and search.go:664-666 (`WithGroupBy`).
   Delete "matching Python/TypeScript SDK nil semantics" from options.go:828-829
   (`WithSearchWhere`). KEEP the actionable guidance in each: callers who want to omit the
   filter/rank/group-by should simply not call the option.

2. Reword the error phrasing in those same three comments. These constructors have no error
   return — the error surfaces when the option is applied. Replace "Passing nil returns a
   validation error" with wording along the lines of "Passing nil causes the enclosing search
   request to fail with a validation error", and name the sentinel added in Task 2 (e.g.
   `[ErrNilRank]`) so callers know what to match on.

3. Correct the "treated as 'no filter'" claim on `WithSearchWhere` (options.go:825-829) and
   `WithFilter` (search.go:418-422). State the real, verified contract: nil clears any metadata
   filter previously set on the request (ordinary last-write-wins, not nil-specific), while IDs
   added via [WithIDs] are preserved. Keep the statement that the asymmetry with the
   struct-based options is intentional, but drop the word "tested" from "an intentional, tested
   asymmetry" — Task 1 adds the tests, so either drop "tested" or leave the claim unqualified.

4. Update the `## [v0.4.2] - Unreleased` section of CHANGELOG.md. Add entries for: the
   typed-nil `WhereClause` panic fix (Fixed), the `WithFilter(nil)` empty-`filter`-object
   payload fix (Fixed), and the new `ErrNilFilter`/`ErrNilRank`/`ErrNilGroupBy` sentinels
   (Added). Relocate the Python/TypeScript SDK comparison rationale here — the existing v0.4.2
   entry already carries it, so keep that prose in the CHANGELOG and leave it out of godoc.

5. Run the full verification gate:
   - `make lint`
   - `go test -tags=basicv2 ./pkg/api/v2/...`
   Fix anything either surfaces before finishing.
  </action>
  <verify>
    <automated>make lint &amp;&amp; go test -tags=basicv2 ./pkg/api/v2/... &amp;&amp; ! grep -n "Python and TypeScript SDKs\|Python/TypeScript SDK nil semantics\|intentional, tested asymmetry" pkg/api/v2/search.go pkg/api/v2/options.go</automated>
  </verify>
  <done>No godoc in search.go/options.go asserts cross-SDK behavior or an untested asymmetry; the nil-clears-previous-filter contract is stated accurately; CHANGELOG v0.4.2 documents all three changes; `make lint` and the full basicv2 suite pass.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| caller code → v2 option constructors | Caller-supplied interface values (WhereClause, Rank, *GroupBy) cross into library validation paths |
| SearchRequest → Chroma server | Marshalled filter/rank JSON crosses to the remote API |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-fzu-01 | Denial of Service | `searchWhereOption.ApplyToSearchRequest` (options.go:869) | mitigate | Typed-nil interface values currently panic and crash the host application. Replace the `!= nil` guard with `isNilInterface` (Task 1). |
| T-fzu-02 | Denial of Service | `WhereClauseWhereClauses.Validate` (where.go:467) | mitigate | Same typed-nil panic for clauses nested in `And`/`Or`. Guard with `isNilInterface` (Task 1). |
| T-fzu-03 | Information Disclosure | Godoc cross-SDK claims | mitigate | Doc asserts behavior of other SDKs that CI cannot verify and that may drift, misleading callers into unsafe assumptions. Remove the claims; move rationale to CHANGELOG (Task 3). |
| T-fzu-04 | Tampering | `{"filter":{}}` sent to server | accept | Empty filter object is semantically inert server-side; fixed for payload fidelity (Task 1), not as a security control. |
| T-fzu-SC | Tampering | npm/pip/cargo installs | n/a | No package-manager installs in this plan — Go-only edits to existing files, no new dependencies. |
</threat_model>

<verification>
- `make lint` passes with no new findings
- `go test -tags=basicv2 ./pkg/api/v2/...` passes in full (no regressions in existing tests)
- No new panic paths: typed-nil WhereClause is inert at the option boundary and errors (does not panic) when nested in And/Or
- `WithFilter(nil)` JSON equals the omit-the-option JSON, byte for byte
- `errors.Is` discriminates all three nil sentinels, including through `errors.Wrap`
- No out-of-scope edits: `cloneRank` (collection_http.go:472), missing `Validate()` calls in rankOption/searchFilterOption, non-pointer reflect kinds in isNilRank, and WithSearchFilter's wholesale `req.Filter` overwrite are all left untouched
</verification>

<success_criteria>
- Typed-nil `WhereClause` never panics via `WithFilter`, `WithSearchWhere`, or nested `And`/`Or`
- `WithFilter`/`WithSearchWhere` still accept nil as "no filter" (asymmetry with the struct-based options intact)
- `WithFilter(nil)` emits no `filter` key
- `ErrNilFilter`, `ErrNilRank`, `ErrNilGroupBy` exported and matched via `errors.Is`
- Godoc contains no unverifiable cross-SDK assertions and no inaccurate "no filter"/"tested" wording
- CHANGELOG v0.4.2 covers the panic fix, the payload fix, and the new sentinels
- `make lint` and `go test -tags=basicv2 ./pkg/api/v2/...` both pass
</success_criteria>

<output>
Create `.planning/quick/260730-fzu-fix-typed-nil-whereclause-panic-and-doc-/260730-fzu-SUMMARY.md` when done
</output>
