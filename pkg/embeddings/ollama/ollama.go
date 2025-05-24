package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	chttp "github.com/amikos-tech/chroma-go/pkg/commons/http"
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
		b, err := json.Marshal(e.Input)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal embedding input")
		}
		return b, nil
	} else if len(e.Inputs) > 0 {
		b, err := json.Marshal(e.Inputs)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal embedding input")
		}
		return b, nil
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
		return "", errors.Wrap(err, "failed to marshal embedding request JSON")
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
			return nil, errors.Wrap(err, "failed to apply Ollama option")
		}
	}
	return client, nil
}

func (c *OllamaClient) createEmbedding(ctx context.Context, req *CreateEmbeddingRequest) (*CreateEmbeddingResponse, error) {
	reqJSON, err := req.JSON()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal embedding request JSON")
	}
	endpoint, err := url.JoinPath(c.BaseURL, "/api/embed")
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse Ollama embedding endpoint")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(reqJSON))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}
	for k, v := range c.DefaultHeaders {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", chttp.ChromaGoClientUserAgent)

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make HTTP request to Ollama embedding endpoint")
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected code [%v] while making a request to %v: %v", resp.Status, endpoint, string(respData))
	}

	var embeddingResponse CreateEmbeddingResponse
	if err := json.Unmarshal(respData, &embeddingResponse); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal embedding response")
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
		return nil, errors.Wrap(err, "failed to initialize OllamaClient")
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
		return nil, errors.Wrap(err, "failed to embed documents")
	}
	return embeddings.NewEmbeddingsFromFloat32(response.Embeddings)
}

func (e *OllamaEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (embeddings.Embedding, error) {
	response, err := e.apiClient.createEmbedding(ctx, &CreateEmbeddingRequest{
		Model: string(e.apiClient.Model),
		Input: &EmbeddingInput{Input: document},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to embed query")
	}
	return embeddings.NewEmbeddingFromFloat32(response.Embeddings[0]), nil
}
