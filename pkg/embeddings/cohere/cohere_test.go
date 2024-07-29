//go:build ef

package cohere

import (
	"context"
	ccommons "github.com/amikos-tech/chroma-go/pkg/commons/cohere"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ef(t *testing.T) {
	apiKey := os.Getenv("COHERE_API_KEY")
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv("COHERE_API_KEY")
	}

	t.Run("Test Create Embed", func(t *testing.T) {
		ef, err := NewCohereEmbeddingFunction(WithAPIKey(apiKey))
		require.NoError(t, err)
		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
			// Add more documents as needed
		}
		resp, err := ef.EmbedDocuments(context.Background(), documents)
		require.Nil(t, err)
		require.NotNil(t, resp)
	})

	t.Run("Test Create Embed with model option", func(t *testing.T) {
		ef, err := NewCohereEmbeddingFunction(WithAPIKey(apiKey), WithModel("embed-multilingual-v3.0"))
		require.NoError(t, err)
		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
			// Add more documents as needed
		}
		resp, rerr := ef.EmbedDocuments(context.Background(), documents)
		require.Nil(t, rerr)
		require.NotNil(t, resp)
	})

	t.Run("Test Create Embed with model option embeddings type uint8", func(t *testing.T) {
		ef, err := NewCohereEmbeddingFunction(WithAPIKey(apiKey), WithModel("embed-multilingual-v3.0"), WithEmbeddingTypes(EmbeddingTypeUInt8))
		require.NoError(t, err)
		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
			// Add more documents as needed
		}
		resp, err := ef.EmbedDocuments(context.Background(), documents)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Empty(t, resp[0].ArrayOfFloat32)
		require.NotNil(t, resp[0].ArrayOfInt32)
	})

	t.Run("Test Create Embed with model option embeddings type int8", func(t *testing.T) {
		ef, err := NewCohereEmbeddingFunction(WithEnvAPIKey(), WithModel("embed-multilingual-v3.0"), WithEmbeddingTypes(EmbeddingTypeInt8))
		require.NoError(t, err)
		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
			// Add more documents as needed
		}
		resp, err := ef.EmbedDocuments(context.Background(), documents)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp, 2)
		require.Empty(t, resp[0].ArrayOfFloat32)
		require.NotNil(t, resp[0].ArrayOfInt32)
	})

	t.Run("Test Create Embed for query", func(t *testing.T) {
		ef, err := NewCohereEmbeddingFunction(
			WithEnvAPIKey(),
			WithModel("embed-multilingual-v3.0"),
		)
		require.NoError(t, err)
		resp, err := ef.EmbedQuery(context.Background(), "This is a query")
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.ArrayOfFloat32)
		require.Empty(t, resp.ArrayOfInt32)
	})

	t.Run("Test With API options", func(t *testing.T) {
		ef, err := NewCohereEmbeddingFunction(
			WithEnvAPIKey(),
			WithBaseURL(ccommons.DefaultBaseURL),
			WithAPIVersion(ccommons.DefaultAPIVersion),
			WithModel("embed-multilingual-v3.0"),
		)
		require.NoError(t, err)
		resp, err := ef.EmbedQuery(context.Background(), "This is a query")
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.ArrayOfFloat32)
		require.Empty(t, resp.ArrayOfInt32)
	})
}
