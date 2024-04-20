//go:build test

package together

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"testing"
)

func Test_client(t *testing.T) {
	apiKey := os.Getenv("TOGETHER_API_KEY")
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv("TOGETHER_API_KEY")
	}
	client, err := NewTogetherClient(WithEnvAPIKey())
	require.NoError(t, err)

	t.Run("Test CreateEmbedding", func(t *testing.T) {
		req := CreateEmbeddingRequest{
			Model: "togethercomputer/m2-bert-80M-8k-retrieval",
			Input: &EmbeddingInputs{Input: "Test document"},
		}
		resp, rerr := client.CreateEmbedding(context.Background(), &req)

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		require.Len(t, resp.Data, 1)
	})
}

func Test_together_embedding_function(t *testing.T) {
	apiKey := os.Getenv("TOGETHER_API_KEY")
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
	}

	t.Run("Test EmbedDocuments with env-based API Key", func(t *testing.T) {
		client, err := NewTogetherEmbeddingFunction(WithEnvAPIKey())
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Len(t, *resp[0].ArrayOfFloat32, 768)

	})

	t.Run("Test EmbedDocuments for model with env-based API Key", func(t *testing.T) {
		client, err := NewTogetherEmbeddingFunction(WithEnvAPIKey(), WithDefaultModel("togethercomputer/m2-bert-80M-2k-retrieval"))
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Len(t, *resp[0].ArrayOfFloat32, 768)
	})

	t.Run("Test EmbedDocuments with too large init batch", func(t *testing.T) {
		_, err := NewTogetherEmbeddingFunction(WithEnvAPIKey(), WithMaxBatchSize(200))
		require.Error(t, err)
		require.Contains(t, err.Error(), "max batch size must be less than")
	})

	t.Run("Test EmbedDocuments with too large batch at inference", func(t *testing.T) {
		client, err := NewTogetherEmbeddingFunction(WithEnvAPIKey())
		require.NoError(t, err)
		docs200 := make([]string, 200)
		for i := 0; i < 200; i++ {
			docs200[i] = "Test document"
		}
		_, err = client.EmbedDocuments(context.Background(), docs200)
		require.Error(t, err)
		require.Contains(t, err.Error(), "number of documents exceeds the maximum batch")
	})

	t.Run("Test EmbedQuery", func(t *testing.T) {
		client, err := NewTogetherEmbeddingFunction(WithEnvAPIKey())
		require.NoError(t, err)
		resp, err := client.EmbedQuery(context.Background(), "Test query")
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.Len(t, *resp.ArrayOfFloat32, 768)
	})

	t.Run("Test EmbedDocuments with env-based API Key and WithDefaultHeaders", func(t *testing.T) {
		client, err := NewTogetherEmbeddingFunction(WithEnvAPIKey(), WithDefaultModel("togethercomputer/m2-bert-80M-2k-retrieval"), WithDefaultHeaders(map[string]string{"X-Test-Header": "test"}))
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Len(t, *resp[0].ArrayOfFloat32, 768)
	})

	t.Run("Test EmbedDocuments with var API Key", func(t *testing.T) {
		client, err := NewTogetherEmbeddingFunction(WithAPIToken(os.Getenv("TOGETHER_API_KEY")))
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Len(t, *resp[0].ArrayOfFloat32, 768)
	})

	t.Run("Test EmbedDocuments with var token and account id and http client", func(t *testing.T) {
		client, err := NewTogetherEmbeddingFunction(WithAPIToken(os.Getenv("TOGETHER_API_KEY")), WithHTTPClient(http.DefaultClient))
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Len(t, *resp[0].ArrayOfFloat32, 768)
	})
}
