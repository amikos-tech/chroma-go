# Quick Task 260730-cjl: Normalize nil-handling across V2 SearchRequestOption helpers (GH #503) - Context

**Gathered:** 2026-07-30
**Status:** Ready for planning

<domain>
## Task Boundary

GH #503: Phase 22 made `WithGroupBy(nil)` fail fast with a validation error instead of
silently acting as omission. This left sibling `SearchRequestOption` implementations in
`pkg/api/v2/search.go` with inconsistent nil semantics. This task audits and normalizes
nil-handling across those option helpers so the V2 Search API presents a uniform contract
for explicit nil input.

</domain>

<decisions>
## Implementation Decisions

### WithSearchFilter(nil)
- Reject with an error, matching the `WithGroupBy(nil)` precedent (`errors.New("groupBy cannot be nil")` style).
- Currently `searchFilterOption.ApplyToSearchRequest` (pkg/api/v2/search.go:392) silently sets `req.Filter = nil` — this is the inconsistency being fixed.

### WithRank(nil)
- Reject with an error, matching the `WithGroupBy(nil)` precedent.
- `Rank` (pkg/api/v2/rank.go:51) is an interface type, so a plain `== nil` check is safe — no typed-nil footgun.
- Currently `rankOption.ApplyToSearchRequest` (pkg/api/v2/search.go:610) silently sets `req.Rank` to nil — this is the inconsistency being fixed.

### WithSelect() with zero keys
- Keep as a valid no-op. `WithSelect()` called with no projection keys is a legitimate
  empty selection, not the same nil-pointer footgun as GroupBy/Filter/Rank. No behavior
  change needed here — out of scope for this fix.

### Cross-SDK consistency check (Python/TS)
- Verified against `/Users/tazarov/RustroverProjects/chroma`: Python (`chromadb/execution/expression/plan.py`)
  and TypeScript (`clients/new-js/packages/chromadb/src/execution/expression/`) both treat
  `None`/`null`/`undefined` as a silent no-op/clear for filter, rank, group_by, AND select —
  none of them raise on nil for any of these four options.
- Explicit decision: chroma-go stays stricter and does NOT match Python/TS here. Rationale —
  Go's functional-options pattern already has a dedicated omission mechanism (don't call the
  option), so an explicit nil is usually a caller bug (e.g. a conditionally-nil variable
  threaded through), not a deliberate "clear this field" signal the way Python's `None`
  kwarg sentinel is. Silently swallowing a nil filter/rank risks a worse failure mode
  (unintentionally unfiltered/unranked search) than a loud, immediate error.
- This mirrors the reasoning already used for `WithGroupBy(nil)` in Phase 22 (see
  `.planning/phases/22-withgroupby-validation/22-DISCUSSION-LOG.md`) — extending, not
  reversing, that decision.
- **Added scope:** document this as an intentional, permanent divergence from Python/TS nil
  semantics (a short doc comment near the affected `With*` functions and/or a CHANGELOG note),
  so it isn't mistaken for an oversight in the future.

### Claude's Discretion
- Exact error message wording for the two new validation errors (follow the existing
  `WithGroupBy` error style/tone for consistency).
- Whether to add a shared helper for the nil-check pattern if it reduces duplication
  across the three option types, vs. inlining each check (keep radically simple —
  prefer inlining unless duplication is significant).
- Test structure/naming for the new validation error cases.
- Exact placement/wording of the Python/TS divergence doc note (doc comment vs CHANGELOG vs both).

</decisions>

<specifics>
## Specific Ideas

Reference implementation for the target contract — `groupByOption.ApplyToSearchRequest`
(pkg/api/v2/search.go:635-644):

```go
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

Apply the same `if o.X == nil { return errors.New("X cannot be nil") }` pattern to
`searchFilterOption.ApplyToSearchRequest` and `rankOption.ApplyToSearchRequest`.

</specifics>

<canonical_refs>
## Canonical References

- GH issue #503 (this task)
- Follow-up from PR #502 review on Phase 22 (`WithGroupBy(nil)` validation)
- `pkg/api/v2/search.go` — all `SearchRequestOption` implementations live here

</canonical_refs>
