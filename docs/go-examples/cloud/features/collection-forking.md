# Collection Forking - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/cloud/features/collection-forking)

## Overview

Forking lets you create a new collection from an existing one instantly, using copy-on-write under the hood. The forked collection initially shares its data with the source and only incurs additional storage for incremental changes you make afterward.

> **Note**: Forking is available in Chroma Cloud only. The storage engine on single-node Chroma does not support forking.

## Go Examples

### Basic Collection Fork

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
source_collection = client.get_collection(name="main-repo-index")

# Create a forked collection. Name must be unique within the database.
forked_collection = source_collection.fork(name="main-repo-index-pr-1234")

# Forked collection is immediately queryable; changes are isolated
forked_collection.add(documents=["new content"], ids=["doc-pr-1"])  # billed as incremental storage
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
	client, err := v2.NewCloudClient(
		v2.WithCloudAPIKey("your-api-key"),
		v2.WithDatabaseAndTenant("database", "tenant"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Get the source collection
	sourceCollection, err := client.GetCollection(ctx, "main-repo-index")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create a forked collection. Name must be unique within the database.
	forkedCollection, err := sourceCollection.Fork(ctx, "main-repo-index-pr-1234")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Forked collection is immediately queryable; changes are isolated
	err = forkedCollection.Add(ctx,
		v2.WithIDs("doc-pr-1"),
		v2.WithDocuments("new content"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Forked collection '%s' created from '%s'",
		forkedCollection.Name(), sourceCollection.Name())
}
```
{% /codetab %}
{% /codetabs %}

### Git-like Workflow Example

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import chromadb

client = chromadb.CloudClient(
    tenant="your-tenant",
    database="your-database",
    api_key="your-api-key"
)

# Get the main branch collection
main_collection = client.get_collection(name="codebase-index")

# Fork for a feature branch
feature_collection = main_collection.fork(name="codebase-index-feature-xyz")

# Add new files from the feature branch
feature_collection.add(
    ids=["file1", "file2"],
    documents=["new feature code", "updated tests"]
)

# Query the fork - includes both original and new data
results = feature_collection.query(query_texts=["feature implementation"])

# Original collection remains unchanged
original_count = main_collection.count()
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"fmt"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	client, err := v2.NewCloudClient(
		v2.WithCloudAPIKey("your-api-key"),
		v2.WithDatabaseAndTenant("your-database", "your-tenant"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Get the main branch collection
	mainCollection, err := client.GetCollection(ctx, "codebase-index")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Fork for a feature branch
	featureCollection, err := mainCollection.Fork(ctx, "codebase-index-feature-xyz")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Add new files from the feature branch
	err = featureCollection.Add(ctx,
		v2.WithIDs("file1", "file2"),
		v2.WithDocuments("new feature code", "updated tests"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Query the fork - includes both original and new data
	results, err := featureCollection.Query(ctx,
		v2.WithQueryTexts("feature implementation"),
		v2.WithNResults(10),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Found %d results in forked collection\n", len(results.IDs[0]))

	// Original collection remains unchanged
	originalCount, err := mainCollection.Count(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Original collection still has %d documents\n", originalCount)
}
```
{% /codetab %}
{% /codetabs %}

### Data Versioning / Checkpointing

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import chromadb
from datetime import datetime

client = chromadb.CloudClient(
    tenant="your-tenant",
    database="your-database",
    api_key="your-api-key"
)

# Get the production collection
prod_collection = client.get_collection(name="knowledge-base")

# Create a checkpoint before major update
timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
checkpoint = prod_collection.fork(name=f"knowledge-base-checkpoint-{timestamp}")

# Now safe to update production
prod_collection.update(
    ids=["doc1"],
    documents=["updated content"]
)

# If something goes wrong, checkpoint has the original data
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	client, err := v2.NewCloudClient(
		v2.WithCloudAPIKey("your-api-key"),
		v2.WithDatabaseAndTenant("your-database", "your-tenant"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Get the production collection
	prodCollection, err := client.GetCollection(ctx, "knowledge-base")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create a checkpoint before major update
	timestamp := time.Now().Format("20060102_150405")
	checkpointName := fmt.Sprintf("knowledge-base-checkpoint-%s", timestamp)

	checkpoint, err := prodCollection.Fork(ctx, checkpointName)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Checkpoint created: %s", checkpoint.Name())

	// Now safe to update production
	err = prodCollection.Update(ctx,
		v2.WithIDs("doc1"),
		v2.WithDocuments("updated content"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Println("Production collection updated")

	// If something goes wrong, checkpoint has the original data
	// You can query it or use it to restore
}
```
{% /codetab %}
{% /codetabs %}

### A/B Testing with Forks

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import chromadb

client = chromadb.CloudClient(
    tenant="your-tenant",
    database="your-database",
    api_key="your-api-key"
)

# Get baseline collection
baseline = client.get_collection(name="search-index")

# Create variant A - different embedding model
variant_a = baseline.fork(name="search-index-variant-a")

# Create variant B - different chunking strategy
variant_b = baseline.fork(name="search-index-variant-b")

# Modify variant A with new embeddings
variant_a.update(
    ids=["doc1", "doc2"],
    embeddings=[new_embeddings_a]
)

# Modify variant B with different data
variant_b.delete(ids=["doc1"])
variant_b.add(ids=["doc1a", "doc1b"], documents=["chunk1", "chunk2"])

# Run queries against all three for comparison
query = "search query"
baseline_results = baseline.query(query_texts=[query])
variant_a_results = variant_a.query(query_texts=[query])
variant_b_results = variant_b.query(query_texts=[query])
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"fmt"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	client, err := v2.NewCloudClient(
		v2.WithCloudAPIKey("your-api-key"),
		v2.WithDatabaseAndTenant("your-database", "your-tenant"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Get baseline collection
	baseline, err := client.GetCollection(ctx, "search-index")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create variant A - different embedding model
	variantA, err := baseline.Fork(ctx, "search-index-variant-a")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create variant B - different chunking strategy
	variantB, err := baseline.Fork(ctx, "search-index-variant-b")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Modify variant A with new embeddings
	err = variantA.Update(ctx,
		v2.WithIDs("doc1", "doc2"),
		v2.WithEmbeddings([][]float32{{0.1, 0.2}, {0.3, 0.4}}),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Modify variant B with different chunking
	err = variantB.Delete(ctx, v2.WithDeleteIDs("doc1"))
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	err = variantB.Add(ctx,
		v2.WithIDs("doc1a", "doc1b"),
		v2.WithDocuments("chunk1", "chunk2"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Run queries against all three for comparison
	query := "search query"

	baselineResults, err := baseline.Query(ctx,
		v2.WithQueryTexts(query),
		v2.WithNResults(5),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	variantAResults, err := variantA.Query(ctx,
		v2.WithQueryTexts(query),
		v2.WithNResults(5),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	variantBResults, err := variantB.Query(ctx,
		v2.WithQueryTexts(query),
		v2.WithNResults(5),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Compare results
	fmt.Printf("Baseline results: %d\n", len(baselineResults.IDs[0]))
	fmt.Printf("Variant A results: %d\n", len(variantAResults.IDs[0]))
	fmt.Printf("Variant B results: %d\n", len(variantBResults.IDs[0]))
}
```
{% /codetab %}
{% /codetabs %}

## Fork API Reference

| Python | Go Function | Description |
|--------|-------------|-------------|
| `collection.fork(name="new-name")` | `collection.Fork(ctx, "new-name")` | Create fork of collection |

## Notes

- **Copy-on-write**: Forks share data blocks with the source collection. New writes to either branch allocate new blocks; unchanged data remains shared.
- **Instant**: Forking a collection of any size completes quickly.
- **Isolation**: Changes to a fork do not affect the source, and vice versa.
- **Pricing**: $0.03 per fork call, plus storage for incremental blocks written after the fork.
- **Quota**: Default limit is 4,096 fork edges per tree. Deleted collections still count toward this limit.
- **Database scope**: Forked collections belong to the same database as the source collection.

## Use Cases

1. **Data versioning/checkpointing**: Maintain consistent snapshots as your data evolves
2. **Git-like workflows**: Index a branch by forking from its divergence point, then apply the diff
3. **A/B testing**: Compare different embedding strategies or data configurations
4. **Safe experimentation**: Test changes without affecting production data

