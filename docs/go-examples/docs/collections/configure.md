# Configuring Collections - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/collections/configure)

## Overview

Chroma collections have a configuration that determines how their embeddings index is constructed and used. You can customize index configuration values when creating a collection.

## HNSW Index Configuration

### Creating Collection with HNSW Configuration

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection = client.create_collection(
    name="my-collection",
    embedding_function=OpenAIEmbeddingFunction(model_name="text-embedding-3-small"),
    configuration={
        "hnsw": {
            "space": "cosine",
            "ef_construction": 200
        }
    }
)
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"log"
	"os"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
)

func main() {
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create OpenAI embedding function
	ef, err := openai.NewOpenAIEmbeddingFunction(
		os.Getenv("OPENAI_API_KEY"),
		openai.WithModel(openai.TextEmbedding3Small),
	)
	if err != nil {
		log.Fatalf("Error creating embedding function: %v", err)
	}

	// Create collection with HNSW configuration
	collection, err := client.CreateCollection(ctx, "my-collection",
		v2.WithEmbeddingFunctionCreate(ef),
		v2.WithHNSWSpaceCreate(v2.Cosine),        // Distance function
		v2.WithHNSWConstructionEfCreate(200),     // Construction EF
	)
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	log.Printf("Created collection: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

### Distance Space Options

{% codetabs group="lang" %}
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
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// L2 (Squared L2 norm) - default
	// Measures absolute geometric distance
	l2Collection, err := client.CreateCollection(ctx, "l2-collection",
		v2.WithHNSWSpaceCreate(v2.L2),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Cosine similarity
	// Measures angle between vectors (ignoring magnitude)
	// Ideal for text embeddings
	cosineCollection, err := client.CreateCollection(ctx, "cosine-collection",
		v2.WithHNSWSpaceCreate(v2.Cosine),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Inner product
	// Focuses on vector alignment and magnitude
	// Often used for recommendation systems
	ipCollection, err := client.CreateCollection(ctx, "ip-collection",
		v2.WithHNSWSpaceCreate(v2.IP),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Collections created: %s, %s, %s",
		l2Collection.Name(),
		cosineCollection.Name(),
		ipCollection.Name(),
	)
}
```
{% /codetab %}
{% /codetabs %}

### Full HNSW Configuration

{% codetabs group="lang" %}
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
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create collection with all HNSW parameters
	collection, err := client.CreateCollection(ctx, "configured-collection",
		v2.WithHNSWSpaceCreate(v2.Cosine),        // Distance function
		v2.WithHNSWConstructionEfCreate(200),     // Construction EF (index quality)
		v2.WithHNSWSearchEfCreate(100),           // Search EF (recall vs speed)
		v2.WithHNSWMCreate(16),                   // Max neighbors per node
		v2.WithHNSWBatchSizeCreate(100),          // Vectors per batch
		v2.WithHNSWSyncThresholdCreate(1000),     // Sync to storage threshold
		v2.WithHNSWResizeFactorCreate(1.2),       // Index resize factor
	)
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	log.Printf("Created collection with HNSW config: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

## Embedding Function Configuration

### Collection with OpenAI Embedding Function

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import os
from chromadb.utils.embedding_functions import OpenAIEmbeddingFunction

openai_collection = client.create_collection(
    name="my_openai_collection",
    embedding_function=OpenAIEmbeddingFunction(
        model_name="text-embedding-3-small"
    ),
    configuration={"hnsw": {"space": "cosine"}}
)
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"log"
	"os"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
)

func main() {
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create OpenAI embedding function
	openaiEF, err := openai.NewOpenAIEmbeddingFunction(
		os.Getenv("OPENAI_API_KEY"),
		openai.WithModel(openai.TextEmbedding3Small),
	)
	if err != nil {
		log.Fatalf("Error creating embedding function: %v", err)
	}

	// Create collection with OpenAI embeddings and cosine distance
	collection, err := client.CreateCollection(ctx, "my_openai_collection",
		v2.WithEmbeddingFunctionCreate(openaiEF),
		v2.WithHNSWSpaceCreate(v2.Cosine),
	)
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	log.Printf("Created OpenAI collection: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

### Collection with Cohere Embedding Function

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb.utils.embedding_functions import CohereEmbeddingFunction

cohere_collection = client.get_or_create_collection(
    name="my_cohere_collection",
    configuration={
        "embedding_function": CohereEmbeddingFunction(
            model_name="embed-english-light-v2.0",
            truncate="NONE"
        ),
        "hnsw": {"space": "cosine"}
    }
)
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"log"
	"os"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/cohere"
)

func main() {
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create Cohere embedding function
	cohereEF, err := cohere.NewCohereEmbeddingFunction(
		os.Getenv("COHERE_API_KEY"),
		cohere.WithModel("embed-english-light-v2.0"),
	)
	if err != nil {
		log.Fatalf("Error creating embedding function: %v", err)
	}

	// Create collection with Cohere embeddings
	collection, err := client.GetOrCreateCollection(ctx, "my_cohere_collection",
		v2.WithEmbeddingFunctionCreate(cohereEF),
		v2.WithHNSWSpaceCreate(v2.Cosine),
	)
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	log.Printf("Created Cohere collection: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

## HNSW Parameters Reference

| Parameter | Default | Description |
|-----------|---------|-------------|
| `space` | `l2` | Distance function: `l2`, `cosine`, or `ip` |
| `ef_construction` | `100` | Index quality (higher = better, slower build) |
| `ef_search` | `100` | Search quality (higher = better recall, slower query) |
| `max_neighbors` (M) | `16` | Max connections per node |
| `batch_size` | `100` | Vectors processed per batch |
| `sync_threshold` | `1000` | When to sync to storage |
| `resize_factor` | `1.2` | Index growth factor |

## Go Option Functions

| Python Config | Go Function |
|--------------|-------------|
| `space: "l2"` | `v2.WithHNSWSpaceCreate(v2.L2)` |
| `space: "cosine"` | `v2.WithHNSWSpaceCreate(v2.Cosine)` |
| `space: "ip"` | `v2.WithHNSWSpaceCreate(v2.IP)` |
| `ef_construction: N` | `v2.WithHNSWConstructionEfCreate(N)` |
| `ef_search: N` | `v2.WithHNSWSearchEfCreate(N)` |
| `max_neighbors: N` | `v2.WithHNSWMCreate(N)` |

## Notes

- Choose `space` based on your embedding function's recommendation
- Higher `ef_construction` improves index quality but increases build time
- Higher `ef_search` improves recall but increases query time
- Embedding functions are persisted with the collection (Chroma >= 1.1.13)
- API keys for embedding providers are read from standard environment variables
