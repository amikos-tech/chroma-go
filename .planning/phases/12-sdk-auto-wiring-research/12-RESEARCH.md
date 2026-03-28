# Phase 12: SDK Auto-Wiring Research - Research

**Researched:** 2026-03-28
**Domain:** Embedding function auto-wiring behavior across Chroma SDKs
**Confidence:** HIGH

## Summary

This research traces embedding function (EF) auto-wiring behavior across the Python SDK (`chromadb` 1.5.5), JavaScript SDK (`chromadb` in `clients/js` and `clients/new-js`), and the community Rust SDK (`chromadb-rs` by Anush008). All three plus chroma-go are compared across four behavior areas: auto-wiring in collection operations, config persistence, close/cleanup lifecycle, and factory/registry patterns.

The key finding is that chroma-go's auto-wiring approach is broadly consistent with the official SDKs but has some Go-specific enhancements (content EF auto-wiring, close-once lifecycle management) that are not present in Python or JS. These should be documented as deliberate enhancements per decision D-08.

**Primary recommendation:** Document chroma-go's content EF auto-wiring and close lifecycle as Go-specific enhancements. No removals needed -- behavior aligns with official SDK intent while providing additional safety.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Source reading only -- read latest stable release source of each SDK on GitHub. No live test setup, no docs cross-referencing.
- **D-02:** Target latest stable release tags for Python (`chromadb`), JavaScript (`chromadb`), and Rust SDKs.
- **D-03:** Include Rust SDK in addition to Python and JS (three comparison points total, plus chroma-go).
- **D-04:** Comparison matrix format -- markdown table with rows = operations, columns = SDKs.
- **D-05:** Document lives in phase directory as `12-RESEARCH.md`.
- **D-06:** Full scope -- trace four behavior areas across all SDKs.
- **D-07:** Document differences AND include a "Recommendations" section with proposed changes for downstream phases.
- **D-08:** If chroma-go does something extra that official SDKs don't, document it as a deliberate Go-specific enhancement. Only flag as "remove" if it causes actual bugs or confusion. Default posture: keep enhancements.

### Claude's Discretion
- Exact structure of comparison matrix subsections (may group by behavior area or by SDK)
- Level of code snippet inclusion in the research doc (enough to support claims, not full reproductions)
- Whether to include a summary table or just per-area tables

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

## Project Constraints (from CLAUDE.md)

- Keep things radically simple
- Use conventional commits
- Always lint before committing
- No pushing to main without PR
- No unnecessary comments -- code/names should be self-explanatory
- No scope creep -- do exactly what is asked

## Behavior Area 1: Auto-Wiring in Collection Operations

### Comparison Matrix

| Operation | Python SDK | JS SDK (old) | JS SDK (new-js) | Rust SDK (community) | chroma-go |
|-----------|-----------|-------------|-----------------|---------------------|-----------|
| **create_collection** | User-provided EF stored in config dict; sent to server; returned Collection holds user-provided EF | User-provided EF stored in config; conflict check with config.embedding_function; sent to server | Similar to old JS; also resolves from schema | No EF parameter on create | User-provided EF only; no auto-wiring (no server config exists yet) |
| **get_collection** | Fetches model from server; extracts `configuration_json.embedding_function`; validates conflict with user param; Collection holds user-provided EF (NOT auto-wired from config) | Fetches model; extracts config; conflict check; **prefers server config over param**: `configObj.embedding_function ?? embeddingFunction` | Similar fallback chain from config -> param -> default | No EF handling; returns bare collection | Auto-wires contentEF then EF from server config; derives dense EF from content EF if possible |
| **get_or_create_collection** | Same as create + get validation | Same as create flow with `get_or_create: true` | Same | Delegates to create with flag | Same as create (delegates) |
| **list_collections** | Returns Collection objects with NO embedding function | Returns collection **names only** (strings, not Collection objects) | Returns collections with EF resolution per item | Returns bare collections | Auto-wires EF (not contentEF) per collection from config |

### Key Differences

**Python `get_collection` does NOT auto-wire from config.** It validates that the user-provided EF doesn't conflict with persisted config, but the Collection always receives the user-provided EF parameter (which defaults to `DefaultEmbeddingFunction()`). The server config is used for validation, not instantiation.

**JS (old) `getCollection` DOES auto-wire from config.** Server config takes precedence: `configObj.embedding_function ?? embeddingFunction`. This means if the server has a stored EF config that can be deserialized, it is used; otherwise falls back to user parameter.

**chroma-go goes further** by auto-wiring both `contentEmbeddingFunction` AND `embeddingFunction` in `GetCollection`, with a derivation chain (content -> dense unwrap -> dense build). This is a Go-specific enhancement.

**`list_collections` behavior varies widely:**
- Python: Returns Collection objects but without EF
- JS (old): Returns just names (no Collection objects at all)
- JS (new): Returns collections with EF resolution
- Rust: Returns bare collections
- chroma-go: Returns Collection objects with auto-wired EF (but not contentEF)

### Confidence: HIGH
Source: Direct reading of GitHub source at tag 1.5.5 for Python and JS; master branch for Rust community SDK.

## Behavior Area 2: Config Persistence

### How EF Config is Stored/Retrieved on Server

| Aspect | Python SDK | JS SDK | Rust SDK | chroma-go |
|--------|-----------|--------|----------|-----------|
| **Storage format** | `configuration["embedding_function"] = ef_instance` (Python object stored in config dict, serialized by server layer) | `{ type: "known", name: "...", config: {...} }` via `serializeEmbeddingFunction()` | No config persistence | `{ type: "known", name: "...", config: {...} }` via `SetEmbeddingFunction()` |
| **Where stored** | In collection's `configuration_json` on server | In collection's `configuration_json` on server | N/A | In collection's `configuration_json` on server |
| **When stored** | At create time; EF param merged into config dict before server call | At create time; serialized via `serializeEmbeddingFunction` | Never | At create time; via `SetEmbeddingFunction` on configuration |
| **Retrieval** | `model.configuration_json.get("embedding_function")` returns raw dict | `loadCollectionConfigurationFromJson` deserializes from response | N/A | `GetEmbeddingFunctionInfo()` extracts from raw config map |

### Key Observations

The Python and JS SDKs both persist EF config at create time. The Python SDK stores the EF more opaquely (as a Python object reference in the config dict that the server serializes), while JS uses an explicit `{ type, name, config }` serialization format. chroma-go matches the JS serialization format with its `EmbeddingFunctionInfo` struct.

### Confidence: HIGH

## Behavior Area 3: Close/Cleanup Lifecycle

| Aspect | Python SDK | JS SDK | Rust SDK | chroma-go |
|--------|-----------|--------|----------|-----------|
| **Collection.close()** | None | None | None | Yes -- closes EF and contentEF |
| **Client.close()** | None (for collection EFs) | None | None | Yes -- iterates collection cache, closes all |
| **EF ownership tracking** | No | No | No | Yes -- `ownsEF` atomic bool |
| **Close-once safety** | N/A | N/A | N/A | Yes -- `sync.Once`-based wrappers |
| **Fork/share safety** | N/A | N/A | N/A | Yes -- forked collections share EF with close-once wrapper |

### Key Finding

**No other SDK manages EF lifecycle.** Python, JS, and Rust all rely on garbage collection / runtime cleanup. chroma-go is unique in having explicit close/cleanup because Go lacks finalizers and garbage-collected resource cleanup for external resources (HTTP clients, ONNX runtimes, etc.).

This is a clear Go-specific enhancement driven by Go's resource management idioms. Per D-08, document as deliberate enhancement.

### Confidence: HIGH

## Behavior Area 4: Factory/Registry Patterns

| Aspect | Python SDK | JS SDK | Rust SDK | chroma-go |
|--------|-----------|--------|----------|-----------|
| **Registry exists** | Yes -- `known_embedding_functions` dict | Yes -- `knownEmbeddingFunctions` registry | No | Yes -- dense, multimodal, content registries |
| **Registration pattern** | Name -> builder function mapping | Name -> builder function mapping | N/A | `RegisterDense`, `RegisterMultimodal`, `RegisterContent` |
| **Config round-trip** | `build_from_config(config)` from stored name + config | `build_from_config(efConfig.config)` from stored name + config | N/A | `BuildDense(name, config)`, `BuildMultimodal(name, config)`, `BuildContent(name, config)` |
| **Fallback chain** | Instance EF -> config EF -> schema EF -> default | Config EF -> param EF -> default | N/A | Content -> multimodal -> dense (in `BuildContent`) |
| **Content EF support** | No separate content EF concept | No separate content EF concept | N/A | Yes -- separate `ContentEmbeddingFunction` registry and factory |

### Key Finding

chroma-go has a richer registry system with three tiers (dense, multimodal, content) and a fallback chain in `BuildContent` that wraps lower-level EFs into content adapters. The Python and JS SDKs have flat registries with a single EF concept. This is consistent with chroma-go's multimodal foundations work (phases 1-11).

### Confidence: HIGH

## Summary Comparison Table

| Behavior | Python | JS (old) | JS (new) | Rust | chroma-go | Consistent? |
|----------|--------|----------|----------|------|-----------|-------------|
| Auto-wire on get | Validate only, use param | Yes, config preferred | Yes, config preferred | No | Yes, config preferred | chroma-go matches JS |
| Auto-wire on create | No (expected) | No (expected) | No (expected) | No | No | All consistent |
| Auto-wire on list | No EF on collections | Returns names only | Yes, per item | No | Yes, EF only | chroma-go matches new-js |
| ContentEF auto-wire | No concept | No concept | No concept | No | Yes (get only) | Go-specific enhancement |
| Config persistence | At create time | At create time | At create time | No | At create time | Consistent |
| Config format | `{type, name, config}` | `{type, name, config}` | `{type, name, config}` | N/A | `{type, name, config}` | Consistent |
| Close lifecycle | None | None | None | None | Full (ownsEF, close-once) | Go-specific enhancement |
| Registry/factory | Single tier | Single tier | Single tier | None | Three tiers | Go-specific enhancement |
| EF derivation chain | Instance->config->schema | Config->param->default | Config->param->schema->default | None | Content->unwrap->dense | Go-specific enhancement |

## Recommendations

### No Changes Needed (Consistent Behavior)
1. **create_collection**: All SDKs use user-provided EF only. chroma-go is consistent.
2. **Config persistence format**: `{type, name, config}` matches JS exactly.
3. **Config round-trip via registry**: All SDKs that support it use the same pattern.

### Document as Go-Specific Enhancements (D-08)
1. **ContentEF auto-wiring on GetCollection**: No other SDK has the concept of `ContentEmbeddingFunction`. This is part of chroma-go's multimodal foundation and should be documented as an enhancement.
2. **Close/cleanup lifecycle**: Go-specific need due to explicit resource management. No other SDK does this.
3. **Three-tier registry (dense/multimodal/content)**: Richer than official SDKs' single-tier registries. Supports the multimodal contract.
4. **EF derivation chain in GetCollection**: Content -> unwrap dense -> build dense fallback. More sophisticated than other SDKs.

### Potential Improvements for Downstream Phases
1. **list_collections contentEF gap**: chroma-go auto-wires dense EF but not contentEF on ListCollections. The new-js client resolves EF per item including from schema. Consider whether contentEF should also be wired for ListCollections, or if the current "lighter weight" approach is better. **Recommendation**: Keep as-is (lighter weight is intentional per existing design).
2. **Conflict validation on get**: Python SDK validates that user-provided EF doesn't conflict with persisted config. chroma-go silently prefers auto-wired config. Consider adding a warning log when user provides an EF that differs from stored config. **Recommendation**: Low priority, consider for future phase.
3. **Default EF behavior**: Python defaults to `DefaultEmbeddingFunction()` when no EF provided. chroma-go returns nil (no EF). This is a deliberate difference -- Go callers must opt in. **Recommendation**: Keep as-is (explicit is better in Go).

## Common Pitfalls

### Pitfall 1: Assuming Python Auto-Wires on Get
**What goes wrong:** Assuming Python `get_collection` reconstructs EF from server config like JS does.
**Why it happens:** Python validates against persisted config but always uses the user-provided parameter.
**How to avoid:** Read the actual source -- Python's `get_collection` returns Collection with `embedding_function=embedding_function` (the parameter), not the deserialized config.

### Pitfall 2: Confusing Old JS and New JS Behavior
**What goes wrong:** Treating old JS (`clients/js`) and new JS (`clients/new-js`) as identical.
**Why it happens:** Both exist in the same repo at the same tag.
**How to avoid:** Document both; the new-js client has richer schema-based EF resolution.

### Pitfall 3: Rust SDK as Official Reference
**What goes wrong:** Treating the community Rust SDK (Anush008/chromadb-rs) as an official reference for expected behavior.
**Why it happens:** It appears in crates.io as `chromadb`.
**How to avoid:** Note it's community-maintained with minimal EF support; use Python and JS as primary comparison points.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | go test |
| Config file | Makefile targets with build tags |
| Quick run command | `go test -tags=basicv2 -run TestAutoWire ./test/client_v2/... -count=1` |
| Full suite command | `make test` |

### Phase Requirements to Test Map
This is a research-only phase (D-07 specifies no code changes). No new tests are needed. The deliverable is this document itself.

### Wave 0 Gaps
None -- this is a documentation-only phase with no code implementation.

## Sources

### Primary (HIGH confidence)
- `chromadb/api/client.py` at tag 1.5.5 -- Python get_collection, create_collection, list_collections implementations
- `chromadb/api/async_client.py` at tag 1.5.5 -- Python async equivalents (same patterns)
- `chromadb/api/collection_configuration.py` at tag 1.5.5 -- Python EF validation and serialization
- `chromadb/api/models/CollectionCommon.py` at tag 1.5.5 -- Python Collection EF lifecycle
- `clients/js/packages/chromadb-core/src/ChromaClient.ts` at tag 1.5.5 -- JS old client
- `clients/js/packages/chromadb-core/src/CollectionConfiguration.ts` at tag 1.5.5 -- JS old config handling
- `clients/js/packages/chromadb-core/src/Collection.ts` at tag 1.5.5 -- JS Collection lifecycle
- `clients/new-js/packages/chromadb/src/chroma-client.ts` at tag 1.5.5 -- JS new client
- `clients/new-js/packages/chromadb/src/collection-configuration.ts` at tag 1.5.5 -- JS new config
- `Anush008/chromadb-rs` master branch -- Rust community SDK
- `pkg/api/v2/client_http.go` -- chroma-go GetCollection, ListCollections, CreateCollection
- `pkg/api/v2/configuration.go` -- chroma-go BuildEmbeddingFunctionFromConfig, BuildContentEFFromConfig

### Secondary (MEDIUM confidence)
- GitHub issue #455 -- original research scope definition

## Metadata

**Confidence breakdown:**
- Auto-wiring behavior: HIGH -- direct source reading of all four SDKs
- Config persistence: HIGH -- direct source reading
- Close lifecycle: HIGH -- direct source reading (absence is easy to verify)
- Registry patterns: HIGH -- direct source reading

**Research date:** 2026-03-28
**Valid until:** 2026-04-28 (stable; SDK patterns change slowly)
