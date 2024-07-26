package cohere

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
)

type APIVersion string

type CohereModel string // generic type for Cohere models

func (m CohereModel) String() string {
	return string(m)
}

const (
	APIKeyEnv                    = "COHERE_API_KEY"
	DefaultBaseURL               = "https://api.cohere.ai"
	APIVersionV1      APIVersion = "v1"
	DefaultAPIVersion            = APIVersionV1
	ClientName                   = "chroma-go-client"
)

// CohereClient is a common struct for various Cohere integrations - Embeddings, Rerank etc.
type CohereClient struct {
	BaseURL      string     `validate:"required"`
	APIVersion   APIVersion `validate:"required"`
	apiKey       string     `validate:"required"`
	Client       *http.Client
	DefaultModel CohereModel `validate:"required"`
}

func NewCohereClient(opts ...Option) (*CohereClient, error) {
	client := &CohereClient{
		Client:     &http.Client{},
		BaseURL:    DefaultBaseURL,
		APIVersion: DefaultAPIVersion,
	}

	for _, opt := range opts {
		err := opt(client)
		if err != nil {
			return nil, err
		}
	}
	validate := validator.New(validator.WithRequiredStructEnabled(), validator.WithPrivateFieldValidation())
	err := validate.Struct(client)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (c *CohereClient) GetAPIEndpoint(endpoint string) string {
	return strings.ReplaceAll(fmt.Sprintf("%s/%s/%s", c.BaseURL, c.APIVersion, endpoint), "^[:]//", "/")
}

func (c *CohereClient) GetRequest(ctx context.Context, method string, endpoint string, content string) (*http.Request, error) {
	if _, err := url.ParseRequestURI(endpoint); err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewBufferString(content))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", ClientName)
	httpReq.Header.Set("X-Client-Name", ClientName)
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	return httpReq, nil
}

type Option func(p *CohereClient) error

func NoOp() Option {
	return func(p *CohereClient) error {
		return nil
	}
}

func WithBaseURL(baseURL string) Option {
	return func(p *CohereClient) error {
		p.BaseURL = strings.TrimSuffix(baseURL, "/")
		return nil
	}
}

func WithAPIKey(apiKey string) Option {
	return func(p *CohereClient) error {
		p.apiKey = apiKey
		return nil
	}
}

func WithEnvAPIKey() Option {
	return func(p *CohereClient) error {
		if apiKey := os.Getenv(APIKeyEnv); apiKey != "" {
			p.apiKey = apiKey
			return nil
		}
		return fmt.Errorf(fmt.Sprintf("API key env variable %s not found or does not contain a key.", APIKeyEnv))
	}
}

func WithAPIVersion(version APIVersion) Option {
	return func(p *CohereClient) error {
		if version == "" {
			return fmt.Errorf("API version can't be empty")
		}
		p.APIVersion = version
		return nil
	}
}

// WithHTTPClient sets the HTTP client for the Cohere client
func WithHTTPClient(client *http.Client) Option {
	return func(p *CohereClient) error {
		if client == nil {
			return fmt.Errorf("HTTP client is nil")
		}
		p.Client = client
		return nil
	}
}

// WithDefaultModel sets the default model for the Cohere client
func WithDefaultModel(model CohereModel) Option {
	return func(p *CohereClient) error {
		if model == "" {
			return fmt.Errorf("model can't be empty")
		}
		p.DefaultModel = model
		return nil
	}
}
