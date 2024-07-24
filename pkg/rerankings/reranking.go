package rerankings

import (
	"context"

	chromago "github.com/amikos-tech/chroma-go"
)

type RankedResult struct {
	Index  int // Index in the original input []string
	String string
	Rank   float32
}

type RerankedChromaResults struct {
	chromago.QueryResults
	Ranks map[string][][]float32 // each reranker adds a rank for each result
}

type Result struct {
	Text   *string
	Object *any
}

func FromText(text string) Result {
	return Result{
		Text: &text,
	}
}

func FromTexts(texts []string) []Result {
	results := make([]Result, len(texts))
	for i, text := range texts {
		results[i] = FromText(text)
	}
	return results
}

func FromObject(object any) Result {
	return Result{
		Object: &object,
	}
}

func (r Result) ToText() string {
	if r.Text != nil {
		return *r.Text
	}
	return ""
}

func IsText(r Result) bool {
	return r.Text != nil
}

func IsObject(r Result) bool {
	return r.Object != nil
}

type RerankingFunction interface {
	ID() string
	Rerank(ctx context.Context, query string, results []Result) (map[string][]RankedResult, error)
	RerankResults(ctx context.Context, queryResults *chromago.QueryResults) (*RerankedChromaResults, error)
}
