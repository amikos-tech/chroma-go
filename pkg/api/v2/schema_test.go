package v2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchema(t *testing.T) {
	schema, err := NewSchema()
	require.NoError(t, err)
	assert.NotNil(t, schema)
	assert.NotNil(t, schema.Defaults())
	assert.Equal(t, 0, len(schema.Keys()))
}

func TestNewSchemaWithDefaults(t *testing.T) {
	schema, err := NewSchemaWithDefaults()
	require.NoError(t, err)
	assert.NotNil(t, schema)

	// Check vector index exists on #embedding key with L2 space
	embeddingVT, ok := schema.GetKey(EmbeddingKey)
	assert.True(t, ok)
	assert.NotNil(t, embeddingVT.FloatList)
	assert.NotNil(t, embeddingVT.FloatList.VectorIndex)
	assert.True(t, embeddingVT.FloatList.VectorIndex.Enabled)
	assert.Equal(t, SpaceL2, embeddingVT.FloatList.VectorIndex.Config.Space)

	// Other indexes (FTS, string, int, float, bool) are enabled by default
	// in Chroma, so they don't need to be explicitly set in the schema
}

func TestNewSchema_WithOptions(t *testing.T) {
	schema, err := NewSchema(
		WithDefaultVectorIndex(NewVectorIndexConfig(
			WithSpace(SpaceCosine),
			WithHnsw(NewHnswConfig(
				WithEfConstruction(200),
				WithMaxNeighbors(32),
				WithEfSearch(20),
			)),
		)),
	)
	require.NoError(t, err)
	assert.NotNil(t, schema)

	// Verify vector config is on #embedding key
	embeddingVT, ok := schema.GetKey(EmbeddingKey)
	assert.True(t, ok)
	assert.NotNil(t, embeddingVT.FloatList)
	assert.NotNil(t, embeddingVT.FloatList.VectorIndex)
	assert.True(t, embeddingVT.FloatList.VectorIndex.Enabled)
	assert.Equal(t, SpaceCosine, embeddingVT.FloatList.VectorIndex.Config.Space)
	assert.Equal(t, uint(200), embeddingVT.FloatList.VectorIndex.Config.Hnsw.EfConstruction)
	assert.Equal(t, uint(32), embeddingVT.FloatList.VectorIndex.Config.Hnsw.MaxNeighbors)
	assert.Equal(t, uint(20), embeddingVT.FloatList.VectorIndex.Config.Hnsw.EfSearch)
}

func TestNewSchema_WithKeyOverrides(t *testing.T) {
	schema, err := NewSchema(
		WithDefaultVectorIndex(NewVectorIndexConfig(WithSpace(SpaceL2))),
		WithStringIndex("category"),
		WithIntIndex("price"),
	)
	require.NoError(t, err)
	assert.NotNil(t, schema)

	// Check keys were created (3 keys: #embedding, category, price)
	keys := schema.Keys()
	assert.Equal(t, 3, len(keys))
	assert.Contains(t, keys, EmbeddingKey)
	assert.Contains(t, keys, "category")
	assert.Contains(t, keys, "price")

	// Check category key has string inverted index
	categoryVT, ok := schema.GetKey("category")
	assert.True(t, ok)
	assert.NotNil(t, categoryVT.String)
	assert.NotNil(t, categoryVT.String.StringInvertedIndex)
	assert.True(t, categoryVT.String.StringInvertedIndex.Enabled)

	// Check price key has int inverted index
	priceVT, ok := schema.GetKey("price")
	assert.True(t, ok)
	assert.NotNil(t, priceVT.Int)
	assert.NotNil(t, priceVT.Int.IntInvertedIndex)
	assert.True(t, priceVT.Int.IntInvertedIndex.Enabled)
}

func TestNewSchema_ErrorHandling(t *testing.T) {
	// Test nil vector config
	_, err := NewSchema(WithDefaultVectorIndex(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vector index config cannot be nil")

	// Test nil sparse vector config
	_, err = NewSchema(WithDefaultSparseVectorIndex(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sparse vector index config cannot be nil")

	// Test empty key
	_, err = NewSchema(WithStringIndex(""))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key cannot be empty")
}

func TestHnswConfig_Options(t *testing.T) {
	config := NewHnswConfig(
		WithEfConstruction(200),
		WithMaxNeighbors(32),
		WithEfSearch(50),
		WithNumThreads(8),
		WithBatchSize(1000),
		WithSyncThreshold(2000),
		WithResizeFactor(1.5),
	)

	assert.Equal(t, uint(200), config.EfConstruction)
	assert.Equal(t, uint(32), config.MaxNeighbors)
	assert.Equal(t, uint(50), config.EfSearch)
	assert.Equal(t, uint(8), config.NumThreads)
	assert.Equal(t, uint(1000), config.BatchSize)
	assert.Equal(t, uint(2000), config.SyncThreshold)
	assert.Equal(t, 1.5, config.ResizeFactor)
}

func TestHnswConfig_Defaults(t *testing.T) {
	config, err := NewHnswConfigWithDefaults()
	require.NoError(t, err)

	assert.Equal(t, uint(100), config.EfConstruction)
	assert.Equal(t, uint(16), config.MaxNeighbors)
	assert.Equal(t, uint(100), config.EfSearch)
	assert.Equal(t, uint(1), config.NumThreads)
	assert.Equal(t, uint(100), config.BatchSize)
	assert.Equal(t, uint(1000), config.SyncThreshold)
	assert.Equal(t, 1.2, config.ResizeFactor)
}

func TestHnswConfig_DefaultsWithOverride(t *testing.T) {
	config, err := NewHnswConfigWithDefaults(
		WithEfConstruction(200),
		WithMaxNeighbors(32),
	)
	require.NoError(t, err)

	assert.Equal(t, uint(200), config.EfConstruction)
	assert.Equal(t, uint(32), config.MaxNeighbors)
	// Other values should be defaults
	assert.Equal(t, uint(100), config.EfSearch)
	assert.Equal(t, uint(1000), config.SyncThreshold)
}

func TestHnswConfig_ValidationRejectsInvalid(t *testing.T) {
	// BatchSize < 2 should fail
	_, err := NewHnswConfigWithDefaults(WithBatchSize(1))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")

	// SyncThreshold < 2 should fail
	_, err = NewHnswConfigWithDefaults(WithSyncThreshold(1))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")

	// BatchSize = 0 should fail
	_, err = NewHnswConfigWithDefaults(WithBatchSize(0))
	assert.Error(t, err)

	// SyncThreshold = 0 should fail
	_, err = NewHnswConfigWithDefaults(WithSyncThreshold(0))
	assert.Error(t, err)
}

func TestSpannConfig_Options(t *testing.T) {
	config := NewSpannConfig(
		WithSpannSearchNprobe(64),
		WithSpannSearchRngFactor(1.0),
		WithSpannSearchRngEpsilon(10.0),
		WithSpannNReplicaCount(8),
		WithSpannWriteRngFactor(1.0),
		WithSpannWriteRngEpsilon(5.0),
		WithSpannSplitThreshold(50),
		WithSpannNumSamplesKmeans(1000),
		WithSpannInitialLambda(100.0),
		WithSpannReassignNeighborCount(64),
		WithSpannMergeThreshold(25),
		WithSpannNumCentersToMergeTo(8),
		WithSpannWriteNprobe(32),
		WithSpannEfConstruction(200),
		WithSpannEfSearch(200),
		WithSpannMaxNeighbors(64),
	)

	assert.Equal(t, uint(64), config.SearchNprobe)
	assert.Equal(t, 1.0, config.SearchRngFactor)
	assert.Equal(t, 10.0, config.SearchRngEpsilon)
	assert.Equal(t, uint(8), config.NReplicaCount)
	assert.Equal(t, 1.0, config.WriteRngFactor)
	assert.Equal(t, 5.0, config.WriteRngEpsilon)
	assert.Equal(t, uint(50), config.SplitThreshold)
	assert.Equal(t, uint(1000), config.NumSamplesKmeans)
	assert.Equal(t, 100.0, config.InitialLambda)
	assert.Equal(t, uint(64), config.ReassignNeighborCount)
	assert.Equal(t, uint(25), config.MergeThreshold)
	assert.Equal(t, uint(8), config.NumCentersToMergeTo)
	assert.Equal(t, uint(32), config.WriteNprobe)
	assert.Equal(t, uint(200), config.EfConstruction)
	assert.Equal(t, uint(200), config.EfSearch)
	assert.Equal(t, uint(64), config.MaxNeighbors)
}

func TestVectorIndexConfig_Options(t *testing.T) {
	hnswCfg := NewHnswConfig(WithEfConstruction(100))
	config := NewVectorIndexConfig(
		WithSpace(SpaceIP),
		WithSourceKey(DocumentKey),
		WithHnsw(hnswCfg),
	)

	assert.Equal(t, SpaceIP, config.Space)
	assert.Equal(t, DocumentKey, config.SourceKey)
	assert.NotNil(t, config.Hnsw)
	assert.Equal(t, uint(100), config.Hnsw.EfConstruction)
}

func TestVectorIndexConfig_WithSpann(t *testing.T) {
	spannCfg := NewSpannConfig(
		WithSpannSearchNprobe(64),
		WithSpannEfConstruction(200),
	)
	config := NewVectorIndexConfig(
		WithSpace(SpaceCosine),
		WithSpann(spannCfg),
	)

	assert.Equal(t, SpaceCosine, config.Space)
	assert.NotNil(t, config.Spann)
	assert.Equal(t, uint(64), config.Spann.SearchNprobe)
	assert.Equal(t, uint(200), config.Spann.EfConstruction)
}

func TestSpannConfig_Defaults(t *testing.T) {
	config, err := NewSpannConfigWithDefaults()
	require.NoError(t, err)

	assert.Equal(t, uint(64), config.SearchNprobe)
	assert.Equal(t, 1.0, config.SearchRngFactor)
	assert.Equal(t, 10.0, config.SearchRngEpsilon)
	assert.Equal(t, uint(8), config.NReplicaCount)
	assert.Equal(t, 1.0, config.WriteRngFactor)
	assert.Equal(t, 5.0, config.WriteRngEpsilon)
	assert.Equal(t, uint(50), config.SplitThreshold)
	assert.Equal(t, uint(1000), config.NumSamplesKmeans)
	assert.Equal(t, 100.0, config.InitialLambda)
	assert.Equal(t, uint(64), config.ReassignNeighborCount)
	assert.Equal(t, uint(25), config.MergeThreshold)
	assert.Equal(t, uint(8), config.NumCentersToMergeTo)
	assert.Equal(t, uint(32), config.WriteNprobe)
	assert.Equal(t, uint(200), config.EfConstruction)
	assert.Equal(t, uint(200), config.EfSearch)
	assert.Equal(t, uint(64), config.MaxNeighbors)
}

func TestSpannConfig_DefaultsWithOverride(t *testing.T) {
	config, err := NewSpannConfigWithDefaults(
		WithSpannSearchNprobe(100),
		WithSpannMergeThreshold(50),
	)
	require.NoError(t, err)

	assert.Equal(t, uint(100), config.SearchNprobe)
	assert.Equal(t, uint(50), config.MergeThreshold)
	// Other values should be defaults
	assert.Equal(t, uint(64), config.MaxNeighbors)
}

func TestSpannConfig_ValidationRejectsInvalid(t *testing.T) {
	// SearchNprobe > 128 should fail
	_, err := NewSpannConfigWithDefaults(WithSpannSearchNprobe(200))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")

	// MergeThreshold < 25 should fail
	_, err = NewSpannConfigWithDefaults(WithSpannMergeThreshold(10))
	assert.Error(t, err)

	// MergeThreshold > 100 should fail
	_, err = NewSpannConfigWithDefaults(WithSpannMergeThreshold(150))
	assert.Error(t, err)

	// SplitThreshold < 50 should fail
	_, err = NewSpannConfigWithDefaults(WithSpannSplitThreshold(25))
	assert.Error(t, err)

	// NReplicaCount > 8 should fail
	_, err = NewSpannConfigWithDefaults(WithSpannNReplicaCount(10))
	assert.Error(t, err)
}

func TestSparseVectorIndexConfig_Options(t *testing.T) {
	config := NewSparseVectorIndexConfig(
		WithSparseSourceKey(DocumentKey),
		WithBM25(true),
	)

	assert.Equal(t, DocumentKey, config.SourceKey)
	assert.True(t, config.BM25)
}

func TestSchema_MultipleKeyOptions(t *testing.T) {
	schema, err := NewSchema(
		WithStringIndex("field1"),
		WithIntIndex("field2"),
		WithFloatIndex("field3"),
		WithBoolIndex("field4"),
		WithFtsIndex("field5"),
	)
	require.NoError(t, err)

	// Verify all keys were created
	keys := schema.Keys()
	assert.Equal(t, 5, len(keys))

	// Verify each key has correct index type
	vt, ok := schema.GetKey("field1")
	assert.True(t, ok)
	assert.NotNil(t, vt.String.StringInvertedIndex)

	vt, ok = schema.GetKey("field2")
	assert.True(t, ok)
	assert.NotNil(t, vt.Int.IntInvertedIndex)
}

func TestSchema_WithVectorIndex(t *testing.T) {
	cfg := NewVectorIndexConfig(WithSpace(SpaceCosine))
	schema, err := NewSchema(
		WithVectorIndex(EmbeddingKey, cfg),
	)
	require.NoError(t, err)

	vt, ok := schema.GetKey(EmbeddingKey)
	assert.True(t, ok)
	assert.NotNil(t, vt.FloatList)
	assert.NotNil(t, vt.FloatList.VectorIndex)
	assert.True(t, vt.FloatList.VectorIndex.Enabled)
	assert.Equal(t, SpaceCosine, vt.FloatList.VectorIndex.Config.Space)
}

func TestSchema_MarshalJSON(t *testing.T) {
	schema, err := NewSchema(
		WithDefaultVectorIndex(NewVectorIndexConfig(WithSpace(SpaceL2))),
		WithStringIndex("my_field"),
	)
	require.NoError(t, err)

	data, err := json.Marshal(schema)
	require.NoError(t, err)
	assert.NotNil(t, data)

	// Verify JSON structure
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)
	assert.Contains(t, result, "defaults")
	assert.Contains(t, result, "keys")

	// Verify keys contains my_field
	keysMap, ok := result["keys"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, keysMap, "my_field")
}

func TestSchema_UnmarshalJSON(t *testing.T) {
	// Create a schema, marshal it, then unmarshal
	original, err := NewSchema(
		WithDefaultVectorIndex(NewVectorIndexConfig(WithSpace(SpaceL2))),
		WithStringIndex("test_field"),
	)
	require.NoError(t, err)

	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal into new schema
	unmarshaled := &Schema{}
	err = json.Unmarshal(data, unmarshaled)
	require.NoError(t, err)

	// Verify structure preserved
	assert.NotNil(t, unmarshaled.Defaults())
	assert.Equal(t, len(original.Keys()), len(unmarshaled.Keys()))
}

func TestSpaceConstants(t *testing.T) {
	assert.Equal(t, Space("l2"), SpaceL2)
	assert.Equal(t, Space("cosine"), SpaceCosine)
	assert.Equal(t, Space("ip"), SpaceIP)
}

func TestReservedKeyConstants(t *testing.T) {
	assert.Equal(t, "#document", DocumentKey)
	assert.Equal(t, "#embedding", EmbeddingKey)
}

// Disable options tests

func TestDisableStringIndex(t *testing.T) {
	schema, err := NewSchema(
		DisableStringIndex("excluded_field"),
	)
	require.NoError(t, err)

	vt, ok := schema.GetKey("excluded_field")
	assert.True(t, ok)
	assert.NotNil(t, vt.String)
	assert.NotNil(t, vt.String.StringInvertedIndex)
	assert.False(t, vt.String.StringInvertedIndex.Enabled)
}

func TestDisableIntIndex(t *testing.T) {
	schema, err := NewSchema(DisableIntIndex("temp_id"))
	require.NoError(t, err)

	vt, ok := schema.GetKey("temp_id")
	assert.True(t, ok)
	assert.False(t, vt.Int.IntInvertedIndex.Enabled)
}

func TestDisableFloatIndex(t *testing.T) {
	schema, err := NewSchema(DisableFloatIndex("temp_score"))
	require.NoError(t, err)

	vt, ok := schema.GetKey("temp_score")
	assert.True(t, ok)
	assert.False(t, vt.Float.FloatInvertedIndex.Enabled)
}

func TestDisableBoolIndex(t *testing.T) {
	schema, err := NewSchema(DisableBoolIndex("temp_flag"))
	require.NoError(t, err)

	vt, ok := schema.GetKey("temp_flag")
	assert.True(t, ok)
	assert.False(t, vt.Bool.BoolInvertedIndex.Enabled)
}

func TestDisableFtsIndex(t *testing.T) {
	schema, err := NewSchema(DisableFtsIndex("notes"))
	require.NoError(t, err)

	vt, ok := schema.GetKey("notes")
	assert.True(t, ok)
	assert.False(t, vt.String.FtsIndex.Enabled)
}

func TestDisableFtsIndex_CannotDisableDocument(t *testing.T) {
	_, err := NewSchema(DisableFtsIndex(DocumentKey))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot disable FTS index on #document")
}

func TestDisableDefaultStringIndex(t *testing.T) {
	schema, err := NewSchema(DisableDefaultStringIndex())
	require.NoError(t, err)
	assert.False(t, schema.Defaults().String.StringInvertedIndex.Enabled)
}

func TestDisableDefaultIntIndex(t *testing.T) {
	schema, err := NewSchema(DisableDefaultIntIndex())
	require.NoError(t, err)
	assert.False(t, schema.Defaults().Int.IntInvertedIndex.Enabled)
}

func TestDisableDefaultFloatIndex(t *testing.T) {
	schema, err := NewSchema(DisableDefaultFloatIndex())
	require.NoError(t, err)
	assert.False(t, schema.Defaults().Float.FloatInvertedIndex.Enabled)
}

func TestDisableDefaultBoolIndex(t *testing.T) {
	schema, err := NewSchema(DisableDefaultBoolIndex())
	require.NoError(t, err)
	assert.False(t, schema.Defaults().Bool.BoolInvertedIndex.Enabled)
}

func TestDisableDefaultFtsIndex(t *testing.T) {
	schema, err := NewSchema(DisableDefaultFtsIndex())
	require.NoError(t, err)
	assert.False(t, schema.Defaults().String.FtsIndex.Enabled)
}

func TestDisableIndex_EmptyKey(t *testing.T) {
	_, err := NewSchema(DisableStringIndex(""))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key cannot be empty")

	_, err = NewSchema(DisableIntIndex(""))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key cannot be empty")

	_, err = NewSchema(DisableFloatIndex(""))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key cannot be empty")

	_, err = NewSchema(DisableBoolIndex(""))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key cannot be empty")

	_, err = NewSchema(DisableFtsIndex(""))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key cannot be empty")
}
