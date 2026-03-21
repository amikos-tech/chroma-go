package gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// TestDefaultModelChanged verifies the model constants (D-01).
func TestDefaultModelChanged(t *testing.T) {
	assert.Equal(t, "gemini-embedding-2-preview", DefaultEmbeddingModel)
	assert.Equal(t, "gemini-embedding-001", LegacyEmbeddingModel)
}

// TestCapabilitiesForModel verifies capability derivation for known models (GEM-01).
func TestCapabilitiesForModel(t *testing.T) {
	t.Run("gemini-embedding-2-preview returns 5 modalities", func(t *testing.T) {
		caps := capabilitiesForModel("gemini-embedding-2-preview")
		assert.Len(t, caps.Modalities, 5)
		assert.Contains(t, caps.Modalities, embeddings.ModalityText)
		assert.Contains(t, caps.Modalities, embeddings.ModalityImage)
		assert.Contains(t, caps.Modalities, embeddings.ModalityAudio)
		assert.Contains(t, caps.Modalities, embeddings.ModalityVideo)
		assert.Contains(t, caps.Modalities, embeddings.ModalityPDF)
		assert.Len(t, caps.Intents, 5)
		assert.True(t, caps.SupportsBatch)
		assert.True(t, caps.SupportsMixedPart)
		assert.True(t, caps.SupportsRequestOption(embeddings.RequestOptionDimension))
		assert.True(t, caps.SupportsRequestOption(embeddings.RequestOptionProviderHints))
	})

	t.Run("gemini-embedding-001 is text-only", func(t *testing.T) {
		caps := capabilitiesForModel("gemini-embedding-001")
		assert.Len(t, caps.Modalities, 1)
		assert.Equal(t, embeddings.ModalityText, caps.Modalities[0])
		assert.False(t, caps.SupportsMixedPart)
		assert.True(t, caps.SupportsBatch)
	})

	t.Run("unknown model falls back to text-only", func(t *testing.T) {
		caps := capabilitiesForModel("unknown-model")
		assert.Len(t, caps.Modalities, 1)
		assert.Equal(t, embeddings.ModalityText, caps.Modalities[0])
		assert.False(t, caps.SupportsMixedPart)
	})
}

// TestGeminiCapabilities verifies Capabilities() on GeminiEmbeddingFunction (GEM-01).
func TestGeminiCapabilities(t *testing.T) {
	t.Run("default model returns 5 modalities", func(t *testing.T) {
		ef := &GeminiEmbeddingFunction{
			apiClient: &Client{DefaultModel: DefaultEmbeddingModel},
		}
		caps := ef.Capabilities()
		assert.True(t, caps.SupportsMixedPart)
		assert.Len(t, caps.Modalities, 5)
	})

	t.Run("legacy model returns text-only", func(t *testing.T) {
		ef := &GeminiEmbeddingFunction{
			apiClient: &Client{DefaultModel: LegacyEmbeddingModel},
		}
		caps := ef.Capabilities()
		assert.False(t, caps.SupportsMixedPart)
		assert.Len(t, caps.Modalities, 1)
	})
}

// TestMapIntent verifies all 5 neutral intents map to correct Gemini task types (GEM-02).
func TestMapIntent(t *testing.T) {
	ef := &GeminiEmbeddingFunction{
		apiClient: &Client{DefaultModel: DefaultEmbeddingModel},
	}

	cases := []struct {
		intent   embeddings.Intent
		expected string
	}{
		{embeddings.IntentRetrievalQuery, "RETRIEVAL_QUERY"},
		{embeddings.IntentRetrievalDocument, "RETRIEVAL_DOCUMENT"},
		{embeddings.IntentClassification, "CLASSIFICATION"},
		{embeddings.IntentClustering, "CLUSTERING"},
		{embeddings.IntentSemanticSimilarity, "SEMANTIC_SIMILARITY"},
	}

	for _, tc := range cases {
		t.Run(string(tc.intent), func(t *testing.T) {
			result, err := ef.MapIntent(tc.intent)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestMapIntentRejectsNonNeutral verifies non-neutral intents are rejected with a helpful error (GEM-02).
func TestMapIntentRejectsNonNeutral(t *testing.T) {
	ef := &GeminiEmbeddingFunction{
		apiClient: &Client{DefaultModel: DefaultEmbeddingModel},
	}

	_, err := ef.MapIntent(embeddings.Intent("CODE_RETRIEVAL_QUERY"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported intent")
	assert.Contains(t, err.Error(), "ProviderHints")
}

// TestResolveMIME verifies MIME resolution logic (D-06).
func TestResolveMIME(t *testing.T) {
	t.Run("explicit MIMEType is returned directly", func(t *testing.T) {
		source := &embeddings.BinarySource{MIMEType: "image/png"}
		mime, err := resolveMIME(source)
		require.NoError(t, err)
		assert.Equal(t, "image/png", mime)
	})

	t.Run("file extension fallback for jpg", func(t *testing.T) {
		source := &embeddings.BinarySource{Kind: embeddings.SourceKindFile, FilePath: "/tmp/test.jpg"}
		mime, err := resolveMIME(source)
		require.NoError(t, err)
		assert.Equal(t, "image/jpeg", mime)
	})

	t.Run("file extension fallback for pdf", func(t *testing.T) {
		source := &embeddings.BinarySource{Kind: embeddings.SourceKindFile, FilePath: "/tmp/doc.pdf"}
		mime, err := resolveMIME(source)
		require.NoError(t, err)
		assert.Equal(t, "application/pdf", mime)
	})

	t.Run("no MIME and no file path returns error", func(t *testing.T) {
		source := &embeddings.BinarySource{Kind: embeddings.SourceKindBytes, Bytes: []byte("data")}
		_, err := resolveMIME(source)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MIME type is required")
	})

	t.Run("unknown file extension returns error", func(t *testing.T) {
		source := &embeddings.BinarySource{Kind: embeddings.SourceKindFile, FilePath: "/tmp/file.xyz"}
		_, err := resolveMIME(source)
		require.Error(t, err)
	})
}

// TestValidateMIMEModality verifies MIME-modality consistency checks (D-07).
func TestValidateMIMEModality(t *testing.T) {
	validCases := []struct {
		modality embeddings.Modality
		mimeType string
	}{
		{embeddings.ModalityImage, "image/png"},
		{embeddings.ModalityAudio, "audio/mpeg"},
		{embeddings.ModalityVideo, "video/mp4"},
		{embeddings.ModalityPDF, "application/pdf"},
		{embeddings.ModalityText, "text/plain"},
	}

	for _, tc := range validCases {
		t.Run(string(tc.modality)+"_valid", func(t *testing.T) {
			err := validateMIMEModality(tc.modality, tc.mimeType)
			require.NoError(t, err)
		})
	}

	invalidCases := []struct {
		modality    embeddings.Modality
		mimeType    string
		errContains string
	}{
		{embeddings.ModalityImage, "audio/mpeg", "image modality requires image/*"},
		{embeddings.ModalityAudio, "image/png", "audio modality requires audio/*"},
		{embeddings.ModalityVideo, "application/pdf", "video modality requires video/*"},
		{embeddings.ModalityPDF, "image/jpeg", "pdf modality requires application/pdf"},
	}

	for _, tc := range invalidCases {
		t.Run(string(tc.modality)+"_invalid", func(t *testing.T) {
			err := validateMIMEModality(tc.modality, tc.mimeType)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errContains)
		})
	}
}

// TestResolveBytesKinds verifies resolveBytes for all supported source kinds (D-09).
func TestResolveBytesKinds(t *testing.T) {
	ctx := context.Background()

	t.Run("SourceKindBytes returns bytes directly", func(t *testing.T) {
		source := &embeddings.BinarySource{
			Kind:  embeddings.SourceKindBytes,
			Bytes: []byte("hello"),
		}
		data, err := resolveBytes(ctx, source)
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), data)
	})

	t.Run("SourceKindBase64 decodes correctly", func(t *testing.T) {
		source := &embeddings.BinarySource{
			Kind:   embeddings.SourceKindBase64,
			Base64: base64.StdEncoding.EncodeToString([]byte("hello")),
		}
		data, err := resolveBytes(ctx, source)
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), data)
	})

	t.Run("SourceKindFile reads file contents", func(t *testing.T) {
		tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.bin")
		require.NoError(t, err)
		expected := []byte("file-content-123")
		_, err = tmpFile.Write(expected)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		source := &embeddings.BinarySource{
			Kind:     embeddings.SourceKindFile,
			FilePath: tmpFile.Name(),
		}
		data, err := resolveBytes(ctx, source)
		require.NoError(t, err)
		assert.Equal(t, expected, data)
	})

	t.Run("SourceKindURL is skipped in unit tests", func(t *testing.T) {
		t.Skip("requires HTTP server — tested in integration tests")
	})
}

// TestConvertToGenaiContentText verifies text-only content conversion.
func TestConvertToGenaiContentText(t *testing.T) {
	content := embeddings.Content{
		Parts: []embeddings.Part{
			{Modality: embeddings.ModalityText, Text: "hello world"},
		},
	}
	result, err := convertToGenaiContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Parts, 1)
	assert.Equal(t, "user", result.Role)
}

// TestConvertToGenaiContentBinary verifies binary image content conversion.
func TestConvertToGenaiContentBinary(t *testing.T) {
	content := embeddings.Content{
		Parts: []embeddings.Part{
			{
				Modality: embeddings.ModalityImage,
				Source: &embeddings.BinarySource{
					Kind:     embeddings.SourceKindBytes,
					Bytes:    []byte("fake-png"),
					MIMEType: "image/png",
				},
			},
		},
	}
	result, err := convertToGenaiContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Parts, 1)
}

// TestConvertToGenaiContentMixedParts verifies mixed text + image content conversion.
func TestConvertToGenaiContentMixedParts(t *testing.T) {
	content := embeddings.Content{
		Parts: []embeddings.Part{
			{Modality: embeddings.ModalityText, Text: "describe this image"},
			{
				Modality: embeddings.ModalityImage,
				Source: &embeddings.BinarySource{
					Kind:     embeddings.SourceKindBytes,
					Bytes:    []byte("fake-jpeg"),
					MIMEType: "image/jpeg",
				},
			},
		},
	}
	result, err := convertToGenaiContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Parts, 2)
}

// TestConvertToGenaiContentMissingMIME verifies error when MIME cannot be resolved.
func TestConvertToGenaiContentMissingMIME(t *testing.T) {
	content := embeddings.Content{
		Parts: []embeddings.Part{
			{
				Modality: embeddings.ModalityImage,
				Source: &embeddings.BinarySource{
					Kind:  embeddings.SourceKindBytes,
					Bytes: []byte("data"),
				},
			},
		},
	}
	_, err := convertToGenaiContent(context.Background(), content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MIME type is required")
}

// TestConvertToGenaiContentMIMEModalityMismatch verifies error for mismatched MIME and modality.
func TestConvertToGenaiContentMIMEModalityMismatch(t *testing.T) {
	content := embeddings.Content{
		Parts: []embeddings.Part{
			{
				Modality: embeddings.ModalityImage,
				Source: &embeddings.BinarySource{
					Kind:     embeddings.SourceKindBytes,
					Bytes:    []byte("data"),
					MIMEType: "audio/mpeg",
				},
			},
		},
	}
	_, err := convertToGenaiContent(context.Background(), content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "image modality requires image/*")
}

// TestConvertToGenaiContents verifies batch conversion of multiple content items.
func TestConvertToGenaiContents(t *testing.T) {
	ctx := context.Background()

	t.Run("converts multiple contents", func(t *testing.T) {
		contents := []embeddings.Content{
			{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "first"}}},
			{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "second"}}},
			{Parts: []embeddings.Part{
				{Modality: embeddings.ModalityText, Text: "text part"},
				{Modality: embeddings.ModalityImage, Source: &embeddings.BinarySource{
					Kind: embeddings.SourceKindBytes, Bytes: []byte("fake-png"), MIMEType: "image/png",
				}},
			}},
		}
		results, err := convertToGenaiContents(ctx, contents)
		require.NoError(t, err)
		require.Len(t, results, 3)
		assert.Len(t, results[0].Parts, 1)
		assert.Len(t, results[1].Parts, 1)
		assert.Len(t, results[2].Parts, 2)
	})

	t.Run("error in one content propagates with index", func(t *testing.T) {
		contents := []embeddings.Content{
			{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "ok"}}},
			{Parts: []embeddings.Part{{Modality: embeddings.ModalityImage, Source: &embeddings.BinarySource{
				Kind: embeddings.SourceKindBytes, Bytes: []byte("data"),
				// Missing MIMEType → error
			}}}},
		}
		_, err := convertToGenaiContents(ctx, contents)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "content[1]")
	})
}

// TestResolveBytesURL verifies URL source resolution using a local HTTP server.
func TestResolveBytesURL(t *testing.T) {
	expected := []byte("url-fetched-content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(expected)
	}))
	defer srv.Close()

	source := &embeddings.BinarySource{
		Kind: embeddings.SourceKindURL,
		URL:  srv.URL,
	}
	data, err := resolveBytes(context.Background(), source)
	require.NoError(t, err)
	assert.Equal(t, expected, data)
}

// TestResolveBytesUnsupportedKind verifies error for unknown source kinds.
func TestResolveBytesUnsupportedKind(t *testing.T) {
	source := &embeddings.BinarySource{Kind: embeddings.SourceKind("unknown")}
	_, err := resolveBytes(context.Background(), source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported source kind")
}

// TestResolveTaskTypeForContent verifies the priority chain for task type resolution (D-15, D-16).
func TestResolveTaskTypeForContent(t *testing.T) {
	mapper := &GeminiEmbeddingFunction{
		apiClient: &Client{DefaultModel: DefaultEmbeddingModel},
	}
	defaultType := TaskTypeRetrievalDocument

	t.Run("no hints no intent returns default", func(t *testing.T) {
		content := embeddings.Content{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}}}
		tt, err := resolveTaskTypeForContent(content, defaultType, mapper)
		require.NoError(t, err)
		assert.Equal(t, defaultType, tt)
	})

	t.Run("intent without hints maps via mapper", func(t *testing.T) {
		content := embeddings.Content{
			Parts:  []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
			Intent: embeddings.IntentRetrievalQuery,
		}
		tt, err := resolveTaskTypeForContent(content, defaultType, mapper)
		require.NoError(t, err)
		assert.Equal(t, TaskTypeRetrievalQuery, tt)
	})

	t.Run("ProviderHints task_type overrides intent", func(t *testing.T) {
		content := embeddings.Content{
			Parts:  []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
			Intent: embeddings.IntentRetrievalQuery,
			ProviderHints: map[string]any{
				"task_type": "CODE_RETRIEVAL_QUERY",
			},
		}
		tt, err := resolveTaskTypeForContent(content, defaultType, mapper)
		require.NoError(t, err)
		assert.Equal(t, TaskTypeCodeRetrievalQuery, tt)
	})

	t.Run("ProviderHints wins over intent (priority order)", func(t *testing.T) {
		content := embeddings.Content{
			Parts:  []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
			Intent: embeddings.IntentClustering,
			ProviderHints: map[string]any{
				"task_type": string(TaskTypeSemanticSimilarity),
			},
		}
		tt, err := resolveTaskTypeForContent(content, defaultType, mapper)
		require.NoError(t, err)
		assert.Equal(t, TaskTypeSemanticSimilarity, tt)
	})

	t.Run("invalid hint returns error", func(t *testing.T) {
		content := embeddings.Content{
			Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}},
			ProviderHints: map[string]any{
				"task_type": "INVALID",
			},
		}
		_, err := resolveTaskTypeForContent(content, defaultType, mapper)
		require.Error(t, err)
	})
}

// TestEmbedContentLegacyModelRejectsMultimodal verifies legacy model rejects image content (D-03, D-04).
func TestEmbedContentLegacyModelRejectsMultimodal(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")

	ef := &GeminiEmbeddingFunction{
		apiClient: &Client{DefaultModel: LegacyEmbeddingModel},
	}

	content := embeddings.Content{
		Parts: []embeddings.Part{
			{
				Modality: embeddings.ModalityImage,
				Source: &embeddings.BinarySource{
					Kind:     embeddings.SourceKindBytes,
					Bytes:    []byte("fake"),
					MIMEType: "image/png",
				},
			},
		},
	}

	_, err := ef.EmbedContent(context.Background(), content)
	require.Error(t, err)
	// ValidateContentSupport returns a ValidationError whose message mentions modality rejection.
	assert.True(t,
		strings.Contains(err.Error(), "unsupported") || strings.Contains(err.Error(), "does not support"),
		"expected error about unsupported modality, got: %s", err.Error(),
	)
}

// TestGeminiContentRegistration verifies "google_genai" is registered as a content factory (GEM-03).
func TestGeminiContentRegistration(t *testing.T) {
	assert.True(t, embeddings.HasContent("google_genai"))

	t.Setenv("GEMINI_API_KEY", "test-key")
	ef, err := embeddings.BuildContent("google_genai", embeddings.EmbeddingFunctionConfig{
		"api_key_env_var": "GEMINI_API_KEY",
	})
	require.NoError(t, err)
	require.NotNil(t, ef)

	ca, ok := ef.(embeddings.CapabilityAware)
	require.True(t, ok, "expected result to implement CapabilityAware")
	caps := ca.Capabilities()
	assert.Len(t, caps.Modalities, 5)
}

// TestGeminiContentConfigRoundTrip verifies Name()+GetConfig()->BuildContent produces a working instance (GEM-03, D-24, D-27).
func TestGeminiContentConfigRoundTrip(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")

	dim := int32(256)
	original := &GeminiEmbeddingFunction{
		apiClient: &Client{
			APIKeyEnvVar:     APIKeyEnvVar,
			DefaultModel:     DefaultEmbeddingModel,
			DefaultTaskType:  TaskTypeRetrievalDocument,
			DefaultDimension: &dim,
		},
	}

	name := original.Name()
	assert.Equal(t, "google_genai", name)

	cfg := original.GetConfig()
	require.NotNil(t, cfg)

	rebuilt, err := embeddings.BuildContent(name, cfg)
	require.NoError(t, err)
	require.NotNil(t, rebuilt)

	ca, ok := rebuilt.(embeddings.CapabilityAware)
	require.True(t, ok, "rebuilt instance should implement CapabilityAware")
	caps := ca.Capabilities()
	originalCaps := original.Capabilities()
	assert.Equal(t, len(originalCaps.Modalities), len(caps.Modalities))
	assert.Equal(t, originalCaps.SupportsMixedPart, caps.SupportsMixedPart)
}

// --- Mock-based tests for defensive error paths ---

// newMockGenaiClient creates a genai.Client that uses the given HTTP handler for all requests.
func newMockGenaiClient(t *testing.T, handler http.Handler) *genai.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  "fake-key",
		Backend: genai.BackendGeminiAPI,
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{base: http.DefaultTransport, target: srv.URL},
		},
	})
	require.NoError(t, err)
	return client
}

// rewriteTransport redirects all requests to the mock server.
type rewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.target, "http://")
	return t.base.RoundTrip(req)
}

// geminiEmbedResponse builds a JSON response matching the Gemini batchEmbedContents format.
func geminiEmbedResponse(vectors ...[]float32) []byte {
	type emb struct {
		Values []float32 `json:"values"`
	}
	type resp struct {
		Embeddings []emb `json:"embeddings"`
	}
	r := resp{}
	for _, v := range vectors {
		r.Embeddings = append(r.Embeddings, emb{Values: v})
	}
	data, _ := json.Marshal(r)
	return data
}

func TestCreateContentEmbeddingMock(t *testing.T) {
	t.Run("successful single content embedding", func(t *testing.T) {
		client := newMockGenaiClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(geminiEmbedResponse([]float32{1.0, 2.0, 3.0}))
		}))
		c := &Client{
			Client:       client,
			DefaultModel: DefaultEmbeddingModel,
			APIKey:       embeddings.NewSecret("fake"),
		}
		mapper := &GeminiEmbeddingFunction{apiClient: c}
		contents := []embeddings.Content{
			{Parts: []embeddings.Part{embeddings.NewTextPart("hello")}},
		}
		results, err := c.CreateContentEmbedding(context.Background(), contents, mapper)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, []float32{1.0, 2.0, 3.0}, results[0].ContentAsFloat32())
	})

	t.Run("empty embeddings response returns error", func(t *testing.T) {
		client := newMockGenaiClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"embeddings":[]}`))
		}))
		c := &Client{
			Client:       client,
			DefaultModel: DefaultEmbeddingModel,
			APIKey:       embeddings.NewSecret("fake"),
		}
		mapper := &GeminiEmbeddingFunction{apiClient: c}
		contents := []embeddings.Content{
			{Parts: []embeddings.Part{embeddings.NewTextPart("hello")}},
		}
		_, err := c.CreateContentEmbedding(context.Background(), contents, mapper)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no embeddings returned")
	})

	t.Run("API error propagates", func(t *testing.T) {
		client := newMockGenaiClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"message":"server error","status":"INTERNAL"}}`))
		}))
		c := &Client{
			Client:       client,
			DefaultModel: DefaultEmbeddingModel,
			APIKey:       embeddings.NewSecret("fake"),
		}
		mapper := &GeminiEmbeddingFunction{apiClient: c}
		contents := []embeddings.Content{
			{Parts: []embeddings.Part{embeddings.NewTextPart("hello")}},
		}
		_, err := c.CreateContentEmbedding(context.Background(), contents, mapper)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to embed contents")
	})

	t.Run("batch embedding returns multiple vectors", func(t *testing.T) {
		client := newMockGenaiClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(geminiEmbedResponse([]float32{1, 2, 3}, []float32{4, 5, 6}))
		}))
		c := &Client{
			Client:       client,
			DefaultModel: DefaultEmbeddingModel,
			APIKey:       embeddings.NewSecret("fake"),
		}
		mapper := &GeminiEmbeddingFunction{apiClient: c}
		contents := []embeddings.Content{
			{Parts: []embeddings.Part{embeddings.NewTextPart("first")}},
			{Parts: []embeddings.Part{embeddings.NewTextPart("second")}},
		}
		results, err := c.CreateContentEmbedding(context.Background(), contents, mapper)
		require.NoError(t, err)
		require.Len(t, results, 2)
		assert.Equal(t, []float32{1, 2, 3}, results[0].ContentAsFloat32())
		assert.Equal(t, []float32{4, 5, 6}, results[1].ContentAsFloat32())
	})
}

func TestEmbedContentMock(t *testing.T) {
	t.Run("happy path returns single embedding", func(t *testing.T) {
		client := newMockGenaiClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(geminiEmbedResponse([]float32{0.1, 0.2, 0.3}))
		}))
		ef := &GeminiEmbeddingFunction{apiClient: &Client{
			Client:       client,
			DefaultModel: DefaultEmbeddingModel,
			APIKey:       embeddings.NewSecret("fake"),
		}}
		content := embeddings.Content{
			Parts: []embeddings.Part{embeddings.NewTextPart("test")},
		}
		emb, err := ef.EmbedContent(context.Background(), content)
		require.NoError(t, err)
		assert.Equal(t, []float32{0.1, 0.2, 0.3}, emb.ContentAsFloat32())
	})

	t.Run("empty result from API returns error", func(t *testing.T) {
		client := newMockGenaiClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"embeddings":[]}`))
		}))
		ef := &GeminiEmbeddingFunction{apiClient: &Client{
			Client:       client,
			DefaultModel: DefaultEmbeddingModel,
			APIKey:       embeddings.NewSecret("fake"),
		}}
		content := embeddings.Content{
			Parts: []embeddings.Part{embeddings.NewTextPart("test")},
		}
		_, err := ef.EmbedContent(context.Background(), content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no embeddings returned")
	})
}

func TestEmbedContentsMock(t *testing.T) {
	t.Run("happy path returns batch", func(t *testing.T) {
		client := newMockGenaiClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(geminiEmbedResponse([]float32{1, 2}, []float32{3, 4}))
		}))
		ef := &GeminiEmbeddingFunction{apiClient: &Client{
			Client:       client,
			DefaultModel: DefaultEmbeddingModel,
			APIKey:       embeddings.NewSecret("fake"),
		}}
		contents := []embeddings.Content{
			{Parts: []embeddings.Part{embeddings.NewTextPart("a")}},
			{Parts: []embeddings.Part{embeddings.NewTextPart("b")}},
		}
		results, err := ef.EmbedContents(context.Background(), contents)
		require.NoError(t, err)
		require.Len(t, results, 2)
	})

	t.Run("validation rejects unsupported modality", func(t *testing.T) {
		ef := &GeminiEmbeddingFunction{apiClient: &Client{
			DefaultModel: LegacyEmbeddingModel,
			APIKey:       embeddings.NewSecret("fake"),
		}}
		contents := []embeddings.Content{
			{Parts: []embeddings.Part{{Modality: embeddings.ModalityImage, Source: &embeddings.BinarySource{
				Kind: embeddings.SourceKindBytes, Bytes: []byte("x"), MIMEType: "image/png",
			}}}},
		}
		_, err := ef.EmbedContents(context.Background(), contents)
		require.Error(t, err)
	})
}

// TestResolveBytesKindsBase64Invalid verifies base64 decoding error handling.
func TestResolveBytesKindsBase64Invalid(t *testing.T) {
	source := &embeddings.BinarySource{
		Kind:   embeddings.SourceKindBase64,
		Base64: "!!!not-valid-base64!!!",
	}
	_, err := resolveBytes(context.Background(), source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode base64")
}

// TestResolveBytesKindsFileMissing verifies error when file does not exist.
func TestResolveBytesKindsFileMissing(t *testing.T) {
	source := &embeddings.BinarySource{
		Kind:     embeddings.SourceKindFile,
		FilePath: filepath.Join(t.TempDir(), "nonexistent.bin"),
	}
	_, err := resolveBytes(context.Background(), source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file source")
}
