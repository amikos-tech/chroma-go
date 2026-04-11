# Phase 24: GetOrCreateCollection EF Safety - Research

**Researched:** 2026-04-12
**Domain:** Embedded V2 `GetOrCreateCollection` lifecycle, provisional EF ownership, and concurrent miss/create convergence
**Confidence:** HIGH (direct code and test inspection)

## Summary

Phase 24 is not just a narrow `GetOrCreateCollection` wrapper tweak. The real failure shape spans:
- `pkg/api/v2/client_local_embedded.go`
- `pkg/api/v2/close_logging.go`
- embedded lifecycle tests in `pkg/api/v2/client_local_embedded_test.go`

Today `GetOrCreateCollection(...)` first probes with `GetCollection(...)`, forwarding caller-provided dense EF and/or `contentEF` into the get path:

```go
getOptions := []GetCollectionOption{WithDatabaseGet(req.Database)}
if req.embeddingFunction != nil {
	getOptions = append(getOptions, WithEmbeddingFunctionGet(req.embeddingFunction))
}
if req.contentEmbeddingFunction != nil {
	getOptions = append(getOptions, WithContentEmbeddingFunctionGet(req.contentEmbeddingFunction))
}
collection, getErr := client.GetCollection(ctx, req.Name, getOptions...)
```

Inside embedded `GetCollection(...)`, those EFs are wrapped and stored in `collectionState` before the returned collection is revalidated against the runtime:

```go
if contentEF != nil {
	s.contentEmbeddingFunction = wrapContentEFCloseOnce(contentEF)
}
if ef != nil {
	s.embeddingFunction = wrapEFCloseOnce(ef)
}
```

If revalidation later fails, `GetCollection(...)` calls `deleteCollectionState(model.ID)`, and that helper unconditionally closes whatever is stored in state:

```go
if state != nil {
	if err := closeEmbeddingFunctions(state.embeddingFunction, state.contentEmbeddingFunction); err != nil {
		...
	}
}
```

That means a provisional `GetCollection(...)` failure can close caller-provided EFs even though the SDK never successfully handed them off to a durable collection. `GetOrCreateCollection(...)` then falls back to `CreateCollection(...)` with the same logical EF input, and the caller can end up with `errEFClosed` or a returned collection whose temporary fallback EF handling is incorrect under concurrent races.

Phase 23 already tightened the embedded `CreateCollection(..., WithIfNotExistsCreate())` reuse path for SDK-owned default dense EFs. Phase 24 must build on that precedent without regressing it:
- caller-provided EFs stay borrowed until a verified handoff succeeds
- only SDK-owned auto-wired/default EFs are eligible for provisional cleanup
- concurrent losers should converge to winner state when that state is authoritative
- but the already-accepted empty/no-config ambiguity branch must still keep a usable temporary fallback EF instead of returning a nil/unusable handle

**Primary recommendation:** implement ownership-aware embedded state cleanup and use conditional convergence in the `GetOrCreateCollection(...)` fallback path. Do not try to solve this by only changing the top-level `GetOrCreateCollection(...)` function while leaving provisional `GetCollection(...)` cleanup unconditional.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Caller-provided EFs are borrowed during the provisional `GetCollection` path and must never be closed by SDK cleanup when `GetCollection` fails.
- **D-02:** Only SDK-owned EFs created or auto-wired by the SDK remain eligible for cleanup on those failure paths.
- **D-03:** Ownership transfers only after a verified collection/state handoff; Phase 24 does not redefine the global EF ownership contract.
- **D-04:** Use conditional convergence: when concurrent winner state is authoritatively observable, the loser converges to that winner snapshot.
- **D-05:** Exception: if forced convergence would yield a nil or otherwise unusable EF handle in the empty/no-config branch, keep the temporary fallback EF.
- **D-06:** The conditional convergence rule stays narrow; do not reopen the Phase 23 nil/unusable-EF failure shape.
- **D-07:** Cover dense EF, `contentEF`, and dual-interface content EF paths that share the same cleanup mechanism.
- **D-08:** Shared-resource close behavior must still treat dual-interface content EFs as the owning close path when dense and content wrappers resolve to the same underlying resource.
- **D-09:** Verification remains narrow and colocated: one deterministic fallback regression plus one orchestrated concurrent `GetOrCreateCollection` regression intended to pass under `go test -race`.
- **D-10:** Do not add a soak harness or repeated stress loops unless the deterministic concurrent path proves insufficient.

### the agent's Discretion
- Exact state/provenance representation for borrowed vs SDK-owned provisional EFs
- Exact branch shape for deciding when winner state is authoritative enough to converge
- Exact synchronization seam and helper runtime used to force the concurrent fallback race

### Deferred Ideas (OUT OF SCOPE)
- Reference counting or lease-based EF lifecycle management
- Broader repeated stress/soak race coverage
- Cross-backend EF ownership redesign beyond this embedded fallback bug class
</user_constraints>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| EFL-02 | `GetOrCreateCollection` does not pass closed EFs to `CreateCollection` fallback when `GetCollection` fails mid-build | Make provisional `GetCollection` cleanup ownership-aware so borrowed caller EFs survive fallback; then ensure fallback returns a usable collection under race/reuse branches |
| EFL-03 | Tests cover EF lifecycle under `-race` for concurrent `GetOrCreateCollection` calls | Add one deterministic fallback regression and one concurrent miss/create race regression that asserts returned collections stay usable and close counts remain sane |

## Project Constraints

| Constraint | Source | Impact on This Phase |
|-----------|--------|----------------------|
| Prefer V2 API changes in `pkg/api/v2/` | `.planning/codebase/CONVENTIONS.md` | All production changes stay inside embedded V2 client/state code |
| Return errors instead of panicking in runtime paths | `.planning/codebase/CONVENTIONS.md` | Ownership mistakes must surface as returned errors or test failures, not silent panics |
| Use colocated tests with matching build tags | `.planning/codebase/TESTING.md` | New coverage belongs in `pkg/api/v2/client_local_embedded_test.go` under `basicv2` |
| Reuse existing close-once wrapper helpers | `pkg/api/v2/ef_close_once.go` | Do not invent parallel wrapper types; use `wrapEFCloseOnce` / `wrapContentEFCloseOnce` only |
| Existing embedded collections preserve authoritative state | Phase 20 + Phase 23 carry-forward | The fix must not overwrite existing winner state with loser-local temporary EFs except for the narrow empty/no-config ambiguity exception |
| Shared dense/content resource detection already exists | `pkg/api/v2/close_logging.go` | Ownership-aware cleanup must keep dual-interface content EF close semantics aligned with existing helper behavior |

## Standard Stack

No new dependencies are needed.

| Package | Purpose | Status |
|---------|---------|--------|
| `github.com/pkg/errors` | wrapped errors for revalidation, fallback, and cleanup behavior | already used |
| Go stdlib `sync`, `sync/atomic`, `io` | existing concurrency, close, and state mechanics | already used |
| `github.com/stretchr/testify/require` | colocated deterministic and concurrent regressions | already used |

## Architecture Patterns

### Pattern 1: Ownership-aware cleanup must live at the embedded state boundary

The main bug is that provisional `GetCollection(...)` cleanup currently treats every EF in `collectionState` as SDK-owned:
- caller dense EF forwarded through `WithEmbeddingFunctionGet(...)`
- caller `contentEF` forwarded through `WithContentEmbeddingFunctionGet(...)`
- auto-wired dense/content EFs built from config

Those are materially different ownership classes. A Phase 24 fix should add explicit cleanup provenance to embedded state, for example:
- borrowed dense EF vs owned dense EF
- borrowed content EF vs owned content EF
- or an equivalent policy object/flags that let `deleteCollectionState(...)` decide what may be closed

The important invariant is:
- provisional state created from caller-provided EFs is **borrowed**
- provisional state created from SDK auto-wiring/default creation is **owned**
- state that survives a verified handoff may later be owned by the collection/state lifecycle as it already is today

This means the likely fix seam is `embeddedCollectionState`, `upsertCollectionState(...)`, and `deleteCollectionState(...)`, not only `GetOrCreateCollection(...)`.

### Pattern 2: `GetOrCreateCollection(...)` still needs conditional convergence logic

After the initial get miss/failure, `GetOrCreateCollection(...)` falls back to:

```go
createOptions := append([]CreateCollectionOption{}, options...)
createOptions = append(createOptions, WithIfNotExistsCreate())
collection, createErr := client.CreateCollection(ctx, req.Name, createOptions...)
```

Phase 23 already taught `CreateCollection(...)` to reload authoritative existing state in some reuse cases. Phase 24 needs the same spirit at the higher-level miss/create race boundary:
- if concurrent winner state is clearly observable, return/reload the winner snapshot
- if the winner snapshot would otherwise yield a nil/unusable EF handle in the empty/no-config branch, keep the temporary fallback EF so the returned collection still works

Do not blindly force convergence in every race outcome. The Phase 23 review already proved that “always reload the winner” can produce a collection with no usable EF when there is no persisted config to rebuild from.

### Pattern 3: Dense/content/dual-interface parity is mandatory

The user explicitly chose breadth across the shared ownership mechanism, not a dense-only patch.

That means the plan must cover:
- caller-provided dense EF
- caller-provided `contentEF`
- dual-interface content EF that also serves as the dense EF
- shared-resource close behavior where content EF is the owning close path

`closeEmbeddingFunctions(...)` already knows how to avoid double-closing shared dense/content resources. Phase 24 should preserve that behavior while adding ownership gating. A good shape is:
- decide *whether* dense/content EFs are closable based on provenance
- then reuse the existing shared-resource detection when actually closing owned EFs

### Pattern 4: Preserve Phase 23’s SDK-owned default EF cleanup contract

Phase 23 introduced explicit tracking of `sdkOwnedDefaultDenseEF` on `CreateCollectionOp` and cleanup/reload behavior in embedded `CreateCollection(...)`.

Phase 24 must not regress those guarantees:
- SDK-owned temporary defaults are still closable on abandoned/reuse paths
- caller-provided EFs are still not closable just because they were observed on a failed provisional path
- the empty/no-config branch must still be able to keep a usable temporary EF when reloading would produce a nil handle

If a Phase 24 change centralizes cleanup decisions, it must continue to distinguish:
- SDK-owned temporary default EF from Phase 23
- borrowed caller EF from Phase 24
- state-backed/shared collection EF that is already under collection lifecycle control

### Pattern 5: Focused test seams already exist; use them instead of adding new infrastructure

Reusable test assets already in the repo:
- `newCountingMemoryEmbeddedRuntime()` for simple call counting
- `newBlockingGetMemoryEmbeddedRuntime()` for deterministic get/race orchestration
- `newMissingGetCollectionOnceRuntime()` and related helpers for transient get anomalies
- `mockCloseableEF`, `mockCloseableContentEF`, `mockDualEF`, and `mockFailingCloseEF` for close-count and shared-resource checks

The recommended test shape is:
- one deterministic regression that proves a provisional get failure does **not** close the caller EF before fallback/reuse completes
- one concurrent two-goroutine `GetOrCreateCollection(...)` regression under `-race` proving both returned handles remain usable and no double-close panic/race occurs

No new soak harness is needed.

## Recommended File Layout

```text
pkg/api/v2/
├── client_local_embedded.go        # ownership-aware provisional state + conditional convergence
├── close_logging.go                # ownership-gated close helper if cleanup stays centralized
└── client_local_embedded_test.go   # deterministic fallback + concurrent race regressions
```

`pkg/api/v2/client.go` should only change if the chosen implementation needs to propagate existing Phase 23 provenance into the shared cleanup policy. That is optional, not the primary seam.

## Common Pitfalls

### Pitfall 1: Fixing only `GetOrCreateCollection(...)`

If `deleteCollectionState(...)` keeps closing all provisional state EFs unconditionally, the bug class remains for any failed `GetCollection(...)` path, including `contentEF` and dual-interface cases.

**Mitigation:** make cleanup ownership-aware at the state layer.

### Pitfall 2: Transferring ownership too early

A caller EF seen on a provisional get path is not SDK-owned just because it was wrapped and inserted into `collectionState`.

**Mitigation:** borrowed caller EFs stay non-closable until a verified handoff succeeds.

### Pitfall 3: Breaking shared dense/content close behavior

If ownership gating is bolted on by closing dense and content EFs independently without preserving `isDenseEFSharedWithContent(...)`, dual-interface content EFs will regress into double-close territory.

**Mitigation:** keep the existing shared-resource detection in the owned-close path.

### Pitfall 4: Unconditionally converging onto the winner snapshot

That can reintroduce the Phase 23 nil/unusable-EF return shape in empty/no-config races.

**Mitigation:** use the user-approved conditional convergence rule and keep the temporary fallback EF in the narrow ambiguity branch.

### Pitfall 5: Solving the race by holding locks across embedded runtime calls

Serializing the entire get-then-create sequence under `collectionStateMu` would be wider than necessary and risks new contention/deadlock shapes around runtime calls that can re-enter state helpers.

**Mitigation:** keep locking narrow; use explicit state provenance and post-call convergence instead of coarse-grained locking.

## Threat Model Notes

This phase does not add a new external trust boundary. The relevant threats are internal lifecycle and concurrency correctness:

- **T-24-01: Ownership confusion** — provisional cleanup closes a caller-owned EF, so fallback or subsequent embed operations fail with `errEFClosed`
- **T-24-02: Concurrent miss/create race** — a losing caller returns a stale, nil, or otherwise unusable collection handle after another goroutine creates the collection
- **T-24-03: Shared-resource double close** — dense/content shared underlying resources are closed twice if ownership gating bypasses the existing shared-resource logic

Severity is medium for robustness and correctness, low for direct security impact. The planner should still include a concise `<threat_model>` block tied to lifecycle ownership and race convergence.

## Validation Architecture

### Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` |
| **Config file** | `Makefile` / existing `basicv2` build tag |
| **Quick run command** | `go test -tags=basicv2 -run 'TestEmbedded(LocalClient)?GetOrCreateCollection.*|TestEmbeddedGetCollection_Race.*' ./pkg/api/v2/...` |
| **Full suite command** | `make test` |
| **Lint command** | `make lint` |
| **Estimated runtime** | ~15s focused / longer for full V2 suite |

### Verification Strategy

- Add one deterministic fallback regression proving a caller-provided dense EF remains open and usable when provisional `GetCollection(...)` fails and `GetOrCreateCollection(...)` falls back to create/reuse logic.
- Extend that regression shape across `contentEF` / dual-interface ownership if the implementation touches shared cleanup code broadly.
- Add one orchestrated concurrent `GetOrCreateCollection(...)` race using a blocking embedded runtime and run it under `go test -race`.
- Assert that both returned collections remain usable and that close counts do not exceed one on shared resources.
- Run the focused `basicv2` suite after each implementation step, then `make test`, then `make lint`.

### Required Assertions

- Borrowed caller dense EF is not closed by provisional `GetCollection(...)` cleanup.
- Borrowed caller `contentEF` is not closed by provisional `GetCollection(...)` cleanup.
- Shared dual-interface content EF still closes through the content owner path only once.
- `GetOrCreateCollection(...)` fallback never returns a handle whose EF immediately reports `errEFClosed`.
- Concurrent `GetOrCreateCollection(...)` calls pass under `-race` and do not panic or double-close.
- Existing authoritative collection state still wins except for the locked empty/no-config ambiguity exception that preserves a usable temporary EF.

