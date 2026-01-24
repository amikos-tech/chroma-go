# Embedding Functions - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/embeddings/embedding-functions)

## Overview

Embeddings are vector representations of data that enable similarity search. Chroma-go provides wrappers around popular embedding providers.

## Supported Embedding Providers

| Provider | Package | Import Path |
|----------|---------|-------------|
| Default (ONNX) | `default_ef` | `github.com/amikos-tech/chroma-go/pkg/embeddings/default_ef` |
| OpenAI | `openai` | `github.com/amikos-tech/chroma-go/pkg/embeddings/openai` |
| Cohere | `cohere` | `github.com/amikos-tech/chroma-go/pkg/embeddings/cohere` |
| HuggingFace TEI | `hf` | `github.com/amikos-tech/chroma-go/pkg/embeddings/hf` |
| Ollama | `ollama` | `github.com/amikos-tech/chroma-go/pkg/embeddings/ollama` |
| Jina AI | `jina` | `github.com/amikos-tech/chroma-go/pkg/embeddings/jina` |
| Mistral | `mistral` | `github.com/amikos-tech/chroma-go/pkg/embeddings/mistral` |
| Nomic | `nomic` | `github.com/amikos-tech/chroma-go/pkg/embeddings/nomic` |
| Voyage | `voyage` | `github.com/amikos-tech/chroma-go/pkg/embeddings/voyage` |
| Gemini | `gemini` | `github.com/amikos-tech/chroma-go/pkg/embeddings/gemini` |
| Cloudflare | `cloudflare` | `github.com/amikos-tech/chroma-go/pkg/embeddings/cloudflare` |
| Together AI | `together` | `github.com/amikos-tech/chroma-go/pkg/embeddings/together` |
| Chroma Cloud | `chromacloud` | `github.com/amikos-tech/chroma-go/pkg/embeddings/chromacloud` |

## Default Embedding Function

The default embedding function uses a local ONNX model.

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
collection = client.create_collection(name="my_collection")
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Collection uses default embedding function automatically
	collection, err := client.CreateCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	log.Printf("Created collection with default EF: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

## OpenAI Embedding Function

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb.utils.embedding_functions import OpenAIEmbeddingFunction

collection = client.create_collection(
    name="my_collection",
    embedding_function=OpenAIEmbeddingFunction(
        model_name="text-embedding-3-small"
    )
)

collection.add(
    ids=["id1", "id2"],
    documents=["doc1", "doc2"]
)
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"log"
	"os"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
)

func main() {
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create OpenAI embedding function
	ef, err := openai.NewOpenAIEmbeddingFunction(
		os.Getenv("OPENAI_API_KEY"),
		openai.WithModel(openai.TextEmbedding3Small),
	)
	if err != nil {
		log.Fatalf("Error creating embedding function: %v", err)
	}

	// Create collection with OpenAI embeddings
	collection, err := client.CreateCollection(ctx, "my_collection",
		v2.WithEmbeddingFunctionCreate(ef),
	)
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	// Add documents - OpenAI will embed them
	err = collection.Add(ctx,
		v2.WithIDs("id1", "id2"),
		v2.WithTexts("doc1", "doc2"),
	)
	if err != nil {
		log.Fatalf("Error adding documents: %v", err)
	}

	log.Println("Documents added with OpenAI embeddings")
}
```
{% /codetab %}
{% /codetabs %}

## Cohere Embedding Function

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"log"
	"os"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/cohere"
)

func main() {
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create Cohere embedding function
	ef, err := cohere.NewCohereEmbeddingFunction(
		os.Getenv("COHERE_API_KEY"),
		cohere.WithModel("embed-english-v3.0"),
	)
	if err != nil {
		log.Fatalf("Error creating embedding function: %v", err)
	}

	// Create collection with Cohere embeddings
	collection, err := client.CreateCollection(ctx, "my_collection",
		v2.WithEmbeddingFunctionCreate(ef),
	)
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	log.Printf("Created collection with Cohere EF: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

## Ollama Embedding Function (Local)

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/ollama"
)

func main() {
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create Ollama embedding function (local)
	ef, err := ollama.NewOllamaEmbeddingFunction(
		"http://localhost:11434",
		"nomic-embed-text",
	)
	if err != nil {
		log.Fatalf("Error creating embedding function: %v", err)
	}

	// Create collection with Ollama embeddings
	collection, err := client.CreateCollection(ctx, "my_collection",
		v2.WithEmbeddingFunctionCreate(ef),
	)
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	log.Printf("Created collection with Ollama EF: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

## HuggingFace Text Embedding Inference

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"log"
	"os"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/hf"
)

func main() {
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create HuggingFace TEI embedding function
	ef, err := hf.NewHuggingFaceEmbeddingFunction(
		os.Getenv("HF_API_KEY"),
		"BAAI/bge-small-en-v1.5",
	)
	if err != nil {
		log.Fatalf("Error creating embedding function: %v", err)
	}

	// Create collection
	collection, err := client.CreateCollection(ctx, "my_collection",
		v2.WithEmbeddingFunctionCreate(ef),
	)
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	log.Printf("Created collection with HuggingFace EF: %s", collection.Name())
}
```
{% /codetab %}
{% /codetabs %}

## Using Embedding Functions Directly

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb.utils.embedding_functions import DefaultEmbeddingFunction

default_ef = DefaultEmbeddingFunction()
embeddings = default_ef(["foo"])
print(embeddings)

collection.query(query_embeddings=embeddings)
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"log"
	"os"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
)

func main() {
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create embedding function
	ef, err := openai.NewOpenAIEmbeddingFunction(
		os.Getenv("OPENAI_API_KEY"),
		openai.WithModel(openai.TextEmbedding3Small),
	)
	if err != nil {
		log.Fatalf("Error creating embedding function: %v", err)
	}

	// Use embedding function directly
	embeddings, err := ef.EmbedDocuments(ctx, []string{"foo"})
	if err != nil {
		log.Fatalf("Error embedding: %v", err)
	}
	log.Printf("Embeddings: %v", embeddings)

	// Use embeddings in query
	collection, err := client.GetCollection(ctx, "my_collection")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	results, err := collection.Query(ctx,
		v2.WithQueryEmbeddings(embeddings[0]),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	log.Printf("Results: %v", results.GetIDsGroups())
}
```
{% /codetab %}
{% /codetabs %}

## Custom Embedding Functions

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
from chromadb import Documents, EmbeddingFunction, Embeddings

class MyEmbeddingFunction(EmbeddingFunction):
    def __call__(self, input: Documents) -> Embeddings:
        # embed the documents somehow
        return embeddings
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// MyEmbeddingFunction implements the EmbeddingFunction interface
type MyEmbeddingFunction struct {
	apiKey string
}

func NewMyEmbeddingFunction(apiKey string) *MyEmbeddingFunction {
	return &MyEmbeddingFunction{apiKey: apiKey}
}

// EmbedDocuments embeds a list of documents
func (ef *MyEmbeddingFunction) EmbedDocuments(ctx context.Context, docs []string) ([][]float32, error) {
	// Implement your embedding logic here
	embeddings := make([][]float32, len(docs))
	for i := range docs {
		// Generate embedding for each document
		embeddings[i] = []float32{0.1, 0.2, 0.3} // placeholder
	}
	return embeddings, nil
}

// EmbedQuery embeds a single query
func (ef *MyEmbeddingFunction) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	embeddings, err := ef.EmbedDocuments(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	return embeddings[0], nil
}

// Ensure it implements the interface
var _ embeddings.EmbeddingFunction = (*MyEmbeddingFunction)(nil)
```
{% /codetab %}
{% /codetabs %}

## All Available Providers

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
package main

import (
	"os"

	"github.com/amikos-tech/chroma-go/pkg/embeddings/cloudflare"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/cohere"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/gemini"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/hf"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/jina"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/mistral"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/nomic"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/ollama"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/together"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/voyage"
)

func createEmbeddingFunctions() {
	// OpenAI
	openaiEF, _ := openai.NewOpenAIEmbeddingFunction(
		os.Getenv("OPENAI_API_KEY"),
		openai.WithModel(openai.TextEmbedding3Small),
	)

	// Cohere
	cohereEF, _ := cohere.NewCohereEmbeddingFunction(
		os.Getenv("COHERE_API_KEY"),
		cohere.WithModel("embed-english-v3.0"),
	)

	// Ollama (local)
	ollamaEF, _ := ollama.NewOllamaEmbeddingFunction(
		"http://localhost:11434",
		"nomic-embed-text",
	)

	// HuggingFace TEI
	hfEF, _ := hf.NewHuggingFaceEmbeddingFunction(
		os.Getenv("HF_API_KEY"),
		"BAAI/bge-small-en-v1.5",
	)

	// Jina
	jinaEF, _ := jina.NewJinaEmbeddingFunction(
		jina.WithAPIKey(os.Getenv("JINA_API_KEY")),
		jina.WithModel("jina-embeddings-v3"),
		jina.WithTask(jina.TaskTextMatching),
		jina.WithLateChunking(true),
	)

	// Mistral
	mistralEF, _ := mistral.NewMistralEmbeddingFunction(
		os.Getenv("MISTRAL_API_KEY"),
		mistral.WithModel("mistral-embed"),
	)

	// Nomic
	nomicEF, _ := nomic.NewNomicEmbeddingFunction(
		os.Getenv("NOMIC_API_KEY"),
	)

	// Voyage
	voyageEF, _ := voyage.NewVoyageEmbeddingFunction(
		os.Getenv("VOYAGE_API_KEY"),
		voyage.WithModel("voyage-2"),
	)

	// Gemini
	geminiEF, _ := gemini.NewGeminiEmbeddingFunction(
		os.Getenv("GEMINI_API_KEY"),
	)

	// Cloudflare
	cloudflareEF, _ := cloudflare.NewCloudflareEmbeddingFunction(
		os.Getenv("CF_ACCOUNT_ID"),
		os.Getenv("CF_API_TOKEN"),
		"@cf/baai/bge-base-en-v1.5",
	)

	// Together AI
	togetherEF, _ := together.NewTogetherEmbeddingFunction(
		os.Getenv("TOGETHER_API_KEY"),
		together.WithModel("togethercomputer/m2-bert-80M-8k-retrieval"),
	)

	_ = openaiEF
	_ = cohereEF
	_ = ollamaEF
	_ = hfEF
	_ = jinaEF
	_ = mistralEF
	_ = nomicEF
	_ = voyageEF
	_ = geminiEF
	_ = cloudflareEF
	_ = togetherEF
}
```
{% /codetab %}
{% /codetabs %}

## Notes

- API keys are typically read from environment variables
- Use `defer client.Close()` to properly release embedding function resources
- Embedding dimensions must match when adding to existing collections
- The default embedding function uses ONNX Runtime locally
- Set `CHROMAGO_ONNX_RUNTIME_PATH` to use a custom ONNX Runtime library
