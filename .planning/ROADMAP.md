# Roadmap: Chroma Go

## Milestones

- ✅ **v0.4.1 Provider-Neutral Multimodal Foundations** — Phases 1-20 (shipped 2026-04-08)
- 🚧 **v0.4.2 Bug Fixes and Robustness** — Phases 21-30 (in progress)

## Phases

<details>
<summary>v0.4.1 Provider-Neutral Multimodal Foundations (Phases 1-20) - SHIPPED 2026-04-08</summary>

See: [v0.4.1 Archived Roadmap](milestones/v0.4.1-ROADMAP.md)

</details>

### v0.4.2 Bug Fixes and Robustness (In Progress)

**Milestone Goal:** Fix API bugs, harden embedded client lifecycle, and clean up error handling across embedding providers.

**Phase Numbering:**
- Integer phases (21, 22, ...): Planned milestone work
- Decimal phases (21.1, 21.2): Urgent insertions (marked with INSERTED)

- [x] **Phase 21: RrfRank Arithmetic Fix** - RrfRank arithmetic methods compute correct results instead of silently returning self (completed 2026-04-09)
- [x] **Phase 22: WithGroupBy Validation** - WithGroupBy(nil) returns an error instead of silently skipping grouping (completed 2026-04-10)
- [x] **Phase 23: ORT EF Leak Fix** - Default ORT EF is properly closed when CreateCollection finds an existing collection (completed 2026-04-11)
- [x] **Phase 24: GetOrCreateCollection EF Safety** - GetOrCreateCollection does not pass closed EFs to CreateCollection fallback (completed 2026-04-12)
- [ ] **Phase 25: Error Body Truncation** - Embedding provider error messages truncate raw HTTP bodies to safe display lengths
- [ ] **Phase 26: Twelve Labs Async Embedding** - Twelve Labs provider handles async task responses for long-running media
- [ ] **Phase 27: Download Stack Consolidation** - default_ef download code uses shared downloadutil instead of its own HTTP implementation
- [ ] **Phase 28: Morph Test Fix** - Morph EF integration test handles upstream 404 gracefully
- [ ] **Phase 29: Rank Expression Composition Robustness** - Reject silent footguns in rank composition (nil operands, degenerate RRF compositions)
- [ ] **Phase 30: V2 SearchRequestOption Nil Consistency** - Normalize explicit-nil handling across sibling search option helpers to match the Phase 22 contract

## Phase Details

### Phase 21: RrfRank Arithmetic Fix
**Goal**: RrfRank arithmetic operations produce correct composite rank expressions
**Depends on**: Nothing (independent bug fix)
**Requirements**: RANK-01, RANK-02
**Success Criteria** (what must be TRUE):
  1. Calling Multiply, Sub, Add, Div, or Negate on an RrfRank returns a new rank value reflecting the computation, not the original receiver
  2. The computed rank values marshal to valid JSON that Chroma accepts
  3. Tests confirm each arithmetic method produces distinct output from its input
**Plans**: 1 plan

Plans:
- [x] 21-01-PLAN.md — Fix RrfRank arithmetic methods and add test coverage

### Phase 21.1: RRF cloud integration test coverage including arithmetic compositions (INSERTED)

**Goal:** Add Chroma Cloud integration test coverage for all 10 RrfRank arithmetic methods (Add, Sub, Multiply, Div, Negate, Abs, Exp, Log, Max, Min) end-to-end against a real Chroma Cloud instance, closing the cloud-test-bar gap left by Phase 21 (which shipped structural unit tests only).
**Requirements**: D-01..D-22 (CONTEXT.md decision IDs — phase has no REQ-IDs because it's an inserted urgent-work phase)
**Depends on:** Phase 21
**Success Criteria** (what must be TRUE):
  1. `TestCloudClientSearchRRFArithmetic` exists in `pkg/api/v2/client_cloud_test.go` exercising all 10 methods in a single table-driven function under build tag `basicv2 && cloud`
  2. Safe-bucket methods (Add, Sub, Multiply, Div) assert strict differential against an RRF baseline
  3. Semflip + degenerate methods (Negate, Abs, Exp, Log, Max(0), Min(0)) have empirically pinned assertions reflecting actual server behavior
  4. `make test-cloud -run TestCloudClientSearchRRFArithmetic` passes against a real Chroma Cloud instance (D-21, user-run gate per D-22)
**Plans**: 2 plans

Plans:
- [x] 21.1-01-PLAN.md — Pass 1 scaffolding: TestCloudClientSearchRRFArithmetic with all 10 rows, safe-bucket strict differential, semflip+degenerate observe-only
- [x] 21.1-02-PLAN.md — Pass 2 empirical tightening: per-row pinned assertions from user observations + [BUG] issues + D-21 user-run gate

### Phase 22: WithGroupBy Validation
**Goal**: WithGroupBy rejects nil input with a clear error
**Depends on**: Nothing (independent bug fix)
**Requirements**: GRP-01
**Success Criteria** (what must be TRUE):
  1. Passing nil to WithGroupBy returns a validation error before the request is sent
  2. Non-nil WithGroupBy calls continue to work as before
**Plans**: 1 plan

Plans:
- [x] 22-01-PLAN.md — Validate `WithGroupBy(nil)` fail-fast behavior and pin direct/request-construction regressions

### Phase 23: ORT EF Leak Fix
**Goal**: Default ORT embedding function is properly cleaned up when CreateCollection encounters an existing collection
**Depends on**: Nothing (independent bug fix)
**Requirements**: EFL-01
**Success Criteria** (what must be TRUE):
  1. When CreateCollection finds an existing collection, any default ORT EF created by PrepareAndValidateCollectionRequest is closed
  2. No ORT runtime resources remain open after CreateCollection returns in the existing-collection path
**Plans**: 1 plan

Plans:
- [x] 23-01-PLAN.md — Close the temporary SDK-owned default ORT EF on the embedded existing-collection path and pin close/error/state-preservation regressions

### Phase 24: GetOrCreateCollection EF Safety
**Goal**: GetOrCreateCollection never passes a closed EF to CreateCollection fallback
**Depends on**: Phase 23
**Requirements**: EFL-02, EFL-03
**Success Criteria** (what must be TRUE):
  1. When GetCollection fails and GetOrCreateCollection falls back to CreateCollection, the EF passed is still open and usable
  2. Concurrent GetOrCreateCollection calls under `-race` do not trigger data races or double-close panics
  3. Tests demonstrate the EF lifecycle under concurrent access
**Plans**: 1 plan

Plans:
- [x] 24-01-PLAN.md — Harden provisional embedded EF ownership and concurrent loser convergence for `GetOrCreateCollection`

### Phase 25: Error Body Truncation
**Goal**: Embedding provider errors display safe-length messages instead of arbitrarily large raw HTTP bodies
**Depends on**: Nothing (independent cleanup)
**Requirements**: ERR-01, ERR-02
**Success Criteria** (what must be TRUE):
  1. A shared SanitizeErrorBody utility exists that truncates bodies exceeding a safe display length and appends a `[truncated]` suffix
  2. All embedding providers use SanitizeErrorBody when constructing error messages from HTTP responses
  3. Error messages from providers with large error bodies are readable (not multi-KB dumps)
**Plans**: 4 plans

Plans:
- [x] 25-01-PLAN.md — Create the shared sanitizer contract with panic recovery and normalize Perplexity/OpenRouter
- [x] 25-02-PLAN.md — Migrate the representative raw-body providers and add OpenAI/Baseten regressions
- [x] 25-03-PLAN.md — Finish batch A with Cloudflare/Cohere/HF/Jina and preserve Cloudflare mixed formatting
- [x] 25-04-PLAN.md — Finish batch B, sanitize Twelve Labs structured errors, and run the full embedding sweep/lint

### Phase 26: Twelve Labs Async Embedding
**Goal**: Twelve Labs provider handles async task responses for long-running audio and video embeddings
**Depends on**: Phase 25 (error truncation should be in place before new provider work)
**Requirements**: TLA-01, TLA-02, TLA-03, TLA-04
**Success Criteria** (what must be TRUE):
  1. When the sync endpoint returns an async task response, the provider detects it and enters a polling loop
  2. Polling respects the caller's context.Context for cancellation and timeout
  3. Terminal states (ready, failed) are handled with appropriate result delivery or error messages
  4. Tests cover async task creation, polling to completion, polling to failure, and context cancellation
**Plans**: TBD

Plans:
- [ ] 26-01: TBD

### Phase 27: Download Stack Consolidation
**Goal**: default_ef download code uses the shared downloadutil package instead of its own HTTP download implementation
**Depends on**: Nothing (independent refactor)
**Requirements**: DL-01, DL-02
**Success Criteria** (what must be TRUE):
  1. `default_ef/download_utils.go` delegates to `downloadutil.DownloadFile` instead of implementing its own HTTP download
  2. All existing download tests pass unchanged after the consolidation
**Plans**: TBD

Plans:
- [ ] 27-01: TBD

### Phase 28: Morph Test Fix
**Goal**: Morph EF integration test handles upstream 404 gracefully
**Depends on**: Nothing (independent test fix)
**Requirements**: MORPH-01
**Success Criteria** (what must be TRUE):
  1. The Morph EF integration test does not fail due to upstream 404 errors (either skips with a message or uses an updated URL)
  2. When Morph upstream is available, the test exercises the embedding function end-to-end
**Plans**: TBD

Plans:
- [ ] 28-01: TBD

### Phase 29: Rank Expression Composition Robustness
**Goal**: Rank expression composition fails loud on programmer errors and rejects mathematically meaningless RRF compositions before sending to the server
**Depends on**: Phase 21 (arithmetic methods must build expression trees before they can be validated)
**Requirements**: COMP-01, COMP-02, COMP-03
**Issues**: amikos-tech/chroma-go#499, amikos-tech/chroma-go#500, amikos-tech/chroma-go#501
**Success Criteria** (what must be TRUE):
  1. Passing nil to any `*Rank.Add/Sub/Multiply/Div/Max/Min` produces a rank whose `MarshalJSON` reports a clear error instead of silently substituting `Val(0)` (#499)
  2. `RrfRank.Log()` and `RrfRank.Max(Val(0))` reject the composition at build time with a descriptive error instead of producing a degenerate query (#501)
  3. Client detects and reports result-shape mismatch (empty inner `Scores` with populated inner `IDs`) from `Search` responses so callers see silent server-side degeneration (#500)
  4. `TestCloudClientSearchRRFArithmetic` is updated to assert the new client-side errors on degenerate rows instead of pinning the current fallback behavior
**Plans**: TBD

Plans:
- [ ] 29-01: TBD — Fix `operandToRank` nil handling (#499)
- [ ] 29-02: TBD — Client-side rejection of degenerate RRF compositions (#501)
- [ ] 29-03: TBD — Result-shape validation in `Search` response handling (#500)

### Phase 30: V2 SearchRequestOption Nil Consistency
**Goal**: Normalize explicit-nil handling across sibling V2 SearchRequestOption helpers so callers get a consistent contract instead of a mix of silent omission and validation errors
**Depends on**: Phase 22
**Requirements**: OPT-01
**Issues**: amikos-tech/chroma-go#503
**Success Criteria** (what must be TRUE):
  1. The nil-handling contract for sibling SearchRequestOption helpers in `pkg/api/v2/search.go` is explicitly defined and tested
  2. Options that should reject explicit nil inputs fail with clear validation errors instead of silently degrading caller intent
  3. Any options intentionally left permissive are documented and covered by tests so the API behavior is consistent and discoverable
**Plans**: TBD

Plans:
- [ ] 30-01: TBD — Audit sibling SearchRequestOption helpers and pin a consistent explicit-nil contract (#503)

## Progress

**Execution Order:**
Phases execute in numeric order: 21 -> 22 -> 23 -> 24 -> ... -> 30.
Phases 21, 22, 25, 27, 28 are independent and may execute in any order relative to each other.
Phase 24 depends on Phase 23. Phase 26 depends on Phase 25. Phase 29 depends on Phase 21. Phase 30 depends on Phase 22.

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 21. RrfRank Arithmetic Fix | v0.4.2 | 1/1 | Complete    | 2026-04-09 |
| 22. WithGroupBy Validation | v0.4.2 | 1/1 | Complete    | 2026-04-10 |
| 23. ORT EF Leak Fix | v0.4.2 | 1/1 | Complete    | 2026-04-11 |
| 24. GetOrCreateCollection EF Safety | v0.4.2 | 1/1 | Complete    | 2026-04-12 |
| 25. Error Body Truncation | v0.4.2 | 2/4 | In Progress|  |
| 26. Twelve Labs Async Embedding | v0.4.2 | 0/0 | Not started | - |
| 27. Download Stack Consolidation | v0.4.2 | 0/0 | Not started | - |
| 28. Morph Test Fix | v0.4.2 | 0/0 | Not started | - |
| 29. Rank Expression Composition Robustness | v0.4.2 | 0/3 | Not started | - |
| 30. V2 SearchRequestOption Nil Consistency | v0.4.2 | 0/0 | Not started | - |
