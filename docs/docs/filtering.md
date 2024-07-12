# Filtering

Chroma offers two types of filters:

- Metadata - filtering based on metadata attribute values
- Documents - filtering based on document content (contains or not contains)

## Metadata

* TODO - Add builder example
* TODO - Describe all available operations

```go
package main

import (
	"context"
	"fmt"
	chroma "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
	"github.com/amikos-tech/chroma-go/types"
	"github.com/amikos-tech/chroma-go/where"
)

func main() {
	embeddingF, err := openai.NewOpenAIEmbeddingFunction("sk-xxxx")
	if err != nil {
		fmt.Println(err)
		return
	}
	client, err := chroma.NewClient() // connects to localhost:8000
	if err != nil {
		fmt.Println(err)
		return
	}
	collection, err := client.GetCollection(context.TODO(), "my-collection", embeddingF)
	if err != nil {
		fmt.Println(err)
		return
	}
	// Filter by metadata

	result, err := collection.GetWithOptions(
		context.Background(),
		types.WithWhere(
			where.Or(
				where.Eq("category", "Chroma"),
				where.Eq("type", "vector database"),
			),
		),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	// do something with result
	fmt.Println(result)
}

```

## Document

* TODO - Add builder example
* TODO - Describe all available operations

```go
package main

import (
	"context"
	"fmt"
	chroma "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
	"github.com/amikos-tech/chroma-go/types"
	"github.com/amikos-tech/chroma-go/where_document"
)

func main() {
	embeddingF, err := openai.NewOpenAIEmbeddingFunction("sk-xxxx")
	if err != nil {
		fmt.Println(err)
		return
	}
	client, err := chroma.NewClient(chroma.WithBasePath("http://localhost:8000"))
	if err != nil {
		fmt.Println(err)
		return
	}
	collection, err := client.GetCollection(context.TODO(), "my-collection", embeddingF)
	if err != nil {
		fmt.Println(err)
		return
	}
	// Filter by metadata

	result, err := collection.GetWithOptions(
		context.Background(),
		types.WithWhereDocument(
			wheredoc.Or(
				wheredoc.Contains("Vector database"),
				wheredoc.Contains("Chroma"),
			),
		),
	)

	if err != nil {
		fmt.Println(err)
		return
	}
	// do something with result
	fmt.Println(result)
}
```