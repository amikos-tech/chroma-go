//go:build test

package test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/types"
)

func Compare(t *testing.T, actual, expected map[string]interface{}) bool {
	builtExprJSON, _ := json.Marshal(actual)
	expectedJSON, _ := json.Marshal(expected)
	require.Equal(t, string(expectedJSON), string(builtExprJSON))
	return true
}

func GetTestDocumentTest() ([]string, []string, []map[string]interface{}, []*types.Embedding) {
	var documents = []string{
		"Document 1 content here",
		"Document 2 content here",
	}
	var ids = []string{
		"ID1",
		"ID2",
	}

	var metadatas = []map[string]interface{}{
		{"key1": "value1"},
		{"key2": "value2"},
	}
	var embeddings = [][]float32{
		[]float32{0.1, 0.2, 0.3},
		[]float32{0.4, 0.5, 0.6},
	}
	return documents, ids, metadatas, types.NewEmbeddingsFromFloat32(embeddings)
}
