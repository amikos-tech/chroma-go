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
	ef := NewOpenAIEmbeddingFunction(apiKey)

	t.Run("Test DefaultApiService Add", func(t *testing.T) {
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
	})
}
