package mistral

import (
	"net/http"
	"net/url"
	"os"

	"github.com/pkg/errors"
)

type Option func(p *Client) error

// WithDefaultModel sets the default model for the client
func WithDefaultModel(model string) Option {
	return func(p *Client) error {
		if model == "" {
			return errors.Errorf("default model cannot be empty")
		}
		p.DefaultModel = model
		return nil
	}
}

// WithAPIKey sets the API key for the client
func WithAPIKey(apiKey string) Option {
	return func(p *Client) error {
		if apiKey == "" {
			return errors.Errorf("api key cannot be empty")
		}
		p.apiKey = apiKey
		return nil
	}
}

// WithEnvAPIKey sets the API key for the client from the environment variable GOOGLE_API_KEY
func WithEnvAPIKey() Option {
	return func(p *Client) error {
		if apiKey := os.Getenv(APIKeyEnvVar); apiKey != "" {
			p.apiKey = apiKey
			return nil
		}
		return errors.Errorf("%s not set", APIKeyEnvVar)
	}
}

// WithHTTPClient sets the generative AI client for the client
func WithHTTPClient(client *http.Client) Option {
	return func(p *Client) error {
		if client == nil {
			return errors.Errorf("http client cannot be nil")
		}
		p.Client = client
		return nil
	}
}

// WithMaxBatchSize sets the max batch size for the client - this acts as a limit for the number of embeddings that can be sent in a single request
func WithMaxBatchSize(maxBatchSize int) Option {
	return func(p *Client) error {
		if maxBatchSize <= 0 {
			return errors.Errorf("max batch size must be greater than 0")
		}
		p.MaxBatchSize = maxBatchSize
		return nil
	}
}

// WithBaseURL sets the base URL for the client
func WithBaseURL(baseURL string) Option {
	return func(p *Client) error {
		if baseURL == "" {
			return errors.Errorf("base URL cannot be empty")
		}
		var err error
		p.EmbeddingEndpoint, err = url.JoinPath(baseURL, EmbeddingsEndpoint)
		if err != nil {
			return errors.Wrap(err, "failed to parse embedding endpoint")
		}
		return nil
	}
}
