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
	collectionName := "hybrid_search_example"
	collection, err := client.GetOrCreateCollection(ctx, collectionName,
		v2.WithCollectionMetadataCreate(
			v2.NewMetadata(
				v2.NewStringAttribute("description", "Hybrid search strategies example"),
			),
		))
	if err != nil {
		client.Close()
		log.Fatalf("Failed to get/create collection: %v", err)
	}
	defer client.Close()

	// Add sample e-commerce product documents
	fmt.Println("Adding product documents...")
	md1, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"category": "audio", "type": "headphones", "wireless": true, "price": 199.99, "rating": 4.7, "in_stock": true,
	})
	md2, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"category": "accessories", "type": "mouse", "wireless": true, "price": 49.99, "rating": 4.5, "in_stock": true,
	})
	md3, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"category": "accessories", "type": "keyboard", "wireless": false, "price": 129.99, "rating": 4.8, "in_stock": true,
	})
	md4, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"category": "audio", "type": "earbuds", "wireless": true, "price": 89.99, "rating": 4.4, "in_stock": true,
	})
	md5, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"category": "accessories", "type": "cable", "wireless": false, "price": 12.99, "rating": 4.3, "in_stock": true,
	})
	md6, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"category": "audio", "type": "speaker", "wireless": true, "price": 79.99, "rating": 4.6, "in_stock": false,
	})
	md7, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"category": "audio", "type": "headset", "wireless": true, "price": 149.99, "rating": 4.8, "in_stock": true,
	})
	md8, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"category": "accessories", "type": "charger", "wireless": true, "price": 39.99, "rating": 4.2, "in_stock": true,
	})
	md9, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"category": "audio", "type": "headphones", "wireless": false, "price": 299.99, "rating": 4.9, "in_stock": true,
	})
	md10, _ := v2.NewDocumentMetadataFromMap(map[string]interface{}{
		"category": "accessories", "type": "combo", "wireless": true, "price": 69.99, "rating": 4.4, "in_stock": true,
	})
	err = collection.Add(ctx,
		v2.WithTexts(
			"Premium wireless Bluetooth headphones with active noise cancellation and 30-hour battery life",
			"Ergonomic wireless mouse with precision tracking and customizable buttons for productivity",
			"Mechanical gaming keyboard with RGB backlighting and programmable macro keys",
			"High-quality wireless earbuds with deep bass and water resistance for sports",
			"USB-C fast charging cable with durable braided design and 6ft length",
			"Portable Bluetooth speaker with 360-degree sound and waterproof design",
			"Wireless gaming headset with surround sound and noise-canceling microphone",
			"Smart wireless charger with fast charging for multiple devices simultaneously",
			"Professional studio headphones with flat frequency response for audio production",
			"Compact wireless keyboard and mouse combo for travel and remote work",
		),
		v2.WithIDs("prod1", "prod2", "prod3", "prod4", "prod5", "prod6", "prod7", "prod8", "prod9", "prod10"),
		v2.WithMetadatas(md1, md2, md3, md4, md5, md6, md7, md8, md9, md10),
	)
	if err != nil {
		log.Printf("Failed to add documents: %v", err)
		return
	}

	fmt.Println("\n========================================")
	fmt.Println("Example 1: Semantic-Only Search")
	fmt.Println("========================================")

	// Pure semantic search
	results, err := collection.Search(ctx,
		v2.WithSearchRankKnnTexts([]string{"audio equipment for music"}, 5),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	printProductResults(results, collection, "Semantic search: 'audio equipment for music'")

	fmt.Println("\n========================================")
	fmt.Println("Example 2: Hybrid - Semantic + Business Rules")
	fmt.Println("========================================")

	// Semantic search + business filters
	results, err = collection.Search(ctx,
		v2.WithSearchRankKnnTexts([]string{"wireless headphones for gaming"}, 5),
		v2.WithSearchWhere(v2.And(
			v2.EqBool("in_stock", true),
			v2.GteFloat("rating", 4.5),
			v2.LteFloat("price", 200.0),
		)),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	printProductResults(results, collection, "Hybrid: semantic + business rules (in_stock, rating>=4.5, price<=200)")

	fmt.Println("\n========================================")
	fmt.Println("Example 3: Multi-Query RRF for Broader Matching")
	fmt.Println("========================================")

	// Combine multiple search strategies
	specificQuery := &v2.KnnRank{
		QueryTexts: []string{"wireless headphones"},
		K:          10,
	}

	broadQuery := &v2.KnnRank{
		QueryTexts: []string{"audio devices"},
		K:          10,
	}

	results, err = collection.Search(ctx,
		v2.WithSearchRankRrf(
			[]v2.RankExpression{specificQuery, broadQuery},
			60,
			true,
		),
		v2.WithSearchWhere(v2.EqString("category", "audio")),
		v2.WithSearchLimit(5, 0),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	printProductResults(results, collection, "RRF: specific ('wireless headphones') + broad ('audio devices')")

	fmt.Println("\n========================================")
	fmt.Println("Example 4: Hybrid with Score Transformation")
	fmt.Println("========================================")

	// Boost top results exponentially
	baseRank := &v2.KnnRank{
		QueryTexts: []string{"premium audio quality"},
		K:          10,
	}

	boostedRank := v2.ExpRank(baseRank)

	results, err = collection.Search(ctx,
		v2.WithSearchRank(boostedRank),
		v2.WithSearchWhere(v2.GteFloat("rating", 4.5)),
		v2.WithSearchLimit(5, 0),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	printProductResults(results, collection, "Exponential boost on 'premium audio quality' + rating filter")

	fmt.Println("\n========================================")
	fmt.Println("Example 5: Intent-Based Hybrid Search")
	fmt.Println("========================================")

	// User intent: "wireless accessories for work"
	// Strategy: Combine wireless devices + work-related queries

	wirelessQuery := &v2.KnnRank{
		QueryTexts: []string{"wireless Bluetooth devices"},
		K:          10,
	}

	workQuery := &v2.KnnRank{
		QueryTexts: []string{"productivity office remote work"},
		K:          10,
	}

	results, err = collection.Search(ctx,
		v2.WithSearchRankRrf(
			[]v2.RankExpression{wirelessQuery, workQuery},
			60,
			true,
		),
		v2.WithSearchWhere(v2.And(
			v2.EqBool("wireless", true),
			v2.LteFloat("price", 100.0), // Budget constraint
		)),
		v2.WithSearchLimit(5, 0),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	printProductResults(results, collection, "Intent-based: wireless + work-related + budget constraint")

	fmt.Println("\n========================================")
	fmt.Println("Example 6: Complex Hybrid - Multiple Strategies")
	fmt.Println("========================================")

	// User searches: "best wireless audio"
	// Strategy: Product name match + category match + reviews

	productNameQuery := &v2.KnnRank{
		QueryTexts: []string{"wireless headphones earbuds speaker"},
		K:          10,
	}

	categoryQuery := &v2.KnnRank{
		QueryTexts: []string{"audio equipment sound quality"},
		K:          10,
	}

	reviewQuery := &v2.KnnRank{
		QueryTexts: []string{"premium high-quality professional"},
		K:          10,
	}

	results, err = collection.Search(ctx,
		v2.WithSearchRankRrf(
			[]v2.RankExpression{productNameQuery, categoryQuery, reviewQuery},
			60,
			true,
		),
		v2.WithSearchWhere(v2.And(
			v2.EqString("category", "audio"),
			v2.EqBool("wireless", true),
			v2.GteFloat("rating", 4.6),
		)),
		v2.WithSearchLimit(5, 0),
		v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
	)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	printProductResults(results, collection, "Complex: product name + category + quality signals + filters")

	// Clean up
	fmt.Println("\nCleaning up...")
	err = client.DeleteCollection(ctx, collectionName)
	if err != nil {
		log.Printf("Warning: Failed to delete collection: %v", err)
	}

	fmt.Println("Done!")
}

func printProductResults(results v2.SearchResult, collection v2.Collection, title string) {
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

	// Get product metadata
	ctx := context.Background()
	products, err := collection.Get(ctx,
		v2.WithIDsGet(ids...),
		v2.WithIncludeGet(v2.IncludeMetadatas),
	)
	if err != nil {
		log.Printf("Warning: Failed to get metadata: %v", err)
	}

	for i, id := range ids {
		fmt.Printf("\n%d. Product ID: %s\n", i+1, id)
		fmt.Printf("   Score: %.4f\n", scores[i])
		fmt.Printf("   Description: %s\n", docs[i].ContentString())

		if products != nil && i < len(products.GetMetadatas()) {
			metadata := products.GetMetadatas()[i]
			if metadata != nil {
				if price, ok := metadata.GetFloat("price"); ok {
					fmt.Printf("   Price: $%.2f\n", price)
				}
				if rating, ok := metadata.GetFloat("rating"); ok {
					fmt.Printf("   Rating: %.1f/5.0\n", rating)
				}
				if inStock, ok := metadata.GetBool("in_stock"); ok {
					stockStatus := "In Stock"
					if !inStock {
						stockStatus = "Out of Stock"
					}
					fmt.Printf("   Availability: %s\n", stockStatus)
				}
				if category, ok := metadata.GetString("category"); ok {
					fmt.Printf("   Category: %s\n", category)
				}
			}
		}
	}
}
