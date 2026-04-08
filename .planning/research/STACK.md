# Stack Research

**Domain:** Brownfield Go SDK milestone for provider-neutral multimodal embeddings
**Researched:** 2026-03-18 (updated 2026-04-08 for v0.4.2)
**Confidence:** HIGH

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.24.11 | Primary implementation language for shared contracts, providers, config flows, and tests | The repo already standardizes on Go 1.24.x and all public API surfaces live in Go packages |
| `pkg/embeddings` shared contracts | current repo | Define additive multimodal types, intents, capabilities, and compatibility shims | The existing embedding and multimodal interfaces already anchor provider implementations and registry behavior |
| `pkg/api/v2/configuration.go` plus `pkg/embeddings/registry.go` | current repo | Persist, rebuild, and auto-wire embedding functions from stored configuration | Richer multimodal foundations must fit the current config round-trip and factory architecture instead of bypassing it |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/pkg/errors` | v0.9.1 | Wrapped runtime and validation errors | Keep using it for explicit unsupported-modality and unsupported-intent failures |
| `github.com/go-playground/validator/v10` | v10.30.1 | Provider option validation | Reuse for new request and option validation where struct validation is practical |
| `github.com/stretchr/testify` | v1.11.1 | Unit and regression assertions | Use for compatibility, config round-trip, and validation tests |
| `net/http/httptest` | stdlib | Provider API and config-path stubs | Use when mapping neutral intents to provider-native requests without live credentials |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `gofmt` and `golangci-lint` | Formatting and lint checks | Match repo conventions and import ordering before committing |
| `go test` with build tags | Unit, provider, and integration coverage | Shared-contract work should at minimum cover unit/config tests and targeted provider tests |
| Existing docs/examples tree | Public API documentation and examples | Keep docs aligned with any new portable multimodal contract or compatibility shim |

## Installation

```bash
# Core workspace
go test ./...

# Canonical targeted checks
make test
make test-ef
make lint
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Extend `pkg/embeddings` with additive richer multimodal types | Create a separate experimental package for multimodal foundations | Only if the current API cannot be extended compatibly, which issue `#442` explicitly says to avoid |
| Keep config persistence in existing `EmbeddingFunctionConfig` maps | Introduce a second unrelated persistence format | Only if additive config keys prove impossible to support, which would raise migration cost materially |
| Implement provider-neutral intents in shared contracts | Expose provider-native task strings directly in the shared API | Only for provider-specific escape hatches, not as the portable primary contract |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Provider-native task names as the shared interface contract | They differ across providers and make cross-provider portability fragile | Provider-neutral intents plus explicit provider mapping |
| Breaking `EmbeddingFunction` or current image-only multimodal callers | This would violate the milestone acceptance criteria and create avoidable upgrade pain | Additive richer interfaces and compatibility adapters |
| Persisting secret values directly in config | The repo already uses env-var indirection for provider secrets | Continue persisting env-var names and provider-safe config values only |

## Stack Patterns by Variant

**If the change is shared-contract only:**
- Modify `pkg/embeddings/embedding.go` and related tests first
- Because provider packages and config flows depend on those interfaces

**If the change affects provider reconstruction:**
- Update `pkg/embeddings/registry.go`, provider `GetConfig()` / `FromConfig` flows, and `pkg/api/v2/configuration.go`
- Because runtime auto-wiring is a first-class behavior in this repo

**If the change affects public behavior:**
- Update `docs/docs/embeddings.md` and relevant example or test coverage
- Because docs drift is already a known concern for multimodal support

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| Go 1.24.11 | current `go.mod` dependency graph | Keep changes aligned with the repo toolchain baseline |
| Richer multimodal contract | existing `EmbeddingFunction` and `MultimodalEmbeddingFunction` callers | Must be additive and compatibility-tested |
| Config map extensions | current collection configuration auto-wiring | New keys must not break older dense or image-only providers |

---

## v0.4.2 Addendum: Stack for New Features

*Added 2026-04-08. Covers only the three net-new capability areas in v0.4.2.*

### Feature 1: Twelve Labs Async Embedding (#479)

**No new dependencies.** All async needs are satisfied by the Go standard library.

| Capability | Package | Already in go.mod |
|------------|---------|-------------------|
| HTTP POST/GET | `net/http` | Yes |
| JSON marshal/unmarshal | `encoding/json` | Yes |
| Context-aware polling loop | `context` | Yes |
| Sleep between polls | `time` | Yes |
| Error wrapping | `github.com/pkg/errors` | Yes |

**API contract (HIGH confidence — verified from official Python SDK source at HEAD):**

- Create task: `POST /v1.3/embed-v2/tasks` with body `{input_type, model_name, audio?, video?}`
- Response: `{_id: string, status: "processing", data: null}`
- Poll: `GET /v1.3/embed-v2/tasks/{task_id}`
- Poll response statuses: `processing` | `ready` | `failed`
- Ready response: `data` array of `{embedding: []float64, embedding_option, embedding_scope, start_sec, end_sec}`

**Key implementation decisions:**
- Polling uses `time.Sleep` with `context.Done()` select, not `time.NewTicker` — simpler for sequential poll-then-wait
- Tasks endpoint URL is `BaseAPI + "/tasks"` — derive from existing `BaseAPI` field rather than adding a second constant
- Expose `WithAsyncPollInterval(d time.Duration)` and `WithUseAsync(bool)` options on `TwelveLabsClient`; default poll interval 3 seconds
- Timeout is caller-controlled via context; do not add a separate async timeout option
- The `data` array can contain multiple segments (clip-scoped). The first `asset`-scoped item is the canonical single embedding; if only clip-scoped items exist, return the first one. Document this choice clearly.
- Keep `EmbedContent` / `EmbedContents` interface unchanged — async is an internal routing decision

**Scope:** Only audio and video modalities use the async path. Text and image continue to use the sync endpoint.

### Feature 2: Error Body Truncation (#478)

**No new dependencies.** All implementation uses stdlib string/rune operations.

**What to add to `pkg/commons/http/utils.go`:**
- `const MaxErrorBodyChars = 512`
- `func SanitizeErrorBody(body []byte, maxChars int) string` — rune-slice truncation with `"...(truncated)"` suffix (UTF-8-safe, matching the existing Perplexity pattern)

**What to change across providers:** Replace `string(respData)` in non-200 error paths in all 15+ affected providers with `chttp.SanitizeErrorBody(respData, chttp.MaxErrorBodyChars)`. Refactor Perplexity and OpenRouter to use the shared function and delete their local copies.

The `ReadLimitedBody` 200 MB cap already prevents memory exhaustion; this change limits error message length in logs — a separate concern.

### Feature 3: Download Stack Consolidation (#412)

**No new dependencies.** `pkg/internal/downloadutil` already exists and is the right target.

**Current state:**
- `pkg/internal/downloadutil` — canonical download helper, already used by `pkg/tokenizers/libtokenizers/library_download.go` and `pkg/api/v2/client_local_library_download.go`
- `pkg/embeddings/default_ef/download_utils.go` — has its own `downloadFile` function (lines 60-150) that duplicates temp-file-and-rename and HTTP transport setup from `downloadutil`

**Minimal v0.4.2 scope:** Wire `default_ef/download_utils.go` to use `downloadutil.DownloadFile` instead of the local `downloadFile`. Pass `downloadutil.Config{MaxBytes: ..., Timeout: 10*time.Minute, ...}` matching the current timeouts.

**Broader issue #412 scope** (checksum/cosign/mirror unification across local shim and tokenizers) is larger than what v0.4.2 needs and should be scoped separately. Do not expand into it here.

---

## Sources

- `go.mod` — current toolchain and dependency baseline
- `.planning/codebase/ARCHITECTURE.md` — contract and config layering in this repo
- `.planning/codebase/CONCERNS.md` — existing contract fragmentation and multimodal risks
- `pkg/embeddings/embedding.go` — current dense and image-only multimodal interfaces
- `pkg/embeddings/registry.go` — dense, sparse, and multimodal registry structure
- `pkg/api/v2/configuration.go` — build-from-config and auto-wiring paths
- GitHub issue `#442` — active milestone scope and acceptance criteria
- Twelve Labs Python SDK (auto-generated from OpenAPI): `src/twelvelabs/embed/v_2/tasks/raw_client.py`,
  `src/twelvelabs/types/embedding_task_response.py`, `src/twelvelabs/types/embedding_data.py`
  — github.com/twelvelabs-io/twelvelabs-python (cloned HEAD 2026-04-08)
- Twelve Labs async guide: https://docs.twelvelabs.io/docs/guides/create-embeddings/audio
- Issues #479, #478, #412 — verified scope and affected files against codebase
- `pkg/commons/http/utils.go` — `ReadLimitedBody` and `MaxResponseBodySize` verified
- `pkg/embeddings/perplexity/perplexity.go` — `sanitizeErrorBody` reference implementation verified
- `pkg/internal/downloadutil/download.go` — existing shared download helper verified

---
*Stack research for: brownfield Go SDK multimodal foundations*
*Originally researched: 2026-03-18 | Updated for v0.4.2: 2026-04-08*
