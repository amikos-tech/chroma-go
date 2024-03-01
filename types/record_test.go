package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecordProducer(t *testing.T) {
	t.Run("Test invalid Metadata", func(t *testing.T) {
		recordProducer, err := NewRecordProducer()
		require.NoError(t, err)
		_, err = recordProducer.Produce(
			WithID("1"),
			WithEmbedding(*NewEmbeddingFromFloat32([]float32{0.1, 0.2, 0.3})),
			WithDocument("test document"),
			WithMetadata("testKey", map[string]interface{}{"invalid": "value"}),
			WithMetadata("testKey", 1),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Invalid metadata value type")
	})

	t.Run("Test With All", func(t *testing.T) {
		recordProducer, err := NewRecordProducer()
		require.NoError(t, err)
		record, err := recordProducer.Produce(
			WithID("1"),
			WithEmbedding(*NewEmbeddingFromFloat32([]float32{0.1, 0.2, 0.3})),
			WithDocument("test document"),
			WithMetadata("testKey", 1),
			WithMetadatas(map[string]interface{}{"testKey1": "testValue"}),
			WithURI("testURI"),
		)
		require.NoError(t, err)
		require.Equal(t, record.ID, "1")
		require.Equal(t, record.Embedding, *NewEmbeddingFromFloat32([]float32{0.1, 0.2, 0.3}))
		require.Equal(t, record.Document, "test document")
		require.Equal(t, record.Metadata["testKey"], 1)
		require.Equal(t, record.Metadata["testKey1"], "testValue")
		require.Equal(t, record.URI, "testURI")
	})

	t.Run("Test ID Generator", func(t *testing.T) {
		recordProducer, err := NewRecordProducer(WithIDGenerator(NewULIDGenerator()))
		require.NoError(t, err)
		record, err := recordProducer.Produce(
			WithEmbedding(*NewEmbeddingFromFloat32([]float32{0.1, 0.2, 0.3})),
			WithDocument("test document"),
			WithMetadata("testKey", 1),
		)
		require.NoError(t, err)
		require.NotNil(t, record.ID)
		require.Equal(t, record.Embedding, *NewEmbeddingFromFloat32([]float32{0.1, 0.2, 0.3}))
		require.Equal(t, record.Document, "test document")
		require.Equal(t, record.Metadata["testKey"], 1)
	})

	t.Run("Test without ID Generator and without ID", func(t *testing.T) {
		recordProducer, err := NewRecordProducer()
		require.NoError(t, err)
		_, err = recordProducer.Produce(
			WithEmbedding(*NewEmbeddingFromFloat32([]float32{0.1, 0.2, 0.3})),
			WithDocument("test document"),
			WithMetadata("testKey", 1),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "either id or id generator is required")
	})

	t.Run("Test InPlace Embedding function", func(t *testing.T) {
		recordProducer, err := NewRecordProducer(WithInPlaceEmbeddingFunction(NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		record, err := recordProducer.Produce(
			WithID("1"),
			WithDocument("test document"),
			WithMetadata("testKey", 1),
		)
		require.NoError(t, err)
		require.NotNil(t, record.ID)
		require.Equal(t, record.Document, "test document")
		require.Equal(t, record.Metadata["testKey"], 1)
		require.True(t, record.Embedding.IsDefined())
	})

	t.Run("Test Without Doc or Embedding", func(t *testing.T) {
		recordProducer, err := NewRecordProducer()
		require.NoError(t, err)
		_, err = recordProducer.Produce(
			WithMetadata("testKey", 1),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "document or embedding or InPlace EmbeddingFunction must be provided")
	})
}
