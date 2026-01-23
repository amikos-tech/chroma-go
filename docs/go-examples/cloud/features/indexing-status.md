# Indexing Status - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/cloud/features/indexing-status)

## Overview

Chroma Cloud uses a write-ahead log (WAL) to durably store writes before compacting them into the index. The Indexing Status API lets you check how much of the WAL has been indexed, which is useful for monitoring data ingestion progress or determining when recently added data will be searchable with `ReadLevelIndexOnly`.

> **Note**: Indexing Status is available in Chroma Cloud only (requires Chroma >= 1.4.1).

## Go Examples

### Basic Usage

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import chromadb

client = chromadb.CloudClient(
    tenant="your-tenant",
    database="your-database",
    api_key="your-api-key"
)

collection = client.get_collection(name="my_collection")
status = collection.indexing_status()

print(f"Total operations: {status.total_ops}")
print(f"Indexed operations: {status.num_indexed_ops}")
print(f"Unindexed operations: {status.num_unindexed_ops}")
print(f"Progress: {status.op_indexing_progress * 100:.1f}%")
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

	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	status, err := collection.IndexingStatus(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Total operations: %d\n", status.TotalOps)
	fmt.Printf("Indexed operations: %d\n", status.NumIndexedOps)
	fmt.Printf("Unindexed operations: %d\n", status.NumUnindexedOps)
	fmt.Printf("Progress: %.1f%%\n", status.OpIndexingProgress*100)
}
```
{% /codetab %}
{% /codetabs %}

### Waiting for Indexing Completion

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import chromadb
import time

client = chromadb.CloudClient(
    tenant="your-tenant",
    database="your-database",
    api_key="your-api-key"
)

collection = client.get_collection(name="my_collection")

# Add some documents
collection.add(
    ids=["doc1", "doc2", "doc3"],
    documents=["first document", "second document", "third document"]
)

# Wait for indexing to complete
while True:
    status = collection.indexing_status()
    if status.op_indexing_progress >= 1.0:
        print("Indexing complete!")
        break
    print(f"Indexing progress: {status.op_indexing_progress * 100:.1f}%")
    time.sleep(1)
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

	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Add some documents
	err = collection.Add(ctx,
		v2.WithIDs("doc1", "doc2", "doc3"),
		v2.WithTexts("first document", "second document", "third document"),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Wait for indexing to complete
	for {
		status, err := collection.IndexingStatus(ctx)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		if status.OpIndexingProgress >= 1.0 {
			fmt.Println("Indexing complete!")
			break
		}

		fmt.Printf("Indexing progress: %.1f%%\n", status.OpIndexingProgress*100)
		time.Sleep(1 * time.Second)
	}
}
```
{% /codetab %}
{% /codetabs %}

### Using with Read Levels

Combine indexing status with read levels to optimize search performance:

{% codetabs group="lang" %}
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

	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Check indexing status to decide which read level to use
	status, err := collection.IndexingStatus(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	var readLevel v2.ReadLevel
	if status.OpIndexingProgress >= 1.0 {
		// All data is indexed, use faster index-only reads
		readLevel = v2.ReadLevelIndexOnly
		fmt.Println("Using index-only reads (faster)")
	} else {
		// Some data not yet indexed, read from WAL to see all data
		readLevel = v2.ReadLevelIndexAndWAL
		fmt.Printf("Using WAL reads (%.1f%% indexed)\n", status.OpIndexingProgress*100)
	}

	// Execute search with the appropriate read level
	results, err := collection.Search(ctx,
		v2.NewSearchRequest(
			v2.WithKnnRank(v2.KnnQueryText("machine learning")),
			v2.NewPage(v2.Limit(10)),
			v2.WithSelect(v2.KDocument, v2.KScore),
		),
		v2.WithReadLevel(readLevel),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	for _, row := range results.Rows() {
		fmt.Printf("ID: %s, Score: %.3f\n", row.ID, row.Score)
	}
}
```
{% /codetab %}
{% /codetabs %}

### Monitoring Batch Ingestion

{% codetabs group="lang" %}
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

	collection, err := client.CreateCollection(ctx, "batch_ingestion_test")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Simulate batch ingestion
	batchSize := 100
	for batch := 0; batch < 5; batch++ {
		ids := make([]v2.DocumentID, batchSize)
		texts := make([]string, batchSize)
		for i := 0; i < batchSize; i++ {
			ids[i] = v2.DocumentID(fmt.Sprintf("doc_%d_%d", batch, i))
			texts[i] = fmt.Sprintf("Document %d from batch %d", i, batch)
		}

		err = collection.Add(ctx, v2.WithIDs(ids...), v2.WithTexts(texts...))
		if err != nil {
			log.Fatalf("Error adding batch %d: %v", batch, err)
		}

		// Check indexing status after each batch
		status, err := collection.IndexingStatus(ctx)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		fmt.Printf("Batch %d complete - Total ops: %d, Indexed: %d, Progress: %.1f%%\n",
			batch+1,
			status.TotalOps,
			status.NumIndexedOps,
			status.OpIndexingProgress*100,
		)
	}

	// Wait for all indexing to complete
	fmt.Println("\nWaiting for indexing to complete...")
	for {
		status, err := collection.IndexingStatus(ctx)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		if status.OpIndexingProgress >= 1.0 {
			fmt.Printf("All %d operations indexed!\n", status.TotalOps)
			break
		}

		fmt.Printf("Progress: %.1f%% (%d/%d)\n",
			status.OpIndexingProgress*100,
			status.NumIndexedOps,
			status.TotalOps,
		)
		time.Sleep(2 * time.Second)
	}
}
```
{% /codetab %}
{% /codetabs %}

## IndexingStatus Response

| Field | Go Type | Description |
|-------|---------|-------------|
| `num_indexed_ops` | `uint64` | Number of operations that have been compacted into the index |
| `num_unindexed_ops` | `uint64` | Number of operations still in the WAL waiting to be indexed |
| `total_ops` | `uint64` | Total number of operations (`num_indexed_ops + num_unindexed_ops`) |
| `op_indexing_progress` | `float64` | Progress from 0.0 to 1.0 (`num_indexed_ops / total_ops`) |

## API Reference

| Python | Go Method | Description |
|--------|-----------|-------------|
| `collection.indexing_status()` | `collection.IndexingStatus(ctx)` | Get the indexing status of a collection |

## Notes

- **Cloud Only**: This API is available only in Chroma Cloud (requires Chroma >= 1.4.1)
- **Operations Include**: Add, update, upsert, and delete operations are all counted
- **Read Levels**: When `op_indexing_progress < 1.0`, use `ReadLevelIndexAndWAL` to see all data; use `ReadLevelIndexOnly` for faster queries on indexed data only
- **Eventual Consistency**: The index is eventually consistent; recently added data will become searchable with `ReadLevelIndexOnly` once indexed
