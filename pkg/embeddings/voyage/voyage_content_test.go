//go:build ef

package voyage

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// TestVoyageCapabilitiesForModel verifies capability derivation for known models (VOY-01).
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

// TestVoyageCapabilities verifies Capabilities() on VoyageAIEmbeddingFunction (VOY-01).
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

// TestVoyageMapIntent verifies supported intent mapping (VOY-02).
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

// TestVoyageMapIntentRejects verifies unsupported intents are rejected (VOY-02).
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
		assert.Contains(t, err.Error(), "unsupported intent")
	})
}

// TestVoyageResolveInputType verifies priority chain for input type resolution (VOY-02).
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

// TestVoyageConvertToVoyageInput verifies content-to-VoyageInput conversion (VOY-01).
func TestVoyageConvertToVoyageInput(t *testing.T) {
	ctx := context.Background()

	t.Run("text part", func(t *testing.T) {
		content := embeddings.Content{
			Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
		}
		result, err := convertToVoyageInput(ctx, content, maxMultimodalFileSize)
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
		result, err := convertToVoyageInput(ctx, content, maxMultimodalFileSize)
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
		result, err := convertToVoyageInput(ctx, content, maxMultimodalFileSize)
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
		result, err := convertToVoyageInput(ctx, content, maxMultimodalFileSize)
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
		result, err := convertToVoyageInput(ctx, content, maxMultimodalFileSize)
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
		_, err := convertToVoyageInput(ctx, content, maxMultimodalFileSize)
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

	t.Run("no MIME and no extension fails", func(t *testing.T) {
		source := &embeddings.BinarySource{
			Kind:  embeddings.SourceKindBytes,
			Bytes: []byte("data"),
		}
		_, err := resolveMIME(source)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MIME type is required")
	})
}

// TestVoyageMultimodalURL verifies multimodal endpoint URL derivation.
func TestVoyageMultimodalURL(t *testing.T) {
	t.Run("default URL replaced", func(t *testing.T) {
		result := multimodalURL("https://api.voyageai.com/v1/embeddings")
		assert.Equal(t, "https://api.voyageai.com/v1/multimodalembeddings", result)
	})

	t.Run("custom URL uses hardcoded", func(t *testing.T) {
		result := multimodalURL("https://custom.proxy.com/api")
		assert.Equal(t, multimodalBaseAPI, result)
	})
}

// TestVoyageBatchRejectsPerItemOverrides verifies batch rejection of per-item overrides (VOY-01, D-16).
func TestVoyageBatchRejectsPerItemOverrides(t *testing.T) {
	mockResp := CreateEmbeddingResponse{
		Object: "list",
		Data: []EmbeddingResult{
			{Object: "embedding", Embedding: &EmbeddingTypeResult{Floats: []float32{1, 2, 3}}, Index: 0},
			{Object: "embedding", Embedding: &EmbeddingTypeResult{Floats: []float32{4, 5, 6}}, Index: 1},
		},
		Model: "voyage-multimodal-3.5",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer srv.Close()

	ef, err := NewVoyageAIEmbeddingFunction(
		WithBaseURL(srv.URL+"/v1/embeddings"),
		WithAPIKey("test-key"),
		WithDefaultModel("voyage-multimodal-3.5"),
		WithInsecure(),
	)
	require.NoError(t, err)

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

// TestVoyageContentRegistration verifies RegisterContent("voyageai") is registered (VOY-03).
func TestVoyageContentRegistration(t *testing.T) {
	assert.True(t, embeddings.HasContent("voyageai"))
}

// TestVoyageConfigRoundTrip verifies config round-trip GetConfig -> FromConfig -> GetConfig (VOY-03).
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

// TestVoyageInterfaceAssertions confirms compile-time interface satisfaction (VOY-01).
func TestVoyageInterfaceAssertions(t *testing.T) {
	var _ embeddings.ContentEmbeddingFunction = (*VoyageAIEmbeddingFunction)(nil)
	var _ embeddings.CapabilityAware = (*VoyageAIEmbeddingFunction)(nil)
	var _ embeddings.IntentMapper = (*VoyageAIEmbeddingFunction)(nil)

	ef := &VoyageAIEmbeddingFunction{
		apiClient: &VoyageAIClient{DefaultModel: "voyage-multimodal-3.5"},
	}
	caps := ef.Capabilities()
	assert.NotEmpty(t, caps.Modalities)

	result, err := ef.MapIntent(embeddings.IntentRetrievalQuery)
	require.NoError(t, err)
	assert.Equal(t, "query", result)
}
