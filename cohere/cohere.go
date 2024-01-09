package cohere

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/amikos-tech/chroma-go"
)

type CohereClient struct {
	BaseURL    string
	APIVersion string
	APIKey     string
	Client     *http.Client
}

func NewCohereClient(apiKey string) *CohereClient {
	return &CohereClient{
		BaseURL:    "https://api.cohere.ai/",
		Client:     &http.Client{},
		APIVersion: "v1",
		APIKey:     apiKey,
	}
}

func (c *CohereClient) SetAPIKey(apiKey string) {
	c.APIKey = apiKey
}

func (c *CohereClient) SetBaseURL(baseURL string) {
	c.BaseURL = baseURL
}

func (c *CohereClient) getAPIKey() string {
	if c.APIKey == "" {
		panic("API Key not set")
	}
	return c.APIKey
}

type TruncateMode string

const (
	NONE  TruncateMode = "NONE"
	START TruncateMode = "START"
	END   TruncateMode = "END"
)

type CreateEmbeddingRequest struct {
	Model               string       `json:"model"`
	Texts               []string     `json:"texts"`
	Truncate            TruncateMode `json:"truncate"`
	Compress            bool         `json:"compress"`
	CompressionCodebook string       `json:"compression_codebook"`
}

type CreateEmbeddingResponse struct {
	ID         string                 `json:"id"`
	Texts      []string               `json:"texts"`
	Embeddings [][]float32            `json:"embeddings"`
	Meta       map[string]interface{} `json:"meta"`
}

func (c *CreateEmbeddingRequest) JSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *CohereClient) CreateEmbedding(ctx context.Context, req *CreateEmbeddingRequest) (*CreateEmbeddingResponse, error) {
	reqJSON, err := req.JSON()
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+c.APIVersion+"/embed", bytes.NewBufferString(reqJSON))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Request-Source", "chroma-go-client")
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

	var createEmbeddingResponse CreateEmbeddingResponse
	if err := json.Unmarshal(respData, &createEmbeddingResponse); err != nil {
		return nil, err
	}

	return &createEmbeddingResponse, nil
}

var _ chroma.EmbeddingFunction = (*CohereEmbeddingFunction)(nil)

type CohereEmbeddingFunction struct {
	apiClient *CohereClient
}

func NewCohereEmbeddingFunction(apiKey string) *CohereEmbeddingFunction {
	cli := &CohereEmbeddingFunction{
		apiClient: NewCohereClient(apiKey),
	}

	return cli
}

func (e *CohereEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
	response, err := e.apiClient.CreateEmbedding(ctx, &CreateEmbeddingRequest{
		Texts: documents,
	})
	if err != nil {
		return nil, err
	}
	return response.Embeddings, nil
}

func (e *CohereEmbeddingFunction) EmbedQuery(ctx context.Context, document string) ([]float32, error) {
	response, err := e.apiClient.CreateEmbedding(ctx, &CreateEmbeddingRequest{
		Texts: []string{document},
	})
	if err != nil {
		return nil, err
	}
	return response.Embeddings[0], nil
}
