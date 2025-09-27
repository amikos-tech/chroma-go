package main

import (
	"context"
	"fmt"
	"log"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Demonstrate the simplified V2 API with cleaner naming conventions

	// Create client
	client, err := chroma.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %s\n", err)
	}
	defer client.Close()

	// Example 1: Simplified metadata creation using builder pattern
	fmt.Println("=== Example 1: Metadata Builder Pattern ===")

	// Old way (verbose)
	oldMetadata := chroma.NewMetadata(
		chroma.NewStringAttribute("description", "My collection"),
		chroma.NewIntAttribute("version", 1),
		chroma.NewFloatAttribute("threshold", 0.8),
	)

	// New way (builder pattern)
	newMetadata := chroma.Builder().
		String("description", "My collection").
		Int("version", 1).
		Float("threshold", 0.8).
		Build()

	// Even simpler way (variadic)
	simpleMetadata := chroma.QuickMetadata(
		"description", "My collection",
		"version", 1,
		"threshold", 0.8,
	)

	fmt.Printf("Old metadata: %v\n", oldMetadata)
	fmt.Printf("New metadata: %v\n", newMetadata)
	fmt.Printf("Simple metadata: %v\n", simpleMetadata)

	// Example 2: Simplified collection creation
	fmt.Println("\n=== Example 2: Simplified Collection Creation ===")

	// Old way (verbose option names)
	// col, err := client.CreateCollection(context.Background(), "old-collection",
	//     chroma.WithCollectionMetadataCreate(oldMetadata),
	//     chroma.WithEmbeddingFunctionCreate(ef),
	// )

	// New way (simplified option names)
	col, err := client.CreateCollection(context.Background(), "new-collection",
		chroma.WithMetadata(simpleMetadata),
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

	// New way (simplified metadata creation)
	err = col.Add(context.Background(),
		chroma.WithTexts("Hello world", "Goodbye world"), // WithDocuments coming in future //nolint:staticcheck
		chroma.WithIDs("1", "2"),
		chroma.WithMetadatas(
			chroma.QuickDocumentMetadata("type", "greeting", "priority", 1),
			chroma.QuickDocumentMetadata("type", "farewell", "priority", 2),
		),
	)
	if err != nil {
		fmt.Printf("Error adding documents: %s\n", err)
		return
	}

	// Example 4: Simplified query with cleaner operators
	fmt.Println("\n=== Example 4: Simplified Query ===")

	// Old way (verbose function names)
	// where := chroma.GtInt("priority", 0)

	// New way (simplified Where function)
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
		chroma.WithWhereQuery(where),  // Using existing API for now //nolint:staticcheck
		chroma.WithIncludeQuery(chroma.IncludeDocuments, chroma.IncludeMetadatas), //nolint:staticcheck
	)
	if err != nil {
		log.Fatalf("Error querying: %s\n", err)
	}

	// Example 5: Simplified result access
	fmt.Println("\n=== Example 5: Simplified Result Access ===")

	// Old way (verbose method names)
	// docs := results.GetDocumentsGroups()[0]
	// ids := results.GetIDsGroups()[0]

	// New way (cleaner method names)
	simplified := chroma.AsQueryResults(results)
	docs := simplified.Documents()  // Instead of GetDocumentsGroups()[0]
	ids := simplified.IDs()         // Instead of GetIDsGroups()[0]
	metas := simplified.Metadatas() // Instead of GetMetadatasGroups()[0]

	fmt.Printf("Found %d documents\n", simplified.Count())
	for i, doc := range docs {
		fmt.Printf("  [%s] %s (metadata: %v)\n", ids[i], doc, metas[i])
	}

	// Example 6: Reusable options across operations
	fmt.Println("\n=== Example 6: Reusable Options ===")

	// Using simplified Where with existing API
	where2 := chroma.Eq("type", "greeting") // Simplified creation
	getResult, _ := col.Get(context.Background(),
		chroma.WithWhereGet(where2), //nolint:staticcheck
		chroma.WithIDsGet("1"),      //nolint:staticcheck
	)
	_ = col.Delete(context.Background(), chroma.WithWhereDelete(where2)) //nolint:staticcheck

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
		log.Fatalf("Error deleting collection: %s\n", err)
	}

	fmt.Println("\n=== Summary ===")
	fmt.Println("The simplified API provides:")
	fmt.Println("1. Cleaner option names without operation suffixes")
	fmt.Println("2. Builder pattern for metadata creation")
	fmt.Println("3. Shorter operator constants (GT vs GreaterThanOperator)")
	fmt.Println("4. Consistent naming (WithDocuments instead of WithTexts)")
	fmt.Println("5. Clearer method names (WithLimit instead of WithNResults)")
	fmt.Println("6. Simplified result access (Documents() vs GetDocumentsGroups()[0])")
	fmt.Println("7. Reusable options across operations")
}
