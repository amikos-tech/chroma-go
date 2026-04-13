---
phase: 25-error-body-truncation
plan: 04
subsystem: embeddings
tags: [go, embeddings, error-handling, twelvelabs, ollama, voyage]
requires:
  - phase: 25-01
    provides: shared panic-safe SanitizeErrorBody contract in pkg/commons/http
  - phase: 25-02
    provides: representative raw-body provider regressions and the first batch-A sanitizer rollout
  - phase: 25-03
    provides: Cloudflare mixed-format coverage and the remaining wave-2 provider migrations
provides:
  - Batch-B providers route raw HTTP error text through chttp.SanitizeErrorBody
  - Twelve Labs sanitizes both parsed message fields and raw fallback bodies, with focused truncation regressions
  - The full embedding provider tree and lint gate pass after the Phase 25 sanitizer rollout
affects: [embeddings, twelvelabs, ollama, error-handling, phase-26]
tech-stack:
  added: []
  patterns: [complete shared-sanitizer rollout across providers, TDD regression coverage for structured provider error text]
key-files:
  created: []
  modified:
    - pkg/embeddings/mistral/mistral.go
    - pkg/embeddings/morph/morph.go
    - pkg/embeddings/nomic/nomic.go
    - pkg/embeddings/ollama/ollama.go
    - pkg/embeddings/roboflow/roboflow.go
    - pkg/embeddings/together/together.go
    - pkg/embeddings/twelvelabs/twelvelabs.go
    - pkg/embeddings/twelvelabs/twelvelabs_test.go
    - pkg/embeddings/voyage/voyage.go
key-decisions:
  - "Kept the batch-B provider edits mechanical by changing only the body-derived error segment and preserving existing status/endpoint wording."
  - "Treated Twelve Labs' parsed apiErr.Message as body-derived text and sanitized it the same way as the raw fallback path."
  - "Worked around the host Docker credsStore timeout by using a temporary authless DOCKER_CONFIG and pre-pulling ollama/ollama:latest before rerunning the required ef gates."
patterns-established:
  - "Structured error-message fields sourced from provider HTTP bodies should be sanitized exactly like raw fallback text."
  - "When Docker credential helpers block public-image testcontainers pulls, pre-pulling the image through an isolated temporary DOCKER_CONFIG restores the intended verification path without changing repository code."
requirements-completed: [ERR-02]
duration: 23 min
completed: 2026-04-13
---

# Phase 25 Plan 04: Error Body Truncation Summary

**Final batch-B providers and Twelve Labs structured errors now truncate body-derived text, and the full embedding tree passes the sanitizer rollout gate**

## Performance

- **Duration:** 23 min
- **Started:** 2026-04-13T07:46:50Z
- **Completed:** 2026-04-13T08:10:04Z
- **Tasks:** 3
- **Files modified:** 9

## Accomplishments

- Migrated Mistral, Morph, Nomic, Ollama, Roboflow, Together, and Voyage away from direct raw-body interpolation while preserving each provider's existing status and endpoint wording.
- Applied the approved Twelve Labs policy to both parsed `message` content and the raw fallback path, then pinned it with focused `[truncated]` regressions.
- Closed Phase 25 with a green `go test -tags=ef ./pkg/commons/http ./pkg/embeddings/... && make lint` gate.

## Task Commits

Each task was committed atomically:

1. **Task 1: Migrate the remaining raw-body providers in batch B** - `0b78545` (fix)
2. **Task 2: Sanitize Twelve Labs structured messages and add the regression** - `cd7ba9a` (test), `28bad02` (fix)
3. **Task 3: Run the full embedding sweep and lint gate** - `215573a` (test)

## Files Created/Modified

- `pkg/embeddings/mistral/mistral.go`, `pkg/embeddings/morph/morph.go`, `pkg/embeddings/nomic/nomic.go`, `pkg/embeddings/ollama/ollama.go`, `pkg/embeddings/roboflow/roboflow.go`, `pkg/embeddings/together/together.go`, and `pkg/embeddings/voyage/voyage.go` - replace direct raw response-body interpolation with `chttp.SanitizeErrorBody(...)`.
- `pkg/embeddings/twelvelabs/twelvelabs.go` - sanitizes both parsed `apiErr.Message` text and raw fallback bodies before formatting the returned error.
- `pkg/embeddings/twelvelabs/twelvelabs_test.go` - adds focused structured-message and raw-fallback regressions asserting `[truncated]` and rejecting the full long payload.

## Decisions Made

- Kept the batch-B provider migrations strictly mechanical to avoid accidental wording or endpoint-format churn.
- Treated Twelve Labs structured JSON `message` values as body-derived text rather than trusted metadata, matching the approved OpenRouter policy.
- Used an isolated temporary Docker config plus a one-time public image pre-pull to unblock the required `ollama` `ef` verification without changing repository code.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Bypassed the host Docker credential-helper timeout to complete the required `ollama` verification**
- **Found during:** Task 1 (Migrate the remaining raw-body providers in batch B)
- **Issue:** The plan's required focused verification timed out in `pkg/embeddings/ollama` because the host `~/.docker/config.json` used `credsStore=desktop`, `docker-credential-desktop get` hung, and `testcontainers-go` exhausted the package's 10-minute timeout while trying to pull `ollama/ollama:latest`.
- **Fix:** Created `/tmp/chroma-go-docker-config/config.json` without the broken credential-helper path, pre-pulled `ollama/ollama:latest` once through that config, and reran the required verification commands with `DOCKER_CONFIG=/tmp/chroma-go-docker-config`.
- **Files modified:** none (verification environment only)
- **Verification:** `go test -tags=ef ./pkg/commons/http ./pkg/embeddings/mistral ./pkg/embeddings/morph ./pkg/embeddings/nomic ./pkg/embeddings/ollama ./pkg/embeddings/roboflow ./pkg/embeddings/together ./pkg/embeddings/voyage && make lint`; `go test -tags=ef ./pkg/commons/http ./pkg/embeddings/... && make lint`
- **Committed in:** none (environment-only unblock)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** The deviation was limited to the local verification environment. The repository changes stayed within the planned sanitizer scope.

## Issues Encountered

- The first Task 2 GREEN verification attempt hit a live Twelve Labs rate limit and returned `429 Too Many Requests` with a retry point of `2026-04-13T08:08:35Z`. Rerunning the exact task gate after that timestamp passed cleanly with no code changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 25 is complete and `ERR-02` is now satisfied across the embedding providers audited in the phase.
- Phase 26 can build on the sanitized Twelve Labs error paths before adding async embedding behavior.

## Known Stubs

- `pkg/embeddings/mistral/mistral.go:92` retains a pre-existing `TODO` about non-float embedding support in the response struct. It predates this plan and does not affect the error-body sanitization scope or verification.

## Verification Evidence

- `go test -tags=ef ./pkg/commons/http ./pkg/embeddings/mistral ./pkg/embeddings/morph ./pkg/embeddings/nomic ./pkg/embeddings/ollama ./pkg/embeddings/roboflow ./pkg/embeddings/together ./pkg/embeddings/voyage && make lint` -> passed after pre-pulling `ollama/ollama:latest` through `DOCKER_CONFIG=/tmp/chroma-go-docker-config`
- `go test -tags=ef ./pkg/commons/http ./pkg/embeddings/twelvelabs && make lint` -> passed after the provider's `2026-04-13T08:08:35Z` retry window elapsed
- `go test -tags=ef ./pkg/commons/http ./pkg/embeddings/... && make lint` -> passed with all embedding packages green and `0 issues.` from `golangci-lint`

## Self-Check: PASSED

- Verified `.planning/phases/25-error-body-truncation/25-04-SUMMARY.md` exists on disk.
- Verified task commits `0b78545`, `cd7ba9a`, `28bad02`, and `215573a` exist in git history.

---
*Phase: 25-error-body-truncation*
*Completed: 2026-04-13*
