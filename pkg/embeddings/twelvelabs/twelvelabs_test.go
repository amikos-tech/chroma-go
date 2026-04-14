//go:build ef

package twelvelabs

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
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

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

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

func TestValidateAsyncPollingBackoffConfig(t *testing.T) {
	t.Run("rejects multiplier below one", func(t *testing.T) {
		client, err := NewTwelveLabsClient(WithAPIKey("test-key"))
		require.NoError(t, err)
		client.asyncPollMultiplier = 0.5

		err = validate(client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "async poll multiplier")
	})

	t.Run("rejects cap below initial", func(t *testing.T) {
		client, err := NewTwelveLabsClient(WithAPIKey("test-key"))
		require.NoError(t, err)
		client.asyncPollInitial = 3 * time.Second
		client.asyncPollCap = 2 * time.Second

		err = validate(client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "async poll cap")
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

// TestTwelveLabsAsyncCreateReturnsFailed covers the rare case where the server
// returns status=failed on the POST /tasks response. The SDK must short-circuit
// rather than fire a wasteful GET on the same terminal state.
func TestTwelveLabsAsyncCreateReturnsFailed(t *testing.T) {
	var gets atomic.Int32
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, taskCreateJSON("task_create_failed", "failed"))
			return
		}
		gets.Add(1)
		fmt.Fprint(w, taskGetJSON("task_create_failed", "failed", nil))
	})

	ef := newTestAsyncEF(srv.URL)
	_, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task_create_failed")
	assert.Contains(t, err.Error(), "terminal status=failed at creation")
	assert.Equal(t, int32(0), gets.Load(), "failed create response must not trigger a poll GET")
}

// TestTwelveLabsAsyncCreateReturnsReadyEmptyData covers the malformed server
// case where status=ready arrives with no data. Since ready is terminal per F-01,
// we surface the malformation immediately instead of polling for the same state.
func TestTwelveLabsAsyncCreateReturnsReadyEmptyData(t *testing.T) {
	var gets atomic.Int32
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, taskCreateJSON("task_empty_ready", "ready"))
			return
		}
		gets.Add(1)
		fmt.Fprint(w, taskGetJSON("task_empty_ready", "ready", nil))
	})

	ef := newTestAsyncEF(srv.URL)
	_, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no embedding returned")
	assert.Equal(t, int32(0), gets.Load(), "terminal ready must not trigger a poll GET even when data is empty")
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

func TestTwelveLabsAsyncCtxCancel(t *testing.T) {
	var gets atomic.Int32
	cancelCh := make(chan struct{})
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, taskCreateJSON("task_cancel", "processing"))
			return
		}
		n := gets.Add(1)
		if n == 1 {
			close(cancelCh) // signal the test to cancel after the first poll response
		}
		fmt.Fprint(w, taskGetJSON("task_cancel", "processing", nil))
	})

	ef := newTestAsyncEF(srv.URL)
	ef.apiClient.asyncPollInitial = 50 * time.Millisecond // slow enough that cancel wins
	ef.apiClient.asyncPollCap = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-cancelCh
		cancel()
	}()
	_, err := ef.EmbedContent(ctx, audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.True(t, stderrors.Is(err, context.Canceled), "expected ctx.Canceled wrapping, got %v", err)
}

func TestTwelveLabsAsyncMaxWait(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, taskCreateJSON("task_maxwait", "processing"))
			return
		}
		fmt.Fprint(w, taskGetJSON("task_maxwait", "processing", nil))
	})

	ef := newTestAsyncEF(srv.URL)
	ef.apiClient.asyncMaxWait = 50 * time.Millisecond

	_, err := ef.EmbedContent(context.Background(), videoContent("https://example.com/v.mp4"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "async polling maxWait")
	assert.True(t, stderrors.Is(err, ErrAsyncMaxWaitExceeded), "maxWait must wrap ErrAsyncMaxWaitExceeded sentinel")
	assert.False(t, stderrors.Is(err, context.DeadlineExceeded), "maxWait must surface distinct from ctx.DeadlineExceeded (D-20)")
	assert.False(t, stderrors.Is(err, context.Canceled))
}

// TestTwelveLabsAsyncMaxWaitHardBound asserts the documented D-09 guarantee:
// total operation time (create + poll) stays within a single asyncMaxWait
// window, not create_time + asyncMaxWait. A slow task-create POST must eat
// into the polling budget, not reset it.
func TestTwelveLabsAsyncMaxWaitHardBound(t *testing.T) {
	const maxWait = 200 * time.Millisecond
	const createDelay = 120 * time.Millisecond // spends >half the budget
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			time.Sleep(createDelay)
			fmt.Fprint(w, taskCreateJSON("task_hard_bound", "processing"))
			return
		}
		fmt.Fprint(w, taskGetJSON("task_hard_bound", "processing", nil))
	})

	ef := newTestAsyncEF(srv.URL)
	ef.apiClient.asyncMaxWait = maxWait

	start := time.Now()
	_, err := ef.EmbedContent(context.Background(), videoContent("https://example.com/v.mp4"))
	elapsed := time.Since(start)

	require.Error(t, err)
	// Either the create call or the poll loop may trip the bound depending on
	// scheduling; both surface the distinct maxWait SDK error, not raw ctx errs.
	assert.Contains(t, err.Error(), "maxWait")
	assert.False(t, stderrors.Is(err, context.DeadlineExceeded))
	assert.False(t, stderrors.Is(err, context.Canceled))
	// Total elapsed must stay within ~maxWait. Allow 50% slack for CI jitter
	// (backoff sleeps, goroutine scheduling). Without the fix this would be
	// ~createDelay + maxWait ≈ 320ms, which is well over the threshold.
	assert.Less(t, elapsed, maxWait+maxWait/2,
		"total elapsed %s must stay within ~asyncMaxWait %s (D-09 hard bound)", elapsed, maxWait)
}

func TestTwelveLabsAsyncPollParentDeadlinePreservesTransportError(t *testing.T) {
	ef := newTestAsyncEF("https://example.test/embed-v2")
	ef.apiClient.Client = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPost {
			return newJSONResponse(http.StatusOK, taskCreateJSON("task_parent_deadline", "processing")), nil
		}
		<-req.Context().Done()
		return nil, fmt.Errorf("simulated retrieve transport failure: %w", req.Context().Err())
	})}
	ef.apiClient.asyncMaxWait = time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := ef.EmbedContent(ctx, audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "async polling deadline exceeded")
	assert.Contains(t, err.Error(), "failed to send task retrieve request")
	assert.True(t, stderrors.Is(err, context.DeadlineExceeded))
}

func TestTwelveLabsAsyncPollParentDeadlineDuringSleep(t *testing.T) {
	var gets atomic.Int32
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, taskCreateJSON("task_sleep_deadline", "processing"))
			return
		}
		gets.Add(1)
		fmt.Fprint(w, taskGetJSON("task_sleep_deadline", "processing", nil))
	})

	ef := newTestAsyncEF(srv.URL)
	ef.apiClient.asyncPollInitial = 200 * time.Millisecond
	ef.apiClient.asyncPollCap = 200 * time.Millisecond
	ef.apiClient.asyncMaxWait = time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := ef.EmbedContent(ctx, audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "async polling deadline exceeded")
	assert.True(t, stderrors.Is(err, context.DeadlineExceeded))
	assert.Equal(t, int32(1), gets.Load(), "deadline during sleep should happen after the first processing poll")
}

func TestTwelveLabsAsyncSkipsTextImage(t *testing.T) {
	vec := make512DimVector()
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/tasks") {
			t.Fatalf("text/image must not hit async endpoint; got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, embedV2Response(vec))
	})

	ef := newTestAsyncEF(srv.URL) // asyncPollingEnabled=true — but text/image skip async per D-07

	// text
	textContent := embeddings.Content{Parts: []embeddings.Part{{Modality: embeddings.ModalityText, Text: "hello"}}}
	emb, err := ef.EmbedContent(context.Background(), textContent)
	require.NoError(t, err)
	require.NotNil(t, emb)

	// image (URL source)
	imageContent := embeddings.Content{Parts: []embeddings.Part{{
		Modality: embeddings.ModalityImage,
		Source:   &embeddings.BinarySource{Kind: embeddings.SourceKindURL, URL: "https://example.com/i.png"},
	}}}
	emb2, err := ef.EmbedContent(context.Background(), imageContent)
	require.NoError(t, err)
	require.NotNil(t, emb2)
}

func TestTwelveLabsAsyncConfigRoundTrip(t *testing.T) {
	t.Setenv(APIKeyEnvVar, "round-trip-key")
	ef, err := NewTwelveLabsEmbeddingFunction(WithEnvAPIKey(), WithAsyncPolling(7*time.Minute))
	require.NoError(t, err)

	cfg := ef.GetConfig()
	assert.Equal(t, true, cfg["async_polling"])
	assert.Equal(t, int64(420000), cfg["async_max_wait_ms"], "7 min = 420000 ms as int64")

	rebuilt, err := NewTwelveLabsEmbeddingFunctionFromConfig(cfg)
	require.NoError(t, err)
	assert.True(t, rebuilt.apiClient.asyncPollingEnabled)
	assert.Equal(t, 7*time.Minute, rebuilt.apiClient.asyncMaxWait)
	// APIKeyEnvVar must survive the round-trip so env-var-sourced EFs
	// rebuilt from a registry still resolve the same way.
	assert.Equal(t, APIKeyEnvVar, rebuilt.apiClient.APIKeyEnvVar)

	// JSON round-trip: registries persist configs as JSON, which decodes
	// integers as float64. ConfigInt must coerce back to int64 so the
	// rebuilt EF honors the original asyncMaxWait.
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	var decoded embeddings.EmbeddingFunctionConfig
	require.NoError(t, json.Unmarshal(raw, &decoded))
	_, isFloat := decoded["async_max_wait_ms"].(float64)
	assert.True(t, isFloat, "sanity check: JSON decoding must produce float64, not int64")
	rebuiltFromJSON, err := NewTwelveLabsEmbeddingFunctionFromConfig(decoded)
	require.NoError(t, err)
	assert.True(t, rebuiltFromJSON.apiClient.asyncPollingEnabled)
	assert.Equal(t, 7*time.Minute, rebuiltFromJSON.apiClient.asyncMaxWait)
}

// TestTwelveLabsAsyncPollingRejectsSubFloorMaxWait asserts that sub-second
// values that can't complete a single poll cycle are rejected at option
// application time rather than deterministically timing out on first poll.
func TestTwelveLabsAsyncPollingRejectsSubFloorMaxWait(t *testing.T) {
	t.Setenv(APIKeyEnvVar, "floor-key")
	_, err := NewTwelveLabsEmbeddingFunction(WithEnvAPIKey(), WithAsyncPolling(100*time.Millisecond))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "below minimum")

	// 0 means "use default" and must still be accepted.
	_, err = NewTwelveLabsEmbeddingFunction(WithEnvAPIKey(), WithAsyncPolling(0))
	require.NoError(t, err)

	// Exactly the floor value is accepted.
	_, err = NewTwelveLabsEmbeddingFunction(WithEnvAPIKey(), WithAsyncPolling(defaultAsyncPollInitial))
	require.NoError(t, err)
}

// TestTwelveLabsAsyncFailedReasonFallbackOnEmptyBody proves the failure
// message uses the generic fallback when the server returns a JSON body with
// no diagnostic fields (only housekeeping), rather than dumping the raw JSON.
func TestTwelveLabsAsyncFailedReasonFallbackOnEmptyBody(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, taskCreateJSON("task_nodetail", "processing"))
			return
		}
		fmt.Fprint(w, `{"_id":"task_nodetail","status":"failed"}`)
	})

	ef := newTestAsyncEF(srv.URL)
	_, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task_nodetail")
	assert.Contains(t, err.Error(), "terminal status=failed")
	assert.Contains(t, err.Error(), "(no failure detail provided)")
	assert.NotContains(t, err.Error(), "_id", "housekeeping fields must not leak into the error message")
}

func TestTwelveLabsAsyncConfigOmitWhenDisabled(t *testing.T) {
	t.Setenv(APIKeyEnvVar, "some-key")
	ef, err := NewTwelveLabsEmbeddingFunction(WithEnvAPIKey())
	require.NoError(t, err)

	cfg := ef.GetConfig()
	_, hasPolling := cfg["async_polling"]
	_, hasWait := cfg["async_max_wait_ms"]
	assert.False(t, hasPolling, "async_polling must be omitted when WithAsyncPolling is absent (D-22)")
	assert.False(t, hasWait, "async_max_wait_ms must be omitted when WithAsyncPolling is absent (D-22)")
}

// TestTwelveLabsAsyncFailedReasonSanitized proves the authentic server
// failure reason (from the raw response body, preserved in
// TaskResponse.FailureDetail by Plan 01) reaches the error — not a
// re-marshaled subset of known fields (D-17 review fix).
func TestTwelveLabsAsyncFailedReasonSanitized(t *testing.T) {
	longReason := strings.Repeat("detailed server-side failure reason — ", 40) // ~1.5KB
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, taskCreateJSON("task_failreason", "processing"))
			return
		}
		// Respond with an extra server-only reason field NOT in TaskResponse.
		// If the plan re-marshaled the parsed struct the reason would be lost.
		fmt.Fprintf(w, `{"_id":"task_failreason","status":"failed","reason":%q,"error":{"code":"E_BAD_MEDIA","detail":%q}}`, longReason, longReason)
	})

	ef := newTestAsyncEF(srv.URL)
	_, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task_failreason")
	assert.Contains(t, err.Error(), "terminal status=failed")
	// Authentic server reason substring must survive sanitization.
	assert.Contains(t, err.Error(), "detailed server-side failure reason", "error must carry the server-provided reason from the raw body, not just the parsed TaskResponse fields")
	// Sanitization must still cap the error size.
	assert.Less(t, len(err.Error()), 4096, "sanitized error body must be truncated to a safe display length")
}

func TestTwelveLabsAsyncFailedReasonPrefersStructuredMessageOverLargeBody(t *testing.T) {
	var dataBuilder strings.Builder
	for i := 0; i < 400; i++ {
		if i > 0 {
			dataBuilder.WriteByte(',')
		}
		dataBuilder.WriteString(fmt.Sprintf("%d", i))
	}

	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, taskCreateJSON("task_failmsg", "processing"))
			return
		}
		fmt.Fprintf(w, `{"_id":"task_failmsg","status":"failed","data":[{"embedding":[%s]}],"message":"upstream media fetch failed"}`, dataBuilder.String())
	})

	ef := newTestAsyncEF(srv.URL)
	_, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upstream media fetch failed")
}

// TestTwelveLabsAsyncBlockedHTTPMaxWait proves the per-HTTP-call deadline
// added in Plan 02 actually unblocks an in-flight GET /tasks/{id} when
// maxWait fires. Without the per-call deadline, a blocked HTTP call would
// hang indefinitely regardless of maxWait (Plan 02 review fix).
func TestTwelveLabsAsyncBlockedHTTPMaxWait(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, taskCreateJSON("task_block", "processing"))
			return
		}
		// Block until the request's context is canceled by the per-call
		// deadline. If the plan fails to bound the HTTP call, this select
		// would block until the server closed and the test would hang until
		// `go test` timeout — a loud failure.
		<-r.Context().Done()
	})

	ef := newTestAsyncEF(srv.URL)
	ef.apiClient.asyncMaxWait = 100 * time.Millisecond

	start := time.Now()
	_, err := ef.EmbedContent(context.Background(), videoContent("https://example.com/v.mp4"))
	elapsed := time.Since(start)

	require.Error(t, err)
	// The distinct SDK-timeout message must fire, not ctx.DeadlineExceeded —
	// parent ctx has no deadline, so maxWait is the only bound.
	assert.Contains(t, err.Error(), "async polling maxWait")
	assert.True(t, stderrors.Is(err, ErrAsyncMaxWaitExceeded), "SDK maxWait must wrap ErrAsyncMaxWaitExceeded sentinel")
	assert.False(t, stderrors.Is(err, context.DeadlineExceeded), "SDK maxWait must not collapse into context.DeadlineExceeded (D-20)")
	// Sanity-check the upper bound so a regression where maxWait is not
	// enforced on in-flight HTTP work fails loudly.
	assert.Less(t, elapsed, 2*time.Second, "maxWait must interrupt the blocked HTTP call; took %s", elapsed)
}

func TestTwelveLabsAsyncTaskCreateClientTimeoutReturnsError(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		select {
		case <-r.Context().Done():
		case <-time.After(200 * time.Millisecond):
		}
	})

	ef := newTestAsyncEF(srv.URL)
	ef.apiClient.Client = &http.Client{Timeout: 50 * time.Millisecond}
	ef.apiClient.asyncMaxWait = time.Second

	emb, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Nil(t, emb)
	assert.Contains(t, err.Error(), "async task create request timed out")
	assert.True(t, stderrors.Is(err, context.DeadlineExceeded), "client timeout should preserve the underlying deadline error")
}

func TestTwelveLabsAsyncTaskCreateParentDeadlinePreservesTransportError(t *testing.T) {
	ef := newTestAsyncEF("https://example.test/embed-v2")
	ef.apiClient.Client = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		<-req.Context().Done()
		return nil, fmt.Errorf("simulated create transport failure: %w", req.Context().Err())
	})}
	ef.apiClient.asyncMaxWait = time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := ef.EmbedContent(ctx, audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "async task create deadline exceeded")
	assert.Contains(t, err.Error(), "failed to send task request")
	assert.True(t, stderrors.Is(err, context.DeadlineExceeded))
}

func TestTwelveLabsAsyncPollClientTimeoutReturnsErrorWithoutPanic(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, taskCreateJSON("task_client_timeout", "processing"))
			return
		}
		select {
		case <-r.Context().Done():
		case <-time.After(200 * time.Millisecond):
		}
	})

	ef := newTestAsyncEF(srv.URL)
	ef.apiClient.Client = &http.Client{Timeout: 50 * time.Millisecond}
	ef.apiClient.asyncMaxWait = time.Second

	var emb embeddings.Embedding
	var err error
	assert.NotPanics(t, func() {
		emb, err = ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	})
	require.Error(t, err)
	assert.Nil(t, emb)
	assert.Contains(t, err.Error(), "async polling request timed out")
	assert.True(t, stderrors.Is(err, context.DeadlineExceeded), "client timeout should preserve the underlying deadline error")
}

// TestTwelveLabsAsyncTaskCreateError proves the doTaskPost non-2xx error
// path mirrors the structured-error-then-raw-fallback logic in doPost
// (twelvelabs.go). Without this test, a regression that breaks the create
// error branch (forgotten sanitize, wrong wrap) would go undetected until
// a real Twelve Labs 4xx arrived in production.
func TestTwelveLabsAsyncTaskCreateError(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"message":"invalid media source","code":"E_BAD_SRC"}`)
	})
	ef := newTestAsyncEF(srv.URL)
	_, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task create error")
	assert.Contains(t, err.Error(), "invalid media source")
	assert.Contains(t, err.Error(), "E_BAD_SRC")
}

func TestTwelveLabsAsyncTaskCreateErrorSanitizesStructuredMessage(t *testing.T) {
	longMessage := strings.Repeat("create-err-", 80)
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"message":%q,"code":"bad_request"}`, longMessage)
	})
	ef := newTestAsyncEF(srv.URL)
	_, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task create error")
	assert.Contains(t, err.Error(), testTwelveLabsTruncatedSuffix)
	assert.NotContains(t, err.Error(), longMessage)
}

func TestTwelveLabsAsyncTaskCreateErrorRawFallback(t *testing.T) {
	longBody := strings.Repeat("raw-create-", 80)
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, longBody)
	})
	ef := newTestAsyncEF(srv.URL)
	_, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status")
	assert.Contains(t, err.Error(), testTwelveLabsTruncatedSuffix)
	assert.NotContains(t, err.Error(), longBody)
}

func TestTwelveLabsAsyncTaskRetrieveErrorIncludesStructuredCode(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, taskCreateJSON("task_retrieve_err", "processing"))
			return
		}
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, `{"message":"task fetch failed","code":"E_TASK_FETCH"}`)
	})
	ef := newTestAsyncEF(srv.URL)
	_, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task retrieve error")
	assert.Contains(t, err.Error(), "task fetch failed")
	assert.Contains(t, err.Error(), "E_TASK_FETCH")
}

func TestTwelveLabsAsyncTaskCreateEmptyIDRejected(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"processing"}`)
	})

	ef := newTestAsyncEF(srv.URL)
	_, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty _id")
}

func TestTwelveLabsAsyncTaskCreateReadyReturnsWithoutPolling(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("ready-on-create must not poll; got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"_id":"task_ready_create","status":"ready","data":[{"embedding":[1,2,3]}]}`)
	})

	ef := newTestAsyncEF(srv.URL)
	emb, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.NoError(t, err)
	require.NotNil(t, emb)
	assert.Equal(t, 3, emb.Len())
}

// TestTwelveLabsAsyncFusedRejected proves the async path rejects
// WithAudioEmbeddingOption("fused") deterministically (RESEARCH F-02 / A5
// review fix). The rejection must happen before any POST /tasks call.
func TestTwelveLabsAsyncFusedRejected(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("no HTTP call expected for fused+async; got %s %s", r.Method, r.URL.Path)
	})
	_ = srv

	ef := newTestAsyncEF(srv.URL)
	// Apply the fused audio option by setting the field directly (the
	// public option setter would also work; direct assignment keeps the
	// test independent of option ordering).
	ef.apiClient.AudioEmbeddingOption = "fused"

	_, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fused")
	assert.Contains(t, err.Error(), "async")
}

func TestTwelveLabsAsyncRejectsUnknownAudioOption(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("no HTTP call expected for unsupported async audio option; got %s %s", r.Method, r.URL.Path)
	})

	ef := newTestAsyncEF(srv.URL)
	ef.apiClient.AudioEmbeddingOption = "sync-only-future-opt"

	_, err := ef.EmbedContent(context.Background(), audioContent("https://example.com/a.mp3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audio embedding option")
	assert.Contains(t, err.Error(), "async")
}

func TestNextBackoff(t *testing.T) {
	t.Run("multiplies until cap", func(t *testing.T) {
		assert.Equal(t, 1500*time.Millisecond, nextBackoff(time.Second, 1.5, 10*time.Second))
	})

	t.Run("clamps at cap", func(t *testing.T) {
		assert.Equal(t, 2*time.Second, nextBackoff(1500*time.Millisecond, 2, 2*time.Second))
	})
}
