package huggingface

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/amikos-tech/chroma-go/pkg/rerankings"
)

type Option func(c *HFRerankingFunction) error

func WithAPIKey(apiKey string) Option {
	return func(c *HFRerankingFunction) error {
		c.apiKey = apiKey
		return nil
	}
}

func WithEnvAPIKey() Option {
	return func(c *HFRerankingFunction) error {
		if os.Getenv("HF_API_KEY") == "" {
			return fmt.Errorf("HF_API_KEY not set")
		}
		c.apiKey = os.Getenv("HF_API_KEY")
		return nil
	}
}

func WithModel(model rerankings.RerankingModel) Option {
	return func(c *HFRerankingFunction) error {
		c.defaultModel = &model
		return nil
	}
}

func WithRerankingEndpoint(endpoint string) Option {
	return func(c *HFRerankingFunction) error {
		c.rerankingEndpoint = endpoint
		return nil
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *HFRerankingFunction) error {
		if client == nil {
			return errors.New("HTTP client cannot be nil")
		}
		c.httpClient = client
		return nil
	}
}
