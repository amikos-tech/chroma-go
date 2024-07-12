package voyage

import (
	"fmt"
	"net/http"
	"os"
)

type Option func(p *VoyageAIClient) error

func WithDefaultModel(model string) Option {
	return func(p *VoyageAIClient) error {
		p.DefaultModel = model
		return nil
	}
}

func WithMaxBatchSize(size int) Option {
	return func(p *VoyageAIClient) error {
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
		return fmt.Errorf("%s not set", APIKeyEnvVar)
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(p *VoyageAIClient) error {
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
		var defaultEncodingFormat = format
		p.DefaultEncodingFormat = &defaultEncodingFormat
		return nil
	}
}
