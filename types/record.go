package types

import (
	"context"
	"fmt"
)

type Record struct {
	ID        string
	Embedding Embedding
	Metadata  map[string]interface{}
	Document  string
	URI       string
	err       error // indicating whether the record is valid or nto
}

var _ InvalidMetadataValueError
var _ InvalidEmbeddingValueError

type Option func(*Record) error

func WithID(id string) Option {
	return func(r *Record) error {
		if id == "" {
			return fmt.Errorf("id cannot be empty")
		}
		r.ID = id
		return nil
	}
}

func WithEmbedding(embedding Embedding) Option {
	return func(r *Record) error {
		r.Embedding = embedding
		return nil
	}
}

func WithMetadatas(metadata map[string]interface{}) Option {
	return func(r *Record) error {
		for k, v := range metadata {
			switch v.(type) {
			case string, int, float32, bool:
				r.Metadata[k] = v
			default:
				return &InvalidMetadataValueError{Key: k, Value: v}
			}
		}
		return nil
	}
}

func WithMetadata(key string, value interface{}) Option {
	return func(r *Record) error {
		if r.Metadata == nil {
			r.Metadata = map[string]interface{}{}
		}
		switch value.(type) {
		case string, int, float32, bool:
			r.Metadata[key] = value
		default:
			return &InvalidMetadataValueError{Key: key, Value: value}
		}
		return nil
	}
}

func WithURI(uri string) Option {
	return func(r *Record) error {
		if uri == "" {
			return fmt.Errorf("uri cannot be empty")
		}
		r.URI = uri
		return nil
	}
}

func WithDocument(document string) Option {
	return func(r *Record) error {
		if document == "" {
			return fmt.Errorf("document cannot be empty")
		}
		r.Document = document
		return nil
	}
}

// Validate checks if the record is valid
func (r *Record) Validate() error {
	if r.err != nil {
		return r.err
	}
	if r.ID == "" {
		return fmt.Errorf("id cannot be empty")
	}

	if !r.Embedding.IsDefined() && r.Document == "" {
		return fmt.Errorf("document or embedding must be provided")
	}
	return nil
}

type RecordProducer struct {
	IDGenerator              IDGenerator
	InPlaceEmbeddingFunction EmbeddingFunction
}

type ProducerOption func(*RecordProducer) error

func NewRecordProducer(opts ...ProducerOption) (*RecordProducer, error) {
	p := &RecordProducer{}
	for _, opt := range opts {
		err := opt(p)
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

func WithIDGenerator(idGenerator IDGenerator) ProducerOption {
	return func(p *RecordProducer) error {
		p.IDGenerator = idGenerator
		return nil
	}
}

// WithInPlaceEmbeddingFunction sets the embedding function to be used for in place embedding.
// IMPORTANT: This is very inefficient and should be used only for testing
func WithInPlaceEmbeddingFunction(embeddingFunction EmbeddingFunction) ProducerOption {
	return func(p *RecordProducer) error {
		p.InPlaceEmbeddingFunction = embeddingFunction
		return nil
	}
}

func (p *RecordProducer) Produce(recordOptions ...Option) (*Record, error) {
	r := &Record{}
	for _, opt := range recordOptions {
		err := opt(r)
		if err != nil {
			return nil, err
		}
	}

	if !r.Embedding.IsDefined() && r.Document == "" && p.InPlaceEmbeddingFunction == nil {
		return nil, fmt.Errorf("document or embedding or InPlace EmbeddingFunction must be provided")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // cancel the context to release resources
	if !r.Embedding.IsDefined() && r.Document != "" && p.InPlaceEmbeddingFunction != nil {
		embedding, err := p.InPlaceEmbeddingFunction.EmbedQuery(ctx, r.Document)
		if err != nil {
			return nil, err
		}
		r.Embedding = *embedding
	}
	if r.ID == "" && p.IDGenerator == nil {
		return nil, fmt.Errorf("either id or id generator is required. Use producer.WithIDGenerator or record.ID to set id")
	}
	if r.ID == "" {
		r.ID = p.IDGenerator.Generate(r.Document)
	}
	return r, nil
}

type RecordSet struct {
	Records []*Record
}

type RecordSetOption func(*RecordSet) error

func NewRecordSet(opts ...RecordSetOption) (*RecordSet, error) {
	rs := &RecordSet{Records: make([]*Record, 0)}
	for _, opt := range opts {
		err := opt(rs)
		if err != nil {
			return nil, err
		}
	}
	return rs, nil
}

func WithRecords(records []*Record) RecordSetOption {
	return func(rs *RecordSet) error {
		rs.Records = append(rs.Records, records...)
		return nil
	}
}

func WithRecord(record *Record) RecordSetOption {
	return func(rs *RecordSet) error {
		rs.Records = append(rs.Records, record)
		return nil
	}
}

func (rs *RecordSet) GetDocuments() []string {
	documents := make([]string, 0)
	for _, record := range rs.Records {
		documents = append(documents, record.Document)
	}
	return documents
}

func (rs *RecordSet) GetEmbeddings() []*Embedding {
	embeddings := make([]*Embedding, 0)
	for _, record := range rs.Records {
		embeddings = append(embeddings, &record.Embedding)
	}
	return embeddings
}

func (rs *RecordSet) GetIDs() []string {
	ids := make([]string, 0)
	for _, record := range rs.Records {
		ids = append(ids, record.ID)
	}
	return ids
}

func (rs *RecordSet) GetURIs() []string {
	uris := make([]string, 0)
	for _, record := range rs.Records {
		uris = append(uris, record.URI)
	}
	return uris
}

func (rs *RecordSet) GetMetadatas() []map[string]interface{} {
	metadatas := make([]map[string]interface{}, 0)
	for _, record := range rs.Records {
		metadatas = append(metadatas, record.Metadata)
	}
	return metadatas
}

func (rs *RecordSet) Validate() error {
	for _, record := range rs.Records {
		if record.err != nil {
			return record.err
		}
	}
	return nil
}
