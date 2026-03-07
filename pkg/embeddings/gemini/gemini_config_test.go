package gemini

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func TestWithTaskType(t *testing.T) {
	client := &Client{}

	require.NoError(t, WithTaskType(TaskTypeRetrievalDocument)(client))
	assert.Equal(t, TaskTypeRetrievalDocument, client.DefaultTaskType)

	require.Error(t, WithTaskType("")(client))
}

func TestWithDimension(t *testing.T) {
	client := &Client{}

	require.NoError(t, WithDimension(768)(client))
	require.NotNil(t, client.DefaultDimension)
	assert.Equal(t, int32(768), *client.DefaultDimension)

	require.Error(t, WithDimension(0)(client))
	require.Error(t, WithDimension(-1)(client))
	require.Error(t, WithDimension(math.MaxInt32+1)(client))
}

func TestBuildEmbedContentConfig(t *testing.T) {
	cfg := buildEmbedContentConfig("", nil)
	assert.Nil(t, cfg)

	dim := int32(512)
	cfg = buildEmbedContentConfig(TaskTypeRetrievalDocument, nil)
	require.NotNil(t, cfg)
	assert.Equal(t, "RETRIEVAL_DOCUMENT", cfg.TaskType)
	assert.Nil(t, cfg.OutputDimensionality)

	cfg = buildEmbedContentConfig(TaskType(""), &dim)
	require.NotNil(t, cfg)
	assert.Equal(t, "", cfg.TaskType)
	require.NotNil(t, cfg.OutputDimensionality)
	assert.Equal(t, int32(512), *cfg.OutputDimensionality)

	cfg = buildEmbedContentConfig(TaskTypeRetrievalQuery, &dim)
	require.NotNil(t, cfg)
	assert.Equal(t, "RETRIEVAL_QUERY", cfg.TaskType)
	require.NotNil(t, cfg.OutputDimensionality)
	assert.Equal(t, int32(512), *cfg.OutputDimensionality)
}

func TestGeminiGetConfigIncludesTaskTypeAndDimension(t *testing.T) {
	dim := int32(256)
	ef := &GeminiEmbeddingFunction{
		apiClient: &Client{
			APIKeyEnvVar:     APIKeyEnvVar,
			DefaultModel:     DefaultEmbeddingModel,
			DefaultTaskType:  TaskTypeRetrievalDocument,
			DefaultDimension: &dim,
		},
	}

	cfg := ef.GetConfig()
	assert.Equal(t, APIKeyEnvVar, cfg["api_key_env_var"])
	assert.Equal(t, string(DefaultEmbeddingModel), cfg["model_name"])
	assert.Equal(t, "RETRIEVAL_DOCUMENT", cfg["task_type"])
	assert.Equal(t, 256, cfg["dimension"])
}

func TestGeminiGetConfigOmitsUnsetOptionalFields(t *testing.T) {
	ef := &GeminiEmbeddingFunction{
		apiClient: &Client{
			APIKeyEnvVar: APIKeyEnvVar,
			DefaultModel: DefaultEmbeddingModel,
		},
	}

	cfg := ef.GetConfig()
	_, hasTaskType := cfg["task_type"]
	_, hasDimension := cfg["dimension"]
	assert.False(t, hasTaskType)
	assert.False(t, hasDimension)
}

func TestIntToInt32PtrValidation(t *testing.T) {
	_, err := intToInt32Ptr(0)
	require.Error(t, err)
	_, err = intToInt32Ptr(-1)
	require.Error(t, err)
	_, err = intToInt32Ptr(math.MaxInt32 + 1)
	require.Error(t, err)

	v, err := intToInt32Ptr(1024)
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, int32(1024), *v)
}

func TestOutputDimensionalityFromContextValidation(t *testing.T) {
	fallback := int32(128)
	emptyCtx := context.Background()
	v, err := outputDimensionalityFromContext(emptyCtx, &fallback)
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, int32(128), *v)

	ctx := ContextWithDimension(context.Background(), 256)
	v, err = outputDimensionalityFromContext(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, int32(256), *v)

	ctx = ContextWithDimension(context.Background(), math.MaxInt32+1)
	_, err = outputDimensionalityFromContext(ctx, nil)
	require.Error(t, err)

	badCtx := context.WithValue(context.Background(), dimensionContextKey, "256")
	_, err = outputDimensionalityFromContext(badCtx, nil)
	require.Error(t, err)
}

func TestNewGeminiEmbeddingFunctionFromConfig_InvalidTypes(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")

	_, err := NewGeminiEmbeddingFunctionFromConfig(embeddings.EmbeddingFunctionConfig{
		"api_key_env_var": "GEMINI_API_KEY",
		"model_name":      "gemini-embedding-001",
		"task_type":       123,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task_type must be a string")

	_, err = NewGeminiEmbeddingFunctionFromConfig(embeddings.EmbeddingFunctionConfig{
		"api_key_env_var": "GEMINI_API_KEY",
		"model_name":      "gemini-embedding-001",
		"dimension":       "768",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dimension must be an integer")
}
