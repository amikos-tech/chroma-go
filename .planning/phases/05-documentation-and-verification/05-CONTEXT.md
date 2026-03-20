# Phase 5: Documentation and Verification - Context

**Gathered:** 2026-03-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Document the portable multimodal Content API and verify the foundation through docs, examples, and acceptance-criteria-driven test audit before Gemini (Phase 6) and vLLM/Nemotron (Phase 7) provider adoption. This phase does not add new provider implementations, change shared contract types, or update provider-specific doc sections.

</domain>

<decisions>
## Implementation Decisions

### Docs structure
- Rewrite `docs/go-examples/docs/embeddings/multimodal.md` to BE the new Content API page — one page, one location
- The rewritten page replaces the current "not available in Go" Python-only content with Go-native multimodal Content API docs
- No separate dedicated page — multimodal.md becomes the Content API page
- Content API caller-facing docs only — do not document CapabilityAware, IntentMapper, or provider-author internals (defer to godoc)
- Add a brief cross-link from the top of `docs/docs/embeddings.md` pointing to the rewritten multimodal.md for multimodal usage
- No changes to individual provider sections in embeddings.md — provider-specific updates happen in Phases 6-7

### Page flow and sections
- Use "Quick start → Deep dive" narrative structure:
  1. Quick Start — minimal text-only EmbedContent example
  2. Mixed-Part Requests — text + image content construction
  3. Portable Intents — setting retrieval_query, retrieval_document, etc.
  4. Request Options — dimension, with an admonition note: "Advanced: You can pass raw intent strings and provider-specific hints. See godoc for details."
  5. Compatibility with Legacy API — comparison table + brief adapter mention

### Example scenarios (doc snippets only)
- Text-only content via EmbedContent/EmbedContents — demonstrates simplest case with the new API
- Mixed-part content (text + image) — demonstrates core multimodal value using Roboflow
- Intent and dimension usage — shows setting portable intents and output dimension on Content requests
- Legacy compatibility — shows EmbedDocuments/EmbedImages code still works unchanged
- No error handling snippets — happy paths only
- No runnable examples/v2/ programs — doc snippets in the page are sufficient; real provider examples come in Phases 6-7

### Migration framing
- Frame as "new API alongside old" — no deprecation signal, both APIs coexist indefinitely
- Include a simple "when to use which API" table: text-only → EmbedDocuments works fine | mixed media → Content API | need intents/dimensions → Content API
- Brief one-sentence mention that existing providers auto-work with Content API through built-in adapters

### Escape hatch documentation
- Just a note — single admonition box in Request Options section referencing godoc
- Do not document the "portable field wins" conflict rule — implementation detail users discover naturally

### Provider-specific updates
- Defer all provider doc changes to Phases 6-7
- No updates to Roboflow, Gemini, or any provider section in embeddings.md

### README/changelog
- Defer README and changelog updates until the v0.4.1 milestone fully ships (after Phases 6-7)
- Phase 5 does not advertise the multimodal foundations in the repo README

### Test verification
- Acceptance-criteria audit approach: produce a coverage gap report first, then fix gaps
- Walk DOCS-02 requirements: shared type validation, compatibility adapters, registry/config round-trips, unsupported-combination failures
- Verify existing Phase 1-4 tests cover each criterion; write missing tests where gaps exist
- No build tag for new tests — unit tests in pkg/embeddings with no external dependencies follow existing pattern (multimodal_test.go, capabilities_test.go have no tags)
- Tests go directly in pkg/embeddings test files

### Claude's Discretion
- Exact wording and formatting of the cross-link in embeddings.md
- Specific code snippets and import paths in doc examples
- Coverage report format and organization
- How to structure the gap-filling tests within existing test files vs new files

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Docs targets
- `docs/go-examples/docs/embeddings/multimodal.md` — Current page to rewrite completely (currently Python-only, says "not available in Go")
- `docs/docs/embeddings.md` — Provider reference page; add cross-link only, no content changes

### Shared contract types (source of truth for doc content)
- `pkg/embeddings/multimodal.go` — Content, Part, Intent, Modality types and 5 neutral intent constants
- `pkg/embeddings/capabilities.go` — CapabilityMetadata struct (not documented in user-facing docs, but informs what's possible)
- `pkg/embeddings/multimodal_validate.go` — ValidationError, validation codes
- `pkg/embeddings/multimodal_compat.go` — Compatibility adapters (brief mention in docs, not detailed)
- `pkg/embeddings/embedding.go` — ContentEmbeddingFunction, EmbeddingFunction, MultimodalEmbeddingFunction interfaces
- `pkg/embeddings/intent_mapper.go` or inline in embedding.go — IntentMapper interface (not documented in user-facing docs)

### Existing tests to audit
- `pkg/embeddings/multimodal_test.go` — Content construction, part ordering tests
- `pkg/embeddings/multimodal_validation_test.go` — Validation rule tests
- `pkg/embeddings/capabilities_test.go` — Capability metadata query tests
- `pkg/embeddings/registry_test.go` — Registry and content factory tests
- `pkg/embeddings/content_validate_test.go` — Content support validation tests
- `pkg/embeddings/intent_mapper_test.go` — Intent mapping contract tests

### Requirements
- `.planning/ROADMAP.md` — Phase 5 goal, success criteria [DOCS-01, DOCS-02]
- `.planning/REQUIREMENTS.md` — DOCS-01 (portable intent docs, escape hatches, compatibility) and DOCS-02 (test coverage for validation, compatibility, registry, unsupported failures)

### Prior phase context
- `.planning/phases/01-shared-multimodal-contract/01-CONTEXT.md` — Request shape, intent ergonomics, validation decisions
- `.planning/phases/03-registry-and-config-integration/03-CONTEXT.md` — Registry extension, config build chain, auto-wiring decisions
- `.planning/phases/04-provider-mapping-and-explicit-failures/04-CONTEXT.md` — IntentMapper contract, escape hatch behavior, failure semantics

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/embeddings/multimodal.go`: Content, Part constructors (NewTextPart, NewImagePartFromFile, etc.), Intent constants — primary source for doc examples
- `pkg/embeddings/embedding.go`: EmbedContent/EmbedContents signatures — the API to document
- `pkg/embeddings/roboflow/roboflow.go`: Only current multimodal provider — basis for mixed-part doc examples
- `docs/docs/embeddings.md`: Roboflow section with existing text/image examples that show legacy API patterns

### Established Patterns
- Doc pages in `docs/go-examples/docs/` use markdown with `{% codetabs %}` / `{% codetab %}` template syntax for multi-language examples
- Test files in `pkg/embeddings/` use no build tags for unit tests, `testify` for assertions
- Existing test files already cover per-phase concerns; Phase 5 gap-filling extends these files

### Integration Points
- `docs/go-examples/docs/embeddings/multimodal.md` — rewrite target
- `docs/docs/embeddings.md` — cross-link insertion point (near top, before provider sections)
- `pkg/embeddings/*_test.go` — gap-filling test insertion points

</code_context>

<specifics>
## Specific Ideas

- The "Quick Start" snippet should be a minimal text-only EmbedContent call — shows the new API works for the simplest case, not just multimodal
- The comparison table should make clear that EmbedDocuments is NOT deprecated — it's the right choice for text-only use cases
- The adapter mention should be exactly one sentence: "Existing providers automatically work with the Content API through built-in adapters"
- The escape hatch admonition should point to godoc, not attempt to explain the full provider-hints mechanism inline

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 05-documentation-and-verification*
*Context gathered: 2026-03-20*
