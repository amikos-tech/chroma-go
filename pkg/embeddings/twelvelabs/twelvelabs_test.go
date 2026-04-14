//go:build ef

package twelvelabs

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

const (
	testTwelveLabsErrorBodyLimit  = 512
	testTwelveLabsTruncatedSuffix = "[truncated]"
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

func TestTwelveLabsEmbedDocumentsRejectsEmptyText(t *testing.T) {
	ef := newTestEF("http://localhost")
	_, err := ef.EmbedDocuments(context.Background(), []string{"hello", ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "texts[1]")
	assert.Contains(t, err.Error(), "text cannot be empty")
}

func TestTwelveLabsEmbedQueryRejectsEmptyText(t *testing.T) {
	ef := newTestEF("http://localhost")
	_, err := ef.EmbedQuery(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "texts[0]")
	assert.Contains(t, err.Error(), "text cannot be empty")
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

func TestNewTwelveLabsClientValidation(t *testing.T) {
	t.Run("fails with empty API key option", func(t *testing.T) {
		_, err := NewTwelveLabsClient(WithAPIKey(""))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key cannot be empty")
	})

	t.Run("fails with empty model option", func(t *testing.T) {
		_, err := NewTwelveLabsClient(WithAPIKey("test-key"), WithModel(""))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "model cannot be empty")
	})

	t.Run("fails with nil HTTP client option", func(t *testing.T) {
		_, err := NewTwelveLabsClient(WithAPIKey("test-key"), WithHTTPClient(nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP client cannot be nil")
	})

	t.Run("fails with HTTP base URL without insecure override", func(t *testing.T) {
		_, err := NewTwelveLabsClient(WithAPIKey("test-key"), WithBaseURL("http://example.com"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "base URL must use HTTPS")
	})

	t.Run("fails with invalid audio embedding option", func(t *testing.T) {
		_, err := NewTwelveLabsClient(WithAPIKey("test-key"), WithAudioEmbeddingOption("invalid"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid audio embedding option")
	})
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
		fmt.Fprint(w, `{"message":"invalid request","code":"parameter_invalid"}`)
	})

	ef := newTestEF(srv.URL)
	_, err := ef.EmbedQuery(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid request")
	assert.Contains(t, err.Error(), "parameter_invalid")
}

func TestTwelveLabsAPIErrorSanitizesStructuredMessage(t *testing.T) {
	longMessage := strings.Repeat("structured-", 80)

	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"message":%q,"code":"bad_request"}`, longMessage)
	})

	ef := newTestEF(srv.URL)
	_, err := ef.EmbedQuery(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "[truncated]")
	assert.NotContains(t, err.Error(), longMessage)

	prefix := strings.TrimSuffix(err.Error()[strings.LastIndex(err.Error(), ": ")+2:], testTwelveLabsTruncatedSuffix)
	require.True(t, utf8.ValidString(prefix))
	assert.Len(t, []rune(prefix), testTwelveLabsErrorBodyLimit)
}

func TestTwelveLabsAPIErrorSanitizesStructuredCode(t *testing.T) {
	const tailMarker = "tl-code-tail-marker"
	longCode := strings.Repeat("c", testTwelveLabsErrorBodyLimit+64) + tailMarker

	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"message":"invalid request","code":%q}`, longCode)
	})

	ef := newTestEF(srv.URL)
	_, err := ef.EmbedQuery(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid request")
	assert.Contains(t, err.Error(), testTwelveLabsTruncatedSuffix)
	assert.NotContains(t, err.Error(), tailMarker)
}

func TestTwelveLabsAPIErrorSanitizesRawFallbackBody(t *testing.T) {
	longBody := strings.Repeat("raw-body-", 80)

	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, longBody)
	})

	ef := newTestEF(srv.URL)
	_, err := ef.EmbedQuery(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "[truncated]")
	assert.NotContains(t, err.Error(), longBody)
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

// --- Async polling test helpers ---

func newTestAsyncEF(serverURL string) *TwelveLabsEmbeddingFunction {
	ef := newTestEF(serverURL)
	ef.apiClient.asyncPollingEnabled = true
	ef.apiClient.asyncMaxWait = 5 * time.Second
	ef.apiClient.asyncPollInitial = 1 * time.Millisecond
	ef.apiClient.asyncPollMultiplier = 1.5
	ef.apiClient.asyncPollCap = 10 * time.Millisecond
	return ef
}

func audioContent(url string) embeddings.Content {
	return embeddings.Content{Parts: []embeddings.Part{{
		Modality: embeddings.ModalityAudio,
		Source:   &embeddings.BinarySource{Kind: embeddings.SourceKindURL, URL: url},
	}}}
}

func videoContent(url string) embeddings.Content {
	return embeddings.Content{Parts: []embeddings.Part{{
		Modality: embeddings.ModalityVideo,
		Source:   &embeddings.BinarySource{Kind: embeddings.SourceKindURL, URL: url},
	}}}
}

// taskCreateJSON and taskGetJSON produce fixtures with the _id alias (Pitfall 1).
func taskCreateJSON(id, status string) string {
	return fmt.Sprintf(`{"_id":%q,"status":%q}`, id, status)
}

func taskGetJSON(id, status string, data []float64) string {
	if data == nil {
		return fmt.Sprintf(`{"_id":%q,"status":%q}`, id, status)
	}
	b, _ := json.Marshal(data)
	return fmt.Sprintf(`{"_id":%q,"status":%q,"data":[{"embedding":%s}]}`, id, status, b)
}

// --- Async polling tests ---

func TestTwelveLabsAsyncTaskCreate(t *testing.T) {
	vec := make512DimVector()
	var attempts atomic.Int32
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/tasks"):
			var req AsyncEmbedV2Request
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, "audio", req.InputType)
			require.NotNil(t, req.Audio)
			assert.Equal(t, []string{"audio"}, req.Audio.EmbeddingOption, "async endpoint requires embedding_option as []string (F-02)")
			fmt.Fprint(w, taskCreateJSON("task_abc", "processing"))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/tasks/task_abc"):
			fmt.Fprint(w, taskGetJSON("task_abc", "ready", vec))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	ef := newTestAsyncEF(srv.URL)
	emb, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.NoError(t, err)
	require.NotNil(t, emb)
	assert.Equal(t, 512, emb.Len())
	assert.GreaterOrEqual(t, attempts.Load(), int32(2), "expected at least 1 POST + 1 GET")
}

func TestTwelveLabsAsyncPollToReady(t *testing.T) {
	vec := make512DimVector()
	var gets atomic.Int32
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, taskCreateJSON("task_ready", "processing"))
			return
		}
		n := gets.Add(1)
		if n < 3 {
			fmt.Fprint(w, taskGetJSON("task_ready", "processing", nil))
			return
		}
		fmt.Fprint(w, taskGetJSON("task_ready", "ready", vec))
	})

	ef := newTestAsyncEF(srv.URL)
	emb, err := ef.EmbedContent(context.Background(), videoContent("https://example.com/v.mp4"))
	require.NoError(t, err)
	require.NotNil(t, emb)
	assert.Equal(t, 512, emb.Len())
	assert.Equal(t, int32(3), gets.Load(), "expected 3 GETs before ready")
}

func TestTwelveLabsAsyncPollToFailed(t *testing.T) {
	var gets atomic.Int32
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, taskCreateJSON("task_fail", "processing"))
			return
		}
		n := gets.Add(1)
		if n < 2 {
			fmt.Fprint(w, taskGetJSON("task_fail", "processing", nil))
			return
		}
		fmt.Fprint(w, taskGetJSON("task_fail", "failed", nil))
	})

	ef := newTestAsyncEF(srv.URL)
	emb, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Nil(t, emb)
	assert.Contains(t, err.Error(), "task_fail")
	assert.Contains(t, err.Error(), "terminal status=failed")
	assert.False(t, stderrors.Is(err, context.Canceled))
	assert.False(t, stderrors.Is(err, context.DeadlineExceeded))
}

func TestTwelveLabsAsyncUnexpectedStatus(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, taskCreateJSON("task_weird", "processing"))
			return
		}
		fmt.Fprint(w, taskGetJSON("task_weird", "weird", nil))
	})

	ef := newTestAsyncEF(srv.URL)
	_, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status")
	assert.Contains(t, err.Error(), "weird")
	assert.Contains(t, err.Error(), "task_weird")
}
