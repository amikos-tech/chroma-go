# Stack Research

**Domain:** Brownfield Go SDK milestone for provider-neutral multimodal embeddings
**Researched:** 2026-03-18
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

## Sources

- `go.mod` — current toolchain and dependency baseline
- `.planning/codebase/ARCHITECTURE.md` — contract and config layering in this repo
- `.planning/codebase/CONCERNS.md` — existing contract fragmentation and multimodal risks
- `pkg/embeddings/embedding.go` — current dense and image-only multimodal interfaces
- `pkg/embeddings/registry.go` — dense, sparse, and multimodal registry structure
- `pkg/api/v2/configuration.go` — build-from-config and auto-wiring paths
- GitHub issue `#442` — active milestone scope and acceptance criteria

---
*Stack research for: brownfield Go SDK multimodal foundations*
*Researched: 2026-03-18*
