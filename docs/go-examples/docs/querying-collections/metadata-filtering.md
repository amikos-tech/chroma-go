# Metadata Filtering - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/querying-collections/metadata-filtering)

## Overview

The `where` argument in `get` and `query` is used to filter records by their metadata. Use `v2.K("field")` to clearly mark metadata field names in filter expressions.

## Go Examples

### Basic Equality Filter

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.query(
    query_texts=["first query", "second query"],
    where={"page": 10}
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
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	// Query with metadata filter - page equals 10
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("first query", "second query"),
		v2.WithWhereQuery(v2.EqInt(v2.K("page"), 10)),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	log.Printf("Results: %v", results.GetDocumentsGroups())
}
```
{% /codetab %}
{% /codetabs %}

### Comparison Operators

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.query(
    query_texts=["first query", "second query"],
    where={"page": { "$gt": 10 }}
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
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	// Query with greater than filter
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("first query", "second query"),
		v2.WithWhereQuery(v2.GtInt(v2.K("page"), 10)),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	log.Printf("Results: %v", results.GetDocumentsGroups())
}
```
{% /codetab %}
{% /codetabs %}

### Logical AND Operator

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.query(
    query_texts=["first query", "second query"],
    where={
        "$and": [
            {"page": {"$gte": 5 }},
            {"page": {"$lte": 10 }},
        ]
    }
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
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	// Query with AND filter - page between 5 and 10
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("first query", "second query"),
		v2.WithWhereQuery(
			v2.And(
				v2.GteInt(v2.K("page"), 5),
				v2.LteInt(v2.K("page"), 10),
			),
		),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	log.Printf("Results: %v", results.GetDocumentsGroups())
}
```
{% /codetab %}
{% /codetabs %}

### Logical OR Operator

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.get(
    where={
        "$or": [
            {"color": "red"},
            {"color": "blue"},
        ]
    }
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
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	// Get with OR filter - color is red OR blue
	results, err := collection.Get(ctx,
		v2.WithWhereGet(
			v2.Or(
				v2.EqString(v2.K("color"), "red"),
				v2.EqString(v2.K("color"), "blue"),
			),
		),
	)
	if err != nil {
		log.Fatalf("Error getting: %v", err)
	}

	log.Printf("Results: %v", results.GetDocuments())
}
```
{% /codetab %}
{% /codetabs %}

### Inclusion Operators ($in)

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.get(
    where={
       "author": {"$in": ["Rowling", "Fitzgerald", "Herbert"]}
    }
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
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	// Get with $in filter - author in list
	results, err := collection.Get(ctx,
		v2.WithWhereGet(
			v2.InString(v2.K("author"), "Rowling", "Fitzgerald", "Herbert"),
		),
	)
	if err != nil {
		log.Fatalf("Error getting: %v", err)
	}

	log.Printf("Results: %v", results.GetDocuments())
}
```
{% /codetab %}
{% /codetabs %}

### Exclusion Operators ($nin)

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
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	// Get with $nin filter - author NOT in list
	results, err := collection.Get(ctx,
		v2.WithWhereGet(
			v2.NinString(v2.K("author"), "Unknown", "Anonymous"),
		),
	)
	if err != nil {
		log.Fatalf("Error getting: %v", err)
	}

	log.Printf("Results: %v", results.GetDocuments())
}
```
{% /codetab %}
{% /codetabs %}

### ID Filtering

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Include only specific IDs
collection.get(
    where={"#id": {"$in": ["doc1", "doc2", "doc3"]}}
)

# Exclude specific IDs
collection.get(
    where={"#id": {"$nin": ["excluded1", "excluded2"]}}
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
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	// Include only specific IDs using IDIn
	results1, err := collection.Get(ctx,
		v2.WithWhereGet(v2.IDIn("doc1", "doc2", "doc3")),
	)
	if err != nil {
		log.Fatalf("Error getting: %v", err)
	}

	// Exclude specific IDs using IDNotIn
	results2, err := collection.Get(ctx,
		v2.WithWhereGet(v2.IDNotIn("excluded1", "excluded2")),
	)
	if err != nil {
		log.Fatalf("Error getting: %v", err)
	}

	// Combine ID filter with metadata filter
	results3, err := collection.Query(ctx,
		v2.WithQueryTexts("search query"),
		v2.WithWhereQuery(
			v2.And(
				v2.IDNotIn("seen1", "seen2", "seen3"),
				v2.EqString(v2.K("status"), "active"),
			),
		),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	log.Printf("Results: %v, %v, %v", results1, results2, results3)
}
```
{% /codetab %}
{% /codetabs %}

### Document Content Filtering

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Filter documents containing specific text
collection.get(
    where={"#document": {"$contains": "machine learning"}}
)

# Filter documents NOT containing specific text
collection.get(
    where={"#document": {"$not_contains": "deprecated"}}
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
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	// Filter documents containing specific text
	results1, err := collection.Get(ctx,
		v2.WithWhereGet(v2.DocumentContains("machine learning")),
	)
	if err != nil {
		log.Fatalf("Error getting: %v", err)
	}

	// Filter documents NOT containing specific text
	results2, err := collection.Get(ctx,
		v2.WithWhereGet(v2.DocumentNotContains("deprecated")),
	)
	if err != nil {
		log.Fatalf("Error getting: %v", err)
	}

	// Combine with other filters
	results3, err := collection.Query(ctx,
		v2.WithQueryTexts("AI research"),
		v2.WithWhereQuery(
			v2.And(
				v2.DocumentContains("neural network"),
				v2.EqString(v2.K("category"), "research"),
				v2.GteInt(v2.K("year"), 2023),
			),
		),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	log.Printf("Results: %v, %v, %v", results1, results2, results3)
}
```
{% /codetab %}
{% /codetabs %}

### Combining Metadata and Document Filters

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.query(
    query_texts=["doc10", "thus spake zarathustra"],
    n_results=10,
    where={"metadata_field": "is_equal_to_this"},
    where_document={"$contains":"search_string"}
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
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	// Query with both metadata and document filters
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("doc10", "thus spake zarathustra"),
		v2.WithNResults(10),
		v2.WithWhereQuery(v2.EqString(v2.K("metadata_field"), "is_equal_to_this")),
		v2.WithWhereDocumentQuery(v2.ContainsString("search_string")),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	log.Printf("Results: %v", results.GetDocumentsGroups())
}
```
{% /codetab %}
{% /codetabs %}

## Available Filter Functions

| Python Operator | Go Function | Description |
|----------------|-------------|-------------|
| `{"field": value}` | `v2.EqString(v2.K("field"), value)` | Equal to (string) |
| `{"field": value}` | `v2.EqInt(v2.K("field"), value)` | Equal to (integer) |
| `{"field": value}` | `v2.EqFloat(v2.K("field"), value)` | Equal to (float) |
| `{"$ne": value}` | `v2.NeString(v2.K("field"), value)` | Not equal to |
| `{"$gt": value}` | `v2.GtInt(v2.K("field"), value)` | Greater than |
| `{"$gte": value}` | `v2.GteInt(v2.K("field"), value)` | Greater than or equal |
| `{"$lt": value}` | `v2.LtInt(v2.K("field"), value)` | Less than |
| `{"$lte": value}` | `v2.LteInt(v2.K("field"), value)` | Less than or equal |
| `{"$in": [...]}` | `v2.InString(v2.K("field"), ...)` | Value in list |
| `{"$nin": [...]}` | `v2.NinString(v2.K("field"), ...)` | Value not in list |
| `{"#id": {"$in": [...]}}` | `v2.IDIn(...)` | Include specific document IDs |
| `{"#id": {"$nin": [...]}}` | `v2.IDNotIn(...)` | Exclude specific document IDs |
| `{"#document": {"$contains": ...}}` | `v2.DocumentContains(...)` | Document contains text |
| `{"#document": {"$not_contains": ...}}` | `v2.DocumentNotContains(...)` | Document doesn't contain text |
| `{"$and": [...]}` | `v2.And(...)` | All conditions must match |
| `{"$or": [...]}` | `v2.Or(...)` | Any condition must match |

## Notes

- Use `v2.K("field")` to clearly mark metadata field names in filter expressions
- Filters can be applied to both `Query` and `Get` operations
- Use appropriate type-specific functions (EqString, EqInt, EqFloat)
- Logical operators can be nested for complex queries
- `$in` and `$nin` work with strings, ints, floats, and bools
- `IDIn` and `IDNotIn` are convenience functions for filtering by document IDs
- `DocumentContains` and `DocumentNotContains` filter by document text content
- All filter types can be combined with `And()` and `Or()`
