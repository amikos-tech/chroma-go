# Troubleshooting - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/overview/troubleshooting)

## Overview

This page covers common issues when using chroma-go and how to resolve them.

## Connection Issues

### Cannot Connect to Server

```go
package main

import (
	"context"
	"log"
	"time"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Create client with timeout
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://localhost:8000"),
	)
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	// Use context with timeout for connection check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if server is running
	heartbeat, err := client.Heartbeat(ctx)
	if err != nil {
		log.Fatalf("Cannot connect to Chroma server at localhost:8000: %v", err)
		log.Println("Make sure the Chroma server is running:")
		log.Println("  docker run -p 8000:8000 chromadb/chroma")
		log.Println("  OR")
		log.Println("  chroma run --path /path/to/data")
	}

	log.Printf("Connected! Heartbeat: %d", heartbeat)
}
```

### SSL/TLS Issues

```go
package main

import (
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// For HTTPS connections
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("https://chroma.example.com"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	// If using self-signed certificates, you may need custom transport
	// See Go's http.Transport for TLS configuration
}
```

## Embeddings Issues

### Embeddings Return nil

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Embeddings are not returned by default
results = collection.query(
    query_texts="hello",
    n_results=1,
    include=["embeddings", "documents", "metadatas", "distances"],
)
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	collection, _ := client.GetCollection(ctx, "my_collection")

	// By default, embeddings are NOT returned (they're large)
	results, _ := collection.Query(ctx,
		v2.WithQueryTexts("hello"),
		v2.WithNResults(1),
	)
	// results.Embeddings will be nil

	// To include embeddings, use WithInclude
	resultsWithEmbeddings, err := collection.Query(ctx,
		v2.WithQueryTexts("hello"),
		v2.WithNResults(1),
		v2.WithInclude(v2.IncludeEmbeddings, v2.IncludeDocuments, v2.IncludeMetadatas, v2.IncludeDistances),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Now embeddings are included
	if resultsWithEmbeddings.Embeddings != nil {
		log.Printf("Embeddings: %v", resultsWithEmbeddings.Embeddings[0][0][:5]) // First 5 dims
	}
}
```
{% /codetab %}
{% /codetabs %}

### Embedding Function Not Set

```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
)

func main() {
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create embedding function
	ef, err := openai.NewOpenAIEmbeddingFunction(
		openai.WithAPIKey("your-api-key"),
	)
	if err != nil {
		log.Fatalf("Error creating embedding function: %v", err)
	}

	// Create collection WITH embedding function
	collection, err := client.CreateCollection(ctx, "my_collection",
		v2.WithCollectionEmbeddingFunction(ef),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Now you can add documents without providing embeddings
	err = collection.Add(ctx,
		v2.WithIDs("doc1"),
		v2.WithDocuments("This will be embedded automatically"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
```

## HNSW Configuration Issues

### "Cannot return results in contiguous 2D array"

This error occurs when HNSW parameters are too restrictive:

```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create collection with larger HNSW parameters
	schema, _ := v2.NewSchema(
		v2.WithDefaultVectorIndex(v2.NewVectorIndexConfig(
			v2.WithSpace(v2.SpaceCosine),
			v2.WithHnsw(v2.NewHnswConfig(
				v2.WithEfConstruction(200), // Increase from default 100
				v2.WithMaxNeighbors(32),    // Increase from default 16
				v2.WithEfSearch(100),       // Increase for better recall
			)),
		)),
	)

	collection, err := client.CreateCollection(ctx, "better_hnsw_collection",
		v2.WithCollectionSchema(schema),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Or reduce n_results in your query
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("query"),
		v2.WithNResults(5), // Reduce from larger number
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Found %d results", len(results.IDs[0]))
}
```

## Data Type Issues

### Metadata Type Assertions

```go
package main

import (
	"context"
	"fmt"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	client, _ := v2.NewHTTPClient()
	defer client.Close()

	ctx := context.Background()
	collection, _ := client.GetCollection(ctx, "my_collection")

	results, _ := collection.Get(ctx,
		v2.WithGetIDs("doc1"),
		v2.WithGetInclude(v2.IncludeMetadatas),
	)

	// Safely access metadata with type assertions
	if len(results.Metadatas) > 0 && len(results.Metadatas[0]) > 0 {
		metadata := results.Metadatas[0][0]

		// Use Get method which returns (value, ok)
		if category, ok := metadata.Get("category"); ok {
			fmt.Printf("Category: %s\n", category)
		}

		// Type assertion for specific types
		if year, ok := metadata.Get("year"); ok {
			switch v := year.(type) {
			case int:
				fmt.Printf("Year (int): %d\n", v)
			case float64:
				fmt.Printf("Year (float64): %.0f\n", v)
			default:
				fmt.Printf("Year (unknown type): %v\n", v)
			}
		}
	}
}
```

## Context and Timeout Issues

### Operations Timing Out

```go
package main

import (
	"context"
	"log"
	"time"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	// Use longer timeout for large operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	collection, _ := client.GetCollection(ctx, "large_collection")

	// Large batch operation with timeout
	ids := make([]string, 10000)
	docs := make([]string, 10000)
	for i := range ids {
		ids[i] = fmt.Sprintf("doc_%d", i)
		docs[i] = fmt.Sprintf("Document content %d", i)
	}

	err = collection.Add(ctx,
		v2.WithIDs(ids...),
		v2.WithDocuments(docs...),
	)
	if err != nil {
		log.Fatalf("Error (check timeout): %v", err)
	}
}
```

## Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| `connection refused` | Server not running | Start Chroma server first |
| `context deadline exceeded` | Operation timeout | Increase context timeout |
| `embeddings is nil` | Embeddings not requested | Use `WithInclude(v2.IncludeEmbeddings)` |
| `collection not found` | Collection doesn't exist | Use `GetOrCreateCollection` |
| `invalid api key` | Authentication failed | Check API key configuration |

## Notes

- Always check errors returned from chroma-go functions
- Use contexts with timeouts for production code
- For large batch operations, consider chunking data
- When using embedding functions, ensure API keys are configured
- Check Chroma server logs for server-side errors
- Visit [GitHub Issues](https://github.com/amikos-tech/chroma-go/issues) for more help

