package api

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strconv"
)

type Embedding interface {
	Content() []float32
	Compare(other Embedding, metric DistanceMetricOperator) float32
	FromFloat32(content ...float32) error
}

type Embeddings interface {
	Content() []Embedding
	FromEmbeddings(embeddings ...Embedding)
}

type EmbeddingFunction interface {
	// EmbedDocuments returns a vector for each text.
	EmbedDocuments(ctx context.Context, texts []string) ([]Embedding, error)
	// EmbedQuery embeds a single text.
	EmbedQuery(ctx context.Context, text string) (Embedding, error)
	// EmbedRecords embeds a list of records.
	EmbedRecords(ctx context.Context, records []Record, force bool) error
}

type ConsistentHashEmbeddingFunction struct{ dim int }

func (e *ConsistentHashEmbeddingFunction) EmbedQuery(_ context.Context, document string) (*Embedding, error) {
	if document == "" {
		return nil, fmt.Errorf("document must not be empty")
	}
	hasher := sha256.New()
	hasher.Write([]byte(document))
	hashedText := fmt.Sprintf("%x", hasher.Sum(nil))

	// Pad or truncate
	repeat := e.dim / len(hashedText)
	remainder := e.dim % len(hashedText)
	paddedText := fmt.Sprintf("%s%s",
		fmt.Sprintf("%.*s", repeat*len(hashedText), hashedText), // Repeat pattern
		hashedText[:remainder], // Append any remaining characters
	)

	// Convert to embedding
	var embedding = make([]float32, e.dim)
	for i, char := range paddedText {
		val, _ := strconv.ParseInt(string(char), 16, 64)
		embedding[i] = float32(val) / 15.0
	}

	return NewEmbeddingFromFloat32(embedding), nil
}

func (e *ConsistentHashEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]*Embedding, error) {
	var embeddings = make([]*Embedding, 0)
	for _, document := range documents {
		embedding, err := e.EmbedQuery(ctx, document)
		if err != nil {
			return nil, err
		}
		embeddings = append(embeddings, embedding)
	}
	return embeddings, nil
}

func (e *ConsistentHashEmbeddingFunction) EmbedRecords(ctx context.Context, records []*Record, force bool) error {
	return EmbedRecordsDefaultImpl(e, ctx, records, force)
}
