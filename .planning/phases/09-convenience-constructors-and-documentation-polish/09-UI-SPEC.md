---
phase: 9
slug: convenience-constructors-and-documentation-polish
status: draft
shadcn_initialized: false
preset: none
created: 2026-03-25
---

# Phase 9 — UI Design Contract

> Developer experience and documentation copywriting contract for a Go library phase. This phase has no frontend UI -- the "interface" is Go API surface and Markdown documentation. Standard visual dimensions (spacing, color, typography, registry) are not applicable and marked N/A.

---

## Design System

| Property | Value |
|----------|-------|
| Tool | none |
| Preset | not applicable |
| Component library | not applicable (Go library) |
| Icon library | not applicable |
| Font | not applicable |

**Rationale:** Phase 9 adds Go convenience constructors and updates Markdown documentation. There is no frontend, no CSS, and no component rendering.

---

## Spacing Scale

Not applicable. This phase produces Go source code and Markdown documentation only.

---

## Typography

Not applicable. Documentation uses the existing MkDocs Material theme. No typography decisions are needed.

---

## Color

Not applicable. No frontend surfaces exist.

---

## API Naming Contract

The following naming rules govern all new Go symbols introduced in this phase. These are the "visual contract" equivalents for a library API.

### Constructor Naming

| Constructor | Signature | Returns |
|-------------|-----------|---------|
| `NewTextContent` | `NewTextContent(text string, opts ...ContentOption) Content` | Single text part |
| `NewImageURL` | `NewImageURL(url string, opts ...ContentOption) Content` | Single URL-backed image part |
| `NewImageFile` | `NewImageFile(path string, opts ...ContentOption) Content` | Single file-backed image part |
| `NewVideoURL` | `NewVideoURL(url string, opts ...ContentOption) Content` | Single URL-backed video part |
| `NewVideoFile` | `NewVideoFile(path string, opts ...ContentOption) Content` | Single file-backed video part |
| `NewAudioFile` | `NewAudioFile(path string, opts ...ContentOption) Content` | Single file-backed audio part |
| `NewPDFFile` | `NewPDFFile(path string, opts ...ContentOption) Content` | Single file-backed PDF part |
| `NewContent` | `NewContent(parts []Part, opts ...ContentOption) Content` | Multi-part compositor |

**Source:** CONTEXT.md D-01, D-02, D-03

### Option Naming

| Option | Signature | Sets field |
|--------|-----------|------------|
| `WithIntent` | `WithIntent(intent Intent) ContentOption` | `Content.Intent` |
| `WithDimension` | `WithDimension(dim int) ContentOption` | `Content.Dimension` |
| `WithProviderHints` | `WithProviderHints(hints map[string]any) ContentOption` | `Content.ProviderHints` |

**Source:** CONTEXT.md D-05, RESEARCH.md Pattern 1

### Naming Rules

1. Single-modality constructors use `New{Modality}{SourceKind}` pattern (e.g., `NewImageURL`, `NewVideoFile`).
2. Text is the exception: `NewTextContent` (not `NewTextText` or `NewTextString`).
3. All constructors return `Content` by value (not `*Content`).
4. All constructors accept variadic `...ContentOption` as last parameter.
5. `ContentOption` is `func(*Content)` -- no error return.
6. Option functions use `With{FieldName}` prefix matching existing repo conventions.

---

## Copywriting Contract

### Godoc Comments

Each constructor godoc follows this template: `{FuncName} creates a Content with a single {source-kind}-backed {modality} part.`

| Function | Godoc |
|----------|-------|
| `NewTextContent` | "NewTextContent creates a Content with a single text part." |
| `NewImageURL` | "NewImageURL creates a Content with a single URL-backed image part." |
| `NewImageFile` | "NewImageFile creates a Content with a single file-backed image part." |
| `NewVideoURL` | "NewVideoURL creates a Content with a single URL-backed video part." |
| `NewVideoFile` | "NewVideoFile creates a Content with a single file-backed video part." |
| `NewAudioFile` | "NewAudioFile creates a Content with a single file-backed audio part." |
| `NewPDFFile` | "NewPDFFile creates a Content with a single file-backed PDF part." |
| `NewContent` | "NewContent creates a Content from pre-built parts with optional configuration." |
| `ContentOption` | "ContentOption configures optional fields on a Content item created by convenience constructors." |
| `WithIntent` | "WithIntent sets the provider-neutral intent on the content." |
| `WithDimension` | "WithDimension sets the output dimensionality override on the content." |
| `WithProviderHints` | "WithProviderHints sets provider-specific hints on the content." |

### Documentation Copy

#### Multimodal page (docs/docs/embeddings/multimodal.md)

**New section heading:** "Convenience Constructors"

**Section intro copy:** "For single-modality content, use the shorthand constructors instead of building Content structs manually:"

**Updated Part table row format:**

| Modality | What it represents | Shorthand | Verbose |
|----------|--------------------|-----------|---------|
| `ModalityText` | Plain text | `NewTextContent("...")` | `Content{Parts: []Part{NewTextPart("...")}}` |
| `ModalityImage` | Image (PNG, JPEG, WebP, GIF) | `NewImageURL(url)` / `NewImageFile(path)` | `Content{Parts: []Part{NewPartFromSource(ModalityImage, source)}}` |
| `ModalityVideo` | Video (MP4) | `NewVideoURL(url)` / `NewVideoFile(path)` | `Content{Parts: []Part{NewPartFromSource(ModalityVideo, source)}}` |
| `ModalityAudio` | Audio (MP3, WAV) | `NewAudioFile(path)` | `Content{Parts: []Part{NewPartFromSource(ModalityAudio, source)}}` |
| `ModalityPDF` | PDF document | `NewPDFFile(path)` | `Content{Parts: []Part{NewPartFromSource(ModalityPDF, source)}}` |

**Updated recipe headings and code (shorthand-first per D-07):**

| Recipe | Before (verbose) | After (shorthand) |
|--------|-------------------|--------------------|
| Embed text | `Content{Parts: []Part{NewTextPart("What is Chroma?")}}` | `NewTextContent("What is Chroma?")` |
| Embed image from URL | `Content{Parts: []Part{NewPartFromSource(ModalityImage, NewBinarySourceFromURL(...))}}` | `NewImageURL("https://example.com/cat.jpg")` |
| Embed image from file | `Content{Parts: []Part{NewPartFromSource(ModalityImage, NewBinarySourceFromFile(...))}}` | `NewImageFile("/path/to/photo.png")` |
| Embed with intent | `Content{Parts: []Part{...}, Intent: IntentRetrievalQuery}` | `NewTextContent("how do lionesses hunt?", WithIntent(IntentRetrievalQuery))` |
| Embed with dimension | `Content{Parts: ..., Dimension: &dim}` | `NewTextContent("doc text", WithDimension(256))` |
| Embed text + image | Keep verbose `Content{Parts: ...}` or use `NewContent([]Part{...})` | `NewContent([]Part{NewTextPart("..."), NewPartFromSource(...)})` |

**Verbose forms note (appears after shorthand examples):** "For mixed-part content and manual Content struct construction, see the verbose form examples below."

#### Provider sections (docs/docs/embeddings.md)

**Gemini multimodal subsection update rule:** Replace verbose `Content{Parts: ...}` examples with shorthand constructors. Add a link: "For verbose construction and mixed-part examples, see the [Content API reference](embeddings/multimodal.md)."

**VoyageAI multimodal subsection update rule:** Same pattern as Gemini.

#### Examples

**Gemini example (examples/v2/gemini_multimodal/main.go):**
- Single content: Replace verbose struct literal with `NewContent([]Part{...})` for mixed text+image
- Batch: Replace verbose single-modality items with shorthand constructors (`NewTextContent(...)`, `NewImageFile(...)`)
- Keep mixed text+video item using `NewContent([]Part{...})`

**VoyageAI example (examples/v2/voyage_multimodal/main.go):**
- Same transformation pattern as Gemini

### Example Code Copy

**Single-modality examples shown in provider docs:**

```go
// Embed a single image
imageEmb, err := ef.EmbedContent(ctx, embeddings.NewImageFile("photo.png"))

// Embed text with a retrieval intent
queryEmb, err := ef.EmbedContent(ctx,
    embeddings.NewTextContent("how do lionesses hunt?",
        embeddings.WithIntent(embeddings.IntentRetrievalQuery),
    ),
)
```

**Multi-part example shown in generic multimodal page:**

```go
// Mixed text + image: use NewContent with Part helpers
content := embeddings.NewContent([]embeddings.Part{
    embeddings.NewTextPart("A lioness hunting at sunset"),
    embeddings.NewPartFromSource(
        embeddings.ModalityImage,
        embeddings.NewBinarySourceFromFile("lioness.png"),
    ),
})
emb, err := ef.EmbedContent(ctx, content)
```

**Batch example in docs:**

```go
contents := []embeddings.Content{
    embeddings.NewTextContent("The golden hour on the Serengeti"),
    embeddings.NewImageFile(filepath.Join(testdata, "lioness.png")),
    embeddings.NewContent([]embeddings.Part{
        embeddings.NewTextPart("A lioness pouncing on prey"),
        embeddings.NewPartFromSource(
            embeddings.ModalityVideo,
            embeddings.NewBinarySourceFromFile(filepath.Join(testdata, "the_pounce.mp4")),
        ),
    }),
}
results, err := ef.EmbedContents(ctx, contents)
```

---

## Registry Safety

| Registry | Blocks Used | Safety Gate |
|----------|-------------|-------------|
| not applicable | none | not applicable |

No component registries, third-party blocks, or frontend dependencies are involved in this phase.

---

## Checker Sign-Off

- [ ] Dimension 1 Copywriting: PENDING (godoc templates, doc copy, example code patterns)
- [ ] Dimension 2 Visuals: N/A (Go library, no visual components)
- [ ] Dimension 3 Color: N/A (Go library, no visual components)
- [ ] Dimension 4 Typography: N/A (Go library, no visual components)
- [ ] Dimension 5 Spacing: N/A (Go library, no visual components)
- [ ] Dimension 6 Registry Safety: N/A (no registries)

**Applicable dimensions:** 1 of 6 (Copywriting only)

**Approval:** pending
