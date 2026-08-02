---
phase: quick-260801-qmr
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - pkg/api/v2/where.go
  - pkg/api/v2/where_test.go
autonomous: true
requirements:
  - GH-533

must_haves:
  truths:
    - "A caller can validate a nested $and/$or Where expression within MaxExpressionDepth without changing its current valid result."
    - "A Where expression one recursive child beyond MaxExpressionDepth returns a deterministic validation error instead of continuing unbounded recursion."
    - "JSON marshalling a too-deep Where expression returns that validation error without panicking or encoding a partial expression."
    - "Existing operator, empty-operand, and typed-nil validation behavior remains unchanged."
  artifacts:
    - path: "pkg/api/v2/where.go"
      provides: "Depth-aware recursive validation for built-in compound Where clauses"
      contains: "MaxExpressionDepth"
    - path: "pkg/api/v2/where_test.go"
      provides: "Boundary and JSON-path regressions for nested Where expressions"
      contains: "TestWhereClauseExpressionDepthGuard"
  key_links:
    - from: "pkg/api/v2/where.go:WhereClauseWhereClauses.Validate"
      to: "pkg/api/v2/rank.go:MaxExpressionDepth"
      via: "shared package-level expression-depth limit"
      pattern: "MaxExpressionDepth"
    - from: "pkg/api/v2/where.go:WhereClauseWhereClauses.MarshalJSON"
      to: "pkg/api/v2/where.go:WhereClauseWhereClauses.Validate"
      via: "validation before json.Marshal"
      pattern: "w\\.Validate\\(\\)"
---

<objective>
Close issue 533 by putting a bounded recursive validation path around nested `$and` and `$or` metadata clauses.

Purpose: adversarial or accidental nesting must fail predictably before it can exhaust the Go call stack during validation or JSON marshalling, using the SDK's established expression-depth policy.
Output: a depth-aware `WhereClauseWhereClauses` validator and focused boundary regressions.
</objective>

<execution_context>
@/Users/tazarov/.codex/get-shit-done/workflows/execute-plan.md
@/Users/tazarov/.codex/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/PROJECT.md
@pkg/api/v2/where.go
@pkg/api/v2/where_test.go
@pkg/api/v2/rank.go
@pkg/api/v2/rank_test.go

**Working directory:** `/Users/tazarov/GolandProjects/chroma-go`

**Scope and compatibility constraints:**
- Reuse the existing package-level `MaxExpressionDepth = 100`; do not add a Where-specific limit or dependency.
- Match rank's zero-based convention: validate the root compound at depth `0`, permit at most `MaxExpressionDepth` recursive child calls below it, and reject only the next level with `where expression exceeds maximum depth of %d`.
- Keep the public `WhereClause` interface, `And`/`Or` constructors, existing JSON shape, and lazy validation behavior for non-compound clauses unchanged.
- Preserve the current validation order for each compound node: invalid operator, empty operand, and nil/typed-nil child errors must retain their existing behavior before a child is delegated for recursive validation.
- Keep the change confined to `pkg/api/v2/where.go` and its colocated build-tagged test file. Preserve unrelated working-tree edits; do not add packages, services, or configuration. Execution creates one atomic local commit for this task and performs no push, pull-request, or merge operation; any later merge must use squash merge.
</context>

<interfaces>
<!-- Existing contracts the executor should use directly. -->

From `pkg/api/v2/where.go`:

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

type WhereClauseWhereClauses struct {
	WhereClauseBase
	operand []WhereClause
}

func (w *WhereClauseWhereClauses) Validate() error
func (w *WhereClauseWhereClauses) MarshalJSON() ([]byte, error)
func And(clauses ...WhereClause) WhereClause
func Or(clauses ...WhereClause) WhereClause
```

`WhereClauseWhereClauses.MarshalJSON` already calls `Validate` before encoding, so the depth guard belongs in the validation path and protects both direct validation and JSON marshalling.

From `pkg/api/v2/rank.go`:

```go
const MaxExpressionDepth = 100
```

The rank implementation uses a zero-based depth counter and errors when `depth > MaxExpressionDepth`; retain that boundary convention without changing rank behavior.
</interfaces>

<source_audit>

| Source | ID | Required item | Plan | Status | Notes |
| --- | --- | --- | --- | --- | --- |
| GOAL | — | Bound recursive `$and`/`$or` Where validation to prevent stack exhaustion | 01 | COVERED | Implemented and exercised through `Validate` and JSON marshalling. |
| REQ | GH-533 | Reuse `MaxExpressionDepth` and add focused tests | 01 | COVERED | Uses the existing package constant and boundary regressions. |
| RESEARCH | — | Follow the rank depth-guard convention and retain existing API behavior | 01 | COVERED | Root-zero accounting, same limit, no new public API. |
| CONTEXT | — | No CONTEXT.md or locked decisions were supplied for this quick task | 01 | N/A | No deferred ideas or decisions to implement. |

</source_audit>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Guard recursive compound Where validation at the shared expression limit</name>
  <files>pkg/api/v2/where.go, pkg/api/v2/where_test.go</files>
  <behavior>
    - A generated alternating `And`/`Or` expression with exactly `MaxExpressionDepth` recursive child calls below its root validates and marshals successfully.
    - The next recursive child level fails both `WhereClause.Validate` and `json.Marshal` without a panic, with `where expression exceeds maximum depth of 100` in the returned error.
    - A shallow valid compound expression retains its existing JSON output.
    - Existing invalid operator, empty compound operand, nil child, and typed-nil child results keep their established errors.
  </behavior>
  <action>
    Start with a failing `TestWhereClauseExpressionDepthGuard` in `pkg/api/v2/where_test.go`, following the file's `basicv2 && !cloud` build tag, `testify/require` assertions, and `encoding/json` usage. Add a local test builder that starts from a valid leaf such as `EqString(K("depth"), "leaf")` and wraps it with alternating `And` and `Or` compound clauses. Exercise the zero-based boundary precisely: a tree with `MaxExpressionDepth` recursive child calls below the root succeeds, while the next child level fails. For the rejection case, separately call `Validate` and `json.Marshal`; use `require.NotPanics`, require an error, and assert it contains `fmt.Sprintf("where expression exceeds maximum depth of %d", MaxExpressionDepth)`. Include a shallow compound JSON assertion so the test names the compatibility baseline.

    In `pkg/api/v2/where.go`, retain `WhereClauseWhereClauses.Validate()` as the public entry point but have it invoke a private depth-aware compound validator at depth `0`. Add a small unexported child-validation helper that receives the current depth. It must preserve the existing parent checks for a nil or typed-nil `WhereClause` before delegating, route an exact `*WhereClauseWhereClauses` child to the private depth-aware validator with `depth + 1` instead of calling its public `Validate` method, and call `Validate` normally for all other `WhereClause` implementations. At each recursive child boundary, reject `depth > MaxExpressionDepth` using `errors.Errorf("where expression exceeds maximum depth of %d", MaxExpressionDepth)`. This counts every recursive validation call, including the final leaf, exactly as the rank guard counts recursive marshalling calls.

    Do not introduce a new exported helper, alter the `WhereClause` interface, change `And`/`Or`, weaken the existing typed-nil protection in `MarshalJSON`, or change rank code. `MarshalJSON` must continue to call the public `Validate` once before it constructs the existing `$and`/`$or` map, so a too-deep tree is rejected before `encoding/json` traverses it. This task is one atomic implementation commit.
  </action>
  <verify>
    <automated>go test -tags=basicv2 ./pkg/api/v2 -run '^TestWhereClauseExpressionDepthGuard$' -count=1 -timeout=60s</automated>
  </verify>
  <done>`WhereClauseWhereClauses.Validate` has a root-zero, shared-limit recursion guard; nesting one child past `MaxExpressionDepth` returns the stable Where-depth error through both validation and JSON marshalling without panic; and the focused tagged regression passes.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
| --- | --- |
| Caller-controlled `WhereClause` tree -> recursive validation | A public `$and`/`$or` expression can contain arbitrarily deep nested compound clauses before it reaches the SDK's validator or JSON encoder. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
| --- | --- | --- | --- | --- |
| T-qmr-533-01 | Denial of Service | `WhereClauseWhereClauses.Validate` | mitigate | Track recursion with the existing `MaxExpressionDepth` and reject the first over-limit child before recursive validation or JSON encoding continues. |
| T-qmr-533-02 | Tampering | `WhereClauseWhereClauses.MarshalJSON` | mitigate | Retain validate-before-encode wiring and prove a too-deep caller tree returns an error rather than producing a partial JSON expression. |
| T-qmr-533-SC | Tampering | Package installation | accept | This plan performs no package-manager operation or dependency change. |
</threat_model>

<verification>

Run the focused regression, then the tagged package suite and whitespace check:

```text
go test -tags=basicv2 ./pkg/api/v2 -run '^TestWhereClauseExpressionDepthGuard$' -count=1 -timeout=60s
go test -tags=basicv2 ./pkg/api/v2 -count=1 -timeout=60s
git diff --check
```
</verification>

<success_criteria>

- One package-level expression limit bounds all recursive built-in `$and`/`$or` child validation; no separate limit or public API is introduced.
- The deepest permitted expression validates and has the existing JSON form, while one additional child returns the stable depth error from both public paths without panic.
- The complete `basicv2` V2 package test suite passes with unchanged existing validation behavior.
</success_criteria>

<output>
Create `.planning/quick/260801-qmr-validate-and-address-issue-533/260801-qmr-SUMMARY.md` when implementation is complete.
</output>
