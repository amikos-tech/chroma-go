# Phase 9: Convenience Constructors and Documentation Polish - Research

**Researched:** 2026-03-25
**Domain:** Go API design (functional options, constructor patterns), documentation updates
**Confidence:** HIGH

## Summary

Phase 9 adds syntactic sugar to the existing Content API. The Content, Part, and BinarySource types already exist with full validation logic. The 7 single-modality constructors (`NewTextContent`, `NewImageURL`, `NewImageFile`, `NewVideoURL`, `NewVideoFile`, `NewAudioFile`, `NewPDFFile`) plus the multi-part `NewContent` compositor compose existing helpers (`NewTextPart`, `NewPartFromSource`, `NewBinarySourceFromURL`, `NewBinarySourceFromFile`) and return `Content` structs. The `ContentOption` functional options (`WithIntent`, `WithDimension`, `WithProviderHints`) apply to the returned Content struct before it is returned. No new types, interfaces, or provider logic is needed.

The documentation update rewrites Gemini and VoyageAI multimodal examples to use the shorthand constructors and updates the `docs/docs/embeddings.md` provider multimodal sections and `docs/docs/embeddings/multimodal.md` generic Content API page.

**Primary recommendation:** Implement all 8 constructors + 3 ContentOption functions in a single new file `pkg/embeddings/content_constructors.go`, unit test in `pkg/embeddings/content_constructors_test.go`, then update docs and examples.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Convenience constructors return `Content` (not `Part`), ready to pass directly to `EmbedContent`. Existing `NewTextPart`/`NewPartFromSource` remain for composing mixed-part content.
- **D-02:** Ship 7 single-modality constructors: `NewTextContent(text)`, `NewImageURL(url)`, `NewImageFile(path)`, `NewVideoURL(url)`, `NewVideoFile(path)`, `NewAudioFile(path)`, `NewPDFFile(path)`.
- **D-03:** Add `NewContent(parts []Part, opts ...ContentOption)` for composing multi-part content from Part-level helpers. Uses slice for parts (not variadic) to stay consistent with existing repo patterns -- no marker interfaces.
- **D-04:** No base64 or bytes variants in this phase. Workaround: use the verbose `Content{Parts: []Part{NewPartFromSource(...)}}` pattern. Expand later only if real usage demands it.
- **D-05:** All convenience constructors accept variadic `ContentOption` functional options for Intent, Dimension, and ProviderHints (e.g. `NewImageFile(path, WithIntent(...), WithDimension(...))`).
- **D-06:** `ContentOption` is a parameter type for convenience constructors only. Manual `Content{}` struct literals continue to use direct field assignment -- no `Apply` method added.
- **D-07:** Shorthand-first in provider docs -- lead with convenience constructors as primary examples. Verbose forms referenced via link to the generic Content API page (`docs/docs/embeddings/multimodal.md`).
- **D-08:** Rewrite existing Gemini and VoyageAI multimodal examples (`examples/v2/gemini_multimodal/`, `examples/v2/voyage_multimodal/`) in-place to use convenience constructors.

### Claude's Discretion
- File placement for new constructors (new file vs extending `multimodal_compat.go`)
- Exact `ContentOption` type definition (func type vs interface -- follow whichever is simpler)
- Whether `WithProviderHints` takes `map[string]any` or individual key-value pairs
- Test structure and assertion patterns

### Deferred Ideas (OUT OF SCOPE)
- **Base64/Bytes convenience constructors** -- Add `NewImage{Base64,Bytes}` etc. if real usage demands it.
- **`Content.Apply(opts ...ContentOption)` method** -- Universal option application for manual Content{} construction.
</user_constraints>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library | 1.26.1 | All constructor logic is pure Go | No external deps needed for struct composition |
| `github.com/stretchr/testify` | (existing) | Test assertions | Already used in all embeddings package tests |

### Supporting
No new dependencies are required. All constructors compose existing types and helpers already in `pkg/embeddings/`.

## Architecture Patterns

### Recommended File Structure
```
pkg/embeddings/
  multimodal.go                 # Content, Part, BinarySource types (UNCHANGED)
  multimodal_compat.go          # Part-level helpers, adapters (UNCHANGED)
  multimodal_validate.go        # Validate() methods (UNCHANGED)
  content_constructors.go       # NEW: ContentOption, convenience constructors
  content_constructors_test.go  # NEW: unit tests for constructors
```

**Rationale:** A new file is better than extending `multimodal_compat.go`. The compat file is about adapters bridging legacy interfaces; convenience constructors are about ergonomics for new Content API users. Separate concerns = separate files.

### Pattern 1: ContentOption as func type

**What:** Define `ContentOption` as a simple function type, matching the dominant pattern in the repo.
**When to use:** This is the only pattern needed for this phase.

```go
// ContentOption configures optional fields on a Content item created by
// convenience constructors. It is not used with manual Content{} struct literals.
type ContentOption func(*Content)

// WithIntent sets the provider-neutral intent on the content.
func WithIntent(intent Intent) ContentOption {
    return func(c *Content) {
        c.Intent = intent
    }
}

// WithDimension sets the output dimensionality override on the content.
func WithDimension(dim int) ContentOption {
    return func(c *Content) {
        c.Dimension = &dim
    }
}

// WithProviderHints sets provider-specific hints on the content.
func WithProviderHints(hints map[string]any) ContentOption {
    return func(c *Content) {
        c.ProviderHints = hints
    }
}
```

**Key design decision:** Use `func(*Content)` (no error return) rather than `func(*Content) error`. The options set simple fields (Intent, Dimension, ProviderHints) that cannot fail. Validation happens later in `Content.Validate()` when the content is passed to `EmbedContent`. This matches the simplicity principle and avoids forcing callers to handle errors on what are effectively struct field assignments.

The repo has both patterns: error-returning options (e.g., `ClientOption func(client *BaseAPIClient) error`) and non-error options (e.g., `PageOption func(*Page)`, `HnswOption func(*HnswIndexConfig)`). Since ContentOption sets simple value fields with no I/O or state to validate, the non-error pattern is appropriate.

### Pattern 2: Single-modality constructor shape

**What:** Each single-modality constructor composes existing Part/BinarySource helpers, applies options, and returns Content.

```go
// NewImageURL creates a Content with a single URL-backed image part.
func NewImageURL(url string, opts ...ContentOption) Content {
    c := Content{
        Parts: []Part{NewPartFromSource(ModalityImage, NewBinarySourceFromURL(url))},
    }
    for _, opt := range opts {
        opt(&c)
    }
    return c
}
```

**Why return Content (not *Content):** The existing Content type is used by-value throughout the codebase (e.g., `EmbedContent(ctx, Content)` takes Content by value, not `*Content`). Returning Content by value is consistent. The struct is small (1 slice, 1 string, 1 pointer, 1 map).

### Pattern 3: Multi-part constructor

```go
// NewContent creates a Content from pre-built parts with optional configuration.
func NewContent(parts []Part, opts ...ContentOption) Content {
    c := Content{
        Parts: parts,
    }
    for _, opt := range opts {
        opt(&c)
    }
    return c
}
```

**Why slice not variadic for parts:** D-03 mandates this. It also matches the Content struct itself (`Parts []Part`) and avoids ambiguity with the variadic `opts`.

### Anti-Patterns to Avoid
- **Returning `*Content`:** The whole codebase uses `Content` by value. Do not introduce pointer returns.
- **Adding error returns to constructors:** These constructors set fields; validation is deferred to `Validate()`. Adding error returns would create an inconsistency with the Part-level helpers (`NewTextPart`, `NewPartFromSource`) which also don't return errors.
- **Adding `Apply` method to Content:** D-06 explicitly defers this. Do not add it.
- **Duplicating validation in constructors:** `Validate()` already exists and is called by `EmbedContent`. Don't re-validate in constructors.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| BinarySource creation | Custom URL/File field assignment | `NewBinarySourceFromURL`, `NewBinarySourceFromFile` | Already handle SourceKind assignment correctly |
| Part creation for non-text | Manual Part{Modality, Source} | `NewPartFromSource(modality, source)` | Copies bytes, sets modality consistently |
| Text Part creation | Manual Part{Modality: ModalityText, Text: text} | `NewTextPart(text)` | Single canonical way to create text parts |

**Key insight:** Every convenience constructor is a thin composition of existing helpers. The new code is 5-10 lines per constructor with zero business logic.

## Common Pitfalls

### Pitfall 1: Pointer aliasing in WithDimension
**What goes wrong:** If `WithDimension` captures the pointer to the caller's variable instead of creating a new allocation, multiple Content items could share the same dimension pointer.
**Why it happens:** Closing over a pointer parameter.
**How to avoid:** Take `dim int` (value) and create `&dim` inside the closure. The pattern `c.Dimension = &dim` where `dim` is the parameter of the outer function (captured by closure) correctly creates a new allocation per call.
**Warning signs:** Test that two Contents created with different dimensions don't alias.

### Pitfall 2: Forgetting to update doc examples consistently
**What goes wrong:** Docs show old verbose patterns alongside new shorthand, creating confusion about which is "correct."
**Why it happens:** Partial updates.
**How to avoid:** D-07 is clear: shorthand-first in provider docs. Verbose forms stay in the generic multimodal.md page. Systematically update all four locations: `embeddings.md` (Gemini multimodal section + VoyageAI multimodal section), `examples/v2/gemini_multimodal/main.go`, `examples/v2/voyage_multimodal/main.go`.
**Warning signs:** Grep for `NewPartFromSource` in provider docs -- it should only appear in `multimodal.md` verbose section.

### Pitfall 3: Breaking existing tests/examples
**What goes wrong:** Renaming or removing existing helpers during refactoring.
**Why it happens:** Over-enthusiastic cleanup.
**How to avoid:** D-01 is explicit: existing `NewTextPart`/`NewPartFromSource` remain. Constructors are additive. Run `go test ./pkg/embeddings/ -count=1` and `go build ./examples/...` before considering done.
**Warning signs:** Compilation errors in downstream code.

### Pitfall 4: Inconsistent option naming
**What goes wrong:** Option names that don't follow repo conventions.
**Why it happens:** Not checking existing naming.
**How to avoid:** The repo uses `With{FieldName}` consistently (e.g., `WithDimension`, `WithDefaultModel`, `WithTaskType`). Use `WithIntent`, `WithDimension`, `WithProviderHints`.
**Warning signs:** Linter or code review flags.

## Code Examples

### Complete constructor file structure

```go
package embeddings

// ContentOption configures optional fields on a Content item created by
// convenience constructors.
type ContentOption func(*Content)

// WithIntent sets the provider-neutral intent.
func WithIntent(intent Intent) ContentOption {
    return func(c *Content) {
        c.Intent = intent
    }
}

// WithDimension sets the output dimensionality override.
func WithDimension(dim int) ContentOption {
    return func(c *Content) {
        c.Dimension = &dim
    }
}

// WithProviderHints sets provider-specific hints.
func WithProviderHints(hints map[string]any) ContentOption {
    return func(c *Content) {
        c.ProviderHints = hints
    }
}

func applyContentOptions(c *Content, opts []ContentOption) {
    for _, opt := range opts {
        opt(c)
    }
}

// NewTextContent creates a Content with a single text part.
func NewTextContent(text string, opts ...ContentOption) Content {
    c := Content{Parts: []Part{NewTextPart(text)}}
    applyContentOptions(&c, opts)
    return c
}

// NewImageURL creates a Content with a single URL-backed image part.
func NewImageURL(url string, opts ...ContentOption) Content {
    c := Content{Parts: []Part{NewPartFromSource(ModalityImage, NewBinarySourceFromURL(url))}}
    applyContentOptions(&c, opts)
    return c
}

// NewImageFile creates a Content with a single file-backed image part.
func NewImageFile(path string, opts ...ContentOption) Content {
    c := Content{Parts: []Part{NewPartFromSource(ModalityImage, NewBinarySourceFromFile(path))}}
    applyContentOptions(&c, opts)
    return c
}

// NewVideoURL creates a Content with a single URL-backed video part.
func NewVideoURL(url string, opts ...ContentOption) Content {
    c := Content{Parts: []Part{NewPartFromSource(ModalityVideo, NewBinarySourceFromURL(url))}}
    applyContentOptions(&c, opts)
    return c
}

// NewVideoFile creates a Content with a single file-backed video part.
func NewVideoFile(path string, opts ...ContentOption) Content {
    c := Content{Parts: []Part{NewPartFromSource(ModalityVideo, NewBinarySourceFromFile(path))}}
    applyContentOptions(&c, opts)
    return c
}

// NewAudioFile creates a Content with a single file-backed audio part.
func NewAudioFile(path string, opts ...ContentOption) Content {
    c := Content{Parts: []Part{NewPartFromSource(ModalityAudio, NewBinarySourceFromFile(path))}}
    applyContentOptions(&c, opts)
    return c
}

// NewPDFFile creates a Content with a single file-backed PDF part.
func NewPDFFile(path string, opts ...ContentOption) Content {
    c := Content{Parts: []Part{NewPartFromSource(ModalityPDF, NewBinarySourceFromFile(path))}}
    applyContentOptions(&c, opts)
    return c
}

// NewContent creates a Content from pre-built parts with optional configuration.
func NewContent(parts []Part, opts ...ContentOption) Content {
    c := Content{Parts: parts}
    applyContentOptions(&c, opts)
    return c
}
```

### Example rewrite pattern (Gemini)

Before (verbose):
```go
content := embeddings.Content{
    Parts: []embeddings.Part{
        embeddings.NewTextPart("A lioness hunting at sunset"),
        embeddings.NewPartFromSource(
            embeddings.ModalityImage,
            embeddings.NewBinarySourceFromFile(filepath.Join(testdata, "lioness.png")),
        ),
    },
}
```

After (shorthand for single-modality, NewContent for mixed):
```go
// Single image
imageContent := embeddings.NewImageFile(filepath.Join(testdata, "lioness.png"))

// Mixed text + image (uses NewContent with Part helpers)
mixedContent := embeddings.NewContent([]embeddings.Part{
    embeddings.NewTextPart("A lioness hunting at sunset"),
    embeddings.NewPartFromSource(
        embeddings.ModalityImage,
        embeddings.NewBinarySourceFromFile(filepath.Join(testdata, "lioness.png")),
    ),
})
```

### Doc update pattern for provider sections

```go
// Shorthand: embed a single image
imageContent := embeddings.NewImageFile("/path/to/image.png")
emb, err := ef.EmbedContent(context.Background(), imageContent)

// With intent
queryContent := embeddings.NewTextContent("how do lionesses hunt?",
    embeddings.WithIntent(embeddings.IntentRetrievalQuery),
)

// For mixed-part content and verbose construction, see the
// [Content API reference](embeddings/multimodal.md).
```

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify (existing) |
| Config file | None needed (standard `go test`) |
| Quick run command | `go test ./pkg/embeddings/ -count=1 -run TestContent` |
| Full suite command | `go test ./pkg/embeddings/ -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SC-01 | 7 single-modality constructors return correct Content | unit | `go test ./pkg/embeddings/ -count=1 -run TestNew` | Wave 0 |
| SC-02 | NewContent composes multi-part Content | unit | `go test ./pkg/embeddings/ -count=1 -run TestNewContent` | Wave 0 |
| SC-03 | ContentOption (WithIntent, WithDimension, WithProviderHints) apply | unit | `go test ./pkg/embeddings/ -count=1 -run TestWith` | Wave 0 |
| SC-04 | Existing tests and examples still compile/pass | unit+build | `go test ./pkg/embeddings/ -count=1 && go build ./examples/...` | Existing |
| SC-05 | Validate() works on constructor-built Content | unit | `go test ./pkg/embeddings/ -count=1 -run TestNew.*Validate` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./pkg/embeddings/ -count=1`
- **Per wave merge:** `go test ./pkg/embeddings/ -count=1 && go build ./examples/... && make lint`
- **Phase gate:** Full suite green + `make lint` clean

### Wave 0 Gaps
- [ ] `pkg/embeddings/content_constructors_test.go` -- covers SC-01 through SC-05

## Sources

### Primary (HIGH confidence)
- `pkg/embeddings/multimodal.go` -- Content, Part, BinarySource type definitions (read directly)
- `pkg/embeddings/multimodal_compat.go` -- Existing Part-level helpers: NewTextPart, NewPartFromSource, NewBinarySourceFrom{URL,File,Base64,Bytes} (read directly)
- `pkg/embeddings/multimodal_validate.go` -- Content.Validate(), Part.Validate() (read directly)
- `pkg/embeddings/multimodal_test.go` -- Existing test patterns for Content types (read directly)
- `pkg/api/v2/client.go` -- Functional options pattern reference (`ClientOption func(*BaseAPIClient) error`) (read directly)
- `pkg/api/v2/schema.go` -- Non-error functional options pattern reference (`HnswOption func(*HnswIndexConfig)`) (read directly)
- `examples/v2/gemini_multimodal/main.go` -- Current verbose example, rewrite target (read directly)
- `examples/v2/voyage_multimodal/main.go` -- Current verbose example, rewrite target (read directly)
- `docs/docs/embeddings.md` -- Provider multimodal sections, update target (read directly)
- `docs/docs/embeddings/multimodal.md` -- Generic Content API page, update target (read directly)

## Project Constraints (from CLAUDE.md)

- Use conventional commits
- Always lint before committing (`make lint`)
- Use `testify` for assertions
- Never panic in production code (constructors are simple struct composition -- no panic risk)
- Keep things radically simple for as long as possible

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new deps, pure composition of existing types
- Architecture: HIGH -- file placement, constructor shape, and option pattern all follow existing repo patterns
- Pitfalls: HIGH -- pitfalls are straightforward and well-understood (pointer aliasing, doc consistency)

**Research date:** 2026-03-25
**Valid until:** 2026-04-25 (stable domain -- no external version dependencies)
