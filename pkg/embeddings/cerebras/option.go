package cerebras

import (
	"net/http"
	"net/url"
	"os"

	"github.com/pkg/errors"

	"github.com/guiperry/chroma-go_cerebras/pkg/embeddings"
)

// Default environment variable names
const (
	DefaultAPIKeyEnvVarName  = "CEREBRAS_API_KEY"
	DefaultBaseURLEnvVarName = "CEREBRAS_BASE_URL"
)

// WithBaseURL sets the base URL for the Cerebras API.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) error {
		if baseURL == "" {
			return errors.New("base URL cannot be empty")
		}
		if _, err := url.ParseRequestURI(baseURL); err != nil {
			return errors.Wrap(err, "invalid base URL")
		}
		c.BaseURL = baseURL
		return nil
	}
}

// WithBaseURLFromEnv sets the base URL for the Cerebras API from an environment variable.
func WithBaseURLFromEnv(envVarName string) Option {
	return func(c *Client) error {
		baseURL := os.Getenv(envVarName)
		if baseURL == "" {
			return errors.Errorf("environment variable %s not set or is empty", envVarName)
		}
		if _, err := url.ParseRequestURI(baseURL); err != nil {
			return errors.Wrapf(err, "invalid base URL from environment variable %s", envVarName)
		}
		c.BaseURL = baseURL
		return nil
	}
}

// WithDefaultModel sets the default chat model for generating embeddings.
func WithDefaultModel(model CerebrasModel) Option {
	return func(c *Client) error {
		if model == "" {
			return errors.New("model cannot be empty")
		}
		c.DefaultModel = embeddings.EmbeddingModel(model)
		return nil
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) error {
		if httpClient == nil {
			return errors.New("HTTP client cannot be nil")
		}
		c.HTTPClient = httpClient
		return nil
	}
}

// WithAPIKeyFromEnv sets the API key from an environment variable.
// The `envVarName` parameter specifies which environment variable to read.
func WithAPIKeyFromEnv(envVarName string) Option {
	return func(c *Client) error {
		apiKey := os.Getenv(envVarName)
		if apiKey == "" {
			return errors.Errorf("environment variable %s not set or is empty", envVarName)
		}
		c.APIKey = apiKey
		return nil
	}
}