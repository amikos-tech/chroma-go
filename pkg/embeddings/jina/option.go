package jina

import (
	"fmt"
	"github.com/amikos-tech/chroma-go/types"
	"os"
)

type Option func(c *JinaEmbeddingFunction) error

func WithAPIKey(apiKey string) Option {
	return func(c *JinaEmbeddingFunction) error {
		c.apiKey = apiKey
		return nil
	}
}

func WithEnvAPIKey() Option {
	return func(c *JinaEmbeddingFunction) error {
		if os.Getenv("JINA_API_KEY") == "" {
			return fmt.Errorf("JINA_API_KEY not set")
		}
		c.apiKey = os.Getenv("JINA_API_KEY")
		return nil
	}
}

func WithModel(model types.EmbeddingModel) Option {
	return func(c *JinaEmbeddingFunction) error {
		c.defaultModel = model
		return nil
	}
}

func WithEmbeddingEndpoint(endpoint string) Option {
	return func(c *JinaEmbeddingFunction) error {
		c.embeddingEndpoint = endpoint
		return nil
	}
}

// WithNormalized sets the flag to indicate to Jina whether to normalize (L2 norm) the output embeddings or not. Defaults to true
func WithNormalized(normalized bool) Option {
	return func(c *JinaEmbeddingFunction) error {
		c.normalized = normalized
		return nil
	}
}

// WithEmbeddingType sets the type of the embedding to be returned by Jina. The default is float. Right now no other options are supported
func WithEmbeddingType(embeddingType EmbeddingType) Option {
	return func(c *JinaEmbeddingFunction) error {
		c.embeddingType = embeddingType
		return nil
	}
}
