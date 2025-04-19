package ollama

import (
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

type Option func(p *OllamaClient) error

func WithBaseURL(baseURL string) Option {
	return func(p *OllamaClient) error {
		p.BaseURL = baseURL
		return nil
	}
}
func WithModel(model embeddings.EmbeddingModel) Option {
	return func(p *OllamaClient) error {
		p.Model = model
		return nil
	}
}
