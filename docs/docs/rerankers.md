# Reranking Functions

!!! note "V2 API Support"

    The reranking functions work with both V1 and V2 APIs. The `Rerank` method accepts plain text and works universally. The `RerankResults` method interface shown below references V1 types (`chromago.QueryResults`). For V1-specific usage, pin your dependency to `v0.2.5` or earlier:
    ```bash
    go get github.com/amikos-tech/chroma-go@v0.2.5
    ```

Reranking functions allow users to feed Chroma results into a reranking model such
as `cross-encoder/ms-marco-MiniLM-L-6-v2` to improve the quality of the search results.

Rerankers take the returned documents from Chroma and the original query and rank each result's relevance to the query.

## How To Use Rerankers

Each reranker exposes the following methods:

- `Rerank` which takes plain text query and results and returns a list of ranked results.
- `RerankResults` which takes a `QueryResults` object and returns a list of `RerankedChromaResults` objects. RerankedChromaResults inherits from `QueryResults` and adds a `Ranks` field which contains the ranks of each result.

```go
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
	cohere "github.com/amikos-tech/chroma-go/pkg/rerankings/cohere"
	"os"
)

func main() {
	var query = "What is the capital of the United States?"
	var results = []string{
		"Carson City is the capital city of the American state of Nevada.",
		"The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean that are a political division controlled by the United States. Its capital is Saipan.",
		"Charlotte Amalie is the capital and largest city of the United States Virgin Islands. It has about 20,000 people. The city is on the island of Saint Thomas.",
		"Washington, D.C. (also known as simply Washington or D.C., and officially as the District of Columbia) is the capital of the United States.",
		"Capital punishment (the death penalty) has existed in the United States since before the United States was a country.",
	}

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
	jina "github.com/amikos-tech/chroma-go/pkg/rerankings/jina"
	"os"
)

func main() {
	var query = "What is the capital of the United States?"
	var results = []string{
		"Carson City is the capital city of the American state of Nevada.",
		"The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean that are a political division controlled by the United States. Its capital is Saipan.",
		"Charlotte Amalie is the capital and largest city of the United States Virgin Islands. It has about 20,000 people. The city is on the island of Saint Thomas.",
		"Washington, D.C. (also known as simply Washington or D.C., and officially as the District of Columbia) is the capital of the United States.",
		"Capital punishment (the death penalty) has existed in the United States since before the United States was a country.",
	}

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
	hf "github.com/amikos-tech/chroma-go/pkg/rerankings/hf"
	"os"
)

func main() {
	var query = "What is the capital of the United States?"
	var results = []string{
		"Carson City is the capital city of the American state of Nevada.",
		"The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean that are a political division controlled by the United States. Its capital is Saipan.",
		"Charlotte Amalie is the capital and largest city of the United States Virgin Islands. It has about 20,000 people. The city is on the island of Saint Thomas.",
		"Washington, D.C. (also known as simply Washington or D.C., and officially as the District of Columbia) is the capital of the United States.",
		"Capital punishment (the death penalty) has existed in the United States since before the United States was a country.",
	}

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
	together "github.com/amikos-tech/chroma-go/pkg/rerankings/together"
	"os"
)

func main() {
	var query = "What is the capital of the United States?"
	var results = []string{
		"Carson City is the capital city of the American state of Nevada.",
		"The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean that are a political division controlled by the United States. Its capital is Saipan.",
		"Charlotte Amalie is the capital and largest city of the United States Virgin Islands. It has about 20,000 people. The city is on the island of Saint Thomas.",
		"Washington, D.C. (also known as simply Washington or D.C., and officially as the District of Columbia) is the capital of the United States.",
		"Capital punishment (the death penalty) has existed in the United States since before the United States was a country.",
	}

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
