# Chroma Cloud Ranking API - Go Examples

This document provides Go equivalents for all ranking examples from the [Chroma Cloud Ranking Documentation](https://docs.trychroma.com/cloud/search-api/ranking).

## Table of Contents

- [Basic KNN Usage](#basic-knn-usage)
- [Multiple KNN with Document Filtering](#multiple-knn-with-document-filtering)
- [Weighted Combinations](#weighted-combinations)
- [Mathematical Functions](#mathematical-functions)
- [Query Format Examples](#query-format-examples)
- [Dictionary/JSON Syntax](#dictionaryjson-syntax)
- [Edge Cases](#edge-cases)
- [Complete Real-World Example](#complete-real-world-example)

---

## Basic KNN Usage

### Single Query (Default limit of 16)

**Python:**
```python
rank = Knn(query="machine learning research")
```

**Go:**
```go
rank := NewKnnRank(KnnQueryText("machine learning research"))
```

Only the 16 nearest documents get scored (default limit).

---

### Text Query with Custom Parameters

**Python:**
```python
Knn(
    query="What is machine learning?",
    key="#embedding",
    limit=100,
    return_rank=False
)
```

**Go:**
```go
rank := NewKnnRank(
    KnnQueryText("What is machine learning?"),
    WithKnnKey(KEmbedding),  // "#embedding" is the default
    WithKnnLimit(100),
    // return_rank defaults to false, use WithKnnReturnRank() to set true
)
```

---

## Multiple KNN with Document Filtering

### Restrictive Filtering (default=None)

Documents must appear in BOTH top-100 lists to be scored.

**Python:**
```python
rank = Knn(query="research papers", limit=100) + \
       Knn(query="academic publications", limit=100, key="sparse_embedding")
```

**Go:**
```go
rank := NewKnnRank(
    KnnQueryText("research papers"),
    WithKnnLimit(100),
).Add(
    NewKnnRank(
        KnnQueryText("academic publications"),
        WithKnnLimit(100),
        WithKnnKey(K("sparse_embedding")),
    ),
)
```

---

### Inclusive Filtering (with default values)

Documents in either top-100 list can be scored.

**Python:**
```python
rank = (
    Knn(query="machine learning", limit=100, default=10.0) * 0.7 +
    Knn(query="deep learning", limit=100, default=10.0) * 0.3
)
```

**Go:**
```go
rank := NewKnnRank(
    KnnQueryText("machine learning"),
    WithKnnLimit(100),
    WithKnnDefault(10.0),
).Multiply(FloatOperand(0.7)).Add(
    NewKnnRank(
        KnnQueryText("deep learning"),
        WithKnnLimit(100),
        WithKnnDefault(10.0),
    ).Multiply(FloatOperand(0.3)),
)
```

---

## Weighted Combinations

### Dense + Sparse Hybrid Search

Linear weighting of dense and sparse embedding searches.

**Python:**
```python
text_score = Knn(query="machine learning research")
sparse_score = Knn(query=sparse_vector, key="sparse_embedding")
combined = text_score * 0.7 + sparse_score * 0.3
```

**Go:**
```go
import "github.com/amikos-tech/chroma-go/pkg/embeddings"

// Create sparse vector
sparseVector := embeddings.NewSparseVector(
    []int{1, 5, 10, 50},
    []float32{0.5, 0.3, 0.8, 0.2},
)

textScore := NewKnnRank(KnnQueryText("machine learning research"))
sparseScore := NewKnnRank(
    KnnQuerySparseVector(sparseVector),
    WithKnnKey(K("sparse_embedding")),
)

combined := textScore.Multiply(FloatOperand(0.7)).Add(
    sparseScore.Multiply(FloatOperand(0.3)),
)
```

---

### Multi-Query Perspective Blending

Combining different semantic perspectives with asymmetric weights.

**Python:**
```python
general = Knn(query="artificial intelligence overview")
specific = Knn(query="neural network architectures")
multi_query = general * 0.4 + specific * 0.6
```

**Go:**
```go
general := NewKnnRank(KnnQueryText("artificial intelligence overview"))
specific := NewKnnRank(KnnQueryText("neural network architectures"))

multiQuery := general.Multiply(FloatOperand(0.4)).Add(
    specific.Multiply(FloatOperand(0.6)),
)
```

---

## Mathematical Functions

### Exponential Amplification

Amplifying differences between scores.

**Python:**
```python
score = Knn(query="machine learning").exp()
```

**Go:**
```go
score := NewKnnRank(KnnQueryText("machine learning")).Exp()
```

---

### Logarithmic Compression

Compressing score ranges while avoiding log(0).

**Python:**
```python
compressed = (Knn(query="deep learning") + 1).log()
```

**Go:**
```go
compressed := NewKnnRank(KnnQueryText("deep learning")).Add(
    FloatOperand(1),
).Log()
```

---

### Score Clamping

Restricting scores to a defined range [0, 1].

**Python:**
```python
clamped = Knn(query="artificial intelligence").min(0.0).max(1.0)
```

**Go:**
```go
clamped := NewKnnRank(KnnQueryText("artificial intelligence")).
    Min(FloatOperand(0.0)).
    Max(FloatOperand(1.0))
```

---

### Negation

**Python:**
```python
negated = -Knn(query="example")
```

**Go:**
```go
negated := NewKnnRank(KnnQueryText("example")).Negate()
```

---

### Absolute Value

**Python:**
```python
absolute = abs(Knn(query="example"))
```

**Go:**
```go
absolute := NewKnnRank(KnnQueryText("example")).Abs()
```

---

### Division

**Python:**
```python
normalized = Knn(query="example") / 10.0
```

**Go:**
```go
normalized := NewKnnRank(KnnQueryText("example")).Div(FloatOperand(10.0))
```

---

### Subtraction

**Python:**
```python
difference = Knn(query="AI") - Knn(query="ML")
```

**Go:**
```go
difference := NewKnnRank(KnnQueryText("AI")).Sub(
    NewKnnRank(KnnQueryText("ML")),
)
```

---

## Query Format Examples

### Dense Vector Input

Direct vector submission bypassing text embedding.

**Python:**
```python
Knn(query=[0.1, 0.2, 0.3, 0.4])

import numpy as np
embedding = np.array([0.1, 0.2, 0.3, 0.4])
Knn(query=embedding)
```

**Go:**
```go
import "github.com/amikos-tech/chroma-go/pkg/embeddings"

// Using a float32 slice
vector := embeddings.NewEmbeddingFromFloat32([]float32{0.1, 0.2, 0.3, 0.4})
rank := NewKnnRank(KnnQueryVector(vector))
```

---

### Sparse Vector Format

Sparse vector format: dictionary with indices and values.

**Python:**
```python
sparse_vector = {
    "indices": [1, 5, 10, 50],
    "values": [0.5, 0.3, 0.8, 0.2]
}
Knn(query=sparse_vector, key="sparse_embedding")
```

**Go:**
```go
import "github.com/amikos-tech/chroma-go/pkg/embeddings"

sparseVector := embeddings.NewSparseVector(
    []int{1, 5, 10, 50},
    []float32{0.5, 0.3, 0.8, 0.2},
)

rank := NewKnnRank(
    KnnQuerySparseVector(sparseVector),
    WithKnnKey(K("sparse_embedding")),
)
```

---

## Dictionary/JSON Syntax

The Go API produces JSON that matches the Chroma dictionary syntax.

### KNN Dictionary Representation

**Python:**
```python
rank_dict = {
    "$knn": {
        "query": "machine learning research",
        "key": "#embedding",
        "limit": 100,
        "return_rank": False
    }
}
```

**Go:**
```go
rank := NewKnnRank(
    KnnQueryText("machine learning research"),
    WithKnnKey(KEmbedding),
    WithKnnLimit(100),
)

// Serialize to JSON
jsonBytes, _ := rank.MarshalJSON()
// Output: {"$knn":{"key":"#embedding","limit":100,"query":"machine learning research"}}
```

---

### Arithmetic Operations as Dictionary

**Python (Sum):**
```python
sum_dict = {
    "$sum": [
        {"$knn": {"query": "deep learning"}},
        {"$val": 0.5}
    ]
}
```

**Go:**
```go
sumRank := NewKnnRank(KnnQueryText("deep learning")).Add(FloatOperand(0.5))

jsonBytes, _ := sumRank.MarshalJSON()
// Output: {"$sum":[{"$knn":{"key":"#embedding","limit":16,"query":"deep learning"}},{"$val":0.5}]}
```

**Python (Multiply):**
```python
mul_dict = {
    "$mul": [
        {"$knn": {"query": "neural networks"}},
        {"$val": 0.8}
    ]
}
```

**Go:**
```go
mulRank := NewKnnRank(KnnQueryText("neural networks")).Multiply(FloatOperand(0.8))

jsonBytes, _ := mulRank.MarshalJSON()
// Output: {"$mul":[{"$knn":{"key":"#embedding","limit":16,"query":"neural networks"}},{"$val":0.8}]}
```

---

### Complex Expression Structure

Nested operations combining dense and sparse embeddings.

**Python:**
```python
weighted_combo = {
    "$sum": [
        {"$mul": [
            {"$knn": {"query": "machine learning"}},
            {"$val": 0.7}
        ]},
        {"$mul": [
            {"$knn": {"query": "machine learning", "key": "sparse_embedding"}},
            {"$val": 0.3}
        ]}
    ]
}
```

**Go:**
```go
weightedCombo := NewKnnRank(
    KnnQueryText("machine learning"),
).Multiply(FloatOperand(0.7)).Add(
    NewKnnRank(
        KnnQueryText("machine learning"),
        WithKnnKey(K("sparse_embedding")),
    ).Multiply(FloatOperand(0.3)),
)

jsonBytes, _ := weightedCombo.MarshalJSON()
```

---

## Reciprocal Rank Fusion (RRF)

RRF combines multiple ranking strategies using the formula: `-sum(weight_i / (k + rank_i))`

**Go:**
```go
// Basic RRF with equal weights
rrf, _ := NewRrfRank(
    WithRffRanks(
        NewKnnRank(
            KnnQueryText("machine learning"),
            WithKnnReturnRank(),
        ).WithWeight(1.0),
        NewKnnRank(
            KnnQueryText("deep learning"),
            WithKnnReturnRank(),
        ).WithWeight(1.0),
    ),
    WithRffK(60),  // default smoothing constant
)

// RRF with custom weights and normalization
rrf, _ := NewRrfRank(
    WithRffRanks(
        NewKnnRank(
            KnnQueryText("scientific papers"),
            WithKnnLimit(50),
            WithKnnDefault(1000.0),
            WithKnnKey(K("sparse_embedding")),
        ).Multiply(FloatOperand(0.5)).WithWeight(0.5),
        NewKnnRank(
            KnnQueryText("AI research"),
            WithKnnLimit(100),
        ).Multiply(FloatOperand(0.5)).WithWeight(0.5),
    ),
    WithRffK(100),
    WithRffNormalize(),
)
```

---

## Edge Cases

### No Ranking (Index Order)

Results returned in index order (typically insertion order).

**Python:**
```python
search = Search().where(K("status") == "active").limit(10)
```

**Go:**
```go
// Search without ranking - just filtering
col.Search(ctx,
    NewSearchRequest(
        WithFilter(EqString("status", "active")),
        WithPage(WithLimit(10)),
    ),
)
```

---

### Vector Dimension Alignment

Query vectors must match indexed embedding dimensions.

**Go:**
```go
// Error - only 3 dimensions (assuming 384-dim index)
wrongDim := embeddings.NewEmbeddingFromFloat32([]float32{0.1, 0.2, 0.3})

// Correct - 384 dimensions
correctDim := embeddings.NewEmbeddingFromFloat32(make([]float32, 384))
rank := NewKnnRank(KnnQueryVector(correctDim))
```

---

## Complete Real-World Example

**Python:**
```python
from chromadb import Search, K, Knn, Val

search = (Search()
    .where(
        (K("status") == "published") &
        (K("category").is_in(["tech", "science"]))
    )
    .rank(
        (
            Knn(query="latest AI research developments") * 0.7 +
            Knn(query="artificial intelligence breakthroughs") * 0.3
        ).exp()
        .min(0.0)
    )
    .limit(20)
    .select(K.DOCUMENT, K.SCORE, "title", "category")
)
```

**Go:**
```go
import (
    "context"
    v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

// Build the rank expression
rankExpr := v2.NewKnnRank(
    v2.KnnQueryText("latest AI research developments"),
).Multiply(v2.FloatOperand(0.7)).Add(
    v2.NewKnnRank(
        v2.KnnQueryText("artificial intelligence breakthroughs"),
    ).Multiply(v2.FloatOperand(0.3)),
).Exp().Min(v2.FloatOperand(0.0))

// Execute search
result, err := col.Search(ctx,
    v2.NewSearchRequest(
        // Filter: status == "published" AND category IN ["tech", "science"]
        v2.WithFilter(
            v2.And(
                v2.EqString("status", "published"),
                v2.InStrings("category", []string{"tech", "science"}),
            ),
        ),
        // Rank expression with weighted KNN, exp, and min
        v2.WithKnnRank(
            v2.KnnQueryText("latest AI research developments"),
            v2.WithKnnLimit(100),
        ),
        // Pagination
        v2.WithPage(v2.WithLimit(20)),
        // Select specific fields
        v2.WithSelect(v2.KDocument, v2.KScore, v2.K("title"), v2.K("category")),
    ),
)
if err != nil {
    log.Fatal(err)
}
```

---

## Quick Reference

| Python                           | Go                                                             |
|----------------------------------|----------------------------------------------------------------|
| `Knn(query="text")`              | `NewKnnRank(KnnQueryText("text"))`                             |
| `Knn(query=vector)`              | `NewKnnRank(KnnQueryVector(embedding))`                        |
| `Knn(query=sparse, key="field")` | `NewKnnRank(KnnQuerySparseVector(sv), WithKnnKey(K("field")))` |
| `knn.limit=100`                  | `WithKnnLimit(100)`                                            |
| `knn.default=10.0`               | `WithKnnDefault(10.0)`                                         |
| `knn.return_rank=True`           | `WithKnnReturnRank()`                                          |
| `a + b`                          | `a.Add(b)`                                                     |
| `a - b`                          | `a.Sub(b)`                                                     |
| `a * b`                          | `a.Multiply(b)`                                                |
| `a / b`                          | `a.Div(b)`                                                     |
| `-a`                             | `a.Negate()`                                                   |
| `abs(a)`                         | `a.Abs()`                                                      |
| `a.exp()`                        | `a.Exp()`                                                      |
| `a.log()`                        | `a.Log()`                                                      |
| `a.max(b)`                       | `a.Max(b)`                                                     |
| `a.min(b)`                       | `a.Min(b)`                                                     |
| `Val(0.5)`                       | `Val(0.5)` or `FloatOperand(0.5)`                              |
