package cohere

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type CohereClient struct {
	BaseURL    string
	APIVersion string
	APIKey     string
	Client     *http.Client
}

func NewOpenAIClient(apiKey string) *CohereClient {
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
	Id         string                 `json:"id"`
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

func (c *CohereClient) CreateEmbedding(req *CreateEmbeddingRequest) (*CreateEmbeddingResponse, error) {
	reqJSON, err := req.JSON()
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.BaseURL+c.APIVersion+"/embed", bytes.NewBufferString(reqJSON))
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

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var createEmbeddingResponse CreateEmbeddingResponse
	if err := json.Unmarshal(respData, &createEmbeddingResponse); err != nil {
		return nil, err
	}

	return &createEmbeddingResponse, nil
}

type CohereEmbeddingFunction struct {
	apiClient *CohereClient
}

func NewCohereEmbeddingFunction(apiKey string) *CohereEmbeddingFunction {
	cli := &CohereEmbeddingFunction{
		apiClient: NewOpenAIClient(apiKey),
	}

	return cli
}

func (e *CohereEmbeddingFunction) CreateEmbedding(documents []string) ([][]float32, error) {

	response, err := e.apiClient.CreateEmbedding(&CreateEmbeddingRequest{
		Texts: documents,
	})
	if err != nil {
		return nil, err
	}
	return response.Embeddings, nil
}

func (e *CohereEmbeddingFunction) CreateEmbeddingWithModel(documents []string, model string) ([][]float32, error) {
	response, err := e.apiClient.CreateEmbedding(&CreateEmbeddingRequest{
		Model: model,
		Texts: documents,
	})
	if err != nil {
		return nil, err
	}
	return response.Embeddings, nil
}
