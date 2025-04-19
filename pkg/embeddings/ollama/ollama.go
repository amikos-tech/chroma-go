package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

type OllamaClient struct {
	BaseURL        string
	Model          embeddings.EmbeddingModel
	Client         *http.Client
	DefaultHeaders map[string]string
}

type EmbeddingInput struct {
	Input  string
	Inputs []string
}

func (e EmbeddingInput) MarshalJSON() ([]byte, error) {
	if e.Input != "" {
		return json.Marshal(e.Input)
	} else if len(e.Inputs) > 0 {
		return json.Marshal(e.Inputs)
	}
	return json.Marshal(nil)
}

type CreateEmbeddingRequest struct {
	Model string          `json:"model"`
	Input *EmbeddingInput `json:"input"`
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

func NewOllamaClient(opts ...Option) (*OllamaClient, error) {
	client := &OllamaClient{
		Client: &http.Client{},
	}
	for _, opt := range opts {
		err := opt(client)
		if err != nil {
			return nil, err
		}
	}
	return client, nil
}

func (c *OllamaClient) createEmbedding(ctx context.Context, req *CreateEmbeddingRequest) (*CreateEmbeddingResponse, error) {
	reqJSON, err := req.JSON()
	if err != nil {
		return nil, err
	}
	var url string
	if !strings.HasSuffix(c.BaseURL, "/") {
		url = c.BaseURL + "/api/embed"
	} else {
		url = c.BaseURL + "api/embed"
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(reqJSON))
	if err != nil {
		return nil, err
	}
	for k, v := range c.DefaultHeaders {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected code [%v] while making a request to %v", resp.Status, url)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var embeddingResponse CreateEmbeddingResponse
	if err := json.Unmarshal(respData, &embeddingResponse); err != nil {
		return nil, err
	}
	return &embeddingResponse, nil
}

type OllamaEmbeddingFunction struct {
	apiClient *OllamaClient
}

var _ embeddings.EmbeddingFunction = (*OllamaEmbeddingFunction)(nil)

func NewOllamaEmbeddingFunction(option ...Option) (*OllamaEmbeddingFunction, error) {
	client, err := NewOllamaClient(option...)
	if err != nil {
		return nil, err
	}
	return &OllamaEmbeddingFunction{
		apiClient: client,
	}, nil
}

func (e *OllamaEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]embeddings.Embedding, error) {
	response, err := e.apiClient.createEmbedding(ctx, &CreateEmbeddingRequest{
		Model: string(e.apiClient.Model),
		Input: &EmbeddingInput{Inputs: documents},
	})
	if err != nil {
		return nil, err
	}
	return embeddings.NewEmbeddingsFromFloat32(response.Embeddings)
}

func (e *OllamaEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (embeddings.Embedding, error) {
	response, err := e.apiClient.createEmbedding(ctx, &CreateEmbeddingRequest{
		Model: string(e.apiClient.Model),
		Input: &EmbeddingInput{Input: document},
	})
	if err != nil {
		return nil, err
	}
	return embeddings.NewEmbeddingFromFloat32(response.Embeddings[0]), nil
}
