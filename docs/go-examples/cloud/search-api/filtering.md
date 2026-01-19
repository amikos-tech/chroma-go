# Filtering with Where - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/cloud/search-api/filtering)

## Overview

Filter search results using Where expressions to narrow down your search to specific documents, IDs, or metadata values.

## Go Examples

### Basic Metadata Filtering

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import K

# Filter by metadata field
K("status") == "active"

# Filter by document content
K.DOCUMENT.contains("machine learning")

# Filter by document IDs (include only these)
K.ID.is_in(["doc1", "doc2", "doc3"])

# Filter by document IDs (exclude these)
K.ID.not_in(["excluded1", "excluded2"])
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

	// Filter by metadata field: K("status") == "active"
	result1, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.EqString(v2.K("status"), "active")),
			v2.WithPage(v2.WithLimit(10)),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Filter by document content: K.DOCUMENT.contains(...)
	result2, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.DocumentContains("machine learning")),
			v2.WithPage(v2.WithLimit(10)),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Filter by specific document IDs: K.ID.is_in([...])
	// Option 1: Using WithFilterIDs convenience function
	result3, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilterIDs("doc1", "doc2", "doc3"),
			v2.WithPage(v2.WithLimit(10)),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Option 2: Using IDIn within WithFilter
	result4, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.IDIn("doc1", "doc2", "doc3")),
			v2.WithPage(v2.WithLimit(10)),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Exclude specific document IDs: K.ID.not_in([...])
	result5, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.IDNotIn("excluded1", "excluded2")),
			v2.WithPage(v2.WithLimit(10)),
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Results: %v, %v, %v, %v, %v", result1, result2, result3, result4, result5)
}
```
{% /codetab %}
{% /codetabs %}

### Comparison Operators

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Equality and inequality (all types)
K("status") == "published"     # String equality
K("views") != 0                # Numeric inequality
K("featured") == True          # Boolean equality

# Numeric comparisons (numbers only)
K("price") > 100               # Greater than
K("rating") >= 4.5             # Greater than or equal
K("stock") < 10                # Less than
K("discount") <= 0.25          # Less than or equal
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

	// String equality: K("status") == "published"
	result1, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.EqString(v2.K("status"), "published")),
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	// Numeric inequality: K("views") != 0
	result2, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.NeInt(v2.K("views"), 0)),
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	// Boolean equality: K("featured") == True
	result3, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.EqBool(v2.K("featured"), true)),
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	// Greater than: K("price") > 100
	result4, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.GtFloat(v2.K("price"), 100)),
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	// Greater than or equal: K("rating") >= 4.5
	result5, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.GteFloat(v2.K("rating"), 4.5)),
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	// Less than: K("stock") < 10
	result6, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.LtInt(v2.K("stock"), 10)),
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	// Less than or equal: K("discount") <= 0.25
	result7, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.LteFloat(v2.K("discount"), 0.25)),
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	log.Printf("Results: %v, %v, %v, %v, %v, %v, %v",
		result1, result2, result3, result4, result5, result6, result7)
}
```
{% /codetab %}
{% /codetabs %}

### Set Membership Operators

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Set membership operators (works on all fields)
K.ID.is_in(["doc1", "doc2", "doc3"])           # Match any ID in list
K.ID.not_in(["excluded1", "excluded2"])        # Exclude specific IDs
K("category").is_in(["tech", "science"])       # Match any category
K("status").not_in(["draft", "deleted"])       # Exclude specific values

# String content operators (currently K.DOCUMENT only)
K.DOCUMENT.contains("machine learning")        # Substring search in document
K.DOCUMENT.not_contains("deprecated")          # Exclude documents with text
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

	// K.ID.is_in([...])
	result1, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.IDIn("doc1", "doc2", "doc3")),
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	// K.ID.not_in([...])
	result2, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.IDNotIn("excluded1", "excluded2")),
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	// K("category").is_in(["tech", "science"])
	result3, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.InString(v2.K("category"), "tech", "science")),
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	// K("status").not_in(["draft", "deleted"])
	result4, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.NinString(v2.K("status"), "draft", "deleted")),
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	// K.DOCUMENT.contains(...)
	result5, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.DocumentContains("machine learning")),
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	// K.DOCUMENT.not_contains(...)
	result6, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(v2.DocumentNotContains("deprecated")),
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	log.Printf("Results: %v, %v, %v, %v, %v, %v",
		result1, result2, result3, result4, result5, result6)
}
```
{% /codetab %}
{% /codetabs %}

### Logical Operators (AND/OR)

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# AND operator (&) - all conditions must match
(K("status") == "published") & (K("year") >= 2020)

# OR operator (|) - any condition can match
(K("category") == "tech") | (K("category") == "science")

# Complex nesting - use parentheses for clarity
(
    (K("status") == "published") &
    ((K("category") == "tech") | (K("category") == "science")) &
    (K("rating") >= 4.0)
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

	// AND operator - all conditions must match
	result1, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(
				v2.And(
					v2.EqString(v2.K("status"), "published"),
					v2.GteInt(v2.K("year"), 2020),
				),
			),
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	// OR operator - any condition can match
	result2, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(
				v2.Or(
					v2.EqString(v2.K("category"), "tech"),
					v2.EqString(v2.K("category"), "science"),
				),
			),
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	// Complex nesting with AND and OR
	result3, _ := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(
				v2.And(
					v2.EqString(v2.K("status"), "published"),
					v2.Or(
						v2.EqString(v2.K("category"), "tech"),
						v2.EqString(v2.K("category"), "science"),
					),
					v2.GteFloat(v2.K("rating"), 4.0),
				),
			),
			v2.WithPage(v2.WithLimit(10)),
		),
	)

	log.Printf("Results: %v, %v, %v", result1, result2, result3)
}
```
{% /codetab %}
{% /codetabs %}

### Complete Filter Example

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Search, K, Knn

# Complex filter combining IDs, document content, and metadata
search = (Search()
    .where(
        # Exclude specific documents
        K.ID.not_in(["excluded_001", "excluded_002"]) &

        # Must contain specific content
        K.DOCUMENT.contains("artificial intelligence") &

        # Metadata conditions
        (K("status") == "published") &
        (K("quality_score") >= 0.75) &
        (
            (K("category") == "research") |
            (K("category") == "tutorial")
        ) &
        (K("year") >= 2023)
    )
    .rank(Knn(query="latest AI research developments"))
    .limit(10)
    .select(K.DOCUMENT, "title", "author", "year")
)

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

	// Complex filter combining IDs, document content, and metadata
	result, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithFilter(
				v2.And(
					// K.ID.not_in([...])
					v2.IDNotIn("excluded_001", "excluded_002"),
					// K.DOCUMENT.contains(...)
					v2.DocumentContains("artificial intelligence"),
					// K("status") == "published"
					v2.EqString(v2.K("status"), "published"),
					// K("quality_score") >= 0.75
					v2.GteFloat(v2.K("quality_score"), 0.75),
					// (K("category") == "research") | (K("category") == "tutorial")
					v2.Or(
						v2.EqString(v2.K("category"), "research"),
						v2.EqString(v2.K("category"), "tutorial"),
					),
					// K("year") >= 2023
					v2.GteInt(v2.K("year"), 2023),
				),
			),
			v2.WithKnnRank(
				v2.KnnQueryText("latest AI research developments"),
				v2.WithKnnLimit(100),
			),
			v2.WithPage(v2.WithLimit(10)),
			v2.WithSelect(v2.KDocument, v2.K("title"), v2.K("author"), v2.K("year")),
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

## Filter Functions Reference

| Python Operator | Go Function | Description |
|----------------|-------------|-------------|
| `K("field") == value` | `v2.EqString(v2.K("field"), value)` | Equality |
| `K("field") != value` | `v2.NeString(v2.K("field"), value)` | Inequality |
| `K("field") > value` | `v2.GtInt(v2.K("field"), value)` | Greater than |
| `K("field") >= value` | `v2.GteInt(v2.K("field"), value)` | Greater than or equal |
| `K("field") < value` | `v2.LtInt(v2.K("field"), value)` | Less than |
| `K("field") <= value` | `v2.LteInt(v2.K("field"), value)` | Less than or equal |
| `K("field").is_in([...])` | `v2.InString(v2.K("field"), ...)` | Value in list |
| `K("field").not_in([...])` | `v2.NinString(v2.K("field"), ...)` | Value not in list |
| `K.ID.is_in([...])` | `v2.IDIn(...)` | Include specific document IDs |
| `K.ID.not_in([...])` | `v2.IDNotIn(...)` | Exclude specific document IDs |
| `K.DOCUMENT.contains("text")` | `v2.DocumentContains(...)` | Document contains text |
| `K.DOCUMENT.not_contains("text")` | `v2.DocumentNotContains(...)` | Document doesn't contain |
| `... & ...` | `v2.And(...)` | Logical AND |
| `... \| ...` | `v2.Or(...)` | Logical OR |

## Notes

- Use `v2.K("field")` to clearly mark metadata field names in filter expressions
- String comparisons are case-sensitive
- All filter clauses can be combined using `v2.And()` and `v2.Or()` within `WithFilter()`
- Use `WithFilterIDs()` as a convenience shortcut to restrict search to specific document IDs
- Standard keys are available as constants: `v2.KID`, `v2.KDocument`, `v2.KScore`, `v2.KMetadata`, `v2.KEmbedding`

### Go vs Python Syntax

Python uses `K("field")` for metadata filtering:
```python
K("status") == "active"
K("category").is_in(["tech", "ai"])
```

Go uses `v2.K("field")` with filter functions for the same semantic clarity:
```go
v2.EqString(v2.K("status"), "active")
v2.InString(v2.K("category"), "tech", "ai")
```

For special fields, Go provides dedicated functions that mirror Python's `K.ID` and `K.DOCUMENT`:
```go
v2.IDIn("doc1", "doc2")           // K.ID.is_in([...])
v2.IDNotIn("excluded1")           // K.ID.not_in([...])
v2.DocumentContains("text")       // K.DOCUMENT.contains(...)
v2.DocumentNotContains("text")    // K.DOCUMENT.not_contains(...)
```

