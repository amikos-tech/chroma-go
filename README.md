# Chroma Go

A simple Chroma Vector Database client written in Go

Works with Chroma Version: v0.4.3 - v0.4.21

## Feature Parity with ChromaDB API

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
- 🚫 Authentication (Basic, Token with Authorization header, Token with X-Chroma-Token header)

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

or:

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

Consider the following example where:

- We create a new collection
- Add documents using OpenAI embedding function
- Query the collection using the same embedding function

```go
package main

import (
	"fmt"
	"log"
	"os"

	chroma "github.com/amikos-tech/chroma-go"
	openai "github.com/amikos-tech/chroma-go/openai"
	godotenv "github.com/joho/godotenv"
)

func main() {
	client := chroma.NewClient("http://localhost:8000")
	collectionName := "test-collection"
	metadata := map[string]interface{}{}
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Printf("Error loading .env file: %s", err)
		return
	}
	embeddingFunction := openai.NewOpenAIEmbeddingFunction(os.Getenv("OPENAI_API_KEY")) //create a new OpenAI Embedding function
	distanceFunction := chroma.L2
	_, errRest := client.Reset() //reset the database
	if errRest != nil {
		log.Fatalf("Error resetting database: %s \n", errRest.Error())
	}
	col, err := client.CreateCollection(collectionName, metadata, true, embeddingFunction, distanceFunction)
	if err != nil {
		fmt.Printf("Error create collection: %s \n", err.Error())
		return
	}
	documents := []string{
		"This is a document about cats. Cats are great.",
		"this is a document about dogs. Dogs are great.",
	}
	ids := []string{
		"ID1",
		"ID2",
	}

	metadatas := []map[string]interface{}{
		{"key1": "value1"},
		{"key2": "value2"},
	}
	_, addError := col.Add(nil, metadatas, documents, ids)
	if addError != nil {
		log.Fatalf("Error adding documents: %s \n", addError)
	}
	countDocs, qrerr := col.Count()
	if qrerr != nil {
		log.Fatalf("Error counting documents: %s \n", qrerr)
	}
	fmt.Printf("countDocs: %v\n", countDocs) //this should result in 2
	qr, qrerr := col.Query([]string{"I love dogs"}, 5, nil, nil, nil)
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
make gotest
```

### Generate ChromaDB API Client

```bash
make generate 
```


## References

- [Official Chroma documentation](https://docs.trychroma.com/)
- [Chroma Helm chart](https://github.com/amikos-tech/chromadb-chart) for cloud-native deployments
