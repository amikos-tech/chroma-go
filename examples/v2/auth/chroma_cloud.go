package main

import (
	"context"
	"fmt"
	"log"
	"os"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Get cloud configuration from environment variables
	apiKey := os.Getenv("CHROMA_CLOUD_API_KEY")
	if apiKey == "" {
		log.Fatal("CHROMA_CLOUD_API_KEY environment variable is required")
	}

	host := getEnvOrDefault("CHROMA_CLOUD_HOST", "api.trychroma.com")
	tenant := getEnvOrDefault("CHROMA_CLOUD_TENANT", "default-tenant")
	database := getEnvOrDefault("CHROMA_CLOUD_DATABASE", "default-database")

	// Create Chroma Cloud client
	client, err := v2.NewCloudClient(
		v2.WithCloudHost(host),
		v2.WithCloudAPIKey(apiKey),
		v2.WithCloudTenant(tenant),
		v2.WithCloudDatabase(database),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Test the connection
	if err := client.Heartbeat(context.Background()); err != nil {
		log.Fatal("Failed to connect to Chroma Cloud:", err)
	}

	fmt.Println("Successfully connected to Chroma Cloud")
	fmt.Printf("Host: %s\n", host)
	fmt.Printf("Tenant: %s\n", tenant)
	fmt.Printf("Database: %s\n", database)

	// List collections in the cloud
	collections, err := client.ListCollections(context.Background())
	if err != nil {
		log.Fatal("Failed to list collections:", err)
	}

	fmt.Printf("\nFound %d collections:\n", len(collections))
	for _, col := range collections {
		fmt.Printf("- %s (ID: %s)\n", col.Name(), col.ID())
	}

	// Create a new collection in the cloud
	collectionName := "cloud_example_collection"
	collection, err := client.GetOrCreateCollection(
		context.Background(),
		collectionName,
		v2.WithDescription("Collection in Chroma Cloud"),
		v2.WithEmbeddingDimension(1536), // OpenAI dimension
	)
	if err != nil {
		log.Fatal("Failed to create collection:", err)
	}

	fmt.Printf("\nUsing collection: %s\n", collection.Name())

	// Add sample data
	err = collection.Add(
		context.Background(),
		[]string{"cloud-doc-1", "cloud-doc-2", "cloud-doc-3"},
		nil, // let Chroma generate embeddings
		[]map[string]interface{}{
			{"source": "cloud", "category": "demo"},
			{"source": "cloud", "category": "test"},
			{"source": "cloud", "category": "example"},
		},
		[]string{
			"This is a document stored in Chroma Cloud",
			"Cloud-based vector database example",
			"Scalable vector search with Chroma Cloud",
		},
	)
	if err != nil {
		log.Fatal("Failed to add documents:", err)
	}

	fmt.Printf("Successfully added 3 documents to cloud collection\n")

	// Query the collection
	results, err := collection.Query(
		context.Background(),
		[]string{"vector database cloud"},
		2, // return top 2 results
		nil,
		nil,
		v2.WithInclude(v2.IDocuments, v2.IMetadatas, v2.IDistances),
	)
	if err != nil {
		log.Fatal("Failed to query collection:", err)
	}

	fmt.Printf("\nQuery results:\n")
	for i := range results.Ids[0] {
		fmt.Printf("%d. ID: %s\n", i+1, results.Ids[0][i])
		fmt.Printf("   Document: %s\n", results.Documents[0][i])
		fmt.Printf("   Distance: %.4f\n", results.Distances[0][i])
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
