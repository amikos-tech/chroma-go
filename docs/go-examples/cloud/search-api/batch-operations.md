# Batch Operations - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/cloud/search-api/batch-operations)

## Overview

Execute multiple searches in a single API call for better performance and easier comparison of results.

## Go Examples

### Running Multiple Searches

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, K, Knn

searches = [
    # Search 1: Recent articles
    (Search()
        .where((K("type") == "article") & (K("year") >= 2024))
        .rank(Knn(query="machine learning applications"))
        .limit(5)
        .select(K.DOCUMENT, K.SCORE, "title")),

    # Search 2: Papers by specific authors
    (Search()
        .where(K("author").is_in(["Smith", "Jones"]))
        .rank(Knn(query="neural network research"))
        .limit(10)
        .select(K.DOCUMENT, K.SCORE, "title", "author")),

    # Search 3: Featured content
    Search()
        .where(K("status") == "featured")
        .limit(20)
        .select("title", "date")
]

# Execute all searches in one request
results = collection.search(searches)
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

	// Execute multiple searches in one call
	results, err := collection.Search(ctx,
		// Search 1: Recent articles
		v2.NewSearchRequest(
			v2.WithFilter(
				v2.And(
					v2.EqString("type", "article"),
					v2.GteInt("year", 2024),
				),
			),
			v2.WithKnnRank(
				v2.KnnQueryText("machine learning applications"),
				v2.WithKnnLimit(50),
			),
			v2.WithPage(v2.WithLimit(5)),
			v2.WithSelect(v2.KDocument, v2.KScore, v2.K("title")),
		),
		// Search 2: Papers by specific authors
		v2.NewSearchRequest(
			v2.WithFilter(v2.InString("author", "Smith", "Jones")),
			v2.WithKnnRank(
				v2.KnnQueryText("neural network research"),
				v2.WithKnnLimit(50),
			),
			v2.WithPage(v2.WithLimit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore, v2.K("title"), v2.K("author")),
		),
		// Search 3: Featured content (no ranking)
		v2.NewSearchRequest(
			v2.WithFilter(v2.EqString("status", "featured")),
			v2.WithPage(v2.WithLimit(20)),
			v2.WithSelect(v2.K("title"), v2.K("date")),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Access results by index
	log.Printf("Search 1 results: %d", len(results.IDs[0]))
	log.Printf("Search 2 results: %d", len(results.IDs[1]))
	log.Printf("Search 3 results: %d", len(results.IDs[2]))
}
```
{% /codetab %}
{% /codetabs %}

### Accessing Batch Results

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Batch search returns multiple result sets
results = collection.search([search1, search2, search3])

# Access results by index
ids_1 = results.ids[0]    # IDs from search1
ids_2 = results.ids[1]    # IDs from search2
ids_3 = results.ids[2]    # IDs from search3

# Process each search's results
for search_index, ids in enumerate(results.ids):
    print(f"Results from search {search_index + 1}:")
    for i, id in enumerate(ids):
        print(f"  - {id}")
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

	// Execute batch searches
	results, err := collection.Search(ctx,
		v2.NewSearchRequest(v2.WithPage(v2.WithLimit(5))),
		v2.NewSearchRequest(v2.WithPage(v2.WithLimit(10))),
		v2.NewSearchRequest(v2.WithPage(v2.WithLimit(20))),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Access results by index
	ids1 := results.IDs[0] // IDs from search 1
	ids2 := results.IDs[1] // IDs from search 2
	ids3 := results.IDs[2] // IDs from search 3

	// Process each search's results
	for searchIndex, ids := range results.IDs {
		fmt.Printf("Results from search %d:\n", searchIndex+1)
		for _, id := range ids {
			fmt.Printf("  - %s\n", id)
		}
	}

	log.Printf("Total searches: %d (%d, %d, %d)",
		len(results.IDs), len(ids1), len(ids2), len(ids3))
}
```
{% /codetab %}
{% /codetabs %}

### Comparing Different Queries

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Compare different query variations
query_variations = [
    "machine learning",
    "machine learning algorithms and applications",
    "modern machine learning techniques"
]

searches = [
    Search()
        .rank(Knn(query=q))
        .limit(10)
        .select(K.DOCUMENT, K.SCORE, "title")
    for q in query_variations
]

results = collection.search(searches)

# Compare top results from each variation
for i, query_name in enumerate(["Original", "Expanded", "Refined"]):
    print(f"{query_name} Query Top Result:")
    if results.scores[i]:
        print(f"  Score: {results.scores[i][0]:.3f}")
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

	// Compare different query variations
	queryVariations := []string{
		"machine learning",
		"machine learning algorithms and applications",
		"modern machine learning techniques",
	}

	// Build search options for each query
	searchOpts := make([]v2.SearchCollectionOption, len(queryVariations))
	for i, q := range queryVariations {
		searchOpts[i] = v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText(q), v2.WithKnnLimit(50)),
			v2.WithPage(v2.WithLimit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore, v2.K("title")),
		)
	}

	// Execute all at once
	results, err := collection.Search(ctx, searchOpts...)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Compare top results from each variation
	queryNames := []string{"Original", "Expanded", "Refined"}
	for i, queryName := range queryNames {
		fmt.Printf("%s Query Top Result:\n", queryName)
		if results.Scores != nil && len(results.Scores[i]) > 0 {
			fmt.Printf("  Score: %.3f\n", results.Scores[i][0])
		}
	}
}
```
{% /codetab %}
{% /codetabs %}

### Searching Across Categories

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Different category filters
categories = ["technology", "science", "business"]

searches = [
    Search()
        .where(K("category") == category)
        .rank(Knn(query="artificial intelligence"))
        .limit(5)
        .select("title", "category", K.SCORE)
    for category in categories
]

results = collection.search(searches)
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

	// Different category filters
	categories := []string{"technology", "science", "business"}

	// Build search options for each category
	searchOpts := make([]v2.SearchCollectionOption, len(categories))
	for i, category := range categories {
		searchOpts[i] = v2.NewSearchRequest(
			v2.WithFilter(v2.EqString("category", category)),
			v2.WithKnnRank(
				v2.KnnQueryText("artificial intelligence"),
				v2.WithKnnLimit(50),
			),
			v2.WithPage(v2.WithLimit(5)),
			v2.WithSelect(v2.K("title"), v2.K("category"), v2.KScore),
		)
	}

	// Execute all at once
	results, err := collection.Search(ctx, searchOpts...)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Process results by category using RowGroups()
	rowGroups := results.RowGroups()
	for i, category := range categories {
		fmt.Printf("\nCategory: %s\n", category)
		if i >= len(rowGroups) || len(rowGroups[i]) == 0 {
			fmt.Println("  No results found")
			continue
		}
		for j, row := range rowGroups[i] {
			title := ""
			if row.Metadata != nil {
				if t, ok := row.Metadata.Get("title"); ok {
					title = fmt.Sprintf("%v", t)
				}
			}
			fmt.Printf("  %d. %s (Score: %.3f, ID: %s)\n", j+1, title, row.Score, row.ID)
		}
	}
}
```
{% /codetab %}
{% /codetabs %}

### Performance: Batch vs Sequential

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Sequential execution (slow) - multiple API calls
results = []
for search in searches:
    result = collection.search(search)
    results.append(result)

# Batch execution (fast) - single API call
results = collection.search(searches)
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

	queries := []string{"query1", "query2", "query3"}

	// Sequential execution (slow) - multiple API calls
	sequentialResults := make([]*v2.SearchResultImpl, len(queries))
	for i, q := range queries {
		result, _ := collection.Search(ctx,
			v2.NewSearchRequest(
				v2.WithKnnRank(v2.KnnQueryText(q)),
				v2.WithPage(v2.WithLimit(10)),
			),
		)
		sequentialResults[i] = result
	}

	// Batch execution (fast) - single API call
	searchOpts := make([]v2.SearchCollectionOption, len(queries))
	for i, q := range queries {
		searchOpts[i] = v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText(q)),
			v2.WithPage(v2.WithLimit(10)),
		)
	}
	batchResult, _ := collection.Search(ctx, searchOpts...)

	log.Printf("Sequential: %d searches, Batch: %d result sets",
		len(sequentialResults), len(batchResult.IDs))
}
```
{% /codetab %}
{% /codetabs %}

## Iterating Over Batch Results

Use the `RowGroups()` method for ergonomic iteration over batch results:

```go
results, err := collection.Search(ctx, search1, search2, search3)
if err != nil {
    log.Fatalf("Error: %v", err)
}

// Iterate using RowGroups() - each group corresponds to a search request
for i, rows := range results.RowGroups() {
    fmt.Printf("Search %d results:\n", i+1)
    for j, row := range rows {
        fmt.Printf("  %d. ID: %s, Score: %.3f\n", j+1, row.ID, row.Score)
        if row.Metadata != nil {
            if title, ok := row.Metadata.Get("title"); ok {
                fmt.Printf("      Title: %v\n", title)
            }
        }
    }
}

// Or use At(group, index) for safe indexed access
if row, ok := results.At(0, 0); ok {
    fmt.Printf("First result of first search: %s\n", row.ID)
}
```

## Notes

- Batch operations reduce network overhead with a single API call
- Results maintain the same order as input searches
- Each search can have different filters, rankings, and field selections
- Use batch operations for 3-10x speedup compared to sequential execution
- Mixed field selection is supported - each search returns only its requested fields
- Use `RowGroups()` for easy iteration over all search result groups
- Use `At(group, index)` for safe indexed access with bounds checking

