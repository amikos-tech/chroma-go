# Reranking Functions

Reranking functions allow users to feed Chroma results into a reranking model such
as `cross-encoder/ms-marco-MiniLM-L-6-v2` to improve the quality of the search results.

Rerankers take the returned documents from Chroma and the original query and rank each result's relevance to the query.

## How To Use Rerankers

Each reranker exposes the following methods:

- `Rerank` which takes a query string and `[]Result` inputs, returning a `map[string][]RankedResult` keyed by the reranker's ID.
- `RerankResults` which takes query texts and a `*QueryResultImpl` and returns `*RerankedChromaResults` with ranking information.

```go
package rerankings

import (
	"context"

	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

type RankedResult struct {
	Index  int // Index in the original input
	String string
	Rank   float32
}

type RerankedChromaResults struct {
	*chromago.QueryResultImpl
	QueryTexts []string
	Ranks      map[string][][]float32
}

type RerankingFunction interface {
	ID() string
	Rerank(ctx context.Context, query string, results []Result) (map[string][]RankedResult, error)
	RerankResults(ctx context.Context, queryTexts []string, queryResults *chromago.QueryResultImpl) (*RerankedChromaResults, error)
}
```

The `Result` type wraps either a text string or an arbitrary object:

```go
// Create results from text strings
results := rerankings.FromTexts([]string{"text1", "text2"})

// Or from a single text
result := rerankings.FromText("some text")
```

## Supported Rerankers

- Cohere - ✅
- Jina AI - ✅
- HuggingFace Text Embedding Inference - ✅
- Together AI - ✅
- HuggingFace Inference API - coming soon

### Cohere Reranker

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/amikos-tech/chroma-go/pkg/rerankings"
	cohere "github.com/amikos-tech/chroma-go/pkg/rerankings/cohere"
)

func main() {
	var query = "What is the capital of the United States?"
	var results = rerankings.FromTexts([]string{
		"Carson City is the capital city of the American state of Nevada.",
		"The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean that are a political division controlled by the United States. Its capital is Saipan.",
		"Charlotte Amalie is the capital and largest city of the United States Virgin Islands. It has about 20,000 people. The city is on the island of Saint Thomas.",
		"Washington, D.C. (also known as simply Washington or D.C., and officially as the District of Columbia) is the capital of the United States.",
		"Capital punishment (the death penalty) has existed in the United States since before the United States was a country.",
	})

	rf, err := cohere.NewCohereRerankingFunction(cohere.WithAPIKey(os.Getenv("COHERE_API_KEY")))
	if err != nil {
		fmt.Printf("Error creating Cohere reranking function: %s \n", err)
	}

	res, err := rf.Rerank(context.Background(), query, results)
	if err != nil {
		fmt.Printf("Error reranking: %s \n", err)
	}

	for _, rs := range res[rf.ID()] {
		fmt.Printf("Rank: %f, Index: %d\n", rs.Rank, rs.Index)
	}
}
```

### Jina AI Reranker

To use Jina AI reranking, you will need to get an [API Key](https://jina.ai) (trial API keys are freely available
without any registration, scroll down the page and find the automatically generated API key).

Supported models - https://api.jina.ai/redoc#tag/rerank/operation/rank_v1_rerank_post

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/amikos-tech/chroma-go/pkg/rerankings"
	jina "github.com/amikos-tech/chroma-go/pkg/rerankings/jina"
)

func main() {
	var query = "What is the capital of the United States?"
	var results = rerankings.FromTexts([]string{
		"Carson City is the capital city of the American state of Nevada.",
		"The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean that are a political division controlled by the United States. Its capital is Saipan.",
		"Charlotte Amalie is the capital and largest city of the United States Virgin Islands. It has about 20,000 people. The city is on the island of Saint Thomas.",
		"Washington, D.C. (also known as simply Washington or D.C., and officially as the District of Columbia) is the capital of the United States.",
		"Capital punishment (the death penalty) has existed in the United States since before the United States was a country.",
	})

	rf, err := jina.NewJinaRerankingFunction(jina.WithAPIKey(os.Getenv("JINA_API_KEY")))
	if err != nil {
		fmt.Printf("Error creating Jina reranking function: %s \n", err)
	}

	res, err := rf.Rerank(context.Background(), query, results)
	if err != nil {
		fmt.Printf("Error reranking: %s \n", err)
	}

	for _, rs := range res[rf.ID()] {
		fmt.Printf("Rank: %f, Index: %d\n", rs.Rank, rs.Index)
	}
}
```

### HFEI Reranker

You need to run a local [HFEI server](https://github.com/huggingface/text-embeddings-inference?tab=readme-ov-file#sequence-classification-and-re-ranking). You can do that by running the following command:

```bash
docker run --rm -p 8080:80 -v $PWD/data:/data --platform linux/amd64 ghcr.io/huggingface/text-embeddings-inference:cpu-latest --model-id BAAI/bge-reranker-base
```

```go
package main

import (
	"context"
	"fmt"

	"github.com/amikos-tech/chroma-go/pkg/rerankings"
	hf "github.com/amikos-tech/chroma-go/pkg/rerankings/hf"
)

func main() {
	var query = "What is the capital of the United States?"
	var results = rerankings.FromTexts([]string{
		"Carson City is the capital city of the American state of Nevada.",
		"The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean that are a political division controlled by the United States. Its capital is Saipan.",
		"Charlotte Amalie is the capital and largest city of the United States Virgin Islands. It has about 20,000 people. The city is on the island of Saint Thomas.",
		"Washington, D.C. (also known as simply Washington or D.C., and officially as the District of Columbia) is the capital of the United States.",
		"Capital punishment (the death penalty) has existed in the United States since before the United States was a country.",
	})

	rf, err := hf.NewHFRerankingFunction(hf.WithRerankingEndpoint("http://127.0.0.1:8080/rerank"))
	if err != nil {
		fmt.Printf("Error creating HFEI reranking function: %s \n", err)
	}

	res, err := rf.Rerank(context.Background(), query, results)
	if err != nil {
		fmt.Printf("Error reranking: %s \n", err)
	}

	for _, rs := range res[rf.ID()] {
		fmt.Printf("Rank: %f, Index: %d\n", rs.Rank, rs.Index)
	}
}
```

### Together AI Reranker

To use Together AI reranking, you will need to get an [API Key](https://api.together.ai/).

The default model is `Salesforce/Llama-Rank-V1`.

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/amikos-tech/chroma-go/pkg/rerankings"
	together "github.com/amikos-tech/chroma-go/pkg/rerankings/together"
)

func main() {
	var query = "What is the capital of the United States?"
	var results = rerankings.FromTexts([]string{
		"Carson City is the capital city of the American state of Nevada.",
		"The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean that are a political division controlled by the United States. Its capital is Saipan.",
		"Charlotte Amalie is the capital and largest city of the United States Virgin Islands. It has about 20,000 people. The city is on the island of Saint Thomas.",
		"Washington, D.C. (also known as simply Washington or D.C., and officially as the District of Columbia) is the capital of the United States.",
		"Capital punishment (the death penalty) has existed in the United States since before the United States was a country.",
	})

	rf, err := together.NewTogetherRerankingFunction(together.WithAPIKey(os.Getenv("TOGETHER_API_KEY")))
	if err != nil {
		fmt.Printf("Error creating Together reranking function: %s \n", err)
	}

	res, err := rf.Rerank(context.Background(), query, results)
	if err != nil {
		fmt.Printf("Error reranking: %s \n", err)
	}

	for _, rs := range res[rf.ID()] {
		fmt.Printf("Rank: %f, Index: %d\n", rs.Rank, rs.Index)
	}
}
```
