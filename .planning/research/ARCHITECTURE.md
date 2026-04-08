# Architecture Research: v0.4.2 Bug Fixes and Robustness

**Domain:** Brownfield Go SDK — bug fixes, lifecycle hardening, async embedding, download stack consolidation
**Researched:** 2026-04-08
**Overall confidence:** HIGH (all findings from direct code inspection)

---

## Integration Map: Each Change vs Existing Architecture

### 1. RrfRank Arithmetic Silent No-ops (#481)

**Location:** `pkg/api/v2/rank.go`, lines 1129–1168

**What the code does today:**
Every arithmetic method on `RrfRank` (`Multiply`, `Sub`, `Add`, `Div`, `Negate`, `Abs`, `Exp`, `Log`, `Max`, `Min`) returns `r` unchanged:
```go
func (r *RrfRank) Multiply(operand Operand) Rank { return r }  // no-op
func (r *RrfRank) Add(operand Operand) Rank       { return r }  // no-op
```
All other concrete rank types (`ValRank`, `SumRank`, `SubRank`, `MulRank`, etc.) create and return new rank nodes. The `RrfRank` is the only type that short-circuits.

**Fix integration:**
Change each method to construct the appropriate composite rank wrapping `r`, identical to how every other Rank implementation works:
```go
func (r *RrfRank) Multiply(operand Operand) Rank {
    return &MulRank{ranks: []Rank{r, operandToRank(operand)}}
}
```
No new types, no new files. Pure method body changes inside one file.

**Component boundary:** Change is entirely contained within `pkg/api/v2/rank.go`. The `Rank` interface contract and all callers remain unchanged.

**Test surface:** New unit tests in `pkg/api/v2/rank_test.go` (already exists) asserting that each arithmetic method returns a different `Rank` instance whose `MarshalJSON` produces a composite expression node, not the RRF JSON.

---

### 2. WithGroupBy(nil) Silent No-op (#482)

**Location:** `pkg/api/v2/search.go`, lines 631–644

**What the code does today:**
```go
func (o *groupByOption) ApplyToSearchRequest(req *SearchRequest) error {
    if o.groupBy == nil {
        return nil  // silently ignores nil
    }
    ...
}
```
A caller writing `WithGroupBy(nil)` intends grouping but passes a nil pointer. The current code treats it as "no grouping", hiding the likely bug at the call site.

**Fix integration:**
Change the nil check to return an error:
```go
if o.groupBy == nil {
    return errors.New("WithGroupBy: groupBy cannot be nil")
}
```
No new types or files. One line change in `pkg/api/v2/search.go`. The error surfaces through `NewSearchRequest`'s option-application loop and propagates to the caller the same way any other `SearchRequestOption` error does.

**Component boundary:** Contained within `pkg/api/v2/search.go`. No interface or signature changes. Callers using a non-nil `GroupBy` are unaffected.

**Test surface:** `pkg/api/v2/search_test.go` — add a test asserting that `NewSearchRequest(WithGroupBy(nil))` returns an error.

---

### 3. Embedded GetOrCreateCollection Passes Closed EFs (#493)

**Location:** `pkg/api/v2/client_local_embedded.go`, `GetOrCreateCollection`, lines 420–461

**Root cause:**
The `GetOrCreateCollection` flow is:
1. Try `GetCollection` with the caller's EF.
2. On not-found, call `CreateCollection` (which wraps the EF with `wrapEFCloseOnce`).

When `CreateCollection` takes the fallback path and the collection already exists (i.e. `CreateIfNotExists` + `isNewCreation=false`, lines 408–411), it sets `overrideEF = nil` and builds the collection without an EF. That is correct for the create path.

But the actual `GetOrCreateCollection` bug is different: `createOptions` is assembled from the caller's original `options` slice (line 454) and re-passed to `CreateCollection`. `CreateCollection` always calls `PrepareAndValidateCollectionRequest`, which may re-validate or double-wrap EF state. More critically, if `GetOrCreateCollection` is called a second time on a collection that already exists, the `GetCollection` call succeeds and returns immediately at line 447 — correct. But if the caller passed a default ORT EF (issue #494 below), that EF was created and then not closed.

**The precise #493 bug:**
When `GetOrCreateCollection` falls through to `CreateCollection` (collection was not found by `GetCollection`), `createOptions` includes the caller-supplied EF from `options`. `CreateCollection` wraps this EF with `wrapEFCloseOnce` for new collections. However, if `CreateCollection` internally finds the collection already exists (race or concurrent creation) and takes `isNewCreation=false` (line 373), the EF passed in is neither wrapped nor stored in `collectionState`. The returned collection has no EF, but the wrapped-once close-once guard around the caller-provided EF never gets created, so the caller's EF may be closed prematurely when the collection is later closed.

**Fix integration:**
The fix belongs inside `GetOrCreateCollection`. Before calling `CreateCollection`, check whether the collection now exists (retry the get), and if so, return that result instead of continuing to create. This avoids the race path. Alternatively, ensure `GetOrCreateCollection` returns a usable collection with the caller's EF wired correctly regardless of which internal path `CreateCollection` takes.

No new types. The change is isolated to `GetOrCreateCollection` in `client_local_embedded.go`.

**Component boundary:** `pkg/api/v2/client_local_embedded.go` only. The `collectionState` map and `embeddedCollectionState` types are unchanged.

---

### 4. Default ORT EF Leaked in Embedded CreateCollection (#494)

**Location:** `pkg/api/v2/client_local_embedded.go`, `CreateCollection` and `buildEmbeddedCollection`, lines 341–418 and 836–919

**Root cause:**
When `CreateCollection` is called with no explicit EF, `PrepareAndValidateCollectionRequest` auto-creates a default ORT EF (default embedding function). This EF is passed to `buildEmbeddedCollection` as `overrideEF`. Inside `buildEmbeddedCollection`, `wrapEFCloseOnce` is called again around the already-once-wrapped EF (line 910). The `embeddedCollection.Close()` method only closes the EF if `ownsEF` is true (line 1552).

**The leak scenario:**
If `CreateCollection` finds an existing collection (line 373: `isNewCreation=false`) and enters the `else` branch setting `overrideEF = nil` (line 409), the default ORT EF that was auto-created during request preparation is abandoned without being closed. No Close is ever called on it.

**Fix integration:**
Two approaches are viable:
- In `CreateCollection`, when `isNewCreation=false` and an EF was auto-created (i.e. `req.embeddingFunction != nil` and no caller-supplied EF existed before prepare), explicitly close that EF before returning.
- Alternatively, defer the default EF creation until after the existence check, so no EF is created unless a new collection will actually be built.

The second approach is cleaner and does not require tracking "was this EF auto-created." Move the EF construction out of `PrepareAndValidateCollectionRequest` and into a later step only executed on the new-collection path.

**Component boundary:** `pkg/api/v2/client_local_embedded.go`, `pkg/api/v2/collection.go` (where `PrepareAndValidateCollectionRequest` lives). No changes to collection interface or calling code.

---

### 5. Error Truncation for Embedding Provider Error Bodies (#478)

**Current state:**

`pkg/commons/http/utils.go` provides two functions:
- `ReadLimitedBody(r io.Reader) ([]byte, error)` — caps at 200 MB, used by `ChromaErrorFromHTTPResponse` in `errors.go` and by the Twelve Labs provider in `doPost`.
- `ReadRespBody(resp io.Reader) string` — unbounded `io.ReadAll`, used by `client_http.go` for Chroma API responses (heartbeat, version, get-tenant, etc.).

Provider error responses (Twelve Labs, and others that have inline error JSON) use `ReadLimitedBody` or `chttp.ReadLimitedBody`, which is already bounded at 200 MB. That is too large for an error message.

**Where to truncate:**
The right layer is `chttp` because it is the single shared utility consumed by all providers. Two targeted changes:
1. Add a `ReadLimitedBody` variant with a smaller limit (`MaxErrorBodySize`, e.g. 64 KB) for error response bodies specifically, or add a `TruncateErrorBody(data []byte, maxLen int) string` helper that providers call after reading.
2. Providers that construct error strings from raw response bytes (e.g. Twelve Labs `doPost` line 186, any provider using `string(respData)` in error paths) should call the truncation helper.

The `ReadRespBody` function used by the Chroma HTTP client itself does not need truncation because it reads structured JSON responses (version string, heartbeat, tenant), not arbitrary error blobs.

**New component:** Add `TruncateBody(data []byte, maxLen int) string` to `pkg/commons/http/utils.go`. No new files. Providers call this when constructing error messages from raw response data.

**Component boundary:** `pkg/commons/http/utils.go` (one new helper), individual provider `doPost`/error-construction paths in `pkg/embeddings/*/`. No changes to interfaces or calling conventions.

---

### 6. Release Download Stack Consolidation (#412)

**Current duplication:**
There are three separate download implementations:

| Package | File | Approach |
|---------|------|----------|
| `pkg/api/v2` | `client_local_library_download.go` | Uses `downloadutil.DownloadFileWithRetry` + cosign verification, lock file, heartbeat, asset resolution, URL fallback |
| `pkg/tokenizers/libtokenizers` | `library_download.go` | Re-implements download HTTP client from scratch (own `http.Client` construction, own retry loop, own lock, own cosign), uses `downloadutil` only indirectly |
| `pkg/embeddings/default_ef` | `download_utils.go` | Full private `downloadFile` function with its own HTTP client construction, own temp-file logic, own checksum, distinct from `downloadutil` |

`pkg/internal/downloadutil/download.go` exists as the canonical utility, and `client_local_library_download.go` already delegates to it correctly. The tokenizer and default_ef stacks have not been migrated.

**Consolidation target:**
Migrate `pkg/tokenizers/libtokenizers/library_download.go` and `pkg/embeddings/default_ef/download_utils.go` to use `downloadutil.DownloadFileWithRetry` and `downloadutil.Config` instead of their own HTTP client construction and retry logic. Checksum verification, lock files, and cosign can remain in each caller since they are domain-specific.

**Component boundary:** `pkg/internal/downloadutil/download.go` (possibly expand `Config` with any missing options), `pkg/tokenizers/libtokenizers/library_download.go`, `pkg/embeddings/default_ef/download_utils.go`. The public API of each package (the exported functions callers use) remains unchanged.

**Key constraint:** `default_ef/download_utils.go` does not have a lock or cosign layer — it has a single `downloadFile` private function and a `onnxMu` mutex. It can simply replace its `downloadFile` body with a `downloadutil.DownloadFileWithRetry` call.

---

### 7. Twelve Labs Async Embedding (#479)

**Current sync architecture:**
`TwelveLabsEmbeddingFunction.doPost` in `twelvelabs.go` sends a synchronous POST to `https://api.twelvelabs.io/v1.3/embed-v2` and reads the response immediately. This works for text and short inputs. For long audio/video, the Twelve Labs API creates a task and requires polling.

**Async API shape (from Twelve Labs documentation):**
- POST to `embed-v2` returns a `task_id` when the input requires async processing.
- GET to `embed-v2/tasks/{task_id}` returns task status (`pending`, `processing`, `ready`, `failed`) and eventually a `video_embedding` array.

**Integration approach:**
Add async-aware logic inside `embedContent` in `content.go`. When the sync response contains a `task_id` instead of an embedding, switch to polling:

```
POST embed-v2
  → response has data[]        → return embeddings immediately (sync path, unchanged)
  → response has task_id       → enter polling loop
      GET embed-v2/tasks/{task_id} until status=ready or failed or context done
      → return embeddings from task response
```

**New components needed:**
- `EmbedV2TaskResponse` struct for the task creation response (`task_id`, `status`).
- `EmbedV2TaskStatusResponse` struct for the poll response (`status`, `video_embedding`).
- A `pollTask(ctx context.Context, taskID string) (embeddings.Embedding, error)` private method on `TwelveLabsEmbeddingFunction`.
- Polling configuration: interval and max-attempts fields on `TwelveLabsClient`, settable via new `Option` functions (`WithPollInterval`, `WithPollMaxAttempts`).

**Modified components:**
- `twelvelabs.go` — add poll config fields to `TwelveLabsClient`, add defaults in `applyDefaults`.
- `content.go` — modify `embedContent` to detect task_id in response and invoke `pollTask`.
- `option.go` — add `WithPollInterval` and `WithPollMaxAttempts`.
- `twelvelabs_test.go` / `twelvelabs_content_test.go` — add tests for async path (mock server that returns task_id on first call, status on subsequent calls).

**Component boundary:** All changes contained within `pkg/embeddings/twelvelabs/`. The `EmbedContent`, `EmbedContents`, `EmbedDocuments`, and `EmbedQuery` signatures are unchanged. Callers see no difference — polling is transparent.

**Context propagation:** The polling loop must respect `ctx.Done()` and return a meaningful error on cancellation.

---

### 8. Morph EF Integration Test (#465)

**Location:** `pkg/embeddings/morph/morph_test.go`

**Issue:** The upstream endpoint `https://api.morphllm.com/v1/` returns 404. This is a live URL change, not a code bug.

**Fix options:**
- Update the base URL constant and test expectations to match the current Morph API URL.
- Or stub the HTTP call in the test and remove the dependency on the live endpoint.

The `MorphClient.BaseURL` default is set via `creasty/defaults` struct tag: `default:"https://api.morphllm.com/v1/"`. The test file uses `go:build ef` so it only runs in the `ef` test suite.

**Component boundary:** `pkg/embeddings/morph/morph.go` (URL constant/default), `pkg/embeddings/morph/morph_test.go`. No interface changes.

---

## System-Level Data Flow Diagram

```text
Caller
  │
  ├─ Search API path
  │    └── NewSearchRequest(WithGroupBy(...), WithRrfRank(...), ...)
  │              │
  │         SearchRequest.Rank = RrfRank | KnnRank | arithmetic composition
  │              │
  │         RrfRank.MarshalJSON() → builds composite expression via .Add/.Mul/etc.
  │                                (currently no-ops — fixed by #481)
  │
  ├─ Embedded Client path
  │    └── GetOrCreateCollection(ctx, name, opts...)
  │              │
  │         GetCollection (returns on hit)
  │              │ miss
  │         CreateCollection(ctx, name, opts + WithIfNotExistsCreate)
  │              │
  │         isNewCreation check (lines 366–374)
  │              │ false path (race/existing) → overrideEF=nil → EF abandoned (#494)
  │              │ true path → wrapEFCloseOnce → collectionState upsert → buildEmbeddedCollection
  │                                                                              │
  │                                                                      collection.ownsEF=true
  │
  └─ Twelve Labs EF async path (new)
       └── EmbedContent(ctx, content)
                 │
            embedContent → doPost (sync POST)
                 │
            response has task_id?
            ├─ no  → embeddingFromResponse (current path, unchanged)
            └─ yes → pollTask(ctx, taskID)
                          │
                     GET /embed-v2/tasks/{id} (with ctx-aware interval)
                          │ status=ready → return embedding
                          │ status=failed → return error
                          └─ ctx.Done → return ctx error
```

---

## Component Boundary Summary

| Change | Files Modified | New Files | Interface Changes |
|--------|---------------|-----------|-------------------|
| #481 RrfRank arithmetic | `pkg/api/v2/rank.go` | None | None |
| #482 WithGroupBy nil | `pkg/api/v2/search.go` | None | None — error return is already in the interface |
| #493 GetOrCreateCollection EF | `pkg/api/v2/client_local_embedded.go` | None | None |
| #494 ORT EF leak | `pkg/api/v2/client_local_embedded.go`, `pkg/api/v2/collection.go` | None | None |
| #478 Error truncation | `pkg/commons/http/utils.go`, individual provider `*.go` files | None | None — new unexported helper |
| #412 Download stack | `pkg/tokenizers/libtokenizers/library_download.go`, `pkg/embeddings/default_ef/download_utils.go` | None | None — internal implementation only |
| #479 Twelve Labs async | `pkg/embeddings/twelvelabs/twelvelabs.go`, `content.go`, `option.go` | None (all additions are within existing files) | None |
| #465 Morph test | `pkg/embeddings/morph/morph.go`, `morph_test.go` | None | None |

---

## Build Order and Dependencies

Changes are independent of each other. No feature requires another to land first. Suggested ordering by risk and blast radius:

1. **#481 RrfRank arithmetic** — pure method-body fix, no dependencies, very low risk. Do first to confirm test pattern.
2. **#482 WithGroupBy nil** — one-line validation change, independent.
3. **#478 Error truncation** — add helper to `chttp`, then update individual providers. Low risk, high value. Do before async work so new Twelve Labs polling errors are also truncated.
4. **#412 Download stack** — refactoring with no behavior change. Independent. Can proceed in parallel with the above.
5. **#493 + #494 Embedded EF lifecycle** — closely related, likely best addressed together since both are inside `CreateCollection` / `GetOrCreateCollection`. Medium risk; requires carefully tracking `isNewCreation` logic.
6. **#479 Twelve Labs async** — new feature, most code added. Isolated to one provider package. Can proceed after #478 so the error truncation utility is available.
7. **#465 Morph test** — blocked only on knowing the current Morph API URL. Trivial once URL is confirmed.

---

## Anti-Patterns to Avoid

**Truncating at `ReadRespBody`:** The `ReadRespBody` function is used for Chroma API responses (structured JSON), not error blobs. Truncating there would corrupt valid responses. Truncation must only apply at the error-construction site, not at the read site.

**Adding a new download abstraction layer:** `downloadutil` already exists and is correct. Do not introduce a third interface or wrapper. Callers should use `downloadutil.Config` + `downloadutil.DownloadFileWithRetry` directly and own their domain-specific layers (checksums, locks, cosign) themselves.

**Blocking async polling without context propagation:** The polling loop in #479 must check `ctx.Done()` on every iteration. A blocking `time.Sleep` without select on context will leak goroutines when callers cancel.

**Wrapping EFs multiple times:** `buildEmbeddedCollection` calls `wrapEFCloseOnce` on whatever EF it receives, including EFs already wrapped by `CreateCollection`. Double-wrapping is harmless for Close semantics (close-once wrappers are idempotent) but creates unnecessary wrapper nesting. The #493/#494 fix should ensure each EF is wrapped exactly once.

---

## Sources

- `pkg/api/v2/rank.go` — RrfRank implementation, lines 1127–1168 (direct inspection)
- `pkg/api/v2/search.go` — WithGroupBy nil check, line 636 (direct inspection)
- `pkg/api/v2/client_local_embedded.go` — GetOrCreateCollection/CreateCollection/buildEmbeddedCollection, lines 341–461 and 836–919 (direct inspection)
- `pkg/commons/http/utils.go` — ReadLimitedBody and ReadRespBody, current limits (direct inspection)
- `pkg/embeddings/twelvelabs/twelvelabs.go` and `content.go` — sync EmbedContent path (direct inspection)
- `pkg/internal/downloadutil/download.go` — canonical download utility (direct inspection)
- `pkg/embeddings/default_ef/download_utils.go` — private duplicate download implementation (direct inspection)
- `pkg/tokenizers/libtokenizers/library_download.go` — second duplicate download implementation (direct inspection)
- `pkg/embeddings/morph/morph.go` — base URL constant (direct inspection)
