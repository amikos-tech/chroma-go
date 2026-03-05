// Package ort exposes the canonical ONNX runtime embedding function package path.
package ort

import (
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	defaultef "github.com/amikos-tech/chroma-go/pkg/embeddings/default_ef" //nolint:staticcheck
)

// Option configures the ONNX runtime embedding function.
type Option = defaultef.Option

// DefaultEmbeddingFunction is the default local ONNX embedding implementation.
type DefaultEmbeddingFunction = defaultef.DefaultEmbeddingFunction

// NewDefaultEmbeddingFunction creates the default ONNX embedding function.
func NewDefaultEmbeddingFunction(opts ...Option) (*DefaultEmbeddingFunction, func() error, error) {
	return defaultef.NewDefaultEmbeddingFunction(opts...) //nolint:staticcheck
}

// NewDefaultEmbeddingFunctionFromConfig creates the default ONNX embedding function from config.
func NewDefaultEmbeddingFunctionFromConfig(cfg embeddings.EmbeddingFunctionConfig) (*DefaultEmbeddingFunction, error) {
	return defaultef.NewDefaultEmbeddingFunctionFromConfig(cfg) //nolint:staticcheck
}

// NewOrtEmbeddingFunction is an alias for NewDefaultEmbeddingFunction.
func NewOrtEmbeddingFunction(opts ...Option) (*DefaultEmbeddingFunction, func() error, error) {
	return NewDefaultEmbeddingFunction(opts...)
}
