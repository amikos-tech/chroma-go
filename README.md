# Chroma Go

> [!CAUTION]  
> The current release does not yet work with API v2 of Chroma. We're working to fix this very soon. In the meantime the client will work fine with version <=0.6.2 of Chroma.

A simple Chroma Vector Database client written in Go

Works with Chroma Version: v0.4.3 - v0.6.2

We invite users to visit the docs site for the library for more in-depth
information: [Chroma Go Docs](https://go-client.chromadb.dev/)

## Feature Parity with ChromaDB API

- ✅ Create Tenant
- ✅ Get Tenant
- ✅ Create Database
- ✅ Get Database
- ✅ Reset
- ✅ Heartbeat
- ✅ List Collections
- ✅ Count Collections
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
- ✅ [Authentication](https://go-client.chromadb.dev/auth/) (Basic, Token with Authorization header, Token with X-Chroma-Token header)
- ✅ [Private PKI and self-signed certificate support](https://go-client.chromadb.dev/client/)

## Embedding API and Models Support

- 🔥✅ [Default Embedding](https://go-client.chromadb.dev/embeddings/#default-embeddings) Support - Since `0.2.0`+, we also support the default `all-MiniLM-L6-v2` model running on Onnx Runtime (ORT). 
- ✅ [OpenAI Embedding](https://go-client.chromadb.dev/embeddings/#openai) Support
- ✅ [Cohere](https://go-client.chromadb.dev/embeddings/#cohere) (including Multi-language support)
- ✅ [Sentence Transformers](https://go-client.chromadb.dev/embeddings/#huggingface-inference-api) (HuggingFace Inference API and [HFEI local server]())
- ✅ [Google Gemini Embedding](https://go-client.chromadb.dev/embeddings/#google-gemini-ai) Support
- 🚫 Custom Embedding Function
- ✅ [HuggingFace Embedding Inference Server Support](https://go-client.chromadb.dev/embeddings/#huggingface-embedding-inference-server)
- ✅ [Ollama Embedding](https://go-client.chromadb.dev/embeddings/#ollama) Support
- ✅ [Cloudflare Workers AI Embedding](https://go-client.chromadb.dev/embeddings/#cloudflare-workers-ai) Support
- ✅ [Together AI Embedding](https://go-client.chromadb.dev/embeddings/#together-ai) Support
- ✅ [Voyage AI Embedding](https://go-client.chromadb.dev/embeddings/#voyage-ai) Support
- ✅ [Mistral AI API Embedding](https://go-client.chromadb.dev/embeddings/#mistral-ai) Support
- ✅ [Nomic AI Embedding](https://go-client.chromadb.dev/embeddings/#nomic-ai) Support
- ✅ [Jina AI Embedding](https://go-client.chromadb.dev/embeddings/#jina-ai) Support

## Reranking Functions

From release `0.2.0` the Chroma Go client also supports Reranking functions. The following are supported:

- ✅ [Cohere](https://go-client.chromadb.dev/rerankers/#cohere-reranker)
- ✅ [Jina AI](https://go-client.chromadb.dev/rerankers/#jina-ai-reranker)
- ✅ [HuggingFace Embedding Inference Server Reranker](https://go-client.chromadb.dev/rerankers/#hfei-Reranker)

## Installation

> [!IMPORTANT]  
> There are many new changes leading up to `v0.2.0`, as documented below. If you'd like to use them please install the latest version of the client.
> ```bash
> go get github.com/amikos-tech/chroma-go@main
> ```

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

- [Official documentation](https://docs.trychroma.com/guides#running-chroma-in-client/server-mode)
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
- Add documents using the default embedding function
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
	openai "github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
	"github.com/amikos-tech/chroma-go/types"
)

func main() {
	// Create a new Chroma client
	client,err := chroma.NewClient(chroma.WithBasePath("http://localhost:8000"))
	if err != nil {
        log.Fatalf("Error creating client: %s \n", err)
		return
    }
	// Close the client to release any resources such as local embedding functions
	defer func() {
		err = client.Close()
		if err != nil {
			log.Fatalf("Error closing client: %s \n", err)
        }
    }()
	

	// Create a new collection with options. We don't provide an embedding function here, so the default embedding function will be used
	newCollection, err := client.NewCollection(
		context.TODO(),
        "test-collection",
		collection.WithMetadata("key1", "value1"),
		collection.WithHNSWDistanceFunction(types.L2),
	)
	if err != nil {
		log.Fatalf("Error creating collection: %s \n", err)
	}

	// Create a new record set with to hold the records to insert
	rs, err := types.NewRecordSet(
		types.WithEmbeddingFunction(newCollection.EmbeddingFunction), // we pass the embedding function from the collection
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
	qr, qrerr := newCollection.Query(context.TODO(), []string{"I love dogs"}, 5, nil, nil, nil)
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
