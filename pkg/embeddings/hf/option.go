package hf

import (
	"os"

	"github.com/pkg/errors"
)

type Option func(p *HuggingFaceClient) error

func WithBaseURL(baseURL string) Option {
	return func(p *HuggingFaceClient) error {
		if baseURL == "" {
			return errors.New("base URL cannot be empty")
		}
		p.BaseURL = baseURL
		return nil
	}
}

func WithAPIKey(apiKey string) Option {
	return func(p *HuggingFaceClient) error {
		if apiKey == "" {
			return errors.New("API key cannot be empty")
		}
		p.apiKey = apiKey
		return nil
	}
}

func WithEnvAPIKey() Option {
	return func(p *HuggingFaceClient) error {
		if os.Getenv("HF_API_KEY") == "" {
			return errors.New("HF_API_KEY not set")
		}
		p.apiKey = os.Getenv("HF_API_KEY")
		return nil
	}
}

// WithEnvAPIKey sets the API key for the client from a specified environment variable
func WithAPIKeyFromEnvVar(envVar string) Option {
	return func(p *HuggingFaceClient) error {
		if apiKey := os.Getenv(envVar); apiKey != "" {
			p.apiKey = apiKey
			return nil
		}
		return errors.Errorf("%s not set", envVar)
	}
}

func WithModel(model string) Option {
	return func(p *HuggingFaceClient) error {
		if model == "" {
			return errors.New("model cannot be empty")
		}
		p.Model = model
		return nil
	}
}

func WithDefaultHeaders(headers map[string]string) Option {
	return func(p *HuggingFaceClient) error {
		p.DefaultHeaders = headers
		return nil
	}
}

func WithIsHFEIEndpoint() Option {
	return func(p *HuggingFaceClient) error {
		p.IsHFEIEndpoint = true
		return nil
	}
}
