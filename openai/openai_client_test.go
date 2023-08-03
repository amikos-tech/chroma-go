package openai

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func Test_openapi_DefaultApiService(t *testing.T) {

	err := godotenv.Load("../.env")
	if err != nil {
		assert.Failf(t, "Error loading .env file", "%s", err)
	}
	ef := NewOpenAIEmbeddingFunction(os.Getenv("OPENAI_API_KEY"))

	t.Run("Test DefaultApiService Add", func(t *testing.T) {
		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
			// Add more documents as needed
		}
		resp, rerr := ef.CreateEmbedding(documents)

		require.Nil(t, rerr)
		require.NotNil(t, resp)
		fmt.Printf("resp: %v\n", resp)
		//assert.Equal(t, 201, httpRes.StatusCode)
	})

}
