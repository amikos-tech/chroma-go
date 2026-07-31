---
phase: quick-260731-fvo
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - pkg/api/v2/rank.go
  - pkg/api/v2/rank_test.go
  - .planning/quick/260731-ewm-fix-typed-nil-rank-panics-nested-rrf-nil/260731-ewm-SUMMARY.md
autonomous: true
requirements:
  - FVO-01
  - FVO-02
  - FVO-03
  - FVO-04
  - FVO-05
  - FVO-06

must_haves:
  truths:
    - "Calling SumRank.Add, MulRank.Multiply, MaxRank.Max, or MinRank.Min on its typed-nil receiver does not panic while building the expression"
    - "Expressions built from those typed-nil receivers preserve the invalid receiver and return an error matching ErrNilRank when marshaled"
    - "SumRank, MulRank, MaxRank, and MinRank reject typed-nil children at the first position and within a three-element slice, not only at index 1 of a two-element slice"
    - "A directly constructed invalid RrfRank is revalidated by MarshalJSON and rejected without relying on NewRrfRank"
    - "An untyped nil Operand remains Val(0), and documentation distinguishes that compatibility behavior from typed-nil Rank rejection"
    - "The prior quick-task summary describes only the typed-nil *KnnRank receiver compositions it actually tested"
    - "UnknownRank documentation warns that its promoted arithmetic methods operate on the embedded zero-valued ValRank"
  artifacts:
    - path: "pkg/api/v2/rank.go"
      provides: "Nil-safe self-flattening builders and precise nil/UnknownRank documentation"
      contains: "func (s *SumRank) Add"
    - path: "pkg/api/v2/rank_test.go"
      provides: "Builder, child-position, direct-RRF-marshal, and compatibility regressions"
      contains: "TestSelfFlatteningRankTypedNilReceiverMarshal"
    - path: ".planning/quick/260731-ewm-fix-typed-nil-rank-panics-nested-rrf-nil/260731-ewm-SUMMARY.md"
      provides: "Accurate historical description of the earlier receiver coverage"
      contains: "ten *KnnRank typed-nil receiver compositions"
  key_links:
    - from: "pkg/api/v2/rank.go:self-flattening builders"
      to: "pkg/api/v2/rank.go:marshalRank"
      via: "a typed-nil receiver is retained as a child instead of dereferenced or discarded"
      pattern: "marshalRank\\("
    - from: "pkg/api/v2/rank.go:RrfRank.MarshalJSON"
      to: "pkg/api/v2/rank.go:RrfRank.Validate"
      via: "directly constructed RRF state is validated before expression construction"
      pattern: "r\\.Validate\\(\\)"
---

<objective>
Close the remaining typed-nil rank gaps without changing valid rank expression output or the established untyped-nil Operand behavior.

Purpose: four self-flattening builders currently panic before the shared marshal guard can return ErrNilRank, while adjacent tests and documentation overstate or under-specify the protected cases.

Output: nil-safe Sum/Mul/Max/Min self-flattening, broader child-position regressions, a direct invalid-RRF marshal regression, precise compatibility and UnknownRank notes, and corrected historical summary wording.
</objective>

<execution_context>
@/Users/tazarov/.codex/get-shit-done/workflows/execute-plan.md
@/Users/tazarov/.codex/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@pkg/api/v2/rank.go
@pkg/api/v2/rank_test.go
@.planning/quick/260731-ewm-fix-typed-nil-rank-panics-nested-rrf-nil/260731-ewm-SUMMARY.md

**Locked behavior:**
- Preserve `operandToRank(nil) -> Val(0)` for an untyped nil Operand; this is an existing compatibility contract and is already pinned by `TestOperandConversion`.
- Typed-nil Rank receivers or children must remain invalid and eventually return `ErrNilRank`; never make a nil receiver disappear from a successfully marshaled expression.
- Preserve current flattening and JSON output for non-nil SumRank, MulRank, MaxRank, and MinRank values.
- Keep UnknownRank as the unsupported-Operand leaf sentinel. Document its promoted-method caveat instead of adding arithmetic overrides that current construction paths do not need.
- Correct the earlier summary as a historical record; do not claim this follow-up implementation was part of quick task `260731-ewm`.
- Add no dependency and make no merge.
</context>

<interfaces>
From `pkg/api/v2/rank.go`:

```go
func (s *SumRank) Add(operand Operand) Rank
func (m *MulRank) Multiply(operand Operand) Rank
func (m *MaxRank) Max(operand Operand) Rank
func (m *MinRank) Min(operand Operand) Rank
func (r *RrfRank) Validate() error
func (r *RrfRank) MarshalJSON() ([]byte, error)
func marshalRank(rank Rank) ([]byte, error)
func operandToRank(operand Operand) Rank
```
</interfaces>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Make self-flattening builders nil-safe and close rank regressions</name>
  <files>pkg/api/v2/rank.go, pkg/api/v2/rank_test.go</files>
  <behavior>
    - "A typed-nil *SumRank.Add, *MulRank.Multiply, *MaxRank.Max, and *MinRank.Min call builds without panic; marshaling each result does not panic and returns ErrNilRank"
    - "For SumRank, MulRank, MaxRank, and MinRank, a typed-nil child at index 0 of a two-element slice, index 1 of the existing two-element slice, and the middle of a three-element slice returns ErrNilRank without panic"
    - "Calling MarshalJSON on an invalid &RrfRank value built without NewRrfRank invokes Validate and returns its indexed ErrNilRank failure"
    - "The existing untyped-nil Operand regression still serializes that operand as Val(0), while typed-nil Rank inputs remain rejected"
  </behavior>
  <action>
    Start with regressions in `pkg/api/v2/rank_test.go`. Add `TestSelfFlatteningRankTypedNilReceiverMarshal` with typed-nil `*SumRank`, `*MulRank`, `*MaxRank`, and `*MinRank` receivers. Exercise only their respective flattening methods (`Add`, `Multiply`, `Max`, and `Min`), wrap expression construction and marshaling in `require.NotPanics`, and require the marshal error to match `ErrNilRank`.

    Expand `TestCompositeRankNilChildMarshal` for each of SumRank, MulRank, MaxRank, and MinRank. Retain the current typed-nil child at index 1 of two elements, and add cases with the child at index 0 of two elements and in the middle of three elements. Every case must marshal under `require.NotPanics` and use `require.ErrorIs(err, ErrNilRank)`.

    Add a direct `RrfRank.MarshalJSON` subtest that constructs `&RrfRank{K: 60, Ranks: ...}` with a typed-nil `*KnnRank` child instead of calling `NewRrfRank`. Require an indexed error matching `ErrNilRank`; this proves `MarshalJSON` calls `Validate` for bypassed constructor state. Keep the existing `TestOperandConversion` case proving that an untyped nil Operand becomes `Val(0)`.

    Run the focused tests and confirm the new four-builder table fails because the builders panic before marshaling. Then update only the four self-flattening methods in `pkg/api/v2/rank.go`. Build `newRanks` without reading `receiver.ranks` until the receiver is known non-nil. For a nil receiver, seed `newRanks` with that typed-nil receiver so the existing `marshalRank` guard reports `ErrNilRank`; for a non-nil receiver, copy its current flattened ranks exactly as today. Preserve the existing same-type operand flattening and all valid JSON shapes.

    Correct the exported `Rank` comment so it says an untyped nil passed through the Operand API becomes `Val(0)` for compatibility, while nil or typed-nil Rank children fail with `ErrNilRank` during marshaling. Extend the `UnknownRank` comment with one concise warning: its promoted arithmetic methods operate on the embedded zero-valued `ValRank` and should not be called directly; normal conversion uses it only as a leaf whose `MarshalJSON` returns the unsupported-operand error. Do not add override methods or change `operandToRank`.
  </action>
  <verify>
    <automated>go test -tags=basicv2 ./pkg/api/v2 -run '^(TestSelfFlatteningRankTypedNilReceiverMarshal|TestCompositeRankNilChildMarshal|TestRrfRank|TestOperandConversion|TestUnknownRankError)$' -count=1 -timeout=30s</automated>
  </verify>
  <done>The four self-flattening calls cannot panic on typed-nil receivers, their results fail through ErrNilRank at marshal time, variadic child guards are exercised at the requested positions, direct invalid RRF state is revalidated, untyped nil still becomes Val(0), and comments state both compatibility boundaries plainly.</done>
</task>

<task type="auto">
  <name>Task 2: Correct the earlier receiver-coverage overclaim</name>
  <files>.planning/quick/260731-ewm-fix-typed-nil-rank-panics-nested-rrf-nil/260731-ewm-SUMMARY.md</files>
  <action>
    In the `pkg/api/v2/rank_test.go` bullet under “Files Created/Modified,” replace “Covers all typed-nil receiver compositions” with “Covers the ten *KnnRank typed-nil receiver compositions exercised by this task.” Retain the rest of the bullet's composite, RRF, and operand coverage description. This is a factual correction to what quick task `260731-ewm` tested at that time; do not rewrite its commits, verification results, or accomplishments to include the new SumRank, MulRank, MaxRank, and MinRank builder fix.
  </action>
  <verify>
    <automated>rg -n -F 'Covers the ten *KnnRank typed-nil receiver compositions exercised by this task' .planning/quick/260731-ewm-fix-typed-nil-rank-panics-nested-rrf-nil/260731-ewm-SUMMARY.md</automated>
  </verify>
  <done>The prior summary names the exact *KnnRank receiver coverage it had and no longer implies the four self-flattening builders were covered before this follow-up.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Caller-provided Rank interface -> fluent self-flattening builder | An interface can dispatch a method on a nil concrete pointer before serialization guards run |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-fvo-01 | Denial of Service | SumRank.Add, MulRank.Multiply, MaxRank.Max, MinRank.Min | mitigate | Avoid nil receiver dereference, preserve the typed-nil receiver for marshal validation, and cover build plus marshal under no-panic tests |
| T-fvo-02 | Tampering | Variadic composite child traversal | mitigate | Exercise typed nils at the first, existing second, and three-element middle positions for all four slice-backed composites |
| T-fvo-SC | Tampering | Package installation | accept | This plan adds no dependency and performs no package-manager install |
</threat_model>

<verification>
1. `go test -tags=basicv2 ./pkg/api/v2 -run '^(TestSelfFlatteningRankTypedNilReceiverMarshal|TestCompositeRankNilChildMarshal|TestRrfRank|TestOperandConversion|TestUnknownRankError)$' -count=1 -timeout=30s`
2. `go test -tags=basicv2 ./pkg/api/v2/... -count=1 -timeout=120s`
3. `make lint`
4. `git diff --check`
5. Inspect `git diff -- pkg/api/v2/rank.go pkg/api/v2/rank_test.go .planning/quick/260731-ewm-fix-typed-nil-rank-panics-nested-rrf-nil/260731-ewm-SUMMARY.md` and confirm the diff contains only the four nil-safe builders, requested tests/comments, and historical wording correction.
</verification>

<source_coverage_audit>
| Source | ID | Feature/Requirement | Plan | Status | Notes |
|--------|----|---------------------|------|--------|-------|
| GOAL | — | Fix typed-nil rank builder panics and close related test/documentation gaps | 01 | COVERED | Tasks 1-2 |
| REQ | FVO-01 | Four self-flattening builders are safe on typed-nil receivers | 01 | COVERED | Task 1 |
| REQ | FVO-02 | Composite nil-child tests cover index 0 and a three-element slice | 01 | COVERED | Task 1 |
| REQ | FVO-03 | Direct invalid RrfRank marshaling proves Validate runs | 01 | COVERED | Task 1 |
| REQ | FVO-04 | Untyped nil remains Val(0) with precise documentation | 01 | COVERED | Task 1 |
| REQ | FVO-05 | Prior EWM summary overclaim is corrected historically | 01 | COVERED | Task 2 |
| REQ | FVO-06 | UnknownRank promoted arithmetic caveat is documented | 01 | COVERED | Task 1 |
| RESEARCH | — | No research phase | — | EXCLUDED | The task explicitly forbids research and follows established rank patterns |
| CONTEXT | — | Prefer the least code needed | 01 | COVERED | Four local nil branches and comments; no new helper, dependency, or UnknownRank method matrix |
| CONTEXT | — | Preserve repository artifact restrictions | 01 | COVERED | Planned artifacts contain no prohibited internal information |
| CONTEXT | — | Always use squash merge | — | EXCLUDED | No merge is part of this plan |
</source_coverage_audit>

<success_criteria>
- The four reported typed-nil builder calls build and marshal without panic, returning ErrNilRank.
- Variadic composite nil-child coverage includes the first position and a three-element slice for Sum/Mul/Max/Min.
- Directly marshaling an invalid RrfRank proves constructor bypass cannot skip validation.
- Untyped nil Operand compatibility remains pinned as Val(0), and the Rank comment no longer implies otherwise.
- UnknownRank's promoted arithmetic caveat is clear without adding unreachable implementation code.
- The prior summary precisely limits its historical receiver claim to the ten *KnnRank cases it exercised.
- Focused tests, the full basicv2 package suite, lint, and diff checks pass.
</success_criteria>

<output>
Create `.planning/quick/260731-fvo-fix-typed-nil-rank-builder-panics-and-cl/260731-fvo-SUMMARY.md` when done.
</output>
