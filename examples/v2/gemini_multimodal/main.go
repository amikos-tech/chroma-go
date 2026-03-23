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
			embeddings.NewTextPart("A cat sitting on a windowsill"),
			embeddings.NewPartFromSource(
				embeddings.ModalityImage,
				embeddings.NewBinarySourceFromURL("https://upload.wikimedia.org/wikipedia/commons/thumb/3/3a/Cat03.jpg/1200px-Cat03.jpg"),
			),
		},
	}
	emb, err := ef.EmbedContent(context.Background(), content)
	if err != nil {
		log.Fatalf("Error embedding content: %s", err)
	}
	fmt.Printf("Single content embedding dimension: %d\n", len(emb.ArrayOfFloat32))

	// Embed a batch of content items with different modalities (image and video).
	contents := []embeddings.Content{
		{Parts: []embeddings.Part{embeddings.NewTextPart("A dog playing fetch")}},
		{Parts: []embeddings.Part{
			embeddings.NewPartFromSource(
				embeddings.ModalityImage,
				embeddings.NewBinarySourceFromURL("https://upload.wikimedia.org/wikipedia/commons/thumb/2/26/YellowLabradorLooking_new.jpg/1200px-YellowLabradorLooking_new.jpg"),
			),
		}},
		{Parts: []embeddings.Part{
			embeddings.NewTextPart("A short lecture on embeddings"),
			embeddings.NewPartFromSource(
				embeddings.ModalityVideo,
				embeddings.NewBinarySourceFromURL("https://example.com/lecture.mp4"),
			),
		}},
	}
	results, err := ef.EmbedContents(context.Background(), contents)
	if err != nil {
		log.Fatalf("Error embedding contents: %s", err)
	}
	fmt.Printf("Batch results: %d embeddings\n", len(results))
}
