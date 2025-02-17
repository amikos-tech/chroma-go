package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimpleRecord(t *testing.T) {
	record, err := NewSimpleRecord(WithRecordID("1"),
		WithRecordEmbedding(NewEmbeddingFromFloat32([]float32{1, 2, 3})),
		WithRecordMetadatas(map[string]interface{}{"key": "value"}))
	require.NoError(t, err)
	require.NotNil(t, record)
}
