# Deleting Data - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/collections/delete-data)

## Overview

Chroma supports deleting items from a collection by ID using `.Delete`. The embeddings, documents, and metadata associated with each item will be deleted.

> **Warning**: Deleting data is destructive and cannot be undone.

## Go Examples

### Delete by IDs

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.delete(
    ids=["id1", "id2", "id3"],
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

	// Delete records by IDs
	err = collection.Delete(ctx,
		v2.WithIDsDelete("id1", "id2", "id3"),
	)
	if err != nil {
		log.Fatalf("Error deleting documents: %v", err)
	}

	log.Println("Documents deleted successfully")
}
```
{% /codetab %}
{% /codetabs %}

### Delete with Where Filter

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection.delete(
    ids=["id1", "id2", "id3"],
    where={"chapter": "20"}
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

	// Delete records by IDs with additional where filter
	err = collection.Delete(ctx,
		v2.WithIDsDelete("id1", "id2", "id3"),
		v2.WithWhereDelete(v2.EqString("chapter", "20")),
	)
	if err != nil {
		log.Fatalf("Error deleting documents: %v", err)
	}

	log.Println("Documents matching filter deleted")
}
```
{% /codetab %}
{% /codetabs %}

### Delete All Matching Filter (No IDs)

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

	// Delete all records matching the where filter
	// No IDs specified - deletes all matching documents
	err = collection.Delete(ctx,
		v2.WithWhereDelete(v2.EqString("status", "archived")),
	)
	if err != nil {
		log.Fatalf("Error deleting documents: %v", err)
	}

	log.Println("All archived documents deleted")
}
```
{% /codetab %}
{% /codetabs %}

### Delete with Complex Filter

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

	// Delete with complex filter using AND
	err = collection.Delete(ctx,
		v2.WithWhereDelete(
			v2.And(
				v2.EqString("category", "draft"),
				v2.LtInt("version", 5),
			),
		),
	)
	if err != nil {
		log.Fatalf("Error deleting documents: %v", err)
	}

	log.Println("Draft documents with version < 5 deleted")
}
```
{% /codetab %}
{% /codetabs %}

## Notes

- Delete is a destructive operation and cannot be undone
- If no IDs are supplied, all records matching the `where` filter will be deleted
- Use `where` filters to narrow down which records to delete
- Deleting removes embeddings, documents, and metadata for each record
