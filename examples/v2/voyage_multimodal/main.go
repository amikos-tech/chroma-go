package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	voyage "github.com/amikos-tech/chroma-go/pkg/embeddings/voyage"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// findRepoRoot walks up from the current working directory looking for go.mod.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root (no go.mod found)")
		}
		dir = parent
	}
}

func run() error {
	root, err := findRepoRoot()
	if err != nil {
		return err
	}
	testdata := filepath.Join(root, "pkg", "embeddings", "testdata")

	// Create a VoyageAI embedding function with the multimodal model.
	// Set VOYAGE_API_KEY in your environment before running.
	ef, err := voyage.NewVoyageAIEmbeddingFunction(
		voyage.WithEnvAPIKey(),
		voyage.WithDefaultModel("voyage-multimodal-3.5"),
	)
	if err != nil {
		return fmt.Errorf("error creating embedding function: %w", err)
	}

	// Embed a single content item with text and an image.
	content := embeddings.NewContent([]embeddings.Part{
		embeddings.NewTextPart("A lioness hunting at sunset"),
		embeddings.NewPartFromSource(
			embeddings.ModalityImage,
			embeddings.NewBinarySourceFromFile(filepath.Join(testdata, "lioness.png")),
		),
	})
	emb, err := ef.EmbedContent(context.Background(), content)
	if err != nil {
		return fmt.Errorf("error embedding content: %w", err)
	}
	fmt.Printf("Single content embedding dimension: %d\n", emb.Len())

	// Embed a batch of content items with different modalities.
	// Uses the small video (480x480, 2s) to stay within VoyageAI's 32K token context window.
	contents := []embeddings.Content{
		embeddings.NewTextContent("The golden hour on the Serengeti"),
		embeddings.NewImageFile(filepath.Join(testdata, "lioness.png")),
		embeddings.NewContent([]embeddings.Part{
			embeddings.NewTextPart("A lioness pouncing on prey"),
			embeddings.NewPartFromSource(
				embeddings.ModalityVideo,
				embeddings.NewBinarySourceFromFile(filepath.Join(testdata, "the_pounce_small.mp4")),
			),
		}),
	}
	results, err := ef.EmbedContents(context.Background(), contents)
	if err != nil {
		return fmt.Errorf("error embedding contents: %w", err)
	}
	fmt.Printf("Batch results: %d embeddings\n", len(results))

	return nil
}
