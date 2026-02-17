# Filtering

Chroma Go provides powerful filtering capabilities for Get, Query, Delete, and Search operations.

## Unified Filter Options

The V2 API uses unified options that work across multiple operations:

| Option | Get | Query | Delete | Search |
|--------|:---:|:-----:|:------:|:------:|
| `WithWhere` | ✓ | ✓ | ✓ | |
| `WithWhereDocument` | ✓ | ✓ | ✓ | |
| `WithIDs` | ✓ | ✓ | ✓ | ✓ |
| `WithFilter` | | | | ✓ |

## Metadata Filters

Filter documents based on metadata field values using type-safe filter functions.

### Equality Operators

```go
import chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"

// String equality
chroma.EqString("status", "active")
chroma.NeString("status", "deleted")

// Integer equality
chroma.EqInt("count", 10)
chroma.NeInt("priority", 0)

// Float equality
chroma.EqFloat("score", 0.95)
chroma.NeFloat("threshold", 0.5)

// Boolean equality
chroma.EqBool("published", true)
chroma.NeBool("draft", false)
```

### Comparison Operators

```go
// Greater than
chroma.GtInt("year", 2020)
chroma.GtFloat("score", 0.8)
chroma.GtString("name", "A")  // lexicographic

// Greater than or equal
chroma.GteInt("version", 2)
chroma.GteFloat("rating", 4.0)

// Less than
chroma.LtInt("age", 30)
chroma.LtFloat("price", 100.0)

// Less than or equal
chroma.LteInt("priority", 5)
chroma.LteFloat("discount", 0.25)
```

### Set Operators

```go
// Value in set
chroma.InString("category", "ml", "ai", "dl")
chroma.InInt("priority", 1, 2, 3)
chroma.InFloat("score", 0.9, 0.95, 1.0)

// Value not in set
chroma.NinString("status", "deleted", "archived")
chroma.NinInt("level", 0, 1)
```

### Array Contains Operators

Filter documents by checking if an array metadata field contains a specific value. Requires Chroma >= 1.5.0.

```go
// String array contains
chroma.MetadataContainsString(chroma.K("tags"), "science")
chroma.MetadataNotContainsString(chroma.K("tags"), "deprecated")

// Integer array contains
chroma.MetadataContainsInt(chroma.K("scores"), 100)
chroma.MetadataNotContainsInt(chroma.K("scores"), 0)

// Float array contains
chroma.MetadataContainsFloat(chroma.K("ratios"), 1.5)
chroma.MetadataNotContainsFloat(chroma.K("ratios"), 0.0)

// Boolean array contains
chroma.MetadataContainsBool(chroma.K("flags"), true)
chroma.MetadataNotContainsBool(chroma.K("flags"), false)
```

### Logical Operators

```go
// AND - all conditions must match
chroma.AndFilter(
    chroma.EqString("category", "tech"),
    chroma.GtInt("year", 2020),
    chroma.EqBool("published", true),
)

// OR - any condition can match
chroma.OrFilter(
    chroma.EqString("author", "Alice"),
    chroma.EqString("author", "Bob"),
)

// Nested logic
chroma.AndFilter(
    chroma.EqString("status", "published"),
    chroma.OrFilter(
        chroma.EqString("category", "tech"),
        chroma.EqString("category", "science"),
    ),
)
```

## Document Content Filters

Filter documents based on their text content.

```go
// Contains substring
chroma.Contains("machine learning")

// Does not contain substring
chroma.NotContains("deprecated")

// Combine with AND
chroma.AndDocumentFilter(
    chroma.Contains("neural network"),
    chroma.NotContains("outdated"),
)

// Combine with OR
chroma.OrDocumentFilter(
    chroma.Contains("Python"),
    chroma.Contains("Go"),
)
```

## Usage Examples

### Get with Filters

```go
// Get by metadata
results, err := col.Get(ctx,
    chroma.WithWhere(chroma.EqString("author", "Alice")),
    chroma.WithInclude(chroma.IncludeDocuments, chroma.IncludeMetadatas),
)

// Get by document content
results, err := col.Get(ctx,
    chroma.WithWhereDocument(chroma.Contains("machine learning")),
)

// Combine metadata and document filters
results, err := col.Get(ctx,
    chroma.WithWhere(chroma.GtInt("year", 2022)),
    chroma.WithWhereDocument(chroma.NotContains("draft")),
)
```

### Query with Filters

```go
// Semantic search with metadata filter
results, err := col.Query(ctx,
    chroma.WithQueryTexts("neural networks"),
    chroma.WithWhere(chroma.AndFilter(
        chroma.EqString("category", "deep-learning"),
        chroma.EqBool("peer_reviewed", true),
    )),
    chroma.WithNResults(10),
)

// Query with document content filter
results, err := col.Query(ctx,
    chroma.WithQueryTexts("transformers"),
    chroma.WithWhereDocument(chroma.Contains("attention mechanism")),
    chroma.WithNResults(5),
)

// Limit search to specific IDs
results, err := col.Query(ctx,
    chroma.WithQueryTexts("optimization"),
    chroma.WithIDs("paper1", "paper2", "paper3"),
    chroma.WithNResults(2),
)
```

### Delete with Filters

```go
// Delete by IDs
err := col.Delete(ctx, chroma.WithIDs("id1", "id2"))

// Delete by metadata
err := col.Delete(ctx,
    chroma.WithWhere(chroma.EqString("status", "archived")),
)

// Delete by document content
err := col.Delete(ctx,
    chroma.WithWhereDocument(chroma.Contains("DEPRECATED")),
)

// Delete with combined filters
err := col.Delete(ctx,
    chroma.WithWhere(chroma.LtInt("year", 2020)),
    chroma.WithWhereDocument(chroma.NotContains("classic")),
)
```

### Query with Array Contains Filter

```go
// Find documents where the "tags" array contains "science"
results, err := col.Query(ctx,
    chroma.WithQueryTexts("research papers"),
    chroma.WithWhere(chroma.MetadataContainsString(chroma.K("tags"), "science")),
    chroma.WithNResults(10),
)

// Combine array contains with other filters
results, err := col.Query(ctx,
    chroma.WithQueryTexts("research"),
    chroma.WithWhere(chroma.AndFilter(
        chroma.MetadataContainsString(chroma.K("tags"), "science"),
        chroma.GtInt(chroma.K("year"), 2022),
    )),
    chroma.WithNResults(10),
)
```

### Search API Filters

The Search API uses a slightly different filter syntax with the `K()` function:

```go
// Basic filter
results, err := col.Search(ctx,
    chroma.NewSearchRequest(
        chroma.WithKnnRank(chroma.KnnQueryText("query")),
        chroma.WithFilter(chroma.EqString(chroma.K("status"), "published")),
    ),
)

// Complex filter
results, err := col.Search(ctx,
    chroma.NewSearchRequest(
        chroma.WithKnnRank(chroma.KnnQueryText("machine learning")),
        chroma.WithFilter(chroma.And(
            chroma.EqString(chroma.K("category"), "research"),
            chroma.GtInt(chroma.K("citations"), 100),
            chroma.Or(
                chroma.EqString(chroma.K("venue"), "NeurIPS"),
                chroma.EqString(chroma.K("venue"), "ICML"),
            ),
        )),
        chroma.NewPage(chroma.Limit(20)),
    ),
)

// Filter by IDs
results, err := col.Search(ctx,
    chroma.NewSearchRequest(
        chroma.WithKnnRank(chroma.KnnQueryText("query")),
        chroma.WithIDs("doc1", "doc2", "doc3"),
    ),
)
```

## Deprecated Functions

The following operation-specific functions are deprecated. Use the unified options instead:

| Deprecated | Replacement |
|------------|-------------|
| `WithIDsGet` | `WithIDs` |
| `WithIDsQuery` | `WithIDs` |
| `WithIDsUpdate` | `WithIDs` |
| `WithIDsDelete` | `WithIDs` |
| `WithWhereGet` | `WithWhere` |
| `WithWhereQuery` | `WithWhere` |
| `WithWhereDelete` | `WithWhere` |
| `WithWhereDocumentGet` | `WithWhereDocument` |
| `WithWhereDocumentQuery` | `WithWhereDocument` |
| `WithWhereDocumentDelete` | `WithWhereDocument` |
| `WithIncludeGet` | `WithInclude` |
| `WithIncludeQuery` | `WithInclude` |
| `WithLimitGet` | `WithLimit` |
| `WithOffsetGet` | `WithOffset` |
| `WithTextsUpdate` | `WithTexts` |
| `WithMetadatasUpdate` | `WithMetadatas` |
| `WithEmbeddingsUpdate` | `WithEmbeddings` |
| `WithFilterIDs` | `WithIDs` |
| `WithPage` | `NewPage` |
| `PageLimit` | `Limit` |
| `PageOffset` | `Offset` |
