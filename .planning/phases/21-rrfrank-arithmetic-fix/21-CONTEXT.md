# Phase 21: RrfRank Arithmetic Fix - Context

**Gathered:** 2026-04-08
**Status:** Ready for planning

<domain>
## Phase Boundary

Fix all 10 arithmetic/math methods on `RrfRank` so they build expression trees instead of silently returning the receiver. This brings the Go SDK into parity with the Python SDK where RRF inherits composable arithmetic from the base `Rank` class.

</domain>

<decisions>
## Implementation Decisions

### Fix Scope
- **D-01:** All 10 methods are fixed: Multiply, Sub, Add, Div, Negate, Abs, Exp, Log, Max, Min
- **D-02:** Each method follows the exact pattern already established by `KnnRank` and `ValRank` — wrap receiver + operand in the appropriate expression node type

### Pattern Reference
- **D-03:** `Multiply` → `&MulRank{ranks: []Rank{r, operandToRank(operand)}}`
- **D-04:** `Sub` → `&SubRank{left: r, right: operandToRank(operand)}`
- **D-05:** `Add` → `&SumRank{ranks: []Rank{r, operandToRank(operand)}}`
- **D-06:** `Div` → `&DivRank{left: r, right: operandToRank(operand)}`
- **D-07:** `Negate` → `&MulRank{ranks: []Rank{Val(-1), r}}`
- **D-08:** `Abs` → `&AbsRank{rank: r}`
- **D-09:** `Exp` → `&ExpRank{rank: r}`
- **D-10:** `Log` → `&LogRank{rank: r}`
- **D-11:** `Max` → `&MaxRank{ranks: []Rank{r, operandToRank(operand)}}`
- **D-12:** `Min` → `&MinRank{ranks: []Rank{r, operandToRank(operand)}}`

### Claude's Discretion
- Test structure and naming for RrfRank arithmetic tests

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Bug Report
- GitHub issue #481 — RrfRank arithmetic methods silently return self

### Implementation Pattern
- `pkg/api/v2/rank.go` lines 931-960 — KnnRank arithmetic methods (reference pattern)
- `pkg/api/v2/rank.go` lines 110-148 — ValRank arithmetic methods (reference pattern)
- `pkg/api/v2/rank.go` lines 1130-1168 — RrfRank methods to fix

### Existing Tests
- `pkg/api/v2/rank_test.go` — TestArithmeticOperations, TestMathFunctions, TestRrfRank (extend for RrfRank arithmetic)

### Cross-SDK Parity
- Python SDK: RRF inherits arithmetic from base Rank class (Sum, Sub, Mul, Div expression builders)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- All expression node types already exist: `MulRank`, `SubRank`, `SumRank`, `DivRank`, `AbsRank`, `ExpRank`, `LogRank`, `MaxRank`, `MinRank`
- `operandToRank()` helper converts `Operand` to `Rank` (handles IntOperand, FloatOperand, Rank)
- `UnknownRank` sentinel for invalid operand types

### Established Patterns
- Every `Rank` implementor follows identical method signatures and wrapping pattern
- `KnnRank` is the canonical reference for a complex rank type with full arithmetic support
- Tests use `MarshalJSON` + `require.JSONEq` to verify expression tree structure

### Integration Points
- `RrfRank.MarshalJSON` already produces `$mul`, `$sum`, `$div` nodes internally — wrapping in further arithmetic adds the same node types the server already handles
- Build tag: `basicv2` for rank tests

</code_context>

<specifics>
## Specific Ideas

- Python SDK was researched to confirm RRF arithmetic is supported cross-SDK (not a Go-only concern)
- The `// no-op` comment at line 1129 confirms this was a known stub, not intentional behavior

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 21-rrfrank-arithmetic-fix*
*Context gathered: 2026-04-08*
