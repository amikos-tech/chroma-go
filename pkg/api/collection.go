package api

import (
	"context"
	"encoding/json"
	"errors"
)

type Collection interface {
	// Name returns the name of the collection
	Name() string
	// ID returns the id of the collection
	ID() string
	// Tenant returns the tenant of the collection
	Tenant() Tenant
	// Database returns the database of the collection
	Database() Database
	// Metadata returns the metadata of the collection
	Metadata() CollectionMetadata
	// Configuration returns the configuration of the collection
	Configuration() CollectionConfiguration
	// Add adds a document to the collection
	Add(ctx context.Context, opts ...CollectionUpdateOption) error
	// Upsert updates or adds a document to the collection
	Upsert(ctx context.Context, opts ...CollectionUpdateOption) error
	// Update updates a document in the collection
	Update(ctx context.Context, opts ...CollectionUpdateOption) error
	// Delete deletes documents from the collection
	Delete(ctx context.Context, opts ...CollectionDeleteOption) error
	// Count returns the number of documents in the collection
	Count(ctx context.Context) (int, error)
	// ModifyName modifies the name of the collection
	ModifyName(ctx context.Context, newName string) error
	// ModifyMetadata modifies the metadata of the collection
	ModifyMetadata(ctx context.Context, newMetadata CollectionMetadata) error
	// ModifyConfiguration modifies the configuration of the collection
	ModifyConfiguration(ctx context.Context, newConfig CollectionConfiguration) error // not supported yet
	// Get gets documents from the collection
	Get(ctx context.Context, opts ...CollectionGetOption) (GetResult, error)
	// Query queries the collection
	Query(ctx context.Context, opts ...CollectionQueryOption) (QueryResult, error)
	// BatchAdd(ctx context.Context, opts ...CollectionUpdateOption) error
}

type CollectionBase struct {
	Name          string                  `json:"name"`
	CollectionID  string                  `json:"id"`
	Tenant        Tenant                  `json:"tenant"`
	Database      Database                `json:"database"`
	Metadata      CollectionMetadata      `json:"metadata"`
	Configuration CollectionConfiguration `json:"configuration_json"`
}

type CollectionOp interface {
	// PrepareAndValidate validates the operation. Each operation must implement this method to ensure the operation is valid and can be sent over the wire
	PrepareAndValidate() error
	// MarshalJSON marshals the operation to JSON
	MarshalJSON() ([]byte, error)
	// UnmarshalJSON unmarshals the operation from JSON
	UnmarshalJSON(b []byte) error
}

type FilterOp struct {
	Where         WhereFilter         `json:"where"`
	WhereDocument WhereDocumentFilter `json:"where_document"`
}

type FilterIDOp struct {
	Ids []DocumentID `json:"ids"`
}

type FilterTextsOp struct {
	QueryTexts []string `json:"query_texts"`
}

type FilterEmbeddingsOp struct {
	QueryEmbeddings []Embedding `json:"query_embeddings"`
}

type ProjectOp struct {
	Include []Include `json:"include"`
}

type LimitAndOffsetOp struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type LimitResultOp struct {
	NResults int `json:"n_results"`
}

type SortOp struct {
	Sort string `json:"sort"`
}

type CollectionGetOption func(get *CollectionGetOp) error

type CollectionGetOp struct {
	FilterOp         // ability to filter by where and whereDocument
	FilterIDOp       // ability to filter by id
	ProjectOp        // include metadatas, documents, embeddings, uris, ids
	LimitAndOffsetOp // limit and offset
	SortOp           // sort
	ResourceOperation
}

func NewCollectionGetOp(opts ...CollectionGetOption) (*CollectionGetOp, error) {
	get := &CollectionGetOp{
		ProjectOp: ProjectOp{Include: []Include{IncludeDocuments, IncludeMetadatas, IncludeIDs}},
	}
	for _, opt := range opts {
		err := opt(get)
		if err != nil {
			return nil, err
		}
	}
	return get, nil
}

func (c *CollectionGetOp) Validate() error {
	if c.Sort != "" {
		return errors.New("sort is not supported yet")
	}
	if c.Limit <= 0 {
		return errors.New("limit must be greater than 0")
	}
	if c.Offset < 0 {
		return errors.New("offset must be greater than or equal to 0")
	}
	if len(c.Include) == 0 {
		return errors.New("at least one include option is required")
	}
	if c.Where != nil {
		if err := c.Where.Validate(); err != nil {
			return err
		}
	}
	if c.WhereDocument != nil {
		if err := c.WhereDocument.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *CollectionGetOp) MarshalJSON() ([]byte, error) {
	type Alias CollectionGetOp
	return json.Marshal(struct{ *Alias }{Alias: (*Alias)(c)})
}

func (c *CollectionGetOp) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, c)
}

func (c *CollectionGetOp) Resource() Resource {
	return ResourceCollection
}

func (c *CollectionGetOp) Operation() OperationType {
	return OperationGet
}

func WithWhereGet(where WhereFilter) CollectionGetOption {
	return func(query *CollectionGetOp) error {
		query.Where = where
		return nil
	}
}

func WithWhereDocumentGet(whereDocument WhereDocumentFilter) CollectionGetOption {
	return func(query *CollectionGetOp) error {
		query.WhereDocument = whereDocument
		return nil
	}
}

func WithIncludeGet(include ...Include) CollectionGetOption {
	return func(query *CollectionGetOp) error {
		query.Include = include
		return nil
	}
}

func WithLimitGet(limit int) CollectionGetOption {
	return func(query *CollectionGetOp) error {
		if limit <= 0 {
			return errors.New("limit must be greater than 0")
		}
		query.Limit = limit
		return nil
	}
}

func WithOffsetGet(offset int) CollectionGetOption {
	return func(query *CollectionGetOp) error {
		if offset <= 0 {
			return errors.New("offset must be greater than or equal to 0")
		}
		query.Offset = offset
		return nil
	}
}

// Query

type CollectionQueryOp struct {
	FilterOp
	FilterEmbeddingsOp
	FilterTextsOp
	LimitResultOp
}

func NewCollectionQueryOp(opts ...CollectionQueryOption) (*CollectionQueryOp, error) {
	query := &CollectionQueryOp{
		LimitResultOp: LimitResultOp{NResults: 10},
	}
	for _, opt := range opts {
		err := opt(query)
		if err != nil {
			return nil, err
		}
	}
	return query, nil
}

func (c *CollectionQueryOp) Validate() error {
	if len(c.QueryEmbeddings) == 0 && len(c.QueryTexts) == 0 {
		return errors.New("at least one query embedding or query text is required")
	}
	if c.NResults <= 0 {
		return errors.New("nResults must be greater than 0")
	}
	if c.Where != nil {
		if err := c.Where.Validate(); err != nil {
			return err
		}
	}
	if c.WhereDocument != nil {
		if err := c.WhereDocument.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c *CollectionQueryOp) MarshalJSON() ([]byte, error) {
	type Alias CollectionQueryOp
	return json.Marshal(struct{ *Alias }{Alias: (*Alias)(c)})
}

func (c *CollectionQueryOp) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, c)
}

func (c *CollectionQueryOp) Resource() Resource {
	return ResourceCollection
}

func (c *CollectionQueryOp) Operation() OperationType {
	return OperationQuery
}

type CollectionQueryOption func(query *CollectionQueryOp) error

func WithWhereQuery(where WhereFilter) CollectionQueryOption {
	return func(query *CollectionQueryOp) error {
		query.Where = where
		return nil
	}
}

func WithWhereDocumentQuery(whereDocument WhereDocumentFilter) CollectionQueryOption {
	return func(query *CollectionQueryOp) error {
		query.WhereDocument = whereDocument
		return nil
	}
}

func WithNResults(nResults int) CollectionQueryOption {
	return func(query *CollectionQueryOp) error {
		if nResults <= 0 {
			return errors.New("nResults must be greater than 0")
		}
		query.NResults = nResults
		return nil
	}
}

// Add, Upsert, Update

type CollectionUpdateOp struct {
	Ids        []DocumentID       `json:"ids"`
	Documents  []Document         `json:"documents"`
	Metadatas  []DocumentMetadata `json:"metadatas"`
	Embeddings []Embedding        `json:"embeddings"`
	Records    []Record           `json:"-"`
}

func NewCollectionUpdateOp(opts ...CollectionUpdateOption) (*CollectionUpdateOp, error) {
	update := &CollectionUpdateOp{}
	for _, opt := range opts {
		err := opt(update)
		if err != nil {
			return nil, err
		}
	}
	return update, nil
}

func (c *CollectionUpdateOp) PrepareAndValidate() error {
	if len(c.Ids) == 0 && len(c.Records) == 0 {
		return errors.New("at least one record is required")
	}
	if len(c.Ids) > 0 && len(c.Documents) == 0 && len(c.Embeddings) == 0 {
		return errors.New("at least one document, metadata or embedding is required")
	}
	// for each non 0 length slice of Documents, Metadatas, Embeddings ensure its length is equal to the length of the other slices
	if len(c.Documents) > 0 && len(c.Ids) != len(c.Documents) {
		return errors.New("number of ids must be equal to number of documents")
	}
	if len(c.Metadatas) > 0 && len(c.Ids) != len(c.Metadatas) {
		return errors.New("number of ids must be equal to number of metadatas")
	}
	if len(c.Embeddings) > 0 && len(c.Ids) != len(c.Embeddings) {
		return errors.New("number of ids must be equal to number of embeddings")
	}
	if len(c.Records) > 0 {
		for _, record := range c.Records {
			err := record.Validate()
			if err != nil {
				return err
			}
			recordIds, recordDocuments, recordEmbeddings, recordMetadata := record.Unwrap()
			c.Ids = append(c.Ids, recordIds)
			c.Documents = append(c.Documents, recordDocuments)
			c.Metadatas = append(c.Metadatas, recordMetadata)
			c.Embeddings = append(c.Embeddings, recordEmbeddings)
		}
	}

	return nil
}

func (c *CollectionUpdateOp) MarshalJSON() ([]byte, error) {
	type Alias CollectionUpdateOp
	return json.Marshal(struct{ *Alias }{Alias: (*Alias)(c)})
}

func (c *CollectionUpdateOp) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, c)
}

func (c *CollectionUpdateOp) Resource() Resource {
	return ResourceCollection
}

func (c *CollectionUpdateOp) Operation() OperationType {
	return OperationCreate
}

type CollectionUpdateOption func(update *CollectionUpdateOp) error

func WithTexts(documents ...string) CollectionUpdateOption {
	return func(update *CollectionUpdateOp) error {
		if len(documents) == 0 {
			return errors.New("at least one document is required")
		}
		if update.Documents == nil {
			update.Documents = make([]Document, 0)
		}
		for _, text := range documents {
			update.Documents = append(update.Documents, NewTextDocument(text))
		}
		return nil
	}
}

func WithMetadatas(metadatas ...DocumentMetadata) CollectionUpdateOption {
	return func(update *CollectionUpdateOp) error {
		update.Metadatas = metadatas
		return nil
	}
}

func WithIDs(ids ...string) CollectionUpdateOption {
	return func(update *CollectionUpdateOp) error {
		for _, id := range ids {
			update.Ids = append(update.Ids, DocumentID(id))
		}
		return nil
	}
}

func WithEmbeddings(embeddings ...Embedding) CollectionUpdateOption {
	return func(update *CollectionUpdateOp) error {
		update.Embeddings = embeddings
		return nil
	}
}

// Delete

type CollectionDeleteOp struct {
	FilterOp
	FilterIDOp
}

func NewCollectionDeleteOp(opts ...CollectionDeleteOption) (*CollectionDeleteOp, error) {
	del := &CollectionDeleteOp{}
	for _, opt := range opts {
		err := opt(del)
		if err != nil {
			return nil, err
		}
	}
	return del, nil
}

func (c *CollectionDeleteOp) Validate() error {
	if len(c.Ids) == 0 && c.Where == nil && c.WhereDocument == nil {
		return errors.New("at least one filter is required, ids, where or whereDocument")
	}

	if c.Where != nil {
		if err := c.Where.Validate(); err != nil {
			return err
		}
	}

	if c.WhereDocument != nil {
		if err := c.WhereDocument.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *CollectionDeleteOp) MarshalJSON() ([]byte, error) {
	type Alias CollectionDeleteOp
	return json.Marshal(struct{ *Alias }{Alias: (*Alias)(c)})
}

func (c *CollectionDeleteOp) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, c)
}

func (c *CollectionDeleteOp) Resource() Resource {
	return ResourceCollection
}

func (c *CollectionDeleteOp) Operation() OperationType {
	return OperationDelete
}

type CollectionDeleteOption func(update *CollectionDeleteOp) error

func WithWhereDelete(where WhereFilter) CollectionDeleteOption {
	return func(delete *CollectionDeleteOp) error {
		delete.Where = where
		return nil
	}
}

func WithWhereDocumentDelete(whereDocument WhereDocumentFilter) CollectionDeleteOption {
	return func(delete *CollectionDeleteOp) error {
		delete.WhereDocument = whereDocument
		return nil
	}
}

func WithIDsDelete(ids ...DocumentID) CollectionDeleteOption {
	return func(delete *CollectionDeleteOp) error {
		delete.Ids = ids
		return nil
	}
}

type CollectionConfiguration interface {
	GetRaw(key string) (interface{}, bool)
}
