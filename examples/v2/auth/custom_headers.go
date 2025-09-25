package main

import (
	"context"
	"fmt"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Build custom headers
	headers := map[string]string{
		"Authorization":   "Bearer " + getEnvOrDefault("AUTH_TOKEN", "custom-token"),
		"X-API-Key":       getEnvOrDefault("API_KEY", "api-key-123"),
		"X-Request-ID":    "req-001",
		"X-Custom-Header": "custom-value",
	}

	// Create client with custom headers
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://localhost:8000"),
		v2.WithDefaultHeaders(headers),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Test the connection
	if err := client.Heartbeat(context.Background()); err != nil {
		log.Fatal("Failed to connect:", err)
	}

	fmt.Println("Successfully connected with custom headers")

	// Create or get collection
	collection, err := client.GetOrCreateCollection(
		context.Background(),
		"custom_headers_collection",
	)
	if err != nil {
		log.Fatal("Failed to get/create collection:", err)
	}

	fmt.Printf("Using collection: %s (ID: %s)\n", collection.Name(), collection.ID())
}
