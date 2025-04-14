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

// Docs:  https://docs.nomic.ai/reference/endpoints/nomic-embed-text

type TaskType string

const (
	DefaultEmbeddingModel             = NomicEmbedTextV1
	ModelContextVar                   = "model"
	DimensionalityContextVar          = "dimensionality"
	TaskTypeContextVar                = "task_type"
	APIKeyEnvVar                      = "NOMIC_API_KEY"
	DefaultBaseURL                    = "https://api-atlas.nomic.ai/v1/embedding"
	TextEmbeddingsEndpoint            = "/text"
	DefaultMaxBatchSize               = 100
	TaskTypeSearchQuery      TaskType = "search_query" //
	TaskTypeSearchDocument   TaskType = "search_document"
	TaskTypeClustering       TaskType = "clustering"
	TaskTypeClassification   TaskType = "classification"
	NomicEmbedTextV1                  = "nomic-embed-text-v1"
	NomicEmbedTextV15                 = "nomic-embed-text-v1.5"
)

type Client struct {
	apiKey                   string
	DefaultModel             embeddings.EmbeddingModel
	Client                   *http.Client
	DefaultContext           *context.Context
	MaxBatchSize             int
	EmbeddingEndpoint        string
	DefaultHeaders           map[string]string
	DefaultTaskType          *TaskType
	DefaultDimensionality    *int
	BaseURL                  string
	EmbeddingsEndpointSuffix string
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
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	if c.EmbeddingsEndpointSuffix == "" {
		c.EmbeddingsEndpointSuffix = TextEmbeddingsEndpoint
	}
	c.EmbeddingEndpoint = fmt.Sprintf("%s%s", c.BaseURL, c.EmbeddingsEndpointSuffix)
	return nil
}

func validate(c *Client) error {
	if c.apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

func NewNomicClient(opts ...Option) (*Client, error) {
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
	Model          embeddings.EmbeddingModel `json:"model"`
	Texts          []string                  `json:"texts"`
	TaskType       *TaskType                 `json:"task_type,omitempty"`
	Dimensionality *int                      `json:"dimensionality,omitempty"`
}

type CreateEmbeddingResponse struct {
	Usage      map[string]any `json:"usage,omitempty"`
	Embeddings [][]float32    `json:"embeddings"`
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
	embs := make([]embeddings.Embedding, len(embeddingResponse.Embeddings))
	for i, e := range embeddingResponse.Embeddings {
		embs[i] = embeddings.NewEmbeddingFromFloat32(e)
	}
	return embs, nil
}

var _ embeddings.EmbeddingFunction = (*NomicEmbeddingFunction)(nil)

type NomicEmbeddingFunction struct {
	apiClient *Client
}

func NewNomicEmbeddingFunction(opts ...Option) (*NomicEmbeddingFunction, error) {
	client, err := NewNomicClient(opts...)
	if err != nil {
		return nil, err
	}

	return &NomicEmbeddingFunction{apiClient: client}, nil
}

func (e *NomicEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]embeddings.Embedding, error) {
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
		model = embeddings.EmbeddingModel(ctx.Value(ModelContextVar).(string))
	}
	var dimensionality = e.apiClient.DefaultDimensionality
	if ctx.Value(DimensionalityContextVar) != nil {
		dimensionality = ctx.Value(DimensionalityContextVar).(*int)
	}
	var taskType = TaskTypeSearchDocument
	if ctx.Value(TaskTypeContextVar) != nil {
		taskType = ctx.Value(TaskTypeContextVar).(TaskType)
	}
	req := CreateEmbeddingRequest{
		Model:          model,
		Texts:          documents,
		Dimensionality: dimensionality,
		TaskType:       &taskType,
	}
	response, err := e.apiClient.CreateEmbedding(ctx, req)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (e *NomicEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (embeddings.Embedding, error) {
	var model = e.apiClient.DefaultModel
	if ctx.Value(ModelContextVar) != nil {
		model = embeddings.EmbeddingModel(ctx.Value(ModelContextVar).(string))
	}
	var dimensionality = e.apiClient.DefaultDimensionality
	if ctx.Value(DimensionalityContextVar) != nil {
		dimensionality = ctx.Value(DimensionalityContextVar).(*int)
	}
	var taskType = TaskTypeSearchQuery
	if ctx.Value(TaskTypeContextVar) != nil {
		taskType = ctx.Value(TaskTypeContextVar).(TaskType)
	}
	req := CreateEmbeddingRequest{
		Model:          model,
		Texts:          []string{document},
		Dimensionality: dimensionality,
		TaskType:       &taskType,
	}
	response, err := e.apiClient.CreateEmbedding(ctx, req)
	if err != nil {
		return nil, err
	}
	return response[0], nil
}
