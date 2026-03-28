# Phase 11: Fork Double-Close Bug - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-26
**Phase:** 11-fork-double-close-bug
**Areas discussed:** Ownership model, Scope of fix, Close semantics

---

## Ownership Model

| Option | Description | Selected |
|--------|-------------|----------|
| Owner flag | Add ownsEF bool. Fork sets false, constructors set true. Close() gates EF teardown. | |
| Reference counting | Atomic ref-count wrapper. EF stays alive until all holders release. | |
| Client-level dedup | Move EF close to client.Close() with pointer identity dedup. | |
| Close-once wrapper | Wrap shared EFs in sync.Once shim. Second close is no-op. | |
| Owner flag + close-once | Combo: owner flag prevents double-close, close-once wrapper prevents panic on manual close of original. | ✓ |

**User's choice:** Owner flag + close-once wrapper combo
**Notes:** User asked about edge case where original collection is manually closed while fork is live. Wanted defensive protection beyond just fixing client.Close() iteration. Reference counting was considered but deemed over-engineered. The combo approach provides correctness (owner flag) + safety (close-once wrapper) without ref-counting complexity.

---

## Scope of Fix

| Option | Description | Selected |
|--------|-------------|----------|
| HTTP only | Fix only CollectionImpl in client_http.go | |
| Both paths | Fix both HTTP and embedded client paths | ✓ |
| Shared abstraction | Extract common EF ownership logic | |

**User's choice:** Both paths
**Notes:** Embedded path has the same structural bug and arguably worse (zero sharing guard in embeddedCollection.Close()). Symmetric fix applied in parallel.

---

## Close Semantics

| Option | Description | Selected |
|--------|-------------|----------|
| Skip EF close only | Fork's Close() skips EF teardown but runs other cleanup | ✓ |
| Full no-op | Fork's Close() does nothing at all | |

**User's choice:** Skip EF close only
**Notes:** Not a full no-op — only EF close is gated by owner flag. If collections ever gain non-EF resources, forks would still clean those up.

---

## Claude's Discretion

- Close-once wrapper implementation details
- Test structure and naming
- Owner flag field naming

## Deferred Ideas

None
