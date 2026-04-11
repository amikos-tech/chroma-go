---
phase: 23
fixed_at: 2026-04-11T10:41:48Z
review_path: .planning/phases/23-ort-ef-leak-fix/23-REVIEW.md
iteration: 1
findings_in_scope: 1
fixed: 1
skipped: 0
status: all_fixed
---

# Phase 23: Code Review Fix Report

**Fixed at:** 2026-04-11T10:41:48Z
**Source review:** `.planning/phases/23-ort-ef-leak-fix/23-REVIEW.md`
**Iteration:** 1

**Summary:**
- Findings in scope: 1
- Fixed: 1
- Skipped: 0

## Fixed Issues

### WR-01: Existing-collection cleanup still hinges on a stale preflight lookup

**Status:** fixed: requires human verification
**Files modified:** `pkg/api/v2/client_local_embedded.go`, `pkg/api/v2/client_local_embedded_test.go`
**Commit:** `9c4eb13`
**Applied fix:** Replaced the preflight boolean with returned-model identity checks plus existing client-owned state/cache detection, so `CreateCollection(..., WithIfNotExistsCreate())` only runs the existing-collection cleanup path when the returned model is truly the collection this client already owns. Added regressions for both a missed preflight probe against an already-owned collection and a stale-positive probe that returns a deleted collection before a replacement is created.

---

_Fixed: 2026-04-11T10:41:48Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
