package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/amikos-tech/chroma-go/types"
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
	Model string `json:"model"`
	User  string `json:"user"`
	Input *Input `json:"input"`
}

func (c *CreateEmbeddingRequest) JSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
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
	BaseURL string
	APIKey  string
	OrgID   string
	Client  *http.Client
	Model   string
}

func NewOpenAIClient(apiKey string, opts ...Option) (*OpenAIClient, error) {
	client := &OpenAIClient{
		BaseURL: "https://api.openai.com/v1/",
		Client:  &http.Client{},
		APIKey:  apiKey,
		Model:   string(TextEmbeddingAda002),
	}
	err := applyClientOptions(client, opts...)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *OpenAIClient) SetAPIKey(apiKey string) {
	c.APIKey = apiKey
}

func (c *OpenAIClient) SetOrgID(orgID string) {
	c.OrgID = orgID
}

func (c *OpenAIClient) SetBaseURL(baseURL string) {
	c.BaseURL = baseURL
}

func (c *OpenAIClient) getAPIKey() string {
	if c.APIKey == "" {
		panic("API Key not set")
	}
	return c.APIKey
}

func (c *OpenAIClient) CreateEmbedding(ctx context.Context, req *CreateEmbeddingRequest) (*CreateEmbeddingResponse, error) {
	if req.Model == "" {
		req.Model = c.Model
	}
	reqJSON, err := req.JSON()
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"embeddings", bytes.NewBufferString(reqJSON))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.getAPIKey())

	// OpenAI Organization ID (Optional)
	if c.OrgID != "" {
		httpReq.Header.Set("OpenAI-Organization", c.OrgID)
	}

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println(err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected code %v", resp.Status)
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

var _ types.EmbeddingFunction = (*OpenAIEmbeddingFunction)(nil)

type OpenAIEmbeddingFunction struct {
	apiClient *OpenAIClient
}

func NewOpenAIEmbeddingFunction(apiKey string, opts ...Option) (*OpenAIEmbeddingFunction, error) {
	apiClient, err := NewOpenAIClient(apiKey, opts...)
	if err != nil {
		return nil, err
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

func (e *OpenAIEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]*types.Embedding, error) {
	response, err := e.apiClient.CreateEmbedding(ctx, &CreateEmbeddingRequest{
		User: "chroma-go-client",
		Input: &Input{
			Texts: documents,
		},
	})
	if err != nil {
		return nil, err
	}
	return types.NewEmbeddingsFromFloat32(ConvertToMatrix(response)), nil
}

func (e *OpenAIEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (*types.Embedding, error) {
	response, err := e.apiClient.CreateEmbedding(ctx, &CreateEmbeddingRequest{
		Model: "text-embedding-ada-002",
		User:  "chroma-go-client",
		Input: &Input{
			Texts: []string{document},
		},
	})
	if err != nil {
		return nil, err
	}
	return types.NewEmbeddingFromFloat32(ConvertToMatrix(response)[0]), nil
}

func (e *OpenAIEmbeddingFunction) EmbedRecords(ctx context.Context, records []*types.Record, force bool) error {
	return types.EmbedRecordsDefaultImpl(e, ctx, records, force)
}
