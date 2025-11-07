package main

import (
	"context"
	"fmt"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// This example demonstrates how to use the new Search API with rank expressions
func main() {
	ctx := context.Background()

	// Create a client (adjust the URL as needed)
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Get or create a collection
	collection, err := client.GetOrCreateCollection(ctx, "search_example_collection")
	if err != nil {
		client.Close()
		log.Fatalf("Failed to get/create collection: %v", err)
	}
	defer client.Close()

	// Add some sample documents
	md1, err := v2.NewDocumentMetadataFromMap(map[string]interface{}{"category": "animals", "score": 10})
	if err != nil {
		log.Printf("Error creating metadata: %v\n", err)
		return
	}
	md2, err := v2.NewDocumentMetadataFromMap(map[string]interface{}{"category": "tech", "score": 20})
	if err != nil {
		log.Printf("Error creating metadata: %v\n", err)
		return
	}
	md3, err := v2.NewDocumentMetadataFromMap(map[string]interface{}{"category": "tech", "score": 15})
	if err != nil {
		log.Printf("Error creating metadata: %v\n", err)
		return
	}
	err = collection.Add(ctx,
		v2.WithTexts(
			"The quick brown fox jumps over the lazy dog",
			"Machine learning is a subset of artificial intelligence",
			"Python is a popular programming language for data science",
		),
		v2.WithIDs("doc1", "doc2", "doc3"),
		v2.WithMetadatas(md1, md2, md3),
	)
	if err != nil {
		log.Printf("Failed to add documents: %v", err)
		return
	}

	// Example 1: Simple KNN search with query text
	fmt.Println("=== Example 1: Simple KNN Search ===")
	results, err := collection.Search(ctx,
		v2.WithSearchRankKnnTexts([]string{"artificial intelligence"}, 2),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}
	printSearchResults(results, "KNN Search")

	// Example 2: KNN search with where filter
	fmt.Println("\n=== Example 2: KNN Search with Filter ===")
	results, err = collection.Search(ctx,
		v2.WithSearchRankKnnTexts([]string{"programming"}, 5),
		v2.WithSearchWhere(v2.EqString("category", "tech")),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search with filter failed: %v", err)
		return
	}
	printSearchResults(results, "Filtered Search")

	// Example 3: KNN search with limit and offset
	fmt.Println("\n=== Example 3: KNN Search with Pagination ===")
	results, err = collection.Search(ctx,
		v2.WithSearchRankKnnTexts([]string{"data"}, 10),
		v2.WithSearchLimit(1, 1), // limit=1, offset=1 (get second result)
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search with pagination failed: %v", err)
		return
	}
	printSearchResults(results, "Paginated Search")

	// Example 4: KNN search with embeddings directly
	fmt.Println("\n=== Example 4: KNN Search with Embeddings ===")
	// Create a sample embedding (in practice, use an embedding function)
	queryEmbedding := embeddings.NewEmbeddingFromFloat32(make([]float32, 768)) // Adjust dimension as needed
	results, err = collection.Search(ctx,
		v2.WithSearchRankKnn([]embeddings.Embedding{queryEmbedding}, 3),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search with embeddings failed: %v", err)
		return
	}
	printSearchResults(results, "Embedding Search")

	// Example 5: Reciprocal Rank Fusion (RRF) combining multiple KNN searches
	fmt.Println("\n=== Example 5: RRF Search ===")
	rank1 := &v2.KnnRank{
		QueryTexts: []string{"artificial intelligence"},
		K:          5,
	}
	rank2 := &v2.KnnRank{
		QueryTexts: []string{"machine learning"},
		K:          5,
	}
	results, err = collection.Search(ctx,
		v2.WithSearchRankRrf([]v2.RankExpression{rank1, rank2}, 60, true),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("RRF search failed: %v", err)
		return
	}
	printSearchResults(results, "RRF Search")

	// Example 6: Arithmetic rank operations
	fmt.Println("\n=== Example 6: Arithmetic Rank Operations ===")
	knn1 := &v2.KnnRank{
		QueryTexts: []string{"programming"},
		K:          5,
	}
	knn2 := &v2.KnnRank{
		QueryTexts: []string{"data science"},
		K:          5,
	}
	// Average of two KNN searches
	avgRank := v2.DivRanks(v2.AddRanks(knn1, knn2), knn1) // (knn1 + knn2) / knn1
	results, err = collection.Search(ctx,
		v2.WithSearchRank(avgRank),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Arithmetic rank search failed: %v", err)
		return
	}
	printSearchResults(results, "Arithmetic Rank Search")

	// Clean up
	err = client.DeleteCollection(ctx, "search_example_collection")
	if err != nil {
		log.Printf("Warning: Failed to delete collection: %v", err)
	}
}

func printSearchResults(results v2.SearchResult, title string) {
	fmt.Printf("--- %s Results ---\n", title)
	idGroups := results.GetIDGroups()
	docGroups := results.GetDocumentsGroups()
	scoreGroups := results.GetScoresGroups()

	for i := 0; i < len(idGroups); i++ {
		fmt.Printf("Query %d results:\n", i+1)
		ids := idGroups[i]
		var docs v2.Documents
		if i < len(docGroups) {
			docs = docGroups[i]
		}
		var scores embeddings.Distances
		if i < len(scoreGroups) {
			scores = scoreGroups[i]
		}

		for j, id := range ids {
			fmt.Printf("  %d. ID: %s", j+1, id)
			if docs != nil && j < len(docs) {
				fmt.Printf(", Document: %s", docs[j].ContentString())
			}
			if scores != nil && j < len(scores) {
				fmt.Printf(", Score: %.4f", scores[j])
			}
			fmt.Println()
		}
	}
}
