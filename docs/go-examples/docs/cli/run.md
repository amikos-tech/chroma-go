# Running a Chroma Server - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/cli/run)

## Overview

The Chroma CLI lets you run a Chroma server locally. This document shows how to connect to a running Chroma server from Go.

## Running the Server

Start a Chroma server using the CLI:

```bash
# Basic usage
chroma run --path /path/to/persist/data

# With custom host and port
chroma run --path /path/to/data --host 0.0.0.0 --port 8080

# With configuration file
chroma run --config ./config.yaml
```

## Go Examples

### Connecting to Your Chroma Server

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import chromadb

chroma_client = chromadb.HttpClient(host='localhost', port=8000)
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
	// Connect to local Chroma server
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://localhost:8000"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Verify connection
	heartbeat, err := client.Heartbeat(ctx)
	if err != nil {
		log.Fatalf("Cannot connect to Chroma server: %v", err)
	}

	log.Printf("Connected to Chroma server. Heartbeat: %d", heartbeat)
}
```
{% /codetab %}
{% /codetabs %}

### Using Default Connection

```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Default connection: http://localhost:8000
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Check server version
	version, err := client.Version(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Chroma version: %s", version)
}
```

### Custom Configuration

```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Connect with custom configuration
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("https://chroma-server.example.com:8080"),
		v2.WithDatabaseAndTenant("my_database", "my_tenant"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// List collections in the configured database/tenant
	collections, err := client.ListCollections(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	for _, col := range collections {
		log.Printf("Collection: %s", col.Name())
	}
}
```

### Using Base URL

```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Connect using full base URL
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://localhost:8000"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	heartbeat, err := client.Heartbeat(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Connected. Heartbeat: %d", heartbeat)
}
```

### With Authentication

```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Bearer token authentication
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://localhost:8000"),
		v2.WithAuth(v2.NewTokenAuthCredentialsProvider("your-token", v2.AuthorizationTokenHeader)),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	collections, err := client.ListCollections(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Found %d collections", len(collections))
}
```

### Complete Example

```go
package main

import (
	"context"
	"fmt"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Create client
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://localhost:8000"),
	)
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Check connection
	heartbeat, err := client.Heartbeat(ctx)
	if err != nil {
		log.Fatalf("Server not responding: %v", err)
	}
	fmt.Printf("Connected! Heartbeat: %d\n", heartbeat)

	// Get server version
	version, err := client.Version(ctx)
	if err != nil {
		log.Fatalf("Error getting version: %v", err)
	}
	fmt.Printf("Chroma version: %s\n", version)

	// Create or get a collection
	collection, err := client.GetOrCreateCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error with collection: %v", err)
	}
	fmt.Printf("Using collection: %s\n", collection.Name())

	// Add documents
	err = collection.Add(ctx,
		v2.WithIDs("doc1", "doc2", "doc3"),
		v2.WithDocuments(
			"The quick brown fox",
			"Machine learning basics",
			"Vector database concepts",
		),
	)
	if err != nil {
		log.Fatalf("Error adding documents: %v", err)
	}
	fmt.Println("Documents added successfully")

	// Query the collection
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("fox"),
		v2.WithNResults(2),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	fmt.Printf("\nQuery results:\n")
	for i, row := range results.Rows() {
		fmt.Printf("  %d. %s\n", i+1, row.ID)
	}
}
```

## Server Configuration Reference

| CLI Argument | Description | Default |
|--------------|-------------|---------|
| `--path` | Data persistence directory | `./chroma` |
| `--host` | Server hostname | `localhost` |
| `--port` | Server port | `8000` |
| `--config_path` | Configuration file path | - |

## Notes

- Start the Chroma server before running your Go application
- Use `v2.NewHTTPClient()` with no arguments for default localhost:8000 connection
- Always call `client.Close()` to release resources (use `defer`)
- Check the server is running with `client.Heartbeat(ctx)`
- For production, consider using Chroma Cloud instead of self-hosted

