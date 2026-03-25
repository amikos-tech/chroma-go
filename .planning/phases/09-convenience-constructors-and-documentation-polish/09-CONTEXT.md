# Phase 9: Convenience Constructors and Documentation Polish - Context

**Gathered:** 2026-03-25
**Status:** Ready for planning

<domain>
## Phase Boundary

Add shorthand constructors to reduce Content API verbosity for common modality+source combinations. Update multimodal docs and provider examples to use the shorthand forms. Constructors are additive sugar — they do not replace or deprecate the existing `Content{}` struct literal or Part-level helpers.

</domain>

<decisions>
## Implementation Decisions

### Return type
- **D-01:** Convenience constructors return `Content` (not `Part`), ready to pass directly to `EmbedContent`. Existing `NewTextPart`/`NewPartFromSource` remain for composing mixed-part content.

### Constructor scope
- **D-02:** Ship 7 single-modality constructors: `NewTextContent(text)`, `NewImageURL(url)`, `NewImageFile(path)`, `NewVideoURL(url)`, `NewVideoFile(path)`, `NewAudioFile(path)`, `NewPDFFile(path)`.
- **D-03:** Add `NewContent(parts []Part, opts ...ContentOption)` for composing multi-part content from Part-level helpers. Uses slice for parts (not variadic) to stay consistent with existing repo patterns — no marker interfaces.
- **D-04:** No base64 or bytes variants in this phase. Workaround: use the verbose `Content{Parts: []Part{NewPartFromSource(...)}}` pattern. Expand later only if real usage demands it.

### Options support
- **D-05:** All convenience constructors accept variadic `ContentOption` functional options for Intent, Dimension, and ProviderHints (e.g. `NewImageFile(path, WithIntent(...), WithDimension(...))`).
- **D-06:** `ContentOption` is a parameter type for convenience constructors only. Manual `Content{}` struct literals continue to use direct field assignment — no `Apply` method added.

### Documentation
- **D-07:** Shorthand-first in provider docs — lead with convenience constructors as primary examples. Verbose forms referenced via link to the generic Content API page (`docs/docs/embeddings/multimodal.md`).
- **D-08:** Rewrite existing Gemini and VoyageAI multimodal examples (`examples/v2/gemini_multimodal/`, `examples/v2/voyage_multimodal/`) in-place to use convenience constructors.

### Claude's Discretion
- File placement for new constructors (new file vs extending `multimodal_compat.go`)
- Exact `ContentOption` type definition (func type vs interface — follow whichever is simpler)
- Whether `WithProviderHints` takes `map[string]any` or individual key-value pairs
- Test structure and assertion patterns

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Content API types (source of truth for Content/Part/BinarySource)
- `pkg/embeddings/multimodal.go` — Content, Part, BinarySource, Modality, Intent type definitions
- `pkg/embeddings/multimodal_compat.go` — Existing Part-level helpers: NewTextPart, NewPartFromSource, NewBinarySourceFrom{URL,File,Base64,Bytes}

### Provider implementations (must work with new constructors)
- `pkg/embeddings/gemini/content.go` — Gemini ContentEmbeddingFunction (consumes Content)
- `pkg/embeddings/voyage/content.go` — VoyageAI ContentEmbeddingFunction (consumes Content)

### Docs and examples (update targets)
- `docs/docs/embeddings/multimodal.md` — Generic Content API page (Phase 5). Verbose forms stay here.
- `docs/docs/embeddings.md` — Provider sections with multimodal subsections (Phase 8)
- `examples/v2/gemini_multimodal/main.go` — Rewrite target
- `examples/v2/voyage_multimodal/main.go` — Rewrite target

### Prior decisions
- `.planning/phases/05-documentation-and-verification/05-CONTEXT.md` — Both APIs coexist indefinitely, no deprecation signal
- `.planning/phases/08-document-gemini-and-nemotron-multimodal-embedding-functions/08-CONTEXT.md` — Doc structure, provider section patterns

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `NewTextPart(text)` — Part-level text constructor, already exists
- `NewPartFromSource(modality, source)` — Part-level binary constructor, already exists
- `NewBinarySourceFrom{URL,File,Base64,Bytes}` — BinarySource constructors, already exist
- Functional option pattern used extensively in `pkg/api/v2/` (e.g. `ClientOption`, `SchemaOption`, `SearchCollectionOption`)

### Established Patterns
- Functional options as `type FooOption func(*FooConfig) error` — dominant pattern in repo
- `ApplyToX` interface pattern in `pkg/api/v2/options.go` for cross-operation options
- No marker/sealed interface pattern in codebase — slice approach chosen for `NewContent`

### Integration Points
- `ContentEmbeddingFunction.EmbedContent(ctx, Content)` — all providers consume Content directly
- `ContentEmbeddingFunction.EmbedContents(ctx, []Content)` — batch variant
- Examples and docs already use the verbose Content construction pattern

</code_context>

<specifics>
## Specific Ideas

- `NewContent(parts []Part, opts ...ContentOption)` bridges the gap between Part-level and Content-level APIs for mixed-part use cases
- The 7 single-modality constructors cover the common cases (URL + file for visual modalities, file-only for audio/PDF) without API bloat
- ContentOption functions (`WithIntent`, `WithDimension`, `WithProviderHints`) match the functional options idiom used throughout the codebase

</specifics>

<deferred>
## Deferred Ideas

- **Base64/Bytes convenience constructors** — Add `NewImage{Base64,Bytes}` etc. if real usage demands it. Users can use verbose pattern or convert data types as workaround.
- **`Content.Apply(opts ...ContentOption)` method** — Universal option application for manual Content{} construction. Deferred to keep scope minimal.

</deferred>

---

*Phase: 09-convenience-constructors-and-documentation-polish*
*Context gathered: 2026-03-25*
