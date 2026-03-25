# Multimodal Embeddings — Content API

The Content API lets you embed text, images, audio, video, and PDFs through a single portable interface. Instead of using provider-specific methods, you describe **what** you want to embed and the library handles the **how**.

## Why the Content API?

The legacy `EmbedDocuments`/`EmbedQuery` API works great for text. But when you need to embed an image alongside a description, or a video with context, you need a way to express "these things belong together." That's what the Content API does.

```
Legacy API:  EmbedDocuments(["text1", "text2"])     → []Embedding
Content API: EmbedContent(Content{text + image})    → Embedding
```

Both APIs coexist — use whichever fits. The Content API adds multimodal support without replacing anything.

## Core Concepts

The Content API has four building blocks:

```
Content                          ← One thing to embed
├── Parts[]                      ← The pieces that make it up
│   ├── TextPart("a cat photo")  ← Text piece
│   └── ImagePart(source)        ← Image/video/audio/PDF piece
│       └── BinarySource         ← Where the data comes from (URL, file, bytes)
├── Intent                       ← What you're using the embedding for (optional)
└── Dimension                    ← Output vector size override (optional)
```

### Content

A `Content` is one semantic unit you want to embed — a document, a query, or a media item. It contains one or more `Part`s that the provider combines into a single embedding vector.

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
// A photo with its description → one embedding
content := embeddings.NewContent([]embeddings.Part{
    embeddings.NewTextPart("A lioness hunting at sunset"),
    embeddings.NewPartFromSource(
        embeddings.ModalityImage,
        embeddings.NewBinarySourceFromFile("lioness.png"),
    ),
})
```
{% /codetab %}
{% /codetabs %}

### Part

A `Part` is one piece of content — text, an image, a video clip. Each part has a **modality** that declares what type of content it is:

| Modality | What it represents | Shorthand | Verbose |
|----------|--------------------|-----------|---------|
| `ModalityText` | Plain text | `NewTextContent("...")` | `Content{Parts: []Part{NewTextPart("...")}}` |
| `ModalityImage` | Image (PNG, JPEG, WebP, GIF) | `NewImageURL(url)` / `NewImageFile(path)` | `Content{Parts: []Part{NewPartFromSource(ModalityImage, source)}}` |
| `ModalityVideo` | Video (MP4) | `NewVideoURL(url)` / `NewVideoFile(path)` | `Content{Parts: []Part{NewPartFromSource(ModalityVideo, source)}}` |
| `ModalityAudio` | Audio (MP3, WAV) | `NewAudioFile(path)` | `Content{Parts: []Part{NewPartFromSource(ModalityAudio, source)}}` |
| `ModalityPDF` | PDF document | `NewPDFFile(path)` | `Content{Parts: []Part{NewPartFromSource(ModalityPDF, source)}}` |

Not every provider supports every modality. See [Provider Support](#provider-support) below.

### BinarySource

A `BinarySource` tells the library **where** to find non-text content. You don't construct it directly — use one of the helpers:

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
// From a URL (provider fetches it)
embeddings.NewBinarySourceFromURL("https://example.com/cat.jpg")

// From a local file (library reads and encodes it)
embeddings.NewBinarySourceFromFile("/path/to/photo.png")

// From raw bytes already in memory
embeddings.NewBinarySourceFromBytes(imageBytes)

// From a base64-encoded string
embeddings.NewBinarySourceFromBase64(b64String)
```
{% /codetab %}
{% /codetabs %}

### Intent

An `Intent` tells the provider **why** you're embedding this content. Providers that support intents (like Gemini and VoyageAI) use this to optimize the embedding for your use case.

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
// Embedding a query to search against stored documents
query := embeddings.NewTextContent("how do lionesses hunt?",
    embeddings.WithIntent(embeddings.IntentRetrievalQuery),
)

// Embedding a document to be searched later
doc := embeddings.NewTextContent("Lionesses hunt cooperatively...",
    embeddings.WithIntent(embeddings.IntentRetrievalDocument),
)
```
{% /codetab %}
{% /codetabs %}

**When to use which intent:**

| Intent | Use when... | Example |
|--------|-------------|---------|
| `IntentRetrievalQuery` | Embedding a search query | User types "find sunset photos" |
| `IntentRetrievalDocument` | Embedding content to be searched | Indexing a photo description |
| `IntentClassification` | Categorizing content | Sorting images into categories |
| `IntentClustering` | Grouping similar content | Finding related documents |
| `IntentSemanticSimilarity` | Comparing two items | Checking if two descriptions match |

Intents are optional. If you skip them, the provider uses its default behavior.

!!! note "Not all providers support all intents"

    Gemini supports all five. VoyageAI supports only `IntentRetrievalQuery` and `IntentRetrievalDocument`. Unsupported intents return a clear error — they never silently degrade.

## Convenience Constructors

For single-modality content, use the shorthand constructors instead of building Content structs manually:

{% codetabs group="lang" %}
{% codetab label="Go" %}

| Modality | Shorthand | Equivalent verbose form |
|----------|-----------|-------------------------|
| Text | `NewTextContent("...")` | `Content{Parts: []Part{NewTextPart("...")}}` |
| Image (URL) | `NewImageURL(url)` | `Content{Parts: []Part{NewPartFromSource(ModalityImage, NewBinarySourceFromURL(url))}}` |
| Image (file) | `NewImageFile(path)` | `Content{Parts: []Part{NewPartFromSource(ModalityImage, NewBinarySourceFromFile(path))}}` |
| Video (URL) | `NewVideoURL(url)` | `Content{Parts: []Part{NewPartFromSource(ModalityVideo, NewBinarySourceFromURL(url))}}` |
| Video (file) | `NewVideoFile(path)` | `Content{Parts: []Part{NewPartFromSource(ModalityVideo, NewBinarySourceFromFile(path))}}` |
| Audio (file) | `NewAudioFile(path)` | `Content{Parts: []Part{NewPartFromSource(ModalityAudio, NewBinarySourceFromFile(path))}}` |
| PDF (file) | `NewPDFFile(path)` | `Content{Parts: []Part{NewPartFromSource(ModalityPDF, NewBinarySourceFromFile(path))}}` |

{% /codetab %}
{% /codetabs %}

All constructors accept optional `ContentOption` arguments for intent, dimension, and provider hints:

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
// Embed text for retrieval
query := embeddings.NewTextContent("how do lionesses hunt?",
    embeddings.WithIntent(embeddings.IntentRetrievalQuery),
)

// Embed with custom output dimensions
doc := embeddings.NewTextContent("document text",
    embeddings.WithDimension(256),
)
```
{% /codetab %}
{% /codetabs %}

For mixed-part content, use `NewContent` with Part helpers:

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
content := embeddings.NewContent([]embeddings.Part{
    embeddings.NewTextPart("A lioness hunting at sunset"),
    embeddings.NewPartFromSource(
        embeddings.ModalityImage,
        embeddings.NewBinarySourceFromFile("lioness.png"),
    ),
})
```
{% /codetab %}
{% /codetabs %}

## Common Recipes

### Embed text

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
ef, err := gemini.NewGeminiEmbeddingFunction(gemini.WithEnvAPIKey())
if err != nil {
    log.Fatal(err)
}

content := embeddings.NewTextContent("What is Chroma?")
emb, err := ef.EmbedContent(context.Background(), content)
```
{% /codetab %}
{% /codetabs %}

### Embed an image from a URL

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
content := embeddings.NewImageURL("https://example.com/cat.jpg")
emb, err := ef.EmbedContent(context.Background(), content)
```
{% /codetab %}
{% /codetabs %}

### Embed an image from a local file

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
content := embeddings.NewImageFile("/path/to/photo.png")
emb, err := ef.EmbedContent(context.Background(), content)
```
{% /codetab %}
{% /codetabs %}

### Embed text + image together

When you combine parts, the provider fuses them into a single embedding that captures both the text and visual content:

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
content := embeddings.NewContent([]embeddings.Part{
    embeddings.NewTextPart("A lioness hunting at sunset"),
    embeddings.NewPartFromSource(
        embeddings.ModalityImage,
        embeddings.NewBinarySourceFromFile("lioness.png"),
    ),
})
emb, err := ef.EmbedContent(context.Background(), content)
```
{% /codetab %}
{% /codetabs %}

!!! note "Verbose construction"

    For manual `Content{}` struct literals and advanced Part composition, see the [Convenience Constructors](#convenience-constructors) table above for the verbose equivalents.

### Embed a batch of items

Use `EmbedContents` to embed multiple content items in one call. Each item produces its own embedding:

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
contents := []embeddings.Content{
    embeddings.NewTextContent("The golden hour on the Serengeti"),
    embeddings.NewImageFile("lioness.png"),
    embeddings.NewContent([]embeddings.Part{
        embeddings.NewTextPart("A lioness pouncing on prey"),
        embeddings.NewPartFromSource(
            embeddings.ModalityVideo,
            embeddings.NewBinarySourceFromFile("the_pounce.mp4"),
        ),
    }),
}
results, err := ef.EmbedContents(context.Background(), contents)
// results[0] = text embedding, results[1] = image embedding, results[2] = text+video embedding
```
{% /codetab %}
{% /codetabs %}

### Embed with an intent

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
content := embeddings.NewTextContent("how do lionesses hunt?",
    embeddings.WithIntent(embeddings.IntentRetrievalQuery),
)
emb, err := ef.EmbedContent(context.Background(), content)
```
{% /codetab %}
{% /codetabs %}

## Provider Support

| Provider | Models | Modalities | Mixed Parts | Intents |
|----------|--------|------------|-------------|---------|
| **Gemini** | `gemini-embedding-2-preview` | text, image, audio, video, PDF | yes | all 5 |
| | `gemini-embedding-001` (legacy) | text only | no | all 5 |
| **VoyageAI** | `voyage-multimodal-3.5` | text, image, video | yes | query, document |
| | `voyage-2` (default) | text only | no | query, document |
| **Roboflow** | CLIP | text, image | no (one part per Content) | none |

See the [Embeddings](../embeddings.md) page for provider setup, API keys, and option functions.

## Advanced

### Custom output dimensions

Some providers support truncated embeddings for storage efficiency. Use the `Dimension` field:

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
content := embeddings.NewTextContent("document text",
    embeddings.WithDimension(256),
)
emb, err := ef.EmbedContent(context.Background(), content)
// emb.Len() == 256
```
{% /codetab %}
{% /codetabs %}

### Provider hints

For provider-specific options that don't have a portable equivalent, use `ProviderHints`:

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
content := embeddings.NewTextContent("classify this",
    embeddings.WithProviderHints(map[string]any{
        "task_type": "CLASSIFICATION",  // Gemini-specific
    }),
)
```
{% /codetab %}
{% /codetabs %}

!!! warning

    `ProviderHints` bypass portable intent mapping. They're an escape hatch — prefer `Intent` when a neutral constant fits your use case.

### Capability inspection

Check what a provider supports at runtime:

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
if capAware, ok := ef.(embeddings.CapabilityAware); ok {
    caps := capAware.Capabilities()
    fmt.Println("Modalities:", caps.Modalities)     // e.g. [text image audio video pdf]
    fmt.Println("Mixed parts:", caps.SupportsMixedPart) // true
    fmt.Println("Intents:", caps.Intents)            // e.g. [retrieval_query retrieval_document ...]
}
```
{% /codetab %}
{% /codetabs %}

## Compatibility with Legacy API

Both APIs coexist indefinitely — neither is deprecated.

| Use case | Recommended API |
|----------|----------------|
| Text-only embeddings | `EmbedDocuments` / `EmbedQuery` |
| Mixed media (text + images + video) | `EmbedContent` / `EmbedContents` |
| Portable intents or per-request dimensions | `EmbedContent` / `EmbedContents` |

Existing providers automatically work with the Content API when retrieved through the registry. The registry wraps them with built-in adapters.
