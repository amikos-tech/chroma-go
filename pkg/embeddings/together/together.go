package together

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// Docs:  https://docs.together.ai/docs/embeddings-rest.  Models - https://docs.together.ai/docs/embeddings-models.

const (
	defaultBaseAPI = "https://api.together.xyz/v1/embeddings"
	// https://docs.together.ai/reference/embeddings
	defaultMaxSize = 100
)

type TogetherAIClient struct {
	BaseAPI        string
	APIToken       string
	DefaultModel   embeddings.EmbeddingModel
	MaxBatchSize   int
	DefaultHeaders map[string]string
	Client         *http.Client
}

func applyDefaults(c *TogetherAIClient) {
	if c.Client == nil {
		c.Client = http.DefaultClient
	}
	if c.BaseAPI == "" {
		c.BaseAPI = defaultBaseAPI
	}
	if c.MaxBatchSize == 0 {
		c.MaxBatchSize = defaultMaxSize
	}
	if c.DefaultModel == "" {
		c.DefaultModel = "togethercomputer/m2-bert-80M-8k-retrieval"
	}
}

func validate(c *TogetherAIClient) error {
	if c.APIToken == "" {
		return fmt.Errorf("API key is required")
	}
	if c.MaxBatchSize < 1 {
		return fmt.Errorf("max batch size must be greater than 0")
	}
	if c.MaxBatchSize > defaultMaxSize {
		return fmt.Errorf("max batch size must be less than %d", defaultMaxSize)
	}
	return nil
}

func NewTogetherClient(opts ...Option) (*TogetherAIClient, error) {
	client := &TogetherAIClient{}

	for _, opt := range opts {
		err := opt(client)
		if err != nil {
			return nil, err
		}
	}
	applyDefaults(client)
	if err := validate(client); err != nil {
		return nil, err
	}
	return client, nil
}

type EmbeddingInputs struct {
	Input  string
	Inputs []string
}

func (e *EmbeddingInputs) MarshalJSON() ([]byte, error) {
	if e.Input != "" {
		return json.Marshal(e.Input)
	}
	if e.Inputs != nil {
		return json.Marshal(e.Inputs)
	}
	return nil, fmt.Errorf("EmbeddingInput has no data")
}

type CreateEmbeddingRequest struct {
	Model string           `json:"model"`
	Input *EmbeddingInputs `json:"input"`
}

type EmbeddingResult struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type CreateEmbeddingResponse struct {
	Object    string            `json:"object"`
	Data      []EmbeddingResult `json:"data"`
	Model     string            `json:"model"`
	RequestID string            `json:"request_id"`
}

func (c *CreateEmbeddingRequest) JSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *TogetherAIClient) CreateEmbedding(ctx context.Context, req *CreateEmbeddingRequest) (*CreateEmbeddingResponse, error) {
	reqJSON, err := req.JSON()
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseAPI, bytes.NewBufferString(reqJSON))
	if err != nil {
		return nil, err
	}
	for k, v := range c.DefaultHeaders {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIToken)
	resp, err := c.Client.Do(httpReq)

	if err != nil {
		return nil, err
	}
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var embeddings CreateEmbeddingResponse
	if err := json.Unmarshal(respData, &embeddings); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected code [%v] while making a request to %v. errors: %v", resp.Status, c.BaseAPI, string(respData))
	}

	return &embeddings, nil
}

var _ embeddings.EmbeddingFunction = (*TogetherEmbeddingFunction)(nil)

type TogetherEmbeddingFunction struct {
	apiClient *TogetherAIClient
}

func NewTogetherEmbeddingFunction(opts ...Option) (*TogetherEmbeddingFunction, error) {
	client, err := NewTogetherClient(opts...)
	if err != nil {
		return nil, err
	}

	return &TogetherEmbeddingFunction{apiClient: client}, nil
}

func (e *TogetherEmbeddingFunction) getModelFromContext(ctx context.Context) embeddings.EmbeddingModel {
	model := e.apiClient.DefaultModel
	if m, ok := ctx.Value("model").(string); ok {
		model = embeddings.EmbeddingModel(m)
	}
	return model
}

func (e *TogetherEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]embeddings.Embedding, error) {
	if len(documents) > e.apiClient.MaxBatchSize {
		return nil, fmt.Errorf("number of documents exceeds the maximum batch size %v", e.apiClient.MaxBatchSize)
	}
	if len(documents) == 0 {
		return embeddings.NewEmptyEmbeddings(), nil
	}
	req := &CreateEmbeddingRequest{
		Model: string(e.getModelFromContext(ctx)),
		Input: &EmbeddingInputs{Inputs: documents},
	}
	response, err := e.apiClient.CreateEmbedding(ctx, req)
	if err != nil {
		return nil, err
	}
	emb := make([]embeddings.Embedding, 0, len(response.Data))
	for _, result := range response.Data {
		emb = append(emb, embeddings.NewEmbeddingFromFloat32(result.Embedding))
	}
	return emb, nil
}

func (e *TogetherEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (embeddings.Embedding, error) {
	req := &CreateEmbeddingRequest{
		Model: string(e.getModelFromContext(ctx)),
		Input: &EmbeddingInputs{Input: document},
	}
	response, err := e.apiClient.CreateEmbedding(ctx, req)
	if err != nil {
		return nil, err
	}
	return embeddings.NewEmbeddingFromFloat32(response.Data[0].Embedding), nil
}
