package jina

import (
	"fmt"
	"os"

	"github.com/amikos-tech/chroma-go/types"
)

type Option func(c *JinaRerankingFunction) error

func WithAPIKey(apiKey string) Option {
	return func(c *JinaRerankingFunction) error {
		c.apiKey = apiKey
		return nil
	}
}

func WithEnvAPIKey() Option {
	return func(c *JinaRerankingFunction) error {
		if os.Getenv("JINA_API_KEY") == "" {
			return fmt.Errorf("JINA_API_KEY not set")
		}
		c.apiKey = os.Getenv("JINA_API_KEY")
		return nil
	}
}

func WithModel(model types.RerankingModel) Option {
	return func(c *JinaRerankingFunction) error {
		c.defaultModel = model
		return nil
	}
}

func WithRerankingEndpoint(endpoint string) Option {
	return func(c *JinaRerankingFunction) error {
		c.rerankingEndpoint = endpoint
		return nil
	}
}

func WithTopN(topN int) Option {
	return func(c *JinaRerankingFunction) error {
		if topN <= 0 {
			return fmt.Errorf("topN must be a positive integer")
		}
		c.topN = &topN
		return nil
	}
}

func WithReturnDocuments(returnDocuments bool) Option {
	return func(c *JinaRerankingFunction) error {
		c.returnDocuments = &returnDocuments
		return nil
	}
}
