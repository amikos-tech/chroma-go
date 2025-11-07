package main

import (
	"context"
	"fmt"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	ctx := context.Background()

	// Create client
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Get or create collection
	collectionName := "simple_knn_example"
	collection, err := client.GetOrCreateCollection(ctx, collectionName,
		v2.WithCollectionMetadataCreate(
			v2.NewMetadata(
				v2.NewStringAttribute("description", "Simple KNN search example"),
			),
		))
	if err != nil {
		client.Close()
		log.Fatalf("Failed to get/create collection: %v", err)
	}
	defer client.Close()

	// Add sample documents
	fmt.Println("Adding sample documents...")
	md1, err := v2.NewDocumentMetadataFromMap(map[string]interface{}{"category": "general", "topic": "animals"})
	if err != nil {
		log.Printf("Error creating metadata: %v\n", err)
		return
	}
	md2, err := v2.NewDocumentMetadataFromMap(map[string]interface{}{"category": "tech", "topic": "ai"})
	if err != nil {
		log.Printf("Error creating metadata: %v\n", err)
		return
	}
	md3, err := v2.NewDocumentMetadataFromMap(map[string]interface{}{"category": "tech", "topic": "programming"})
	if err != nil {
		log.Printf("Error creating metadata: %v\n", err)
		return
	}
	md4, err := v2.NewDocumentMetadataFromMap(map[string]interface{}{"category": "tech", "topic": "ai"})
	if err != nil {
		log.Printf("Error creating metadata: %v\n", err)
		return
	}
	md5, err := v2.NewDocumentMetadataFromMap(map[string]interface{}{"category": "tech", "topic": "ai"})
	if err != nil {
		log.Printf("Error creating metadata: %v\n", err)
		return
	}
	md6, err := v2.NewDocumentMetadataFromMap(map[string]interface{}{"category": "tech", "topic": "ai"})
	if err != nil {
		log.Printf("Error creating metadata: %v\n", err)
		return
	}
	md7, err := v2.NewDocumentMetadataFromMap(map[string]interface{}{"category": "tech", "topic": "ai"})
	if err != nil {
		log.Printf("Error creating metadata: %v\n", err)
		return
	}
	md8, err := v2.NewDocumentMetadataFromMap(map[string]interface{}{"category": "tech", "topic": "data"})
	if err != nil {
		log.Printf("Error creating metadata: %v\n", err)
		return
	}
	err = collection.Add(ctx,
		v2.WithTexts(
			"The quick brown fox jumps over the lazy dog",
			"Machine learning is a subset of artificial intelligence",
			"Python is a popular programming language for data science",
			"Deep learning uses neural networks to learn from data",
			"Natural language processing enables computers to understand human language",
			"Computer vision allows machines to interpret visual information",
			"Reinforcement learning trains agents through trial and error",
			"Data mining extracts patterns from large datasets",
		),
		v2.WithIDs("doc1", "doc2", "doc3", "doc4", "doc5", "doc6", "doc7", "doc8"),
		v2.WithMetadatas(md1, md2, md3, md4, md5, md6, md7, md8),
	)
	if err != nil {
		log.Printf("Failed to add documents: %v", err)
		return
	}

	fmt.Println("\n========================================")
	fmt.Println("Example 1: Basic KNN Search")
	fmt.Println("========================================")

	// Perform KNN search
	results, err := collection.Search(ctx,
		v2.WithSearchRankKnnTexts([]string{"artificial intelligence and machine learning"}, 3),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	printResults(results, "KNN Search for 'artificial intelligence and machine learning'")

	fmt.Println("\n========================================")
	fmt.Println("Example 2: KNN Search with Different Query")
	fmt.Println("========================================")

	// Search for programming-related content
	results, err = collection.Search(ctx,
		v2.WithSearchRankKnnTexts([]string{"programming languages and coding"}, 3),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	printResults(results, "KNN Search for 'programming languages and coding'")

	fmt.Println("\n========================================")
	fmt.Println("Example 3: KNN Search with More Results")
	fmt.Println("========================================")

	// Get top 5 results
	results, err = collection.Search(ctx,
		v2.WithSearchRankKnnTexts([]string{"neural networks and learning"}, 5),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	printResults(results, "KNN Search for 'neural networks and learning' (top 5)")

	// Clean up
	fmt.Println("\nCleaning up...")
	err = client.DeleteCollection(ctx, collectionName)
	if err != nil {
		log.Printf("Warning: Failed to delete collection: %v", err)
	}

	fmt.Println("Done!")
}

func printResults(results v2.SearchResult, title string) {
	fmt.Printf("\n%s\n", title)
	fmt.Println("----------------------------------------")

	idGroups := results.GetIDGroups()
	docGroups := results.GetDocumentsGroups()
	scoreGroups := results.GetScoresGroups()

	if len(idGroups) == 0 {
		fmt.Println("No results found")
		return
	}

	ids := idGroups[0]
	docs := docGroups[0]
	scores := scoreGroups[0]

	for i, id := range ids {
		fmt.Printf("\n%d. ID: %s\n", i+1, id)
		fmt.Printf("   Score: %.4f\n", scores[i])
		fmt.Printf("   Document: %s\n", docs[i].ContentString())
	}
}
