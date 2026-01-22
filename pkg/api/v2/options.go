package v2

import "errors"

/*
Unified Options API

This file defines the unified options pattern for Chroma collection operations.
Options are designed to work across multiple operations where semantically appropriate,
reducing API surface and improving discoverability.

# Option Compatibility Matrix

The following table shows which options work with which operations:

	Option              | Get | Query | Delete | Add | Update | Search
	--------------------|-----|-------|--------|-----|--------|-------
	WithIDs             |  ✓  |   ✓   |   ✓    |  ✓  |   ✓    |   ✓
	WithWhere           |  ✓  |   ✓   |   ✓    |     |        |
	WithWhereDocument   |  ✓  |   ✓   |   ✓    |     |        |
	WithInclude         |  ✓  |   ✓   |        |     |        |
	WithLimit           |  ✓  |       |        |     |        |
	WithOffset          |  ✓  |       |        |     |        |
	WithNResults        |     |   ✓   |        |     |        |
	WithQueryTexts      |     |   ✓   |        |     |        |
	WithQueryEmbeddings |     |   ✓   |        |     |        |
	WithTexts           |     |       |        |  ✓  |   ✓    |
	WithEmbeddings      |     |       |        |  ✓  |   ✓    |
	WithMetadatas       |     |       |        |  ✓  |   ✓    |
	WithIDGenerator     |     |       |        |  ✓  |        |
	WithSearchWhere     |     |       |        |     |        |   ✓
	WithFilter          |     |       |        |     |        |   ✓
	WithPage            |     |       |        |     |        |   ✓
	WithSelect          |     |       |        |     |        |   ✓
	WithRank            |     |       |        |     |        |   ✓
	WithGroupBy         |     |       |        |     |        |   ✓

# Basic Usage

	// Get documents by ID
	results, err := collection.Get(ctx, WithIDs("id1", "id2"))

	// Query with text and metadata filter
	results, err := collection.Query(ctx,
	    WithQueryTexts("machine learning"),
	    WithWhere(EqString("status", "published")),
	    WithNResults(10),
	)

	// Add documents
	err := collection.Add(ctx,
	    WithIDs("doc1", "doc2"),
	    WithTexts("First document", "Second document"),
	    WithMetadatas(meta1, meta2),
	)

	// Delete by filter
	err := collection.Delete(ctx,
	    WithWhere(EqString("status", "archived")),
	)

	// Search with ranking
	results, err := collection.Search(ctx,
	    NewSearchRequest(
	        WithKnnRank(KnnQueryText("query")),
	        WithFilter(EqString(K("category"), "tech")),
	        WithPage(PageLimit(20)),
	    ),
	)
*/

// GetOption configures a [Collection.Get] operation.
// Implementations include [WithIDs], [WithWhere], [WithWhereDocument],
// [WithInclude], [WithLimit], and [WithOffset].
type GetOption interface {
	ApplyToGet(*CollectionGetOp) error
}

// QueryOption configures a [Collection.Query] operation.
// Implementations include [WithIDs], [WithWhere], [WithWhereDocument],
// [WithInclude], [WithNResults], [WithQueryTexts], and [WithQueryEmbeddings].
type QueryOption interface {
	ApplyToQuery(*CollectionQueryOp) error
}

// DeleteOption configures a [Collection.Delete] operation.
// Implementations include [WithIDs], [WithWhere], and [WithWhereDocument].
// At least one filter option must be provided.
type DeleteOption interface {
	ApplyToDelete(*CollectionDeleteOp) error
}

// AddOption configures a [Collection.Add] or [Collection.Upsert] operation.
// Implementations include [WithIDs], [WithTexts], [WithEmbeddings],
// [WithMetadatas], and [WithIDGenerator].
type AddOption interface {
	ApplyToAdd(*CollectionAddOp) error
}

// UpdateOption configures a [Collection.Update] operation.
// Implementations include [WithIDs], [WithTexts], [WithEmbeddings], and [WithMetadatas].
type UpdateOption interface {
	ApplyToUpdate(*CollectionUpdateOp) error
}

// SearchRequestOption configures a [SearchRequest] for [Collection.Search].
// Implementations include [WithIDs], [WithSearchWhere], [WithFilter],
// [WithPage], [WithSelect], [WithSelectAll], [WithRank], [WithKnnRank], and [WithGroupBy].
type SearchRequestOption interface {
	ApplyToSearchRequest(*SearchRequest) error
}

// GetOptionFunc wraps a function as a [GetOption].
// Use this to create custom Get options without defining a new type.
//
//	customOpt := GetOptionFunc(func(op *CollectionGetOp) error {
//	    op.Limit = 50
//	    return nil
//	})
type GetOptionFunc func(*CollectionGetOp) error

// ApplyToGet implements [GetOption].
func (f GetOptionFunc) ApplyToGet(op *CollectionGetOp) error { return f(op) }

// QueryOptionFunc wraps a function as a [QueryOption].
// Use this to create custom Query options without defining a new type.
type QueryOptionFunc func(*CollectionQueryOp) error

// ApplyToQuery implements [QueryOption].
func (f QueryOptionFunc) ApplyToQuery(op *CollectionQueryOp) error { return f(op) }

// DeleteOptionFunc wraps a function as a [DeleteOption].
// Use this to create custom Delete options without defining a new type.
type DeleteOptionFunc func(*CollectionDeleteOp) error

// ApplyToDelete implements [DeleteOption].
func (f DeleteOptionFunc) ApplyToDelete(op *CollectionDeleteOp) error { return f(op) }

// AddOptionFunc wraps a function as an [AddOption].
// Use this to create custom Add options without defining a new type.
type AddOptionFunc func(*CollectionAddOp) error

// ApplyToAdd implements [AddOption].
func (f AddOptionFunc) ApplyToAdd(op *CollectionAddOp) error { return f(op) }

// UpdateOptionFunc wraps a function as an [UpdateOption].
// Use this to create custom Update options without defining a new type.
type UpdateOptionFunc func(*CollectionUpdateOp) error

// ApplyToUpdate implements [UpdateOption].
func (f UpdateOptionFunc) ApplyToUpdate(op *CollectionUpdateOp) error { return f(op) }

// SearchRequestOptionFunc wraps a function as a [SearchRequestOption].
// Use this to create custom Search options without defining a new type.
type SearchRequestOptionFunc func(*SearchRequest) error

// ApplyToSearchRequest implements [SearchRequestOption].
func (f SearchRequestOptionFunc) ApplyToSearchRequest(req *SearchRequest) error { return f(req) }

// idsOption implements ID filtering for all operations.
// Use [WithIDs] to create this option.
type idsOption struct {
	ids []DocumentID
}

// WithIDs specifies document IDs for filtering or identification.
//
// This is a unified option that works with multiple operations:
//   - [Collection.Get]: Retrieve specific documents by ID
//   - [Collection.Query]: Limit semantic search to specific documents
//   - [Collection.Delete]: Delete specific documents by ID
//   - [Collection.Add]: Specify IDs for new documents
//   - [Collection.Update]: Identify documents to update
//   - [Collection.Search]: Filter search results to specific IDs
//
// # Get Example
//
//	results, err := collection.Get(ctx, WithIDs("doc1", "doc2", "doc3"))
//
// # Query Example
//
//	results, err := collection.Query(ctx,
//	    WithQueryTexts("machine learning"),
//	    WithIDs("doc1", "doc2"),  // Only search within these documents
//	)
//
// # Delete Example
//
//	err := collection.Delete(ctx, WithIDs("doc1", "doc2"))
//
// # Add Example
//
//	err := collection.Add(ctx,
//	    WithIDs("doc1", "doc2"),
//	    WithTexts("First document", "Second document"),
//	)
func WithIDs(ids ...DocumentID) *idsOption {
	return &idsOption{ids: ids}
}

func (o *idsOption) ApplyToGet(op *CollectionGetOp) error {
	if len(o.ids) == 0 {
		return errors.New("at least one id is required")
	}
	op.Ids = append(op.Ids, o.ids...)
	return nil
}

func (o *idsOption) ApplyToQuery(op *CollectionQueryOp) error {
	if len(o.ids) == 0 {
		return errors.New("at least one id is required")
	}
	op.Ids = append(op.Ids, o.ids...)
	return nil
}

func (o *idsOption) ApplyToDelete(op *CollectionDeleteOp) error {
	if len(o.ids) == 0 {
		return errors.New("at least one id is required")
	}
	op.Ids = append(op.Ids, o.ids...)
	return nil
}

func (o *idsOption) ApplyToAdd(op *CollectionAddOp) error {
	if len(o.ids) == 0 {
		return errors.New("at least one id is required")
	}
	op.Ids = append(op.Ids, o.ids...)
	return nil
}

func (o *idsOption) ApplyToUpdate(op *CollectionUpdateOp) error {
	if len(o.ids) == 0 {
		return errors.New("at least one id is required")
	}
	op.Ids = append(op.Ids, o.ids...)
	return nil
}

func (o *idsOption) ApplyToSearchRequest(req *SearchRequest) error {
	if len(o.ids) == 0 {
		return errors.New("at least one id is required")
	}
	if req.Filter == nil {
		req.Filter = &SearchFilter{}
	}
	req.Filter.IDs = append(req.Filter.IDs, o.ids...)
	return nil
}

// whereOption implements metadata filtering for Get, Query, and Delete operations.
// Use [WithWhere] to create this option.
type whereOption struct {
	where WhereFilter
}

// WithWhere filters documents by metadata field values.
//
// This is a unified option that works with:
//   - [Collection.Get]: Filter which documents to retrieve
//   - [Collection.Query]: Filter semantic search results
//   - [Collection.Delete]: Delete documents matching the filter
//
// Note: Calling WithWhere multiple times will overwrite the previous filter,
// not merge them. To combine filters, use [AndFilter] or [OrFilter].
//
// # Available Filter Functions
//
// Equality:
//   - [EqString], [EqInt], [EqFloat], [EqBool] - exact match
//   - [NeString], [NeInt], [NeFloat], [NeBool] - not equal
//
// Comparison (numeric/string):
//   - [GtInt], [GtFloat], [GtString] - greater than
//   - [GteInt], [GteFloat], [GteString] - greater than or equal
//   - [LtInt], [LtFloat], [LtString] - less than
//   - [LteInt], [LteFloat], [LteString] - less than or equal
//
// Set operations:
//   - [InString], [InInt], [InFloat] - value in set
//   - [NinString], [NinInt], [NinFloat] - value not in set
//
// Logical:
//   - [AndFilter], [OrFilter] - combine multiple filters
//
// # Get Example
//
//	results, err := collection.Get(ctx,
//	    WithWhere(EqString("status", "published")),
//	)
//
// # Query Example
//
//	results, err := collection.Query(ctx,
//	    WithQueryTexts("machine learning"),
//	    WithWhere(AndFilter(
//	        EqString("category", "tech"),
//	        GtInt("views", 1000),
//	    )),
//	)
//
// # Delete Example
//
//	err := collection.Delete(ctx,
//	    WithWhere(EqString("status", "archived")),
//	)
func WithWhere(where WhereFilter) *whereOption {
	return &whereOption{where: where}
}

func (o *whereOption) ApplyToGet(op *CollectionGetOp) error {
	if o.where != nil {
		if err := o.where.Validate(); err != nil {
			return err
		}
	}
	op.Where = o.where
	return nil
}

func (o *whereOption) ApplyToQuery(op *CollectionQueryOp) error {
	if o.where != nil {
		if err := o.where.Validate(); err != nil {
			return err
		}
	}
	op.Where = o.where
	return nil
}

func (o *whereOption) ApplyToDelete(op *CollectionDeleteOp) error {
	if o.where != nil {
		if err := o.where.Validate(); err != nil {
			return err
		}
	}
	op.Where = o.where
	return nil
}

// whereDocumentOption implements document content filtering for Get, Query, and Delete operations.
// Use [WithWhereDocument] to create this option.
type whereDocumentOption struct {
	whereDocument WhereDocumentFilter
}

// WithWhereDocument filters documents by their text content.
//
// This is a unified option that works with:
//   - [Collection.Get]: Filter which documents to retrieve
//   - [Collection.Query]: Filter semantic search results
//   - [Collection.Delete]: Delete documents matching the content filter
//
// Note: Calling WithWhereDocument multiple times will overwrite the previous filter,
// not merge them. To combine filters, use [AndDocumentFilter] or [OrDocumentFilter].
//
// # Available Filter Functions
//
//   - [Contains] - document contains the substring
//   - [NotContains] - document does not contain the substring
//   - [AndDocumentFilter] - combine multiple document filters with AND
//   - [OrDocumentFilter] - combine multiple document filters with OR
//
// # Get Example
//
//	results, err := collection.Get(ctx,
//	    WithWhereDocument(Contains("machine learning")),
//	)
//
// # Query Example
//
//	results, err := collection.Query(ctx,
//	    WithQueryTexts("AI research"),
//	    WithWhereDocument(AndDocumentFilter(
//	        Contains("neural network"),
//	        NotContains("deprecated"),
//	    )),
//	)
//
// # Delete Example
//
//	err := collection.Delete(ctx,
//	    WithWhereDocument(Contains("DRAFT:")),
//	)
func WithWhereDocument(whereDocument WhereDocumentFilter) *whereDocumentOption {
	return &whereDocumentOption{whereDocument: whereDocument}
}

func (o *whereDocumentOption) ApplyToGet(op *CollectionGetOp) error {
	if o.whereDocument != nil {
		if err := o.whereDocument.Validate(); err != nil {
			return err
		}
	}
	op.WhereDocument = o.whereDocument
	return nil
}

func (o *whereDocumentOption) ApplyToQuery(op *CollectionQueryOp) error {
	if o.whereDocument != nil {
		if err := o.whereDocument.Validate(); err != nil {
			return err
		}
	}
	op.WhereDocument = o.whereDocument
	return nil
}

func (o *whereDocumentOption) ApplyToDelete(op *CollectionDeleteOp) error {
	if o.whereDocument != nil {
		if err := o.whereDocument.Validate(); err != nil {
			return err
		}
	}
	op.WhereDocument = o.whereDocument
	return nil
}

// includeOption implements projection for Get and Query operations.
// Use [WithInclude] to create this option.
type includeOption struct {
	include []Include
}

// WithInclude specifies which fields to include in Get and Query results.
//
// This option works with:
//   - [Collection.Get]: Control which fields are returned
//   - [Collection.Query]: Control which fields are returned with search results
//
// # Available Include Constants
//
//   - [IncludeDocuments] - include document text content
//   - [IncludeMetadatas] - include document metadata
//   - [IncludeEmbeddings] - include vector embeddings
//   - [IncludeDistances] - include distance scores (Query only)
//   - [IncludeURIs] - include document URIs
//
// By default, Get returns documents and metadatas. Query returns IDs and distances.
//
// # Get Example
//
//	results, err := collection.Get(ctx,
//	    WithIDs("doc1", "doc2"),
//	    WithInclude(IncludeDocuments, IncludeMetadatas, IncludeEmbeddings),
//	)
//
// # Query Example
//
//	results, err := collection.Query(ctx,
//	    WithQueryTexts("machine learning"),
//	    WithInclude(IncludeDocuments, IncludeDistances),
//	    WithNResults(10),
//	)
func WithInclude(include ...Include) *includeOption {
	return &includeOption{include: include}
}

func (o *includeOption) ApplyToGet(op *CollectionGetOp) error {
	op.Include = o.include
	return nil
}

func (o *includeOption) ApplyToQuery(op *CollectionQueryOp) error {
	op.Include = o.include
	return nil
}

// limitOption implements limit for Get operations.
// Use [WithLimit] to create this option.
type limitOption struct {
	limit int
}

// WithLimit sets the maximum number of documents to return from [Collection.Get].
//
// Use this with [WithOffset] for pagination. The limit must be greater than 0.
//
// For [Collection.Query], use [WithNResults] instead.
// For [Collection.Search], use [WithPage] with [PageLimit] instead.
//
// # Example
//
//	// Get first 100 documents
//	results, err := collection.Get(ctx, WithLimit(100))
//
// # Pagination Example
//
//	// Get page 3 (documents 200-299)
//	results, err := collection.Get(ctx,
//	    WithLimit(100),
//	    WithOffset(200),
//	)
func WithLimit(limit int) *limitOption {
	return &limitOption{limit: limit}
}

func (o *limitOption) ApplyToGet(op *CollectionGetOp) error {
	if o.limit <= 0 {
		return ErrInvalidLimit
	}
	op.Limit = o.limit
	return nil
}

// offsetOption implements offset for Get operations.
// Use [WithOffset] to create this option.
type offsetOption struct {
	offset int
}

// WithOffset sets the number of documents to skip in [Collection.Get] results.
//
// Use this with [WithLimit] for pagination. The offset must be >= 0.
//
// For [Collection.Search], use [WithPage] with [PageOffset] instead.
//
// # Pagination Example
//
//	pageSize := 25
//	pageNum := 3 // 0-indexed
//
//	results, err := collection.Get(ctx,
//	    WithLimit(pageSize),
//	    WithOffset(pageNum * pageSize),
//	)
func WithOffset(offset int) *offsetOption {
	return &offsetOption{offset: offset}
}

func (o *offsetOption) ApplyToGet(op *CollectionGetOp) error {
	if o.offset < 0 {
		return ErrInvalidOffset
	}
	op.Offset = o.offset
	return nil
}

// nResultsOption implements result limit for Query operations.
// Use [WithNResults] to create this option.
type nResultsOption struct {
	nResults int
}

// WithNResults sets the number of nearest neighbors to return from [Collection.Query].
//
// This controls how many semantically similar documents are returned per query.
// The value must be greater than 0. Default is 10 if not specified.
//
// For [Collection.Get], use [WithLimit] instead.
// For [Collection.Search], use [WithPage] with [PageLimit] instead.
//
// # Example
//
//	results, err := collection.Query(ctx,
//	    WithQueryTexts("machine learning"),
//	    WithNResults(5),
//	)
//
// # Multiple Query Example
//
//	// Each query text returns up to 5 results
//	results, err := collection.Query(ctx,
//	    WithQueryTexts("AI", "robotics", "automation"),
//	    WithNResults(5),
//	)
func WithNResults(nResults int) *nResultsOption {
	return &nResultsOption{nResults: nResults}
}

func (o *nResultsOption) ApplyToQuery(op *CollectionQueryOp) error {
	if o.nResults <= 0 {
		return ErrInvalidNResults
	}
	op.NResults = o.nResults
	return nil
}

// queryTextsOption implements query text input for Query operations.
// Use [WithQueryTexts] to create this option.
type queryTextsOption struct {
	texts []string
}

// WithQueryTexts sets the text queries for semantic search in [Collection.Query].
//
// The texts are embedded using the collection's embedding function and used
// to find semantically similar documents. At least one text is required.
//
// Each query text produces a separate result set. Use [WithNResults] to control
// how many results are returned per query.
//
// For pre-computed embeddings, use [WithQueryEmbeddings] instead.
//
// # Single Query Example
//
//	results, err := collection.Query(ctx,
//	    WithQueryTexts("What is machine learning?"),
//	    WithNResults(10),
//	)
//
// # Multiple Query Example
//
//	results, err := collection.Query(ctx,
//	    WithQueryTexts(
//	        "machine learning algorithms",
//	        "deep neural networks",
//	        "natural language processing",
//	    ),
//	    WithNResults(5),
//	)
//	// results.IDs[0] contains results for first query
//	// results.IDs[1] contains results for second query
//	// results.IDs[2] contains results for third query
func WithQueryTexts(texts ...string) *queryTextsOption {
	return &queryTextsOption{texts: texts}
}

func (o *queryTextsOption) ApplyToQuery(op *CollectionQueryOp) error {
	if len(o.texts) == 0 {
		return ErrNoQueryTexts
	}
	op.QueryTexts = o.texts
	return nil
}

// textsOption implements document text input for Add and Update operations.
// Use [WithTexts] to create this option.
type textsOption struct {
	texts []string
}

// WithTexts sets the document text content for [Collection.Add], [Collection.Upsert],
// and [Collection.Update] operations.
//
// The texts are automatically embedded using the collection's embedding function
// unless embeddings are also provided via [WithEmbeddings].
//
// The number of texts must match the number of IDs provided via [WithIDs].
//
// # Add Example
//
//	err := collection.Add(ctx,
//	    WithIDs("doc1", "doc2", "doc3"),
//	    WithTexts(
//	        "Introduction to machine learning",
//	        "Deep learning fundamentals",
//	        "Natural language processing basics",
//	    ),
//	)
//
// # Update Example
//
//	err := collection.Update(ctx,
//	    WithIDs("doc1"),
//	    WithTexts("Updated: Introduction to machine learning v2"),
//	)
//
// # Upsert Example
//
//	err := collection.Upsert(ctx,
//	    WithIDs("doc1", "doc2"),
//	    WithTexts("New or updated doc 1", "New or updated doc 2"),
//	    WithMetadatas(meta1, meta2),
//	)
func WithTexts(texts ...string) *textsOption {
	return &textsOption{texts: texts}
}

func (o *textsOption) ApplyToAdd(op *CollectionAddOp) error {
	if len(o.texts) == 0 {
		return ErrNoTexts
	}
	if op.Documents == nil {
		op.Documents = make([]Document, 0, len(o.texts))
	}
	for _, text := range o.texts {
		op.Documents = append(op.Documents, NewTextDocument(text))
	}
	return nil
}

func (o *textsOption) ApplyToUpdate(op *CollectionUpdateOp) error {
	if len(o.texts) == 0 {
		return ErrNoTexts
	}
	if op.Documents == nil {
		op.Documents = make([]Document, 0, len(o.texts))
	}
	for _, text := range o.texts {
		op.Documents = append(op.Documents, NewTextDocument(text))
	}
	return nil
}

// searchWhereOption implements metadata filtering for Search operations.
// Use [WithSearchWhere] or [WithFilter] to create this option.
type searchWhereOption struct {
	where WhereClause
}

// WithSearchWhere filters [Collection.Search] results by metadata.
//
// This option uses the Search API's [WhereClause] filter syntax, which differs
// slightly from the Query API's [WhereFilter]. Use [K] to create field keys.
//
// For a more intuitive API, consider using [WithFilter] which is an alias.
//
// # Available Filter Functions
//
// Equality:
//   - [EqString], [EqInt], [EqFloat], [EqBool]
//   - [NeString], [NeInt], [NeFloat], [NeBool]
//
// Comparison:
//   - [GtInt], [GtFloat], [GtString]
//   - [GteInt], [GteFloat], [GteString]
//   - [LtInt], [LtFloat], [LtString]
//   - [LteInt], [LteFloat], [LteString]
//
// Set operations:
//   - [InString], [InInt], [InFloat]
//   - [NinString], [NinInt], [NinFloat]
//
// Logical:
//   - [And], [Or] - combine clauses
//
// Special:
//   - [IDIn] - filter by document IDs
//
// # Example
//
//	result, err := collection.Search(ctx,
//	    NewSearchRequest(
//	        WithKnnRank(KnnQueryText("query")),
//	        WithSearchWhere(And(
//	            EqString(K("status"), "published"),
//	            GtInt(K("views"), 1000),
//	        )),
//	    ),
//	)
func WithSearchWhere(where WhereClause) *searchWhereOption {
	return &searchWhereOption{where: where}
}

func (o *searchWhereOption) ApplyToSearchRequest(req *SearchRequest) error {
	if o.where != nil {
		if err := o.where.Validate(); err != nil {
			return err
		}
	}
	if req.Filter == nil {
		req.Filter = &SearchFilter{}
	}
	req.Filter.Where = o.where
	return nil
}

// Option validation errors.
//
// These errors are returned when option validation fails during operation construction.
var (
	// ErrInvalidLimit is returned when [WithLimit] receives a value <= 0.
	ErrInvalidLimit = errorString("limit must be greater than 0")

	// ErrInvalidOffset is returned when [WithOffset] receives a negative value.
	ErrInvalidOffset = errorString("offset must be greater than or equal to 0")

	// ErrInvalidNResults is returned when [WithNResults] receives a value <= 0.
	ErrInvalidNResults = errorString("nResults must be greater than 0")

	// ErrNoQueryTexts is returned when [WithQueryTexts] is called with no texts.
	ErrNoQueryTexts = errorString("at least one query text is required")

	// ErrNoTexts is returned when [WithTexts] is called with no texts.
	ErrNoTexts = errorString("at least one text is required")
)

// errorString is a simple error type for constant errors.
type errorString string

func (e errorString) Error() string { return string(e) }
