package types

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRecordSet(t *testing.T) {
	t.Run("Test NewRecordSet", func(t *testing.T) {
		recordSet, err := NewRecordSet()
		require.NoError(t, err)
		require.NotNil(t, recordSet)
		require.NotNil(t, recordSet.Records)
		require.Equal(t, len(recordSet.Records), 0)
	})
	t.Run("Test NewRecordSet with options", func(t *testing.T) {
		recordSet, err := NewRecordSet(WithIDGenerator(NewULIDGenerator()), WithEmbeddingFunction(NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		require.NotNil(t, recordSet)
		require.NotNil(t, recordSet.IDGenerator)
		require.NotNil(t, recordSet.EmbeddingFunction)
		require.NotNil(t, recordSet.Records)
		require.Equal(t, len(recordSet.Records), 0)
	})

	t.Run("Test NewRecordSet with IDGenerator", func(t *testing.T) {
		recordSet, err := NewRecordSet(WithIDGenerator(NewULIDGenerator()))
		recordSet.WithRecord(WithDocument("test document"))
		require.NoError(t, err)
		require.NotNil(t, recordSet)
		require.NotNil(t, recordSet.IDGenerator)
		require.Nil(t, recordSet.EmbeddingFunction)
		require.NotNil(t, recordSet.Records)
		require.Equal(t, len(recordSet.Records), 1)
		require.NotNil(t, recordSet.Records[0].ID)
	})

	t.Run("Test NewRecordSet with EmbeddingFunction", func(t *testing.T) {
		recordSet, err := NewRecordSet(WithEmbeddingFunction(NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		recordSet.WithRecord(WithDocument("test document"), WithID("1"))
		_, err = recordSet.BuildAndValidate(context.TODO())
		require.NoError(t, err)
		require.NotNil(t, recordSet)
		require.Nil(t, recordSet.IDGenerator)
		require.NotNil(t, recordSet.EmbeddingFunction)
		require.NotNil(t, recordSet.Records)
		require.Equal(t, len(recordSet.Records), 1)
		require.NotNil(t, recordSet.Records[0].ID)
		require.Equal(t, recordSet.Records[0].ID, "1")
		require.NotNil(t, recordSet.Records[0].Document)
		require.Equal(t, recordSet.Records[0].Document, "test document")
		require.NotNil(t, recordSet.Records[0].Embedding.GetFloat32())
	})

	t.Run("Test NewRecordSet with complete Record", func(t *testing.T) {
		recordSet, err := NewRecordSet(WithEmbeddingFunction(NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		var embeddings = []float32{0.1, 0.2, 0.3}
		recordSet.WithRecord(
			WithID("1"),
			WithDocument("test document"),
			WithEmbedding(*NewEmbeddingFromFloat32(embeddings)),
			WithMetadata("testKey", 1),
		)
		_, err = recordSet.BuildAndValidate(context.TODO())
		require.NoError(t, err)
		require.NotNil(t, recordSet)
		require.Nil(t, recordSet.IDGenerator)
		require.NotNil(t, recordSet.EmbeddingFunction)
		require.NotNil(t, recordSet.Records)
		require.Equal(t, len(recordSet.Records), 1)
		require.NotNil(t, recordSet.Records[0].ID)
		require.Equal(t, recordSet.Records[0].ID, "1")
		require.NotNil(t, recordSet.Records[0].Embedding.GetFloat32())
		require.Equal(t, recordSet.Records[0].Embedding.GetFloat32(), &embeddings)
		require.Equal(t, recordSet.Records[0].Document, "test document")
		require.Equal(t, recordSet.Records[0].Metadata["testKey"], 1)
	})
}
