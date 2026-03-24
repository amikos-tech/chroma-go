//go:build ef

package voyage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func testdataPath(t *testing.T, name string) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "testdata", name))
	require.NoError(t, err)
	return abs
}

func requireVoyageAPIKey(t *testing.T) {
	t.Helper()
	if os.Getenv("VOYAGE_API_KEY") == "" {
		t.Skip("VOYAGE_API_KEY not set")
	}
}

func newContentEF(t *testing.T) *VoyageAIEmbeddingFunction {
	t.Helper()
	requireVoyageAPIKey(t)
	ef, err := NewVoyageAIEmbeddingFunction(
		WithEnvAPIKey(),
		WithDefaultModel("voyage-multimodal-3.5"),
	)
	require.NoError(t, err)
	return ef
}

func TestContentEmbedText(t *testing.T) {
	ef := newContentEF(t)

	text, err := os.ReadFile(testdataPath(t, "the_golden_hour.md"))
	require.NoError(t, err)

	content := embeddings.Content{
		Parts: []embeddings.Part{embeddings.NewTextPart(string(text))},
	}
	emb, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, emb)
	assert.Greater(t, emb.Len(), 0)
}

func TestContentEmbedImage(t *testing.T) {
	ef := newContentEF(t)

	content := embeddings.Content{
		Parts: []embeddings.Part{
			embeddings.NewPartFromSource(
				embeddings.ModalityImage,
				embeddings.BinarySource{
					Kind:     embeddings.SourceKindFile,
					FilePath: testdataPath(t, "lioness.png"),
					MIMEType: "image/png",
				},
			),
		},
	}
	emb, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, emb)
	assert.Greater(t, emb.Len(), 0)
}

func TestContentEmbedVideo(t *testing.T) {
	t.Skip("VoyageAI encodes video as inline base64 — the 5.3MB test asset exceeds the 32K token context window")
}

func TestContentEmbedMixedParts(t *testing.T) {
	ef := newContentEF(t)

	content := embeddings.Content{
		Parts: []embeddings.Part{
			embeddings.NewTextPart("A lioness hunting at sunset"),
			embeddings.NewPartFromSource(
				embeddings.ModalityImage,
				embeddings.BinarySource{
					Kind:     embeddings.SourceKindFile,
					FilePath: testdataPath(t, "lioness.png"),
					MIMEType: "image/png",
				},
			),
		},
	}
	emb, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, emb)
	assert.Greater(t, emb.Len(), 0)
}

func TestContentEmbedContentsBatch(t *testing.T) {
	ef := newContentEF(t)

	contents := []embeddings.Content{
		{Parts: []embeddings.Part{embeddings.NewTextPart("The golden hour on the Serengeti")}},
		{Parts: []embeddings.Part{
			embeddings.NewPartFromSource(
				embeddings.ModalityImage,
				embeddings.BinarySource{
					Kind:     embeddings.SourceKindFile,
					FilePath: testdataPath(t, "lioness.png"),
					MIMEType: "image/png",
				},
			),
		}},
	}
	results, err := ef.EmbedContents(context.Background(), contents)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Greater(t, results[0].Len(), 0)
	assert.Greater(t, results[1].Len(), 0)
}

func TestContentEmbedWithIntent(t *testing.T) {
	ef := newContentEF(t)

	content := embeddings.Content{
		Parts:  []embeddings.Part{embeddings.NewTextPart("How do lionesses hunt?")},
		Intent: embeddings.IntentRetrievalQuery,
	}
	emb, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, emb)
	assert.Greater(t, emb.Len(), 0)
}
