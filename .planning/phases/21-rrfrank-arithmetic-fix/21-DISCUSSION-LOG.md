# Phase 21: RrfRank Arithmetic Fix - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-08
**Phase:** 21-RrfRank Arithmetic Fix
**Areas discussed:** Fix scope

---

## Fix Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Full parity | Fix all 10 methods (Multiply, Sub, Add, Div, Negate, Abs, Exp, Log, Max, Min) to build expression trees, matching KnnRank/ValRank pattern and Python SDK behavior | ✓ |
| Core arithmetic only | Fix only the 5 methods in success criteria (Multiply, Sub, Add, Div, Negate). Leave math functions as-is | |
| Error instead of no-op | Make methods return an error sentinel to explicitly reject RRF arithmetic | |

**User's choice:** Full parity
**Notes:** User initiated cross-SDK research before deciding. Python SDK confirms RRF inherits arithmetic from base Rank class — all methods build expression trees. Go SDK no-ops break parity.

---

## Pre-discussion Research

User asked whether Python/JS SDKs also have no-op RRF arithmetic. Research found:
- **Python:** RRF inherits arithmetic from base `Rank` class — builds real expression trees (Sum, Sub, Mul, Div)
- **JS:** Less conclusive, but same Rank interface pattern
- This confirmed the no-ops are a Go SDK bug, not intentional design

## Claude's Discretion

- Test structure and naming for RrfRank arithmetic tests

## Deferred Ideas

None
