package chromacloud

import (
	"net/http"
	"os"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

type Option func(c *Client) error

func WithModel(model embeddings.EmbeddingModel) Option {
	return func(c *Client) error {
		if model == "" {
			return errors.New("model cannot be empty")
		}
		c.Model = model
		return nil
	}
}

func WithTask(task Task) Option {
	return func(c *Client) error {
		c.Task = task
		return nil
	}
}

func WithAPIKey(apiKey string) Option {
	return func(c *Client) error {
		if apiKey == "" {
			return errors.New("API key cannot be empty")
		}
		c.APIKey = apiKey
		return nil
	}
}

func WithEnvAPIKey() Option {
	return func(c *Client) error {
		if apiKey := os.Getenv(APIKeyEnvVar); apiKey != "" {
			c.APIKey = apiKey
			return nil
		}
		return errors.Errorf("%s not set", APIKeyEnvVar)
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) error {
		if client == nil {
			return errors.New("HTTP client cannot be nil")
		}
		c.HTTPClient = client
		return nil
	}
}

func WithBaseURL(url string) Option {
	return func(c *Client) error {
		if url == "" {
			return errors.New("base URL cannot be empty")
		}
		c.BaseURL = url
		return nil
	}
}
