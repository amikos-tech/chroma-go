# Chroma Go

## What This Is

Chroma Go is a Go SDK for Chroma that supports remote HTTP, Chroma Cloud, and embedded local runtime usage, plus pluggable dense, sparse, and multimodal embedding functions and rerankers. The SDK provides a shared Content API for provider-neutral multimodal embeddings with Gemini, VoyageAI, and Twelve Labs adoptions, convenience constructors, and full embedded client parity.

## Core Value

Go applications can use Chroma and embedding providers through a stable, portable API that minimizes provider-specific friction.

## Requirements

### Validated

- ✓ Users can connect to Chroma over HTTP, Chroma Cloud, or embedded local runtime — existing
- ✓ Users can create collections, add/query/search data, and persist collection configuration — existing
- ✓ Users can use multiple dense and sparse embedding providers with env-var-backed config reconstruction — existing
- ✓ Users can use at least one multimodal provider (Roboflow) for shared text and image embeddings — existing
- ✓ Users can use reranking providers, docs, examples, and build-tagged tests across the V2 API — existing
- ✓ Provider-neutral multimodal Content API with ordered mixed-part requests, neutral intents, and per-request options — v0.4.1
- ✓ Provider capability metadata and backward-compatible adapters for legacy callers — v0.4.1
- ✓ Content registry with fallback chain, config persistence, and collection auto-wiring — v0.4.1
- ✓ Intent mapping with explicit failure for unsupported modality/intent combinations — v0.4.1
- ✓ Convenience constructors reducing Content API verbosity — v0.4.1
- ✓ Gemini, VoyageAI, and Twelve Labs multimodal provider adoptions — v0.4.1
- ✓ OpenRouter standalone provider with ProviderPreferences routing — v0.4.1
- ✓ Fork double-close bug fixed with close-once EF wrappers — v0.4.1
- ✓ Delete-with-limit, Collection.ForkCount, embedded contentEF parity — v0.4.1
- ✓ Cloud integration tests for Search API RRF and GroupBy — v0.4.1
- ✓ Code cleanups: shared pathutil, context.Context fix, registry test cleanup — v0.4.1
- ✓ SDK auto-wiring behavior documented across Python, JS, Rust, Go — v0.4.1
- ✓ RrfRank arithmetic methods build correct expression trees instead of silent no-ops — v0.4.2 Phase 21
- ✓ WithGroupBy(nil) rejects explicit nil input with a stable validation error — v0.4.2 Phase 22

## Current Milestone: v0.4.2 Bug Fixes and Robustness

**Goal:** Fix API bugs, harden embedded client lifecycle, and clean up error handling across embedding providers.

**Target features:**
- Fix RrfRank arithmetic silent no-ops (#481)
- Fix WithGroupBy(nil) silently skipping grouping (#482)
- Fix embedded GetOrCreateCollection passing closed EFs (#493)
- Fix default ORT EF leak in embedded CreateCollection (#494)
- Fix Morph EF integration test (#465)
- Truncate raw error bodies in embedding providers (#478)
- Refactor release download stack (#412)
- Add Twelve Labs async embedding support (#479)

### Active
- Embedded GetOrCreateCollection passes closed EFs to CreateCollection fallback — #493
- Default ORT EF leaked when CreateCollection finds existing collection — #494
- Morph EF integration test broken by upstream 404 — #465
- Raw error bodies can be arbitrarily large in provider error messages — #478
- Release download stack has excessive duplication across providers — #412
- Twelve Labs lacks async embedding for long audio/video — #479

### Out of Scope

- Replacing or removing existing `EmbeddingFunction` and image-only multimodal APIs — backwards compatibility is maintained
- Changing collection/query semantics outside the embedding abstraction boundary

## Context

- Brownfield Go library with public API in `pkg/api/v2`, shared embedding contracts in `pkg/embeddings`, configuration auto-wiring in `pkg/api/v2/configuration.go`, docs in `docs/docs/embeddings.md`, and a codebase map already present under `.planning/codebase/`
- Issue `#442` defines the foundation scope: richer multimodal inputs, neutral intents, per-request options, capability introspection, registry/factory support, explicit unsupported-combination failures, and documentation guidance
- Issue `#443` defines Gemini multimodal adoption scope: wire Gemini into the shared contract with full modality support
- vLLM/Nemotron validation targets nvidia/omni-embed-nemotron-3b via an internal vLLM API to prove the contract is portable beyond Gemini
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
| Treat issue `#442` as the active initialization scope for GSD planning | The user explicitly named this work before requesting project initialization | ✓ Good |
| Use the existing codebase map as brownfield context instead of re-running codebase mapping | `.planning/codebase/` already captures architecture, concerns, structure, and testing | ✓ Good |
| Rebrand milestone from `v0.5` to `v0.4.1` | All changes since v0.4.0 are purely additive with no public API breakage — patch bump is correct semver | ✓ Good |
| Add Gemini multimodal as Phase 6 (issue #443) | First concrete provider adoption validates the shared contract end-to-end | ✓ Good |
| Pivot Phase 7 from vLLM/Nemotron to VoyageAI | vLLM lacks NVOmniEmbedModel support; VoyageAI multimodal validates portability with text/image/video | ✓ Good |
| Add Twelve Labs as third multimodal provider (Phase 16) | Validates contract portability across text/image/audio/video with a non-Google/non-Voyage provider | ✓ Good |
| Close-once EF wrappers for Fork double-close fix | Defense-in-depth alongside ownsEF flag; prevents panics even if ownership logic is bypassed | ✓ Good |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-10 — Phase 22 (WithGroupBy validation) complete*
