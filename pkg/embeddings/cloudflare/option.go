package cloudflare

import (
	"fmt"
	"net/http"
	"os"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

type Option func(p *CloudflareClient) error

func WithGatewayEndpoint(endpoint string) Option {
	return func(p *CloudflareClient) error {
		p.BaseAPI = endpoint
		p.IsGateway = true
		return nil
	}
}

func WithDefaultModel(model embeddings.EmbeddingModel) Option {
	return func(p *CloudflareClient) error {
		p.DefaultModel = model
		return nil
	}
}

func WithMaxBatchSize(size int) Option {
	return func(p *CloudflareClient) error {
		p.MaxBatchSize = size
		return nil
	}
}

func WithDefaultHeaders(headers map[string]string) Option {
	return func(p *CloudflareClient) error {
		p.DefaultHeaders = headers
		return nil
	}
}

func WithAPIToken(apiToken string) Option {
	return func(p *CloudflareClient) error {
		p.APIToken = apiToken
		return nil
	}
}

func WithAccountID(accountID string) Option {
	return func(p *CloudflareClient) error {
		p.AccountID = accountID
		return nil
	}
}

func WithEnvAPIToken() Option {
	return func(p *CloudflareClient) error {
		if apiToken := os.Getenv("CF_API_TOKEN"); apiToken != "" {
			p.APIToken = apiToken
			return nil
		}
		return fmt.Errorf("CF_API_TOKEN not set")
	}
}

func WithEnvAccountID() Option {
	return func(p *CloudflareClient) error {
		if accountID := os.Getenv("CF_ACCOUNT_ID"); accountID != "" {
			p.AccountID = accountID
			return nil
		}
		return fmt.Errorf("CF_ACCOUNT_ID not set")
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(p *CloudflareClient) error {
		p.Client = client
		return nil
	}
}

func WithEnvGatewayEndpoint() Option {
	return func(p *CloudflareClient) error {
		if endpoint := os.Getenv("CF_GATEWAY_ENDPOINT"); endpoint != "" {
			p.BaseAPI = endpoint
			p.IsGateway = true
			return nil
		}
		return fmt.Errorf("CF_GATEWAY_ENDPOINT not set")
	}
}
