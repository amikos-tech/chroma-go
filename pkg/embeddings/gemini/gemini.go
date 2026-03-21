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

// ContextWithModel sets a model override in context.
// Validation happens when the request is built in CreateEmbedding.
func ContextWithModel(ctx context.Context, model string) context.Context {
	return context.WithValue(ctx, modelContextKey, model)
}

// ContextWithTaskType sets a task type override in context.
// Validation happens when the request is built in CreateEmbedding.
func ContextWithTaskType(ctx context.Context, taskType TaskType) context.Context {
	return context.WithValue(ctx, taskTypeContextKey, taskType)
}

// ContextWithDimension sets an output dimensionality override in context.
// Validation happens when the request is built in CreateEmbedding.
func ContextWithDimension(ctx context.Context, dimension int) context.Context {
	return context.WithValue(ctx, dimensionContextKey, dimension)
}

const (
	DefaultEmbeddingModel = "gemini-embedding-2-preview"
	LegacyEmbeddingModel  = "gemini-embedding-001"
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
	MaxFileSize      int64
}

func applyDefaults(c *Client) (err error) {
	if c.DefaultModel == "" {
		c.DefaultModel = DefaultEmbeddingModel
	}

	if c.DefaultContext == nil {
		ctx := context.Background()
		c.DefaultContext = &ctx
	}

	if c.MaxFileSize == 0 {
		c.MaxFileSize = 100 * 1024 * 1024 // 100 MB — matches Gemini API inline payload limit
	}

	if c.MaxBatchSize == 0 {
		c.MaxBatchSize = 250
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
	model, err := modelFromContext(ctx, string(c.DefaultModel))
	if err != nil {
		return nil, errors.Wrap(err, "invalid model override")
	}
	taskType, err := taskTypeFromContext(ctx, c.DefaultTaskType)
	if err != nil {
		return nil, errors.Wrap(err, "invalid task_type override")
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

// CreateContentEmbedding embeds multimodal content items using the Gemini API.
// For a single item, ProviderHints and intent are honoured per-item via resolveTaskTypeForContent.
// For batches, the default task type is used for all items (Gemini applies one config per batch).
func (c *Client) CreateContentEmbedding(ctx context.Context, contents []embeddings.Content, mapper embeddings.IntentMapper) ([]embeddings.Embedding, error) {
	model, err := modelFromContext(ctx, string(c.DefaultModel))
	if err != nil {
		return nil, errors.Wrap(err, "invalid model override")
	}
	defaultTaskType, err := taskTypeFromContext(ctx, c.DefaultTaskType)
	if err != nil {
		return nil, errors.Wrap(err, "invalid task_type override")
	}
	contextDim := ctx.Value(dimensionContextKey)
	outputDimensionality, err := outputDimensionalityFromContext(ctx, c.DefaultDimension)
	if err != nil {
		return nil, errors.Wrap(err, "invalid dimension override")
	}

	taskType := defaultTaskType
	if len(contents) == 1 {
		taskType, err = resolveTaskTypeForContent(contents[0], defaultTaskType, mapper)
		if err != nil {
			return nil, errors.Wrap(err, "failed to resolve task type for content")
		}
		// Per-content dimension overrides client default, but not an explicit context override.
		if contents[0].Dimension != nil && contextDim == nil {
			dim, dimErr := intToInt32Ptr(*contents[0].Dimension)
			if dimErr != nil {
				return nil, errors.Wrap(dimErr, "invalid content dimension")
			}
			outputDimensionality = dim
		}
	}

	genaiContents, err := convertToGenaiContents(ctx, contents, c.MaxFileSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert content to Gemini format")
	}

	res, err := c.Client.Models.EmbedContent(ctx, model, genaiContents, buildEmbedContentConfig(taskType, outputDimensionality))
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
	d, ok := val.(int)
	if !ok {
		return nil, errors.Errorf("dimension context value must be int, got %T", val)
	}
	return intToInt32Ptr(d)
}

func taskTypeFromContext(ctx context.Context, fallback TaskType) (TaskType, error) {
	val := ctx.Value(taskTypeContextKey)
	if val == nil {
		if fallback == "" {
			return "", nil
		}
		if !fallback.IsValid() {
			return "", errors.Errorf("invalid task type: %q", fallback)
		}
		return fallback, nil
	}
	taskType, ok := val.(TaskType)
	if !ok {
		return "", errors.Errorf("task_type context value must be TaskType, got %T", val)
	}
	if taskType == "" {
		return "", errors.New("task type cannot be empty")
	}
	if !taskType.IsValid() {
		return "", errors.Errorf("invalid task type: %q", taskType)
	}
	return taskType, nil
}

func modelFromContext(ctx context.Context, fallback string) (string, error) {
	val := ctx.Value(modelContextKey)
	if val == nil {
		if fallback == "" {
			return "", errors.New("model cannot be empty")
		}
		return fallback, nil
	}
	model, ok := val.(string)
	if !ok {
		return "", errors.Errorf("model context value must be string, got %T", val)
	}
	if model == "" {
		return "", errors.New("model cannot be empty")
	}
	return model, nil
}

// Close is a no-op for the genai SDK client, which doesn't require cleanup.
func (c *Client) Close() error {
	return nil
}

var _ embeddings.EmbeddingFunction = (*GeminiEmbeddingFunction)(nil)
var _ embeddings.Closeable = (*GeminiEmbeddingFunction)(nil)
var _ embeddings.ContentEmbeddingFunction = (*GeminiEmbeddingFunction)(nil)
var _ embeddings.CapabilityAware = (*GeminiEmbeddingFunction)(nil)
var _ embeddings.IntentMapper = (*GeminiEmbeddingFunction)(nil)

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

// Capabilities returns the capability metadata for the configured Gemini model.
func (e *GeminiEmbeddingFunction) Capabilities() embeddings.CapabilityMetadata {
	return capabilitiesForModel(string(e.apiClient.DefaultModel))
}

// capabilitiesForContext returns capabilities for the effective model,
// honoring any model override set in the context.
func (e *GeminiEmbeddingFunction) capabilitiesForContext(ctx context.Context) embeddings.CapabilityMetadata {
	model, err := modelFromContext(ctx, string(e.apiClient.DefaultModel))
	if err != nil {
		return capabilitiesForModel(string(e.apiClient.DefaultModel))
	}
	return capabilitiesForModel(model)
}

// MapIntent translates a neutral shared intent to a Gemini task type string.
// Only the 5 neutral intents are accepted; provider-native intents should use ProviderHints["task_type"].
func (e *GeminiEmbeddingFunction) MapIntent(intent embeddings.Intent) (string, error) {
	if !embeddings.IsNeutralIntent(intent) {
		return "", errors.Errorf("unsupported intent %q: use ProviderHints[\"task_type\"] for Gemini-native task types", intent)
	}
	tt, ok := neutralIntentToTaskType[intent]
	if !ok {
		return "", errors.Errorf("intent %q has no Gemini task type mapping", intent)
	}
	return string(tt), nil
}

// EmbedContent embeds a single multimodal content item using the shared Content API.
func (e *GeminiEmbeddingFunction) EmbedContent(ctx context.Context, content embeddings.Content) (embeddings.Embedding, error) {
	if err := content.Validate(); err != nil {
		return nil, err
	}
	caps := e.capabilitiesForContext(ctx)
	if err := embeddings.ValidateContentSupport(content, caps); err != nil {
		return nil, err
	}
	result, err := e.apiClient.CreateContentEmbedding(ctx, []embeddings.Content{content}, e)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, errors.New("no embedding returned")
	}
	return result[0], nil
}

// EmbedContents embeds a batch of multimodal content items using the shared Content API.
func (e *GeminiEmbeddingFunction) EmbedContents(ctx context.Context, contents []embeddings.Content) ([]embeddings.Embedding, error) {
	if err := embeddings.ValidateContents(contents); err != nil {
		return nil, err
	}
	if e.apiClient.MaxBatchSize > 0 && len(contents) > e.apiClient.MaxBatchSize {
		return nil, errors.Errorf("number of contents exceeds the maximum batch size %v", e.apiClient.MaxBatchSize)
	}
	caps := e.capabilitiesForContext(ctx)
	if err := embeddings.ValidateContentsSupport(contents, caps); err != nil {
		return nil, err
	}
	return e.apiClient.CreateContentEmbedding(ctx, contents, e)
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
	if taskTypeRaw, exists := cfg["task_type"]; exists && taskTypeRaw != nil {
		taskType, ok := taskTypeRaw.(string)
		if !ok {
			return nil, errors.New("task_type must be a string")
		}
		opts = append(opts, WithTaskType(TaskType(taskType)))
	}
	if dimRaw, exists := cfg["dimension"]; exists && dimRaw != nil {
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
	if err := embeddings.RegisterContent("google_genai", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.ContentEmbeddingFunction, error) {
		return NewGeminiEmbeddingFunctionFromConfig(cfg)
	}); err != nil {
		panic(err)
	}
}
