---
gsd_state_version: 1.0
milestone: v0.4.1
milestone_name: Provider-Neutral Multimodal Foundations
status: unknown
stopped_at: Completed 07-01-PLAN.md
last_updated: "2026-03-22T16:09:52.039Z"
progress:
  total_phases: 8
  completed_phases: 6
  total_plans: 18
  completed_plans: 17
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-18)

**Core value:** Go applications can use Chroma and embedding providers through a stable, portable API that minimizes provider-specific friction.
**Current focus:** Phase 07 — voyage-multimodal-adoption

## Current Position

Phase: 07 (voyage-multimodal-adoption) — EXECUTING
Plan: 2 of 2

## Performance Metrics

**Velocity:**

- Total plans completed: 10
- Average duration: 5 min
- Total execution time: 51 min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| Phase 01 | 4 | 19 min | 5 min |
| Phase 02 | 3 | 17 min | 6 min |
| Phase 03 | 3 | 11 min | 4 min |

**Recent Trend:**

- Last 5 plans: -
- Trend: Stable

| Phase | Duration | Tasks | Files |
|-------|----------|-------|-------|
| Phase 02 P02 | 6min | 2 tasks | 2 files |
| Phase 02 P03 | 7min | 2 tasks | 3 files |
| Phase 03 P01 | 3min | 2 tasks | 2 files |
| Phase 03 P02 | 2min | 2 tasks | 4 files |
| Phase 03 P03 | 6min | 2 tasks | 2 files |
| Phase 04 P01 | 8 | 2 tasks | 3 files |
| Phase 04 P02 | 4 | 2 tasks | 2 files |
| Phase 05 P02 | 2 | 1 tasks | 1 files |
| Phase 05 P01 | 2 | 2 tasks | 2 files |
| Phase 06 P01 | 5 | 2 tasks | 2 files |
| Phase 06 P02 | 10min | 1 tasks | 1 files |
| Phase 07 P01 | 3min | 2 tasks | 2 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Init]: Use issue #442 as the first planned milestone for GSD tracking.
- [Init]: Reuse the existing codebase map instead of re-running brownfield mapping.
- [Init]: Rebranded milestone from `v0.5` to `v0.4.1` — all changes are additive, no public API breakage.
- [Phase 01-shared-multimodal-contract]: Keep the shared multimodal request model in a dedicated file so later validation and compatibility work can layer on without disturbing legacy APIs.
- [Phase 01-shared-multimodal-contract]: Add ContentEmbeddingFunction beside MultimodalEmbeddingFunction instead of widening the legacy image-only interface in place.
- [Phase 02]: Keep shared capability metadata provider-neutral by modeling only modalities, intents, and request options. — This preserves room for non-Roboflow providers and avoids baking provider-native task names into the shared contract.
- [Phase 02]: Expose capability inspection through an additive CapabilityAware interface instead of widening legacy embedding interfaces. — Phase 2 must preserve existing EmbeddingFunction and MultimodalEmbeddingFunction callers while adding new discovery behavior.
- [Phase 02]: Reject shared-content fields that legacy interfaces cannot represent safely — Compatibility adapters must fail explicitly instead of silently dropping Intent, Dimension, ProviderHints, mixed parts, or bytes-backed image sources.
- [Phase 02]: Delegate Roboflow shared-content support through the compatibility adapter — Using the additive adapter keeps shared-content behavior aligned with existing text and image methods and avoids duplicating provider request logic.
- [Phase 02]: Test capability discovery through shared interfaces and adapter stubs — Phase 2 should prove the additive shared surface itself, not provider-specific concrete type assertions, so regressions are caught at the contract boundary.
- [Phase 02]: Skip transient Roboflow live failures in the default suite — Upstream 429/5xx availability noise should not make the default regression suite flaky once provider-specific live tests are runnable by default.
- [Phase 03-01]: BuildContent fallback chain releases mu.RLock before each factory call to avoid recursive lock deadlock
- [Phase 03-01]: inferCaps uses CapabilityAware metadata when available and falls back to interface-typed defaults for multimodal and dense EFs
- [Phase 03-02]: Derive dense EF from content EF at GetCollection time when content implements EmbeddingFunction, avoiding double initialization
- [Phase 03-02]: Close contentEF first in CollectionImpl.Close() to avoid double-close when contentEF wraps denseEF (adapter case)
- [Phase 03-03]: Extract deriveEFFromContent helper to test auto-wiring logic without triggering staticcheck SA4023 on concrete-type nil comparisons
- [Phase 03-03]: Register test multimodal factory inside test function body (not init) to keep registry state isolated between test runs
- [Phase 04-01]: IntentMapper is an opt-in interface (type-assert pattern) rather than widening ContentEmbeddingFunction
- [Phase 04-01]: ValidateContentSupport passes through when caps.Modalities is empty to preserve backward compatibility with non-CapabilityAware providers
- [Phase 04-01]: Custom intents bypass capability intent enforcement — only neutral intents checked against declared caps.Intents
- [Phase 05-02]: Use NewEmbeddingFromFloat32 helper for mock construction; use distinct fixed values [1,2,3] vs [4,5,6] to distinguish native vs adapter dispatch paths
- [Phase 05-01]: Show mixed-part Roboflow example with separate Content items via EmbedContents (one Part per Content due to adapter constraint)
- [Phase 05-01]: Frame both EmbedDocuments and Content API as coexisting indefinitely — no deprecation signal in docs
- [Phase 05-01]: Escape-hatch admonition for ProviderHints references godoc rather than documenting mechanism inline
- [Phase 06-01]: Default model updated to gemini-embedding-2-preview; LegacyEmbeddingModel constant added for gemini-embedding-001
- [Phase 06-01]: Batch requests use default task type for all items; single-item requests allow per-item ProviderHints override
- [Phase 06-01]: resolveMIME falls back from BinarySource.MIMEType to file extension; fails explicitly when neither resolves
- [Phase 06-02]: Construct GeminiEmbeddingFunction via struct literal in unit tests to avoid genai.NewClient network calls while keeping tests hermetic
- [Phase 06-02]: EmbedContentLegacyModelRejectsMultimodal uses dual-string check because ValidateContentSupport produces message with 'does not support' not 'unsupported'
- [Phase 07]: Copied resolveBytes/resolveMIME helpers from Gemini rather than extracting to shared package
- [Phase 07]: Batch requests reject per-item Intent/Dimension/ProviderHints with explicit errors matching Gemini pattern
- [Phase 07]: multimodalURL derives endpoint by replacing /v1/embeddings suffix, falling back to constant for custom base URLs

### Roadmap Evolution

- Project initialized around provider-neutral multimodal embedding foundations (#442).
- Rebranded milestone v0.5 → v0.4.1 (all changes additive, no public API breakage).
- Added Phase 6: Gemini Multimodal Adoption (#443).
- Added Phase 7: Originally vLLM/Nemotron, pivoted to Voyage Multimodal Adoption (vLLM lacks NVOmniEmbedModel support).
- Added Phase 8: Document Gemini and Nemotron multimodal embedding functions.

### Pending Todos

None yet.

### Blockers/Concerns

- The neutral multimodal contract must avoid overfitting to the current Roboflow implementation.
- Phase 7 pivoted from vLLM/Nemotron to VoyageAI multimodal — vLLM does not support NVOmniEmbedModel architecture.

## Decisions Made

| Phase | Summary | Rationale |
|-------|---------|-----------|
| Init | Scope the first roadmap milestone to issue #442 | The user explicitly named this work before initialization |
| Init | Reuse the generated codebase map | Brownfield architecture context already exists under `.planning/codebase/` |
| Init | Rebrand milestone from `v0.5` to `v0.4.1` | All changes are additive — patch bump is correct semver |

## Blockers

- Provider-neutral intent design validated against Gemini (Phase 6), VoyageAI multimodal (Phase 7).
- Phase 7 requires `VOYAGE_API_KEY` env var for integration tests.

## Session

**Last Date:** 2026-03-22T16:09:52.037Z
**Stopped At:** Completed 07-01-PLAN.md
**Resume File:** None
