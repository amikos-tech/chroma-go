# Chroma Go Client

An experimental Go client for ChromaDB.


## Installation

Add the library to your project:

```bash
go get github.com/amikos-tech/chroma-go
```

## Usage

Import the library:

```go
package main

import (
    chroma "github.com/amikos-tech/chroma-go"
    "github.com/amikos-tech/chroma-go/collection"
    openai "github.com/amikos-tech/chroma-go/openai"
    "github.com/amikos-tech/chroma-go/types"
)
```

### Create a new collection

Here's a simple example of creating a new collection:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	chroma "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/openai"
	"github.com/amikos-tech/chroma-go/types"
)

func main() {
	ctx := context.Background()
	client := chroma.NewClient("localhost:8000")

	openaiEf, err := openai.NewOpenAIEmbeddingFunction(os.Getenv("OPENAI_API_KEY"))
	if err != nil {
		log.Fatalf("Error creating OpenAI embedding function: %s \n", err)
	}

	// Create a new collection with OpenAI embedding function, L2 distance function and metadata
	collection, err := client.CreateCollection(ctx, "my-collection", map[string]interface{}{"key1": "value1"}, true, openaiEf, types.L2)
	if err != nil {
		log.Fatalf("Failed to create collection: %v", err)
	}
	fmt.Printf("Collection created: %v\n", collection)
}
```
