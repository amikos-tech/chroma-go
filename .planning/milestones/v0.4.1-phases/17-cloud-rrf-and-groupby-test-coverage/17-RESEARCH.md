# Phase 17: Cloud RRF and GroupBy Test Coverage - Research

**Researched:** 2026-04-02
**Domain:** Cloud integration testing -- Search API RRF (Reciprocal Rank Fusion) and GroupBy primitives
**Confidence:** HIGH

## Summary

This phase adds end-to-end cloud integration tests for two Search API features: RRF (Reciprocal Rank Fusion) for hybrid dense+sparse ranking, and GroupBy for metadata-based result grouping with MinK/MaxK aggregation. No production code changes are required -- this is test-only work.

The existing cloud test file (`pkg/api/v2/client_cloud_test.go`) provides all necessary infrastructure: `setupCloudClient()`, collection naming conventions, build tags, and cleanup patterns. The Search API implementation (`search.go`, `rank.go`, `groupby.go`, `aggregate.go`) is fully built and unit-tested locally; what is missing is cloud-level verification that these constructs are accepted and produce correct behavioral results against a live Chroma Cloud instance.

**Primary recommendation:** Add 3-4 subtests under a new `TestCloudClientSearchRRF` function and 2-3 subtests under a new `TestCloudClientSearchGroupBy` function, both following the established cloud test patterns. RRF tests require a collection with a sparse vector index (using `chromacloudsplade` EF), while GroupBy tests work on standard collections with metadata.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Use `chromacloudsplade` embedding function for real server-side sparse vectors in RRF tests (already imported in cloud test file). Do not use manual/hand-crafted sparse embeddings.
- **D-02:** Behavioral assertions -- verify ordering changes with different RRF weights, confirm GroupBy enforces per-group MinK/MaxK caps. Not just smoke-level "no errors" checks.
- **D-03:** Add all new tests to the existing `pkg/api/v2/client_cloud_test.go`. Same build tags (`basicv2 && cloud`), same `setupCloudClient()` helper, same cleanup pattern.

### Claude's Discretion
- Test data content (document texts, metadata values) -- choose whatever produces meaningful behavioral assertions
- Number of documents per test collection -- enough to demonstrate ranking/grouping effects
- Specific RRF weight values -- pick values that produce observable ordering differences
- Exact RRF k parameter values for custom-k tests

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `testing` | stdlib | Test framework | Go standard |
| `testify` | v1.x | Assertions (require/assert) | Project convention per CLAUDE.md |
| `chromacloudsplade` | internal | Sparse EF for RRF tests | Decision D-01: server-side SPLADE embeddings |
| `chromacloud` | internal | Dense EF for collection creation | Already used in cloud test infrastructure |
| `uuid` | google/uuid | Unique collection names | Existing cloud test pattern |
| `godotenv` | joho/godotenv | Load .env for cloud credentials | Existing cloud test pattern |

### Supporting
No additional libraries needed. All dependencies are already imported in the existing cloud test file.

## Architecture Patterns

### Test File Organization (LOCKED by D-03)

All new tests go into `pkg/api/v2/client_cloud_test.go` with existing build tags:
```go
//go:build basicv2 && cloud
```

### RRF Test Structure

RRF tests require a collection with BOTH dense and sparse vector indexes. The pattern for this already exists in `TestCloudClientAutoWire`:

```go
// 1. Create sparse EF
sparseEF, err := chromacloudsplade.NewEmbeddingFunction(chromacloudsplade.WithEnvAPIKey())
require.NoError(t, err)

// 2. Build schema with sparse vector index
schema, err := NewSchema(
    WithDefaultVectorIndex(NewVectorIndexConfig(WithSpace(SpaceL2))),
    WithSparseVectorIndex("sparse_embedding", NewSparseVectorIndexConfig(
        WithSparseEmbeddingFunction(sparseEF),
        WithSparseSourceKey("#document"),
    )),
)
require.NoError(t, err)

// 3. Create collection with schema
collection, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
```

### RRF Search Pattern

```go
// Create KNN ranks with WithKnnReturnRank() -- REQUIRED for RRF
denseKnn, err := NewKnnRank(
    KnnQueryText("search text"),
    WithKnnReturnRank(),
    WithKnnLimit(100),
)
require.NoError(t, err)

sparseKnn, err := NewKnnRank(
    KnnQueryText("search text"),
    WithKnnKey(K("sparse_embedding")),
    WithKnnReturnRank(),
    WithKnnLimit(100),
)
require.NoError(t, err)

// Execute RRF search
result, err := collection.Search(ctx,
    NewSearchRequest(
        WithRrfRank(
            WithRrfRanks(
                denseKnn.WithWeight(1.0),
                sparseKnn.WithWeight(1.0),
            ),
            WithRrfK(60),
        ),
        NewPage(Limit(10)),
        WithSelect(KDocument, KScore, KID),
    ),
)
```

### GroupBy Search Pattern

```go
result, err := collection.Search(ctx,
    NewSearchRequest(
        WithKnnRank(KnnQueryText("query"), WithKnnLimit(100)),
        WithGroupBy(NewGroupBy(
            NewMinK(2, KScore),     // 2 results per group
            K("category"),          // group by "category" metadata
        )),
        NewPage(Limit(30)),
        WithSelect(KDocument, KScore, KMetadata),
    ),
)
```

### Established Test Patterns (from existing cloud tests)

1. **Collection naming:** `"test_" + descriptiveName + "-" + uuid.New().String()`
2. **Indexing delay:** `time.Sleep(2 * time.Second)` after `Add()` to allow indexing
3. **Result type assertion:** `sr := results.(*SearchResultImpl)` then use `.Rows()`, `.IDs`, `.Scores`
4. **Context with timeout:** Use `context.Background()` for simple tests; `context.WithTimeout()` for longer operations
5. **Cleanup:** TestCloudCleanup sweeps `test_` prefixed collections automatically

### Anti-Patterns to Avoid
- **Missing `WithKnnReturnRank()`:** RRF ranks MUST use return_rank=true. Without it, ranks get distance values instead of position ordinals, producing incorrect RRF computation.
- **Reusing KnnRank across search calls:** Each `NewKnnRank()` should be freshly created per search request since the object may carry state.
- **Insufficient data for behavioral assertions:** GroupBy needs enough documents across enough groups to verify MinK/MaxK caps. Use at least 3 groups with 3+ documents each.
- **Smoke-only assertions:** Decision D-02 explicitly requires behavioral verification, not just "no errors."

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Sparse embeddings | Manual sparse vectors | `chromacloudsplade` EF | Decision D-01; server-side SPLADE produces realistic sparse vectors |
| Collection cleanup | Per-test cleanup logic | `TestCloudCleanup` + `test_` prefix naming | Existing cleanup infrastructure |
| Cloud client setup | Manual env var parsing | `setupCloudClient(t)` | Existing helper with proper cleanup |
| UUID generation | Custom ID schemes | `uuid.New().String()` for collection names; explicit IDs for documents | Existing pattern |

## Common Pitfalls

### Pitfall 1: Indexing Delay Insufficiency
**What goes wrong:** Tests add documents then immediately query, getting empty or incomplete results.
**Why it happens:** Cloud indexing is asynchronous; documents may not be searchable immediately after Add.
**How to avoid:** Always include `time.Sleep(2 * time.Second)` after Add operations, matching the existing test pattern.
**Warning signs:** Flaky tests that pass sometimes and fail on CI.

### Pitfall 2: Sparse Index Not Created for RRF
**What goes wrong:** RRF search with sparse KNN returns an error because the collection has no sparse vector index.
**Why it happens:** Regular `CreateCollection` does not create sparse indexes. Must use `WithSchemaCreate` with explicit `WithSparseVectorIndex`.
**How to avoid:** Follow the schema pattern from `TestCloudClientAutoWire` -- create schema with sparse vector index configuration.
**Warning signs:** Error mentioning "sparse_embedding" key not found.

### Pitfall 3: RRF Without ReturnRank
**What goes wrong:** RRF computation uses raw distances instead of rank positions, producing meaningless fusion scores.
**Why it happens:** Forgetting `WithKnnReturnRank()` on component KNN ranks.
**How to avoid:** Always include `WithKnnReturnRank()` on every KNN rank passed to `WithRrfRanks()`.
**Warning signs:** RRF scores that look like negative distances rather than rank-based fusion scores.

### Pitfall 4: GroupBy Requires Sufficient KNN Limit
**What goes wrong:** GroupBy returns fewer results per group than expected.
**Why it happens:** The KNN candidate pool (WithKnnLimit) is too small to include enough documents from all groups.
**How to avoid:** Set `WithKnnLimit` high enough to cover all expected groups. For 3 groups with 3+ docs each, use at least `WithKnnLimit(50)`.
**Warning signs:** Some groups missing from results; fewer results per group than the MinK/MaxK k value.

### Pitfall 5: Score Ordering Direction
**What goes wrong:** Assertions about "best" results check the wrong direction.
**Why it happens:** RRF scores are negated (`-sum(...)`) so more negative = better. Standard KNN distance scores: lower = better.
**How to avoid:** For RRF, the most relevant results have the most negative (smallest) scores. For KNN, smallest scores are best.
**Warning signs:** Assertions failing because top results have "worse" scores than expected.

## Code Examples

### RRF Smoke Test Pattern (dense + sparse)
```go
// Source: Existing cloud test patterns + rank.go API
t.Run("RRF with dense and sparse KNN ranks", func(t *testing.T) {
    ctx := context.Background()
    collectionName := "test_rrf_smoke-" + uuid.New().String()
    
    sparseEF, err := chromacloudsplade.NewEmbeddingFunction(chromacloudsplade.WithEnvAPIKey())
    require.NoError(t, err)
    
    schema, err := NewSchema(
        WithDefaultVectorIndex(NewVectorIndexConfig(WithSpace(SpaceL2))),
        WithSparseVectorIndex("sparse_embedding", NewSparseVectorIndexConfig(
            WithSparseEmbeddingFunction(sparseEF),
            WithSparseSourceKey("#document"),
        )),
    )
    require.NoError(t, err)
    
    collection, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
    require.NoError(t, err)
    
    // Add diverse documents
    err = collection.Add(ctx,
        WithIDs("1", "2", "3", "4", "5"),
        WithTexts(
            "quantum computing advances in 2024",
            "classical music theory and harmony",
            "quantum mechanics and particle physics",
            "cooking recipes for beginners",
            "quantum entanglement research papers",
        ),
    )
    require.NoError(t, err)
    time.Sleep(2 * time.Second)
    
    // Dense KNN
    denseKnn, err := NewKnnRank(
        KnnQueryText("quantum physics"),
        WithKnnReturnRank(),
        WithKnnLimit(10),
    )
    require.NoError(t, err)
    
    // Sparse KNN on sparse index
    sparseKnn, err := NewKnnRank(
        KnnQueryText("quantum physics"),
        WithKnnKey(K("sparse_embedding")),
        WithKnnReturnRank(),
        WithKnnLimit(10),
    )
    require.NoError(t, err)
    
    results, err := collection.Search(ctx,
        NewSearchRequest(
            WithRrfRank(
                WithRrfRanks(
                    denseKnn.WithWeight(1.0),
                    sparseKnn.WithWeight(1.0),
                ),
            ),
            NewPage(Limit(5)),
            WithSelect(KID, KDocument, KScore),
        ),
    )
    require.NoError(t, err)
    
    sr := results.(*SearchResultImpl)
    require.NotEmpty(t, sr.IDs)
    require.NotEmpty(t, sr.Scores)
    // Verify quantum-related docs rank higher
})
```

### GroupBy with MinK Pattern
```go
// Source: groupby.go + aggregate.go + existing cloud test patterns
t.Run("GroupBy with MinK caps results per group", func(t *testing.T) {
    ctx := context.Background()
    collectionName := "test_groupby_mink-" + uuid.New().String()
    
    collection, err := client.CreateCollection(ctx, collectionName)
    require.NoError(t, err)
    
    // Add documents with category metadata -- 3 categories, multiple docs each
    err = collection.Add(ctx,
        WithIDs("1", "2", "3", "4", "5", "6", "7", "8", "9"),
        WithTexts(
            "machine learning basics", "deep learning tutorial", "neural network guide",
            "python web framework", "javascript frontend", "react component design",
            "quantum computing intro", "quantum algorithms", "quantum error correction",
        ),
        WithMetadatas(
            NewDocumentMetadata(NewStringAttribute("category", "AI")),
            NewDocumentMetadata(NewStringAttribute("category", "AI")),
            NewDocumentMetadata(NewStringAttribute("category", "AI")),
            NewDocumentMetadata(NewStringAttribute("category", "web")),
            NewDocumentMetadata(NewStringAttribute("category", "web")),
            NewDocumentMetadata(NewStringAttribute("category", "web")),
            NewDocumentMetadata(NewStringAttribute("category", "quantum")),
            NewDocumentMetadata(NewStringAttribute("category", "quantum")),
            NewDocumentMetadata(NewStringAttribute("category", "quantum")),
        ),
    )
    require.NoError(t, err)
    time.Sleep(2 * time.Second)
    
    results, err := collection.Search(ctx,
        NewSearchRequest(
            WithKnnRank(KnnQueryText("learning"), WithKnnLimit(50)),
            WithGroupBy(NewGroupBy(
                NewMinK(2, KScore), // At most 2 per group
                K("category"),
            )),
            NewPage(Limit(20)),
            WithSelect(KID, KDocument, KScore, KMetadata),
        ),
    )
    require.NoError(t, err)
    
    sr := results.(*SearchResultImpl)
    require.NotEmpty(t, sr.IDs)
    
    // Count results per category
    categoryCounts := map[string]int{}
    for _, row := range sr.Rows() {
        cat, ok := row.Metadata.GetString("category")
        require.True(t, ok)
        categoryCounts[cat]++
    }
    
    // Each category should have at most 2 results (MinK k=2)
    for cat, count := range categoryCounts {
        assert.LessOrEqual(t, count, 2, "category %s has %d results, expected <= 2", cat, count)
    }
})
```

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify v1.x |
| Config file | Makefile targets + build tags |
| Quick run command | `go test -tags=basicv2,cloud -run TestCloudClientSearchRRF -v -timeout=5m ./pkg/api/v2/...` |
| Full suite command | `make test-cloud` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SC-01 | RRF smoke: dense + sparse KNN ranks with WithKnnReturnRank | cloud integration | `go test -tags=basicv2,cloud -run TestCloudClientSearchRRF/smoke -v -timeout=5m ./pkg/api/v2/...` | Wave 0 |
| SC-02 | RRF weighted/custom-k: request accepted, ordering changes | cloud integration | `go test -tags=basicv2,cloud -run TestCloudClientSearchRRF/weighted -v -timeout=5m ./pkg/api/v2/...` | Wave 0 |
| SC-03 | GroupBy MinK: per-group caps enforced | cloud integration | `go test -tags=basicv2,cloud -run TestCloudClientSearchGroupBy/MinK -v -timeout=5m ./pkg/api/v2/...` | Wave 0 |
| SC-04 | GroupBy MaxK: per-group caps enforced, flattened limits | cloud integration | `go test -tags=basicv2,cloud -run TestCloudClientSearchGroupBy/MaxK -v -timeout=5m ./pkg/api/v2/...` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -tags=basicv2,cloud -run "TestCloudClientSearch(RRF|GroupBy)" -v -timeout=5m ./pkg/api/v2/...`
- **Per wave merge:** `make test-cloud`
- **Phase gate:** Full cloud suite green before `/gsd:verify-work`

### Wave 0 Gaps
None -- the existing test file infrastructure covers all phase requirements. No new test framework setup needed. Tests are purely additive to `client_cloud_test.go`.

## Project Constraints (from CLAUDE.md)

- Use build tags `basicv2 && cloud` for cloud tests
- Use `testify` for assertions (`require` and `assert`)
- Run `make lint` before committing
- Use conventional commits
- Do not add verbose comments; code/function names should be self-explanatory
- Never panic in production code (not applicable -- test-only phase)
- `Must*` functions are acceptable in test files

## Key API Reference

### RRF Construction

| Function | Purpose |
|----------|---------|
| `NewRrfRank(opts...)` | Create RRF rank expression |
| `WithRrfRanks(ranks...)` | Add weighted KNN ranks |
| `WithRrfK(k)` | Set smoothing parameter (default: 60) |
| `WithRrfNormalize()` | Normalize weights to sum to 1.0 |
| `knn.WithWeight(w)` | Attach weight to a KNN rank for RRF |
| `WithKnnReturnRank()` | REQUIRED -- return rank position instead of distance |

### GroupBy Construction

| Function | Purpose |
|----------|---------|
| `NewGroupBy(aggregate, keys...)` | Create GroupBy with aggregate and metadata keys |
| `NewMinK(k, keys...)` | Keep k records with smallest values (lower = better) |
| `NewMaxK(k, keys...)` | Keep k records with largest values (higher = better) |

### Search Request Options

| Function | Purpose |
|----------|---------|
| `WithRrfRank(opts...)` | Apply RRF ranking to search request |
| `WithKnnRank(query, opts...)` | Apply KNN ranking to search request |
| `WithGroupBy(groupBy)` | Apply GroupBy to search request |
| `NewPage(Limit(n))` | Set result limit (newer API) |
| `WithSelect(keys...)` | Select fields to return |
| `WithFilter(where)` | Add metadata filter |

### Result Access

| Method | Purpose |
|--------|---------|
| `results.(*SearchResultImpl)` | Type assertion from SearchResult interface |
| `.Rows()` | Get first group as `[]ResultRow` |
| `.RowGroups()` | Get all groups as `[][]ResultRow` |
| `.At(group, index)` | Bounds-checked random access |
| `.IDs[g][i]` | Direct ID access |
| `.Scores[g][i]` | Direct score access |
| `.Metadatas[g][i]` | Direct metadata access |

## Open Questions

1. **Cloud indexing delay for sparse vectors**
   - What we know: Dense vectors typically need 2s sleep. Sparse vector indexing might take longer.
   - What's unclear: Whether the same 2s delay is sufficient for SPLADE sparse vectors to be indexed.
   - Recommendation: Start with 2s. If tests are flaky, increase to 5s. Existing sparse test in `TestCloudClientAutoWire` does not perform search after add (it only tests EF auto-wiring), so there is no established sparse indexing delay pattern.

2. **RRF score direction verification**
   - What we know: RRF scores are negated (`-sum(w/(k+rank))`), so more negative = better. The `Rows()` method returns scores directly.
   - What's unclear: Whether Cloud returns scores sorted in ascending order (best first) or if the client needs to verify ordering.
   - Recommendation: Assert that returned scores are monotonically non-increasing (since RRF negation means first result should have most negative score). If Cloud sorts differently, adjust assertions.

## Sources

### Primary (HIGH confidence)
- `pkg/api/v2/rank.go` -- RRF implementation, KnnRank, WithKnnReturnRank, WithRrfRank
- `pkg/api/v2/search.go` -- Search API, SearchResultImpl, Rows(), SearchRequest
- `pkg/api/v2/groupby.go` -- GroupBy, NewGroupBy
- `pkg/api/v2/aggregate.go` -- MinK, MaxK aggregation types
- `pkg/api/v2/client_cloud_test.go` -- Existing cloud test infrastructure and patterns
- `pkg/api/v2/rank_test.go` -- Unit test patterns for RRF serialization
- `pkg/api/v2/groupby_test.go` -- Unit test patterns for GroupBy serialization
- `docs/go-examples/cloud/search-api/hybrid-search.md` -- RRF usage patterns
- `docs/go-examples/cloud/search-api/group-by.md` -- GroupBy usage patterns

### Secondary (MEDIUM confidence)
- `docs/go-examples/cloud/search-api/ranking.md` -- KNN ranking patterns, score behavior

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all libraries already imported in test file, no new dependencies
- Architecture: HIGH -- test patterns directly replicated from existing cloud tests
- Pitfalls: HIGH -- derived from source code analysis and established cloud test patterns

**Research date:** 2026-04-02
**Valid until:** 2026-05-02 (stable -- test-only phase against well-established API)
