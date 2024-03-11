# Chroma Go Client

An experimental Go client for ChromaDB.

## Installation

Add the library to your project:

```bash
go get github.com/amikos-tech/chroma-go
```

## Getting Started

Concepts:

- [Client Options](client.md) - How to configure chroma go client
- [Embeddings](embeddings.md) - List of available embedding functions and how to use them
- [Records](records.md) - How to work with records
- [Filtering](filtering.md) - How to filter results

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

New client:

!!! note Client Options
    
    Check [Client Options](client.md) for more details.

```go
package main

import (
	chroma "github.com/amikos-tech/chroma-go"
	"fmt"
)

func main() {
    client,err := chroma.NewClient("localhost:8000")
    if err != nil {
        fmt.Printf("Failed to create client: %v", err)
    }
	// do something with client
}
```

### Embedding Functions

The client supports a number of embedding wrapper functions. See [Embeddings](embeddings.md) for more details.

### CRUD Operations

Ensure you have a running instance of Chroma running.
See [this doc](https://cookbook.chromadb.dev/running/running-chroma/#running-chroma) for more info how to run local
Chroma instance.

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
	_, err = client.CreateCollection(ctx, "my-collection", map[string]interface{}{"key1": "value1"}, true, openaiEf, types.L2)
	if err != nil {
		log.Fatalf("Failed to create collection: %v", err)
	}
	
	// Get collection
	collection, err := client.GetCollection(ctx, "my-collection", openaiEf)
	if err != nil {
        log.Fatalf("Failed to get collection: %v", err)
    }
	
	// Modify collection
	_, err = collection.Update(ctx, "new-collection",nil)
	if err != nil {
        log.Fatalf("Failed to update collection: %v", err)
    }
	
	// Delete collection
	_, err = client.DeleteCollection(ctx, "new-collection")
	if err != nil {
        log.Fatalf("Failed to delete collection: %v", err)
    }
}
```

### Add documents

Here's a simple example of adding documents to a collection:

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
	// Get the collection we created earlier
	collection, err := client.GetCollection(ctx, "my-collection", openaiEf)
	if err != nil {
		log.Fatalf("Failed to create collection: %v", err)
		return
	}
	_, err = collection.Add(context.TODO(), nil, []map[string]interface{}{{"key1": "value1"}}, []string{"My name is John and I have three dogs."}, []string{"ID1"})
	if err != nil {
		log.Fatalf("Error adding documents: %v\n", err)
		return
	}
	data, err := collection.Get(context.TODO(), nil, nil, nil, nil)
	if err != nil {
		log.Fatalf("Error getting documents: %v\n", err)
		return
	}
	// see GetResults struct for more details
	fmt.Printf("Collection data: %v\n", data)
}
```

### Query Collection

Here's a simple example of querying documents in a collection:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	chroma "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/openai"
)

func main() {
	ctx := context.Background()
	client := chroma.NewClient("localhost:8000")

	openaiEf, err := openai.NewOpenAIEmbeddingFunction(os.Getenv("OPENAI_API_KEY"))
	if err != nil {
		log.Fatalf("Error creating OpenAI embedding function: %s \n", err)
	}
	// Get the collection we created earlier
	collection, err := client.GetCollection(ctx, "my-collection", openaiEf)
	if err != nil {
		log.Fatalf("Failed to create collection: %v", err)
		return
	}
	data, err := collection.Query(context.TODO(), []string{"I love dogs"}, 5, nil, nil, nil)
	if err != nil {
		log.Fatalf("Error querying documents: %v\n", err)
		return
	}
	// see QueryResults struct for more details
	fmt.Printf("Collection data: %v\n", data)
}
```

### Delete Documents

Here's a simple example of deleting documents from a collection:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	chroma "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/openai"
)

func main() {
	ctx := context.Background()
	client := chroma.NewClient("localhost:8000")

	openaiEf, err := openai.NewOpenAIEmbeddingFunction(os.Getenv("OPENAI_API_KEY"))
	if err != nil {
		log.Fatalf("Error creating OpenAI embedding function: %s \n", err)
	}
	// Get the collection we created earlier
	collection, err := client.GetCollection(ctx, "my-collection", openaiEf)
	if err != nil {
		log.Fatalf("Failed to create collection: %v", err)
		return
	}
	_, err = collection.Delete(context.TODO(), []string{"ID1"}, nil, nil)
	if err != nil {
		log.Fatalf("Error deleting documents: %v\n", err)
		return
	}
	fmt.Printf("Documents deleted\n")
}
```
