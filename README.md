# Chroma Go

> [!WARNING]
> **V1 API Removed**: The V1 API has been removed in v0.3.0.
> If you require V1 API compatibility, please use versions prior to v0.3.0.
> ```bash
> go get github.com/amikos-tech/chroma-go@v0.2.4
> ```

A simple Chroma Vector Database client written in Go

Works with Chroma Version: v0.6.3+

We invite users to visit the docs site for the library for more in-depth
information: [Chroma Go Docs](https://go-client.chromadb.dev/)

## Feature Parity with ChromaDB API

| Operation                            | Support    |
|--------------------------------------|------------|
| Create Tenant                        | ✅          |
| Get Tenant                           | ✅          |
| Create Database                      | ✅          |
| Get Database                         | ✅          |
| Delete Database                      | ✅          |
| Reset                                | ✅          |
| Heartbeat                            | ✅          |
| List Collections                     | ✅          |
| Count Collections                    | ✅          |
| Get Version                          | ✅          |
| Create Collection                    | ✅          |
| Delete Collection                    | ✅          |
| Collection Add                       | ✅          |
| Collection Get                       | ✅          |
| Collection Count                     | ✅          |
| Collection Query                     | ✅          |
| Collection Update                    | ✅          |
| Collection Upsert                    | ✅          |
| Collection Delete (delete documents) | ✅          |
| Modify Collection                    | ⚒️ partial |
| Search API                           | ✅          |

Additional support features:

- ✅ [Authentication](https://go-client.chromadb.dev/auth/) (Basic, Token with Authorization header, Token with
  X-Chroma-Token header)
- ✅ [Private PKI and self-signed certificate support](https://go-client.chromadb.dev/client/)
- ✅ Chroma Cloud support
- ✅ [Structured Logging](https://go-client.chromadb.dev/logging/) - Injectable logger with Zap bridge for structured
  logging
- ⚒️ Persistent Embedding Function support (coming soon) - automatically load embedding function from Chroma collection
  configuration
- ⚒️ Persistent Client support (coming soon) - Run/embed full-featured Chroma in your go application without the need
  for Chroma server.
- ✅ [Search API Support](https://go-client.chromadb.dev/search/)

## Embedding API and Models Support

- ✅ [Default Embedding](https://go-client.chromadb.dev/embeddings/#default-embeddings) Support - the default
  `all-MiniLM-L6-v2` model running on Onnx Runtime (ORT).
- ✅ [OpenAI Embedding](https://go-client.chromadb.dev/embeddings/#openai) Support
- ✅ [Cohere](https://go-client.chromadb.dev/embeddings/#cohere) (including Multi-language support)
- ✅ [Sentence Transformers](https://go-client.chromadb.dev/embeddings/#huggingface-inference-api) (HuggingFace Inference
  API and [HFEI local server]())
- ✅ [Google Gemini Embedding](https://go-client.chromadb.dev/embeddings/#google-gemini-ai) Support
- ✅ [HuggingFace Embedding Inference Server Support](https://go-client.chromadb.dev/embeddings/#huggingface-embedding-inference-server)
- ✅ [Ollama Embedding](https://go-client.chromadb.dev/embeddings/#ollama) Support
- ✅ [Cloudflare Workers AI Embedding](https://go-client.chromadb.dev/embeddings/#cloudflare-workers-ai) Support
- ✅ [Together AI Embedding](https://go-client.chromadb.dev/embeddings/#together-ai) Support
- ✅ [Voyage AI Embedding](https://go-client.chromadb.dev/embeddings/#voyage-ai) Support
- ✅ [Mistral AI API Embedding](https://go-client.chromadb.dev/embeddings/#mistral-ai) Support
- ✅ [Nomic AI Embedding](https://go-client.chromadb.dev/embeddings/#nomic-ai) Support
- ✅ [Jina AI Embedding](https://go-client.chromadb.dev/embeddings/#jina-ai) Support
- ✅ [Roboflow CLIP Embedding](https://go-client.chromadb.dev/embeddings/#roboflow) Support (Multimodal: text + images)

## Reranking Functions

The Chroma Go client supports Reranking functions:

- ✅ [Cohere](https://go-client.chromadb.dev/rerankers/#cohere-reranker)
- ✅ [Jina AI](https://go-client.chromadb.dev/rerankers/#jina-ai-reranker)
- ✅ [HuggingFace Embedding Inference Server Reranker](https://go-client.chromadb.dev/rerankers/#hfei-Reranker)
- ✅ [Together AI](https://go-client.chromadb.dev/rerankers/#together-ai-reranker)

## Installation

```bash
go get github.com/amikos-tech/chroma-go
```

Import:

```go
import (
	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
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
helm install chroma chroma/chromadb --set chromadb.allowReset=true
```

> [!NOTE]
> To delete the minikube cluster: `minikube delete --profile chromago`

### Getting Started

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

	err = col.Delete(context.Background(), chroma.WithIDs("1", "2"))
	if err != nil {
		log.Fatalf("Error deleting collection: %s \n", err)
		return
	}
}
```

### Unified Options API

The V2 API provides a unified options pattern where common options work across multiple operations:

| Option              | Get | Query | Delete | Add | Update | Search |
|---------------------|-----|-------|--------|-----|--------|--------|
| `WithIDs`           | ✓   | ✓     | ✓      | ✓   | ✓      | ✓      |
| `WithWhere`         | ✓   | ✓     | ✓      |     |        |        |
| `WithWhereDocument` | ✓   | ✓     | ✓      |     |        |        |
| `WithInclude`       | ✓   | ✓     |        |     |        |        |
| `WithTexts`         |     |       |        | ✓   | ✓      |        |
| `WithMetadatas`     |     |       |        | ✓   | ✓      |        |
| `WithEmbeddings`    |     |       |        | ✓   | ✓      |        |

```go
// Get documents by ID or filter
results, _ := col.Get(ctx,
chroma.WithIDs("id1", "id2"),
chroma.WithWhere(chroma.EqString("status", "active")),
chroma.WithInclude(chroma.IncludeDocuments, chroma.IncludeMetadatas),
)

// Query with semantic search
results, _ := col.Query(ctx,
chroma.WithQueryTexts("machine learning"),
chroma.WithWhere(chroma.GtInt("year", 2020)),
chroma.WithNResults(10),
)

// Delete by filter
_ = col.Delete(ctx, chroma.WithWhere(chroma.EqString("status", "archived")))

// Search API with ranking and pagination
results, _ := col.Search(ctx,
chroma.NewSearchRequest(
chroma.WithKnnRank(chroma.KnnQueryText("query")),
chroma.WithFilter(chroma.EqString(chroma.K("category"), "tech")),
chroma.NewPage(chroma.Limit(20)),
chroma.WithSelect(chroma.KDocument, chroma.KScore),
),
)
```

### Structured Logging

The client supports injectable loggers for structured logging. Here's a quick example using Zap:

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
	devLogger, _ := chromalogger.NewDevelopmentZapLogger()
	debugClient, _ := chroma.NewHTTPClient(
		chroma.WithBaseURL("http://localhost:8000"),
		chroma.WithLogger(devLogger),
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

```bash
make test
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
