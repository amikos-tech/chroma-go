# Chroma Go Search API

This document describes the new Search API for Chroma Go, which provides advanced search capabilities with flexible ranking expressions.

## Overview

The Search API extends Chroma's query capabilities by introducing a flexible ranking system that supports:
- K-Nearest Neighbors (KNN) search
- Reciprocal Rank Fusion (RRF) for combining multiple searches
- Arithmetic operations on rank expressions
- Mathematical functions on rank expressions
- Filtering and field selection

## Key Concepts

### Rank Expressions

Rank expressions define how search results are scored and ordered. All rank expressions implement the `RankExpression` interface:

```go
type RankExpression interface {
    ToJSON() interface{}
    Validate() error
}
```

### Available Rank Types

#### 1. KNN (K-Nearest Neighbors)

Traditional vector similarity search using embeddings:

```go
// With query texts (will be embedded automatically)
v2.WithSearchRankKnnTexts([]string{"artificial intelligence"}, 5)

// With pre-computed embeddings
v2.WithSearchRankKnn([]embeddings.Embedding{queryEmb}, 5)
```

#### 2. RRF (Reciprocal Rank Fusion)

Combines multiple ranking expressions using reciprocal rank fusion:

```go
rank1 := &v2.KnnRank{QueryTexts: []string{"AI"}, K: 10}
rank2 := &v2.KnnRank{QueryTexts: []string{"machine learning"}, K: 10}
v2.WithSearchRankRrf([]v2.RankExpression{rank1, rank2}, 60, true)
```

The RRF formula is: `score = sum(1 / (k + rank_i))` where `k` is a constant (typically 60).

#### 3. Arithmetic Operations

Combine rank expressions using arithmetic:

```go
// Addition
v2.AddRanks(rank1, rank2)

// Subtraction
v2.SubRanks(rank1, rank2)

// Multiplication
v2.MulRanks(rank1, rank2)

// Division
v2.DivRanks(rank1, rank2)
```

#### 4. Mathematical Functions

Apply mathematical functions to rank expressions:

```go
// Exponential
v2.ExpRank(rank)

// Natural logarithm
v2.LogRank(rank)

// Absolute value
v2.AbsRank(rank)

// Maximum
v2.MaxRank(rank)

// Minimum
v2.MinRank(rank)
```

### Field Selection

Select which fields to include in results:

```go
v2.WithSearchSelect(
    v2.SelectID,        // Document ID
    v2.SelectDocument,  // Document content
    v2.SelectEmbedding, // Embedding vector
    v2.SelectScore,     // Search score
    v2.SelectURI,       // Document URI
)
```

You can also select custom metadata fields by using `SelectKey("field_name")`.

### Filtering

Use the same where filters as Query:

```go
v2.WithSearchWhere(v2.EqString("category", "tech"))
v2.WithSearchWhere(v2.And(
    v2.GtInt("score", 10),
    v2.EqString("status", "active"),
))
```

### Pagination

Control result limits and offsets:

```go
v2.WithSearchLimit(10, 0)  // Get first 10 results
v2.WithSearchLimit(10, 10) // Get results 11-20
```

## Complete Examples

### Example 1: Simple KNN Search

```go
results, err := collection.Search(ctx,
    v2.WithSearchRankKnnTexts([]string{"machine learning"}, 5),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
```

### Example 2: Filtered Search

```go
results, err := collection.Search(ctx,
    v2.WithSearchRankKnnTexts([]string{"programming"}, 10),
    v2.WithSearchWhere(v2.EqString("language", "python")),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
```

### Example 3: Multi-Query RRF

```go
rank1 := &v2.KnnRank{QueryTexts: []string{"neural networks"}, K: 10}
rank2 := &v2.KnnRank{QueryTexts: []string{"deep learning"}, K: 10}

results, err := collection.Search(ctx,
    v2.WithSearchRankRrf([]v2.RankExpression{rank1, rank2}, 60, true),
    v2.WithSearchLimit(5, 0),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
```

### Example 4: Complex Rank Expression

Combine multiple operations:

```go
knn1 := &v2.KnnRank{QueryTexts: []string{"AI"}, K: 10}
knn2 := &v2.KnnRank{QueryTexts: []string{"automation"}, K: 10}

// Create a weighted combination: (knn1 * 0.7 + knn2 * 0.3)
// Note: Direct scalar multiplication not yet supported, use multiple additions
weighted := v2.AddRanks(knn1, knn2)

results, err := collection.Search(ctx,
    v2.WithSearchRank(weighted),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
```

### Example 5: Paginated Results

```go
// Get first page
page1, err := collection.Search(ctx,
    v2.WithSearchRankKnnTexts([]string{"data science"}, 20),
    v2.WithSearchLimit(10, 0),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)

// Get second page
page2, err := collection.Search(ctx,
    v2.WithSearchRankKnnTexts([]string{"data science"}, 20),
    v2.WithSearchLimit(10, 10),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
```

## Result Structure

Search results are returned as `SearchResult` interface:

```go
type SearchResult interface {
    GetIDGroups() []DocumentIDs
    GetDocumentsGroups() []Documents
    GetMetadatasGroups() []DocumentMetadatas
    GetEmbeddingsGroups() []embeddings.Embeddings
    GetScoresGroups() []embeddings.Distances
    ToRecordsGroups() []Records
    CountGroups() int
}
```

Results are grouped because the API supports multiple simultaneous searches (though the current implementation sends one search at a time).

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

for i, id := range ids {
    fmt.Printf("ID: %s, Score: %.4f\n", id, scores[i])
    fmt.Printf("Document: %s\n", docs[i].ContentString())
}
```

## API Comparison: Search vs Query

### When to Use Search

Use the `Search` API when you need:
- Multiple ranking strategies combined with RRF
- Arithmetic operations on rankings
- Advanced score manipulation
- Custom ranking functions

### When to Use Query

Use the traditional `Query` API when you need:
- Simple KNN search
- Backward compatibility
- Simpler API surface

### Example Comparison

**Query API:**
```go
results, err := collection.Query(ctx,
    v2.WithQueryTexts("machine learning"),
    v2.WithNResults(10),
    v2.WithWhereQuery(v2.EqString("category", "tech")),
)
```

**Search API (equivalent):**
```go
results, err := collection.Search(ctx,
    v2.WithSearchRankKnnTexts([]string{"machine learning"}, 10),
    v2.WithSearchWhere(v2.EqString("category", "tech")),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
```

## Advanced Use Cases

### Hybrid Search with Custom Weighting

Combine semantic and keyword search (conceptual example):

```go
semanticRank := &v2.KnnRank{QueryTexts: []string{"semantic query"}, K: 10}
keywordRank := &v2.KnnRank{QueryTexts: []string{"keyword query"}, K: 10}

// In future: support scalar weights directly
// For now, use RRF for combination
results, err := collection.Search(ctx,
    v2.WithSearchRankRrf([]v2.RankExpression{semanticRank, keywordRank}, 60, true),
)
```

### Multi-Vector Search

Search across different embedding spaces:

```go
titleRank := &v2.KnnRank{
    QueryEmbeddings: []embeddings.Embedding{titleEmbedding},
    K: 10,
}
contentRank := &v2.KnnRank{
    QueryEmbeddings: []embeddings.Embedding{contentEmbedding},
    K: 10,
}

results, err := collection.Search(ctx,
    v2.WithSearchRankRrf([]v2.RankExpression{titleRank, contentRank}, 60, true),
)
```

### Re-ranking with Score Transformation

Apply transformations to scores:

```go
baseRank := &v2.KnnRank{QueryTexts: []string{"query"}, K: 20}
// Apply exponential transformation to emphasize top results
transformedRank := v2.ExpRank(baseRank)

results, err := collection.Search(ctx,
    v2.WithSearchRank(transformedRank),
    v2.WithSearchLimit(10, 0),
)
```

## Best Practices

1. **Start Simple**: Begin with basic KNN searches before exploring advanced ranking

2. **Validate Rank Expressions**: Always check errors when building complex rank expressions

3. **Use Appropriate K Values**: For RRF, typical k values are 60-100

4. **Select Only Needed Fields**: Use `WithSearchSelect` to reduce response size

5. **Paginate Large Results**: Use `WithSearchLimit` for better performance

6. **Test Ranking Strategies**: Experiment with different combinations to find what works best

7. **Monitor Performance**: Complex rank expressions may be slower than simple KNN

## Error Handling

All rank expressions and search operations validate inputs:

```go
// Validation happens during PrepareAndValidate
searchOp, err := v2.NewCollectionSearchOp(
    v2.WithSearchRankKnnTexts([]string{}, 5), // Empty query texts
)
if err != nil {
    // Handle construction error
}

err = searchOp.PrepareAndValidate()
if err != nil {
    // Handle validation error
}
```

Common validation errors:
- Missing query embeddings or texts in KNN
- K value ≤ 0
- Less than 2 ranks in RRF
- Invalid operators in arithmetic expressions
- Negative limit or offset values

## Migration Guide

### From Query to Search

If you're currently using `Query`, here's how to migrate:

**Before:**
```go
results, err := collection.Query(ctx,
    v2.WithQueryTexts("my query"),
    v2.WithNResults(10),
    v2.WithWhereQuery(filter),
    v2.WithIncludeQuery(v2.IncludeDocuments, v2.IncludeMetadatas),
)
```

**After:**
```go
results, err := collection.Search(ctx,
    v2.WithSearchRankKnnTexts([]string{"my query"}, 10),
    v2.WithSearchWhere(filter),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectMetadata),
)
```

Note: The main API difference is:
- `WithNResults(n)` → `K` parameter in KNN rank
- `WithIncludeQuery(...)` → `WithSearchSelect(...)`
- Result type changes from `QueryResult` to `SearchResult`

## Limitations

Current limitations (may be addressed in future versions):

1. **Single Search per Request**: The API supports multiple searches in the request format, but the current implementation sends one search at a time

2. **No Direct Scalar Multiplication**: Use repeated additions or RRF for weighting

3. **Embedding Required for Text Queries**: Query texts must be embedded using the collection's embedding function

4. **Server Support**: Requires Chroma server version that supports the search endpoint (typically 1.1.0+)

## See Also

- [Query API Documentation](./QUERY_API.md)
- [Filtering Documentation](./FILTERING.md)
- [Examples](../examples/v2/search_example.go)
- [Python Search API](https://github.com/chroma-core/chroma/blob/main/chromadb/execution/expression/plan.py)
