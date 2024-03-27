package ollama

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcollama "github.com/testcontainers/testcontainers-go/modules/ollama"
)

func Test_ollama(t *testing.T) {
	ctx := context.Background()
	ollamaContainer, err := tcollama.RunContainer(ctx, testcontainers.WithImage("ollama/ollama:latest"))
	require.NoError(t, err)
	// Clean up the container
	defer func() {
		if err := ollamaContainer.Terminate(ctx); err != nil {
			fmt.Printf("failed to terminate container: %s\n", err)
		}
	}()

	model := "nomic-embed-text"
	_, _, err = ollamaContainer.Exec(ctx, []string{"ollama", "pull", model})
	require.NoError(t, err)
	connectionStr, err := ollamaContainer.ConnectionString(ctx)
	require.NoError(t, err)
	client, _ := NewOllamaClient(WithBaseURL(connectionStr+"/api/embeddings"), WithModel("nomic-embed-text"))
	t.Run("Test Create Embed", func(t *testing.T) {
		resp, rerr := client.createEmbedding(context.Background(), &CreateEmbeddingRequest{Model: "nomic-embed-text", Prompt: "Document 1 content here"})
		require.Nil(t, rerr)
		require.NotNil(t, resp)
	})
	t.Run("Test Create Embed", func(t *testing.T) {
		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
		}
		ef, err := NewOllamaEmbeddingFunction(WithBaseURL(connectionStr+"/api/embeddings"), WithModel("nomic-embed-text"))
		require.NoError(t, err)

		resp, rerr := ef.EmbedDocuments(context.Background(), documents)
		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
	})
}
