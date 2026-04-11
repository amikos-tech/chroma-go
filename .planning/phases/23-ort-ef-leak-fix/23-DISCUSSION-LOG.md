# Phase 23: ORT EF Leak Fix - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-10
**Phase:** 23-ort-ef-leak-fix
**Areas discussed:** Fix shape, Verification style, Cleanup failure contract

---

## Fix shape

| Option | Description | Selected |
|--------|-------------|----------|
| Branch-local cleanup in embedded `CreateCollection` with an internal auto-created-default marker (Recommended) | Keep the fix local to the embedded existing-collection path. Detect that the default ORT EF was SDK-created for this request, close it, then discard the temporary override. | ✓ |
| Defer default EF creation until after the existence check | Avoid creating the default ORT EF on the existing-collection path at all by changing shared request preparation timing. | |
| Broader lifecycle refactor across prepare/build/create | Redesign EF ownership and creation flow more generally across the request-prep and embedded collection lifecycle. | |

**User's choice:** Recommended option
**Notes:** This keeps Phase 23 aligned with the existing Phase 20 behavior where existing embedded collections preserve their original state-backed EF/contentEF and do not adopt new overrides.

---

## Verification style

| Option | Description | Selected |
|--------|-------------|----------|
| Narrow close-spy/unit regression with a default-EF factory seam | Add a focused deterministic regression proving the temporary default ORT EF is closed exactly once on the existing-collection path. | |
| Real ORT integration or leak-detection proof | Use a heavier ORT-native or leak-detection style test to prove end-to-end teardown of runtime resources. | |
| Mixed: focused seam test plus broader existing-collection lifecycle regression (Recommended) | Pair the focused close regression with a broader behavioral regression proving the existing collection still preserves its original EF/state and does not adopt the temporary default. | ✓ |

**User's choice:** Recommended option
**Notes:** The mixed option was preferred because it fits the repo’s existing `basicv2` lifecycle-test style without expanding Phase 23 into a brittle ORT-specific integration harness.

---

## Cleanup failure contract

| Option | Description | Selected |
|--------|-------------|----------|
| Return an error if SDK-owned default ORT EF cleanup fails (Recommended) | Treat this as a synchronous request-path failure: if the temporary SDK-created default EF cannot be closed, `CreateCollection` must return an error. | ✓ |
| Log cleanup failure but still return the existing collection | Preserve idempotent success semantics and rely on logging/stderr for visibility if cleanup fails. | |
| Conditional or error-class-based behavior | Return success or failure depending on the cleanup error type or whether the error seems benign. | |

**User's choice:** Recommended option
**Notes:** The stricter contract was preferred because log-only handling in this repo is mostly reserved for asynchronous cache/state cleanup after ownership has already moved elsewhere, not for synchronous request-path cleanup of a temporary SDK-owned resource.

---

## Research notes that informed the decision

- The leak is caused by `PrepareAndValidateCollectionRequest` eagerly creating a default ORT EF before embedded `CreateCollection` discovers the collection already exists.
- Existing embedded collection behavior is already locked by Phase 20: on the existing-collection path, new override EFs are ignored and the state-backed EF remains authoritative.
- The repo already separates synchronous setup/cleanup failures from asynchronous background cleanup failures:
  - setup/promotion cleanup failures are returned
  - cache/state shutdown cleanup failures are logged

## Deferred Ideas

- Refactor shared create/request-prep flow to delay default EF creation until after the existence check
- Fold Phase 23 and Phase 24 into a unified embedded EF lifecycle redesign
- Add an ORT-native leak harness or env-gated integration suite for this path
- Introduce typed cleanup-error classes for conditional behavior
