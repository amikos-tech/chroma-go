# Phase 3: Registry and Config Integration - Research

**Researched:** 2026-03-20
**Domain:** Go embedding registry extension, config persistence, collection auto-wiring
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Registry extension**
- Add a dedicated 4th factory map (`contentFactories`) with `RegisterContent`, `BuildContent`, `ListContent`, `HasContent`, `BuildContentCloseable` — full parity with the existing dense/sparse/multimodal pattern
- `BuildContent` uses a fallback chain: content factory (native) → multimodal factory + `AdaptMultimodalEmbeddingFunctionToContent` → dense factory + `AdaptEmbeddingFunctionToContent` → error
- Existing providers get content support automatically through the adapter fallback — no changes to existing provider `init()` functions
- When auto-adapting, if the built provider implements `CapabilityAware`, use its actual metadata; otherwise infer minimal capabilities from the interface type:
  - Dense: `{Modalities: [text], SupportsBatch: true}`
  - Multimodal: `{Modalities: [text, image], SupportsBatch: true}`
- New content-native providers (Gemini Phase 6, vLLM Phase 7) will register directly in the content map later

**Config build chain**
- `BuildEmbeddingFunctionFromConfig` keeps its existing return type (`EmbeddingFunction`) but gains a multimodal fallback: dense → multimodal → schema (existing)
- Add a new `BuildContentEFFromConfig` function returning `ContentEmbeddingFunction` with fallback chain: content → multimodal+adapt → dense+adapt
- Unknown provider names return `nil, nil` (matching existing auto-wiring contract where missing providers are silently skipped)

**Collection auto-wiring**
- Collections gain an optional `contentEF` field alongside the existing `ef` field
- Auto-wiring populates both: `BuildEmbeddingFunctionFromConfig` for `ef`, `BuildContentEFFromConfig` for `contentEF`
- Add `WithContentEmbeddingFunction(ContentEmbeddingFunction)` as a new collection option alongside existing `WithEmbeddingFunction`
- If both are set, content takes priority; dense EF is derived from the content one
- Add `SetContentEmbeddingFunction` on `CollectionConfigurationImpl` for config persistence

**Config round-trip**
- Config persistence shape remains unchanged: `{type, name, config}` — no capability metadata stored
- Content-native providers implement both `EmbeddingFunction` and `ContentEmbeddingFunction` (same pattern as `MultimodalEmbeddingFunction` embedding `EmbeddingFunction`)
- `SetContentEmbeddingFunction` type-asserts to `EmbeddingFunction` to extract `Name()`/`GetConfig()` for persistence; if assertion fails, skip persistence (runtime-only content EF)
- Capability metadata is purely a runtime concern — discovered at build time from the provider impl, not persisted

### Claude's Discretion
- Internal helper names for the adaptation logic in the build chain
- Exact placement of the auto-wiring content EF population within the HTTP client flow
- Whether `BuildContentCloseable` wraps the adapted closer or chains closers
- Test scaffolding organization and specific assertion patterns

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| REG-01 | Factory and registry code can build richer multimodal embedding functions from stored config using additive shared interfaces | Addressed by the 4th content factory map in `registry.go` + `BuildContentEFFromConfig` in `configuration.go`; the fallback chain (content → multimodal+adapt → dense+adapt) enables existing stored configs to yield `ContentEmbeddingFunction` without any migration |
| REG-02 | Collection configuration auto-wiring keeps working for existing dense and multimodal providers after the richer interfaces are introduced | Addressed by leaving `BuildEmbeddingFunctionFromConfig` / the dense `ef` path entirely intact; the content EF path is additive alongside it; existing providers auto-adapt via the fallback without touching their `init()` registration |
</phase_requirements>

---

## Summary

Phase 3 extends the embedding registry and configuration layer from three factory maps (dense, sparse, multimodal) to four, adding a content map for `ContentEmbeddingFunction`. All design decisions are locked in CONTEXT.md — no alternative approaches need evaluation.

The codebase is fully readable and consistent. The three existing factory maps in `pkg/embeddings/registry.go` share one `sync.RWMutex` and follow an identical structural pattern: a private map variable, a `Register*` function, a `Build*` function, a `Build*Closeable` function, a `List*` function, and a `Has*` function. The 4th map must clone this pattern exactly.

Config persistence lives in `pkg/api/v2/configuration.go`. `BuildEmbeddingFunctionFromConfig` currently tries dense only (then schema), and must gain a multimodal fallback. A parallel `BuildContentEFFromConfig` function is new. Collection auto-wiring in `client_http.go:421-441` calls `BuildEmbeddingFunctionFromConfig` after a `GetCollection` HTTP response; Phase 3 adds a second call for content EF alongside it. The `CollectionImpl` struct in `collection_http.go` currently holds a single `embeddingFunction embeddings.EmbeddingFunction` field; Phase 3 adds a parallel `contentEmbeddingFunction embeddings.ContentEmbeddingFunction` field.

**Primary recommendation:** Implement changes in strict file order — registry.go first (foundation), then configuration.go (build chain), then collection_http.go (struct + options + auto-wiring) — so each plan has no forward dependencies.

---

## Standard Stack

### Core (all verified from source)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/pkg/errors` | in-use | Error wrapping throughout `registry.go`, `configuration.go` | Already used in all files under extension |
| `sync.RWMutex` | stdlib | Concurrent registry access | Current pattern across all 3 factory maps |
| `github.com/stretchr/testify` | in-use | `assert`/`require` in all embedding tests | Project-wide test standard |

No new dependencies are needed. Phase 3 is a pure Go extension of existing interfaces and patterns.

**Installation:** None required.

---

## Architecture Patterns

### Recommended Project Structure

No new directories needed. All changes land in existing files:

```
pkg/embeddings/
├── registry.go               # Add 4th factory map + 6 new functions
├── embedding.go              # No changes needed (ContentEmbeddingFunction already defined)
├── multimodal_compat.go      # No changes needed (adapters already exist)
pkg/api/v2/
├── configuration.go          # Extend BuildEmbeddingFunctionFromConfig + add BuildContentEFFromConfig + SetContentEmbeddingFunction
├── collection_http.go        # Add contentEF field, WithContentEmbeddingFunction option, auto-wiring call
pkg/embeddings/               # Test files (new)
├── registry_test.go          # Extend with content map tests
pkg/embeddings/ (or pkg/api/v2/)
└── (new) content_config_test.go  # Round-trip and auto-wiring tests
```

### Pattern 1: 4th Factory Map (content map)

**What:** A `contentFactories` map alongside the three existing maps, sharing the same `mu sync.RWMutex`. Mirrors the dense/sparse/multimodal pattern exactly.

**When to use:** Any code that needs to build a `ContentEmbeddingFunction` from a name+config.

**Example — factory type and map variable (mirrors `multimodalFactories`):**
```go
// Source: pkg/embeddings/registry.go (existing multimodal pattern)

// ContentEmbeddingFunctionFactory creates a ContentEmbeddingFunction from config.
type ContentEmbeddingFunctionFactory func(config EmbeddingFunctionConfig) (ContentEmbeddingFunction, error)

var contentFactories = make(map[string]ContentEmbeddingFunctionFactory)
// Note: shares the existing mu sync.RWMutex — no new mutex needed
```

**Example — BuildContent with fallback chain:**
```go
// Source: design decision from 03-CONTEXT.md + adapter functions from multimodal_compat.go

func BuildContent(name string, config EmbeddingFunctionConfig) (ContentEmbeddingFunction, error) {
    // Step 1: native content factory
    mu.RLock()
    factory, ok := contentFactories[name]
    mu.RUnlock()
    if ok {
        return factory(config)
    }

    // Step 2: multimodal factory + adapt
    mu.RLock()
    mmFactory, ok := multimodalFactories[name]
    mu.RUnlock()
    if ok {
        mmEF, err := mmFactory(config)
        if err != nil {
            return nil, err
        }
        caps := inferCaps(mmEF) // CapabilityAware check or default {text, image}
        return AdaptMultimodalEmbeddingFunctionToContent(mmEF, caps), nil
    }

    // Step 3: dense factory + adapt
    mu.RLock()
    denseFactory, ok := denseFactories[name]
    mu.RUnlock()
    if ok {
        ef, err := denseFactory(config)
        if err != nil {
            return nil, err
        }
        caps := inferCaps(ef) // CapabilityAware check or default {text}
        return AdaptEmbeddingFunctionToContent(ef, caps), nil
    }

    return nil, errors.Errorf("unknown content embedding function: %s", name)
}

// inferCaps reads CapabilityAware if implemented, otherwise falls back to
// minimal inferred metadata based on interface type.
func inferCaps(ef interface{}) CapabilityMetadata {
    if ca, ok := ef.(CapabilityAware); ok {
        return ca.Capabilities()
    }
    if _, ok := ef.(MultimodalEmbeddingFunction); ok {
        return CapabilityMetadata{Modalities: []Modality{ModalityText, ModalityImage}, SupportsBatch: true}
    }
    return CapabilityMetadata{Modalities: []Modality{ModalityText}, SupportsBatch: true}
}
```

**Example — BuildContentCloseable closer chaining:**
```go
// Source: BuildDenseCloseable pattern from registry.go (lines 63-75)

func BuildContentCloseable(name string, config EmbeddingFunctionConfig) (ContentEmbeddingFunction, func() error, error) {
    ef, err := BuildContent(name, config)
    if err != nil {
        return nil, nil, err
    }
    closer := func() error {
        if c, ok := ef.(Closeable); ok {
            return c.Close()
        }
        return nil
    }
    return ef, closer, nil
}
// Note: adapted ContentEmbeddingFunctions already implement Closeable (verified in multimodal_compat.go:9-14)
```

### Pattern 2: BuildEmbeddingFunctionFromConfig multimodal fallback

**What:** Extend the existing function to try the multimodal registry when the dense registry has no match. Preserves `nil, nil` for unknown names.

**When to use:** Existing auto-wiring callsite at `client_http.go:424`.

**Example:**
```go
// Source: pkg/api/v2/configuration.go BuildEmbeddingFunctionFromConfig (current: lines 195-216)

func BuildEmbeddingFunctionFromConfig(cfg *CollectionConfigurationImpl) (embeddings.EmbeddingFunction, error) {
    if cfg == nil {
        return nil, nil
    }
    efInfo, ok := cfg.GetEmbeddingFunctionInfo()
    if ok && efInfo != nil && efInfo.IsKnown() {
        // Try dense first (existing behavior preserved)
        if embeddings.HasDense(efInfo.Name) {
            return embeddings.BuildDense(efInfo.Name, efInfo.Config)
        }
        // New: try multimodal (MultimodalEmbeddingFunction embeds EmbeddingFunction)
        if embeddings.HasMultimodal(efInfo.Name) {
            return embeddings.BuildMultimodal(efInfo.Name, efInfo.Config)
        }
    }
    // Schema path unchanged
    schema := cfg.GetSchema()
    if schema != nil {
        ef := schema.GetEmbeddingFunction()
        if ef != nil {
            return ef, nil
        }
    }
    return nil, nil
}
```

### Pattern 3: BuildContentEFFromConfig

**What:** New function parallel to `BuildEmbeddingFunctionFromConfig`, returning `ContentEmbeddingFunction`.

**Example:**
```go
// Source: design decision from 03-CONTEXT.md

func BuildContentEFFromConfig(cfg *CollectionConfigurationImpl) (embeddings.ContentEmbeddingFunction, error) {
    if cfg == nil {
        return nil, nil
    }
    efInfo, ok := cfg.GetEmbeddingFunctionInfo()
    if !ok || efInfo == nil || !efInfo.IsKnown() {
        return nil, nil
    }
    // Delegates to BuildContent which handles fallback chain internally
    if embeddings.HasContent(efInfo.Name) || embeddings.HasMultimodal(efInfo.Name) || embeddings.HasDense(efInfo.Name) {
        return embeddings.BuildContent(efInfo.Name, efInfo.Config)
    }
    return nil, nil
}
```

### Pattern 4: SetContentEmbeddingFunction on CollectionConfigurationImpl

**What:** Mirrors `SetEmbeddingFunction`. Type-asserts to `EmbeddingFunction` for persistence; silently skips if assertion fails (runtime-only content EF case).

**Example:**
```go
// Source: SetEmbeddingFunction pattern from configuration.go lines 150-159

func (c *CollectionConfigurationImpl) SetContentEmbeddingFunction(ef embeddings.ContentEmbeddingFunction) {
    if ef == nil {
        return
    }
    // Content-native providers also implement EmbeddingFunction (for Name()/GetConfig())
    denseEF, ok := ef.(embeddings.EmbeddingFunction)
    if !ok {
        return // runtime-only content EF — skip persistence
    }
    c.SetEmbeddingFunction(denseEF) // reuses existing SetEmbeddingFunction
}
```

### Pattern 5: CollectionImpl content EF field and option

**What:** Add `contentEmbeddingFunction embeddings.ContentEmbeddingFunction` field to `CollectionImpl`. Add `WithContentEmbeddingFunction` functional option. Auto-wiring in the GetCollection path populates both fields.

**Example — struct extension (collection_http.go):**
```go
// Source: CollectionImpl struct (collection_http.go:48-59)

type CollectionImpl struct {
    // existing fields unchanged ...
    embeddingFunction        embeddings.EmbeddingFunction
    contentEmbeddingFunction embeddings.ContentEmbeddingFunction  // NEW
}
```

**Example — auto-wiring in client_http.go (after line 428):**
```go
// Source: existing auto-wiring pattern client_http.go:421-428

contentEF := req.contentEmbeddingFunction  // from explicit option
if contentEF == nil {
    autoWiredContentEF, buildErr := BuildContentEFFromConfig(configuration)
    if buildErr != nil {
        client.logger.Warn("failed to auto-wire content embedding function", logger.ErrorField("error", buildErr))
    }
    contentEF = autoWiredContentEF
}
```

### Anti-Patterns to Avoid

- **Adding a new `sync.RWMutex` for the content map:** The existing `mu` covers all four maps — adding a second mutex creates deadlock risk when functions like `BuildContent` acquire the lock multiple times in the fallback chain.
- **Acquiring `mu` inside the fallback chain multiple times without releasing:** Each step of the fallback should release the lock before trying the next map. The current `BuildDense`/`BuildMultimodal` pattern already does this correctly.
- **Storing capability metadata in persistence:** Locked decision — capabilities are runtime-only, not persisted. Planner must not add fields to `EmbeddingFunctionInfo`.
- **Modifying existing provider `init()` functions:** The fallback chain provides content support automatically. No `RegisterContent` calls in existing providers.
- **Using `Must*` functions:** Project-wide prohibition from CLAUDE.md. Use error-checked variants.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Adapting dense EF to ContentEmbeddingFunction | Custom adapter struct | `AdaptEmbeddingFunctionToContent` (multimodal_compat.go:34) | Already handles validation, batch, text-only path |
| Adapting multimodal EF to ContentEmbeddingFunction | Custom adapter struct | `AdaptMultimodalEmbeddingFunctionToContent` (multimodal_compat.go:42) | Already handles text/image routing, batch |
| Closer for adapted content EF | Manual close logic | `Closeable` interface check in `BuildContentCloseable` | Adapters already implement `Closeable` by delegating to wrapped EF (multimodal_compat.go:53-58, 96-101) |
| Capability metadata inference | Hard-coded per-provider logic | `CapabilityAware` interface check with typed fallback | Uniform pattern; future native content providers will provide their own metadata |

**Key insight:** The Phase 2 adapters were designed specifically to be the bridge. The content factory fallback chain is the primary consumer of those adapters — Phase 3 completes the circuit.

---

## Common Pitfalls

### Pitfall 1: Double Lock Acquisition in BuildContent Fallback

**What goes wrong:** `BuildContent` acquires `mu.RLock()` three times during the fallback chain. If `BuildDense`/`BuildMultimodal` are called inside the lock, they will try to acquire `mu.RLock()` again. In Go, `sync.RWMutex` is not reentrant — this deadlocks.

**Why it happens:** `BuildDense` and `BuildMultimodal` each acquire `mu.RLock()` internally. If `BuildContent` holds the lock while calling them, deadlock occurs.

**How to avoid:** `BuildContent` must check the map under the lock but release before calling the factory, or replicate the factory lookup inline (check map → unlock → call factory). The existing `BuildDense`/`BuildMultimodal` code in `registry.go:50-58` shows the correct pattern: lock, read factory, unlock, then call factory.

**Warning signs:** Any test involving a fallback path (dense or multimodal used as content) hanging indefinitely.

### Pitfall 2: `BuildEmbeddingFunctionFromConfig` Multimodal Fallback Breaks `nil, nil` Contract

**What goes wrong:** The existing `BuildEmbeddingFunctionFromConfig` returns `nil, nil` for unknown provider names. If the multimodal fallback path returns an error instead of `nil, nil` for an unknown name, existing auto-wiring breaks (collections with unknown providers currently load without EF; after the change they would fail to load).

**Why it happens:** Inconsistent error handling between the dense and multimodal branches.

**How to avoid:** The fallback for `BuildEmbeddingFunctionFromConfig` must only try the multimodal path if `HasMultimodal(name)` is true. If neither `HasDense` nor `HasMultimodal` matches, return `nil, nil` exactly as before.

### Pitfall 3: `WithContentEmbeddingFunction` Priority Not Propagated to Dense EF Field

**What goes wrong:** The locked decision says "If both are set, content takes priority; dense EF is derived from the content one." This means when `WithContentEmbeddingFunction` is provided and the `ContentEmbeddingFunction` also implements `EmbeddingFunction`, the collection's `embeddingFunction` field should be populated from it — not left nil.

**Why it happens:** Treating the two fields as fully independent; forgetting that content-native providers embed `EmbeddingFunction`.

**How to avoid:** In the `WithContentEmbeddingFunction` option function (or in the collection constructor), type-assert the content EF to `EmbeddingFunction` and set `embeddingFunction` if the assertion succeeds.

### Pitfall 4: `SetContentEmbeddingFunction` Silently Skips Without Signal

**What goes wrong:** The persistence method silently skips if the `ContentEmbeddingFunction` does not implement `EmbeddingFunction`. Callers may believe persistence succeeded when it didn't.

**Why it happens:** The silent-skip is by design (runtime-only content EFs exist), but callers of `SetContentEmbeddingFunction` from auto-wiring should not expect a guarantee.

**How to avoid:** Document clearly that `SetContentEmbeddingFunction` provides best-effort persistence. No caller should check persistence by reading back the stored value immediately unless it also handles the `nil` case.

### Pitfall 5: Test Package Isolation for Registry State

**What goes wrong:** Registry tests in `registry_test.go` use `package embeddings` (internal package), so they share the global factory maps. Tests that `Register*` a name cannot re-register the same name in a sibling test, causing `"already registered"` panics if tests are run repeatedly or in the wrong order.

**Why it happens:** Global mutable state + test isolation. Current `registry_test.go` uses unique names per test (e.g., `"test_dense_ef"`, `"test_content_ef"`) to avoid collisions.

**How to avoid:** Use unique names per test case for any new content registry tests, following the existing pattern in `registry_test.go`.

---

## Code Examples

### Existing Closeable Wrapper Pattern (replicate for content)

```go
// Source: pkg/embeddings/registry.go:63-75 (BuildDenseCloseable)
func BuildDenseCloseable(name string, config EmbeddingFunctionConfig) (EmbeddingFunction, func() error, error) {
    ef, err := BuildDense(name, config)
    if err != nil {
        return nil, nil, err
    }
    closer := func() error {
        if c, ok := ef.(Closeable); ok {
            return c.Close()
        }
        return nil
    }
    return ef, closer, nil
}
```

### Existing auto-wiring callsite (extend, do not replace)

```go
// Source: pkg/api/v2/client_http.go:421-441
configuration := NewCollectionConfigurationFromMap(cm.ConfigurationJSON)
// Auto-wire EF: explicit option takes priority, otherwise build from server config
ef := req.embeddingFunction
if ef == nil {
    autoWiredEF, buildErr := BuildEmbeddingFunctionFromConfig(configuration)
    if buildErr != nil {
        client.logger.Warn("failed to auto-wire embedding function", logger.ErrorField("error", buildErr))
    }
    ef = autoWiredEF
}
c := &CollectionImpl{
    // ...
    embeddingFunction: ef,
}
```

### Existing SetEmbeddingFunction (model for SetContentEmbeddingFunction)

```go
// Source: pkg/api/v2/configuration.go:150-159
func (c *CollectionConfigurationImpl) SetEmbeddingFunction(ef embeddings.EmbeddingFunction) {
    if ef == nil {
        return
    }
    c.SetEmbeddingFunctionInfo(&EmbeddingFunctionInfo{
        Type:   efTypeKnown,
        Name:   ef.Name(),
        Config: ef.GetConfig(),
    })
}
```

### Existing init() registration pattern (for test mocks only)

```go
// Source: pkg/embeddings/roboflow/roboflow.go:355-366
func init() {
    if err := embeddings.RegisterDense("roboflow", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.EmbeddingFunction, error) {
        return NewRoboflowEmbeddingFunctionFromConfig(cfg)
    }); err != nil {
        panic(err)
    }
    if err := embeddings.RegisterMultimodal("roboflow", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.MultimodalEmbeddingFunction, error) {
        return NewRoboflowEmbeddingFunctionFromConfig(cfg)
    }); err != nil {
        panic(err)
    }
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single `embeddingFunction` field on collection | Parallel `contentEmbeddingFunction` field added additively | Phase 3 | Callers using dense EF are unchanged; content EF callers use the new field |
| `BuildEmbeddingFunctionFromConfig` tries dense only | Gains multimodal fallback (dense → multimodal → schema) | Phase 3 | Multimodal providers (Roboflow) can now be auto-wired as `EmbeddingFunction` |
| No content build-from-config path | `BuildContentEFFromConfig` added | Phase 3 | Any registered provider can be loaded as `ContentEmbeddingFunction` |

**No deprecated items for this phase.** All additions are additive; no existing functions are removed or signature-changed.

---

## Open Questions

1. **`BuildContentCloseable` closer chain when the underlying provider is closeable but wrapped by an adapter**
   - What we know: The adapters in `multimodal_compat.go` already implement `Closeable` by delegating to the wrapped EF (lines 53-58, 96-101). The `BuildContentCloseable` closer `if c, ok := ef.(Closeable); ok` will fire for adapted EFs since the adapters implement `Closeable`.
   - What's unclear: Whether Phase 3 needs a "chain both closers" pattern or the single delegate is sufficient. Given adapters already delegate, a single `Closeable` check on the result of `BuildContent` is sufficient.
   - Recommendation: Single `Closeable` check (matching the dense/multimodal pattern). No chaining needed — adapters handle delegation internally.

2. **`WithContentEmbeddingFunction` for `GetCollectionOption` vs `CreateCollectionOption`**
   - What we know: `WithEmbeddingFunctionGet` and `WithEmbeddingFunctionCreate` are separate option functions for the two operations. A `WithContentEmbeddingFunction` option needs the same split, or a shared `Option` on `CollectionImpl`.
   - What's unclear: The locked decision says "new collection option alongside existing `WithEmbeddingFunction`" without specifying whether it applies to Get, Create, or both.
   - Recommendation: Add `WithContentEmbeddingFunctionGet` for the Get path (where auto-wiring matters), and `WithContentEmbeddingFunctionCreate` if create-time content EF injection is needed. The auto-wiring test path only requires the Get variant for Phase 3.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing + testify (assert/require) |
| Config file | none (build tags control suite selection) |
| Quick run command | `go test ./pkg/embeddings/ -run TestRegister` |
| Full suite command | `go test ./pkg/embeddings/... ./pkg/api/v2/...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REG-01 | `RegisterContent` / `BuildContent` / `HasContent` / `ListContent` work correctly | unit | `go test ./pkg/embeddings/ -run TestRegisterAndBuildContent` | ❌ Wave 0 |
| REG-01 | `BuildContent` fallback chain: content → multimodal+adapt → dense+adapt | unit | `go test ./pkg/embeddings/ -run TestBuildContentFallback` | ❌ Wave 0 |
| REG-01 | `BuildContentEFFromConfig` builds from stored config | unit | `go test ./pkg/api/v2/ -run TestBuildContentEFFromConfig` | ❌ Wave 0 |
| REG-01 | Config round-trip: dense EF config → `BuildContentEFFromConfig` → `ContentEmbeddingFunction` | unit | `go test ./pkg/embeddings/ -run TestContentConfigRoundTrip` | ❌ Wave 0 |
| REG-02 | `BuildEmbeddingFunctionFromConfig` still returns dense EF for existing providers | unit | `go test ./pkg/api/v2/ -run TestBuildEmbeddingFunctionFromConfig` | ❌ Wave 0 |
| REG-02 | `BuildEmbeddingFunctionFromConfig` gains multimodal fallback for Roboflow-style dual-registered providers | unit | `go test ./pkg/api/v2/ -run TestBuildEFFromConfigMultimodalFallback` | ❌ Wave 0 |
| REG-02 | Collection auto-wiring populates both `ef` and `contentEF` fields | unit | `go test ./pkg/api/v2/ -run TestAutoWiring` | ❌ Wave 0 |
| REG-02 | `WithContentEmbeddingFunction` option overrides auto-wired content EF | unit | `go test ./pkg/api/v2/ -run TestWithContentEmbeddingFunction` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./pkg/embeddings/ -run TestRegister`
- **Per wave merge:** `go test ./pkg/embeddings/... ./pkg/api/v2/...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

All test files for Phase 3 are new:

- [ ] `pkg/embeddings/registry_test.go` — extend existing file with content map tests (REG-01 registry functions)
- [ ] `pkg/api/v2/configuration_test.go` (new) — covers `BuildContentEFFromConfig`, `BuildEmbeddingFunctionFromConfig` multimodal fallback, `SetContentEmbeddingFunction` (REG-01, REG-02)
- [ ] `pkg/api/v2/collection_content_test.go` (new) — covers auto-wiring, `WithContentEmbeddingFunction`, priority logic (REG-02)

No new framework installation needed — existing `go test` + `testify` infrastructure is sufficient.

---

## Sources

### Primary (HIGH confidence)

- `pkg/embeddings/registry.go` — Verified full registry pattern (dense/sparse/multimodal maps, mutex, all 6 functions per type)
- `pkg/embeddings/embedding.go` — Verified `ContentEmbeddingFunction`, `CapabilityAware`, `Closeable` interface definitions
- `pkg/embeddings/capabilities.go` — Verified `CapabilityMetadata` struct and query methods
- `pkg/embeddings/multimodal_compat.go` — Verified both adapters + `Closeable` implementation + validation behavior
- `pkg/api/v2/configuration.go` — Verified `BuildEmbeddingFunctionFromConfig` (dense-only, then schema), `SetEmbeddingFunction`, `EmbeddingFunctionInfo` shape
- `pkg/api/v2/client_http.go:421-441` — Verified auto-wiring callsite structure
- `pkg/api/v2/collection_http.go:48-59` — Verified `CollectionImpl` struct with single `embeddingFunction` field
- `pkg/embeddings/registry_test.go` — Verified test patterns (unique name strategy, closeable tests)
- `pkg/embeddings/persistence_test.go` — Verified EF build tag, round-trip pattern, mock structures
- `pkg/embeddings/roboflow/roboflow.go:355-367` — Verified dual registration (dense + multimodal) pattern

### Secondary (MEDIUM confidence)

- `.planning/phases/03-registry-and-config-integration/03-CONTEXT.md` — All design decisions verified against code; no contradictions found

### Tertiary (LOW confidence)

None.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, all in-use libraries verified in source
- Architecture: HIGH — all patterns derived directly from existing code in the repo
- Pitfalls: HIGH — deadlock and nil-nil contract pitfalls verified from actual code structure

**Research date:** 2026-03-20
**Valid until:** 2026-09-20 (stable Go project; no external API dependencies in this phase)
