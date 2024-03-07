# Filtering


## Metadata

```go
package main

import (
	"context"
	"fmt"
	"github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/openai"
)

func main() {
	embeddingF,err := openai.NewOpenAIEmbeddingFunction("sk-xxxx")
	if err != nil {
        fmt.Println(err)
        return
    }
	client, err := chroma.NewClient("http://localhost:11434/v1/")
	if err != nil {
		fmt.Println(err)
		return
	}
	collection,err := client.GetCollection(context.TODO(), "my-collection",embeddingF)
	if err != nil {
        fmt.Println(err)
        return
    }
    // Filter by metadata
	
	//collection.GetWithOptions(context.TODO(),types.Where)
		
	
}

```

## Document