package v2

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
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
	// Dimension returns the dimension of the embeddings in the collection
	Dimension() int
	// Configuration returns the configuration of the collection
	Configuration() CollectionConfiguration
	// Add adds a document to the collection
	Add(ctx context.Context, opts ...CollectionAddOption) error
	// Upsert updates or adds a document to the collection
	Upsert(ctx context.Context, opts ...CollectionAddOption) error
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
	// Close closes the collection and releases any resources
	Close() error
}

type CollectionOp interface {
	// PrepareAndValidate validates the operation. Each operation must implement this method to ensure the operation is valid and can be sent over the wire
	PrepareAndValidate() error
	EmbedData(ctx context.Context, ef embeddings.EmbeddingFunction) error
	// MarshalJSON marshals the operation to JSON
	MarshalJSON() ([]byte, error)
	// UnmarshalJSON unmarshals the operation from JSON
	UnmarshalJSON(b []byte) error
}

type FilterOp struct {
	Where         WhereFilter         `json:"where,omitempty"`
	WhereDocument WhereDocumentFilter `json:"where_document,omitempty"`
}

type FilterIDOp struct {
	Ids []DocumentID `json:"ids,omitempty"`
}

type FilterTextsOp struct {
	QueryTexts []string `json:"-"`
}

type FilterEmbeddingsOp struct {
	QueryEmbeddings []embeddings.Embedding `json:"query_embeddings"`
}

type ProjectOp struct {
	Include []Include `json:"include,omitempty"`
}

type LimitAndOffsetOp struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

type LimitResultOp struct {
	NResults int `json:"n_results"`
}

type SortOp struct {
	Sort string `json:"sort,omitempty"`
}

type CollectionGetOption func(get *CollectionGetOp) error

type CollectionGetOp struct {
	FilterOp          // ability to filter by where and whereDocument
	FilterIDOp        // ability to filter by id
	ProjectOp         // include metadatas, documents, embeddings, uris, ids
	LimitAndOffsetOp  // limit and offset
	SortOp            // sort
	ResourceOperation `json:"-"`
}

func NewCollectionGetOp(opts ...CollectionGetOption) (*CollectionGetOp, error) {
	get := &CollectionGetOp{
		ProjectOp: ProjectOp{Include: []Include{IncludeDocuments, IncludeMetadatas}},
	}
	for _, opt := range opts {
		err := opt(get)
		if err != nil {
			return nil, err
		}
	}
	return get, nil
}

func (c *CollectionGetOp) PrepareAndValidate() error {
	if c.Sort != "" {
		return errors.New("sort is not supported yet")
	}
	if c.Limit < 0 {
		return errors.New("limit must be greater than or equal to 0")
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

func WithIDsGet(ids ...DocumentID) CollectionGetOption {
	return func(query *CollectionGetOp) error {
		for _, id := range ids {
			query.Ids = append(query.Ids, DocumentID(id))
		}
		return nil
	}
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
		if offset < 0 {
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
	ProjectOp // include metadatas, documents, embeddings, uris
	FilterIDOp
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

func (c *CollectionQueryOp) PrepareAndValidate() error {
	if len(c.QueryEmbeddings) == 0 && len(c.QueryTexts) == 0 {
		return errors.New("at least one query embedding or query text is required")
	}
	if c.NResults <= 0 {
		return errors.New("nResults must be greater than 0")
	}
	if c.Where != nil {
		if err := c.Where.Validate(); err != nil {
			return errors.Wrap(err, "where validation failed")
		}
	}
	if c.WhereDocument != nil {
		if err := c.WhereDocument.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c *CollectionQueryOp) EmbedData(ctx context.Context, ef embeddings.EmbeddingFunction) error {
	if len(c.QueryTexts) > 0 && len(c.QueryEmbeddings) == 0 {
		if ef == nil {
			return errors.New("embedding function is required")
		}
		embeddings, err := ef.EmbedDocuments(ctx, c.QueryTexts)
		if err != nil {
			return errors.Wrap(err, "embedding failed")
		}
		c.QueryEmbeddings = embeddings
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

func WithQueryTexts(queryTexts ...string) CollectionQueryOption {
	return func(query *CollectionQueryOp) error {
		if len(queryTexts) == 0 {
			return errors.New("at least one query text is required")
		}
		query.QueryTexts = queryTexts
		return nil
	}
}

func WithQueryEmbeddings(queryEmbeddings ...embeddings.Embedding) CollectionQueryOption {
	return func(query *CollectionQueryOp) error {
		if len(queryEmbeddings) == 0 {
			return errors.New("at least one query embedding is required")
		}
		query.QueryEmbeddings = queryEmbeddings
		return nil
	}
}

// WithIncludeQuery is used to include metadatas, documents, embeddings, uris in the query response.
func WithIncludeQuery(include ...Include) CollectionQueryOption {
	return func(query *CollectionQueryOp) error {
		query.Include = include
		return nil
	}
}

// WithIDsQuery is used to filter the query by IDs. This is only available for Chroma version 1.0.3 and above.
func WithIDsQuery(ids ...DocumentID) CollectionQueryOption {
	return func(query *CollectionQueryOp) error {
		if len(ids) == 0 {
			return errors.New("at least one id is required")
		}
		if query.Ids == nil {
			query.Ids = make([]DocumentID, 0)
		}
		query.Ids = append(query.Ids, ids...)
		return nil
	}
}

// Add, Upsert, Update

type CollectionAddOp struct {
	Ids         []DocumentID           `json:"ids"`
	Documents   []Document             `json:"documents,omitempty"`
	Metadatas   []DocumentMetadata     `json:"metadatas,omitempty"`
	Embeddings  []embeddings.Embedding `json:"embeddings"`
	Records     []Record               `json:"-"`
	IDGenerator IDGenerator            `json:"-"`
}

func NewCollectionAddOp(opts ...CollectionAddOption) (*CollectionAddOp, error) {
	update := &CollectionAddOp{}
	for _, opt := range opts {
		err := opt(update)
		if err != nil {
			return nil, err
		}
	}
	return update, nil
}

func (c *CollectionAddOp) EmbedData(ctx context.Context, ef embeddings.EmbeddingFunction) error {
	// invariants:
	// documents only - we embed
	// documents + embeddings - we skip
	// embeddings only - we skip
	if len(c.Documents) > 0 && len(c.Embeddings) == 0 {
		if ef == nil {
			return errors.New("embedding function is required")
		}
		texts := make([]string, len(c.Documents))
		for i, doc := range c.Documents {
			texts[i] = doc.ContentString()
		}
		embeddings, err := ef.EmbedDocuments(ctx, texts)
		if err != nil {
			return errors.Wrap(err, "embedding failed")
		}
		for i, embedding := range embeddings {
			if i >= len(c.Embeddings) {
				c.Embeddings = append(c.Embeddings, embedding)
			} else {
				c.Embeddings[i] = embedding
			}
		}
	}
	return nil
}

func (c *CollectionAddOp) GenerateIDs() error {
	if c.IDGenerator == nil {
		return nil
	}
	generatedIDLen := 0
	switch {
	case len(c.Documents) > 0:
		generatedIDLen = len(c.Documents)
	case len(c.Embeddings) > 0:
		generatedIDLen = len(c.Embeddings)
	case len(c.Records) > 0:
		return errors.New("not implemented yet")
	default:
		return errors.New("at least one document or embedding is required")
	}
	c.Ids = make([]DocumentID, 0)
	for i := 0; i < generatedIDLen; i++ {
		switch {
		case len(c.Documents) > 0:
			c.Ids = append(c.Ids, DocumentID(c.IDGenerator.Generate(WithDocument(c.Documents[i].ContentString()))))
		case len(c.Embeddings) > 0:
			c.Ids = append(c.Ids, DocumentID(c.IDGenerator.Generate()))

		case len(c.Records) > 0:
			return errors.New("not implemented yet")
		}
	}
	return nil
}

func (c *CollectionAddOp) PrepareAndValidate() error {
	// invariants
	// - at least one ID or one record is required
	// - if IDs are provided, they must be unique
	// - if IDs are provided, the number of documents or embeddings must match the number of IDs
	// - if IDs are provided, if metadatas are also provided they must match the number of IDs

	if (len(c.Ids) == 0 && c.IDGenerator == nil) && len(c.Records) == 0 {
		return errors.New("at least one ID or record is required. Alternatively, an ID generator can be provided") // TODO add link to docs
	}

	// should we generate IDs?
	if c.IDGenerator != nil {
		err := c.GenerateIDs()
		if err != nil {
			return errors.Wrap(err, "failed to generate IDs")
		}
	}

	// if IDs are provided, they must be unique
	idSet := make(map[DocumentID]struct{})
	for _, id := range c.Ids {
		if _, exists := idSet[id]; exists {
			return errors.Errorf("duplicate id found: %s", id)
		}
		idSet[id] = struct{}{}
	}

	// if IDs are provided, the number of documents or embeddings must match the number of IDs
	if len(c.Documents) > 0 && len(c.Ids) != len(c.Documents) {
		return errors.Errorf("documents (%d) must match the number of ids (%d)", len(c.Documents), len(c.Ids))
	}

	if len(c.Embeddings) > 0 && len(c.Ids) != len(c.Embeddings) {
		return errors.Errorf("embeddings (%d) must match the number of ids (%d)", len(c.Embeddings), len(c.Ids))
	}

	// if IDs are provided, if metadatas are also provided they must match the number of IDs

	if len(c.Metadatas) > 0 && len(c.Ids) != len(c.Metadatas) {
		return errors.Errorf("metadatas (%d) must match the number of ids (%d)", len(c.Metadatas), len(c.Ids))
	}

	if len(c.Records) > 0 {
		for _, record := range c.Records {
			err := record.Validate()
			if err != nil {
				return errors.Wrap(err, "record validation failed")
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

func (c *CollectionAddOp) MarshalJSON() ([]byte, error) {
	type Alias CollectionAddOp
	return json.Marshal(struct{ *Alias }{Alias: (*Alias)(c)})
}

func (c *CollectionAddOp) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, c)
}

func (c *CollectionAddOp) Resource() Resource {
	return ResourceCollection
}

func (c *CollectionAddOp) Operation() OperationType {
	return OperationCreate
}

type CollectionAddOption func(update *CollectionAddOp) error

func WithTexts(documents ...string) CollectionAddOption {
	return func(update *CollectionAddOp) error {
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

func WithMetadatas(metadatas ...DocumentMetadata) CollectionAddOption {
	return func(update *CollectionAddOp) error {
		update.Metadatas = metadatas
		return nil
	}
}

func WithIDs(ids ...DocumentID) CollectionAddOption {
	return func(update *CollectionAddOp) error {
		for _, id := range ids {
			update.Ids = append(update.Ids, DocumentID(id))
		}
		return nil
	}
}

func WithIDGenerator(idGenerator IDGenerator) CollectionAddOption {
	return func(update *CollectionAddOp) error {
		update.IDGenerator = idGenerator
		return nil
	}
}

func WithEmbeddings(embeddings ...embeddings.Embedding) CollectionAddOption {
	return func(update *CollectionAddOp) error {
		update.Embeddings = embeddings
		return nil
	}
}

// Update

type CollectionUpdateOp struct {
	Ids        []DocumentID           `json:"ids"`
	Documents  []Document             `json:"documents,omitempty"`
	Metadatas  []DocumentMetadata     `json:"metadatas,omitempty"`
	Embeddings []embeddings.Embedding `json:"embeddings"`
	Records    []Record               `json:"-"`
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

func (c *CollectionUpdateOp) EmbedData(ctx context.Context, ef embeddings.EmbeddingFunction) error {
	// invariants:
	// documents only - we embed
	// documents + embeddings - we skip
	// embeddings only - we skip
	if len(c.Documents) > 0 && len(c.Embeddings) == 0 {
		if ef == nil {
			return errors.New("embedding function is required")
		}
		texts := make([]string, len(c.Documents))
		for i, doc := range c.Documents {
			texts[i] = doc.ContentString()
		}
		embeddings, err := ef.EmbedDocuments(ctx, texts)
		if err != nil {
			return errors.Wrap(err, "embedding failed")
		}
		for i, embedding := range embeddings {
			if i >= len(c.Embeddings) {
				c.Embeddings = append(c.Embeddings, embedding)
			} else {
				c.Embeddings[i] = embedding
			}
		}
	}
	return nil
}

func (c *CollectionUpdateOp) PrepareAndValidate() error {
	// invariants
	// - at least one ID or one record is required
	// - if IDs are provided, they must be unique
	// - if IDs are provided, the number of documents or embeddings must match the number of IDs
	// - if IDs are provided, if metadatas are also provided they must match the number of IDs

	if len(c.Ids) == 0 && len(c.Records) == 0 {
		return errors.New("at least one ID or record is required.") // TODO add link to docs
	}

	// if IDs are provided, they must be unique
	idSet := make(map[DocumentID]struct{})
	for _, id := range c.Ids {
		if _, exists := idSet[id]; exists {
			return errors.Errorf("duplicate id found: %s", id)
		}
		idSet[id] = struct{}{}
	}

	// if IDs are provided, the number of documents or embeddings must match the number of IDs
	if len(c.Documents) > 0 && len(c.Ids) != len(c.Documents) {
		return errors.Errorf("documents (%d) must match the number of ids (%d)", len(c.Documents), len(c.Ids))
	}

	if len(c.Embeddings) > 0 && len(c.Ids) != len(c.Embeddings) {
		return errors.Errorf("embeddings (%d) must match the number of ids (%d)", len(c.Embeddings), len(c.Ids))
	}

	// if IDs are provided, if metadatas are also provided they must match the number of IDs

	if len(c.Metadatas) > 0 && len(c.Ids) != len(c.Metadatas) {
		return errors.Errorf("metadatas (%d) must match the number of ids (%d)", len(c.Metadatas), len(c.Ids))
	}

	if len(c.Records) > 0 {
		for _, record := range c.Records {
			err := record.Validate()
			if err != nil {
				return errors.Wrap(err, "record validation failed")
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
	return OperationUpdate
}

type CollectionUpdateOption func(update *CollectionUpdateOp) error

func WithTextsUpdate(documents ...string) CollectionUpdateOption {
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

func WithMetadatasUpdate(metadatas ...DocumentMetadata) CollectionUpdateOption {
	return func(update *CollectionUpdateOp) error {
		update.Metadatas = metadatas
		return nil
	}
}

func WithIDsUpdate(ids ...DocumentID) CollectionUpdateOption {
	return func(update *CollectionUpdateOp) error {
		for _, id := range ids {
			update.Ids = append(update.Ids, DocumentID(id))
		}
		return nil
	}
}

func WithEmbeddingsUpdate(embeddings ...embeddings.Embedding) CollectionUpdateOption {
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

func (c *CollectionDeleteOp) PrepareAndValidate() error {
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
		for _, id := range ids {
			delete.Ids = append(delete.Ids, DocumentID(id))
		}
		return nil
	}
}

type CollectionConfiguration interface {
	GetRaw(key string) (interface{}, bool)
}
