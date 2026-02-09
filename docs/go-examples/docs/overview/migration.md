# Migration - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/overview/migration)

## Overview

This document covers migration notes and breaking changes when upgrading Chroma versions. Since chroma-go is an HTTP client, most migration concerns relate to API changes and server compatibility.

## Go Client Considerations

### Version Compatibility

The chroma-go client is tested against Chroma versions 0.4.8 to 1.5.0. When upgrading your Chroma server, check for any API changes that might affect your client code.

### API v1 vs v2

chroma-go provides two API versions:

{% codetabs group="lang" %}
{% codetab label="v1 API (Legacy)" %}
```go
package main

import (
	"context"
	"log"

	chroma "github.com/amikos-tech/chroma-go"
)

func main() {
	// V1 API - Legacy, maintained for backward compatibility
	client, err := chroma.NewClient("http://localhost:8000")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	ctx := context.Background()
	collection, err := client.GetCollection(ctx, "my_collection", nil)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Collection: %s", collection.Name)
}
```
{% /codetab %}
{% codetab label="v2 API (Current)" %}
```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// V2 API - Current primary API, all new features go here
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Collection: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

### Key v2 API Changes

1. **Client creation returns error**: All client constructors return `(client, error)`
2. **Close method**: Clients should be closed with `defer client.Close()`
3. **Functional options**: Configuration uses functional options pattern
4. **Collection methods**: `collection.Name()` instead of `collection.Name`
5. **Search API**: New Search API with `collection.Search()` for advanced queries

### Migrating from v1 to v2

{% codetabs group="lang" %}
{% codetab label="v1 API" %}
```go
package main

import (
	"context"

	chroma "github.com/amikos-tech/chroma-go"
)

func main() {
	// v1 client creation
	client, _ := chroma.NewClient("http://localhost:8000")

	ctx := context.Background()

	// v1 collection operations
	collection, _ := client.GetCollection(ctx, "my_collection", nil)

	// v1 query
	results, _ := collection.Query(ctx,
		[]string{"query text"},
		10,
		nil,  // where
		nil,  // whereDocument
		nil,  // include
	)

	_ = results
}
```
{% /codetab %}
{% codetab label="v2 API" %}
```go
package main

import (
	"context"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// v2 client creation with error handling
	client, err := v2.NewHTTPClient()
	if err != nil {
		panic(err)
	}
	defer client.Close()

	ctx := context.Background()

	// v2 collection operations
	collection, _ := client.GetCollection(ctx, "my_collection")

	// v2 query with functional options
	results, _ := collection.Query(ctx,
		v2.WithQueryTexts("query text"),
		v2.WithNResults(10),
	)

	_ = results
}
```
{% /codetab %}
{% /codetabs %}

## Server Migration Notes

### v1.0.0 Changes (March 2025)

Key changes that may affect Go clients:

1. **Authentication changes**: Chroma no longer provides built-in authentication implementations
2. **Configuration changes**: Server now uses config files instead of environment variables

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# list_collections now returns Collection objects again
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
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// ListCollections returns collection objects with metadata
	collections, err := client.ListCollections(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	for _, col := range collections {
		log.Printf("Collection: %s (ID: %s)", col.Name(), col.ID())
	}
}
```
{% /codetab %}
{% /codetabs %}

### v0.5.11 Changes (September 2024)

Query results ordering changed:

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Results from get() are now ordered by internal IDs (insertion order)
# Previously ordered by user-provided IDs
results = collection.get(limit=2, offset=2)
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
	client, _ := v2.NewHTTPClient()
	defer client.Close()

	ctx := context.Background()
	collection, _ := client.GetCollection(ctx, "my_collection")

	// Results ordered by internal IDs (newer documents have larger IDs)
	// Limit and offset behavior depends on this ordering
	results, err := collection.Get(ctx,
		v2.WithGetLimit(2),
		v2.WithGetOffset(2),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("IDs: %v", results.Ids)
}
```
{% /codetab %}
{% /codetabs %}

### v0.5.17 Changes (October 2024)

Empty filters are no longer supported:

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# This is no longer supported:
# collection.get(ids=["id1", "id2"], where={})

# Use this instead:
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
	client, _ := v2.NewHTTPClient()
	defer client.Close()

	ctx := context.Background()
	collection, _ := client.GetCollection(ctx, "my_collection")

	// Don't pass empty where filters - omit them instead
	results, err := collection.Get(ctx,
		v2.WithGetIDs("id1", "id2"),
		// Don't use empty where: v2.WithGetWhere(...)
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("IDs: %v", results.Ids)
}
```
{% /codetab %}
{% /codetabs %}

## Authentication Migration

When upgrading to Chroma v1.0.0+, update your authentication configuration:

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import chromadb

# Token auth
client = chromadb.HttpClient(
    host="localhost",
    port=8000,
    headers={"Authorization": "Bearer your-token"}
)
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
	// Token auth (Bearer)
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://localhost:8000"),
		v2.WithAuth(v2.NewTokenAuthCredentialsProvider("your-token", v2.AuthorizationTokenHeader)),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	// Or use X-Chroma-Token header
	clientWithToken, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://localhost:8000"),
		v2.WithAuth(v2.NewTokenAuthCredentialsProvider("your-token", v2.XChromaTokenHeader)),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer clientWithToken.Close()

	// Or use Basic auth
	clientWithBasic, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://localhost:8000"),
		v2.WithAuth(v2.NewBasicAuthCredentialsProvider("username", "password")),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer clientWithBasic.Close()
}
```
{% /codetab %}
{% /codetabs %}

## Notes

- chroma-go is an HTTP-only client; it always connects to a Chroma server
- No in-process/embedded mode is available in Go (unlike Python's PersistentClient/EphemeralClient)
- API v2 is recommended for all new projects
- Always check the [chroma-go releases](https://github.com/amikos-tech/chroma-go/releases) for client-specific changes
- Server upgrades may require updating your Go client version for full compatibility

