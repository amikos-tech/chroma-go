package main

import (
	"context"
	"fmt"
	"log"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	gemini "github.com/amikos-tech/chroma-go/pkg/embeddings/gemini"
)

func main() {
	// Create a Gemini embedding function using the default multimodal model (gemini-embedding-2-preview).
	// Set GEMINI_API_KEY in your environment before running.
	ef, err := gemini.NewGeminiEmbeddingFunction(gemini.WithEnvAPIKey())
	if err != nil {
		log.Fatalf("Error creating embedding function: %s", err)
	}

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
		log.Fatalf("Error embedding content: %s", err)
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
		log.Fatalf("Error embedding contents: %s", err)
	}
	fmt.Printf("Batch results: %d embeddings\n", len(results))

	_ = ef.Close()
}
