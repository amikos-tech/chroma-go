# Phase 22: WithGroupBy Validation - Context

**Gathered:** 2026-04-09
**Status:** Ready for planning

<domain>
## Phase Boundary

Fix `WithGroupBy` so explicitly passing `nil` returns a clear validation error during request construction instead of silently omitting grouping. Keep existing non-nil `GroupBy` behavior and search request serialization unchanged.

**In scope:**
- Fail fast when callers explicitly invoke `WithGroupBy(nil)`
- Return a nil-specific validation error before any request is sent
- Preserve current behavior for valid non-nil `GroupBy` values
- Add/update colocated V2 tests for the new contract

**Out of scope:**
- Broader redesign of search option nil-handling semantics
- Any change to valid `GroupBy` JSON shape or aggregate/key validation rules
- Docs/examples updates for group-by behavior in this phase

</domain>

<decisions>
## Implementation Decisions

### Nil handling contract
- **D-01:** `WithGroupBy(nil)` is invalid and must fail fast during request construction/application; it is no longer treated as a silent no-op.
- **D-02:** Callers who want "no grouping" must omit the `WithGroupBy(...)` option entirely rather than passing `nil`.

### Error contract
- **D-03:** The nil failure must use a stable, nil-specific validation message so tests can assert exact text.
- **D-04:** The error should be returned from the option application path before any search request can be marshaled or sent.

### Scope and verification
- **D-05:** Phase 22 stays code-and-tests only; no docs/examples update is required for this bug fix.
- **D-06:** Existing valid non-nil `GroupBy` flows must remain unchanged, so regression coverage should include both the new nil-error path and a passing non-nil path.

### the agent's Discretion
- Exact nil-specific error wording, as long as it is stable and follows existing repo style for validation errors
- Whether to introduce a dedicated exported error constant or keep the stable message as an unexported direct error
- Exact unit-test structure in `groupby_test.go`

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone scope
- `.planning/ROADMAP.md` — Phase 22 goal, requirement mapping, and success criteria
- `.planning/REQUIREMENTS.md` — `GRP-01` requires `WithGroupBy(nil)` to return a validation error instead of silently skipping grouping
- `.planning/PROJECT.md` — v0.4.2 milestone context and issue `#482` bug statement

### Implementation targets
- `pkg/api/v2/search.go` — `WithGroupBy` and `groupByOption.ApplyToSearchRequest` currently treat nil as a no-op
- `pkg/api/v2/groupby.go` — existing `GroupBy.Validate()` behavior for non-nil values
- `pkg/api/v2/groupby_test.go` — current `TestWithGroupBy` coverage, including the nil-no-op case to replace

### Repo conventions
- `.planning/codebase/CONVENTIONS.md` — repo style for early validation and direct `"X cannot be nil"` error messages
- `.planning/codebase/TESTING.md` — colocated V2 tests, `require`-based assertions, and build-tag expectations
- `CLAUDE.md` — V2-first changes and colocated-test expectations

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `(*GroupBy).Validate()` in `pkg/api/v2/groupby.go` already enforces non-nil aggregate and required keys for valid group-by values
- `TestWithGroupBy` and `TestSearchRequestWithGroupBy` in `pkg/api/v2/groupby_test.go` already cover valid and invalid non-nil cases and provide the natural place for nil-contract tests

### Established Patterns
- Search option validation happens in `ApplyToSearchRequest` before request marshaling/sending
- The repo commonly uses direct nil-validation messages like `"tenant cannot be nil"` and `"database cannot be nil"`
- Public V2 bug fixes are tested with colocated unit tests under `pkg/api/v2/*_test.go`

### Integration Points
- `pkg/api/v2/search.go`: change the nil branch in `groupByOption.ApplyToSearchRequest`
- `pkg/api/v2/groupby_test.go`: replace the existing nil-no-op expectation with fail-fast error assertions and keep a passing valid case
- Any regression test that builds a `SearchRequest` through `NewSearchRequest(...)` should continue to pass for non-nil `GroupBy`

</code_context>

<specifics>
## Specific Ideas

- Best DX is explicit failure: "fail and fail fast, clear message and invariants enforced right away."
- The nil case should be treated as programmer error, not as shorthand for "skip grouping."
- No specific doc wording requested because this phase is intentionally code+tests only.

</specifics>

<deferred>
## Deferred Ideas

- Public docs/examples note that `WithGroupBy(nil)` is invalid — intentionally deferred because this phase is code+tests only
- Broader audit of other option setters that may still treat explicit `nil` as a no-op — separate consistency phase if needed

</deferred>

---

*Phase: 22-withgroupby-validation*
*Context gathered: 2026-04-09*
