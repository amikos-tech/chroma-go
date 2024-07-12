package mistral

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
)

type Option func(p *Client) error

// WithDefaultModel sets the default model for the client
func WithDefaultModel(model string) Option {
	return func(p *Client) error {
		p.DefaultModel = model
		return nil
	}
}

// WithAPIKey sets the API key for the client
func WithAPIKey(apiKey string) Option {
	return func(p *Client) error {
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
		return fmt.Errorf(APIKeyEnvVar + " not set")
	}
}

// WithHTTPClient sets the generative AI client for the client
func WithHTTPClient(client *http.Client) Option {
	return func(p *Client) error {
		if client == nil {
			return fmt.Errorf("mistral client is nil")
		}
		p.Client = client
		return nil
	}
}

// WithMaxBatchSize sets the max batch size for the client - this acts as a limit for the number of embeddings that can be sent in a single request
func WithMaxBatchSize(maxBatchSize int) Option {
	return func(p *Client) error {
		if maxBatchSize < 1 {
			return fmt.Errorf("max batch size must be greater than 0")
		}
		p.MaxBatchSize = maxBatchSize
		return nil
	}
}

// WithBaseURL sets the base URL for the client
func WithBaseURL(baseURL string) Option {
	return func(p *Client) error {
		if baseURL == "" {
			return fmt.Errorf("base can't be empty")
		}
		if _, err := url.ParseRequestURI(baseURL); err != nil {
			return fmt.Errorf("invalid basePath URL: %s", err)
		}
		p.BaseURL = baseURL
		return nil
	}
}

// WithTextEmbeddings sets the endpoint to text embeddings
func WithTextEmbeddings() Option {
	return func(p *Client) error {
		p.EmbeddingsEndpointSuffix = TextEmbeddingsEndpoint
		return nil
	}
}
