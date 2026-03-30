//go:build ef

package openrouter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
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
	server := mockEmbeddingServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
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

func TestEmptyModelRejected(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	_, err := NewOpenRouterEmbeddingFunction(WithEnvAPIKey())
	require.Error(t, err)
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

func TestRegistryRegistration(t *testing.T) {
	require.True(t, embeddings.HasDense("openrouter"))
}
