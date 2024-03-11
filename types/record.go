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

type RecordSetOption func(*RecordSet) error

func WithIDGenerator(idGenerator IDGenerator) RecordSetOption {
	return func(p *RecordSet) error {
		p.IDGenerator = idGenerator
		return nil
	}
}

// WithEmbeddingFunction sets the embedding function to be used for in place embedding.
func WithEmbeddingFunction(embeddingFunction EmbeddingFunction) RecordSetOption {
	return func(p *RecordSet) error {
		p.EmbeddingFunction = embeddingFunction
		return nil
	}
}

type RecordSet struct {
	Records           []*Record
	IDGenerator       IDGenerator
	EmbeddingFunction EmbeddingFunction
}

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

func (rs *RecordSet) WithRecords(records []*Record) *RecordSet {
	rs.Records = append(rs.Records, records...)
	return rs
}

func (rs *RecordSet) WithRecord(recordOpts ...Option) *RecordSet {
	record := &Record{}
	for _, opt := range recordOpts {
		err := opt(record)
		if err != nil {
			record.err = err
			// TODO optionally write error to log
		}
	}
	if record.ID == "" && rs.IDGenerator == nil {
		record.err = fmt.Errorf("either id or id generator is required. Use producer.WithIDGenerator or record.ID to set id")
	}

	if record.ID == "" && rs.IDGenerator != nil {
		record.ID = rs.IDGenerator.Generate(record.Document)
	}
	rs.Records = append(rs.Records, record)
	return rs
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

// Validate the whole record set by calling record.Validate
func (rs *RecordSet) Validate() error {
	if rs.EmbeddingFunction == nil {
		return fmt.Errorf("embedding function is required")
	}
	for _, record := range rs.Records {
		err := record.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

func (rs *RecordSet) BuildAndValidate(ctx context.Context) ([]*Record, error) {
	err := rs.Validate()
	if err != nil {
		return nil, err
	}
	err = rs.EmbeddingFunction.EmbedRecords(ctx, rs.Records, false)

	if err != nil {
		return nil, err
	}

	if err := rs.Validate(); err != nil {
		return nil, err
	}
	return rs.Records, nil
}
