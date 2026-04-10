---
phase: 22-withgroupby-validation
reviewed: 2026-04-10T04:45:37Z
depth: standard
files_reviewed: 2
files_reviewed_list:
  - pkg/api/v2/search.go
  - pkg/api/v2/groupby_test.go
findings:
  critical: 0
  warning: 1
  info: 0
  total: 1
status: issues_found
---

# Phase 22: Code Review Report

**Reviewed:** 2026-04-10T04:45:37Z
**Depth:** standard
**Files Reviewed:** 2
**Status:** issues_found

## Summary

Reviewed the `WithGroupBy` nil-validation change in `pkg/api/v2/search.go` and the new coverage in `pkg/api/v2/groupby_test.go`. The targeted tests pass locally, but the phase changes the behavior of a public option from a no-op to a hard error, which is a compatibility regression for callers that build option lists with an optional `*GroupBy`.

Targeted verification:

```bash
go test -tags=basicv2 ./pkg/api/v2 -run 'Test(MinK|MaxK|GroupBy|WithGroupBy|SearchRequestWithGroupBy)$'
```

Passed locally.

## Warnings

### WR-01: `WithGroupBy(nil)` is now a breaking API change

**File:** `pkg/api/v2/search.go:635-638`
**Issue:** Prior to this phase, passing `WithGroupBy(nil)` was effectively a no-op because `ApplyToSearchRequest` returned `nil` without mutating the request. The new branch now returns `groupBy cannot be nil`, and the updated tests lock that behavior in. That breaks any existing caller that conditionally threads a `*GroupBy` through an option list and uses `nil` to mean "no grouping". For a public functional-option API, that is a behavior regression rather than a pure validation improvement.
**Fix:**

```go
func (o *groupByOption) ApplyToSearchRequest(req *SearchRequest) error {
	if o.groupBy == nil {
		return nil
	}
	if err := o.groupBy.Validate(); err != nil {
		return err
	}
	req.GroupBy = o.groupBy
	return nil
}
```

If strict rejection of explicit `nil` is required, ship it as a documented breaking change (or under a new strict helper) instead of changing the existing option semantics in place.

---

_Reviewed: 2026-04-10T04:45:37Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
