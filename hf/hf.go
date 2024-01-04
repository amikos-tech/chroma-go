package hf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func (c *HuggingFaceClient) CreateEmbedding(req *CreateEmbeddingRequest) (*CreateEmbeddingResponse, error) {
	reqJSON, err := req.JSON()
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.BaseURL+c.Model, bytes.NewBufferString(reqJSON))
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

type HuggingFaceEmbeddingFunction struct {
	apiClient *HuggingFaceClient
}

func NewHuggingFaceEmbeddingFunction(apiKey string, model string) *HuggingFaceEmbeddingFunction {
	cli := &HuggingFaceEmbeddingFunction{
		apiClient: NewHuggingFaceClient(apiKey, model),
	}

	return cli
}

func (e *HuggingFaceEmbeddingFunction) CreateEmbedding(documents []string) ([][]float32, error) {
	response, err := e.apiClient.CreateEmbedding(&CreateEmbeddingRequest{
		Inputs: documents,
	})
	if err != nil {
		return nil, err
	}
	return response.Embeddings, nil
}

func (e *HuggingFaceEmbeddingFunction) CreateEmbeddingWithModel(documents []string, model string) ([][]float32, error) {
	e.apiClient.Model = model
	response, err := e.apiClient.CreateEmbedding(&CreateEmbeddingRequest{
		Inputs: documents,
	})
	if err != nil {
		return nil, err
	}
	return response.Embeddings, nil
}
