# Managing Collections - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/collections/manage-collections)

## Overview

Chroma lets you manage collections of embeddings. Collections are the fundamental unit of storage and querying in Chroma.

## Creating Collections

Collection names must follow these rules:
- Length between 3 and 512 characters
- Start and end with a lowercase letter or digit
- Can contain dots, dashes, and underscores in between
- No two consecutive dots
- Not a valid IP address

### Basic Collection Creation

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection = client.create_collection(name="my_collection")
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

### Collection with Embedding Function

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import os
from chromadb.utils.embedding_functions import OpenAIEmbeddingFunction

collection = client.create_collection(
    name="my_collection",
    embedding_function=OpenAIEmbeddingFunction(
        api_key=os.getenv("OPENAI_API_KEY"),
        model_name="text-embedding-3-small"
    )
)
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"log"
	"os"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
)

func main() {
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create OpenAI embedding function
	ef, err := openai.NewOpenAIEmbeddingFunction(
		os.Getenv("OPENAI_API_KEY"),
		openai.WithModel(openai.TextEmbedding3Small),
	)
	if err != nil {
		log.Fatalf("Error creating embedding function: %v", err)
	}

	// Create collection with embedding function
	collection, err := client.CreateCollection(ctx, "my_collection",
		v2.WithEmbeddingFunctionCreate(ef),
	)
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	log.Printf("Created collection: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

### Collection without Embedding Function

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection = client.create_collection(
    name="my_collection",
    embedding_function=None
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

	// Create collection without embedding function
	// You must provide embeddings directly when adding data
	collection, err := client.CreateCollection(ctx, "my_collection",
		v2.WithEmbeddingFunctionCreate(nil),
	)
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	log.Printf("Created collection: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

### Collection with Metadata

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from datetime import datetime

collection = client.create_collection(
    name="my_collection",
    embedding_function=emb_fn,
    metadata={
        "description": "my first Chroma collection",
        "created": str(datetime.now())
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
	"time"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create collection with metadata
	collection, err := client.CreateCollection(ctx, "my_collection",
		v2.WithCollectionMetadataCreate(
			v2.NewMetadata(
				v2.NewStringAttribute("description", "my first Chroma collection"),
				v2.NewStringAttribute("created", time.Now().String()),
			),
		),
	)
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	log.Printf("Created collection: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

For Go metadata maps:

- `v2.NewMetadataFromMap` is best-effort and silently skips invalid `[]interface{}` values.
- `v2.NewMetadataFromMapStrict` returns an error when metadata is invalid.
- `v2.WithCollectionMetadataMapCreateStrict(...)` uses strict map conversion for collection create/get-or-create options.

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
collection, err := client.CreateCollection(ctx, "my_collection",
	v2.WithCollectionMetadataMapCreateStrict(map[string]interface{}{
		"description": "my first Chroma collection",
		"tags":        []interface{}{"go", "sdk"},
	}),
)
if err != nil {
	log.Fatalf("Error creating collection: %v", err)
}
```
{% /codetab %}
{% /codetabs %}

## Getting Collections

### Get Collection by Name

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection = client.get_collection(name="my-collection")
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

	// Get existing collection
	collection, err := client.GetCollection(ctx, "my-collection")
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	log.Printf("Got collection: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

### Get or Create Collection

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection = client.get_or_create_collection(
    name="my-collection",
    metadata={"description": "..."}
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

	// Get or create collection with metadata
	collection, err := client.GetOrCreateCollection(ctx, "my-collection",
		v2.WithCollectionMetadataCreate(
			v2.NewMetadata(
				v2.NewStringAttribute("description", "..."),
			),
		),
	)
	if err != nil {
		log.Fatalf("Error getting/creating collection: %v", err)
	}

	log.Printf("Collection: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

### List Collections

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collections = client.list_collections()
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

	// List all collections
	collections, err := client.ListCollections(ctx)
	if err != nil {
		log.Fatalf("Error listing collections: %v", err)
	}

	for _, col := range collections {
		log.Printf("Collection: %s", col.Name)
	}
}
```
{% /codetab %}
{% /codetabs %}

### List Collections with Pagination

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
first_collections_batch = client.list_collections(limit=100)
second_collections_batch = client.list_collections(limit=100, offset=100)
collections_subset = client.list_collections(limit=20, offset=50)
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

	// Get first 100 collections
	firstBatch, err := client.ListCollections(ctx,
		v2.ListWithLimit(100),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Get next 100 collections
	secondBatch, err := client.ListCollections(ctx,
		v2.ListWithLimit(100),
		v2.ListWithOffset(100),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Get 20 collections starting from the 50th
	subset, err := client.ListCollections(ctx,
		v2.ListWithLimit(20),
		v2.ListWithOffset(50),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("First batch: %d, Second batch: %d, Subset: %d",
		len(firstBatch), len(secondBatch), len(subset))
}
```
{% /codetab %}
{% /codetabs %}

## Modifying Collections

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.modify(
   name="new-name",
   metadata={"description": "new description"}
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

	collection, err := client.GetCollection(ctx, "my-collection")
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	// Modify collection name
	err = collection.ModifyName(ctx, "new-name")
	if err != nil {
		log.Fatalf("Error modifying collection name: %v", err)
	}

	// Modify collection metadata from a raw map with strict validation
	newMetadata, err := v2.NewMetadataFromMapStrict(map[string]interface{}{
		"description": "new description",
	})
	if err != nil {
		log.Fatalf("Invalid metadata map: %v", err)
	}

	err = collection.ModifyMetadata(ctx, newMetadata)
	if err != nil {
		log.Fatalf("Error modifying collection metadata: %v", err)
	}

	log.Println("Collection modified successfully")
}
```
{% /codetab %}
{% /codetabs %}

## Deleting Collections

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
client.delete_collection(name="my-collection")
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

	// Delete collection by name
	err = client.DeleteCollection(ctx, "my-collection")
	if err != nil {
		log.Fatalf("Error deleting collection: %v", err)
	}

	log.Println("Collection deleted successfully")
}
```
{% /codetab %}
{% /codetabs %}

## Convenience Methods

### Count and Peek

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.count()
collection.peek()
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

	collection, err := client.GetCollection(ctx, "my-collection")
	if err != nil {
		log.Fatalf("Error getting collection: %v", err)
	}

	// Get document count
	count, err := collection.Count(ctx)
	if err != nil {
		log.Fatalf("Error getting count: %v", err)
	}
	log.Printf("Collection has %d documents", count)

	// Peek at first 10 documents
	peek, err := collection.Peek(ctx)
	if err != nil {
		log.Fatalf("Error peeking: %v", err)
	}
	log.Printf("First documents: %v", peek.GetDocuments())
}
```
{% /codetab %}
{% /codetabs %}

## Notes

- Collection names must be unique within a database
- The embedding function is persisted with the collection (Chroma >= 1.1.13)
- Use `GetOrCreateCollection` to avoid errors when collection might already exist
- Deleting a collection is permanent and removes all associated data
- Use `defer client.Close()` to properly release resources
