package main

import (
	"context"
	"fmt"
	"log"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	voyage "github.com/amikos-tech/chroma-go/pkg/embeddings/voyage"
)

func main() {
	// Create a VoyageAI embedding function with the multimodal model.
	// Set VOYAGE_API_KEY in your environment before running.
	ef, err := voyage.NewVoyageAIEmbeddingFunction(
		voyage.WithEnvAPIKey(),
		voyage.WithDefaultModel("voyage-multimodal-3.5"),
	)
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
	// Uses the small video (480x480, 2s) to stay within VoyageAI's 32K token context window.
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
				embeddings.NewBinarySourceFromFile("pkg/embeddings/testdata/the_pounce_small.mp4"),
			),
		}},
	}
	results, err := ef.EmbedContents(context.Background(), contents)
	if err != nil {
		log.Fatalf("Error embedding contents: %s", err)
	}
	fmt.Printf("Batch results: %d embeddings\n", len(results))
}
