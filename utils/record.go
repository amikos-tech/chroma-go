package utils

import (
	"github.com/amikos-tech/chroma-go"
)

type Record struct {
	ID        string
	Embedding []interface{}
	Metadata  map[string]interface{}
	Document  string
	URI       string
	err       error
}

var _ chroma.InvalidMetadataValueError
var _ chroma.InvalidEmbeddingValueError

type RecordBuilder struct {
	Record *Record
}

func NewRecordBuilder() *RecordBuilder {
	return &RecordBuilder{Record: &Record{}}
}

func (r *RecordBuilder) WithID(id string) *RecordBuilder {
	if r.Record.err != nil {
		return r
	}
	r.Record.ID = id
	return r
}

func (r *RecordBuilder) WithEmbedding(embedding []interface{}) *RecordBuilder {
	if r.Record.err != nil {
		return r
	}
	for _, v := range embedding {
		switch v.(type) {
		case int, float32:
		default:
			r.Record.err = &chroma.InvalidEmbeddingValueError{Value: v}
			return r
		}
	}
	r.Record.Embedding = embedding
	return r
}

func (r *RecordBuilder) WithMetadatas(metadata map[string]interface{}) *RecordBuilder {
	if r.Record.err != nil {
		return r
	}
	for k, v := range metadata {
		switch v.(type) {
		case string, int, float32, bool:
			r.Record.Metadata[k] = v
		default:
			r.Record.err = &chroma.InvalidMetadataValueError{Key: k, Value: v}
			return r
		}
	}
	r.Record.Metadata = metadata
	return r
}

func (r *RecordBuilder) WithMetadata(key string, value interface{}) *RecordBuilder {
	if r.Record.err != nil {
		return r
	}
	switch value.(type) {
	case string, int, float32, bool:
		r.Record.Metadata[key] = value
	default:
		r.Record.err = &chroma.InvalidMetadataValueError{Key: key, Value: value}
	}
	return r
}
func (r *RecordBuilder) WithURI(uri string) *RecordBuilder {
	if r.Record.err != nil {
		return r
	}
	r.Record.URI = uri
	return r
}

func (r *RecordBuilder) WithDocument(document string) *RecordBuilder {
	if r.Record.err != nil {
		return r
	}
	r.Record.Document = document
	return r
}

func (r *RecordBuilder) Build() (*Record, error) {
	if r.Record.err != nil {
		return nil, r.Record.err
	}
	return r.Record, nil
}

type RecordSetBuilder struct {
	RecordSet         *[]Record
	RecordSetForEmbed *[]Record
	Generator         chroma.IDGenerator
}

func NewRecordSetBuilder(idGenerator chroma.IDGenerator) *RecordSetBuilder {
	return &RecordSetBuilder{
		Generator:         idGenerator,
		RecordSet:         &[]Record{},
		RecordSetForEmbed: &[]Record{},
	}
}

func (r *RecordSetBuilder) WithRecord(record Record) *RecordSetBuilder {
	if (record.Document == "" || record.URI == "") && record.Embedding == nil {
		record.err = &chroma.ErrNoDocumentOrEmbedding{}
	}
	if record.ID == "" && r.Generator != nil {
		record.ID = r.Generator.Generate(record.Document)
	}
	if record.Document != "" && record.Embedding == nil {
		*r.RecordSetForEmbed = append(*r.RecordSetForEmbed, record)
	} else {
		*r.RecordSet = append(*r.RecordSet, record)
	}
	return r
}
