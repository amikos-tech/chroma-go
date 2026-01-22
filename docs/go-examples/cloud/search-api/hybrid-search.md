# Hybrid Search with RRF - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/cloud/search-api/hybrid-search)

## Overview

Learn how to combine multiple ranking strategies using Reciprocal Rank Fusion (RRF). RRF is ideal for hybrid search scenarios where you want to merge results from different ranking methods (e.g., dense and sparse embeddings).

## Go Examples

### Basic RRF

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, Knn, Rrf

# RRF combines multiple rankings by rank position
rrf = Rrf([
    Knn(query="machine learning", return_rank=True),
    Knn(query="machine learning", key="sparse_embedding", return_rank=True)
])

search = Search().rank(rrf).limit(10)
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
	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create KNN ranks with return_rank=True for RRF
	denseKnn, _ := v2.NewKnnRank(
		v2.KnnQueryText("machine learning"),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(100),
	)

	sparseKnn, _ := v2.NewKnnRank(
		v2.KnnQueryText("machine learning"),
		v2.WithKnnKey(v2.K("sparse_embedding")),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(100),
	)

	// Create RRF with weighted ranks
	result, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithRffRank(
				v2.WithRffRanks(
					denseKnn.WithWeight(1.0),
					sparseKnn.WithWeight(1.0),
				),
			),
			v2.WithPage(v2.PageLimit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Results: %v", result)
}
```
{% /codetab %}
{% /codetabs %}

### RRF with Custom Weights

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Custom weights - adjust relative importance
rrf = Rrf(
    ranks=[
        Knn(query="neural networks", return_rank=True),
        Knn(query="neural networks", key="sparse_embedding", return_rank=True)
    ],
    weights=[3.0, 1.0]  # Dense 3x more important than sparse
)

# Normalized weights - ensures weights sum to 1.0
rrf = Rrf(
    ranks=[
        Knn(query="neural networks", return_rank=True),
        Knn(query="neural networks", key="sparse_embedding", return_rank=True)
    ],
    weights=[75, 25],     # Will be normalized to [0.75, 0.25]
    normalize=True
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
	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create KNN ranks
	denseKnn, _ := v2.NewKnnRank(
		v2.KnnQueryText("neural networks"),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(100),
	)

	sparseKnn, _ := v2.NewKnnRank(
		v2.KnnQueryText("neural networks"),
		v2.WithKnnKey(v2.K("sparse_embedding")),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(100),
	)

	// Custom weights - Dense 3x more important
	result1, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithRffRank(
				v2.WithRffRanks(
					denseKnn.WithWeight(3.0),
					sparseKnn.WithWeight(1.0),
				),
			),
			v2.WithPage(v2.PageLimit(10)),
		),
	)

	// Normalized weights - ensures weights sum to 1.0
	denseKnn2, _ := v2.NewKnnRank(
		v2.KnnQueryText("neural networks"),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(100),
	)

	sparseKnn2, _ := v2.NewKnnRank(
		v2.KnnQueryText("neural networks"),
		v2.WithKnnKey(v2.K("sparse_embedding")),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(100),
	)

	result2, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithRffRank(
				v2.WithRffRanks(
					denseKnn2.WithWeight(75),
					sparseKnn2.WithWeight(25),
				),
				v2.WithRffNormalize(), // Normalize to sum to 1.0
			),
			v2.WithPage(v2.PageLimit(10)),
		),
	)

	log.Printf("Results: %v, %v", result1, result2)
}
```
{% /codetab %}
{% /codetabs %}

### RRF with Custom k Parameter

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Small k - top results heavily weighted
rrf = Rrf(ranks=[...], k=10)

# Default k - balanced (standard in literature)
rrf = Rrf(ranks=[...], k=60)

# Large k - more uniform weighting across ranks
rrf = Rrf(ranks=[...], k=200)
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
	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	denseKnn, _ := v2.NewKnnRank(
		v2.KnnQueryText("search query"),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(100),
	)

	sparseKnn, _ := v2.NewKnnRank(
		v2.KnnQueryText("search query"),
		v2.WithKnnKey(v2.K("sparse_embedding")),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(100),
	)

	// Small k=10 - top results heavily weighted
	result1, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithRffRank(
				v2.WithRffRanks(
					denseKnn.WithWeight(1.0),
					sparseKnn.WithWeight(1.0),
				),
				v2.WithRffK(10),
			),
			v2.WithPage(v2.PageLimit(10)),
		),
	)

	// Default k=60 - balanced (default)
	denseKnn2, _ := v2.NewKnnRank(
		v2.KnnQueryText("search query"),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(100),
	)
	sparseKnn2, _ := v2.NewKnnRank(
		v2.KnnQueryText("search query"),
		v2.WithKnnKey(v2.K("sparse_embedding")),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(100),
	)

	result2, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithRffRank(
				v2.WithRffRanks(
					denseKnn2.WithWeight(1.0),
					sparseKnn2.WithWeight(1.0),
				),
				// k=60 is default, so not needed
			),
			v2.WithPage(v2.PageLimit(10)),
		),
	)

	log.Printf("Results: %v, %v", result1, result2)
}
```
{% /codetab %}
{% /codetabs %}

### Dense + Sparse Hybrid Search

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, K, Knn, Rrf

# Dense semantic embeddings
dense_rank = Knn(
    query="machine learning research",
    key="#embedding",
    return_rank=True,
    limit=200
)

# Sparse keyword embeddings
sparse_rank = Knn(
    query="machine learning research",
    key="sparse_embedding",
    return_rank=True,
    limit=200
)

# Combine with RRF
hybrid_rank = Rrf(
    ranks=[dense_rank, sparse_rank],
    weights=[0.7, 0.3],  # 70% semantic, 30% keyword
    k=60
)

search = (Search()
    .where(K("status") == "published")
    .rank(hybrid_rank)
    .limit(20)
    .select(K.DOCUMENT, K.SCORE, "title")
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
	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Dense semantic embeddings (default #embedding key)
	denseRank, _ := v2.NewKnnRank(
		v2.KnnQueryText("machine learning research"),
		v2.WithKnnKey(v2.KEmbedding), // Default key
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(200),
	)

	// Sparse keyword embeddings
	sparseRank, _ := v2.NewKnnRank(
		v2.KnnQueryText("machine learning research"),
		v2.WithKnnKey(v2.K("sparse_embedding")),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(200),
	)

	// Combine with RRF - 70% semantic, 30% keyword
	result, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.EqString("status", "published")),
			v2.WithRffRank(
				v2.WithRffRanks(
					denseRank.WithWeight(0.7),
					sparseRank.WithWeight(0.3),
				),
				v2.WithRffK(60),
			),
			v2.WithPage(v2.PageLimit(20)),
			v2.WithSelect(v2.KDocument, v2.KScore, v2.K("title")),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Results: %v", result)
}
```
{% /codetab %}
{% /codetabs %}

### Complete Hybrid Search Example

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, K, Knn, Rrf

# Create RRF ranking with text query
hybrid_rank = Rrf(
    ranks=[
        Knn(query="machine learning applications", return_rank=True, limit=300),
        Knn(query="machine learning applications", key="sparse_embedding", return_rank=True, limit=300)
    ],
    weights=[2.0, 1.0],  # Dense 2x more important
    k=60
)

# Build complete search
search = (Search()
    .where(
        (K("language") == "en") &
        (K("year") >= 2020)
    )
    .rank(hybrid_rank)
    .limit(10)
    .select(K.DOCUMENT, K.SCORE, "title", "year")
)

results = collection.search(search)
rows = results.rows()[0]

for i, row in enumerate(rows, 1):
    print(f"{i}. {row['metadata']['title']} ({row['metadata']['year']})")
    print(f"   RRF Score: {row['score']:.4f}")
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
	client, err := v2.NewCloudClient(
		v2.WithCloudAPIKey("your-api-key"),
		v2.WithDatabaseAndTenant("database", "tenant"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create KNN ranks for hybrid search
	denseKnn, _ := v2.NewKnnRank(
		v2.KnnQueryText("machine learning applications"),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(300),
	)

	sparseKnn, _ := v2.NewKnnRank(
		v2.KnnQueryText("machine learning applications"),
		v2.WithKnnKey(v2.K("sparse_embedding")),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(300),
	)

	// Build complete search with filter and RRF ranking
	result, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(
				v2.And(
					v2.EqString("language", "en"),
					v2.GteInt("year", 2020),
				),
			),
			v2.WithRffRank(
				v2.WithRffRanks(
					denseKnn.WithWeight(2.0),  // Dense 2x more important
					sparseKnn.WithWeight(1.0),
				),
				v2.WithRffK(60),
			),
			v2.WithPage(v2.PageLimit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore, v2.K("title"), v2.K("year")),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Process results using Rows()
	for i, row := range result.Rows() {
		title := ""
		year := 0
		if row.Metadata != nil {
			if t, ok := row.Metadata.Get("title"); ok {
				title = fmt.Sprintf("%v", t)
			}
			if y, ok := row.Metadata.Get("year"); ok {
				year = int(y.(float64))
			}
		}
		fmt.Printf("%d. %s (%d)\n", i+1, title, year)
		fmt.Printf("   RRF Score: %.4f\n", row.Score)
	}
}
```
{% /codetab %}
{% /codetabs %}

## RRF Parameters Reference

| Python | Go Function | Description |
|--------|-------------|-------------|
| `ranks=[...]` | `v2.WithRffRanks()` | List of weighted ranks |
| `weights=[...]` | `.WithWeight(w)` on each rank | Weight for each ranking |
| `k=N` | `v2.WithRffK(N)` | Smoothing parameter (default: 60) |
| `normalize=True` | `v2.WithRffNormalize()` | Normalize weights to sum to 1.0 |

## RRF Formula

RRF combines rankings using: `score = -sum(weight_i / (k + rank_i))`

Where:
- `weight_i` = weight for ranking i (default: 1.0)
- `rank_i` = rank position from ranking i (0, 1, 2, ...)
- `k` = smoothing parameter (default: 60)

The score is negative because Chroma uses ascending order (lower = better).

## Notes

- Always use `WithKnnReturnRank()` for all KNN expressions in RRF
- Set appropriate limits on component KNN expressions (usually 100-500)
- Default k=60 works well for most cases
- Use `.WithWeight()` to adjust relative importance of each ranking
- RRF is scale-agnostic, making it ideal for combining different embedding types
- Use `Rows()` for ergonomic iteration over results
- Use `At(group, index)` for safe indexed access with bounds checking

