//go:build ef

package twelvelabs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
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

	t.Run("accepts plain http URL", func(t *testing.T) {
		got, err := buildMediaSource(&embeddings.BinarySource{Kind: embeddings.SourceKindURL, URL: "http://example.com/audio.mp3"})
		require.NoError(t, err)
		assert.Equal(t, MediaSource{URL: "http://example.com/audio.mp3"}, got)
	})

	t.Run("rejects non-http(s) schemes", func(t *testing.T) {
		for _, url := range []string{
			"ftp://example.com/clip.mp3",
			"gopher://example.com/clip.mp3",
			"file://localhost/etc/passwd",
		} {
			_, err := buildMediaSource(&embeddings.BinarySource{Kind: embeddings.SourceKindURL, URL: url})
			require.Error(t, err, "scheme in %q must be rejected", url)
			assert.Contains(t, err.Error(), "not supported", "url=%s", url)
		}
	})

	t.Run("rejects opaque URLs with unsupported schemes", func(t *testing.T) {
		// javascript: and data: URLs parse with empty host; the earlier
		// absolute-URL check handles them, but this pins the behavior.
		_, err := buildMediaSource(&embeddings.BinarySource{Kind: embeddings.SourceKindURL, URL: "javascript:alert(1)"})
		require.Error(t, err)
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

// TestResolveBytesBase64AtSizeBoundary proves the cheap pre-check admits a
// payload decoding to exactly maxMediaSourceSize. The former len*3/4 estimate
// folded "=" padding into the payload and overshot by 2 bytes, rejecting the
// one size the authoritative post-decode check is required to accept.
func TestResolveBytesBase64AtSizeBoundary(t *testing.T) {
	if testing.Short() {
		t.Skip("allocates ~240MB to hit the 100MB boundary exactly")
	}
	// 104857600 = 3*34952533 + 1, so the final group carries one byte and
	// encodes as two chars plus "==".
	atLimit := embeddings.NewBinarySourceFromBase64(strings.Repeat("A", 139810132) + "AA==")
	data, err := resolveBytes(&atLimit)
	require.NoError(t, err, "a payload decoding to exactly the limit must be accepted")
	assert.Equal(t, maxMediaSourceSize, int64(len(data)))

	// One group beyond the limit must still be rejected.
	overLimit := embeddings.NewBinarySourceFromBase64(atLimit.Base64 + strings.Repeat("A", 4))
	_, err = resolveBytes(&overLimit)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum")
}
