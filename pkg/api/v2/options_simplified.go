package v2

import (
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// Simplified Option Aliases - These provide cleaner naming without changing the underlying API.
// All aliases maintain full type safety by directly wrapping the original typed options.

// Collection Creation Simplified Options

// WithMetadata sets metadata when creating a collection.
// This replaces WithCollectionMetadataCreate.
func WithMetadata(metadata CollectionMetadata) CreateCollectionOption {
	return WithCollectionMetadataCreate(metadata)
}

// WithCreateIfNotExists enables get-or-create behavior when creating a collection.
// This replaces WithIfNotExistsCreate.
func WithCreateIfNotExists() CreateCollectionOption {
	return WithIfNotExistsCreate()
}

// Query Simplified Options

// WithLimit sets the result limit for Query operations.
// This replaces WithNResults for better naming consistency.
func WithLimit(limit int) CollectionQueryOption {
	return WithNResults(limit)
}

// WithQueryText sets query texts for similarity search.
// Simplified version of WithQueryTexts for single text queries.
func WithQueryText(text string) CollectionQueryOption {
	return WithQueryTexts(text)
}

// WithQueryEmbedding sets a single query embedding for similarity search.
// Simplified version of WithQueryEmbeddings for single embedding queries.
func WithQueryEmbedding(embedding embeddings.Embedding) CollectionQueryOption {
	return WithQueryEmbeddings(embedding)
}
