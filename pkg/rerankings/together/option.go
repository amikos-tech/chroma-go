package together

import (
	"fmt"
	"os"

	"github.com/amikos-tech/chroma-go/pkg/rerankings"
)

type Option func(c *TogetherRerankingFunction) error

func WithAPIKey(apiKey string) Option {
	return func(c *TogetherRerankingFunction) error {
		c.apiKey = apiKey
		return nil
	}
}

func WithEnvAPIKey() Option {
	return func(c *TogetherRerankingFunction) error {
		if os.Getenv("TOGETHER_API_KEY") == "" {
			return fmt.Errorf("TOGETHER_API_KEY not set")
		}
		c.apiKey = os.Getenv("TOGETHER_API_KEY")
		return nil
	}
}

func WithModel(model rerankings.RerankingModel) Option {
	return func(c *TogetherRerankingFunction) error {
		c.defaultModel = model
		return nil
	}
}

func WithRerankingEndpoint(endpoint string) Option {
	return func(c *TogetherRerankingFunction) error {
		c.rerankingEndpoint = endpoint
		return nil
	}
}

func WithTopN(topN int) Option {
	return func(c *TogetherRerankingFunction) error {
		if topN <= 0 {
			return fmt.Errorf("topN must be a positive integer")
		}
		c.topN = &topN
		return nil
	}
}

func WithReturnDocuments(returnDocuments bool) Option {
	return func(c *TogetherRerankingFunction) error {
		c.returnDocuments = &returnDocuments
		return nil
	}
}
