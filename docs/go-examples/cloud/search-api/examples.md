# Examples & Patterns - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/cloud/search-api/examples)

## Overview

Complete end-to-end examples demonstrating real-world use cases of the Search API.

## E-commerce Product Search

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, K, Knn, And

def search_products(collection, user_query, min_price=None, max_price=None,
                   category=None, in_stock_only=True, page=0, page_size=20):
    # Build filter conditions
    combined_filter = And([])

    if in_stock_only:
        combined_filter &= K("in_stock") == True

    if category:
        combined_filter &= K("category") == category

    if min_price is not None:
        combined_filter &= K("price") >= min_price

    if max_price is not None:
        combined_filter &= K("price") <= max_price

    # Build search
    search = (Search()
        .where(combined_filter)
        .rank(Knn(query=user_query))
        .limit(page_size, offset=page * page_size)
        .select(K.DOCUMENT, K.SCORE, "name", "price", "category", "rating"))

    results = collection.search(search)
    return results.rows()[0]
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

type ProductSearchOptions struct {
	UserQuery   string
	MinPrice    *float64
	MaxPrice    *float64
	Category    string
	InStockOnly bool
	Page        int
	PageSize    int
}

func searchProducts(
	ctx context.Context,
	collection v2.CollectionAPI,
	opts ProductSearchOptions,
) (*v2.SearchResultImpl, error) {
	// Build filter conditions
	var filters []v2.WhereClause

	if opts.InStockOnly {
		filters = append(filters, v2.EqBool("in_stock", true))
	}

	if opts.Category != "" {
		filters = append(filters, v2.EqString("category", opts.Category))
	}

	if opts.MinPrice != nil {
		filters = append(filters, v2.GteFloat("price", *opts.MinPrice))
	}

	if opts.MaxPrice != nil {
		filters = append(filters, v2.LteFloat("price", *opts.MaxPrice))
	}

	// Build search options
	searchOpts := []v2.SearchOption{
		v2.WithKnnRank(
			v2.KnnQueryText(opts.UserQuery),
			v2.WithKnnLimit(100),
		),
		v2.WithPage(
			v2.WithLimit(opts.PageSize),
			v2.WithOffset(opts.Page*opts.PageSize),
		),
		v2.WithSelect(v2.KDocument, v2.KScore, v2.K("name"), v2.K("price"), v2.K("category"), v2.K("rating")),
	}

	// Add combined filter if we have conditions
	if len(filters) > 0 {
		searchOpts = append([]v2.SearchOption{v2.WithFilter(v2.And(filters...))}, searchOpts...)
	}

	return collection.Search(ctx, v2.NewSearchRequest(searchOpts...))
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
	collection, err := client.GetCollection(ctx, "products")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Search for products
	minPrice := 50.0
	maxPrice := 300.0
	results, err := searchProducts(ctx, collection, ProductSearchOptions{
		UserQuery:   "noise cancelling headphones for travel",
		MinPrice:    &minPrice,
		MaxPrice:    &maxPrice,
		Category:    "electronics",
		InStockOnly: true,
		Page:        0,
		PageSize:    20,
	})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Display results using the ergonomic Rows() API
	for i, row := range results.Rows() {
		name := ""
		price := 0.0
		rating := 0.0
		if row.Metadata != nil {
			if n, ok := row.Metadata.Get("name"); ok {
				name = fmt.Sprintf("%v", n)
			}
			if p, ok := row.Metadata.Get("price"); ok {
				price = p.(float64)
			}
			if r, ok := row.Metadata.Get("rating"); ok {
				rating = r.(float64)
			}
		}

		fmt.Printf("%d. %s\n", i+1, name)
		fmt.Printf("   Price: $%.2f | Rating: %.1f/5\n", price, rating)
		fmt.Printf("   Relevance: %.3f\n\n", row.Score)
	}
}
```
{% /codetab %}
{% /codetabs %}

## Content Recommendation System

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, K, Knn, Rrf

def get_recommendations(collection, user_preferences, seen_content_ids, num_recommendations=10):
    # Build filter to exclude seen content
    combined_filter = K.ID.not_in(seen_content_ids)

    # Filter by preferred categories
    if user_preferences.get("categories"):
        combined_filter &= K("category").is_in(user_preferences["categories"])

    # Filter by minimum rating
    min_rating = user_preferences.get("min_rating", 3.5)
    combined_filter &= K("rating") >= min_rating

    # Create hybrid search with multiple signals
    user_interest_query = " ".join(user_preferences.get("interests", ["general"]))
    favorite_topics_query = " ".join(user_preferences.get("favorite_topics", []))

    hybrid_rank = Rrf(
        ranks=[
            Knn(query=user_interest_query, return_rank=True, limit=200),
            Knn(query=favorite_topics_query, return_rank=True, limit=200)
        ],
        weights=[0.6, 0.4],
        k=60
    )

    search = (Search()
        .where(combined_filter)
        .rank(hybrid_rank)
        .limit(num_recommendations)
        .select(K.DOCUMENT, K.SCORE, "title", "category", "author", "rating"))

    return collection.search(search)
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

type UserPreferences struct {
	Interests      []string
	FavoriteTopics []string
	Categories     []string
	MinRating      float64
}

func getRecommendations(
	ctx context.Context,
	collection v2.CollectionAPI,
	prefs UserPreferences,
	seenContentIDs []string,
	numRecommendations int,
) (*v2.SearchResultImpl, error) {
	// Build filter conditions
	var filters []v2.WhereClause

	// Filter by preferred categories
	if len(prefs.Categories) > 0 {
		filters = append(filters, v2.InString("category", prefs.Categories...))
	}

	// Filter by minimum rating
	minRating := prefs.MinRating
	if minRating == 0 {
		minRating = 3.5
	}
	filters = append(filters, v2.GteFloat("rating", minRating))

	// Create hybrid search with multiple signals
	interests := prefs.Interests
	if len(interests) == 0 {
		interests = []string{"general"}
	}
	userInterestQuery := strings.Join(interests, " ")
	favoriteTopicsQuery := strings.Join(prefs.FavoriteTopics, " ")

	// Create KNN ranks for RRF
	interestKnn, _ := v2.NewKnnRank(
		v2.KnnQueryText(userInterestQuery),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(200),
	)

	topicsKnn, _ := v2.NewKnnRank(
		v2.KnnQueryText(favoriteTopicsQuery),
		v2.WithKnnReturnRank(),
		v2.WithKnnLimit(200),
	)

	// Build search
	searchOpts := []v2.SearchOption{
		v2.WithRffRank(
			v2.WithRffRanks(
				interestKnn.WithWeight(0.6),
				topicsKnn.WithWeight(0.4),
			),
			v2.WithRffK(60),
		),
		v2.WithPage(v2.WithLimit(numRecommendations)),
		v2.WithSelect(v2.KDocument, v2.KScore, v2.K("title"), v2.K("category"), v2.K("author"), v2.K("rating")),
	}

	if len(filters) > 0 {
		searchOpts = append([]v2.SearchOption{v2.WithFilter(v2.And(filters...))}, searchOpts...)
	}

	// Add ID filter for seen content (using FilterIDs is for inclusion, so we handle exclusion differently)
	// Note: ID exclusion might need to be handled at the filter level if supported

	return collection.Search(ctx, v2.NewSearchRequest(searchOpts...))
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
	collection, err := client.GetCollection(ctx, "content")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	prefs := UserPreferences{
		Interests:      []string{"machine learning", "artificial intelligence", "data science"},
		FavoriteTopics: []string{"neural networks", "deep learning", "transformers"},
		Categories:     []string{"technology", "science", "research"},
		MinRating:      4.0,
	}

	seenContent := []string{"content_001", "content_045", "content_123"}

	results, err := getRecommendations(ctx, collection, prefs, seenContent, 10)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Println("Personalized Recommendations:")
	for i, row := range results.Rows() {
		title := ""
		category := ""
		author := ""
		rating := 0.0
		if row.Metadata != nil {
			if t, ok := row.Metadata.Get("title"); ok {
				title = fmt.Sprintf("%v", t)
			}
			if c, ok := row.Metadata.Get("category"); ok {
				category = fmt.Sprintf("%v", c)
			}
			if a, ok := row.Metadata.Get("author"); ok {
				author = fmt.Sprintf("%v", a)
			}
			if r, ok := row.Metadata.Get("rating"); ok {
				rating = r.(float64)
			}
		}

		fmt.Printf("\n%d. %s\n", i+1, title)
		fmt.Printf("   Category: %s | Author: %s\n", category, author)
		fmt.Printf("   Rating: %.1f/5 | Match Score: %.3f\n", rating, row.Score)
	}
}
```
{% /codetab %}
{% /codetabs %}

## Multi-Category Search with Batch Operations

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, K, Knn

def search_across_categories(collection, user_query, categories, results_per_category=5):
    # Build a search for each category
    searches = [
        (Search()
            .where(K("category") == category)
            .rank(Knn(query=user_query))
            .limit(results_per_category)
            .select(K.DOCUMENT, K.SCORE, "title", "category", "date"))
        for category in categories
    ]

    # Execute all searches in one batch
    results = collection.search(searches)

    # Process results by category
    category_results = {}
    for i, category in enumerate(categories):
        rows = results.rows()[i]
        category_results[category] = rows

    return category_results
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

type CategoryResult struct {
	Title string
	Date  string
	Score float64
}

func searchAcrossCategories(
	ctx context.Context,
	collection v2.CollectionAPI,
	userQuery string,
	categories []string,
	resultsPerCategory int,
) (map[string][]CategoryResult, error) {
	// Build a search for each category
	searchOpts := make([]v2.SearchCollectionOption, len(categories))
	for i, category := range categories {
		searchOpts[i] = v2.NewSearchRequest(
			v2.WithFilter(v2.EqString("category", category)),
			v2.WithKnnRank(
				v2.KnnQueryText(userQuery),
				v2.WithKnnLimit(50),
			),
			v2.WithPage(v2.WithLimit(resultsPerCategory)),
			v2.WithSelect(v2.KDocument, v2.KScore, v2.K("title"), v2.K("category"), v2.K("date")),
		)
	}

	// Execute all searches in one batch
	results, err := collection.Search(ctx, searchOpts...)
	if err != nil {
		return nil, err
	}

	// Process results by category using RowGroups() for batch operations
	categoryResults := make(map[string][]CategoryResult)
	rowGroups := results.RowGroups()
	for i, category := range categories {
		var items []CategoryResult
		for _, row := range rowGroups[i] {
			title := ""
			date := ""
			if row.Metadata != nil {
				if t, ok := row.Metadata.Get("title"); ok {
					title = fmt.Sprintf("%v", t)
				}
				if d, ok := row.Metadata.Get("date"); ok {
					date = fmt.Sprintf("%v", d)
				}
			}
			items = append(items, CategoryResult{
				Title: title,
				Date:  date,
				Score: row.Score,
			})
		}
		categoryResults[category] = items
	}

	return categoryResults, nil
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
	collection, err := client.GetCollection(ctx, "articles")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	query := "latest developments in renewable energy"
	categories := []string{"technology", "science", "news", "research"}

	results, err := searchAcrossCategories(ctx, collection, query, categories, 3)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Display results
	for category, items := range results {
		fmt.Printf("\n%s\n", strings.Repeat("=", 60))
		fmt.Printf("Category: %s\n", strings.ToUpper(category))
		fmt.Printf("%s\n", strings.Repeat("=", 60))

		if len(items) == 0 {
			fmt.Println("  No results found")
			continue
		}

		for i, item := range items {
			fmt.Printf("\n  %d. %s\n", i+1, item.Title)
			fmt.Printf("     Date: %s\n", item.Date)
			fmt.Printf("     Relevance: %.3f\n", item.Score)
		}
	}
}
```
{% /codetab %}
{% /codetabs %}

## Ergonomic Result Handling

The Go client provides `Rows()` and `RowGroups()` methods for ergonomic result iteration, eliminating manual index tracking.

### ResultRow Structure

```go
type ResultRow struct {
    ID        DocumentID       // Document ID
    Document  string           // Document text (if included)
    Metadata  DocumentMetadata // Metadata map (if included)
    Embedding []float32        // Embedding vector (if included)
    Score     float64          // Relevance score
}
```

### Single Search: Rows()

```go
// For a single search request, use Rows()
results, _ := collection.Search(ctx, v2.NewSearchRequest(...))

for i, row := range results.Rows() {
    fmt.Printf("ID: %s, Score: %.3f\n", row.ID, row.Score)
    if row.Metadata != nil {
        if title, ok := row.Metadata.Get("title"); ok {
            fmt.Printf("Title: %v\n", title)
        }
    }
}
```

### Batch Search: RowGroups()

```go
// For batch operations with multiple search requests, use RowGroups()
results, _ := collection.Search(ctx,
    v2.NewSearchRequest(...), // First search
    v2.NewSearchRequest(...), // Second search
)

for groupIdx, rows := range results.RowGroups() {
    fmt.Printf("Search %d results:\n", groupIdx+1)
    for _, row := range rows {
        fmt.Printf("  ID: %s, Score: %.3f\n", row.ID, row.Score)
    }
}
```

### Safe Index Access: At()

```go
// For safe indexed access with bounds checking
if row, ok := results.At(0, 5); ok {
    fmt.Printf("Row 5 of search 0: %s\n", row.ID)
}
```

## Best Practices

Based on these examples:

1. **Use Rows() for iteration** - The `Rows()` method provides clean iteration without manual index tracking
2. **Use RowGroups() for batch results** - When using multiple search requests, `RowGroups()` returns results grouped by search
3. **Build filters incrementally** - Construct complex filters by combining simpler conditions
4. **Use batch operations** - When searching multiple variations, use batch operations for better performance
5. **Select only needed fields** - Reduce data transfer by selecting only the fields you'll use
6. **Handle empty results gracefully** - Always check if results exist before processing
7. **Use hybrid search for personalization** - Combine multiple ranking signals with RRF for better recommendations
8. **Paginate large result sets** - Use limit and offset for efficient pagination

## Notes

- Go uses functional options pattern for building search requests
- Use `Rows()` for single search results, `RowGroups()` for batch operations
- Use `At(group, index)` for safe indexed access with bounds checking
- Error handling is explicit in Go - always check returned errors
- Batch operations significantly reduce API call overhead

