# Phase 17: Cloud RRF and GroupBy Test Coverage - Context

**Gathered:** 2026-04-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Add end-to-end cloud integration tests that exercise Search API RRF (Reciprocal Rank Fusion) and GroupBy primitives against live Chroma Cloud. No new production code — tests only.

</domain>

<decisions>
## Implementation Decisions

### Sparse Embedding Setup
- **D-01:** Use `chromacloudsplade` embedding function for real server-side sparse vectors in RRF tests (already imported in cloud test file). Do not use manual/hand-crafted sparse embeddings.

### Assertion Depth
- **D-02:** Behavioral assertions — verify ordering changes with different RRF weights, confirm GroupBy enforces per-group MinK/MaxK caps. Not just smoke-level "no errors" checks.

### Test File Organization
- **D-03:** Add all new tests to the existing `pkg/api/v2/client_cloud_test.go`. Same build tags (`basicv2 && cloud`), same `setupCloudClient()` helper, same cleanup pattern.

### Claude's Discretion
- Test data content (document texts, metadata values) — choose whatever produces meaningful behavioral assertions
- Number of documents per test collection — enough to demonstrate ranking/grouping effects
- Specific RRF weight values — pick values that produce observable ordering differences

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Search API Implementation
- `pkg/api/v2/search.go` — Search request builder, WithKnnRank, WithGroupBy, WithFilter, WithSelect, WithPage
- `pkg/api/v2/rank.go` — KnnRank, RRFRank, WithKnnReturnRank, RankWithWeight, NewRRFRank, NewRRFRankWithCustomK
- `pkg/api/v2/groupby.go` — GroupBy struct, NewGroupBy
- `pkg/api/v2/aggregate.go` — MinK, MaxK aggregations for GroupBy

### Existing Cloud Tests
- `pkg/api/v2/client_cloud_test.go` — Cloud test infrastructure (setupCloudClient, TestCloudCleanup, TestCloudClientSearch)

### Existing Unit Tests (patterns to follow)
- `pkg/api/v2/rank_test.go` — RRF rank unit tests with serialization
- `pkg/api/v2/groupby_test.go` — GroupBy unit tests
- `pkg/api/v2/collection_http_test.go` — HTTP-level search tests including RRF serialization

### Documentation (API patterns)
- `docs/go-examples/cloud/search-api/ranking.md` — RRF ranking API patterns
- `docs/go-examples/cloud/search-api/hybrid-search.md` — Hybrid dense+sparse search patterns
- `docs/go-examples/cloud/search-api/group-by.md` — GroupBy API patterns

### Issue
- GitHub issue #462 — Cloud RRF and GroupBy test coverage

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `setupCloudClient(t)` — Creates cloud client with env var loading and cleanup
- `TestCloudCleanup` — Sweeps `test_` prefixed collections after test run
- `chromacloud.NewChromaCloudEmbeddingFunction` — Dense cloud EF
- `chromacloudsplade.NewChromaCloudSpladeEmbeddingFunction` — Sparse cloud EF
- `NewRRFRank(k, ranks...)` and `NewRRFRankWithCustomK(k, ranks...)` — RRF builders
- `NewGroupBy(aggregate, keys...)` with `NewMinK(k, keys...)` / `NewMaxK(k, keys...)` — GroupBy builders

### Established Patterns
- Cloud test collections use `"test_" + uuid.New().String()` naming
- `time.Sleep(2 * time.Second)` after Add for indexing delay
- Search results cast to `*SearchResultImpl` for field access
- Build tags: `//go:build basicv2 && cloud`

### Integration Points
- Tests run via `make test-cloud` with `gotestsum`
- Env vars: `CHROMA_API_KEY`, `CHROMA_DATABASE`, `CHROMA_TENANT` (or `.env` file)

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 17-cloud-rrf-and-groupby-test-coverage*
*Context gathered: 2026-04-02*
