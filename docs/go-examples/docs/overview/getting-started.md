# Getting Started - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/overview/getting-started)

## Overview

Chroma is an AI-native open-source vector database. This guide shows how to get started with Chroma using the Go client.

## 1. Install

```terminal
go get github.com/amikos-tech/chroma-go
```

## 2. Create a Chroma Client

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import chromadb
chroma_client = chromadb.Client()
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// The Go client connects to a running Chroma server
	// Start the server first: chroma run --path ./getting-started
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()
}
```
{% /codetab %}
{% /codetabs %}

> **Note**: Unlike Python's ephemeral `Client()`, the Go client requires a running Chroma server. Start the server with `chroma run --path ./data` or use Docker.

## 3. Create a Collection

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection = chroma_client.create_collection(name="my_collection")
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

	// Create a new collection
	collection, err := client.CreateCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	log.Printf("Created collection: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

## 4. Add Documents

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.add(
    ids=["id1", "id2"],
    documents=[
        "This is a document about pineapple",
        "This is a document about oranges"
    ]
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

	collection, err := client.GetOrCreateCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	// Add documents with IDs
	err = collection.Add(ctx,
		v2.WithIDs("id1", "id2"),
		v2.WithTexts(
			"This is a document about pineapple",
			"This is a document about oranges",
		),
	)
	if err != nil {
		log.Fatalf("Error adding documents: %v", err)
	}

	log.Println("Documents added successfully")
}
```
{% /codetab %}
{% /codetabs %}

## 5. Query the Collection

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
results = collection.query(
    query_texts=["This is a query document about hawaii"],
    n_results=2
)
print(results)
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

	collection, err := client.GetOrCreateCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	// Query the collection
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("This is a query document about hawaii"),
		v2.WithNResults(2),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	// Access results
	documents := results.GetDocumentsGroups()
	ids := results.GetIDsGroups()
	distances := results.GetDistances()

	log.Printf("Documents: %v", documents)
	log.Printf("IDs: %v", ids)
	log.Printf("Distances: %v", distances)
}
```
{% /codetab %}
{% /codetabs %}

## 6. Inspect Results

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
{
  'documents': [[
      'This is a document about pineapple',
      'This is a document about oranges'
  ]],
  'ids': [['id1', 'id2']],
  'distances': [[1.0404009819030762, 1.243080496788025]],
  'uris': None,
  'data': None,
  'metadatas': [[None, None]],
  'embeddings': None,
}
```
{% /codetab %}
{% codetab label="Go" %}
```go
// In Go, results are accessed through typed methods:
documents := results.GetDocumentsGroups() // [][]string
ids := results.GetIDsGroups()             // [][]string
distances := results.GetDistances()       // [][]float32
metadatas := results.GetMetadatasGroups() // [][]map[string]interface{}
embeddings := results.GetEmbeddingsGroups() // [][][]float32

// Example output:
// documents[0] = ["This is a document about pineapple", "This is a document about oranges"]
// ids[0] = ["id1", "id2"]
// distances[0] = [1.0404009819030762, 1.243080496788025]
```
{% /codetab %}
{% /codetabs %}

## 7. Complete Example

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import chromadb
chroma_client = chromadb.Client()

collection = chroma_client.get_or_create_collection(name="my_collection")

collection.upsert(
    documents=[
        "This is a document about pineapple",
        "This is a document about oranges"
    ],
    ids=["id1", "id2"]
)

results = collection.query(
    query_texts=["This is a query document about florida"],
    n_results=2
)

print(results)
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
	// Create client (requires running Chroma server)
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Get or create collection
	collection, err := client.GetOrCreateCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	// Upsert documents (insert or update)
	err = collection.Upsert(ctx,
		v2.WithIDs("id1", "id2"),
		v2.WithTexts(
			"This is a document about pineapple",
			"This is a document about oranges",
		),
	)
	if err != nil {
		log.Fatalf("Error upserting documents: %v", err)
	}

	// Query the collection
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("This is a query document about florida"),
		v2.WithNResults(2),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	// Print results
	log.Printf("Documents: %v", results.GetDocumentsGroups())
	log.Printf("IDs: %v", results.GetIDsGroups())
	log.Printf("Distances: %v", results.GetDistances())
}
```
{% /codetab %}
{% /codetabs %}

## Notes

- The Go client requires a running Chroma server (unlike Python's ephemeral client)
- Start the server with: `chroma run --path ./data` or use Docker
- Always use `defer client.Close()` to release resources
- All operations require a `context.Context` for cancellation/timeout support
- Results are accessed through typed getter methods (`GetDocumentsGroups()`, `GetIDsGroups()`, etc.)
