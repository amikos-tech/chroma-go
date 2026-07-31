---
phase: quick-260731-ewm
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - pkg/api/v2/rank.go
  - pkg/api/v2/rank_test.go
  - pkg/api/v2/options.go
  - pkg/api/v2/search.go
  - pkg/api/v2/collection_http.go
  - pkg/api/v2/collection_http_test.go
  - pkg/api/v2/search_test.go
  - .planning/quick/260731-e0k-address-typed-nil-rank-arithmetic-and-re/260731-e0k-PLAN.md
  - .planning/quick/260731-e0k-address-typed-nil-rank-arithmetic-and-re/260731-e0k-SUMMARY.md
autonomous: true
requirements:
  - EWM-01
  - EWM-02
  - EWM-03
  - EWM-04
  - EWM-05
  - EWM-06
  - EWM-07
  - EWM-08

must_haves:
  truths:
    - "Arithmetic and unary expressions built from a typed-nil *KnnRank receiver never panic during composite marshaling and return an error matching ErrNilRank"
    - "SumRank, SubRank, MulRank, DivRank, AbsRank, ExpRank, LogRank, MaxRank, and MinRank reject nil or typed-nil children before invoking their MarshalJSON methods"
    - "A typed-nil Rank operand reports ErrNilRank, while an untyped nil Operand retains Val(0) chaining and a genuinely unsupported Operand retains the unknown-operand error"
    - "A typed-nil child in RrfRank is preserved through cloneRank and fails through Collection.Search instead of being silently converted into a best-ranked Val(0) term"
    - "WithRank rejects nil, a direct typed-nil SearchRequest.Rank remains an omitted optional field, and exported documentation states this difference"
    - "The HTTP request-body regression cannot block forever waiting on its capture channel"
    - "The earlier plan and summary accurately say their arithmetic test used a valid *ValRank receiver with a typed-nil *KnnRank operand"
  artifacts:
    - path: "pkg/api/v2/rank.go"
      provides: "Shared nil-child marshal guard, accurate invalid-rank conversion, and RRF child validation"
      contains: "func marshalRank"
    - path: "pkg/api/v2/rank_test.go"
      provides: "No-panic regressions for typed-nil receivers, every composite shape, RRF, and distinct nil/unknown errors"
      contains: "TestRankTypedNilReceiverMarshal"
    - path: "pkg/api/v2/collection_http.go"
      provides: "Deep cloning that preserves nested typed-nil Rank identity"
      contains: "func cloneRank"
    - path: "pkg/api/v2/collection_http_test.go"
      provides: "Collection.Search nested-RRF failure regression and bounded request-body capture"
      contains: "TestCollectionSearchTypedNilRank"
    - path: "pkg/api/v2/search.go"
      provides: "Exported SearchRequest.Rank nil-behavior documentation"
      contains: "type SearchRequest struct"
    - path: ".planning/quick/260731-e0k-address-typed-nil-rank-arithmetic-and-re/260731-e0k-SUMMARY.md"
      provides: "Correct historical account of the earlier arithmetic regression"
      contains: "valid *ValRank"
  key_links:
    - from: "pkg/api/v2/rank.go:composite MarshalJSON methods"
      to: "pkg/api/v2/rank.go:marshalRank"
      via: "all nine composite shapes marshal children through the nil-aware helper"
      pattern: "marshalRank\\("
    - from: "pkg/api/v2/rank.go:marshalRank"
      to: "pkg/api/v2/options.go:ErrNilRank"
      via: "typed-nil children return the canonical nil-rank error"
      pattern: "ErrNilRank"
    - from: "pkg/api/v2/collection_http.go:cloneRank"
      to: "pkg/api/v2/rank.go:RrfRank.Validate"
      via: "the clone retains a nested typed nil so RRF validation can reject it"
      pattern: "return rank"
    - from: "pkg/api/v2/collection_http_test.go:TestCollectionSearchTypedNilRank"
      to: "pkg/api/v2/collection_http.go:CollectionImpl.Search"
      via: "the full HTTP path returns ErrNilRank without panicking or sending a substituted term"
      pattern: "ErrorIs.*ErrNilRank"
---

<objective>
Make every composite rank serialization boundary reject nil children predictably, and stop nested RRF cloning from turning invalid input into a valid-looking best-ranked term.

Purpose: typed-nil ranks must produce an accurate, stable error instead of a panic or silent score corruption, while the intentionally different top-level SearchRequest omission behavior remains documented and tested.

Output: one shared composite marshal guard, preserved nested typed-nil clone semantics, corrected RRF/Search regressions, bounded HTTP test synchronization, public nil-contract documentation, and corrected historical GSD wording.
</objective>

<execution_context>
@/Users/tazarov/.codex/get-shit-done/workflows/execute-plan.md
@/Users/tazarov/.codex/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@pkg/api/v2/rank.go
@pkg/api/v2/rank_test.go
@pkg/api/v2/options.go
@pkg/api/v2/search.go
@pkg/api/v2/search_test.go
@pkg/api/v2/collection_http.go
@pkg/api/v2/collection_http_test.go
@.planning/quick/260731-e0k-address-typed-nil-rank-arithmetic-and-re/260731-e0k-PLAN.md
@.planning/quick/260731-e0k-address-typed-nil-rank-arithmetic-and-re/260731-e0k-SUMMARY.md

**Locked behavior:**
- Keep `operandToRank(nil) -> Val(0)` for an untyped nil Operand.
- Keep `WithRank(nil or typed-nil)` returning `ErrNilRank`.
- Keep a direct `SearchRequest{Rank: typedNil}` rank-free: both direct marshaling and `Collection.Search` omit the optional `rank` field.
- Reject typed-nil arithmetic operands, typed-nil children created from receiver calls, and typed-nil `RankWithWeight.Rank` values with `ErrNilRank`; do not route nil input through `UnknownRank`.
- Keep `UnknownRank` for genuinely unsupported Operand implementations and retain an accurate unknown-operand error for that case.
- Preserve a nested typed-nil Rank's dynamic type in `cloneRank`; do not normalize it to untyped nil.
- Use the nine composite `MarshalJSON` methods as the serialization boundary. Do not add nil guards across the arithmetic/unary method matrix.
- Keep `isNilRank` and `isNilInterface` in `search.go`: moving two working helpers into a new file would add churn without changing this quick task's behavior or contract (EWM-07).
</context>

<interfaces>
<!-- Existing contracts the executor should use directly. -->

From `pkg/api/v2/rank.go`:

```go
type Rank interface {
	Operand
	Multiply(operand Operand) Rank
	Sub(operand Operand) Rank
	Add(operand Operand) Rank
	Div(operand Operand) Rank
	Negate() Rank
	Abs() Rank
	Exp() Rank
	Log() Rank
	Max(operand Operand) Rank
	Min(operand Operand) Rank
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(b []byte) error
}

type RankWithWeight struct {
	Rank   Rank
	Weight float64
}

func operandToRank(operand Operand) Rank
```

From `pkg/api/v2/options.go`:

```go
var ErrNilRank = errors.New("rank cannot be nil")
```

From `pkg/api/v2/search.go`:

```go
type SearchRequest struct {
	Filter  *SearchFilter
	Limit   *SearchPage
	Rank    Rank
	Select  *SearchSelect
	GroupBy *GroupBy
}

func WithRank(rank Rank) *rankOption
func isNilRank(rank Rank) bool
```

From `pkg/api/v2/collection_http.go`:

```go
func cloneRank(rank Rank) Rank
```
</interfaces>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Reject nil children at every composite marshal boundary</name>
  <files>pkg/api/v2/rank.go, pkg/api/v2/rank_test.go, pkg/api/v2/options.go, pkg/api/v2/search.go</files>
  <behavior>
    - "For `var k *KnnRank`, each of Add, Multiply, Sub, Div, Max, Min, Negate, Abs, Exp, and Log can build an expression; marshaling that expression does not panic and returns an error matching ErrNilRank"
    - "Sum/Mul/Max/Min reject a nil child in their variadic slices; Sub/Div reject nil on either side; Abs/Exp/Log reject their nil unary child"
    - "RrfRank rejects a nil or typed-nil RankWithWeight.Rank with an indexed error matching ErrNilRank"
    - "A typed-nil Rank passed as an arithmetic operand reports ErrNilRank, an untyped nil Operand still serializes as Val(0), and a custom unsupported Operand still reports unknown operand type"
  </behavior>
  <action>
    For EWM-01, EWM-03, and EWM-08, first update `pkg/api/v2/rank_test.go`. Change `TestRankArithmeticTypedNilOperandMarshal` to require `ErrorIs(err, ErrNilRank)` rather than the misleading unknown-operand text. Add `TestRankTypedNilReceiverMarshal`, using `var k *KnnRank` (also assigned to a `Rank` interface for at least one row) and a table over all six binary and four unary methods; build the expression outside or inside `require.NotPanics` as appropriate, then marshal inside `require.NotPanics` and require `ErrNilRank`. This receiver table is specifically for the non-dereferencing `KnnRank` builders; do not claim arbitrary methods on every nil concrete Rank are valid.

    Add `TestCompositeRankNilChildMarshal` as a table that directly constructs all nine package-private composite shapes. Cover SumRank, MulRank, MaxRank, and MinRank with a typed-nil child in the slice; SubRank and DivRank with separate left/right nil cases; and AbsRank, ExpRank, and LogRank with a typed-nil child. Every row must call `MarshalJSON` under `require.NotPanics` and require `ErrorIs(err, ErrNilRank)`. Add an RRF nil-child case to `TestRrfRank`, including `ErrorIs`, and extend operand conversion coverage with both untyped nil (still `Val(0)`) and a small test-only unsupported Operand type (still the unknown-operand error). Run the focused tests once before production edits and confirm the receiver/composite/RRF cases fail by panic or wrong error.

    In `pkg/api/v2/rank.go`, add one unexported `marshalRank(rank Rank) ([]byte, error)` helper that returns `ErrNilRank` when `isNilRank(rank)` and otherwise delegates to `rank.MarshalJSON`. Route every child serialization in `SumRank.MarshalJSON`, `SubRank.MarshalJSON`, `MulRank.MarshalJSON`, `DivRank.MarshalJSON`, `AbsRank.MarshalJSON`, `ExpRank.MarshalJSON`, `LogRank.MarshalJSON`, `MaxRank.MarshalJSON`, and `MinRank.MarshalJSON` through this helper. Preserve term/depth/division checks and JSON shapes.

    In `operandToRank`, retain the leading untyped-nil `Val(0)` branch, but return a true nil Rank from the typed-nil `case Rank` path so the enclosing composite reaches `marshalRank`; keep unsupported types returning `&UnknownRank{}`. In `RrfRank.Validate`, reject every nil or typed-nil `rw.Rank` before weight processing with an index-bearing error that wraps `ErrNilRank`. Update the `UnknownRank` and `operandToRank` comments so UnknownRank describes only unsupported Operand conversion and nil input describes `ErrNilRank`; do not use unknown-operand wording for nil rank input.

    For EWM-04, update exported documentation without hiding the asymmetry: the `Rank` comment must state that expressions containing typed-nil operands/children fail with `ErrNilRank` during marshaling; `RankWithWeight.Rank` must be non-nil; `WithRank` rejects nil and typed nil while applying the option; and direct assignment of nil or typed nil to the optional `SearchRequest.Rank` field is omitted by `SearchRequest.MarshalJSON` and `Collection.Search`. Also broaden the `ErrNilRank` comment in `options.go` beyond only WithRank. State that directly calling methods on arbitrary nil concrete pointers is invalid; the guarantee is at supported option and composite serialization boundaries. Keep the existing helpers in `search.go` per EWM-07.
  </action>
  <verify>
    <automated>go test -tags=basicv2 ./pkg/api/v2 -run '^(TestUnknownRankError|TestRankArithmeticTypedNilOperandMarshal|TestRankTypedNilReceiverMarshal|TestCompositeRankNilChildMarshal|TestOperandConversion|TestRrfRank)$' -count=1 -timeout=30s</automated>
  </verify>
  <done>All ten typed-nil KnnRank receiver compositions and every composite child shape marshal without panic, nil errors match ErrNilRank, unsupported operands retain their distinct error, untyped nil retains Val(0), and exported docs accurately distinguish the supported entry paths.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Preserve nested typed nils and fail RRF searches cleanly</name>
  <files>pkg/api/v2/collection_http.go, pkg/api/v2/collection_http_test.go, pkg/api/v2/search_test.go</files>
  <behavior>
    - "cloneRank(nil) remains nil, while cloneRank of a typed-nil Rank preserves its concrete dynamic type instead of laundering it to untyped nil"
    - "A typed-nil Rank nested in RrfRank remains typed nil after the full RRF deep clone"
    - "Collection.Search with a nested typed-nil RRF child does not panic, returns an error matching ErrNilRank, and sends no substituted Val(0) request"
    - "Direct typed-nil SearchRequest.Rank omission and mixed-batch ordering continue to produce their existing exact payloads"
    - "Every expected request body is received with a bounded select/time.After assertion"
  </behavior>
  <action>
    For EWM-02 and EWM-08, update `TestCloneRankTypedNil` in `pkg/api/v2/search_test.go` before production code. Replace the “clones to true nil” expectation with assertions that a typed-nil `*KnnRank` or `*ValRank` remains `isNilRank` but retains the same concrete type after cloning. In the nested RRF subtest, type-assert the cloned child back to `*KnnRank` and require the pointer itself is nil; this distinguishes preservation from an untyped nil interface. Keep the top-level `embedTextQueries` regression proving a direct typed-nil request normalizes to omission.

    In `cloneRank`, change the nil fast path to return the incoming `rank` value rather than a literal nil. This keeps untyped nil unchanged and preserves typed-nil interface identity. Update its comment to explain that the caller handles top-level omission before cloning, whereas nested typed nils must survive until validation; do not change the deep-copy branches for non-nil rank nodes.

    Refactor `TestCollectionSearchTypedNilRank` for EWM-02 and EWM-06 so each subtest has isolated request capture. Keep the direct typed-nil case expecting `{"searches":[{}]}` and the direct-typed-nil-plus-`Val(2)` case expecting `{"searches":[{},{"rank":{"$val":2}}]}`. Change the nested RRF case from the old successful `Val(0)` payload to `require.NotPanics`, a nil result, and `require.ErrorIs(searchErr, ErrNilRank)` through the real `Collection.Search` path. For successful cases, replace the blocking `<-requestBodies` read with a `select` that receives and runs `require.JSONEq`, or calls `t.Fatal` after `time.After(time.Second)`. For the error case, assert the isolated server received no body using a bounded select; import `time`. No test may wait indefinitely, and no nested case may assert the former substituted `$val:0` term.
  </action>
  <verify>
    <automated>go test -tags=basicv2 ./pkg/api/v2 -run '^(TestCollectionSearchTypedNilRank|TestCloneRankTypedNil|TestSearchRequestMarshalTypedNilRank)$' -count=1 -timeout=30s</automated>
  </verify>
  <done>cloneRank preserves nested typed-nil identity, Collection.Search returns ErrNilRank without panic or a corrupted HTTP payload, direct omission/mixed batches remain unchanged, and request-body assertions are bounded.</done>
</task>

<task type="auto">
  <name>Task 3: Correct the earlier GSD receiver claim and mark the RRF contract superseded</name>
  <files>.planning/quick/260731-e0k-address-typed-nil-rank-arithmetic-and-re/260731-e0k-PLAN.md, .planning/quick/260731-e0k-address-typed-nil-rank-arithmetic-and-re/260731-e0k-SUMMARY.md</files>
  <action>
    For EWM-05, correct the prior plan and summary without rewriting commit history. In the old PLAN Task 1 description, objective, and success criteria, add the exact clarification “The regression uses a valid `*ValRank` arithmetic receiver and a typed-nil `*KnnRank` operand; the panic occurs later when the composite serializer invokes the operand.” Remove wording that says or implies the test invoked arithmetic on a nil receiver.

    Replace the old SUMMARY RED sentence with: “Added a six-operation regression using a valid `*ValRank` arithmetic receiver and a typed-nil `*KnnRank` operand; composite marshaling then panicked when it invoked the operand.” Preserve commit `b3ef969`, its implemented guard, verification results, and all unrelated historical details.

    The prior PLAN/SUMMARY also record the nested RRF `Val(0)` substitution as intentional. Preserve that as history but annotate the relevant scope/decision text as superseded by quick task `260731-ewm`, which identifies the substitution as a defect and changes nested typed-nil RRF input to an error. Do not make the old artifacts claim that the new production fix was part of `b3ef969`.
  </action>
  <verify>
    <automated>rg -n -F 'valid `*ValRank` arithmetic receiver and a typed-nil `*KnnRank` operand' .planning/quick/260731-e0k-address-typed-nil-rank-arithmetic-and-re/260731-e0k-PLAN.md .planning/quick/260731-e0k-address-typed-nil-rank-arithmetic-and-re/260731-e0k-SUMMARY.md &amp;&amp; rg -n 'superseded.*260731-ewm|260731-ewm.*supersed' .planning/quick/260731-e0k-address-typed-nil-rank-arithmetic-and-re/260731-e0k-PLAN.md .planning/quick/260731-e0k-address-typed-nil-rank-arithmetic-and-re/260731-e0k-SUMMARY.md</automated>
  </verify>
  <done>Both historical artifacts accurately describe the old test's receiver and operand, clearly mark the nested-RRF substitution contract as superseded, and retain the original commit and verification record.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Caller-provided Rank interface -> composite serializer | An interface can contain a nil concrete pointer that bypasses ordinary `rank == nil` checks |
| User rank tree -> cloned Collection.Search request | Clone normalization must not convert invalid nested input into a valid expression with different ranking semantics |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-ewm-01 | Denial of Service | Nine composite `MarshalJSON` methods | mitigate | Route every child through `marshalRank`, detect typed nil before method dispatch, and cover every composite shape plus all KnnRank receiver compositions under no-panic tests |
| T-ewm-02 | Tampering | `cloneRank` -> `RrfRank.MarshalJSON` | mitigate | Preserve nested typed-nil identity and reject it with ErrNilRank so it cannot be changed into a best-ranked Val(0) term |
| T-ewm-03 | Repudiation | Prior PLAN/SUMMARY account | mitigate | Correct receiver/operand wording and mark the former nested-RRF contract superseded without rewriting the original commit record |
| T-ewm-SC | Tampering | Package installation | accept | This plan adds no dependency and runs no package-manager install |
</threat_model>

<verification>
1. `go test -tags=basicv2 ./pkg/api/v2 -run '^(TestUnknownRankError|TestRankArithmeticTypedNilOperandMarshal|TestRankTypedNilReceiverMarshal|TestCompositeRankNilChildMarshal|TestOperandConversion|TestRrfRank)$' -count=1 -timeout=30s`
2. `go test -tags=basicv2 ./pkg/api/v2 -run '^(TestCollectionSearchTypedNilRank|TestCloneRankTypedNil|TestSearchRequestMarshalTypedNilRank)$' -count=1 -timeout=30s`
3. `go test -tags=basicv2 ./pkg/api/v2/... -count=1 -timeout=120s`
4. `make lint`
5. `git diff --check`
6. Inspect `git diff -- pkg/api/v2 .planning/quick/260731-e0k-address-typed-nil-rank-arithmetic-and-re` and confirm changes are limited to nil handling, tests/docs, bounded synchronization, and factual record corrections.
</verification>

<source_coverage_audit>
| Source | ID | Feature/Requirement | Plan | Status | Notes |
|--------|----|---------------------|------|--------|-------|
| GOAL | — | Fix typed-nil rank panics, nested RRF laundering, errors/docs/tests, and inaccurate GSD records | 01 | COVERED | Tasks 1-3 |
| REQ | EWM-01 | Typed-nil receivers fail through composite marshal instead of panicking | 01 | COVERED | Task 1 |
| REQ | EWM-02 | Nested RRF typed nil is not laundered into Val(0) | 01 | COVERED | Task 2, with RRF validation from Task 1 |
| REQ | EWM-03 | Nil-rank and unknown-operand errors are accurate and distinct | 01 | COVERED | Task 1 |
| REQ | EWM-04 | Exported Rank and SearchRequest nil behavior is documented across entry paths | 01 | COVERED | Task 1 |
| REQ | EWM-05 | Earlier PLAN/SUMMARY receiver wording is corrected | 01 | COVERED | Task 3 |
| REQ | EWM-06 | Blocking request-body receive is bounded | 01 | COVERED | Task 2 |
| REQ | EWM-07 | Consider nil-helper relocation without unnecessary churn | 01 | COVERED | Helpers remain in search.go by explicit cohesion/churn decision |
| REQ | EWM-08 | Add receiver/composite/RRF/error/no-panic regressions and run focused/package tests | 01 | COVERED | Tasks 1-2 and overall verification |
| RESEARCH | — | No research phase | — | EXCLUDED | Quick mode explicitly forbids a research phase; existing package patterns are sufficient |
| CONTEXT | PROJECT-01 | Prefer the smallest complete code change | 01 | COVERED | One child-marshal helper and one clone fast-path correction; no method-matrix guards or helper move |
| CONTEXT | PROJECT-02 | Keep repository artifacts free of prohibited internal details | 01 | COVERED | Plan and planned artifacts contain no such details |
| CONTEXT | PROJECT-03 | Use squash merge | — | EXCLUDED | No merge operation is part of this planning or execution scope |
</source_coverage_audit>

<success_criteria>
- Typed-nil KnnRank receiver compositions and all nine composite child shapes return ErrNilRank without panicking.
- Typed-nil input no longer uses the misleading UnknownRank error; genuinely unsupported Operand implementations still do.
- Nested RRF typed nils retain their dynamic type through cloning and fail consistently in direct marshal/validation and Collection.Search.
- Direct optional SearchRequest.Rank typed nil remains omitted, WithRank still rejects it, and exported docs explain the difference.
- HTTP regressions cannot hang on request-body capture.
- The earlier PLAN/SUMMARY accurately distinguish the valid ValRank receiver from the typed-nil KnnRank operand and flag the old nested-RRF contract as superseded.
- Focused tests, the full basicv2 package suite, lint, and diff checks pass.
</success_criteria>

<output>
Create `.planning/quick/260731-ewm-fix-typed-nil-rank-panics-nested-rrf-nil/260731-ewm-SUMMARY.md` when done.
</output>
