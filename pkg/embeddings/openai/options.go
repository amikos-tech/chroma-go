package openai

import (
	"net/url"

	"github.com/pkg/errors"
)

// Option is a function type that can be used to modify the client.
type Option func(c *OpenAIClient) error

func WithBaseURL(baseURL string) Option {
	return func(p *OpenAIClient) error {
		if baseURL == "" {
			return errors.New("Base URL cannot be empty")
		}
		if _, err := url.ParseRequestURI(baseURL); err != nil {
			return errors.Wrap(err, "invalid base URL")
		}
		p.BaseURL = baseURL
		return nil
	}
}

// WithOpenAIOrganizationID is an option for setting the OpenAI org id.
func WithOpenAIOrganizationID(orgID string) Option {
	return func(c *OpenAIClient) error {
		if orgID == "" {
			return errors.New("OrgID cannot be empty")
		}
		c.OrgID = orgID
		return nil
	}
}

// WithModel is an option for setting the model to use. Must be one of: text-embedding-ada-002, text-embedding-3-small, text-embedding-3-large
func WithModel(model EmbeddingModel) Option {
	return func(c *OpenAIClient) error {
		if string(model) == "" {
			return errors.New("Model cannot be empty")
		}
		if model != TextEmbeddingAda002 && model != TextEmbedding3Small && model != TextEmbedding3Large {
			return errors.Errorf("invalid model name %s. Must be one of: %v", model, []string{string(TextEmbeddingAda002), string(TextEmbedding3Small), string(TextEmbedding3Large)})
		}
		c.Model = string(model)
		return nil
	}
}
func WithDimensions(dimensions int) Option {
	return func(c *OpenAIClient) error {
		if dimensions <= 0 {
			return errors.Errorf("dimensions must be greater than 0, got %d", dimensions)
		}
		c.Dimensions = &dimensions
		return nil
	}
}
