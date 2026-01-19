# Full Text Search and Regex - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/querying-collections/full-text-search)

## Overview

The `where_document` argument in `get` and `query` is used to filter records based on their document content. Chroma supports full-text search with `$contains`/`$not_contains` operators and regex pattern matching with `$regex`/`$not_regex` operators.

## Go Examples

### Contains Search

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.get(
   where_document={"$contains": "search string"}
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

	// Get documents containing a specific string
	results, err := collection.Get(ctx,
		v2.WithWhereDocumentGet(v2.ContainsString("search string")),
	)
	if err != nil {
		log.Fatalf("Error getting: %v", err)
	}

	log.Printf("Documents: %v", results.GetDocuments())
}
```
{% /codetab %}
{% /codetabs %}

### Regex Pattern Matching

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.get(
   where_document={
       "$regex": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$"
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

	// Get documents matching email regex pattern
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	results, err := collection.Get(ctx,
		v2.WithWhereDocumentGet(v2.MatchesRegex(emailRegex)),
	)
	if err != nil {
		log.Fatalf("Error getting: %v", err)
	}

	log.Printf("Documents matching email pattern: %v", results.GetDocuments())
}
```
{% /codetab %}
{% /codetabs %}

### Logical AND with Document Filters

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.query(
    query_texts=["query1", "query2"],
    where_document={
        "$and": [
            {"$contains": "search_string_1"},
            {"$regex": "[a-z]+"},
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

	// Query with AND document filter
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("query1", "query2"),
		v2.WithWhereDocumentQuery(
			v2.AndDocument(
				v2.ContainsString("search_string_1"),
				v2.MatchesRegex("[a-z]+"),
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

### Logical OR with Document Filters

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.query(
    query_texts=["query1", "query2"],
    where_document={
        "$or": [
            {"$contains": "search_string_1"},
            {"$not_contains": "search_string_2"},
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

	// Query with OR document filter
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("query1", "query2"),
		v2.WithWhereDocumentQuery(
			v2.OrDocument(
				v2.ContainsString("search_string_1"),
				v2.NotContainsString("search_string_2"),
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

	// Combined metadata and document filters
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("doc10", "thus spake zarathustra"),
		v2.WithNResults(10),
		v2.WithWhereQuery(v2.EqString("metadata_field", "is_equal_to_this")),
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

## Available Document Filter Functions

| Python Operator | Go Function | Description |
|----------------|-------------|-------------|
| `{"$contains": "text"}` | `v2.ContainsString("text")` | Document contains text |
| `{"$not_contains": "text"}` | `v2.NotContainsString("text")` | Document does not contain text |
| `{"$regex": "pattern"}` | `v2.MatchesRegex("pattern")` | Document matches regex |
| `{"$not_regex": "pattern"}` | `v2.NotMatchesRegex("pattern")` | Document does not match regex |
| `{"$and": [...]}` | `v2.AndDocument(...)` | All conditions must match |
| `{"$or": [...]}` | `v2.OrDocument(...)` | Any condition must match |

## Notes

- Full-text search is case-sensitive
- Regex patterns use standard regex syntax
- Document filters can be combined with metadata filters
- Use `Query` for semantic search + filtering, `Get` for filtering only
