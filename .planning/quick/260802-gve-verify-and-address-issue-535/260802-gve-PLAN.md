---
phase: quick-260802-gve
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - pkg/api/v2/where.go
autonomous: true
requirements:
  - GH-535

must_haves:
  truths:
    - "Future maintainers can see from either compound Where helper that its operator and operand checks deliberately mirror the other helper."
    - "Compound Where validation and JSON marshalling retain their current behavior, including their single-pass depth-aware traversal."
  artifacts:
    - path: "pkg/api/v2/where.go"
      provides: "Reciprocal maintenance comments on the paired compound Where validation and JSON-marshalling helpers"
      contains: "marshalJSONWithDepth"
  key_links:
    - from: "pkg/api/v2/where.go:WhereClauseWhereClauses.validateWithDepth"
      to: "pkg/api/v2/where.go:WhereClauseWhereClauses.marshalJSONWithDepth"
      via: "reciprocal documentation comments"
      pattern: "validateWithDepth|marshalJSONWithDepth"
---

<objective>
Close issue 535 with reciprocal maintenance comments on the two intentionally duplicated compound-Where traversal helpers.

Purpose: preserve the deliberate single-pass design by making the coupling visible to future maintainers, without changing executable code or API behavior.
Output: one documentation-only change in `pkg/api/v2/where.go`, backed by the existing focused regression.
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
- This is a no-behavior-change maintainability fix: modify only comments in `pkg/api/v2/where.go`; do not change conditionals, error text, control flow, interfaces, tests, dependencies, formatting outside the comment area, or generated files.
- The existing checks in `validateWithDepth` and `marshalJSONWithDepth` are intentionally duplicated so each path validates and processes its children in one depth-aware pass. The comments must preserve that rationale, not recommend extracting shared validation.
- Do not create a commit, push, pull request, or merge from this quick plan. Any later merge must use squash merge.
</context>

<interfaces>
<!-- Existing paired private helpers. Their behavior is intentionally coupled, while both remain separate single-pass traversals. -->

From `pkg/api/v2/where.go`:

```go
func (w *WhereClauseWhereClauses) validateWithDepth(depth int) error
func (w *WhereClauseWhereClauses) marshalJSONWithDepth(depth int) ([]byte, error)
```

`validateWithDepth` rejects invalid `$and`/`$or` operators and empty operands before depth-aware child validation. `marshalJSONWithDepth` repeats those checks while validating and encoding every child in one traversal, avoiding repeated nested-subtree validation.
</interfaces>

<source_audit>

| Source | ID | Required item | Plan | Status | Notes |
| --- | --- | --- | --- | --- | --- |
| GOAL | — | Prevent future drift between intentionally duplicated compound-Where validation and marshalling checks | 01 | COVERED | Reciprocal comments make the maintenance dependency explicit at both functions. |
| REQ | GH-535 | Add a short cross-reference comment to `validateWithDepth` and `marshalJSONWithDepth` with no behavior change | 01 | COVERED | One documentation-only edit; focused regression proves the current path remains intact. |
| RESEARCH | — | No RESEARCH.md supplied; existing source establishes the deliberate single-pass design | 01 | COVERED | Retain separate validations and the current traversal shape. |
| CONTEXT | — | No CONTEXT.md or locked decisions supplied for this quick task | 01 | N/A | No deferred ideas or decisions to implement. |

</source_audit>

<tasks>

<task type="auto">
  <name>Task 1: Add reciprocal single-pass maintenance comments to compound Where helpers</name>
  <files>pkg/api/v2/where.go</files>
  <action>Add a short Go-style documentation comment immediately above `WhereClauseWhereClauses.validateWithDepth` that explicitly points maintainers to `marshalJSONWithDepth` and says their operator/operand checks intentionally remain synchronized for separate single-pass traversal paths. Amend the existing `marshalJSONWithDepth` documentation comment so it explicitly points back to `validateWithDepth` and states that its repeated operator/operand checks are deliberate and must stay synchronized with that validator. Keep both comments concise and accurate to the current implementation. Do not alter either function body, nearby helpers, imports, whitespace outside the comment blocks, or test files; this task must be a documentation-only diff.</action>
  <verify>
    <automated>go test -tags=basicv2 ./pkg/api/v2 -run '^TestWhereClauseExpressionDepthGuard$' -count=1 -timeout=60s</automated>
  </verify>
  <done>`pkg/api/v2/where.go` has concise reciprocal comments immediately associated with both private helpers; each names the other helper and preserves the intentional single-pass, synchronized-check rationale, while the file's executable code is unchanged and the existing depth regression passes.</done>
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
| T-gve-535-01 | Tampering | `WhereClauseWhereClauses.validateWithDepth` and `marshalJSONWithDepth` | mitigate | Add reciprocal comments that name the paired helper and require their deliberately duplicated operator/operand checks to stay synchronized. |
| T-gve-535-02 | Denial of Service | Compound Where JSON traversal | accept | No executable behavior changes are authorized; retain the existing single-pass depth-aware implementation and verify its focused depth regression. |
| T-gve-535-SC | Tampering | Package installation | accept | This plan performs no package-manager operation or dependency change. |
</threat_model>

<verification>

Run the existing focused compound-Where depth regression and inspect the diff:

```text
go test -tags=basicv2 ./pkg/api/v2 -run '^TestWhereClauseExpressionDepthGuard$' -count=1 -timeout=60s
git diff --check
git diff -- pkg/api/v2/where.go
```
</verification>

<success_criteria>

- Both private compound-Where helpers have concise reciprocal comments that explain why their validation checks remain deliberately duplicated.
- The diff for `pkg/api/v2/where.go` contains comment-only changes.
- The existing focused depth regression passes unchanged.
</success_criteria>

<output>
Create `.planning/quick/260802-gve-verify-and-address-issue-535/260802-gve-SUMMARY.md` when implementation is complete.
</output>
