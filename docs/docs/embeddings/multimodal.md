# Multimodal Embeddings - Content API

The Content API provides a unified interface for embedding text, images, and other modalities. It introduces `Content` and `Part` as the canonical request types alongside portable `Intent` constants and per-request options like `Dimension`.

## Quick Start

Embed a single piece of text using `EmbedContent`:

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
import (
    "context"

    "github.com/amikos-tech/chroma-go/pkg/embeddings"
)

content := embeddings.Content{
    Parts: []embeddings.Part{
        embeddings.NewTextPart("What is Chroma?"),
    },
}
embedding, err := ef.EmbedContent(ctx, content)
```
{% /codetab %}
{% /codetabs %}

The `ef` variable is any `embeddings.ContentEmbeddingFunction` — for example, a Roboflow embedding function (see the mixed-part section below for construction).

## Mixed-Part Requests

Use `EmbedContents` to embed a batch of content items with different modalities. Because Roboflow processes one item per request, pass text and image as separate `Content` items:

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
import (
    "context"

    "github.com/amikos-tech/chroma-go/pkg/embeddings"
    "github.com/amikos-tech/chroma-go/pkg/embeddings/roboflow"
)

ef, err := roboflow.NewRoboflowEmbeddingFunction(roboflow.WithEnvAPIKey())

contents := []embeddings.Content{
    {Parts: []embeddings.Part{embeddings.NewTextPart("A dog running on a beach")}},
    {Parts: []embeddings.Part{
        embeddings.NewPartFromSource(
            embeddings.ModalityImage,
            embeddings.NewBinarySourceFromURL("https://example.com/dog.jpg"),
        ),
    }},
}
results, err := ef.EmbedContents(ctx, contents)
```
{% /codetab %}
{% /codetabs %}

Other binary source constructors are available for different input types:

- `embeddings.NewBinarySourceFromFile("/path/to/image.png")`
- `embeddings.NewBinarySourceFromBase64("base64data==")`
- `embeddings.NewBinarySourceFromBytes(rawBytes)`

## Portable Intents

Set an `Intent` on a `Content` to communicate the purpose of the request to providers that support it:

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
content := embeddings.Content{
    Parts:  []embeddings.Part{embeddings.NewTextPart("retrieval query text")},
    Intent: embeddings.IntentRetrievalQuery,
}
```
{% /codetab %}
{% /codetabs %}

The five neutral intent constants map to a provider-independent vocabulary:

| Constant | Value |
|----------|-------|
| `IntentRetrievalQuery` | `retrieval_query` |
| `IntentRetrievalDocument` | `retrieval_document` |
| `IntentClassification` | `classification` |
| `IntentClustering` | `clustering` |
| `IntentSemanticSimilarity` | `semantic_similarity` |

## Request Options

Use the `Dimension` field to request a specific output vector size from providers that support truncated embeddings:

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
dim := 256
content := embeddings.Content{
    Parts:     []embeddings.Part{embeddings.NewTextPart("document text")},
    Dimension: &dim,
}
```
{% /codetab %}
{% /codetabs %}

!!! note "Advanced"

    You can pass raw intent strings and provider-specific hints via the `ProviderHints` field. See [godoc](https://pkg.go.dev/github.com/amikos-tech/chroma-go/pkg/embeddings) for details.

## Compatibility with Legacy API

Both the Content API and the legacy `EmbedDocuments` / `EmbedQuery` API coexist and neither is deprecated. Use whichever fits your use case:

| Use Case | Recommended API |
|----------|----------------|
| Text-only embeddings | `EmbedDocuments` / `EmbedQuery` |
| Mixed media (text + images) | Content API (`EmbedContent` / `EmbedContents`) |
| Portable intents or dimensions | Content API (`EmbedContent` / `EmbedContents`) |

The legacy API continues to work exactly as before:

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
embeddings, err := ef.EmbedDocuments(ctx, []string{"text1", "text2"})
queryEmb, err := ef.EmbedQuery(ctx, "query text")
```
{% /codetab %}
{% /codetabs %}

Existing providers automatically work with the Content API when retrieved through the registry (`BuildContent`). The registry wraps them with built-in adapters.
