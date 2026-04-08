# Phase 12: SDK Auto-Wiring Research - Context

**Gathered:** 2026-03-28
**Status:** Ready for planning

<domain>
## Phase Boundary

Trace contentEmbeddingFunction and embeddingFunction auto-wiring behavior in official Chroma SDKs (Python, JavaScript, Rust) to verify chroma-go's approach is consistent or document deliberate differences. Deliverable is a comparison document with recommendations — no code changes in this phase.

</domain>

<decisions>
## Implementation Decisions

### Research Methodology
- **D-01:** Source reading only — read latest stable release source of each SDK on GitHub. No live test setup, no docs cross-referencing.
- **D-02:** Target latest stable release tags for Python (`chromadb`), JavaScript (`chromadb`), and Rust SDKs.
- **D-03:** Include Rust SDK in addition to Python and JS (three comparison points total, plus chroma-go).

### Deliverable Format
- **D-04:** Comparison matrix format — markdown table with rows = operations (get/list/create), columns = SDKs. Includes sections for each traced behavior area.
- **D-05:** Document lives in phase directory as `12-RESEARCH.md` under `.planning/phases/12-sdk-auto-wiring-research/`.

### Scope of Tracing
- **D-06:** Full scope — trace four behavior areas across all SDKs:
  1. Auto-wiring behavior in get_collection, list_collections, create_collection
  2. Config persistence (how EF config is stored/retrieved on server)
  3. Close/cleanup lifecycle (how EF resources are disposed)
  4. Factory/registry patterns (how stored config maps back to EF instances)

### Action on Differences
- **D-07:** Document differences AND include a "Recommendations" section with proposed changes for downstream phases. No code implementation in this phase.
- **D-08:** If chroma-go does something extra that official SDKs don't, document it as a deliberate Go-specific enhancement. Only flag as "remove" if it causes actual bugs or confusion. Default posture: keep enhancements.

### Claude's Discretion
- Exact structure of comparison matrix subsections (may group by behavior area or by SDK)
- Level of code snippet inclusion in the research doc (enough to support claims, not full reproductions)
- Whether to include a summary table or just per-area tables

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Issue Context
- `GitHub issue #455` — Defines the research scope: trace auto-wiring in Python, JS, compare with chroma-go

### chroma-go Auto-Wiring Code
- `pkg/api/v2/client_http.go` — GetCollection auto-wiring (lines ~420-460), ListCollections auto-wiring (lines ~530-537)
- `pkg/api/v2/configuration.go` — BuildEmbeddingFunctionFromConfig, BuildContentEFFromConfig, config key constants
- `pkg/api/v2/client.go` — GetCollectionOp struct with embeddingFunction and contentEmbeddingFunction fields

### Prior Phase Context
- `.planning/phases/11-fork-double-close-bug/11-CONTEXT.md` — Close lifecycle decisions (ownsEF flag, close-once wrappers)
- `.planning/phases/03-registry-and-config-integration/03-CONTEXT.md` — Registry and config round-trip decisions

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `BuildEmbeddingFunctionFromConfig` and `BuildContentEFFromConfig` in `configuration.go` — the two factory functions that drive auto-wiring
- `wrapEFCloseOnce` / `wrapContentEFCloseOnce` — close-once wrappers from Phase 11
- Registry in `pkg/embeddings/` — factory registration patterns for both EF and contentEF

### Established Patterns
- GetCollection auto-wires both contentEF (first) and EF (fallback) from server config
- ListCollections auto-wires only EF, not contentEF (lighter weight)
- CreateCollection relies on user-provided EF only (no server config exists yet)
- Auto-wiring failures are logged as warnings, not errors — collection is still usable without EF

### Integration Points
- Config persistence uses `ConfigurationJSON` from server response → `NewCollectionConfigurationFromMap`
- Registry lookup uses `embedding_function.type` + `embedding_function.name` keys from stored config

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches for SDK source reading and comparison documentation.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 12-sdk-auto-wiring-research*
*Context gathered: 2026-03-28*
