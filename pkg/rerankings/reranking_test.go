package rerankings

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	chromago "github.com/amikos-tech/chroma-go"
)

type DummyRerankingFunction struct {
}

func NewDummyRerankingFunction() *DummyRerankingFunction {
	return &DummyRerankingFunction{}
}

func (d *DummyRerankingFunction) Rerank(_ context.Context, _ string, results []string) ([]*RankedResult, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no results to rerank")
	}
	rerankedResults := make([]*RankedResult, len(results))
	for i, result := range results {
		rerankedResults[i] = &RankedResult{
			String: result,
			ID:     i,
			Rank:   rand.Float32(),
		}
	}
	return rerankedResults, nil
}

func (d *DummyRerankingFunction) RerankResults(_ context.Context, queryResults *chromago.QueryResults) (*RerankedChromaResults, error) {
	if len(queryResults.Ids) == 0 {
		return nil, fmt.Errorf("no results to rerank")
	}
	results := &RerankedChromaResults{
		QueryResults: *queryResults,
		Ranks:        make([][]float32, len(queryResults.Ids)),
	}
	for i, qr := range queryResults.Ids {
		results.Ranks[i] = make([]float32, len(qr))
		for j := range qr {
			results.Ranks[i][j] = rand.Float32()
		}
	}
	return results, nil
}

func Test_reranking_function(t *testing.T) {
	rerankingFunction := NewDummyRerankingFunction()
	t.Run("Rerank string results", func(t *testing.T) {
		query := "hello world"
		results := []string{"hello", "world"}
		rerankedResults, err := rerankingFunction.Rerank(context.Background(), query, results)
		require.NoError(t, err)
		require.NotNil(t, rerankedResults)
		require.Equal(t, len(results), len(rerankedResults))
		for _, result := range rerankedResults {
			require.Equal(t, results[result.ID], result.String)
		}
	})

	t.Run("Rerank chroma results", func(t *testing.T) {
		query := "hello world"
		results := &chromago.QueryResults{
			Ids:        [][]string{{"1"}, {"2"}},
			Documents:  [][]string{{"hello"}, {"world"}},
			Distances:  [][]float32{{0.1}, {0.2}},
			QueryTexts: []string{query},
		}
		rerankedResults, err := rerankingFunction.RerankResults(context.Background(), results)
		require.NoError(t, err)
		require.NotNil(t, rerankedResults)
		require.Equal(t, len(results.Ids), len(rerankedResults.Ids))
		require.Equal(t, results.Ids, rerankedResults.Ids)
		require.Equal(t, results.Documents, rerankedResults.Documents)
		require.Equal(t, results.QueryTexts, rerankedResults.QueryTexts)
	})
}
