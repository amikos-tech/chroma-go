package openai

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_openai_client(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		err := godotenv.Load("../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	t.Run("Test DefaultApiService Add", func(t *testing.T) {
		ef, efErr := NewOpenAIEmbeddingFunction(apiKey)
		require.NoError(t, efErr)

		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
			// Add more documents as needed
		}
		resp, reqErr := ef.EmbedDocuments(context.Background(), documents)
		require.NoError(t, reqErr)
		require.NotNil(t, resp)
		// assert.Equal(t, 201, httpRes.StatusCode)
		require.Empty(t, ef.apiClient.OrgID)
	})

	t.Run("Test Adding Organization Id with NewOpenAIClient", func(t *testing.T) {
		apiClient, efError := NewOpenAIClient(apiKey, WithOpenAIOrganizationID("org-123"))
		require.NoError(t, efError)

		require.Equal(t, "org-123", apiClient.OrgID)
	})

	t.Run("Test Adding Organization Id with NewOpenAIEmbeddingFunction", func(t *testing.T) {
		ef, efError := NewOpenAIEmbeddingFunction(apiKey, WithOpenAIOrganizationID("org-123"))
		require.NoError(t, efError)

		require.Equal(t, "org-123", ef.apiClient.OrgID)
	})

	t.Run("Test With Model text-embedding-3-small", func(t *testing.T) {
		ef, erErr := NewOpenAIEmbeddingFunction(apiKey, WithModel(TextEmbedding3Small))
		require.NoError(t, erErr)
		documents := []string{
			"Document 1 content here",
		}
		resp, reqErr := ef.EmbedDocuments(context.Background(), documents)
		require.Nil(t, reqErr)
		require.NotNil(t, resp)
		require.Empty(t, ef.apiClient.OrgID)
		require.Len(t, resp[0], 1536)
	})

	t.Run("Test With Model text-embedding-3-large", func(t *testing.T) {
		ef, efErr := NewOpenAIEmbeddingFunction(apiKey, WithModel(TextEmbedding3Large))
		require.NoError(t, efErr)
		documents := []string{
			"Document 1 content here",
		}
		resp, reqErr := ef.EmbedDocuments(context.Background(), documents)
		require.Nil(t, reqErr)
		require.NotNil(t, resp)
		require.Empty(t, ef.apiClient.OrgID)
		require.Len(t, resp[0], 3072)
	})

	t.Run("Test With Invalid Model", func(t *testing.T) {
		_, efErr := NewOpenAIEmbeddingFunction(apiKey, WithModel("invalid-model"))
		require.Error(t, efErr)
		require.Contains(t, efErr.Error(), "invalid model name invalid-model")
	})
}
