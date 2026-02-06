# Embedding Models

The following embedding wrappers are available:

| Embedding Model                                                                   | Description                                                                                                                                                 |
|-----------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [Default Embeddings](#default-embeddings)                                         | The default Chroma embedding function running `all-MiniLM-L6-v2` on Onnx Runtime                                                                            |
| [OpenAI](#openai)                                                                 | OpenAI embeddings API.<br/>All models are supported - see OpenAI [docs](https://platform.openai.com/docs/guides/embeddings/embedding-models) for more info. |
| [Cohere](#cohere)                                                                 | Cohere embeddings API.<br/>All models are supported - see Cohere [API docs](https://docs.cohere.com/reference/embed) for more info.                         |
| [HuggingFace Inference API](#huggingface-inference-api)                           | HuggingFace Inference API.<br/>All models supported by the API.                                                                                             |
| [HuggingFace Embedding Inference Server](#huggingface-embedding-inference-server) | HuggingFace Embedding Inference Server.<br/>[Models supported](https://github.com/huggingface/text-embeddings-inference) by the inference server.           |
| [Ollama](#ollama)                                                                 | Ollama embeddings API.<br/>All models are supported - see Ollama [models lib](https://ollama.com/library) for more info.                                    |
| [Cloudflare Workers AI](#cloudflare-workers-ai)                                   | Cloudflare Workers AI Embedding.<br/> For more info see [CF API Docs](https://developers.cloudflare.com/workers-ai/models/embedding/).                      |
| [Together AI](#together-ai)                                                       | Together AI Embedding.<br/> For more info see [Together API Docs](https://docs.together.ai/reference/embeddings).                                           |
| [Voyage AI](#voyage-ai)                                                           | Voyage AI Embedding.<br/> For more info see [Together API Docs](https://docs.voyageai.com/reference/embeddings-api).                                        |
| [Google Gemini](#google-gemini)                                                   | Google Gemini Embedding.<br/> For more info see [Gemini Docs](https://ai.google.dev/gemini-api/docs/embeddings).                                            |
| [Mistral AI](#mistral-ai)                                                         | Mistral AI Embedding.<br/> For more info see [Mistral AI API Docs](https://docs.mistral.ai/capabilities/embeddings/).                                       |
| [Nomic AI](#nomic-ai)                                                             | Nomic AI Embedding.<br/> For more info see [Nomic AI API Docs](https://docs.nomic.ai/atlas/models/text-embedding).                                          |
| [Jina AI](#jina-ai)                                                               | Jina AI Embedding.<br/> For more info see [Jina AI API Docs](https://api.jina.ai/redoc#tag/embeddings/operation/create_embedding_v1_embeddings_post).       |
| [Roboflow](#roboflow)                                                             | Roboflow CLIP Embedding (Multimodal: text + images).<br/> For more info see [Roboflow Docs](https://inference.roboflow.com/).                               |
| [Baseten](#baseten)                                                               | Baseten BEI (Baseten Embeddings Inference).<br/> Deploy your own embedding models. See [Baseten Docs](https://docs.baseten.co/engines/bei/overview).        |
| [Amazon Bedrock](#amazon-bedrock)                                                 | Amazon Bedrock Embeddings (Titan models).<br/> For more info see [Bedrock Docs](https://docs.aws.amazon.com/bedrock/latest/userguide/embeddings.html).      |

## Default Embeddings

> Note: Supported from 0.2.0+

The default embedding function uses the `all-MiniLM-L6-v2` model running on Onnx Runtime. The default EF is configured
by default if no EF is provided when creating or getting a collection.

Note: As the EF relies on C bindings to avoid memory leaks make sure to call the close callback, alternatively if you
are passing the EF to a client e.g. when getting or creating a collection you can use the client's close method to
ensure proper resource release.

```go
package main

import (
	"context"
	"fmt"

	defaultef "github.com/amikos-tech/chroma-go/pkg/embeddings/default_ef"
)

func main() {
	ef, closeef, efErr := defaultef.NewDefaultEmbeddingFunction()

	// make sure to call this to ensure proper resource release
	defer func() {
		err := closeef()
		if err != nil {
			fmt.Printf("Error closing default embedding function: %s \n", err)
		}
	}()
	if efErr != nil {
		fmt.Printf("Error creating OpenAI embedding function: %s \n", efErr)
	}
	documents := []string{
		"Document 1 content here",
	}
	resp, reqErr := ef.EmbedDocuments(context.Background(), documents)
	if reqErr != nil {
		fmt.Printf("Error embedding documents: %s \n", reqErr)
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

### ONNX Runtime Configuration

The ONNX Runtime library can be customized using environment variables:

- `CHROMAGO_ONNX_RUNTIME_PATH` - Absolute path to a custom ONNX Runtime library file (e.g., `/usr/local/lib/libonnxruntime.1.23.2.dylib`). When set, skips auto-download.
- `CHROMAGO_ONNX_RUNTIME_VERSION` - Version of ONNX Runtime to download (default: `1.22.0`). Only used when `CHROMAGO_ONNX_RUNTIME_PATH` is not set.

Example:
```bash
# Use a specific version
export CHROMAGO_ONNX_RUNTIME_VERSION=1.23.0

# Or point to a manually downloaded library
export CHROMAGO_ONNX_RUNTIME_PATH=/usr/local/lib/libonnxruntime.1.23.2.dylib
```

## OpenAI

Supported Embedding Function Options:

- `WithModel` - Set the OpenAI model to use. Default is `TextEmbeddingAda002` (`text-embedding-ada-002`).
- `WithBaseURL` - Set the OpenAI base URL. Default is `https://api.openai.com/v1`. This allows you to point the EF to a
  compatible OpenAI API endpoint.
- `WithDimensions` - Set the number of dimensions for the embeddings. Default is `None` which returns the full
  embeddings.

```go
package main

import (
	"context"
	"fmt"
	"os"

	openai "github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
)

func main() {
	ef, efErr := openai.NewOpenAIEmbeddingFunction(os.Getenv("OPENAI_API_KEY"), openai.WithModel(openai.TextEmbedding3Large))
	if efErr != nil {
		fmt.Printf("Error creating OpenAI embedding function: %s \n", efErr)
	}
	documents := []string{
		"Document 1 content here",
	}
	resp, reqErr := ef.EmbedDocuments(context.Background(), documents)
	if reqErr != nil {
		fmt.Printf("Error embedding documents: %s \n", reqErr)
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

## Cohere

```go
package main

import (
	"context"
	"fmt"
	"os"

	cohere "github.com/amikos-tech/chroma-go/pkg/embeddings/cohere"
)

func main() {
	ef := cohere.NewCohereEmbeddingFunction(os.Getenv("COHERE_API_KEY"))
	documents := []string{
		"Document 1 content here",
	}
	resp, reqErr := ef.EmbedDocuments(context.Background(), documents)
	if reqErr != nil {
		fmt.Printf("Error embedding documents: %s \n", reqErr)
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

## HuggingFace Inference API

```go
package main

import (
	"context"
	"fmt"
	"os"

	huggingface "github.com/amikos-tech/chroma-go/pkg/embeddings/hf"
)

func main() {
	ef := huggingface.NewHuggingFaceEmbeddingFunction(os.Getenv("HUGGINGFACE_API_KEY"), "sentence-transformers/all-MiniLM-L6-v2")
	documents := []string{
		"Document 1 content here",
	}
	resp, reqErr := ef.EmbedDocuments(context.Background(), documents)
	if reqErr != nil {
		fmt.Printf("Error embedding documents: %s \n", reqErr)
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

## HuggingFace Embedding Inference Server

The embedding server allows you to run supported model locally on your machine with CPU and GPU inference. For more
information check the [HuggingFace Embedding Inference Server](https://github.com/huggingface/text-embeddings-inference)
repository.

```go
package main

import (
	"context"
	"fmt"

	huggingface "github.com/amikos-tech/chroma-go/hf"
)

func main() {
	ef, err := huggingface.NewHuggingFaceEmbeddingInferenceFunction("http://localhost:8001/embed") //set this to the URL of the HuggingFace Embedding Inference Server
	if err != nil {
		fmt.Printf("Error creating HuggingFace embedding function: %s \n", err)
	}
	documents := []string{
		"Document 1 content here",
	}
	resp, reqErr := ef.EmbedDocuments(context.Background(), documents)
	if reqErr != nil {
		fmt.Printf("Error embedding documents: %s \n", reqErr)
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

## Ollama

!!! note "Assumptions"

    The below example assumes that you have an Ollama server running locally on `http://127.0.0.1:11434`.

Use the following command to start the Ollama server:

```bash
    docker run -d -v ./ollama:/root/.ollama -p 11434:11434 --name ollama ollama/ollama
    docker exec -it ollama ollama run nomic-embed-text # press Ctrl+D to exit after model downloads successfully
    # test it
    curl http://localhost:11434/api/embed -d '{"model": "nomic-embed-text","input": ["Here is an article about llamas..."]}'
 ```

```go
package main

import (
	"context"
	"fmt"
	ollama "github.com/amikos-tech/chroma-go/pkg/embeddings/ollama"
)

func main() {
	documents := []string{
		"Document 1 content here",
		"Document 2 content here",
	}
	// the `/api/embeddings` endpoint is automatically appended to the base URL
	ef, err := ollama.NewOllamaEmbeddingFunction(ollama.WithBaseURL("http://127.0.0.1:11434"), ollama.WithModel("nomic-embed-text"))
	if err != nil {
		fmt.Printf("Error creating Ollama embedding function: %s \n", err)
	}
	resp, err := ef.EmbedDocuments(context.Background(), documents)
	if err != nil {
		fmt.Printf("Error embedding documents: %s \n", err)
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

## Cloudflare Workers AI

You will need to register for a Cloudflare account and create a API Token for Workers AI -
see [docs](https://developers.cloudflare.com/workers-ai/get-started/rest-api/#1-get-an-api-token) for more info.

Models can be found in
the [Cloudflare Workers AI docs](https://developers.cloudflare.com/workers-ai/models/#text-embeddings).
`@cf/baai/bge-base-en-v1.5`
is the default model.

```go
package main

import (
	"context"
	"fmt"
	cf "github.com/amikos-tech/chroma-go/pkg/embeddings/cloudflare"
)

func main() {
	documents := []string{
		"Document 1 content here",
		"Document 2 content here",
	}
	// Make sure that you have the `CF_API_TOKEN` and `CF_ACCOUNT_ID` set in your environment
	ef, err := cf.NewCloudflareEmbeddingFunction(cf.WithEnvAPIToken(), cf.WithEnvAccountID(), cf.WithDefaultModel("@cf/baai/bge-small-en-v1.5"))
	if err != nil {
		fmt.Printf("Error creating Cloudflare embedding function: %s \n", err)
	}
	resp, err := ef.EmbedDocuments(context.Background(), documents)
	if err != nil {
		fmt.Printf("Error embedding documents: %s \n", err)
	}
	fmt.Printf("Embedding response: %v \n", resp)
}

```

## Together AI

To use Together AI embeddings, you will need to register for a Together AI account and create
an [API Key](https://api.together.xyz/settings/api-keys).

Available models can be
in [Together AI docs](https://docs.together.ai/docs/embedding-models). `togethercomputer/m2-bert-80M-8k-retrieval` is
the default model.

```go
package main

import (
	"context"
	"fmt"
	t "github.com/amikos-tech/chroma-go/pkg/embeddings/together"
)

func main() {
	documents := []string{
		"Document 1 content here",
		"Document 2 content here",
	}
	// Make sure that you have the `TOGETHER_API_KEY` set in your environment
	ef, err := t.NewTogetherEmbeddingFunction(t.WithEnvAPIKey(), t.WithDefaultModel("togethercomputer/m2-bert-80M-2k-retrieval"))
	if err != nil {
		fmt.Printf("Error creating Together embedding function: %s \n", err)
	}
	resp, err := ef.EmbedDocuments(context.Background(), documents)
	if err != nil {
		fmt.Printf("Error embedding documents: %s \n", err)
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

## Voyage AI

To use Voyage AI embeddings, you will need to register for a Voyage AI account and create
an [API Key](https://dash.voyageai.com/api-keys).

Available models can be
in [Voyage AI docs](https://docs.voyageai.com/docs/embeddings). `voyage-2` is the default model.

```go
package main

import (
	"context"
	"fmt"
	t "github.com/amikos-tech/chroma-go/pkg/embeddings/voyage"
)

func main() {
	documents := []string{
		"Document 1 content here",
		"Document 2 content here",
	}
	// Make sure that you have the `VOYAGE_API_KEY` set in your environment
	ef, err := t.NewVoyageAIEmbeddingFunction(t.WithEnvAPIKey(), t.WithDefaultModel("voyage-large-2"))
	if err != nil {
		fmt.Printf("Error creating Together embedding function: %s \n", err)
	}
	resp, err := ef.EmbedDocuments(context.Background(), documents)
	if err != nil {
		fmt.Printf("Error embedding documents: %s \n", err)
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

## Google Gemini

To use Google Gemini AI embeddings, you will need to create an [API Key](https://aistudio.google.com/app/apikey).

Available models can be
in [Gemini Models](https://ai.google.dev/gemini-api/docs/models/gemini#text-embedding). `text-embedding-004` is the
default model.

```go
package main

import (
	"context"
	"fmt"
	g "github.com/amikos-tech/chroma-go/pkg/embeddings/gemini"
)

func main() {
	documents := []string{
		"Document 1 content here",
		"Document 2 content here",
	}
	// Make sure that you have the `GEMINI_API_KEY` set in your environment
	ef, err := g.NewGeminiEmbeddingFunction(g.WithEnvAPIKey(), g.WithDefaultModel("text-embedding-004"))
	if err != nil {
		fmt.Printf("Error creating Gemini embedding function: %s \n", err)
	}
	resp, err := ef.EmbedDocuments(context.Background(), documents)
	if err != nil {
		fmt.Printf("Error embedding documents: %s \n", err)
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

## Mistral AI

To use Mistral AI embeddings, you will need to create an [API Key](https://console.mistral.ai/api-keys/).

Currently, (as of July 2024) only `mistral-embed` model is available, which is the default model we use.

```go
package main

import (
	"context"
	"fmt"
	mistral "github.com/amikos-tech/chroma-go/pkg/embeddings/mistral"
)

func main() {
	documents := []string{
		"Document 1 content here",
		"Document 2 content here",
	}
	// Make sure that you have the `MISTRAL_API_KEY` set in your environment
	ef, err := mistral.NewMistralEmbeddingFunction(mistral.WithEnvAPIKey(), mistral.WithDefaultModel("mistral-embed"))
	if err != nil {
		fmt.Printf("Error creating Mistral embedding function: %s \n", err)
	}
	resp, err := ef.EmbedDocuments(context.Background(), documents)
	if err != nil {
		fmt.Printf("Error embedding documents: %s \n", err)
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

## Nomic AI

To use Nomic AI embeddings, you will need to create an [API Key](https://atlas.nomic.ai).

Supported models - https://docs.nomic.ai/atlas/models/text-embedding

```go
package main

import (
	"context"
	"fmt"
	nomic "github.com/amikos-tech/chroma-go/pkg/embeddings/nomic"
)

func main() {
	documents := []string{
		"Document 1 content here",
		"Document 2 content here",
	}
	// Make sure that you have the `NOMIC_API_KEY` set in your environment
	ef, err := nomic.NewNomicEmbeddingFunction(nomic.WithEnvAPIKey(), nomic.WithDefaultModel(nomic.NomicEmbedTextV1))
	if err != nil {
		fmt.Printf("Error creating Nomic embedding function: %s \n", err)
	}
	resp, err := ef.EmbedDocuments(context.Background(), documents)
	if err != nil {
		fmt.Printf("Error embedding documents: %s \n", err)
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

## Jina AI

To use Jina AI embeddings, you will need to get an [API Key](https://jina.ai) (trial API keys are freely available
without any registration, scroll down the page and find the automatically generated API key).

Supported models - https://api.jina.ai/redoc#tag/embeddings/operation/create_embedding_v1_embeddings_post

Supported Embedding Function Options:

- `WithModel` - Set the Jina model to use. Default is `jina-embeddings-v3`.
- `WithTask` - Set the task type (`retrieval.query`, `retrieval.passage`, `classification`, `text-matching`, `separation`).
- `WithNormalized` - Whether to normalize (L2 norm) the output embeddings. Default is `true`.
- `WithLateChunking` - Enable late chunking mode which concatenates all sentences and treats them as a single input for contextual token-level embeddings. Default is `false`.
- `WithEmbeddingEndpoint` - Set a custom API endpoint.

```go
package main

import (
	"context"
	"fmt"
	jina "github.com/amikos-tech/chroma-go/pkg/embeddings/jina"
)

func main() {
	documents := []string{
		"Document 1 content here",
		"Document 2 content here",
	}
	// Make sure that you have the `JINA_API_KEY` set in your environment
	ef, err := jina.NewJinaEmbeddingFunction(
		jina.WithEnvAPIKey(),
		jina.WithTask(jina.TaskTextMatching),
		jina.WithLateChunking(true),
	)
	if err != nil {
		fmt.Printf("Error creating Jina embedding function: %s \n", err)
	}
	resp, err := ef.EmbedDocuments(context.Background(), documents)
	if err != nil {
		fmt.Printf("Error embedding documents: %s \n", err)
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

## Roboflow

Roboflow provides CLIP-based embeddings that support both text and images (multimodal). This is useful for building
applications that need to search across both text and image content. Text and images are mapped to the same embedding
space, enabling cross-modal similarity search (e.g., searching images with text queries).

- [Getting Started](https://inference.roboflow.com/start/overview/)
- [CLIP API Documentation](https://inference.roboflow.com/foundation/clip/)
- [OpenAPI Spec](https://inference.roboflow.com/openapi.json)

To use Roboflow embeddings, you will need to create an [API Key](https://app.roboflow.com/settings/api).

Supported Embedding Function Options:

- `WithAPIKey` - Set the API key directly.
- `WithEnvAPIKey` - Use the `ROBOFLOW_API_KEY` environment variable.
- `WithAPIKeyFromEnvVar` - Use a custom environment variable for the API key.
- `WithBaseURL` - Set a custom base URL (default: `https://infer.roboflow.com`).
- `WithCLIPVersion` - Set the CLIP model version (default: `CLIPVersionViTB16`). See [available versions](#clip-model-versions).
- `WithHTTPClient` - Use a custom HTTP client.
- `WithInsecure` - Allow HTTP connections (for local development only).

### CLIP Model Versions

The following CLIP model versions are available:

| Constant | Value | Description |
|----------|-------|-------------|
| `CLIPVersionViTB16` | `ViT-B-16` | Default. Good balance of speed and accuracy |
| `CLIPVersionViTB32` | `ViT-B-32` | Faster, slightly lower accuracy |
| `CLIPVersionViTL14` | `ViT-L-14` | Higher accuracy, slower |
| `CLIPVersionViTL14336px` | `ViT-L-14-336px` | Higher resolution variant |
| `CLIPVersionRN50` | `RN50` | ResNet-50 based |
| `CLIPVersionRN101` | `RN101` | ResNet-101 based |
| `CLIPVersionRN50x4` | `RN50x4` | Scaled ResNet-50 |
| `CLIPVersionRN50x16` | `RN50x16` | Larger scaled ResNet-50 |
| `CLIPVersionRN50x64` | `RN50x64` | Largest scaled ResNet-50 |

!!! note "Embedding Space Consistency"
    Use the same CLIP version for both text and image embeddings to ensure they share the same embedding space.

### Text Embeddings

```go
package main

import (
	"context"
	"fmt"
	roboflow "github.com/amikos-tech/chroma-go/pkg/embeddings/roboflow"
)

func main() {
	documents := []string{
		"Document 1 content here",
		"Document 2 content here",
	}
	// Make sure that you have the `ROBOFLOW_API_KEY` set in your environment
	ef, err := roboflow.NewRoboflowEmbeddingFunction(
		roboflow.WithEnvAPIKey(),
		roboflow.WithCLIPVersion(roboflow.CLIPVersionViTL14), // optional: use a specific CLIP version
	)
	if err != nil {
		fmt.Printf("Error creating Roboflow embedding function: %s \n", err)
	}
	resp, err := ef.EmbedDocuments(context.Background(), documents)
	if err != nil {
		fmt.Printf("Error embedding documents: %s \n", err)
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

### Image Embeddings

Roboflow supports embedding images from multiple sources: base64-encoded data, URLs, or local file paths.

!!! note "URL Handling"
    For URL inputs, the URL is passed directly to the Roboflow API for fetching. For file inputs, the image is read
    locally and sent as base64.

!!! note "Sequential Processing"
    The Roboflow CLIP API processes one item per request. When embedding multiple documents or images, requests are
    sent sequentially. For large batches, consider the API rate limits and potential latency.

```go
package main

import (
	"context"
	"fmt"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	roboflow "github.com/amikos-tech/chroma-go/pkg/embeddings/roboflow"
)

func main() {
	// Make sure that you have the `ROBOFLOW_API_KEY` set in your environment
	// Use the same CLIP version for both text and images for consistent embeddings
	ef, err := roboflow.NewRoboflowEmbeddingFunction(
		roboflow.WithEnvAPIKey(),
		roboflow.WithCLIPVersion(roboflow.CLIPVersionViTL14),
	)
	if err != nil {
		fmt.Printf("Error creating Roboflow embedding function: %s \n", err)
	}

	// Create image inputs from different sources
	images := []embeddings.ImageInput{
		embeddings.NewImageInputFromFile("/path/to/image.png"),
		embeddings.NewImageInputFromURL("https://example.com/image.jpg"),
		embeddings.NewImageInputFromBase64("base64EncodedImageData..."),
	}

	// Embed multiple images
	resp, err := ef.EmbedImages(context.Background(), images)
	if err != nil {
		fmt.Printf("Error embedding images: %s \n", err)
	}
	fmt.Printf("Embedding response: %v \n", resp)

	// Or embed a single image
	singleResp, err := ef.EmbedImage(context.Background(), embeddings.NewImageInputFromFile("/path/to/image.png"))
	if err != nil {
		fmt.Printf("Error embedding image: %s \n", err)
	}
	fmt.Printf("Single image embedding: %v \n", singleResp)
}
```

## Baseten

Baseten allows you to deploy your own embedding models using BEI (Baseten Embeddings Inference), a high-performance
embedding inference engine. This is useful when you need to run custom or self-hosted embedding models with GPU
acceleration.

- [BEI Overview](https://docs.baseten.co/engines/bei/overview)
- [Supported Models](https://docs.baseten.co/engines/bei/models)
- [API Reference](https://docs.baseten.co/api-reference/embeddings)

To use Baseten embeddings, you will need to:

1. Create a [Baseten account](https://www.baseten.co/)
2. Get an [API Key](https://app.baseten.co/settings/api_keys)
3. Deploy an embedding model (see [Deploying a Model](#deploying-a-model) below)

Supported Embedding Function Options:

- `WithAPIKey` - Set the API key directly.
- `WithEnvAPIKey` - Use the `BASETEN_API_KEY` environment variable.
- `WithAPIKeyFromEnvVar` - Use a custom environment variable for the API key.
- `WithBaseURL` - Set your Baseten deployment URL (**required**).
- `WithModelID` - Set the model identifier (optional, depends on deployment).
- `WithHTTPClient` - Use a custom HTTP client.
- `WithInsecure` - Allow HTTP connections (for local development only).

### Basic Usage

```go
package main

import (
	"context"
	"fmt"
	"os"

	baseten "github.com/amikos-tech/chroma-go/pkg/embeddings/baseten"
)

func main() {
	documents := []string{
		"Document 1 content here",
		"Document 2 content here",
	}

	// Make sure BASETEN_API_KEY is set in your environment
	// Replace the base URL with your actual Baseten deployment URL (without /v1 suffix)
	ef, err := baseten.NewBasetenEmbeddingFunction(
		baseten.WithEnvAPIKey(),
		baseten.WithBaseURL("https://model-xxxxxx.api.baseten.co/environments/production/sync"),
	)
	if err != nil {
		fmt.Printf("Error creating Baseten embedding function: %s \n", err)
		return
	}

	resp, err := ef.EmbedDocuments(context.Background(), documents)
	if err != nil {
		fmt.Printf("Error embedding documents: %s \n", err)
		return
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

### Deploying a Model

Baseten uses [Truss](https://github.com/basetenlabs/truss) to deploy models. Here's how to deploy a lightweight
embedding model:

**1. Install Truss:**

```bash
pip install truss
```

**2. Authenticate with Baseten:**

```bash
truss login
# Enter your BASETEN_API_KEY when prompted
```

**3. Create a deployment config:**

Create a `config.yaml` file (example for `all-MiniLM-L6-v2`):

```yaml
model_name: BEI-all-MiniLM-L6-v2

resources:
  accelerator: H100_40GB
  cpu: "1"
  memory: 10Gi
  use_gpu: true

trt_llm:
  build:
    base_model: encoder_bert  # Use encoder_bert for BERT-like models (MiniLM, BGE, etc.)
    num_builder_gpus: 4       # Required for T4 builds
    checkpoint_repository:
      repo: sentence-transformers/all-MiniLM-L6-v2
      revision: main
      source: HF
    quantization_type: no_quant
  runtime:
    webserver_default_route: /v1/embeddings
```

**4. Deploy:**

```bash
truss push --publish --promote
```

After deployment, you'll receive a model URL like:
```
https://model-xxxxxx.api.baseten.co/environments/production/sync/v1
```

Use this URL **without the `/v1` suffix** as the `BaseURL` in your embedding function configuration:
```
https://model-xxxxxx.api.baseten.co/environments/production/sync
```

The embedding function automatically appends `/v1/embeddings` to the base URL.

!!! note "Pre-built Config"
    A ready-to-use Truss config for `all-MiniLM-L6-v2` is available in the repository at
    `pkg/embeddings/baseten/truss/config.yaml`.

### Popular Models for BEI

| Model | HuggingFace Repo | Use Case |
|-------|------------------|----------|
| all-MiniLM-L6-v2 | `sentence-transformers/all-MiniLM-L6-v2` | Fast, lightweight embeddings |
| BGE Large | `BAAI/bge-large-en-v1.5` | High-quality English embeddings |
| Nomic Embed | `nomic-ai/nomic-embed-text-v1.5` | Long-context embeddings |
| E5 Large | `intfloat/e5-large-v2` | Multilingual embeddings |

For more models and configuration options, see the [BEI documentation](https://docs.baseten.co/engines/bei/overview).

## Amazon Bedrock

Amazon Bedrock provides access to Amazon Titan embedding models via the AWS SDK or Bedrock API keys (bearer tokens).

- [Bedrock Embeddings User Guide](https://docs.aws.amazon.com/bedrock/latest/userguide/embeddings.html)
- [Titan Text Embeddings Models](https://docs.aws.amazon.com/bedrock/latest/userguide/titan-embedding-models.html)
- [Bedrock API Keys](https://docs.aws.amazon.com/bedrock/latest/userguide/api-keys.html)

### Authentication

Bedrock supports two authentication methods:

**Option 1: Bedrock API Key (Bearer Token)** - Recommended for simplicity.

1. Go to **AWS Console** -> **Amazon Bedrock** -> **API keys**
2. Click **Create API key**
3. Choose **Short-term key** (up to 12 hours) or **Long-term key** (1-365 days)
4. Copy the generated key
5. Set it as an environment variable:

```bash
export AWS_BEARER_TOKEN_BEDROCK=ABSK...your-key-here...
```

**Option 2: AWS SDK Credentials** - Uses the standard AWS credential chain (env vars, shared config, IAM roles).

```bash
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
export AWS_REGION=us-east-1
```

### Enable Model Access

Before using any model, you must enable it in your AWS account:

1. Go to **AWS Console** -> **Amazon Bedrock** -> **Model access** (left sidebar)
2. Click **Manage model access**
3. Check **Titan Embeddings G1 - Text** (`amazon.titan-embed-text-v1`) - approval is instant
4. Click **Save changes**

### Supported Models

| Model ID | Dimensions | Description |
|----------|-----------|-------------|
| `amazon.titan-embed-text-v1` | 1536 | Default. General-purpose text embeddings |
| `amazon.titan-embed-text-v2:0` | 256/512/1024 | Configurable dimensions, supports normalization |

### Supported Embedding Function Options

- `WithModel` - Set the Bedrock model ID. Default is `amazon.titan-embed-text-v1`.
- `WithRegion` - Set the AWS region. Default is `us-east-1`.
- `WithProfile` - Set the AWS profile name for shared credentials.
- `WithAWSConfig` - Inject a pre-configured `aws.Config`.
- `WithBedrockClient` - Inject a pre-built Bedrock runtime client (for testing).
- `WithDimensions` - Set output dimensions (Titan v2 only).
- `WithNormalize` - Enable output normalization (Titan v2 only).
- `WithBearerToken` - Set a Bedrock API key (bearer token) directly.
- `WithEnvBearerToken` - Use the `AWS_BEARER_TOKEN_BEDROCK` environment variable.
- `WithBearerTokenFromEnvVar` - Use a custom environment variable for the bearer token.

### Bearer Token Authentication

```go
package main

import (
	"context"
	"fmt"
	bedrock "github.com/amikos-tech/chroma-go/pkg/embeddings/bedrock"
)

func main() {
	documents := []string{
		"Document 1 content here",
		"Document 2 content here",
	}
	// Make sure that you have `AWS_BEARER_TOKEN_BEDROCK` set in your environment
	ef, err := bedrock.NewBedrockEmbeddingFunction(bedrock.WithEnvBearerToken())
	if err != nil {
		fmt.Printf("Error creating Bedrock embedding function: %s \n", err)
		return
	}
	resp, err := ef.EmbedDocuments(context.Background(), documents)
	if err != nil {
		fmt.Printf("Error embedding documents: %s \n", err)
		return
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```

### AWS SDK Authentication

```go
package main

import (
	"context"
	"fmt"
	bedrock "github.com/amikos-tech/chroma-go/pkg/embeddings/bedrock"
)

func main() {
	documents := []string{
		"Document 1 content here",
		"Document 2 content here",
	}
	// Uses the AWS default credential chain (env vars, shared config, IAM roles)
	// Make sure AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, and AWS_REGION are set
	ef, err := bedrock.NewBedrockEmbeddingFunction(
		bedrock.WithRegion("us-east-1"),
		bedrock.WithModel("amazon.titan-embed-text-v2:0"),
		bedrock.WithDimensions(512),
		bedrock.WithNormalize(true),
	)
	if err != nil {
		fmt.Printf("Error creating Bedrock embedding function: %s \n", err)
		return
	}
	resp, err := ef.EmbedDocuments(context.Background(), documents)
	if err != nil {
		fmt.Printf("Error embedding documents: %s \n", err)
		return
	}
	fmt.Printf("Embedding response: %v \n", resp)
}
```
