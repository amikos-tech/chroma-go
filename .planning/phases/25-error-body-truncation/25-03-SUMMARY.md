---
phase: 25-error-body-truncation
plan: 03
subsystem: embeddings
tags: [go, embeddings, error-handling, cloudflare, cohere, huggingface, jina]
requires:
  - phase: 25-01
    provides: shared panic-safe SanitizeErrorBody contract in pkg/commons/http
provides:
  - Cloudflare preserves parsed embeddings.Errors while truncating only the appended raw-body tail
  - Cohere, Hugging Face, and Jina route raw HTTP error text through the shared sanitizer
  - focused Cloudflare regression coverage for the mixed structured/raw error string contract
affects: [embeddings, cloudflare, cohere, hf, jina, error-handling]
tech-stack:
  added: []
  patterns: [provider-specific body sanitization via shared helper, httptest regression coverage for mixed structured/raw error output]
key-files:
  created:
    - pkg/embeddings/cloudflare/cloudflare_error_test.go
  modified:
    - pkg/embeddings/cloudflare/cloudflare.go
    - pkg/embeddings/cohere/cohere.go
    - pkg/embeddings/hf/hf.go
    - pkg/embeddings/jina/jina.go
key-decisions:
  - "Kept Cloudflare's parsed embeddings.Errors segment intact and sanitized only the appended raw-body portion."
  - "Proved the Cloudflare mixed-format contract with a focused httptest regression instead of a source-only assertion."
  - "Updated Cohere's default embed model to embed-english-v3.0 after the retired v2.0 default blocked live ef verification on April 13, 2026."
patterns-established:
  - "Providers that combine structured metadata with body text should sanitize only the body-derived segment, not the parsed structured fields."
  - "Focused provider error-format regressions can use local httptest servers to verify emitted error strings without relying on remote provider behavior."
requirements-completed: []
duration: 12 min
completed: 2026-04-13
---

# Phase 25 Plan 03: Error Body Truncation Summary

**Cloudflare mixed structured/raw errors preserved while Cohere, Hugging Face, and Jina now truncate body-derived error text through the shared sanitizer**

## Performance

- **Duration:** 12 min
- **Started:** 2026-04-13T10:28:00+03:00
- **Completed:** 2026-04-13T10:40:05+03:00
- **Tasks:** 1
- **Files modified:** 5

## Accomplishments

- Kept Cloudflare's existing `embeddings.Errors` formatting and replaced only the appended raw response-body segment with `chttp.SanitizeErrorBody(respData)`.
- Migrated Cohere, Hugging Face, and Jina away from direct `string(respData)` interpolation in provider error messages.
- Added a focused `ef` regression in `pkg/embeddings/cloudflare/cloudflare_error_test.go` that proves the emitted error string contains structured error content, a sanitized `[truncated]` tail, and not the full payload.

## Task Commits

Each task was committed atomically:

1. **Task 1: Finish batch A and preserve the Cloudflare mixed structured/raw format** - `6bfd60b` (fix)

## Files Created/Modified

- `pkg/embeddings/cloudflare/cloudflare.go` - preserves the structured `embeddings.Errors` segment while sanitizing only the appended raw response-body text.
- `pkg/embeddings/cloudflare/cloudflare_error_test.go` - exercises the non-`ef` Cloudflare error path via `httptest` and asserts the mixed structured/raw error-string contract.
- `pkg/embeddings/cohere/cohere.go` - sanitizes raw-body error text and updates the default Cohere embed model to a currently supported v3 model so the live `ef` suite can run.
- `pkg/embeddings/hf/hf.go` - routes Hugging Face raw error bodies through `chttp.SanitizeErrorBody`.
- `pkg/embeddings/jina/jina.go` - routes Jina raw error bodies through `chttp.SanitizeErrorBody`.

## Decisions Made

- Kept the Cloudflare change narrowly scoped to the appended raw-body tail so the approved `embeddings.Errors` output remains intact.
- Used a dedicated regression test for Cloudflare because this provider's mixed structured/raw formatting is a special case that a grep-only check would miss.
- Treated the retired Cohere default model as a blocking verification bug and moved the default to `embed-english-v3.0`, which is listed in Cohere's current embed model docs.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated Cohere's retired default embed model to restore the required ef verification**
- **Found during:** Task 1 (Finish batch A and preserve the Cloudflare mixed structured/raw format)
- **Issue:** The plan's required `go test -tags=ef ... ./pkg/embeddings/cohere ...` verification failed because Cohere removed `embed-english-v2.0`; the default-path test returned a live `404 Not Found` retirement error on April 13, 2026.
- **Fix:** Changed `DefaultEmbedModel` from `embed-english-v2.0` to `embed-english-v3.0` in `pkg/embeddings/cohere/cohere.go`, then reran the full plan verification successfully.
- **Files modified:** `pkg/embeddings/cohere/cohere.go`
- **Verification:** `go test -count=1 -tags=ef ./pkg/commons/http ./pkg/embeddings/cloudflare ./pkg/embeddings/cohere ./pkg/embeddings/hf ./pkg/embeddings/jina && make lint`
- **Committed in:** `6bfd60b`

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** The deviation was necessary to complete the plan's required live verification. It did not widen the sanitizer scope beyond the plan's provider slice.

## Issues Encountered

- The first full verification run failed in Cohere's existing `ef` suite because the provider's default model had been retired upstream. Updating the default to a currently supported v3 model resolved the blocker and kept the required verification green.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 25's wave-2 Cloudflare/Cohere/HF/Jina slice is complete and isolated from the 25-02 file set.
- `ERR-02` remains phase-partial until Plan 25-04 finishes the remaining provider sweep.

## Known Stubs

- `pkg/embeddings/jina/jina.go:53` retains a pre-existing `TODO` about non-float embedding-type support in the response struct. It predates this plan and does not affect the error-body sanitization scope or verification.

## Verification Evidence

- `go test -tags=ef ./pkg/embeddings/cloudflare -run TestCreateEmbeddingPreservesStructuredErrorsWhileSanitizingRawTail` -> passed (`ok github.com/amikos-tech/chroma-go/pkg/embeddings/cloudflare`)
- `go test -count=1 -tags=ef ./pkg/commons/http ./pkg/embeddings/cloudflare ./pkg/embeddings/cohere ./pkg/embeddings/hf ./pkg/embeddings/jina && make lint` -> passed (`ok` for all five packages, `0 issues.` from `golangci-lint`)

## Self-Check: PASSED

- Verified `.planning/phases/25-error-body-truncation/25-03-SUMMARY.md` exists on disk.
- Verified task commit `6bfd60b` exists in git history.

---
*Phase: 25-error-body-truncation*
*Completed: 2026-04-13*
