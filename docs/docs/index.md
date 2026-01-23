# Chroma Go Client

A Go client library for ChromaDB vector database.

## Installation

Add the library to your project:

```bash
go get github.com/amikos-tech/chroma-go
```

## Getting Started

Import the V2 API:

```go
package main

import (
    chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)
```

Concepts:

- [Client Options](client.md) - How to configure the Chroma Go client
- [Embeddings](embeddings.md) - Available embedding functions
- [Filtering](filtering.md) - How to filter results
- [Search API](search.md) - Advanced search with ranking and pagination

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
    ctx := context.Background()

    // Create client (connects to localhost:8000 by default)
    client, err := chroma.NewHTTPClient()
    if err != nil {
        log.Fatalf("Error creating client: %s", err)
    }
    defer client.Close()

    // Create or get a collection
    col, err := client.GetOrCreateCollection(ctx, "my-collection")
    if err != nil {
        log.Fatalf("Error creating collection: %s", err)
    }

    // Add documents
    err = col.Add(ctx,
        chroma.WithIDs("doc1", "doc2", "doc3"),
        chroma.WithTexts(
            "Machine learning is a subset of AI",
            "Natural language processing enables computers to understand text",
            "Deep learning uses neural networks with many layers",
        ),
        chroma.WithMetadatas(
            map[string]any{"category": "ml", "year": 2024},
            map[string]any{"category": "nlp", "year": 2024},
            map[string]any{"category": "dl", "year": 2023},
        ),
    )
    if err != nil {
        log.Fatalf("Error adding documents: %s", err)
    }

    // Query for similar documents
    results, err := col.Query(ctx,
        chroma.WithQueryTexts("What is artificial intelligence?"),
        chroma.WithNResults(2),
    )
    if err != nil {
        log.Fatalf("Error querying: %s", err)
    }

    fmt.Printf("Found %d results\n", len(results.GetIDsGroups()[0]))
    for i, doc := range results.GetDocumentsGroups()[0] {
        fmt.Printf("  %d: %s\n", i+1, doc)
    }
}
```

## Unified Options API

The V2 API uses a unified options pattern where common options work across multiple operations:

| Option | Get | Query | Delete | Add | Update | Search |
|--------|:---:|:-----:|:------:|:---:|:------:|:------:|
| `WithIDs` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `WithWhere` | ✓ | ✓ | ✓ | | | |
| `WithWhereDocument` | ✓ | ✓ | ✓ | | | |
| `WithInclude` | ✓ | ✓ | | | | |
| `WithLimit` | ✓ | | | | | ✓ |
| `WithOffset` | ✓ | | | | | ✓ |
| `NewPage` | ✓ | | | | | ✓ |
| `WithNResults` | | ✓ | | | | |
| `WithQueryTexts` | | ✓ | | | | |
| `WithTexts` | | | | ✓ | ✓ | |
| `WithEmbeddings` | | | | ✓ | ✓ | |
| `WithMetadatas` | | | | ✓ | ✓ | |
| `WithIDGenerator` | | | | ✓ | | |

## CRUD Operations

### Add Documents

```go
// Add with explicit IDs
err := col.Add(ctx,
    chroma.WithIDs("id1", "id2"),
    chroma.WithTexts("First document", "Second document"),
    chroma.WithMetadatas(
        map[string]any{"author": "Alice"},
        map[string]any{"author": "Bob"},
    ),
)

// Add with auto-generated IDs
err := col.Add(ctx,
    chroma.WithTexts("Document without explicit ID"),
    chroma.WithIDGenerator(chroma.NewULIDGenerator()),
)

// Add with pre-computed embeddings
err := col.Add(ctx,
    chroma.WithIDs("id1"),
    chroma.WithEmbeddings([]float32{0.1, 0.2, 0.3, ...}),
    chroma.WithMetadatas(map[string]any{"source": "external"}),
)
```

### Get Documents

```go
// Get by IDs
results, err := col.Get(ctx, chroma.WithIDs("id1", "id2"))

// Get with metadata filter
results, err := col.Get(ctx,
    chroma.WithWhere(chroma.EqString("author", "Alice")),
    chroma.WithInclude(chroma.IncludeDocuments, chroma.IncludeMetadatas),
)

// Get with pagination
results, err := col.Get(ctx,
    chroma.WithLimit(10),
    chroma.WithOffset(20),
)

// Get with document content filter
results, err := col.Get(ctx,
    chroma.WithWhereDocument(chroma.Contains("machine learning")),
)
```

### Query (Semantic Search)

```go
// Basic query
results, err := col.Query(ctx,
    chroma.WithQueryTexts("machine learning algorithms"),
    chroma.WithNResults(5),
)

// Query with metadata filter
results, err := col.Query(ctx,
    chroma.WithQueryTexts("neural networks"),
    chroma.WithWhere(chroma.AndFilter(
        chroma.EqString("category", "dl"),
        chroma.GtInt("year", 2022),
    )),
    chroma.WithNResults(10),
)

// Query with multiple queries
results, err := col.Query(ctx,
    chroma.WithQueryTexts("AI", "robotics", "automation"),
    chroma.WithNResults(3),
)
// results.GetIDsGroups()[0] - results for "AI"
// results.GetIDsGroups()[1] - results for "robotics"
// results.GetIDsGroups()[2] - results for "automation"
```

### Update Documents

```go
// Update document content
err := col.Update(ctx,
    chroma.WithIDs("id1"),
    chroma.WithTexts("Updated document content"),
)

// Update metadata
err := col.Update(ctx,
    chroma.WithIDs("id1", "id2"),
    chroma.WithMetadatas(
        map[string]any{"status": "reviewed"},
        map[string]any{"status": "reviewed"},
    ),
)
```

### Upsert Documents

```go
// Insert or update documents
err := col.Upsert(ctx,
    chroma.WithIDs("id1", "id2"),
    chroma.WithTexts("New or updated doc 1", "New or updated doc 2"),
    chroma.WithMetadatas(meta1, meta2),
)
```

### Delete Documents

```go
// Delete by IDs
err := col.Delete(ctx, chroma.WithIDs("id1", "id2"))

// Delete by metadata filter
err := col.Delete(ctx,
    chroma.WithWhere(chroma.EqString("status", "archived")),
)

// Delete by document content
err := col.Delete(ctx,
    chroma.WithWhereDocument(chroma.Contains("DEPRECATED")),
)
```

## Metadata Filters

The library provides type-safe filter functions:

```go
// Equality
chroma.EqString("field", "value")
chroma.EqInt("count", 10)
chroma.EqFloat("score", 0.95)
chroma.EqBool("active", true)

// Not equal
chroma.NeString("status", "deleted")

// Comparison (numeric and string)
chroma.GtInt("year", 2020)      // greater than
chroma.GteInt("year", 2020)     // greater than or equal
chroma.LtFloat("score", 0.5)    // less than
chroma.LteFloat("score", 0.5)   // less than or equal

// Set operations
chroma.InString("category", "ml", "ai", "dl")
chroma.NinInt("priority", 1, 2)

// Logical operators
chroma.AndFilter(filter1, filter2, ...)
chroma.OrFilter(filter1, filter2, ...)
```

## Document Content Filters

```go
// Contains substring
chroma.Contains("machine learning")

// Does not contain
chroma.NotContains("deprecated")

// Combine filters
chroma.AndDocumentFilter(
    chroma.Contains("neural"),
    chroma.NotContains("outdated"),
)
```

## Search API

For advanced use cases, the Search API provides more control:

```go
results, err := col.Search(ctx,
    chroma.NewSearchRequest(
        chroma.WithKnnRank(chroma.KnnQueryText("machine learning")),
        chroma.WithFilter(chroma.EqString(chroma.K("status"), "published")),
        chroma.NewPage(chroma.Limit(10)),
        chroma.WithSelect(chroma.KDocument, chroma.KScore, chroma.K("author")),
    ),
)

// Access results
for _, row := range results.(*chroma.SearchResultImpl).Rows() {
    fmt.Printf("ID: %s, Score: %f, Doc: %s\n", row.ID, row.Score, row.Document)
}
```

See [Search API documentation](search.md) for more details.

## Embedding Functions

The client supports multiple embedding providers. See [Embeddings](embeddings.md) for the full list.

```go
import "github.com/amikos-tech/chroma-go/pkg/embeddings/openai"

// Create embedding function
ef, err := openai.NewOpenAIEmbeddingFunction(os.Getenv("OPENAI_API_KEY"))

// Use with collection
col, err := client.GetOrCreateCollection(ctx, "my-collection",
    chroma.WithEmbeddingFunction(ef),
)
```

## V1 API (Deprecated)

!!! warning "V1 API Removed"

    The V1 API has been removed in version `v0.3.0`. If you need V1 compatibility:
    ```bash
    go get github.com/amikos-tech/chroma-go@v0.2.5
    ```
