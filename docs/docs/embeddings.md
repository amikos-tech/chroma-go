# Embeddings

The following embedding wrappers are available:

| Embedding Model           | Description                                                                                                                                                 |
|---------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------|
| OpenAI                    | OpenAI embeddings API.<br/>All models are supported - see OpenAI [docs](https://platform.openai.com/docs/guides/embeddings/embedding-models) for more info. |
| Cohere                    | Cohere embeddings API.<br/>All models are supported - see Cohere [API docs](https://docs.cohere.com/reference/embed) for more info.                         |
| HuggingFace Inference API | HuggingFace Inference API.<br/>All models supported by the API.                                                                                             |

## OpenAI

```go
package main

import (
	"context"
	"fmt"
	"os"

	openai "github.com/amikos-tech/chroma-go/openai"
)

func main() {
	ef, efErr := openai.NewOpenAIEmbeddingFunction(os.Getenv("OPENAI_API_KEY"), openai.WithModel(openai.TextEmbedding3Large))
	if efErr != nil {
		fmt.Printf("Error creating OpenAI embedding function: %s \n", efErr)
	}
	documents := []string{
		"Document 1 content here",
	}
	resp, reqErr := ef.EmbedDocuments(context.Background(), documents)
	if reqErr != nil {
		fmt.Printf("Error embedding documents: %s \n", reqErr)
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

## Cohere

```go
package main

import (
	"context"
	"fmt"
	"os"

	cohere "github.com/amikos-tech/chroma-go/cohere"
)

func main() {
	ef := cohere.NewCohereEmbeddingFunction(os.Getenv("COHERE_API_KEY"))
	documents := []string{
		"Document 1 content here",
	}
	resp, reqErr := ef.EmbedDocuments(context.Background(), documents)
	if reqErr != nil {
		fmt.Printf("Error embedding documents: %s \n", reqErr)
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

## HuggingFace Inference API

```go
package main

import (
    "context"
    "fmt"
    "os"

    huggingface "github.com/amikos-tech/chroma-go/hf"
)

func main() {
    ef := huggingface.NewHuggingFaceEmbeddingFunction(os.Getenv("HUGGINGFACE_API_KEY"),"sentence-transformers/all-MiniLM-L6-v2")
    documents := []string{
        "Document 1 content here",
    }
    resp, reqErr := ef.EmbedDocuments(context.Background(), documents)
    if reqErr != nil {
        fmt.Printf("Error embedding documents: %s \n", reqErr)
    }
    fmt.Printf("Embedding response: %v \n", resp)
}
```
