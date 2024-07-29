//go:build ef

package mistral

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"testing"
	"time"
)

func Test_nomic_client(t *testing.T) {
	apiKey := os.Getenv(APIKeyEnvVar)
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv(APIKeyEnvVar)
	}
	client, err := NewNomicClient(WithEnvAPIKey())
	require.NoError(t, err)

	t.Run("Test CreateEmbedding", func(t *testing.T) {
		req := CreateEmbeddingRequest{
			Model: DefaultEmbeddingModel,
			Texts: []string{"Test document"},
		}
		resp, rerr := client.CreateEmbedding(context.Background(), req)
		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 1)
		require.Len(t, *resp[0].ArrayOfFloat32, 768)
		time.Sleep(2 * time.Second)
	})
}

func Test_nomic_embedding_function(t *testing.T) {
	apiKey := os.Getenv(APIKeyEnvVar)
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv(APIKeyEnvVar)
	}

	t.Run("Test EmbedDocuments with env-based api key", func(t *testing.T) {
		embeddingFunction, err := NewNomicEmbeddingFunction(WithEnvAPIKey())
		require.NoError(t, err)
		resp, rerr := embeddingFunction.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})
		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Len(t, *resp[0].ArrayOfFloat32, 768)
		time.Sleep(2 * time.Second)
	})

	t.Run("Test EmbedDocuments with provided API key", func(t *testing.T) {
		embeddingFunction, err := NewNomicEmbeddingFunction(WithAPIKey(apiKey))
		require.NoError(t, err)
		resp, rerr := embeddingFunction.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Len(t, *resp[0].ArrayOfFloat32, 768)

	})

	t.Run("Test EmbedDocuments with provided model", func(t *testing.T) {
		embeddingFunction, err := NewNomicEmbeddingFunction(WithEnvAPIKey(), WithDefaultModel(NomicEmbedTextV15))

		require.NoError(t, err)
		resp, rerr := embeddingFunction.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Len(t, *resp[0].ArrayOfFloat32, 768)
		time.Sleep(2 * time.Second)
	})

	t.Run("Test EmbedQuery", func(t *testing.T) {
		embeddingFunction, err := NewNomicEmbeddingFunction(WithEnvAPIKey(), WithDefaultModel(DefaultEmbeddingModel))
		require.NoError(t, err)
		resp, rerr := embeddingFunction.EmbedQuery(context.Background(), "this is my query")
		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, *resp.ArrayOfFloat32, 768)
		time.Sleep(2 * time.Second)
	})

	t.Run("Test wrong model", func(t *testing.T) {
		embeddingFunction, err := NewNomicEmbeddingFunction(WithEnvAPIKey(), WithDefaultModel("model-does-not-exist"))
		require.NoError(t, err)
		_, rerr := embeddingFunction.EmbedQuery(context.Background(), "this is my query")
		require.Contains(t, rerr.Error(), "422 Unprocessable Entity")
		require.Error(t, rerr)
	})

	t.Run("Test wrong API key", func(t *testing.T) {
		embeddingFunction, err := NewNomicEmbeddingFunction(WithAPIKey("wrong-api-key"))
		require.NoError(t, err)
		_, rerr := embeddingFunction.EmbedQuery(context.Background(), "this is my query")
		require.Contains(t, rerr.Error(), "403 Forbidden")
		require.Error(t, rerr)
	})

	t.Run("Test with BaseURL", func(t *testing.T) {
		embeddingFunction, err := NewNomicEmbeddingFunction(WithEnvAPIKey(), WithBaseURL(DefaultBaseURL))
		require.NoError(t, err)
		resp, rerr := embeddingFunction.EmbedQuery(context.Background(), "this is my query")
		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, *resp.ArrayOfFloat32, 768)
		time.Sleep(2 * time.Second)
	})

	t.Run("Test with max batch size", func(t *testing.T) {
		embeddingFunction, err := NewNomicEmbeddingFunction(WithEnvAPIKey(), WithMaxBatchSize(2))
		require.NoError(t, err)
		_, rerr := embeddingFunction.EmbedDocuments(context.Background(), []string{"Test document", "Another test document", "Another test document", "Another test document"})
		require.Contains(t, rerr.Error(), "exceeds the maximum batch size")
		require.Error(t, rerr)
	})

	t.Run("Test with http client", func(t *testing.T) {
		embeddingFunction, err := NewNomicEmbeddingFunction(WithEnvAPIKey(), WithHTTPClient(http.DefaultClient))
		require.NoError(t, err)
		resp, rerr := embeddingFunction.EmbedQuery(context.Background(), "this is my query")
		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, *resp.ArrayOfFloat32, 768)
		time.Sleep(2 * time.Second)
	})

	t.Run("Test with text embeddings", func(t *testing.T) {
		embeddingFunction, err := NewNomicEmbeddingFunction(WithEnvAPIKey(), WithTextEmbeddings())
		require.NoError(t, err)
		resp, rerr := embeddingFunction.EmbedQuery(context.Background(), "this is my query")
		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, *resp.ArrayOfFloat32, 768)
		time.Sleep(2 * time.Second)
	})
}
