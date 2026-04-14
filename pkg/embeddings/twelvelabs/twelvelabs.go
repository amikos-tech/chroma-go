package twelvelabs

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	chttp "github.com/amikos-tech/chroma-go/pkg/commons/http"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

const (
	defaultBaseAPI              = "https://api.twelvelabs.io/v1.3/embed-v2"
	defaultModel                = "marengo3.0"
	defaultAudioEmbeddingOption = "audio"
	APIKeyEnvVar                = "TWELVE_LABS_API_KEY"
)

type contextKey struct{ name string }

var modelContextKey = contextKey{"model"}

// ContextWithModel sets a per-request model override.
func ContextWithModel(ctx context.Context, model string) context.Context {
	return context.WithValue(ctx, modelContextKey, model)
}

// TwelveLabsClient holds the transport-level configuration.
type TwelveLabsClient struct {
	BaseAPI              string
	APIKey               embeddings.Secret `json:"-" validate:"required"`
	APIKeyEnvVar         string
	DefaultModel         embeddings.EmbeddingModel
	Client               *http.Client
	Insecure             bool
	AudioEmbeddingOption string

	// Async polling state — wired up by WithAsyncPolling (Plan 26-03)
	// and consumed by the polling loop (Plan 26-02).
	asyncPollingEnabled bool          //nolint:unused // consumed by Plans 26-02/03
	asyncMaxWait        time.Duration //nolint:unused // consumed by Plans 26-02/03
	asyncPollInitial    time.Duration
	asyncPollMultiplier float64
	asyncPollCap        time.Duration
}

func applyDefaults(c *TwelveLabsClient) {
	if c.Client == nil {
		c.Client = &http.Client{}
	}
	if c.BaseAPI == "" {
		c.BaseAPI = defaultBaseAPI
	}
	if c.DefaultModel == "" {
		c.DefaultModel = defaultModel
	}
	if c.AudioEmbeddingOption == "" {
		c.AudioEmbeddingOption = defaultAudioEmbeddingOption
	}
	if c.asyncPollInitial == 0 {
		c.asyncPollInitial = 2 * time.Second
	}
	if c.asyncPollMultiplier == 0 {
		c.asyncPollMultiplier = 1.5
	}
	if c.asyncPollCap == 0 {
		c.asyncPollCap = 60 * time.Second
	}
}

func validate(c *TwelveLabsClient) error {
	if err := embeddings.NewValidator().Struct(c); err != nil {
		return err
	}
	parsed, err := url.Parse(c.BaseAPI)
	if err != nil {
		return errors.Wrap(err, "invalid base URL")
	}
	if !c.Insecure && !strings.EqualFold(parsed.Scheme, "https") {
		return errors.New("base URL must use HTTPS scheme for secure API key transmission; use WithInsecure() to override")
	}
	return nil
}

// NewTwelveLabsClient creates a configured API client.
func NewTwelveLabsClient(opts ...Option) (*TwelveLabsClient, error) {
	client := &TwelveLabsClient{}
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, errors.Wrap(err, "failed to apply Twelve Labs option")
		}
	}
	applyDefaults(client)
	if err := validate(client); err != nil {
		return nil, errors.Wrap(err, "failed to validate Twelve Labs client options")
	}
	return client, nil
}

// --- Request / Response types ---

// EmbedV2Request is the JSON request body for the Twelve Labs embed endpoint.
type EmbedV2Request struct {
	InputType string      `json:"input_type"`
	ModelName string      `json:"model_name"`
	Text      *TextInput  `json:"text,omitempty"`
	Image     *ImageInput `json:"image,omitempty"`
	Audio     *AudioInput `json:"audio,omitempty"`
	Video     *VideoInput `json:"video,omitempty"`
}

type TextInput struct {
	InputText string `json:"input_text"`
}

type MediaSource struct {
	URL          string `json:"url,omitempty"`
	Base64String string `json:"base64_string,omitempty"`
}

type ImageInput struct {
	MediaSource MediaSource `json:"media_source"`
}

type AudioInput struct {
	MediaSource     MediaSource `json:"media_source"`
	EmbeddingOption string      `json:"embedding_option,omitempty"`
}

type VideoInput struct {
	MediaSource MediaSource `json:"media_source"`
}

// AsyncEmbedV2Request is the JSON body for POST /v1.3/embed-v2/tasks.
// The async endpoint uses a distinct shape from the sync endpoint
// (embedding_option is a list, not a single string). See RESEARCH F-02.
type AsyncEmbedV2Request struct {
	InputType string           `json:"input_type"`
	ModelName string           `json:"model_name"`
	Audio     *AsyncAudioInput `json:"audio,omitempty"`
	Video     *AsyncVideoInput `json:"video,omitempty"`
}

type AsyncAudioInput struct {
	MediaSource     MediaSource `json:"media_source"`
	EmbeddingOption []string    `json:"embedding_option,omitempty"`
}

type AsyncVideoInput struct {
	MediaSource MediaSource `json:"media_source"`
}

// TaskCreateResponse is returned from POST /v1.3/embed-v2/tasks.
// NOTE: task ID uses `_id` alias (Mongo-style) — RESEARCH Pitfall 1.
type TaskCreateResponse struct {
	ID     string            `json:"_id"`
	Status string            `json:"status"`
	Data   []EmbedV2DataItem `json:"data,omitempty"`
}

// TaskResponse is returned from GET /v1.3/embed-v2/tasks/{id}.
// Serves BOTH polling and retrieval (RESEARCH F-01 — only two
// endpoints exist; there is no separate /status sub-path).
//
// FailureDetail holds the raw HTTP response body so that on status=failed
// Plan 02 can sanitize the actual server-provided failure reason rather
// than re-marshaling this struct's subset of fields (D-17 compliance).
// The `json:"-"` tag excludes it from unmarshaling; doTaskGet populates
// it directly from the body bytes.
type TaskResponse struct {
	ID            string            `json:"_id"`
	Status        string            `json:"status"` // "processing" | "ready" | "failed"
	Data          []EmbedV2DataItem `json:"data,omitempty"`
	FailureDetail json.RawMessage   `json:"-"`
}

// EmbedV2Response is the response body from the embed-v2 endpoint.
type EmbedV2Response struct {
	Data []EmbedV2DataItem `json:"data"`
}

type EmbedV2DataItem struct {
	Embedding []float64 `json:"embedding"`
}

type EmbedV2ErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

// --- Embedding function ---

var _ embeddings.EmbeddingFunction = (*TwelveLabsEmbeddingFunction)(nil)
var _ embeddings.ContentEmbeddingFunction = (*TwelveLabsEmbeddingFunction)(nil)
var _ embeddings.CapabilityAware = (*TwelveLabsEmbeddingFunction)(nil)

// TwelveLabsEmbeddingFunction provides embeddings via the Twelve Labs Embed API v2.
type TwelveLabsEmbeddingFunction struct {
	apiClient *TwelveLabsClient
}

// NewTwelveLabsEmbeddingFunction creates a new Twelve Labs embedding function.
func NewTwelveLabsEmbeddingFunction(opts ...Option) (*TwelveLabsEmbeddingFunction, error) {
	client, err := NewTwelveLabsClient(opts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize Twelve Labs client")
	}
	return &TwelveLabsEmbeddingFunction{apiClient: client}, nil
}

// doPost sends a JSON POST to the embed-v2 endpoint and returns the parsed response.
func (e *TwelveLabsEmbeddingFunction) doPost(ctx context.Context, req EmbedV2Request) (*EmbedV2Response, error) {
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request JSON")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, e.apiClient.BaseAPI, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}
	httpReq.Header.Set("x-api-key", e.apiClient.APIKey.Value())
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", chttp.ChromaGoClientUserAgent)

	resp, err := e.apiClient.Client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send request to %s", e.apiClient.BaseAPI)
	}
	defer resp.Body.Close()

	respData, err := chttp.ReadLimitedBody(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}
	if resp.StatusCode != http.StatusOK {
		var apiErr EmbedV2ErrorResponse
		if jsonErr := json.Unmarshal(respData, &apiErr); jsonErr == nil && apiErr.Message != "" {
			if apiErr.Code != "" {
				return nil, errors.Errorf("Twelve Labs API error [%s] (%s): %s", resp.Status, chttp.SanitizeErrorBody([]byte(apiErr.Code)), chttp.SanitizeErrorBody([]byte(apiErr.Message)))
			}
			return nil, errors.Errorf("Twelve Labs API error [%s]: %s", resp.Status, chttp.SanitizeErrorBody([]byte(apiErr.Message)))
		}
		return nil, errors.Errorf("unexpected status [%s] from %s: %s", resp.Status, e.apiClient.BaseAPI, chttp.SanitizeErrorBody(respData))
	}

	var embedResp EmbedV2Response
	if err := json.Unmarshal(respData, &embedResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}
	return &embedResp, nil
}

// doTaskPost creates an async embedding task via POST {BaseAPI}/tasks.
// Used when WithAsyncPolling is enabled and the content modality is
// audio or video. See CONTEXT.md D-01, D-07.
//
//nolint:unused // consumed by Plans 26-02/03
func (e *TwelveLabsEmbeddingFunction) doTaskPost(ctx context.Context, req AsyncEmbedV2Request) (*TaskCreateResponse, error) {
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal async task request")
	}
	endpoint := strings.TrimRight(e.apiClient.BaseAPI, "/") + "/tasks"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}
	httpReq.Header.Set("x-api-key", e.apiClient.APIKey.Value())
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", chttp.ChromaGoClientUserAgent)

	resp, err := e.apiClient.Client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send task request to %s", endpoint)
	}
	defer resp.Body.Close()

	respData, err := chttp.ReadLimitedBody(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr EmbedV2ErrorResponse
		if jsonErr := json.Unmarshal(respData, &apiErr); jsonErr == nil && apiErr.Message != "" {
			return nil, errors.Errorf("Twelve Labs task create error [%s]: %s", resp.Status, chttp.SanitizeErrorBody([]byte(apiErr.Message)))
		}
		return nil, errors.Errorf("unexpected status [%s] from %s: %s", resp.Status, endpoint, chttp.SanitizeErrorBody(respData))
	}

	var out TaskCreateResponse
	if err := json.Unmarshal(respData, &out); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal task create response")
	}
	return &out, nil
}

// doTaskGet retrieves an async embedding task via GET {BaseAPI}/tasks/{id}.
// Per RESEARCH F-01 this single endpoint serves BOTH status polling and
// final result retrieval — the response carries status + data.
//
//nolint:unused // consumed by Plans 26-02/03
func (e *TwelveLabsEmbeddingFunction) doTaskGet(ctx context.Context, taskID string) (*TaskResponse, error) {
	if taskID == "" {
		return nil, errors.New("task ID cannot be empty (check _id JSON tag on create response)")
	}
	endpoint := strings.TrimRight(e.apiClient.BaseAPI, "/") + "/tasks/" + url.PathEscape(taskID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}
	httpReq.Header.Set("x-api-key", e.apiClient.APIKey.Value())
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", chttp.ChromaGoClientUserAgent)

	resp, err := e.apiClient.Client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send task retrieve request to %s", endpoint)
	}
	defer resp.Body.Close()

	respData, err := chttp.ReadLimitedBody(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr EmbedV2ErrorResponse
		if jsonErr := json.Unmarshal(respData, &apiErr); jsonErr == nil && apiErr.Message != "" {
			return nil, errors.Errorf("Twelve Labs task retrieve error [%s]: %s", resp.Status, chttp.SanitizeErrorBody([]byte(apiErr.Message)))
		}
		return nil, errors.Errorf("unexpected status [%s] from %s: %s", resp.Status, endpoint, chttp.SanitizeErrorBody(respData))
	}

	var out TaskResponse
	if err := json.Unmarshal(respData, &out); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal task retrieve response")
	}
	// Preserve raw body so pollTask can sanitize the authentic server reason
	// on status=failed (D-17). Copy respData — the underlying buffer may be
	// reused; json.RawMessage needs stable bytes.
	out.FailureDetail = append(json.RawMessage(nil), respData...)
	return &out, nil
}

func (e *TwelveLabsEmbeddingFunction) Name() string {
	return "twelvelabs"
}

func (e *TwelveLabsEmbeddingFunction) resolveModel(ctx context.Context) string {
	if m, ok := ctx.Value(modelContextKey).(string); ok && m != "" {
		return m
	}
	return string(e.apiClient.DefaultModel)
}

func validateTexts(texts []string) error {
	for i, text := range texts {
		if text == "" {
			return errors.Errorf("texts[%d]: text cannot be empty", i)
		}
	}
	return nil
}

// EmbedDocuments embeds a batch of text documents.
func (e *TwelveLabsEmbeddingFunction) EmbedDocuments(ctx context.Context, texts []string) ([]embeddings.Embedding, error) {
	if len(texts) == 0 {
		return embeddings.NewEmptyEmbeddings(), nil
	}
	if err := validateTexts(texts); err != nil {
		return nil, err
	}
	model := e.resolveModel(ctx)
	result := make([]embeddings.Embedding, 0, len(texts))
	for _, t := range texts {
		req := EmbedV2Request{
			InputType: "text",
			ModelName: model,
			Text:      &TextInput{InputText: t},
		}
		resp, err := e.doPost(ctx, req)
		if err != nil {
			return nil, errors.Wrap(err, "failed to embed text")
		}
		emb, err := embeddingFromResponse(resp)
		if err != nil {
			return nil, err
		}
		result = append(result, emb)
	}
	return result, nil
}

// EmbedQuery embeds a single query text.
func (e *TwelveLabsEmbeddingFunction) EmbedQuery(ctx context.Context, text string) (embeddings.Embedding, error) {
	embs, err := e.EmbedDocuments(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embs) == 0 {
		return nil, errors.New("no embedding returned for query")
	}
	return embs[0], nil
}

func (e *TwelveLabsEmbeddingFunction) DefaultSpace() embeddings.DistanceMetric {
	return embeddings.COSINE
}

func (e *TwelveLabsEmbeddingFunction) SupportedSpaces() []embeddings.DistanceMetric {
	return []embeddings.DistanceMetric{embeddings.COSINE, embeddings.L2, embeddings.IP}
}

// GetConfig returns a serializable config map for registry round-trip.
func (e *TwelveLabsEmbeddingFunction) GetConfig() embeddings.EmbeddingFunctionConfig {
	envVar := e.apiClient.APIKeyEnvVar
	if envVar == "" {
		envVar = APIKeyEnvVar
	}
	cfg := embeddings.EmbeddingFunctionConfig{
		"api_key_env_var":        envVar,
		"model_name":             string(e.apiClient.DefaultModel),
		"audio_embedding_option": e.apiClient.AudioEmbeddingOption,
	}
	if e.apiClient.BaseAPI != defaultBaseAPI {
		cfg["base_url"] = e.apiClient.BaseAPI
	}
	if e.apiClient.Insecure {
		cfg["insecure"] = true
	}
	if e.apiClient.asyncPollingEnabled {
		cfg["async_polling"] = true
		cfg["async_max_wait_ms"] = e.apiClient.asyncMaxWait.Milliseconds() // int64
	}
	return cfg
}

// NewTwelveLabsEmbeddingFunctionFromConfig creates a Twelve Labs EF from a config map.
func NewTwelveLabsEmbeddingFunctionFromConfig(cfg embeddings.EmbeddingFunctionConfig) (*TwelveLabsEmbeddingFunction, error) {
	envVar, ok := cfg["api_key_env_var"].(string)
	if !ok || envVar == "" {
		return nil, errors.New("api_key_env_var is required in config")
	}
	opts := []Option{WithAPIKeyFromEnvVar(envVar)}
	if model, ok := cfg["model_name"].(string); ok && model != "" {
		opts = append(opts, WithModel(embeddings.EmbeddingModel(model)))
	}
	if baseURL, ok := cfg["base_url"].(string); ok && baseURL != "" {
		opts = append(opts, WithBaseURL(baseURL))
	}
	if insecure, ok := cfg["insecure"].(bool); ok && insecure {
		opts = append(opts, WithInsecure())
	} else if embeddings.AllowInsecureFromEnv() {
		embeddings.LogInsecureEnvVarWarning("TwelveLabs")
		opts = append(opts, WithInsecure())
	}
	if audioOpt, ok := cfg["audio_embedding_option"].(string); ok && audioOpt != "" {
		opts = append(opts, WithAudioEmbeddingOption(audioOpt))
	}
	// Only enable async when BOTH keys are present and parseable. A missing or
	// malformed async_max_wait_ms with async_polling=true is treated as a broken
	// round-trip — we deliberately do NOT fall back to WithAsyncPolling(0) (the
	// 30-minute default) because that would silently enable a 30-minute blocking
	// bound on config the caller didn't specify. Missing key → not enabled.
	if enabled, ok := cfg["async_polling"].(bool); ok && enabled {
		if ms, ok := embeddings.ConfigInt(cfg, "async_max_wait_ms"); ok && ms >= 0 {
			opts = append(opts, WithAsyncPolling(time.Duration(ms)*time.Millisecond))
		}
	}
	return NewTwelveLabsEmbeddingFunction(opts...)
}

func float64sToFloat32s(in []float64) []float32 {
	out := make([]float32, len(in))
	for i, v := range in {
		out[i] = float32(v)
	}
	return out
}

func embeddingFromResponse(resp *EmbedV2Response) (embeddings.Embedding, error) {
	if resp == nil || len(resp.Data) == 0 {
		return nil, errors.New("no embedding returned from Twelve Labs API")
	}
	if len(resp.Data) > 1 {
		return nil, errors.Errorf("expected 1 embedding from Twelve Labs API, got %d", len(resp.Data))
	}
	if len(resp.Data[0].Embedding) == 0 {
		return nil, errors.New("empty embedding vector returned from Twelve Labs API")
	}
	return embeddings.NewEmbeddingFromFloat32(float64sToFloat32s(resp.Data[0].Embedding)), nil
}

func init() {
	if err := embeddings.RegisterDense("twelvelabs", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.EmbeddingFunction, error) {
		return NewTwelveLabsEmbeddingFunctionFromConfig(cfg)
	}); err != nil {
		panic(err)
	}
	if err := embeddings.RegisterContent("twelvelabs", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.ContentEmbeddingFunction, error) {
		return NewTwelveLabsEmbeddingFunctionFromConfig(cfg)
	}); err != nil {
		panic(err)
	}
}
