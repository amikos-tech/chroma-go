# Pitfalls Research — v0.4.2 Bug Fixes and Robustness

**Domain:** Brownfield Go SDK — targeted bug fixes and hardening on an existing public API
**Researched:** 2026-04-08
**Confidence:** HIGH (all findings grounded in the live codebase)

---

## Critical Pitfalls

### Pitfall 1: RrfRank Arithmetic Methods Are Currently No-Ops — Return Self Instead of a New Rank

**What exists now:** Every arithmetic method on `RrfRank` (Add, Sub, Multiply, Div, Negate, Abs, Exp, Log, Max, Min) returns `r` — the receiver itself — with a `// no-op` comment. This is the bug. RRF is designed to be the terminal node in an expression tree; composition was intentionally blocked but was never intended to silently succeed.

**What the fix requires:** Returning an explicit error or panicking is not an option because the `Rank` interface returns `Rank` not `(Rank, error)`. The idiomatic Go option is to return a sentinel `ErrorRank` that carries the error message and fails loudly when `MarshalJSON()` is called. This is the same pattern already used by `UnknownRank`.

**Pitfall when fixing:** The temptation is to simply make the methods build a real expression tree (e.g., `return &SumRank{ranks: []Rank{r, operandToRank(operand)}}`). This would silently change the serialized form for any caller who already works around the no-op by wrapping RRF manually — unlikely given the bug, but still a semantic change.

**Prevention:** Introduce an `ErrorRank` (wrapping `ValRank` like `UnknownRank` does), have all ten arithmetic methods on `RrfRank` return it with a message like "RrfRank cannot be composed with arithmetic expressions; use it as a terminal rank in a SearchRequest". The existing `UnknownRank.MarshalJSON` pattern in `rank.go` is the right template.

**Detection / test signal:** `TestRrfRankArithmetic` must assert that calling any arithmetic method on a `*RrfRank` and then calling `MarshalJSON()` on the result returns a non-nil error. Currently these paths have no test coverage.

**Phase to address:** Phase 1 (fix arithmetic no-ops — #481)

---

### Pitfall 2: WithGroupBy(nil) Silent No-Op Must Become an Error — But the Fix Is Subtle

**What exists now:** `WithGroupBy` accepts `nil` and its `ApplyToSearchRequest` returns `nil` without setting anything, meaning callers who pass `nil` by mistake get silently empty grouping with no signal.

**Correct fix:** `ApplyToSearchRequest` should return an error when `o.groupBy == nil`. The `groupByOption` struct itself also currently holds the nil — so checking at option-construction time in `WithGroupBy` would be the earliest possible feedback.

**Pitfall when fixing:** There is a design tension here. `WithGroupBy(nil)` is logically "clear grouping" (a nil pointer conventionally means absence in Go). A strict callee would argue `nil` input is a programming error; a lenient callee would let it pass. The existing code made the lenient choice — changing it is a behavior change for any caller who passes `nil` intentionally to mean "no grouping". In a public SDK this is the kind of change that belongs in a CHANGELOG note.

**Prevention:** Validate at `WithGroupBy` call time: return an error from construction if `groupBy` is nil. The `WithRrfRank` constructor pattern already returns an `*rrfRankOption` with an embedded error to carry errors lazily — follow the same pattern. Add a test that `NewSearchRequest(..., WithGroupBy(nil))` returns a non-nil error from the request builder.

**Phase to address:** Phase 2 (fix nil GroupBy — #482)

---

### Pitfall 3: Embedded GetOrCreateCollection Passes a Closed EF into the CreateCollection Fallback

**What exists now:** `GetOrCreateCollection` calls `GetCollection` first (passing `req.embeddingFunction` in the get options). If `GetCollection` fails (collection does not exist), it calls `CreateCollection` passing the same original `options` slice. The original `options` slice may contain an EF that was already closed during the `GetCollection` path — specifically via the `closeOnce` wrapper that was applied before the get attempt.

Looking at the actual code path: `GetCollection` at line 440-445 wraps the EF into `getOptions` only for the get side. `CreateCollection` at line 454-456 reuses the raw `options` (pre-wrapping). The actual bug is that `GetOrCreateCollection` calls `GetCollection` and then `CreateCollection` as separate calls, meaning the `WithIfNotExistsCreate()` flag goes to `CreateCollection`. If `GetCollection` returned a wrapped-and-closed EF via `wrapEFCloseOnce` into `collectionState` before failing for another reason, the next `CreateCollection` uses the unwrapped `req.embeddingFunction` (not closed) correctly — but the `collectionState` entry may have been mutated.

**The actual race (issue #493):** When `GetOrCreateCollection` finds no collection (get fails) and falls through to `CreateCollection`, `CreateCollection` checks `isNewCreation` using a secondary `GetCollection` lookup at lines 367-374. If a concurrent `GetCollection` ran between the two and populated `collectionState` with a close-once-wrapped EF that is already closed (because the caller closed something between calls), the `CreateCollection` fallback branches into `isNewCreation = false`, skips EF assignment, and the returned collection has a stale closed EF in state.

**Pitfall when fixing:** The state map is keyed by collection ID (UUID), not name. If the race window is narrow, tests may pass without triggering it. The fix needs to either: (a) copy the EF from the incoming `req` into state unconditionally on the create path when the collection is genuinely new, or (b) hold the state lock across the full get-then-create sequence (expensive). Option (a) is the simpler approach.

**Prevention:** Add a test that calls `GetOrCreateCollection` on a non-existent collection from two goroutines simultaneously and asserts both returned collections can embed without an `errEFClosed` error.

**Phase to address:** Phase 3 (fix closed EF race — #493)

---

### Pitfall 4: ORT EF Leaked in CreateCollection When Collection Already Exists

**What exists now:** `CreateCollection` at lines 366-411 checks `isNewCreation` to decide whether to populate `collectionState`. When `GetOrCreate:true` and the collection already exists (`isNewCreation = false`), `overrideEF` is set to nil. This means the EF supplied by the caller (e.g., a freshly created ORT EF) is never stored and never closed. The caller has no way to know this; the `Collection` returned does not own the EF they passed in.

**Pitfall when fixing:** The fix is to close the caller-supplied EF when it is not used (collection already existed, EF not adopted). But closing it synchronously may surprise a caller who created a shared EF and passed it to multiple collection constructors. The correct fix is to document that callers must not pass EFs whose lifecycle they also manage; the collection either adopts or does not. If the collection does not adopt, the caller's EF is returned as-is and the caller is responsible. Alternatively, the embedded client can expose a `ownsEF` flag decision — if isNewCreation is false, log a warning and let the caller manage it.

**Simpler correct fix:** When `isNewCreation = false`, if the caller's EF was not already in `collectionState[id]`, close it and return an informative error or warning. This requires inspecting `collectionState` before the embedded create call, which is already done at lines 367-374.

**Prevention:** Add a test: create an ORT EF, call `CreateCollection` twice with `GetOrCreate:true`, and use `goleak` or a custom `io.Closer` spy to assert that the EF passed to the second call is either adopted or explicitly closed.

**Phase to address:** Phase 4 (fix ORT EF leak — #494)

---

## Moderate Pitfalls

### Pitfall 5: Twelve Labs Async Polling Introduces Context Cancellation Gaps

**What is being added:** The current Twelve Labs provider uses a synchronous `POST /embed-v2` that blocks until the embedding is ready. For long audio and video content, the API requires an async task flow: `POST /tasks` → poll `GET /tasks/{id}` until terminal state → fetch result.

**Pitfalls when adding the async path:**

1. **Not threading the context into the poll loop.** Every `http.NewRequestWithContext` call in the polling loop must use the caller's context. A poll loop that uses `context.Background()` internally is unreachable by the caller's cancellation or deadline.

2. **Blocking goroutine without a sleep/backoff.** A tight `GET /tasks/{id}` loop with no sleep burns API quota and can hit rate limits. Use `time.Sleep` or an exponential backoff between polls; respect the context between iterations.

3. **Not distinguishing terminal states correctly.** If the Twelve Labs task API returns `"failed"` or `"error"` as terminal states alongside `"ready"`, a loop that only checks for `"ready"` will spin until timeout instead of returning an error immediately.

4. **Mixing async and sync paths silently.** If the async path is chosen based on media duration or content type heuristics, the caller does not know which path ran. Errors from the async path (task creation failure, task poll timeout, task server-side failure) need distinct error messages to avoid misdiagnosis.

5. **Leaking the polling goroutine on context cancellation.** If polling is dispatched to a goroutine, cancellation of the caller's context must unblock the goroutine. Use a `select` on `ctx.Done()` and the poll ticker channel, not just a `time.Sleep`.

**Prevention:** All polling loops must use `select { case <-ctx.Done(): return ctx.Err(); case <-ticker.C: }`. Tests must use `context.WithTimeout` with a short deadline and assert the poll loop returns `context.DeadlineExceeded` rather than hanging. Also test that a task that returns `"failed"` immediately produces a non-nil error without spinning.

**Phase to address:** Phase 5 (Twelve Labs async — #479)

---

### Pitfall 6: Error Body Truncation That Hides Useful Debugging Information

**What exists now:** `ReadLimitedBody` in `pkg/commons/http/utils.go` truncates at 200 MB — a transport-safety cap, not a user-facing display limit. The issue (#478) is that providers include the raw (potentially large) body verbatim in error messages, making logs noisy for large HTML error pages but still potentially hiding information if the truncation limit is set too low for error messages.

**Pitfall when fixing:** The fix is almost certainly to add a second, much lower truncation limit for error message display (e.g., 4 KB) while keeping `ReadLimitedBody` for the actual read. The temptation is to change `MaxResponseBodySize` itself — do not. That constant is a security safeguard for allocations; reducing it would break providers that embed large base64 payloads in response bodies.

**Pitfall: applying the truncation inconsistently.** There are 20+ call sites in provider files (all currently use `ReadLimitedBody` correctly). If the fix is a new helper like `TruncateErrorBody(body []byte, maxLen int) string` and is called at each error-message construction site, it is easy to miss one provider. The safest approach is a helper in `pkg/commons/http` that takes the raw bytes and returns a display string, then audit all `errors.Errorf(..., string(respData))` call sites across providers.

**Pitfall: stripping the truncation indicator.** If a large error body is truncated to 4 KB for display, the error message must say "[truncated]" or include the byte count so users know they are not seeing the full message.

**Prevention:** Add a `FormatErrorBody(body []byte) string` helper to `pkg/commons/http` and replace all `string(respData)` in error paths. Test with a 1 MB synthetic error body and assert the returned error string is below 5 KB with a truncation suffix.

**Phase to address:** Phase 6 (error truncation — #478)

---

### Pitfall 7: Download Stack Refactor Must Not Change Behavior for Existing Callers

**What exists now:** `client_local_library_download.go` has a mature, well-tested download stack with cosign verification, lock files, heartbeats, and fallback URLs. The `downloadutil` package in `pkg/internal/downloadutil` provides a generic HTTP download primitive. The refactor (#412) aims to reduce duplication between the embedded client's download logic and any other provider that fetches artifacts at runtime (primarily ORT model files, if applicable).

**Pitfalls when refactoring:**

1. **Moving or renaming test-injectable function variables.** The download file has `localDownloadFileFunc`, `localEnsureLibraryDownloadedFunc`, etc. as package-level `var` for test injection. If the refactor moves these to a struct or renames them, existing tests in `client_local_library_download_test.go` will break. Preserve the injection points or migrate the tests atomically.

2. **Changing the retry semantics.** `localDownloadFileWithRetry` at line 798 wraps `downloadutil.DownloadFileWithRetry`. If refactoring moves retry logic into the shared util, the retry count or backoff defaults must remain identical — otherwise live embedded client downloads get different retry behavior.

3. **Breaking cosign verification integration.** The cosign path calls `localVerifyCosignCertificateChainFunc` which is also test-injectable. Refactoring must not untangle this from the download path or make it conditional where it was previously unconditional.

**Prevention:** Run the full `client_local_library_download_test.go` suite against the refactored code with no changes to test code. Any test failure is a behavior regression. Keep the public `localDownloadFileWithRetry` and `localEnsureLibraryDownloadedFunc` vars at their original package level.

**Phase to address:** Phase 7 (download stack refactor — #412)

---

### Pitfall 8: Morph EF Test Fix Must Handle Upstream 404 Without Silently Skipping

**What is broken:** The Morph EF integration test returns a 404 from the upstream endpoint (#465). The fix is likely one of: (a) update the endpoint URL, (b) skip with `t.Skip` when the endpoint is unavailable, or (c) mock the endpoint in unit tests.

**Pitfall:** `t.Skip` based on a 404 response is only appropriate for integration tests tagged with a build tag that does not run in CI by default. If the Morph EF test is currently tagged with `ef` and runs in `make test-ef`, silently skipping would hide a real regression in the Morph provider itself.

**Prevention:** Check whether the upstream Morph API endpoint has moved (it may be a permanent URL change). If so, update the URL in the provider. If the endpoint is genuinely flaky or gated, gate the test with an environment variable check (`os.Getenv("MORPH_API_KEY") == ""` → `t.Skip`) rather than relying on the 404 response.

**Phase to address:** Phase 8 (Morph EF test — #465)

---

## Minor Pitfalls

### Pitfall 9: closeOnce Wrappers — Layering Concern When Wrapping an Already-Wrapped EF

**What exists now:** `wrapEFCloseOnce` and `wrapContentEFCloseOnce` both check whether the incoming EF is already a `*closeOnceEF` or `*closeOnceContentEF` and skip double-wrapping. This is correct.

**Pitfall in the #493/#494 fixes:** Any change to `CreateCollection` or `GetOrCreateCollection` that adds additional wrapping calls must go through `wrapEFCloseOnce` — never wrap directly with `&closeOnceEF{ef: ...}`. Bypassing the idempotency guard produces a double-wrapped EF where the outer `Close()` only closes the inner wrapper's `sync.Once`, not the actual resource.

**Prevention:** All wrapping in `client_local_embedded.go` must use `wrapEFCloseOnce` / `wrapContentEFCloseOnce`. Add a test that passes an already-wrapped EF through the code path and asserts the result is pointer-equal to the input (no new wrapper allocated).

---

### Pitfall 10: Using `append([]CreateCollectionOption{}, options...)` Creates a Shallow Copy — EF Reference Is Shared

**What exists now:** `GetOrCreateCollection` at line 454 does `createOptions := append([]CreateCollectionOption{}, options...)`. This copies the slice header but the EF inside the option values is the same pointer. If a close-once EF was constructed earlier in the function and the same option is reused, both paths share the same EF state.

**Pitfall:** This is a latent issue. It only becomes a real bug if a future change closes the EF on the get path before the create path runs. Currently it is safe because `GetCollection` does not close EFs it receives. It becomes dangerous the moment any hardening adds cleanup-on-failure logic to `GetCollection`.

**Prevention:** Document this assumption explicitly in the code with a comment. Do not add EF cleanup to `GetCollection` without auditing all callers that share EF references.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|----------------|------------|
| RrfRank arithmetic fix (#481) | Switching no-ops to real expression nodes changes behavior for any caller who chains on RrfRank | Return an `ErrorRank` sentinel instead; document at the callsite |
| WithGroupBy(nil) fix (#482) | nil sentinel conventionally means "absence" in Go; callers may use it intentionally | Add nil check at `WithGroupBy` construction; CHANGELOG note for the behavior change |
| GetOrCreateCollection EF race (#493) | State map mutation races with concurrent callers | Hold write lock over the get-then-create decision; unit test with `-race` flag |
| ORT EF leak on existing collection (#494) | Closing a caller-owned EF the collection did not adopt | Expose lifecycle ownership clearly; if not adopted, log and leave to caller |
| Twelve Labs async polling (#479) | Context not threaded into poll loop; tight poll loop; terminal states not exhaustive | All poll iterations must `select` on `ctx.Done()`; test with short deadline |
| Error truncation (#478) | Truncation hides info vs. truncation threshold changed for transport safety | Add display-layer truncation helper; do not change `MaxResponseBodySize` |
| Download stack refactor (#412) | Test-injectable vars renamed or moved; retry/cosign semantics changed | Run existing tests unchanged; preserve injection var names |
| Morph EF test fix (#465) | `t.Skip` on 404 masks provider regression | Prefer URL update or env-var gate over `t.Skip` |

## "Looks Done But Isn't" Checklist

- [ ] **RrfRank arithmetic:** All 10 arithmetic methods return an `ErrorRank` (not `self`, not a real expression node). `MarshalJSON()` on the result returns non-nil error. Test coverage added.
- [ ] **WithGroupBy(nil):** `NewSearchRequest(..., WithGroupBy(nil))` returns a non-nil error. Test coverage added.
- [ ] **GetOrCreateCollection EF race:** Test passes under `-race` flag for concurrent get-or-create on a non-existent collection. Returned collections embed successfully.
- [ ] **ORT EF leak:** `goleak` or close-spy test confirms no leaked EF when `CreateCollection` finds an existing collection.
- [ ] **Twelve Labs async:** Poll loop exits with `context.DeadlineExceeded` when context times out. Task `"failed"` state returns error without spinning.
- [ ] **Error truncation:** Error strings from all providers include `[truncated]` suffix when body exceeds display limit. `MaxResponseBodySize` in `utils.go` is unchanged.
- [ ] **Download refactor:** Existing `client_local_library_download_test.go` passes unchanged. Injectable vars preserved.
- [ ] **Morph EF test:** Test does not `t.Skip` on 404 unless gated by an explicit env var or build tag.

## Sources

- `pkg/api/v2/rank.go` — live `RrfRank` no-op methods and `UnknownRank` sentinel pattern (lines 1129–1168, 81–88)
- `pkg/api/v2/search.go` — `WithGroupBy` nil handling (lines 631–640)
- `pkg/api/v2/client_local_embedded.go` — `CreateCollection` isNewCreation logic (lines 366–418), `GetOrCreateCollection` options-reuse (lines 420–461)
- `pkg/api/v2/ef_close_once.go` — `wrapEFCloseOnce` idempotency guard (lines 198–219)
- `pkg/embeddings/twelvelabs/twelvelabs.go` — synchronous `doPost` path (lines 156–194)
- `pkg/embeddings/twelvelabs/content.go` — current sync embedding flow
- `pkg/commons/http/utils.go` — `ReadLimitedBody` (200 MB cap) and `ReadRespBody` (unbounded)
- `pkg/api/v2/client_local_library_download.go` — test-injectable vars (lines 49–68), retry wrapper (line 798)
- `pkg/internal/downloadutil/download.go` — shared download primitive

---
*Pitfalls research for: v0.4.2 bug fixes and robustness — brownfield Go SDK*
*Researched: 2026-04-08*
