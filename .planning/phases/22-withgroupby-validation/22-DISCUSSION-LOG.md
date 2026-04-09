# Phase 22: WithGroupBy Validation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-09
**Phase:** 22-WithGroupBy Validation
**Areas discussed:** Nil contract, Error message strictness, Docs surface

---

## Nil contract

| Option | Description | Selected |
|--------|-------------|----------|
| Fail fast with clear error *(Recommended)* | Treat explicit `WithGroupBy(nil)` as invalid and return an error before request construction completes. Omit the option entirely to skip grouping. | ✓ |
| Backward-compatible no-op | Keep current behavior where `WithGroupBy(nil)` silently does nothing. | |
| Normalize nil internally | Accept nil and silently translate it into "no grouping" or an empty internal value. | |

**User's choice:** Fail fast with clear error
**Notes:** User explicitly preferred the better DX: fail fast, keep invariants enforced immediately, and avoid silent behavior.

---

## Error message strictness

| Option | Description | Selected |
|--------|-------------|----------|
| Stable exact nil-specific message *(Recommended)* | Choose one clear nil-validation message and assert it exactly in tests so the contract stays stable. | ✓ |
| Any clear validation error | Only require that an error occurs and roughly mentions nil/invalid input. | |
| Bubble generic validation error | Let implementation details determine message shape without locking exact text. | |

**User's choice:** Stable exact nil-specific message
**Notes:** User did not require a specific string in discussion, but did require message stability for tests and caller expectations.

---

## Docs surface

| Option | Description | Selected |
|--------|-------------|----------|
| Code+tests only *(Recommended)* | Keep this phase bounded to implementation and tests; no docs/examples updates. | ✓ |
| Add short docs note | Update group-by docs with a one-line nil-invalid note. | |
| Full docs sweep | Review all search/group-by docs and examples for nil-contract messaging. | |

**User's choice:** Code+tests only
**Notes:** User kept scope tightly bounded to the bug fix and regression coverage.

---

## the agent's Discretion

- Exact nil-specific message wording, as long as it stays stable and repo-consistent
- Whether the stable error is represented by a direct error or a dedicated exported sentinel
- Exact test naming and organization in `groupby_test.go`

## Deferred Ideas

1. Docs/examples note for invalid nil usage — deferred to keep Phase 22 code+tests only
2. Broader nil-handling consistency audit across other option setters — potential future cleanup phase

---

*Discussion log generated: 2026-04-09*
