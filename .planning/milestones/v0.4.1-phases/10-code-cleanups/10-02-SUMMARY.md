---
phase: 10-code-cleanups
plan: 02
subsystem: embeddings
tags: [mime-inference, url-fallback, registry-cleanup, test-isolation]

# Dependency graph
requires:
  - phase: 10-code-cleanups
    provides: "Shared pathutil package and context.Context fixes from Plan 01"
  - phase: 06-gemini-multimodal-adoption
    provides: "Gemini content.go with resolveMIME and extToMIME"
  - phase: 07-voyage-multimodal-adoption
    provides: "Voyage content.go with resolveMIME and extToMIME"
provides:
  - "URL path extension fallback in resolveMIME for both Gemini and Voyage providers"
  - "4 unexported unregister helpers for registry test cleanup"
  - "22 t.Cleanup calls preventing global state leaks in registry tests"
affects: [10-code-cleanups]

# Tech tracking
tech-stack:
  added: []
  patterns: ["URL path parsing with net/url for MIME inference", "t.Cleanup with unregister helpers for global registry isolation"]

key-files:
  created: []
  modified:
    - "pkg/embeddings/gemini/content.go"
    - "pkg/embeddings/gemini/gemini_content_test.go"
    - "pkg/embeddings/voyage/content.go"
    - "pkg/embeddings/voyage/voyage_content_test.go"
    - "pkg/embeddings/registry.go"
    - "pkg/embeddings/registry_test.go"

key-decisions:
  - "Remove dead TestVoyageContainsDotDot test that referenced removed containsDotDot function (Rule 3 - blocking)"

patterns-established:
  - "URL MIME inference: use url.Parse to strip query/fragment before filepath.Ext on URL path"
  - "Registry test cleanup: every Register* call paired with t.Cleanup(func() { unregister*(name) })"

requirements-completed: [CLN-04, CLN-05, CLN-06]

# Metrics
duration: 7min
completed: 2026-03-26
---

# Phase 10 Plan 02: URL MIME Inference and Registry Test Cleanup Summary

**resolveMIME gains URL path extension fallback via net/url in both Gemini and Voyage; registry tests use 4 unregister helpers with t.Cleanup across all 22 registration sites**

## Performance

- **Duration:** 7 min
- **Started:** 2026-03-26T10:23:45Z
- **Completed:** 2026-03-26T10:30:45Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Added URL path extension fallback as 3rd tier in resolveMIME for both Gemini and Voyage providers
- URL query strings and fragments are stripped via url.Parse before extension extraction
- Added 4 unexported unregister helpers (dense, sparse, multimodal, content) to registry.go
- Added t.Cleanup with unregister calls to all 22 registration test sites in registry_test.go
- Replaced inline mu.Lock/delete/mu.Unlock cleanup with proper unregister helper calls
- Tests pass with -count=3 proving no global state leaks

## Task Commits

Each task was committed atomically:

1. **Task 1: Add URL path extension fallback to resolveMIME in Gemini and Voyage** - `35e60f1` (feat)
2. **Task 2: Add registry unregister helpers and t.Cleanup to all registration tests** - `dcb8e31` (refactor)

## Files Created/Modified
- `pkg/embeddings/gemini/content.go` - Added net/url import and URL path fallback in resolveMIME
- `pkg/embeddings/gemini/gemini_content_test.go` - Added 4 URL MIME inference test cases
- `pkg/embeddings/voyage/content.go` - Added net/url import and URL path fallback in resolveMIME
- `pkg/embeddings/voyage/voyage_content_test.go` - Added 4 URL MIME inference test cases, removed dead containsDotDot test
- `pkg/embeddings/registry.go` - Added 4 unexported unregister helpers
- `pkg/embeddings/registry_test.go` - Added t.Cleanup to all 22 registration call sites, replaced inline cleanup

## Decisions Made
- Removed dead TestVoyageContainsDotDot test that referenced function removed in Plan 01 (Rule 3 auto-fix)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Removed dead TestVoyageContainsDotDot test**
- **Found during:** Task 1 (resolveMIME URL fallback)
- **Issue:** TestVoyageContainsDotDot referenced `containsDotDot` function that was removed in Plan 01 and replaced with pathutil.ValidateFilePath; the test file would not compile
- **Fix:** Removed the dead test function (pathutil already has its own comprehensive tests in pathutil_test.go)
- **Files modified:** pkg/embeddings/voyage/voyage_content_test.go
- **Verification:** Voyage test suite compiles and passes
- **Committed in:** 35e60f1 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Dead test removal necessary for compilation. No scope creep.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 10 (Code Cleanups) is fully complete across both plans
- All 6 success criteria from the phase are met
- Ready for Phase 11 (Fork Double-Close Bug)

## Self-Check: PASSED

All 7 files exist, both commits (35e60f1, dcb8e31) found, all acceptance criteria verified.

---
*Phase: 10-code-cleanups*
*Completed: 2026-03-26*
