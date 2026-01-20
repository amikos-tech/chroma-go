# Search API Basics - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/cloud/search-api/search-basics)

## Overview

The Search API provides advanced search capabilities with filtering, ranking, pagination, and field selection. This page covers the basics of constructing search requests.

## Go Examples

### Basic Search Request

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search

search = Search()
result = collection.search(search)
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

	// Basic search request
	result, err := collection.Search(ctx,
		v2.NewSearchRequest(),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Results: %v", result)
}
```
{% /codetab %}
{% /codetabs %}

### Search with All Components

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, K, Knn

search = Search(
    where={"status": "active"},
    rank={"$knn": {"query": [0.1, 0.2]}},
    limit=10,
    select=["#document", "#score"]
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

	// Search with filter, ranking, pagination, and projection
	result, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.EqString("status", "active")),
			v2.WithKnnRank(v2.KnnQueryText("machine learning")),
			v2.WithPage(v2.WithLimit(10)),
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

### Builder Pattern with Method Chaining

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, K, Knn

search = (Search()
    .where(K("status") == "published")
    .rank(Knn(query="machine learning applications"))
    .limit(10)
    .select(K.DOCUMENT, K.SCORE))
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

	// Go uses functional options pattern instead of method chaining
	result, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.EqString("status", "published")),
			v2.WithKnnRank(v2.KnnQueryText("machine learning applications")),
			v2.WithPage(v2.WithLimit(10)),
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

### Common Search Patterns

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

	// Pattern 1: Filter only (no ranking)
	filterOnly, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(
				v2.And(
					v2.EqString("category", "science"),
					v2.GteInt("year", 2023),
				),
			),
			v2.WithPage(v2.WithLimit(10)),
			v2.WithSelect(v2.KDocument, v2.KMetadata),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Pattern 2: Rank only (no filtering)
	rankOnly, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText("AI research")),
			v2.WithPage(v2.WithLimit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Pattern 3: Both filter and rank
	combined, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(
				v2.And(
					v2.EqString("category", "science"),
					v2.GteInt("year", 2023),
				),
			),
			v2.WithKnnRank(v2.KnnQueryText("AI research")),
			v2.WithPage(v2.WithLimit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Filter only: %v", filterOnly)
	log.Printf("Rank only: %v", rankOnly)
	log.Printf("Combined: %v", combined)
}
```
{% /codetab %}
{% /codetabs %}

## Read Level

Control whether searches read from the write-ahead log (WAL) or only the compacted index:

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

	// Default: ReadLevelIndexAndWAL - reads from both index and WAL
	// All committed writes will be visible
	result, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText("machine learning")),
			v2.WithPage(v2.WithLimit(10)),
		),
		v2.WithReadLevel(v2.ReadLevelIndexAndWAL),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// ReadLevelIndexOnly - reads only from compacted index
	// Faster, but recent writes that haven't been compacted may not be visible
	fastResult, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText("machine learning")),
			v2.WithPage(v2.WithLimit(10)),
		),
		v2.WithReadLevel(v2.ReadLevelIndexOnly),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Default result: %v", result)
	log.Printf("Fast result: %v", fastResult)
}
```
{% /codetab %}
{% /codetabs %}

| Read Level | Go Constant | Description |
|------------|-------------|-------------|
| Index + WAL | `v2.ReadLevelIndexAndWAL` | Reads from both index and WAL (default). All committed writes visible. |
| Index Only | `v2.ReadLevelIndexOnly` | Reads only from compacted index. Faster, but recent writes may not be visible. |

## Search Components

| Component | Go Function | Description |
|-----------|-------------|-------------|
| Filter | `WithFilter()` | Narrow down results by metadata |
| Filter IDs | `WithFilterIDs()` | Filter by specific document IDs |
| Filter Document | `WithFilterDocument()` | Filter by document content |
| Rank | `WithKnnRank()`, `WithRank()` | Score and order results |
| Page | `WithPage()` | Pagination with limit/offset |
| Select | `WithSelect()` | Choose which fields to return |
| Group By | `WithGroupBy()` | Group results by metadata |
| Read Level | `WithReadLevel()` | Control WAL vs index-only reads |

## Projection Keys

| Key | Go Constant | Description |
|-----|-------------|-------------|
| `#document` | `v2.KDocument` | Document text |
| `#embedding` | `v2.KEmbedding` | Vector embedding |
| `#score` | `v2.KScore` | Ranking score |
| `#metadata` | `v2.KMetadata` | All metadata fields |
| `#id` | `v2.KID` | Document ID |
| Custom field | `v2.K("field")` | Specific metadata field |

## Iterating Over Results

Use the `Rows()` method for ergonomic result iteration:

```go
results, err := collection.Search(ctx,
    v2.NewSearchRequest(
        v2.WithKnnRank(v2.KnnQueryText("machine learning")),
        v2.WithPage(v2.WithLimit(10)),
        v2.WithSelect(v2.KDocument, v2.KScore, v2.K("title")),
    ),
)
if err != nil {
    log.Fatalf("Error: %v", err)
}

// Iterate using Rows() - no manual index tracking needed
for i, row := range results.Rows() {
    fmt.Printf("%d. ID: %s, Score: %.3f\n", i+1, row.ID, row.Score)
    fmt.Printf("   Document: %s\n", row.Document)
    if row.Metadata != nil {
        if title, ok := row.Metadata.Get("title"); ok {
            fmt.Printf("   Title: %v\n", title)
        }
    }
}

// Or use At() for safe indexed access
if row, ok := results.At(0, 0); ok {
    fmt.Printf("First result: %s\n", row.ID)
}
```

## Notes

- Go uses functional options pattern instead of Python's builder pattern
- Use `NewSearchRequest()` to create a search request
- Combine options like `WithFilter()`, `WithKnnRank()`, `WithPage()`, `WithSelect()`
- Use `Rows()` for easy iteration over results
- Use `RowGroups()` when executing multiple search requests in a batch
- Use `At(group, index)` for safe indexed access with bounds checking
