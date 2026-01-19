# Query and Get - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/querying-collections/query-and-get)

## Overview

Query a Chroma collection to run similarity searches using `.Query`, or retrieve records directly using `.Get`.

## Query Examples

### Basic Query with Text

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.query(
    query_texts=["thus spake zarathustra", "the oracle speaks"]
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

	// Query with text - Chroma embeds the query for you
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("thus spake zarathustra", "the oracle speaks"),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	// Results are indexed by query
	// results.GetIDsGroups()[0] contains IDs for first query
	// results.GetIDsGroups()[1] contains IDs for second query
	log.Printf("Query 1 IDs: %v", results.GetIDsGroups()[0])
	log.Printf("Query 2 IDs: %v", results.GetIDsGroups()[1])
}
```
{% /codetab %}
{% /codetabs %}

### Query with Embeddings

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.query(
    query_embeddings=[[11.1, 12.1, 13.1], [1.1, 2.3, 3.2]]
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

	// Query with pre-computed embeddings
	results, err := collection.Query(ctx,
		v2.WithQueryEmbeddings(
			[]float32{11.1, 12.1, 13.1},
			[]float32{1.1, 2.3, 3.2},
		),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	log.Printf("Results: %v", results.GetIDsGroups())
}
```
{% /codetab %}
{% /codetabs %}

### Query with N Results

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.query(
    query_embeddings=[[11.1, 12.1, 13.1], [1.1, 2.3, 3.2]],
    n_results=5
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

	// Query with custom number of results (default is 10)
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("search query"),
		v2.WithNResults(5),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	log.Printf("Got %d results", len(results.GetIDsGroups()[0]))
}
```
{% /codetab %}
{% /codetabs %}

### Query with ID Constraint

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.query(
    query_embeddings=[[11.1, 12.1, 13.1], [1.1, 2.3, 3.2]],
    n_results=5,
    ids=["id1", "id2"]
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

	// Query constrained to specific IDs
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("search query"),
		v2.WithNResults(5),
		v2.WithIDsQuery("id1", "id2"),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	log.Printf("Results: %v", results.GetIDsGroups())
}
```
{% /codetab %}
{% /codetabs %}

### Query with Metadata and Document Filters

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.query(
    query_embeddings=[[11.1, 12.1, 13.1], [1.1, 2.3, 3.2]],
    n_results=5,
    where={"page": 10},
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

	// Query with metadata filter and document filter
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("search query"),
		v2.WithNResults(5),
		v2.WithWhereQuery(v2.EqInt("page", 10)),
		v2.WithWhereDocumentQuery(v2.ContainsString("search string")),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	log.Printf("Filtered results: %v", results.GetDocumentsGroups())
}
```
{% /codetab %}
{% /codetabs %}

## Get Examples

### Get by IDs

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.get(ids=["id1", "id2"])
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

	// Get records by IDs
	results, err := collection.Get(ctx,
		v2.WithIDsGet("id1", "id2"),
	)
	if err != nil {
		log.Fatalf("Error getting documents: %v", err)
	}

	log.Printf("IDs: %v", results.GetIDs())
	log.Printf("Documents: %v", results.GetDocuments())
}
```
{% /codetab %}
{% /codetabs %}

### Get with Pagination

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

	// Get with limit and offset for pagination
	results, err := collection.Get(ctx,
		v2.WithLimitGet(50),
		v2.WithOffsetGet(100),
	)
	if err != nil {
		log.Fatalf("Error getting documents: %v", err)
	}

	log.Printf("Got %d documents", len(results.GetIDs()))
}
```
{% /codetab %}
{% /codetabs %}

## Choosing What Data is Returned

### Include Specific Fields

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.query(query_texts=["my query"]) # 'ids', 'documents', and 'metadatas' are returned

collection.get(include=["documents"]) # Only 'ids' and 'documents' are returned

collection.query(
    query_texts=["my query"],
    include=["documents", "metadatas", "embeddings"]
) # 'ids', 'documents', 'metadatas', and 'embeddings' are returned
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

	// Default query - returns ids, documents, metadatas
	results1, err := collection.Query(ctx,
		v2.WithQueryTexts("my query"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Get with only documents
	results2, err := collection.Get(ctx,
		v2.WithIncludeGet(v2.IncludeDocuments),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Query with specific includes
	results3, err := collection.Query(ctx,
		v2.WithQueryTexts("my query"),
		v2.WithIncludeQuery(v2.IncludeDocuments, v2.IncludeMetadatas, v2.IncludeEmbeddings),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Query 1 docs: %v", results1.GetDocumentsGroups())
	log.Printf("Get docs: %v", results2.GetDocuments())
	log.Printf("Query 3 embeddings: %v", results3.GetEmbeddingsGroups())
}
```
{% /codetab %}
{% /codetabs %}

### Include Distances

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

	// Query with distances included
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("search query"),
		v2.WithNResults(5),
		v2.WithIncludeQuery(v2.IncludeDocuments, v2.IncludeDistances),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Access distances
	distances := results.GetDistances()
	for i, dist := range distances[0] {
		log.Printf("Result %d: distance=%f", i, dist)
	}
}
```
{% /codetab %}
{% /codetabs %}

## Results Structure

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
class QueryResult(TypedDict):
    ids: List[IDs]
    embeddings: Optional[List[Embeddings]],
    documents: Optional[List[List[Document]]]
    metadatas: Optional[List[List[Metadata]]]
    distances: Optional[List[List[float]]]
    included: Include

class GetResult(TypedDict):
    ids: List[ID]
    embeddings: Optional[Embeddings],
    documents: Optional[List[Document]],
    metadatas: Optional[List[Metadata]]
    included: Include
```
{% /codetab %}
{% codetab label="Go" %}
```go
// QueryResult methods (raw access):
results.GetIDsGroups()         // [][]string - IDs grouped by query
results.GetDocumentsGroups()   // [][]string - Documents grouped by query
results.GetMetadatasGroups()   // [][]map[string]interface{} - Metadata grouped by query
results.GetEmbeddingsGroups()  // [][][]float32 - Embeddings grouped by query
results.GetDistancesGroups()   // [][]float32 - Distances grouped by query

// GetResult methods (raw access):
results.GetIDs()        // []string - List of IDs
results.GetDocuments()  // []string - List of documents
results.GetMetadatas()  // []map[string]interface{} - List of metadata
results.GetEmbeddings() // [][]float32 - List of embeddings

// Ergonomic iteration with Rows():
// Query results are indexed by input query:
// results.GetIDsGroups()[0] - Results for first query
// results.GetIDsGroups()[1] - Results for second query
```
{% /codetab %}
{% /codetabs %}

## Ergonomic Result Iteration

Use `Rows()` for clean iteration without manual index tracking:

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Python uses direct iteration over results
results = collection.query(query_texts=["search query"])
for i, doc_id in enumerate(results['ids'][0]):
    print(f"ID: {doc_id}")
    print(f"Document: {results['documents'][0][i]}")
    print(f"Distance: {results['distances'][0][i]}")
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

	// Query results - use Rows() for first query group
	queryResults, err := collection.Query(ctx,
		v2.WithQueryTexts("search query"),
		v2.WithNResults(10),
		v2.WithIncludeQuery(v2.IncludeDocuments, v2.IncludeDistances),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	// Iterate using Rows() - no manual index tracking needed
	fmt.Println("Query Results:")
	for i, row := range queryResults.Rows() {
		fmt.Printf("%d. ID: %s\n", i+1, row.ID)
		fmt.Printf("   Document: %s\n", row.Document)
		fmt.Printf("   Distance: %.4f\n\n", row.Score) // Score contains distance for Query
	}

	// For multiple queries, use RowGroups()
	multiQueryResults, err := collection.Query(ctx,
		v2.WithQueryTexts("first query", "second query"),
		v2.WithNResults(5),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	for queryIdx, rows := range multiQueryResults.RowGroups() {
		fmt.Printf("Query %d results:\n", queryIdx+1)
		for _, row := range rows {
			fmt.Printf("  - %s\n", row.ID)
		}
	}

	// Get results - use Rows() for flat iteration
	getResults, err := collection.Get(ctx,
		v2.WithIDsGet("id1", "id2", "id3"),
		v2.WithIncludeGet(v2.IncludeDocuments, v2.IncludeMetadatas),
	)
	if err != nil {
		log.Fatalf("Error getting: %v", err)
	}

	fmt.Println("\nGet Results:")
	for i, row := range getResults.Rows() {
		fmt.Printf("%d. ID: %s\n", i+1, row.ID)
		fmt.Printf("   Document: %s\n", row.Document)
		if row.Metadata != nil {
			fmt.Printf("   Metadata: %v\n", row.Metadata)
		}
	}

	// Safe indexed access with At()
	if row, ok := queryResults.At(0, 0); ok {
		fmt.Printf("\nFirst result: %s\n", row.ID)
	}
}
```
{% /codetab %}
{% /codetabs %}

## ResultRow Structure

The `ResultRow` struct provides unified access to all result fields:

```go
type ResultRow struct {
    ID        DocumentID       // Document ID
    Document  string           // Document text (if included)
    Metadata  DocumentMetadata // Metadata map (if included)
    Embedding []float32        // Embedding vector (if included)
    Score     float64          // Query: distance, Search: relevance score, Get: 0
}
```

## Notes

- Default `n_results` is 10
- IDs are always returned
- Query results are grouped by input query
- Get results are flat lists
- Use `Rows()` for easy iteration over results (first query group for Query)
- Use `RowGroups()` to iterate over all query groups
- Use `At(group, index)` for safe indexed access with bounds checking
- Use include options to control which fields are returned
- Distances are only available in query results (not get)
- The `Score` field contains distance for Query results, relevance score for Search results
