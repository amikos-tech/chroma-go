package main

import (
	"context"
	"fmt"
	"log"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Create a new Chroma client
	client, err := chroma.NewHTTPClient()
	if err != nil {
		log.Printf("Error creating client: %s \n", err)
		return
	}
	defer func() {
		err = client.Close()
		if err != nil {
			log.Printf("Error closing client: %s \n", err)
			return
		}
	}()

	// Create a collection with array metadata
	col, err := client.GetOrCreateCollection(context.Background(), "array_metadata_example",
		chroma.WithCollectionMetadataCreate(
			chroma.NewMetadata(
				chroma.NewStringAttribute("description", "Collection demonstrating array metadata"),
				chroma.NewStringArrayAttribute("categories", []string{"science", "math", "history"}),
			),
		),
	)
	if err != nil {
		_ = client.Close()
		log.Printf("Error creating collection: %s \n", err)
		return
	}

	// Add documents with array metadata
	err = col.Add(context.Background(),
		chroma.WithIDs("doc1", "doc2", "doc3"),
		chroma.WithTexts(
			"The theory of relativity was published by Einstein",
			"Newton discovered gravity and calculus",
			"The French Revolution started in 1789",
		),
		chroma.WithMetadatas(
			chroma.NewDocumentMetadata(
				chroma.NewStringArrayAttribute("tags", []string{"physics", "science", "einstein"}),
				chroma.NewIntArrayAttribute("years", []int64{1905, 1915}),
				chroma.NewFloatArrayAttribute("scores", []float64{9.8, 9.5}),
			),
			chroma.NewDocumentMetadata(
				chroma.NewStringArrayAttribute("tags", []string{"physics", "math", "newton"}),
				chroma.NewIntArrayAttribute("years", []int64{1687}),
				chroma.NewFloatArrayAttribute("scores", []float64{9.9, 9.7}),
			),
			chroma.NewDocumentMetadata(
				chroma.NewStringArrayAttribute("tags", []string{"history", "revolution", "france"}),
				chroma.NewIntArrayAttribute("years", []int64{1789}),
				chroma.NewBoolArrayAttribute("verified", []bool{true, true}),
			),
		),
	)
	if err != nil {
		log.Printf("Error adding documents: %s \n", err)
		return
	}

	count, err := col.Count(context.Background())
	if err != nil {
		log.Printf("Error counting collection: %s \n", err)
		return
	}
	fmt.Printf("Collection has %d documents\n", count)

	// Query using $contains to filter on array metadata
	// Find documents where tags contain "physics"
	qr, err := col.Query(context.Background(),
		chroma.WithQueryTexts("scientific discoveries"),
		chroma.WithInclude(chroma.IncludeDocuments, chroma.IncludeMetadatas),
		chroma.WithWhere(
			chroma.MetadataContainsString(chroma.K("tags"), "physics"),
		),
	)
	if err != nil {
		log.Printf("Error querying collection: %s \n", err)
		return
	}
	fmt.Printf("\nDocuments with 'physics' tag:\n")
	for i, doc := range qr.GetDocumentsGroups()[0] {
		fmt.Printf("  %d: %s\n", i+1, doc)
	}

	// Query using $not_contains to exclude documents
	// Find documents where tags do NOT contain "history"
	qr, err = col.Query(context.Background(),
		chroma.WithQueryTexts("important events"),
		chroma.WithInclude(chroma.IncludeDocuments, chroma.IncludeMetadatas),
		chroma.WithWhere(
			chroma.MetadataNotContainsString(chroma.K("tags"), "history"),
		),
	)
	if err != nil {
		log.Printf("Error querying collection: %s \n", err)
		return
	}
	fmt.Printf("\nDocuments without 'history' tag:\n")
	for i, doc := range qr.GetDocumentsGroups()[0] {
		fmt.Printf("  %d: %s\n", i+1, doc)
	}

	// Combine array filter with other filters using And/Or
	qr, err = col.Query(context.Background(),
		chroma.WithQueryTexts("scientific work"),
		chroma.WithInclude(chroma.IncludeDocuments, chroma.IncludeMetadatas),
		chroma.WithWhere(
			chroma.And(
				chroma.MetadataContainsString(chroma.K("tags"), "science"),
				chroma.MetadataContainsInt(chroma.K("years"), 1905),
			),
		),
	)
	if err != nil {
		log.Printf("Error querying with combined filters: %s \n", err)
		return
	}
	fmt.Printf("\nDocuments with 'science' tag AND year 1905:\n")
	for i, doc := range qr.GetDocumentsGroups()[0] {
		fmt.Printf("  %d: %s\n", i+1, doc)
	}

	// Clean up
	err = col.Delete(context.Background(), chroma.WithIDs("doc1", "doc2", "doc3"))
	if err != nil {
		log.Printf("Error deleting documents: %s \n", err)
		return
	}
	fmt.Println("\nCleanup complete.")
}
