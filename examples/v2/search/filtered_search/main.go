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
	collectionName := "filtered_search_example"
	collection, err := client.GetOrCreateCollection(ctx, collectionName,
		v2.WithCollectionMetadataCreate(
			v2.NewMetadata(
				v2.NewStringAttribute("description", "Filtered search example"),
			),
		))
	if err != nil {
		client.Close()
		log.Fatalf("Failed to get/create collection: %v", err)
	}
	defer client.Close()

	// Add sample documents with rich metadata
	fmt.Println("Adding sample documents with metadata...")
	md1, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"language": "python", "level": "beginner", "category": "tutorial", "rating": 4.5, "year": 2023,
	})
	md2, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"language": "python", "level": "advanced", "category": "tutorial", "rating": 4.8, "year": 2024,
	})
	md3, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"language": "javascript", "level": "beginner", "category": "tutorial", "rating": 4.2, "year": 2023,
	})
	md4, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"language": "javascript", "level": "intermediate", "category": "framework", "rating": 4.6, "year": 2024,
	})
	md5, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"language": "python", "level": "intermediate", "category": "ml", "rating": 4.7, "year": 2024,
	})
	md6, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"language": "python", "level": "advanced", "category": "ml", "rating": 4.9, "year": 2024,
	})
	md7, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"language": "go", "level": "beginner", "category": "tutorial", "rating": 4.4, "year": 2023,
	})
	md8, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"language": "go", "level": "advanced", "category": "architecture", "rating": 4.7, "year": 2024,
	})
	md9, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"language": "python", "level": "intermediate", "category": "web", "rating": 4.3, "year": 2023,
	})
	md10, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"language": "javascript", "level": "intermediate", "category": "web", "rating": 4.5, "year": 2024,
	})
	err = collection.Add(ctx,
		v2.WithTexts(
			"Introduction to Python programming for beginners",
			"Advanced Python techniques for data science",
			"JavaScript fundamentals and modern ES6 features",
			"React framework for building user interfaces",
			"Machine learning with Python and scikit-learn",
			"Deep learning with TensorFlow and Keras",
			"Getting started with Go programming language",
			"Building microservices with Go",
			"Python web development with Django framework",
			"Node.js backend development with Express",
		),
		v2.WithIDs("doc1", "doc2", "doc3", "doc4", "doc5", "doc6", "doc7", "doc8", "doc9", "doc10"),
		v2.WithMetadatas(md1, md2, md3, md4, md5, md6, md7, md8, md9, md10),
	)
	if err != nil {
		log.Printf("Failed to add documents: %v", err)
		return
	}

	fmt.Println("\n========================================")
	fmt.Println("Example 1: Unfiltered Search (baseline)")
	fmt.Println("========================================")

	// Search without filters
	results, err := collection.Search(ctx,
		v2.WithSearchRankKnnTexts([]string{"Python programming"}, 5),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	printResultsWithMetadata(results, collection, "Unfiltered search for 'Python programming'")

	fmt.Println("\n========================================")
	fmt.Println("Example 2: Filter by Language")
	fmt.Println("========================================")

	// Search only Python documents
	results, err = collection.Search(ctx,
		v2.WithSearchRankKnnTexts([]string{"programming tutorial"}, 5),
		v2.WithSearchWhere(v2.EqString("language", "python")),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	printResultsWithMetadata(results, collection, "Python documents only")

	fmt.Println("\n========================================")
	fmt.Println("Example 3: Filter by Level")
	fmt.Println("========================================")

	// Search only beginner content
	results, err = collection.Search(ctx,
		v2.WithSearchRankKnnTexts([]string{"programming guide"}, 5),
		v2.WithSearchWhere(v2.EqString("level", "beginner")),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	printResultsWithMetadata(results, collection, "Beginner level only")

	fmt.Println("\n========================================")
	fmt.Println("Example 4: Multiple Filters with AND")
	fmt.Println("========================================")

	// Python AND advanced level
	results, err = collection.Search(ctx,
		v2.WithSearchRankKnnTexts([]string{"advanced techniques"}, 5),
		v2.WithSearchWhere(v2.And(
			v2.EqString("language", "python"),
			v2.EqString("level", "advanced"),
		)),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	printResultsWithMetadata(results, collection, "Python AND advanced level")

	fmt.Println("\n========================================")
	fmt.Println("Example 5: Filter by Rating (numeric)")
	fmt.Println("========================================")

	// High-rated content (>= 4.5)
	results, err = collection.Search(ctx,
		v2.WithSearchRankKnnTexts([]string{"tutorial"}, 5),
		v2.WithSearchWhere(v2.GteFloat("rating", 4.5)),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	printResultsWithMetadata(results, collection, "Rating >= 4.5")

	fmt.Println("\n========================================")
	fmt.Println("Example 6: Complex Filter")
	fmt.Println("========================================")

	// (Python OR Go) AND (rating >= 4.5) AND (year = 2024)
	results, err = collection.Search(ctx,
		v2.WithSearchRankKnnTexts([]string{"programming"}, 5),
		v2.WithSearchWhere(v2.And(
			v2.Or(
				v2.EqString("language", "python"),
				v2.EqString("language", "go"),
			),
			v2.GteFloat("rating", 4.5),
			v2.EqInt("year", 2024),
		)),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	printResultsWithMetadata(results, collection, "(Python OR Go) AND rating>=4.5 AND year=2024")

	fmt.Println("\n========================================")
	fmt.Println("Example 7: Filter with IN operator")
	fmt.Println("========================================")

	// Category in [ml, web]
	results, err = collection.Search(ctx,
		v2.WithSearchRankKnnTexts([]string{"development framework"}, 5),
		v2.WithSearchWhere(v2.InString("category", "ml", "web")),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	printResultsWithMetadata(results, collection, "Category in [ml, web]")

	// Clean up
	fmt.Println("\nCleaning up...")
	err = client.DeleteCollection(ctx, collectionName)
	if err != nil {
		log.Printf("Warning: Failed to delete collection: %v", err)
	}

	fmt.Println("Done!")
}

func printResultsWithMetadata(results v2.SearchResult, collection v2.Collection, title string) {
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

	// Get full documents with metadata
	ctx := context.Background()
	fullDocs, err := collection.Get(ctx,
		v2.WithIDsGet(ids...),
		v2.WithIncludeGet(v2.IncludeMetadatas, v2.IncludeDocuments),
	)
	if err != nil {
		log.Printf("Warning: Failed to get metadata: %v", err)
	}

	for i, id := range ids {
		fmt.Printf("\n%d. ID: %s\n", i+1, id)
		fmt.Printf("   Score: %.4f\n", scores[i])
		fmt.Printf("   Document: %s\n", docs[i].ContentString())

		// Print metadata if available
		if fullDocs != nil && i < len(fullDocs.GetMetadatas()) {
			metadata := fullDocs.GetMetadatas()[i]
			if metadata != nil {
				if lang, ok := metadata.GetString("language"); ok {
					fmt.Printf("   Language: %s\n", lang)
				}
				if level, ok := metadata.GetString("level"); ok {
					fmt.Printf("   Level: %s\n", level)
				}
				if category, ok := metadata.GetString("category"); ok {
					fmt.Printf("   Category: %s\n", category)
				}
				if rating, ok := metadata.GetFloat("rating"); ok {
					fmt.Printf("   Rating: %.1f\n", rating)
				}
				if year, ok := metadata.GetInt("year"); ok {
					fmt.Printf("   Year: %d\n", year)
				}
			}
		}
	}
}
