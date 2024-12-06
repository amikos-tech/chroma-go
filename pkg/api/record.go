package api

type Record interface {
	ID() DocumentID
	Document() Document // should work for both text and URI based documents
	Embedding() Embedding
	Metadata() DocumentMetadata
	Validate() error
	Unwrap() (DocumentID, Document, Embedding, DocumentMetadata)
}
