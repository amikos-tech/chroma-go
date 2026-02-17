package v2

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

/*
Collection represents a Chroma vector collection.

A Collection stores documents with their embeddings, metadata, and optional URIs.
It provides methods for CRUD operations and semantic search.

# Creating a Collection

Use [Client.CreateCollection] or [Client.GetOrCreateCollection]:

	client, _ := NewClient()
	collection, _ := client.CreateCollection(ctx, "my-collection",
	    WithEmbeddingFunction(ef),
	    WithCollectionMetadata(map[string]any{"description": "My docs"}),
	)

# Adding Documents

Use [Collection.Add] with unified options:

	err := collection.Add(ctx,
	    WithIDs("doc1", "doc2"),
	    WithTexts("First document", "Second document"),
	    WithMetadatas(meta1, meta2),
	)

# Querying Documents

Use [Collection.Query] for semantic search or [Collection.Get] for direct retrieval:

	// Semantic search
	results, _ := collection.Query(ctx,
	    WithQueryTexts("machine learning"),
	    WithNResults(10),
	)

	// Direct retrieval
	results, _ := collection.Get(ctx,
	    WithIDs("doc1", "doc2"),
	)

# Advanced Search

Use [Collection.Search] for more control over ranking and filtering:

	results, _ := collection.Search(ctx,
	    NewSearchRequest(
	        WithKnnRank(KnnQueryText("query")),
	        WithFilter(EqString(K("status"), "published")),
	        WithPage(PageLimit(20)),
	    ),
	)
*/
type Collection interface {
	// Name returns the name of the collection.
	Name() string

	// ID returns the unique identifier of the collection.
	ID() string

	// Tenant returns the tenant that owns this collection.
	Tenant() Tenant

	// Database returns the database containing this collection.
	Database() Database

	// Metadata returns the collection's metadata.
	Metadata() CollectionMetadata

	// Dimension returns the dimensionality of embeddings in this collection.
	// Returns 0 if not yet determined (no documents added).
	Dimension() int

	// Configuration returns the collection's configuration settings.
	Configuration() CollectionConfiguration

	// Schema returns the collection's schema definition, if any.
	Schema() *Schema

	// Add inserts new documents into the collection.
	//
	// Documents can be provided as text (automatically embedded) or with
	// pre-computed embeddings. IDs must be unique within the collection.
	//
	// Options: [WithIDs], [WithTexts], [WithEmbeddings], [WithMetadatas], [WithIDGenerator]
	//
	//	err := collection.Add(ctx,
	//	    WithIDs("doc1", "doc2"),
	//	    WithTexts("First document", "Second document"),
	//	)
	Add(ctx context.Context, opts ...CollectionAddOption) error

	// Upsert inserts new documents or updates existing ones.
	//
	// If a document with the given ID exists, it is updated. Otherwise, a new
	// document is created. Uses the same options as [Collection.Add].
	//
	//	err := collection.Upsert(ctx,
	//	    WithIDs("doc1"),
	//	    WithTexts("New or updated content"),
	//	)
	Upsert(ctx context.Context, opts ...CollectionAddOption) error

	// Update modifies existing documents in the collection.
	//
	// Only the provided fields are updated. Documents must exist.
	//
	// Options: [WithIDs], [WithTexts], [WithEmbeddings], [WithMetadatas]
	//
	//	err := collection.Update(ctx,
	//	    WithIDs("doc1"),
	//	    WithTexts("Updated content"),
	//	)
	Update(ctx context.Context, opts ...CollectionUpdateOption) error

	// Delete removes documents from the collection.
	//
	// At least one filter must be provided: IDs, Where, or WhereDocument.
	//
	// Options: [WithIDs], [WithWhere], [WithWhereDocument]
	//
	//	// Delete by ID
	//	err := collection.Delete(ctx, WithIDs("doc1", "doc2"))
	//
	//	// Delete by metadata filter
	//	err := collection.Delete(ctx, WithWhere(EqString("status", "archived")))
	Delete(ctx context.Context, opts ...CollectionDeleteOption) error

	// Count returns the total number of documents in the collection.
	Count(ctx context.Context) (int, error)

	// ModifyName changes the collection's name.
	ModifyName(ctx context.Context, newName string) error

	// ModifyMetadata updates the collection's metadata.
	ModifyMetadata(ctx context.Context, newMetadata CollectionMetadata) error

	// ModifyConfiguration updates the collection's configuration.
	// Note: Not all configuration changes may be supported.
	ModifyConfiguration(ctx context.Context, newConfig CollectionConfiguration) error

	// Get retrieves documents from the collection by ID or filter.
	//
	// Returns documents matching the specified criteria. If no filters are
	// provided, returns all documents (subject to limit/offset).
	//
	// Options: [WithIDs], [WithWhere], [WithWhereDocument], [WithInclude], [WithLimit], [WithOffset]
	//
	//	// Get by IDs
	//	results, _ := collection.Get(ctx, WithIDs("doc1", "doc2"))
	//
	//	// Get with pagination
	//	results, _ := collection.Get(ctx, WithLimit(100), WithOffset(200))
	//
	//	// Get with filter
	//	results, _ := collection.Get(ctx,
	//	    WithWhere(EqString("status", "published")),
	//	    WithInclude(IncludeDocuments, IncludeMetadatas),
	//	)
	Get(ctx context.Context, opts ...CollectionGetOption) (GetResult, error)

	// Query performs semantic search using text or embeddings.
	//
	// Finds documents most similar to the query. Use [WithQueryTexts] for text
	// queries (automatically embedded) or [WithQueryEmbeddings] for pre-computed
	// embeddings.
	//
	// Options: [WithQueryTexts], [WithQueryEmbeddings], [WithNResults], [WithWhere],
	// [WithWhereDocument], [WithInclude], [WithIDs]
	//
	//	results, _ := collection.Query(ctx,
	//	    WithQueryTexts("machine learning"),
	//	    WithNResults(10),
	//	    WithWhere(EqString("category", "tech")),
	//	)
	Query(ctx context.Context, opts ...CollectionQueryOption) (QueryResult, error)

	// Search performs advanced search with ranking and filtering.
	//
	// Provides more control than Query, including custom ranking expressions,
	// pagination, field selection, and grouping.
	//
	// Use [NewSearchRequest] to create search requests with options like
	// [WithKnnRank], [WithFilter], [WithPage], [WithSelect], and [WithGroupBy].
	//
	//	results, _ := collection.Search(ctx,
	//	    NewSearchRequest(
	//	        WithKnnRank(KnnQueryText("machine learning")),
	//	        WithFilter(EqString(K("status"), "published")),
	//	        WithPage(PageLimit(10)),
	//	        WithSelect(KDocument, KScore),
	//	    ),
	//	)
	Search(ctx context.Context, opts ...SearchCollectionOption) (SearchResult, error)

	// Fork creates a copy of this collection with a new name.
	// The new collection contains all documents from the original.
	Fork(ctx context.Context, newName string) (Collection, error)

	// IndexingStatus returns the current indexing progress for this collection.
	// Requires Chroma server version >= 1.4.1.
	//
	// Use this to monitor background indexing after adding large amounts of data.
	IndexingStatus(ctx context.Context) (*IndexingStatus, error)

	// Close releases any resources held by the collection.
	// The collection should not be used after calling Close.
	Close() error
}

// IndexingStatus represents the current indexing state of a collection.
//
// After adding documents, Chroma indexes them in the background. Use this
// to monitor indexing progress, especially after bulk inserts.
//
//	status, err := collection.IndexingStatus(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Indexing progress: %.1f%%\n", status.OpIndexingProgress*100)
type IndexingStatus struct {
	// NumIndexedOps is the number of operations that have been indexed.
	NumIndexedOps uint64 `json:"num_indexed_ops"`

	// NumUnindexedOps is the number of operations waiting to be indexed.
	NumUnindexedOps uint64 `json:"num_unindexed_ops"`

	// TotalOps is the total number of operations (indexed + unindexed).
	TotalOps uint64 `json:"total_ops"`

	// OpIndexingProgress is the indexing progress as a fraction (0.0 to 1.0).
	// A value of 1.0 means all operations have been indexed.
	OpIndexingProgress float64 `json:"op_indexing_progress"`
}

// CollectionOp is the interface for all collection operations.
// This is an internal interface used by the HTTP client implementation.
type CollectionOp interface {
	// PrepareAndValidate validates the operation before sending.
	PrepareAndValidate() error

	// EmbedData embeds text data using the provided embedding function.
	EmbedData(ctx context.Context, ef embeddings.EmbeddingFunction) error

	// MarshalJSON serializes the operation to JSON.
	MarshalJSON() ([]byte, error)

	// UnmarshalJSON deserializes the operation from JSON.
	UnmarshalJSON(b []byte) error
}

// FilterOp provides metadata and document content filtering capabilities.
// Embedded in operations that support Where and WhereDocument filters.
type FilterOp struct {
	// Where filters by metadata field values.
	Where WhereFilter `json:"where,omitempty"`

	// WhereDocument filters by document text content.
	WhereDocument WhereDocumentFilter `json:"where_document,omitempty"`
}

// SetWhere sets the metadata filter.
func (f *FilterOp) SetWhere(where WhereFilter) {
	f.Where = where
}

// SetWhereDocument sets the document content filter.
func (f *FilterOp) SetWhereDocument(where WhereDocumentFilter) {
	f.WhereDocument = where
}

// FilterIDOp provides ID-based filtering capabilities.
// Embedded in operations that support filtering by document IDs.
type FilterIDOp struct {
	// Ids contains the document IDs to filter by.
	Ids []DocumentID `json:"ids,omitempty"`
}

// AppendIDs adds document IDs to the filter.
func (f *FilterIDOp) AppendIDs(ids ...DocumentID) {
	f.Ids = append(f.Ids, ids...)
}

// FilterTextsOp holds query texts for semantic search.
// The texts are embedded before search.
type FilterTextsOp struct {
	QueryTexts []string `json:"-"`
}

// FilterEmbeddingsOp holds pre-computed embeddings for semantic search.
type FilterEmbeddingsOp struct {
	QueryEmbeddings []embeddings.Embedding `json:"query_embeddings"`
}

// ProjectOp specifies which fields to include in results.
type ProjectOp struct {
	Include []Include `json:"include,omitempty"`
}

// LimitAndOffsetOp provides pagination for Get operations.
type LimitAndOffsetOp struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// LimitResultOp specifies the number of results for Query operations.
type LimitResultOp struct {
	NResults int `json:"n_results"`
}

// SortOp specifies result ordering (not yet supported).
type SortOp struct {
	Sort string `json:"sort,omitempty"`
}

// CollectionGetOption is an alias for [GetOption] for backward compatibility.
type CollectionGetOption = GetOption

// CollectionGetOp represents a Get operation on a collection.
//
// Use [NewCollectionGetOp] to create instances, or pass options directly
// to [Collection.Get].
//
//	// Using Collection.Get (recommended)
//	results, err := collection.Get(ctx,
//	    WithIDs("doc1", "doc2"),
//	    WithInclude(IncludeDocuments),
//	)
//
//	// Using NewCollectionGetOp (advanced)
//	op, err := NewCollectionGetOp(
//	    WithIDs("doc1", "doc2"),
//	    WithInclude(IncludeDocuments),
//	)
type CollectionGetOp struct {
	FilterOp          // Where and WhereDocument filters
	FilterIDOp        // ID filter
	ProjectOp         // Field projection (Include)
	LimitAndOffsetOp  // Pagination
	SortOp            // Ordering (not yet supported)
	ResourceOperation `json:"-"`
}

// NewCollectionGetOp creates a new Get operation with the given options.
//
// This is primarily for advanced use cases. Most users should use
// [Collection.Get] directly.
func NewCollectionGetOp(opts ...GetOption) (*CollectionGetOp, error) {
	get := &CollectionGetOp{
		ProjectOp: ProjectOp{Include: []Include{IncludeDocuments, IncludeMetadatas}},
	}
	for _, opt := range opts {
		err := opt.ApplyToGet(get)
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

// Deprecated: Use [WithIDs] instead.
func WithIDsGet(ids ...DocumentID) GetOption {
	return WithIDs(ids...)
}

// Deprecated: Use [WithWhere] instead.
func WithWhereGet(where WhereFilter) GetOption {
	return WithWhere(where)
}

// Deprecated: Use [WithWhereDocument] instead.
func WithWhereDocumentGet(whereDocument WhereDocumentFilter) GetOption {
	return WithWhereDocument(whereDocument)
}

// Deprecated: Use [WithInclude] instead.
func WithIncludeGet(include ...Include) GetOption {
	return WithInclude(include...)
}

// Deprecated: Use [WithLimit] instead.
func WithLimitGet(limit int) GetOption {
	return WithLimit(limit)
}

// Deprecated: Use [WithOffset] instead.
func WithOffsetGet(offset int) GetOption {
	return WithOffset(offset)
}

// CollectionQueryOp represents a semantic search Query operation.
//
// Use [Collection.Query] with options like [WithQueryTexts] or [WithQueryEmbeddings].
//
//	results, err := collection.Query(ctx,
//	    WithQueryTexts("machine learning"),
//	    WithNResults(10),
//	    WithWhere(EqString("status", "published")),
//	)
type CollectionQueryOp struct {
	FilterOp           // Where and WhereDocument filters
	FilterEmbeddingsOp // Pre-computed query embeddings
	FilterTextsOp      // Query texts to be embedded
	LimitResultOp      // Number of results per query
	ProjectOp          // Field projection (Include)
	FilterIDOp         // Limit search to specific IDs
}

// NewCollectionQueryOp creates a new Query operation with the given options.
//
// Default NResults is 10. This is primarily for advanced use cases.
// Most users should use [Collection.Query] directly.
func NewCollectionQueryOp(opts ...QueryOption) (*CollectionQueryOp, error) {
	query := &CollectionQueryOp{
		LimitResultOp: LimitResultOp{NResults: 10},
	}
	for _, opt := range opts {
		err := opt.ApplyToQuery(query)
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

// CollectionQueryOption is an alias for QueryOption for backward compatibility.
type CollectionQueryOption = QueryOption

// Deprecated: Use [WithWhere] instead.
func WithWhereQuery(where WhereFilter) QueryOption {
	return WithWhere(where)
}

// Deprecated: Use [WithWhereDocument] instead.
func WithWhereDocumentQuery(whereDocument WhereDocumentFilter) QueryOption {
	return WithWhereDocument(whereDocument)
}

// queryEmbeddingsOption implements query embedding input for Query operations.
type queryEmbeddingsOption struct {
	embeddings []embeddings.Embedding
}

// WithQueryEmbeddings sets the query embeddings. Works with Query.
func WithQueryEmbeddings(queryEmbeddings ...embeddings.Embedding) *queryEmbeddingsOption {
	return &queryEmbeddingsOption{embeddings: queryEmbeddings}
}

func (o *queryEmbeddingsOption) ApplyToQuery(op *CollectionQueryOp) error {
	if len(o.embeddings) == 0 {
		return errors.New("at least one query embedding is required")
	}
	op.QueryEmbeddings = o.embeddings
	return nil
}

// Deprecated: Use [WithInclude] instead.
func WithIncludeQuery(include ...Include) QueryOption {
	return WithInclude(include...)
}

// Deprecated: Use [WithIDs] instead.
func WithIDsQuery(ids ...DocumentID) QueryOption {
	return WithIDs(ids...)
}

// CollectionAddOp represents an Add or Upsert operation.
//
// Use [Collection.Add] or [Collection.Upsert] with options:
//
//	err := collection.Add(ctx,
//	    WithIDs("doc1", "doc2"),
//	    WithTexts("First document", "Second document"),
//	    WithMetadatas(meta1, meta2),
//	)
type CollectionAddOp struct {
	// Ids are the unique identifiers for each document.
	Ids []DocumentID `json:"ids"`

	// Documents contain the text content to be stored and embedded.
	Documents []Document `json:"documents,omitempty"`

	// Metadatas contain key-value metadata for each document.
	Metadatas []DocumentMetadata `json:"metadatas,omitempty"`

	// Embeddings are the vector representations of the documents.
	// If not provided, they are computed from Documents using the embedding function.
	Embeddings []any `json:"embeddings"`

	// Records is an alternative to separate Ids/Documents/Metadatas/Embeddings.
	Records []Record `json:"-"`

	// IDGenerator automatically generates IDs if Ids is empty.
	IDGenerator IDGenerator `json:"-"`
}

// NewCollectionAddOp creates a new Add operation with the given options.
//
// This is primarily for advanced use cases. Most users should use
// [Collection.Add] or [Collection.Upsert] directly.
func NewCollectionAddOp(opts ...AddOption) (*CollectionAddOp, error) {
	add := &CollectionAddOp{}
	for _, opt := range opts {
		err := opt.ApplyToAdd(add)
		if err != nil {
			return nil, err
		}
	}
	return add, nil
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

	if len(c.Metadatas) > 0 {
		if err := validateDocumentMetadatas(c.Metadatas); err != nil {
			return err
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

// CollectionAddOption is an alias for [AddOption] for backward compatibility.
type CollectionAddOption = AddOption

// metadatasOption implements metadata input for Add and Update operations.
// Use [WithMetadatas] to create this option.
type metadatasOption struct {
	metadatas []DocumentMetadata
}

// WithMetadatas sets document metadata for [Collection.Add], [Collection.Upsert],
// and [Collection.Update] operations.
//
// The number of metadatas must match the number of IDs provided.
// Each metadata is a map of string keys to values (string, int, float, bool,
// or arrays of these types).
//
// # Add Example
//
//	meta1 := map[string]any{"author": "Alice", "year": 2024}
//	meta2 := map[string]any{"author": "Bob", "year": 2023}
//
//	err := collection.Add(ctx,
//	    WithIDs("doc1", "doc2"),
//	    WithTexts("First document", "Second document"),
//	    WithMetadatas(meta1, meta2),
//	)
//
// # Update Example
//
//	err := collection.Update(ctx,
//	    WithIDs("doc1"),
//	    WithMetadatas(map[string]any{"status": "reviewed"}),
//	)
//
// Note: At least one metadata must be provided.
func WithMetadatas(metadatas ...DocumentMetadata) *metadatasOption {
	return &metadatasOption{metadatas: metadatas}
}

func (o *metadatasOption) ApplyToAdd(op *CollectionAddOp) error {
	if len(o.metadatas) == 0 {
		return ErrNoMetadatas
	}
	op.Metadatas = o.metadatas
	return nil
}

func (o *metadatasOption) ApplyToUpdate(op *CollectionUpdateOp) error {
	if len(o.metadatas) == 0 {
		return ErrNoMetadatas
	}
	op.Metadatas = o.metadatas
	return nil
}

// idGeneratorOption implements ID generator for Add operations.
// Use [WithIDGenerator] to create this option.
type idGeneratorOption struct {
	generator IDGenerator
}

// WithIDGenerator sets an automatic ID generator for [Collection.Add].
//
// When an ID generator is set, IDs are automatically generated for each
// document based on the generator's strategy. This allows adding documents
// without explicitly providing IDs.
//
// Available generators:
//   - [NewULIDGenerator] - generates unique, lexicographically sortable IDs
//   - [NewUUIDGenerator] - generates random UUIDs
//   - [NewSHA256Generator] - generates content-based IDs (deterministic)
//
// # Example with ULID Generator
//
//	err := collection.Add(ctx,
//	    WithTexts("First document", "Second document"),
//	    WithIDGenerator(NewULIDGenerator()),
//	)
//
// # Example with Content-Based IDs
//
//	// Same content always produces the same ID
//	err := collection.Add(ctx,
//	    WithTexts("Document content"),
//	    WithIDGenerator(NewSHA256Generator()),
//	)
func WithIDGenerator(idGenerator IDGenerator) *idGeneratorOption {
	return &idGeneratorOption{generator: idGenerator}
}

func (o *idGeneratorOption) ApplyToAdd(op *CollectionAddOp) error {
	op.IDGenerator = o.generator
	return nil
}

// embeddingsOption implements embedding input for Add and Update operations.
// Use [WithEmbeddings] to create this option.
type embeddingsOption struct {
	embeddings []embeddings.Embedding
}

// WithEmbeddings sets pre-computed embeddings for [Collection.Add], [Collection.Upsert],
// and [Collection.Update] operations.
//
// Use this when you have already computed embeddings externally and don't want
// the collection's embedding function to re-embed the documents.
//
// The number of embeddings must match the number of IDs provided.
// Each embedding is a slice of float32 values.
//
// # Example
//
//	// Pre-computed 384-dimensional embeddings
//	emb1 := []float32{0.1, 0.2, ...}
//	emb2 := []float32{0.3, 0.4, ...}
//
//	err := collection.Add(ctx,
//	    WithIDs("doc1", "doc2"),
//	    WithEmbeddings(emb1, emb2),
//	    WithMetadatas(meta1, meta2),
//	)
func WithEmbeddings(embs ...embeddings.Embedding) *embeddingsOption {
	return &embeddingsOption{embeddings: embs}
}

func (o *embeddingsOption) ApplyToAdd(op *CollectionAddOp) error {
	if len(o.embeddings) == 0 {
		return errors.New("at least one embedding is required")
	}
	embds := make([]any, 0, len(o.embeddings))
	for _, e := range o.embeddings {
		embds = append(embds, e)
	}
	op.Embeddings = embds
	return nil
}

func (o *embeddingsOption) ApplyToUpdate(op *CollectionUpdateOp) error {
	if len(o.embeddings) == 0 {
		return errors.New("at least one embedding is required")
	}
	embds := make([]any, 0, len(o.embeddings))
	for _, e := range o.embeddings {
		embds = append(embds, e)
	}
	op.Embeddings = embds
	return nil
}

// CollectionUpdateOp represents an Update operation on existing documents.
//
// Use [Collection.Update] with options to modify existing documents:
//
//	err := collection.Update(ctx,
//	    WithIDs("doc1"),
//	    WithTexts("Updated document content"),
//	    WithMetadatas(updatedMeta),
//	)
type CollectionUpdateOp struct {
	// Ids identifies the documents to update (required).
	Ids []DocumentID `json:"ids"`

	// Documents contain updated text content.
	Documents []Document `json:"documents,omitempty"`

	// Metadatas contain updated metadata values.
	Metadatas []DocumentMetadata `json:"metadatas,omitempty"`

	// Embeddings contain updated vector representations.
	Embeddings []any `json:"embeddings"`

	// Records is an alternative to separate fields.
	Records []Record `json:"-"`
}

// NewCollectionUpdateOp creates a new Update operation with the given options.
//
// This is primarily for advanced use cases. Most users should use
// [Collection.Update] directly.
func NewCollectionUpdateOp(opts ...UpdateOption) (*CollectionUpdateOp, error) {
	update := &CollectionUpdateOp{}
	for _, opt := range opts {
		err := opt.ApplyToUpdate(update)
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

	if len(c.Metadatas) > 0 {
		if err := validateDocumentMetadatas(c.Metadatas); err != nil {
			return err
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

// CollectionUpdateOption is an alias for [UpdateOption] for backward compatibility.
type CollectionUpdateOption = UpdateOption

// Deprecated: Use [WithTexts] instead.
func WithTextsUpdate(documents ...string) UpdateOption {
	return WithTexts(documents...)
}

// Deprecated: Use [WithMetadatas] instead.
func WithMetadatasUpdate(metadatas ...DocumentMetadata) UpdateOption {
	return WithMetadatas(metadatas...)
}

// Deprecated: Use [WithIDs] instead.
func WithIDsUpdate(ids ...DocumentID) UpdateOption {
	return WithIDs(ids...)
}

// Deprecated: Use [WithEmbeddings] instead.
func WithEmbeddingsUpdate(embs ...embeddings.Embedding) UpdateOption {
	return WithEmbeddings(embs...)
}

// CollectionDeleteOp represents a Delete operation on a collection.
//
// At least one filter must be provided: IDs, Where, or WhereDocument.
//
// Use [Collection.Delete] with options:
//
//	// Delete by ID
//	err := collection.Delete(ctx, WithIDs("doc1", "doc2"))
//
//	// Delete by metadata filter
//	err := collection.Delete(ctx, WithWhere(EqString("status", "archived")))
//
//	// Delete by document content
//	err := collection.Delete(ctx, WithWhereDocument(Contains("DEPRECATED")))
type CollectionDeleteOp struct {
	FilterOp   // Where and WhereDocument filters
	FilterIDOp // ID filter
}

// NewCollectionDeleteOp creates a new Delete operation with the given options.
//
// This is primarily for advanced use cases. Most users should use
// [Collection.Delete] directly.
func NewCollectionDeleteOp(opts ...DeleteOption) (*CollectionDeleteOp, error) {
	del := &CollectionDeleteOp{}
	for _, opt := range opts {
		err := opt.ApplyToDelete(del)
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

// CollectionDeleteOption is an alias for [DeleteOption] for backward compatibility.
type CollectionDeleteOption = DeleteOption

// Deprecated: Use [WithWhere] instead.
func WithWhereDelete(where WhereFilter) DeleteOption {
	return WithWhere(where)
}

// Deprecated: Use [WithWhereDocument] instead.
func WithWhereDocumentDelete(whereDocument WhereDocumentFilter) DeleteOption {
	return WithWhereDocument(whereDocument)
}

// Deprecated: Use [WithIDs] instead.
func WithIDsDelete(ids ...DocumentID) DeleteOption {
	return WithIDs(ids...)
}

// CollectionConfiguration provides access to collection configuration settings.
type CollectionConfiguration interface {
	// GetRaw returns a configuration value by key.
	// Returns the value and true if found, or nil and false if not found.
	GetRaw(key string) (any, bool)
}
