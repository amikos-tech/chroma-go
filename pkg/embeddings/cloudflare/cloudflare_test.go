//go:build test

package cloudflare

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
	apiKey := os.Getenv("CF_API_TOKEN")
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv("CF_API_TOKEN")
	}
	client, err := NewCloudflareClient(WithEnvAPIToken(), WithEnvAccountID())
	require.NoError(t, err)

	t.Run("Test CreateEmbedding", func(t *testing.T) {
		req := CreateEmbeddingRequest{
			Text: []string{"Test document"},
		}
		resp, rerr := client.CreateEmbedding(context.Background(), &req)

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Result)
		require.NotNil(t, resp.Result.Data)
		require.Len(t, resp.Result.Data, 1)
	})
}

func Test_cloudflare_embedding_function(t *testing.T) {
	apiKey := os.Getenv("CF_API_TOKEN")
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
	}

	t.Run("Test EmbedDocuments with env-based token and account id", func(t *testing.T) {
		client, err := NewCloudflareEmbeddingFunction(WithEnvAPIToken(), WithEnvAccountID())
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Len(t, *resp[0].ArrayOfFloat32, 768)

	})

	t.Run("Test EmbedDocuments with env-based token and gateway", func(t *testing.T) {
		client, err := NewCloudflareEmbeddingFunction(WithEnvAPIToken(), WithEnvGatewayEndpoint())
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Len(t, *resp[0].ArrayOfFloat32, 768)

	})

	t.Run("Test EmbedDocuments for model with env-based token and account id", func(t *testing.T) {
		client, err := NewCloudflareEmbeddingFunction(WithEnvAPIToken(), WithEnvAccountID(), WithDefaultModel("@cf/baai/bge-small-en-v1.5"))
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Len(t, *resp[0].ArrayOfFloat32, 384)
	})

	t.Run("Test EmbedDocuments with too large init batch", func(t *testing.T) {
		_, err := NewCloudflareEmbeddingFunction(WithEnvAPIToken(), WithEnvAccountID(), WithMaxBatchSize(200))
		require.Error(t, err)
		require.Contains(t, err.Error(), "max batch size must be less than")
	})

	t.Run("Test EmbedDocuments with too large batch at inference", func(t *testing.T) {
		client, err := NewCloudflareEmbeddingFunction(WithEnvAPIToken(), WithEnvAccountID())
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
		client, err := NewCloudflareEmbeddingFunction(WithEnvAPIToken(), WithEnvAccountID())
		require.NoError(t, err)
		resp, err := client.EmbedQuery(context.Background(), "Test query")
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.Len(t, *resp.ArrayOfFloat32, 768)
	})

	t.Run("Test EmbedDocuments with env-based token and account id and WithDefaultHeaders", func(t *testing.T) {
		client, err := NewCloudflareEmbeddingFunction(WithEnvAPIToken(), WithEnvAccountID(), WithDefaultModel("@cf/baai/bge-small-en-v1.5"), WithDefaultHeaders(map[string]string{"X-Test-Header": "test"}))
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Len(t, *resp[0].ArrayOfFloat32, 384)
	})

	t.Run("Test EmbedDocuments with var token and account id", func(t *testing.T) {
		client, err := NewCloudflareEmbeddingFunction(WithAPIToken(os.Getenv("CF_API_TOKEN")), WithAccountID(os.Getenv("CF_ACCOUNT_ID")))
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Len(t, *resp[0].ArrayOfFloat32, 768)
	})
	t.Run("Test EmbedDocuments with var token and gateway", func(t *testing.T) {
		client, err := NewCloudflareEmbeddingFunction(WithAPIToken(os.Getenv("CF_API_TOKEN")), WithGatewayEndpoint(os.Getenv("CF_GATEWAY_ENDPOINT")))
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Len(t, *resp[0].ArrayOfFloat32, 768)
	})

	t.Run("Test EmbedDocuments with var token and account id and http client", func(t *testing.T) {
		client, err := NewCloudflareEmbeddingFunction(WithAPIToken(os.Getenv("CF_API_TOKEN")), WithAccountID(os.Getenv("CF_ACCOUNT_ID")), WithHTTPClient(http.DefaultClient))
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Len(t, *resp[0].ArrayOfFloat32, 768)
	})
}
