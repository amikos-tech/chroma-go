# Phase 19: Embedded Client EF Lifecycle Hardening - Research

**Researched:** 2026-04-06
**Domain:** Go concurrency, sync primitives, embedding function lifecycle management
**Confidence:** HIGH

## Summary

This phase fixes six distinct robustness gaps in the embedded client's embedding function (EF) lifecycle management, adds close-once wrapping for defense-in-depth, and introduces structured logging for observability parity with the HTTP client. All fixes mirror established patterns already implemented in the HTTP client path (`collection_http.go`, `client_http.go`).

The codebase has mature infrastructure for all required changes: `wrapEFCloseOnce` / `wrapContentEFCloseOnce` wrappers, `closeEmbeddingFunctions` with sharing detection, `isDenseEFSharedWithContent` for identity comparison, and the `pkg/logger` interface with `NoopLogger` default. The test framework includes comprehensive mock types (`mockCloseableEF`, `mockCloseableContentEF`, `mockSharedContentAdapter`, `mockDualEF`, etc.) and a `newEmbeddedClientForRuntime` helper that constructs `embeddedLocalClient` instances with scripted or memory-backed runtimes.

**Primary recommendation:** Apply each fix as a targeted, self-contained change to the specific function identified in CONTEXT.md, following the HTTP client reference implementation line-by-line. No new packages, no new interfaces, no architectural changes.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** GetCollection auto-wiring uses check-and-set under a full write lock (Lock(), not RLock()). The lock spans the entire check-nil + build + assign cycle (wide lock).
- **D-02:** Wide lock is acceptable because concurrent GetCollection calls for the same collection are not a real-world scenario.
- **D-03:** buildEmbeddedCollection wraps both denseEF and contentEF in close-once wrappers (wrapEFCloseOnce / wrapContentEFCloseOnce), mirroring the HTTP client pattern in collection_http.go.
- **D-04:** Add a WithLogger option to PersistentClient that accepts the existing pkg/logger interface. When set, auto-wire and close errors route through the injected logger (structured). When unset, fall back to stderr (current behavior).
- **D-05:** embeddedLocalClient.Close() iterates all collectionState entries and closes their EFs before clearing the map.
- **D-06:** deleteCollectionState closes EFs first (with sharing detection), then removes the map entry.
- **D-07:** localDeleteCollectionFromCache adds a type switch case for *embeddedCollection to close EFs via the same sharing detection logic before removing from cache.
- **D-08:** isDenseEFSharedWithContent unwraps both denseEF and contentEF before comparing.
- **D-09:** Auto-wired EF is only assigned when the build error is nil.

### Claude's Discretion
- Internal helper decomposition and method ordering
- Exact log message format and severity levels
- Whether to refactor shared cleanup logic into a common helper or keep it inline per type

### Deferred Ideas (OUT OF SCOPE)
None
</user_constraints>

## Project Constraints (from CLAUDE.md)

- All new features target V2 API (`/pkg/api/v2/`)
- Use `testify` for assertions
- Run `make lint` before committing
- NEVER panic in production code; use panic recovery where necessary
- Use conventional commits
- Use `basicv2` build tag for V2 tests
- Do not leave too many or too verbose comments

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `sync` (stdlib) | go stdlib | RWMutex, Once, atomic for concurrency control | Already used throughout `embeddedLocalClient` and `closeOnceState` [VERIFIED: codebase grep] |
| `sync/atomic` | go stdlib | Atomic bool/int for ownership flags and close counters | Already used for `ownsEF atomic.Bool` and test close counters [VERIFIED: codebase grep] |
| `pkg/logger` | internal | Structured logging interface with NoopLogger default | Already used in HTTP client, has Logger interface, Field types, and NoopLogger [VERIFIED: pkg/logger/logger.go] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/stretchr/testify` | existing | Test assertions (require/assert) | All test code [VERIFIED: codebase imports] |
| `github.com/pkg/errors` | existing | Error wrapping (Wrap/Wrapf) | Production error paths [VERIFIED: codebase imports] |

No new dependencies are needed. All required infrastructure exists in the codebase.

## Architecture Patterns

### Files to Modify

```
pkg/api/v2/
  client_local_embedded.go       # TOCTOU fix, Close() cleanup, deleteCollectionState cleanup, logger field
  client_local.go                # WithPersistentLogger option, pass logger to embeddedLocalClient
  close_logging.go               # Symmetric unwrapping in isDenseEFSharedWithContent, add logger-aware variants
  client_http.go                 # localDeleteCollectionFromCache: add *embeddedCollection type switch
                                 # GetCollection/GetOrCreateCollection: add build error guard
  client_local_embedded_test.go  # Tests for TOCTOU, Close(), delete cleanup, close-once wrapping, logger
  close_review_test.go           # Tests for symmetric unwrapping, delete-from-cache embedded type
```

### Pattern 1: Wide Lock for TOCTOU Fix (D-01)

**What:** Replace the RLock + release + conditional auto-wire + upsert pattern with a single write-locked block that checks, builds, and assigns atomically.

**Current code (TOCTOU-vulnerable):**
```go
// client_local_embedded.go:524-564
client.collectionStateMu.RLock()
if s := client.collectionState[model.ID]; s != nil {
    hasContentEF = s.contentEmbeddingFunction != nil
    hasEF = s.embeddingFunction != nil
}
client.collectionStateMu.RUnlock()
// GAP: another goroutine can auto-wire between here
if contentEF == nil && !hasContentEF {
    autoWiredContentEF, buildErr := BuildContentEFFromConfig(configuration)
    // ...
    contentEF = autoWiredContentEF
}
```
[VERIFIED: client_local_embedded.go lines 524-564]

**Fixed pattern:**
```go
client.collectionStateMu.Lock()
s := client.collectionState[model.ID]
if s == nil {
    s = &embeddedCollectionState{}
    client.collectionState[model.ID] = s
}
if contentEF == nil && s.contentEmbeddingFunction == nil {
    autoWiredContentEF, buildErr := BuildContentEFFromConfig(configuration)
    if buildErr != nil {
        // log error
    } else {
        contentEF = autoWiredContentEF  // D-09: only assign on nil error
    }
}
// ... same for denseEF
if contentEF != nil { s.contentEmbeddingFunction = contentEF }
if ef != nil { s.embeddingFunction = ef }
snapshot := *s  // copy for building collection
client.collectionStateMu.Unlock()
```

### Pattern 2: Close-Once Wrapping in buildEmbeddedCollection (D-03)

**What:** Wrap EFs before storing them in the collection struct, matching `client_http.go:458-459`.

**HTTP reference (lines 458-459):**
```go
embeddingFunction:        wrapEFCloseOnce(ef),
contentEmbeddingFunction: wrapContentEFCloseOnce(contentEF),
```
[VERIFIED: client_http.go lines 458-459]

**Fix in buildEmbeddedCollection:** Apply the same wrapping at `client_local_embedded.go:816-817`:
```go
embeddingFunction:        wrapEFCloseOnce(snapshot.embeddingFunction),
contentEmbeddingFunction: wrapContentEFCloseOnce(snapshot.contentEmbeddingFunction),
```

### Pattern 3: EF Cleanup in deleteCollectionState (D-06)

**What:** Close EFs before removing the map entry, using `closeEmbeddingFunctions` which already handles sharing detection.

**Current code:**
```go
func (client *embeddedLocalClient) deleteCollectionState(collectionID string) {
    if collectionID == "" { return }
    client.collectionStateMu.Lock()
    defer client.collectionStateMu.Unlock()
    delete(client.collectionState, collectionID)
}
```
[VERIFIED: client_local_embedded.go lines 681-688]

**Fixed pattern:**
```go
func (client *embeddedLocalClient) deleteCollectionState(collectionID string) {
    if collectionID == "" { return }
    client.collectionStateMu.Lock()
    state := client.collectionState[collectionID]
    delete(client.collectionState, collectionID)
    client.collectionStateMu.Unlock()
    if state != nil {
        if err := closeEmbeddingFunctions(state.embeddingFunction, state.contentEmbeddingFunction); err != nil {
            // log via client.logger or stderr fallback
        }
    }
}
```

### Pattern 4: Client Close Iterates collectionState (D-05)

**What:** Add EF cleanup to `embeddedLocalClient.Close()`, matching `APIClientV2.Close()` which iterates `collectionCache`.

**Current code (lines 648-665):**
```go
func (client *embeddedLocalClient) Close() error {
    var errs []error
    if client.state != nil {
        if err := client.state.Close(); err != nil {
            errs = append(errs, err)
        }
    }
    if client.embedded != nil {
        if err := client.embedded.Close(); err != nil {
            errs = append(errs, errors.Wrap(err, "error closing embedded local runtime"))
        }
    }
    // MISSING: collectionState cleanup
    ...
}
```
[VERIFIED: client_local_embedded.go lines 648-665]

**Fixed pattern:** Before closing state and runtime, iterate collectionState entries and close their EFs.

### Pattern 5: localDeleteCollectionFromCache Embedded Type (D-07)

**What:** Add `*embeddedCollection` type switch case to `localDeleteCollectionFromCache` at `client_http.go:786`.

**Current code only handles `*CollectionImpl`:**
```go
impl, ok := deleted.(*CollectionImpl)
if ok && impl.ownsEF.Load() {
    // ownership transfer / close logic
}
```
[VERIFIED: client_http.go lines 787-810]

**Fix:** Add a second branch for `*embeddedCollection`. Since Fork is unsupported in embedded mode, no ownership transfer is needed -- just close directly when `ownsEF` is true:
```go
case *embeddedCollection:
    if ec.ownsEF.Load() {
        toClose = deleted
    }
```

### Pattern 6: Symmetric Unwrapping (D-08)

**What:** One-line fix in `isDenseEFSharedWithContent` fallback path.

**Current code (close_logging.go:41-43):**
```go
if efFromContent, ok := contentEF.(embeddings.EmbeddingFunction); ok {
    return efFromContent == unwrapped
}
```
[VERIFIED: close_logging.go lines 41-43]

**Fix:**
```go
if efFromContent, ok := contentEF.(embeddings.EmbeddingFunction); ok {
    return unwrapCloseOnceEF(efFromContent) == unwrapped
}
```

### Pattern 7: Logger Injection (D-04)

**What:** Add a `logger` field to `embeddedLocalClient`, set from a new `WithPersistentLogger` option.

**HTTP reference:** `BaseAPIClient` has `logger logger.Logger` field at `client.go:674`. The `WithLogger` function is a `ClientOption` at `client.go:837`. [VERIFIED: codebase grep]

**Embedded approach:**
1. Add `logger logger.Logger` field to `localClientConfig`
2. Add `WithPersistentLogger(l logger.Logger) PersistentClientOption` to `client_local.go`
3. Pass logger to `embeddedLocalClient` in `newEmbeddedLocalClient`
4. Add `logger` field to `embeddedLocalClient` with `NoopLogger` default
5. Update callsites: check `client.logger != nil` (or always use it since defaulted)

### Anti-Patterns to Avoid
- **Holding the mutex during EF close:** Close is potentially slow (network teardown). Copy data under lock, release, then close outside the lock. [VERIFIED: HTTP client does this at client_http.go:781-812]
- **Assigning auto-wired EF on build error:** Even though current factories return nil on error, the nil-guard makes the contract explicit. [VERIFIED: issue #485]
- **Double-wrapping close-once:** `wrapEFCloseOnce` already checks if the input is already a `*closeOnceEF` or `*closeOnceContentEF` and returns it unchanged. [VERIFIED: ef_close_once.go lines 198-209]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| EF close sharing detection | Custom comparison | `isDenseEFSharedWithContent()` + `closeEmbeddingFunctions()` | Already handles unwrapping, nil checks, sharing detection [VERIFIED: close_logging.go:33-66] |
| Idempotent close | Manual bool flag | `wrapEFCloseOnce()` / `wrapContentEFCloseOnce()` | sync.Once-based with use-after-close guards [VERIFIED: ef_close_once.go] |
| Safe EF close | Direct Close() call | `safeCloseEF()` | Panic recovery built in [VERIFIED: close_logging.go:20-28] |
| Structured logging | fmt.Fprintf(os.Stderr, ...) | `pkg/logger.Logger` interface | Already integrated in HTTP client, has noop default [VERIFIED: pkg/logger/logger.go] |

## Common Pitfalls

### Pitfall 1: Holding Lock During IO
**What goes wrong:** Calling `closeEmbeddingFunctions` while holding `collectionStateMu` blocks all GetCollection calls during slow EF teardown.
**Why it happens:** Natural tendency to keep cleanup inside the critical section.
**How to avoid:** Copy references under lock, release lock, then close. The HTTP client's `localDeleteCollectionFromCache` demonstrates this at `client_http.go:781-812` -- it copies `toClose` under lock, then closes outside. [VERIFIED: codebase]
**Warning signs:** Test timeouts or deadlocks in concurrent Close+GetCollection scenarios.

### Pitfall 2: Forgetting to Update Both Auto-Wire Paths
**What goes wrong:** Fixing the error guard in embedded GetCollection but not HTTP GetCollection (or vice versa).
**Why it happens:** Issue #485 affects BOTH client types.
**How to avoid:** Apply the `if buildErr != nil { log } else { assign }` pattern to all four auto-wire callsites: embedded GetCollection (2 sites), HTTP GetCollection (2 sites), and HTTP GetOrCreateCollection (1 site). [VERIFIED: client_http.go lines 426-446, 533-538]
**Warning signs:** Different behavior between HTTP and embedded clients.

### Pitfall 3: Snapshot vs. Reference in buildEmbeddedCollection
**What goes wrong:** Wrapping the snapshot copy in close-once but not the state reference, or vice versa.
**Why it happens:** `upsertCollectionState` returns a snapshot (value copy), but the state map holds the original pointer.
**How to avoid:** Wrap at the point where the EF is assigned to the `embeddedCollection` struct (the collection holds the wrapped reference). The state map should hold the unwrapped original so that sharing detection via `isDenseEFSharedWithContent` works correctly before wrapping. [VERIFIED: buildEmbeddedCollection lines 787-823]
**Warning signs:** `isDenseEFSharedWithContent` returns false for EFs that should be shared.

### Pitfall 4: Race Between deleteCollectionState and Close
**What goes wrong:** `deleteCollectionState` closes an EF, then `embeddedLocalClient.Close()` tries to close the same EF from the state map.
**Why it happens:** Both paths iterate/access collectionState.
**How to avoid:** `Close()` should take the write lock, copy the map, clear it, then release the lock and close. `deleteCollectionState` deletes the entry first under lock, then closes. After `Close()` clears the map, `deleteCollectionState` finds nothing to delete -- no double-close. With close-once wrapping on the collection level, even if both paths race, the second close is a no-op. [VERIFIED: close-once wrapper sync.Once guarantees at ef_close_once.go:51-62]

### Pitfall 5: Type Switch Ordering in localDeleteCollectionFromCache
**What goes wrong:** Adding `*embeddedCollection` case but using `if-else` instead of type switch, or placing it after a default case.
**Why it happens:** The existing code uses `impl, ok := deleted.(*CollectionImpl)` which is not a type switch.
**How to avoid:** Convert to a proper type switch with both `*CollectionImpl` and `*embeddedCollection` cases, or add a separate type assertion block. [VERIFIED: client_http.go:787]

## Code Examples

### Close-Once Wrapping in buildEmbeddedCollection
```go
// Source: Mirrors client_http.go:448-461
collection := &embeddedCollection{
    name:                     model.Name,
    id:                       model.ID,
    // ... other fields from snapshot ...
    embeddingFunction:        wrapEFCloseOnce(snapshot.embeddingFunction),
    contentEmbeddingFunction: wrapContentEFCloseOnce(snapshot.contentEmbeddingFunction),
    client:                   client,
}
```

### Logger-Aware Error Logging
```go
// Source: Mirrors client_http.go:428 pattern
if client.logger != nil {
    client.logger.Warn("failed to auto-wire content embedding function",
        logger.String("collection", model.Name),
        logger.ErrorField("error", buildErr))
} else {
    logAutoWireBuildErrorToStderr(model.Name, "content embedding function", buildErr)
}
```

### Build Error Guard
```go
// Source: Issue #485 fix
autoWiredContentEF, buildErr := BuildContentEFFromConfig(configuration)
if buildErr != nil {
    // log the error
} else {
    contentEF = autoWiredContentEF
}
```

### Symmetric Unwrapping
```go
// Source: close_logging.go fix for issue #489
func isDenseEFSharedWithContent(denseEF embeddings.EmbeddingFunction, contentEF embeddings.ContentEmbeddingFunction) bool {
    if denseEF == nil || contentEF == nil {
        return false
    }
    unwrapped := unwrapCloseOnceEF(denseEF)
    if unwrapper, ok := contentEF.(embeddings.EmbeddingFunctionUnwrapper); ok {
        return unwrapper.UnwrapEmbeddingFunction() == unwrapped
    }
    if efFromContent, ok := contentEF.(embeddings.EmbeddingFunction); ok {
        return unwrapCloseOnceEF(efFromContent) == unwrapped  // symmetric unwrap
    }
    return false
}
```

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify (assert/require) |
| Config file | Makefile (build tags) |
| Quick run command | `go test -tags=basicv2 -run TestEmbedded -count=1 ./pkg/api/v2/...` |
| Full suite command | `make test` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SC-01 | TOCTOU race: concurrent GetCollection auto-wires once | unit | `go test -tags=basicv2 -run TestEmbeddedGetCollection_ConcurrentAutoWire -count=1 ./pkg/api/v2/...` | Wave 0 |
| SC-02 | deleteCollectionState closes EFs before removing | unit | `go test -tags=basicv2 -run TestEmbeddedDeleteCollectionState_ClosesEFs -count=1 ./pkg/api/v2/...` | Wave 0 |
| SC-03 | embeddedLocalClient.Close() iterates collectionState | unit | `go test -tags=basicv2 -run TestEmbeddedLocalClient_Close_CleansUpCollectionState -count=1 ./pkg/api/v2/...` | Wave 0 |
| SC-04 | localDeleteCollectionFromCache handles *embeddedCollection | unit | `go test -tags=basicv2 -run TestDeleteCollectionFromCache_EmbeddedCollection -count=1 ./pkg/api/v2/...` | Wave 0 |
| SC-05 | buildEmbeddedCollection wraps EFs in close-once | unit | `go test -tags=basicv2 -run TestEmbeddedBuildCollection_CloseOnceWrapping -count=1 ./pkg/api/v2/...` | Wave 0 |
| SC-06 | isDenseEFSharedWithContent unwraps both sides | unit | `go test -tags=basicv2 -run TestIsDenseEFSharedWithContent_SymmetricUnwrap -count=1 ./pkg/api/v2/...` | Wave 0 |
| SC-07 | Auto-wired EF only assigned on nil error | unit | `go test -tags=basicv2 -run TestEmbeddedGetCollection_BuildErrorGuard -count=1 ./pkg/api/v2/...` | Wave 0 |
| SC-08 | Structured logger receives auto-wire and close errors | unit | `go test -tags=basicv2 -run TestEmbeddedClient_LoggerReceivesErrors -count=1 ./pkg/api/v2/...` | Wave 0 |
| SC-09 | All existing tests pass (no regressions) | regression | `make test` | Existing |

### Sampling Rate
- **Per task commit:** `go test -tags=basicv2 -count=1 ./pkg/api/v2/...`
- **Per wave merge:** `make test`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] All test functions listed above are new (Wave 0 creates them)
- [ ] Existing mock types (`mockCloseableEF`, `mockCloseableContentEF`, `capturingLogger`, etc.) are sufficient -- no new mocks needed beyond what exists
- [ ] `newEmbeddedClientForRuntime` helper already exists for test setup

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `unwrapCloseOnceEF` is accessible from `close_logging.go` (same package) | Pattern 6 | LOW -- both are in package `v2`, verified by existing usage at close_logging.go:37 |
| A2 | `localDeleteCollectionFromCache` on `APIClientV2` can type-assert `*embeddedCollection` | Pattern 5 | LOW -- both types are in package `v2`, no import cycle. Verified: embeddedCollection defined at client_local_embedded.go:825 |
| A3 | `PersistentClient` does not currently have a logger option | Pattern 7 | LOW -- verified: `localClientConfig` struct at client_local.go:97-113 has no logger field |

**All claims verified or cited; no user confirmation needed for the 3 items above (they are LOW risk architecture observations, not design decisions).**

## Open Questions

1. **HTTP client auto-wire error guard scope**
   - What we know: Issue #485 explicitly says "Apply to both embedded and HTTP client auto-wire paths for consistency."
   - What's unclear: Whether GetOrCreateCollection's auto-wire path at `client_http.go:533` should also get the error guard.
   - Recommendation: Apply to all auto-wire callsites for consistency. The CONTEXT.md decisions are specifically about the embedded client, but applying the same defensive pattern to HTTP is a trivial one-line change per site and matches the issue request.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies identified). This phase is purely code/config changes to existing Go files within the `pkg/api/v2` package. No new tools, services, or external dependencies.

## Security Domain

This phase does not introduce new attack surface. It hardens existing resource lifecycle management (EF close, cleanup on delete/shutdown). The fixes reduce risk of resource leaks, which is an availability concern but not an authentication/authorization/input-validation concern.

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | N/A |
| V3 Session Management | no | N/A |
| V4 Access Control | no | N/A |
| V5 Input Validation | no | N/A (no new inputs) |
| V6 Cryptography | no | N/A |

No security-specific threat patterns apply to this phase.

## Sources

### Primary (HIGH confidence)
- `pkg/api/v2/client_local_embedded.go` -- current embedded client implementation (all line references verified)
- `pkg/api/v2/client_http.go` -- HTTP client reference implementation (all line references verified)
- `pkg/api/v2/ef_close_once.go` -- close-once wrapper infrastructure (full file read)
- `pkg/api/v2/close_logging.go` -- sharing detection and logging helpers (full file read)
- `pkg/api/v2/client_local.go` -- PersistentClient config and options (verified no logger option exists)
- `pkg/logger/logger.go` -- Logger interface definition (full file read)
- GitHub Issues #484, #485, #488, #489 -- bug reports and proposed fixes (read via `gh issue view`)

### Secondary (MEDIUM confidence)
- `pkg/api/v2/client_local_embedded_test.go` -- existing test patterns and helpers (verified)
- `pkg/api/v2/ef_close_once_test.go` -- mock types and close lifecycle tests (verified)
- `pkg/api/v2/close_review_test.go` -- delete-from-cache and logger test patterns (verified)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all libraries already in use, no new dependencies
- Architecture: HIGH - every pattern mirrors an existing HTTP client implementation with verified line references
- Pitfalls: HIGH - derived from actual code analysis and issue reports, not theoretical
- Test patterns: HIGH - verified existing mock types and test helpers are sufficient

**Research date:** 2026-04-06
**Valid until:** 2026-05-06 (stable internal codebase, 30-day validity)
