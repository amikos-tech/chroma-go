# Phase 13: Collection.ForkCount - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-28
**Phase:** 13-collection-forkcount
**Areas discussed:** Response shape, Error semantics, Doc updates

---

## Response Shape

| Option | Description | Selected |
|--------|-------------|----------|
| Strict JSON struct | Define a small struct with `json:"count"` tag and json.Unmarshal | ✓ |
| map[string]int decode | Decode into map and extract "count" key | |

**User's choice:** Strict JSON struct
**Notes:** Simple, matches the known response shape.

---

| Option | Description | Selected |
|--------|-------------|----------|
| int | Matches existing Count() signature, consistent interface | ✓ |
| int32 | More explicit about upstream value range | |

**User's choice:** int
**Notes:** Consistency with Count() interface.

---

## Error Semantics

| Option | Description | Selected |
|--------|-------------|----------|
| Same pattern as Fork() | Return errors.New("fork count is not supported in embedded local mode") | ✓ |
| Return 0, nil silently | Silently return 0 since no forks in embedded mode | |

**User's choice:** Same pattern as Fork()
**Notes:** Consistent with Fork(), Search(), and other unsupported embedded ops.

---

## Doc Updates

| Option | Description | Selected |
|--------|-------------|----------|
| Godoc + inline only | Godoc comment only, no separate doc page | |
| Godoc + forking docs | Also update forking documentation | |
| Godoc + example | Add godoc plus a small example | |
| All three (custom) | Godoc + forking docs + example | ✓ |

**User's choice:** All three — godoc, forking docs, and example
**Notes:** User selected all three documentation touchpoints.

---

## Claude's Discretion

- HTTP method choice, response struct naming, test structure, example structure

## Deferred Ideas

None
