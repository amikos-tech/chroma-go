# Phase 10: Code Cleanups - Context

**Gathered:** 2026-03-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Consolidate duplicated path safety utilities into a shared internal package, fix the *context.Context pointer-to-interface anti-pattern across embedding providers, add registry test cleanup to prevent global state leaks, and fix resolveMIME for URL-backed sources. Four cleanup items tied to issues #456, #461, #466, #469.

</domain>

<decisions>
## Implementation Decisions

### Shared Path Utility Scope
- **D-01:** `pkg/internal/pathutil` contains only path safety functions: `ContainsDotDot`, `ValidateFilePath`, `SafePath`. No file-reading helpers like `resolveBytes` — those stay provider-local since they interleave provider-specific MIME logic.
- **D-02:** Gemini, Voyage, and default_ef replace their local path safety implementations with imports from the shared package.

### DefaultContext Type Change
- **D-03:** Change `*context.Context` to `context.Context` directly in Gemini, Nomic, and Mistral provider structs. No deprecation or compatibility shim — the compiler tells callers exactly what to fix.
- **D-04:** Update all internal usages that take address (`&ctx`) or dereference (`*c.DefaultContext`) to use the value type directly.

### Registry Test Cleanup
- **D-05:** Add unexported unregister helpers (e.g., `unregisterDense(name)`) for use in test cleanup only. Public registry API stays append-only.
- **D-06:** All registry tests that register providers must use `t.Cleanup` with unregister helpers to prevent global state leaks between test runs.

### resolveMIME URL Support
- **D-07:** Gemini and VoyageAI `resolveMIME` functions gain URL path extension extraction (parse URL, get `path.Ext` from URL path) as a fallback when `MIMEType` and `FilePath` are both empty.
- **D-08:** Query strings and fragments in URLs are stripped before extension extraction.

### Claude's Discretion
- Exact function signatures and internal implementation details of the shared pathutil package
- How to handle edge cases in URL MIME resolution (e.g., URLs with no extension)
- Whether to reorder fallback chain in resolveMIME (MIMEType → FilePath → URL → error)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Issues
- GitHub issue #456 — Duplicated path safety utilities consolidation
- GitHub issue #461 — *context.Context pointer-to-interface anti-pattern
- GitHub issue #466 — Registry test cleanup / global state leaks
- GitHub issue #469 — resolveMIME for URL-backed sources

### Key Source Files
- `pkg/embeddings/gemini/content.go` — containsDotDot (L135-138), resolveMIME (L140-156), resolveBytes (L112)
- `pkg/embeddings/voyage/content.go` — containsDotDot (L143-146), resolveMIME (L148-163), resolveBytes (L120)
- `pkg/embeddings/default_ef/download_utils.go` — safePath (L155-164)
- `pkg/embeddings/gemini/gemini.go` — DefaultContext *context.Context (L52)
- `pkg/embeddings/nomic/nomic.go` — DefaultContext *context.Context (L60)
- `pkg/embeddings/mistral/mistral.go` — DefaultContext *context.Context (L36)
- `pkg/embeddings/registry_test.go` — Registry tests with partial cleanup (L510-514, L538-542)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `containsDotDot()` — identical implementation in Gemini and Voyage, ready to extract
- `safePath()` in default_ef — different function (tar extraction safety), also extractable
- `extToMIME` maps in Gemini and Voyage — provider-specific MIME mappings, stay local

### Established Patterns
- Functional options pattern for provider construction (WithContext, WithAPIKey, etc.)
- Build-tagged test files with testify assertions
- Registry uses package-level maps with sync.RWMutex protection
- Two registry tests already demonstrate the t.Cleanup + delete pattern (L510-514, L538-542)

### Integration Points
- `pkg/internal/pathutil` is a new package — no existing internal packages to conflict with
- DefaultContext field change affects struct construction in option functions and tests
- Registry unregister helpers must acquire the same mutex used by register functions

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 10-code-cleanups*
*Context gathered: 2026-03-26*
