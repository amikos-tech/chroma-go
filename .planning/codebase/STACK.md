# Technology Stack

**Analysis Date:** 2026-03-18

## Languages

**Primary:**
- Go 1.24.11 - all library, client, provider, runtime, and most test code (`go.mod`, `pkg/**`, `scripts/offline_bundle/main.go`)

**Secondary:**
- Python 3 - cross-language embedding and local persistence checks (`scripts/cross_lang_ef_test.py`, `scripts/local_persistence_crosscheck.py`)
- Shell - local server/runtime bootstrap helpers (`scripts/chroma_server.sh`, `scripts/fetch_runtime_deps.sh`)
- Markdown + MkDocs config - product docs and examples (`docs/docs/*.md`, `docs/mkdocs.yml`)

## Runtime

**Environment:**
- Go toolchain 1.24.x for build/test/library execution
- Python virtualenv for cross-language tests (`Makefile` target `setup-python-venv`)
- Docker for Chroma integration tests and local server workflows (`Makefile` target `server`, `.github/workflows/go.yml`)

**Package Manager:**
- Go modules
- Lockfile: `go.sum` present

## Frameworks

**Core:**
- Standard library HTTP + JSON for the SDK and provider clients
- `github.com/pkg/errors` for wrapped errors throughout the library

**Testing:**
- `go test` with build tags
- `gotest.tools/gotestsum` via `Makefile` for grouped test runs and JUnit output
- `github.com/stretchr/testify` for assertions
- `github.com/leanovate/gopter` for property-based tests
- `github.com/testcontainers/testcontainers-go` for Chroma integration tests

**Build/Dev:**
- `golangci-lint` with `gci` import formatting (`.golangci.yml`)
- MkDocs Material for docs publishing (`docs/mkdocs.yml`)
- GitHub Actions for CI, smoke, nightly, docs deploy, and perf workflows (`.github/workflows/*.yml`)

## Key Dependencies

**Critical:**
- `github.com/testcontainers/testcontainers-go` - integration tests against real Chroma containers
- `github.com/amikos-tech/chroma-go-local` - embedded local runtime shim for persistent client support
- `github.com/amikos-tech/pure-onnx` - local default embedding runtime dependency
- `github.com/amikos-tech/pure-tokenizers` - tokenizer dependency for local/runtime workflows
- `google.golang.org/genai` - Gemini provider support
- `github.com/aws/aws-sdk-go-v2/.../bedrockruntime` - Amazon Bedrock embedding support
- `go.uber.org/zap` - logging bridge support
- `github.com/go-playground/validator/v10` - provider/config validation

**Infrastructure:**
- GitHub Container Registry images for Chroma test matrices (`.github/workflows/go.yml`)
- GitHub releases / tokens for runtime asset resolution (`scripts/fetch_runtime_deps.sh`, `pkg/api/v2/client_local_library_download.go`)

## Configuration

**Environment:**
- Heavy env-var use for provider API keys, cloud credentials, runtime asset paths, and perf toggles
- Root `.env` exists locally, but code paths generally persist env-var names rather than secret values

**Build:**
- `go.mod`, `go.sum` - module/dependency definition
- `Makefile` - build, lint, unit/integration/provider/perf targets
- `.golangci.yml` - lint and import-order rules
- `docs/mkdocs.yml` - docs site configuration

## Platform Requirements

**Development:**
- Go 1.24.x
- Python 3 for cross-language suites
- Docker for most API integration tests
- Optional `GITHUB_TOKEN` / `GH_TOKEN` to avoid rate limits during runtime bootstrap/download flows

**Production:**
- Primary output is a Go library consumed by other applications
- Supports remote Chroma server/cloud deployments and embedded local runtime usage
- Some features are platform/runtime sensitive: local runtime libraries, tokenizer artifacts, and OS-specific smoke/compile guards

---

*Stack analysis: 2026-03-18*
*Update after major dependency or runtime changes*
