// This example demonstrates the V2 API with type-specific functions and builder patterns
// for metadata creation.
//
//nolint:staticcheck // Showing deprecated methods for migration guidance
package main

import (
	"context"
	"fmt"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Demonstrate the V2 API improvements: builder pattern, cleaner naming, type-safe functions

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
	metadata, err := chroma.Builder().
		String("description", "My collection").
		Int("version", 1).
		Float("threshold", 0.8).
		Build()
	if err != nil {
		fmt.Printf("Error building metadata: %v\n", err)
		return
	}

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
		func() chroma.CollectionAddOption {
			var metas []chroma.DocumentMetadata

			meta1, err1 := chroma.DocumentBuilder().String("type", "greeting").Int("priority", 1).Build()
			if err1 != nil {
				fmt.Printf("Error building metadata 1: %v\n", err1)
			} else {
				metas = append(metas, meta1)
			}

			meta2, err2 := chroma.DocumentBuilder().String("type", "farewell").Int("priority", 2).Build()
			if err2 != nil {
				fmt.Printf("Error building metadata 2: %v\n", err2)
			} else {
				metas = append(metas, meta2)
			}

			return chroma.WithMetadatas(metas...)
		}(),
	)
	if err != nil {
		fmt.Printf("Error adding documents: %s\n", err)
		return
	}

	// Example 4: Type-safe query with type-specific where functions
	fmt.Println("\n=== Example 4: Type-Safe Query ===")

	// Type-specific function ensures compile-time type safety
	where := chroma.GtInt("priority", 0)

	// Deprecated way (WithNResults was confusing)
	// results, err := col.Query(context.Background(),
	//     chroma.WithQueryTexts("hello"),
	//     chroma.WithNResults(5),
	//     chroma.WithWhereQuery(where),
	// )

	// Recommended way (WithLimit is clearer than WithNResults)
	results, err := col.Query(context.Background(),
		chroma.WithQueryText("hello"), // Simplified for single text
		chroma.WithLimit(5),           // Clearer than WithNResults
		chroma.WithWhereQuery(where),
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

	// Using type-specific Where function
	where2 := chroma.EqString("type", "greeting")
	getResult, _ := col.Get(context.Background(),
		chroma.WithWhereGet(where2),
		chroma.WithIDsGet("1"),
	)
	_ = col.Delete(context.Background(), chroma.WithWhereDelete(where2))

	fmt.Printf("Options demonstrate simplified metadata builders\n")
	fmt.Printf("Get result count: %d\n", getResult.Count())

	// Example 7: Document metadata builder
	fmt.Println("\n=== Example 7: Document Metadata Builder ===")

	docMeta, err := chroma.DocumentBuilder().
		String("author", "John Doe").
		Int("year", 2024).
		Bool("published", true).
		Float("score", 0.95).
		Build()
	if err != nil {
		fmt.Printf("Error building document metadata: %v\n", err)
		return
	}

	fmt.Printf("Built document metadata: %v\n", docMeta)

	// Clean up
	err = client.DeleteCollection(context.Background(), "new-collection")
	if err != nil {
		fmt.Printf("Error deleting collection: %s\n", err)
	}

	fmt.Println("\n=== Summary ===")
	fmt.Println("The V2 API provides:")
	fmt.Println("1. Builder pattern for metadata creation")
	fmt.Println("2. Type-specific Where functions (EqString, GtInt, etc.)")
	fmt.Println("3. Shorter operator constants (GT vs GreaterThanOperator)")
	fmt.Println("4. Clearer method names (WithLimit instead of WithNResults)")
	fmt.Println("5. Strong type safety with compile-time checks")
	fmt.Println("6. Clear, explicit APIs without runtime type switching")
}
