package gemini

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithTaskType(t *testing.T) {
	client := &Client{}

	require.NoError(t, WithTaskType("RETRIEVAL_DOCUMENT")(client))
	assert.Equal(t, "RETRIEVAL_DOCUMENT", client.DefaultTaskType)

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
	cfg = buildEmbedContentConfig("RETRIEVAL_QUERY", &dim)
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
			DefaultTaskType:  "RETRIEVAL_DOCUMENT",
			DefaultDimension: &dim,
		},
	}

	cfg := ef.GetConfig()
	assert.Equal(t, APIKeyEnvVar, cfg["api_key_env_var"])
	assert.Equal(t, string(DefaultEmbeddingModel), cfg["model_name"])
	assert.Equal(t, "RETRIEVAL_DOCUMENT", cfg["task_type"])
	assert.Equal(t, 256, cfg["dimension"])
}
