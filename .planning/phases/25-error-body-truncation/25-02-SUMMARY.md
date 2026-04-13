---
phase: 25-error-body-truncation
plan: 02
subsystem: embeddings
tags: [go, embeddings, http, error-handling, openai, baseten, bedrock, chroma-cloud]
requires:
  - phase: 25-01
    provides: shared SanitizeErrorBody contract and provider-facing truncation semantics
provides:
  - OpenAI, Baseten, Bedrock, Chroma Cloud, and Chroma Cloud Splade raw-body error paths migrated to pkg/commons/http.SanitizeErrorBody
  - focused OpenAI and Baseten regressions that pin the [truncated] contract on real provider HTTP scaffolding
  - the first mechanical ERR-02 migration slice after the shared helper landed
affects: [embeddings, error-handling, openai, baseten, bedrock, chroma-cloud, chroma-cloud-splade]
tech-stack:
  added: []
  patterns: [shared raw-body sanitization in provider errors, representative httptest regressions for long provider bodies]
key-files:
  created: []
  modified:
    - pkg/embeddings/openai/openai.go
    - pkg/embeddings/openai/openai_test.go
    - pkg/embeddings/baseten/baseten.go
    - pkg/embeddings/baseten/baseten_test.go
    - pkg/embeddings/bedrock/bedrock.go
    - pkg/embeddings/chromacloud/chromacloud.go
    - pkg/embeddings/chromacloudsplade/chromacloudsplade.go
key-decisions:
  - "Kept provider-specific status and endpoint wording intact and changed only the body-derived segment to use chttp.SanitizeErrorBody(...)."
  - "Used OpenAI and Baseten as the representative long-body regressions for this batch-A slice while keeping Bedrock and Chroma Cloud migrations mechanical."
  - "Left ERR-02 pending in REQUIREMENTS because this plan only covers the first provider batch."
patterns-established:
  - "Raw provider HTTP body text should be interpolated through pkg/commons/http.SanitizeErrorBody instead of string(respData), string(respBody), or string(body)."
  - "Representative provider regressions should assert [truncated] and explicitly reject the full oversized payload."
requirements-completed: []
duration: 2 min
completed: 2026-04-13
---

# Phase 25 Plan 02: Error Body Truncation Summary

**Shared error-body sanitization rolled out to OpenAI, Baseten, Bedrock, Chroma Cloud, and Chroma Cloud Splade with long-body regressions for the first raw-provider batch**

## Performance

- **Duration:** 2 min
- **Started:** 2026-04-13T10:37:08+03:00
- **Completed:** 2026-04-13T10:39:04+03:00
- **Tasks:** 1
- **Files modified:** 7

## Accomplishments

- Replaced raw body interpolation with `chttp.SanitizeErrorBody(...)` in the first batch of representative providers without changing each provider's existing status wording.
- Added OpenAI and Baseten regressions that prove oversized provider bodies now surface `[truncated]` and no longer dump the full payload into the returned error.
- Verified the first post-helper migration slice against the shared helper contract by running the focused Wave 2 package tests and lint.

## Task Commits

This TDD task produced atomic RED and GREEN commits:

1. **Task 1 RED: add failing long-body provider regressions** - `a77236f` (test)
2. **Task 1 GREEN: sanitize batch-A provider error bodies** - `56ae613` (fix)

## Files Created/Modified

- `pkg/embeddings/openai/openai.go` - routes raw OpenAI error bodies through the shared sanitizer.
- `pkg/embeddings/openai/openai_test.go` - adds a long-body regression asserting `[truncated]` and rejecting the full payload.
- `pkg/embeddings/baseten/baseten.go` - sanitizes raw Baseten error-body text before formatting the provider error.
- `pkg/embeddings/baseten/baseten_test.go` - adds the representative Baseten long-body regression.
- `pkg/embeddings/bedrock/bedrock.go` - migrates the bearer-token error path from `string(respBody)` to the shared sanitizer.
- `pkg/embeddings/chromacloud/chromacloud.go` - sanitizes the raw non-200 body segment while preserving the existing status-code wording.
- `pkg/embeddings/chromacloudsplade/chromacloudsplade.go` - applies the same raw-body sanitization pattern to the sparse embedding client.

## Decisions Made

- Preserved provider-specific error wording and only changed the raw-body segment, keeping this plan a mechanical migration instead of an error-format redesign.
- Kept the regression footprint narrow to OpenAI and Baseten because they already had straightforward `httptest` scaffolding and are enough to pin the shared display contract for this batch.
- Left `ERR-02` pending because the remaining provider migrations still belong to Plans `25-03` and `25-04`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Patched the visible state progress after `state record-metric` failed**
- **Found during:** Plan metadata update
- **Issue:** `node ... state record-metric` reported `Performance Metrics section not found in STATE.md`, so the helper did not refresh the human-readable progress/velocity fields after the successful plan advance.
- **Fix:** Kept the successful GSD helper updates for plan position, decisions, roadmap progress, and session state, then manually corrected the visible `STATE.md` last-activity text, progress bar, and completed-plan count.
- **Files modified:** `.planning/STATE.md`
- **Verification:** Re-read `.planning/STATE.md` and confirmed it reflects Plan 3 of 4, 80% progress, and the new Phase 25 execution decisions.
- **Committed in:** metadata commit

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Workflow-only closeout fix. No implementation scope change and no effect on the provider migration itself.

## Issues Encountered

- `state record-metric` still fails against the current `STATE.md` shape, so the visible progress fields were updated manually after the successful GSD helper calls.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The first batch-A slice is green, so the remaining provider sweep can keep using the same `SanitizeErrorBody(...)` replacement pattern.
- `25-03` can proceed independently for the sibling wave-2 providers, and `25-04` can finish `ERR-02` with the remaining batch and final embedding-package sweep.

## Verification Evidence

- `go test -tags=ef ./pkg/commons/http ./pkg/embeddings/openrouter ./pkg/embeddings/perplexity ./pkg/embeddings/openai ./pkg/embeddings/baseten ./pkg/embeddings/bedrock ./pkg/embeddings/chromacloud ./pkg/embeddings/chromacloudsplade` -> passed
- `make lint` -> passed (`0 issues.`)

## Self-Check: PASSED

- Verified `.planning/phases/25-error-body-truncation/25-02-SUMMARY.md` exists on disk.
- Verified task commits `a77236f` and `56ae613` exist in git history.

---
*Phase: 25-error-body-truncation*
*Completed: 2026-04-13*
