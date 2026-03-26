//go:build ef

package voyage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// TestVoyageCapabilitiesForModel verifies capability derivation for known models.
func TestVoyageCapabilitiesForModel(t *testing.T) {
	t.Run("voyage-multimodal-3.5 returns text/image/video", func(t *testing.T) {
		caps := capabilitiesForModel("voyage-multimodal-3.5")
		assert.Len(t, caps.Modalities, 3)
		assert.Contains(t, caps.Modalities, embeddings.ModalityText)
		assert.Contains(t, caps.Modalities, embeddings.ModalityImage)
		assert.Contains(t, caps.Modalities, embeddings.ModalityVideo)
		assert.Len(t, caps.Intents, 2)
		assert.Contains(t, caps.Intents, embeddings.IntentRetrievalQuery)
		assert.Contains(t, caps.Intents, embeddings.IntentRetrievalDocument)
		assert.True(t, caps.SupportsBatch)
		assert.True(t, caps.SupportsMixedPart)
		assert.True(t, caps.SupportsRequestOption(embeddings.RequestOptionDimension))
	})

	t.Run("voyage-multimodal-3 returns text/image only", func(t *testing.T) {
		caps := capabilitiesForModel("voyage-multimodal-3")
		assert.Len(t, caps.Modalities, 2)
		assert.Contains(t, caps.Modalities, embeddings.ModalityText)
		assert.Contains(t, caps.Modalities, embeddings.ModalityImage)
		assert.False(t, caps.SupportsRequestOption(embeddings.RequestOptionDimension))
	})

	t.Run("unknown model returns text-only", func(t *testing.T) {
		caps := capabilitiesForModel("voyage-2")
		assert.Len(t, caps.Modalities, 1)
		assert.Equal(t, embeddings.ModalityText, caps.Modalities[0])
		assert.False(t, caps.SupportsMixedPart)
	})
}

// TestVoyageCapabilities verifies Capabilities() on VoyageAIEmbeddingFunction.
func TestVoyageCapabilities(t *testing.T) {
	t.Run("multimodal model returns 3 modalities", func(t *testing.T) {
		ef := &VoyageAIEmbeddingFunction{
			apiClient: &VoyageAIClient{DefaultModel: "voyage-multimodal-3.5"},
		}
		caps := ef.Capabilities()
		assert.True(t, caps.SupportsMixedPart)
		assert.Len(t, caps.Modalities, 3)
	})

	t.Run("text-only model returns 1 modality", func(t *testing.T) {
		ef := &VoyageAIEmbeddingFunction{
			apiClient: &VoyageAIClient{DefaultModel: "voyage-2"},
		}
		caps := ef.Capabilities()
		assert.False(t, caps.SupportsMixedPart)
		assert.Len(t, caps.Modalities, 1)
	})
}

// TestVoyageMapIntent verifies supported intent mapping.
func TestVoyageMapIntent(t *testing.T) {
	ef := &VoyageAIEmbeddingFunction{
		apiClient: &VoyageAIClient{DefaultModel: "voyage-multimodal-3.5"},
	}

	t.Run("retrieval_query maps to query", func(t *testing.T) {
		result, err := ef.MapIntent(embeddings.IntentRetrievalQuery)
		require.NoError(t, err)
		assert.Equal(t, "query", result)
	})

	t.Run("retrieval_document maps to document", func(t *testing.T) {
		result, err := ef.MapIntent(embeddings.IntentRetrievalDocument)
		require.NoError(t, err)
		assert.Equal(t, "document", result)
	})
}

// TestVoyageMapIntentRejects verifies unsupported intents are rejected.
func TestVoyageMapIntentRejects(t *testing.T) {
	ef := &VoyageAIEmbeddingFunction{
		apiClient: &VoyageAIClient{DefaultModel: "voyage-multimodal-3.5"},
	}

	t.Run("classification rejected", func(t *testing.T) {
		_, err := ef.MapIntent(embeddings.IntentClassification)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})

	t.Run("clustering rejected", func(t *testing.T) {
		_, err := ef.MapIntent(embeddings.IntentClustering)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})

	t.Run("semantic_similarity rejected", func(t *testing.T) {
		_, err := ef.MapIntent(embeddings.IntentSemanticSimilarity)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})

	t.Run("custom intent rejected", func(t *testing.T) {
		_, err := ef.MapIntent(embeddings.Intent("custom_thing"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})
}

// TestVoyageResolveInputType verifies priority chain for input type resolution.
func TestVoyageResolveInputType(t *testing.T) {
	ef := &VoyageAIEmbeddingFunction{
		apiClient: &VoyageAIClient{DefaultModel: "voyage-multimodal-3.5"},
	}

	t.Run("ProviderHints override takes priority", func(t *testing.T) {
		content := embeddings.Content{
			Parts:         []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
			Intent:        embeddings.IntentRetrievalDocument,
			ProviderHints: map[string]any{"input_type": "query"},
		}
		resolved, err := resolveInputTypeForContent(content, nil, ef)
		require.NoError(t, err)
		require.NotNil(t, resolved)
		assert.Equal(t, InputType("query"), *resolved)
	})

	t.Run("intent mapped when no hints", func(t *testing.T) {
		content := embeddings.Content{
			Parts:  []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
			Intent: embeddings.IntentRetrievalQuery,
		}
		resolved, err := resolveInputTypeForContent(content, nil, ef)
		require.NoError(t, err)
		require.NotNil(t, resolved)
		assert.Equal(t, InputTypeQuery, *resolved)
	})

	t.Run("default returned when no intent or hints", func(t *testing.T) {
		content := embeddings.Content{
			Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
		}
		defaultIT := InputTypeDocument
		resolved, err := resolveInputTypeForContent(content, &defaultIT, ef)
		require.NoError(t, err)
		require.NotNil(t, resolved)
		assert.Equal(t, InputTypeDocument, *resolved)
	})

	t.Run("nil returned when no intent, no hints, no default", func(t *testing.T) {
		content := embeddings.Content{
			Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
		}
		resolved, err := resolveInputTypeForContent(content, nil, ef)
		require.NoError(t, err)
		assert.Nil(t, resolved)
	})
}

// TestVoyageResolveInputTypeErrors verifies error paths in input type resolution.
func TestVoyageResolveInputTypeErrors(t *testing.T) {
	ef := &VoyageAIEmbeddingFunction{
		apiClient: &VoyageAIClient{DefaultModel: "voyage-multimodal-3.5"},
	}

	t.Run("non-string ProviderHints input_type", func(t *testing.T) {
		content := embeddings.Content{
			Parts:         []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
			ProviderHints: map[string]any{"input_type": 42},
		}
		_, err := resolveInputTypeForContent(content, nil, ef)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be a non-empty string")
	})

	t.Run("empty string ProviderHints input_type", func(t *testing.T) {
		content := embeddings.Content{
			Parts:         []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
			ProviderHints: map[string]any{"input_type": ""},
		}
		_, err := resolveInputTypeForContent(content, nil, ef)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be a non-empty string")
	})

	t.Run("intent mapping error is wrapped", func(t *testing.T) {
		content := embeddings.Content{
			Parts:  []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
			Intent: embeddings.IntentClassification,
		}
		_, err := resolveInputTypeForContent(content, nil, ef)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to map intent")
	})
}

// TestVoyageConvertToVoyageInput verifies content-to-VoyageInput conversion.
func TestVoyageConvertToVoyageInput(t *testing.T) {
	t.Run("text part", func(t *testing.T) {
		content := embeddings.Content{
			Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
		}
		result, err := convertToVoyageInput(content, maxMultimodalFileSize)
		require.NoError(t, err)
		require.Len(t, result.Content, 1)
		assert.Equal(t, "text", result.Content[0].Type)
		assert.Equal(t, "hello", result.Content[0].Text)
	})

	t.Run("image URL part", func(t *testing.T) {
		content := embeddings.Content{
			Parts: []embeddings.Part{
				{
					Modality: embeddings.ModalityImage,
					Source: &embeddings.BinarySource{
						Kind: embeddings.SourceKindURL,
						URL:  "https://example.com/photo.png",
					},
				},
			},
		}
		result, err := convertToVoyageInput(content, maxMultimodalFileSize)
		require.NoError(t, err)
		require.Len(t, result.Content, 1)
		assert.Equal(t, "image_url", result.Content[0].Type)
		assert.Equal(t, "https://example.com/photo.png", result.Content[0].ImageURL)
	})

	t.Run("image base64 part", func(t *testing.T) {
		content := embeddings.Content{
			Parts: []embeddings.Part{
				{
					Modality: embeddings.ModalityImage,
					Source: &embeddings.BinarySource{
						Kind:     embeddings.SourceKindBytes,
						Bytes:    []byte("fake-png-data"),
						MIMEType: "image/png",
					},
				},
			},
		}
		result, err := convertToVoyageInput(content, maxMultimodalFileSize)
		require.NoError(t, err)
		require.Len(t, result.Content, 1)
		assert.Equal(t, "image_base64", result.Content[0].Type)
		assert.Contains(t, result.Content[0].ImageBase64, "data:image/png;base64,")
	})

	t.Run("video URL part", func(t *testing.T) {
		content := embeddings.Content{
			Parts: []embeddings.Part{
				{
					Modality: embeddings.ModalityVideo,
					Source: &embeddings.BinarySource{
						Kind: embeddings.SourceKindURL,
						URL:  "https://example.com/video.mp4",
					},
				},
			},
		}
		result, err := convertToVoyageInput(content, maxMultimodalFileSize)
		require.NoError(t, err)
		require.Len(t, result.Content, 1)
		assert.Equal(t, "video_url", result.Content[0].Type)
		assert.Equal(t, "https://example.com/video.mp4", result.Content[0].VideoURL)
	})

	t.Run("mixed text+image", func(t *testing.T) {
		content := embeddings.Content{
			Parts: []embeddings.Part{
				{Modality: embeddings.ModalityText, Text: "describe this"},
				{
					Modality: embeddings.ModalityImage,
					Source: &embeddings.BinarySource{
						Kind: embeddings.SourceKindURL,
						URL:  "https://example.com/photo.png",
					},
				},
			},
		}
		result, err := convertToVoyageInput(content, maxMultimodalFileSize)
		require.NoError(t, err)
		require.Len(t, result.Content, 2)
		assert.Equal(t, "text", result.Content[0].Type)
		assert.Equal(t, "image_url", result.Content[1].Type)
	})

	t.Run("unsupported modality audio", func(t *testing.T) {
		content := embeddings.Content{
			Parts: []embeddings.Part{
				{
					Modality: embeddings.ModalityAudio,
					Source: &embeddings.BinarySource{
						Kind:     embeddings.SourceKindBytes,
						Bytes:    []byte("audio-data"),
						MIMEType: "audio/mpeg",
					},
				},
			},
		}
		_, err := convertToVoyageInput(content, maxMultimodalFileSize)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported modality")
	})
}

// TestVoyageResolveMIME verifies MIME resolution logic.
func TestVoyageResolveMIME(t *testing.T) {
	t.Run("explicit MIME type used", func(t *testing.T) {
		source := &embeddings.BinarySource{MIMEType: "image/jpeg"}
		mime, err := resolveMIME(source)
		require.NoError(t, err)
		assert.Equal(t, "image/jpeg", mime)
	})

	t.Run("extension fallback .png", func(t *testing.T) {
		source := &embeddings.BinarySource{
			Kind:     embeddings.SourceKindFile,
			FilePath: "/path/test.png",
		}
		mime, err := resolveMIME(source)
		require.NoError(t, err)
		assert.Equal(t, "image/png", mime)
	})

	t.Run("extension fallback .mp4", func(t *testing.T) {
		source := &embeddings.BinarySource{
			Kind:     embeddings.SourceKindFile,
			FilePath: "/path/clip.mp4",
		}
		mime, err := resolveMIME(source)
		require.NoError(t, err)
		assert.Equal(t, "video/mp4", mime)
	})

	t.Run("no MIME and no extension fails", func(t *testing.T) {
		source := &embeddings.BinarySource{
			Kind:  embeddings.SourceKindBytes,
			Bytes: []byte("data"),
		}
		_, err := resolveMIME(source)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MIME type is required")
	})

	t.Run("nil source", func(t *testing.T) {
		_, err := resolveMIME(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source cannot be nil")
	})

	t.Run("unknown extension fails", func(t *testing.T) {
		source := &embeddings.BinarySource{
			Kind:     embeddings.SourceKindFile,
			FilePath: "/path/test.bmp",
		}
		_, err := resolveMIME(source)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MIME type is required")
	})

	t.Run("case insensitive extension", func(t *testing.T) {
		source := &embeddings.BinarySource{
			Kind:     embeddings.SourceKindFile,
			FilePath: "/path/test.PNG",
		}
		mime, err := resolveMIME(source)
		require.NoError(t, err)
		assert.Equal(t, "image/png", mime)
	})

	t.Run("URL path extension fallback for png", func(t *testing.T) {
		source := &embeddings.BinarySource{Kind: embeddings.SourceKindURL, URL: "https://example.com/photo.png"}
		mime, err := resolveMIME(source)
		require.NoError(t, err)
		assert.Equal(t, "image/png", mime)
	})

	t.Run("URL with query string strips before extension", func(t *testing.T) {
		source := &embeddings.BinarySource{Kind: embeddings.SourceKindURL, URL: "https://example.com/photo.jpg?token=abc"}
		mime, err := resolveMIME(source)
		require.NoError(t, err)
		assert.Equal(t, "image/jpeg", mime)
	})

	t.Run("URL with fragment strips before extension", func(t *testing.T) {
		source := &embeddings.BinarySource{Kind: embeddings.SourceKindURL, URL: "https://example.com/clip.mp4#section"}
		mime, err := resolveMIME(source)
		require.NoError(t, err)
		assert.Equal(t, "video/mp4", mime)
	})

	t.Run("URL with no extension still fails", func(t *testing.T) {
		source := &embeddings.BinarySource{Kind: embeddings.SourceKindURL, URL: "https://example.com/image-no-ext"}
		_, err := resolveMIME(source)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MIME type is required")
	})

	t.Run("URL with unknown extension fails", func(t *testing.T) {
		source := &embeddings.BinarySource{Kind: embeddings.SourceKindURL, URL: "https://example.com/data.parquet"}
		_, err := resolveMIME(source)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MIME type is required")
	})

	t.Run("URL with case-mixed extension resolves", func(t *testing.T) {
		source := &embeddings.BinarySource{Kind: embeddings.SourceKindURL, URL: "https://example.com/photo.PNG"}
		mime, err := resolveMIME(source)
		require.NoError(t, err)
		assert.Equal(t, "image/png", mime)
	})

	t.Run("malformed URL returns parse error", func(t *testing.T) {
		source := &embeddings.BinarySource{Kind: embeddings.SourceKindURL, URL: "://invalid"}
		_, err := resolveMIME(source)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse source URL")
	})
}

// TestVoyageValidateMIMEModality verifies MIME/modality consistency checks.
func TestVoyageValidateMIMEModality(t *testing.T) {
	t.Run("image modality with image MIME passes", func(t *testing.T) {
		err := validateMIMEModality(embeddings.ModalityImage, "image/png")
		assert.NoError(t, err)
	})

	t.Run("video modality with video MIME passes", func(t *testing.T) {
		err := validateMIMEModality(embeddings.ModalityVideo, "video/mp4")
		assert.NoError(t, err)
	})

	t.Run("image modality with video MIME fails", func(t *testing.T) {
		err := validateMIMEModality(embeddings.ModalityImage, "video/mp4")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "image modality requires image/* MIME type")
	})

	t.Run("video modality with image MIME fails", func(t *testing.T) {
		err := validateMIMEModality(embeddings.ModalityVideo, "image/png")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "video modality requires video/* MIME type")
	})

	t.Run("unknown modality fails", func(t *testing.T) {
		err := validateMIMEModality(embeddings.ModalityAudio, "audio/mpeg")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MIME validation not implemented for modality")
	})
}

// TestVoyageResolveBytes verifies binary source resolution.
func TestVoyageResolveBytes(t *testing.T) {
	t.Run("nil source", func(t *testing.T) {
		_, err := resolveBytes(nil, 1024)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source cannot be nil")
	})

	t.Run("bytes source happy path", func(t *testing.T) {
		src := &embeddings.BinarySource{Kind: embeddings.SourceKindBytes, Bytes: []byte("hello")}
		data, err := resolveBytes(src, 1024)
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), data)
	})

	t.Run("bytes source exceeds max size", func(t *testing.T) {
		src := &embeddings.BinarySource{Kind: embeddings.SourceKindBytes, Bytes: []byte("toolarge")}
		_, err := resolveBytes(src, 3)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum")
	})

	t.Run("base64 source happy path", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("hello"))
		src := &embeddings.BinarySource{Kind: embeddings.SourceKindBase64, Base64: encoded}
		data, err := resolveBytes(src, 1024)
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), data)
	})

	t.Run("base64 source invalid encoding", func(t *testing.T) {
		src := &embeddings.BinarySource{Kind: embeddings.SourceKindBase64, Base64: "!!!not-base64!!!"}
		_, err := resolveBytes(src, 1024)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode base64")
	})

	t.Run("base64 source exceeds estimated size", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("toolarge"))
		src := &embeddings.BinarySource{Kind: embeddings.SourceKindBase64, Base64: encoded}
		_, err := resolveBytes(src, 3)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "base64 payload too large")
	})

	t.Run("file source happy path", func(t *testing.T) {
		tmp := t.TempDir()
		fp := filepath.Join(tmp, "test.png")
		require.NoError(t, os.WriteFile(fp, []byte("png-data"), 0644))

		src := &embeddings.BinarySource{Kind: embeddings.SourceKindFile, FilePath: fp}
		data, err := resolveBytes(src, 1024)
		require.NoError(t, err)
		assert.Equal(t, []byte("png-data"), data)
	})

	t.Run("file source not found", func(t *testing.T) {
		src := &embeddings.BinarySource{Kind: embeddings.SourceKindFile, FilePath: "/nonexistent/file.png"}
		_, err := resolveBytes(src, 1024)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open file source")
	})

	t.Run("file source exceeds max size", func(t *testing.T) {
		tmp := t.TempDir()
		fp := filepath.Join(tmp, "big.bin")
		require.NoError(t, os.WriteFile(fp, []byte("toolarge"), 0644))

		src := &embeddings.BinarySource{Kind: embeddings.SourceKindFile, FilePath: fp}
		_, err := resolveBytes(src, 3)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file size exceeds maximum")
	})

	t.Run("file source path traversal rejected", func(t *testing.T) {
		src := &embeddings.BinarySource{Kind: embeddings.SourceKindFile, FilePath: "../../../etc/passwd"}
		_, err := resolveBytes(src, 1024)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal")
	})

	t.Run("URL source returns passthrough error", func(t *testing.T) {
		src := &embeddings.BinarySource{Kind: embeddings.SourceKindURL, URL: "https://example.com/img.png"}
		_, err := resolveBytes(src, 1024)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "URL sources are handled via direct passthrough")
	})

	t.Run("unknown source kind", func(t *testing.T) {
		src := &embeddings.BinarySource{Kind: "unknown"}
		_, err := resolveBytes(src, 1024)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported source kind")
	})
}

// TestVoyageMultimodalURL verifies multimodal endpoint URL derivation.
func TestVoyageMultimodalURL(t *testing.T) {
	t.Run("default URL replaced", func(t *testing.T) {
		result, err := multimodalURL("https://api.voyageai.com/v1/embeddings")
		require.NoError(t, err)
		assert.Equal(t, "https://api.voyageai.com/v1/multimodalembeddings", result)
	})

	t.Run("custom proxy with standard path", func(t *testing.T) {
		result, err := multimodalURL("https://my-proxy.com/v1/embeddings")
		require.NoError(t, err)
		assert.Equal(t, "https://my-proxy.com/v1/multimodalembeddings", result)
	})

	t.Run("non-standard path returns error", func(t *testing.T) {
		_, err := multimodalURL("https://custom.proxy.com/api")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot derive multimodal endpoint")
	})
}

// mockEmbeddingData is a test-only struct that serializes the embedding field
// as a raw []float32 array, matching the Voyage API wire format.
// (EmbeddingTypeResult has a custom UnmarshalJSON that expects a raw array,
// but default MarshalJSON produces {"Floats":[...]} which doesn't round-trip.)
type mockEmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type mockEmbeddingResponse struct {
	Object string               `json:"object"`
	Data   []mockEmbeddingData  `json:"data"`
	Model  string               `json:"model"`
}

// newTestEF creates a VoyageAIEmbeddingFunction backed by the given mock server.
func newTestEF(t *testing.T, srv *httptest.Server) *VoyageAIEmbeddingFunction {
	t.Helper()
	ef, err := NewVoyageAIEmbeddingFunction(
		WithBaseURL(srv.URL+"/v1/embeddings"),
		WithAPIKey("test-key"),
		WithDefaultModel("voyage-multimodal-3.5"),
		WithInsecure(),
	)
	require.NoError(t, err)
	return ef
}

// mockEmbeddingServer returns a test server that responds with the given embedding data.
func mockEmbeddingServer(t *testing.T, results []mockEmbeddingData) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := mockEmbeddingResponse{
			Object: "list",
			Data:   results,
			Model:  "voyage-multimodal-3.5",
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestVoyageEmbedContentHappyPath verifies the EmbedContent full round-trip.
func TestVoyageEmbedContentHappyPath(t *testing.T) {
	var capturedReq CreateMultimodalEmbeddingRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedReq))
		w.Header().Set("Content-Type", "application/json")
		resp := mockEmbeddingResponse{
			Object: "list",
			Data: []mockEmbeddingData{
				{Object: "embedding", Embedding: []float32{0.1, 0.2, 0.3}, Index: 0},
			},
			Model: "voyage-multimodal-3.5",
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	t.Cleanup(srv.Close)

	ef := newTestEF(t, srv)
	content := embeddings.Content{
		Parts:  []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello world"}},
		Intent: embeddings.IntentRetrievalQuery,
	}

	emb, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	assert.Len(t, emb.ContentAsFloat32(), 3)
	assert.InDelta(t, 0.1, emb.ContentAsFloat32()[0], 1e-6)

	// Verify request body
	assert.Equal(t, "voyage-multimodal-3.5", capturedReq.Model)
	require.Len(t, capturedReq.Inputs, 1)
	require.Len(t, capturedReq.Inputs[0].Content, 1)
	assert.Equal(t, "text", capturedReq.Inputs[0].Content[0].Type)
	require.NotNil(t, capturedReq.InputType)
	assert.Equal(t, InputTypeQuery, *capturedReq.InputType)
}

// TestVoyageEmbedContentWithDimension verifies dimension is passed through.
func TestVoyageEmbedContentWithDimension(t *testing.T) {
	var capturedReq CreateMultimodalEmbeddingRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedReq))
		w.Header().Set("Content-Type", "application/json")
		resp := mockEmbeddingResponse{
			Object: "list",
			Data: []mockEmbeddingData{
				{Object: "embedding", Embedding: []float32{0.1, 0.2}, Index: 0},
			},
			Model: "voyage-multimodal-3.5",
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	t.Cleanup(srv.Close)

	ef := newTestEF(t, srv)
	dim := 256
	content := embeddings.Content{
		Parts:     []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
		Dimension: &dim,
	}

	_, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, capturedReq.OutputDimension)
	assert.Equal(t, 256, *capturedReq.OutputDimension)
}

// TestVoyageEmbedContentErrors verifies error responses from EmbedContent.
func TestVoyageEmbedContentErrors(t *testing.T) {
	t.Run("empty response data", func(t *testing.T) {
		srv := mockEmbeddingServer(t, []mockEmbeddingData{})
		ef := newTestEF(t, srv)
		content := embeddings.Content{
			Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
		}
		_, err := ef.EmbedContent(context.Background(), content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no embedding returned")
	})

	t.Run("nil embedding in response", func(t *testing.T) {
		// Server returns a result with no embedding field (null in JSON)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","data":[{"object":"embedding","embedding":null,"index":0}],"model":"voyage-multimodal-3.5"}`))
		}))
		t.Cleanup(srv.Close)
		ef := newTestEF(t, srv)
		content := embeddings.Content{
			Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
		}
		_, err := ef.EmbedContent(context.Background(), content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil embedding")
	})

	t.Run("empty floats in response", func(t *testing.T) {
		srv := mockEmbeddingServer(t, []mockEmbeddingData{{Object: "embedding", Embedding: []float32{}, Index: 0}})
		ef := newTestEF(t, srv)
		content := embeddings.Content{
			Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
		}
		_, err := ef.EmbedContent(context.Background(), content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty embedding vector")
	})

	t.Run("API error is wrapped", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"server error"}`))
		}))
		t.Cleanup(srv.Close)
		ef := newTestEF(t, srv)
		content := embeddings.Content{
			Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
		}
		_, err := ef.EmbedContent(context.Background(), content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to embed multimodal content")
	})
}

// TestVoyageEmbedContentContextInputType verifies context-level input type override.
func TestVoyageEmbedContentContextInputType(t *testing.T) {
	var capturedReq CreateMultimodalEmbeddingRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedReq))
		w.Header().Set("Content-Type", "application/json")
		resp := mockEmbeddingResponse{
			Object: "list",
			Data: []mockEmbeddingData{
				{Object: "embedding", Embedding: []float32{0.1}, Index: 0},
			},
			Model: "voyage-multimodal-3.5",
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	t.Cleanup(srv.Close)

	ef := newTestEF(t, srv)
	ctx := ContextWithInputType(context.Background(), InputTypeDocument)
	content := embeddings.Content{
		Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
	}

	_, err := ef.EmbedContent(ctx, content)
	require.NoError(t, err)
	require.NotNil(t, capturedReq.InputType)
	assert.Equal(t, InputTypeDocument, *capturedReq.InputType)
}

// TestVoyageEmbedContentsHappyPath verifies batch embedding round-trip.
func TestVoyageEmbedContentsHappyPath(t *testing.T) {
	srv := mockEmbeddingServer(t, []mockEmbeddingData{
		{Object: "embedding", Embedding: []float32{1, 2, 3}, Index: 0},
		{Object: "embedding", Embedding: []float32{4, 5, 6}, Index: 1},
	})
	ef := newTestEF(t, srv)

	contents := []embeddings.Content{
		{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "first"}}},
		{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "second"}}},
	}
	embs, err := ef.EmbedContents(context.Background(), contents)
	require.NoError(t, err)
	require.Len(t, embs, 2)
	assert.Len(t, embs[0].ContentAsFloat32(), 3)
	assert.Len(t, embs[1].ContentAsFloat32(), 3)
}

// TestVoyageEmbedContentsContextInputType verifies context-level input type for multi-item batch.
func TestVoyageEmbedContentsContextInputType(t *testing.T) {
	var capturedReq CreateMultimodalEmbeddingRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedReq))
		w.Header().Set("Content-Type", "application/json")
		resp := mockEmbeddingResponse{
			Object: "list",
			Data: []mockEmbeddingData{
				{Object: "embedding", Embedding: []float32{0.1}, Index: 0},
				{Object: "embedding", Embedding: []float32{0.2}, Index: 1},
			},
			Model: "voyage-multimodal-3.5",
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	t.Cleanup(srv.Close)

	ef := newTestEF(t, srv)
	ctx := ContextWithInputType(context.Background(), InputTypeDocument)
	contents := []embeddings.Content{
		{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "a"}}},
		{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "b"}}},
	}

	_, err := ef.EmbedContents(ctx, contents)
	require.NoError(t, err)
	require.NotNil(t, capturedReq.InputType)
	assert.Equal(t, InputTypeDocument, *capturedReq.InputType)
}

// TestVoyageEmbedContentsCountMismatch verifies response count validation.
func TestVoyageEmbedContentsCountMismatch(t *testing.T) {
	// Server returns 1 embedding for 2 inputs
	srv := mockEmbeddingServer(t, []mockEmbeddingData{
		{Object: "embedding", Embedding: []float32{1, 2, 3}, Index: 0},
	})
	ef := newTestEF(t, srv)

	contents := []embeddings.Content{
		{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "first"}}},
		{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "second"}}},
	}
	_, err := ef.EmbedContents(context.Background(), contents)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected 2 embeddings")
}

// TestVoyageEmbedContentsErrors verifies batch error handling.
func TestVoyageEmbedContentsErrors(t *testing.T) {
	t.Run("nil embedding in batch", func(t *testing.T) {
		// Server returns raw JSON with null embedding
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","data":[{"object":"embedding","embedding":[1],"index":0},{"object":"embedding","embedding":null,"index":1}],"model":"voyage-multimodal-3.5"}`))
		}))
		t.Cleanup(srv.Close)
		ef := newTestEF(t, srv)

		contents := []embeddings.Content{
			{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "a"}}},
			{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "b"}}},
		}
		_, err := ef.EmbedContents(context.Background(), contents)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil embedding")
	})

	t.Run("empty floats in batch", func(t *testing.T) {
		srv := mockEmbeddingServer(t, []mockEmbeddingData{
			{Object: "embedding", Embedding: []float32{}, Index: 0},
		})
		ef := newTestEF(t, srv)

		contents := []embeddings.Content{
			{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "a"}}},
		}
		_, err := ef.EmbedContents(context.Background(), contents)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty embedding vector")
	})

	t.Run("API error is wrapped", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"bad request"}`))
		}))
		t.Cleanup(srv.Close)
		ef := newTestEF(t, srv)

		contents := []embeddings.Content{
			{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "a"}}},
		}
		_, err := ef.EmbedContents(context.Background(), contents)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to embed multimodal contents batch")
	})
}

// TestVoyageBatchRejectsPerItemOverrides verifies batch rejection of per-item overrides.
func TestVoyageBatchRejectsPerItemOverrides(t *testing.T) {
	ef := &VoyageAIEmbeddingFunction{
		apiClient: &VoyageAIClient{
			DefaultModel: "voyage-multimodal-3.5",
			MaxBatchSize: 128,
		},
	}

	t.Run("batch rejects per-item Intent", func(t *testing.T) {
		contents := []embeddings.Content{
			{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "a"}}, Intent: embeddings.IntentRetrievalQuery},
			{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "b"}}},
		}
		_, err := ef.EmbedContents(context.Background(), contents)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "per-item Intent")
	})

	t.Run("batch rejects per-item Dimension", func(t *testing.T) {
		dim := 256
		contents := []embeddings.Content{
			{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "a"}}, Dimension: &dim},
			{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "b"}}},
		}
		_, err := ef.EmbedContents(context.Background(), contents)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "per-item Dimension")
	})

	t.Run("batch rejects per-item ProviderHints input_type", func(t *testing.T) {
		contents := []embeddings.Content{
			{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "a"}}, ProviderHints: map[string]any{"input_type": "query"}},
			{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "b"}}},
		}
		_, err := ef.EmbedContents(context.Background(), contents)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "per-item ProviderHints")
	})
}

// TestVoyageContentRegistration verifies RegisterContent("voyageai") is registered.
func TestVoyageContentRegistration(t *testing.T) {
	assert.True(t, embeddings.HasContent("voyageai"))
}

// TestVoyageContentRegistrationDefaultModel verifies content factory defaults to multimodal model.
func TestVoyageContentRegistrationDefaultModel(t *testing.T) {
	t.Setenv("VOYAGE_API_KEY", "test-key")
	cfg := embeddings.EmbeddingFunctionConfig{
		"api_key_env_var": "VOYAGE_API_KEY",
	}
	ef, err := embeddings.BuildContent("voyageai", cfg)
	require.NoError(t, err)

	capAware, ok := ef.(embeddings.CapabilityAware)
	require.True(t, ok, "content factory result should implement CapabilityAware")
	caps := capAware.Capabilities()
	assert.True(t, caps.SupportsMixedPart, "content factory should default to multimodal model")
	assert.Len(t, caps.Modalities, 3, "content factory should support text/image/video by default")
}

// TestVoyageCapabilitiesForContext verifies capabilities honor context model override.
func TestVoyageCapabilitiesForContext(t *testing.T) {
	ef := &VoyageAIEmbeddingFunction{
		apiClient: &VoyageAIClient{DefaultModel: "voyage-multimodal-3.5"},
	}

	t.Run("default model has 3 modalities", func(t *testing.T) {
		caps := ef.capabilitiesForContext(context.Background())
		assert.Len(t, caps.Modalities, 3)
	})

	t.Run("context override to multimodal-3 has 2 modalities", func(t *testing.T) {
		ctx := ContextWithModel(context.Background(), "voyage-multimodal-3")
		caps := ef.capabilitiesForContext(ctx)
		assert.Len(t, caps.Modalities, 2)
		assert.Contains(t, caps.Modalities, embeddings.ModalityText)
		assert.Contains(t, caps.Modalities, embeddings.ModalityImage)
	})

	t.Run("context override to text-only model has 1 modality", func(t *testing.T) {
		ctx := ContextWithModel(context.Background(), "voyage-2")
		caps := ef.capabilitiesForContext(ctx)
		assert.Len(t, caps.Modalities, 1)
	})
}

// TestVoyageEmbedContentRejectsVideoForContextModel verifies validation uses context model.
func TestVoyageEmbedContentRejectsVideoForContextModel(t *testing.T) {
	ef := &VoyageAIEmbeddingFunction{
		apiClient: &VoyageAIClient{
			DefaultModel: "voyage-multimodal-3.5", // supports video
			MaxBatchSize: 128,
		},
	}
	// Override to multimodal-3 which does NOT support video
	ctx := ContextWithModel(context.Background(), "voyage-multimodal-3")
	content := embeddings.Content{
		Parts: []embeddings.Part{
			{
				Modality: embeddings.ModalityVideo,
				Source: &embeddings.BinarySource{
					Kind: embeddings.SourceKindURL,
					URL:  "https://example.com/video.mp4",
				},
			},
		},
	}

	_, err := ef.EmbedContent(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "video")
}

// TestVoyageConfigRoundTrip verifies config round-trip GetConfig -> FromConfig -> GetConfig.
func TestVoyageConfigRoundTrip(t *testing.T) {
	t.Setenv("VOYAGE_API_KEY", "test-key-for-config")

	ef, err := NewVoyageAIEmbeddingFunction(
		WithAPIKey("test-key-for-config"),
		WithDefaultModel("voyage-multimodal-3.5"),
	)
	require.NoError(t, err)

	cfg := ef.GetConfig()
	assert.Equal(t, "voyage-multimodal-3.5", cfg["model_name"])
	assert.Equal(t, "VOYAGE_API_KEY", cfg["api_key_env_var"])

	ef2, err := NewVoyageAIEmbeddingFunctionFromConfig(cfg)
	require.NoError(t, err)

	cfg2 := ef2.GetConfig()
	assert.Equal(t, cfg["model_name"], cfg2["model_name"])
	assert.Equal(t, cfg["api_key_env_var"], cfg2["api_key_env_var"])
}
