---
task: 260803-occ-address-rrf-degenerate-composition-guard
type: execute
wave: 1
depends_on: []
files_modified:
  - pkg/api/v2/search.go
  - pkg/api/v2/results.go
  - pkg/api/v2/rank.go
  - pkg/api/v2/collection_http.go
  - pkg/api/v2/search_test.go
  - pkg/api/v2/rank_test.go
  - pkg/api/v2/collection_http_test.go
  - pkg/api/v2/client_cloud_test.go
autonomous: true
requirements: ["#497", "#498", "#500", "#501"]

must_haves:
  truths:
    - "A search response with a per-element null score or document keeps every remaining element bound to its original ID position"
    - "ResultRow reports HasScore/HasDocument consistently for rows produced by Search, Query and Get"
    - "rrf.Log() produces a Rank that cannot be sent: it fails with an exported sentinel error"
    - "rrf.Max(Val(0)) / rrf.Max(FloatOperand(0)) / rrf.Max(IntOperand(0)) fail with an exported sentinel error; non-zero and non-constant operands stay legal"
    - "rrf.Abs(), rrf.Negate() and rrf.Min(Val(0)) remain legal and unchanged"
    - "Search returns (nil, err) with an exported sentinel when a group selected KScore but score cardinality does not match ID cardinality"
    - "make test (basicv2) and make lint pass; cloud test pins for Log and Max_0 assert the new error paths"
  artifacts:
    - path: "pkg/api/v2/search.go"
      provides: "null-preserving score/document parsing plus presence tracking"
    - path: "pkg/api/v2/results.go"
      provides: "ResultRow.HasScore / ResultRow.HasDocument"
    - path: "pkg/api/v2/rank.go"
      provides: "deferred-error degenerate RRF rank + exported sentinels"
    - path: "pkg/api/v2/collection_http.go"
      provides: "read-time cardinality guard in CollectionImpl.Search"
  key_links:
    - from: "pkg/api/v2/collection_http.go Search"
      to: "sq.Searches[g].Select"
      via: "per-group KScore gating of the cardinality guard"
    - from: "pkg/api/v2/rank.go RrfRank.Log/Max"
      to: "marshalRank / validateBuiltInRankWithDepth"
      via: "deferred-error rank leaf, UnknownRank precedent"
---

<objective>
Fix silent score/document misalignment on null elements in Search responses, then add the
two guards that only become sound once alignment is correct: a build-time deferred-error
guard on provably degenerate `*RrfRank` compositions, and a read-time cardinality guard in
`CollectionImpl.Search`.

Purpose: closes #497, #498, #500, #501. The alignment bug is silent data corruption and is
the actual mechanism behind #500's symptom.
Output: three ordered changes in `pkg/api/v2` plus unit-test coverage and flipped cloud pins.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
</execution_context>

<context>
@.planning/quick/260803-occ-address-rrf-degenerate-composition-guard/260803-occ-CONTEXT.md
@CLAUDE.md
@pkg/api/v2/search.go
@pkg/api/v2/results.go
@pkg/api/v2/rank.go
@pkg/api/v2/collection_http.go

<interfaces>
Existing shapes the executor must build against (already in the codebase — do not re-derive):

pkg/api/v2/results.go:14
  type ResultRow struct { ID DocumentID; Document string; Metadata DocumentMetadata; Embedding []float32; Score float64 }

pkg/api/v2/search.go:773
  type SearchResultImpl struct { IDs [][]DocumentID; Documents [][]string; Metadatas [][]DocumentMetadata; Embeddings [][][]float32; Scores [][]float64 }
  Null-preserving precedent already present: Metadatas at search.go:862, Embeddings at search.go:893.

pkg/api/v2/search.go:177   KScore Key = "#score"
pkg/api/v2/search.go:282   type SearchRequest struct { Filter *SearchFilter; Limit *SearchPage; Rank Rank; Select *SearchSelect; GroupBy *GroupBy }
pkg/api/v2/search.go:105   type SearchQuery struct { ...; Searches []SearchRequest `json:"searches"` }
  SearchSelect carries `Keys []Key` (see WithSelectAll at search.go:613).

pkg/api/v2/rank.go:100  UnknownRank — deferred-error precedent (embeds ValRank, MarshalJSON returns an error)
pkg/api/v2/rank.go:108  marshalRank — built-in type switch, falls through to rank.MarshalJSON()
pkg/api/v2/rank.go:178  validateBuiltInRankWithDepth — `case *UnknownRank, *ValRank:` calls rank.MarshalJSON()
pkg/api/v2/rank.go:1453 func (r *RrfRank) Log() Rank
pkg/api/v2/rank.go:1457 func (r *RrfRank) Max(operand Operand) Rank
pkg/api/v2/rank.go:1534 operandToRank — supports Rank, IntOperand, FloatOperand
pkg/api/v2/collection_http.go:475 cloneRank — `default: return rank` (a new immutable leaf needs no case)
pkg/api/v2/options.go:936  ErrNilRank — exported-sentinel style to match
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Preserve null positions in Search scores/documents and add ResultRow presence flags</name>
  <files>pkg/api/v2/search.go, pkg/api/v2/results.go, pkg/api/v2/search_test.go, pkg/api/v2/results_test.go</files>
  <behavior>
    - `{"ids":[["a","b","c"]],"scores":[[1.5,null,3.5]],"documents":[["da",null,"dc"]]}` yields rows: a/1.5/"da", b/0/"" with HasScore=false and HasDocument=false, c/3.5/"dc". This is the exact regression case from CONTEXT.md.
    - A group whose scores/documents contain no nulls yields HasScore=true and HasDocument=true for every row.
    - A group where scores were not selected at all (field absent) yields HasScore=false for every row.
    - QueryResultImpl rows: HasScore=true only where a distance exists at that index; HasDocument=true only where the document pointer is non-nil.
    - GetResultImpl rows: HasScore=false always (Get carries no score); HasDocument=true only where the document pointer is non-nil.
  </behavior>
  <action>
    In `SearchResultImpl.UnmarshalJSON`, add an explicit nil case to the scores element type
    switch (search.go:930) and the documents element check (search.go:840) so a JSON null
    appends a zero value at its own index instead of being skipped. Mirror the shape already
    used for Metadatas and Embeddings so all four parsers read the same way.

    Because `Documents [][]string` and `Scores [][]float64` cannot represent absence, track it
    in two new unexported parallel slices on `SearchResultImpl` (e.g. `documentPresent [][]bool`,
    `scorePresent [][]bool`), populated only by `UnmarshalJSON`. Keep the exported fields and
    their JSON tags unchanged — this must stay backward compatible.

    Add `HasDocument bool` and `HasScore bool` to `ResultRow` (results.go:14) as additive fields
    per the LOCKED absence-representation decision. Do NOT change `Score` to `*float64`.

    Set both flags on ALL three construction paths, since `ResultRow` is shared:
      - `SearchResultImpl.buildRow` (search.go:997): flag is true when the index is in range AND
        the corresponding presence slice either has no entry for that group (direct struct
        construction, not unmarshalled) or records true. This keeps hand-constructed
        SearchResultImpl values behaving sanely.
      - `QueryResultImpl.buildRow` (results.go:459): HasScore from DistancesLists index-in-range,
        HasDocument from the existing non-nil document check.
      - `GetResultImpl.buildRow` (results.go:192): HasScore always false, HasDocument from the
        existing non-nil document check.

    Document the two new ResultRow fields with a one-line comment each stating that false means
    the server sent null or the field was not selected. No other comments.
  </action>
  <verify>
    <automated>go test -tags=basicv2 -run 'TestSearchResult|TestResultRow|TestQueryResult|TestGetResult' ./pkg/api/v2/... </automated>
  </verify>
  <done>The three-element null fixture round-trips with every score and document bound to its original ID; HasScore/HasDocument are correct on Search, Query and Get rows; existing basicv2 tests still pass.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Deferred-error build-time guard for rrf.Log() and rrf.Max(literal zero)</name>
  <files>pkg/api/v2/rank.go, pkg/api/v2/rank_test.go</files>
  <behavior>
    - `rrf.Log()` returns a non-nil Rank; `json.Marshal` on it and `WithRank(...)` both fail, and the error satisfies `errors.Is(err, ErrDegenerateRrfLog)`.
    - `rrf.Max(Val(0))`, `rrf.Max(FloatOperand(0))`, `rrf.Max(IntOperand(0))` and `rrf.Max(FloatOperand(-0.0))` all fail with `errors.Is(err, ErrDegenerateRrfMaxZero)`.
    - `rrf.Max(FloatOperand(1.0))`, `rrf.Max(Val(0.5))` and `rrf.Max(someKnnRank)` marshal successfully — non-zero and non-constant operands stay legal.
    - `rrf.Abs()`, `rrf.Negate()`, `rrf.Exp()`, `rrf.Min(FloatOperand(0))` and all of Add/Sub/Multiply/Div marshal successfully — unchanged.
    - A degenerate rank nested inside a larger expression (e.g. `rrf.Log().Add(FloatOperand(1))`) still fails at marshal time.
  </behavior>
  <action>
    Per the LOCKED guard-mechanism decision, follow the `*UnknownRank` precedent at rank.go:85-105:
    add an unexported leaf type (e.g. `degenerateRrfRank`) that embeds `ValRank`, carries an
    `err error` field, and whose `MarshalJSON` returns that error wrapped with a message naming
    the offending composition and why it degenerates.

    Export two sentinels next to the existing error style so callers can `errors.Is` them:
    `ErrDegenerateRrfLog` and `ErrDegenerateRrfMaxZero`. Wording should state the cause —
    RrfRank.MarshalJSON negates the fusion sum, so the composed value is always <= 0, making
    log() NaN and max(x, 0) a constant zero.

    Change `(*RrfRank).Log()` (rank.go:1453) to return the degenerate leaf carrying
    ErrDegenerateRrfLog. Change `(*RrfRank).Max(operand)` (rank.go:1457) to return the degenerate
    leaf carrying ErrDegenerateRrfMaxZero ONLY when the operand is a statically-knowable literal
    zero; otherwise keep the existing `&MaxRank{...}` behavior. Add an unexported helper that
    recognizes literal zero for `FloatOperand`, `IntOperand` and `*ValRank` with value 0
    (Go's `== 0` already matches negative zero — no special case needed). Any other operand,
    including any Rank, is not statically knowable and stays legal.

    Do NOT touch Abs, Negate, Exp or Min — LOCKED as legal.

    Wire the new type into `validateBuiltInRankWithDepth` (rank.go:178) by adding it to the
    existing `case *UnknownRank, *ValRank:` leaf branch, so `WithRank` (search.go:655) rejects it
    eagerly rather than only at request marshal time. `marshalRank` already falls through to
    `rank.MarshalJSON()` and `cloneRank`'s default returns the value unchanged, so neither needs
    a new case — verify that is still true rather than adding one.

    Update the `RrfRank` doc comment at rank.go:1305-1328: Log and Max(0) are now rejected at
    build time; Abs and Negate remain warnings only. No panics, no `Must*`.
  </action>
  <verify>
    <automated>go test -tags=basicv2 -run 'TestRrf|TestRank|TestDegenerate|TestWithRank' ./pkg/api/v2/...</automated>
  </verify>
  <done>Both sentinels are exported and reachable via errors.Is from json.Marshal and from WithRank; only Log and literal-zero Max are rejected; every other RRF arithmetic path is unchanged and still passes.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 3: Read-time cardinality guard in CollectionImpl.Search and cloud-pin flip</name>
  <files>pkg/api/v2/collection_http.go, pkg/api/v2/collection_http_test.go, pkg/api/v2/client_cloud_test.go</files>
  <behavior>
    - Server response with `ids[0]` of length 5 and `scores[0]` of length 0, for a request whose Select includes KScore: Search returns (nil, err) with `errors.Is(err, ErrDegenerateSearchResult)`.
    - Same mismatched response for a request whose Select omits KScore (or Select is nil): Search returns the result with no error — this is the #500 false-positive that the gate exists to prevent.
    - Well-formed response with equal ID and score cardinality and KScore selected: no error.
    - Response with per-element nulls (post Task 1, cardinalities equal): no error.
    - Empty `ids[g]`: no error regardless of scores.
  </behavior>
  <action>
    In `CollectionImpl.Search` (collection_http.go:426), after the response unmarshals into
    `result`, run the guard while `sq` is still in scope — the LOCKED decision requires it to fire
    without caller opt-in. For each group index g where `g < len(sq.Searches)`: skip unless that
    request's `Select` is non-nil and its `Keys` contain `KScore`; then error when
    `len(result.IDs[g]) > 0 && len(result.Scores[g]) != len(result.IDs[g])`. Guard every index
    access so a short or absent Scores outer slice cannot panic.

    Add an exported sentinel `ErrDegenerateSearchResult` in the same style as `ErrNilRank`
    (options.go:936). Return `(nil, wrapped)` with the group index and both lengths in the
    message. Placement of the check body (inline vs. an unexported helper) is discretionary,
    but it must execute inside Search.

    Then flip the cloud pins in `TestCloudClientSearchRRFArithmetic`
    (client_cloud_test.go:1728). The `Log` case (line ~1983) and `Max_0` case (line ~2002)
    currently assert the degenerate behavior with `require.NoError`; they must now assert
    `require.Error` plus `errors.Is` against `ErrDegenerateRrfLog` / `ErrDegenerateRrfMaxZero`,
    and stop asserting on results. Leave `Min_0`, `Negate`, `Abs`, `Exp` and the safe bucket
    untouched. Update the corpus-limitation comment block (line ~1846) so it no longer describes
    Log and Max(0) as pinned server observations.
  </action>
  <verify>
    <automated>go test -tags=basicv2 -run 'TestCollection.*Search|TestSearchDegenerate' ./pkg/api/v2/... && go vet -tags='cloud' ./pkg/api/v2/...</automated>
  </verify>
  <done>Search errors with the exported sentinel only when KScore was selected and cardinality diverges; the KScore-absent case stays error-free; the cloud test compiles and its Log/Max_0 subtests assert the new error paths.</done>
</task>

</tasks>

<verification>
1. `make test` — full basicv2 suite green.
2. `make lint` — clean (run before any commit, per CLAUDE.md).
3. `go vet -tags=cloud ./pkg/api/v2/...` — cloud test file still compiles after the pin flip.
4. Manual read-back of the CONTEXT.md null fixture: for scores [1.5, null, 3.5] over ids [a,b,c],
   row a has Score 1.5 / HasScore true, row b has Score 0 / HasScore FALSE (it must NOT receive
   3.5), and row c has Score 3.5 / HasScore true. Same shape for documents.
5. No `Must*` calls and no `panic(` added to production files.
</verification>

<success_criteria>
- Null score/document elements preserve position; a regression test asserts the exact CONTEXT.md fixture.
- `ResultRow.HasScore` / `HasDocument` set correctly on Search, Query and Get paths.
- `ErrDegenerateRrfLog`, `ErrDegenerateRrfMaxZero`, `ErrDegenerateSearchResult` exported and `errors.Is`-matchable.
- Only Log and literal-zero Max rejected at build time; Abs, Negate, Min(0) untouched.
- Read-time guard gated on KScore selection; no false positive on ordinary queries.
- `make test` and `make lint` pass; cloud pins flipped.
</success_criteria>

<output>
Conventional commits per CLAUDE.md. Suggested split:
- `fix(api): preserve null positions in search scores and documents`
- `feat(api): reject degenerate RRF Log and Max(0) compositions`
- `feat(api): guard degenerate search result cardinality`

PR body must close #497, #498, #500, #501 and note that #497's "server-behavior defect"
framing is incorrect — the behavior is deterministic IEEE 754 math with no upstream guard;
the Go client addresses it client-side per its own eager-rejection house style.
</output>
