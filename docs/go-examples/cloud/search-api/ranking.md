# Ranking Expressions - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/cloud/search-api/ranking)

## Overview

Ranking expressions control how search results are scored and ordered. Chroma provides KNN-based ranking with arithmetic operations for customizing result ordering.

## Go Examples

### Basic KNN Ranking

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, Knn

# Text query (auto-embedded)
search = Search().rank(Knn(query="machine learning research"))

# With custom limit
search = Search().rank(Knn(query="AI applications", limit=100))
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

	// Text query (auto-embedded)
	result1, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText("machine learning research")),
			v2.NewPage(v2.Limit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// With custom KNN limit (larger pool before final limit)
	result2, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(
				v2.KnnQueryText("AI applications"),
				v2.WithKnnLimit(100),
			),
			v2.NewPage(v2.Limit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Results: %v, %v", result1, result2)
}
```
{% /codetab %}
{% /codetabs %}

### KNN with Vector Embeddings

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Vector query (pre-computed embedding)
embedding = [0.1, 0.2, 0.3, ...]  # Your embedding vector
search = Search().rank(Knn(query=embedding))
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
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

	// Pre-computed embedding vector
	embedding := embeddings.NewEmbeddingFromFloat32([]float32{0.1, 0.2, 0.3})

	result, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryVector(embedding)),
			v2.NewPage(v2.Limit(10)),
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

### Arithmetic Operations

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Knn, Val

# Weighted ranking
rank = Knn(query="search") * 0.7 + Val(0.3)

# Score adjustment
rank = Knn(query="search") + Val(1.0)

# Log compression
rank = (Knn(query="search") + Val(1.0)).log()

# Combining multiple KNN queries
rank1 = Knn(query="machine learning")
rank2 = Knn(query="deep learning")
combined = rank1 * 0.7 + rank2 * 0.3
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

	// Weighted ranking: Knn * 0.7 + 0.3
	knn, _ := v2.NewKnnRank(v2.KnnQueryText("search"))
	weightedRank := knn.Multiply(v2.FloatOperand(0.7)).Add(v2.Val(0.3))

	result1, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithRank(weightedRank),
			v2.NewPage(v2.Limit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore),
		),
	)

	// Score adjustment: Knn + 1.0
	knn2, _ := v2.NewKnnRank(v2.KnnQueryText("search"))
	adjustedRank := knn2.Add(v2.Val(1.0))

	result2, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithRank(adjustedRank),
			v2.NewPage(v2.Limit(10)),
		),
	)

	// Log compression: log(Knn + 1.0)
	knn3, _ := v2.NewKnnRank(v2.KnnQueryText("search"))
	logRank := knn3.Add(v2.Val(1.0)).Log()

	result3, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithRank(logRank),
			v2.NewPage(v2.Limit(10)),
		),
	)

	// Combining multiple KNN queries: rank1 * 0.7 + rank2 * 0.3
	knn4, _ := v2.NewKnnRank(v2.KnnQueryText("machine learning"))
	knn5, _ := v2.NewKnnRank(v2.KnnQueryText("deep learning"))
	combinedRank := knn4.Multiply(v2.FloatOperand(0.7)).Add(
		knn5.Multiply(v2.FloatOperand(0.3)),
	)

	result4, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithRank(combinedRank),
			v2.NewPage(v2.Limit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore),
		),
	)

	log.Printf("Results: %v, %v, %v, %v", result1, result2, result3, result4)
}
```
{% /codetab %}
{% /codetabs %}

### Advanced Mathematical Functions

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Knn, Val

knn = Knn(query="search query")

# Absolute value
rank = knn.abs()

# Exponential
rank = knn.exp()

# Natural log (requires positive input)
rank = (knn + Val(1.0)).log()

# Min/Max clamping
rank = knn.max(Val(0.0))  # Clamp to minimum 0
rank = knn.min(Val(1.0))  # Clamp to maximum 1

# Negation
rank = -knn  # Reverse score ordering
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

	knn, _ := v2.NewKnnRank(v2.KnnQueryText("search query"))

	// Absolute value
	absRank := knn.Abs()

	// Exponential
	expRank := knn.Exp()

	// Natural log (add offset to ensure positive)
	logRank := knn.Add(v2.Val(1.0)).Log()

	// Min/Max clamping
	clampMinRank := knn.Max(v2.Val(0.0)) // Clamp to minimum 0
	clampMaxRank := knn.Min(v2.Val(1.0)) // Clamp to maximum 1

	// Negation (reverse score ordering)
	negatedRank := knn.Negate()

	// Use any of these in a search
	result, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithRank(absRank),
			v2.NewPage(v2.Limit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore),
		),
	)

	log.Printf("Rank types: %T, %T, %T, %T, %T, %T",
		absRank, expRank, logRank, clampMinRank, clampMaxRank, negatedRank)
	log.Printf("Results: %v", result)
}
```
{% /codetab %}
{% /codetabs %}

### KNN with Default Scores

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Knn

# Default score for non-matching documents
# Documents not in top-K get a default score instead of being excluded
knn = Knn(query="search", limit=50, default=10.0)
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

	// KNN with default score for non-matching documents
	// Documents not in top-K get score 10.0 instead of being excluded
	result, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(
				v2.KnnQueryText("search"),
				v2.WithKnnLimit(50),
				v2.WithKnnDefault(10.0),
			),
			v2.NewPage(v2.Limit(10)),
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

### Constant Values with Val

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Knn, Val

# Val creates constant value rank expressions
base_score = Val(1.0)

# Combine with KNN for offset
rank = Knn(query="search") + base_score

# Use in complex expressions
rank = (Knn(query="search") / Val(2.0)) + Val(0.5)
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

	// Val creates constant value rank expressions
	baseScore := v2.Val(1.0)

	// Combine with KNN for offset
	knn, _ := v2.NewKnnRank(v2.KnnQueryText("search"))
	rankWithOffset := knn.Add(baseScore)

	// Use in complex expressions: (Knn / 2.0) + 0.5
	knn2, _ := v2.NewKnnRank(v2.KnnQueryText("search"))
	complexRank := knn2.Div(v2.Val(2.0)).Add(v2.Val(0.5))

	result1, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithRank(rankWithOffset),
			v2.NewPage(v2.Limit(10)),
		),
	)

	result2, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithRank(complexRank),
			v2.NewPage(v2.Limit(10)),
		),
	)

	log.Printf("Results: %v, %v", result1, result2)
}
```
{% /codetab %}
{% /codetabs %}

## Arithmetic Operations Reference

| Python | Go Method | Description |
|--------|-----------|-------------|
| `rank + value` | `rank.Add()` | Addition |
| `rank - value` | `rank.Sub()` | Subtraction |
| `rank * value` | `rank.Multiply()` | Multiplication |
| `rank / value` | `rank.Div()` | Division |
| `-rank` | `rank.Negate()` | Negation |
| `rank.abs()` | `rank.Abs()` | Absolute value |
| `rank.exp()` | `rank.Exp()` | Exponential (e^x) |
| `rank.log()` | `rank.Log()` | Natural logarithm |
| `rank.max(value)` | `rank.Max()` | Maximum of two values |
| `rank.min(value)` | `rank.Min()` | Minimum of two values |
| `Val(x)` | `v2.Val(x)` | Constant value |

## KNN Options Reference

| Python | Go Function | Description |
|--------|-------------|-------------|
| `query="text"` | `v2.KnnQueryText("text")` | Text query (auto-embedded) |
| `query=[...]` | `v2.KnnQueryVector(vec)` | Vector query |
| `limit=N` | `v2.WithKnnLimit(N)` | Max neighbors to retrieve |
| `key="field"` | `v2.WithKnnKey(K("field"))` | Embedding field to search |
| `default=N` | `v2.WithKnnDefault(N)` | Score for non-matches |
| `return_rank=True` | `v2.WithKnnReturnRank()` | Return rank position (for RRF) |

## Notes

- Lower scores are better (distance-based scoring)
- Use `Val()` to create constant values for arithmetic operations
- Use `FloatOperand()` or `IntOperand()` when multiplying/dividing by constants
- Log requires positive values - add an offset before calling `.Log()`
- The `WithKnnLimit()` controls the candidate pool before filtering
- Use `WithKnnDefault()` for inclusive multi-query searches

