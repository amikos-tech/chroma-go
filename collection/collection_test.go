package collection

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/types"
)

func TestCollectionBuilder(t *testing.T) {
	t.Run("With Name", func(t *testing.T) {
		b := &Builder{}
		err := WithName("test")(b)
		require.NoError(t, err, "Unexpected error: %v", err)
		require.Equal(t, "test", b.Name)
	})

	t.Run("With Embedding Function", func(t *testing.T) {
		b := &Builder{}
		err := WithEmbeddingFunction(nil)(b)
		require.NoError(t, err, "Unexpected error: %v", err)
		require.Nil(t, b.EmbeddingFunction)
	})
	t.Run("With ID Generator", func(t *testing.T) {
		var generator = types.NewULIDGenerator()
		b := &Builder{}
		err := WithIDGenerator(generator)(b)
		require.NoError(t, err, "Unexpected error: %v", err)
		require.Equal(t, generator, b.IDGenerator)
	})
	t.Run("With Create If Not Exist", func(t *testing.T) {
		b := &Builder{}
		err := WithCreateIfNotExist(true)(b)
		require.NoError(t, err, "Unexpected error: %v", err)
		require.True(t, b.CreateIfNotExist)
	})

	t.Run("With HNSW Distance Function", func(t *testing.T) {
		b := &Builder{}
		err := WithHNSWDistanceFunction(types.L2)(b)
		require.NoError(t, err, "Unexpected error: %v", err)
		require.NoError(t, err, "Unexpected error: %v", err)
		require.Equal(t, types.L2, b.Metadata[types.HNSWSpace])
	})

	t.Run("With Metadata", func(t *testing.T) {
		b := &Builder{}
		err := WithMetadata("testKey", "testValue")(b)
		require.NoError(t, err, "Unexpected error: %v", err)
		require.Equal(t, "testValue", b.Metadata["testKey"])
	})

	t.Run("With Metadatas", func(t *testing.T) {
		b := &Builder{}
		err := WithMetadatas(map[string]interface{}{"testKey": "testValue"})(b)
		require.NoError(t, err, "Unexpected error: %v", err)
		require.Equal(t, "testValue", b.Metadata["testKey"])
	})

	t.Run("With Metadatas for existing no override", func(t *testing.T) {
		b := &Builder{}
		b.Metadata = map[string]interface{}{"existingKey": "existingValue"}
		err := WithMetadatas(map[string]interface{}{"testKey": "testValue"})(b)
		require.NoError(t, err, "Unexpected error: %v", err)
		require.Contains(t, b.Metadata, "testKey")
		require.Equal(t, "testValue", b.Metadata["testKey"])
		require.Contains(t, b.Metadata, "existingKey")
		require.Equal(t, "existingValue", b.Metadata["existingKey"])
	})

	t.Run("With Metadatas for existing with override", func(t *testing.T) {
		b := &Builder{}
		b.Metadata = map[string]interface{}{"existingKey": "existingValue"}
		err := WithMetadatas(map[string]interface{}{"existingKey": "newValue"})(b)
		require.NoError(t, err, "Unexpected error: %v", err)
		require.Contains(t, b.Metadata, "existingKey")
		require.Equal(t, "newValue", b.Metadata["existingKey"])
	})

	t.Run("With Metadatas for invalid type", func(t *testing.T) {
		b := &Builder{}
		err := WithMetadatas(map[string]interface{}{"testKey": map[string]interface{}{"invalid": "value"}})(b)
		require.Error(t, err)
	})

	t.Run("With HNSW Batch Size", func(t *testing.T) {
		b := &Builder{}
		err := WithHNSWBatchSize(10)(b)
		require.NoError(t, err, "Unexpected error: %v", err)
		require.Equal(t, int32(10), b.Metadata[types.HNSWBatchSize])
	})

	t.Run("With HNSW Sync Threshold", func(t *testing.T) {
		b := &Builder{}
		err := WithHNSWSyncThreshold(10)(b)
		require.NoError(t, err, "Unexpected error: %v", err)
		require.Equal(t, int32(10), b.Metadata[types.HNSWSyncThreshold])
	})

	t.Run("With HNSWM", func(t *testing.T) {
		b := &Builder{}
		err := WithHNSWM(10)(b)
		require.NoError(t, err, "Unexpected error: %v", err)
		require.Equal(t, int32(10), b.Metadata[types.HNSWM])
	})

	t.Run("With HNSW Construction Ef", func(t *testing.T) {
		b := &Builder{}
		err := WithHNSWConstructionEf(10)(b)
		require.NoError(t, err, "Unexpected error: %v", err)
		require.Equal(t, int32(10), b.Metadata[types.HNSWConstructionEF])
	})

	t.Run("With HNSW Search Ef", func(t *testing.T) {
		b := &Builder{}
		err := WithHNSWSearchEf(10)(b)
		require.NoError(t, err, "Unexpected error: %v", err)
		require.Equal(t, int32(10), b.Metadata[types.HNSWSearchEF])
	})
}
