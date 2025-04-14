package gemini

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// Docs:  https://developers.cloudflare.com/workers-ai/ (Cloudflare Workers AI) and https://developers.cloudflare.com/workers-ai/models/embedding/ (Embedding API)

const (
	DefaultEmbeddingModel = "text-embedding-004"
	ModelContextVar       = "model"
	APIKeyEnvVar          = "GEMINI_API_KEY"
)

type Client struct {
	apiKey         string
	DefaultModel   embeddings.EmbeddingModel
	Client         *genai.Client
	DefaultContext *context.Context
	MaxBatchSize   int
}

func applyDefaults(c *Client) (err error) {
	if c.DefaultModel == "" {
		c.DefaultModel = DefaultEmbeddingModel
	}

	if c.DefaultContext == nil {
		ctx := context.Background()
		c.DefaultContext = &ctx
	}

	if c.Client == nil {
		c.Client, err = genai.NewClient(*c.DefaultContext, option.WithAPIKey(c.apiKey))
		if err != nil {
			return err
		}
	}
	return nil
}

func validate(c *Client) error {
	if c.apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

func NewGeminiClient(opts ...Option) (*Client, error) {
	client := &Client{}

	for _, opt := range opts {
		err := opt(client)
		if err != nil {
			return nil, err
		}
	}
	err := applyDefaults(client)
	if err != nil {
		return nil, err
	}
	if err := validate(client); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) CreateEmbedding(ctx context.Context, req []string) ([]embeddings.Embedding, error) {
	var em *genai.EmbeddingModel
	if ctx.Value(ModelContextVar) != nil {
		em = c.Client.EmbeddingModel(ctx.Value(ModelContextVar).(string))
	} else {
		em = c.Client.EmbeddingModel(string(c.DefaultModel))
	}
	b := em.NewBatch()
	for _, t := range req {
		b.AddContent(genai.Text(t))
	}
	res, err := em.BatchEmbedContents(ctx, b)
	if err != nil {
		return nil, err
	}
	var embs = make([][]float32, 0)
	for _, e := range res.Embeddings {
		embs = append(embs, e.Values)
	}

	return embeddings.NewEmbeddingsFromFloat32(embs)
}

// close closes the underlying client
//
//nolint:unused
func (c *Client) close() error {
	return c.Client.Close()
}

var _ embeddings.EmbeddingFunction = (*GeminiEmbeddingFunction)(nil)

type GeminiEmbeddingFunction struct {
	apiClient *Client
}

func NewGeminiEmbeddingFunction(opts ...Option) (*GeminiEmbeddingFunction, error) {
	client, err := NewGeminiClient(opts...)
	if err != nil {
		return nil, err
	}

	return &GeminiEmbeddingFunction{apiClient: client}, nil
}

// close closes the underlying client
//
//nolint:unused
func (e *GeminiEmbeddingFunction) close() error {
	return e.apiClient.close()
}

func (e *GeminiEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]embeddings.Embedding, error) {
	if e.apiClient.MaxBatchSize > 0 && len(documents) > e.apiClient.MaxBatchSize {
		return nil, fmt.Errorf("number of documents exceeds the maximum batch size %v", e.apiClient.MaxBatchSize)
	}
	if len(documents) == 0 {
		return embeddings.NewEmptyEmbeddings(), nil
	}

	response, err := e.apiClient.CreateEmbedding(ctx, documents)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (e *GeminiEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (embeddings.Embedding, error) {
	response, err := e.apiClient.CreateEmbedding(ctx, []string{document})
	if err != nil {
		return nil, err
	}
	return response[0], nil
}
