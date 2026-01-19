# Schema Basics - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/cloud/schema/schema-basics)

## Overview

Learn how to create and use Schema to configure indexes on your Chroma collections. A Schema controls which fields are indexed and how, with support for vector indexes, full-text search, sparse vectors, and metadata inverted indexes.

## Go Examples

### Creating a Schema

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Schema

# Create an empty schema (starts with defaults)
schema = Schema()

# The schema is now ready to be configured
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Create schema with defaults (L2 vector index with HNSW)
	schema, err := v2.NewSchemaWithDefaults()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Or create an empty schema with specific options
	customSchema, err := v2.NewSchema(
		v2.WithDefaultVectorIndex(v2.NewVectorIndexConfig(
			v2.WithSpace(v2.SpaceCosine),
		)),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Schema created: %v, %v", schema, customSchema)
}
```
{% /codetab %}
{% /codetabs %}

### Configuring Vector Index

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Schema, VectorIndexConfig
from chromadb.utils.embedding_functions import OpenAIEmbeddingFunction

schema = Schema()

# Configure vector index with custom embedding function
embedding_function = OpenAIEmbeddingFunction(
    api_key="your-api-key",
    model_name="text-embedding-3-small"
)

schema.create_index(config=VectorIndexConfig(
    space="cosine",
    embedding_function=embedding_function
))
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
)

func main() {
	// Create embedding function
	ef, err := openai.NewOpenAIEmbeddingFunction(
		openai.WithAPIKey("your-api-key"),
		openai.WithModel(openai.TextEmbedding3Small),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create schema with custom vector index configuration
	schema, err := v2.NewSchema(
		v2.WithDefaultVectorIndex(v2.NewVectorIndexConfig(
			v2.WithSpace(v2.SpaceCosine),
			v2.WithVectorEmbeddingFunction(ef),
			v2.WithHnsw(v2.NewHnswConfig(
				v2.WithEfConstruction(100),
				v2.WithMaxNeighbors(16),
				v2.WithEfSearch(10),
			)),
		)),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Schema created: %v", schema)
}
```
{% /codetab %}
{% /codetabs %}

### Configuring Sparse Vector Index

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Schema, SparseVectorIndexConfig, K
from chromadb.utils.embedding_functions import ChromaCloudSpladeEmbeddingFunction

schema = Schema()

# Add sparse vector index for a specific key (required for hybrid search)
sparse_ef = ChromaCloudSpladeEmbeddingFunction()
schema.create_index(
    config=SparseVectorIndexConfig(
        source_key=K.DOCUMENT,
        embedding_function=sparse_ef
    ),
    key="sparse_embedding"
)
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"log"
	"os"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	splade "github.com/amikos-tech/chroma-go/pkg/embeddings/chroma_cloud_splade"
)

func main() {
	// Create sparse embedding function
	sparseEF, err := splade.NewChromaCloudSpladeEmbeddingFunction(
		splade.WithAPIKey(os.Getenv("CHROMA_API_KEY")),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create schema with sparse vector index on a specific key
	schema, err := v2.NewSchema(
		v2.WithDefaultVectorIndex(v2.NewVectorIndexConfig(
			v2.WithSpace(v2.SpaceCosine),
		)),
		v2.WithSparseVectorIndex("sparse_embedding", v2.NewSparseVectorIndexConfig(
			v2.WithSparseEmbeddingFunction(sparseEF),
			v2.WithSparseSourceKey(v2.DocumentKey), // "#document"
		)),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Schema created: %v", schema)
}
```
{% /codetab %}
{% /codetabs %}

### Configuring Metadata Indexes

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Schema, StringInvertedIndexConfig, IntInvertedIndexConfig

schema = Schema()

# Disable string inverted index globally
schema.delete_index(config=StringInvertedIndexConfig())

# Disable int inverted index for a specific key
schema.delete_index(config=IntInvertedIndexConfig(), key="unimportant_count")

# Disable all indexes for a specific key
schema.delete_index(key="temporary_field")
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Create schema with custom metadata index configuration
	schema, err := v2.NewSchema(
		v2.WithDefaultVectorIndex(v2.NewVectorIndexConfig(
			v2.WithSpace(v2.SpaceL2),
		)),
		// Disable string inverted index globally
		v2.DisableDefaultStringIndex(),
		// Disable int inverted index for a specific key
		v2.DisableIntIndex("unimportant_count"),
		// Enable string index for specific keys
		v2.WithStringIndex("category"),
		v2.WithStringIndex("tags"),
		// Enable other indexes for specific keys
		v2.WithIntIndex("year"),
		v2.WithFloatIndex("score"),
		v2.WithBoolIndex("published"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Schema created: %v", schema)
}
```
{% /codetab %}
{% /codetabs %}

### Method Chaining vs Functional Options

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Schema, StringInvertedIndexConfig, IntInvertedIndexConfig

schema = (Schema()
    .delete_index(config=StringInvertedIndexConfig())  # Disable globally
    .create_index(config=StringInvertedIndexConfig(), key="category")  # Enable for category
    .create_index(config=StringInvertedIndexConfig(), key="tags")  # Enable for tags
    .delete_index(config=IntInvertedIndexConfig()))  # Disable int indexing
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Go uses functional options instead of method chaining
	// All configuration is passed to NewSchema at creation time
	schema, err := v2.NewSchema(
		v2.WithDefaultVectorIndex(v2.NewVectorIndexConfig(
			v2.WithSpace(v2.SpaceL2),
		)),
		// Disable string indexing globally
		v2.DisableDefaultStringIndex(),
		// Enable for specific keys
		v2.WithStringIndex("category"),
		v2.WithStringIndex("tags"),
		// Disable int indexing globally
		v2.DisableDefaultIntIndex(),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Schema created: %v", schema)
}
```
{% /codetab %}
{% /codetabs %}

### Using Schema with Collections

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Create collection with schema
collection = client.create_collection(
    name="my_collection",
    schema=schema
)

# Or use get_or_create_collection
collection = client.get_or_create_collection(
    name="my_collection",
    schema=schema
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
	client, err := v2.NewCloudClient(
		v2.WithCloudAPIKey("your-api-key"),
		v2.WithDatabaseAndTenant("database", "tenant"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create schema with configuration
	schema, err := v2.NewSchema(
		v2.WithDefaultVectorIndex(v2.NewVectorIndexConfig(
			v2.WithSpace(v2.SpaceCosine),
		)),
		v2.WithStringIndex("category"),
		v2.WithIntIndex("year"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create collection with schema
	collection, err := client.CreateCollection(ctx, "my_collection",
		v2.WithCollectionSchema(schema),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Or use GetOrCreateCollection
	collection, err = client.GetOrCreateCollection(ctx, "my_collection",
		v2.WithCollectionSchema(schema),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Collection created: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

### Complete Schema Example

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Schema, VectorIndexConfig, SparseVectorIndexConfig, StringInvertedIndexConfig, K
from chromadb.utils.embedding_functions import OpenAIEmbeddingFunction, ChromaCloudSpladeEmbeddingFunction

# Create embedding functions
dense_ef = OpenAIEmbeddingFunction(api_key="your-api-key", model_name="text-embedding-3-small")
sparse_ef = ChromaCloudSpladeEmbeddingFunction()

# Create schema with full configuration
schema = (Schema()
    .create_index(config=VectorIndexConfig(
        space="cosine",
        embedding_function=dense_ef
    ))
    .create_index(config=SparseVectorIndexConfig(
        source_key=K.DOCUMENT,
        embedding_function=sparse_ef
    ), key="sparse_embedding")
    .delete_index(config=StringInvertedIndexConfig())  # Disable globally
    .create_index(config=StringInvertedIndexConfig(), key="category")
    .create_index(config=StringInvertedIndexConfig(), key="author"))

collection = client.create_collection(name="articles", schema=schema)
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
	splade "github.com/amikos-tech/chroma-go/pkg/embeddings/chroma_cloud_splade"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
)

func main() {
	client, err := v2.NewCloudClient(
		v2.WithCloudAPIKey("your-api-key"),
		v2.WithDatabaseAndTenant("database", "tenant"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create embedding functions
	denseEF, err := openai.NewOpenAIEmbeddingFunction(
		openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		openai.WithModel(openai.TextEmbedding3Small),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	sparseEF, err := splade.NewChromaCloudSpladeEmbeddingFunction(
		splade.WithAPIKey(os.Getenv("CHROMA_API_KEY")),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create schema with full configuration
	schema, err := v2.NewSchema(
		// Configure vector index with embedding function
		v2.WithDefaultVectorIndex(v2.NewVectorIndexConfig(
			v2.WithSpace(v2.SpaceCosine),
			v2.WithVectorEmbeddingFunction(denseEF),
			v2.WithHnsw(v2.NewHnswConfig(
				v2.WithEfConstruction(100),
				v2.WithMaxNeighbors(16),
			)),
		)),
		// Configure sparse vector index for hybrid search
		v2.WithSparseVectorIndex("sparse_embedding", v2.NewSparseVectorIndexConfig(
			v2.WithSparseEmbeddingFunction(sparseEF),
			v2.WithSparseSourceKey(v2.DocumentKey),
		)),
		// Disable string indexing globally
		v2.DisableDefaultStringIndex(),
		// Enable for specific keys
		v2.WithStringIndex("category"),
		v2.WithStringIndex("author"),
		// Enable other metadata indexes
		v2.WithIntIndex("year"),
		v2.WithFloatIndex("rating"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create collection with schema
	collection, err := client.CreateCollection(ctx, "articles",
		v2.WithCollectionSchema(schema),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Collection created: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

## Schema API Reference

| Python | Go Function | Description |
|--------|-------------|-------------|
| `Schema()` | `v2.NewSchema(opts...)` | Create new schema |
| `Schema()` (with defaults) | `v2.NewSchemaWithDefaults()` | Create schema with L2 defaults |
| `VectorIndexConfig(space=...)` | `v2.NewVectorIndexConfig(v2.WithSpace(...))` | Configure vector index |
| `SparseVectorIndexConfig(...)` | `v2.NewSparseVectorIndexConfig(...)` | Configure sparse vector index |
| `.create_index(config=..., key=...)` | `v2.WithVectorIndex(key, cfg)` | Add index to key |
| `.delete_index(config=..., key=...)` | `v2.DisableStringIndex(key)` | Disable index on key |
| `.delete_index(config=StringInvertedIndexConfig())` | `v2.DisableDefaultStringIndex()` | Disable string index globally |

### Space Constants

| Python | Go Constant | Description |
|--------|-------------|-------------|
| `"l2"` | `v2.SpaceL2` | Euclidean distance |
| `"cosine"` | `v2.SpaceCosine` | Cosine similarity |
| `"ip"` | `v2.SpaceIP` | Inner product |

### Reserved Keys

| Python | Go Constant | Description |
|--------|-------------|-------------|
| `K.DOCUMENT` | `v2.DocumentKey` (`"#document"`) | Document text content |
| `K.EMBEDDING` | `v2.EmbeddingKey` (`"#embedding"`) | Dense vector embeddings |

## Notes

- Go uses functional options pattern instead of method chaining
- All schema configuration is passed at creation time via `NewSchema(opts...)`
- HNSW configuration is available via `WithHnsw()` for fine-tuning
- SPANN configuration is available via `WithSpann()` for cloud-scale workloads
- Schema is automatically persisted with the collection
- Embedding functions are serialized/deserialized automatically when supported

