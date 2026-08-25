package cohere

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2"
	ccommons "github.com/amikos-tech/chroma-go/pkg/commons/cohere"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func TestRerankResultsAlignsRanksWithSourceDocuments(t *testing.T) {
	t.Parallel()

	// TopN=2 over 3 documents: the API returns only the top 2 results, but
	// their Index values still point into the full document list.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"t","results":[{"index":2,"relevance_score":0.9},{"index":0,"relevance_score":0.5}]}`))
	}))
	defer server.Close()

	rf := &CohereRerankingFunction{
		TopN:           2,
		RerankEndpoint: server.URL + "/v1/rerank",
	}
	rf.CohereClient.BaseURL = server.URL
	rf.CohereClient.APIVersion = ccommons.DefaultAPIVersion
	rf.CohereClient.APIKey = embeddings.NewSecret("test-key")
	rf.CohereClient.Client = server.Client()
	rf.CohereClient.DefaultModel = embeddings.EmbeddingModel(ModelRerankEnglishV30)

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
	require.Len(t, ranks[0], 3, "ranks must align with the source documents, not the returned top-n")
	assert.Equal(t, float32(0.5), ranks[0][0])
	assert.Equal(t, float32(0), ranks[0][1])
	assert.Equal(t, float32(0.9), ranks[0][2])
}
