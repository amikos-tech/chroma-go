# Group By & Aggregation - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/cloud/search-api/group-by)

## Overview

Learn how to group search results by metadata keys and select the top results from each group. GroupBy is useful for diversifying results, deduplication, and category-aware ranking.

## Go Examples

### Basic GroupBy

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, K, Knn, GroupBy, MinK

# Get top 3 results per category, ordered by score
search = (Search()
    .rank(Knn(query="machine learning research"))
    .group_by(GroupBy(
        keys=K("category"),
        aggregate=MinK(keys=K.SCORE, k=3)
    ))
    .limit(30)
    .select(K.DOCUMENT, K.SCORE, "category"))

results = collection.search(search)
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

	// Get top 3 results per category, ordered by score
	result, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(
				v2.KnnQueryText("machine learning research"),
				v2.WithKnnLimit(100),
			),
			v2.WithGroupBy(
				v2.NewGroupBy(
					v2.NewMinK(3, v2.KScore),     // Top 3 with lowest scores
					v2.K("category"),             // Group by category
				),
			),
			v2.WithPage(v2.PageLimit(30)),
			v2.WithSelect(v2.KDocument, v2.KScore, v2.K("category")),
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

### MinK Aggregation

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import MinK, K

# Keep 3 records with lowest scores per group
MinK(keys=K.SCORE, k=3)

# Keep 2 records with lowest priority, then lowest score as tiebreaker
MinK(keys=[K("priority"), K.SCORE], k=2)
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

	// Keep 3 records with lowest scores per group
	result1, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText("query"), v2.WithKnnLimit(100)),
			v2.WithGroupBy(
				v2.NewGroupBy(
					v2.NewMinK(3, v2.KScore), // Top 3 lowest scores
					v2.K("category"),
				),
			),
			v2.WithPage(v2.PageLimit(30)),
		),
	)

	// Keep 2 records with lowest priority, then score as tiebreaker
	result2, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText("query"), v2.WithKnnLimit(100)),
			v2.WithGroupBy(
				v2.NewGroupBy(
					v2.NewMinK(2, v2.K("priority"), v2.KScore), // Priority first, then score
					v2.K("category"),
				),
			),
			v2.WithPage(v2.PageLimit(30)),
		),
	)

	log.Printf("Results: %v, %v", result1, result2)
}
```
{% /codetab %}
{% /codetabs %}

### MaxK Aggregation

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import MaxK, K

# Keep 3 records with highest ratings per group
MaxK(keys=K("rating"), k=3)

# Keep 2 records with highest year, then highest rating as tiebreaker
MaxK(keys=[K("year"), K("rating")], k=2)
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

	// Keep 3 records with highest ratings per group
	result1, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText("query"), v2.WithKnnLimit(100)),
			v2.WithGroupBy(
				v2.NewGroupBy(
					v2.NewMaxK(3, v2.K("rating")), // Top 3 highest ratings
					v2.K("category"),
				),
			),
			v2.WithPage(v2.PageLimit(30)),
		),
	)

	// Keep 2 with highest year, then highest rating as tiebreaker
	result2, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText("query"), v2.WithKnnLimit(100)),
			v2.WithGroupBy(
				v2.NewGroupBy(
					v2.NewMaxK(2, v2.K("year"), v2.K("rating")), // Year first, then rating
					v2.K("category"),
				),
			),
			v2.WithPage(v2.PageLimit(30)),
		),
	)

	log.Printf("Results: %v, %v", result1, result2)
}
```
{% /codetab %}
{% /codetabs %}

### Multiple Grouping Keys

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Top 1 article per (category, year) combination
search = (Search()
    .rank(Knn(query="renewable energy"))
    .group_by(GroupBy(
        keys=[K("category"), K("year")],
        aggregate=MinK(keys=K.SCORE, k=1)
    ))
    .limit(30))
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

	// Top 1 article per (category, year) combination
	result, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(
				v2.KnnQueryText("renewable energy"),
				v2.WithKnnLimit(100),
			),
			v2.WithGroupBy(
				v2.NewGroupBy(
					v2.NewMinK(1, v2.KScore),
					v2.K("category"), v2.K("year"), // Multiple keys
				),
			),
			v2.WithPage(v2.PageLimit(30)),
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

### Complete Diversified Search Example

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, K, Knn, GroupBy, MinK

# Diversified product search - ensure results from multiple categories
search = (Search()
    .where(K("in_stock") == True)
    .rank(Knn(query="wireless headphones", limit=100))
    .group_by(GroupBy(
        keys=K("category"),
        aggregate=MinK(keys=K.SCORE, k=2)  # Top 2 per category
    ))
    .limit(20)
    .select(K.DOCUMENT, K.SCORE, "name", "category", "price"))

results = collection.search(search)
rows = results.rows()[0]

for row in rows:
    print(f"{row['metadata']['name']}")
    print(f"  Category: {row['metadata']['category']}")
    print(f"  Price: ${row['metadata']['price']:.2f}")
    print(f"  Score: {row['score']:.3f}")
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

	// Diversified product search - ensure results from multiple categories
	result, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.EqBool("in_stock", true)),
			v2.WithKnnRank(
				v2.KnnQueryText("wireless headphones"),
				v2.WithKnnLimit(100),
			),
			v2.WithGroupBy(
				v2.NewGroupBy(
					v2.NewMinK(2, v2.KScore), // Top 2 per category
					v2.K("category"),
				),
			),
			v2.WithPage(v2.PageLimit(20)),
			v2.WithSelect(v2.KDocument, v2.KScore, v2.K("name"), v2.K("category"), v2.K("price")),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Process results using Rows()
	for _, row := range result.Rows() {
		name := ""
		category := ""
		price := 0.0
		if row.Metadata != nil {
			if n, ok := row.Metadata.Get("name"); ok {
				name = fmt.Sprintf("%v", n)
			}
			if c, ok := row.Metadata.Get("category"); ok {
				category = fmt.Sprintf("%v", c)
			}
			if p, ok := row.Metadata.Get("price"); ok {
				price = p.(float64)
			}
		}
		fmt.Printf("%s\n", name)
		fmt.Printf("  Category: %s\n", category)
		fmt.Printf("  Price: $%.2f\n", price)
		fmt.Printf("  Score: %.3f\n\n", row.Score)
	}
}
```
{% /codetab %}
{% /codetabs %}

## GroupBy Reference

| Python | Go Function | Description |
|--------|-------------|-------------|
| `GroupBy(keys=..., aggregate=...)` | `v2.NewGroupBy(aggregate, keys...)` | Create GroupBy with keys and aggregation |
| `MinK(keys=..., k=N)` | `v2.NewMinK(N, keys...)` | Keep N records with smallest values |
| `MaxK(keys=..., k=N)` | `v2.NewMaxK(N, keys...)` | Keep N records with largest values |
| `K.SCORE` | `v2.KScore` | Reference the search score |
| `K("field")` | `v2.K("field")` | Reference a metadata field |

## Notes

- GroupBy requires a ranking expression (KNN)
- Use `MinK` with `KScore` for distance-based scoring (lower is better)
- Use `MaxK` for metrics where higher is better (ratings, popularity)
- Groups with fewer than k records return all available records
- Documents missing the grouping key are grouped together as null
- Set KNN limit high enough to include candidates from all groups
- Use `Rows()` for ergonomic iteration over results
- Use `At(group, index)` for safe indexed access with bounds checking

