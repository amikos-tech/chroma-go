# Search API

The Search API provides advanced semantic search capabilities with flexible ranking, filtering, and pagination options. This is the recommended API for Chroma Cloud and for applications requiring complex search patterns.

!!! note "Cloud Feature"

    The Search API is available in Chroma Cloud. For self-hosted Chroma, use the Query API instead.

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
    "github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
)

func main() {
    client, err := chroma.NewHTTPClient(
        chroma.WithCloudAPIKey("your-api-key"),
        chroma.WithDatabaseAndTenant("your-tenant", "your-database"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Embedding function is required for text queries
    ef, _ := openai.NewOpenAIEmbeddingFunction("sk-xxx")

    col, err := client.GetCollection(context.Background(), "my-collection",
        chroma.WithEmbeddingFunctionGet(ef),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Simple text search
    result, err := col.Search(context.Background(),
        chroma.NewSearchRequest(
            chroma.WithKnnRank(chroma.KnnQueryText("machine learning")),
            chroma.WithLimit(10),
        ),
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found %d results\n", len(result.(*chroma.SearchResultImpl).IDs[0]))
}
```

## Search Components

A search request consists of four optional components:

| Component | Description | Function |
|-----------|-------------|----------|
| **Rank** | How to score and order results | `WithKnnRank`, `WithRrfRank` |
| **Filter** | Which documents to include | `WithFilter`, `WithIDs` |
| **Page** | Pagination (limit/offset) | `NewPage`, `WithLimit`, `WithOffset` |
| **Select** | Which fields to return | `WithSelect`, `WithSelectAll` |

---

## Ranking

### KNN (K-Nearest Neighbors)

KNN search finds documents with embeddings most similar to your query.

```go
// Text query (auto-embedded using collection's embedding function)
chroma.WithKnnRank(chroma.KnnQueryText("search query"))

// Dense vector query
vector := embeddings.NewEmbeddingFromFloat32([]float32{0.1, 0.2, 0.3, ...})
chroma.WithKnnRank(chroma.KnnQueryVector(vector))

// Sparse vector query
sparse, err := embeddings.NewSparseVector([]int{1, 5, 10}, []float32{0.5, 0.3, 0.8})
if err != nil { return err }
chroma.WithKnnRank(
    chroma.KnnQuerySparseVector(sparse),
    chroma.WithKnnKey(chroma.K("sparse_embedding")),
)
```

#### KNN Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithKnnLimit(n)` | Number of nearest neighbors to retrieve | 16 |
| `WithKnnKey(key)` | Which embedding field to search | `#embedding` |
| `WithKnnDefault(score)` | Score for documents not in top-K | excluded |
| `WithKnnReturnRank()` | Return rank position instead of distance | false |

```go
knn, err := chroma.NewKnnRank(
    chroma.KnnQueryText("AI research"),
    chroma.WithKnnLimit(100),
    chroma.WithKnnDefault(10.0),
    chroma.WithKnnKey(chroma.K("dense_embedding")),
)
if err != nil {
    log.Fatal(err)
}
```

### Weighted Combinations

Combine multiple searches with different weights:

```go
// Dense + Sparse hybrid search (70% dense, 30% sparse)
dense, _ := chroma.NewKnnRank(chroma.KnnQueryText("machine learning"))
sparse, _ := chroma.NewKnnRank(
    chroma.KnnQuerySparseVector(sparseVector),
    chroma.WithKnnKey(chroma.K("sparse_embedding")),
)

combined := dense.Multiply(chroma.FloatOperand(0.7)).Add(
    sparse.Multiply(chroma.FloatOperand(0.3)),
)

result, err := col.Search(ctx,
    chroma.NewSearchRequest(
        chroma.WithRank(combined),
        chroma.WithLimit(10),
    ),
)
```

### Mathematical Functions

Transform scores using mathematical operations:

| Method | Description | Example |
|--------|-------------|---------|
| `Add(op)` | Addition | `rank.Add(chroma.FloatOperand(1))` |
| `Sub(op)` | Subtraction | `rank.Sub(otherRank)` |
| `Multiply(op)` | Multiplication | `rank.Multiply(chroma.FloatOperand(0.5))` |
| `Div(op)` | Division | `rank.Div(chroma.FloatOperand(10))` |
| `Negate()` | Negation | `rank.Negate()` |
| `Abs()` | Absolute value | `rank.Abs()` |
| `Exp()` | Exponential (e^x) | `rank.Exp()` |
| `Log()` | Natural log | `rank.Log()` |
| `Max(op)` | Maximum | `rank.Max(chroma.FloatOperand(1.0))` |
| `Min(op)` | Minimum | `rank.Min(chroma.FloatOperand(0.0))` |

```go
// Exponential amplification
amplified, _ := chroma.NewKnnRank(chroma.KnnQueryText("query"))
amplified = amplified.Exp()

// Log compression (add 1 to avoid log(0))
compressed, _ := chroma.NewKnnRank(chroma.KnnQueryText("query"))
compressed = compressed.Add(chroma.FloatOperand(1)).Log()

// Score clamping to [0, 1]
clamped, _ := chroma.NewKnnRank(chroma.KnnQueryText("query"))
clamped = clamped.Min(chroma.FloatOperand(0.0)).Max(chroma.FloatOperand(1.0))
```

### Reciprocal Rank Fusion (RRF)

RRF combines multiple ranking strategies using: `-sum(weight_i / (k + rank_i))`

```go
knn1, _ := chroma.NewKnnRank(
    chroma.KnnQueryText("machine learning"),
    chroma.WithKnnReturnRank(), // Required for RRF
)
knn2, _ := chroma.NewKnnRank(
    chroma.KnnQueryText("deep learning"),
    chroma.WithKnnReturnRank(),
)

rrf, err := chroma.NewRrfRank(
    chroma.WithRrfRanks(
        knn1.WithWeight(0.6),
        knn2.WithWeight(0.4),
    ),
    chroma.WithRrfK(60),         // Smoothing constant (default: 60)
    chroma.WithRrfNormalize(),   // Normalize weights to sum to 1.0
)
if err != nil {
    log.Fatal(err)
}

result, err := col.Search(ctx,
    chroma.NewSearchRequest(
        chroma.WithRank(rrf),
        chroma.WithLimit(10),
    ),
)
```

---

## Filtering

### Metadata Filters

Filter documents by metadata attributes:

```go
// Single condition
chroma.WithFilter(chroma.EqString("status", "published"))

// Combined conditions
chroma.WithFilter(
    chroma.And(
        chroma.EqString("status", "published"),
        chroma.GtInt("views", 100),
    ),
)
```

#### Filter Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `EqString` | String equals | `EqString("type", "article")` |
| `EqInt` | Integer equals | `EqInt("count", 5)` |
| `EqFloat` | Float equals | `EqFloat("score", 0.95)` |
| `EqBool` | Boolean equals | `EqBool("active", true)` |
| `NeString` | Not equals | `NeString("status", "draft")` |
| `GtInt` | Greater than | `GtInt("views", 100)` |
| `GteInt` | Greater than or equal | `GteInt("priority", 1)` |
| `LtInt` | Less than | `LtInt("age", 30)` |
| `LteInt` | Less than or equal | `LteInt("rank", 10)` |
| `InStrings` | In list | `InStrings("category", []string{"tech", "science"})` |
| `NinStrings` | Not in list | `NinStrings("tag", []string{"spam", "test"})` |
| `And` | Logical AND | `And(filter1, filter2)` |
| `Or` | Logical OR | `Or(filter1, filter2)` |

### Document ID Filter

Restrict search to specific document IDs using `WithIDs`:

```go
chroma.WithIDs("doc1", "doc2", "doc3")
```

For more flexible ID filtering that can be combined with other where clauses, use `IDIn` and `IDNotIn`:

> **Note:** `IDIn` and `IDNotIn` are **Cloud-only** features. These functions follow the Python SDK pattern (`K.ID.is_in()`, `K.ID.not_in()`).

```go
// Include only specific IDs (equivalent to Python's K.ID.is_in())
chroma.WithFilter(chroma.IDIn("doc1", "doc2", "doc3"))

// Exclude specific IDs (equivalent to Python's K.ID.not_in())
chroma.WithFilter(chroma.IDNotIn("seen1", "seen2", "seen3"))

// Combine with other filters - useful for excluding already-seen content
chroma.WithFilter(chroma.And(
    chroma.EqString("category", "tech"),
    chroma.IDNotIn("already_read_1", "already_read_2"),
))
```

### Combining Filters

```go
result, err := col.Search(ctx,
    chroma.NewSearchRequest(
        chroma.WithKnnRank(chroma.KnnQueryText("AI research")),
        chroma.WithFilter(
            chroma.And(
                chroma.EqString("status", "published"),
                chroma.InStrings("category", []string{"tech", "science"}),
            ),
        ),
        chroma.WithIDs("doc1", "doc2", "doc3"),
        chroma.NewPage(chroma.Limit(20)),
    ),
)
```

---

## Pagination

Control result pagination with limit and offset. The recommended approach is using the `Page` type for fluent navigation.

### Using Page (Recommended)

```go
// Create a page with custom size - inline usage
result, err := col.Search(ctx,
    chroma.NewSearchRequest(
        chroma.WithKnnRank(chroma.KnnQueryText("query")),
        chroma.NewPage(chroma.Limit(20)),
    ),
)

// Or reuse for navigation
page := chroma.NewPage(chroma.Limit(20))
page = page.Next()  // Offset becomes 20
page = page.Prev()  // Back to offset 0
```

### Iteration Pattern

`Page` eliminates off-by-one errors in pagination loops:

```go
page := chroma.NewPage(chroma.Limit(20))
for {
    result, err := col.Search(ctx,
        chroma.NewSearchRequest(
            chroma.WithKnnRank(chroma.KnnQueryText("query")),
            page,
        ),
    )
    if err != nil {
        break
    }

    rows := result.(*chroma.SearchResultImpl).Rows()
    if len(rows) == 0 {
        break  // No more results
    }

    // Process rows...

    page = page.Next()  // Move to next page
}
```

### Page Methods

| Method | Description |
|--------|-------------|
| `NewPage(opts...)` | Create a page (default limit: 10) |
| `Limit(n)` | Set page size (must be > 0) |
| `Offset(n)` | Set starting offset (must be >= 0) |
| `Next()` | Return new page for next results |
| `Prev()` | Return new page for previous results (clamped to 0) |
| `Number()` | Current page number (0-indexed) |
| `Size()` | Page size (limit) |
| `GetOffset()` | Current offset |

### Using WithLimit and WithOffset

For simple cases, you can use `WithLimit` and `WithOffset` directly:

```go
// First page (10 results)
chroma.WithLimit(10)

// Third page (skip 20 results)
chroma.WithLimit(10)
chroma.WithOffset(20)
```

---

## Projection (Select)

Choose which fields to include in results:

```go
// Select specific fields
chroma.WithSelect(chroma.KDocument, chroma.KScore, chroma.K("title"))

// Select all standard fields
chroma.WithSelectAll()
```

#### Projection Keys

| Key | Description |
|-----|-------------|
| `KID` | Document ID |
| `KDocument` | Document text |
| `KEmbedding` | Vector embedding |
| `KMetadata` | All metadata fields |
| `KScore` | Ranking score |
| `K("field")` | Custom metadata field |

---

## Complete Examples

### Semantic Search with Filters

```go
result, err := col.Search(ctx,
    chroma.NewSearchRequest(
        chroma.WithKnnRank(
            chroma.KnnQueryText("latest AI research"),
            chroma.WithKnnLimit(100),
        ),
        chroma.WithFilter(
            chroma.And(
                chroma.EqString("status", "published"),
                chroma.GtInt("year", 2023),
            ),
        ),
        chroma.WithLimit(10),
        chroma.WithSelect(chroma.KDocument, chroma.KScore, chroma.K("title")),
    ),
)
```

### Hybrid Dense + Sparse Search

```go
dense, _ := chroma.NewKnnRank(
    chroma.KnnQueryText("neural networks"),
    chroma.WithKnnLimit(100),
    chroma.WithKnnDefault(1000.0),
)

sparse, _ := chroma.NewKnnRank(
    chroma.KnnQuerySparseVector(sparseVector),
    chroma.WithKnnKey(chroma.K("sparse_embedding")),
    chroma.WithKnnLimit(100),
    chroma.WithKnnDefault(1000.0),
)

// 70% dense, 30% sparse
hybrid := dense.Multiply(chroma.FloatOperand(0.7)).Add(
    sparse.Multiply(chroma.FloatOperand(0.3)),
)

result, err := col.Search(ctx,
    chroma.NewSearchRequest(
        chroma.WithRank(hybrid),
        chroma.WithLimit(10),
    ),
)
```

### Multi-Query with RRF

```go
semantic, _ := chroma.NewKnnRank(
    chroma.KnnQueryText("machine learning algorithms"),
    chroma.WithKnnReturnRank(),
    chroma.WithKnnLimit(50),
)

keyword, _ := chroma.NewKnnRank(
    chroma.KnnQuerySparseVector(bm25Vector),
    chroma.WithKnnKey(chroma.K("bm25_embedding")),
    chroma.WithKnnReturnRank(),
    chroma.WithKnnLimit(50),
)

rrf, _ := chroma.NewRrfRank(
    chroma.WithRrfRanks(
        semantic.WithWeight(0.6),
        keyword.WithWeight(0.4),
    ),
    chroma.WithRrfK(60),
)

result, err := col.Search(ctx,
    chroma.NewSearchRequest(
        chroma.WithRank(rrf),
        chroma.WithFilter(chroma.EqString("type", "paper")),
        chroma.WithLimit(10),
        chroma.WithSelect(chroma.KDocument, chroma.KScore, chroma.K("title"), chroma.K("authors")),
    ),
)
```

### No Ranking (Index Order)

Retrieve documents without ranking (useful for filtered retrieval):

```go
result, err := col.Search(ctx,
    chroma.NewSearchRequest(
        chroma.WithFilter(chroma.EqString("status", "active")),
        chroma.NewPage(chroma.Limit(100)),
    ),
)
```

---

## Read Level

Control whether searches read from the write-ahead log (WAL) or only the compacted index:

| Level | Description |
|-------|-------------|
| `ReadLevelIndexAndWAL` | Read from both index and WAL (default). All committed writes visible. |
| `ReadLevelIndexOnly` | Read only from compacted index. Faster, but recent writes may not be visible. |

```go
// Default behavior - reads from both index and WAL
result, err := col.Search(ctx,
    chroma.NewSearchRequest(
        chroma.WithKnnRank(chroma.KnnQueryText("query")),
    ),
    chroma.WithReadLevel(chroma.ReadLevelIndexAndWAL),
)

// Faster search - only reads from compacted index
result, err := col.Search(ctx,
    chroma.NewSearchRequest(
        chroma.WithKnnRank(chroma.KnnQueryText("query")),
    ),
    chroma.WithReadLevel(chroma.ReadLevelIndexOnly),
)
```

!!! note "Index-Only Reads"

    `ReadLevelIndexOnly` provides faster searches but documents added recently that haven't been compacted into the index may not appear in results.

---

## Error Handling

Both `NewKnnRank` and `NewRrfRank` return errors that should be checked:

```go
knn, err := chroma.NewKnnRank(
    chroma.KnnQueryText("query"),
    chroma.WithKnnLimit(100),
)
if err != nil {
    log.Fatalf("Failed to create KNN rank: %v", err)
}

rrf, err := chroma.NewRrfRank(
    chroma.WithRrfRanks(knn.WithWeight(1.0)),
)
if err != nil {
    log.Fatalf("Failed to create RRF rank: %v", err)
}
```

---

## API Reference

### Search Options

| Function | Description |
|----------|-------------|
| `NewSearchRequest(opts...)` | Create a search request with options |
| `WithKnnRank(query, opts...)` | Add KNN ranking to request |
| `WithRrfRank(opts...)` | Add RRF ranking to request |
| `WithFilter(where)` | Add metadata filter |
| `WithIDs(ids...)` | Filter by document IDs (unified option) |
| `NewPage(opts...)` | Create a fluent pagination object (recommended) |
| `Limit(n)` | Set page size (for NewPage) |
| `Offset(n)` | Set offset (for NewPage) |
| `WithLimit(n)` | Set result limit directly |
| `WithOffset(n)` | Set result offset directly |
| `WithSelect(keys...)` | Select fields to return |
| `WithSelectAll()` | Select all standard fields |
| `WithReadLevel(level)` | Set read level (`ReadLevelIndexAndWAL` or `ReadLevelIndexOnly`) |

### KNN Options

| Function | Description |
|----------|-------------|
| `KnnQueryText(text)` | Query with text (auto-embedded) |
| `KnnQueryVector(vec)` | Query with dense vector |
| `KnnQuerySparseVector(vec)` | Query with sparse vector |
| `WithKnnLimit(n)` | Set K neighbors to retrieve |
| `WithKnnKey(key)` | Set embedding field to search |
| `WithKnnDefault(score)` | Set default score for non-matches |
| `WithKnnReturnRank()` | Return rank instead of distance |

### RRF Options

| Function | Description |
|----------|-------------|
| `WithRrfRanks(ranks...)` | Add weighted ranks |
| `WithRrfK(k)` | Set smoothing constant |
| `WithRrfNormalize()` | Normalize weights |

### ID Filter Operators (Cloud-only)

| Function | Description |
|----------|-------------|
| `IDIn(ids...)` | Match documents with any of the specified IDs |
| `IDNotIn(ids...)` | Exclude documents with any of the specified IDs |
