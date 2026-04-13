---
phase: 25-error-body-truncation
plan: 01
subsystem: embeddings
tags: [go, embeddings, http, error-handling, openrouter, perplexity]
requires: []
provides:
  - shared panic-safe error body display sanitizer in pkg/commons/http
  - normalized Perplexity and OpenRouter error formatting onto the exact [truncated] contract
  - focused regression coverage for trim, rune-safe truncation, and sanitized OpenRouter structured messages
affects: [embeddings, http, openrouter, perplexity, error-handling]
tech-stack:
  added: []
  patterns: [shared display-layer sanitization, panic-safe best-effort fallback, provider-local structured parsing with shared sanitization]
key-files:
  created: []
  modified:
    - pkg/commons/http/utils.go
    - pkg/commons/http/utils_test.go
    - pkg/embeddings/perplexity/perplexity.go
    - pkg/embeddings/perplexity/perplexity_test.go
    - pkg/embeddings/openrouter/openrouter.go
    - pkg/embeddings/openrouter/openrouter_test.go
key-decisions:
  - "Kept ReadLimitedBody and MaxResponseBodySize unchanged so transport safety and display safety stay separate concerns."
  - "Sanitized OpenRouter's parsed error.message as body-derived text instead of trusting structured JSON fields to remain short."
  - "Left ERR-02 pending in REQUIREMENTS because 25-01 only normalizes Perplexity/OpenRouter; later Phase 25 plans still migrate the remaining providers."
patterns-established:
  - "Shared provider error text should flow through pkg/commons/http.SanitizeErrorBody instead of package-local truncation helpers."
  - "Sanitizers in production code recover and return best-effort sanitized output instead of panicking or leaking raw body text."
requirements-completed: [ERR-01]
duration: 3 min
completed: 2026-04-13
---

# Phase 25 Plan 01: Error Body Truncation Summary

**Shared panic-safe error body sanitizer with the exact `[truncated]` contract for Perplexity and OpenRouter provider errors**

## Performance

- **Duration:** 3 min
- **Started:** 2026-04-13T10:25:59+03:00
- **Completed:** 2026-04-13T10:28:43+03:00
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Added `pkg/commons/http.SanitizeErrorBody(...)` with whitespace trimming, 512-rune truncation, exact `[truncated]` suffixing, and panic recovery that falls back to the same sanitization contract.
- Replaced Perplexity's bespoke truncation helper and OpenRouter's raw/structured local truncation behavior with the shared sanitizer.
- Pinned the contract in focused helper/provider tests, including OpenRouter structured `error.message` sanitization and UTF-8-safe truncation coverage.

## Task Commits

Each task was committed atomically:

1. **Task 1: Pin the shared sanitizer contract in tests before changing production code** - `12443b1` (test)
2. **Task 2: Implement the shared helper with panic recovery and normalize Perplexity/OpenRouter onto it** - `3803267` (fix)

## Files Created/Modified

- `pkg/commons/http/utils.go` - adds the shared panic-safe display sanitizer without changing transport-level body limits.
- `pkg/commons/http/utils_test.go` - pins nil/empty handling, trim behavior, ASCII truncation, and UTF-8-safe truncation for the shared helper.
- `pkg/embeddings/perplexity/perplexity.go` - removes the local truncation helper and routes HTTP error formatting through `chttp.SanitizeErrorBody`.
- `pkg/embeddings/perplexity/perplexity_test.go` - updates Perplexity regressions to the exact `[truncated]` contract and keeps UTF-8-safe provider-path coverage.
- `pkg/embeddings/openrouter/openrouter.go` - sanitizes both parsed `error.message` and raw fallback body text through the shared helper.
- `pkg/embeddings/openrouter/openrouter_test.go` - verifies the shared truncation contract for both structured and raw OpenRouter error paths.

## Decisions Made

- Kept the new behavior in `pkg/commons/http` so later Phase 25 provider migrations can be mechanical replacements instead of repeated contract design work.
- Treated OpenRouter `error.message` as body-derived text and sanitized it, matching the phase threat model instead of trusting structured API fields.
- Marked only `ERR-01` complete in `REQUIREMENTS.md`; `ERR-02` remains open until the remaining provider migrations land in Plans 25-02 through 25-04.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Retried the Task 1 commit after a transient git index lock**
- **Found during:** Task 1 commit
- **Issue:** `git commit` initially failed because `.git/index.lock` was present while another workspace git process was active elsewhere on the machine.
- **Fix:** Confirmed the lock was transient/not held by this repository workflow, then retried the staged commit once the lock cleared.
- **Files modified:** none
- **Verification:** The retry succeeded and produced commit `12443b1`.
- **Committed in:** `12443b1`

**2. [Rule 3 - Blocking] Patched STATE.md manually after `state record-metric` failed**
- **Found during:** Plan metadata update
- **Issue:** `state record-metric` reported that the `Performance Metrics` section was missing even though the section existed, leaving the human-readable state body out of sync with the updated plan position.
- **Fix:** Kept the successful GSD helper updates for plan position/roadmap/requirements, then manually corrected the visible `STATE.md` progress, completed-plan count, last-activity text, and missing Phase 25 decision entry.
- **Files modified:** `.planning/STATE.md`
- **Verification:** Re-read `.planning/STATE.md` and confirmed it reflects Phase 25 plan 2 of 4, 70% progress, and all three Phase 25 execution decisions.
- **Committed in:** metadata commit

### Workflow Adjustments

- Committed the Task 1 RED checkpoint as `12443b1` because the explicit user instruction required atomic per-task commits with `--no-verify`, even though `25-VALIDATION.md` labels `25-01-01` as an intentional no-commit RED checkpoint.

---

**Total deviations:** 2 auto-fixed (2 blocking) plus 1 user-directed workflow override
**Impact on plan:** Both blocking fixes were workflow-only and did not change code scope. The workflow override only changed commit timing for the RED checkpoint; implementation and verification stayed aligned with the plan.

## Issues Encountered

- A transient `.git/index.lock` blocked the first Task 1 commit attempt; retrying after the lock cleared resolved it without touching unrelated files.
- `state record-metric` failed against the current `STATE.md` shape, so the visible state fields were corrected manually after the successful GSD helper updates.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `SanitizeErrorBody(...)` now defines the shared display contract, so Plans 25-02 through 25-04 can migrate the remaining providers without re-deciding suffix, trim, or panic-recovery behavior.
- `ERR-01` is satisfied; `ERR-02` remains phase-partial until the rest of the embedding provider sweep is complete.

## Verification Evidence

- `go test -tags=ef ./pkg/commons/http ./pkg/embeddings/openrouter ./pkg/embeddings/perplexity` -> passed (`ok github.com/amikos-tech/chroma-go/pkg/commons/http`, `ok github.com/amikos-tech/chroma-go/pkg/embeddings/openrouter`, `ok github.com/amikos-tech/chroma-go/pkg/embeddings/perplexity`)
- `make lint` -> passed (`0 issues.`)

## Self-Check: PASSED

- Verified `.planning/phases/25-error-body-truncation/25-01-SUMMARY.md` exists on disk.
- Verified task commits `12443b1` and `3803267` exist in git history.

---
*Phase: 25-error-body-truncation*
*Completed: 2026-04-13*
