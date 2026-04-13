---
phase: 25
fixed_at: 2026-04-13T13:48:57Z
review_path: .planning/phases/25-error-body-truncation/25-REVIEW.md
iteration: 1
findings_in_scope: 3
fixed: 3
skipped: 0
status: all_fixed
---

# Phase 25: Code Review Fix Report

**Fixed at:** 2026-04-13T13:48:57Z
**Source review:** `.planning/phases/25-error-body-truncation/25-REVIEW.md`
**Iteration:** 1

**Summary:**
- Findings in scope: 3
- Fixed: 3
- Skipped: 0

## Fixed Issues

### WR-01: Shared sanitizer still scales with full response size

**Status:** fixed
**Files modified:** `pkg/commons/http/utils.go`, `pkg/commons/http/utils_test.go`
**Commit:** `5507a01`
**Applied fix:** Replaced the full-string/full-rune sanitization path with byte trimming plus incremental UTF-8 scanning, capped builder growth to the display limit, and simplified panic recovery to a constant fallback instead of retrying the expensive conversion path.

### WR-02: Cloudflare structured errors bypass the truncation contract

**Status:** fixed
**Files modified:** `pkg/embeddings/cloudflare/cloudflare.go`, `pkg/embeddings/cloudflare/cloudflare_error_test.go`
**Commits:** `29966e4`, `34d3c9e`
**Applied fix:** Sanitized the marshaled structured `errors` payload before formatting it into the returned error string and tightened the regression coverage so a unique late-message marker cannot leak through the structured segment.

### WR-03: Cohere default model changed in a non-compatibility phase

**Status:** fixed
**Files modified:** `pkg/embeddings/cohere/cohere.go`, `pkg/embeddings/cohere/cohere_config_test.go`
**Commit:** `1ad7696`
**Applied fix:** Restored the runtime default model to `embed-english-v2.0` and added unit coverage that locks both the instantiated client default and serialized config back to the legacy value.

## Verification

- `go test ./pkg/commons/http ./pkg/embeddings/cohere`
- `go test -tags=ef ./pkg/embeddings/cloudflare`

---

_Fixed: 2026-04-13T13:48:57Z_
_Fixer: Codex + gsd-code-fixer_
_Iteration: 1_
