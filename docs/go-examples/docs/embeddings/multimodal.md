# Multimodal Embeddings - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/embeddings/multimodal)

## Overview

Multimodal collections can store and query data from multiple modalities (text, images, etc.) in a single embedding space.

> **Note**: Multimodal support (images, data loaders, OpenCLIP embedding function) is not yet available in chroma-go. This page documents the Python API for reference. Track progress at the [chroma-go GitHub repository](https://github.com/amikos-tech/chroma-go).

## Python Examples (For Reference)

### Multi-modal Embedding Functions

```python
from chromadb.utils.embedding_functions import OpenCLIPEmbeddingFunction
embedding_function = OpenCLIPEmbeddingFunction()
```

### Adding Multimodal Data with Data Loaders

```python
import chromadb
from chromadb.utils.data_loaders import ImageLoader
from chromadb.utils.embedding_functions import OpenCLIPEmbeddingFunction

client = chromadb.Client()

data_loader = ImageLoader()
embedding_function = OpenCLIPEmbeddingFunction()

collection = client.create_collection(
    name='multimodal_collection',
    embedding_function=embedding_function,
    data_loader=data_loader
)
```

### Adding Images via URIs

```python
collection.add(
    ids=["id1", "id2"],
    uris=["path/to/file/1", "path/to/file/2"]
)
```

### Querying with Images

```python
results = collection.query(
    query_images=[...]  # A list of numpy arrays representing images
)
```

### Querying with Text

```python
results = collection.query(
    query_texts=["This is a query document", "This is another query document"]
)
```

## Go Workaround

While multimodal is not directly supported, you can use text-based embeddings or pre-compute image embeddings externally:

{% codetabs group="lang" %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	client, err := v2.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// For multimodal-like functionality, pre-compute embeddings externally
	// and add them directly to the collection

	// Create collection without embedding function (manual embeddings)
	collection, err := client.CreateCollection(ctx, "multimodal_workaround",
		v2.WithEmbeddingFunctionCreate(nil),
	)
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	// Pre-computed CLIP embeddings for images (computed externally)
	imageEmbedding1 := []float32{0.1, 0.2, 0.3} // ... actual CLIP embedding
	imageEmbedding2 := []float32{0.4, 0.5, 0.6} // ... actual CLIP embedding

	// Add with pre-computed embeddings and URIs as metadata
	err = collection.Add(ctx,
		v2.WithIDs("img1", "img2"),
		v2.WithEmbeddings(imageEmbedding1, imageEmbedding2),
		v2.WithMetadatas(
			v2.NewDocumentMetadata(
				v2.NewStringAttribute("uri", "/path/to/image1.jpg"),
				v2.NewStringAttribute("type", "image"),
			),
			v2.NewDocumentMetadata(
				v2.NewStringAttribute("uri", "/path/to/image2.jpg"),
				v2.NewStringAttribute("type", "image"),
			),
		),
	)
	if err != nil {
		log.Fatalf("Error adding: %v", err)
	}

	// Query with pre-computed query embedding
	queryEmbedding := []float32{0.15, 0.25, 0.35} // CLIP embedding of query image
	results, err := collection.Query(ctx,
		v2.WithQueryEmbeddings(queryEmbedding),
		v2.WithNResults(5),
	)
	if err != nil {
		log.Fatalf("Error querying: %v", err)
	}

	log.Printf("Results: %v", results.GetIDsGroups())
}
```
{% /codetab %}
{% /codetabs %}

## Feature Status

| Feature | Python | Go |
|---------|--------|-----|
| OpenCLIP Embedding Function | ✓ | Not available |
| ImageLoader Data Loader | ✓ | Not available |
| `query_images` parameter | ✓ | Not available |
| `images` parameter in add | ✓ | Not available |
| `uris` parameter | ✓ | Store as metadata |
| Pre-computed embeddings | ✓ | ✓ |

## Notes

- Multimodal support requires data loaders and multi-modal embedding functions
- As a workaround, compute embeddings externally (using Python or other tools) and add them directly
- Store URIs in metadata for reference to original files
- Text queries work normally with text embedding functions
