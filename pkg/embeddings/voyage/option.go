package voyage

import (
	"net/http"
	"os"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

type Option func(p *VoyageAIClient) error

func WithDefaultModel(model embeddings.EmbeddingModel) Option {
	return func(p *VoyageAIClient) error {
		if model == "" {
			return errors.New("model cannot be empty")
		}
		p.DefaultModel = model
		return nil
	}
}

func WithMaxBatchSize(size int) Option {
	return func(p *VoyageAIClient) error {
		if size <= 0 {
			return errors.New("max batch size must be greater than 0")
		}
		p.MaxBatchSize = size
		return nil
	}
}

func WithDefaultHeaders(headers map[string]string) Option {
	return func(p *VoyageAIClient) error {
		p.DefaultHeaders = headers
		return nil
	}
}

func WithAPIKey(apiToken string) Option {
	return func(p *VoyageAIClient) error {
		if apiToken == "" {
			return errors.New("API key cannot be empty")
		}
		p.APIKey = apiToken
		return nil
	}
}

func WithEnvAPIKey() Option {
	return func(p *VoyageAIClient) error {
		if apiToken := os.Getenv(APIKeyEnvVar); apiToken != "" {
			p.APIKey = apiToken
			return nil
		}
		return errors.Errorf("%s not set", APIKeyEnvVar)
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(p *VoyageAIClient) error {
		if client == nil {
			return errors.New("HTTP client cannot be nil")
		}
		p.Client = client
		return nil
	}
}

func WithTruncation(truncation bool) Option {
	return func(p *VoyageAIClient) error {
		p.DefaultTruncation = &truncation
		return nil
	}
}

func WithEncodingFormat(format EncodingFormat) Option {
	return func(p *VoyageAIClient) error {
		if format == "" {
			return errors.New("encoding format cannot be empty")
		}
		var defaultEncodingFormat = format
		p.DefaultEncodingFormat = &defaultEncodingFormat
		return nil
	}
}
