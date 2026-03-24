package main

import (
	"context"
	"fmt"
	"log"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	gemini "github.com/amikos-tech/chroma-go/pkg/embeddings/gemini"
)

// Run from the repository root: go run ./examples/v2/gemini_multimodal
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Create a Gemini embedding function using the default multimodal model (gemini-embedding-2-preview).
	// Set GEMINI_API_KEY in your environment before running.
	ef, err := gemini.NewGeminiEmbeddingFunction(gemini.WithEnvAPIKey())
	if err != nil {
		return fmt.Errorf("error creating embedding function: %w", err)
	}
	defer ef.Close()

	// Embed a single content item with text and an image.
	content := embeddings.Content{
		Parts: []embeddings.Part{
			embeddings.NewTextPart("A lioness hunting at sunset"),
			embeddings.NewPartFromSource(
				embeddings.ModalityImage,
				embeddings.NewBinarySourceFromFile("pkg/embeddings/testdata/lioness.png"),
			),
		},
	}
	emb, err := ef.EmbedContent(context.Background(), content)
	if err != nil {
		return fmt.Errorf("error embedding content: %w", err)
	}
	fmt.Printf("Single content embedding dimension: %d\n", emb.Len())

	// Embed a batch of content items with different modalities.
	contents := []embeddings.Content{
		{Parts: []embeddings.Part{embeddings.NewTextPart("The golden hour on the Serengeti")}},
		{Parts: []embeddings.Part{
			embeddings.NewPartFromSource(
				embeddings.ModalityImage,
				embeddings.NewBinarySourceFromFile("pkg/embeddings/testdata/lioness.png"),
			),
		}},
		{Parts: []embeddings.Part{
			embeddings.NewTextPart("A lioness pouncing on prey"),
			embeddings.NewPartFromSource(
				embeddings.ModalityVideo,
				embeddings.NewBinarySourceFromFile("pkg/embeddings/testdata/the_pounce.mp4"),
			),
		}},
	}
	results, err := ef.EmbedContents(context.Background(), contents)
	if err != nil {
		return fmt.Errorf("error embedding contents: %w", err)
	}
	fmt.Printf("Batch results: %d embeddings\n", len(results))

	return nil
}
