//go:build basicv2 && !cloud

package v2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func TestGetResultDeserialization(t *testing.T) {
	var apiResponse = `{
  "documents": [
    "document1",
	"document2"
  ],
  "embeddings": [
    [0.1,0.2],
	[0.3,0.4]
  ],
  "ids": [
    "id1",
	"id2"
  ],
  "include": [
    "distances"
  ],
  "metadatas": [
    {
      "additionalProp1": true,
      "additionalProp2": 1,
      "additionalProp3": "test"
    },
	{"additionalProp1": false}
  ]
}`

	var result GetResultImpl
	err := json.Unmarshal([]byte(apiResponse), &result)
	require.NoError(t, err)
	require.Len(t, result.GetDocuments(), 2)
	require.Len(t, result.GetIDs(), 2)
	require.Equal(t, result.GetIDs()[0], DocumentID("id1"))
	require.Equal(t, result.GetDocuments()[0], NewTextDocument("document1"))
	require.Equal(t, []float32{0.1, 0.2}, result.GetEmbeddings()[0].ContentAsFloat32())
	require.Len(t, result.GetEmbeddings(), 2)
	require.Len(t, result.GetMetadatas(), 2)
}

func TestQueryResultDeserialization(t *testing.T) {
	var apiResponse = `{
  "distances": [
    [
      0.1
    ]
  ],
  "documents": [
    [
      "string"
    ]
  ],
  "embeddings": [
    [
      [
        0.1
      ]
    ]
  ],
  "ids": [
    [
      "id1"
    ]
  ],
  "include": [
    "distances"
  ],
  "metadatas": [
    [
      {
        "additionalProp1": true,
        "additionalProp2": true,
        "additionalProp3": true
      }
    ]
  ]
}`

	var result QueryResultImpl
	err := json.Unmarshal([]byte(apiResponse), &result)
	require.NoError(t, err)
	require.Len(t, result.GetIDGroups(), 1)
	require.Len(t, result.GetIDGroups()[0], 1)
	require.Equal(t, DocumentID("id1"), result.GetIDGroups()[0][0])

	require.Len(t, result.GetDocumentsGroups(), 1)
	require.Len(t, result.GetDocumentsGroups()[0], 1)
	require.Equal(t, NewTextDocument("string"), result.GetDocumentsGroups()[0][0])

	require.Len(t, result.GetEmbeddingsGroups(), 1)
	require.Len(t, result.GetEmbeddingsGroups()[0], 1)
	require.Equal(t, []float32{0.1}, result.GetEmbeddingsGroups()[0][0].ContentAsFloat32())

	require.Len(t, result.GetMetadatasGroups(), 1)
	require.Len(t, result.GetMetadatasGroups()[0], 1)
	metadata := NewDocumentMetadata(
		NewBoolAttribute("additionalProp1", true),
		NewBoolAttribute("additionalProp3", true),
		NewBoolAttribute("additionalProp2", true),
	)
	require.Equal(t, metadata, result.GetMetadatasGroups()[0][0])

	require.Len(t, result.GetDistancesGroups(), 1)
	require.Len(t, result.GetDistancesGroups()[0], 1)
	require.Equal(t, embeddings.Distance(0.1), result.GetDistancesGroups()[0][0])
}
