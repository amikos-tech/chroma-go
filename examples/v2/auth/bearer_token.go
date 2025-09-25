package main

import (
	"context"
	"fmt"
	"log"
	"os"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Get token from environment or use default
	token := os.Getenv("CHROMA_AUTH_TOKEN")
	if token == "" {
		token = "test-token-000000000000000000"
	}

	// Create client with Bearer token authentication
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://localhost:8000"),
		v2.WithAuth(v2.NewTokenAuthCredentialsProvider(token, v2.AuthorizationTokenHeader)),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Test the connection
	if err := client.Heartbeat(context.Background()); err != nil {
		log.Fatal("Failed to connect:", err)
	}

	// List collections
	collections, err := client.ListCollections(context.Background())
	if err != nil {
		log.Fatal("Failed to list collections:", err)
	}
	fmt.Printf("Found %d collections\n", len(collections))
	for _, col := range collections {
		fmt.Printf("- %s\n", col.Name())
	}
	fmt.Println("Successfully connected with Bearer token authentication")
}
