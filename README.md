# Chroma Go

A simple Chroma Vector Database client written in Go

Works with Chroma Version: v0.4.3 - v0.4.24

## Feature Parity with ChromaDB API

- ✅ Create Tenant
- ✅ Get Tenant
- ✅ Create Database
- ✅ Get Database
- ✅ Reset
- ✅ Heartbeat
- ✅ List Collections
- ✅ Get Version
- ✅ Create Collection
- ✅ Delete Collection
- ✅ Collection Add
- ✅ Collection Get (partial without additional parameters)
- ✅ Collection Count
- ✅ Collection Query
- ✅ Collection Modify Embeddings
- ✅ Collection Update
- ✅ Collection Upsert
- ✅ Collection Delete - delete documents in collection
- ✅ Authentication (Basic, Token with Authorization header, Token with X-Chroma-Token header)

## Embedding Functions Support

- ✅ OpenAI API
- ✅ Cohere API (including Multi-language support)
- ✅ Sentence Transformers (HuggingFace Inference API)
- 🚫 PaLM API
- 🚫 Custom Embedding Function
- 🚫 Hugging Face Inference Embedding Function

## Installation

```bash
go get github.com/amikos-tech/chroma-go
```

Import:

```go
import (
    chroma "github.com/amikos-tech/chroma-go"
)
```

## Usage


Ensure you have a running instance of Chroma running. We recommend one of the two following options:

- [Official documentation](https://docs.trychroma.com/usage-guide#running-chroma-in-clientserver-mode)
- If you are a fan of Kubernetes, you can use the [Helm chart](https://github.com/amikos-tech/chromadb-chart) (Note: You
  will need `Docker`, `minikube` and `kubectl` installed)


**The Setup (Cloud-native):**

```bash
minikube start --profile chromago
minikube profile chromago
helm repo add chroma https://amikos-tech.github.io/chromadb-chart/
helm repo update
helm install chroma chroma/chromadb --set chromadb.allowReset=true,chromadb.apiVersion=0.4.5
```

|**Note:** To delete the minikube cluster: `minikube delete --profile chromago`

### Getting Started

Consider the following example where:

- We create a new collection
- Add documents using OpenAI embedding function
- Query the collection using the same embedding function

```go
package main

import (
  "context"
  "fmt"
  "log"
  "os"

  chroma "github.com/amikos-tech/chroma-go"
  "github.com/amikos-tech/chroma-go/collection"
  openai "github.com/amikos-tech/chroma-go/openai"
  "github.com/amikos-tech/chroma-go/types"
)

func main() {
	// Create new OpenAI embedding function
	
    openaiEf, err := openai.NewOpenAIEmbeddingFunction(os.Getenv("OPENAI_API_KEY"))
    if err != nil {
      log.Fatalf("Error creating OpenAI embedding function: %s \n", err)
    }
    // Create a new Chroma client
    client := chroma.NewClient("localhost:8000")
	
    // Create a new collection with options
    newCollection, err := client.NewCollection(
      context.TODO(),
      collection.WithName("test-collection"),
      collection.WithMetadata("key1", "value1"),
      collection.WithEmbeddingFunction(openaiEf),
      collection.WithHNSWDistanceFunction(types.L2),
    )
    if err != nil {
      log.Fatalf("Error creating collection: %s \n", err)
    }
	
    // Create a new record set with to hold the records to insert
    rs, err := types.NewRecordSet(
      types.WithEmbeddingFunction(openaiEf),
      types.WithIDGenerator(types.NewULIDGenerator()),
    )
    if err != nil {
      log.Fatalf("Error creating record set: %s \n", err)
    }
    // Add a few records to the record set
    rs.WithRecord(types.WithDocument("My name is John. And I have two dogs."), types.WithMetadata("key1", "value1"))
    rs.WithRecord(types.WithDocument("My name is Jane. I am a data scientist."), types.WithMetadata("key2", "value2"))
	
	// Build and validate the record set (this will create embeddings if not already present)
    _, err = rs.BuildAndValidate(context.TODO())
    if err != nil {
      log.Fatalf("Error validating record set: %s \n", err)
    }
	
	// Add the records to the collection
    _, err = newCollection.AddRecords(context.Background(), rs)
    if err != nil {
      log.Fatalf("Error adding documents: %s \n", err)
    }
	
	// Count the number of documents in the collection
    countDocs, qrerr := newCollection.Count(context.TODO())
    if qrerr != nil {
      log.Fatalf("Error counting documents: %s \n", qrerr)
    }
	
	// Query the collection
    fmt.Printf("countDocs: %v\n", countDocs) //this should result in 2
    qr, qrerr := newCollection.Query(context.TODO(),[]string{"I love dogs"}, 5, nil, nil, nil)
    if qrerr != nil {
      log.Fatalf("Error querying documents: %s \n", qrerr)
    }
    fmt.Printf("qr: %v\n", qr.Documents[0][0]) //this should result in the document about dogs
}
```

## Development

### Build

```bash
make build
```

### Test

```bash
make test
```

### Generate ChromaDB API Client

```bash
make generate 
```

### Lint

```bash
make lint-fix
```

### Local Server

> Note: Docker must be installed

```bash
make server
```

## References

- [Official Chroma documentation](https://docs.trychroma.com/)
- [Chroma Helm chart](https://github.com/amikos-tech/chromadb-chart) for cloud-native deployments
- [Chroma Cookbook](https://cookbook.chromadb.dev) for examples and recipes
