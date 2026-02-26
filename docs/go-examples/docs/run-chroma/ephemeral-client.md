# Ephemeral Client - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/run-chroma/ephemeral-client)

## Overview

In Python, you can run a Chroma server in-memory with the ephemeral client. This is useful for experimenting and testing without long-lived persistence.

> **Note**: Go does not expose an in-memory-only runtime yet, but `NewLocalClient` gives a close equivalent: run Chroma in-process and use a temporary directory that you delete after the run.

## Go Examples

### Python Ephemeral Client

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import chromadb

# Python can run Chroma in-memory
client = chromadb.EphemeralClient()
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
)

func main() {
	tmpDir, err := os.MkdirTemp("", "chroma-ephemeral-*")
	if err != nil {
		log.Fatalf("Error creating temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	client, err := v2.NewLocalClient(
		v2.WithLocalPersistPath(tmpDir),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create a collection for experimentation
	collection, err := client.CreateCollection(ctx, "test_collection")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Collection created: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

### Using Testcontainers for Testing

For Go tests that need an ephemeral-like experience, use testcontainers:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func TestWithEphemeralChroma(t *testing.T) {
	ctx := context.Background()

	// Start a Chroma container
	req := testcontainers.ContainerRequest{
		Image:        "chromadb/chroma:latest",
		ExposedPorts: []string{"8000/tcp"},
		WaitingFor:   wait.ForHTTP("/api/v2/heartbeat").WithPort("8000"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	defer container.Terminate(ctx)

	// Get the container's host and port
	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "8000")

	// Connect to the ephemeral Chroma instance
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL(fmt.Sprintf("http://%s:%d", host, port.Int())),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Run your tests...
	collection, err := client.CreateCollection(ctx, "test_collection")
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// Data is ephemeral - container and data are destroyed after test
	t.Logf("Created ephemeral collection: %s", collection.Name())
}
```

### Running a Local Server for Development

For development, start a Chroma server locally:

```bash
# Option 1: Using Docker
docker run -p 8000:8000 chromadb/chroma

# Option 2: Using the Chroma CLI (requires Python)
pip install chromadb
chroma run --path /tmp/chroma-data

# Option 3: Using Docker Compose
# docker-compose.yml:
# version: '3'
# services:
#   chroma:
#     image: chromadb/chroma:latest
#     ports:
#       - "8000:8000"

docker-compose up
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
	// Connect to local development server
	client, err := v2.NewHTTPClient() // Defaults to localhost:8000
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

	// Create and use collections
	collection, err := client.GetOrCreateCollection(ctx, "dev_collection")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Using collection: %s", collection.Name())
}
```

## Notes

- `NewLocalClient` provides in-process runtime for local/ephemeral workflows
- Use testcontainers for isolated testing environments
- Use Docker for local development servers
- For Cloud development, use `v2.NewCloudClient()`
- Data in Docker containers without volume mounts is ephemeral by design
