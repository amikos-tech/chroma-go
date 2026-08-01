# Quick Task 260801-g9r: Review, validate, and address the scope of issue 516 - Context

**Gathered:** 2026-08-01
**Status:** Ready for planning

<domain>
## Task Boundary

Review issue 516 against the current API v2 search implementation, validate its claims and proposed scope, then implement the smallest complete fix with regression coverage.

</domain>

<decisions>
## Implementation Decisions

### Rank validation contract
- Validate built-in rank structures recursively when `WithRank` is applied.
- Preserve compatibility with caller-defined `Rank` implementations; do not add a required method to the public `Rank` interface.
- Invalid input must be rejected before it is assigned to the request.

### Filter behavior
- Eagerly validate every non-nil `SearchFilter.Where` clause when `WithSearchFilter` is applied, including nested clauses.
- Keep an empty `SearchFilter` valid.
- Keep IDs-only filters valid.
- Preserve existing nil and typed-nil `Where` behavior unless current documented behavior requires a validation error.

### Error attribution scope
- Make eager option validation and regression tests the required scope.
- During research, inspect the HTTP-layer error wrapping.
- Include a narrowly scoped wrapper improvement only if it cleanly distinguishes pre-send serialization failures without changing public behavior; otherwise record it as out of scope.

### Agent's Discretion
- Choose the smallest internal validation design that covers all built-in rank forms recursively and avoids duplicate validation logic.
- Select focused unit and request-path tests that prove failure timing and prevent request mutation on validation errors.

</decisions>

<specifics>
## Specific Ideas

- Use `groupByOption.ApplyToSearchRequest` and `WithSearchWhere` as existing eager-validation precedents.
- Explicitly cover zero-value `KnnRank`, invalid `RrfRank`, an invalid nested built-in rank, and an invalid nested `Where` clause.
- Confirm that a valid custom `Rank` implementation remains accepted.

</specifics>

<canonical_refs>
## Canonical References

- Public issue 516: https://github.com/amikos-tech/chroma-go/issues/516
- Related typed-nil rank hardening context: public issue 515

</canonical_refs>
