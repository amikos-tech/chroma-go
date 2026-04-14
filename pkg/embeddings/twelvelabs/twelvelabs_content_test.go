//go:build ef

package twelvelabs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func TestTwelveLabsCapabilities(t *testing.T) {
	ef := newTestEF("http://localhost")
	caps := ef.Capabilities()
	assert.Len(t, caps.Modalities, 4)
	assert.Contains(t, caps.Modalities, embeddings.ModalityText)
	assert.Contains(t, caps.Modalities, embeddings.ModalityImage)
	assert.Contains(t, caps.Modalities, embeddings.ModalityAudio)
	assert.Contains(t, caps.Modalities, embeddings.ModalityVideo)
	assert.False(t, caps.SupportsBatch)
	assert.False(t, caps.SupportsMixedPart)
	assert.Empty(t, caps.Intents)
}

func TestTwelveLabsDoesNotImplementIntentMapper(t *testing.T) {
	ef := newTestEF("http://localhost")
	_, ok := interface{}(ef).(embeddings.IntentMapper)
	assert.False(t, ok)
}

func TestTwelveLabsEmbedContentText(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req EmbedV2Request
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "text", req.InputType)
		assert.NotNil(t, req.Text)
		assert.Equal(t, "hello", req.Text.InputText)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, embedV2Response([]float64{1, 2, 3}))
	})

	ef := newTestEF(srv.URL)
	result, err := ef.EmbedContent(context.Background(), embeddings.NewTextContent("hello"))
	require.NoError(t, err)
	assert.Equal(t, 3, result.Len())
}

func TestTwelveLabsEmbedContentImageURL(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req EmbedV2Request
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "image", req.InputType)
		assert.NotNil(t, req.Image)
		assert.Equal(t, "https://example.com/photo.png", req.Image.MediaSource.URL)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, embedV2Response([]float64{4, 5, 6}))
	})

	ef := newTestEF(srv.URL)
	result, err := ef.EmbedContent(context.Background(), embeddings.NewImageURL("https://example.com/photo.png"))
	require.NoError(t, err)
	assert.Equal(t, 3, result.Len())
}

func TestTwelveLabsEmbedContentImageBase64(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req EmbedV2Request
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "image", req.InputType)
		assert.NotNil(t, req.Image)
		assert.NotEmpty(t, req.Image.MediaSource.Base64String)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, embedV2Response([]float64{7, 8, 9}))
	})

	ef := newTestEF(srv.URL)
	content := embeddings.NewContent([]embeddings.Part{
		embeddings.NewPartFromSource(
			embeddings.ModalityImage,
			embeddings.NewBinarySourceFromBase64("aGVsbG8="),
		),
	})
	result, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	assert.Equal(t, 3, result.Len())
}

func TestTwelveLabsEmbedContentAudio(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req EmbedV2Request
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "audio", req.InputType)
		assert.NotNil(t, req.Audio)
		assert.Equal(t, "audio", req.Audio.EmbeddingOption)
		assert.Equal(t, "https://example.com/clip.mp3", req.Audio.MediaSource.URL)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, embedV2Response([]float64{1, 2, 3}))
	})

	ef := newTestEF(srv.URL)
	result, err := ef.EmbedContent(context.Background(), embeddings.NewAudioURL("https://example.com/clip.mp3"))
	require.NoError(t, err)
	assert.Equal(t, 3, result.Len())
}

func TestTwelveLabsEmbedContentVideo(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req EmbedV2Request
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "video", req.InputType)
		assert.NotNil(t, req.Video)
		assert.Equal(t, "https://example.com/clip.mp4", req.Video.MediaSource.URL)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, embedV2Response([]float64{1, 2, 3}))
	})

	ef := newTestEF(srv.URL)
	result, err := ef.EmbedContent(context.Background(), embeddings.NewVideoURL("https://example.com/clip.mp4"))
	require.NoError(t, err)
	assert.Equal(t, 3, result.Len())
}

func TestTwelveLabsEmbedContentMixedPartRejects(t *testing.T) {
	ef := newTestEF("http://localhost")
	content := embeddings.NewContent([]embeddings.Part{
		{Modality: embeddings.ModalityText, Text: "hello"},
		embeddings.NewPartFromSource(
			embeddings.ModalityImage,
			embeddings.NewBinarySourceFromURL("https://example.com/photo.png"),
		),
	})
	_, err := ef.EmbedContent(context.Background(), content)
	require.Error(t, err)
}

func TestTwelveLabsEmbedContents(t *testing.T) {
	var callCount atomic.Int32
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, embedV2Response([]float64{float64(n), 2, 3}))
	})

	ef := newTestEF(srv.URL)
	contents := []embeddings.Content{
		embeddings.NewTextContent("first"),
		embeddings.NewTextContent("second"),
	}
	results, err := ef.EmbedContents(context.Background(), contents)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, int32(2), callCount.Load())
}

func TestTwelveLabsEmbedContentUnsupportedModality(t *testing.T) {
	ef := newTestEF("http://localhost")
	content := embeddings.NewContent([]embeddings.Part{
		{Modality: embeddings.Modality("pdf"), Text: "some pdf"},
	})
	_, err := ef.EmbedContent(context.Background(), content)
	require.Error(t, err)
}

func TestResolveBytes(t *testing.T) {
	t.Run("nil source returns error", func(t *testing.T) {
		_, err := resolveBytes(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source cannot be nil")
	})

	t.Run("empty bytes source returns error", func(t *testing.T) {
		_, err := resolveBytes(&embeddings.BinarySource{Kind: embeddings.SourceKindBytes})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bytes source must include non-empty bytes")
	})

	t.Run("oversized file returns error", func(t *testing.T) {
		tmp, err := os.CreateTemp(t.TempDir(), "twelvelabs-large-*")
		require.NoError(t, err)
		t.Cleanup(func() { _ = tmp.Close() })
		require.NoError(t, tmp.Truncate(100*1024*1024+1))

		_, err = resolveBytes(&embeddings.BinarySource{
			Kind:     embeddings.SourceKindFile,
			FilePath: tmp.Name(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file size exceeds maximum")
	})
}

func TestBuildMediaSourceURLValidation(t *testing.T) {
	t.Run("rejects empty URL", func(t *testing.T) {
		_, err := buildMediaSource(&embeddings.BinarySource{Kind: embeddings.SourceKindURL})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-empty URL")
	})

	t.Run("rejects malformed URL", func(t *testing.T) {
		_, err := buildMediaSource(&embeddings.BinarySource{Kind: embeddings.SourceKindURL, URL: "example.com/audio.mp3"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "absolute URL")
	})

	t.Run("accepts absolute URL", func(t *testing.T) {
		got, err := buildMediaSource(&embeddings.BinarySource{Kind: embeddings.SourceKindURL, URL: "https://example.com/audio.mp3"})
		require.NoError(t, err)
		assert.Equal(t, MediaSource{URL: "https://example.com/audio.mp3"}, got)
	})
}

func TestTwelveLabsEmbedContentValidationIncludesProviderContext(t *testing.T) {
	ef := newTestEF("http://localhost")
	_, err := ef.EmbedContent(context.Background(), embeddings.NewTextContent(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Twelve Labs")
}

func TestTwelveLabsEmbedContentsValidationIncludesProviderContext(t *testing.T) {
	ef := newTestEF("http://localhost")
	_, err := ef.EmbedContents(context.Background(), []embeddings.Content{
		embeddings.NewTextContent(""),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Twelve Labs")
}

func TestTwelveLabsEmbedContentEmptyEmbeddingVector(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, embedV2Response([]float64{}))
	})

	ef := newTestEF(srv.URL)
	_, err := ef.EmbedContent(context.Background(), embeddings.NewTextContent("hello"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty embedding vector")
}
