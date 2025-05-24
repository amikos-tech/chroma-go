package hf

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

type HuggingFaceClient struct {
	BaseURL        string
	APIKey         string
	Model          string
	Client         *http.Client
	DefaultHeaders map[string]string
	IsHFEIEndpoint bool
}

func NewHuggingFaceClient(apiKey string, model string) *HuggingFaceClient {
	return &HuggingFaceClient{
		BaseURL: "https://router.huggingface.co/hf-inference/models/",
		Client:  &http.Client{},
		APIKey:  apiKey,
		Model:   model,
	}
}

func NewHuggingFaceClientFromOptions(opts ...Option) (*HuggingFaceClient, error) {
	c := &HuggingFaceClient{
		BaseURL: "https://router.huggingface.co/hf-inference/models/",
		Client:  &http.Client{},
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, errors.Wrap(err, "failed to apply HuggingFace option")
		}
	}
	return c, nil
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
		return "", errors.Wrap(err, "failed to marshal embedding request")
	}
	return string(data), nil
}

func (c *HuggingFaceClient) CreateEmbedding(ctx context.Context, req *CreateEmbeddingRequest) (*CreateEmbeddingResponse, error) {
	reqJSON, err := req.JSON()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request to JSON")
	}
	var reqURL string
	if c.IsHFEIEndpoint {
		reqURL = c.BaseURL
	} else {
		reqURL, err = url.JoinPath(c.BaseURL, c.Model, "pipeline", "feature-extraction")
		if err != nil {
			return nil, errors.Wrap(err, "failed to build HF request URL")
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewBufferString(reqJSON))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}
	for k, v := range c.DefaultHeaders {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", chttp.ChromaGoClientUserAgent)
	httpReq.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request to Hugging Face API")
	}
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected code [%v] while making a request to %v: %v", resp.Status, reqURL, string(respData))
	}

	var embds [][]float32
	if err := json.Unmarshal(respData, &embds); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}
	var createEmbeddingResponse = CreateEmbeddingResponse{
		Embeddings: embds,
	}

	return &createEmbeddingResponse, nil
}

var _ embeddings.EmbeddingFunction = (*HuggingFaceEmbeddingFunction)(nil)

type HuggingFaceEmbeddingFunction struct {
	apiClient *HuggingFaceClient
}

func NewHuggingFaceEmbeddingFunction(apiKey string, model string) *HuggingFaceEmbeddingFunction {
	cli := &HuggingFaceEmbeddingFunction{
		apiClient: NewHuggingFaceClient(apiKey, model),
	}

	return cli
}

func NewHuggingFaceEmbeddingFunctionFromOptions(opts ...Option) (*HuggingFaceEmbeddingFunction, error) {
	cli, err := NewHuggingFaceClientFromOptions(opts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HuggingFace client")
	}

	return &HuggingFaceEmbeddingFunction{
		apiClient: cli,
	}, nil
}

func NewHuggingFaceEmbeddingInferenceFunction(baseURL string, opts ...Option) (*HuggingFaceEmbeddingFunction, error) {
	opts = append(opts, WithBaseURL(baseURL), WithIsHFEIEndpoint())
	cli, err := NewHuggingFaceClientFromOptions(opts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HFEI client")
	}

	return &HuggingFaceEmbeddingFunction{
		apiClient: cli,
	}, nil
}

func (e *HuggingFaceEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]embeddings.Embedding, error) {
	response, err := e.apiClient.CreateEmbedding(ctx, &CreateEmbeddingRequest{
		Inputs: documents,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to embed documents")
	}
	return embeddings.NewEmbeddingsFromFloat32(response.Embeddings)
}

func (e *HuggingFaceEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (embeddings.Embedding, error) {
	response, err := e.apiClient.CreateEmbedding(ctx, &CreateEmbeddingRequest{
		Inputs: []string{document},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to embed query")
	}
	return embeddings.NewEmbeddingFromFloat32(response.Embeddings[0]), nil
}
