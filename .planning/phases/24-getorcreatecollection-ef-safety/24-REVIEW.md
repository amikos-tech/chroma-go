---
phase: 24-getorcreatecollection-ef-safety
reviewed: 2026-04-12T17:44:16+03:00
depth: standard
files_reviewed: 3
files_reviewed_list:
  - pkg/api/v2/client_local_embedded.go
  - pkg/api/v2/close_logging.go
  - pkg/api/v2/client_local_embedded_test.go
findings:
  critical: 0
  warning: 1
  info: 0
  total: 1
status: issues_found
---

# Phase 24: Code Review Report

**Reviewed:** 2026-04-12T17:44:16+03:00
**Depth:** standard
**Files Reviewed:** 3
**Status:** issues_found

## Summary

Reviewed the Phase 24 ownership-aware embedded fallback change across provisional state tracking, cleanup semantics, and the new deterministic/race regressions. The targeted `GetOrCreateCollection(...)` bug is fixed, but the provisional override path still overwrites previously owned state before revalidation, so a failing revalidation can discard the prior owner wrapper without restoring or closing it.

## Warnings

### WR-01: Failed revalidation after an explicit override can drop the previous owned EF from state

**File:** `pkg/api/v2/client_local_embedded.go:670-717`, `pkg/api/v2/client_local_embedded.go:730-744`, `pkg/api/v2/client_local_embedded.go:977-992`
**Issue:** `GetCollection(...)` now records borrowed-vs-owned provenance correctly, but it still mutates the live `collectionState` entry in place before the revalidation probe. When the collection already has an owned dense/content EF in state and a caller supplies an explicit override, lines 706-715 replace the stored wrapper and ownership bits immediately. If the revalidation call then fails on lines 730-744, `deleteCollectionState(...)` removes the entry and only skips closing the borrowed caller EF. The prior owned wrapper is no longer reachable from state, so client-level cleanup loses the only owner it had for that resource. The new regressions cover the caller-EF survival path, but they do not exercise "existing owned state + override + failing revalidation", so this lifecycle loss remains untested.
**Fix:**
```go
previous := *s

if contentEF != nil {
	s.contentEmbeddingFunction = wrapContentEFCloseOnce(contentEF)
	s.ownsContentEmbeddingFunction = req.contentEmbeddingFunction == nil
}
if ef != nil {
	s.embeddingFunction = wrapEFCloseOnce(ef)
	s.ownsEmbeddingFunction = req.embeddingFunction == nil &&
		!isDenseEFSharedWithContent(s.embeddingFunction, s.contentEmbeddingFunction)
}

...

if verifyErr != nil || verifiedModel == nil || verifiedModel.ID != model.ID {
	client.upsertCollectionState(model.ID, func(state *embeddedCollectionState) {
		*state = previous
	})
	return nil, ...
}
```

Either restore the previous state on build/revalidation failure, or defer mutating the shared state until the handoff is verified while still keeping enough provisional ownership metadata to clean up SDK-owned auto-wired resources safely. Add a regression that seeds an owned state EF, injects a revalidation failure after an explicit override, and asserts the original state EF remains reachable and closed exactly once on later client shutdown.

---

_Reviewed: 2026-04-12T17:44:16+03:00_
_Reviewer: Codex manual review_
_Depth: standard_
