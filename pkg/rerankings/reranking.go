package rerankings

import (
	"context"

	chromago "github.com/amikos-tech/chroma-go"
)

type RankedResult struct {
	ID     int // Index in the original input []string
	String string
	Rank   float32
}

type RerankedChromaResults struct {
	chromago.QueryResults
	Ranks [][]float32
}

type RerankingFunction interface {
	Rerank(ctx context.Context, query string, results []string) ([]*RankedResult, error)
	RerankResults(ctx context.Context, queryResults *chromago.QueryResults) (RerankedChromaResults, error)
}
