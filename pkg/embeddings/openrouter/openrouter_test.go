//go:build ef

package openrouter

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

const (
	testOpenRouterErrorBodyLimit  = 512
	testOpenRouterTruncatedSuffix = "[truncated]"
)

func mockEmbeddingServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func defaultMockHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"object":"list","data":[{"object":"embedding","index":0,"embedding":[0.1,0.2,0.3]}],"model":"openai/text-embedding-3-small","usage":{"prompt_tokens":5,"total_tokens":5}}`))
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func TestRequestSerialization(t *testing.T) {
	var capturedBody string
	var capturedAuth string
	server := mockEmbeddingServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"object":"list","data":[{"object":"embedding","index":0,"embedding":[0.1,0.2,0.3]}],"model":"openai/text-embedding-3-small","usage":{"prompt_tokens":5,"total_tokens":5}}`))
	})
	defer server.Close()

	ef, err := NewOpenRouterEmbeddingFunction(
		WithAPIKey("test-key"),
		WithModel("openai/text-embedding-3-small"),
		WithEncodingFormat("float"),
		WithInputType("search_query"),
		WithProviderPreferences(&ProviderPreferences{
			Order: []string{"OpenAI"},
		}),
		WithBaseURL(server.URL),
		WithInsecure(),
	)
	require.NoError(t, err)

	_, err = ef.EmbedDocuments(context.Background(), []string{"hello world"})
	require.NoError(t, err)

	assert.Contains(t, capturedBody, `"model":"openai/text-embedding-3-small"`)
	assert.Contains(t, capturedBody, `"encoding_format":"float"`)
	assert.Contains(t, capturedBody, `"input_type":"search_query"`)
	assert.Contains(t, capturedBody, `"provider":{`)
	assert.Contains(t, capturedBody, `"order":["OpenAI"]`)
	assert.Equal(t, "Bearer test-key", capturedAuth)
}

func TestProviderPreferences(t *testing.T) {
	t.Run("typed fields only", func(t *testing.T) {
		prefs := ProviderPreferences{
			AllowFallbacks: boolPtr(true),
			Order:          []string{"A", "B"},
		}
		data, err := json.Marshal(prefs)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"allow_fallbacks":true`)
		assert.Contains(t, string(data), `"order":["A","B"]`)
	})

	t.Run("extras only", func(t *testing.T) {
		prefs := ProviderPreferences{
			Extras: map[string]any{"custom_field": "value"},
		}
		data, err := json.Marshal(prefs)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"custom_field":"value"`)
	})

	t.Run("merge without override", func(t *testing.T) {
		prefs := ProviderPreferences{
			AllowFallbacks: boolPtr(false),
			Extras: map[string]any{
				"allow_fallbacks": true,
				"new_field":       42,
			},
		}
		data, err := json.Marshal(prefs)
		require.NoError(t, err)
		s := string(data)
		assert.Contains(t, s, `"allow_fallbacks":false`)
		assert.Contains(t, s, `"new_field":42`)
	})

	t.Run("unmarshal preserves extras", func(t *testing.T) {
		var prefs ProviderPreferences
		err := json.Unmarshal([]byte(`{
			"allow_fallbacks":true,
			"custom_field":"value",
			"nested":{"limit":2}
		}`), &prefs)
		require.NoError(t, err)
		require.NotNil(t, prefs.AllowFallbacks)
		assert.True(t, *prefs.AllowFallbacks)
		require.Contains(t, prefs.Extras, "custom_field")
		assert.Equal(t, "value", prefs.Extras["custom_field"])
		nested, ok := prefs.Extras["nested"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(2), nested["limit"])
	})
}

func TestConfigRoundTrip(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")

	ef, err := NewOpenRouterEmbeddingFunction(
		WithEnvAPIKey(),
		WithModel("openai/text-embedding-3-small"),
		WithEncodingFormat("float"),
		WithInputType("search_document"),
		WithDimensions(256),
		WithProviderPreferences(&ProviderPreferences{
			Only: []string{"OpenAI"},
			ZDR:  boolPtr(true),
		}),
	)
	require.NoError(t, err)

	cfg := ef.GetConfig()
	require.Equal(t, "OPENROUTER_API_KEY", cfg["api_key_env_var"])
	require.Equal(t, "openai/text-embedding-3-small", cfg["model_name"])
	require.Equal(t, "float", cfg["encoding_format"])
	require.Equal(t, "search_document", cfg["input_type"])
	require.Equal(t, 256, cfg["dimensions"])
	provMap, ok := cfg["provider"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, provMap["zdr"])

	ef2, err := NewOpenRouterEmbeddingFunctionFromConfig(cfg)
	require.NoError(t, err)
	cfg2 := ef2.GetConfig()
	require.Equal(t, cfg["model_name"], cfg2["model_name"])
	require.Equal(t, cfg["encoding_format"], cfg2["encoding_format"])
	require.Equal(t, cfg["input_type"], cfg2["input_type"])
	require.Equal(t, cfg["dimensions"], cfg2["dimensions"])
}

func TestConfigRoundTripPreservesProviderExtras(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")

	ef, err := NewOpenRouterEmbeddingFunction(
		WithEnvAPIKey(),
		WithModel("openai/text-embedding-3-small"),
		WithProviderPreferences(&ProviderPreferences{
			Only:   []string{"OpenAI"},
			Extras: map[string]any{"latency_tier": "low"},
		}),
	)
	require.NoError(t, err)

	cfg := ef.GetConfig()
	provMap, ok := cfg["provider"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "low", provMap["latency_tier"])

	ef2, err := NewOpenRouterEmbeddingFunctionFromConfig(cfg)
	require.NoError(t, err)

	cfg2 := ef2.GetConfig()
	provMap2, ok := cfg2["provider"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "low", provMap2["latency_tier"])
}

func TestEmbedQuerySingleInput(t *testing.T) {
	server := mockEmbeddingServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		_, isString := req["input"].(string)
		assert.True(t, isString, "EmbedQuery input should be a single string")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"object":"list","data":[{"object":"embedding","index":0,"embedding":[0.1,0.2,0.3]}],"model":"test","usage":{"prompt_tokens":1,"total_tokens":1}}`))
	})
	defer server.Close()

	ef, err := NewOpenRouterEmbeddingFunction(
		WithAPIKey("test-key"),
		WithModel("test-model"),
		WithBaseURL(server.URL),
		WithInsecure(),
	)
	require.NoError(t, err)

	emb, err := ef.EmbedQuery(context.Background(), "single input")
	require.NoError(t, err)
	require.Equal(t, 3, emb.Len())
}

func TestEmbedDocumentsResponseValidation(t *testing.T) {
	t.Run("count mismatch", func(t *testing.T) {
		server := mockEmbeddingServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"object":"list","data":[{"object":"embedding","index":0,"embedding":[0.1,0.2,0.3]}],"model":"test","usage":{"prompt_tokens":1,"total_tokens":1}}`))
		})
		defer server.Close()

		ef, err := NewOpenRouterEmbeddingFunction(
			WithAPIKey("test-key"),
			WithModel("test-model"),
			WithBaseURL(server.URL),
			WithInsecure(),
		)
		require.NoError(t, err)

		_, err = ef.EmbedDocuments(context.Background(), []string{"doc1", "doc2"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "embedding count mismatch")
	})

	t.Run("empty embedding", func(t *testing.T) {
		server := mockEmbeddingServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"object":"list","data":[{"object":"embedding","index":0,"embedding":[]}],"model":"test","usage":{"prompt_tokens":1,"total_tokens":1}}`))
		})
		defer server.Close()

		ef, err := NewOpenRouterEmbeddingFunction(
			WithAPIKey("test-key"),
			WithModel("test-model"),
			WithBaseURL(server.URL),
			WithInsecure(),
		)
		require.NoError(t, err)

		_, err = ef.EmbedDocuments(context.Background(), []string{"doc1"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty embedding")
	})
}

func TestEmbedQueryResponseValidation(t *testing.T) {
	t.Run("empty response", func(t *testing.T) {
		server := mockEmbeddingServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"object":"list","data":[],"model":"test","usage":{"prompt_tokens":1,"total_tokens":1}}`))
		})
		defer server.Close()

		ef, err := NewOpenRouterEmbeddingFunction(
			WithAPIKey("test-key"),
			WithModel("test-model"),
			WithBaseURL(server.URL),
			WithInsecure(),
		)
		require.NoError(t, err)

		_, err = ef.EmbedQuery(context.Background(), "single input")
		require.Error(t, err)
		require.Contains(t, err.Error(), "no embedding returned")
	})

	t.Run("empty embedding", func(t *testing.T) {
		server := mockEmbeddingServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"object":"list","data":[{"object":"embedding","index":0,"embedding":[]}],"model":"test","usage":{"prompt_tokens":1,"total_tokens":1}}`))
		})
		defer server.Close()

		ef, err := NewOpenRouterEmbeddingFunction(
			WithAPIKey("test-key"),
			WithModel("test-model"),
			WithBaseURL(server.URL),
			WithInsecure(),
		)
		require.NoError(t, err)

		_, err = ef.EmbedQuery(context.Background(), "single input")
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty embedding")
	})
}

func TestAPIErrorResponseParsing(t *testing.T) {
	t.Run("structured json", func(t *testing.T) {
		server := mockEmbeddingServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
		})
		defer server.Close()

		ef, err := NewOpenRouterEmbeddingFunction(
			WithAPIKey("test-key"),
			WithModel("test-model"),
			WithBaseURL(server.URL),
			WithInsecure(),
		)
		require.NoError(t, err)

		_, err = ef.EmbedQuery(context.Background(), "single input")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid api key")
		require.NotContains(t, err.Error(), `{"error"`)
	})

	t.Run("structured json long message is sanitized", func(t *testing.T) {
		longMsg := strings.Repeat("☺", testOpenRouterErrorBodyLimit+10)
		server := mockEmbeddingServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"` + longMsg + `"}}`))
		})
		defer server.Close()

		ef, err := NewOpenRouterEmbeddingFunction(
			WithAPIKey("test-key"),
			WithModel("test-model"),
			WithBaseURL(server.URL),
			WithInsecure(),
		)
		require.NoError(t, err)

		_, err = ef.EmbedQuery(context.Background(), "single input")
		require.Error(t, err)
		msg := err.Error()
		require.Contains(t, msg, testOpenRouterTruncatedSuffix)
		require.NotContains(t, msg, `{"error"}`)

		prefix := strings.TrimSuffix(msg[strings.LastIndex(msg, ": ")+2:], testOpenRouterTruncatedSuffix)
		require.True(t, utf8.ValidString(prefix))
		assert.Len(t, []rune(prefix), testOpenRouterErrorBodyLimit)
	})

	t.Run("plain text fallback", func(t *testing.T) {
		server := mockEmbeddingServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`rate limit exceeded`))
		})
		defer server.Close()

		ef, err := NewOpenRouterEmbeddingFunction(
			WithAPIKey("test-key"),
			WithModel("test-model"),
			WithBaseURL(server.URL),
			WithInsecure(),
		)
		require.NoError(t, err)

		_, err = ef.EmbedQuery(context.Background(), "single input")
		require.Error(t, err)
		require.Contains(t, err.Error(), "429 Too Many Requests")
		require.Contains(t, err.Error(), "rate limit exceeded")
	})
}

func TestInputMarshalJSONZeroValue(t *testing.T) {
	_, err := json.Marshal(&Input{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
}

func TestEmptyModelRejected(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	_, err := NewOpenRouterEmbeddingFunction(WithEnvAPIKey())
	require.Error(t, err)
}

func TestWithProviderPreferencesRejectsNil(t *testing.T) {
	_, err := NewOpenRouterEmbeddingFunction(
		WithAPIKey("test-key"),
		WithModel("test-model"),
		WithProviderPreferences(nil),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "provider preferences cannot be nil")
}

func TestNewOpenRouterEmbeddingFunctionFromConfig_InvalidProvider(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")

	_, err := NewOpenRouterEmbeddingFunctionFromConfig(embeddings.EmbeddingFunctionConfig{
		"api_key_env_var": "OPENROUTER_API_KEY",
		"model_name":      "test-model",
		"provider": map[string]any{
			"unsupported": func() {},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "provider")
}

func TestNewOpenRouterEmbeddingFunctionFromConfig_Validation(t *testing.T) {
	t.Run("missing api key env var", func(t *testing.T) {
		_, err := NewOpenRouterEmbeddingFunctionFromConfig(embeddings.EmbeddingFunctionConfig{
			"model_name": "test-model",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "api_key_env_var")
	})

	t.Run("missing model name", func(t *testing.T) {
		t.Setenv("OPENROUTER_API_KEY", "test-key")

		_, err := NewOpenRouterEmbeddingFunctionFromConfig(embeddings.EmbeddingFunctionConfig{
			"api_key_env_var": "OPENROUTER_API_KEY",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "Model")
	})
}

func TestOptionValidation(t *testing.T) {
	testCases := []struct {
		name    string
		opts    []Option
		wantErr string
	}{
		{
			name: "empty base url",
			opts: []Option{
				WithAPIKey("test-key"),
				WithModel("test-model"),
				WithBaseURL(""),
			},
			wantErr: "base URL cannot be empty",
		},
		{
			name: "empty api key",
			opts: []Option{
				WithAPIKey(""),
				WithModel("test-model"),
			},
			wantErr: "API key cannot be empty",
		},
		{
			name: "invalid dimensions",
			opts: []Option{
				WithAPIKey("test-key"),
				WithModel("test-model"),
				WithDimensions(0),
			},
			wantErr: "dimensions must be greater than 0",
		},
		{
			name: "invalid encoding format",
			opts: []Option{
				WithAPIKey("test-key"),
				WithModel("test-model"),
				WithEncodingFormat("json"),
			},
			wantErr: "encoding_format must be 'float' or 'base64'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewOpenRouterEmbeddingFunction(tc.opts...)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestHTTPSRequiredWithoutInsecure(t *testing.T) {
	_, err := NewOpenRouterEmbeddingFunction(
		WithAPIKey("test-key"),
		WithModel("test-model"),
		WithBaseURL("http://openrouter.local"),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must use HTTPS")
}

func TestNameReturnsOpenRouter(t *testing.T) {
	server := mockEmbeddingServer(t, defaultMockHandler())
	defer server.Close()

	ef, err := NewOpenRouterEmbeddingFunction(
		WithAPIKey("test-key"),
		WithModel("test-model"),
		WithBaseURL(server.URL),
		WithInsecure(),
	)
	require.NoError(t, err)
	require.Equal(t, "openrouter", ef.Name())
}

func encodeFloat32Base64(vals []float32) string {
	buf := make([]byte, len(vals)*4)
	for i, v := range vals {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return base64.StdEncoding.EncodeToString(buf)
}

func TestBase64EmbeddingDecoding(t *testing.T) {
	expected := []float32{0.1, 0.2, 0.3}
	b64 := encodeFloat32Base64(expected)

	server := mockEmbeddingServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf(
			`{"object":"list","data":[{"object":"embedding","index":0,"embedding":"%s"}],"model":"test","usage":{"prompt_tokens":1,"total_tokens":1}}`,
			b64,
		)))
	})
	defer server.Close()

	ef, err := NewOpenRouterEmbeddingFunction(
		WithAPIKey("test-key"),
		WithModel("test-model"),
		WithEncodingFormat("base64"),
		WithBaseURL(server.URL),
		WithInsecure(),
	)
	require.NoError(t, err)

	emb, err := ef.EmbedQuery(context.Background(), "test")
	require.NoError(t, err)
	require.Equal(t, 3, emb.Len())
	floats := emb.ContentAsFloat32()
	for i, exp := range expected {
		require.InDelta(t, exp, floats[i], 1e-6)
	}
}

func TestBase64EmbeddingInvalidPayload(t *testing.T) {
	t.Run("invalid base64", func(t *testing.T) {
		server := mockEmbeddingServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"object":"list","data":[{"object":"embedding","index":0,"embedding":"not-valid-base64!!!"}],"model":"test","usage":{"prompt_tokens":1,"total_tokens":1}}`))
		})
		defer server.Close()

		ef, err := NewOpenRouterEmbeddingFunction(
			WithAPIKey("test-key"),
			WithModel("test-model"),
			WithBaseURL(server.URL),
			WithInsecure(),
		)
		require.NoError(t, err)

		_, err = ef.EmbedQuery(context.Background(), "test")
		require.Error(t, err)
	})

	t.Run("wrong byte length", func(t *testing.T) {
		oddBytes := base64.StdEncoding.EncodeToString([]byte{0x01, 0x02, 0x03})
		server := mockEmbeddingServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf(
				`{"object":"list","data":[{"object":"embedding","index":0,"embedding":"%s"}],"model":"test","usage":{"prompt_tokens":1,"total_tokens":1}}`,
				oddBytes,
			)))
		})
		defer server.Close()

		ef, err := NewOpenRouterEmbeddingFunction(
			WithAPIKey("test-key"),
			WithModel("test-model"),
			WithBaseURL(server.URL),
			WithInsecure(),
		)
		require.NoError(t, err)

		_, err = ef.EmbedQuery(context.Background(), "test")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not a multiple of 4")
	})
}

func TestParseAPIErrorTruncatesLargeBody(t *testing.T) {
	largeBody := []byte(strings.Repeat("x", testOpenRouterErrorBodyLimit+128))
	result := parseAPIError(largeBody)
	require.Equal(t, strings.Repeat("x", testOpenRouterErrorBodyLimit)+testOpenRouterTruncatedSuffix, result)
}

func TestRegistryRegistration(t *testing.T) {
	require.True(t, embeddings.HasDense("openrouter"))
}
