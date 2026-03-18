# Architecture

**Analysis Date:** 2026-03-18

## Pattern Overview

**Overall:** Go SDK/library for Chroma with pluggable embedding and reranking providers, multiple client backends, and a documentation/examples layer.

**Key Characteristics:**
- Public API is centered in `pkg/api/v2`
- Embeddings and rerankers are provider packages behind shared interfaces
- Supports three access modes: remote HTTP, Chroma Cloud, and embedded local runtime
- Configuration persistence and auto-wiring are first-class concerns for embedding functions

## Layers

**Public Client Layer:**
- Purpose: expose the library surface users call from applications
- Contains: `Client`, `Collection`, option structs, metadata/search/schema APIs
- Location: `pkg/api/v2/*.go`
- Depends on: HTTP helpers, embedding interfaces, logger, local runtime helpers
- Used by: examples, external consumers, tests

**Domain / Operation Layer:**
- Purpose: validate request options and convert high-level calls into concrete operations
- Contains: collection ops, query/search ops, schema/configuration handling, record and rank types
- Location: `pkg/api/v2/collection.go`, `pkg/api/v2/search.go`, `pkg/api/v2/schema.go`, `pkg/api/v2/configuration.go`
- Depends on: embedding interfaces and client implementations
- Used by: HTTP/cloud/local client flows

**Provider Abstraction Layer:**
- Purpose: define common embedding and reranking contracts and registry/build-from-config behavior
- Contains: `EmbeddingFunction`, `SparseEmbeddingFunction`, `MultimodalEmbeddingFunction`, registries, secrets, distance metrics
- Location: `pkg/embeddings/embedding.go`, `pkg/embeddings/registry.go`, `pkg/rerankings/reranking.go`
- Depends on: stdlib + validation/util helpers
- Used by: every provider package and config auto-wiring path

**Provider Implementation Layer:**
- Purpose: implement specific remote/local embedding and reranking backends
- Contains: one package per provider with request/response structs, options, config round-tripping, live tests
- Location: `pkg/embeddings/*`, `pkg/rerankings/*`
- Depends on: provider abstractions, commons helpers, external SDKs
- Used by: collection embedding flows and direct provider usage

**Runtime / Support Layer:**
- Purpose: support embedded local runtime, downloads, verification, tokenizers, and logging
- Contains: local runtime clients, artifact download helpers, cosign helpers, logger package, tokenizers
- Location: `pkg/api/v2/client_local*.go`, `pkg/internal/*`, `pkg/tokenizers/*`, `pkg/logger/*`
- Depends on: filesystem, networking, external libraries, Go runtime
- Used by: persistent client and runtime bootstrap/perf flows

## Data Flow

**Remote Collection Flow:**
1. User constructs a client with functional options (`pkg/api/v2/client.go`)
2. Client stores tenant/database/auth/logging state
3. Collection operations validate input and serialize request structs
4. Text inputs may be embedded through the configured embedding function
5. HTTP client sends requests to Chroma server/cloud and hydrates `Collection`/result types

**Auto-Wired Embedding Flow:**
1. A collection stores embedding function info in configuration/schema (`pkg/api/v2/configuration.go`)
2. Client fetches collection metadata/configuration
3. Registry resolves known provider name + config (`pkg/embeddings/registry.go`)
4. Provider constructor rebuilds the embedding function from env-var-based config

**Embedded Runtime Flow:**
1. `NewPersistentClient` resolves runtime library paths/download behavior (`pkg/api/v2/client_local.go`)
2. Embedded local client maintains collection state and dimensions (`pkg/api/v2/client_local_embedded.go`)
3. Operations execute against the in-process runtime instead of a remote server

**State Management:**
- Remote clients are mostly stateless beyond tenant/database/auth/logger/default EF configuration
- Embedded local client tracks collection state, dimensions, and runtime lifecycle in-memory
- Persisted collection configuration bridges server-side state with local provider reconstruction

## Key Abstractions

**Client / Collection:**
- Purpose: primary end-user API
- Examples: `Client`, `Collection`, `PersistentClient`, `CloudAPIClient`
- Pattern: interface + concrete backend implementations

**Operation Structs:**
- Purpose: collect validated options before request execution
- Examples: `CreateCollectionOp`, `GetCollectionOp`, search/rank options
- Pattern: functional options over mutable op structs

**Embedding Registry:**
- Purpose: rebuild known providers from stored config
- Examples: dense, sparse, multimodal registries in `pkg/embeddings/registry.go`
- Pattern: init-time registration + factory lookup

## Entry Points

**Library Entry Points:**
- `pkg/api/v2/client.go` - core client API and options
- `pkg/api/v2/client_cloud.go` - cloud client construction
- `pkg/api/v2/client_local.go` - persistent embedded client construction

**Provider Entry Points:**
- `pkg/embeddings/<provider>/*.go` - provider constructors and config reconstruction
- `pkg/rerankings/<provider>/*.go` - reranker constructors

**Operational Entry Points:**
- `Makefile` - main local build/test interface
- `scripts/offline_bundle/main.go` - offline runtime bundle CLI
- `docs/mkdocs.yml` - docs site build entry

## Error Handling

**Strategy:** return wrapped errors from validation and request boundaries; avoid panics in normal runtime flows.

**Patterns:**
- `github.com/pkg/errors` is used heavily for wrap/context
- Option validation happens early and repeatedly
- Public APIs favor explicit validation over implicit defaults when safety matters
- A notable exception is init-time registry duplication checks, which panic intentionally in several provider packages

## Cross-Cutting Concerns

**Logging:**
- Central logger abstraction in `pkg/logger`
- Client logging is injected/configurable, not hard-coded

**Validation:**
- Struct validation in several providers
- Request validation in op structs before I/O

**Compatibility:**
- The repo actively targets multiple Chroma versions and multiple execution modes
- Build tags partition incompatible suites (`basicv2`, `cloud`, `ef`, `rf`, `crosslang`, `soak`)

---

*Architecture analysis: 2026-03-18*
*Update when client backends, provider contracts, or runtime flows change materially*
