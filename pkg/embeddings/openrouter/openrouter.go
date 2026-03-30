package openrouter

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

type Client struct {
	BaseURL        string               `json:"base_url,omitempty"`
	APIKey         embeddings.Secret    `json:"-" validate:"required"`
	APIKeyEnvVar   string               `json:"-"`
	Model          string               `json:"model,omitempty" validate:"required"`
	Dimensions     *int                 `json:"dimensions,omitempty"`
	User           string               `json:"user,omitempty"`
	EncodingFormat string               `json:"encoding_format,omitempty"`
	InputType      string               `json:"input_type,omitempty"`
	Provider       *ProviderPreferences `json:"provider,omitempty"`
	HTTPClient     *http.Client         `json:"-"`
	Insecure       bool                 `json:"insecure,omitempty"`
}

type Input struct {
	Text  string   `json:"-"`
	Texts []string `json:"-"`
}

func (i *Input) MarshalJSON() ([]byte, error) {
	if i.Texts != nil {
		return json.Marshal(i.Texts)
	}
	if i.Text != "" {
		return json.Marshal(i.Text)
	}
	return nil, errors.New("invalid input: no text provided")
}

type CreateEmbeddingRequest struct {
	Model          string               `json:"model"`
	Input          *Input               `json:"input"`
	Dimensions     *int                 `json:"dimensions,omitempty"`
	User           string               `json:"user,omitempty"`
	EncodingFormat string               `json:"encoding_format,omitempty"`
	InputType      string               `json:"input_type,omitempty"`
	Provider       *ProviderPreferences `json:"provider,omitempty"`
}

type EmbeddingData struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

type CreateEmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func NewOpenRouterClient(opts ...Option) (*Client, error) {
	client := &Client{}
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, errors.Wrap(err, "failed to apply OpenRouter option")
		}
	}
	if client.BaseURL == "" {
		client.BaseURL = DefaultBaseURL
	}
	if client.HTTPClient == nil {
		client.HTTPClient = &http.Client{}
	}
	if err := embeddings.NewValidator().Struct(client); err != nil {
		return nil, errors.Wrap(err, "failed to validate OpenRouter client options")
	}
	parsed, err := url.Parse(client.BaseURL)
	if err != nil {
		return nil, errors.Wrap(err, "invalid base URL")
	}
	if !client.Insecure && !strings.EqualFold(parsed.Scheme, "https") {
		return nil, errors.New("base URL must use HTTPS scheme for secure API key transmission; use WithInsecure() to override")
	}
	return client, nil
}

func (c *Client) CreateEmbedding(ctx context.Context, req *CreateEmbeddingRequest) (*CreateEmbeddingResponse, error) {
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request JSON")
	}
	endpoint, err := url.JoinPath(c.BaseURL, "embeddings")
	if err != nil {
		return nil, errors.Wrap(err, "failed to build endpoint URL")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqData))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", chttp.ChromaGoClientUserAgent)
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey.Value())

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request to OpenRouter API")
	}
	defer resp.Body.Close()

	respData, err := chttp.ReadLimitedBody(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected response %v, %v", resp.Status, string(respData))
	}

	var embResp CreateEmbeddingResponse
	if err := json.Unmarshal(respData, &embResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}
	return &embResp, nil
}

var _ embeddings.EmbeddingFunction = (*OpenRouterEmbeddingFunction)(nil)

type OpenRouterEmbeddingFunction struct {
	apiClient *Client
}

func NewOpenRouterEmbeddingFunction(opts ...Option) (*OpenRouterEmbeddingFunction, error) {
	client, err := NewOpenRouterClient(opts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize OpenRouter client")
	}
	return &OpenRouterEmbeddingFunction{apiClient: client}, nil
}

func (e *OpenRouterEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]embeddings.Embedding, error) {
	if len(documents) == 0 {
		return embeddings.NewEmptyEmbeddings(), nil
	}
	resp, err := e.apiClient.CreateEmbedding(ctx, &CreateEmbeddingRequest{
		Model:          e.apiClient.Model,
		Input:          &Input{Texts: documents},
		Dimensions:     e.apiClient.Dimensions,
		User:           e.apiClient.User,
		EncodingFormat: e.apiClient.EncodingFormat,
		InputType:      e.apiClient.InputType,
		Provider:       e.apiClient.Provider,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to embed documents")
	}
	result := make([]embeddings.Embedding, 0, len(resp.Data))
	for _, d := range resp.Data {
		result = append(result, embeddings.NewEmbeddingFromFloat32(d.Embedding))
	}
	return result, nil
}

func (e *OpenRouterEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (embeddings.Embedding, error) {
	resp, err := e.apiClient.CreateEmbedding(ctx, &CreateEmbeddingRequest{
		Model:          e.apiClient.Model,
		Input:          &Input{Text: document},
		Dimensions:     e.apiClient.Dimensions,
		User:           e.apiClient.User,
		EncodingFormat: e.apiClient.EncodingFormat,
		InputType:      e.apiClient.InputType,
		Provider:       e.apiClient.Provider,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to embed query")
	}
	if len(resp.Data) == 0 {
		return nil, errors.New("no embedding returned from OpenRouter API")
	}
	return embeddings.NewEmbeddingFromFloat32(resp.Data[0].Embedding), nil
}

func (e *OpenRouterEmbeddingFunction) Name() string {
	return "openrouter"
}

func (e *OpenRouterEmbeddingFunction) GetConfig() embeddings.EmbeddingFunctionConfig {
	envVar := e.apiClient.APIKeyEnvVar
	if envVar == "" {
		envVar = APIKeyEnvVar
	}
	cfg := embeddings.EmbeddingFunctionConfig{
		"api_key_env_var": envVar,
		"model_name":      e.apiClient.Model,
	}
	if e.apiClient.BaseURL != DefaultBaseURL {
		cfg["base_url"] = e.apiClient.BaseURL
	}
	if e.apiClient.Dimensions != nil {
		cfg["dimensions"] = *e.apiClient.Dimensions
	}
	if e.apiClient.User != "" {
		cfg["user"] = e.apiClient.User
	}
	if e.apiClient.EncodingFormat != "" {
		cfg["encoding_format"] = e.apiClient.EncodingFormat
	}
	if e.apiClient.InputType != "" {
		cfg["input_type"] = e.apiClient.InputType
	}
	if e.apiClient.Provider != nil {
		provData, err := json.Marshal(e.apiClient.Provider)
		if err == nil {
			var provMap map[string]any
			if err := json.Unmarshal(provData, &provMap); err == nil {
				cfg["provider"] = provMap
			}
		}
	}
	if e.apiClient.Insecure {
		cfg["insecure"] = true
	}
	return cfg
}

func (e *OpenRouterEmbeddingFunction) DefaultSpace() embeddings.DistanceMetric {
	return embeddings.COSINE
}

func (e *OpenRouterEmbeddingFunction) SupportedSpaces() []embeddings.DistanceMetric {
	return []embeddings.DistanceMetric{embeddings.COSINE, embeddings.L2, embeddings.IP}
}

// NewOpenRouterEmbeddingFunctionFromConfig creates an OpenRouter embedding function from a config map.
func NewOpenRouterEmbeddingFunctionFromConfig(cfg embeddings.EmbeddingFunctionConfig) (*OpenRouterEmbeddingFunction, error) {
	envVar, ok := cfg["api_key_env_var"].(string)
	if !ok || envVar == "" {
		return nil, errors.New("api_key_env_var is required in config")
	}
	opts := []Option{WithAPIKeyFromEnvVar(envVar)}
	if model, ok := cfg["model_name"].(string); ok && model != "" {
		opts = append(opts, WithModel(model))
	}
	if baseURL, ok := cfg["base_url"].(string); ok && baseURL != "" {
		opts = append(opts, WithBaseURL(baseURL))
	}
	if dims, ok := embeddings.ConfigInt(cfg, "dimensions"); ok && dims > 0 {
		opts = append(opts, WithDimensions(dims))
	}
	if user, ok := cfg["user"].(string); ok && user != "" {
		opts = append(opts, WithUser(user))
	}
	if format, ok := cfg["encoding_format"].(string); ok && format != "" {
		opts = append(opts, WithEncodingFormat(format))
	}
	if inputType, ok := cfg["input_type"].(string); ok && inputType != "" {
		opts = append(opts, WithInputType(inputType))
	}
	if provMap, ok := cfg["provider"].(map[string]any); ok {
		provData, err := json.Marshal(provMap)
		if err == nil {
			var prefs ProviderPreferences
			if err := json.Unmarshal(provData, &prefs); err == nil {
				opts = append(opts, WithProviderPreferences(&prefs))
			}
		}
	}
	if insecure, ok := cfg["insecure"].(bool); ok && insecure {
		opts = append(opts, WithInsecure())
	} else if embeddings.AllowInsecureFromEnv() {
		embeddings.LogInsecureEnvVarWarning("OpenRouter")
		opts = append(opts, WithInsecure())
	}
	return NewOpenRouterEmbeddingFunction(opts...)
}

func init() {
	if err := embeddings.RegisterDense("openrouter", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.EmbeddingFunction, error) {
		return NewOpenRouterEmbeddingFunctionFromConfig(cfg)
	}); err != nil {
		panic(err)
	}
}
