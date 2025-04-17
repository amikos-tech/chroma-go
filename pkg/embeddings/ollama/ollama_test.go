//go:build ef

package ollama

import (
	"context"
	"fmt"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/stretchr/testify/require"
	tcollama "github.com/testcontainers/testcontainers-go/modules/ollama"
	"io"
	"testing"
)

func Test_ollama(t *testing.T) {
	ctx := context.Background()
	ollamaContainer, err := tcollama.Run(ctx, "ollama/ollama:latest", tcollama.WithUseLocal("OLLAMA_DEBUG=true"))
	require.NoError(t, err)
	// Clean up the container
	defer func() {
		if err := ollamaContainer.Terminate(ctx); err != nil {
			fmt.Printf("failed to terminate container: %s\n", err)
		}
	}()

	model := "nomic-embed-text"
	c, out, err := ollamaContainer.Exec(ctx, []string{"ollama", "pull", model})
	require.NoError(t, err)
	require.Equal(t, c, 0)
	if out != nil {
		outs, err := io.ReadAll(out)
		require.NoError(t, err)
		require.Contains(t, string(outs), "success")
	}
	connectionStr, err := ollamaContainer.ConnectionString(ctx)
	require.NoError(t, err)
	client, err := NewOllamaClient(WithBaseURL(connectionStr), WithModel(embeddings.EmbeddingModel(model)))
	require.NoError(t, err)
	t.Run("Test Create Embed Single document", func(t *testing.T) {
		resp, rerr := client.createEmbedding(context.Background(), &CreateEmbeddingRequest{Model: "nomic-embed-text", Input: &EmbeddingInput{Input: "Document 1 content here"}})
		require.Nil(t, rerr)
		require.NotNil(t, resp)
	})
	t.Run("Test Create Embed multi-document", func(t *testing.T) {
		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
		}
		ef, err := NewOllamaEmbeddingFunction(WithBaseURL(connectionStr), WithModel(embeddings.EmbeddingModel(model)))
		require.NoError(t, err)
		resp, rerr := ef.EmbedDocuments(context.Background(), documents)
		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Equal(t, 2, len(resp))
	})
}
