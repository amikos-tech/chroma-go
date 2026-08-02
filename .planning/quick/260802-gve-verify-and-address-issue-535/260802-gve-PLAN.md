---
phase: quick-260802-gve
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - pkg/api/v2/where.go
  - pkg/api/v2/where_test.go
autonomous: true
requirements:
  - GH-535

must_haves:
  truths:
    - "Compound Where validation and JSON marshalling use one implementation of their shared operator and operand checks."
    - "Compound Where validation and JSON marshalling retain their current behavior, including their single-pass depth-aware traversal."
  artifacts:
    - path: "pkg/api/v2/where.go"
      provides: "Shared node-local validation used by the compound Where validation and JSON-marshalling helpers"
      contains: "validateOperatorAndOperand"
    - path: "pkg/api/v2/where_test.go"
      provides: "Regression coverage for shared compound Where validation"
      contains: "TestCompoundWhereClauseSharedValidation"
  key_links:
    - from: "pkg/api/v2/where.go:WhereClauseWhereClauses.validateWithDepth"
      to: "pkg/api/v2/where.go:WhereClauseWhereClauses.validateOperatorAndOperand"
      via: "direct method call"
      pattern: "validateOperatorAndOperand"
    - from: "pkg/api/v2/where.go:WhereClauseWhereClauses.marshalJSONWithDepth"
      to: "pkg/api/v2/where.go:WhereClauseWhereClauses.validateOperatorAndOperand"
      via: "direct method call"
      pattern: "validateOperatorAndOperand"
---

<objective>
Close issue 535 by removing the duplicated node-local checks from the compound-Where traversal helpers.

Purpose: apply DRY to the operator and operand invariants while preserving the separate single-pass validation and JSON-encoding traversals.
Output: one private shared validation helper in `pkg/api/v2/where.go` and focused regression coverage in `pkg/api/v2/where_test.go`.
</objective>

<execution_context>
@/Users/tazarov/.codex/get-shit-done/workflows/execute-plan.md
@/Users/tazarov/.codex/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@pkg/api/v2/where.go
@pkg/api/v2/where_test.go

**Working directory:** `/Users/tazarov/GolandProjects/chroma-go`

**Scope and compatibility constraints:**
- Work on the existing `fix/issue-535-where-clause-cross-reference` branch; do not switch branches, discard, or edit unrelated working-tree changes.
- This is a behavior-preserving maintainability refactor: centralize only the node-local operator and empty-operand checks; do not change error text, public interfaces, dependencies, or generated files.
- Keep `validateWithDepth` and `marshalJSONWithDepth` as separate traversals. Calling full validation before marshalling would add a second tree walk, while a generic visitor would add more abstraction than the two shared checks justify.
- Do not create a commit, push, pull request, or merge from this quick plan. Any later merge must use squash merge.
</context>

<interfaces>
<!-- Existing paired private helpers. Their behavior is intentionally coupled, while both remain separate single-pass traversals. -->

From `pkg/api/v2/where.go`:

```go
func (w *WhereClauseWhereClauses) validateWithDepth(depth int) error
func (w *WhereClauseWhereClauses) marshalJSONWithDepth(depth int) ([]byte, error)
func (w *WhereClauseWhereClauses) validateOperatorAndOperand() error
```

`validateOperatorAndOperand` owns the invalid-operator and empty-operand checks. `validateWithDepth` and `marshalJSONWithDepth` both call it before their distinct depth-aware child traversals.
</interfaces>

<source_audit>

| Source | ID | Required item | Plan | Status | Notes |
| --- | --- | --- | --- | --- | --- |
| GOAL | — | Prevent drift between compound-Where validation and marshalling checks | 01 | COVERED | Both paths call one private implementation of their shared invariants. |
| REQ | GH-535 | Address the duplicated operator and operand checks without changing behavior | 01 | COVERED | A private helper removes the duplication; focused regression covers both callers. |
| RESEARCH | — | No RESEARCH.md supplied; existing source establishes the single-pass performance requirement | 01 | COVERED | Share the node-local guard while retaining each traversal. |
| CONTEXT | — | No CONTEXT.md or locked decisions supplied for this quick task | 01 | N/A | No deferred ideas or decisions to implement. |

</source_audit>

<tasks>

<task type="auto">
  <name>Task 1: Share compound Where node validation without merging traversals</name>
  <files>pkg/api/v2/where.go, pkg/api/v2/where_test.go</files>
  <action>Extract the duplicated operator and empty-operand checks into a private `validateOperatorAndOperand` method. Call it from both `validateWithDepth` and `marshalJSONWithDepth`, preserving their error text, ordering, depth handling, JSON shape, and separate single-pass child traversals. Add a focused table test that exercises invalid operators and empty operands through both `Validate` and `MarshalJSON`.</action>
  <verify>
    <automated>go test -tags=basicv2 ./pkg/api/v2 -run '^(TestCompoundWhereClauseSharedValidation|TestWhereClauseExpressionDepthGuard|TestWhereClauseEmptyOperandValidation)$' -count=1 -timeout=60s</automated>
  </verify>
  <done>Both traversal helpers call one node-local validator; focused tests prove both public paths retain the same errors and depth behavior.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
| --- | --- |
| Future maintainer -> compound Where traversal helpers | A future change to one of the paired private helpers could accidentally make validation and JSON marshalling accept different operator or operand states. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
| --- | --- | --- | --- | --- |
| T-gve-535-01 | Tampering | `WhereClauseWhereClauses.validateWithDepth` and `marshalJSONWithDepth` | mitigate | Route both callers through one private implementation of their shared invariants. |
| T-gve-535-02 | Denial of Service | Compound Where JSON traversal | mitigate | Retain the existing single-pass depth-aware marshalling traversal and verify its focused depth regression. |
| T-gve-535-SC | Tampering | Package installation | accept | This plan performs no package-manager operation or dependency change. |
</threat_model>

<verification>

Run the existing focused compound-Where depth regression and inspect the diff:

```text
go test -tags=basicv2 ./pkg/api/v2 -run '^(TestCompoundWhereClauseSharedValidation|TestWhereClauseExpressionDepthGuard|TestWhereClauseEmptyOperandValidation)$' -count=1 -timeout=60s
go test -tags=basicv2 ./pkg/api/v2 -count=1 -timeout=5m
git diff --check
git diff -- pkg/api/v2/where.go pkg/api/v2/where_test.go
```
</verification>

<success_criteria>

- Both private compound-Where helpers use the same operator and operand validation implementation.
- Validation and marshalling retain their separate single-pass traversals and existing errors.
- The new shared-validation regression and existing focused depth regression pass.
</success_criteria>

<output>
Create `.planning/quick/260802-gve-verify-and-address-issue-535/260802-gve-SUMMARY.md` when implementation is complete.
</output>
