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
		p.APIKey = apiKey
		return nil
	}
}

func WithEnvAPIKey() Option {
	return func(p *HuggingFaceClient) error {
		if os.Getenv("HF_API_KEY") == "" {
			return errors.New("HF_API_KEY not set")
		}
		p.APIKey = os.Getenv("HF_API_KEY")
		return nil
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
