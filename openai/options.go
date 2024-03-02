package openai

import "fmt"

// Option is a function type that can be used to modify the client.
type Option func(c *OpenAIClient) error

type EmbeddingModel string

const (
	TextEmbeddingAda002 EmbeddingModel = "text-embedding-ada-002"
	TextEmbedding3Small EmbeddingModel = "text-embedding-3-small"
	TextEmbedding3Large EmbeddingModel = "text-embedding-3-large"
)

// WithOpenAIOrganizationID is an option for setting the OpenAI org id.
func WithOpenAIOrganizationID(openAiAPIKey string) Option {
	return func(c *OpenAIClient) error {
		c.SetOrgID(openAiAPIKey)
		return nil
	}
}

// WithModel is an option for setting the model to use. Must be one of: text-embedding-ada-002, text-embedding-3-small, text-embedding-3-large
func WithModel(model EmbeddingModel) Option {
	return func(c *OpenAIClient) error {
		if string(model) == "" {
			return fmt.Errorf("empty model name")
		}
		if model != TextEmbeddingAda002 && model != TextEmbedding3Small && model != TextEmbedding3Large {
			return fmt.Errorf("invalid model name %s. Must be one of: %v", model, []string{string(TextEmbeddingAda002), string(TextEmbedding3Small), string(TextEmbedding3Large)})
		}
		c.Model = string(model)
		return nil
	}
}

func applyClientOptions(c *OpenAIClient, opts ...Option) error {
	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			return err
		}
	}
	return nil
}
