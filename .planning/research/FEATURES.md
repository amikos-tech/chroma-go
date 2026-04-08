# Feature Landscape

**Domain:** Go SDK bug-fix and robustness milestone (v0.4.2)
**Researched:** 2026-04-08
**Confidence:** HIGH (all features sourced from confirmed GitHub issues and direct code inspection)

---

## Table Stakes

Features that must ship in v0.4.2 because the named bugs are actively broken or actively leaking resources.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| RrfRank arithmetic correctness | All 10 methods (`Multiply`, `Sub`, `Add`, `Div`, `Negate`, `Abs`, `Exp`, `Log`, `Max`, `Min`) are confirmed no-ops returning `self`. Users who chain arithmetic on RRF expect scoring changes but get none — silently wrong rankings. | LOW | Fix is wrapping in existing `MulRank`/`SumRank`/etc constructors, same as `ValRank` already does. No new types needed. |
| WithGroupBy(nil) validation | `WithGroupBy(nil)` silently proceeds ungrouped. Any code that ignores a nil-returning factory and passes nil gets incorrect results with no feedback. | LOW | Single nil-check and error return in `ApplyToSearchRequest`. Semantics are unambiguous: nil is always a caller mistake because `NewGroupBy` never returns nil. |
| Embedded GetOrCreateCollection closed-EF safety | `GetCollection` speculatively wraps user EFs in `closeOnce`, and on certain failure paths calls `deleteCollectionState` which closes those wrappers, delegating `Close()` to the user EF. The fallback to `CreateCollection` then passes the now-closed EF. Result: embedded ops fail with `errEFClosed`. | MEDIUM | Fix requires distinguishing auto-wired EFs (closeable on failure) from user-provided EFs (not closeable by the SDK). Requires careful ownership tracking, not just an error check. Confirmed in issue #493. |
| Default ORT EF leak in CreateCollection | When `CreateCollection` is called with `GetOrCreate=true` and the collection already exists, `PrepareAndValidateCollectionRequest` unconditionally creates a default ORT EF. The `isNewCreation=false` path discards the reference without calling `Close()`. Each call leaks one ORT session handle. | LOW | One-liner fix: close the default EF before setting `overrideEF = nil`. Confirmed in issue #494. |
| Error body truncation across providers | `ReadLimitedBody` caps at 200MB but error messages still include the full raw body. Perplexity has a local `sanitizeErrorBody` (512-char), OpenRouter was fixed in #477. The other 15+ providers expose arbitrarily large bodies. The fix is a shared utility in `pkg/commons/http` applied uniformly. | LOW-MEDIUM | 15+ providers need the same one-liner change. Complexity is breadth, not depth. The pattern is fully established (Perplexity/OpenRouter). |
| Twelve Labs async embedding (audio/video up to 4 hours) | Sync `POST /v1.3/embed-v2` only handles content under 10 minutes. The async endpoint `POST /v1.3/embed-v2/tasks` supports up to 4 hours. Without async support, any audio/video longer than 10 minutes returns an error from the API instead of an embedding. | HIGH | Requires: task creation, a polling loop, status handling (`processing` → `ready`/`failed`), timeout/cancellation via context, response mapping. API confirmed from Twelve Labs docs. |

---

## Differentiators

Features that would improve robustness and DX but are not strictly "broken" today.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Download stack consolidation | Today `client_local_library_download.go` (945 lines) and `pkg/tokenizers/libtokenizers/library_download.go` share near-identical logic: mirror fallback, checksum parse, cosign verify, lock/heartbeat, artifact extract. Each change to this flow requires two diffs. Consolidating into a shared internal package means provider or URL switches are config-level only. | HIGH | Identified in issue #412. Requires designing a shared config struct and extracting the download FSM without changing public behavior. Not a bug, but reduces churn for future provider changes. |
| Morph integration test fix | `POST https://api.morphllm.com/v1/embeddings` returns 404. CI fails on every PR. Not a code bug — upstream endpoint is broken. | LOW | Either skip the test in CI pending upstream recovery, or update the endpoint. Blocks CI cleanliness. |

---

## Anti-Features

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| RrfRank arithmetic returning a composite that wraps the full RRF expression in all operators | RRF has a specific mathematical meaning: `sum(weight_i / (k + rank_i))`. Wrapping a completed RRF result in post-hoc arithmetic changes semantics that are not documented and may not be what the server expects. | Wrap in existing expression types (`MulRank`, `SumRank`, etc.) so the JSON output is correct — this is what `ValRank` already does and is how the server-side expression language works. |
| WithGroupBy(nil) treated as "explicitly no grouping" sentinel | Introduces an intentionally overloaded nil, which is impossible to distinguish from an accidental nil at the call site. | Nil is always an error. Users who want no grouping simply omit `WithGroupBy`. |
| Context-based polling timeout for async Twelve Labs (instead of context propagation) | Polling with a hardcoded timeout ignores caller-controlled cancellation. | Use the passed-in `context.Context` as the sole cancellation signal. Provide a default polling interval (e.g., 5s) with an option to override. |
| Storing async task state in `TwelveLabsEmbeddingFunction` fields | Makes the EF stateful across concurrent calls. | Keep task creation + polling fully scoped to a single `embedContent` call stack. |
| Centralizing download logic by duplicating the new shared package back into both callers | Defeats the consolidation purpose. | One internal package (`pkg/internal/releasedownload` or similar), two callers that construct config structs and call it. |

---

## Feature Dependencies

```text
[RrfRank arithmetic fix]
    - no external deps; changes rank.go only

[WithGroupBy(nil) validation]
    - no external deps; changes search.go only

[Default ORT EF leak fix]
    - depends on: understanding of PrepareAndValidateCollectionRequest ownership model
    - related to: [Embedded GetOrCreateCollection EF safety]

[Embedded GetOrCreateCollection EF safety]
    - depends on: close-once wrapper (ef_close_once.go) behavior — already exists
    - must NOT break: collection fork behavior (ownsEF flag + close-once are co-designed)

[Error body truncation]
    - depends on: shared utility added to pkg/commons/http
    - applies to: 15+ provider files independently (no inter-provider deps)

[Twelve Labs async embedding]
    - depends on: existing TwelveLabsClient HTTP transport (reuse)
    - depends on: existing content.go request-building (reuse for video/audio part)
    - new: task types (EmbedV2Task, EmbedV2TaskResponse), polling loop, context cancellation
    - does NOT break: existing sync EmbedContent path (two parallel code paths, same interface)

[Download stack consolidation]
    - depends on: existing downloadutil.DownloadFileWithRetry (already shared)
    - refactors: client_local_library_download.go + libtokenizers/library_download.go
    - does NOT change external-facing behavior of either downloader
```

---

## Expected Behavior: Feature-by-Feature

### 1. RrfRank Arithmetic Fix

**Current behavior:** All 10 methods return `r` (self). `rrf.Multiply(FloatOperand(0.5))` silently returns the unchanged RRF rank.

**Expected behavior:** Each method wraps `r` in the appropriate expression node, identical to what `ValRank` does:
- `Multiply(op)` → `&MulRank{ranks: []Rank{r, operandToRank(op)}}`
- `Add(op)` → `&SumRank{ranks: []Rank{r, operandToRank(op)}}`
- `Sub(op)` → `&SubRank{left: r, right: operandToRank(op)}`
- `Div(op)` → `&DivRank{left: r, right: operandToRank(op)}`
- `Negate()` → `&MulRank{ranks: []Rank{Val(-1), r}}`
- `Abs()` → `&AbsRank{rank: r}`
- `Exp()` → `&ExpRank{rank: r}`
- `Log()` → `&LogRank{rank: r}`
- `Max(op)` → `&MaxRank{ranks: []Rank{r, operandToRank(op)}}`
- `Min(op)` → `&MinRank{ranks: []Rank{r, operandToRank(op)}}`

All of these types already exist. The fix is purely replacing `return r` with the correct constructor call. The serialized JSON output changes accordingly (e.g., `{"$mul": [{"$rrf": ...}, {"$val": 0.5}]}`). Confidence: HIGH — pattern verified against ValRank implementation in rank.go.

---

### 2. WithGroupBy(nil) Behavior

**Current behavior:** `ApplyToSearchRequest` returns `nil` when `o.groupBy == nil`, silently skipping grouping.

**Expected behavior:** Return an explicit error: `"groupBy must not be nil; omit WithGroupBy to search without grouping"`. This is unambiguous because `NewGroupBy` never returns nil — nil can only arrive through a bug in caller code (e.g., ignoring a factory error). There is no use case for intentional nil. Confidence: HIGH — confirmed from issue #482 and `NewGroupBy` signature.

---

### 3. Embedded GetOrCreateCollection Closed-EF Safety

**Current behavior:** If `GetCollection` partially succeeds then fails during `buildEmbeddedCollection` or a re-check, it calls `deleteCollectionState`, which closes the `closeOnce` wrapper around the user-provided EF. `GetOrCreateCollection` then passes the original (now-closed) EF to `CreateCollection`, causing `errEFClosed` on subsequent embed calls.

**Expected behavior:** User-provided EFs are never closed by the SDK on failure paths. Only auto-wired (SDK-owned) EFs should be closed on cleanup. The fix must distinguish ownership: EFs provided by the user via `WithEmbeddingFunctionCreate` are borrowed references — the SDK must not close them. Confidence: HIGH — confirmed from issue #493 with reproduction steps.

---

### 4. Default ORT EF Leak in CreateCollection

**Current behavior:** `PrepareAndValidateCollectionRequest` creates a default ORT EF whenever none is provided. When `GetOrCreate=true` and the collection already exists, the ORT EF is created but the reference is discarded (`overrideEF = nil`) without calling `Close()`.

**Expected behavior:** When the `isNewCreation=false` path is taken, close the default ORT EF before discarding it:
```go
} else {
    if closer, ok := req.embeddingFunction.(io.Closer); ok {
        _ = closer.Close()
    }
    overrideEF = nil
    overrideContentEF = nil
}
```
Confidence: HIGH — confirmed from issue #494 with exact code location.

---

### 5. Error Body Truncation

**Current behavior:** `ReadLimitedBody` caps at 200MB. Error message construction does `string(respData)` — the full body, up to 200MB, can appear in logs.

**Expected behavior:**
- A shared `SanitizeErrorBody(body []byte, maxChars int) string` function in `pkg/commons/http/` that trims whitespace, converts to runes, and appends `"...(truncated)"` when over the limit.
- Default limit: 512 chars (matching Perplexity's established constant).
- All 15+ providers replace `string(respData)` with `chttp.SanitizeErrorBody(respData, 512)` in their non-200 error paths.
- The function itself is simple: 5 lines, already proven in perplexity.go.

Confidence: HIGH — pattern fully established, confirmed from issue #478.

---

### 6. Twelve Labs Async Embedding

**API behavior (confirmed from Twelve Labs docs):**

- `POST /v1.3/embed-v2/tasks` — creates an async task for audio/video up to 4 hours.
  - Request: same model_name + video/audio object as sync endpoint, with optional `embedding_option`, `embedding_scope`, `segmentation`.
  - Response: `{ "id": "<task_id>" }` immediately. No embeddings yet.
- `GET /v1.3/embed-v2/tasks/{task_id}` — retrieves task status.
  - Status lifecycle: `"processing"` → `"ready"` (success) or `"failed"` (terminal error).
  - When `"ready"`: response includes `data` array of embedding objects with `embedding` (float vector), `embedding_option`, `embedding_scope`, `start_sec`, `end_sec`.
  - Polling interval: 5 seconds is the documented recommendation.
  - Max duration: up to 4 hours for video content.

**Expected SDK behavior:**

- A new method or path inside the existing `TwelveLabsEmbeddingFunction` (or a new `TwelveLabsAsyncEmbeddingFunction`) that:
  1. Creates the task via `POST /v1.3/embed-v2/tasks`.
  2. Polls `GET /v1.3/embed-v2/tasks/{task_id}` every 5 seconds (configurable).
  3. Exits when status is `"ready"` or `"failed"`, or when `ctx` is cancelled.
  4. On `"ready"`: extracts the first embedding vector from `data[0].embedding`.
  5. On `"failed"` or ctx cancellation: returns an error.
- Integration with `ContentEmbeddingFunction.EmbedContent` interface: the existing `EmbedContent` dispatch path calls the sync endpoint for text/image and short audio/video, and the async path for audio/video that exceeds the sync limit — OR the async path is triggered via a content option/hint.
- A per-request option `WithAsyncEmbedding()` is the cleanest signal: it opts a specific content item into the async path without changing the sync path.

**Complexity note:** The polling loop is the main new construct. The HTTP transport, auth, request marshaling, and response handling all reuse existing code. The interface question (whether to add an `AsyncEmbedContent` method or route through the existing `EmbedContent` with an option) is the key design decision for this feature.

Confidence: MEDIUM-HIGH — API behavior confirmed from official Twelve Labs documentation; SDK integration shape is design decision.

---

### 7. Download Stack Consolidation

**Current behavior:** `client_local_library_download.go` (945 lines) and `libtokenizers/library_download.go` each implement their own versions of:
- Mirror fallback across primary + GitHub fallback URL
- Metadata download with retry
- Checksum parse + verification
- Cosign certificate chain verification
- File-system lock + heartbeat
- Artifact extraction (tar.gz)

`pkg/internal/downloadutil` already exists and provides the transport layer (`DownloadFileWithRetry`). The duplication is in the orchestration layer above it.

**Expected behavior after refactor:**
- A new `pkg/internal/releasedownload` (or similar) package that encodes the shared download FSM: construct URL → download → verify checksum → verify cosign → extract.
- Both existing callers pass a `ReleaseConfig` struct (base URLs, asset naming, cosign identity template, size limits) and call one function.
- URL/provider changes are constants-only edits. No logic duplication.
- Test scaffolding for signed checksums and mirror fallback is shared.

Confidence: MEDIUM — scope is well-defined from issue #412; the correct abstraction boundary (what goes in shared config vs caller-specific logic) requires design work.

---

## MVP Recommendation

**Ship all Table Stakes features in v0.4.2.** They are confirmed bugs with contained fixes.

Prioritized order (lowest risk/effort first, to unblock CI and clear the easiest wins):

1. **WithGroupBy(nil) validation** — one-liner, zero risk
2. **RrfRank arithmetic fix** — mechanical substitution, no new types
3. **Default ORT EF leak** — one-liner close before discard
4. **Error body truncation** — shared utility + 15 provider replacements; breadth only
5. **Embedded GetOrCreateCollection EF safety** — requires ownership model change; higher risk
6. **Twelve Labs async embedding** — new feature, largest surface area
7. **Download stack consolidation** — large refactor, no user-visible behavior change; can slip to v0.4.3 if needed
8. **Morph test fix** — depends on upstream; skip in CI if endpoint remains 404

**Deferrable:** Download stack consolidation (issue #412) is the only item that could move to a follow-up milestone without leaving a user-visible bug in place.

---

## Sources

- GitHub issues #479, #481, #482, #478, #412, #493, #494, #465 (direct inspection)
- `pkg/api/v2/rank.go` lines 1127-1168 (RrfRank no-op methods)
- `pkg/api/v2/search.go` lines 635-643 (WithGroupBy nil handling)
- `pkg/api/v2/client_local_embedded.go` lines 341-461 (CreateCollection + GetOrCreateCollection)
- `pkg/api/v2/client.go` lines 270-328 (PrepareAndValidateCollectionRequest default ORT EF)
- `pkg/api/v2/ef_close_once.go` (close-once ownership semantics)
- `pkg/commons/http/utils.go` (ReadLimitedBody, 200MB cap)
- `pkg/embeddings/perplexity/perplexity.go` lines 145-151 (sanitizeErrorBody reference impl)
- `pkg/api/v2/client_local_library_download.go` (945 lines, full download FSM)
- `pkg/tokenizers/libtokenizers/library_download.go` (parallel download FSM)
- `pkg/internal/downloadutil/download.go` (existing shared transport layer)
- Twelve Labs API docs: https://docs.twelvelabs.io/docs/guides/create-embeddings/video/new (async task lifecycle: processing → ready/failed; 5s poll interval; 4h max)
- `pkg/embeddings/twelvelabs/twelvelabs.go` + `content.go` (existing sync implementation)
