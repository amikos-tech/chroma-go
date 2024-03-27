package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/amikos-tech/chroma-go/types"
)

type OllamaClient struct {
	BaseURL        string
	Model          string
	Client         *http.Client
	DefaultHeaders map[string]string
}
type CreateEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type CreateEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
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
	var url = c.BaseURL
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

var _ types.EmbeddingFunction = (*OllamaEmbeddingFunction)(nil)

func NewOllamaEmbeddingFunction(option ...Option) (*OllamaEmbeddingFunction, error) {
	client, err := NewOllamaClient(option...)
	if err != nil {
		return nil, err
	}
	return &OllamaEmbeddingFunction{
		apiClient: client,
	}, nil
}

func (e *OllamaEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]*types.Embedding, error) {
	embeddings := make([]*types.Embedding, 0)
	for _, document := range documents {
		response, err := e.apiClient.createEmbedding(ctx, &CreateEmbeddingRequest{
			Model:  e.apiClient.Model,
			Prompt: document,
		})
		if err != nil {
			return nil, err
		}
		embeddings = append(embeddings, types.NewEmbeddingFromFloat32(response.Embedding))
	}
	return embeddings, nil
}

func (e *OllamaEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (*types.Embedding, error) {
	response, err := e.apiClient.createEmbedding(ctx, &CreateEmbeddingRequest{
		Model:  e.apiClient.Model,
		Prompt: document,
	})
	if err != nil {
		return nil, err
	}
	return types.NewEmbeddingFromFloat32(response.Embedding), nil
}

func (e *OllamaEmbeddingFunction) EmbedRecords(ctx context.Context, records []*types.Record, force bool) error {
	return types.EmbedRecordsDefaultImpl(e, ctx, records, force)
}
