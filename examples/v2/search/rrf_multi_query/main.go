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
	client, err := v2.NewClient(v2.WithBasePath("http://localhost:8000"))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Get or create collection
	collectionName := "rrf_example"
	collection, err := client.GetOrCreateCollection(ctx, collectionName,
		v2.WithMetadata("description", "RRF multi-query search example"))
	if err != nil {
		log.Fatalf("Failed to get/create collection: %v", err)
	}

	// Add sample documents about AI and technology
	fmt.Println("Adding sample documents...")
	err = collection.Add(ctx,
		v2.WithTexts(
			"Artificial intelligence is transforming healthcare through diagnostic tools",
			"Machine learning algorithms power recommendation systems",
			"Deep neural networks enable autonomous vehicle navigation",
			"Natural language processing helps chatbots understand queries",
			"Computer vision enables facial recognition in security systems",
			"Reinforcement learning optimizes resource allocation in data centers",
			"Neural networks revolutionize image classification tasks",
			"AI-powered predictive analytics improve business decisions",
			"Deep learning models enhance speech recognition accuracy",
			"Machine learning detects fraud in financial transactions",
			"Artificial neural networks simulate human brain functions",
			"Automated machine learning simplifies model development",
		),
		v2.WithIDs("doc1", "doc2", "doc3", "doc4", "doc5", "doc6", "doc7", "doc8", "doc9", "doc10", "doc11", "doc12"),
		v2.WithMetadatas(
			v2.NewDocumentMetadataFromMap(map[string]interface{}{"domain": "healthcare", "tech": "ai"}),
			v2.NewDocumentMetadataFromMap(map[string]interface{}{"domain": "ecommerce", "tech": "ml"}),
			v2.NewDocumentMetadataFromMap(map[string]interface{}{"domain": "automotive", "tech": "dl"}),
			v2.NewDocumentMetadataFromMap(map[string]interface{}{"domain": "customer_service", "tech": "nlp"}),
			v2.NewDocumentMetadataFromMap(map[string]interface{}{"domain": "security", "tech": "cv"}),
			v2.NewDocumentMetadataFromMap(map[string]interface{}{"domain": "cloud", "tech": "rl"}),
			v2.NewDocumentMetadataFromMap(map[string]interface{}{"domain": "general", "tech": "dl"}),
			v2.NewDocumentMetadataFromMap(map[string]interface{}{"domain": "business", "tech": "ai"}),
			v2.NewDocumentMetadataFromMap(map[string]interface{}{"domain": "general", "tech": "dl"}),
			v2.NewDocumentMetadataFromMap(map[string]interface{}{"domain": "finance", "tech": "ml"}),
			v2.NewDocumentMetadataFromMap(map[string]interface{}{"domain": "research", "tech": "dl"}),
			v2.NewDocumentMetadataFromMap(map[string]interface{}{"domain": "general", "tech": "ml"}),
		),
	)
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}

	fmt.Println("\n========================================")
	fmt.Println("Example 1: Single Query (for comparison)")
	fmt.Println("========================================")

	// Single query for "neural networks"
	results, err := collection.Search(ctx,
		v2.WithSearchRankKnnTexts([]string{"neural networks"}, 5),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	printResults(results, "Single Query: 'neural networks'")

	fmt.Println("\n========================================")
	fmt.Println("Example 2: RRF with Two Queries")
	fmt.Println("========================================")

	// Create two different KNN searches
	query1 := &v2.KnnRank{
		QueryTexts: []string{"neural networks"},
		K:          10,
	}

	query2 := &v2.KnnRank{
		QueryTexts: []string{"machine learning algorithms"},
		K:          10,
	}

	// Combine with RRF
	results, err = collection.Search(ctx,
		v2.WithSearchRankRrf(
			[]v2.RankExpression{query1, query2},
			60,   // k parameter
			true, // normalize scores
		),
		v2.WithSearchLimit(5, 0),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Fatalf("RRF search failed: %v", err)
	}

	printResults(results, "RRF: 'neural networks' OR 'machine learning algorithms'")

	fmt.Println("\n========================================")
	fmt.Println("Example 3: RRF with Three Queries")
	fmt.Println("========================================")

	// Three different aspects of AI
	aiQuery := &v2.KnnRank{
		QueryTexts: []string{"artificial intelligence"},
		K:          10,
	}

	dlQuery := &v2.KnnRank{
		QueryTexts: []string{"deep learning"},
		K:          10,
	}

	mlQuery := &v2.KnnRank{
		QueryTexts: []string{"machine learning"},
		K:          10,
	}

	// Combine all three
	results, err = collection.Search(ctx,
		v2.WithSearchRankRrf(
			[]v2.RankExpression{aiQuery, dlQuery, mlQuery},
			60,
			true,
		),
		v2.WithSearchLimit(5, 0),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Fatalf("RRF search failed: %v", err)
	}

	printResults(results, "RRF: 'AI' OR 'deep learning' OR 'machine learning'")

	fmt.Println("\n========================================")
	fmt.Println("Example 4: RRF for Diverse Queries")
	fmt.Println("========================================")

	// Very different queries - testing RRF's ability to combine
	healthQuery := &v2.KnnRank{
		QueryTexts: []string{"healthcare medical diagnosis"},
		K:          10,
	}

	financeQuery := &v2.KnnRank{
		QueryTexts: []string{"financial fraud detection"},
		K:          10,
	}

	results, err = collection.Search(ctx,
		v2.WithSearchRankRrf(
			[]v2.RankExpression{healthQuery, financeQuery},
			60,
			true,
		),
		v2.WithSearchLimit(5, 0),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Fatalf("RRF search failed: %v", err)
	}

	printResults(results, "RRF: 'healthcare' OR 'finance' (diverse queries)")

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
