# Phase 19: Embedded Client EF Lifecycle Hardening - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-06
**Phase:** 19-embedded-client-ef-lifecycle-hardening
**Areas discussed:** TOCTOU race strategy, Close-once wrapping, Structured logger, Cleanup on delete/close, Symmetric unwrapping, Build error guard, Test strategy

---

## TOCTOU Race Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Check-and-set under write lock | Upgrade to full write lock: read state, check if EF is nil, build EF, assign — all under one Lock(). Prevents two concurrent GetCollection calls from both auto-wiring. | ✓ |
| Compare-and-swap with atomic | Use atomic.CompareAndSwapPointer pattern. Lock-free but requires unsafe.Pointer casting. | |
| Keep current idempotent approach | Auto-wiring is idempotent — building the same EF twice is harmless. Keep RLock for reads. | |

**User's choice:** Check-and-set under write lock
**Notes:** None

### Follow-up: Lock Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Wide lock: hold Lock() for entire build+assign | Simpler code, holds write lock during EF build. | ✓ |
| Narrow lock: double-checked locking pattern | Check-nil under RLock, build outside lock, Lock() for check-nil-again + assign. | |

**User's choice:** Wide lock — simple and safe
**Notes:** User clarified that concurrent GetCollection calls are not a real-world scenario because every collection operation requires a prior GetCollection call, making usage inherently sequential. Wide lock blocking concern is theoretical only.

---

## Close-once Wrapping

| Option | Description | Selected |
|--------|-------------|----------|
| Mirror HTTP: add close-once wrapping | Wrap both EFs in close-once wrappers in buildEmbeddedCollection, matching HTTP client pattern. | ✓ |
| Keep ownsEF + closeOnce pattern | Current approach uses ownsEF atomic.Bool + sync.Once. Already prevents double-close at collection level. | |
| Close-once for contentEF only | Wrap only contentEF; keep denseEF with current ownsEF pattern. | |

**User's choice:** Mirror HTTP: add close-once wrapping
**Notes:** None

---

## Structured Logger

| Option | Description | Selected |
|--------|-------------|----------|
| Optional injected logger | Add WithLogger option using existing pkg/logger interface. Structured when set, stderr fallback when unset. | ✓ |
| Keep stderr-only | Stderr logging is simple. Auto-wire errors are rare edge cases. | |
| Use slog from stdlib | Go 1.21+ slog. No external dependency but different contract from pkg/logger. | |

**User's choice:** Optional injected logger
**Notes:** None

---

## Cleanup on Delete/Close

| Option | Description | Selected |
|--------|-------------|----------|
| Full cleanup on both paths | Close(): iterate all collectionState entries and close EFs. Delete: close EFs first, then remove map entry. | ✓ |
| Cleanup on Close() only | Close() iterates and closes. Delete just removes entry — EF will be GC'd. | |
| Rely on GC for cleanup | No explicit cleanup. Risks resource leaks for EFs with open connections. | |

**User's choice:** Full cleanup on both paths
**Notes:** None

### Follow-up: localDeleteCollectionFromCache

| Option | Description | Selected |
|--------|-------------|----------|
| Type switch for both types | Add *embeddedCollection case to type switch. Close EFs via sharing detection. | ✓ |
| Unified Closer interface | Define a common interface both types implement. More abstract. | |

**User's choice:** Type switch for both types
**Notes:** None

---

## Symmetric Unwrapping

| Option | Description | Selected |
|--------|-------------|----------|
| Unwrap both sides | Unwrap both denseEF and contentEF before comparing. Ensures identity check works with close-once wrappers on both sides. | ✓ |
| Keep current asymmetric approach | Current logic works because contentEF's UnwrapEmbeddingFunction() peels through. | |

**User's choice:** Unwrap both sides
**Notes:** None

---

## Build Error Guard

| Option | Description | Selected |
|--------|-------------|----------|
| Guard: only assign on nil error | if buildErr == nil { contentEF = autoWiredContentEF }. Explicit. | ✓ |
| Assign regardless (current) | BuildContentEFFromConfig returns nil on error, so assigning nil is harmless. | |

**User's choice:** Guard: only assign on nil error
**Notes:** None

---

## Test Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Unit tests per fix | Focused unit tests for each fix with mocks. Fast, isolated. | |
| Integration tests with testcontainers | Test against real embedded Chroma runtime. Slower but full lifecycle. | |
| Both unit and integration | Unit tests for each fix plus integration test for full lifecycle. | ✓ |

**User's choice:** Both unit and integration
**Notes:** None

---

## Claude's Discretion

- Internal helper decomposition and method ordering
- Exact log message format and severity levels
- Whether to refactor shared cleanup logic into a common helper

## Deferred Ideas

None — discussion stayed within phase scope
