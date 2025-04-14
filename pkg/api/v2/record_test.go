package v2

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func TestSimpleRecord(t *testing.T) {
	record, err := NewSimpleRecord(WithRecordID("1"),
		WithRecordEmbedding(embeddings.NewEmbeddingFromFloat32([]float32{1, 2, 3})),
		WithRecordMetadatas(NewDocumentMetadata(NewStringAttribute("key", "value"))))
	require.NoError(t, err)
	require.NotNil(t, record)
}
