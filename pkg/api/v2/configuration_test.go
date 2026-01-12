package v2

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func TestNewCollectionConfiguration(t *testing.T) {
	config := NewCollectionConfiguration()
	assert.NotNil(t, config)
	assert.NotNil(t, config.raw)
}

func TestNewCollectionConfigurationFromMap(t *testing.T) {
	rawData := map[string]interface{}{
		"foo": "bar",
		"baz": 123,
	}

	config := NewCollectionConfigurationFromMap(rawData)
	assert.NotNil(t, config)

	val, ok := config.GetRaw("foo")
	assert.True(t, ok)
	assert.Equal(t, "bar", val)

	val, ok = config.GetRaw("baz")
	assert.True(t, ok)
	assert.Equal(t, 123, val)
}

func TestCollectionConfiguration_GetSetRaw(t *testing.T) {
	config := NewCollectionConfiguration()

	// Test SetRaw and GetRaw
	config.SetRaw("key1", "value1")
	val, ok := config.GetRaw("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	// Test non-existent key
	val, ok = config.GetRaw("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestCollectionConfiguration_Keys(t *testing.T) {
	config := NewCollectionConfiguration()

	// Initially empty
	keys := config.Keys()
	assert.Equal(t, 0, len(keys))

	// Add some values
	config.SetRaw("key1", "value1")
	config.SetRaw("key2", "value2")

	keys = config.Keys()
	assert.Equal(t, 2, len(keys))
	assert.Contains(t, keys, "key1")
	assert.Contains(t, keys, "key2")
}

func TestCollectionConfiguration_MarshalJSON(t *testing.T) {
	config := NewCollectionConfiguration()
	config.SetRaw("custom_key", "custom_value")

	data, err := json.Marshal(config)
	require.NoError(t, err)
	assert.NotNil(t, data)

	// Verify JSON structure
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)
	assert.Contains(t, result, "custom_key")
	assert.Equal(t, "custom_value", result["custom_key"])
}

func TestCollectionConfiguration_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"custom_key": "custom_value",
		"another_key": 42
	}`

	config := &CollectionConfigurationImpl{}
	err := json.Unmarshal([]byte(jsonData), config)
	require.NoError(t, err)

	val, ok := config.GetRaw("custom_key")
	assert.True(t, ok)
	assert.Equal(t, "custom_value", val)

	val, ok = config.GetRaw("another_key")
	assert.True(t, ok)
	// JSON numbers are decoded as float64
	assert.Equal(t, float64(42), val)
}

// mockEmbeddingFunction is a test EF for configuration testing
type mockEmbeddingFunction struct {
	name   string
	config embeddings.EmbeddingFunctionConfig
}

func (m *mockEmbeddingFunction) EmbedDocuments(ctx context.Context, texts []string) ([]embeddings.Embedding, error) {
	return nil, nil
}

func (m *mockEmbeddingFunction) EmbedQuery(ctx context.Context, text string) (embeddings.Embedding, error) {
	return nil, nil
}

func (m *mockEmbeddingFunction) Name() string {
	return m.name
}

func (m *mockEmbeddingFunction) GetConfig() embeddings.EmbeddingFunctionConfig {
	return m.config
}

func (m *mockEmbeddingFunction) DefaultSpace() embeddings.DistanceMetric {
	return embeddings.L2
}

func (m *mockEmbeddingFunction) SupportedSpaces() []embeddings.DistanceMetric {
	return []embeddings.DistanceMetric{embeddings.L2, embeddings.COSINE}
}

func TestEmbeddingFunctionInfo_IsKnown(t *testing.T) {
	tests := []struct {
		name     string
		info     *EmbeddingFunctionInfo
		expected bool
	}{
		{
			name:     "nil info",
			info:     nil,
			expected: false,
		},
		{
			name:     "known type",
			info:     &EmbeddingFunctionInfo{Type: "known", Name: "openai"},
			expected: true,
		},
		{
			name:     "unknown type",
			info:     &EmbeddingFunctionInfo{Type: "unknown", Name: "custom"},
			expected: false,
		},
		{
			name:     "empty type",
			info:     &EmbeddingFunctionInfo{Type: "", Name: "test"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.info.IsKnown())
		})
	}
}

func TestCollectionConfiguration_GetEmbeddingFunctionInfo(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		config := &CollectionConfigurationImpl{raw: nil}
		info, ok := config.GetEmbeddingFunctionInfo()
		assert.False(t, ok)
		assert.Nil(t, info)
	})

	t.Run("no embedding_function key", func(t *testing.T) {
		config := NewCollectionConfiguration()
		config.SetRaw("other_key", "value")
		info, ok := config.GetEmbeddingFunctionInfo()
		assert.False(t, ok)
		assert.Nil(t, info)
	})

	t.Run("embedding_function is not a map", func(t *testing.T) {
		config := NewCollectionConfiguration()
		config.SetRaw("embedding_function", "not_a_map")
		info, ok := config.GetEmbeddingFunctionInfo()
		assert.False(t, ok)
		assert.Nil(t, info)
	})

	t.Run("valid embedding_function", func(t *testing.T) {
		config := NewCollectionConfiguration()
		config.SetRaw("embedding_function", map[string]interface{}{
			"type":   "known",
			"name":   "openai",
			"config": map[string]interface{}{"api_key_env_var": "OPENAI_API_KEY", "model_name": "text-embedding-3-small"},
		})
		info, ok := config.GetEmbeddingFunctionInfo()
		assert.True(t, ok)
		require.NotNil(t, info)
		assert.Equal(t, "known", info.Type)
		assert.Equal(t, "openai", info.Name)
		assert.Equal(t, "OPENAI_API_KEY", info.Config["api_key_env_var"])
		assert.Equal(t, "text-embedding-3-small", info.Config["model_name"])
	})

	t.Run("partial embedding_function", func(t *testing.T) {
		config := NewCollectionConfiguration()
		config.SetRaw("embedding_function", map[string]interface{}{
			"type": "known",
			"name": "default",
		})
		info, ok := config.GetEmbeddingFunctionInfo()
		assert.True(t, ok)
		require.NotNil(t, info)
		assert.Equal(t, "known", info.Type)
		assert.Equal(t, "default", info.Name)
		assert.Nil(t, info.Config)
	})
}

func TestCollectionConfiguration_SetEmbeddingFunctionInfo(t *testing.T) {
	t.Run("nil info does nothing", func(t *testing.T) {
		config := NewCollectionConfiguration()
		config.SetEmbeddingFunctionInfo(nil)
		_, ok := config.GetRaw("embedding_function")
		assert.False(t, ok)
	})

	t.Run("sets info correctly", func(t *testing.T) {
		config := NewCollectionConfiguration()
		info := &EmbeddingFunctionInfo{
			Type:   "known",
			Name:   "cohere",
			Config: map[string]interface{}{"api_key_env_var": "COHERE_API_KEY"},
		}
		config.SetEmbeddingFunctionInfo(info)

		retrieved, ok := config.GetEmbeddingFunctionInfo()
		assert.True(t, ok)
		require.NotNil(t, retrieved)
		assert.Equal(t, "known", retrieved.Type)
		assert.Equal(t, "cohere", retrieved.Name)
		assert.Equal(t, "COHERE_API_KEY", retrieved.Config["api_key_env_var"])
	})

	t.Run("initializes raw map if nil", func(t *testing.T) {
		config := &CollectionConfigurationImpl{raw: nil}
		info := &EmbeddingFunctionInfo{Type: "known", Name: "test"}
		config.SetEmbeddingFunctionInfo(info)

		retrieved, ok := config.GetEmbeddingFunctionInfo()
		assert.True(t, ok)
		assert.NotNil(t, retrieved)
	})
}

func TestCollectionConfiguration_SetEmbeddingFunction(t *testing.T) {
	t.Run("nil EF does nothing", func(t *testing.T) {
		config := NewCollectionConfiguration()
		config.SetEmbeddingFunction(nil)
		_, ok := config.GetRaw("embedding_function")
		assert.False(t, ok)
	})

	t.Run("sets EF from interface", func(t *testing.T) {
		config := NewCollectionConfiguration()
		mockEF := &mockEmbeddingFunction{
			name: "mock_provider",
			config: embeddings.EmbeddingFunctionConfig{
				"api_key_env_var": "MOCK_API_KEY",
				"model_name":      "mock-model",
			},
		}
		config.SetEmbeddingFunction(mockEF)

		info, ok := config.GetEmbeddingFunctionInfo()
		assert.True(t, ok)
		require.NotNil(t, info)
		assert.Equal(t, "known", info.Type)
		assert.Equal(t, "mock_provider", info.Name)
		assert.Equal(t, "MOCK_API_KEY", info.Config["api_key_env_var"])
		assert.Equal(t, "mock-model", info.Config["model_name"])
	})
}

func TestBuildEmbeddingFunctionFromConfig(t *testing.T) {
	t.Run("nil config returns nil", func(t *testing.T) {
		ef, err := BuildEmbeddingFunctionFromConfig(nil)
		assert.NoError(t, err)
		assert.Nil(t, ef)
	})

	t.Run("no EF info returns nil", func(t *testing.T) {
		config := NewCollectionConfiguration()
		ef, err := BuildEmbeddingFunctionFromConfig(config)
		assert.NoError(t, err)
		assert.Nil(t, ef)
	})

	t.Run("unknown type returns nil", func(t *testing.T) {
		config := NewCollectionConfiguration()
		config.SetEmbeddingFunctionInfo(&EmbeddingFunctionInfo{
			Type: "unknown",
			Name: "custom",
		})
		ef, err := BuildEmbeddingFunctionFromConfig(config)
		assert.NoError(t, err)
		assert.Nil(t, ef)
	})

	t.Run("unregistered name returns nil", func(t *testing.T) {
		config := NewCollectionConfiguration()
		config.SetEmbeddingFunctionInfo(&EmbeddingFunctionInfo{
			Type: "known",
			Name: "not_registered_provider_xyz",
		})
		ef, err := BuildEmbeddingFunctionFromConfig(config)
		assert.NoError(t, err)
		assert.Nil(t, ef)
	})
}

func TestEmbeddingFunctionInfo_JSONRoundTrip(t *testing.T) {
	original := &EmbeddingFunctionInfo{
		Type: "known",
		Name: "openai",
		Config: map[string]interface{}{
			"api_key_env_var": "OPENAI_API_KEY",
			"model_name":      "text-embedding-3-small",
			"dimensions":      float64(1536),
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded EmbeddingFunctionInfo
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Type, decoded.Type)
	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Config["api_key_env_var"], decoded.Config["api_key_env_var"])
	assert.Equal(t, original.Config["model_name"], decoded.Config["model_name"])
	assert.Equal(t, original.Config["dimensions"], decoded.Config["dimensions"])
}

func TestCollectionConfiguration_EFConfigFromServerResponse(t *testing.T) {
	// Simulates parsing a server response with EF config
	serverResponse := `{
		"hnsw": {
			"space": "l2",
			"ef_construction": 100
		},
		"embedding_function": {
			"type": "known",
			"name": "default",
			"config": {}
		}
	}`

	config := &CollectionConfigurationImpl{}
	err := json.Unmarshal([]byte(serverResponse), config)
	require.NoError(t, err)

	info, ok := config.GetEmbeddingFunctionInfo()
	assert.True(t, ok)
	require.NotNil(t, info)
	assert.Equal(t, "known", info.Type)
	assert.Equal(t, "default", info.Name)
	assert.True(t, info.IsKnown())
}
