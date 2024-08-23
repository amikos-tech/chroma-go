package jina

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJinaEmbeddingFunction(t *testing.T) {
	apiKey := os.Getenv("JINA_API_KEY")
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv("JINA_API_KEY")
	}

	t.Run("Test with defaults", func(t *testing.T) {
		ef, err := NewJinaEmbeddingFunction(WithAPIKey(apiKey))
		require.NoError(t, err)
		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
		}
		resp, err := ef.EmbedDocuments(context.Background(), documents)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Equal(t, 768, resp[0].Len())
	})

	t.Run("Test with env API key", func(t *testing.T) {
		ef, err := NewJinaEmbeddingFunction(WithEnvAPIKey())
		require.NoError(t, err)
		documents := []string{
			"Document 1 content here",
		}
		resp, err := ef.EmbedDocuments(context.Background(), documents)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp, 1)
		require.Equal(t, 768, resp[0].Len())
	})

	t.Run("Test with normalized off", func(t *testing.T) {
		ef, err := NewJinaEmbeddingFunction(WithEnvAPIKey(), WithNormalized(false))
		require.NoError(t, err)
		documents := []string{
			"Document 1 content here",
		}
		resp, err := ef.EmbedDocuments(context.Background(), documents)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp, 1)
		require.Equal(t, 768, resp[0].Len())
	})

	t.Run("Test with model", func(t *testing.T) {
		ef, err := NewJinaEmbeddingFunction(WithEnvAPIKey(), WithModel("jina-embeddings-v2-base-code"))
		require.NoError(t, err)
		documents := []string{
			"import chromadb;client=chromadb.Client();collection=client.get_or_create_collection('col_name')",
		}
		resp, err := ef.EmbedDocuments(context.Background(), documents)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp, 1)
		require.Equal(t, 768, resp[0].Len())
	})

	t.Run("Test with EmbeddingType float", func(t *testing.T) {
		ef, err := NewJinaEmbeddingFunction(WithEnvAPIKey(), WithEmbeddingType(EmbeddingTypeFloat))
		require.NoError(t, err)
		documents := []string{
			"Document 1 content here",
		}
		resp, err := ef.EmbedDocuments(context.Background(), documents)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp, 1)
		require.Equal(t, 768, resp[0].Len())
	})

	t.Run("Test with embedding endpoint", func(t *testing.T) {
		ef, err := NewJinaEmbeddingFunction(WithEnvAPIKey(), WithEmbeddingEndpoint(DefaultBaseAPIEndpoint))
		require.NoError(t, err)
		documents := []string{
			"Document 1 content here",
		}
		resp, err := ef.EmbedDocuments(context.Background(), documents)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp, 1)
		require.Equal(t, 768, resp[0].Len())
	})
}
