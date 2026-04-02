//go:build ef

package twelvelabs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func loadTwelveLabsAPIKey(t *testing.T) {
	t.Helper()
	apiKey := os.Getenv(APIKeyEnvVar)
	if apiKey == "" {
		_ = godotenv.Load("../../../.env")
		apiKey = os.Getenv(APIKeyEnvVar)
	}
	if apiKey == "" {
		t.Skipf("%s not set", APIKeyEnvVar)
	}
}

func testdataPath(t *testing.T, name string) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "testdata", name))
	require.NoError(t, err)
	return abs
}

func TestIntegration_EmbedDocuments(t *testing.T) {
	loadTwelveLabsAPIKey(t)
	ef, err := NewTwelveLabsEmbeddingFunction(WithEnvAPIKey())
	require.NoError(t, err)

	result, err := ef.EmbedDocuments(context.Background(), []string{"hello world"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, 512, result[0].Len())
}

func TestIntegration_EmbedQuery(t *testing.T) {
	loadTwelveLabsAPIKey(t)
	ef, err := NewTwelveLabsEmbeddingFunction(WithEnvAPIKey())
	require.NoError(t, err)

	result, err := ef.EmbedQuery(context.Background(), "search query")
	require.NoError(t, err)
	assert.Equal(t, 512, result.Len())
}

func TestIntegration_EmbedContentText(t *testing.T) {
	loadTwelveLabsAPIKey(t)
	ef, err := NewTwelveLabsEmbeddingFunction(WithEnvAPIKey())
	require.NoError(t, err)

	result, err := ef.EmbedContent(context.Background(), embeddings.NewTextContent("Twelve Labs multimodal embeddings"))
	require.NoError(t, err)
	assert.Equal(t, 512, result.Len())
}

func TestIntegration_EmbedContentImageFile(t *testing.T) {
	loadTwelveLabsAPIKey(t)
	ef, err := NewTwelveLabsEmbeddingFunction(WithEnvAPIKey())
	require.NoError(t, err)

	content := embeddings.NewContent([]embeddings.Part{
		embeddings.NewPartFromSource(
			embeddings.ModalityImage,
			embeddings.NewBinarySourceFromFile(testdataPath(t, "lioness.png")),
		),
	})
	result, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	assert.Equal(t, 512, result.Len())
}

func TestIntegration_EmbedContentsMultiple(t *testing.T) {
	loadTwelveLabsAPIKey(t)
	ef, err := NewTwelveLabsEmbeddingFunction(WithEnvAPIKey())
	require.NoError(t, err)

	contents := []embeddings.Content{
		embeddings.NewTextContent("first document"),
		embeddings.NewTextContent("second document"),
	}
	results, err := ef.EmbedContents(context.Background(), contents)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, 512, results[0].Len())
	assert.Equal(t, 512, results[1].Len())
}
