package types

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	time "time"

	"github.com/google/uuid"
	"github.com/oklog/ulid"

	openapi "github.com/amikos-tech/chroma-go/swagger"
	"github.com/amikos-tech/chroma-go/where"
	wheredoc "github.com/amikos-tech/chroma-go/where_document"
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

func ToDistanceFunction(str any) (DistanceFunction, error) {
	switch str {
	case L2:
		return L2, nil
	case COSINE:
		return COSINE, nil
	case IP:
		return IP, nil
	}
	if str == "" {
		return L2, nil
	}
	switch strings.ToLower(str.(string)) {
	case "l2":
		return L2, nil
	case "cosine":
		return COSINE, nil
	case "ip":
		return IP, nil
	default:
		return "", fmt.Errorf("invalid distance function: %s", str)
	}
}

type InvalidMetadataValueError struct {
	Key   string
	Value interface{}
}

func (e *InvalidMetadataValueError) Error() string {
	return fmt.Sprintf("Invalid metadata value type for key %s: %T", e.Key, e.Value)
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
	var arrayOfFloat32 = make([]float32, 0)
	var arrayOfInt32 = make([]int32, 0)
	if len(embeddings) == 0 {
		return &Embedding{
			ArrayOfFloat32: &[]float32{},
			ArrayOfInt32:   &([]int32{}),
		}, nil
	}
	for _, v := range embeddings {
		switch val := v.(type) {
		case int:
			arrayOfInt32 = append(arrayOfInt32, int32(val))
		case int32:
			arrayOfInt32 = append(arrayOfInt32, val)
		case float32:
			arrayOfFloat32 = append(arrayOfFloat32, val)
		case float64:
			arrayOfFloat32 = append(arrayOfFloat32, float32(val))
		default:
			return nil, &InvalidEmbeddingValueError{Value: v}
		}
	}
	return &Embedding{ArrayOfFloat32: &arrayOfFloat32, ArrayOfInt32: &arrayOfInt32}, nil
}

func (e *Embedding) String() string {
	if e.ArrayOfFloat32 != nil {
		return fmt.Sprintf("%v", e.ArrayOfFloat32)
	}
	if e.ArrayOfInt32 != nil {
		return fmt.Sprintf("%v", e.ArrayOfInt32)
	}
	return ""
}
func (e *Embedding) GetFloat32() *[]float32 {
	return e.ArrayOfFloat32
}

func (e *Embedding) GetInt32() *[]int32 {
	return e.ArrayOfInt32
}

func (e *Embedding) IsDefined() bool {
	return e.ArrayOfFloat32 != nil && len(*e.ArrayOfFloat32) > 0 || e.ArrayOfInt32 != nil && len(*e.ArrayOfInt32) > 0
}

func NewEmbeddingFromFloat32(embedding []float32) *Embedding {
	return &Embedding{
		ArrayOfFloat32: &embedding,
		ArrayOfInt32:   nil,
	}
}

func NewEmbeddingFromInt32(embedding []int32) *Embedding {
	return &Embedding{
		ArrayOfFloat32: nil,
		ArrayOfInt32:   &embedding,
	}
}

func NewEmbeddingsFromFloat32(embeddings [][]float32) []*Embedding {
	var embeddingsArray = make([]*Embedding, 0)
	if len(embeddings) == 0 {
		return embeddingsArray
	}
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
	var apiEmbeddings = make([]openapi.EmbeddingsInner, 0)
	if len(embeddings) == 0 {
		return apiEmbeddings
	}
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
	EmbedRecords(ctx context.Context, records []*Record, force bool) error
}

func EmbedRecordsDefaultImpl(e EmbeddingFunction, ctx context.Context, records []*Record, force bool) error {
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

func NewConsistentHashEmbeddingFunction() EmbeddingFunction {
	return &ConsistentHashEmbeddingFunction{dim: 378}
}

type CollectionQueryBuilder struct {
	QueryTexts      []string
	QueryEmbeddings []*Embedding
	Where           map[string]interface{}
	WhereDocument   map[string]interface{}
	NResults        int32
	Include         []QueryEnum
	Offset          int32
	Limit           int32
	Ids             []string
}

type CollectionQueryOption func(*CollectionQueryBuilder) error

func WithWhereMap(where map[string]interface{}) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		// TODO validate where
		c.Where = where
		return nil
	}
}

func WithWhere(operation where.WhereOperation) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		expr, err := where.Where(operation)
		if err != nil {
			return err
		}
		c.Where = expr
		return nil
	}
}

func WithWhereDocumentMap(where map[string]interface{}) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		// TODO validate where
		c.WhereDocument = where
		return nil
	}
}
func WithWhereDocument(operation wheredoc.WhereDocumentOperation) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		expr, err := wheredoc.WhereDocument(operation)
		if err != nil {
			return err
		}
		c.WhereDocument = expr
		return nil
	}
}

func WithNResults(nResults int32) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		if nResults < 1 {
			return fmt.Errorf("nResults must be greater than 0")
		}
		c.NResults = nResults
		return nil
	}
}

func WithQueryText(queryText string) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		if queryText == "" {
			return fmt.Errorf("queryText must not be empty")
		}
		c.QueryTexts = append(c.QueryTexts, queryText)
		return nil
	}
}

func WithQueryTexts(queryTexts []string) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		if len(queryTexts) == 0 {
			return fmt.Errorf("queryTexts must not be empty")
		}
		c.QueryTexts = queryTexts
		return nil
	}
}

func WithQueryEmbeddings(queryEmbeddings []*Embedding) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		for _, embedding := range queryEmbeddings {
			if embedding == nil || !embedding.IsDefined() {
				return fmt.Errorf("embedding must not be nil or empty")
			}
		}
		c.QueryEmbeddings = append(c.QueryEmbeddings, queryEmbeddings...)
		return nil
	}
}

func WithQueryEmbedding(queryEmbedding *Embedding) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		if queryEmbedding == nil {
			return fmt.Errorf("embedding must not be empty")
		}
		c.QueryEmbeddings = append(c.QueryEmbeddings, queryEmbedding)
		return nil
	}
}

func WithInclude(include ...QueryEnum) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		c.Include = include
		return nil
	}
}

func WithOffset(offset int32) CollectionQueryOption {
	return func(q *CollectionQueryBuilder) error {
		if offset < 0 {
			return fmt.Errorf("offset must be greater than or equal to 0")
		}
		q.Offset = offset
		return nil
	}
}

func WithLimit(limit int32) CollectionQueryOption {
	return func(q *CollectionQueryBuilder) error {
		if limit < 1 {
			return fmt.Errorf("limit must be greater than 0")
		}
		q.Limit = limit
		return nil
	}
}

func WithIds(ids []string) CollectionQueryOption {
	return func(q *CollectionQueryBuilder) error {
		q.Ids = ids
		return nil
	}
}

type CredentialsProvider interface {
	Authenticate(apiClient *openapi.Configuration) error
}

type BasicAuthCredentialsProvider struct {
	Username string
	Password string
}

func NewBasicAuthCredentialsProvider(username, password string) *BasicAuthCredentialsProvider {
	return &BasicAuthCredentialsProvider{
		Username: username,
		Password: password,
	}
}

func (b *BasicAuthCredentialsProvider) Authenticate(config *openapi.Configuration) error {
	auth := b.Username + ":" + b.Password
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	config.DefaultHeader["Authorization"] = "Basic " + encodedAuth
	return nil
}

type TokenTransportHeader string

const (
	AuthorizationTokenHeader TokenTransportHeader = "Authorization"
	XChromaTokenHeader       TokenTransportHeader = "X-Chroma-Token"
)

type TokenAuthCredentialsProvider struct {
	Token  string
	Header TokenTransportHeader
}

func NewTokenAuthCredentialsProvider(token string, header TokenTransportHeader) *TokenAuthCredentialsProvider {
	return &TokenAuthCredentialsProvider{
		Token:  token,
		Header: header,
	}
}

func (t *TokenAuthCredentialsProvider) Authenticate(config *openapi.Configuration) error {
	switch t.Header {
	case AuthorizationTokenHeader:
		config.DefaultHeader[string(t.Header)] = "Bearer " + t.Token
		return nil
	case XChromaTokenHeader:
		config.DefaultHeader[string(t.Header)] = t.Token
		return nil
	default:
		return fmt.Errorf("unsupported token header: %v", t.Header)
	}
}
