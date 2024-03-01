package hf

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/amikos-tech/chroma-go/types"
)

type HuggingFaceClient struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

func NewHuggingFaceClient(apiKey string, model string) *HuggingFaceClient {
	return &HuggingFaceClient{
		BaseURL: "https://api-inference.huggingface.co/pipeline/feature-extraction/",
		Client:  &http.Client{},
		APIKey:  apiKey,
		Model:   model,
	}
}

func (c *HuggingFaceClient) SetAPIKey(apiKey string) {
	c.APIKey = apiKey
}

func (c *HuggingFaceClient) SetBaseURL(baseURL string) {
	c.BaseURL = baseURL
}

func (c *HuggingFaceClient) getAPIKey() string {
	if c.APIKey == "" {
		panic("API Key not set")
	}
	return c.APIKey
}

func (c *HuggingFaceClient) SetModel(model string) {
	c.Model = model
}

type CreateEmbeddingRequest struct {
	Inputs  []string               `json:"inputs"`
	Options map[string]interface{} `json:"options"`
}

type CreateEmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func (c *CreateEmbeddingRequest) JSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *HuggingFaceClient) CreateEmbedding(ctx context.Context, req *CreateEmbeddingRequest) (*CreateEmbeddingResponse, error) {
	reqJSON, err := req.JSON()
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+c.Model, bytes.NewBufferString(reqJSON))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.getAPIKey())

	resp, err := c.Client.Do(httpReq)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected code %v", resp.Status)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var embeddings [][]float32
	if err := json.Unmarshal(respData, &embeddings); err != nil {
		return nil, err
	}
	var createEmbeddingResponse = CreateEmbeddingResponse{
		Embeddings: embeddings,
	}

	return &createEmbeddingResponse, nil
}

var _ types.EmbeddingFunction = (*HuggingFaceEmbeddingFunction)(nil)

type HuggingFaceEmbeddingFunction struct {
	apiClient *HuggingFaceClient
}

func NewHuggingFaceEmbeddingFunction(apiKey string, model string) *HuggingFaceEmbeddingFunction {
	cli := &HuggingFaceEmbeddingFunction{
		apiClient: NewHuggingFaceClient(apiKey, model),
	}

	return cli
}

func (e *HuggingFaceEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]*types.Embedding, error) {
	response, err := e.apiClient.CreateEmbedding(ctx, &CreateEmbeddingRequest{
		Inputs: documents,
	})
	if err != nil {
		return nil, err
	}
	return types.NewEmbeddingsFromFloat32(response.Embeddings), nil
}

func (e *HuggingFaceEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (*types.Embedding, error) {
	response, err := e.apiClient.CreateEmbedding(ctx, &CreateEmbeddingRequest{
		Inputs: []string{document},
	})
	if err != nil {
		return nil, err
	}
	return types.NewEmbeddingFromFloat32(response.Embeddings[0]), nil
}

func (e *HuggingFaceEmbeddingFunction) EmbedRecords(ctx context.Context, records []types.Record, force bool) error {
	return types.EmbedRecordsDefaultImpl(e, ctx, records, force)
}
