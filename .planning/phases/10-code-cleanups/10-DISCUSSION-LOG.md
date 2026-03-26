# Phase 10: Code Cleanups - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-26
**Phase:** 10-code-cleanups
**Areas discussed:** Shared path utility scope, DefaultContext breaking change strategy, Registry unregister helpers

---

## Shared Path Utility Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Strict — only path safety | `ContainsDotDot`, `ValidateFilePath`, `SafePath` only. Leave `resolveBytes` provider-local. | ✓ |
| Include resolveBytes | Also extract file-reading + path-check logic into shared helper. | |
| You decide | Claude picks the right boundary based on code analysis. | |

**User's choice:** Strict — only path safety
**Notes:** `resolveBytes` stays provider-local because it interleaves provider-specific MIME logic.

---

## DefaultContext Breaking Change Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Just change the type | `*context.Context` → `context.Context` directly. Compiler catches breakage. | ✓ |
| Deprecate + add WithContext | Keep field but add option function, then change type. Softens migration. | |
| You decide | Claude picks based on actual construction patterns. | |

**User's choice:** Just change the type
**Notes:** Clean break. Go compiler tells callers exactly what to fix.

---

## Registry Unregister Helpers

| Option | Description | Selected |
|--------|-------------|----------|
| Test-only (unexported) | Unexported helpers called via `t.Cleanup`. Public API stays append-only. | ✓ |
| Public API | Export `UnregisterDenseEmbeddingFunction(name)` etc. for dynamic management. | |
| You decide | Claude picks based on whether real use case exists beyond tests. | |

**User's choice:** Test-only (unexported)
**Notes:** Keeps public registry surface clean.

---

## Claude's Discretion

- Exact function signatures for pathutil package
- URL MIME resolution edge cases
- resolveMIME fallback chain ordering

## Deferred Ideas

None
