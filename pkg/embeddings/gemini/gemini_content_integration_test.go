//go:build ef

package gemini

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func testdataPath(name string) string {
	abs, err := filepath.Abs(filepath.Join("..", "testdata", name))
	if err != nil {
		panic(err)
	}
	return abs
}

func newContentEF(t *testing.T) *GeminiEmbeddingFunction {
	t.Helper()
	_ = requireGeminiAPIKey(t)
	ef, err := NewGeminiEmbeddingFunction(WithEnvAPIKey())
	require.NoError(t, err)
	t.Cleanup(func() { _ = ef.Close() })
	return ef
}

func TestContentEmbedText(t *testing.T) {
	ef := newContentEF(t)

	text, err := os.ReadFile(testdataPath("the_golden_hour.md"))
	require.NoError(t, err)

	content := embeddings.Content{
		Parts: []embeddings.Part{embeddings.NewTextPart(string(text))},
	}
	emb, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, emb)
	assert.Equal(t, 3072, emb.Len())
}

func TestContentEmbedImage(t *testing.T) {
	ef := newContentEF(t)

	content := embeddings.Content{
		Parts: []embeddings.Part{
			embeddings.NewPartFromSource(
				embeddings.ModalityImage,
				embeddings.BinarySource{
					Kind:     embeddings.SourceKindFile,
					FilePath: testdataPath("lioness.png"),
					MIMEType: "image/png",
				},
			),
		},
	}
	emb, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, emb)
	assert.Equal(t, 3072, emb.Len())
}

func TestContentEmbedAudio(t *testing.T) {
	ef := newContentEF(t)

	content := embeddings.Content{
		Parts: []embeddings.Part{
			embeddings.NewPartFromSource(
				embeddings.ModalityAudio,
				embeddings.BinarySource{
					Kind:     embeddings.SourceKindFile,
					FilePath: testdataPath("the_golden_hour.mp3"),
					MIMEType: "audio/mpeg",
				},
			),
		},
	}
	emb, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, emb)
	assert.Equal(t, 3072, emb.Len())
}

func TestContentEmbedVideo(t *testing.T) {
	ef := newContentEF(t)

	content := embeddings.Content{
		Parts: []embeddings.Part{
			embeddings.NewPartFromSource(
				embeddings.ModalityVideo,
				embeddings.BinarySource{
					Kind:     embeddings.SourceKindFile,
					FilePath: testdataPath("the_pounce.mp4"),
					MIMEType: "video/mp4",
				},
			),
		},
	}
	emb, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, emb)
	assert.Equal(t, 3072, emb.Len())
}

func TestContentEmbedPDF(t *testing.T) {
	ef := newContentEF(t)

	content := embeddings.Content{
		Parts: []embeddings.Part{
			embeddings.NewPartFromSource(
				embeddings.ModalityPDF,
				embeddings.BinarySource{
					Kind:     embeddings.SourceKindFile,
					FilePath: testdataPath("the_golden_hour.pdf"),
					MIMEType: "application/pdf",
				},
			),
		},
	}
	emb, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, emb)
	assert.Equal(t, 3072, emb.Len())
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
					FilePath: testdataPath("lioness.png"),
					MIMEType: "image/png",
				},
			),
		},
	}
	emb, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, emb)
	assert.Equal(t, 3072, emb.Len())
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
					FilePath: testdataPath("lioness.png"),
					MIMEType: "image/png",
				},
			),
		}},
	}
	results, err := ef.EmbedContents(context.Background(), contents)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, 3072, results[0].Len())
	assert.Equal(t, 3072, results[1].Len())
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
	assert.Equal(t, 3072, emb.Len())
}

func TestContentEmbedWithProviderHints(t *testing.T) {
	ef := newContentEF(t)

	content := embeddings.Content{
		Parts: []embeddings.Part{embeddings.NewTextPart("Lioness hunting behavior")},
		ProviderHints: map[string]any{
			"task_type": "CLASSIFICATION",
		},
	}
	emb, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, emb)
	assert.Equal(t, 3072, emb.Len())
}

func TestContentEmbedWithDimension(t *testing.T) {
	_ = requireGeminiAPIKey(t)
	ef, err := NewGeminiEmbeddingFunction(WithEnvAPIKey(), WithDimension(768))
	require.NoError(t, err)
	t.Cleanup(func() { _ = ef.Close() })

	content := embeddings.Content{
		Parts: []embeddings.Part{embeddings.NewTextPart("Reduced dimension test")},
	}
	emb, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, emb)
	assert.Equal(t, 768, emb.Len())
}

func TestContentCreateContentEmbeddingDirect(t *testing.T) {
	_ = requireGeminiAPIKey(t)
	client, err := NewGeminiClient(WithEnvAPIKey())
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	mapper := &GeminiEmbeddingFunction{apiClient: client}
	contents := []embeddings.Content{
		{Parts: []embeddings.Part{embeddings.NewTextPart("Direct client call")}},
	}
	results, err := client.CreateContentEmbedding(context.Background(), contents, mapper)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, 3072, results[0].Len())
}
