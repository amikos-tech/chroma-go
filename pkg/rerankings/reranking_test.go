//go:build rf

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

func (d *DummyRerankingFunction) ID() string {
	return "dummy"
}
func (d *DummyRerankingFunction) Rerank(_ context.Context, _ string, results []Result) (map[string][]RankedResult, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no results to rerank")
	}
	rerankedResults := make([]RankedResult, len(results))
	for i, result := range results {
		doc, err := result.ToText()
		if err != nil {
			return nil, err
		}
		rerankedResults[i] = RankedResult{
			String: doc,
			Index:  i,
			Rank:   rand.Float32(),
		}
	}
	return map[string][]RankedResult{d.ID(): rerankedResults}, nil
}

func (d *DummyRerankingFunction) RerankResults(_ context.Context, queryResults *chromago.QueryResults) (*RerankedChromaResults, error) {
	if len(queryResults.Ids) == 0 {
		return nil, fmt.Errorf("no results to rerank")
	}
	results := &RerankedChromaResults{
		QueryResults: *queryResults,
		Ranks:        map[string][][]float32{d.ID(): make([][]float32, len(queryResults.Ids))},
	}
	for i, qr := range queryResults.Ids {
		results.Ranks[d.ID()][i] = make([]float32, len(qr))
		for j := range qr {
			results.Ranks[d.ID()][i][j] = rand.Float32()
		}
	}
	return results, nil
}

func Test_reranking_function(t *testing.T) {
	rerankingFunction := NewDummyRerankingFunction()
	t.Run("Rerank string results", func(t *testing.T) {
		query := "hello world"
		results := []string{"hello", "world"}
		rerankedResults, err := rerankingFunction.Rerank(context.Background(), query, FromTexts(results))
		require.NoError(t, err)
		require.NotNil(t, rerankedResults)
		require.Contains(t, rerankedResults, rerankingFunction.ID())
		require.Equal(t, len(results), len(rerankedResults[rerankingFunction.ID()]))
		for _, result := range rerankedResults[rerankingFunction.ID()] {
			require.Equal(t, results[result.Index], result.String)
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
		require.Contains(t, rerankedResults.Ranks, rerankingFunction.ID())
		require.Equal(t, len(results.Ids), len(rerankedResults.Ids))
		require.Equal(t, results.Ids, rerankedResults.Ids)
		require.Equal(t, results.Documents, rerankedResults.Documents)
		require.Equal(t, results.QueryTexts, rerankedResults.QueryTexts)
	})
}
