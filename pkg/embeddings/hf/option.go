package hf

import (
	"fmt"
	"os"
)

type Option func(p *HuggingFaceClient) error

func WithBaseURL(baseURL string) Option {
	return func(p *HuggingFaceClient) error {
		p.BaseURL = baseURL

		return nil
	}
}

func WithAPIKey(apiKey string) Option {
	return func(p *HuggingFaceClient) error {
		p.APIKey = apiKey
		return nil
	}
}

func WithEnvAPIKey() Option {
	return func(p *HuggingFaceClient) error {
		if os.Getenv("HF_API_KEY") == "" {
			return fmt.Errorf("HF_API_KEY not set")
		}
		p.APIKey = os.Getenv("HF_API_KEY")
		return nil
	}
}

func WithModel(model string) Option {
	return func(p *HuggingFaceClient) error {
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
