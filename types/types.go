package types

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	time "time"

	"github.com/google/uuid"
	"github.com/oklog/ulid"

	openapi "github.com/amikos-tech/chroma-go/swagger"
)

type DistanceFunction string
type QueryEnum string

const (
	L2                 DistanceFunction = "l2"
	COSINE             DistanceFunction = "cosine"
	IP                 DistanceFunction = "ip"
	DefaultTenant                       = "default_tenant"
	DefaultDatabase                     = "default_database"
	IDocuments         QueryEnum        = "documents"
	IEmbeddings        QueryEnum        = "embeddings"
	IMetadatas         QueryEnum        = "metadatas"
	IDistances         QueryEnum        = "distances"
	HNSWSpace                           = "hnsw:space"
	HNSWConstructionEF                  = "hnsw:construction_ef"
	HNSWBatchSize                       = "hnsw:batch_size"
	HNSWSyncThreshold                   = "hnsw:sync_threshold"
	HNSWM                               = "hnsw:M"
	HNSWSearchEF                        = "hnsw:search_ef"
	HNSWNumThreads                      = "hnsw:num_threads"
	HNSWResizeFactor                    = "hnsw:resize_factor"
	DefaultTimeout                      = 30 * time.Second
)

type InvalidMetadataValueError struct {
	Key   string
	Value interface{}
}

func (e *InvalidMetadataValueError) Error() string {
	return fmt.Sprintf("Invalid metadata value type for key %s: %T", e.Key, e.Value)
}

type InvalidWhereValueError struct {
	Key   string
	Value interface{}
}

func (e *InvalidWhereValueError) Error() string {
	return fmt.Sprintf("Invalid value for where clause for key %s: %v. Allowed values are string, int, float, bool", e.Key, e.Value)
}

type InvalidWhereDocumentValueError struct {
	Value interface{}
}

func (e *InvalidWhereDocumentValueError) Error() string {
	return fmt.Sprintf("Invalid value for where document clause for value %v. Allowed values are string", e.Value)
}

type InvalidEmbeddingValueError struct {
	Value interface{}
}

func (e *InvalidEmbeddingValueError) Error() string {
	return fmt.Sprintf("Embedding can be only int or float32. Actual: %v", e.Value)
}

type Embedding struct {
	ArrayOfFloat32 *[]float32
	ArrayOfInt32   *[]int32
}

func NewEmbeddings(embeddings []interface{}) (*Embedding, error) {
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
	return &Embedding{ArrayOfFloat32: &arrayOfFloat32, ArrayOfInt32: &arrayOfInt32}, nil
}

func (e *Embedding) GetFloat32() *[]float32 {
	return e.ArrayOfFloat32
}

func (e *Embedding) GetInt32() *[]int32 {
	return e.ArrayOfInt32
}

func (e *Embedding) IsDefined() bool {
	return e.ArrayOfFloat32 != nil || e.ArrayOfInt32 != nil
}

func NewEmbeddingFromFloat32(embedding []float32) *Embedding {
	return &Embedding{
		ArrayOfFloat32: &embedding,
		ArrayOfInt32:   nil,
	}
}

func NewEmbeddingsFromFloat32(embeddings [][]float32) []*Embedding {
	var embeddingsArray []*Embedding
	for _, embedding := range embeddings {
		embeddingsArray = append(embeddingsArray, NewEmbeddingFromFloat32(embedding))
	}
	return embeddingsArray
}

func NewEmbeddingFromAPI(apiEmbedding openapi.EmbeddingsInner) *Embedding {
	return &Embedding{
		ArrayOfFloat32: apiEmbedding.ArrayOfFloat32,
		ArrayOfInt32:   apiEmbedding.ArrayOfInt32,
	}
}

func (e *Embedding) ToAPI() openapi.EmbeddingsInner {
	return openapi.EmbeddingsInner{
		ArrayOfFloat32: e.ArrayOfFloat32,
		ArrayOfInt32:   e.ArrayOfInt32,
	}
}

func ToAPIEmbeddings(embeddings []*Embedding) []openapi.EmbeddingsInner {
	var apiEmbeddings []openapi.EmbeddingsInner
	for _, embedding := range embeddings {
		apiEmbeddings = append(apiEmbeddings, embedding.ToAPI())
	}
	return apiEmbeddings
}

type EmbeddingFunction interface {
	// EmbedDocuments returns a vector for each text.
	EmbedDocuments(ctx context.Context, texts []string) ([]*Embedding, error)
	// EmbedQuery embeds a single text.
	EmbedQuery(ctx context.Context, text string) (*Embedding, error)
	EmbedRecords(ctx context.Context, records []Record, force bool) error
}

func EmbedRecordsDefaultImpl(e EmbeddingFunction, ctx context.Context, records []Record, force bool) error {
	m := make(map[string]int)
	keys := make([]string, 0)
	for i, r := range records {
		if r.Document == "" && !r.Embedding.IsDefined() {
			return fmt.Errorf("embedding without document")
		}
		if r.Document != "" && (force || !r.Embedding.IsDefined()) {
			m[r.Document] = i
			keys = append(keys, r.Document)
		}
		if r.Document != "" && r.Embedding.IsDefined() && !force {
			continue
		}
		if r.Document == "" && r.Embedding.IsDefined() {
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
		records[m[d]].Embedding = *embeddings[i]
	}
	return nil
}

func F32ToInterface(f []float32) []interface{} {
	i := make([]interface{}, len(f))
	for index, value := range f {
		i[index] = value
	}
	return i
}

type EmbeddableContext struct {
	Documents []string
}
type Embeddable func(document string) ([]float32, error)

func (e *EmbeddableContext) Apply(ctx context.Context, embeddingFunction EmbeddingFunction) ([]*Embedding, error) {
	return embeddingFunction.EmbedDocuments(ctx, e.Documents)
}

type IDGenerator interface {
	Generate(document string) string
}

type UUIDGenerator struct{}

func (u *UUIDGenerator) Generate(_ string) string {
	uuidV4 := uuid.New()
	return uuidV4.String()
}

func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

type SHA256Generator struct{}

func (s *SHA256Generator) Generate(document string) string {
	hasher := sha256.New()
	hasher.Write([]byte(document))
	sha256Hash := hex.EncodeToString(hasher.Sum(nil))
	return sha256Hash
}

func NewSHA256Generator() *SHA256Generator {
	return &SHA256Generator{}
}

type ULIDGenerator struct{}

func (u *ULIDGenerator) Generate(_ string) string {
	t := time.Now()
	entropy := rand.New(rand.NewSource(t.UnixNano()))
	docULID := ulid.MustNew(ulid.Timestamp(t), entropy)
	return docULID.String()
}

func NewULIDGenerator() *ULIDGenerator {
	return &ULIDGenerator{}
}

type ConsistentHashEmbeddingFunction struct{}

func (e *ConsistentHashEmbeddingFunction) EmbedQuery(_ context.Context, document string) (*Embedding, error) {
	hasher := sha256.New()
	hasher.Write([]byte(document))
	hashBytes := hasher.Sum(nil) // Get the SHA-256 hash as a byte slice

	// Interpret groups of bytes as float32 values
	floatArray := make([]float32, len(hashBytes)/4) // Assuming 32-bit floats
	for i := range floatArray {
		bits := binary.LittleEndian.Uint32(hashBytes[i*4 : (i+1)*4])
		floatArray[i] = math.Float32frombits(bits)
	}

	return NewEmbeddingFromFloat32(floatArray), nil
}

func (e *ConsistentHashEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]*Embedding, error) {
	var embeddings []*Embedding
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

func NewConsistentHashEmbeddingFunction() EmbeddingFunction {
	return &ConsistentHashEmbeddingFunction{}
}
