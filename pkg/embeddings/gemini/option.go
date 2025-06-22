package gemini

import (
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/pkg/errors"

	"github.com/guiperry/chroma-go_cerebras/pkg/embeddings"
)

type Option func(p *Client) error

// WithDefaultModel sets the default model for the client
func WithDefaultModel(model embeddings.EmbeddingModel) Option {
	return func(p *Client) error {
		if model == "" {
			return errors.New("model cannot be empty")
		}
		p.DefaultModel = model
		return nil
	}
}

// WithAPIKey sets the API key for the client
func WithAPIKey(apiKey string) Option {
	return func(p *Client) error {
		if apiKey == "" {
			return errors.New("API key cannot be empty")
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

// WithClient sets the generative AI client for the client
func WithClient(client *genai.Client) Option {
	return func(p *Client) error {
		if client == nil {
			return errors.New("google generative AI client is nil")
		}
		p.Client = client
		return nil
	}
}

// WithMaxBatchSize sets the max batch size for the client - this acts as a limit for the number of embeddings that can be sent in a single request
func WithMaxBatchSize(maxBatchSize int) Option {
	return func(p *Client) error {
		if maxBatchSize < 1 {
			return errors.New("max batch size must be greater than 0")
		}
		p.MaxBatchSize = maxBatchSize
		return nil
	}
}
