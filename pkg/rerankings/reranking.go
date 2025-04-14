package rerankings

import (
	"context"
	"encoding/json"
	"fmt"

	chromago "github.com/amikos-tech/chroma-go"
)

type RerankingModel string

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

func FromObjects(objects []any) []Result {
	results := make([]Result, len(objects))
	for i, object := range objects {
		results[i] = FromObject(object)
	}
	return results
}

func (r *Result) ToText() (string, error) {
	if r.IsText() {
		return *r.Text, nil
	} else if r.IsObject() {
		marshal, err := json.Marshal(r.Object)
		if err != nil {
			return "", err
		}
		return string(marshal), nil
	}
	return "", fmt.Errorf("result is neither text nor object")
}

func (r *Result) IsText() bool {
	return r.Text != nil
}

func (r *Result) IsObject() bool {
	return r.Object != nil
}

type RerankingFunction interface {
	ID() string
	Rerank(ctx context.Context, query string, results []Result) (map[string][]RankedResult, error)
	RerankResults(ctx context.Context, queryResults *chromago.QueryResults) (*RerankedChromaResults, error)
}
