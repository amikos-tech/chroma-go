---
task: 260803-occ-address-rrf-degenerate-composition-guard
type: execute
status: complete
requirements: ["#497", "#498", "#500", "#501"]
key-files:
  modified:
    - pkg/api/v2/search.go
    - pkg/api/v2/results.go
    - pkg/api/v2/rank.go
    - pkg/api/v2/collection_http.go
    - pkg/api/v2/search_test.go
    - pkg/api/v2/results_test.go
    - pkg/api/v2/rank_test.go
    - pkg/api/v2/collection_http_test.go
    - pkg/api/v2/client_cloud_test.go
commits:
  - 528fcd2 fix(api): preserve null positions in search scores and documents
  - db664d2 feat(api): reject degenerate RRF Log and Max(0) compositions
  - 66a63dc feat(api): guard degenerate search result cardinality
---

# Quick Task 260803-occ: RRF degenerate composition guards — Summary

Fixed silent score/document misalignment on null elements in Search responses, then added
the build-time guard on degenerate `*RrfRank` compositions and the read-time cardinality
guard in `CollectionImpl.Search` that only become sound once alignment is correct.

## Task 1 — null-position preservation (528fcd2)

- `SearchResultImpl.UnmarshalJSON` now appends a zero value at the original index for a
  JSON `null` (or any non-matching type) in `scores` and `documents`, matching the
  nil-handling already used for `Metadatas` and `Embeddings`.
- Two unexported parallel slices (`documentPresent`, `scorePresent`) track per-element
  presence; exported fields and JSON tags are unchanged (backward compatible).
- `ResultRow` gained additive `HasDocument` / `HasScore` booleans, set on all three
  construction paths (`SearchResultImpl.buildRow`, `QueryResultImpl.buildRow`,
  `GetResultImpl.buildRow`). `elementPresent` treats a missing presence entry as present,
  so hand-constructed `SearchResultImpl` values still report `true`.
- Regression test asserts the exact CONTEXT.md fixture: ids `[a,b,c]`, scores
  `[1.5,null,3.5]`, documents `["da",null,"dc"]` → row b keeps score 0 / `HasScore=false`
  and row c keeps 3.5.

## Task 2 — build-time degenerate RRF guard (db664d2)

- Exported sentinels `ErrDegenerateRrfLog` and `ErrDegenerateRrfMaxZero` in `rank.go`.
- New unexported `degenerateRrfRank` leaf embeds `ValRank` and fails in `MarshalJSON`,
  following the `*UnknownRank` deferred-error precedent. Its own arithmetic methods return
  the receiver so the error survives further composition
  (`rrf.Log().Add(FloatOperand(1))` still fails).
- `(*RrfRank).Log()` always returns the leaf. `(*RrfRank).Max(operand)` returns it only for
  a statically-knowable literal zero (`IntOperand`, `FloatOperand`, `*ValRank`; negative
  zero compares equal so needs no case). Everything else keeps `&MaxRank{...}`.
- Wired into `validateBuiltInRankWithDepth`'s leaf branch so `WithRank` rejects eagerly.
  `marshalRank` (falls through to `MarshalJSON`) and `cloneRank` (default returns the
  value) needed no new cases — verified, not modified.
- `RrfRank` doc comment updated: Log and Max(0) are build-time rejections; Abs and Negate
  remain warnings only.

## Task 3 — read-time cardinality guard and cloud pin flip (66a63dc)

- Exported `ErrDegenerateSearchResult` plus `checkSearchResultCardinality` / `selectsScore`
  in `collection_http.go`, invoked inside `Search` right after the response unmarshals so
  it fires without caller opt-in.
- Gate: only for group `g` where `sq.Searches[g].Select` is non-nil and includes `KScore`,
  and `len(IDs[g]) > 0`; errors when the score count differs. All index accesses are bounds
  checked, so a short or absent `Scores` outer slice cannot panic. `Search` returns
  `(nil, err)` wrapped with the group index and both lengths.
- Cloud pins in `TestCloudClientSearchRRFArithmetic` flipped: `Log` and `Max_0` now assert
  `require.Error` + `errors.Is` against the new sentinels and stop asserting on results.
  `Min_0`, `Negate`, `Abs`, `Exp` and the safe bucket are untouched. The corpus-limitation
  comment no longer describes Log and Max(0) as pinned server observations.

## Deviations from Plan

**1. [Rule 3 - Blocking] Existing `TestRrfRankArithmetic` pinned the old behavior**
- Found during: Task 2.
- Issue: the table cases `Log` and `Max` (with `FloatOperand(0.0)`) and the
  "wrappers remain independent" subtest asserted successful marshaling of the now-rejected
  compositions.
- Fix: removed the `Log` case, changed the `Max` case operand to `FloatOperand(2.0)`, and
  switched the independence subtest's third wrapper from `rrf.Log()` to `rrf.Exp()`. The new
  `TestDegenerateRrfCompositions` covers the rejected paths.
- Commit: db664d2.

**2. [Scope] `degenerateRrfRank` overrides the promoted arithmetic methods**
- The plan only specified `MarshalJSON`. Because `ValRank` is embedded by value, promoted
  methods would compose against the zero-valued embedded `ValRank` and silently drop the
  error, which contradicts the plan's own behavior requirement that a nested degenerate rank
  still fails. Ten one-line methods return the receiver instead.

**3. [Test shape] Query/Get null-document tests use direct construction**
- `NewTextDocumentsFromInterface` rejects a JSON `null` document with an error on the
  Query/Get paths, so a null-document wire fixture cannot reach `buildRow` there. Those two
  tests construct `Documents{doc, nil}` directly. Changing the Query/Get document parser is
  out of scope for this task and was not touched.

No architectural changes, no auth gates, no deferred issues.

## Verification

- `make lint` — 0 issues (run before each commit).
- `make test` (basicv2) — 2087 tests, 7 skipped, all passing.
- `go vet -tags=cloud ./pkg/api/v2/...` — clean; the cloud test file compiles after the pin
  flip. Cloud tests were not run (no credentials); compilation was the agreed bar.
- No `Must*` calls and no `panic(` added to production files.

## Self-Check: PASSED

- All four production files and five test files exist and are committed.
- Commits 528fcd2, db664d2, 66a63dc all present in `git log`.
- Working tree clean after Task 3.
