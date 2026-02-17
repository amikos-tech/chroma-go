# Adding Data - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/collections/add-data)

## Overview

Add data to a Chroma collection with the `.Add` method. It takes unique string IDs and documents. Chroma will embed these documents using the collection's embedding function.

## Go Examples

### Add Documents with Metadata

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.add(
    ids=["id1", "id2", "id3"],
    documents=["lorem ipsum...", "doc2", "doc3"],
    metadatas=[{"chapter": 3, "verse": 16}, {"chapter": 3, "verse": 5}, {"chapter": 29, "verse": 11}],
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

	// Add documents with metadata
	err = collection.Add(ctx,
		v2.WithIDs("id1", "id2", "id3"),
		v2.WithTexts("lorem ipsum...", "doc2", "doc3"),
		v2.WithMetadatas(
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
	)
	if err != nil {
		log.Fatalf("Error adding documents: %v", err)
	}

	log.Println("Documents added successfully")
}
```
{% /codetab %}
{% /codetabs %}

### Add Documents with Pre-computed Embeddings

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.add(
    ids=["id1", "id2", "id3"],
    embeddings=[[1.1, 2.3, 3.2], [4.5, 6.9, 4.4], [1.1, 2.3, 3.2]],
    documents=["doc1", "doc2", "doc3"],
    metadatas=[{"chapter": 3, "verse": 16}, {"chapter": 3, "verse": 5}, {"chapter": 29, "verse": 11}],
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

	// Add documents with pre-computed embeddings
	err = collection.Add(ctx,
		v2.WithIDs("id1", "id2", "id3"),
		v2.WithEmbeddings(
			[]float32{1.1, 2.3, 3.2},
			[]float32{4.5, 6.9, 4.4},
			[]float32{1.1, 2.3, 3.2},
		),
		v2.WithTexts("doc1", "doc2", "doc3"),
		v2.WithMetadatas(
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
	)
	if err != nil {
		log.Fatalf("Error adding documents: %v", err)
	}

	log.Println("Documents with embeddings added successfully")
}
```
{% /codetab %}
{% /codetabs %}

### Add Embeddings Only (No Documents)

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.add(
    embeddings=[[1.1, 2.3, 3.2], [4.5, 6.9, 4.4], [1.1, 2.3, 3.2]],
    metadatas=[{"chapter": 3, "verse": 16}, {"chapter": 3, "verse": 5}, {"chapter": 29, "verse": 11}],
    ids=["id1", "id2", "id3"]
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

	// Create collection without embedding function for manual embeddings
	collection, err := client.GetOrCreateCollection(ctx, "my_collection",
		v2.WithEmbeddingFunctionCreate(nil),
	)
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	// Add only embeddings and metadata (documents stored elsewhere)
	err = collection.Add(ctx,
		v2.WithIDs("id1", "id2", "id3"),
		v2.WithEmbeddings(
			[]float32{1.1, 2.3, 3.2},
			[]float32{4.5, 6.9, 4.4},
			[]float32{1.1, 2.3, 3.2},
		),
		v2.WithMetadatas(
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
	)
	if err != nil {
		log.Fatalf("Error adding embeddings: %v", err)
	}

	log.Println("Embeddings added successfully")
}
```
{% /codetab %}
{% /codetabs %}

### Add Documents with Array Metadata

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Array metadata support requires Chroma >= 1.5.0
collection.add(
    ids=["id1", "id2"],
    documents=["doc about science", "doc about math"],
    metadatas=[
        {"tags": ["science", "research"], "scores": [95, 87]},
        {"tags": ["math", "education"], "scores": [100, 92]},
    ],
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

	// Add documents with array metadata (Chroma >= 1.5.0)
	err = collection.Add(ctx,
		v2.WithIDs("id1", "id2"),
		v2.WithTexts("doc about science", "doc about math"),
		v2.WithMetadatas(
			v2.NewDocumentMetadata(
				v2.NewStringArrayAttribute("tags", []string{"science", "research"}),
				v2.NewIntArrayAttribute("scores", []int64{95, 87}),
			),
			v2.NewDocumentMetadata(
				v2.NewStringArrayAttribute("tags", []string{"math", "education"}),
				v2.NewIntArrayAttribute("scores", []int64{100, 92}),
			),
		),
	)
	if err != nil {
		log.Fatalf("Error adding documents: %v", err)
	}

	log.Println("Documents with array metadata added successfully")
}
```
{% /codetab %}
{% /codetabs %}

### Add with ID Generator

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

	// Use ULID generator for automatic ID generation
	err = collection.Add(ctx,
		v2.WithIDGenerator(v2.NewULIDGenerator()),
		v2.WithTexts("hello world", "goodbye world"),
	)
	if err != nil {
		log.Fatalf("Error adding documents: %v", err)
	}

	log.Println("Documents added with auto-generated IDs")
}
```
{% /codetab %}
{% /codetabs %}

## Notes

- If you add a record with an ID that already exists, it will be ignored
- Embedding dimensions must match those already in the collection
- Use `Upsert` instead of `Add` if you want to update existing records
- Go uses `float32` slices for embeddings
- Use `v2.NewDocumentMetadata()` with attribute helpers for type-safe metadata
- Array metadata (Chroma >= 1.5.0) uses `v2.NewStringArrayAttribute`, `v2.NewIntArrayAttribute`, `v2.NewFloatArrayAttribute`, `v2.NewBoolArrayAttribute`
