# Reranking Functions

Reranking functions allow users to feed Chroma results into a reranking model such
as `cross-encoder/ms-marco-MiniLM-L-6-v2` to improve the quality of the search results.

Rerankers take the returned documents from Chroma and the original query and rank each result's relevance to the query.

## How To Use Rerankers

Each reranker exposes the following methods:

- `Rerank` which takes plain text query and results and returns a list of ranked results.
- `RerankResults` which takes a `QueryResults` object and returns a list of `RerankedChromaResults` objects. RerankedChromaResults inherits from `QueryResults` and adds a `Ranks` field which contains the ranks of each result.

```go
package main

import (
	"context"
	chromago "github.com/amikos-tech/chroma-go"
)

type RankedResult struct {
	String string
	Rank   float32
}

type RerankedChromaResults struct {
	chromago.QueryResults
	Ranks [][]float32
}

type RerankingFunction interface {
	Rerank(ctx context.Context, query string, results []string) ([]*RankedResult, error)
	RerankResults(ctx context.Context, queryResults *chromago.QueryResults) ([]*RerankedChromaResults, error)
}
```

## Supported Rerankers

TBD

