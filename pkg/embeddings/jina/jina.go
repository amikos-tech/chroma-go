package jina

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	chttp "github.com/amikos-tech/chroma-go/pkg/commons/http"
	"github.com/amikos-tech/chroma-go/types"
)

type EmbeddingType string

const (
	EmbeddingTypeFloat     EmbeddingType        = "float"
	DefaultBaseAPIEndpoint                      = "https://api.jina.ai/v1/embeddings"
	DefaultEmbeddingModel  types.EmbeddingModel = "jina-embeddings-v2-base-en"
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

var _ types.EmbeddingFunction = (*JinaEmbeddingFunction)(nil)

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
	defaultModel      types.EmbeddingModel
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
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", e.embeddingEndpoint, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", chttp.ChromaGoClientUserAgent)
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", e.apiKey))

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		// TODO serialize body in error
		return nil, fmt.Errorf("unexpected response %v: %s", resp.Status, respData)
	}
	var response *EmbeddingResponse
	if err := json.Unmarshal(respData, &response); err != nil {
		return nil, err
	}

	return response, nil
}

func (e *JinaEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]*types.Embedding, error) {
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
		return nil, err
	}
	var embeddings []*types.Embedding
	for _, data := range response.Data {
		embeddings = append(embeddings, types.NewEmbeddingFromFloat32(data.Embedding))
	}

	return embeddings, nil
}

func (e *JinaEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (*types.Embedding, error) {
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
		return nil, err
	}

	return types.NewEmbeddingFromFloat32(response.Data[0].Embedding), nil
}

func (e *JinaEmbeddingFunction) EmbedRecords(ctx context.Context, records []*types.Record, force bool) error {
	return types.EmbedRecordsDefaultImpl(e, ctx, records, force)
}
