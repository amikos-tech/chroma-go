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

	tenant := getEnvOrDefault("CHROMA_CLOUD_TENANT", "default-tenant")
	database := getEnvOrDefault("CHROMA_CLOUD_DATABASE", "default-database")

	// Create Chroma Cloud client
	client, err := v2.NewCloudAPIClient(
		v2.WithCloudAPIKey(apiKey),
		v2.WithDatabaseAndTenant(tenant, database),
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
	)
	if err != nil {
		log.Fatal("Failed to create collection:", err)
	}

	fmt.Printf("\nUsing collection: %s\n", collection.Name())

}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
