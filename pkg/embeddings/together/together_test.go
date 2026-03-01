//go:build ef

package together

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	defaultTogetherTestModel = "togethercomputer/m2-bert-80M-8k-retrieval"
	testModelEnvVar          = "TOGETHER_TEST_MODEL"
)

func togetherTestModel() string {
	if model := os.Getenv(testModelEnvVar); model != "" {
		return model
	}
	return defaultTogetherTestModel
}

func requireTogetherSuccessOrSkip(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	// The Together CI account can lose access to specific models over time.
	// Skip these provider-side failures to keep EF checks stable.
	if strings.Contains(err.Error(), "model_not_available") ||
		strings.Contains(err.Error(), "Unable to access non-serverless model") {
		t.Skipf("Skipping test due to Together model availability: %v", err)
	}
	require.NoError(t, err)
}

func Test_client(t *testing.T) {
	apiKey := os.Getenv("TOGETHER_API_KEY")
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv("TOGETHER_API_KEY")
	}
	client, err := NewTogetherClient(WithEnvAPIToken())
	require.NoError(t, err)

	t.Run("Test CreateEmbedding", func(t *testing.T) {
		req := CreateEmbeddingRequest{
			Model: togetherTestModel(),
			Input: &EmbeddingInputs{Input: "Test document"},
		}
		resp, rerr := client.CreateEmbedding(context.Background(), &req)

		requireTogetherSuccessOrSkip(t, rerr)
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
		client, err := NewTogetherEmbeddingFunction(WithEnvAPIToken())
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		requireTogetherSuccessOrSkip(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Greater(t, resp[0].Len(), 0)

	})

	t.Run("Test EmbedDocuments for model with env-based API Key", func(t *testing.T) {
		client, err := NewTogetherEmbeddingFunction(WithEnvAPIToken(), WithDefaultModel(embeddings.EmbeddingModel(togetherTestModel())))
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		requireTogetherSuccessOrSkip(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Greater(t, resp[0].Len(), 0)
	})

	t.Run("Test EmbedDocuments with too large init batch", func(t *testing.T) {
		_, err := NewTogetherEmbeddingFunction(WithEnvAPIToken(), WithMaxBatchSize(200))
		require.Error(t, err)
		require.Contains(t, err.Error(), "max batch size must be less than")
	})

	t.Run("Test EmbedDocuments with too large batch at inference", func(t *testing.T) {
		client, err := NewTogetherEmbeddingFunction(WithEnvAPIToken())
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
		client, err := NewTogetherEmbeddingFunction(WithEnvAPIToken())
		require.NoError(t, err)
		resp, err := client.EmbedQuery(context.Background(), "Test query")
		requireTogetherSuccessOrSkip(t, err)
		require.NotNil(t, resp)
		require.Greater(t, resp.Len(), 0)
	})

	t.Run("Test EmbedDocuments with env-based API Key and WithDefaultHeaders", func(t *testing.T) {
		client, err := NewTogetherEmbeddingFunction(WithEnvAPIToken(), WithDefaultModel(embeddings.EmbeddingModel(togetherTestModel())), WithDefaultHeaders(map[string]string{"X-Test-Header": "test"}))
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		requireTogetherSuccessOrSkip(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Greater(t, resp[0].Len(), 0)
	})

	t.Run("Test EmbedDocuments with var API Key", func(t *testing.T) {
		client, err := NewTogetherEmbeddingFunction(WithAPIToken(os.Getenv("TOGETHER_API_KEY")))
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		requireTogetherSuccessOrSkip(t, rerr)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Greater(t, resp[0].Len(), 0)
	})

	t.Run("Test EmbedDocuments with var token and account id and http client", func(t *testing.T) {
		client, err := NewTogetherEmbeddingFunction(WithAPIToken(os.Getenv("TOGETHER_API_KEY")), WithHTTPClient(http.DefaultClient))
		require.NoError(t, err)
		resp, rerr := client.EmbedDocuments(context.Background(), []string{"Test document", "Another test document"})

		requireTogetherSuccessOrSkip(t, rerr)
		require.NotNil(t, resp)
		require.Equal(t, 2, len(resp))
		require.Greater(t, resp[0].Len(), 0)
	})
}
