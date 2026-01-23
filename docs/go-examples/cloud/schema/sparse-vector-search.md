# Sparse Vector Search Setup - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/cloud/schema/sparse-vector-search)

## Overview

Learn how to configure and use sparse vectors for keyword-based search, and combine them with dense embeddings for powerful hybrid search capabilities.

## Go Examples

### Enabling Sparse Vector Index

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Schema, SparseVectorIndexConfig, K
from chromadb.utils.embedding_functions import ChromaCloudSpladeEmbeddingFunction

schema = Schema()

# Add sparse vector index for keyword-based search
# "sparse_embedding" is just a metadata key name - use any name you prefer
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

	// Create schema with sparse vector index
	// "sparse_embedding" is the metadata key name where sparse vectors are stored
	schema, err := v2.NewSchema(
		v2.WithDefaultVectorIndex(v2.NewVectorIndexConfig(
			v2.WithSpace(v2.SpaceCosine),
		)),
		v2.WithSparseVectorIndex("sparse_embedding", v2.NewSparseVectorIndexConfig(
			v2.WithSparseEmbeddingFunction(sparseEF),
			v2.WithSparseSourceKey(v2.DocumentKey), // Generate from document text
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

### Create Collection with Schema

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import chromadb

client = chromadb.CloudClient(
    tenant="your-tenant",
    database="your-database",
    api_key="your-api-key"
)

collection = client.create_collection(
    name="hybrid_search_collection",
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
	"os"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	splade "github.com/amikos-tech/chroma-go/pkg/embeddings/chroma_cloud_splade"
)

func main() {
	client, err := v2.NewCloudClient(
		v2.WithCloudAPIKey(os.Getenv("CHROMA_API_KEY")),
		v2.WithDatabaseAndTenant("your-database", "your-tenant"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create sparse embedding function
	sparseEF, err := splade.NewChromaCloudSpladeEmbeddingFunction(
		splade.WithAPIKey(os.Getenv("CHROMA_API_KEY")),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create schema with sparse vector index
	schema, err := v2.NewSchema(
		v2.WithDefaultVectorIndex(v2.NewVectorIndexConfig(
			v2.WithSpace(v2.SpaceCosine),
		)),
		v2.WithSparseVectorIndex("sparse_embedding", v2.NewSparseVectorIndexConfig(
			v2.WithSparseEmbeddingFunction(sparseEF),
			v2.WithSparseSourceKey(v2.DocumentKey),
		)),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create collection with schema
	collection, err := client.CreateCollection(ctx, "hybrid_search_collection",
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

### Add Data

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.add(
    ids=["doc1", "doc2", "doc3"],
    documents=[
        "The quick brown fox jumps over the lazy dog",
        "A fast auburn fox leaps over a sleepy canine",
        "Machine learning is a subset of artificial intelligence"
    ],
    metadatas=[
        {"category": "animals"},
        {"category": "animals"},
        {"category": "technology"}
    ]
)

# Sparse embeddings for "sparse_embedding" are generated automatically
# from the documents (source_key=K.DOCUMENT)
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
	// Assuming collection is already created with sparse vector schema
	ctx := context.Background()

	// Add documents - sparse embeddings are generated automatically
	// from the documents based on the schema configuration
	err := collection.Add(ctx,
		v2.WithIDs("doc1", "doc2", "doc3"),
		v2.WithDocuments(
			"The quick brown fox jumps over the lazy dog",
			"A fast auburn fox leaps over a sleepy canine",
			"Machine learning is a subset of artificial intelligence",
		),
		v2.WithMetadatas(
			map[string]interface{}{"category": "animals"},
			map[string]interface{}{"category": "animals"},
			map[string]interface{}{"category": "technology"},
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Println("Documents added with auto-generated sparse embeddings")
}
```
{% /codetab %}
{% /codetabs %}

### Sparse Vector Search

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, K, Knn

# Search using sparse embeddings only
sparse_rank = Knn(query="fox animal", key="sparse_embedding")

# Build and execute search
search = (Search()
    .rank(sparse_rank)
    .limit(10)
    .select(K.DOCUMENT, K.SCORE))

results = collection.search(search)

# Process results
for row in results.rows()[0]:
    print(f"Score: {row['score']:.3f} - {row['document']}")
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"fmt"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	ctx := context.Background()

	// Search using sparse embeddings only
	// The query text will be converted to sparse vector using the configured embedding function
	result, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(
				v2.KnnQueryText("fox animal"),
				v2.WithKnnKey(v2.K("sparse_embedding")), // Search on sparse index
				v2.WithKnnLimit(50),
			),
			v2.NewPage(v2.Limit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Process results using Rows()
	for _, row := range result.Rows() {
		fmt.Printf("Score: %.3f - %s\n", row.Score, row.Document)
	}
}
```
{% /codetab %}
{% /codetabs %}

### Hybrid Search with RRF

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, K, Knn, Rrf

# Create RRF ranking combining dense and sparse embeddings
hybrid_rank = Rrf(
    ranks=[
        Knn(query="fox animal", return_rank=True),           # Dense semantic search
        Knn(query="fox animal", key="sparse_embedding", return_rank=True)  # Sparse keyword search
    ],
    weights=[0.7, 0.3],  # 70% semantic, 30% keyword
    k=60
)

# Build and execute search
search = (Search()
    .rank(hybrid_rank)
    .limit(10)
    .select(K.DOCUMENT, K.SCORE))

results = collection.search(search)

# Process results
for row in results.rows()[0]:
    print(f"Score: {row['score']:.3f} - {row['document']}")
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"fmt"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	ctx := context.Background()

	// Create KNN ranks for dense and sparse search
	denseKnn, err := v2.NewKnnRank(
		v2.KnnQueryText("fox animal"),
		v2.WithKnnReturnRank(), // Required for RRF
		v2.WithKnnLimit(100),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	sparseKnn, err := v2.NewKnnRank(
		v2.KnnQueryText("fox animal"),
		v2.WithKnnKey(v2.K("sparse_embedding")), // Search on sparse index
		v2.WithKnnReturnRank(),                   // Required for RRF
		v2.WithKnnLimit(100),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create RRF ranking combining dense and sparse
	result, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithRrfRank(
				v2.WithRrfRanks(
					denseKnn.WithWeight(0.7),  // 70% semantic
					sparseKnn.WithWeight(0.3), // 30% keyword
				),
				v2.WithRrfK(60),
			),
			v2.NewPage(v2.Limit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Process results using Rows()
	for _, row := range result.Rows() {
		fmt.Printf("Score: %.3f - %s\n", row.Score, row.Document)
	}
}
```
{% /codetab %}
{% /codetabs %}

### Complete Hybrid Search Example

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import chromadb
from chromadb import Schema, SparseVectorIndexConfig, VectorIndexConfig, K, Search, Knn, Rrf
from chromadb.utils.embedding_functions import OpenAIEmbeddingFunction, ChromaCloudSpladeEmbeddingFunction

# Create client
client = chromadb.CloudClient(
    tenant="your-tenant",
    database="your-database",
    api_key="your-api-key"
)

# Create embedding functions
dense_ef = OpenAIEmbeddingFunction(api_key="your-openai-key")
sparse_ef = ChromaCloudSpladeEmbeddingFunction()

# Create schema
schema = (Schema()
    .create_index(config=VectorIndexConfig(space="cosine", embedding_function=dense_ef))
    .create_index(config=SparseVectorIndexConfig(
        source_key=K.DOCUMENT,
        embedding_function=sparse_ef
    ), key="sparse_embedding"))

# Create collection
collection = client.create_collection(name="hybrid_demo", schema=schema)

# Add data
collection.add(
    ids=["1", "2", "3"],
    documents=[
        "Python is a programming language",
        "Machine learning uses algorithms",
        "Deep learning is a subset of ML"
    ]
)

# Hybrid search
results = collection.search(Search()
    .rank(Rrf(
        ranks=[
            Knn(query="programming", return_rank=True),
            Knn(query="programming", key="sparse_embedding", return_rank=True)
        ],
        weights=[0.6, 0.4]
    ))
    .limit(10)
    .select(K.DOCUMENT, K.SCORE))
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	splade "github.com/amikos-tech/chroma-go/pkg/embeddings/chroma_cloud_splade"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
)

func main() {
	// Create client
	client, err := v2.NewCloudClient(
		v2.WithCloudAPIKey(os.Getenv("CHROMA_API_KEY")),
		v2.WithDatabaseAndTenant("your-database", "your-tenant"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create embedding functions
	denseEF, err := openai.NewOpenAIEmbeddingFunction(
		openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
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

	// Create schema with both dense and sparse indexes
	schema, err := v2.NewSchema(
		v2.WithDefaultVectorIndex(v2.NewVectorIndexConfig(
			v2.WithSpace(v2.SpaceCosine),
			v2.WithVectorEmbeddingFunction(denseEF),
		)),
		v2.WithSparseVectorIndex("sparse_embedding", v2.NewSparseVectorIndexConfig(
			v2.WithSparseEmbeddingFunction(sparseEF),
			v2.WithSparseSourceKey(v2.DocumentKey),
		)),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create collection
	collection, err := client.CreateCollection(ctx, "hybrid_demo",
		v2.WithCollectionSchema(schema),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Add data - embeddings generated automatically
	err = collection.Add(ctx,
		v2.WithIDs("1", "2", "3"),
		v2.WithDocuments(
			"Python is a programming language",
			"Machine learning uses algorithms",
			"Deep learning is a subset of ML",
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create KNN ranks for hybrid search
	denseKnn, _ := v2.NewKnnRank(
		v2.KnnQueryText("programming"),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(100),
	)

	sparseKnn, _ := v2.NewKnnRank(
		v2.KnnQueryText("programming"),
		v2.WithKnnKey(v2.K("sparse_embedding")),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(100),
	)

	// Execute hybrid search
	result, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithRrfRank(
				v2.WithRrfRanks(
					denseKnn.WithWeight(0.6),
					sparseKnn.WithWeight(0.4),
				),
			),
			v2.NewPage(v2.Limit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Print results using Rows()
	fmt.Println("Hybrid Search Results:")
	for _, row := range result.Rows() {
		fmt.Printf("  %.3f: %s\n", row.Score, row.Document)
	}
}
```
{% /codetab %}
{% /codetabs %}

## Sparse Vector Configuration Reference

| Python | Go Function | Description |
|--------|-------------|-------------|
| `SparseVectorIndexConfig(...)` | `v2.NewSparseVectorIndexConfig(...)` | Create sparse vector config |
| `source_key=K.DOCUMENT` | `v2.WithSparseSourceKey(v2.DocumentKey)` | Set source field for text |
| `embedding_function=ef` | `v2.WithSparseEmbeddingFunction(ef)` | Set sparse embedding function |
| `schema.create_index(..., key="name")` | `v2.WithSparseVectorIndex("name", cfg)` | Add sparse index to schema |

## Available Sparse Embedding Functions

| Provider | Go Package | Description |
|----------|------------|-------------|
| Chroma Cloud SPLADE | `github.com/amikos-tech/chroma-go/pkg/embeddings/chroma_cloud_splade` | Cloud-hosted SPLADE |
| HuggingFace | `github.com/amikos-tech/chroma-go/pkg/embeddings/hf` | Hugging Face sparse models |

## Notes

- Sparse vectors excel at exact keyword matching and domain-specific terms
- Dense vectors capture semantic meaning - use both for best results
- The `source_key` specifies which field generates sparse embeddings (usually `#document`)
- Sparse embeddings are auto-generated when adding documents with a schema
- Use RRF to combine dense and sparse search for hybrid retrieval
- Weight tuning depends on your use case - start with 0.7/0.3 and adjust
- Use `Rows()` for ergonomic iteration over results
- Use `At(group, index)` for safe indexed access with bounds checking

