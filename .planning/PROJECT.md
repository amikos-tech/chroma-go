# Chroma Go

## What This Is

Chroma Go is a Go SDK for Chroma that supports remote HTTP, Chroma Cloud, and embedded local runtime usage, plus pluggable dense, sparse, multimodal embedding functions and rerankers. This planning workspace is being initialized around the current brownfield milestone: provider-neutral multimodal embedding foundations that expand the shared contract without breaking existing text-only or image-only consumers.

## Core Value

Go applications can use Chroma and embedding providers through a stable, portable API that minimizes provider-specific friction.

## Requirements

### Validated

- ✓ Users can connect to Chroma over HTTP, Chroma Cloud, or embedded local runtime — existing
- ✓ Users can create collections, add/query/search data, and persist collection configuration — existing
- ✓ Users can use multiple dense and sparse embedding providers with env-var-backed config reconstruction — existing
- ✓ Users can use at least one multimodal provider (Roboflow) for shared text and image embeddings — existing
- ✓ Users can use reranking providers, docs, examples, and build-tagged tests across the V2 API — existing

### Active

- [ ] Add a provider-neutral multimodal input model that supports mixed-part requests across text, image, audio, video, and PDF.
- [ ] Add provider-neutral intent semantics and per-request multimodal options without breaking current text-only and image-only flows.
- [ ] Add capability introspection, config and registry support, and docs/tests for portable multimodal embeddings.

### Out of Scope

- Shipping every provider on the new multimodal contract in the first milestone — foundation first, incremental adoption later
- Replacing or removing existing `EmbeddingFunction` and image-only multimodal APIs — backwards compatibility is an explicit acceptance criterion
- Changing collection/query semantics outside the embedding abstraction boundary — keep the milestone scoped to shared embedding foundations

## Context

- Brownfield Go library with public API in `pkg/api/v2`, shared embedding contracts in `pkg/embeddings`, configuration auto-wiring in `pkg/api/v2/configuration.go`, docs in `docs/docs/embeddings.md`, and a codebase map already present under `.planning/codebase/`
- Issue `#442` defines the active scope: richer multimodal inputs, neutral intents, per-request options, capability introspection, registry/factory support, explicit unsupported-combination failures, and documentation guidance
- Current multimodal support is provider-specific: the shared `EmbeddingFunction` is text-only, the shared `MultimodalEmbeddingFunction` only adds image methods, and Roboflow is the only registered multimodal provider today
- Repo conventions emphasize V2-first changes, colocated tests with proper build tags, config round-tripping, no panics in production code, and docs/examples updates for public API changes

## Constraints

- **Compatibility**: Existing text-only and image-only callers must keep compiling and behaving the same — required by issue `#442` and expected of a public SDK
- **Tech stack**: Changes should align with Go 1.24.x, the existing provider package layout, and V2 API conventions — avoid introducing parallel abstractions without a migration reason
- **Persistence**: Registry/build-from-config behavior must continue to use serializable config maps and env-var-based secret indirection — collection auto-wiring depends on it
- **Validation**: Unsupported modality and intent combinations must fail explicitly instead of silently degrading — part of the milestone acceptance criteria
- **Documentation**: Public behavior changes require docs and examples that distinguish portable intent usage from provider-specific hints — otherwise users cannot adopt the new API safely

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Treat issue `#442` as the active initialization scope for GSD planning | The user explicitly named this work before requesting project initialization | — Pending |
| Use the existing codebase map as brownfield context instead of re-running codebase mapping | `.planning/codebase/` already captures architecture, concerns, structure, and testing | ✓ Good |
| Label the current roadmap milestone as `v0.5 Provider-Neutral Multimodal Foundations` | The latest shipped tag is `v0.4.0`; the roadmap tooling benefits from a parsable next-minor placeholder | ⚠️ Revisit |

---
*Last updated: 2026-03-18 after initialization*
