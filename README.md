# Chroma Go

> [!WARNING]
> **V1 API Deprecation Notice**: The V1 API will be removed in the upcoming v0.3.0 release.
> Users currently using the V1 API (root package import) should migrate to the V2 API (`github.com/amikos-tech/chroma-go/pkg/api/v2`).
> The V2 API offers feature parity with V1 plus additional capabilities.
> For continued V1 support, please use v0.2.x releases.

A simple Chroma Vector Database client written in Go

Works with Chroma Version: v0.4.3 - v1.0.x

We invite users to visit the docs site for the library for more in-depth
information: [Chroma Go Docs](https://go-client.chromadb.dev/)

> [!NOTE]
> v0.2.0 documentation is still being updated. Please consult the tests under `pkg/api/v2` for more detailed usage
> examples. We are working on updating the documentation with full usage examples (also feel free to contribute if you
> have any examples you would like to share).

## Feature Parity with ChromaDB API

| Operation                            | V1 support (Self-Hosted) | V2 support (Self-Hosted) | Cloud |
|--------------------------------------|--------------------------|-------------|-------|
| Create Tenant                        | ✅                        | ✅           | ✅     |
| Get Tenant                           | ✅                        | ✅           | ✅     |
| Create Database                      | ✅                        | ✅           | ✅     |
| Get Database                         | ✅                        | ✅           | ✅     |
| Delete Database                      | ❌                        | ✅           | ✅     |
| Reset                                | ✅                        | ✅           | ✅     |
| Heartbeat                            | ✅                        | ✅           | ✅     |
| List Collections                     | ✅                        | ✅           | ✅     |
| Count Collections                    | ✅                        | ✅           | ✅     |
| Get Version                          | ✅                        | ✅           | ✅     |
| Create Collection                    | ✅                        | ✅           | ✅     |
| Delete Collection                    | ✅                        | ✅           | ✅     |
| Collection Add                       | ✅                        | ✅           | ✅     |
| Collection Get                       | ✅                        | ✅           | ✅     |
| Collection Count                     | ✅                        | ✅           | ✅     |
| Collection Query                     | ✅                        | ✅           | ✅     |
| Collection Update                    | ✅                        | ✅           | ✅     |
| Collection Upsert                    | ✅                        | ✅           | ✅     |
| Collection Delete (delete documents) | ✅                        | ✅           | ✅     |
| Modify  Collection                   | ✅                        | ⚒️ partial  | ✅     |
| Search API                           | ❌                        | ❌           | ✅     |

Additional support features:

- ✅ [Authentication](https://go-client.chromadb.dev/auth/) (Basic, Token with Authorization header, Token with
  X-Chroma-Token header)
- ✅ [Private PKI and self-signed certificate support](https://go-client.chromadb.dev/client/)
- ✅ Chroma Cloud support
- 🔥✅ [Structured Logging](https://go-client.chromadb.dev/logging/) (V2 API only) - Injectable logger with Zap bridge for structured logging
- ⚒️ Persistent Embedding Function support (coming soon) - automatically load embedding function from Chroma collection
  configuration
- ⚒️ Persistent Client support (coming soon) - Run/embed full-featured Chroma in your go application without the need
  for Chroma server.
- 🔥✅ [Search API Support](https://go-client.chromadb.dev/search/) - Since `0.3.0-alpha.1`+, we also support the Chroma Search API.

## Embedding API and Models Support

- 🔥✅ [Default Embedding](https://go-client.chromadb.dev/embeddings/#default-embeddings) Support - Since `0.2.0`+, we
  also support the default `all-MiniLM-L6-v2` model running on Onnx Runtime (ORT).
- ✅ [OpenAI Embedding](https://go-client.chromadb.dev/embeddings/#openai) Support
- ✅ [Cohere](https://go-client.chromadb.dev/embeddings/#cohere) (including Multi-language support)
- ✅ [Sentence Transformers](https://go-client.chromadb.dev/embeddings/#huggingface-inference-api) (HuggingFace Inference
  API and [HFEI local server]())
- ✅ [Google Gemini Embedding](https://go-client.chromadb.dev/embeddings/#google-gemini-ai) Support
- 🚫 Custom Embedding Function
-
✅ [HuggingFace Embedding Inference Server Support](https://go-client.chromadb.dev/embeddings/#huggingface-embedding-inference-server)
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
- ✅ [Together AI](https://go-client.chromadb.dev/rerankers/#together-ai-reranker)

## Installation

> [!IMPORTANT]  
> There are many new changes leading up to `v0.2.0`, as documented below. If you'd like to use them please install the
> latest version of the client.
> ```bash
> go get github.com/amikos-tech/chroma-go@main
> ```

```bash
go get github.com/amikos-tech/chroma-go
```

Import `v1`:

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
helm install chroma chroma/chromadb --set chromadb.allowReset=true,chromadb.apiVersion=0.
```

> [!NOTE]
> To delete the minikube cluster: `minikube delete --profile chromago`

### Getting Started (Chroma API v2)

- We create a new collection
- Add documents using the default embedding function
- Query the collection using the same embedding function
- Delete documents from the collection

```go
package main

import (
	"context"
	"fmt"
	"log"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Create a new Chroma client
	client, err := chroma.NewHTTPClient()
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
	col, err := client.GetOrCreateCollection(context.Background(), "col1",
		chroma.WithCollectionMetadataCreate(
			chroma.NewMetadata(
				chroma.NewStringAttribute("str", "hello"),
				chroma.NewIntAttribute("int", 1),
				chroma.NewFloatAttribute("float", 1.1),
			),
		),
	)
	if err != nil {
		log.Fatalf("Error creating collection: %s \n", err)
		return
	}

	err = col.Add(context.Background(),
		//chroma.WithIDGenerator(chroma.NewULIDGenerator()),
		chroma.WithIDs("1", "2"),
		chroma.WithTexts("hello world", "goodbye world"),
		chroma.WithMetadatas(
			chroma.NewDocumentMetadata(chroma.NewIntAttribute("int", 1)),
			chroma.NewDocumentMetadata(chroma.NewStringAttribute("str", "hello")),
		))
	if err != nil {
		log.Fatalf("Error adding collection: %s \n", err)
	}

	count, err := col.Count(context.Background())
	if err != nil {
		log.Fatalf("Error counting collection: %s \n", err)
		return
	}
	fmt.Printf("Count collection: %d\n", count)

	qr, err := col.Query(context.Background(), chroma.WithQueryTexts("say hello"))
	if err != nil {
		log.Fatalf("Error querying collection: %s \n", err)
		return
	}
	fmt.Printf("Query result: %v\n", qr.GetDocumentsGroups()[0][0])

	err = col.Delete(context.Background(), chroma.WithIDsDelete("1", "2"))
	if err != nil {
		log.Fatalf("Error deleting collection: %s \n", err)
		return
	}
}
```

### Structured Logging (V2 API)

The V2 API client supports injectable loggers for structured logging. Here's a quick example using Zap:

```go
package main

import (
	"context"
	"log"

	"go.uber.org/zap"
	chromalogger "github.com/amikos-tech/chroma-go/pkg/logger"
	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Create a zap logger
	zapLogger, _ := zap.NewDevelopment()
	defer zapLogger.Sync()

	// Wrap it in the Chroma logger
	logger := chromalogger.NewZapLogger(zapLogger)

	// Create client with the logger
	client, err := chroma.NewHTTPClient(
		chroma.WithBaseURL("http://localhost:8000"),
		chroma.WithLogger(logger),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// All client operations will now be logged with structured logging
	ctx := context.Background()
	collections, _ := client.ListCollections(ctx)

	// You can also log directly
	logger.Info("Retrieved collections",
		chromalogger.Int("count", len(collections)),
	)

	// For debug logging, use WithLogger with a debug-level logger
	// Note: WithDebug() is deprecated - use WithLogger instead
	devLogger, _ := chromalogger.NewDevelopmentZapLogger()
	debugClient, _ := chroma.NewHTTPClient(
		chroma.WithBaseURL("http://localhost:8000"),
		chroma.WithLogger(devLogger), // Use a logger with debug level enabled
	)
	defer debugClient.Close()
}
```

See the [logging documentation](https://go-client.chromadb.dev/logging/) for more details.

## Development

### Build

```bash
make build
```

### Test

**V1 API:**

```bash
make test
```

**V2 API:**

```bash
make test-v2
```

### Generate ChromaDB API Client (only for v1)

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
