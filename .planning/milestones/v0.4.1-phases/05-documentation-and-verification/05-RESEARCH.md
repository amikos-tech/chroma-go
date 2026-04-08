# Phase 5: Documentation and Verification - Research

**Researched:** 2026-03-20
**Domain:** Go documentation authoring + Go unit test gap analysis for a multimodal embedding API
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Docs structure**
- Rewrite `docs/go-examples/docs/embeddings/multimodal.md` to BE the new Content API page — one page, one location
- The rewritten page replaces the current "not available in Go" Python-only content with Go-native multimodal Content API docs
- No separate dedicated page — multimodal.md becomes the Content API page
- Content API caller-facing docs only — do not document CapabilityAware, IntentMapper, or provider-author internals (defer to godoc)
- Add a brief cross-link from the top of `docs/docs/embeddings.md` pointing to the rewritten multimodal.md for multimodal usage
- No changes to individual provider sections in embeddings.md — provider-specific updates happen in Phases 6-7

**Page flow and sections**
- Use "Quick start → Deep dive" narrative structure:
  1. Quick Start — minimal text-only EmbedContent example
  2. Mixed-Part Requests — text + image content construction
  3. Portable Intents — setting retrieval_query, retrieval_document, etc.
  4. Request Options — dimension, with an admonition note: "Advanced: You can pass raw intent strings and provider-specific hints. See godoc for details."
  5. Compatibility with Legacy API — comparison table + brief adapter mention

**Example scenarios (doc snippets only)**
- Text-only content via EmbedContent/EmbedContents — demonstrates simplest case with the new API
- Mixed-part content (text + image) — demonstrates core multimodal value using Roboflow
- Intent and dimension usage — shows setting portable intents and output dimension on Content requests
- Legacy compatibility — shows EmbedDocuments/EmbedImages code still works unchanged
- No error handling snippets — happy paths only
- No runnable examples/v2/ programs — doc snippets in the page are sufficient; real provider examples come in Phases 6-7

**Migration framing**
- Frame as "new API alongside old" — no deprecation signal, both APIs coexist indefinitely
- Include a simple "when to use which API" table: text-only → EmbedDocuments works fine | mixed media → Content API | need intents/dimensions → Content API
- Brief one-sentence mention that existing providers auto-work with Content API through built-in adapters

**Escape hatch documentation**
- Just a note — single admonition box in Request Options section referencing godoc
- Do not document the "portable field wins" conflict rule — implementation detail users discover naturally

**Provider-specific updates**
- Defer all provider doc changes to Phases 6-7
- No updates to Roboflow, Gemini, or any provider section in embeddings.md

**README/changelog**
- Defer README and changelog updates until the v0.4.1 milestone fully ships (after Phases 6-7)
- Phase 5 does not advertise the multimodal foundations in the repo README

**Test verification**
- Acceptance-criteria audit approach: produce a coverage gap report first, then fix gaps
- Walk DOCS-02 requirements: shared type validation, compatibility adapters, registry/config round-trips, unsupported-combination failures
- Verify existing Phase 1-4 tests cover each criterion; write missing tests where gaps exist
- No build tag for new tests — unit tests in pkg/embeddings with no external dependencies follow existing pattern
- Tests go directly in pkg/embeddings test files

### Claude's Discretion
- Exact wording and formatting of the cross-link in embeddings.md
- Specific code snippets and import paths in doc examples
- Coverage report format and organization
- How to structure the gap-filling tests within existing test files vs new files

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DOCS-01 | Public docs explain portable intent usage, provider-specific escape hatches, and compatibility expectations for multimodal callers | All five Content API sections mapped below; intent constants and adapter sentence ready for doc authoring |
| DOCS-02 | Tests cover shared type validation, compatibility adapters, registry/config round-trips, and unsupported-combination failures | Gap analysis below shows which criteria are covered and which need new test cases |
</phase_requirements>

---

## Summary

Phase 5 is a documentation and test-gap-fill phase, not a code-change phase. There are two concrete deliverables: (1) a rewritten `docs/go-examples/docs/embeddings/multimodal.md` that introduces the Go-native Content API, and (2) any missing unit tests in `pkg/embeddings/` that close DOCS-02 acceptance criteria gaps.

The codebase is fully implemented through Phase 4. All shared contract types (`Content`, `Part`, `Intent`, `BinarySource`), validation helpers (`ValidateContents`, `ValidateContentSupport`), compatibility adapters (`AdaptEmbeddingFunctionToContent`, `AdaptMultimodalEmbeddingFunctionToContent`), the registry (`BuildContent`, `RegisterContent`), and the intent mapper contract (`IntentMapper`, `IsNeutralIntent`) exist and are in `pkg/embeddings/`. Test files for each concern also exist: `multimodal_test.go`, `multimodal_validation_test.go`, `capabilities_test.go`, `registry_test.go`, `content_validate_test.go`, and `intent_mapper_test.go`.

The gap analysis below (see DOCS-02) shows most DOCS-02 criteria are already covered. The one gap is an explicit end-to-end registry/config round-trip test using `BuildContent` that exercises both the multimodal-registry path and the capability-aware metadata that comes back. The existing `registry_test.go` has `TestBuildContentFallbackCapabilityAware` which is close, but no test that combines `RegisterContent` + `BuildContent` + verifying `EmbedContent` invokes the correct path. This is a minor addition within `registry_test.go`.

**Primary recommendation:** Two sequential work units — (1) rewrite the markdown page using the exact code shapes in the source, then (2) run the test suite, check per-criterion coverage, and add the one missing round-trip test.

---

## Standard Stack

### Core (no new dependencies needed)

This phase introduces no new Go packages. All dependencies are already present.

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `pkg/embeddings` (internal) | current | Content, Part, Intent types and validation | Source of truth for doc snippets |
| `pkg/embeddings/roboflow` (internal) | current | Only existing `ContentEmbeddingFunction` provider — basis for mixed-part doc example | Verified in roboflow.go |
| `github.com/stretchr/testify` | current | Test assertions | Already used across all test files |

### Markdown Authoring

| Concern | Standard | Notes |
|---------|----------|-------|
| Page template | Docusaurus-flavored Markdown (`.md`) | Confirmed by existing pages |
| Multi-language tabs | `{% codetabs group="lang" %}` / `{% codetab label="Go" %}` / `{% /codetabs %}` | Confirmed by `embedding-functions.md` and `multimodal.md` |
| Admonition boxes | Standard Markdown `> **Note:**` or `!!! note` block | Both styles appear; `!!! note` style used in `docs/docs/embeddings.md` |
| Cross-links | Relative markdown links `[text](./path/to/page)` | Standard for the doc site |

---

## Architecture Patterns

### Doc Page Structure

The decided narrative order is:

```
1. Quick Start            — minimal EmbedContent(text-only) snippet
2. Mixed-Part Requests    — text + image with Roboflow
3. Portable Intents       — Intent constants on Content
4. Request Options        — Dimension + escape-hatch admonition
5. Compatibility          — "when to use" table + one-sentence adapter mention
```

This matches the "Quick start → Deep dive" pattern used by all other pages in `docs/go-examples/docs/`.

### Import Path for Doc Snippets

All snippets use the public import path `github.com/amikos-tech/chroma-go/pkg/embeddings`. This is the sole import needed for the shared types. Roboflow examples also import `github.com/amikos-tech/chroma-go/pkg/embeddings/roboflow`.

### Verified API Shapes (from source — HIGH confidence)

**Text-only Content (Quick Start):**
```go
// Source: pkg/embeddings/multimodal.go + multimodal_compat.go
content := embeddings.Content{
    Parts: []embeddings.Part{
        embeddings.NewTextPart("What is Chroma?"),
    },
}
embedding, err := ef.EmbedContent(ctx, content)
```

**Mixed-Part Content (text + image URL):**
```go
// Source: pkg/embeddings/multimodal_compat.go NewBinarySourceFromURL, NewPartFromSource
content := embeddings.Content{
    Parts: []embeddings.Part{
        embeddings.NewTextPart("A dog running on a beach"),
        embeddings.NewPartFromSource(
            embeddings.ModalityImage,
            embeddings.NewBinarySourceFromURL("https://example.com/dog.jpg"),
        ),
    },
}
```
Note: The legacy `multimodalEmbeddingFunctionContentAdapter` (used by Roboflow) requires exactly one Part per Content item. Mixed parts in a single Content work for native `ContentEmbeddingFunction` implementations, but not through the adapter. For Roboflow doc examples, each Content should have one part (either text or image), passed as a batch via `EmbedContents`.

**Intent usage:**
```go
// Source: pkg/embeddings/multimodal.go — Intent constants
content := embeddings.Content{
    Parts:  []embeddings.Part{embeddings.NewTextPart("retrieval query text")},
    Intent: embeddings.IntentRetrievalQuery,
}
// Available: IntentRetrievalQuery, IntentRetrievalDocument, IntentClassification,
//            IntentClustering, IntentSemanticSimilarity
```

**Dimension usage:**
```go
// Source: pkg/embeddings/multimodal.go — Content.Dimension field
dim := 256
content := embeddings.Content{
    Parts:     []embeddings.Part{embeddings.NewTextPart("document text")},
    Dimension: &dim,
}
```

**Legacy API (still works, not deprecated):**
```go
// Source: pkg/embeddings/embedding.go — EmbeddingFunction interface
embeddings, err := ef.EmbedDocuments(ctx, []string{"text1", "text2"})
imageEmbedding, err := ef.EmbedImage(ctx, embeddings.NewImageInputFromURL("https://..."))
```

**Roboflow constructor for examples:**
```go
// Source: pkg/embeddings/roboflow/roboflow.go
ef, err := roboflow.NewRoboflowEmbeddingFunction(
    roboflow.WithEnvAPIKey(),
)
```

### Cross-Link Target

In `docs/docs/embeddings.md`, the cross-link goes near the top, before the provider table. It should reference the Go-specific multimodal doc page. Exact path TBD at writing time based on where the doc site serves the file.

### Test File Pattern (no build tags)

All existing Phase 1-4 test files in `pkg/embeddings/` have no build tags — they are standard `package embeddings` unit tests:

```go
package embeddings

import (
    "testing"
    "github.com/stretchr/testify/require"
)
```

Gap-filling tests follow the same pattern and go into the existing test files (not new files), unless a new file makes the grouping clearer.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Mixed-part doc example provider | A custom ContentEmbeddingFunction just for docs | Roboflow (already implements ContentEmbeddingFunction) | Keeps examples grounded in a real implementation |
| Registry round-trip test harness | A new mock framework | Extend `registry_test.go` with existing mock types (`mockContentEmbeddingFunction`, `mockCapabilityAwareEmbeddingFunction`) | Mock types already defined and tested |

---

## Common Pitfalls

### Pitfall 1: Mixed-Part Requests via the Legacy Adapter

**What goes wrong:** A doc snippet shows a single Content with both a text Part and an image Part sent through Roboflow, which uses `AdaptMultimodalEmbeddingFunctionToContent`. The adapter requires exactly one Part per Content and will return a `validationCodeOneOf` error.

**Why it happens:** The mixed-part constraint lives in `validateCompatibleContent` in `multimodal_compat.go`. Native `ContentEmbeddingFunction` implementations can support `SupportsMixedPart: true`; Roboflow's adapter does not.

**How to avoid:** In the mixed-part doc section, show separate Content items for text and image passed together via `EmbedContents`, not a single Content with both parts.

**Warning signs:** Error message contains "exactly one Part" — `validationCodeOneOf` at path `parts`.

### Pitfall 2: Incorrect Import Paths in Snippets

**What goes wrong:** Doc snippets use `chroma-go/embeddings` or `chroma-go/pkg/embeddings/multimodal` instead of the single package `github.com/amikos-tech/chroma-go/pkg/embeddings`.

**How to avoid:** All shared types (`Content`, `Part`, `Intent`, `BinarySource`, `NewTextPart`, `NewPartFromSource`, `NewBinarySourceFromURL`, etc.) are in the single `embeddings` package. Only the Roboflow constructor requires its own import.

### Pitfall 3: Documenting Adapter Constraints as User-Facing Rules

**What goes wrong:** Explaining that "Intent and Dimension are forbidden" when they are only forbidden by the legacy compatibility adapters, not by the Content API itself.

**How to avoid:** The user-facing framing is "Intent and Dimension work with the Content API; some older providers may not support them yet." Keep adapter failure semantics in godoc, not in the user page.

### Pitfall 4: Test Gap Report Scoped Too Narrowly

**What goes wrong:** The gap analysis only checks test function names and misses that `TestBuildContentFallbackCapabilityAware` tests capability passthrough but does not call `EmbedContent` on the resulting adapter to verify the round-trip dispatches correctly.

**How to avoid:** Check both that the test exists AND that it exercises the DOCS-02 acceptance criterion end-to-end (call returns embedding, not just that the factory builds without error).

---

## DOCS-02 Gap Analysis

This is the core of the test verification work. Walk each DOCS-02 criterion against existing tests.

### Criterion 1: Shared type validation

**Requirement:** Tests cover shared type validation (Content, Part, BinarySource, ValidateContents).

**Covered by:**
- `multimodal_validation_test.go`: `TestMultimodalValidationErrors` — 13 sub-cases covering empty content, dimension, modality, text field rules
- `multimodal_validation_test.go`: `TestMultimodalIntentValidation` — whitespace and valid intent cases
- `multimodal_validation_test.go`: `TestNewImagePartFromImageInput` — bridge function validation
- `content_validate_test.go`: All 8 functions covering `ValidateContentSupport` and `ValidateContentsSupport`

**Gap:** None. Coverage is comprehensive.

### Criterion 2: Compatibility adapters

**Requirement:** Tests cover compatibility adapter behavior (text-only adapter, multimodal adapter, rejection of unsupported content).

**Covered by:**
- `capabilities_test.go`: `TestLegacyTextCompatibility` — text adapter round-trip
- `capabilities_test.go`: `TestLegacyImageCompatibility` — multimodal adapter text + image paths
- `capabilities_test.go`: `TestCompatibilityAdapterRejectsUnsupportedContent` — 8 rejection sub-cases

**Gap:** None. Both adapters tested for happy path and rejection cases.

### Criterion 3: Registry/config round-trips

**Requirement:** Tests cover registry registration, BuildContent fallback chain, and config round-trips.

**Covered by:**
- `registry_test.go`: `TestRegisterAndBuildContent`, `TestBuildContentFallbackMultimodal`, `TestBuildContentFallbackDense`, `TestBuildContentFallbackCapabilityAware`, `TestBuildContentCloseableWithCloseable`, `TestBuildContentUnknown`, `TestListContent`, `TestHasContent`, `TestRegisterContentDuplicate`

**Gap (minor):** No test calls `EmbedContent` on the result of `BuildContent` to verify the factory→adapter→embed dispatch is correct end-to-end. `TestBuildContentFallbackCapabilityAware` asserts the factory builds and capability metadata passes through, but stops before calling `EmbedContent`. A one-call addition to that test (or a new `TestBuildContentRoundTrip`) closes this gap.

**Config persistence note:** There is no config serialization/deserialization round-trip test specifically for the content factory path. However, `GetConfig`/`NewFromConfig` is tested in provider-specific packages (Roboflow). For the shared contract, the registry tests confirm the factory→build chain; config persistence specifics live in provider packages.

### Criterion 4: Unsupported-combination failures

**Requirement:** Tests cover unsupported modality, unsupported intent, and unsupported dimension failures.

**Covered by:**
- `content_validate_test.go`: `TestValidateContentSupportModality` — unsupported modality returns correct error
- `content_validate_test.go`: `TestValidateContentSupportIntent` — unsupported intent returns correct error
- `content_validate_test.go`: `TestValidateContentSupportDimension` — unsupported dimension returns correct error
- `content_validate_test.go`: `TestValidateContentSupportPassThrough` — empty caps pass through
- `content_validate_test.go`: `TestValidateContentSupportCustomIntentBypass` — custom intents bypass neutral check
- `content_validate_test.go`: `TestValidateContentSupportFailOnFirst` — batch fail-on-first behavior
- `content_validate_test.go`: `TestValidateContentsSupportBatch` — batch index prefix on error

**Gap:** None. Failure semantics are comprehensively covered.

### Summary of Gaps

| Gap | Severity | Where to Fix |
|-----|----------|-------------|
| No `EmbedContent` call after `BuildContent` to verify dispatch round-trip | Minor | Extend `TestBuildContentFallbackCapabilityAware` or add `TestBuildContentRoundTrip` in `registry_test.go` |

---

## Code Examples

Verified patterns directly from source files.

### Constructor (roboflow — only existing ContentEmbeddingFunction provider)

```go
// Source: pkg/embeddings/roboflow/roboflow.go NewRoboflowEmbeddingFunction
import (
    "github.com/amikos-tech/chroma-go/pkg/embeddings"
    "github.com/amikos-tech/chroma-go/pkg/embeddings/roboflow"
)

ef, err := roboflow.NewRoboflowEmbeddingFunction(
    roboflow.WithEnvAPIKey(),
)
```

### EmbedContent — text only

```go
// Source: pkg/embeddings/embedding.go ContentEmbeddingFunction interface
// Source: pkg/embeddings/multimodal_compat.go NewTextPart
content := embeddings.Content{
    Parts: []embeddings.Part{
        embeddings.NewTextPart("What is Chroma?"),
    },
}
embedding, err := ef.EmbedContent(ctx, content)
```

### EmbedContents — mixed batch (text + image as separate items)

```go
// Source: pkg/embeddings/multimodal_compat.go NewPartFromSource, NewBinarySourceFromURL
contents := []embeddings.Content{
    {Parts: []embeddings.Part{embeddings.NewTextPart("A dog running on a beach")}},
    {Parts: []embeddings.Part{
        embeddings.NewPartFromSource(
            embeddings.ModalityImage,
            embeddings.NewBinarySourceFromURL("https://example.com/dog.jpg"),
        ),
    }},
}
embeddings, err := ef.EmbedContents(ctx, contents)
```

### Intent on Content

```go
// Source: pkg/embeddings/multimodal.go Intent constants
content := embeddings.Content{
    Parts:  []embeddings.Part{embeddings.NewTextPart("retrieval query")},
    Intent: embeddings.IntentRetrievalQuery,
    // Other neutral intents: IntentRetrievalDocument, IntentClassification,
    // IntentClustering, IntentSemanticSimilarity
}
```

### Dimension on Content

```go
// Source: pkg/embeddings/multimodal.go Content.Dimension field (*int)
dim := 256
content := embeddings.Content{
    Parts:     []embeddings.Part{embeddings.NewTextPart("document")},
    Dimension: &dim,
}
```

### BinarySource variants (for image parts)

```go
// Source: pkg/embeddings/multimodal_compat.go — four constructors
embeddings.NewBinarySourceFromURL("https://example.com/image.jpg")
embeddings.NewBinarySourceFromFile("/path/to/image.png")
embeddings.NewBinarySourceFromBase64("base64data==")
embeddings.NewBinarySourceFromBytes(rawBytes)
```

### Legacy API (unchanged — for comparison table)

```go
// Source: pkg/embeddings/embedding.go EmbeddingFunction interface
embeddings_result, err := ef.EmbedDocuments(ctx, []string{"doc1", "doc2"})
query_embedding, err := ef.EmbedQuery(ctx, "query text")
image_embedding, err := ef.EmbedImage(ctx, embeddings.NewImageInputFromURL("https://..."))
```

---

## State of the Art

| Old Approach | Current Approach | Status | Impact |
|--------------|------------------|--------|--------|
| `multimodal.md` is Python-only stub saying "not available in Go" | Rewrite as Go-native Content API reference page | Phase 5 deliverable | Removes false "not available" information |
| Text-only `EmbedDocuments` / image-only `EmbedImages` as the sole public surface | `EmbedContent` / `EmbedContents` on `ContentEmbeddingFunction` interface | Implemented in Phases 1-4 | New API is available now; old API stays |
| No portable intent vocabulary | 5 neutral `Intent` constants in `pkg/embeddings` | Implemented in Phase 1 | Callers can set intents portably |

**Deprecated/outdated:**
- The "not available in Go" framing in `multimodal.md` — replace entirely.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard test + testify (assert/require) |
| Config file | none — uses `go test` directly |
| Quick run command | `go test ./pkg/embeddings/... -run TestMultimodal\|TestCompatibility\|TestValidate\|TestRegistry\|TestIntent\|TestCapability` |
| Full suite command | `go test ./pkg/embeddings/...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DOCS-01 | Doc page accurately reflects API shapes | manual | n/a — doc review | N/A |
| DOCS-02 | Shared type validation covered | unit | `go test ./pkg/embeddings/... -run TestMultimodalValidation\|TestMultimodalIntent` | ✅ `multimodal_validation_test.go` |
| DOCS-02 | Compatibility adapters covered | unit | `go test ./pkg/embeddings/... -run TestLegacy\|TestCompatibility` | ✅ `capabilities_test.go` |
| DOCS-02 | Registry/config round-trips covered | unit | `go test ./pkg/embeddings/... -run TestBuildContent\|TestRegisterContent` | ✅ `registry_test.go` (minor EmbedContent gap) |
| DOCS-02 | Unsupported-combination failures covered | unit | `go test ./pkg/embeddings/... -run TestValidateContentSupport` | ✅ `content_validate_test.go` |

### Sampling Rate

- **Per task commit:** `go test ./pkg/embeddings/...`
- **Per wave merge:** `go test ./pkg/embeddings/...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] Extend `TestBuildContentFallbackCapabilityAware` in `pkg/embeddings/registry_test.go` to call `EmbedContent` and verify dispatch — closes DOCS-02 registry round-trip gap

*(All test infrastructure, framework, and fixtures already exist — only the one test extension is needed.)*

---

## Open Questions

1. **Cross-link path in embeddings.md**
   - What we know: The file to link to is `docs/go-examples/docs/embeddings/multimodal.md`
   - What's unclear: How the doc site resolves relative vs absolute links from `docs/docs/embeddings.md`
   - Recommendation: Write the link as a relative path and check local doc preview; fall back to an absolute URL to the deployed page if relative links resolve incorrectly.

2. **Mixed-part example provider choice**
   - What we know: Roboflow is the only shipped `ContentEmbeddingFunction`, but its adapter requires one Part per Content
   - What's unclear: Whether the doc example should call out this constraint explicitly or just show the correct pattern silently
   - Recommendation: Show the correct batch pattern (two separate Content items) without calling out the constraint — users discover the error message if they attempt mixed parts, which is acceptable for doc happy-path scope.

---

## Sources

### Primary (HIGH confidence)

- `pkg/embeddings/multimodal.go` — Content, Part, Intent, Modality, SourceKind types; 5 neutral intent constants; IsNeutralIntent
- `pkg/embeddings/embedding.go` — EmbeddingFunction, ContentEmbeddingFunction, MultimodalEmbeddingFunction, IntentMapper interfaces
- `pkg/embeddings/multimodal_validate.go` — ValidationError, ValidationIssue, Validate(), ValidateContents(), ValidateContentSupport()
- `pkg/embeddings/multimodal_compat.go` — AdaptEmbeddingFunctionToContent, AdaptMultimodalEmbeddingFunctionToContent, NewTextPart, NewPartFromSource, NewBinarySourceFrom* constructors
- `pkg/embeddings/capabilities.go` — CapabilityMetadata, RequestOption constants
- `pkg/embeddings/registry.go` — RegisterContent, BuildContent, BuildContentCloseable, RegisterMultimodal
- `pkg/embeddings/roboflow/roboflow.go` — RoboflowEmbeddingFunction interface assertions, Capabilities(), EmbedContent/EmbedContents via adapter
- All six `pkg/embeddings/*_test.go` files — confirmed existing test coverage
- `docs/go-examples/docs/embeddings/multimodal.md` — current page content to replace
- `docs/go-examples/docs/embeddings/embedding-functions.md` — confirmed `{% codetabs %}` template syntax
- `docs/docs/embeddings.md` — confirmed cross-link insertion point, admonition style

### Secondary (MEDIUM confidence)

- `.planning/phases/05-documentation-and-verification/05-CONTEXT.md` — locked decisions that constrain all findings above

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all packages read directly from source
- Architecture patterns: HIGH — page structure from CONTEXT.md decisions; API shapes verified in source
- Pitfalls: HIGH — mixed-part constraint verified in multimodal_compat.go; import path verified in registry.go and roboflow.go
- Test gap analysis: HIGH — all six test files read and mapped to DOCS-02 criteria

**Research date:** 2026-03-20
**Valid until:** 2026-06-20 (stable — no external dependencies, all internal)
