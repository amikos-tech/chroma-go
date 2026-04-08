# Phase 18: Embedded Client contentEmbeddingFunction Parity - Research

**Researched:** 2026-04-02
**Domain:** Go embedded client collection lifecycle / content embedding function parity
**Confidence:** HIGH

## Summary

Phase 18 adds `contentEmbeddingFunction` support to the embedded collection path (`embeddedCollection`, `embeddedCollectionState`, `buildEmbeddedCollection`) so that it mirrors the HTTP client (`CollectionImpl`) for content embedding lifecycle, auto-wiring from config, and Close() sharing detection. All infrastructure needed -- close-once wrappers, `BuildContentEFFromConfig`, `WithContentEmbeddingFunctionGet` option, `EmbeddingFunctionUnwrapper` interface, `safeCloseEF` / `reportClosePanic` helpers -- already exists in the `v2` package and is directly reusable.

The scope is narrow: one file to modify (`client_local_embedded.go`) plus tests. The HTTP reference implementation in `collection_http.go` and `client_http.go` provides a line-by-line pattern to follow. Fork() propagation is explicitly skipped (D-01) because Fork returns unsupported error in embedded mode.

**Primary recommendation:** Mirror the HTTP `CollectionImpl` patterns for struct fields, GetCollection auto-wiring, and Close() sharing detection verbatim -- all helper functions are already package-accessible.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Skip contentEF propagation in Fork(). Fork returns an unsupported error in embedded mode -- adding contentEF close-once wrapping to the stub would be dead code.
- **D-02:** Full auto-wiring via `BuildContentEFFromConfig` in embedded `GetCollection`, matching HTTP behavior. When no explicit `WithContentEmbeddingFunctionGet` is passed, attempt to build contentEF from stored collection configuration.
- **D-03:** Full mirror of HTTP sharing detection in `embeddedCollection.Close()`. Close contentEF first, then check if denseEF shares the same resource (via `EmbeddingFunctionUnwrapper` unwrapper check + identity check) before closing denseEF.

### Claude's Discretion
- `embeddedCollectionState` struct field naming and initialization details
- Test structure and file organization
- Whether to add `contentEmbeddingFunction` to `CreateCollectionOp` or keep it GetCollection-only (matching current HTTP behavior)

### Deferred Ideas (OUT OF SCOPE)
None
</user_constraints>

## Project Constraints (from CLAUDE.md)

- **Conventional commits** required for all commits
- **No panics in production code** -- use `safeCloseEF` / `reportClosePanic` for close paths
- **Run `make lint` before committing** -- all code must pass linting
- **Use `testify` for assertions** in tests
- **Build tags** must be used for test segregation (`basicv2`)
- **V2 API** is the target for new features (`/pkg/api/v2/`)
- **Minimal comments** -- code and names should be self-explanatory
- **Keep things radically simple** for as long as possible

## Architecture Patterns

### Reference Implementation (HTTP Path)

The HTTP client already implements full contentEF support. The embedded client must mirror these patterns exactly:

#### 1. Struct Fields (collection_http.go:51-66)
```go
type CollectionImpl struct {
    // ... existing fields ...
    embeddingFunction        embeddings.EmbeddingFunction
    contentEmbeddingFunction embeddings.ContentEmbeddingFunction
    ownsEF                   atomic.Bool
    closeOnce                sync.Once
    closeErr                 error
}
```

#### 2. GetCollection Auto-Wiring (client_http.go:421-462)
```go
// Auto-wire content EF first to avoid double factory instantiation
contentEF := req.contentEmbeddingFunction
if contentEF == nil {
    autoWiredContentEF, buildErr := BuildContentEFFromConfig(configuration)
    if buildErr != nil {
        client.logger.Warn("failed to auto-wire content embedding function", ...)
    }
    contentEF = autoWiredContentEF
}
// Auto-wire dense EF: try unwrapping from content adapter first
ef := req.embeddingFunction
if ef == nil {
    if unwrapper, ok := contentEF.(embeddings.EmbeddingFunctionUnwrapper); ok {
        ef = unwrapper.UnwrapEmbeddingFunction()
    } else if denseFromContent, ok := contentEF.(embeddings.EmbeddingFunction); ok {
        ef = denseFromContent
    }
    if ef == nil {
        autoWiredEF, buildErr := BuildEmbeddingFunctionFromConfig(configuration)
        // ...
    }
}
```

#### 3. Close() Sharing Detection (collection_http.go:709-746)
```go
func (c *CollectionImpl) Close() error {
    if !c.ownsEF.Load() {
        return nil
    }
    c.closeOnce.Do(func() {
        var errs []error
        // Close contentEF first
        if c.contentEmbeddingFunction != nil {
            if closer, ok := c.contentEmbeddingFunction.(io.Closer); ok {
                if err := safeCloseEF(closer); err != nil {
                    errs = append(errs, err)
                }
            }
        }
        // Check sharing before closing dense EF
        if c.embeddingFunction != nil {
            shared := false
            denseEF := unwrapCloseOnceEF(c.embeddingFunction)
            if unwrapper, ok := c.contentEmbeddingFunction.(embeddings.EmbeddingFunctionUnwrapper); ok {
                shared = unwrapper.UnwrapEmbeddingFunction() == denseEF
            } else if ef, ok := c.contentEmbeddingFunction.(embeddings.EmbeddingFunction); ok {
                shared = ef == denseEF
            }
            if !shared {
                if closer, ok := c.embeddingFunction.(io.Closer); ok {
                    if err := safeCloseEF(closer); err != nil {
                        errs = append(errs, err)
                    }
                }
            }
        }
        c.closeErr = stderrors.Join(errs...)
    })
    return c.closeErr
}
```

### Embedded Client Modification Points

Five specific locations in `client_local_embedded.go` need changes:

| Location | Lines | Change |
|----------|-------|--------|
| `embeddedCollectionState` struct | 21-27 | Add `contentEmbeddingFunction` field |
| `embeddedCollection` struct | 776-793 | Add `contentEmbeddingFunction` field |
| `GetCollection` | 495-527 | Add auto-wiring + explicit option handling |
| `buildEmbeddedCollection` | 697-774 | Accept and wire contentEF parameter |
| `Close()` | 1405-1423 | Replace single-EF close with dual-EF sharing detection |

Supporting locations that need minor updates:

| Location | Lines | Change |
|----------|-------|--------|
| `upsertCollectionState` | 646-671 | Include contentEF in snapshot return |
| `GetOrCreateCollection` | 410-448 | Pass contentEF option through to GetCollection fallback |

### Anti-Patterns to Avoid
- **Closing denseEF before contentEF:** Content EF may wrap dense EF (adapter pattern). Always close content first, then check sharing.
- **Skipping `unwrapCloseOnceEF` in sharing detection:** Both fields may be wrapped in close-once wrappers. Must unwrap before identity comparison.
- **Adding contentEF to CreateCollection:** HTTP CreateCollection does not wire contentEF -- it is GetCollection-only. Keep parity.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Close-once wrapping | Custom sync.Once wrapper | `wrapContentEFCloseOnce()` from ef_close_once.go | Already tested, handles panic recovery |
| Config auto-wiring | Manual registry lookup | `BuildContentEFFromConfig()` from configuration.go | Handles full fallback chain (content -> multimodal -> dense) |
| Sharing detection unwrap | Manual type switch | `unwrapCloseOnceEF()` from client_http.go | Handles both closeOnceEF and closeOnceContentEF wrapping |
| Safe close | Bare closer.Close() | `safeCloseEF()` from close_logging.go | Panic recovery + stderr reporting |
| Content EF option | New option function | `WithContentEmbeddingFunctionGet` from client.go:199-206 | Already exists and validates non-nil |

## Common Pitfalls

### Pitfall 1: Snapshot Return in upsertCollectionState
**What goes wrong:** The snapshot returned by `upsertCollectionState` omits the new `contentEmbeddingFunction` field, so `buildEmbeddedCollection` never sees the contentEF stored from a prior call.
**Why it happens:** The snapshot is constructed manually at lines 665-671 and must be updated whenever `embeddedCollectionState` gains a field.
**How to avoid:** Add `contentEmbeddingFunction: state.contentEmbeddingFunction` to the snapshot construction.
**Warning signs:** Content EF silently nil after GetCollection with auto-wiring.

### Pitfall 2: GetOrCreateCollection Drops contentEF
**What goes wrong:** `GetOrCreateCollection` builds `getOptions` and only propagates `req.embeddingFunction`, not `req.contentEmbeddingFunction`. The `GetCollection` fallback loses the contentEF.
**Why it happens:** HTTP `GetOrCreateCollection` delegates to `CreateCollection` (which passes through to the server and re-fetches), but the embedded path calls `GetCollection` directly with a subset of options.
**How to avoid:** When the request has a content EF (which currently it cannot since `CreateCollectionOp` lacks the field), propagate `WithContentEmbeddingFunctionGet` to the getOptions. Since D-02 specifies auto-wiring in GetCollection, the auto-wiring will fire regardless -- this pitfall only matters if a user explicitly passes contentEF via CreateCollection options in the future.
**Warning signs:** Non-obvious because auto-wiring masks the gap for config-based providers.

### Pitfall 3: embeddingFunctionSnapshot() Only Returns Dense EF
**What goes wrong:** The existing `embeddingFunctionSnapshot()` method returns only the dense EF. Close() currently calls it to get the EF to close. The new Close() must also snapshot contentEF under the same lock.
**Why it happens:** The method was written before contentEF existed on embedded collections.
**How to avoid:** Either add a `contentEmbeddingFunctionSnapshot()` method or take both snapshots together in Close().
**Warning signs:** Data race on `contentEmbeddingFunction` field during concurrent Close().

### Pitfall 4: Missing io Import
**What goes wrong:** The file already imports `io` so this is not an issue for this specific file, but the new Close() logic uses `io.Closer` type assertions.
**How to avoid:** Verify `io` is in the import block (it is -- line 8).

## Code Examples

### embeddedCollectionState with contentEF
```go
type embeddedCollectionState struct {
    embeddingFunction        embeddingspkg.EmbeddingFunction
    contentEmbeddingFunction embeddingspkg.ContentEmbeddingFunction
    metadata                 CollectionMetadata
    configuration            CollectionConfiguration
    schema                   *Schema
    dimension                int
}
```

### embeddedCollection with contentEF
```go
type embeddedCollection struct {
    mu sync.RWMutex

    name          string
    id            string
    tenant        Tenant
    database      Database
    metadata      CollectionMetadata
    configuration CollectionConfiguration
    schema        *Schema
    dimension     int

    embeddingFunction        embeddingspkg.EmbeddingFunction
    contentEmbeddingFunction embeddingspkg.ContentEmbeddingFunction
    client                   *embeddedLocalClient
    ownsEF                   atomic.Bool
    closeOnce                sync.Once
    closeErr                 error
}
```

### Close() with sharing detection (mirroring HTTP)
```go
func (c *embeddedCollection) Close() error {
    if !c.ownsEF.Load() {
        return nil
    }
    c.mu.RLock()
    ef := c.embeddingFunction
    contentEF := c.contentEmbeddingFunction
    c.mu.RUnlock()
    c.closeOnce.Do(func() {
        var errs []error
        if contentEF != nil {
            if closer, ok := contentEF.(io.Closer); ok {
                if err := safeCloseEF(closer); err != nil {
                    errs = append(errs, err)
                }
            }
        }
        if ef != nil {
            shared := false
            denseEF := unwrapCloseOnceEF(ef)
            if unwrapper, ok := contentEF.(embeddings.EmbeddingFunctionUnwrapper); ok {
                shared = unwrapper.UnwrapEmbeddingFunction() == denseEF
            } else if efFromContent, ok := contentEF.(embeddings.EmbeddingFunction); ok {
                shared = efFromContent == denseEF
            }
            if !shared {
                if closer, ok := ef.(io.Closer); ok {
                    if err := safeCloseEF(closer); err != nil {
                        errs = append(errs, err)
                    }
                }
            }
        }
        c.closeErr = stderrors.Join(errs...)
    })
    return c.closeErr
}
```

### GetCollection auto-wiring
```go
func (client *embeddedLocalClient) GetCollection(ctx context.Context, name string, opts ...GetCollectionOption) (Collection, error) {
    // ... existing request setup ...

    // Auto-wire content EF first
    contentEF := req.contentEmbeddingFunction
    if contentEF == nil {
        configuration := NewCollectionConfigurationFromMap(model.ConfigurationJSON)
        autoWiredContentEF, _ := BuildContentEFFromConfig(configuration)
        contentEF = autoWiredContentEF
    }
    // Auto-wire dense EF: derive from contentEF if possible
    ef := req.embeddingFunction
    if ef == nil {
        if unwrapper, ok := contentEF.(embeddingspkg.EmbeddingFunctionUnwrapper); ok {
            ef = unwrapper.UnwrapEmbeddingFunction()
        } else if denseFromContent, ok := contentEF.(embeddingspkg.EmbeddingFunction); ok {
            ef = denseFromContent
        }
    }
    // ... upsert state, build collection ...
}
```

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify v1.x |
| Config file | None (build tags in file headers) |
| Quick run command | `go test -v -tags=basicv2 -run TestEmbedded ./pkg/api/v2/...` |
| Full suite command | `make test` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SC-1 | embeddedCollection struct includes contentEF field | unit | `go test -v -tags=basicv2 -run TestEmbeddedCollection_ContentEF -count=1 ./pkg/api/v2/` | -- Wave 0 |
| SC-2 | buildEmbeddedCollection wires contentEF | unit | `go test -v -tags=basicv2 -run TestEmbeddedBuild_ContentEF -count=1 ./pkg/api/v2/` | -- Wave 0 |
| SC-3 | Close() handles contentEF with sharing detection | unit | `go test -v -tags=basicv2 -run TestEmbeddedCollection_Close.*Content -count=1 ./pkg/api/v2/` | -- Wave 0 |
| SC-4 | Fork() does NOT propagate contentEF (D-01) | unit | N/A -- Fork returns unsupported error, no change needed | Existing |
| SC-5 | GetCollection respects WithContentEmbeddingFunctionGet | unit | `go test -v -tags=basicv2 -run TestEmbeddedGetCollection.*ContentEF -count=1 ./pkg/api/v2/` | -- Wave 0 |
| SC-6 | GetCollection auto-wires contentEF from config | unit | `go test -v -tags=basicv2 -run TestEmbeddedGetCollection.*AutoWire -count=1 ./pkg/api/v2/` | -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -v -tags=basicv2 -run TestEmbedded -count=1 ./pkg/api/v2/`
- **Per wave merge:** `make test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] New test functions for embedded contentEF lifecycle in `close_review_test.go` or `client_local_embedded_test.go`
- [ ] Tests for Close() sharing detection scenarios (unwrapper case, dual-interface case, independent case)
- [ ] Tests for GetCollection auto-wiring with contentEF
- [ ] Tests for GetCollection with explicit `WithContentEmbeddingFunctionGet`

## Existing Infrastructure Inventory

All required helper functions exist and are accessible from the embedded client file (same `v2` package):

| Function | File | Purpose |
|----------|------|---------|
| `wrapContentEFCloseOnce()` | ef_close_once.go:211-219 | Close-once wrapper for contentEF |
| `wrapEFCloseOnce()` | ef_close_once.go:198-209 | Close-once wrapper for dense EF |
| `unwrapCloseOnceEF()` | client_http.go:866-876 | Unwrap close-once for sharing detection |
| `BuildContentEFFromConfig()` | configuration.go:233-245 | Auto-wire contentEF from collection config |
| `BuildEmbeddingFunctionFromConfig()` | configuration.go:196-223 | Auto-wire dense EF from config |
| `safeCloseEF()` | close_logging.go:18-25 | Close with panic recovery |
| `reportClosePanic()` | close_logging.go:10-14 | Format panic for Close errors |
| `WithContentEmbeddingFunctionGet` | client.go:199-206 | Option for explicit contentEF in GetCollection |

| Mock Type | File | Purpose |
|-----------|------|---------|
| `mockCloseableEF` | ef_close_once_test.go:20-50 | Dense EF with atomic close counter |
| `mockCloseableContentEF` | ef_close_once_test.go:53-71 | Content EF with atomic close counter |
| `mockSharedContentAdapter` | close_review_test.go:20-44 | Content adapter wrapping dense EF (unwrapper) |
| `mockDualEF` | ef_close_once_test.go:119-132 | Both EmbeddingFunction + ContentEmbeddingFunction |
| `mockPanickingCloseContentEF` | ef_close_once_test.go:850-856 | Content EF that panics on Close |

## Design Recommendation: Keep contentEF GetCollection-Only

HTTP `CreateCollection` does NOT set `contentEmbeddingFunction` on the result `CollectionImpl` (line 344-355 of `client_http.go`). ContentEF is only wired in `GetCollection`. Since `CreateCollectionOp` struct lacks a `contentEmbeddingFunction` field, and this is under Claude's discretion (CONTEXT.md), the recommendation is to keep the embedded path identical: contentEF is wired only through `GetCollection` and auto-wiring. This avoids expanding the API surface unnecessarily.

## Design Recommendation: Test Organization

Add embedded contentEF close/lifecycle tests to `close_review_test.go` (which already tests both `CollectionImpl` and `embeddedCollection` Close scenarios). Add GetCollection auto-wiring tests to `client_local_embedded_test.go` (which already tests embedded GetCollection/GetOrCreateCollection). This keeps tests co-located with related scenarios.

## Open Questions

1. **GetOrCreateCollection contentEF propagation**
   - What we know: The embedded `GetOrCreateCollection` builds `getOptions` from `req.embeddingFunction` only. `CreateCollectionOp` has no `contentEmbeddingFunction` field, so there is nothing to propagate.
   - What's unclear: If a future phase adds contentEF to `CreateCollectionOp`, this path will need updating.
   - Recommendation: No action needed now. Auto-wiring in GetCollection (D-02) covers the embedded path. Document the gap as a future consideration if `CreateCollectionOp` gains a contentEF field.

## Sources

### Primary (HIGH confidence)
- Direct code reading of `pkg/api/v2/collection_http.go` (HTTP reference implementation)
- Direct code reading of `pkg/api/v2/client_http.go` (GetCollection auto-wiring)
- Direct code reading of `pkg/api/v2/client_local_embedded.go` (current embedded implementation)
- Direct code reading of `pkg/api/v2/ef_close_once.go` (close-once infrastructure)
- Direct code reading of `pkg/api/v2/configuration.go` (BuildContentEFFromConfig)
- Direct code reading of `pkg/api/v2/client.go` (GetCollectionOp, WithContentEmbeddingFunctionGet)
- Direct code reading of `pkg/api/v2/close_logging.go` (safeCloseEF, reportClosePanic)
- GitHub Issue #472 (gap analysis)
- Phase 18 CONTEXT.md (locked decisions)
- Phase 11 CONTEXT.md (close-once wrapper decisions)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - no new dependencies, all infrastructure exists in-package
- Architecture: HIGH - exact pattern to mirror exists in HTTP client, line-by-line reference available
- Pitfalls: HIGH - thoroughly mapped from HTTP implementation and snapshot pattern analysis

**Research date:** 2026-04-02
**Valid until:** 2026-05-02 (stable internal codebase patterns)
