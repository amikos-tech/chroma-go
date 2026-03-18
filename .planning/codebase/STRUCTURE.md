# Codebase Structure

**Analysis Date:** 2026-03-18

## Directory Layout

```text
chroma-go/
├── .github/workflows/    # CI, docs deploy, review, perf, nightly jobs
├── docs/                 # MkDocs site config and end-user docs
├── examples/v2/          # Self-contained V2 examples, many with their own go.mod
├── pkg/api/v2/           # Main public client API, collections, schema, search, cloud/local clients
├── pkg/commons/          # Shared HTTP/provider helper packages
├── pkg/embeddings/       # Shared embedding contracts plus one package per provider
├── pkg/internal/         # Internal helpers (downloads, cosign verification)
├── pkg/logger/           # Logger abstraction and Zap bridge
├── pkg/rerankings/       # Shared reranking API plus provider packages
├── pkg/tokenizers/       # Tokenizer/runtime support libraries
├── scripts/              # Runtime bootstrap, cross-language checks, local server helpers
├── data/                 # Checked-in model/runtime data used by reranking/runtime workflows
├── Makefile              # Canonical build/test/lint entry points
├── README.md             # High-level feature and usage documentation
└── CLAUDE.md             # Repo-specific working conventions
```

## Directory Purposes

**`pkg/api/v2/`:**
- Purpose: current primary user-facing API
- Contains: client backends, collection operations, metadata, schema, ranking/search, configuration persistence
- Key files: `client.go`, `client_http.go`, `client_cloud.go`, `client_local.go`, `collection.go`, `schema.go`, `search.go`
- Subdirectories: none; flat but broad package

**`pkg/embeddings/`:**
- Purpose: embedding contracts and provider implementations
- Contains: shared interfaces/registry plus subpackages like `openai`, `cohere`, `jina`, `roboflow`, `bedrock`, `default_ef`
- Key files: `embedding.go`, `registry.go`, `secret.go`
- Subdirectories: 20+ provider/runtime packages

**`pkg/rerankings/`:**
- Purpose: reranking abstractions and provider implementations
- Contains: shared reranking types plus `cohere`, `hf`, `jina`, `together`
- Key files: `reranking.go`

**`docs/`:**
- Purpose: MkDocs-powered documentation site
- Contains: end-user docs in `docs/docs/`, site config in `docs/mkdocs.yml`, HTML overrides and assets
- Key files: `docs/docs/client.md`, `docs/docs/embeddings.md`, `docs/docs/rerankers.md`

**`examples/v2/`:**
- Purpose: runnable examples for V2 features
- Contains: feature-specific example modules, often with their own `go.mod`
- Key files: `examples/v2/basic/main.go`, `examples/v2/persistent_client/main.go`, `examples/v2/search/main.go`

**`scripts/`:**
- Purpose: helper automation around runtime bootstrap and cross-language verification
- Contains: shell scripts, Python tests, Go-based offline bundle tool
- Key files: `scripts/fetch_runtime_deps.sh`, `scripts/cross_lang_ef_test.py`, `scripts/local_persistence_crosscheck.py`

## Key File Locations

**Entry Points:**
- `pkg/api/v2/client.go` - primary client and option API
- `pkg/api/v2/client_cloud.go` - cloud-specific entry
- `pkg/api/v2/client_local.go` - embedded runtime entry
- `scripts/offline_bundle/main.go` - offline dependency bundle generator

**Configuration:**
- `go.mod` - module and dependency versions
- `.golangci.yml` - lint and formatting rules
- `Makefile` - build/test commands
- `docs/mkdocs.yml` - docs site build config

**Core Logic:**
- `pkg/api/v2/*.go` - API surface and operation orchestration
- `pkg/embeddings/*.go` + provider subdirs - embedding contracts and implementations
- `pkg/rerankings/*.go` + provider subdirs - reranking functionality

**Testing:**
- Adjacent `*_test.go` files throughout `pkg/`
- Cross-language helpers in `scripts/`
- Workflow-level validation in `.github/workflows/`

**Documentation:**
- `README.md` - overview
- `docs/docs/*.md` - canonical docs pages
- `docs/go-examples/README.md` - example index

## Naming Conventions

**Files:**
- lowercase Go filenames, usually domain-oriented (`client_local.go`, `configuration.go`, `roboflow.go`)
- tests colocated as `*_test.go`
- major docs/config files in uppercase or standard project names (`README.md`, `CLAUDE.md`, `Makefile`)

**Directories:**
- lowercase package/provider names (`pkg/embeddings/openai`, `pkg/rerankings/jina`)
- versioned examples under `examples/v2/*`

**Special Patterns:**
- provider packages usually contain `option.go`, main implementation file, and tests
- examples frequently carry standalone module files (`go.mod`, `go.sum`) to isolate usage

## Where to Add New Code

**New API feature:**
- Primary code: `pkg/api/v2/`
- Tests: colocated `pkg/api/v2/*_test.go`
- Docs/examples: `docs/docs/*.md`, `examples/v2/`

**New embedding provider or shared embedding capability:**
- Shared contract/registry: `pkg/embeddings/embedding.go`, `pkg/embeddings/registry.go`
- Provider-specific implementation: `pkg/embeddings/<provider>/`
- Persistence/auto-wire changes: `pkg/api/v2/configuration.go`, schema/config tests

**New reranker:**
- Implementation: `pkg/rerankings/<provider>/`
- Shared abstractions: `pkg/rerankings/reranking.go`

**Runtime/bootstrap helpers:**
- Internal helpers: `pkg/internal/`
- User-facing scripts/tooling: `scripts/`

## Special Directories

**`data/`:**
- Purpose: bundled model/runtime data used by some local/test workflows
- Source: checked-in artifacts, not generated during normal package builds
- Committed: Yes

**`artifacts/`:**
- Purpose: generated offline runtime bundles and perf outputs
- Source: test/perf/bootstrap commands
- Committed: typically no

**`examples/v2/*`:**
- Purpose: isolated example modules
- Source: maintained examples, not generated
- Committed: Yes

---

*Structure analysis: 2026-03-18*
*Update when package layout or example/docs organization changes*
