// This example demonstrates the simplified V2 API with single patterns for each operation,
// following Go's "one obvious way" principle.
//
//nolint:staticcheck // Showing deprecated methods for migration guidance
package main

import (
	"context"
	"fmt"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Demonstrate the simplified V2 API with cleaner naming conventions

	// Create client
	client, err := chroma.NewHTTPClient()
	if err != nil {
		fmt.Printf("Error creating client: %s\n", err)
		return
	}
	defer client.Close()

	// Example 1: Metadata creation using builder pattern (single approach)
	fmt.Println("=== Example 1: Metadata Builder Pattern ===")

	// Deprecated way (verbose)
	// oldMetadata := chroma.NewMetadata(
	//     chroma.NewStringAttribute("description", "My collection"),
	//     chroma.NewIntAttribute("version", 1),
	//     chroma.NewFloatAttribute("threshold", 0.8),
	// )

	// Single recommended way (builder pattern)
	metadata := chroma.Builder().
		String("description", "My collection").
		Int("version", 1).
		Float("threshold", 0.8).
		Build()

	fmt.Printf("Built metadata: %v\n", metadata)

	// Example 2: Simplified collection creation
	fmt.Println("\n=== Example 2: Simplified Collection Creation ===")

	// Old way (verbose option names)
	// col, err := client.CreateCollection(context.Background(), "old-collection",
	//     chroma.WithCollectionMetadataCreate(oldMetadata),
	//     chroma.WithEmbeddingFunctionCreate(ef),
	// )

	// Create collection with metadata
	col, err := client.CreateCollection(context.Background(), "new-collection",
		chroma.WithMetadata(metadata),
		chroma.WithCreateIfNotExists(),
	)
	if err != nil {
		fmt.Printf("Error creating collection: %s\n", err)
		return
	}

	// Example 3: Simplified document operations
	fmt.Println("\n=== Example 3: Simplified Document Operations ===")

	// Old way (WithTexts adds Documents, confusing naming)
	// err = col.Add(context.Background(),
	//     chroma.WithTexts("doc1", "doc2"),
	//     chroma.WithIDs("1", "2"),
	//     chroma.WithMetadatas(meta1, meta2),
	// )

	// Add documents with metadata using builder pattern
	err = col.Add(context.Background(),
		chroma.WithTexts("Hello world", "Goodbye world"),
		chroma.WithIDs("1", "2"),
		chroma.WithMetadatas(
			chroma.DocumentBuilder().String("type", "greeting").Int("priority", 1).Build(),
			chroma.DocumentBuilder().String("type", "farewell").Int("priority", 2).Build(),
		),
	)
	if err != nil {
		fmt.Printf("Error adding documents: %s\n", err)
		return
	}

	// Example 4: Simplified query with cleaner operators
	fmt.Println("\n=== Example 4: Simplified Query ===")

	// Deprecated way (type-specific function)
	// where := chroma.GtInt("priority", 0)

	// Single way (type-agnostic function)
	where := chroma.Gt("priority", 0) // Auto-detects type

	// Old way (WithNResults is confusing)
	// results, err := col.Query(context.Background(),
	//     chroma.WithQueryTexts("hello"),
	//     chroma.WithNResults(5),
	//     chroma.WithWhereQuery(where),
	// )

	// New way (WithLimit is clearer)
	results, err := col.Query(context.Background(),
		chroma.WithQueryText("hello"), // Simplified for single text
		chroma.WithLimit(5),           // Clearer than WithNResults
		chroma.WithWhereQuery(where),  // Using existing API for now
		chroma.WithIncludeQuery(chroma.IncludeDocuments, chroma.IncludeMetadatas),
	)
	if err != nil {
		fmt.Printf("Error querying: %s\n", err)
		return
	}

	// Example 5: Result access
	fmt.Println("\n=== Example 5: Result Access ===")

	// Standard way to access results
	docs := results.GetDocumentsGroups()[0]
	ids := results.GetIDGroups()[0]
	metas := results.GetMetadatasGroups()[0]

	fmt.Printf("Found %d documents\n", len(ids))
	for i, doc := range docs {
		fmt.Printf("  [%s] %s (metadata: %v)\n", ids[i], doc, metas[i])
	}

	// Example 6: Reusable options across operations
	fmt.Println("\n=== Example 6: Reusable Options ===")

	// Using type-agnostic Where function
	where2 := chroma.Eq("type", "greeting") // Type-agnostic
	getResult, _ := col.Get(context.Background(),
		chroma.WithWhereGet(where2),
		chroma.WithIDsGet("1"),
	)
	_ = col.Delete(context.Background(), chroma.WithWhereDelete(where2))

	fmt.Printf("Options demonstrate simplified metadata builders\n")
	fmt.Printf("Get result count: %d\n", getResult.Count())

	// Example 7: Document metadata builder
	fmt.Println("\n=== Example 7: Document Metadata Builder ===")

	docMeta := chroma.DocumentBuilder().
		String("author", "John Doe").
		Int("year", 2024).
		Bool("published", true).
		Float("score", 0.95).
		Build()

	fmt.Printf("Built document metadata: %v\n", docMeta)

	// Clean up
	err = client.DeleteCollection(context.Background(), "new-collection")
	if err != nil {
		fmt.Printf("Error deleting collection: %s\n", err)
	}

	fmt.Println("\n=== Summary ===")
	fmt.Println("The simplified API provides:")
	fmt.Println("1. Single Builder pattern for metadata creation")
	fmt.Println("2. Type-agnostic Where functions (Eq, Gt, Lt, etc.)")
	fmt.Println("3. Shorter operator constants (GT vs GreaterThanOperator)")
	fmt.Println("4. Clearer method names (WithLimit instead of WithNResults)")
	fmt.Println("5. One obvious way to accomplish each task")
	fmt.Println("6. Follows Go ecosystem best practices (AWS SDK v2, stdlib)")
}
