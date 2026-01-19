# Pagination & Field Selection - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/cloud/search-api/pagination-selection)

## Overview

Control how many results to return and which fields to include in your search results.

## Go Examples

### Pagination with Limit and Offset

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search

# Limit results
search = Search().limit(10)  # Return top 10 results

# Pagination with offset
search = Search().limit(10, offset=20)  # Skip first 20, return next 10
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

	// Limit results - return top 10
	result1, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithPage(v2.WithLimit(10)),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Pagination with offset - skip first 20, return next 10
	result2, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithPage(
				v2.WithLimit(10),
				v2.WithOffset(20),
			),
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

### Pagination Patterns

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Page through results (0-indexed)
page_size = 10

# Page 0: Results 1-10
page_0 = Search().limit(page_size, offset=0)

# Page 1: Results 11-20
page_1 = Search().limit(page_size, offset=10)

# Page 2: Results 21-30
page_2 = Search().limit(page_size, offset=20)

# General formula
def get_page(page_number, page_size=10):
    return Search().limit(page_size, offset=page_number * page_size)
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

	pageSize := 10

	// Page 0: Results 1-10
	page0, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithPage(v2.WithLimit(pageSize), v2.WithOffset(0)),
		),
	)

	// Page 1: Results 11-20
	page1, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithPage(v2.WithLimit(pageSize), v2.WithOffset(10)),
		),
	)

	// Page 2: Results 21-30
	page2, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithPage(v2.WithLimit(pageSize), v2.WithOffset(20)),
		),
	)

	// General formula for page N
	getPage := func(pageNumber int, pageSize int) v2.SearchCollectionOption {
		return v2.NewSearchRequest(
			v2.WithPage(
				v2.WithLimit(pageSize),
				v2.WithOffset(pageNumber*pageSize),
			),
		)
	}

	// Get page 5
	page5, _ := collection.Search(ctx, getPage(5, 10))

	log.Printf("Pages: %v, %v, %v, %v", page0, page1, page2, page5)
}
```
{% /codetab %}
{% /codetabs %}

### Field Selection with Select

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, K

# Default - returns IDs only
search = Search()

# Select specific fields
search = Search().select(K.DOCUMENT, K.SCORE)

# Select metadata fields
search = Search().select("title", "author", "date")

# Mix predefined and metadata fields
search = Search().select(K.DOCUMENT, K.SCORE, "title", "author")

# Select all available fields
search = Search().select_all()
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

	// Default - returns IDs only
	result1, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	// Select specific predefined fields
	result2, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText("query")),
			v2.WithPage(v2.WithLimit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore),
		),
	)

	// Select metadata fields using K()
	result3, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithPage(v2.WithLimit(10)),
			v2.WithSelect(v2.K("title"), v2.K("author"), v2.K("date")),
		),
	)

	// Mix predefined and custom metadata fields
	result4, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText("query")),
			v2.WithPage(v2.WithLimit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore, v2.K("title"), v2.K("author")),
		),
	)

	// Select all available fields
	result5, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText("query")),
			v2.WithPage(v2.WithLimit(10)),
			v2.WithSelectAll(),
		),
	)

	log.Printf("Results: %v, %v, %v, %v, %v",
		result1, result2, result3, result4, result5)
}
```
{% /codetab %}
{% /codetabs %}

### Performance-Optimized Selection

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Fast - minimal data
search = Search().limit(100)  # IDs only

# Moderate - just what you need
search = Search().limit(100).select(K.SCORE, "title", "date")

# Slower - large fields
search = Search().limit(100).select(K.DOCUMENT, K.EMBEDDING)

# Slowest - everything
search = Search().limit(100).select_all()
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

	// Fast - minimal data (IDs only)
	fast, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithPage(v2.WithLimit(100)),
		),
	)

	// Moderate - just what you need
	moderate, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText("query")),
			v2.WithPage(v2.WithLimit(100)),
			v2.WithSelect(v2.KScore, v2.K("title"), v2.K("date")),
		),
	)

	// Slower - large fields
	slower, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText("query")),
			v2.WithPage(v2.WithLimit(100)),
			v2.WithSelect(v2.KDocument, v2.KEmbedding),
		),
	)

	// Slowest - everything
	slowest, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText("query")),
			v2.WithPage(v2.WithLimit(100)),
			v2.WithSelectAll(),
		),
	)

	log.Printf("Results: %d, %d, %d, %d",
		len(fast.IDs[0]),
		len(moderate.IDs[0]),
		len(slower.IDs[0]),
		len(slowest.IDs[0]),
	)
}
```
{% /codetab %}
{% /codetabs %}

### Complete Pagination Example

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, K, Knn

def search_with_pagination(collection, query_text, page_size=20):
    current_page = 0

    while True:
        search = (Search()
            .where(K("status") == "published")
            .rank(Knn(query=query_text))
            .limit(page_size, offset=current_page * page_size)
            .select(K.DOCUMENT, K.SCORE, "title", "author", "date")
        )

        results = collection.search(search)
        rows = results.rows()[0]

        if not rows:
            break

        print(f"\n--- Page {current_page + 1} ---")
        for i, row in enumerate(rows, 1):
            print(f"{i}. {row['metadata']['title']}")

        current_page += 1
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

func searchWithPagination(
	ctx context.Context,
	collection v2.CollectionAPI,
	queryText string,
	pageSize int,
) error {
	currentPage := 0

	for {
		result, err := collection.Search(ctx,
			v2.NewSearchRequest(
				v2.WithFilter(v2.EqString("status", "published")),
				v2.WithKnnRank(
					v2.KnnQueryText(queryText),
					v2.WithKnnLimit(100),
				),
				v2.WithPage(
					v2.WithLimit(pageSize),
					v2.WithOffset(currentPage*pageSize),
				),
				v2.WithSelect(v2.KDocument, v2.KScore, v2.K("title"), v2.K("author"), v2.K("date")),
			),
		)
		if err != nil {
			return err
		}

		// Check if we have results using Rows()
		rows := result.Rows()
		if len(rows) == 0 {
			break
		}

		fmt.Printf("\n--- Page %d ---\n", currentPage+1)
		for i, row := range rows {
			title := ""
			if row.Metadata != nil {
				if t, ok := row.Metadata.Get("title"); ok {
					title = fmt.Sprintf("%v", t)
				}
			}
			fmt.Printf("%d. %s (ID: %s)\n", i+1, title, row.ID)
		}

		currentPage++

		// Stop if we got fewer results than requested (last page)
		if len(rows) < pageSize {
			break
		}
	}

	return nil
}

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

	err = searchWithPagination(ctx, collection, "machine learning", 10)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
```
{% /codetab %}
{% /codetabs %}

## Selectable Fields Reference

| Python | Go Constant | Description |
|--------|-------------|-------------|
| `K.DOCUMENT` | `v2.KDocument` | Full document text |
| `K.EMBEDDING` | `v2.KEmbedding` | Vector embeddings |
| `K.SCORE` | `v2.KScore` | Ranking score |
| `K.METADATA` | `v2.KMetadata` | All metadata fields |
| `K.ID` | `v2.KID` | Document ID |
| `"field_name"` | `v2.K("field_name")` | Specific metadata field |

## Pagination Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | None | Maximum results to return |
| `offset` | int | 0 | Number of results to skip |

## Notes

- Pagination uses 0-based indexing (first page is page 0)
- Select only needed fields to improve performance
- IDs are always included in results
- Use `WithSelectAll()` sparingly - only when you need all fields
- Avoid selecting embeddings unless necessary (large data transfer)
- Use `Rows()` for ergonomic iteration over results
- Use `At(group, index)` for safe indexed access with bounds checking

