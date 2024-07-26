package cohere

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	ccommons "github.com/amikos-tech/chroma-go/pkg/commons/cohere"
	"github.com/amikos-tech/chroma-go/types"
)

const (
	DefaultEmbedEndpoint = "embed"
)

type CohereModel = ccommons.CohereModel

const (
	ModelEmbedEnglishV20      CohereModel = "embed-english-v2.0"
	ModelEmbedEnglishV30      CohereModel = "embed-english-v3.0"
	ModelEmbedMultilingualV20 CohereModel = "embed-multilingual-v2.0"
	ModelEmbedMultilingualV30 CohereModel = "embed-multilingual-v3.0"
	ModelEmbedEnglishLightV20 CohereModel = "embed-english-light-v2.0"
	ModelEmbedEnglishLightV30 CohereModel = "embed-english-light-v3.0"
	DefaultEmbedModel         CohereModel = ModelEmbedEnglishV20
)

type TruncateMode string

const (
	NONE  TruncateMode = "NONE"
	START TruncateMode = "START"
	END   TruncateMode = "END"
)

type InputType string

const (
	InputTypeSearchDocument InputType = "search_document"
	InputTypeSearchQuery    InputType = "search_query"
	InputTypeClassification InputType = "classification"
	InputTypeClustering     InputType = "clustering"
)

type EmbeddingType string

const (
	EmbeddingTypeFloat32 EmbeddingType = "float"
	EmbeddingTypeInt8    EmbeddingType = "int8"
	EmbeddingTypeUInt8   EmbeddingType = "uint8"
	EmbeddingTypeBinary  EmbeddingType = "binary"
	EmbeddingTypeUBinary EmbeddingType = "ubinary"
)

type CreateEmbeddingRequest struct {
	Model          string          `json:"model"`
	Texts          []string        `json:"texts"`
	Truncate       TruncateMode    `json:"truncate,omitempty"`
	EmbeddingTypes []EmbeddingType `json:"embedding_types,omitempty"`
	InputType      InputType       `json:"input_type,omitempty"`
}

type EmbeddingTypes struct {
	Float32 [][]float32 `json:"float"`
	Int8    [][]int8    `json:"int8"`
	UInt8   [][]uint8   `json:"uint8"`
}

type EmbeddingsResponse struct {
	Embeddings      [][]float32
	EmbeddingsTypes *EmbeddingTypes
}

var _ types.EmbeddingFunction = (*CohereEmbeddingFunction)(nil)

type CohereEmbeddingFunction struct {
	ccommons.CohereClient
	DefaultTruncateMode   TruncateMode
	DefaultEmbeddingTypes []EmbeddingType
	DefaultInputType      InputType
	EmbeddingEndpoint     string
}

func NewCohereEmbeddingFunction(opts ...Option) (*CohereEmbeddingFunction, error) {
	ef := &CohereEmbeddingFunction{}
	ccOpts := make([]ccommons.Option, 0)
	ccOpts = append(ccOpts, ccommons.WithDefaultModel(DefaultEmbedModel))
	// stagger the options to pass to the cohere client
	for _, opt := range opts {
		ccOpts = append(ccOpts, opt(ef))
	}
	cohereCommonClient, err := ccommons.NewCohereClient(ccOpts...)
	if err != nil {
		return nil, err
	}
	ef.CohereClient = *cohereCommonClient
	ef.EmbeddingEndpoint = cohereCommonClient.GetAPIEndpoint(DefaultEmbedEndpoint)

	return ef, nil
}

func (c *CohereEmbeddingFunction) CreateEmbedding(ctx context.Context, req *CreateEmbeddingRequest) (*CreateEmbeddingResponse, error) {
	reqJSON, err := req.JSON()
	if err != nil {
		return nil, err
	}

	httpReq, err := c.CohereClient.GetRequest(ctx, "POST", c.EmbeddingEndpoint, reqJSON)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.CohereClient.Client.Do(httpReq)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		respData, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected code %v for response: %s", resp.Status, respData)
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

// EmbedDocuments embeds the given documents and returns the embeddings.
// Accepts value model in context to override the default model.
// Accepts value embedding_types in context to override the default embedding types.
func (c *CohereEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]*types.Embedding, error) {
	_model := c.CohereClient.DefaultModel
	if ctx.Value("model") != nil {
		_model = ctx.Value("model").(CohereModel)
	}
	_embeddingTypes := c.DefaultEmbeddingTypes
	if ctx.Value("embedding_types") != nil {
		_embeddingTypes = []EmbeddingType{ctx.Value("embedding_types").(EmbeddingType)}
	}
	response, err := c.CreateEmbedding(ctx, &CreateEmbeddingRequest{
		Texts:          documents,
		Model:          _model.String(),
		InputType:      InputTypeSearchDocument,
		EmbeddingTypes: _embeddingTypes,
	})
	if err != nil {
		return nil, err
	}
	switch {
	case response.Embeddings.Embeddings != nil:
		return types.NewEmbeddingsFromFloat32(response.Embeddings.Embeddings), nil

	case response.Embeddings.EmbeddingsTypes != nil:
		switch {
		case response.Embeddings.EmbeddingsTypes.Float32 != nil:
			return types.NewEmbeddingsFromFloat32(response.Embeddings.EmbeddingsTypes.Float32), nil

		case response.Embeddings.EmbeddingsTypes.Int8 != nil:
			return types.NewEmbeddingsFromInt32(int32FromInt8Embeddings(response.Embeddings.EmbeddingsTypes.Int8)), nil

		case response.Embeddings.EmbeddingsTypes.UInt8 != nil:
			return types.NewEmbeddingsFromInt32(int32FromUInt8Embeddings(response.Embeddings.EmbeddingsTypes.UInt8)), nil

		default:
			return nil, fmt.Errorf("unsupported embedding type")
		}

	default:
		return nil, fmt.Errorf("unexpected response from API")
	}
}

// EmbedQuery embeds the given query and returns the embedding.
// Accepts value model in context to override the default model.
// Accepts value embedding_types in context to override the default embedding types.
func (c *CohereEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (*types.Embedding, error) {
	_model := c.CohereClient.DefaultModel
	if ctx.Value("model") != nil {
		_model = ctx.Value("model").(CohereModel)
	}
	_embeddingTypes := c.DefaultEmbeddingTypes
	if ctx.Value("embedding_types") != nil {
		_embeddingTypes = []EmbeddingType{ctx.Value("embedding_types").(EmbeddingType)}
	}
	response, err := c.CreateEmbedding(ctx, &CreateEmbeddingRequest{
		Texts:          []string{document},
		Model:          _model.String(),
		InputType:      InputTypeSearchQuery,
		EmbeddingTypes: _embeddingTypes,
	})
	if err != nil {
		return nil, err
	}
	switch {
	case response.Embeddings.Embeddings != nil:
		return types.NewEmbeddingFromFloat32(response.Embeddings.Embeddings[0]), nil

	case response.Embeddings.EmbeddingsTypes != nil:
		switch {
		case response.Embeddings.EmbeddingsTypes.Float32 != nil:
			return types.NewEmbeddingFromFloat32(response.Embeddings.EmbeddingsTypes.Float32[0]), nil

		case response.Embeddings.EmbeddingsTypes.Int8 != nil:
			return types.NewEmbeddingFromInt32(int32FromInt8Embeddings(response.Embeddings.EmbeddingsTypes.Int8)[0]), nil

		case response.Embeddings.EmbeddingsTypes.UInt8 != nil:
			return types.NewEmbeddingFromInt32(int32FromUInt8Embeddings(response.Embeddings.EmbeddingsTypes.UInt8)[0]), nil

		default:
			return nil, fmt.Errorf("unsupported embedding type")
		}

	default:
		return nil, fmt.Errorf("unexpected response from API")
	}
}

// EmbedRecords embeds the given records and returns the embeddings.
// Accepts value model in context to override the default model.
// Accepts value embedding_types in context to override the default embedding types.
func (c *CohereEmbeddingFunction) EmbedRecords(ctx context.Context, records []*types.Record, force bool) error {
	return types.EmbedRecordsDefaultImpl(c, ctx, records, force)
}

func (e *EmbeddingsResponse) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &e.Embeddings); err == nil {
		return nil
	}
	if err := json.Unmarshal(b, &e.EmbeddingsTypes); err == nil {
		return nil
	}
	return fmt.Errorf("EmbeddingInput must be a string or an array of strings")
}

func (e *EmbeddingsResponse) MarshalJSON() ([]byte, error) {
	if e.Embeddings != nil {
		return json.Marshal(e.Embeddings)
	}
	if e.EmbeddingsTypes != nil {
		return json.Marshal(e.EmbeddingsTypes)
	}
	return nil, fmt.Errorf("EmbeddingsResponse has no data")
}

type CreateEmbeddingResponse struct {
	ID         string              `json:"id"`
	Texts      []string            `json:"texts"`
	Embeddings *EmbeddingsResponse `json:"embeddings"`
	Meta       map[string]any      `json:"meta"`
}

func (c *CreateEmbeddingRequest) JSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func int32FromUInt8Embeddings(embeddings [][]uint8) [][]int32 {
	var int32s = make([][]int32, len(embeddings))

	for i, innerSlice := range embeddings {
		newInnerSlice := make([]int32, len(innerSlice)) // Pre-allocate with the exact size
		for j, num := range innerSlice {
			newInnerSlice[j] = int32(num)
		}
		int32s[i] = newInnerSlice
	}
	return int32s
}

func int32FromInt8Embeddings(embeddings [][]int8) [][]int32 {
	var int32s = make([][]int32, len(embeddings))

	for i, innerSlice := range embeddings {
		newInnerSlice := make([]int32, len(innerSlice)) // Pre-allocate with the exact size
		for j, num := range innerSlice {
			newInnerSlice[j] = int32(num)
		}
		int32s[i] = newInnerSlice
	}
	return int32s
}
