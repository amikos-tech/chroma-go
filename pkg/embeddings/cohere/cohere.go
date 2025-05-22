package cohere

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	ccommons "github.com/amikos-tech/chroma-go/pkg/commons/cohere"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/pkg/errors"
)

const (
	DefaultEmbedEndpoint = "embed"
)

const (
	ModelEmbedEnglishV20      embeddings.EmbeddingModel = "embed-english-v2.0"
	ModelEmbedEnglishV30      embeddings.EmbeddingModel = "embed-english-v3.0"
	ModelEmbedMultilingualV20 embeddings.EmbeddingModel = "embed-multilingual-v2.0"
	ModelEmbedMultilingualV30 embeddings.EmbeddingModel = "embed-multilingual-v3.0"
	ModelEmbedEnglishLightV20 embeddings.EmbeddingModel = "embed-english-light-v2.0"
	ModelEmbedEnglishLightV30 embeddings.EmbeddingModel = "embed-english-light-v3.0"
	DefaultEmbedModel         embeddings.EmbeddingModel = ModelEmbedEnglishV20
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

type EmbeddingsResponse struct {
	Float32 [][]float32 `json:"float,omitempty"`
	Int8    [][]int8    `json:"int8,omitempty"`
	UInt8   [][]uint8   `json:"uint8,omitempty"`
}

func (e *EmbeddingsResponse) UnmarshalJSON(b []byte) error {
	s := string(b)
	switch {
	case strings.Contains(s, "uint"):
		var tstruct = struct {
			Uint8 [][]uint8 `json:"uint8,omitempty"`
		}{
			Uint8: make([][]uint8, 0),
		}
		err := json.Unmarshal(b, &tstruct)
		if err != nil {
			return err
		}
		e.UInt8 = tstruct.Uint8
	case strings.Contains(string(b), "int8"):
		var tstruct = struct {
			Int8 [][]int8 `json:"int8,omitempty"`
		}{
			Int8: make([][]int8, 0),
		}
		err := json.Unmarshal(b, &tstruct)
		if err != nil {
			return err
		}
		e.Int8 = tstruct.Int8
	case strings.Contains(string(b), "binary"):
		return errors.New("binary embedding type not supported")
	case strings.Contains(string(b), "ubinary"):
		return errors.New("ubinary embedding type not supported")
	default:
		err := json.Unmarshal(b, &e.Float32)
		if err != nil {
			return err
		}
	}
	return nil
}

var _ embeddings.EmbeddingFunction = (*CohereEmbeddingFunction)(nil)

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

	httpReq, err := c.GetRequest(ctx, "POST", c.EmbeddingEndpoint, reqJSON)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.DoRequest(httpReq)

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
	fmt.Println(string(respData))
	var createEmbeddingResponse CreateEmbeddingResponse
	if err := json.Unmarshal(respData, &createEmbeddingResponse); err != nil {
		return nil, err
	}

	return &createEmbeddingResponse, nil
}

// EmbedDocuments embeds the given documents and returns the embeddings.
// Accepts value model in context to override the default model.
// Accepts value embedding_types in context to override the default embedding types.
func (c *CohereEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]embeddings.Embedding, error) {
	_model := c.DefaultModel
	if ctx.Value("model") != nil {
		_model = embeddings.EmbeddingModel(ctx.Value("model").(string))
	}
	_embeddingTypes := c.DefaultEmbeddingTypes
	if ctx.Value("embedding_types") != nil {
		_embeddingTypes = []EmbeddingType{ctx.Value("embedding_types").(EmbeddingType)}
	}
	response, err := c.CreateEmbedding(ctx, &CreateEmbeddingRequest{
		Texts:          documents,
		Model:          string(_model),
		InputType:      InputTypeSearchDocument,
		EmbeddingTypes: _embeddingTypes,
	})
	if err != nil {
		return nil, err
	}
	fmt.Println("response.Embeddings.EmbeddingsTypes: ", response.Embeddings.Float32)
	switch {
	case len(response.Embeddings.Float32) > 0:
		return embeddings.NewEmbeddingsFromFloat32(response.Embeddings.Float32)

	case len(response.Embeddings.Int8) > 0:
		return embeddings.NewEmbeddingsFromInt32(int32FromInt8Embeddings(response.Embeddings.Int8))

	case len(response.Embeddings.UInt8) > 0:
		return embeddings.NewEmbeddingsFromInt32(int32FromUInt8Embeddings(response.Embeddings.UInt8))

	default:
		return nil, fmt.Errorf("unsupported embedding type")
	}
}

// EmbedQuery embeds the given query and returns the embedding.
// Accepts value model in context to override the default model.
// Accepts value embedding_types in context to override the default embedding types.
func (c *CohereEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (embeddings.Embedding, error) {
	_model := c.DefaultModel
	if ctx.Value("model") != nil {
		_model = embeddings.EmbeddingModel(ctx.Value("model").(string))
	}
	_embeddingTypes := c.DefaultEmbeddingTypes
	if ctx.Value("embedding_types") != nil {
		_embeddingTypes = []EmbeddingType{ctx.Value("embedding_types").(EmbeddingType)}
	}
	response, err := c.CreateEmbedding(ctx, &CreateEmbeddingRequest{
		Texts:          []string{document},
		Model:          string(_model),
		InputType:      InputTypeSearchQuery,
		EmbeddingTypes: _embeddingTypes,
	})
	if err != nil {
		return nil, err
	}
	switch {
	case len(response.Embeddings.Float32) > 0:
		return embeddings.NewEmbeddingFromFloat32(response.Embeddings.Float32[0]), nil

	case len(response.Embeddings.Int8) > 0:
		return embeddings.NewInt32Embedding(int32FromInt8Embeddings(response.Embeddings.Int8)[0]), nil

	case len(response.Embeddings.UInt8) > 0:
		return embeddings.NewInt32Embedding(int32FromUInt8Embeddings(response.Embeddings.UInt8)[0]), nil

	default:
		return nil, fmt.Errorf("unsupported embedding type")
	}
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
