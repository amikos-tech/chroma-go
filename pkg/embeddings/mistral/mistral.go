package mistral

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

const (
	DefaultEmbeddingModel = "mistral-embed"
	ModelContextVar       = "model"
	APIKeyEnvVar          = "MISTRAL_API_KEY"
	DefaultBaseURL        = "https://api.mistral.ai"
	EmbeddingsEndpoint    = "/v1/embeddings"
	DefaultMaxBatchSize   = 100
)

type Client struct {
	apiKey            string
	DefaultModel      string
	Client            *http.Client
	DefaultContext    *context.Context
	MaxBatchSize      int
	EmbeddingEndpoint string
	DefaultHeaders    map[string]string
}

func applyDefaults(c *Client) (err error) {
	if c.DefaultModel == "" {
		c.DefaultModel = DefaultEmbeddingModel
	}

	if c.DefaultContext == nil {
		ctx := context.Background()
		c.DefaultContext = &ctx
	}

	if c.Client == nil {
		c.Client = http.DefaultClient
	}
	if c.MaxBatchSize == 0 {
		c.MaxBatchSize = DefaultMaxBatchSize
	}
	var s = DefaultBaseURL + EmbeddingsEndpoint
	c.EmbeddingEndpoint = s
	return nil
}

func validate(c *Client) error {
	if c.apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

func NewMistralClient(opts ...Option) (*Client, error) {
	client := &Client{}

	for _, opt := range opts {
		err := opt(client)
		if err != nil {
			return nil, err
		}
	}
	err := applyDefaults(client)
	if err != nil {
		return nil, err
	}
	if err := validate(client); err != nil {
		return nil, err
	}
	return client, nil
}

type CreateEmbeddingRequest struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	EncodingFormat string   `json:"encoding_format,omitempty"`
}

type Embedding struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"` // TODO this can be also ints depending on encoding format
	Index     int       `json:"index"`
}

type CreateEmbeddingResponse struct {
	ID     string         `json:"id"`
	Object string         `json:"object"`
	Model  string         `json:"model"`
	Usage  map[string]any `json:"usage"`
	Data   []Embedding    `json:"data"`
}

func (c *CreateEmbeddingRequest) JSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *Client) CreateEmbedding(ctx context.Context, req CreateEmbeddingRequest) ([]embeddings.Embedding, error) {
	reqJSON, err := req.JSON()
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.EmbeddingEndpoint, bytes.NewBufferString(reqJSON))
	if err != nil {
		return nil, err
	}
	for k, v := range c.DefaultHeaders {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected code [%v] while making a request to %v", resp.Status, c.EmbeddingEndpoint)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var embeddingResponse CreateEmbeddingResponse
	if err := json.Unmarshal(respData, &embeddingResponse); err != nil {
		return nil, err
	}
	embs := make([]embeddings.Embedding, len(embeddingResponse.Data))
	for i, e := range embeddingResponse.Data {
		embs[i] = embeddings.NewEmbeddingFromFloat32(e.Embedding)
	}
	return embs, nil
}

var _ embeddings.EmbeddingFunction = (*MistralEmbeddingFunction)(nil)

type MistralEmbeddingFunction struct {
	apiClient *Client
}

func NewMistralEmbeddingFunction(opts ...Option) (*MistralEmbeddingFunction, error) {
	client, err := NewMistralClient(opts...)
	if err != nil {
		return nil, err
	}

	return &MistralEmbeddingFunction{apiClient: client}, nil
}

func (e *MistralEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]embeddings.Embedding, error) {
	if len(documents) > e.apiClient.MaxBatchSize {
		return nil, fmt.Errorf("number of documents exceeds the maximum batch size %v", e.apiClient.MaxBatchSize)
	}
	if e.apiClient.MaxBatchSize > 0 && len(documents) > e.apiClient.MaxBatchSize {
		return nil, fmt.Errorf("number of documents exceeds the maximum batch size %v", e.apiClient.MaxBatchSize)
	}
	if len(documents) == 0 {
		return embeddings.NewEmptyEmbeddings(), nil
	}
	var model = e.apiClient.DefaultModel
	if ctx.Value(ModelContextVar) != nil {
		model = ctx.Value(ModelContextVar).(string)
	}
	req := CreateEmbeddingRequest{
		Model: model,
		Input: documents,
	}
	response, err := e.apiClient.CreateEmbedding(ctx, req)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (e *MistralEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (embeddings.Embedding, error) {
	var model = e.apiClient.DefaultModel
	if ctx.Value(ModelContextVar) != nil {
		model = ctx.Value(ModelContextVar).(string)
	}
	req := CreateEmbeddingRequest{
		Model: model,
		Input: []string{document},
	}
	response, err := e.apiClient.CreateEmbedding(ctx, req)
	if err != nil {
		return nil, err
	}
	return response[0], nil
}
