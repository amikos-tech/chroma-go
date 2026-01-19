# Client-Server Mode - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/run-chroma/client-server)

## Overview

Chroma can run in client/server mode where the Chroma client connects to a Chroma server running in a separate process. This document shows how to connect to a Chroma server using the Go client.

## Starting the Server

Start the Chroma server:

```terminal
chroma run --path /db_path
```

Or using Docker:

```terminal
docker pull chromadb/chroma
docker run -p 8000:8000 chromadb/chroma
```

## Go Examples

### Basic HTTP Client

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
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Create HTTP client connecting to default localhost:8000
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	// Client is now connected and ready to use
	log.Println("Connected to Chroma server")
}
```
{% /codetab %}
{% /codetabs %}

### HTTP Client with Custom Host and Port

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import chromadb

chroma_client = chromadb.HttpClient(host='my-chroma-server.com', port=8080)
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
	// Create HTTP client with custom base URL
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://my-chroma-server.com:8080"),
	)
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	log.Println("Connected to custom Chroma server")
}
```
{% /codetab %}
{% /codetabs %}

### Async Client Example

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import asyncio
import chromadb

async def main():
    client = await chromadb.AsyncHttpClient()

    collection = await client.create_collection(name="my_collection")
    await collection.add(
        documents=["hello world"],
        ids=["id1"]
    )

asyncio.run(main())
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
	// Go uses goroutines for concurrency instead of async/await
	// The chroma-go client is synchronous but can be used with goroutines

	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create collection
	collection, err := client.CreateCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	// Add documents
	err = collection.Add(ctx,
		v2.WithIDs("id1"),
		v2.WithTexts("hello world"),
	)
	if err != nil {
		log.Fatalf("Error adding documents: %v", err)
	}

	log.Println("Successfully added documents")
}
```
{% /codetab %}
{% /codetabs %}

> **Note**: Go does not have async/await syntax like Python. Instead, Go uses goroutines and channels for concurrency. The chroma-go client methods are synchronous, but you can run them in goroutines if needed for concurrent operations.

### Client with SSL/TLS

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import chromadb

# Python HttpClient with SSL
chroma_client = chromadb.HttpClient(
    host='my-chroma-server.com',
    port=443,
    ssl=True
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
	// Use HTTPS URL for SSL/TLS connections
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("https://my-chroma-server.com:443"),
	)
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	log.Println("Connected via HTTPS")
}
```
{% /codetab %}
{% /codetabs %}

### Client with Custom Timeout

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
package main

import (
	"log"
	"time"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Create client with custom timeout
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://localhost:8000"),
		v2.WithTimeout(60*time.Second),
	)
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	log.Println("Client created with 60 second timeout")
}
```
{% /codetab %}
{% /codetabs %}

## Notes

- Always call `defer client.Close()` to properly release resources, especially embedding functions
- The default URL is `http://localhost:8000/api/v2`
- Environment variables `CHROMA_TENANT` and `CHROMA_DATABASE` are automatically applied if set
- Use `v2.WithBaseURL()` to configure custom server addresses
- Use `v2.WithTimeout()` to set custom request timeouts
