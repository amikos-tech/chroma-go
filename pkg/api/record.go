package api

import (
	"fmt"

	"github.com/pkg/errors"
)

type Record interface {
	ID() DocumentID
	Document() Document // should work for both text and URI based documents
	Embedding() Embedding
	Metadata() DocumentMetadata
	Validate() error
	Unwrap() (DocumentID, Document, Embedding, DocumentMetadata)
}

type SimpleRecord struct {
	id        string
	embedding Embedding
	metadata  map[string]interface{}
	document  string
	uri       string
	err       error // indicating whether the record is valid or nto
}
type RecordOption func(record *SimpleRecord) error

func WithRecordID(id string) RecordOption {
	return func(r *SimpleRecord) error {
		r.id = id
		return nil
	}
}

func WithRecordEmbedding(embedding Embedding) RecordOption {
	return func(r *SimpleRecord) error {
		r.embedding = embedding
		return nil
	}
}

func WithRecordMetadatas(metadata map[string]interface{}) RecordOption {
	return func(r *SimpleRecord) error {
		r.metadata = metadata
		return nil
	}
}
func (r *SimpleRecord) constructValidate() error {
	if r.id == "" {
		return errors.New("record id is empty")
	}
	return nil
}
func NewSimpleRecord(opts ...RecordOption) (*SimpleRecord, error) {
	r := &SimpleRecord{}
	for _, opt := range opts {
		err := opt(r)
		if err != nil {
			return nil, errors.Wrap(err, "error applying record option")
		}
	}
	fmt.Println("=123121dswq")

	err := r.constructValidate()
	if err != nil {
		return nil, errors.Wrap(err, "error validating record")
	}
	return r, nil
}

func (r *SimpleRecord) ID() DocumentID {
	return DocumentID(r.id)
}

func (r *SimpleRecord) Document() Document {
	return NewTextDocument(r.document)
}

func (r *SimpleRecord) URI() string {
	return r.uri
}

func (r *SimpleRecord) Embedding() Embedding {
	return r.embedding
}

func (r *SimpleRecord) Metadata() DocumentMetadata {
	return NewDocumentMetadata(r.metadata)
}

func (r *SimpleRecord) Validate() error {
	return r.err
}

func (r *SimpleRecord) Unwrap() (DocumentID, Document, Embedding, DocumentMetadata) {
	return r.ID(), r.Document(), r.Embedding(), r.Metadata()
}
