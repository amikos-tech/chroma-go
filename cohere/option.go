package cohere

import "fmt"

type Option func(p *CohereClient) error

// WithBaseURL sets the base URL for the Cohere API - the default is https://api.cohere.ai
func WithBaseURL(baseURL string) Option {
	return func(p *CohereClient) error {
		p.BaseURL = baseURL
		return nil
	}
}

// WithModel sets the default model for the Cohere API - Available models:
// embed-english-v3.0 1024
// embed-multilingual-v3.0 1024
// embed-english-light-v3.0 384
// embed-multilingual-light-v3.0 384
// embed-english-v2.0 4096 (default)
// embed-english-light-v2.0 1024
// embed-multilingual-v2.0 768
func WithModel(model string) Option {
	return func(p *CohereClient) error {
		p.DefaultModel = model
		return nil
	}
}

// WithTruncateMode sets the default truncate mode for the Cohere API - Available modes:
// NONE
// START
// END (default)
func WithTruncateMode(truncate TruncateMode) Option {
	return func(p *CohereClient) error {
		p.DefaultTruncateMode = truncate
		return nil
	}
}

// WithEmbeddingTypes sets the default embedding types for the Cohere API - Available types:
// float (default)
// int8
// uint8
// binary
// ubinary
// TODO we do not have support for returning multiple embedding types from the EmbeddingFunction, so for float->int8, unit8 are supported and returned in the that order
func WithEmbeddingTypes(embeddingTypes ...EmbeddingType) Option {
	return func(p *CohereClient) error {
		// if embeddingstypes contains binary or ubinary error
		for _, et := range embeddingTypes {
			if et == EmbeddingTypeBinary || et == EmbeddingTypeUBinary {
				return fmt.Errorf("embedding types binary and ubinary are not supported")
			}
		}
		// if embeddingstypes is empty, set to default
		if len(embeddingTypes) == 0 {
			embeddingTypes = []EmbeddingType{EmbeddingTypeFloat32}
		}
		p.DefaultEmbeddingTypes = embeddingTypes
		return nil
	}
}
