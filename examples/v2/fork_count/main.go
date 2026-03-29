// Package main demonstrates Chroma Cloud collection forking and fork counts.
//
// Requirements:
// - Chroma Cloud account with API key
// - Environment variables: CHROMA_CLOUD_API_KEY, CHROMA_CLOUD_TENANT, CHROMA_CLOUD_DATABASE
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	apiKey := os.Getenv("CHROMA_CLOUD_API_KEY")
	tenant := os.Getenv("CHROMA_CLOUD_TENANT")
	database := os.Getenv("CHROMA_CLOUD_DATABASE")
	if apiKey == "" || tenant == "" || database == "" {
		return fmt.Errorf("set CHROMA_CLOUD_API_KEY, CHROMA_CLOUD_TENANT, and CHROMA_CLOUD_DATABASE")
	}

	client, err := v2.NewCloudClient(
		v2.WithCloudAPIKey(apiKey),
		v2.WithDatabaseAndTenant(database, tenant),
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	source, err := client.CreateCollection(ctx, "fork-count-demo")
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer cleanupCollection(ctx, client, "fork-count-demo")

	forked, err := source.Fork(ctx, "fork-count-demo-fork")
	if err != nil {
		return fmt.Errorf("failed to fork collection: %w", err)
	}
	defer cleanupCollection(ctx, client, "fork-count-demo-fork")

	// ForkCount is lineage-wide: both source and fork report the same count
	sourceCount, err := source.ForkCount(ctx)
	if err != nil {
		return fmt.Errorf("failed to get source fork count: %w", err)
	}

	forkedCount, err := forked.ForkCount(ctx)
	if err != nil {
		return fmt.Errorf("failed to get forked fork count: %w", err)
	}

	fmt.Printf("Source fork count: %d\n", sourceCount)
	fmt.Printf("Forked fork count: %d\n", forkedCount)
	fmt.Printf("Both report the same lineage-wide count: %v\n", sourceCount == forkedCount)

	return nil
}

func cleanupCollection(ctx context.Context, client v2.Client, name string) {
	if err := client.DeleteCollection(ctx, name); err != nil {
		log.Printf("cleanup: failed to delete collection %q: %v", name, err)
	}
}
