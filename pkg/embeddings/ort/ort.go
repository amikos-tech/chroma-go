package ort

import (
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	defaultef "github.com/amikos-tech/chroma-go/pkg/embeddings/default_ef"
)

// Option configures the ONNX runtime embedding function.
type Option = defaultef.Option

// DefaultEmbeddingFunction is the default local ONNX embedding implementation.
type DefaultEmbeddingFunction = defaultef.DefaultEmbeddingFunction

// NewDefaultEmbeddingFunction creates the default ONNX embedding function.
func NewDefaultEmbeddingFunction(opts ...Option) (*DefaultEmbeddingFunction, func() error, error) {
	return defaultef.NewDefaultEmbeddingFunction(opts...)
}

// NewDefaultEmbeddingFunctionFromConfig creates the default ONNX embedding function from config.
func NewDefaultEmbeddingFunctionFromConfig(cfg embeddings.EmbeddingFunctionConfig) (*DefaultEmbeddingFunction, error) {
	return defaultef.NewDefaultEmbeddingFunctionFromConfig(cfg)
}

// NewOrtEmbeddingFunction is the canonical constructor in this package.
func NewOrtEmbeddingFunction(opts ...Option) (*DefaultEmbeddingFunction, func() error, error) {
	return NewDefaultEmbeddingFunction(opts...)
}

