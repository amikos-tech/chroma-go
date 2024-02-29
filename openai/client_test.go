package openai

import (
	"context"
	"fmt"
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
		ef := NewOpenAIEmbeddingFunction(apiKey)

		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
			// Add more documents as needed
		}
		resp, rerr := ef.EmbedDocuments(context.Background(), documents)
		require.Nil(t, rerr)
		require.NotNil(t, resp)
		fmt.Printf("resp: %v\n", resp)
		// assert.Equal(t, 201, httpRes.StatusCode)
		require.Empty(t, ef.apiClient.OrgID)
	})

	t.Run("Test Adding Organization Id with NewOpenAIClient", func(t *testing.T) {
		apiClient := NewOpenAIClient(apiKey, WithOpenAIOrganizationID("org-123"))

		require.Equal(t, "org-123", apiClient.OrgID)
	})

	t.Run("Test Adding Organization Id with NewOpenAIEmbeddingFunction", func(t *testing.T) {
		ef := NewOpenAIEmbeddingFunction(apiKey, WithOpenAIOrganizationID("org-123"))

		require.Equal(t, "org-123", ef.apiClient.OrgID)
	})
}
