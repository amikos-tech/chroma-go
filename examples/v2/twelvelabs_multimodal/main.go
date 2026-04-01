//go:build ef

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	twelvelabs "github.com/amikos-tech/chroma-go/pkg/embeddings/twelvelabs"
)

func run() error {
	if os.Getenv("TWELVE_LABS_API_KEY") == "" {
		return fmt.Errorf("TWELVE_LABS_API_KEY environment variable not set")
	}

	ef, err := twelvelabs.NewTwelveLabsEmbeddingFunction(
		twelvelabs.WithEnvAPIKey(),
	)
	if err != nil {
		return fmt.Errorf("failed to create embedding function: %w", err)
	}

	ctx := context.Background()

	// Text embedding
	textEmb, err := ef.EmbedContent(ctx, embeddings.NewTextContent("Twelve Labs multimodal embeddings"))
	if err != nil {
		return fmt.Errorf("text embedding failed: %w", err)
	}
	fmt.Printf("Text embedding dimensions: %d\n", textEmb.Len())

	// Image embedding from URL
	imgEmb, err := ef.EmbedContent(ctx, embeddings.NewImageURL("https://picsum.photos/id/237/200/300"))
	if err != nil {
		return fmt.Errorf("image embedding failed: %w", err)
	}
	fmt.Printf("Image embedding dimensions: %d\n", imgEmb.Len())

	fmt.Println("Twelve Labs multimodal embedding example completed successfully")
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
