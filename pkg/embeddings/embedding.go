package embeddings

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type EmbeddingModel string

type Embedding interface {
	Len() int
	ContentAsFloat32() []float32
	ContentAsInt32() []int32
	FromFloat32(content ...float32) error
	Compare(other Embedding, metric DistanceMetricOperator) float32
	IsDefined() bool
}

type KnnVector interface {
	Len() int
	ValuesAsFloat32() []float32
}

// SparseVector represents a sparse embedding vector
type SparseVector struct {
	Indices []int     `json:"indices"`
	Values  []float32 `json:"values"`
}

// NewSparseVector creates a new sparse vector.
// Returns an error if:
//   - indices and values have different lengths
//   - any index is negative
//   - any index is duplicated
//   - any value is NaN or infinite
func NewSparseVector(indices []int, values []float32) (*SparseVector, error) {
	s := &SparseVector{
		Indices: indices,
		Values:  values,
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}

// Len returns the number of non-zero elements
func (s *SparseVector) Len() int {
	return len(s.Values)
}

// ValuesAsFloat32 returns the non-zero values
func (s *SparseVector) ValuesAsFloat32() []float32 {
	return s.Values
}

// MarshalJSON implements JSON marshaling for sparse vectors
func (s *SparseVector) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]interface{}{
		"indices": s.Indices,
		"values":  s.Values,
	})
}

// Validate checks that the sparse vector is valid.
// A valid sparse vector has:
//   - matching lengths for indices and values
//   - all indices are non-negative
//   - no duplicate indices
//   - no NaN or infinite values
func (s *SparseVector) Validate() error {
	if s == nil {
		return errors.New("sparse vector is nil")
	}
	if len(s.Indices) != len(s.Values) {
		return errors.New("indices and values must have the same length")
	}
	seen := make(map[int]struct{})
	for i, idx := range s.Indices {
		if idx < 0 {
			return errors.Errorf("index at position %d is negative: %d", i, idx)
		}
		if _, exists := seen[idx]; exists {
			return errors.Errorf("duplicate index at position %d: %d", i, idx)
		}
		seen[idx] = struct{}{}
	}
	for i, val := range s.Values {
		if math.IsNaN(float64(val)) {
			return errors.Errorf("value at position %d is NaN", i)
		}
		if math.IsInf(float64(val), 0) {
			return errors.Errorf("value at position %d is infinite", i)
		}
	}
	return nil
}

type Embeddings []Embedding

type Float32Embedding struct {
	ArrayOfFloat32 *[]float32
}

func (e *Float32Embedding) IsDefined() bool {
	return e.ArrayOfFloat32 != nil
}

func (e *Float32Embedding) ContentAsFloat32() []float32 {
	return *e.ArrayOfFloat32
}

func (e *Float32Embedding) ContentAsInt32() []int32 {
	return make([]int32, 0)
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
	return float32(metric.Compare(e.ContentAsFloat32(), other.ContentAsFloat32()))
}

func (e *Float32Embedding) FromFloat32(content ...float32) error {
	e.ArrayOfFloat32 = &content
	return nil
}

func (e *Float32Embedding) MarshalJSON() ([]byte, error) {
	if e.ArrayOfFloat32 == nil {
		return []byte("null"), nil
	}
	return json.Marshal(e.ArrayOfFloat32)
}

func (e *Float32Embedding) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &e.ArrayOfFloat32)
	if err != nil {
		return err
	}
	return nil
}

type Int32Embedding struct {
	ArrayOfInt32 *[]int32
}

func NewInt32Embedding(embedding []int32) Embedding {
	return &Int32Embedding{
		ArrayOfInt32: &embedding,
	}
}

func (e *Int32Embedding) FromFloat32(_ ...float32) error {
	return errors.New("cannot convert float32 to int32")
}
func (e *Int32Embedding) IsDefined() bool {
	return e.ArrayOfInt32 != nil
}

func (e *Int32Embedding) ContentAsFloat32() []float32 {
	return make([]float32, 0)
}

func (e *Int32Embedding) ContentAsInt32() []int32 {
	return *e.ArrayOfInt32
}

func (e *Int32Embedding) Len() int {
	return len(*e.ArrayOfInt32)
}

func (e *Int32Embedding) Compare(other Embedding, metric DistanceMetricOperator) float32 {
	if e.Len() != other.Len() {
		return -1.0
	}
	return float32(metric.Compare(e.ContentAsFloat32(), other.ContentAsFloat32()))
}

func (e *Int32Embedding) FromInt32(content ...int32) error {
	e.ArrayOfInt32 = &content
	return nil
}

func (e *Int32Embedding) MarshalJSON() ([]byte, error) {
	if e.ArrayOfInt32 == nil {
		return []byte("null"), nil
	}
	return json.Marshal(e.ArrayOfInt32)
}

func (e *Int32Embedding) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &e.ArrayOfInt32)
	if err != nil {
		return err
	}
	return nil
}

type EmbeddingFunction interface {
	// EmbedDocuments returns a vector for each text.
	EmbedDocuments(ctx context.Context, texts []string) ([]Embedding, error)
	// EmbedQuery embeds a single text.
	EmbedQuery(ctx context.Context, text string) (Embedding, error)
}

type SparseEmbeddingFunction interface {
	// EmbedDocumentsSparse returns a sparse vector for each text.
	EmbedDocumentsSparse(ctx context.Context, texts []string) ([]*SparseVector, error)
	// EmbedQuerySparse embeds a single text as a sparse vector.
	EmbedQuerySparse(ctx context.Context, text string) (*SparseVector, error)
}

func NewEmbeddingFromFloat32(embedding []float32) Embedding {
	return &Float32Embedding{
		ArrayOfFloat32: &embedding,
	}
}

func NewEmbeddingFromInt32(embedding []int32) Embedding {
	emb := make([]float32, len(embedding))
	for i, val := range embedding {
		emb[i] = float32(val)
	}
	return &Float32Embedding{
		ArrayOfFloat32: &emb,
	}
}

func NewEmbeddingFromFloat64(embedding []float64) Embedding {
	emb := make([]float32, len(embedding))
	for i, val := range embedding {
		emb[i] = float32(val)
	}
	return &Float32Embedding{
		ArrayOfFloat32: &emb,
	}
}
func NewEmptyEmbedding() Embedding {
	return &Float32Embedding{
		ArrayOfFloat32: nil,
	}
}
func NewEmptyEmbeddings() []Embedding {
	return make([]Embedding, 0)
}
func NewEmbeddingsFromInterface(lst []interface{}) ([]Embedding, error) {
	var result []Embedding
	for _, embedding := range lst {
		switch expr := embedding.(type) {
		case []interface{}:
			vals := make([]float32, 0)
			for _, c := range expr {
				switch val := c.(type) {
				case json.Number:
					numStr := string(val)
					if strings.Contains(numStr, ".") || strings.Contains(numStr, "e") || strings.Contains(numStr, "E") {
						// Has decimal point or scientific notation - treat as float
						if floatVal, err := val.Float64(); err == nil {
							vals = append(vals, float32(floatVal))
						} else {
							return nil, errors.Errorf("invalid embedding type: %T for %v", val, c)
						}
					} else {
						// No decimal indicators - treat as integer
						if intVal, err := val.Int64(); err == nil {
							vals = append(vals, float32(intVal))
						} else {
							return nil, errors.Errorf("invalid embedding type: %T for %v", val, c)
						}
					}
				case float32:
					vals = append(vals, val)
				case float64:
					vals = append(vals, float32(val))
				default:
					return nil, errors.Errorf("invalid embedding type: %T for %v", val, c)
				}
			}
			emb := NewEmbeddingFromFloat32(vals)
			result = append(result, emb)
		default:
			return nil, errors.Errorf("invalid embedding type: %T for %v", expr, embedding)
		}
	}
	return result, nil
}

func NewEmbeddingsFromFloat32(lst [][]float32) ([]Embedding, error) {
	var result []Embedding
	for _, embedding := range lst {
		emb := NewEmbeddingFromFloat32(embedding)
		result = append(result, emb)
	}
	return result, nil
}

func NewEmbeddingsFromInt32(lst [][]int32) ([]Embedding, error) {
	var result []Embedding
	for _, embedding := range lst {
		emb := NewInt32Embedding(embedding)
		result = append(result, emb)
	}
	return result, nil
}

type ConsistentHashEmbeddingFunction struct{ dim int }

func NewConsistentHashEmbeddingFunction() EmbeddingFunction {
	return &ConsistentHashEmbeddingFunction{dim: 384}
}

func (e *ConsistentHashEmbeddingFunction) EmbedQuery(_ context.Context, document string) (Embedding, error) {
	if document == "" {
		return nil, errors.Errorf("document must not be empty")
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

//
// func EmbedRecordsDefaultImpl(e EmbeddingFunction, ctx context.Context, records []v2.Record, force bool) error {
//	m := make(map[string]int)
//	keys := make([]string, 0)
//	for i, r := range records {
//		if r.Document().ContentString() == "" && !r.Embedding().IsDefined() {
//			return fmt.Errorf("embedding without document")
//		}
//		if r.Document() != nil && (force || !r.Embedding().IsDefined()) {
//			m[r.Document().ContentString()] = i
//			keys = append(keys, r.Document().ContentString())
//		}
//		if r.Document() != nil && r.Embedding().IsDefined() && !force {
//			continue
//		}
//		if r.Document().ContentString() == "" && r.Embedding().IsDefined() {
//			continue
//		}
//	}
//	// batch embed
//	embeddings, err := e.EmbedDocuments(ctx, keys)
//	if err != nil {
//		return err
//	}
//	// update original records
//	for i, d := range keys {
//		err := records[m[d]].Embedding().FromFloat32(embeddings[i].ContentAsFloat32()...) // TODO: this is suboptimal as it copies the data
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}

func (e *ConsistentHashEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]Embedding, error) {
	var embeddings = make([]Embedding, 0)
	for _, document := range documents {
		embedding, err := e.EmbedQuery(ctx, document)
		if err != nil {
			return nil, errors.Wrap(err, "failed to embed document")
		}
		embeddings = append(embeddings, embedding)
	}
	return embeddings, nil
}

// func (e *ConsistentHashEmbeddingFunction) EmbedRecords(ctx context.Context, records []v2.Record, force bool) error {
//	return EmbedRecordsDefaultImpl(e, ctx, records, force)
//}
