# Persistent Client - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/run-chroma/persistent-client)

## Overview

In Python, you can use `PersistentClient` to save and load data from your local machine. In Go, you always connect to a Chroma server via HTTP, so persistence is handled by the server's configuration.

## Go Examples

### Python vs Go Client

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import chromadb

# Python can run Chroma embedded with persistence
client = chromadb.PersistentClient(path="/path/to/save/to")
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
	// Go connects to a Chroma server - persistence is server-side
	// Start server with: chroma run --path /path/to/save/to

	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://localhost:8000"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	// Data persists on the server based on server configuration
}
```
{% /codetab %}
{% /codetabs %}

### Running a Persistent Chroma Server

Start a Chroma server with persistence enabled:

```bash
# Using the Chroma CLI
chroma run --path /path/to/save/to

# Using Docker with a volume mount for persistence
docker run -p 8000:8000 -v /path/to/save/to:/data chromadb/chroma
```

Then connect from Go:

```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Connect to persistent Chroma server
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://localhost:8000"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Data will persist on the server between restarts
	collection, err := client.GetOrCreateCollection(ctx, "persistent_collection")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Add data
	err = collection.Add(ctx,
		v2.WithIDs("doc1", "doc2"),
		v2.WithDocuments(
			"First document content",
			"Second document content",
		),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Data added to collection: %s", collection.Name())
	log.Println("Data will persist on server restart")
}
```

### Utility Methods

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
client.heartbeat()
client.reset()
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

	// Heartbeat - check if server is running
	heartbeat, err := client.Heartbeat(ctx)
	if err != nil {
		log.Fatalf("Server not responding: %v", err)
	}
	log.Printf("Server heartbeat: %d nanoseconds", heartbeat)

	// Reset - empties and resets the database
	// Warning: This is destructive and not reversible!
	// Server must have allow_reset=true in config
	err = client.Reset(ctx)
	if err != nil {
		log.Printf("Reset failed (may be disabled): %v", err)
	}
}
```
{% /codetab %}
{% /codetabs %}

### Docker Compose Example

Create a `docker-compose.yml` for persistent Chroma:

```yaml
version: '3'
services:
  chroma:
    image: chromadb/chroma:latest
    ports:
      - "8000:8000"
    volumes:
      - ./chroma-data:/data
    environment:
      - CHROMA_HOST=0.0.0.0
```

Connect from Go:

```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://localhost:8000"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Check connection
	_, err = client.Heartbeat(ctx)
	if err != nil {
		log.Fatalf("Cannot connect to Chroma: %v", err)
	}

	// List existing collections (will persist across restarts)
	collections, err := client.ListCollections(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Found %d persistent collections", len(collections))
	for _, col := range collections {
		count, _ := col.Count(ctx)
		log.Printf("  - %s: %d documents", col.Name(), count)
	}
}
```

### Environment Variables

Configure the Go client using environment variables:

```go
package main

import (
	"log"
	"os"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Set CHROMA_URL environment variable
	// export CHROMA_URL=http://chroma-server:8000

	chromaURL := os.Getenv("CHROMA_URL")
	if chromaURL == "" {
		chromaURL = "http://localhost:8000"
	}

	client, err := v2.NewHTTPClient(
		v2.WithBaseURL(chromaURL),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	log.Printf("Connected to: %s", chromaURL)
}
```

## Notes

- Go is HTTP-only - persistence is configured on the server side
- Use Docker volumes to persist data when running Chroma in containers
- The `reset()` method requires `allow_reset=true` in server configuration
- For production, always mount a persistent volume to `/data` in the Docker container
- Consider using Chroma Cloud for managed persistence

