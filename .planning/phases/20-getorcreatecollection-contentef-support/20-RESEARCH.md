# Phase 20: GetOrCreateCollection contentEF Support - Research

**Researched:** 2026-04-07
**Domain:** Go client V2 API -- collection lifecycle, embedding function wiring
**Confidence:** HIGH

## Summary

Phase 20 extends `CreateCollection` and `GetOrCreateCollection` to accept and propagate a `contentEmbeddingFunction` option, mirroring the existing `denseEF` wiring pattern. The changes are localized to three files: `client.go` (op struct + option + config persistence), `client_http.go` (HTTP CreateCollection constructor), and `client_local_embedded.go` (embedded CreateCollection state + GetOrCreateCollection forwarding).

All infrastructure is already in place from prior phases: `wrapContentEFCloseOnce` (Phase 11), `SetContentEmbeddingFunction` on `CollectionConfigurationImpl` (Phase 3), `WithContentEmbeddingFunctionGet` (Phase 19), `buildEmbeddedCollection` already accepts `overrideContentEF`, and `embeddedCollectionState` already has a `contentEmbeddingFunction` field. This phase wires existing components together -- no new abstractions needed.

**Primary recommendation:** Add `contentEmbeddingFunction` field to `CreateCollectionOp`, create `WithContentEmbeddingFunctionCreate` option, persist contentEF config in `PrepareAndValidateCollectionRequest`, pass contentEF through HTTP and embedded client constructors, and forward contentEF in embedded `GetOrCreateCollection` to `GetCollection` via `WithContentEmbeddingFunctionGet`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** HTTP GetOrCreateCollection does NOT auto-wire contentEF from server config for existing collections. ContentEF comes only from the user-provided option (or nil if not given). Only GetCollection auto-wires from server config.
- **D-02:** Forward contentEF from CreateCollectionOp to GetCollection via WithContentEmbeddingFunctionGet, same pattern as existing denseEF forwarding.
- **D-03:** When embedded CreateCollection handles an existing collection (isNewCreation=false), ignore the user-provided contentEF and use existing state -- set overrideContentEF=nil for existing collections.
- **D-04:** No EF conflict validation in this phase.
- **D-05:** Persist contentEF config in PrepareAndValidateCollectionRequest when contentEF is provided. Call SetContentEmbeddingFunction on Configuration (or Schema).
- **D-06:** Mirror the denseEF close-once + ownsEF pattern. Add contentEmbeddingFunction: wrapContentEFCloseOnce(req.contentEmbeddingFunction) to the CollectionImpl constructor in HTTP CreateCollection.

### Claude's Discretion
- Test structure and file organization
- Whether to add contentEF wiring to embedded CreateCollection's isNewCreation=true path (storing in state via upsertCollectionState)
- Internal helper decomposition

### Deferred Ideas (OUT OF SCOPE)
- Full EF conflict detection (separate GH issue)
- Embedded GetOrCreateCollection restructuring to align with Python/JS single-call pattern
</user_constraints>

## Project Constraints (from CLAUDE.md)

- Use functional options pattern for client initialization
- New features target V2 API (`/pkg/api/v2/`)
- Tests use `testify` for assertions
- Tests use `testcontainers-go` for integration tests (but unit tests with httptest/mock runtimes are preferred here)
- Build tags segregate test suites (`basicv2` for V2 tests)
- Never panic in production code
- Run `make lint` before committing
- Use conventional commits
- Keep things radically simple

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `net/http`, `encoding/json` | N/A | HTTP client, JSON marshalling | Already used throughout codebase [VERIFIED: codebase grep] |
| `github.com/pkg/errors` | existing | Error wrapping | Project-wide convention [VERIFIED: codebase grep] |
| `github.com/stretchr/testify` | existing | Test assertions | Project test convention [VERIFIED: CLAUDE.md] |
| `sync`, `sync/atomic` | N/A | Close-once wrappers, ownership flags | Already used for EF lifecycle [VERIFIED: ef_close_once.go] |

No new dependencies needed. All work uses existing packages.

## Architecture Patterns

### Files to Modify
```
pkg/api/v2/
  client.go                   # CreateCollectionOp + WithContentEmbeddingFunctionCreate + PrepareAndValidate
  client_http.go              # HTTP CreateCollection constructor (add contentEF field)
  client_local_embedded.go    # Embedded CreateCollection state + GetOrCreateCollection forwarding
```

### Test Files to Create/Modify
```
pkg/api/v2/
  client_http_test.go              # HTTP GetOrCreateCollection with contentEF
  client_local_embedded_test.go    # Embedded CreateCollection + GetOrCreateCollection with contentEF
```

### Pattern 1: Functional Option for CreateCollectionOp
**What:** Add `WithContentEmbeddingFunctionCreate` following the exact pattern of `WithEmbeddingFunctionCreate` (line 466-474 of `client.go`).
**When to use:** User wants to associate a contentEF with a new or get-or-create collection.
**Example:**
```go
// Source: client.go:466-474 (existing pattern)
func WithContentEmbeddingFunctionCreate(ef embeddings.ContentEmbeddingFunction) CreateCollectionOption {
    return func(op *CreateCollectionOp) error {
        if ef == nil {
            return errors.New("content embedding function cannot be nil")
        }
        op.contentEmbeddingFunction = ef
        return nil
    }
}
```

### Pattern 2: Config Persistence in PrepareAndValidateCollectionRequest
**What:** After persisting denseEF config, also persist contentEF config if provided.
**When to use:** During collection creation when contentEF is set.
**Example:**
```go
// Source: client.go:268-301 (existing pattern, extend at end)
// After denseEF config persistence, add contentEF config persistence:
if op.contentEmbeddingFunction != nil {
    if op.Schema != nil {
        op.Schema.SetContentEmbeddingFunction(op.contentEmbeddingFunction)
    } else {
        if op.Configuration == nil {
            op.Configuration = NewCollectionConfiguration()
        }
        op.Configuration.SetContentEmbeddingFunction(op.contentEmbeddingFunction)
    }
}
```

### Pattern 3: HTTP CreateCollection ContentEF Wiring
**What:** Add `contentEmbeddingFunction: wrapContentEFCloseOnce(req.contentEmbeddingFunction)` to the `CollectionImpl` constructor at line 344 of `client_http.go`.
**When to use:** HTTP CreateCollection path.
**Example:**
```go
// Source: client_http.go:344-358 (extend existing constructor)
c := &CollectionImpl{
    name:                     cm.Name,
    id:                       cm.ID,
    // ... existing fields ...
    embeddingFunction:        wrapEFCloseOnce(req.embeddingFunction),
    contentEmbeddingFunction: wrapContentEFCloseOnce(req.contentEmbeddingFunction), // NEW
    dimension:                cm.Dimension,
}
```

### Pattern 4: Embedded CreateCollection ContentEF State Management
**What:** In the embedded CreateCollection `isNewCreation=true` path, store contentEF in state via `upsertCollectionState`. In `isNewCreation=false`, set `overrideContentEF=nil`.
**When to use:** Embedded client collection creation.
**Example:**
```go
// Source: client_local_embedded.go:389-412 (extend existing pattern)
overrideContentEF := req.contentEmbeddingFunction
if isNewCreation {
    overrideEF = wrapEFCloseOnce(req.embeddingFunction)
    overrideContentEF = wrapContentEFCloseOnce(req.contentEmbeddingFunction)
    client.upsertCollectionState(model.ID, func(state *embeddedCollectionState) {
        state.embeddingFunction = overrideEF
        state.contentEmbeddingFunction = overrideContentEF  // NEW
        // ... existing metadata/config/schema ...
    })
} else {
    overrideEF = nil
    overrideContentEF = nil  // D-03: ignore user-provided contentEF for existing collections
}

collection, err := client.buildEmbeddedCollection(*model, req.Database, overrideEF, overrideContentEF, true, true)
```

### Pattern 5: Embedded GetOrCreateCollection ContentEF Forwarding
**What:** Forward contentEF from `CreateCollectionOp` to `GetCollection` via `WithContentEmbeddingFunctionGet` option.
**When to use:** Embedded GetOrCreateCollection tries GetCollection first.
**Example:**
```go
// Source: client_local_embedded.go:434-437 (extend existing pattern)
getOptions := []GetCollectionOption{WithDatabaseGet(req.Database)}
if req.embeddingFunction != nil {
    getOptions = append(getOptions, WithEmbeddingFunctionGet(req.embeddingFunction))
}
if req.contentEmbeddingFunction != nil {  // NEW
    getOptions = append(getOptions, WithContentEmbeddingFunctionGet(req.contentEmbeddingFunction))
}
```

### Anti-Patterns to Avoid
- **Auto-wiring contentEF in HTTP GetOrCreateCollection:** Per D-01, only GetCollection auto-wires. GetOrCreateCollection (which delegates to CreateCollection in HTTP) passes through user-provided contentEF only.
- **Adding EF conflict detection:** Per D-04, out of scope. No conflict validation between denseEF and contentEF.
- **Wrapping contentEF in close-once twice:** `wrapContentEFCloseOnce` already guards against double-wrapping (checks `*closeOnceContentEF` type).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| ContentEF close-once wrapping | Custom wrapper | `wrapContentEFCloseOnce` | Already exists, nil-safe, idempotent [VERIFIED: ef_close_once.go:211-219] |
| ContentEF config persistence | Manual config injection | `SetContentEmbeddingFunction` on Configuration/Schema | Already delegates to SetEmbeddingFunction [VERIFIED: configuration.go:247-258] |
| ContentEF auto-wiring from config | Custom builder | `BuildContentEFFromConfig` | Used by GetCollection, not needed in Create path but exists [VERIFIED: configuration.go:225-245] |

## Common Pitfalls

### Pitfall 1: Forgetting to pass overrideContentEF in buildEmbeddedCollection call
**What goes wrong:** Embedded CreateCollection calls `buildEmbeddedCollection` but passes `nil` for `overrideContentEF`, so the contentEF never reaches the collection.
**Why it happens:** The existing call at line 408 already passes `nil` for contentEF: `buildEmbeddedCollection(*model, req.Database, overrideEF, nil, true, true)`.
**How to avoid:** Replace the `nil` with `overrideContentEF` variable that mirrors the `overrideEF` logic.
**Warning signs:** Tests show contentEF is nil on the returned collection despite being passed as an option.

### Pitfall 2: Not persisting contentEF config in PrepareAndValidateCollectionRequest
**What goes wrong:** ContentEF is set on the collection but not persisted in Configuration/Schema, so future GetCollection calls cannot auto-wire from config.
**Why it happens:** Config persistence is done in PrepareAndValidateCollectionRequest, and it currently only handles denseEF.
**How to avoid:** Add contentEF config persistence after the denseEF config persistence block (D-05).
**Warning signs:** GetCollection auto-wiring returns nil contentEF for collections created with contentEF.

### Pitfall 3: Schema.SetContentEmbeddingFunction may not exist yet
**What goes wrong:** Compile error if Schema doesn't have a SetContentEmbeddingFunction method.
**Why it happens:** Configuration has it, but Schema may not.
**How to avoid:** Check if Schema has the method. If not, fall through to Configuration.
**Warning signs:** Compilation failure.

## Code Examples

### CreateCollectionOp Field Addition
```go
// Source: client.go:244-253 (extend struct)
type CreateCollectionOp struct {
    Name                     string                              `json:"name"`
    CreateIfNotExists        bool                                `json:"get_or_create,omitempty"`
    embeddingFunction        embeddings.EmbeddingFunction        `json:"-"`
    contentEmbeddingFunction embeddings.ContentEmbeddingFunction `json:"-"` // NEW
    Metadata                 CollectionMetadata                  `json:"metadata,omitempty"`
    Configuration            *CollectionConfigurationImpl        `json:"configuration,omitempty"`
    Schema                   *Schema                             `json:"schema,omitempty"`
    Database                 Database                            `json:"-"`
    disableEFConfigStorage   bool                                `json:"-"`
}
```

### HTTP Test Pattern (httptest mock server)
```go
// Source: client_http_test.go (existing TestCreateCollection pattern)
func TestGetOrCreateCollectionWithContentEF(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(CollectionModel{
            Name: "test-col",
            ID:   "test-id",
            // ...
        })
    }))
    defer server.Close()

    client, err := NewHTTPClient(WithBaseURL(server.URL), WithLogger(testLogger()))
    require.NoError(t, err)

    contentEF := &mockCloseableContentEF{}
    col, err := client.GetOrCreateCollection(ctx, "test-col",
        WithContentEmbeddingFunctionCreate(contentEF),
    )
    require.NoError(t, err)
    impl := col.(*CollectionImpl)
    require.NotNil(t, impl.contentEmbeddingFunction)
}
```

### Embedded Test Pattern (memory runtime)
```go
// Source: client_local_embedded_test.go (existing TestEmbeddedLocalClientGetOrCreateCollection pattern)
func TestEmbeddedGetOrCreateCollection_ContentEF_ForwardedToGetCollection(t *testing.T) {
    runtime := newCountingMemoryEmbeddedRuntime()
    client := newEmbeddedClientForRuntime(t, runtime)
    ctx := context.Background()

    _, err := client.CreateCollection(ctx, "test-col", WithEmbeddingFunctionCreate(denseEF))
    require.NoError(t, err)

    contentEF := &mockCloseableContentEF{}
    got, err := client.GetOrCreateCollection(ctx, "test-col",
        WithContentEmbeddingFunctionCreate(contentEF),
    )
    require.NoError(t, err)

    ec := got.(*embeddedCollection)
    ec.mu.RLock()
    gotContentEF := ec.contentEmbeddingFunction
    ec.mu.RUnlock()
    require.Same(t, contentEF, unwrapCloseOnceContentEF(gotContentEF))
}
```

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify v1.x |
| Config file | None (Go built-in) |
| Quick run command | `go test -tags=basicv2 -run TestGetOrCreate ./pkg/api/v2/... -count=1` |
| Full suite command | `make test` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SC-1 | CreateCollectionOp has contentEmbeddingFunction field | unit | `go test -tags=basicv2 -run TestGetOrCreateCollectionWithContentEF ./pkg/api/v2/... -count=1` | Wave 0 |
| SC-2 | WithContentEmbeddingFunctionCreate option works | unit | `go test -tags=basicv2 -run TestGetOrCreateCollectionWithContentEF ./pkg/api/v2/... -count=1` | Wave 0 |
| SC-3 | GetOrCreateCollection forwards contentEF to GetCollection (embedded) | unit | `go test -run TestEmbeddedGetOrCreateCollection.*ContentEF ./pkg/api/v2/... -count=1` | Wave 0 |
| SC-4 | HTTP path wires contentEF into CollectionImpl | unit | `go test -tags=basicv2 -run TestGetOrCreateCollectionWithContentEF ./pkg/api/v2/... -count=1` | Wave 0 |
| SC-5 | Embedded CreateCollection stores contentEF in state for new collections | unit | `go test -run TestEmbeddedCreateCollection.*ContentEF ./pkg/api/v2/... -count=1` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -tags=basicv2 -run TestGetOrCreate ./pkg/api/v2/... -count=1`
- **Per wave merge:** `make test && make lint`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- None -- existing test infrastructure (httptest servers, memory embedded runtime, mock EF types) covers all requirements. New test functions need to be added but no framework or fixture gaps.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `Schema.SetContentEmbeddingFunction` method exists | Architecture Patterns (Pattern 2) | If it doesn't exist, config persistence must go through Configuration only. Compile error would surface immediately. LOW risk. |

## Open Questions (RESOLVED)

1. **Does Schema have SetContentEmbeddingFunction?** — RESOLVED: Schema does NOT have `SetContentEmbeddingFunction`. Only `CollectionConfigurationImpl` does. For the Schema path, type-assert contentEF to `embeddings.EmbeddingFunction` and call `Schema.SetEmbeddingFunction`. This mirrors the delegation pattern used by `CollectionConfigurationImpl.SetContentEmbeddingFunction` internally.

## Sources

### Primary (HIGH confidence)
- `pkg/api/v2/client.go` -- CreateCollectionOp struct, PrepareAndValidateCollectionRequest, WithEmbeddingFunctionCreate pattern
- `pkg/api/v2/client_http.go` -- HTTP CreateCollection and GetOrCreateCollection implementation
- `pkg/api/v2/client_local_embedded.go` -- Embedded CreateCollection, GetOrCreateCollection, buildEmbeddedCollection, embeddedCollectionState
- `pkg/api/v2/ef_close_once.go` -- wrapContentEFCloseOnce, wrapEFCloseOnce
- `pkg/api/v2/configuration.go` -- SetContentEmbeddingFunction, BuildContentEFFromConfig
- `pkg/api/v2/collection_http.go` -- CollectionImpl struct with contentEmbeddingFunction field
- `pkg/api/v2/client_http_test.go` -- Existing HTTP test patterns
- `pkg/api/v2/client_local_embedded_test.go` -- Existing embedded test patterns with mock runtime

### Secondary (MEDIUM confidence)
- CONTEXT.md decisions D-01 through D-06 -- behavioral constraints from user discussion

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies, all infrastructure verified in codebase
- Architecture: HIGH -- all patterns verified from existing denseEF implementation in same files
- Pitfalls: HIGH -- pitfalls derived from direct code inspection of the exact functions to modify

**Research date:** 2026-04-07
**Valid until:** 2026-05-07 (stable Go codebase, patterns well-established)
