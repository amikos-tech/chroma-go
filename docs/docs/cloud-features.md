# Cloud Features

This page documents features that are only available in Chroma Cloud.

## Indexing Status

Chroma Cloud uses a write-ahead log (WAL) to durably store writes before compacting them into the index. The Indexing Status API lets you check how much of the WAL has been indexed.

!!! note "Cloud Only"
    This feature requires Chroma Cloud with version >= 1.4.1

### Usage

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

### IndexingStatus Response

| Field | Type | Description |
|-------|------|-------------|
| `NumIndexedOps` | `uint64` | Number of operations compacted into the index |
| `NumUnindexedOps` | `uint64` | Number of operations still in the WAL |
| `TotalOps` | `uint64` | Total number of operations |
| `OpIndexingProgress` | `float64` | Progress from 0.0 to 1.0 |

### Use Cases

- **Monitor batch ingestion**: Track progress when loading large datasets
- **Optimize read levels**: Use `ReadLevelIndexOnly` when indexing is complete for faster queries
- **Wait for data availability**: Ensure recently added data is searchable before querying

### Integration with Read Levels

```go
status, err := collection.IndexingStatus(ctx)
if err != nil {
    log.Fatalf("Error: %v", err)
}

var readLevel v2.ReadLevel
if status.OpIndexingProgress >= 1.0 {
    // All data indexed - use faster index-only reads
    readLevel = v2.ReadLevelIndexOnly
} else {
    // Some data not indexed - read from WAL to see all data
    readLevel = v2.ReadLevelIndexAndWAL
}

results, err := collection.Search(ctx,
    v2.NewSearchRequest(
        v2.WithKnnRank(v2.KnnQueryText("machine learning")),
        v2.WithPage(v2.PageLimit(10)),
    ),
    v2.WithReadLevel(readLevel),
)
```

For more examples, see [Indexing Status Go Examples](../go-examples/cloud/features/indexing-status.md).

## Collection Forking

Forking lets you create a new collection from an existing one instantly using copy-on-write.

!!! note "Cloud Only"
    Collection forking is available in Chroma Cloud only.

```go
// Get source collection
sourceCollection, err := client.GetCollection(ctx, "main-repo-index")
if err != nil {
    log.Fatalf("Error: %v", err)
}

// Create a forked collection
forkedCollection, err := sourceCollection.Fork(ctx, "main-repo-index-pr-1234")
if err != nil {
    log.Fatalf("Error: %v", err)
}

// Forked collection is immediately queryable
err = forkedCollection.Add(ctx,
    v2.WithIDs("doc-pr-1"),
    v2.WithTexts("new content"),
)
```

For more examples, see [Collection Forking Go Examples](../go-examples/cloud/features/collection-forking.md).
