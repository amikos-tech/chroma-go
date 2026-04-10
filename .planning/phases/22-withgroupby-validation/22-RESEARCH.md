# Phase 22: WithGroupBy Validation - Research

**Researched:** 2026-04-09
**Domain:** Go V2 Search API option validation and request construction
**Confidence:** HIGH (phase is localized to one option path and colocated tests)

## Summary

Phase 22 is a narrow contract fix in `pkg/api/v2/search.go`. Today, `groupByOption.ApplyToSearchRequest` returns `nil` when `o.groupBy == nil`, which silently turns an explicit `WithGroupBy(nil)` call into "omit grouping." That behavior violates `GRP-01` and the phase goal.

The implementation change should stay small:
- change the nil branch in `groupByOption.ApplyToSearchRequest` from silent success to a deterministic validation error
- keep non-nil validation delegated to `(*GroupBy).Validate()`
- update `pkg/api/v2/groupby_test.go` so the nil path asserts the exact stable error string and valid non-nil behavior still passes
- add one higher-level regression that proves `NewSearchRequest(..., WithGroupBy(nil))` fails before the search request is appended or sent

`pkg/api/v2/groupby.go` already enforces non-nil aggregate and required keys for valid `GroupBy` instances and does not need semantic changes for this phase.

**Primary recommendation:** Use a direct nil-validation message consistent with existing repo style, specifically `groupBy cannot be nil`, and keep the entire fix within `pkg/api/v2/search.go` and `pkg/api/v2/groupby_test.go`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** `WithGroupBy(nil)` is invalid and must fail fast during request construction/application; it is no longer treated as a silent no-op.
- **D-02:** Callers who want "no grouping" must omit the `WithGroupBy(...)` option entirely rather than passing `nil`.
- **D-03:** The nil failure must use a stable, nil-specific validation message so tests can assert exact text.
- **D-04:** The error should be returned from the option application path before any search request can be marshaled or sent.
- **D-05:** Phase 22 stays code-and-tests only; no docs/examples update is required.
- **D-06:** Existing valid non-nil `GroupBy` flows must remain unchanged.

### Claude's Discretion
- Exact nil-specific message text, as long as it is stable and matches repo style
- Whether to keep the nil error as a direct `errors.New(...)` string or promote it to a package-level variable
- Exact unit-test structure inside `groupby_test.go`

### Deferred Ideas (OUT OF SCOPE)
- Broader audit of other option setters that may still treat explicit `nil` as a no-op
- Any redesign of `GroupBy` JSON shape or aggregate validation
- Docs/examples updates
</user_constraints>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| GRP-01 | `WithGroupBy(nil)` returns a validation error instead of silently skipping grouping | Direct nil guard in `groupByOption.ApplyToSearchRequest`, exact-error regression test, and request-construction regression test |

## Project Constraints

| Constraint | Source | Impact on This Phase |
|-----------|--------|----------------------|
| New work targets V2 API | `CLAUDE.md` | All changes stay in `pkg/api/v2/` |
| Library code should fail with errors, not panic | `CLAUDE.md` | Use early validation in the option path; no runtime panics |
| Validation messages are direct and user-readable | `.planning/codebase/CONVENTIONS.md` | Prefer a simple stable string such as `groupBy cannot be nil` |
| Colocated tests with existing build tags | `.planning/codebase/TESTING.md`, `CLAUDE.md` | Keep changes in `pkg/api/v2/groupby_test.go` under `//go:build basicv2 && !cloud` |
| `require` assertions are standard | `.planning/codebase/TESTING.md` | Use `require.EqualError`, `require.NoError`, `require.Nil`, `require.Len` |

## Standard Stack

No new dependencies are needed.

| Package | Purpose | Status |
|---------|---------|--------|
| `github.com/pkg/errors` | direct validation error in `search.go` | already imported |
| `github.com/stretchr/testify/require` | exact error and regression assertions | already used in `groupby_test.go` |
| Go stdlib `encoding/json` | existing request serialization checks | already used in `groupby_test.go` |

## Architecture Patterns

### Recommended File Layout

```
pkg/api/v2/
├── search.go         # change WithGroupBy nil handling
└── groupby_test.go   # replace nil-no-op test and add request-construction regression
```

### Pattern 1: Fail fast in the option application path

The existing option path is already the correct validation boundary. Keep the nil guard in `groupByOption.ApplyToSearchRequest`, but make it return an error instead of silently succeeding.

```go
// Source: pkg/api/v2/search.go:631-642 (update nil branch only)
func (o *groupByOption) ApplyToSearchRequest(req *SearchRequest) error {
	if o.groupBy == nil {
		return errors.New("groupBy cannot be nil")
	}
	if err := o.groupBy.Validate(); err != nil {
		return err
	}
	req.GroupBy = o.groupBy
	return nil
}
```

Why here:
- it satisfies D-04 because the error happens before request serialization or network I/O
- it preserves valid `GroupBy` validation through the existing `Validate()` method
- it keeps omission semantics intact for callers who simply do not supply the option

### Pattern 2: Preserve valid non-nil behavior unchanged

`(*GroupBy).Validate()` in `pkg/api/v2/groupby.go` already enforces:
- aggregate is required
- aggregate itself validates
- at least one grouping key is required

Do not move nil-handling into `GroupBy.Validate()` or `MarshalJSON()`. That would shift the failure later than required and would not distinguish "explicit nil option passed" from "no option provided."

### Pattern 3: Test both the direct option path and the request-construction path

The current nil test only checks `WithGroupBy(nil).ApplyToSearchRequest(req)`. Phase 22 should also pin the higher-level contract that `NewSearchRequest(...)` fails before it appends a search.

Recommended coverage:
- `TestWithGroupBy/nil groupby returns exact validation error`
  - `err := WithGroupBy(nil).ApplyToSearchRequest(req)`
  - assert `require.EqualError(t, err, "groupBy cannot be nil")`
  - assert `require.Nil(t, req.GroupBy)`
- keep `apply valid groupby to search request`
- keep invalid non-nil cases unchanged
- add `TestSearchRequestWithGroupBy/search with nil groupby fails before append`
  - build a `SearchQuery{}`
  - call `NewSearchRequest(WithKnnRank(...), WithGroupBy(nil))`
  - assert error text is exact
  - assert `require.Len(t, sq.Searches, 0)`

### Pattern 4: Keep omission semantics by omission, not nil

Because `SearchRequest.GroupBy` is tagged `json:"group_by,omitempty"`, callers who want no grouping already get the correct behavior by not adding `WithGroupBy(...)`. The phase should not add any alternate sentinel behavior for nil.

## Common Pitfalls

### Pitfall 1: Returning the nil error too late
If the nil error is raised from `MarshalJSON()` or a collection method, the contract "before the request is sent" is weaker and easier to regress. Keep the check in `ApplyToSearchRequest`.

### Pitfall 2: Using a fuzzy error assertion
The context explicitly asks for a stable nil-specific message. Prefer `require.EqualError` over `require.Contains` so the contract is pinned exactly.

### Pitfall 3: Forgetting the higher-level request regression
Only testing `ApplyToSearchRequest` leaves the `NewSearchRequest(...)` composition path unpinned. Add one request-construction regression so a future refactor cannot accidentally swallow the error while still leaving the direct unit test green.

### Pitfall 4: Modifying `groupby.go` unnecessarily
This phase is not a redesign of `GroupBy`. Changing constructor or JSON behavior would expand scope without helping `GRP-01`.

## Threat Model Notes

This phase does not introduce a new trust boundary, external input channel, or privileged operation. The relevant defect is correctness and safety of client-side request construction:
- **Threat:** explicit programmer input (`WithGroupBy(nil)`) is silently downgraded into omitted grouping, causing unexpected query semantics
- **Severity:** low correctness / DX issue, not a security exploit
- **Mitigation:** deterministic fail-fast validation in the option application path with a stable message

No high-severity threats are introduced or required to be mitigated beyond the fail-fast validation behavior above.

## Validation Architecture

### Test Infrastructure

| Property | Value |
|----------|-------|
| Framework | `go test` |
| Config file | `Makefile` and existing `basicv2` build tag |
| Quick run command | `go test -tags=basicv2 -run 'TestWithGroupBy|TestSearchRequestWithGroupBy' ./pkg/api/v2/...` |
| Full suite command | `make test` |
| Lint command | `make lint` |
| Estimated runtime | quick run ~5s, full suite depends on local cache but typically short for V2-only scope |

### Verification Strategy

- Run the focused V2 tests after the implementation change to prove the nil-contract change without waiting on the full suite.
- Run `make test` before closing the phase to confirm no broader V2 regressions.
- Run `make lint` before phase completion because the repo treats lint as a standard pre-ship guard.
- No manual-only verification is required; the behavior is fully automatable with unit tests.

### Required Assertions

- Exact error string for explicit nil input: `groupBy cannot be nil`
- `req.GroupBy` remains `nil` on the error path
- Non-nil valid `GroupBy` still applies successfully
- Non-nil invalid `GroupBy` still returns the existing validation errors
- `NewSearchRequest(..., WithGroupBy(nil))` returns the error before any search is appended

---

_Research synthesized locally on 2026-04-09 after GSD researcher subagents stalled in this session._
