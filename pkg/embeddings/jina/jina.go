package jina

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"

	chttp "github.com/guiperry/chroma-go_cerebras/pkg/commons/http"
	"github.com/guiperry/chroma-go_cerebras/pkg/embeddings"
)

type EmbeddingType string

const (
	EmbeddingTypeFloat     EmbeddingType             = "float"
	DefaultBaseAPIEndpoint                           = "https://api.jina.ai/v1/embeddings"
	DefaultEmbeddingModel  embeddings.EmbeddingModel = "jina-embeddings-v2-base-en"
)

type EmbeddingRequest struct {
	Model         string              `json:"model"`
	Normalized    bool                `json:"normalized,omitempty"`
	EmbeddingType EmbeddingType       `json:"embedding_type,omitempty"`
	Input         []map[string]string `json:"input"`
}

type EmbeddingResponse struct {
	Model  string `json:"model"`
	Object string `json:"object"`
	Usage  struct {
		TotalTokens  int `json:"total_tokens"`
		PromptTokens int `json:"prompt_tokens"`
	}
	Data []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"` // TODO what about other embedding types - see cohere for example
	}
}

var _ embeddings.EmbeddingFunction = (*JinaEmbeddingFunction)(nil)

func getDefaults() *JinaEmbeddingFunction {
	return &JinaEmbeddingFunction{
		httpClient:        http.DefaultClient,
		defaultModel:      DefaultEmbeddingModel,
		embeddingEndpoint: DefaultBaseAPIEndpoint,
		normalized:        true,
		embeddingType:     EmbeddingTypeFloat,
	}
}

type JinaEmbeddingFunction struct {
	httpClient        *http.Client
	apiKey            string
	defaultModel      embeddings.EmbeddingModel
	embeddingEndpoint string
	normalized        bool
	embeddingType     EmbeddingType
}

func NewJinaEmbeddingFunction(opts ...Option) (*JinaEmbeddingFunction, error) {
	ef := getDefaults()
	for _, opt := range opts {
		err := opt(ef)
		if err != nil {
			return nil, err
		}
	}
	return ef, nil
}

func (e *JinaEmbeddingFunction) sendRequest(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal embedding request body")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, e.embeddingEndpoint, bytes.NewBuffer(payload))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create embedding request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", chttp.ChromaGoClientUserAgent)
	httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send embedding request")
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected response %v: %s", resp.Status, string(respData))
	}
	var response *EmbeddingResponse
	if err := json.Unmarshal(respData, &response); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal embedding response")
	}

	return response, nil
}

func (e *JinaEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]embeddings.Embedding, error) {
	var Input = make([]map[string]string, len(documents))

	for i, doc := range documents {
		Input[i] = map[string]string{
			"text": doc,
		}
	}
	req := &EmbeddingRequest{
		Model: string(e.defaultModel),
		Input: Input,
	}
	response, err := e.sendRequest(ctx, req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to embed documents")
	}
	var embs []embeddings.Embedding
	for _, data := range response.Data {
		embs = append(embs, embeddings.NewEmbeddingFromFloat32(data.Embedding))
	}

	return embs, nil
}

func (e *JinaEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (embeddings.Embedding, error) {
	var Input = make([]map[string]string, 1)

	Input[0] = map[string]string{
		"text": document,
	}
	req := &EmbeddingRequest{
		Model: string(e.defaultModel),
		Input: Input,
	}
	response, err := e.sendRequest(ctx, req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to embed query")
	}

	return embeddings.NewEmbeddingFromFloat32(response.Data[0].Embedding), nil
}
