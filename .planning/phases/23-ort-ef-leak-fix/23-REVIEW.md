---
phase: 23-ort-ef-leak-fix
reviewed: 2026-04-11T09:48:23Z
depth: standard
files_reviewed: 3
files_reviewed_list:
  - pkg/api/v2/client.go
  - pkg/api/v2/client_local_embedded.go
  - pkg/api/v2/client_local_embedded_test.go
findings:
  critical: 0
  warning: 0
  info: 0
  total: 0
status: clean
---

# Phase 23: Code Review Report

## Summary

Reviewed the Phase 23 changes in `pkg/api/v2/client.go`, `pkg/api/v2/client_local_embedded.go`, and `pkg/api/v2/client_local_embedded_test.go` covering the default ORT EF leak fix for embedded idempotent create calls.

**Verdict: clean.** The implementation follows the narrow phase boundary, avoids the rejected package-global seam, preserves existing embedded collection state precedence, and adds the expected regression coverage for cleanup success, cleanup failure, and new-collection ownership.

## Checks Performed

- Confirmed `CreateCollectionOp` uses a per-op `defaultDenseEFFactory` seam and `sdkOwnedDefaultDenseEF` pointer tracking rather than a boolean-only provenance marker.
- Confirmed `PrepareAndValidateCollectionRequest()` clears `sdkOwnedDefaultDenseEF` up front and nils it again when dual-interface content EF promotion replaces the temporary default.
- Confirmed embedded `CreateCollection()` only closes the tracked SDK-owned default EF when the live runtime dense EF is still that exact instance.
- Confirmed cleanup failures return the exact wrapped error `error closing default embedding function for existing collection`.
- Confirmed the three focused `basicv2` regressions exercise the required cleanup, error, and ownership-transfer paths without widening the phase scope.

## Notes

- Targeted verification passed locally:

```bash
go test -tags=basicv2 -run 'TestEmbeddedLocalClientCreateCollection_IfNotExistsExistingDoesNotOverrideState|TestEmbeddedCreateCollection_DefaultORT.*' ./pkg/api/v2/...
```

---

_Reviewed: 2026-04-11T09:48:23Z_
_Reviewer: Codex manual review_
_Depth: standard_
