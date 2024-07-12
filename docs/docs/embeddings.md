# Embedding Models

The following embedding wrappers are available:

| Embedding Model                                                                   | Description                                                                                                                                                 |
|-----------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------|
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
    curl http://localhost:11434/api/embeddings -d '{"model": "nomic-embed-text","prompt": "Here is an article about llamas..."}'
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
the [Cloudflare Workers AI docs](https://developers.cloudflare.com/workers-ai/models/#text-embeddings). `@cf/baai/bge-base-en-v1.5`
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

