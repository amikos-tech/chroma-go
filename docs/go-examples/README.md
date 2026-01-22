# Chroma Go Examples

Go code examples for the [Chroma](https://trychroma.com/) vector database, using the [chroma-go](https://github.com/amikos-tech/chroma-go) client library.

Each document provides side-by-side Python and Go examples, referencing the [official Chroma documentation](https://docs.trychroma.com/).

## Quick Start

```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Connect to Chroma server
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create or get a collection
	collection, _ := client.GetOrCreateCollection(ctx, "my_collection")

	// Add documents
	collection.Add(ctx,
		v2.WithIDs("doc1", "doc2"),
		v2.WithDocuments("Hello world", "Goodbye world"),
	)

	// Query
	results, _ := collection.Query(ctx,
		v2.WithQueryTexts("Hello"),
		v2.WithNResults(1),
	)

	// Iterate using Rows() for ergonomic access
	for _, row := range results.Rows() {
		log.Printf("Found: %s", row.ID)
	}
}
```

## Documentation Index

### Getting Started

| Topic | Description |
|-------|-------------|
| [Getting Started](docs/overview/getting-started.md) | Quick introduction to chroma-go |
| [Client-Server Mode](docs/run-chroma/client-server.md) | Connect to a Chroma server via HTTP |
| [Cloud Client](docs/run-chroma/cloud-client.md) | Connect to Chroma Cloud |
| [Persistent Client](docs/run-chroma/persistent-client.md) | Working with persistent data |
| [Ephemeral Client](docs/run-chroma/ephemeral-client.md) | Testing and experimentation |
| [Running a Server](docs/cli/run.md) | Start and connect to Chroma |

### Collections

| Topic | Description |
|-------|-------------|
| [Manage Collections](docs/collections/manage-collections.md) | Create, get, list, delete collections |
| [Add Data](docs/collections/add-data.md) | Add documents and embeddings |
| [Update Data](docs/collections/update-data.md) | Update and upsert documents |
| [Delete Data](docs/collections/delete-data.md) | Delete documents by ID or filter |
| [Configure Collections](docs/collections/configure.md) | HNSW parameters and distance metrics |

### Querying

| Topic | Description |
|-------|-------------|
| [Query and Get](docs/querying-collections/query-and-get.md) | Query by text/embedding, get by ID |
| [Metadata Filtering](docs/querying-collections/metadata-filtering.md) | Filter with $eq, $gt, $in, $and, $or |
| [Full-Text Search](docs/querying-collections/full-text-search.md) | Search document content with $contains |

### Embedding Functions

| Topic | Description |
|-------|-------------|
| [Embedding Functions](docs/embeddings/embedding-functions.md) | OpenAI, Cohere, HuggingFace, and more |
| [Multimodal Embeddings](docs/embeddings/multimodal.md) | Image embedding support |

### Cloud Search API

| Topic | Description |
|-------|-------------|
| [Search Basics](cloud/search-api/search-basics.md) | Introduction to the Search API |
| [Filtering](cloud/search-api/filtering.md) | Metadata filtering in search |
| [Ranking](cloud/search-api/ranking.md) | KNN ranking and arithmetic operations |
| [Hybrid Search](cloud/search-api/hybrid-search.md) | RRF with dense and sparse vectors |
| [Pagination & Selection](cloud/search-api/pagination-selection.md) | Limit, offset, and field selection |
| [Group By](cloud/search-api/group-by.md) | Group results by metadata |
| [Batch Operations](cloud/search-api/batch-operations.md) | Multiple searches in one request |
| [Examples](cloud/search-api/examples.md) | E-commerce, recommendations, multi-category |

### Schema & Advanced Features

| Topic | Description |
|-------|-------------|
| [Schema Basics](cloud/schema/schema-basics.md) | Configure collection indexes |
| [Sparse Vector Search](cloud/schema/sparse-vector-search.md) | SPLADE and hybrid search setup |
| [Collection Forking](cloud/features/collection-forking.md) | Copy-on-write collection cloning |

### Reference

| Topic | Description |
|-------|-------------|
| [Migration](docs/overview/migration.md) | Version migration guide |
| [Telemetry](docs/overview/telemetry.md) | Anonymous usage tracking |
| [Troubleshooting](docs/overview/troubleshooting.md) | Common issues and solutions |

## API Quick Reference

### Client Creation

```go
// HTTP client (default: localhost:8000)
client, _ := v2.NewHTTPClient()

// HTTP client with options
client, _ := v2.NewHTTPClient(
    v2.WithBaseURL("http://localhost:8000"),
    v2.WithAuth(v2.NewTokenAuthCredentialsProvider("token", v2.AuthorizationTokenHeader)),
)

// Cloud client
client, _ := v2.NewCloudClient(
    v2.WithCloudAPIKey("api-key"),
    v2.WithDatabaseAndTenant("database", "tenant"),
)
```

### Collection Operations

```go
// Create/Get
col, _ := client.CreateCollection(ctx, "name")
col, _ := client.GetCollection(ctx, "name")
col, _ := client.GetOrCreateCollection(ctx, "name")

// List/Delete
cols, _ := client.ListCollections(ctx)
client.DeleteCollection(ctx, "name")
```

### Data Operations

```go
// Add
col.Add(ctx,
    v2.WithIDs("id1", "id2"),
    v2.WithDocuments("doc1", "doc2"),
    v2.WithMetadatas(map[string]any{"key": "value"}),
)

// Query
results, _ := col.Query(ctx,
    v2.WithQueryTexts("search text"),
    v2.WithNResults(10),
    v2.WithWhere(v2.EqString("key", "value")),
)

// Get
results, _ := col.Get(ctx,
    v2.WithGetIDs("id1", "id2"),
)

// Update/Upsert
col.Update(ctx, v2.WithIDs("id1"), v2.WithDocuments("new doc"))
col.Upsert(ctx, v2.WithIDs("id1"), v2.WithDocuments("doc"))

// Delete
col.Delete(ctx, v2.WithDeleteIDs("id1", "id2"))
col.Delete(ctx, v2.WithDeleteWhere(v2.EqString("key", "value")))
```

### Search API (Cloud)

```go
// Basic search
result, _ := col.Search(ctx,
    v2.NewSearchRequest(
        v2.WithKnnRank(v2.KnnQueryText("query")),
        v2.NewPage(v2.Limit(10)),
        v2.WithSelect(v2.KDocument, v2.KScore),
    ),
)

// Hybrid search with RRF
denseKnn, _ := v2.NewKnnRank(v2.KnnQueryText("query"), v2.WithKnnReturnRank())
sparseKnn, _ := v2.NewKnnRank(v2.KnnQueryText("query"), v2.WithKnnKey(v2.K("sparse")), v2.WithKnnReturnRank())

result, _ := col.Search(ctx,
    v2.NewSearchRequest(
        v2.WithRrfRank(
            v2.WithRrfRanks(denseKnn.WithWeight(0.7), sparseKnn.WithWeight(0.3)),
        ),
        v2.NewPage(v2.Limit(10)),
    ),
)
```

### Where Filters

```go
// Comparison
v2.EqString("key", "value")
v2.GtInt("count", 10)
v2.LteFloat("score", 0.95)

// Logical
v2.And(v2.EqString("a", "b"), v2.GtInt("c", 5))
v2.Or(v2.EqString("x", "1"), v2.EqString("x", "2"))

// In/Not In
v2.InString("category", "a", "b", "c")
v2.NinInt("status", 0, -1)
```

## Requirements

- Go 1.21+
- Chroma server (local or Chroma Cloud)
- chroma-go v2 (`github.com/amikos-tech/chroma-go/pkg/api/v2`)

## Installation

```bash
go get github.com/amikos-tech/chroma-go
```

## Links

- [chroma-go GitHub](https://github.com/amikos-tech/chroma-go)
- [Chroma Documentation](https://docs.trychroma.com/)
- [Chroma Cloud](https://trychroma.com/)

