package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	chttp "github.com/amikos-tech/chroma-go/pkg/commons/http"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

type EmbeddingModel string

const (
	ModelContextVar                     = "model"
	DimensionsContextVar                = "dimensions"
	TextEmbeddingAda002  EmbeddingModel = "text-embedding-ada-002"
	TextEmbedding3Small  EmbeddingModel = "text-embedding-3-small"
	TextEmbedding3Large  EmbeddingModel = "text-embedding-3-large"
)

type Input struct {
	Text                 string   `json:"-"`
	Texts                []string `json:"-"`
	Integers             []int    `json:"-"`
	ListOfListOfIntegers [][]int  `json:"-"`
}

func (i *Input) MarshalJSON() ([]byte, error) {
	switch {
	case i.Text != "":
		return json.Marshal(i.Text)
	case i.Texts != nil:
		return json.Marshal(i.Texts)
	case i.Integers != nil:
		return json.Marshal(i.Integers)
	case i.ListOfListOfIntegers != nil:
		return json.Marshal(i.ListOfListOfIntegers)
	default:
		return nil, fmt.Errorf("invalid input")
	}
}

type CreateEmbeddingRequest struct {
	Model      string `json:"model"`
	User       string `json:"user"`
	Input      *Input `json:"input"`
	Dimensions *int   `json:"dimensions,omitempty"`
}

func (c *CreateEmbeddingRequest) JSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal embedding request JSON")
	}
	return string(data), nil
}

func (c *CreateEmbeddingRequest) String() string {
	data, _ := json.Marshal(c)
	return string(data)
}

type EmbeddingData struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

type Usage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type CreateEmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  Usage           `json:"usage"`
}

func (c *CreateEmbeddingResponse) String() string {
	data, _ := json.Marshal(c)
	return string(data)
}

type OpenAIClient struct {
	BaseURL    string
	APIKey     string
	OrgID      string
	Client     *http.Client
	Model      string
	Dimensions *int
	User       string
}

func applyDefaults(c *OpenAIClient) {
	if c.BaseURL == "" {
		c.BaseURL = "https://api.openai.com/v1/"
	}
	if c.Client == nil {
		c.Client = &http.Client{}
	}
	if c.User == "" {
		c.User = chttp.ChromaGoClientUserAgent
	}
}

func validate(c *OpenAIClient) error {
	if c.APIKey == "" {
		return errors.New("API key is required")
	}
	if c.BaseURL == "" {
		return errors.New("Base URL is required")
	}
	return nil
}

func NewOpenAIClient(apiKey string, opts ...Option) (*OpenAIClient, error) {
	client := &OpenAIClient{
		BaseURL: "https://api.openai.com/v1/",
		Client:  &http.Client{},
		APIKey:  apiKey,
		Model:   string(TextEmbeddingAda002),
	}
	for _, opt := range opts {
		err := opt(client)
		if err != nil {
			return nil, errors.Wrap(err, "failed to apply OpenAI option")
		}
	}
	applyDefaults(client)
	if err := validate(client); err != nil {
		return nil, errors.Wrap(err, "failed to validate OpenAI client options")
	}
	return client, nil
}

func (c *OpenAIClient) CreateEmbedding(ctx context.Context, req *CreateEmbeddingRequest) (*CreateEmbeddingResponse, error) {
	if req.Model == "" {
		req.Model = c.Model
	}
	reqJSON, err := req.JSON()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request JSON")
	}
	endpoint, err := url.JoinPath(c.BaseURL, "embeddings")
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse URL")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(reqJSON))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", chttp.ChromaGoClientUserAgent)
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	// OpenAI Organization ID (Optional)
	if c.OrgID != "" {
		httpReq.Header.Set("OpenAI-Organization", c.OrgID)
	}

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request to OpenAI API")
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected response %v, %v", resp.Status, string(respData))
	}

	var createEmbeddingResponse CreateEmbeddingResponse
	if err := json.Unmarshal(respData, &createEmbeddingResponse); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}

	return &createEmbeddingResponse, nil
}

var _ embeddings.EmbeddingFunction = (*OpenAIEmbeddingFunction)(nil)

type OpenAIEmbeddingFunction struct {
	apiClient *OpenAIClient
}

func NewOpenAIEmbeddingFunction(apiKey string, opts ...Option) (*OpenAIEmbeddingFunction, error) {
	apiClient, err := NewOpenAIClient(apiKey, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize OpenAI client")
	}
	cli := &OpenAIEmbeddingFunction{
		apiClient: apiClient,
	}

	return cli, nil
}

func ConvertToMatrix(response *CreateEmbeddingResponse) [][]float32 {
	var matrix [][]float32

	for _, embeddingData := range response.Data {
		matrix = append(matrix, embeddingData.Embedding)
	}

	return matrix
}

// getModel returns the model from the context if it exists, otherwise it returns the default model
func (e *OpenAIEmbeddingFunction) getModel(ctx context.Context) string {
	model := e.apiClient.Model
	if m, ok := ctx.Value(ModelContextVar).(string); ok {
		model = m
	}
	return model
}

// getDimensions returns the dimensions from the context if it exists, otherwise it returns the default dimensions
func (e *OpenAIEmbeddingFunction) getDimensions(ctx context.Context) *int {
	dimensions := e.apiClient.Dimensions
	if dims, ok := ctx.Value(DimensionsContextVar).(*int); ok {
		dimensions = dims
	}
	return dimensions
}

func (e *OpenAIEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]embeddings.Embedding, error) {
	response, err := e.apiClient.CreateEmbedding(ctx, &CreateEmbeddingRequest{
		User:  e.apiClient.User,
		Model: e.getModel(ctx),
		Input: &Input{
			Texts: documents,
		},
		Dimensions: e.getDimensions(ctx),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to embed documents")
	}
	return embeddings.NewEmbeddingsFromFloat32(ConvertToMatrix(response))
}

func (e *OpenAIEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (embeddings.Embedding, error) {
	response, err := e.apiClient.CreateEmbedding(ctx, &CreateEmbeddingRequest{
		Model: e.getModel(ctx),
		User:  e.apiClient.User,
		Input: &Input{
			Texts: []string{document},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to embed query")
	}
	return embeddings.NewEmbeddingFromFloat32(ConvertToMatrix(response)[0]), nil
}
