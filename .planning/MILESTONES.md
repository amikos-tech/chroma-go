# Milestones

## v0.4.1 Provider-Neutral Multimodal Foundations (Shipped: 2026-04-08)

**Phases:** 20 | **Plans:** 42 | **Tasks:** 78
**Timeline:** 2026-03-18 → 2026-04-08 (21 days)
**Go changes:** 78 files, +13,757 / -269 lines

**Key accomplishments:**

1. Shared multimodal Content API with ordered mixed-part requests, neutral intents, per-request options, and typed validation
2. Provider capability metadata and backward-compatible adapters for text-only and image-only callers
3. Content registry with 3-step fallback chain, config persistence, and collection auto-wiring
4. Gemini, VoyageAI, and Twelve Labs multimodal provider adoptions via the shared contract
5. OpenRouter standalone provider with ProviderPreferences routing
6. Fork double-close bug fix with close-once EF wrappers and ownership tracking
7. Delete-with-limit, Collection.ForkCount, and embedded client contentEF parity
8. Convenience constructors reducing Content API verbosity from 5+ lines to one call
9. Cloud integration tests for Search API RRF and GroupBy primitives

**Issues resolved:** #190, #438, #439, #440, #442, #443, #447, #448, #454, #455, #456, #460, #461, #462, #466, #469, #472, #474

**Archives:** [ROADMAP](milestones/v0.4.1-ROADMAP.md) | [REQUIREMENTS](milestones/v0.4.1-REQUIREMENTS.md)

---
