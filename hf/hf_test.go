package hf

import (
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_huggingface_client(t *testing.T) {
	apiKey := os.Getenv("HF_API_KEY")
	if apiKey == "" {
		err := godotenv.Load("../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv("HF_API_KEY")
	}
	ef := NewHuggingFaceEmbeddingFunction(apiKey, "sentence-transformers/all-MiniLM-L6-v2")

	t.Run("Test Create Embed", func(t *testing.T) {
		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
			// Add more documents as needed
		}
		resp, rerr := ef.CreateEmbedding(documents)

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		fmt.Printf("resp: %v\n", resp)
		assert.Equal(t, 2, len(resp))
		// assert.Equal(t, 201, httpRes.StatusCode)
	})
}
