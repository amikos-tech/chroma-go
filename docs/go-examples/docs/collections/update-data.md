# Updating Data - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/collections/update-data)

## Overview

Any property of records in a collection can be updated with `.Update`. Chroma also supports `.Upsert` which updates existing items or adds them if they don't exist.

## Go Examples

### Update Records

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.update(
    ids=["id1", "id2", "id3"],
    embeddings=[[1.1, 2.3, 3.2], [4.5, 6.9, 4.4], [1.1, 2.3, 3.2]],
    metadatas=[{"chapter": 3, "verse": 16}, {"chapter": 3, "verse": 5}, {"chapter": 29, "verse": 11}],
    documents=["doc1", "doc2", "doc3"],
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

	// Update existing records
	err = collection.Update(ctx,
		v2.WithIDsUpdate("id1", "id2", "id3"),
		v2.WithEmbeddingsUpdate(
			[]float32{1.1, 2.3, 3.2},
			[]float32{4.5, 6.9, 4.4},
			[]float32{1.1, 2.3, 3.2},
		),
		v2.WithMetadatasUpdate(
			v2.NewDocumentMetadata(
				v2.NewIntAttribute("chapter", 3),
				v2.NewIntAttribute("verse", 16),
			),
			v2.NewDocumentMetadata(
				v2.NewIntAttribute("chapter", 3),
				v2.NewIntAttribute("verse", 5),
			),
			v2.NewDocumentMetadata(
				v2.NewIntAttribute("chapter", 29),
				v2.NewIntAttribute("verse", 11),
			),
		),
		v2.WithTextsUpdate("doc1", "doc2", "doc3"),
	)
	if err != nil {
		log.Fatalf("Error updating documents: %v", err)
	}

	log.Println("Documents updated successfully")
}
```
{% /codetab %}
{% /codetabs %}

### Update Documents Only

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

	// Update only documents - embeddings will be recomputed
	err = collection.Update(ctx,
		v2.WithIDsUpdate("id1", "id2"),
		v2.WithTextsUpdate("new document 1", "new document 2"),
	)
	if err != nil {
		log.Fatalf("Error updating documents: %v", err)
	}

	log.Println("Documents updated - embeddings recomputed")
}
```
{% /codetab %}
{% /codetabs %}

### Upsert Records

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.upsert(
    ids=["id1", "id2", "id3"],
    embeddings=[[1.1, 2.3, 3.2], [4.5, 6.9, 4.4], [1.1, 2.3, 3.2]],
    metadatas=[{"chapter": 3, "verse": 16}, {"chapter": 3, "verse": 5}, {"chapter": 29, "verse": 11}],
    documents=["doc1", "doc2", "doc3"],
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

	// Upsert - insert or update records
	err = collection.Upsert(ctx,
		v2.WithIDs("id1", "id2", "id3"),
		v2.WithEmbeddings(
			[]float32{1.1, 2.3, 3.2},
			[]float32{4.5, 6.9, 4.4},
			[]float32{1.1, 2.3, 3.2},
		),
		v2.WithMetadatas(
			v2.NewDocumentMetadata(
				v2.NewStringAttribute("chapter", "3"),
				v2.NewStringAttribute("verse", "16"),
			),
			v2.NewDocumentMetadata(
				v2.NewStringAttribute("chapter", "3"),
				v2.NewStringAttribute("verse", "5"),
			),
			v2.NewDocumentMetadata(
				v2.NewStringAttribute("chapter", "29"),
				v2.NewStringAttribute("verse", "11"),
			),
		),
		v2.WithTexts("doc1", "doc2", "doc3"),
	)
	if err != nil {
		log.Fatalf("Error upserting documents: %v", err)
	}

	log.Println("Documents upserted successfully")
}
```
{% /codetab %}
{% /codetabs %}

### Simple Upsert with Documents

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

	collection, err := client.GetOrCreateCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	// Simple upsert with just IDs and documents
	// Embeddings will be computed automatically
	err = collection.Upsert(ctx,
		v2.WithIDs("id1", "id2"),
		v2.WithTexts(
			"This document will be inserted or updated",
			"Another document to upsert",
		),
	)
	if err != nil {
		log.Fatalf("Error upserting: %v", err)
	}

	log.Println("Upsert completed")
}
```
{% /codetab %}
{% /codetabs %}

## Notes

- If an ID is not found during `Update`, an error will be logged and ignored
- If documents are supplied without embeddings, embeddings are recomputed
- Embedding dimensions must match the collection's existing embeddings
- `Upsert` creates new records for IDs that don't exist
- `Upsert` updates records for IDs that already exist
