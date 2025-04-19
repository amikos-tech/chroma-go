package together

import (
	"fmt"
	"net/http"
	"os"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

type Option func(p *TogetherAIClient) error

func WithDefaultModel(model embeddings.EmbeddingModel) Option {
	return func(p *TogetherAIClient) error {
		p.DefaultModel = model
		return nil
	}
}

func WithMaxBatchSize(size int) Option {
	return func(p *TogetherAIClient) error {
		p.MaxBatchSize = size
		return nil
	}
}

func WithDefaultHeaders(headers map[string]string) Option {
	return func(p *TogetherAIClient) error {
		p.DefaultHeaders = headers
		return nil
	}
}

func WithAPIToken(apiToken string) Option {
	return func(p *TogetherAIClient) error {
		p.APIToken = apiToken
		return nil
	}
}

func WithEnvAPIKey() Option {
	return func(p *TogetherAIClient) error {
		if apiToken := os.Getenv("TOGETHER_API_KEY"); apiToken != "" {
			p.APIToken = apiToken
			return nil
		}
		return fmt.Errorf("TOGETHER_API_KEY not set")
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(p *TogetherAIClient) error {
		p.Client = client
		return nil
	}
}
