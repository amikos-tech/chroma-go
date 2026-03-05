# Chroma Go

A simple Chroma Vector Database client written in Go.
Current `chroma-go` release lines (`v0.3.x` and `v0.4.x`) are compatible with Chroma `v1.x`.
For older Chroma versions, use older `chroma-go` releases (for example `v0.2.x`). See [compatibility](#compatibility).

> [!WARNING]
> **V1 API Removed**: The V1 API is removed in `v0.3.x` and later releases.
> If you require V1 API compatibility, please use versions prior to `v0.3.0` (for example `v0.2.x`).
> ```bash
> go get github.com/amikos-tech/chroma-go@v0.2.4
> ```

We invite users to visit the docs site for the library for more in-depth
information: [Chroma Go Docs](https://go-client.chromadb.dev/)

## Compatibility

- `chroma-go` `v0.3.x` and `v0.4.x` are compatible with Chroma `v1.x`.
- For older Chroma versions, use older `chroma-go` release lines (for example `v0.2.x`).
- Older client versions: [GitHub Releases](https://github.com/amikos-tech/chroma-go/releases)

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

## Quick Start

### Persistent Client

Run Chroma locally in-process (no external server) with `NewPersistentClient`.
The runtime auto-downloads the correct shim library on first use and caches it under `~/.cache/chroma/local_shim`.
Override with `CHROMA_LIB_PATH` or `WithPersistentLibraryPath(...)`.

```go
package main

import (
	"context"
	"fmt"
	"log"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	client, err := chroma.NewPersistentClient(
		chroma.WithPersistentPath("./chroma_data"),
	)
	if err != nil {
		log.Fatalf("Error creating client: %s", err)
	}
	defer client.Close()

	col, err := client.GetOrCreateCollection(context.Background(), "my_collection")
	if err != nil {
		log.Fatalf("Error creating collection: %s", err)
	}

	err = col.Add(context.Background(),
		chroma.WithIDs("1", "2"),
		chroma.WithTexts("hello world", "goodbye world"),
	)
	if err != nil {
		log.Fatalf("Error adding documents: %s", err)
	}

	qr, err := col.Query(context.Background(),
		chroma.WithQueryTexts("say hello"),
		chroma.WithNResults(1),
		chroma.WithInclude(chroma.IncludeDocuments),
	)
	if err != nil {
		log.Fatalf("Error querying: %s", err)
	}
	fmt.Printf("Result: %v\n", qr.GetDocumentsGroups()[0][0])
}
```

Full runnable example: [`examples/v2/persistent_client`](https://github.com/amikos-tech/chroma-go/tree/main/examples/v2/persistent_client)

### Self-Hosted (HTTP)

Connect to a Chroma server running on `http://localhost:8000`:

```bash
docker run -d --name chroma -p 8000:8000 -e ALLOW_RESET=TRUE chromadb/chroma:latest
```

Then create the client (default Chroma URL: `http://localhost:8000`):

```go
package main

import (
	"context"
	"fmt"
	"log"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	client, err := chroma.NewHTTPClient(
		chroma.WithBaseURL("http://localhost:8000"),
	)
	if err != nil {
		log.Fatalf("Error creating client: %s", err)
	}
	defer client.Close()

	col, err := client.GetOrCreateCollection(context.Background(), "my_collection")
	if err != nil {
		log.Fatalf("Error creating collection: %s", err)
	}

	err = col.Add(context.Background(),
		chroma.WithIDs("1", "2"),
		chroma.WithTexts("hello world", "goodbye world"),
	)
	if err != nil {
		log.Fatalf("Error adding documents: %s", err)
	}

	qr, err := col.Query(context.Background(),
		chroma.WithQueryTexts("say hello"),
		chroma.WithNResults(1),
		chroma.WithInclude(chroma.IncludeDocuments),
	)
	if err != nil {
		log.Fatalf("Error querying: %s", err)
	}
	fmt.Printf("Result: %v\n", qr.GetDocumentsGroups()[0][0])
}
```

Stop the local container when done: `docker stop chroma && docker rm chroma`.
Alternative local startup helper: `make server` (requires Docker).
See the [official documentation](https://docs.trychroma.com/guides#running-chroma-in-client/server-mode) for other deployment options.

### Chroma Cloud

Connect to [Chroma Cloud](https://www.trychroma.com/) using your API key:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	client, err := chroma.NewCloudClient(
		chroma.WithCloudAPIKey(os.Getenv("CHROMA_API_KEY")),
		chroma.WithDatabaseAndTenant(
			os.Getenv("CHROMA_DATABASE"),
			os.Getenv("CHROMA_TENANT"),
		),
	)
	if err != nil {
		log.Fatalf("Error creating client: %s", err)
	}
	defer client.Close()

	col, err := client.GetOrCreateCollection(context.Background(), "my_collection")
	if err != nil {
		log.Fatalf("Error creating collection: %s", err)
	}

	fmt.Printf("Collection: %s\n", col.Name())
}
```

Full auth example: [`examples/v2/auth`](https://github.com/amikos-tech/chroma-go/tree/main/examples/v2/auth)

## Examples

| Example | Path | Entry | Focus |
|---------|------|-------|-------|
| Basic usage | [`examples/v2/basic`](./examples/v2/basic) | `main.go` | CRUD flow with `NewHTTPClient` |
| Persistent client | [`examples/v2/persistent_client`](./examples/v2/persistent_client) | `main.go` | Local embedded runtime with `NewPersistentClient` |
| Authentication | [`examples/v2/auth`](./examples/v2/auth) | `main.go` | Basic/token/cloud auth patterns |
| Tenant and database | [`examples/v2/tenant_and_db`](./examples/v2/tenant_and_db) | `main.go` | Multi-tenant and database scoping |
| Metadata filters | [`examples/v2/metadata_filters`](./examples/v2/metadata_filters) | `main.go` | `where` filters and query conditions |
| Array metadata | [`examples/v2/array_metadata`](./examples/v2/array_metadata) | `main.go` | Array metadata and contains operators |
| Schema | [`examples/v2/schema`](./examples/v2/schema) | `main.go` | Schema/index configuration |
| Search API | [`examples/v2/search`](./examples/v2/search) | `main.go` | Ranking/filtering/pagination search flow |
| Embedding functions | [`examples/v2/embedding_function_basic`](./examples/v2/embedding_function_basic) | `main.go` | Built-in embedding function setup |
| Custom embedding function | [`examples/v2/custom_embedding_function`](./examples/v2/custom_embedding_function) | `README.md` | Custom embedder integration guide |
| Reranking functions | [`examples/v2/reranking_function_basic`](./examples/v2/reranking_function_basic) | `README.md` | Reranker usage patterns |
| Logging (Zap) | [`examples/v2/logging`](./examples/v2/logging) | `main.go` | Structured logging with Zap |
| Logging (slog) | [`examples/v2/logging_slog`](./examples/v2/logging_slog) | `main.go` | Structured logging with `log/slog` |

## Offline / Air-Gapped Environments

The default embedding function and persistent client runtime require native libraries that are normally downloaded on first use.
For offline or air-gapped environments, pre-download all runtime dependencies:

```bash
./scripts/fetch_runtime_deps.sh
```

Then run the offline smoke test to verify:

```bash
make offline-smoke
```

See [Offline Runtime Bundle](./docs/docs/offline-runtime-bundle.md) for full details and available flags.

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
| Modify Collection                    | ✅          |
| Search API                           | ✅          |

Additional support features:

- ✅ [Authentication](https://go-client.chromadb.dev/auth/) (Basic, Token with Authorization header, Token with
  X-Chroma-Token header)
- ✅ [Private PKI and self-signed certificate support](https://go-client.chromadb.dev/client/)
- ✅ Chroma Cloud support
- ✅ [Structured Logging](https://go-client.chromadb.dev/logging/) - Injectable logger with Zap bridge for structured
  logging
- ✅ Persistent Embedding Function support - automatically load embedding function from Chroma collection
  configuration
- ✅ Persistent Client support - Run/embed full-featured Chroma in your Go application without running an external
  Chroma server process.
- ✅ [Search API Support](https://go-client.chromadb.dev/search/)
- ✅ Array Metadata support with `$contains`/`$not_contains` operators (Chroma >= 1.5.0)

## Embedding API and Models Support

- ✅ [Default Embedding](https://go-client.chromadb.dev/embeddings/#default-embeddings) Support - the default
  `all-MiniLM-L6-v2` model running on Onnx Runtime (ORT).
- ✅ [Offline Runtime Setup Script](./docs/docs/offline-runtime-bundle.md) - download and cache default embedding runtime files locally before running smoke/offline workflows.
- ✅ [OpenAI Embedding](https://go-client.chromadb.dev/embeddings/#openai) Support
- ✅ [Cohere](https://go-client.chromadb.dev/embeddings/#cohere) (including Multi-language support)
- ✅ [Sentence Transformers](https://go-client.chromadb.dev/embeddings/#huggingface-inference-api) (HuggingFace Inference
  API and [HFEI local server](https://go-client.chromadb.dev/embeddings/#huggingface-embedding-inference-server))
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
- ✅ [Amazon Bedrock Embedding](https://go-client.chromadb.dev/embeddings/#amazon-bedrock) Support (Titan models, bearer token + SDK auth)
- ✅ [Baseten Embedding](https://go-client.chromadb.dev/embeddings/#baseten) Support
- ✅ [Morph Embedding](https://go-client.chromadb.dev/embeddings/#morph) Support
- ✅ [Perplexity Embedding](https://go-client.chromadb.dev/embeddings/#perplexity) Support

**Sparse & Specialized Embedding Functions:**

- ✅ [Chroma Cloud Embedding](https://go-client.chromadb.dev/embeddings/#chroma-cloud) Support
- ✅ [Chroma Cloud Splade Embedding](https://go-client.chromadb.dev/embeddings/#chroma-cloud-splade) Support (sparse)
- ✅ [BM25 Embedding](https://go-client.chromadb.dev/embeddings/#bm25) Support (sparse)

## Reranking Functions

The Chroma Go client supports Reranking functions:

- ✅ [Cohere](https://go-client.chromadb.dev/rerankers/#cohere-reranker)
- ✅ [Jina AI](https://go-client.chromadb.dev/rerankers/#jina-ai-reranker)
- ✅ [HuggingFace Embedding Inference Server Reranker](https://go-client.chromadb.dev/rerankers/#hfei-Reranker)
- ✅ [Together AI](https://go-client.chromadb.dev/rerankers/#together-ai-reranker)

## Schema Quickstart

```go
// NewSchemaWithDefaults (L2 + HNSW defaults)
schema, err := chroma.NewSchemaWithDefaults()
if err != nil {
	panic(err)
}
```

```go
// Custom schema: vector + FTS + metadata indexes
schema, err := chroma.NewSchema(
	chroma.WithDefaultVectorIndex(chroma.NewVectorIndexConfig(
		chroma.WithSpace(chroma.SpaceCosine),
		chroma.WithHnsw(chroma.NewHnswConfig(
			chroma.WithEfConstruction(200),
			chroma.WithMaxNeighbors(32),
		)),
	)),
	chroma.WithDefaultFtsIndex(&chroma.FtsIndexConfig{}),
	chroma.WithStringIndex("category"),
	chroma.WithIntIndex("year"),
	chroma.WithFloatIndex("rating"),
)
if err != nil {
	panic(err)
}
```

```go
// Disable an index for one field
schema, err := chroma.NewSchema(
	chroma.WithDefaultVectorIndex(chroma.NewVectorIndexConfig(chroma.WithSpace(chroma.SpaceL2))),
	chroma.DisableStringIndex("large_text_field"),
)
if err != nil {
	panic(err)
}
```

```go
// SPANN (Chroma Cloud)
schema, err := chroma.NewSchema(
	chroma.WithDefaultVectorIndex(chroma.NewVectorIndexConfig(
		chroma.WithSpace(chroma.SpaceCosine),
		chroma.WithSpann(chroma.NewSpannConfig(
			chroma.WithSpannSearchNprobe(64),
			chroma.WithSpannEfConstruction(200),
		)),
	)),
)
if err != nil {
	panic(err)
}
```

Runnable schema example: [`examples/v2/schema`](https://github.com/amikos-tech/chroma-go/tree/main/examples/v2/schema)

### Strict Metadata Map Validation

When metadata comes from `map[string]interface{}`:

- `NewMetadataFromMap` is best-effort and silently skips invalid `[]interface{}` values.
- `NewMetadataFromMapStrict` returns an error for invalid or unsupported values.
- `WithCollectionMetadataMapCreateStrict` applies strict conversion in create/get-or-create flows and returns a deferred option error before any HTTP request is sent.

```go
// Strict create/get-or-create metadata map conversion
col, err := client.GetOrCreateCollection(context.Background(), "col1",
	chroma.WithCollectionMetadataMapCreateStrict(map[string]interface{}{
		"description": "validated metadata",
		"tags":        []interface{}{"a", "b"},
	}),
)
if err != nil {
	log.Fatalf("Error creating collection: %s", err)
}

// Strict metadata map conversion before collection metadata update
newMetadata, err := chroma.NewMetadataFromMapStrict(map[string]interface{}{
	"description": "updated description",
	"tags":        []interface{}{"x", "y"},
})
if err != nil {
	log.Fatalf("Invalid metadata map: %s", err)
}
if err := col.ModifyMetadata(context.Background(), newMetadata); err != nil {
	log.Fatalf("Error modifying metadata: %s", err)
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

### Performance Validation

Local/persistent runtime soak/load validation is available through a dedicated
soak-tagged test harness.

Smoke profile (PR gate, hard-fail thresholds, <=10m timeout target):

```bash
make test-local-load-smoke
```

Nightly soak profile (report-only thresholds, <=70m timeout target):

```bash
make test-local-soak-nightly
```

Useful environment variables:

- `CHROMA_PERF_PROFILE` - `smoke` or `soak`.
- `CHROMA_PERF_ENFORCE` - `true` to fail on threshold breaches, `false` for report-only.
- `CHROMA_PERF_INCLUDE_DEFAULT_EF` - include `default_ef` scenarios (default: true for soak, false for smoke).
- `CHROMA_PERF_REPORT_DIR` - output directory for `perf-summary-*.json` and `perf-summary-*.md`.
- `CHROMA_PERF_ENABLE_DELETE_REINSERT` - enables delete+reinsert write operations (disabled by default due to current local runtime stability issues under delete-heavy load).

See `docs/docs/performance-testing.md` for profile definitions, thresholds, and report schema.

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
