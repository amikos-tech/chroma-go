# Phase 8: Document Gemini and VoyageAI Multimodal Embedding Functions - Context

**Gathered:** 2026-03-23
**Status:** Ready for planning

<domain>
## Phase Boundary

Update provider-specific documentation for Gemini and VoyageAI to show Content API multimodal usage. Add runnable examples. Update README and changelog to close the v0.4.1 milestone. The generic Content API page (`docs/docs/embeddings/multimodal.md`) already exists from Phase 5 — this phase adds provider-specific content on top of that foundation.

</domain>

<decisions>
## Implementation Decisions

### Scope correction
- **D-01:** Phase covers Gemini + VoyageAI documentation (not Nemotron). Phase 7 pivoted from vLLM/Nemotron to VoyageAI — docs must match what was built.
- **D-02:** Correct the phase name in ROADMAP.md to reference VoyageAI instead of Nemotron.

### Provider section updates in embeddings.md
- **D-03:** Add a "Multimodal (Content API)" subsection under both the Gemini and VoyageAI sections in `docs/docs/embeddings.md`.
- **D-04:** Keep existing text-only `EmbedDocuments` examples intact — do not restructure sections multimodal-first.
- **D-05:** Show Content API usage (`EmbedContent`/`EmbedContents`) with image and video embedding examples in each provider's multimodal subsection.
- **D-06:** Update Gemini default model reference to `gemini-embedding-2-preview` (changed in Phase 6).
- **D-07:** Update VoyageAI section to list available option functions (currently minimal compared to Gemini section).

### Runnable examples
- **D-08:** Add `examples/v2/gemini_multimodal/` with a runnable Go program demonstrating Gemini Content API usage (text + image embedding).
- **D-09:** Add `examples/v2/voyage_multimodal/` with a runnable Go program demonstrating VoyageAI Content API usage (text + image embedding).

### README and changelog
- **D-10:** Update README.md to mention multimodal Content API support, Gemini multimodal, and VoyageAI multimodal as new capabilities.
- **D-11:** Prepare changelog entries summarizing v0.4.1: Content API, portable intents, Gemini multimodal adoption, VoyageAI multimodal adoption.

### Claude's Discretion
- Exact code snippets in doc subsections and examples (follow established patterns from other provider sections)
- Changelog format and level of detail
- README section placement and wording
- Whether VoyageAI option functions list matches Gemini's level of detail or stays briefer

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Content API foundation
- `docs/docs/embeddings/multimodal.md` — Existing Content API page (Phase 5). Provider docs should cross-reference, not duplicate.
- `docs/docs/embeddings.md` — Main embeddings page with all provider sections. Lines 361-394 (VoyageAI) and 396-445 (Gemini) are the update targets.

### Provider implementations (source of truth for API surface)
- `pkg/embeddings/gemini/content.go` — Gemini ContentEmbeddingFunction implementation
- `pkg/embeddings/gemini/gemini.go` — Gemini base embedding function and options
- `pkg/embeddings/gemini/option.go` — Gemini option functions
- `pkg/embeddings/voyage/content.go` — VoyageAI ContentEmbeddingFunction implementation
- `pkg/embeddings/voyage/voyage.go` — VoyageAI base embedding function and options
- `pkg/embeddings/voyage/option.go` — VoyageAI option functions

### Prior decisions
- `.planning/phases/05-documentation-and-verification/05-CONTEXT.md` — Phase 5 doc decisions: page structure, migration framing, no deprecation signal

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `docs/docs/embeddings/multimodal.md` — Content API page already written with Quick Start, Mixed-Part, Intents, Options, and Compatibility sections
- `docs/docs/embeddings.md` — Provider sections follow consistent pattern: intro paragraph, option functions list, code example
- `examples/v2/` — Existing examples follow pattern: single `main.go` per directory with package main

### Established Patterns
- Provider doc sections: brief intro → API key setup → available models link → option functions list → code example
- Doc snippets use `{% codetabs %}` template syntax for language tabs
- Examples are standalone `main.go` files with inline comments

### Integration Points
- Gemini section (embeddings.md line 396) — insert multimodal subsection after existing text-only example
- VoyageAI section (embeddings.md line 361) — insert multimodal subsection after existing text-only example
- README.md — add multimodal mention in features/capabilities section
- CHANGELOG or release notes — new file or section for v0.4.1

</code_context>

<specifics>
## Specific Ideas

- Gemini default model changed to `gemini-embedding-2-preview` in Phase 6 — docs must reflect this
- VoyageAI section is currently sparse compared to Gemini — bring it up to comparable detail level
- Phase 5 framed both APIs as coexisting indefinitely — maintain this framing in provider sections

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 08-document-gemini-and-nemotron-multimodal-embedding-functions*
*Context gathered: 2026-03-23*
