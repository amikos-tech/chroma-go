package api

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strconv"
)

type Embedding interface {
	Len() int
	Content() []float32
	Compare(other Embedding, metric DistanceMetricOperator) float32
	FromFloat32(content ...float32) error
	IsDefined() bool
}

type Embeddings interface {
	Content() []Embedding
	FromEmbeddings(embeddings ...Embedding)
}

type Float32Embedding struct {
	ArrayOfFloat32 *[]float32
}

func (e *Float32Embedding) IsDefined() bool {
	return e.ArrayOfFloat32 != nil
}

func (e *Float32Embedding) Content() []float32 {
	return *e.ArrayOfFloat32
}

func (e *Float32Embedding) Len() int {
	if e.ArrayOfFloat32 == nil {
		return 0
	}
	return len(*e.ArrayOfFloat32)
}

func (e *Float32Embedding) Compare(other Embedding, metric DistanceMetricOperator) float32 {
	if e.Len() != other.Len() {
		return -1.0
	}
	return float32(metric.Compare(e.Content(), other.Content()))
}

func (e *Float32Embedding) FromFloat32(content ...float32) error {
	e.ArrayOfFloat32 = &content
	return nil
}

type EmbeddingFunction interface {
	// EmbedDocuments returns a vector for each text.
	EmbedDocuments(ctx context.Context, texts []string) ([]Embedding, error)
	// EmbedQuery embeds a single text.
	EmbedQuery(ctx context.Context, text string) (Embedding, error)
	// EmbedRecords embeds a list of records.
	EmbedRecords(ctx context.Context, records []Record, force bool) error
}

func NewEmbeddingFromFloat32(embedding []float32) Embedding {
	return &Float32Embedding{
		ArrayOfFloat32: &embedding,
	}
}

type ConsistentHashEmbeddingFunction struct{ dim int }

func (e *ConsistentHashEmbeddingFunction) EmbedQuery(_ context.Context, document string) (Embedding, error) {
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

func EmbedRecordsDefaultImpl(e EmbeddingFunction, ctx context.Context, records []Record, force bool) error {
	m := make(map[string]int)
	keys := make([]string, 0)
	for i, r := range records {
		if r.Document().ContentString() == "" && !r.Embedding().IsDefined() {
			return fmt.Errorf("embedding without document")
		}
		if r.Document() != nil && (force || !r.Embedding().IsDefined()) {
			m[r.Document().ContentString()] = i
			keys = append(keys, r.Document().ContentString())
		}
		if r.Document() != nil && r.Embedding().IsDefined() && !force {
			continue
		}
		if r.Document().ContentString() == "" && r.Embedding().IsDefined() {
			continue
		}
	}
	// batch embed
	embeddings, err := e.EmbedDocuments(ctx, keys)
	if err != nil {
		return err
	}
	// update original records
	for i, d := range keys {
		err := records[m[d]].Embedding().FromFloat32(embeddings[i].Content()...) // TODO: this is suboptimal as it copies the data
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *ConsistentHashEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]Embedding, error) {
	var embeddings = make([]Embedding, 0)
	for _, document := range documents {
		embedding, err := e.EmbedQuery(ctx, document)
		if err != nil {
			return nil, err
		}
		embeddings = append(embeddings, embedding)
	}
	return embeddings, nil
}

func (e *ConsistentHashEmbeddingFunction) EmbedRecords(ctx context.Context, records []Record, force bool) error {
	return EmbedRecordsDefaultImpl(e, ctx, records, force)
}
