# Telemetry - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/overview/telemetry)

## Overview

Chroma contains a telemetry feature that collects **anonymous** usage information. This is configured on the Chroma server side, not in the Go client.

## Server-Side Configuration

Telemetry is controlled by the Chroma server, not the client. When using the Go client, you need to configure telemetry on your Chroma server.

### Opting Out

{% codetabs group="lang" %}
{% codetab label="Python (Server Config)" %}
```python
from chromadb.config import Settings

# Python in-process mode can disable telemetry
client = chromadb.Client(Settings(anonymized_telemetry=False))

# Or with PersistentClient
client = chromadb.PersistentClient(
    path="/path/to/save/to",
    settings=Settings(anonymized_telemetry=False)
)
```
{% /codetab %}
{% codetab label="Go (Client)" %}
```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Go client cannot control server telemetry
	// Telemetry is configured on the Chroma server side

	// Connect to server (which may have telemetry enabled/disabled)
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://localhost:8000"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Normal operations...
	collections, err := client.ListCollections(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Found %d collections", len(collections))
}
```
{% /codetab %}
{% /codetabs %}

### Using Environment Variables

To disable telemetry on your Chroma server:

```bash
# Set environment variable before starting Chroma server
export ANONYMIZED_TELEMETRY=False
chroma run --path /path/to/data
```

Or in a Docker environment:

```bash
# docker-compose.yml
version: '3'
services:
  chroma:
    image: chromadb/chroma:latest
    ports:
      - "8000:8000"
    environment:
      - ANONYMIZED_TELEMETRY=False
    volumes:
      - ./chroma-data:/data
```

Or with an `.env` file:

```
# .env file in the same directory as docker-compose.yml
ANONYMIZED_TELEMETRY=False
```

### Connecting to Configured Server

```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Connect to Chroma server (telemetry status depends on server config)
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Check connection
	heartbeat, err := client.Heartbeat(ctx)
	if err != nil {
		log.Fatalf("Server not responding: %v", err)
	}

	log.Printf("Connected to Chroma server. Heartbeat: %d", heartbeat)

	// Your application code...
	collection, err := client.GetOrCreateCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Operations here are tracked by server telemetry (if enabled)
	err = collection.Add(ctx,
		v2.WithIDs("doc1"),
		v2.WithDocuments("Sample document"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Document added to collection: %s", collection.Name())
}
```

## What Chroma Tracks

When telemetry is enabled on the server, it tracks:

- Chroma version and environment details
- Usage of embedding functions
- Collection commands (add, update, query, get, delete)
- Anonymized collection UUIDs and item counts

Chroma does **not** collect:

- Usernames, hostnames, or file names
- Environment variables
- Actual document content or embeddings
- Personally-identifiable information

## Notes

- Telemetry is a server-side setting, not controlled by the Go client
- The Go client simply makes HTTP requests; tracking happens on the server
- Use environment variables or Docker configuration to control telemetry
- Telemetry data is stored in Posthog for product analytics
- For Chroma Cloud, telemetry helps improve the service

