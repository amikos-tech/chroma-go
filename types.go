package chroma

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/google/uuid" //nolint:gci
	"github.com/oklog/ulid"
	"math/rand"
	"time" //nolint:gci
)

type InvalidEmbeddingValueError struct {
	Value interface{}
}

func (e *InvalidEmbeddingValueError) Error() string {
	return fmt.Sprintf("Embedding can be only int or float32. Actual: %v", e.Value)
}

type InvalidMetadataValueError struct {
	Key   string
	Value interface{}
}

func (e *InvalidMetadataValueError) Error() string {
	return fmt.Sprintf("Expected metadata values are int, float32, bool and string. Invalid metadata value for key %s: %v", e.Key, e.Value)
}

type ErrNoDocumentOrEmbedding struct{}

func (e *ErrNoDocumentOrEmbedding) Error() string {
	return "Document or URI or Embedding must be provided"
}

type IDGenerator interface {
	Generate(document string) string
}

type UUIDGenerator struct{}

func (u *UUIDGenerator) Generate(_ string) string {
	uuidV4 := uuid.New()
	return uuidV4.String()
}

type SHA256Generator struct{}

func (s *SHA256Generator) Generate(document string) string {
	hasher := sha256.New()
	hasher.Write([]byte(document))
	sha256Hash := hex.EncodeToString(hasher.Sum(nil))
	return sha256Hash
}

type ULIDGenerator struct{}

func (u *ULIDGenerator) Generate(_ string) string {
	t := time.Now()
	entropy := rand.New(rand.NewSource(t.UnixNano()))
	docULID := ulid.MustNew(ulid.Timestamp(t), entropy)
	return docULID.String()
}

type Embeddings struct {
	ArrayOfFloat32 *[]float32
	ArrayOfInt32   *[]int32
}

func NewEmbeddings(embeddings []interface{}) (*Embeddings, error) {
	var arrayOfFloat32 []float32
	var arrayOfInt32 []int32
	for _, v := range embeddings {
		switch val := v.(type) {
		case int:
			arrayOfInt32 = append(arrayOfInt32, int32(val))
		case float32:
			arrayOfFloat32 = append(arrayOfFloat32, val)
		default:
			return nil, &InvalidEmbeddingValueError{Value: v}
		}
	}
	return &Embeddings{ArrayOfFloat32: &arrayOfFloat32, ArrayOfInt32: &arrayOfInt32}, nil
}
