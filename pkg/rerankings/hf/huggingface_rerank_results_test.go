package huggingface

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func TestRerankResultsAlignsRanksWithSourceDocuments(t *testing.T) {
	t.Parallel()

	// The rerank server may return fewer results than the texts sent; the
	// result Index values still point into the full text list.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"index":2,"score":0.9},{"index":0,"score":0.5}]`))
	}))
	defer server.Close()

	rf, err := NewHFRerankingFunction(WithRerankingEndpoint(server.URL + "/rerank"))
	require.NoError(t, err)

	queryResults := &chromago.QueryResultImpl{
		IDLists: []chromago.DocumentIDs{{"id-1", "id-2", "id-3"}},
		DocumentsLists: []chromago.Documents{
			{
				chromago.NewTextDocument("doc one"),
				chromago.NewTextDocument("doc two"),
				chromago.NewTextDocument("doc three"),
			},
		},
	}

	res, err := rf.RerankResults(context.Background(), []string{"query"}, queryResults)
	require.NoError(t, err)

	ranks := res.Ranks[rf.ID()]
	require.Len(t, ranks, 1)
	require.Len(t, ranks[0], 3, "ranks must align with the source documents, not the returned results")
	assert.Equal(t, float32(0.5), ranks[0][0])
	assert.Equal(t, float32(0), ranks[0][1])
	assert.Equal(t, float32(0.9), ranks[0][2])
}
