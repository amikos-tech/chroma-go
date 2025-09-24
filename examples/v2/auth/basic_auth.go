package main

import (
	"context"
	"fmt"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Create client with basic authentication
	client, err := v2.NewHTTPClient(
		v2.WithBaseURL("http://localhost:8000"),
		v2.WithAuth(v2.NewBasicAuthCredentialsProvider("admin", "password")),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Test the connection
	if err := client.Heartbeat(context.Background()); err != nil {
		log.Fatal("Failed to connect:", err)
	}

	fmt.Println("Successfully connected with basic authentication")

	// Create a collection
	collection, err := client.CreateCollection(
		context.Background(),
		"test_collection",
		v2.WithDescription("Collection created with basic auth"),
	)
	if err != nil {
		log.Fatal("Failed to create collection:", err)
	}

	fmt.Printf("Created collection: %s\n", collection.Name())
}
