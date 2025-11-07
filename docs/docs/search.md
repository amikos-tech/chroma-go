# Search API

The Search API provides advanced querying capabilities with flexible ranking expressions that go beyond simple K-Nearest Neighbors (KNN) search.

!!! info "API Version"
    The Search API is available in the V2 API (`github.com/amikos-tech/chroma-go/pkg/api/v2`).

!!! info "Official Documentation"
    This is a Go implementation of Chroma's Search API. For the official Chroma documentation, see:

    - [Search API Overview](https://docs.trychroma.com/cloud/search-api/overview)
    - [Search API Guide](https://docs.trychroma.com/guides/search)

!!! tip "When to Use Search vs Query"
    - **Use Search** for: Multi-query fusion, weighted combinations, score transformations, hybrid search
    - **Use Query** for: Simple KNN similarity search, backward compatibility

## Overview

The Search API introduces **rank expressions** - composable building blocks that define how search results are scored and ordered. You can combine multiple search strategies, apply mathematical transformations, and create sophisticated ranking formulas.

## Key Features

- **KNN Ranking**: Traditional vector similarity search
- **RRF (Reciprocal Rank Fusion)**: Combine multiple searches intelligently
- **Arithmetic Operations**: Add, subtract, multiply, divide rank expressions
- **Mathematical Functions**: Apply exp, log, abs, max, min transformations
- **Field Selection**: Choose which fields to return (ID, Document, Embedding, Score, URI)
- **Filtering**: Same powerful where filters as Query API
- **Pagination**: Full support for limit and offset

## Quick Start

```go
import (
    v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

// Simple KNN search
results, err := collection.Search(ctx,
    v2.WithSearchRankKnnTexts([]string{"machine learning"}, 5),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
```

## Rank Expressions

Rank expressions define how results are scored and ordered. All rank expressions implement the `RankExpression` interface.

### KNN (K-Nearest Neighbors)

Standard vector similarity search using embeddings.

**With query texts** (will be embedded automatically):

```go
v2.WithSearchRankKnnTexts([]string{"artificial intelligence"}, 10)
```

**With pre-computed embeddings**:

```go
queryEmb := embeddings.NewEmbeddingFromFloat32([]float32{...})
v2.WithSearchRankKnn([]embeddings.Embedding{queryEmb}, 10)
```

**Direct construction**:

```go
knnRank := &v2.KnnRank{
    QueryTexts: []string{"neural networks"},
    K: 10,
}
```

### RRF (Reciprocal Rank Fusion)

Combines multiple ranking expressions using reciprocal rank fusion. Documents appearing in multiple searches get higher scores.

```go
rank1 := &v2.KnnRank{QueryTexts: []string{"AI"}, K: 10}
rank2 := &v2.KnnRank{QueryTexts: []string{"machine learning"}, K: 10}

v2.WithSearchRankRrf(
    []v2.RankExpression{rank1, rank2},
    60,    // k parameter (typically 60)
    true,  // normalize scores
)
```

**RRF Formula**: `score = sum(1 / (k + rank_i))` where k is a constant.

### Arithmetic Operations

Combine rank expressions using mathematical operations.

```go
knn1 := &v2.KnnRank{QueryTexts: []string{"query1"}, K: 10}
knn2 := &v2.KnnRank{QueryTexts: []string{"query2"}, K: 10}

// Addition
combined := v2.AddRanks(knn1, knn2)

// Subtraction
diff := v2.SubRanks(knn1, knn2)

// Multiplication
product := v2.MulRanks(knn1, knn2)

// Division
ratio := v2.DivRanks(knn1, knn2)
```

### Mathematical Functions

Apply mathematical transformations to rank expressions.

```go
baseRank := &v2.KnnRank{QueryTexts: []string{"query"}, K: 20}

// Exponential (emphasize top results)
boosted := v2.ExpRank(baseRank)

// Natural logarithm
logged := v2.LogRank(baseRank)

// Absolute value
absolute := v2.AbsRank(baseRank)

// Maximum
maxed := v2.MaxRank(baseRank)

// Minimum
minimized := v2.MinRank(baseRank)
```

## Field Selection

Control which fields are included in search results.

### System Fields

```go
v2.SelectID        // Document ID
v2.SelectDocument  // Document content
v2.SelectEmbedding // Embedding vector
v2.SelectScore     // Search relevance score
v2.SelectURI       // Document URI
```

### Custom Metadata Fields

```go
v2.SelectKey("author")   // Custom metadata field
v2.SelectKey("title")    // Custom metadata field
```

### Usage

```go
v2.WithSearchSelect(
    v2.SelectID,
    v2.SelectDocument,
    v2.SelectScore,
    v2.SelectKey("category"),
)
```

## Filtering

Use the same where filters as the Query API.

```go
// Simple filter
v2.WithSearchWhere(v2.EqString("category", "tech"))

// Complex filter
v2.WithSearchWhere(v2.And(
    v2.GtInt("rating", 4),
    v2.EqString("status", "published"),
    v2.InString("language", "en", "es"),
))
```

See [Filtering Documentation](filtering.md) for complete filter syntax.

## Pagination

Control result limits and offsets for pagination.

```go
// First page (results 1-10)
v2.WithSearchLimit(10, 0)

// Second page (results 11-20)
v2.WithSearchLimit(10, 10)

// Third page (results 21-30)
v2.WithSearchLimit(10, 20)
```

## Complete Examples

### Example 1: Simple KNN Search

```go
results, err := collection.Search(ctx,
    v2.WithSearchRankKnnTexts([]string{"machine learning"}, 5),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
if err != nil {
    log.Fatal(err)
}

// Process results
for i, id := range results.GetIDGroups()[0] {
    doc := results.GetDocumentsGroups()[0][i]
    score := results.GetScoresGroups()[0][i]

    fmt.Printf("ID: %s, Score: %.4f\n", id, score)
    fmt.Printf("Doc: %s\n\n", doc.ContentString())
}
```

### Example 2: Filtered Search

```go
results, err := collection.Search(ctx,
    v2.WithSearchRankKnnTexts([]string{"Python programming"}, 10),
    v2.WithSearchWhere(v2.And(
        v2.EqString("type", "tutorial"),
        v2.GtInt("rating", 4),
    )),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
```

### Example 3: Multi-Query with RRF

```go
aiSearch := &v2.KnnRank{
    QueryTexts: []string{"artificial intelligence"},
    K: 20,
}

dlSearch := &v2.KnnRank{
    QueryTexts: []string{"deep learning"},
    K: 20,
}

results, err := collection.Search(ctx,
    v2.WithSearchRankRrf(
        []v2.RankExpression{aiSearch, dlSearch},
        60,
        true,
    ),
    v2.WithSearchLimit(10, 0),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
```

### Example 4: Hybrid Search with Score Transformation

```go
semanticRank := &v2.KnnRank{
    QueryTexts: []string{"database optimization"},
    K: 20,
}

// Apply exponential transformation to boost top results
boostedRank := v2.ExpRank(semanticRank)

results, err := collection.Search(ctx,
    v2.WithSearchRank(boostedRank),
    v2.WithSearchWhere(v2.EqString("category", "performance")),
    v2.WithSearchLimit(5, 0),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
```

### Example 5: Paginated Search

```go
const pageSize = 10
const page = 2 // zero-indexed

results, err := collection.Search(ctx,
    v2.WithSearchRankKnnTexts([]string{"data science"}, 50),
    v2.WithSearchLimit(pageSize, page*pageSize),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
```

## Result Structure

Search results are returned as a `SearchResult` interface with group-based results.

```go
type SearchResult interface {
    GetIDGroups() []DocumentIDs
    GetDocumentsGroups() []Documents
    GetMetadatasGroups() []DocumentMetadatas
    GetEmbeddingsGroups() []embeddings.Embeddings
    GetScoresGroups() []embeddings.Distances
    CountGroups() int
}
```

### Processing Results

```go
results, err := collection.Search(ctx, /* options */)
if err != nil {
    log.Fatal(err)
}

// Get the first (and typically only) result group
ids := results.GetIDGroups()[0]
docs := results.GetDocumentsGroups()[0]
scores := results.GetScoresGroups()[0]
metadatas := results.GetMetadatasGroups()[0]

for i, id := range ids {
    fmt.Printf("Result %d:\n", i+1)
    fmt.Printf("  ID: %s\n", id)
    fmt.Printf("  Score: %.4f\n", scores[i])
    fmt.Printf("  Document: %s\n", docs[i].ContentString())

    if metadatas != nil && i < len(metadatas) {
        if author, ok := metadatas[i].GetString("author"); ok {
            fmt.Printf("  Author: %s\n", author)
        }
    }
    fmt.Println()
}
```

## Real-World Use Cases

### Use Case 1: E-commerce Product Search

Combine semantic understanding with keyword matching and business rules.

```go
semanticRank := &v2.KnnRank{
    QueryTexts: []string{"wireless headphones bluetooth audio"},
    K: 30,
}

keywordRank := &v2.KnnRank{
    QueryTexts: []string{"wireless headphones"},
    K: 30,
}

results, err := collection.Search(ctx,
    v2.WithSearchRankRrf(
        []v2.RankExpression{semanticRank, keywordRank},
        60,
        true,
    ),
    v2.WithSearchWhere(v2.And(
        v2.EqBool("in_stock", true),
        v2.GteFloat("rating", 4.0),
        v2.LteFloat("price", 200.0),
    )),
    v2.WithSearchLimit(20, 0),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
```

### Use Case 2: Documentation Search

Search across titles and content with recency filtering.

```go
titleSearch := &v2.KnnRank{
    QueryTexts: []string{"authentication security"},
    K: 15,
}

contentSearch := &v2.KnnRank{
    QueryTexts: []string{"OAuth JWT tokens"},
    K: 15,
}

results, err := collection.Search(ctx,
    v2.WithSearchRankRrf(
        []v2.RankExpression{titleSearch, contentSearch},
        60,
        true,
    ),
    v2.WithSearchWhere(v2.And(
        v2.EqString("status", "published"),
        v2.GteInt("year", 2023),
    )),
    v2.WithSearchLimit(20, 0),
    v2.WithSearchSelect(
        v2.SelectID,
        v2.SelectDocument,
        v2.SelectScore,
        v2.SelectKey("title"),
        v2.SelectKey("author"),
    ),
)
```

### Use Case 3: Multi-Language Search

Search across multiple language-specific embeddings.

```go
enRank := &v2.KnnRank{
    QueryTexts: []string{"climate change"},
    K: 10,
}

esRank := &v2.KnnRank{
    QueryTexts: []string{"cambio climático"},
    K: 10,
}

results, err := collection.Search(ctx,
    v2.WithSearchRankRrf(
        []v2.RankExpression{enRank, esRank},
        60,
        true,
    ),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
```

## Migration from Query API

If you're using the Query API, here's how to migrate:

### Before (Query API)

```go
results, err := collection.Query(ctx,
    v2.WithQueryTexts("machine learning"),
    v2.WithNResults(10),
    v2.WithWhereQuery(v2.EqString("category", "tech")),
    v2.WithIncludeQuery(v2.IncludeDocuments, v2.IncludeMetadatas),
)
```

### After (Search API)

```go
results, err := collection.Search(ctx,
    v2.WithSearchRankKnnTexts([]string{"machine learning"}, 10),
    v2.WithSearchWhere(v2.EqString("category", "tech")),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument),
)
```

**Key Differences:**

| Query API | Search API | Notes |
|-----------|------------|-------|
| `WithNResults(n)` | `K` parameter in rank | Specifies number of results |
| `WithIncludeQuery(...)` | `WithSearchSelect(...)` | Field selection |
| `QueryResult` | `SearchResult` | Different result types |
| Single strategy | Multiple strategies | RRF, arithmetic, functions |

## Best Practices

1. **Start Simple**: Begin with KNN searches before exploring advanced ranking
2. **Validate Inputs**: Check errors when building complex rank expressions
3. **Use Appropriate K Values**: For RRF, typical k values are 60-100
4. **Select Only Needed Fields**: Reduce response size with targeted selection
5. **Paginate Large Results**: Use `WithSearchLimit` for better performance
6. **Test Ranking Strategies**: Experiment to find what works for your use case
7. **Monitor Performance**: Complex rank expressions may be slower than simple KNN

## Error Handling

All rank expressions and search operations validate inputs:

```go
searchOp, err := v2.NewCollectionSearchOp(
    v2.WithSearchRankKnnTexts([]string{}, 5), // Empty query texts
)
if err != nil {
    log.Fatal(err) // Will fail here
}

err = searchOp.PrepareAndValidate()
if err != nil {
    log.Fatal(err) // Or will fail here
}
```

**Common validation errors:**

- Missing query embeddings or texts in KNN
- K value ≤ 0
- Less than 2 ranks in RRF
- Invalid operators in arithmetic expressions
- Negative limit or offset values

## Limitations

Current limitations (may be addressed in future versions):

1. **Single Search per Request**: API supports multiple searches in request format, but current implementation sends one at a time
2. **No Direct Scalar Multiplication**: Use repeated operations or RRF for weighting
3. **Embedding Required**: Query texts must be embedded using collection's embedding function
4. **Server Version**: Requires Chroma server 1.1.0+ with search endpoint support

## Examples

See the [examples/v2/search](https://github.com/amikos-tech/chroma-go/tree/main/examples/v2/search) directory for complete working examples:

- `simple_knn/` - Basic KNN search
- `rrf_multi_query/` - Reciprocal rank fusion
- `filtered_search/` - Search with metadata filters
- `hybrid_search/` - Advanced hybrid search strategies

## API Reference

### Search Options

| Option | Description |
|--------|-------------|
| `WithSearchRankKnn(embeddings, k)` | KNN with embeddings |
| `WithSearchRankKnnTexts(texts, k)` | KNN with query texts |
| `WithSearchRankRrf(ranks, k, normalize)` | Reciprocal rank fusion |
| `WithSearchRank(rank)` | Custom rank expression |
| `WithSearchWhere(filter)` | Metadata filtering |
| `WithSearchLimit(limit, offset)` | Pagination |
| `WithSearchSelect(keys...)` | Field selection |

### Rank Constructors

| Function | Description |
|----------|-------------|
| `AddRanks(left, right)` | Addition operation |
| `SubRanks(left, right)` | Subtraction operation |
| `MulRanks(left, right)` | Multiplication operation |
| `DivRanks(left, right)` | Division operation |
| `ExpRank(operand)` | Exponential function |
| `LogRank(operand)` | Natural logarithm |
| `AbsRank(operand)` | Absolute value |
| `MaxRank(operand)` | Maximum function |
| `MinRank(operand)` | Minimum function |

## See Also

- [Chroma Search API Overview](https://docs.trychroma.com/cloud/search-api/overview) - Official Chroma documentation
- [Chroma Search Guide](https://docs.trychroma.com/guides/search) - Official search guide
- [Query API (Traditional)](client.md) - Simple KNN querying
- [Filtering](filtering.md) - Metadata filtering syntax
- [Embeddings](embeddings.md) - Embedding functions
- [Examples](https://github.com/amikos-tech/chroma-go/tree/main/examples/v2/search) - Working code examples
