package gemini

import (
	"context"
	"math"

	"github.com/pkg/errors"
	"google.golang.org/genai"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

type contextKey struct{ name string }

var (
	modelContextKey     = contextKey{"model"}
	taskTypeContextKey  = contextKey{"task_type"}
	dimensionContextKey = contextKey{"dimension"}
)

func ContextWithModel(ctx context.Context, model string) context.Context {
	return context.WithValue(ctx, modelContextKey, model)
}

func ContextWithTaskType(ctx context.Context, taskType TaskType) context.Context {
	return context.WithValue(ctx, taskTypeContextKey, taskType)
}

func ContextWithDimension(ctx context.Context, dimension int) context.Context {
	return context.WithValue(ctx, dimensionContextKey, dimension)
}

const (
	DefaultEmbeddingModel = "gemini-embedding-001"
	APIKeyEnvVar          = "GEMINI_API_KEY"
)

type Client struct {
	APIKey           embeddings.Secret `json:"-" validate:"required"`
	APIKeyEnvVar     string
	DefaultModel     embeddings.EmbeddingModel
	DefaultTaskType  TaskType
	DefaultDimension *int32
	Client           *genai.Client
	DefaultContext   *context.Context
	MaxBatchSize     int
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
		c.Client, err = genai.NewClient(*c.DefaultContext, &genai.ClientConfig{
			APIKey:  c.APIKey.Value(),
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func validate(c *Client) error {
	return embeddings.NewValidator().Struct(c)
}

func NewGeminiClient(opts ...Option) (*Client, error) {
	client := &Client{}

	for _, opt := range opts {
		err := opt(client)
		if err != nil {
			return nil, errors.Wrap(err, "failed to apply Gemini option")
		}
	}
	err := applyDefaults(client)
	if err != nil {
		return nil, err
	}
	if err := validate(client); err != nil {
		return nil, errors.Wrap(err, "failed to validate Gemini client options")
	}
	return client, nil
}

func (c *Client) CreateEmbedding(ctx context.Context, req []string) ([]embeddings.Embedding, error) {
	model := string(c.DefaultModel)
	if m, ok := ctx.Value(modelContextKey).(string); ok {
		model = m
	}
	taskType := c.DefaultTaskType
	if t, ok := ctx.Value(taskTypeContextKey).(TaskType); ok {
		taskType = t
	} else if t, ok := ctx.Value(taskTypeContextKey).(string); ok {
		// Backward compatibility for callers that previously stored plain string manually.
		taskType = TaskType(t)
	}
	outputDimensionality, err := outputDimensionalityFromContext(ctx, c.DefaultDimension)
	if err != nil {
		return nil, errors.Wrap(err, "invalid dimension override")
	}
	contents := make([]*genai.Content, len(req))
	for i, t := range req {
		contents[i] = genai.NewContentFromText(t, genai.RoleUser)
	}
	res, err := c.Client.Models.EmbedContent(ctx, model, contents, buildEmbedContentConfig(taskType, outputDimensionality))
	if err != nil {
		return nil, errors.Wrap(err, "failed to embed contents")
	}
	if res == nil || len(res.Embeddings) == 0 {
		return nil, errors.New("no embeddings returned from Gemini API")
	}
	embs := make([][]float32, 0, len(res.Embeddings))
	for _, e := range res.Embeddings {
		embs = append(embs, e.Values)
	}

	return embeddings.NewEmbeddingsFromFloat32(embs)
}

func buildEmbedContentConfig(taskType TaskType, outputDimensionality *int32) *genai.EmbedContentConfig {
	if taskType == "" && outputDimensionality == nil {
		return nil
	}
	return &genai.EmbedContentConfig{
		TaskType:             string(taskType),
		OutputDimensionality: outputDimensionality,
	}
}

func cloneInt32Ptr(v *int32) *int32 {
	if v == nil {
		return nil
	}
	clone := *v
	return &clone
}

func intToInt32Ptr(v int) (*int32, error) {
	if v <= 0 {
		return nil, errors.New("dimension must be greater than 0")
	}
	if v > math.MaxInt32 {
		return nil, errors.Errorf("dimension must be <= %d", math.MaxInt32)
	}
	conv := int32(v)
	return &conv, nil
}

func outputDimensionalityFromContext(ctx context.Context, fallback *int32) (*int32, error) {
	val := ctx.Value(dimensionContextKey)
	if val == nil {
		return cloneInt32Ptr(fallback), nil
	}
	switch d := val.(type) {
	case int:
		return intToInt32Ptr(d)
	case *int:
		// Backward compatibility for callers that previously stored *int manually.
		if d == nil {
			return nil, nil
		}
		return intToInt32Ptr(*d)
	default:
		return nil, errors.Errorf("dimension context value must be int, got %T", val)
	}
}

// Close is a no-op for the genai SDK client, which doesn't require cleanup.
func (c *Client) Close() error {
	return nil
}

var _ embeddings.EmbeddingFunction = (*GeminiEmbeddingFunction)(nil)
var _ embeddings.Closeable = (*GeminiEmbeddingFunction)(nil)

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

// Close implements the Closeable interface; currently this is a no-op.
func (e *GeminiEmbeddingFunction) Close() error {
	return e.apiClient.Close()
}

func (e *GeminiEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]embeddings.Embedding, error) {
	if e.apiClient.MaxBatchSize > 0 && len(documents) > e.apiClient.MaxBatchSize {
		return nil, errors.Errorf("number of documents exceeds the maximum batch size %v", e.apiClient.MaxBatchSize)
	}
	if len(documents) == 0 {
		return embeddings.NewEmptyEmbeddings(), nil
	}

	response, err := e.apiClient.CreateEmbedding(ctx, documents)
	if err != nil {
		return nil, errors.Wrap(err, "failed to embed documents")
	}
	return response, nil
}

func (e *GeminiEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (embeddings.Embedding, error) {
	response, err := e.apiClient.CreateEmbedding(ctx, []string{document})
	if err != nil {
		return nil, errors.Wrap(err, "failed to embed query")
	}
	if len(response) == 0 {
		return nil, errors.New("no embedding returned from Gemini API")
	}
	return response[0], nil
}

func (e *GeminiEmbeddingFunction) Name() string {
	return "google_genai"
}

func (e *GeminiEmbeddingFunction) GetConfig() embeddings.EmbeddingFunctionConfig {
	envVar := e.apiClient.APIKeyEnvVar
	if envVar == "" {
		envVar = APIKeyEnvVar
	}
	cfg := embeddings.EmbeddingFunctionConfig{
		"model_name":      string(e.apiClient.DefaultModel),
		"api_key_env_var": envVar,
	}
	if e.apiClient.DefaultTaskType != "" {
		cfg["task_type"] = string(e.apiClient.DefaultTaskType)
	}
	if e.apiClient.DefaultDimension != nil {
		cfg["dimension"] = int(*e.apiClient.DefaultDimension)
	}
	return cfg
}

func (e *GeminiEmbeddingFunction) DefaultSpace() embeddings.DistanceMetric {
	return embeddings.COSINE
}

func (e *GeminiEmbeddingFunction) SupportedSpaces() []embeddings.DistanceMetric {
	return []embeddings.DistanceMetric{embeddings.COSINE, embeddings.L2, embeddings.IP}
}

// NewGeminiEmbeddingFunctionFromConfig creates a Gemini embedding function from a config map.
// Uses schema-compliant field names: api_key_env_var, model_name, task_type, dimension.
func NewGeminiEmbeddingFunctionFromConfig(cfg embeddings.EmbeddingFunctionConfig) (*GeminiEmbeddingFunction, error) {
	envVar, ok := cfg["api_key_env_var"].(string)
	if !ok || envVar == "" {
		return nil, errors.New("api_key_env_var is required in config")
	}
	opts := []Option{WithAPIKeyFromEnvVar(envVar)}
	if model, ok := cfg["model_name"].(string); ok && model != "" {
		opts = append(opts, WithDefaultModel(embeddings.EmbeddingModel(model)))
	}
	if taskTypeRaw, exists := cfg["task_type"]; exists {
		taskType, ok := taskTypeRaw.(string)
		if !ok {
			return nil, errors.New("task_type must be a string")
		}
		opts = append(opts, WithTaskType(TaskType(taskType)))
	}
	if _, exists := cfg["dimension"]; exists {
		dim, ok := embeddings.ConfigInt(cfg, "dimension")
		if !ok {
			return nil, errors.New("dimension must be an integer")
		}
		opts = append(opts, WithDimension(dim))
	}
	return NewGeminiEmbeddingFunction(opts...)
}

func init() {
	if err := embeddings.RegisterDense("google_genai", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.EmbeddingFunction, error) {
		return NewGeminiEmbeddingFunctionFromConfig(cfg)
	}); err != nil {
		panic(err)
	}
}
