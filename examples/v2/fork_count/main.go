package main

import (
	"context"
	"fmt"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	client, err := v2.NewHTTPClient(v2.WithBaseURL("http://localhost:8000"))
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Create a source collection
	source, err := client.CreateCollection(ctx, "fork-count-demo")
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer client.DeleteCollection(ctx, "fork-count-demo") //nolint:errcheck

	// Fork the collection
	forked, err := source.Fork(ctx, "fork-count-demo-fork")
	if err != nil {
		return fmt.Errorf("failed to fork collection: %w", err)
	}
	defer client.DeleteCollection(ctx, "fork-count-demo-fork") //nolint:errcheck

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
