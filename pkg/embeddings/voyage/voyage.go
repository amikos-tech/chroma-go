package voyage

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// Docs:  https://docs.together.ai/docs/embeddings-rest.  Models - https://docs.together.ai/docs/embeddings-models.

type InputType string
type EncodingFormat string

const (
	defaultBaseAPI = "https://api.voyageai.com/v1/embeddings"
	// https://docs.voyageai.com/docs/embeddings
	defaultMaxSize                          = 128
	DefaultTruncation                       = true
	InputTypeQuery           InputType      = "query"
	InputTypeDocument        InputType      = "document"
	defaultModel                            = "voyage-2"
	EncodingFormatBase64     EncodingFormat = "base64"
	InputTypeContextVar                     = "inputType"
	ModelContextVar                         = "model"
	TruncationContextVar                    = "truncation"
	EncodingFormatContextVar                = "encodingFormat"
	APIKeyEnvVar                            = "VOYAGE_API_KEY"
)

type VoyageAIClient struct {
	BaseAPI               string
	APIKey                string
	DefaultModel          embeddings.EmbeddingModel
	MaxBatchSize          int
	DefaultHeaders        map[string]string
	DefaultTruncation     *bool
	DefaultEncodingFormat *EncodingFormat
	Client                *http.Client
}

func applyDefaults(c *VoyageAIClient) {
	if c.Client == nil {
		c.Client = http.DefaultClient
	}
	if c.BaseAPI == "" {
		c.BaseAPI = defaultBaseAPI
	}

	if c.DefaultTruncation == nil {
		var defaultTruncation = DefaultTruncation
		c.DefaultTruncation = &defaultTruncation
	}

	if c.MaxBatchSize == 0 {
		c.MaxBatchSize = defaultMaxSize
	}
	if c.DefaultModel == "" {
		c.DefaultModel = defaultModel
	}
}

func validate(c *VoyageAIClient) error {
	if c.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	if c.MaxBatchSize < 1 {
		return fmt.Errorf("max batch size must be greater than 0")
	}
	if c.MaxBatchSize > defaultMaxSize {
		return fmt.Errorf("max batch size must be less than %d", defaultMaxSize)
	}
	return nil
}

func NewVoyageAIClient(opts ...Option) (*VoyageAIClient, error) {
	client := &VoyageAIClient{}

	for _, opt := range opts {
		err := opt(client)
		if err != nil {
			return nil, err
		}
	}
	applyDefaults(client)
	if err := validate(client); err != nil {
		return nil, err
	}
	return client, nil
}

type EmbeddingInputs struct {
	Input  string
	Inputs []string
}

func (e *EmbeddingInputs) MarshalJSON() ([]byte, error) {
	if e.Input != "" {
		return json.Marshal(e.Input)
	}
	if e.Inputs != nil {
		return json.Marshal(e.Inputs)
	}
	return nil, fmt.Errorf("EmbeddingInput has no data")
}

// from voyageai python client - https://github.com/voyage-ai/voyageai-python/blob/e565fb60b854e80ead526a57ea0e6eb1db9efc33/voyageai/api_resources/embedding.py#L30-L32
func bytesToFloat32s(b []byte) ([]float32, error) {
	if len(b)%4 != 0 {
		return nil, fmt.Errorf("byte slice length must be a multiple of 4")
	}

	result := make([]float32, len(b)/4)
	for i := range result {
		bits := binary.LittleEndian.Uint32(b[i*4:]) // Or binary.BigEndian
		result[i] = math.Float32frombits(bits)
	}
	return result, nil
}

func (e *EmbeddingTypeResult) UnmarshalJSON(data []byte) error {
	var str string
	var floats []float32
	if err := json.Unmarshal(data, &str); err == nil {
		decoded, err := base64.StdEncoding.DecodeString(str)
		if err != nil {
			return err
		}
		e.Floats, err = bytesToFloat32s(decoded)
		if err != nil {
			return err
		}
		return nil
	}
	if err := json.Unmarshal(data, &floats); err == nil {
		e.Floats = floats
		return nil
	}
	return fmt.Errorf("unexpected data type %v", string(data))
}

type CreateEmbeddingRequest struct {
	Model          string           `json:"model"`
	Input          *EmbeddingInputs `json:"input"`
	InputType      *InputType       `json:"input_type"`
	Truncation     *bool            `json:"truncation"`
	EncodingFormat *EncodingFormat  `json:"encoding_format"`
}

type EmbeddingTypeResult struct {
	Floats []float32
}

type EmbeddingResult struct {
	Object    string               `json:"object"`
	Embedding *EmbeddingTypeResult `json:"embedding"`
	Index     int                  `json:"index"`
}

type UsageResult struct {
	TotalTokens int `json:"total_tokens"`
}

type CreateEmbeddingResponse struct {
	Object string            `json:"object"`
	Data   []EmbeddingResult `json:"data"`
	Model  string            `json:"model"`
	Usage  *UsageResult      `json:"usage"`
}

func (c *CreateEmbeddingRequest) JSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *VoyageAIClient) CreateEmbedding(ctx context.Context, req *CreateEmbeddingRequest) (*CreateEmbeddingResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	reqJSON, err := req.JSON()
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseAPI, bytes.NewBufferString(reqJSON))
	if err != nil {
		return nil, err
	}
	for k, v := range c.DefaultHeaders {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	resp, err := c.Client.Do(httpReq)

	if err != nil {
		return nil, err
	}
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var embeddings CreateEmbeddingResponse
	if err := json.Unmarshal(respData, &embeddings); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected code [%v] while making a request to %v. errors: %v", resp.Status, c.BaseAPI, string(respData))
	}

	return &embeddings, nil
}

var _ embeddings.EmbeddingFunction = (*VoyageAIEmbeddingFunction)(nil)

type VoyageAIEmbeddingFunction struct {
	apiClient *VoyageAIClient
}

func NewVoyageAIEmbeddingFunction(opts ...Option) (*VoyageAIEmbeddingFunction, error) {
	client, err := NewVoyageAIClient(opts...)
	if err != nil {
		return nil, err
	}

	return &VoyageAIEmbeddingFunction{apiClient: client}, nil
}

// getModel returns the model from the context if it exists, otherwise it returns the default model
func (e *VoyageAIEmbeddingFunction) getModel(ctx context.Context) embeddings.EmbeddingModel {
	model := e.apiClient.DefaultModel
	if m, ok := ctx.Value(ModelContextVar).(string); ok {
		model = embeddings.EmbeddingModel(m)
	}
	return model
}

// getTruncation returns the truncation from the context if it exists, otherwise it returns the default truncation
func (e *VoyageAIEmbeddingFunction) getTruncation(ctx context.Context) *bool {
	model := e.apiClient.DefaultTruncation
	if m, ok := ctx.Value(TruncationContextVar).(*bool); ok {
		model = m
	}
	return model
}

// getInputType returns the input type from the context if it exists, otherwise it returns the default input type
func (e *VoyageAIEmbeddingFunction) getInputType(ctx context.Context, inputType InputType) *InputType {
	model := &inputType
	if m, ok := ctx.Value(InputTypeContextVar).(*InputType); ok {
		model = m
	}
	return model
}

func (e *VoyageAIEmbeddingFunction) getEncodingFormat(ctx context.Context) *EncodingFormat {
	model := e.apiClient.DefaultEncodingFormat
	if m, ok := ctx.Value(EncodingFormatContextVar).(*EncodingFormat); ok {
		model = m
	}
	return model
}

func (e *VoyageAIEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]embeddings.Embedding, error) {
	if len(documents) > e.apiClient.MaxBatchSize {
		return nil, fmt.Errorf("number of documents exceeds the maximum batch size %v", e.apiClient.MaxBatchSize)
	}
	if len(documents) == 0 {
		return embeddings.NewEmptyEmbeddings(), nil
	}

	req := &CreateEmbeddingRequest{
		Model:          string(e.getModel(ctx)),
		Input:          &EmbeddingInputs{Inputs: documents},
		Truncation:     e.getTruncation(ctx),
		InputType:      e.getInputType(ctx, InputTypeDocument),
		EncodingFormat: e.getEncodingFormat(ctx),
	}
	response, err := e.apiClient.CreateEmbedding(ctx, req)
	if err != nil {
		return nil, err
	}
	embs := make([]embeddings.Embedding, 0, len(response.Data))
	for _, result := range response.Data {
		embs = append(embs, embeddings.NewEmbeddingFromFloat32(result.Embedding.Floats))
	}
	return embs, nil
}

func (e *VoyageAIEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (embeddings.Embedding, error) {
	req := &CreateEmbeddingRequest{
		Model:          string(e.getModel(ctx)),
		Input:          &EmbeddingInputs{Input: document},
		Truncation:     e.getTruncation(ctx),
		InputType:      e.getInputType(ctx, InputTypeDocument),
		EncodingFormat: e.getEncodingFormat(ctx),
	}
	response, err := e.apiClient.CreateEmbedding(ctx, req)
	if err != nil {
		return nil, err
	}
	return embeddings.NewEmbeddingFromFloat32(response.Data[0].Embedding.Floats), nil
}
