# Phase 23: ORT EF Leak Fix - Research

**Researched:** 2026-04-10
**Domain:** Embedded V2 collection creation lifecycle and default ORT embedding-function ownership
**Confidence:** HIGH (direct code and test inspection)

## Summary

Phase 23 is a narrow embedded-client lifecycle fix centered on two files:
- `pkg/api/v2/client.go`
- `pkg/api/v2/client_local_embedded.go`

Today `CreateCollectionOp.PrepareAndValidateCollectionRequest()` eagerly creates a default ORT embedding function when the caller does not provide a dense EF:

```go
defaultedDenseEF := op.embeddingFunction == nil
if defaultedDenseEF {
	ef, _, err := ort.NewDefaultEmbeddingFunction()
	...
	op.embeddingFunction = ef
}
```

Then embedded `CreateCollection()` checks whether `get_or_create` resolved to an existing collection:

```go
overrideEF := req.embeddingFunction
...
if isNewCreation {
	overrideEF = wrapEFCloseOnce(req.embeddingFunction)
	...
} else {
	overrideEF = nil
	overrideContentEF = nil
}
```

On the `isNewCreation=false` branch, the temporary request EF is dropped before `buildEmbeddedCollection(...)` runs. If that EF is the SDK-created default ORT EF, it is leaked because nothing closes it. This directly violates `EFL-01`.

**Primary recommendation:** keep the phase narrow and fix the leak in embedded `CreateCollection` itself. Add explicit provenance tracking to `CreateCollectionOp` so the embedded client can distinguish:
- SDK-created default ORT EF that must be cleaned up on the existing-collection path
- caller-provided dense EF that must never be closed here
- dual-interface contentEF promotion cases where the temporary default ORT EF was already closed during validation and must not be touched again

The implementation should stay scoped to:
- `pkg/api/v2/client.go` for provenance tracking on `CreateCollectionOp`
- `pkg/api/v2/client_local_embedded.go` for cleanup on the `isNewCreation=false` branch
- colocated `basicv2` tests in `pkg/api/v2/client_local_embedded_test.go` (and only add a helper seam in production code if required to substitute a close-counting default EF)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Phase 23 uses a narrow embedded-path fix, not a shared create-flow refactor.
- **D-02:** The fix happens in embedded `CreateCollection` when `isNewCreation=false`: if the dense EF was auto-created by the SDK as the default ORT EF for this request, it must be explicitly closed before the temporary override is discarded.
- **D-03:** Existing-collection EF precedence does not change. Existing embedded collections keep their state-backed EF/contentEF; temporary override EFs on the existing path are not adopted.
- **D-04:** Verification uses two layers:
  - one focused close-spy/unit regression around default EF creation
  - one broader existing-collection lifecycle regression proving the returned collection still preserves original state and does not adopt the temporary default
- **D-05:** Verification stays in the standard colocated `basicv2` suite. No ORT-native leak harness is required.
- **D-06:** If cleanup of the SDK-owned default ORT EF fails on the existing-collection path, `CreateCollection` returns an error.
- **D-07:** Log-only cleanup is for async/background cleanup paths, not this synchronous request path.
- **D-08:** Do not defer default EF creation until after the existence check in this phase.
- **D-09:** Do not introduce conditional or error-class-specific cleanup behavior. Use a plain fail-on-cleanup-error contract.

### the agent's Discretion
- Exact representation of the “SDK-created default dense EF” marker
- Exact seam shape for tests that need a close-counting default EF
- Exact test naming and assertion structure, as long as the two locked verification goals are covered

### Deferred Ideas (OUT OF SCOPE)
- Deferring default EF creation until after the existence check
- Folding Phase 23 and Phase 24 into one larger lifecycle redesign
- ORT-native/env-gated leak harnesses
- Typed cleanup error classes or best-effort cleanup behavior
</user_constraints>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| EFL-01 | Default ORT EF created by `PrepareAndValidateCollectionRequest` is closed when `CreateCollection` finds an existing collection | Add request-level provenance for SDK-created default ORT EF, close it on embedded `isNewCreation=false`, and pin close/error/state-preservation regressions |

## Project Constraints

| Constraint | Source | Impact on This Phase |
|-----------|--------|----------------------|
| New work targets V2 API | `CLAUDE.md` | All production changes stay in `pkg/api/v2/` |
| Library code should return errors, not panic | `CLAUDE.md`, `.planning/codebase/CONVENTIONS.md` | Cleanup failure must surface as a returned error, not stderr-only logging or panic |
| Validate inputs and ownership early | `.planning/codebase/CONVENTIONS.md` | Track default-EF provenance during request preparation so embedded create can make a deterministic cleanup decision |
| Colocated tests with matching build tags | `CLAUDE.md`, `.planning/codebase/TESTING.md` | Add/extend `basicv2` tests in `pkg/api/v2/client_local_embedded_test.go` |
| Existing collection state must remain authoritative | Phase 20 context `D-03` | Existing embedded collection state-backed EF/contentEF must still win on the existing-collection path |
| Close-once wrappers are the only valid wrapping entry points | `pkg/api/v2/ef_close_once.go`, Phase 23 context | Do not introduce ad hoc wrappers while fixing cleanup ownership |

## Standard Stack

No new dependencies are needed.

| Package | Purpose | Status |
|---------|---------|--------|
| `github.com/pkg/errors` | wrapped error return for cleanup failure | already used across V2 client code |
| Go stdlib `io` | detect/close closeable EFs when needed | already used in `pkg/api/v2/client.go` |
| `github.com/stretchr/testify/require` / `assert` | close-count and existing-state regressions | already used in embedded lifecycle tests |

## Architecture Patterns

### Pattern 1: Track provenance, not just presence

The current code only knows that `req.embeddingFunction` is non-nil after validation. That is not enough for Phase 23 because three cases collapse into the same field:
- caller supplied a dense EF
- SDK created default ORT EF
- SDK created default ORT EF and then replaced it with a dual-interface contentEF during promotion

The plan should explicitly preserve provenance on `CreateCollectionOp`, for example:
- `autoCreatedDefaultDenseEF bool`
- or an equivalent marker that remains true only when the current runtime dense EF is still the SDK-created default ORT EF

That marker must be set when `ort.NewDefaultEmbeddingFunction()` succeeds and cleared if the default EF is closed/replaced during contentEF promotion.

### Pattern 2: Cleanup belongs in embedded `CreateCollection` existing-path handling

The embedded `CreateCollection` function already computes `isNewCreation` before deciding whether to publish runtime state. That is the correct place to perform the Phase 23 cleanup because it has all needed context:
- whether the collection already existed
- the final `req.embeddingFunction`
- whether existing state should remain authoritative

Recommended shape:
- detect `!isNewCreation && req.autoCreatedDefaultDenseEF`
- if `req.embeddingFunction` implements `io.Closer`, call `Close()` before nulling the override
- on `Close()` error, return a wrapped error and do not return a collection
- then continue using existing state-backed EF/contentEF unchanged

This preserves D-02, D-03, D-06, and D-07 without restructuring the broader create flow.

### Pattern 3: Preserve Phase 20 existing-state precedence

`CreateCollection(..., WithIfNotExistsCreate(), WithEmbeddingFunctionCreate(...))` already preserves the original state-backed EF when the collection exists:

```go
require.Same(t, initialEF, unwrapCloseOnceEF(gotCollection.embeddingFunctionSnapshot()))
```

Phase 23 must keep that behavior. The leak fix is cleanup of an unused temporary SDK-owned default EF, not adoption of a new override and not any change to state precedence.

### Pattern 4: Use a lightweight seam for default EF creation only if tests need it

The hard part of verification is proving the SDK-created default EF was closed exactly once on the existing path without depending on real ORT runtime setup. The simplest path is a package-level creation seam in `pkg/api/v2/client.go`, for example:

```go
var newDefaultDenseEF = ort.NewDefaultEmbeddingFunction
```

Tests can temporarily replace it with a factory that returns a close-counting mock EF. This keeps the production behavior identical while allowing a fast `basicv2` regression.

Do not introduce a broader factory abstraction or plumb new interfaces through the public API; that would exceed the phase boundary.

### Pattern 5: Keep synchronous cleanup failures explicit

The repo already distinguishes between:
- synchronous setup/ownership failures that return errors
- asynchronous cleanup failures that may log to stderr

Phase 23 belongs to the first category. If closing the temporary SDK-owned default EF fails, the request path should fail immediately with a returned error. This matches the existing repository style and the locked context.

## Recommended File Layout

```text
pkg/api/v2/
├── client.go                       # provenance for SDK-created default ORT EF
├── client_local_embedded.go        # cleanup on existing-collection path
└── client_local_embedded_test.go   # close-count + existing-state regressions
```

## Common Pitfalls

### Pitfall 1: Using “embeddingFunction was nil at input” as the only ownership test

That signal becomes wrong when a dual-interface contentEF is promoted. `PrepareAndValidateCollectionRequest()` can create a default ORT EF, close it during promotion, and replace `op.embeddingFunction` with the contentEF-derived dense EF. A stale “input was nil” flag would make embedded `CreateCollection` think it still owns the current dense EF.

**Mitigation:** track whether the current final dense EF is still the SDK-created default ORT EF after validation completes.

### Pitfall 2: Closing caller-provided or state-backed EFs

The existing collection path intentionally ignores new override EFs and preserves prior state. Phase 23 must not broaden that into “close any unused override EF.” The cleanup target is only the SDK-created temporary default ORT EF.

**Mitigation:** gate cleanup on the explicit provenance marker, not on generic “overrideEF != nil”.

### Pitfall 3: Moving default EF creation later in the flow

That would be cleaner by construction, but it is a shared-flow refactor and was explicitly deferred in the phase context.

**Mitigation:** stay with the existing eager creation flow and add narrow cleanup/provenance logic around it.

### Pitfall 4: Hiding cleanup failure behind stderr logging

`close_logging.go` patterns are used for background cleanup, cache cleanup, and panic-safe teardown. This phase is a synchronous request path where the SDK still owns the temporary EF.

**Mitigation:** return a wrapped error from `CreateCollection` if the cleanup `Close()` fails.

### Pitfall 5: Relying only on existing-state regression tests

The current tests already prove that existing collections preserve original state, but they do not prove the temporary default ORT EF was cleaned up.

**Mitigation:** add both:
- one close-spy regression around default EF creation on the existing path
- one existing-state regression showing original EF/state still wins after the fix

## Threat Model Notes

This phase does not introduce a new external trust boundary. The relevant risk is resource/lifecycle correctness inside the SDK:
- **Threat:** repeated `CreateCollection(..., WithIfNotExistsCreate())` calls with no explicit dense EF leak ORT runtime resources when the collection already exists
- **Impact:** process-level resource growth, hidden lifecycle bug, and nondeterministic failures over time
- **Severity:** medium robustness risk, low direct security risk
- **Mitigation:** deterministic cleanup of SDK-owned temporary default EF, plus regression coverage for close-exactly-once and preserved existing-state behavior

No high-severity STRIDE-style threats were found. The planner should still include a concise `<threat_model>` block covering ownership confusion and resource leakage.

## Validation Architecture

### Test Infrastructure

| Property | Value |
|----------|-------|
| Framework | `go test` |
| Config file | `Makefile` / existing `basicv2` build tag |
| Quick run command | `go test -tags=basicv2 -run 'TestEmbeddedLocalClientCreateCollection_IfNotExistsExistingDoesNotOverrideState|TestEmbeddedCreateCollection_DefaultORT.*' ./pkg/api/v2/...` |
| Full suite command | `make test` |
| Lint command | `make lint` |
| Estimated runtime | quick run ~10s, full suite depends on local cache/runtime setup |

### Verification Strategy

- Add a focused close-spy regression that forces `PrepareAndValidateCollectionRequest()` to create a mock default EF and proves the embedded existing-collection path closes it exactly once.
- Add a focused failure regression proving that when the temporary SDK-owned default EF `Close()` returns an error, `CreateCollection(..., WithIfNotExistsCreate())` returns an error rather than success.
- Keep or extend the existing existing-state regression so the returned collection still uses the original state-backed EF and metadata/content state on the existing path.
- Run the focused `basicv2` tests after the code change, then `make test`, then `make lint`.

### Required Assertions

- The request-prep path records whether the final runtime dense EF is the SDK-created default ORT EF.
- On the existing-collection path, that SDK-owned default EF is closed exactly once before the override is discarded.
- A close failure on that cleanup path returns a non-nil error from `CreateCollection`.
- Existing collection state still wins: the returned embedded collection uses the original stored EF/contentEF rather than the temporary default.
- Non-existing collection creation behavior is unchanged.

---

_Research synthesized locally on 2026-04-10 after the dedicated researcher did not materialize an artifact in this session._
