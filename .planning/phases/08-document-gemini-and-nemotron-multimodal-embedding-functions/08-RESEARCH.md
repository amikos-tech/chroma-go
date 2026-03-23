# Phase 8: Document Gemini and VoyageAI Multimodal Embedding Functions - Research

**Researched:** 2026-03-23
**Domain:** Documentation (provider-specific multimodal docs, runnable examples, README, changelog)
**Confidence:** HIGH

## Summary

Phase 8 is a documentation-only phase that closes the v0.4.1 milestone. All implementation work is complete (Phases 1-7). This phase updates provider-specific documentation for Gemini and VoyageAI to show Content API multimodal usage, adds runnable examples, updates the README to mention multimodal capabilities, prepares a changelog, and corrects the phase name in ROADMAP.md from "Nemotron" to "VoyageAI".

The existing codebase provides all source-of-truth information needed: option functions, constructors, capability metadata, Content API signatures, and the existing documentation structure. The multimodal Content API generic page (`docs/docs/embeddings/multimodal.md`) already exists from Phase 5. Phase 8 layers provider-specific content on top of that foundation.

**Primary recommendation:** Follow the established provider documentation pattern (intro, option functions list, code example) and add a "Multimodal (Content API)" subsection after the existing text-only example in each provider's section of `embeddings.md`. Add standalone `main.go` examples in `examples/v2/gemini_multimodal/` and `examples/v2/voyage_multimodal/`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Phase covers Gemini + VoyageAI documentation (not Nemotron). Phase 7 pivoted from vLLM/Nemotron to VoyageAI -- docs must match what was built.
- **D-02:** Correct the phase name in ROADMAP.md to reference VoyageAI instead of Nemotron.
- **D-03:** Add a "Multimodal (Content API)" subsection under both the Gemini and VoyageAI sections in `docs/docs/embeddings.md`.
- **D-04:** Keep existing text-only `EmbedDocuments` examples intact -- do not restructure sections multimodal-first.
- **D-05:** Show Content API usage (`EmbedContent`/`EmbedContents`) with image and video embedding examples in each provider's multimodal subsection.
- **D-06:** Update Gemini default model reference to `gemini-embedding-2-preview` (changed in Phase 6).
- **D-07:** Update VoyageAI section to list available option functions (currently minimal compared to Gemini section).
- **D-08:** Add `examples/v2/gemini_multimodal/` with a runnable Go program demonstrating Gemini Content API usage (text + image embedding).
- **D-09:** Add `examples/v2/voyage_multimodal/` with a runnable Go program demonstrating VoyageAI Content API usage (text + image embedding).
- **D-10:** Update README.md to mention multimodal Content API support, Gemini multimodal, and VoyageAI multimodal as new capabilities.
- **D-11:** Prepare changelog entries summarizing v0.4.1: Content API, portable intents, Gemini multimodal adoption, VoyageAI multimodal adoption.

### Claude's Discretion
- Exact code snippets in doc subsections and examples (follow established patterns from other provider sections)
- Changelog format and level of detail
- README section placement and wording
- Whether VoyageAI option functions list matches Gemini's level of detail or stays briefer

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

## Architecture Patterns

### Documentation File Structure (existing)
```
docs/docs/
  embeddings.md              # Main provider reference (UPDATE target)
  embeddings/
    multimodal.md            # Content API page (Phase 5, cross-reference target)

examples/v2/
  gemini_multimodal/         # NEW -- runnable Gemini multimodal example
    main.go
  voyage_multimodal/         # NEW -- runnable VoyageAI multimodal example
    main.go

README.md                    # UPDATE -- add multimodal mentions
CHANGELOG.md                 # NEW -- v0.4.1 changelog
.planning/ROADMAP.md         # UPDATE -- correct phase name
```

### Pattern 1: Provider Section Structure in embeddings.md
**What:** Each provider section follows a consistent pattern in `embeddings.md`.
**Current Gemini pattern (lines 396-445):**
1. Provider intro with API key link
2. Available models link with default model stated
3. "Supported Embedding Function Options" list
4. Code example using `EmbedDocuments`

**Current VoyageAI pattern (lines 361-394):**
1. Provider intro with API key link
2. Available models link with default model stated
3. Code example using `EmbedDocuments`
4. **Missing:** Option functions list (D-07 addresses this)

**Phase 8 addition pattern:** Insert a "### Multimodal (Content API)" subsection after the existing text-only code block in each provider section. This subsection shows `EmbedContent`/`EmbedContents` usage with image embedding.

### Pattern 2: Runnable Example Structure
**What:** Standalone `main.go` files in `examples/v2/<name>/`.
**Established pattern** (from `embedding_function_basic/main.go`):
- `package main` with inline imports
- Create embedding function with `WithEnvAPIKey()`
- Demonstrate the API call
- Print results
- Use `log.Fatalf` for error handling

### Pattern 3: README Embedding Section
**What:** The README lists embedding providers in a flat bullet list under "Embedding API and Models Support" (lines 267-295).
**Update pattern:** Add multimodal mention to existing Gemini and VoyageAI lines, and add a new bullet for Content API support.

### Anti-Patterns to Avoid
- **Restructuring existing sections multimodal-first:** D-04 explicitly forbids this. Add multimodal as a subsection after existing content.
- **Duplicating Content API docs:** The generic Content API page (`multimodal.md`) exists. Provider sections should cross-reference it, not re-explain Content/Part/Intent concepts.
- **Referencing Nemotron in docs:** The pivot happened in Phase 7. All documentation must reference VoyageAI.
- **Using deprecated default model:** Gemini docs currently say `gemini-embedding-001` is default. Code uses `gemini-embedding-2-preview`. Fix per D-06.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Changelog format | Custom format | Standard Keep a Changelog format (keepachangelog.com) | Universally recognized, scannable |
| Doc examples | Fabricated API calls | Real constructor signatures from source code | Must match actual API surface |

**Key insight:** This is a documentation phase. All content must be sourced directly from existing code (option.go, content.go, gemini.go, voyage.go). No API surface changes.

## Common Pitfalls

### Pitfall 1: Stale Default Model in Gemini Docs
**What goes wrong:** Docs say `gemini-embedding-001` is default but code was updated to `gemini-embedding-2-preview` in Phase 6.
**Why it happens:** The embeddings.md Gemini section was written before Phase 6.
**How to avoid:** Update the default model reference in the Gemini section intro text AND in any code examples showing `WithDefaultModel`.
**Warning signs:** Any occurrence of `gemini-embedding-001` as "default" in docs.

### Pitfall 2: Incomplete VoyageAI Option Functions List
**What goes wrong:** VoyageAI section has no option functions list, unlike Gemini which lists 8 options.
**Why it happens:** The VoyageAI section was written before the multimodal work added new option functions.
**How to avoid:** Audit `pkg/embeddings/voyage/option.go` and list all available options: `WithDefaultModel`, `WithMaxBatchSize`, `WithDefaultHeaders`, `WithAPIKey`, `WithEnvAPIKey`, `WithAPIKeyFromEnvVar`, `WithHTTPClient`, `WithTruncation`, `WithEncodingFormat`, `WithBaseURL`, `WithInsecure`.
**Warning signs:** VoyageAI section significantly shorter than Gemini section.

### Pitfall 3: Missing Cross-References to Content API Page
**What goes wrong:** Provider multimodal subsections re-explain Content/Part/Intent types instead of linking to multimodal.md.
**Why it happens:** Natural tendency to make each section self-contained.
**How to avoid:** Keep multimodal subsections focused on provider-specific usage (construction + EmbedContent call). Link to `multimodal.md` for Content API concepts.

### Pitfall 4: Wrong Modality Coverage in Examples
**What goes wrong:** Showing audio/PDF in VoyageAI examples when VoyageAI only supports text, image, and video.
**Why it happens:** Copying Gemini patterns without checking VoyageAI capabilities.
**How to avoid:** Check `capabilitiesForModel` in each provider's `content.go`:
- **Gemini** (`gemini-embedding-2-preview`): text, image, audio, video, PDF
- **VoyageAI** (`voyage-multimodal-3.5`): text, image, video

### Pitfall 5: Incorrect ROADMAP Phase Name
**What goes wrong:** Phase 8 heading in ROADMAP.md still says "Nemotron".
**Why it happens:** ROADMAP was written before Phase 7 pivoted from vLLM/Nemotron to VoyageAI.
**How to avoid:** D-02 explicitly requires correcting the ROADMAP.md phase name.

### Pitfall 6: VoyageAI Default Model Mismatch
**What goes wrong:** Docs say `voyage-2` is default (true for text-only) but multimodal uses `voyage-multimodal-3.5`.
**Why it happens:** VoyageAI has separate models for text-only vs multimodal.
**How to avoid:** In the multimodal subsection, clearly state that `voyage-multimodal-3.5` is the default multimodal model. Keep the existing text-only section referencing `voyage-2`.

## Code Examples

These are the source-of-truth API surfaces that documentation must reflect.

### Gemini Content API -- EmbedContent (text + image)
```go
// Source: pkg/embeddings/gemini/gemini.go lines 317-323, 415-432
// Source: pkg/embeddings/gemini/content.go (conversion + capabilities)
ef, err := gemini.NewGeminiEmbeddingFunction(gemini.WithEnvAPIKey())
// Default model is "gemini-embedding-2-preview" (multimodal capable)

content := embeddings.Content{
    Parts: []embeddings.Part{
        embeddings.NewTextPart("A cat sitting on a windowsill"),
        embeddings.NewPartFromSource(
            embeddings.ModalityImage,
            embeddings.NewBinarySourceFromURL("https://example.com/cat.jpg"),
        ),
    },
}
emb, err := ef.EmbedContent(ctx, content)
```

### Gemini Option Functions (complete list from option.go)
```
WithAPIKey(apiKey string)              -- Provide Gemini API key directly.
WithEnvAPIKey()                        -- Load API key from GEMINI_API_KEY.
WithAPIKeyFromEnvVar(envVar string)    -- Load API key from a custom env var.
WithDefaultModel(model)                -- Set model. Default: gemini-embedding-2-preview.
WithTaskType(taskType TaskType)        -- Set embedding task type.
WithDimension(dimension int)           -- Set reduced output dimensionality.
WithMaxBatchSize(maxBatchSize int)     -- Upper bound on items per embedding call.
WithClient(client *genai.Client)       -- Provide a preconfigured genai client.
WithMaxFileSize(maxBytes int64)        -- Max payload size for inline binary sources.
```

### VoyageAI Content API -- EmbedContent (text + image)
```go
// Source: pkg/embeddings/voyage/voyage.go lines 297-303
// Source: pkg/embeddings/voyage/content.go lines 317-358
ef, err := voyage.NewVoyageAIEmbeddingFunction(
    voyage.WithEnvAPIKey(),
    voyage.WithDefaultModel("voyage-multimodal-3.5"),
)

content := embeddings.Content{
    Parts: []embeddings.Part{
        embeddings.NewTextPart("A dog running on a beach"),
        embeddings.NewPartFromSource(
            embeddings.ModalityImage,
            embeddings.NewBinarySourceFromURL("https://example.com/dog.jpg"),
        ),
    },
}
emb, err := ef.EmbedContent(ctx, content)
```

### VoyageAI Option Functions (complete list from option.go)
```
WithDefaultModel(model)                -- Set model. Default (text): voyage-2.
WithMaxBatchSize(size int)             -- Upper bound (max 128).
WithDefaultHeaders(headers)            -- Custom HTTP headers.
WithAPIKey(apiToken string)            -- Provide API key directly.
WithEnvAPIKey()                        -- Load API key from VOYAGE_API_KEY.
WithAPIKeyFromEnvVar(envVar string)    -- Load API key from a custom env var.
WithHTTPClient(client *http.Client)    -- Custom HTTP client.
WithTruncation(truncation bool)        -- Enable/disable input truncation.
WithEncodingFormat(format)             -- Response encoding format (e.g., base64).
WithBaseURL(baseURL string)            -- Custom API base URL.
WithInsecure()                         -- Allow HTTP (no TLS). Dev/testing only.
```

### VoyageAI Context Overrides (voyage.go)
```go
// Per-request overrides via context
ctx = voyage.ContextWithInputType(ctx, voyage.InputTypeQuery)
ctx = voyage.ContextWithModel(ctx, "voyage-multimodal-3.5")
ctx = voyage.ContextWithTruncation(ctx, false)
```

### Gemini Context Overrides (gemini.go)
```go
// Per-request overrides via context
ctx = gemini.ContextWithModel(ctx, "gemini-embedding-2-preview")
ctx = gemini.ContextWithTaskType(ctx, gemini.TaskTypeRetrievalQuery)
ctx = gemini.ContextWithDimension(ctx, 256)
```

### Capability Summary Table (for doc reference)

| Provider | Model | Modalities | Intents | Dimension | Mixed-Part |
|----------|-------|------------|---------|-----------|------------|
| Gemini | gemini-embedding-2-preview | text, image, audio, video, PDF | all 5 neutral | yes | yes |
| Gemini | gemini-embedding-001 (legacy) | text only | all 5 neutral | yes | no |
| VoyageAI | voyage-multimodal-3.5 | text, image, video | retrieval_query, retrieval_document | yes | yes |
| VoyageAI | voyage-multimodal-3 | text, image | retrieval_query, retrieval_document | no | yes |
| VoyageAI | voyage-2 (default text) | text only | -- | no | no |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Gemini default `gemini-embedding-001` | Gemini default `gemini-embedding-2-preview` | Phase 6 | Docs must reference new default; legacy constant available as `LegacyEmbeddingModel` |
| Phase 7 was "vLLM/Nemotron" | Phase 7 is "VoyageAI multimodal" | Phase 7 pivot | ROADMAP and phase name must reference VoyageAI |
| VoyageAI text-only | VoyageAI multimodal (text, image, video) | Phase 7 | New multimodal subsection needed in VoyageAI docs |
| No Content API docs | Generic Content API page exists | Phase 5 | Provider sections cross-reference multimodal.md |

## Specific Edit Targets

### embeddings.md Updates

**VoyageAI section (starts line 361):**
1. Add option functions list after intro (D-07)
2. Keep existing `EmbedDocuments` code example unchanged (D-04)
3. Add "### Multimodal (Content API)" subsection after code example (D-03, D-05)
4. Cross-link to `multimodal.md` for Content API concepts

**Gemini section (starts line 396):**
1. Update default model reference from `gemini-embedding-001` to `gemini-embedding-2-preview` in intro text (D-06)
2. Update `WithDefaultModel` description to reflect new default (D-06)
3. Add `WithMaxFileSize` to option functions list (added in Phase 6, missing from docs)
4. Keep existing `EmbedDocuments` code example but update model in example to `gemini-embedding-2-preview` (D-06)
5. Add "### Multimodal (Content API)" subsection after code example (D-03, D-05)
6. Cross-link to `multimodal.md` for Content API concepts

### README.md Updates (D-10)

**Section: "Embedding API and Models Support" (lines 267-295):**
- Update Gemini line to mention multimodal support
- Update VoyageAI line to mention multimodal support
- Add a new capability bullet in "Additional support features" for Content API / multimodal foundations

**Section: "Examples" table (lines 193-207):**
- Add rows for `gemini_multimodal` and `voyage_multimodal` examples

### ROADMAP.md Corrections (D-02)

**Lines to update:**
- Line 13: Milestone goal mentions "vLLM/Nemotron" -- update to VoyageAI
- Line 147: Phase 8 heading says "Nemotron" -- update to VoyageAI
- Phase 7 description (line 23) already says VoyageAI -- verify consistency

### Changelog (D-11)

**New file: `CHANGELOG.md`**
Format: Keep a Changelog (keepachangelog.com)
Content: v0.4.1 release notes covering:
- Content API (shared multimodal request types, portable intents, per-request options)
- Capability metadata and compatibility adapters
- Registry/config integration for multimodal functions
- Intent mapping and explicit failure semantics
- Gemini multimodal adoption (text, image, audio, video, PDF)
- VoyageAI multimodal adoption (text, image, video)

## Project Constraints (from CLAUDE.md)

- Use conventional commits
- Always lint before committing (`make lint`)
- Keep things radically simple
- Do not leave too verbose comments
- Run `make lint` before committing
- Examples target V2 API (`/examples/v2/`)
- No panic in production code (not relevant for docs, but examples should use `log.Fatalf` not `panic`)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify |
| Config file | Makefile build tags |
| Quick run command | `make lint` |
| Full suite command | `make lint && make build` |

### Phase Requirements -> Test Map

This is a documentation-only phase. There are no new code requirements to test. Validation is:

| Aspect | Validation Method | Automated? |
|--------|-------------------|------------|
| Docs compile/render | `make lint` (catches Go syntax in examples) | Yes |
| Links work | Manual review | No |
| Code examples match API | Visual audit against source | No |
| ROADMAP consistency | Visual audit | No |

### Sampling Rate
- **Per task commit:** `make lint`
- **Per wave merge:** `make lint && make build`
- **Phase gate:** Lint clean + visual doc review

### Wave 0 Gaps
None -- this is a documentation phase with no new test infrastructure needed.

## Open Questions

1. **Changelog location**
   - What we know: No existing `CHANGELOG.md` or `CHANGES.md` in repo root.
   - What's unclear: Whether the project prefers GitHub releases over a changelog file.
   - Recommendation: Create `CHANGELOG.md` in repo root per D-11. If the project uses GitHub releases, the same content can be used for the release notes.

2. **Examples as compilable vs illustrative**
   - What we know: Existing examples in `examples/v2/` are full `package main` programs. The multimodal examples need API keys to actually run.
   - What's unclear: Whether to include build instructions or just the Go file.
   - Recommendation: Single `main.go` per example directory following existing pattern. Users set env vars (GEMINI_API_KEY / VOYAGE_API_KEY) to run.

## Sources

### Primary (HIGH confidence)
- `pkg/embeddings/gemini/option.go` -- complete Gemini option functions list
- `pkg/embeddings/gemini/gemini.go` -- DefaultEmbeddingModel constant, constructor, Content API methods
- `pkg/embeddings/gemini/content.go` -- capabilitiesForModel, convertToGenaiContent
- `pkg/embeddings/voyage/option.go` -- complete VoyageAI option functions list
- `pkg/embeddings/voyage/voyage.go` -- defaultModel constant, constructor, Content API methods
- `pkg/embeddings/voyage/content.go` -- capabilitiesForModel, multimodal types
- `docs/docs/embeddings.md` -- current provider docs structure (lines 361-445)
- `docs/docs/embeddings/multimodal.md` -- existing Content API page from Phase 5
- `examples/v2/embedding_function_basic/main.go` -- example file pattern
- `README.md` -- current features section structure

### Secondary (MEDIUM confidence)
- `.planning/phases/05-documentation-and-verification/05-CONTEXT.md` -- Phase 5 doc framing decisions

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new libraries, all documentation of existing code
- Architecture: HIGH -- following established documentation patterns with clear edit targets
- Pitfalls: HIGH -- identified from direct code inspection (stale model refs, missing option lists, modality differences)

**Research date:** 2026-03-23
**Valid until:** 2026-04-23 (stable -- documentation of finalized implementation)
