# External Integrations

**Analysis Date:** 2026-03-18

## APIs & External Services

**Chroma Server / Cloud:**
- Chroma HTTP API - primary remote backend for collection/database operations
  - Client code: `pkg/api/v2/client.go`, `pkg/api/v2/client_http.go`, `pkg/api/v2/client_cloud.go`
  - Test/development runtime: Docker + GHCR images in `pkg/api/v2/*integration_test.go` and `.github/workflows/go.yml`
  - Cloud auth: `CHROMA_API_KEY`, `CHROMA_TENANT`, `CHROMA_DATABASE`

**Embedding Providers:**
- OpenAI, Cohere, HuggingFace, Google Gemini, Jina, Mistral, Nomic, Ollama, Cloudflare, Together, Voyage, Roboflow, Amazon Bedrock, Baseten, Morph, Perplexity, Chroma Cloud, and local/default providers
  - Implementations live in `pkg/embeddings/*`
  - Auth is almost entirely env-var based and provider-specific
  - Several providers expose task/dimension/model configuration via persisted config maps

**Reranking Providers:**
- Cohere, HuggingFace, Jina, Together
  - Implementations in `pkg/rerankings/*`
  - Live tests are env-gated similarly to embedding providers

## Data Storage

**Databases / Vector Storage:**
- External Chroma server or Chroma Cloud - main document/vector storage target
  - Accessed over HTTP via `pkg/api/v2`

**Embedded Runtime Storage:**
- `chroma-go-local` persistent runtime - in-process/local storage path for `NewPersistentClient`
  - Entry points: `pkg/api/v2/client_local.go`, `pkg/api/v2/client_local_embedded.go`
  - Runtime library path/config: `CHROMA_LIB_PATH`, `CHROMAGO_ONNX_RUNTIME_PATH`, `TOKENIZERS_LIB_PATH`

**Local Artifacts / Bundles:**
- Offline runtime bundles generated under `artifacts/`
  - Scripts: `scripts/fetch_runtime_deps.sh`, `scripts/offline_bundle/main.go`

## Authentication & Identity

**Cloud / API Access:**
- Chroma Cloud credentials via env vars and client options
- Provider authentication typically via `WithEnvAPIKey`, `WithAPIKeyFromEnvVar`, bearer token helpers, or SDK credentials

**Runtime Trust / Verification:**
- Local runtime artifact verification includes download, checksum, and cosign-related helpers
  - Packages: `pkg/internal/cosignutil`, `pkg/internal/downloadutil`

## Monitoring & Observability

**Logging:**
- Internal logger abstraction and Zap bridge (`pkg/logger`)
- Structured logging examples in `examples/v2/logging` and `examples/v2/logging_slog`

**Docs Analytics / Site Integrations:**
- Google Analytics and EthicalAds on the MkDocs site (`docs/mkdocs.yml`, `docs/docs/javascripts/*`)

## CI/CD & Deployment

**CI Pipeline:**
- GitHub Actions
  - Go test/lint/matrix runs: `.github/workflows/go.yml`, `.github/workflows/lint.yaml`, `.github/workflows/nightly.yml`
  - Perf smoke/soak: `.github/workflows/perf-local.yml`
  - Docs deploy: `.github/workflows/mkdocs.yml`

**Hosting:**
- GitHub Pages for docs site generated from MkDocs
  - Source: `docs/docs/*.md`
  - Deploy workflow: `.github/workflows/mkdocs.yml`

## Environment Configuration

**Development:**
- Local Chroma URL via `CHROMA_URL`
- Provider keys loaded from shell environment
- Python venv installed from `requirements-test.txt` for cross-language tests

**CI / Test:**
- GitHub Actions secrets for cloud/API tokens and GitHub auth
- Test matrix uses container images, build tags, and env-gated live provider tests

## Webhooks & Callbacks

**Incoming:**
- None in the product/library itself

**Outgoing:**
- HTTP calls to provider APIs and Chroma deployments
- GitHub API/release metadata access for runtime bootstrap/download flows

---

*Integration audit: 2026-03-18*
*Update when adding/removing providers, cloud flows, or runtime asset sources*
