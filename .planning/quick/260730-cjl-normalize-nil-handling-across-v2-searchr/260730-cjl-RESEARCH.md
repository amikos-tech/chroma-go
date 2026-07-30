# Quick Task 260730-cjl: Research

**Note:** The research subagent hit repeated transient 529 (server overload) errors across
three attempts. These findings were gathered directly via grep/read against the live repo
instead, covering the same focus areas.

## No existing nil-call breakage

`grep -rn "WithSearchFilter(" --include="*.go" .` and `grep -rn "WithRank(" --include="*.go" .`
across the whole repo (including `examples/` and `test/`) turn up zero call sites passing
`nil` literally, and no conditionally-nil variables threaded in. All existing `WithRank(...)`
calls (`examples/v2/search/main.go:203,255`, `pkg/api/v2/client_cloud_test.go:1797,1880`) pass
concrete non-nil `Rank` values. `WithSearchFilter` has no call sites outside its own
definition. **Rejecting nil for both will not break any existing test or example.**

## Error style to match

`pkg/api/v2/search.go` imports `"github.com/pkg/errors"` (not stdlib `errors`) — line 7.
The existing `WithGroupBy(nil)` check (search.go:635-644) uses:

```go
if o.groupBy == nil {
	return errors.New("groupBy cannot be nil")
}
```

New checks should follow this exact shape/tone:

```go
if o.filter == nil {
	return errors.New("filter cannot be nil")
}
```
```go
if o.rank == nil {
	return errors.New("rank cannot be nil")
}
```

## Test structure to mirror

`pkg/api/v2/groupby_test.go:182-216` (`TestWithGroupBy`) is the pattern to copy — a
`t.Run` subtest table per case:
- `"apply valid <x> to search request"` — non-nil happy path, asserts `req.<Field>` is set
- `"nil <x> returns exact validation error"` — asserts `require.EqualError(t, err, "<exact message>")` and `require.Nil(t, req.<Field>)`

`pkg/api/v2/groupby_test.go:218-251` (`TestSearchRequestWithGroupBy`) additionally pins the
composed `NewSearchRequest(..., WithGroupBy(nil))` fail-before-append path — the Phase 22
summary calls this out explicitly ("Pinned both the direct option path and the composed
NewSearchRequest path"). The new Filter/Rank tests should add an equivalent composed-path
case: `NewSearchRequest(WithSearchFilter(nil))` / `NewSearchRequest(WithRank(nil))` must
return an error and must NOT append to `SearchQuery.Searches`.

Filter and rank tests currently live inline in `search_test.go` (no dedicated
`searchfilter_test.go`/`rank_test.go` exists) — new nil-contract subtests can go in
whichever test file already covers `WithSearchFilter`/`WithRank`'s happy path, or a new
`t.Run` block appended there. Match existing file organization rather than creating new files.

## CHANGELOG format

`CHANGELOG.md` follows Keep a Changelog format. The `WithGroupBy(nil)` change from Phase 22
is already documented under `## [v0.4.2] - Unreleased` → `### Changed` (line 11):

```markdown
- **Search API** - `WithGroupBy(nil)` now returns a validation error instead of silently omitting grouping. Callers that want no grouping should omit `WithGroupBy(...)` entirely.
```

The new entry for this task should extend/replace this line in the same `### Changed`
section, covering all three options plus the explicit Python/TS divergence note, e.g.:

```markdown
- **Search API** - `WithSearchFilter(nil)` and `WithRank(nil)` now return validation errors, matching `WithGroupBy(nil)`'s existing behavior. Callers that want to omit a filter/rank/group-by should omit the option entirely rather than passing nil. This is an intentional divergence from the Python and TypeScript SDKs, which treat nil/None/undefined as a no-op for these options — chroma-go treats an explicit nil as a likely caller bug rather than an omission signal.
```

## Doc comment placement

Add a short note to the package-level or option-level doc comments near `WithSearchFilter`,
`WithRank`, and `WithGroupBy` (search.go) stating the nil-rejection contract and the
intentional Python/TS divergence, so it's discoverable via `go doc` without needing the
CHANGELOG. Keep it to 1-2 lines per function — CLAUDE.md's "no comments beyond what's
necessary" guidance applies; this qualifies as non-obvious (a permanent cross-SDK behavioral
choice) so it earns a comment.

## No pitfalls found

- `Rank` is a plain interface (`pkg/api/v2/rank.go:51`) — no typed-nil footgun.
- `SearchFilter` is `*SearchFilter` (a concrete pointer) — no typed-nil footgun either, since
  `searchFilterOption.filter` is assigned directly from the `WithSearchFilter(filter *SearchFilter)`
  parameter with no interface boxing in between.
