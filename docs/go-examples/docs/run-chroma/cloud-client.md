# Cloud Client - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/run-chroma/cloud-client)

## Overview

The Cloud Client connects to Chroma Cloud, a fast, scalable, and serverless database-as-a-service.

## Go Examples

### Basic Cloud Client

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
client = CloudClient(
    tenant='Tenant ID',
    database='Database name',
    api_key='Chroma Cloud API key'
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
	// Create Cloud client with explicit credentials
	client, err := v2.NewCloudClient(
		v2.WithCloudAPIKey("your-api-key"),
		v2.WithDatabaseAndTenant("database-name", "tenant-id"),
	)
	if err != nil {
		log.Fatalf("Error creating cloud client: %v", err)
	}
	defer client.Close()

	log.Println("Connected to Chroma Cloud")
}
```
{% /codetab %}
{% /codetabs %}

### Cloud Client with Environment Variables

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
# Set environment variables:
# CHROMA_API_KEY, CHROMA_TENANT, CHROMA_DATABASE

client = CloudClient()
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"log"
	"os"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Environment variables used:
	// CHROMA_CLOUD_API_KEY - Cloud API key
	// CHROMA_CLOUD_TENANT - Cloud tenant ID
	// CHROMA_CLOUD_DATABASE - Cloud database name

	client, err := v2.NewCloudClient(
		v2.WithCloudAPIKey(os.Getenv("CHROMA_CLOUD_API_KEY")),
		v2.WithDatabaseAndTenant(
			os.Getenv("CHROMA_CLOUD_DATABASE"),
			os.Getenv("CHROMA_CLOUD_TENANT"),
		),
	)
	if err != nil {
		log.Fatalf("Error creating cloud client: %v", err)
	}
	defer client.Close()

	log.Println("Connected to Chroma Cloud using environment variables")
}
```
{% /codetab %}
{% /codetabs %}

### Complete Cloud Client Example

{% codetabs group="lang" %}
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
	// Create Cloud client
	client, err := v2.NewCloudClient(
		v2.WithCloudAPIKey(os.Getenv("CHROMA_CLOUD_API_KEY")),
		v2.WithDatabaseAndTenant(
			os.Getenv("CHROMA_CLOUD_DATABASE"),
			os.Getenv("CHROMA_CLOUD_TENANT"),
		),
	)
	if err != nil {
		log.Fatalf("Error creating cloud client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create or get a collection
	collection, err := client.GetOrCreateCollection(ctx, "my_cloud_collection")
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	// Add documents
	err = collection.Add(ctx,
		v2.WithIDs("doc1", "doc2"),
		v2.WithTexts(
			"This is a document about machine learning",
			"This is a document about data science",
		),
	)
	if err != nil {
		log.Fatalf("Error adding documents: %v", err)
	}

	// Query the collection
	results, err := collection.Query(ctx,
		v2.WithQueryTexts("AI and ML"),
		v2.WithNResults(2),
	)
	if err != nil {
		log.Fatalf("Error querying collection: %v", err)
	}

	log.Printf("Query results: %v", results.GetDocumentsGroups())
}
```
{% /codetab %}
{% /codetabs %}

## Notes

- The Go client uses `CHROMA_CLOUD_API_KEY`, `CHROMA_CLOUD_TENANT`, and `CHROMA_CLOUD_DATABASE` environment variable names
- Always call `defer client.Close()` to properly release resources
- Cloud client automatically configures the correct API endpoint for Chroma Cloud
