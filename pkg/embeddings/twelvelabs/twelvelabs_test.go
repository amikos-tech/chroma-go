//go:build ef

package twelvelabs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func newTestEF(serverURL string) *TwelveLabsEmbeddingFunction {
	return &TwelveLabsEmbeddingFunction{
		apiClient: &TwelveLabsClient{
			BaseAPI:              serverURL,
			APIKey:               embeddings.NewSecret("test-key"),
			DefaultModel:         defaultModel,
			Client:               http.DefaultClient,
			AudioEmbeddingOption: defaultAudioEmbeddingOption,
		},
	}
}

func newMockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func embedV2Response(embedding []float64) string {
	data := EmbedV2Response{Data: []EmbedV2DataItem{{Embedding: embedding}}}
	b, _ := json.Marshal(data)
	return string(b)
}

func make512DimVector() []float64 {
	v := make([]float64, 512)
	for i := range v {
		v[i] = float64(i) * 0.001
	}
	return v
}

func TestTwelveLabsEmbedDocuments(t *testing.T) {
	vec := make512DimVector()
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req EmbedV2Request
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "text", req.InputType)
		assert.Equal(t, "marengo3.0", req.ModelName)
		assert.NotNil(t, req.Text)
		assert.Equal(t, "hello world", req.Text.InputText)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, embedV2Response(vec))
	})

	ef := newTestEF(srv.URL)
	result, err := ef.EmbedDocuments(context.Background(), []string{"hello world"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, 512, result[0].Len())
}

func TestTwelveLabsEmbedDocumentsEmptyInput(t *testing.T) {
	ef := newTestEF("http://localhost")
	result, err := ef.EmbedDocuments(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestTwelveLabsEmbedDocumentsResponseValidation(t *testing.T) {
	t.Run("empty response returns error", func(t *testing.T) {
		srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":[]}`)
		})

		ef := newTestEF(srv.URL)
		_, err := ef.EmbedDocuments(context.Background(), []string{"hello world"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no embedding returned")
	})

	t.Run("empty embedding vector returns error", func(t *testing.T) {
		srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, embedV2Response([]float64{}))
		})

		ef := newTestEF(srv.URL)
		_, err := ef.EmbedDocuments(context.Background(), []string{"hello world"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty embedding vector")
	})
}

func TestTwelveLabsEmbedQuery(t *testing.T) {
	vec := make512DimVector()
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, embedV2Response(vec))
	})

	ef := newTestEF(srv.URL)
	result, err := ef.EmbedQuery(context.Background(), "search query")
	require.NoError(t, err)
	assert.Equal(t, 512, result.Len())
}

func TestTwelveLabsAuthHeader(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, embedV2Response([]float64{1, 2, 3}))
	})

	ef := newTestEF(srv.URL)
	_, err := ef.EmbedQuery(context.Background(), "test")
	require.NoError(t, err)
}

func TestTwelveLabsName(t *testing.T) {
	ef := newTestEF("http://localhost")
	assert.Equal(t, "twelvelabs", ef.Name())
}

func TestNewTwelveLabsClientDefaultsUseDedicatedHTTPClient(t *testing.T) {
	client, err := NewTwelveLabsClient(WithAPIKey("test-key"))
	require.NoError(t, err)
	require.NotNil(t, client.Client)
	assert.NotSame(t, http.DefaultClient, client.Client)
}

func TestTwelveLabsGetConfig(t *testing.T) {
	ef := &TwelveLabsEmbeddingFunction{
		apiClient: &TwelveLabsClient{
			BaseAPI:              defaultBaseAPI,
			APIKey:               embeddings.NewSecret("test-key"),
			APIKeyEnvVar:         "MY_TL_KEY",
			DefaultModel:         "marengo3.0",
			AudioEmbeddingOption: "fused",
		},
	}
	cfg := ef.GetConfig()
	assert.Equal(t, "MY_TL_KEY", cfg["api_key_env_var"])
	assert.Equal(t, "marengo3.0", cfg["model_name"])
	assert.Equal(t, "fused", cfg["audio_embedding_option"])
	_, hasBaseURL := cfg["base_url"]
	assert.False(t, hasBaseURL, "default base URL should not be in config")
}

func TestTwelveLabsConfigRoundTrip(t *testing.T) {
	original := &TwelveLabsEmbeddingFunction{
		apiClient: &TwelveLabsClient{
			BaseAPI:              defaultBaseAPI,
			APIKey:               embeddings.NewSecret("test-key"),
			APIKeyEnvVar:         APIKeyEnvVar,
			DefaultModel:         "marengo3.0",
			AudioEmbeddingOption: "fused",
			Client:               http.DefaultClient,
		},
	}
	cfg := original.GetConfig()

	t.Setenv(APIKeyEnvVar, "round-trip-key")
	restored, err := NewTwelveLabsEmbeddingFunctionFromConfig(cfg)
	require.NoError(t, err)
	assert.Equal(t, original.apiClient.DefaultModel, restored.apiClient.DefaultModel)
	assert.Equal(t, original.apiClient.AudioEmbeddingOption, restored.apiClient.AudioEmbeddingOption)
}

func TestTwelveLabsRegistration(t *testing.T) {
	t.Setenv(APIKeyEnvVar, "reg-test-key")
	cfg := embeddings.EmbeddingFunctionConfig{
		"api_key_env_var":        APIKeyEnvVar,
		"model_name":             "marengo3.0",
		"audio_embedding_option": "audio",
	}
	dense, err := embeddings.BuildDense("twelvelabs", cfg)
	require.NoError(t, err)
	assert.NotNil(t, dense)

	content, err := embeddings.BuildContent("twelvelabs", cfg)
	require.NoError(t, err)
	assert.NotNil(t, content)
}

func TestTwelveLabsAPIError(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"message":"invalid request","code":"bad_request"}`)
	})

	ef := newTestEF(srv.URL)
	_, err := ef.EmbedQuery(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid request")
}

func TestTwelveLabsContextModel(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req EmbedV2Request
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "custom-model", req.ModelName)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, embedV2Response([]float64{1, 2, 3}))
	})

	ef := newTestEF(srv.URL)
	ctx := ContextWithModel(context.Background(), "custom-model")
	_, err := ef.EmbedQuery(ctx, "test")
	require.NoError(t, err)
}
