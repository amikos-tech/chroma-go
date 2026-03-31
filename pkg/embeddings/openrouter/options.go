package openrouter

import (
	"net/url"
	"os"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

const (
	DefaultBaseURL = "https://openrouter.ai/api/v1/"
	APIKeyEnvVar   = "OPENROUTER_API_KEY"
)

type Option func(c *Client) error

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

func WithModel(model string) Option {
	return func(c *Client) error {
		if model == "" {
			return errors.New("model cannot be empty")
		}
		c.Model = model
		return nil
	}
}

func WithAPIKey(apiKey string) Option {
	return func(c *Client) error {
		if apiKey == "" {
			return errors.New("API key cannot be empty")
		}
		c.APIKey = embeddings.NewSecret(apiKey)
		return nil
	}
}

func WithEnvAPIKey() Option {
	return func(c *Client) error {
		if apiKey := os.Getenv(APIKeyEnvVar); apiKey != "" {
			c.APIKey = embeddings.NewSecret(apiKey)
			c.APIKeyEnvVar = APIKeyEnvVar
			return nil
		}
		return errors.Errorf("%s not set", APIKeyEnvVar)
	}
}

func WithAPIKeyFromEnvVar(envVar string) Option {
	return func(c *Client) error {
		if apiKey := os.Getenv(envVar); apiKey != "" {
			c.APIKey = embeddings.NewSecret(apiKey)
			c.APIKeyEnvVar = envVar
			return nil
		}
		return errors.Errorf("%s not set", envVar)
	}
}

func WithDimensions(dimensions int) Option {
	return func(c *Client) error {
		if dimensions <= 0 {
			return errors.Errorf("dimensions must be greater than 0, got %d", dimensions)
		}
		c.Dimensions = &dimensions
		return nil
	}
}

func WithUser(user string) Option {
	return func(c *Client) error {
		c.User = user
		return nil
	}
}

func WithEncodingFormat(format string) Option {
	return func(c *Client) error {
		if format != "float" && format != "base64" {
			return errors.Errorf("encoding_format must be 'float' or 'base64', got %q", format)
		}
		c.EncodingFormat = format
		return nil
	}
}

func WithInputType(inputType string) Option {
	return func(c *Client) error {
		c.InputType = inputType
		return nil
	}
}

func WithProviderPreferences(prefs *ProviderPreferences) Option {
	return func(c *Client) error {
		if prefs == nil {
			return errors.New("provider preferences cannot be nil")
		}
		c.Provider = prefs
		return nil
	}
}

func WithInsecure() Option {
	return func(c *Client) error {
		c.Insecure = true
		return nil
	}
}
