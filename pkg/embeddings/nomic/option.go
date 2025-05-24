package nomic

import (
	"net/http"
	"net/url"
	"os"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

type Option func(p *Client) error

// WithDefaultModel sets the default model for the client
func WithDefaultModel(model embeddings.EmbeddingModel) Option {
	return func(p *Client) error {
		if model == "" {
			return errors.New("default model cannot be empty")
		}
		p.DefaultModel = model
		return nil
	}
}

// WithAPIKey sets the API key for the client
func WithAPIKey(apiKey string) Option {
	return func(p *Client) error {
		if apiKey == "" {
			return errors.New("api key cannot be empty")
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
			return errors.New("mistral client is nil")
		}
		p.Client = client
		return nil
	}
}

// WithMaxBatchSize sets the max batch size for the client - this acts as a limit for the number of embeddings that can be sent in a single request
func WithMaxBatchSize(maxBatchSize int) Option {
	return func(p *Client) error {
		if maxBatchSize <= 0 {
			return errors.New("max batch size must be greater than 0")
		}
		p.MaxBatchSize = maxBatchSize
		return nil
	}
}

// WithBaseURL sets the base URL for the client
func WithBaseURL(baseURL string) Option {
	return func(p *Client) error {
		if baseURL == "" {
			return errors.New("base URL cannot be empty")
		}
		if _, err := url.ParseRequestURI(baseURL); err != nil {
			return errors.Wrap(err, "invalid basePath URL")
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
