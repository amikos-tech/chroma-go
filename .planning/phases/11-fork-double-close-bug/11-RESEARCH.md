# Phase 11: Fork Double-Close Bug - Research

**Researched:** 2026-03-26
**Domain:** Go resource lifecycle management (shared pointer ownership in collection Fork/Close)
**Confidence:** HIGH

## Summary

The Fork() method in both HTTP and embedded collection implementations copies embedding function pointers by reference into the forked collection. Both original and forked collections end up in the client's collection cache. When `APIClientV2.Close()` iterates the cache calling `Close()` on each collection, the same underlying EF resource gets closed twice. For EFs that hold native resources (e.g., ONNX Runtime), this causes panics or use-after-close errors.

The fix requires two complementary mechanisms: (1) an `ownsEF` flag on collection structs that gates EF teardown in Close(), and (2) a `sync.Once`-based close wrapper around the EF that makes Close() idempotent as a defence-in-depth layer. The codebase already has a close-once pattern in `DefaultEmbeddingFunction` using `sync.Once` + `atomic.Int32` that validates this approach.

**Primary recommendation:** Add `ownsEF bool` field to both `CollectionImpl` and `embeddedCollection`. Fork() sets `ownsEF = false` on the forked copy. Close() checks `ownsEF` before calling EF teardown. Additionally, wrap shared EFs in a thin close-once adapter that implements `io.Closer`, `EmbeddingFunction`, `ContentEmbeddingFunction`, and `EmbeddingFunctionUnwrapper` by delegation, making repeated Close() calls return nil (or a clean error) instead of panicking.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Use an **owner flag + close-once wrapper** combo. Add `ownsEF bool` field to `CollectionImpl` and `embeddedCollection`. Fork() sets it `false`; constructors (CreateCollection, GetCollection, GetOrCreateCollection) set it `true`. Close() gates EF teardown on `ownsEF`.
- **D-02:** Wrap shared EFs in a thin `sync.Once`-based close wrapper that makes Close() idempotent. This is a defence-in-depth layer: if a user manually closes the original collection while a fork is still live, the fork won't panic on subsequent operations -- the EF returns a clean error instead of undefined behavior.
- **D-03:** The `ownsEF` flag is set once at creation time and never changes. Forked collections can use the EF for all embedding operations (Add, Query, Search) normally -- the flag only gates the Close() teardown path.
- **D-04:** Fix both HTTP client (`CollectionImpl.Fork` + `CollectionImpl.Close`) and embedded client (`embeddedCollection.Fork` + `embeddedCollection.Close`). The bug is structurally identical in both paths. The embedded path is arguably worse because `embeddedCollection.Close()` has zero sharing guard today.
- **D-05:** Forked collection's Close() skips EF teardown (owner flag) but still runs any other cleanup the collection might need. It is not a full no-op -- only the EF close is gated.
- **D-06:** The close-once wrapper on the EF is the safety net for edge cases where the original collection is closed before the fork.

### Claude's Discretion
- Close-once wrapper implementation details (struct shape, interface delegation)
- Test structure and naming
- Whether the owner flag field is named `ownsEF`, `isOwner`, or similar

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

## Architecture Patterns

### Bug Anatomy

The double-close bug manifests through this call chain:

1. `CreateCollection()` creates a `CollectionImpl` with EF pointers and adds it to `collectionCache`
2. `collection.Fork()` creates a new `CollectionImpl` that copies the same EF pointers (lines 415-416 of `collection_http.go`) and also adds it to `collectionCache` (line 418)
3. `client.Close()` iterates `collectionCache` and calls `Close()` on each collection
4. Both original and fork call `closer.Close()` on the same underlying EF -- double close

The embedded path is identical: `embeddedCollection.Fork()` copies the `embeddingFunction` pointer, `buildEmbeddedCollection()` adds to cache via `state.localAddCollectionToCache()`, and `embeddedLocalClient.Close()` delegates to `client.state.Close()` which is an `APIClientV2` that iterates its cache.

### Affected Structs and Methods

**HTTP path:**
- `CollectionImpl` struct (`collection_http.go:48-60`): Has `embeddingFunction` and `contentEmbeddingFunction` fields
- `CollectionImpl.Fork()` (`collection_http.go:388-420`): Copies both EF pointers directly
- `CollectionImpl.Close()` (`collection_http.go:684-710`): Has intra-collection sharing detection (Unwrapper pattern) but no cross-collection ownership guard
- `APIClientV2.Close()` (`client_http.go:693-729`): Iterates collectionCache calling Close() on each

**Embedded path:**
- `embeddedCollection` struct (`client_local_embedded.go:774-788`): Has only `embeddingFunction` (no `contentEmbeddingFunction`)
- `embeddedCollection.Fork()` (`client_local_embedded.go:1360-1399`): Copies EF pointer via snapshot, builds new collection via `buildEmbeddedCollection()`
- `embeddedCollection.Close()` (`client_local_embedded.go:1421-1429`): No sharing guard at all -- calls EF Close() unconditionally
- `embeddedLocalClient.Close()` (`client_local_embedded.go:603-620`): Delegates to `state.Close()` which is `APIClientV2.Close()` iterating cache

### Pattern 1: Owner Flag on Collection Structs

**What:** Add `ownsEF bool` field to both collection struct types. Only collections that created/acquired the EF (via CreateCollection, GetCollection, GetOrCreateCollection) are owners. Fork() produces non-owning copies.

**When to use:** In `Close()` to gate EF teardown.

**Implementation points for CollectionImpl:**
```go
type CollectionImpl struct {
    // ... existing fields ...
    embeddingFunction        embeddings.EmbeddingFunction
    contentEmbeddingFunction embeddings.ContentEmbeddingFunction
    ownsEF                   bool  // true for originals, false for forks
}
```

Fork() sets `ownsEF: false`:
```go
forkedCollection := &CollectionImpl{
    // ... existing fields ...
    embeddingFunction:        c.embeddingFunction,
    contentEmbeddingFunction: c.contentEmbeddingFunction,
    ownsEF:                   false,
}
```

Close() gates on ownsEF:
```go
func (c *CollectionImpl) Close() error {
    if !c.ownsEF {
        return nil  // fork -- skip EF teardown
    }
    // ... existing close logic (contentEF first, then denseEF with sharing check) ...
}
```

**Implementation points for embeddedCollection:**
```go
type embeddedCollection struct {
    // ... existing fields ...
    embeddingFunction embeddingspkg.EmbeddingFunction
    ownsEF            bool
}
```

Same pattern: Fork path sets `ownsEF: false`, Close() gates on it.

### Pattern 2: Close-Once Wrapper (Defence-in-Depth)

**What:** A thin wrapper struct that delegates all EF interface methods but makes Close() idempotent via `sync.Once`. After the first Close(), subsequent calls return nil. The wrapped EF operations can optionally check the closed flag and return a clean error.

**Recommended struct shape:**
```go
type closeOnceEF struct {
    ef       embeddings.EmbeddingFunction
    once     sync.Once
    closed   atomic.Bool
    closeErr error  // captured from first close
}

func (w *closeOnceEF) Close() error {
    w.once.Do(func() {
        w.closed.Store(true)
        if closer, ok := w.ef.(io.Closer); ok {
            w.closeErr = closer.Close()
        }
    })
    return w.closeErr
}

func (w *closeOnceEF) EmbedDocuments(ctx context.Context, docs []string) ([]embeddings.Embedding, error) {
    if w.closed.Load() {
        return nil, errors.New("embedding function is closed")
    }
    return w.ef.EmbedDocuments(ctx, docs)
}

func (w *closeOnceEF) EmbedQuery(ctx context.Context, doc string) (embeddings.Embedding, error) {
    if w.closed.Load() {
        return nil, errors.New("embedding function is closed")
    }
    return w.ef.EmbedQuery(ctx, doc)
}
```

A similar wrapper is needed for `ContentEmbeddingFunction`. The wrapper must also implement `EmbeddingFunctionUnwrapper` if the inner EF does, so that `CollectionImpl.Close()` intra-collection sharing detection still works:

```go
func (w *closeOnceEF) UnwrapEmbeddingFunction() embeddings.EmbeddingFunction {
    if unwrapper, ok := w.ef.(embeddings.EmbeddingFunctionUnwrapper); ok {
        return unwrapper.UnwrapEmbeddingFunction()
    }
    return w.ef
}
```

**Where to place:** In `pkg/api/v2/` as an internal helper (e.g., `ef_close_once.go`), not in the `pkg/embeddings/` package -- this is collection lifecycle infrastructure, not embedding function infrastructure.

### Pattern 3: Setting ownsEF=true in Constructors

All collection-creating paths in both clients must set `ownsEF = true`:

**HTTP client (`client_http.go`):**
- `CreateCollection()` around line 345-356: Set `ownsEF: true` on the `CollectionImpl` literal
- `GetCollection()` around line 440-461: Set `ownsEF: true` on the `CollectionImpl` literal
- `GetOrCreateCollection()` delegates to `CreateCollection()` -- covered

**Embedded client (`client_local_embedded.go`):**
- `buildEmbeddedCollection()` around line 758-771: Set `ownsEF: true` on the `embeddedCollection` literal. This is used by all creation paths AND by Fork() -- so Fork() must override to `false` AFTER build, or the build function needs a parameter.

Important nuance: `embeddedCollection.Fork()` calls `buildEmbeddedCollection()` which creates with `ownsEF: true`. The Fork() caller must then set `ownsEF = false` on the returned collection. Options:
1. Set the field after `buildEmbeddedCollection()` returns (field is exported within package)
2. Add a parameter to `buildEmbeddedCollection()`
3. Set `ownsEF = true` only in direct Create/Get paths, not in `buildEmbeddedCollection()`

Option 1 is simplest. The fork path already has access to the returned `*embeddedCollection` before it returns it.

### Anti-Patterns to Avoid
- **Reference counting:** Over-engineered for this use case. Fork creates exactly one additional reference. The owner flag is sufficient.
- **Removing forked collections from cache:** Would break the lifecycle contract -- forked collections need to be tracked for other close operations.
- **Mutex-protected close:** The `sync.Once` pattern is already the standard Go idiom for this. Don't reinvent it with manual mutex + bool.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Idempotent close | Manual mutex + bool guard | `sync.Once` + `atomic.Bool` | sync.Once is the standard Go pattern; race-free by design |
| Interface delegation | Code generation | Hand-written delegation (3-4 methods) | Only a handful of methods, code gen is overkill |

## Common Pitfalls

### Pitfall 1: EmbeddingFunctionUnwrapper Breakage
**What goes wrong:** The close-once wrapper breaks the `EmbeddingFunctionUnwrapper` interface detection in `CollectionImpl.Close()`, causing the intra-collection sharing check (contentEF wraps denseEF) to fail. This could re-introduce double-close within a single collection.
**Why it happens:** `CollectionImpl.Close()` type-asserts `c.contentEmbeddingFunction` to `EmbeddingFunctionUnwrapper` to check if it wraps `c.embeddingFunction`. If both are wrapped in close-once wrappers, the identity check (`unwrapper.UnwrapEmbeddingFunction() == c.embeddingFunction`) fails because the wrapper addresses differ.
**How to avoid:** The close-once wrapper for ContentEmbeddingFunction must implement `EmbeddingFunctionUnwrapper` and delegate to the inner's unwrap. The identity comparison should compare the inner EFs, not the wrappers. Alternatively, since the owner flag gates the entire Close() path for forks, the close-once wrapper only needs to handle the original collection's Close() -- and the original's EFs are wrapped at creation time before being stored, so the identity comparison works as long as wrapping happens consistently.
**Warning signs:** Test that has a content adapter wrapping a dense EF where both are close-once wrapped -- verify only one close call happens.

### Pitfall 2: Embedded Collection buildEmbeddedCollection Sets ownsEF
**What goes wrong:** `buildEmbeddedCollection()` is called by both normal creation paths AND Fork(). If it always sets `ownsEF = true`, the Fork path must remember to override it.
**Why it happens:** Shared builder function serves both owners and non-owners.
**How to avoid:** Either (a) set `ownsEF = true` in `buildEmbeddedCollection()` and override in Fork(), or (b) pass a parameter. Option (a) is cleaner since Fork is the only non-owner path.
**Warning signs:** Fork test passes but `client.Close()` still double-closes.

### Pitfall 3: Close-Once Wrapper Must Handle nil EF
**What goes wrong:** If the EF is nil, wrapping it in a close-once adapter introduces a non-nil wrapper around a nil inner. Code that checks `if c.embeddingFunction != nil` would think an EF exists.
**Why it happens:** Not all collections have an EF (server-side default).
**How to avoid:** Only wrap non-nil EFs. Keep the nil check before wrapping.
**Warning signs:** Nil pointer dereference when calling methods on a wrapped nil EF.

### Pitfall 4: Library Must Never Panic
**What goes wrong:** The close-once wrapper's closed-state path returns an error, but if any code path assumes EF methods never fail for non-error reasons, it could surface confusing errors to users.
**Why it happens:** Per CLAUDE.md, this library must never panic. The wrapper returning an error on use-after-close is correct behavior.
**How to avoid:** The error message should clearly indicate the EF has been closed. Tests should verify the error message is descriptive.
**Warning signs:** Test that closes the original, then tries to use the fork's EF -- should get a clean error, not a panic.

### Pitfall 5: ContentEmbeddingFunction Close-Once Wrapper
**What goes wrong:** `CollectionImpl` has both `embeddingFunction` and `contentEmbeddingFunction`. The close-once wrapper needs to handle both interface types. If only `EmbeddingFunction` is wrapped, the `contentEmbeddingFunction` can still be double-closed.
**Why it happens:** Two different interface types require two wrapper implementations.
**How to avoid:** Create wrappers for both `EmbeddingFunction` and `ContentEmbeddingFunction`. The content wrapper must also implement `EmbeddingFunctionUnwrapper` and `Closeable` interfaces.
**Warning signs:** Content EF close called twice in tests.

## Code Examples

### Existing Close-Once Pattern in DefaultEmbeddingFunction
```go
// Source: pkg/embeddings/default_ef/default_ef.go:224-249
func (e *DefaultEmbeddingFunction) Close() error {
    if atomic.LoadInt32(&e.closed) == 1 {
        return nil
    }
    initLock.Lock()
    defer initLock.Unlock()

    var closeErr error
    e.closeOnce.Do(func() {
        atomic.StoreInt32(&e.closed, 1)
        // ... cleanup logic ...
    })
    return closeErr
}
```

### Current CollectionImpl.Close() with Intra-Collection Sharing Detection
```go
// Source: pkg/api/v2/collection_http.go:684-710
func (c *CollectionImpl) Close() error {
    var firstErr error
    if c.contentEmbeddingFunction != nil {
        if closer, ok := c.contentEmbeddingFunction.(io.Closer); ok {
            firstErr = closer.Close()
        }
    }
    if c.embeddingFunction != nil {
        shared := false
        if unwrapper, ok := c.contentEmbeddingFunction.(embeddings.EmbeddingFunctionUnwrapper); ok {
            shared = unwrapper.UnwrapEmbeddingFunction() == c.embeddingFunction
        } else if ef, ok := c.contentEmbeddingFunction.(embeddings.EmbeddingFunction); ok {
            shared = ef == c.embeddingFunction
        }
        if !shared {
            if closer, ok := c.embeddingFunction.(io.Closer); ok {
                if err := closer.Close(); err != nil && firstErr == nil {
                    firstErr = err
                }
            }
        }
    }
    return firstErr
}
```

### EmbeddingFunctionUnwrapper Interface
```go
// Source: pkg/embeddings/multimodal_compat.go:8-11
type EmbeddingFunctionUnwrapper interface {
    UnwrapEmbeddingFunction() EmbeddingFunction
}
```

### embeddingFunctionContentAdapter.Close() (delegates to inner)
```go
// Source: pkg/embeddings/multimodal_compat.go:64-69
func (a *embeddingFunctionContentAdapter) Close() error {
    if c, ok := a.ef.(Closeable); ok {
        return c.Close()
    }
    return nil
}
```

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify v1 |
| Config file | none (standard go test) |
| Quick run command | `go test -tags=basicv2 -run TestFork -count=1 ./pkg/api/v2/...` |
| Full suite command | `make test` |

### Phase Requirements to Test Map

Since no formal requirement IDs were provided, mapping from success criteria:

| Criterion | Behavior | Test Type | Automated Command | File Exists? |
|-----------|----------|-----------|-------------------|-------------|
| SC-01 | Forked collections do not double-close shared EF resources when client.Close() is called | unit | `go test -tags=basicv2 -run TestForkCloseDoesNotDoubleClose -count=1 ./pkg/api/v2/...` | No -- Wave 0 |
| SC-02 | Both embeddingFunction and contentEmbeddingFunction ownership handled correctly | unit | `go test -tags=basicv2 -run TestForkOwnership -count=1 ./pkg/api/v2/...` | No -- Wave 0 |
| SC-03 | Tests cover Fork + Close lifecycle without panics or use-after-close errors | unit | `go test -tags=basicv2 -run TestForkClose -count=1 ./pkg/api/v2/...` | No -- Wave 0 |
| SC-04 | Existing fork tests continue to pass | integration | `make test` | Yes (existing) |

### Sampling Rate
- **Per task commit:** `go test -tags=basicv2 -run "TestFork|TestClose" -count=1 ./pkg/api/v2/...`
- **Per wave merge:** `make test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] Fork + Close double-close test (mock closeable EF, verify Close() called exactly once)
- [ ] Fork + Close with contentEmbeddingFunction test (both EF types, owner flag gating)
- [ ] Close-once wrapper unit tests (idempotent close, use-after-close returns error)
- [ ] Embedded collection fork + close test (same pattern, embedded path)

## Project Constraints (from CLAUDE.md)

- **Never panic in production code** -- close-once wrapper must never panic, must return clean errors
- **Use `testify` for assertions** in tests
- **Build tags:** Tests use `basicv2` tag
- **Run `make lint` before committing**
- **Conventional commits** required
- **V2 API** target (`/pkg/api/v2/`)
- **Keep things radically simple** -- owner flag + close-once wrapper is the simplest solution that covers all cases
- **Do not leave too many or too verbose comments** -- code should be self-explanatory

## Open Questions

1. **Where to put the close-once wrapper?**
   - What we know: It is collection lifecycle infrastructure, not embedding function infrastructure
   - Recommendation: `pkg/api/v2/ef_close_once.go` (internal to the v2 package)

2. **Should the close-once wrapper be applied at Fork() time or at creation time?**
   - What we know: D-02 says "wrap shared EFs" which implies at Fork() time. But wrapping at creation time (once for all future forks) is also valid.
   - Recommendation: Wrap at Fork() time. The original collection's EFs remain unwrapped (preserving existing behavior). Only forked collections get the close-once safety net. This keeps the owner flag as the primary mechanism and the wrapper as pure defence-in-depth.
   - Alternative: Wrap at creation time unconditionally. This is simpler (every collection gets the wrapper) but adds a layer of indirection to all collections, not just forks. Given the "radically simple" directive, Fork-time wrapping is preferred since it limits the blast radius.

3. **Should the wrapper check closed state on EmbedDocuments/EmbedQuery?**
   - What we know: D-02 says "the fork won't panic on subsequent operations -- the EF returns a clean error instead of undefined behavior"
   - Recommendation: Yes, check closed state and return a descriptive error. This is the defence-in-depth purpose of the wrapper.

## Sources

### Primary (HIGH confidence)
- Direct code inspection of `collection_http.go`, `client_http.go`, `client_local_embedded.go` -- all findings are from current codebase on branch `feat/phase-11-fork-double-close-bug`
- GitHub Issue #454 -- bug report confirming the problem and suggested fixes
- `pkg/embeddings/default_ef/default_ef.go` -- existing `sync.Once` + `atomic` close-once pattern
- `pkg/embeddings/multimodal_compat.go` -- `EmbeddingFunctionUnwrapper` interface and adapter Close() delegation

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies, pure Go stdlib (`sync.Once`, `sync/atomic`)
- Architecture: HIGH -- direct code inspection of all affected paths, clear understanding of the bug
- Pitfalls: HIGH -- identified all interface delegation concerns, nil-handling edge cases, and the EmbeddingFunctionUnwrapper interaction

**Research date:** 2026-03-26
**Valid until:** 2026-04-26 (stable -- internal bug fix, no external dependency changes)
