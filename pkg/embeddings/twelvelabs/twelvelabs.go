package twelvelabs

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

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
}

func applyDefaults(c *TwelveLabsClient) {
	if c.Client == nil {
		c.Client = http.DefaultClient
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

// EmbedV2Request is the request body for POST /v1.3/embed-v2.
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
	Base64String string `json:"base_64_string,omitempty"`
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
var _ embeddings.IntentMapper = (*TwelveLabsEmbeddingFunction)(nil)

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
			return nil, errors.Errorf("Twelve Labs API error [%s]: %s", resp.Status, apiErr.Message)
		}
		return nil, errors.Errorf("unexpected status [%s] from %s: %s", resp.Status, e.apiClient.BaseAPI, string(respData))
	}

	var embedResp EmbedV2Response
	if err := json.Unmarshal(respData, &embedResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}
	return &embedResp, nil
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

// EmbedDocuments embeds a batch of text documents.
func (e *TwelveLabsEmbeddingFunction) EmbedDocuments(ctx context.Context, texts []string) ([]embeddings.Embedding, error) {
	if len(texts) == 0 {
		return embeddings.NewEmptyEmbeddings(), nil
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
		if len(resp.Data) == 0 {
			return nil, errors.New("no embedding returned from Twelve Labs API")
		}
		result = append(result, embeddings.NewEmbeddingFromFloat32(float64sToFloat32s(resp.Data[0].Embedding)))
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
		return nil, errors.New("no embedding returned from Twelve Labs API")
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
	return NewTwelveLabsEmbeddingFunction(opts...)
}

func float64sToFloat32s(in []float64) []float32 {
	out := make([]float32, len(in))
	for i, v := range in {
		out[i] = float32(v)
	}
	return out
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
