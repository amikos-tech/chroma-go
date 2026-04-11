---
phase: 23-ort-ef-leak-fix
reviewed: 2026-04-11T10:14:44Z
depth: standard
files_reviewed: 3
files_reviewed_list:
  - pkg/api/v2/client.go
  - pkg/api/v2/client_local_embedded.go
  - pkg/api/v2/client_local_embedded_test.go
findings:
  critical: 0
  warning: 1
  info: 0
  total: 1
status: issues_found
---

# Phase 23: Code Review Report

**Reviewed:** 2026-04-11T10:14:44Z
**Depth:** standard
**Files Reviewed:** 3
**Status:** issues_found

## Summary

Reviewed the Phase 23 ORT EF lifecycle change across request preparation, embedded `CreateCollection`, and the new lifecycle regressions. The steady-state existing/new paths are covered, but the cleanup decision still depends on a separate preflight lookup instead of the actual `CreateCollection` result, which leaves a race window around ownership and leak handling.

## Warnings

### WR-01: Existing-collection cleanup still hinges on a stale preflight lookup

**File:** `pkg/api/v2/client_local_embedded.go:367-421`, `pkg/api/v2/client_local_embedded_test.go:1661-1748`
**Issue:** `CreateCollection(..., WithIfNotExistsCreate())` decides whether to preserve or close the temporary default EF from a separate `GetCollection` probe before it calls `embedded.CreateCollection`. That boolean is then treated as authoritative for the cleanup on lines 409-418. If the probe races with delete/recreate or transiently misses the existing collection, the Phase 23 fix can still do the wrong thing: an existing collection can go down the "new" branch and leak/adopt the temporary default, or a recreated collection can go down the "existing" branch and skip wiring the EF it now owns. The new tests only exercise the stable memory-runtime case, so this TOCTOU path is not covered.
**Fix:**
```go
existingID := ""
if req.CreateIfNotExists {
	existing, lookupErr := client.embedded.GetCollection(localchroma.EmbeddedGetCollectionRequest{
		Name:         req.Name,
		TenantID:     req.Database.Tenant().Name(),
		DatabaseName: req.Database.Name(),
	})
	if lookupErr == nil && existing != nil {
		existingID = existing.ID
	}
}

model, err := client.embedded.CreateCollection(...)
if err != nil {
	return nil, err
}

// Only treat this as the existing-path cleanup case when the returned model
// matches the collection observed before the create call.
sameExistingCollection := existingID != "" && model.ID == existingID
```

Use that stronger signal for the cleanup branch, or preferably extend the embedded runtime API to return whether the collection was newly created versus reused. Add a regression runtime that makes the probe stale or fail while `CreateCollection(GetOrCreate=true)` returns a valid model, and assert the temporary default EF is closed only when the returned model really is the pre-existing collection.

---

_Reviewed: 2026-04-11T10:14:44Z_
_Reviewer: Codex (gsd-code-reviewer)_
_Depth: standard_
