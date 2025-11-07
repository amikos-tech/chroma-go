package v2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func TestNewSchema(t *testing.T) {
	schema := NewSchema()
	assert.NotNil(t, schema)
	assert.Equal(t, 0, len(schema.defaults))
	assert.Equal(t, 0, len(schema.keyOverrides))
}

func TestNewSchemaWithDefaults(t *testing.T) {
	schema := NewSchemaWithDefaults()
	assert.NotNil(t, schema)

	// Check default vector index exists
	vectorConfig, ok := schema.GetDefault("VectorValue")
	assert.True(t, ok)
	assert.NotNil(t, vectorConfig)
	assert.Equal(t, "VectorIndex", vectorConfig.IndexType())

	// Check default FTS index exists
	ftsConfig, ok := schema.GetDefault("DocumentValue")
	assert.True(t, ok)
	assert.NotNil(t, ftsConfig)
	assert.Equal(t, "FTS", ftsConfig.IndexType())
}

func TestSchema_SetDefault(t *testing.T) {
	schema := NewSchema()

	// Test setting a vector index config
	vectorConfig := &VectorIndexConfig{
		Space: embeddings.COSINE,
		HnswConfig: &HnswIndexConfig{
			M:              16,
			ConstructionEF: 100,
			SearchEF:       10,
		},
	}

	err := schema.SetDefault("VectorValue", vectorConfig)
	require.NoError(t, err)

	retrieved, ok := schema.GetDefault("VectorValue")
	assert.True(t, ok)
	assert.Equal(t, vectorConfig, retrieved)

	// Test error cases
	err = schema.SetDefault("", vectorConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "value type cannot be empty")

	err = schema.SetDefault("VectorValue", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestSchema_CreateIndex(t *testing.T) {
	schema := NewSchema()

	// Test creating an index for a key
	stringIndexConfig := &StringInvertedIndexConfig{}
	err := schema.CreateIndex("my_field", stringIndexConfig)
	require.NoError(t, err)

	// Verify the index was created
	retrieved, ok := schema.GetIndexForKey("my_field", "StringValue", "InvertedIndex")
	assert.True(t, ok)
	assert.Equal(t, stringIndexConfig, retrieved)

	// Test error cases
	err = schema.CreateIndex("", stringIndexConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key cannot be empty")

	err = schema.CreateIndex("my_field", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestSchema_DeleteIndex(t *testing.T) {
	schema := NewSchema()

	// Create an index first
	stringIndexConfig := &StringInvertedIndexConfig{}
	err := schema.CreateIndex("my_field", stringIndexConfig)
	require.NoError(t, err)

	// Delete the index
	err = schema.DeleteIndex("my_field", "StringValue", "InvertedIndex")
	require.NoError(t, err)

	// Verify it's gone
	_, ok := schema.GetIndexForKey("my_field", "StringValue", "InvertedIndex")
	assert.False(t, ok)

	// Test deleting non-existent index
	err = schema.DeleteIndex("nonexistent", "StringValue", "InvertedIndex")
	assert.Error(t, err)
}

func TestSchema_GetAllIndexesForKey(t *testing.T) {
	schema := NewSchema()

	// Create multiple indexes for a key
	err := schema.CreateIndex("my_field", &StringInvertedIndexConfig{})
	require.NoError(t, err)

	indexes := schema.GetAllIndexesForKey("my_field")
	assert.NotNil(t, indexes)
	assert.Equal(t, 1, len(indexes))
	assert.NotNil(t, indexes["StringValue"])
	assert.Equal(t, 1, len(indexes["StringValue"]))

	// Test non-existent key
	indexes = schema.GetAllIndexesForKey("nonexistent")
	assert.Nil(t, indexes)
}

func TestSchema_Keys(t *testing.T) {
	schema := NewSchema()

	// Initially empty
	keys := schema.Keys()
	assert.Equal(t, 0, len(keys))

	// Add some indexes
	err := schema.CreateIndex("field1", &StringInvertedIndexConfig{})
	require.NoError(t, err)
	err = schema.CreateIndex("field2", &IntInvertedIndexConfig{})
	require.NoError(t, err)

	keys = schema.Keys()
	assert.Equal(t, 2, len(keys))
	assert.Contains(t, keys, "field1")
	assert.Contains(t, keys, "field2")
}

func TestIndexConfig_Types(t *testing.T) {
	tests := []struct {
		name          string
		config        IndexConfig
		expectedType  string
		expectedValue string
	}{
		{
			name:          "VectorIndexConfig",
			config:        &VectorIndexConfig{},
			expectedType:  "VectorIndex",
			expectedValue: "VectorValue",
		},
		{
			name:          "HnswIndexConfig",
			config:        &HnswIndexConfig{},
			expectedType:  "HNSW",
			expectedValue: "VectorValue",
		},
		{
			name:          "SpannIndexConfig",
			config:        &SpannIndexConfig{},
			expectedType:  "SPANN",
			expectedValue: "VectorValue",
		},
		{
			name:          "FtsIndexConfig",
			config:        &FtsIndexConfig{},
			expectedType:  "FTS",
			expectedValue: "DocumentValue",
		},
		{
			name:          "SparseVectorIndexConfig",
			config:        &SparseVectorIndexConfig{},
			expectedType:  "SparseVectorIndex",
			expectedValue: "SparseVectorValue",
		},
		{
			name:          "StringInvertedIndexConfig",
			config:        &StringInvertedIndexConfig{},
			expectedType:  "InvertedIndex",
			expectedValue: "StringValue",
		},
		{
			name:          "IntInvertedIndexConfig",
			config:        &IntInvertedIndexConfig{},
			expectedType:  "InvertedIndex",
			expectedValue: "IntValue",
		},
		{
			name:          "FloatInvertedIndexConfig",
			config:        &FloatInvertedIndexConfig{},
			expectedType:  "InvertedIndex",
			expectedValue: "FloatValue",
		},
		{
			name:          "BoolInvertedIndexConfig",
			config:        &BoolInvertedIndexConfig{},
			expectedType:  "InvertedIndex",
			expectedValue: "BoolValue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedType, tt.config.IndexType())
			assert.Equal(t, tt.expectedValue, tt.config.ValueType())
		})
	}
}

func TestSchema_MarshalJSON(t *testing.T) {
	schema := NewSchemaWithDefaults()

	// Add a custom index
	err := schema.CreateIndex("my_field", &StringInvertedIndexConfig{})
	require.NoError(t, err)

	data, err := json.Marshal(schema)
	require.NoError(t, err)
	assert.NotNil(t, data)

	// Verify JSON contains expected fields
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)
	assert.Contains(t, result, "defaults")
	assert.Contains(t, result, "key_overrides")
}

func TestVectorIndexConfig(t *testing.T) {
	config := &VectorIndexConfig{
		Space: embeddings.L2,
		HnswConfig: &HnswIndexConfig{
			M:              16,
			ConstructionEF: 100,
			SearchEF:       10,
			NumThreads:     4,
			ResizeFactor:   1.2,
		},
	}

	assert.Equal(t, embeddings.L2, config.Space)
	assert.NotNil(t, config.HnswConfig)
	assert.Equal(t, 16, config.HnswConfig.M)
	assert.Equal(t, 100, config.HnswConfig.ConstructionEF)
	assert.Equal(t, 10, config.HnswConfig.SearchEF)
	assert.Equal(t, 4, config.HnswConfig.NumThreads)
	assert.Equal(t, 1.2, config.HnswConfig.ResizeFactor)
}

func TestFtsIndexConfig(t *testing.T) {
	config := &FtsIndexConfig{
		Tokenizer: "whitespace",
	}

	assert.Equal(t, "whitespace", config.Tokenizer)
	assert.Equal(t, "FTS", config.IndexType())
	assert.Equal(t, "DocumentValue", config.ValueType())
}
