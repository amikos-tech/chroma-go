package v2

import (
	"bytes"
	"encoding/json"

	"github.com/pkg/errors"
)

/*
Search API

The Search API provides advanced vector search capabilities with fine-grained
control over ranking, filtering, pagination, and field selection.

# Basic Usage

	result, err := collection.Search(ctx,
	    NewSearchRequest(
	        WithKnnRank(KnnQueryText("machine learning")),
	        WithLimit(10),
	    ),
	)

# Filtering

Use [WithFilter] or [WithIDs] to narrow search results:

	result, err := collection.Search(ctx,
	    NewSearchRequest(
	        WithKnnRank(KnnQueryText("query")),
	        WithFilter(And(
	            EqString(K("status"), "published"),
	            GtInt(K("views"), 1000),
	        )),
	    ),
	)

# Field Selection

Use [WithSelect] to control which fields are returned:

	result, err := collection.Search(ctx,
	    NewSearchRequest(
	        WithKnnRank(KnnQueryText("query")),
	        WithSelect(KDocument, KScore, K("title")),
	    ),
	)

# Pagination

Use [WithLimit] and [WithOffset] for pagination:

	result, err := collection.Search(ctx,
	    NewSearchRequest(
	        WithKnnRank(KnnQueryText("query")),
	        WithLimit(20),
	        WithOffset(40),  // Page 3
	    ),
	)

# Grouping

Use [WithGroupBy] to group results by metadata field:

	result, err := collection.Search(ctx,
	    NewSearchRequest(
	        WithKnnRank(KnnQueryText("query")),
	        WithGroupBy(NewGroupBy(NewMinK(3, KScore), K("category"))),
	    ),
	)

# Read Levels

Use [WithReadLevel] to control read consistency:

	result, err := collection.Search(ctx,
	    NewSearchRequest(WithKnnRank(KnnQueryText("query"))),
	    WithReadLevel(ReadLevelIndexOnly),  // Faster but may miss recent writes
	)
*/

// ReadLevel controls whether search queries read from the write-ahead log (WAL).
//
// Use [WithReadLevel] to set this on a search query.
type ReadLevel string

const (
	// ReadLevelIndexAndWAL reads from both the compacted index and the WAL (default).
	// All committed writes will be visible. This is the safest option for
	// read-after-write consistency.
	ReadLevelIndexAndWAL ReadLevel = "index_and_wal"

	// ReadLevelIndexOnly reads only from the compacted index, skipping the WAL.
	// Faster for large collections, but recent writes that haven't been compacted
	// may not be visible. Use this for performance-critical searches where
	// eventual consistency is acceptable.
	ReadLevelIndexOnly ReadLevel = "index_only"
)

// SearchQuery holds one or more search requests to execute as a batch.
//
// Use [NewSearchRequest] to add requests and [WithReadLevel] to set consistency.
type SearchQuery struct {
	// Searches contains the individual search requests.
	Searches []SearchRequest `json:"searches"`

	// ReadLevel controls read consistency (default: ReadLevelIndexAndWAL).
	ReadLevel ReadLevel `json:"read_level,omitempty"`
}

// SearchResult represents the result of a search operation.
// The concrete type is [*SearchResultImpl].
type SearchResult interface{}

// Key identifies a metadata field for filtering or projection in the Search API.
//
// Key is a type alias for string, so raw strings work directly for backward
// compatibility. Both patterns are valid and compile correctly:
//
//	EqString(K("status"), "active")  // K() marks field names clearly (recommended for Search API)
//	EqString("status", "active")     // Raw string also works (common in Query API)
//
// The [K] function is a no-op that returns the string unchanged, serving purely
// as documentation to distinguish field names from values in filter expressions.
//
// For built-in fields, use the predefined constants: [KDocument], [KEmbedding],
// [KScore], [KMetadata], and [KID].
type Key = string

// K creates a Key for a metadata field name.
//
// Use this to clearly mark field names in filter and projection expressions.
// This improves code readability by distinguishing field names from values.
//
// K() is a no-op identity function - it returns the string unchanged.
// Both K("field") and "field" compile to the same value. The function exists
// solely for documentation purposes.
//
// # Filter Example (Search API style with K())
//
//	WithFilter(And(
//	    EqString(K("status"), "published"),
//	    GtInt(K("views"), 1000),
//	))
//
// # Filter Example (Query API style without K())
//
//	WithWhere(AndFilter(
//	    EqString("status", "published"),
//	    GtInt("views", 1000),
//	))
//
// # Projection Example
//
//	WithSelect(KDocument, KScore, K("title"), K("author"))
func K(key string) Key {
	return key
}

// Standard keys for document fields in Search API expressions.
//
// Use these constants with [WithSelect] to specify which fields to include
// in search results, or with filters to query these fields directly.
const (
	// KDocument represents the document text content.
	// Use in WithSelect to include document text in results.
	KDocument Key = "#document"

	// KEmbedding represents the vector embedding.
	// Use in WithSelect to include embeddings in results.
	KEmbedding Key = "#embedding"

	// KScore represents the ranking/similarity score.
	// Use in WithSelect to include scores in results.
	KScore Key = "#score"

	// KMetadata represents all metadata fields as a map.
	// Use in WithSelect to include all metadata in results.
	KMetadata Key = "#metadata"

	// KID represents the document ID.
	// Use in WithSelect to include IDs in results (usually included by default).
	KID Key = "#id"
)

// SearchFilter specifies which documents to include in search results.
//
// Filters can combine ID-based and metadata-based criteria. When both IDs
// and Where clauses are provided, they are combined with AND logic.
//
// Use [WithIDs] and [WithFilter] (or [WithSearchWhere]) to build filters:
//
//	NewSearchRequest(
//	    WithKnnRank(KnnQueryText("query")),
//	    WithIDs("doc1", "doc2", "doc3"),
//	    WithFilter(EqString(K("status"), "published")),
//	)
type SearchFilter struct {
	// IDs limits results to specific document IDs.
	// Converted to #id $in clause during serialization.
	IDs []DocumentID `json:"-"`

	// Where specifies metadata filter criteria.
	Where WhereClause `json:"-"`
}

// AppendIDs adds document IDs to the filter.
func (f *SearchFilter) AppendIDs(ids ...DocumentID) {
	f.IDs = append(f.IDs, ids...)
}

// SetSearchWhere sets the metadata filter clause.
func (f *SearchFilter) SetSearchWhere(where WhereClause) {
	f.Where = where
}

func (f *SearchFilter) MarshalJSON() ([]byte, error) {
	var clauses []WhereClause

	// Convert IDs to #id $in clause
	if len(f.IDs) > 0 {
		clauses = append(clauses, IDIn(f.IDs...))
	}

	// Add where clause
	if f.Where != nil {
		clauses = append(clauses, f.Where)
	}

	if len(clauses) == 0 {
		return []byte("{}"), nil
	}

	// If single clause, serialize directly; otherwise combine with $and
	var result WhereClause
	if len(clauses) == 1 {
		result = clauses[0]
	} else {
		result = And(clauses...)
	}

	// Validate the composed filter before serializing
	if err := result.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid search filter")
	}

	return json.Marshal(result)
}

// SearchSelect specifies which fields to include in search results.
//
// Use [WithSelect] or [WithSelectAll] to configure field selection:
//
//	WithSelect(KDocument, KScore, K("title"), K("author"))
type SearchSelect struct {
	// Keys contains the field keys to include in results.
	// Use [KDocument], [KEmbedding], [KScore], [KMetadata], [KID] for standard fields,
	// or [K]("field_name") for custom metadata fields.
	Keys []Key `json:"keys,omitempty"`
}

// SearchRequest represents a single search operation with filter, ranking,
// pagination, and projection.
//
// Create using [NewSearchRequest] with options:
//
//	NewSearchRequest(
//	    WithKnnRank(KnnQueryText("machine learning")),
//	    WithFilter(EqString(K("status"), "published")),
//	    WithPage(PageLimit(10)),
//	    WithSelect(KDocument, KScore),
//	)
type SearchRequest struct {
	// Filter specifies which documents to include (by ID or metadata).
	Filter *SearchFilter `json:"filter,omitempty"`

	// Limit specifies pagination (limit and offset).
	Limit *SearchPage `json:"limit,omitempty"`

	// Rank specifies the ranking expression (e.g., KNN similarity).
	Rank Rank `json:"rank,omitempty"`

	// Select specifies which fields to include in results.
	Select *SearchSelect `json:"select,omitempty"`

	// GroupBy groups results by metadata field values.
	GroupBy *GroupBy `json:"group_by,omitempty"`
}

func (r *SearchRequest) MarshalJSON() ([]byte, error) {
	result := make(map[string]any)

	if r.Filter != nil {
		filterData, err := r.Filter.MarshalJSON()
		if err != nil {
			return nil, err
		}
		if filterData != nil {
			var filterMap map[string]any
			if err := json.Unmarshal(filterData, &filterMap); err != nil {
				return nil, err
			}
			result["filter"] = filterMap
		}
	}

	if r.Limit != nil {
		result["limit"] = r.Limit
	}

	if r.Rank != nil {
		rankData, err := r.Rank.MarshalJSON()
		if err != nil {
			return nil, err
		}
		var rankMap any
		if err := json.Unmarshal(rankData, &rankMap); err != nil {
			return nil, err
		}
		result["rank"] = rankMap
	}

	if r.Select != nil && len(r.Select.Keys) > 0 {
		keys := make([]string, len(r.Select.Keys))
		for i, k := range r.Select.Keys {
			keys[i] = string(k)
		}
		result["select"] = map[string][]string{"keys": keys}
	}

	if r.GroupBy != nil {
		groupByData, err := r.GroupBy.MarshalJSON()
		if err != nil {
			return nil, err
		}
		var groupByMap any
		if err := json.Unmarshal(groupByData, &groupByMap); err != nil {
			return nil, err
		}
		result["group_by"] = groupByMap
	}

	return json.Marshal(result)
}

// SearchCollectionOption configures a [SearchQuery] for [Collection.Search].
//
// Use [NewSearchRequest] to create search requests, and [WithReadLevel] to
// set the read consistency level.
type SearchCollectionOption func(update *SearchQuery) error

// WithReadLevel sets the read consistency level for the search query.
//
// Use [ReadLevelIndexOnly] for faster searches when eventual consistency is
// acceptable. Recent writes that haven't been compacted may not be visible.
//
// Default is [ReadLevelIndexAndWAL] which includes all committed writes.
//
// # Example
//
//	result, err := collection.Search(ctx,
//	    NewSearchRequest(WithKnnRank(KnnQueryText("query"))),
//	    WithReadLevel(ReadLevelIndexOnly),  // Faster but eventual consistency
//	)
func WithReadLevel(level ReadLevel) SearchCollectionOption {
	return func(query *SearchQuery) error {
		if level != ReadLevelIndexAndWAL && level != ReadLevelIndexOnly {
			return errors.Errorf("invalid read level: %q (expected %q or %q)", level, ReadLevelIndexAndWAL, ReadLevelIndexOnly)
		}
		query.ReadLevel = level
		return nil
	}
}

// SearchOption is an alias for [SearchRequestOption] for backward compatibility.
type SearchOption = SearchRequestOption

// searchFilterOption implements search filter for Search operations.
// Use [WithSearchFilter] to create this option.
type searchFilterOption struct {
	filter *SearchFilter
}

// WithSearchFilter sets a pre-built [SearchFilter] on the search request.
//
// For most cases, use [WithFilter] and [WithIDs] instead, which are simpler.
func WithSearchFilter(filter *SearchFilter) *searchFilterOption {
	return &searchFilterOption{filter: filter}
}

func (o *searchFilterOption) ApplyToSearchRequest(req *SearchRequest) error {
	req.Filter = o.filter
	return nil
}

// WithFilter adds a metadata filter to the search.
//
// Use [K] to mark metadata field names in filter expressions. Combine multiple
// conditions with [And] and [Or].
//
// # Available Filter Functions
//
// Equality: [EqString], [EqInt], [EqFloat], [EqBool], [NeString], [NeInt], etc.
// Comparison: [GtInt], [GteInt], [LtInt], [LteInt], [GtFloat], etc.
// Set operations: [InString], [InInt], [NinString], [NinInt], etc.
// Logical: [And], [Or]
// ID filtering: [IDIn]
//
// # Example
//
//	NewSearchRequest(
//	    WithKnnRank(KnnQueryText("query")),
//	    WithFilter(And(
//	        EqString(K("status"), "published"),
//	        GtInt(K("views"), 100),
//	    )),
//	)
func WithFilter(where WhereClause) SearchRequestOption {
	return WithSearchWhere(where)
}

// Deprecated: Use [WithIDs] instead.
func WithFilterIDs(ids ...DocumentID) SearchRequestOption {
	return WithIDs(ids...)
}

// Deprecated: Use [NewPage] with [Limit] instead.
// SearchWithLimit is kept for backward compatibility.
func SearchWithLimit(limit int) PageOpts {
	return PageLimit(limit)
}

// Deprecated: Use [NewPage] with [Offset] instead.
// SearchWithOffset is kept for backward compatibility.
func SearchWithOffset(offset int) PageOpts {
	return PageOffset(offset)
}

// SearchPage specifies pagination for search results.
//
// Deprecated: Use [NewPage] with [Limit] and [Offset] instead.
type SearchPage struct {
	// Limit is the maximum number of results to return.
	Limit int `json:"limit,omitempty"`

	// Offset is the number of results to skip.
	Offset int `json:"offset,omitempty"`
}

// PageOpts configures pagination options for [WithPage].
//
// Deprecated: Use [NewPage] with [Limit] and [Offset] instead.
type PageOpts func(page *SearchPage) error

// Deprecated: Use [NewPage] with [Limit] instead.
// PageLimit sets the maximum number of results to return.
func PageLimit(limit int) PageOpts {
	return func(page *SearchPage) error {
		if limit < 1 {
			return errors.New("invalid limit, must be >= 1")
		}
		page.Limit = limit
		return nil
	}
}

// Deprecated: Use [NewPage] with [Offset] instead.
// PageOffset sets the number of results to skip.
func PageOffset(offset int) PageOpts {
	return func(page *SearchPage) error {
		if offset < 0 {
			return errors.New("invalid offset, must be >= 0")
		}
		page.Offset = offset
		return nil
	}
}

// pageOption implements pagination for Search operations.
type pageOption struct {
	page *SearchPage
	err  error
}

// Deprecated: Use [NewPage] with [Limit] and [Offset] instead.
//
// # Migration Example
//
// Before:
//
//	NewSearchRequest(WithPage(PageLimit(20), PageOffset(40)))
//
// After (using Page):
//
//	page, _ := NewPage(Limit(20), Offset(40))
//	NewSearchRequest(page)
//
// Or (using WithLimit/WithOffset directly):
//
//	NewSearchRequest(WithLimit(20), WithOffset(40))
func WithPage(pageOpts ...PageOpts) *pageOption {
	page := &SearchPage{}
	for _, opt := range pageOpts {
		if err := opt(page); err != nil {
			return &pageOption{page: page, err: err}
		}
	}
	return &pageOption{page: page}
}

func (o *pageOption) ApplyToSearchRequest(req *SearchRequest) error {
	if o.err != nil {
		return o.err
	}
	req.Limit = o.page
	return nil
}

// selectOption implements field selection for Search operations.
// Use [WithSelect] to create this option.
type selectOption struct {
	keys []Key
}

// WithSelect specifies which fields to include in search results.
//
// By default, only IDs are returned. Use this to include additional fields
// like document content, embeddings, scores, or specific metadata fields.
//
// # Standard Fields
//
//   - [KID] - document ID (usually included by default)
//   - [KDocument] - document text content
//   - [KEmbedding] - vector embedding
//   - [KScore] - ranking/similarity score
//   - [KMetadata] - all metadata as a map
//
// # Custom Metadata Fields
//
// Use [K]("field_name") to select specific metadata fields.
//
// # Example
//
//	NewSearchRequest(
//	    WithKnnRank(KnnQueryText("query")),
//	    WithSelect(KDocument, KScore, K("title"), K("author")),
//	)
func WithSelect(projectionKeys ...Key) *selectOption {
	return &selectOption{keys: projectionKeys}
}

func (o *selectOption) ApplyToSearchRequest(req *SearchRequest) error {
	if req.Select == nil {
		req.Select = &SearchSelect{}
	}
	req.Select.Keys = append(req.Select.Keys, o.keys...)
	return nil
}

// selectAllOption implements selecting all fields for Search operations.
// Use [WithSelectAll] to create this option.
type selectAllOption struct{}

// WithSelectAll includes all standard fields in search results.
//
// Equivalent to WithSelect(KID, KDocument, KEmbedding, KMetadata, KScore).
//
// # Example
//
//	NewSearchRequest(
//	    WithKnnRank(KnnQueryText("query")),
//	    WithSelectAll(),
//	)
func WithSelectAll() *selectAllOption {
	return &selectAllOption{}
}

func (o *selectAllOption) ApplyToSearchRequest(req *SearchRequest) error {
	req.Select = &SearchSelect{
		Keys: []Key{KID, KDocument, KEmbedding, KMetadata, KScore},
	}
	return nil
}

// rankOption implements ranking for Search operations.
type rankOption struct {
	rank Rank
}

// WithRank sets a custom ranking expression on the search request.
// Use this for complex rank expressions built from arithmetic operations.
//
// Example:
//
//	knn1, _ := NewKnnRank(KnnQueryText("query1"))
//	knn2, _ := NewKnnRank(KnnQueryText("query2"))
//	combined := knn1.Multiply(FloatOperand(0.7)).Add(knn2.Multiply(FloatOperand(0.3)))
//
//	result, err := col.Search(ctx,
//	    NewSearchRequest(
//	        WithRank(combined),
//	        WithPage(PageLimit(10)),
//	    ),
//	)
func WithRank(rank Rank) *rankOption {
	return &rankOption{rank: rank}
}

func (o *rankOption) ApplyToSearchRequest(req *SearchRequest) error {
	req.Rank = o.rank
	return nil
}

// groupByOption implements grouping for Search operations.
type groupByOption struct {
	groupBy *GroupBy
}

// WithGroupBy groups results by metadata keys using the specified aggregation.
//
// Example:
//
//	result, err := col.Search(ctx,
//	    NewSearchRequest(
//	        WithKnnRank(KnnQueryText("query"), WithKnnLimit(100)),
//	        WithGroupBy(NewGroupBy(NewMinK(3, KScore), K("category"))),
//	        WithPage(PageLimit(30)),
//	    ),
//	)
func WithGroupBy(groupBy *GroupBy) *groupByOption {
	return &groupByOption{groupBy: groupBy}
}

func (o *groupByOption) ApplyToSearchRequest(req *SearchRequest) error {
	if o.groupBy == nil {
		return nil
	}
	if err := o.groupBy.Validate(); err != nil {
		return err
	}
	req.GroupBy = o.groupBy
	return nil
}

// NewSearchRequest creates a search request and adds it to the query.
//
// Example:
//
//	result, err := collection.Search(ctx,
//	    NewSearchRequest(
//	        WithKnnRank(KnnQueryText("machine learning"), WithKnnLimit(50)),
//	        WithFilter(EqString(K("status"), "published")),
//	        WithPage(PageLimit(10)),
//	        WithSelect(KDocument, KScore),
//	    ),
//	)
func NewSearchRequest(opts ...SearchRequestOption) SearchCollectionOption {
	return func(update *SearchQuery) error {
		search := &SearchRequest{}
		for _, opt := range opts {
			if err := opt.ApplyToSearchRequest(search); err != nil {
				return err
			}
		}
		update.Searches = append(update.Searches, *search)
		return nil
	}
}

// SearchResultImpl holds the results of a search operation.
//
// Results are organized as nested arrays where the outer array corresponds
// to each search request in the batch, and the inner arrays contain the
// matching documents.
//
// # Accessing Results
//
// Use [SearchResultImpl.Rows] for single-query results:
//
//	result, _ := collection.Search(ctx, NewSearchRequest(...))
//	for _, row := range result.(*SearchResultImpl).Rows() {
//	    fmt.Printf("ID: %s, Score: %f\n", row.ID, row.Score)
//	}
//
// Use [SearchResultImpl.RowGroups] for batch queries:
//
//	result, _ := collection.Search(ctx, req1, req2, req3)
//	for i, group := range result.(*SearchResultImpl).RowGroups() {
//	    fmt.Printf("Query %d results:\n", i)
//	    for _, row := range group {
//	        fmt.Printf("  ID: %s, Score: %f\n", row.ID, row.Score)
//	    }
//	}
//
// Use [SearchResultImpl.At] for random access:
//
//	row, ok := result.(*SearchResultImpl).At(0, 5)  // First query, 6th result
type SearchResultImpl struct {
	// IDs contains document IDs for each query result set.
	// IDs[queryIndex][resultIndex] is the ID of the result.
	IDs [][]DocumentID `json:"ids,omitempty"`

	// Documents contains document text for each query result set.
	// Only populated if [WithSelect] included [KDocument].
	Documents [][]string `json:"documents,omitempty"`

	// Metadatas contains metadata for each query result set.
	// Only populated if [WithSelect] included [KMetadata] or specific fields.
	Metadatas [][]DocumentMetadata `json:"metadatas,omitempty"`

	// Embeddings contains vector embeddings for each query result set.
	// Only populated if [WithSelect] included [KEmbedding].
	Embeddings [][][]float32 `json:"embeddings,omitempty"`

	// Scores contains ranking scores for each query result set.
	// Only populated if [WithSelect] included [KScore].
	Scores [][]float64 `json:"scores,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshalling for SearchResultImpl.
// This is necessary because DocumentMetadata is an interface type that
// cannot be directly unmarshalled by the standard JSON decoder.
func (r *SearchResultImpl) UnmarshalJSON(data []byte) error {
	var temp map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&temp); err != nil {
		return errors.Wrap(err, "failed to unmarshal SearchResult")
	}

	// Parse IDs
	if idsRaw, ok := temp["ids"]; ok && idsRaw != nil {
		if idsList, ok := idsRaw.([]interface{}); ok {
			r.IDs = make([][]DocumentID, 0, len(idsList))
			for _, idsGroup := range idsList {
				if idsGroup == nil {
					r.IDs = append(r.IDs, nil)
					continue
				}
				if group, ok := idsGroup.([]interface{}); ok {
					ids := make([]DocumentID, 0, len(group))
					for _, id := range group {
						if idStr, ok := id.(string); ok {
							ids = append(ids, DocumentID(idStr))
						}
					}
					r.IDs = append(r.IDs, ids)
				}
			}
		}
	}

	// Parse Documents
	if docsRaw, ok := temp["documents"]; ok && docsRaw != nil {
		if docsList, ok := docsRaw.([]interface{}); ok {
			r.Documents = make([][]string, 0, len(docsList))
			for _, docsGroup := range docsList {
				if docsGroup == nil {
					r.Documents = append(r.Documents, nil)
					continue
				}
				if group, ok := docsGroup.([]interface{}); ok {
					docs := make([]string, 0, len(group))
					for _, doc := range group {
						if docStr, ok := doc.(string); ok {
							docs = append(docs, docStr)
						}
					}
					r.Documents = append(r.Documents, docs)
				}
			}
		}
	}

	// Parse Metadatas - needs special handling for interface type
	if metasRaw, ok := temp["metadatas"]; ok && metasRaw != nil {
		if metasList, ok := metasRaw.([]interface{}); ok {
			r.Metadatas = make([][]DocumentMetadata, 0, len(metasList))
			for _, metasGroup := range metasList {
				if metasGroup == nil {
					r.Metadatas = append(r.Metadatas, nil)
					continue
				}
				if group, ok := metasGroup.([]interface{}); ok {
					metas := make([]DocumentMetadata, 0, len(group))
					for _, meta := range group {
						if meta == nil {
							metas = append(metas, nil)
							continue
						}
						if metaMap, ok := meta.(map[string]interface{}); ok {
							docMeta, err := NewDocumentMetadataFromMap(metaMap)
							if err != nil {
								return errors.Wrap(err, "failed to parse document metadata")
							}
							metas = append(metas, docMeta)
						}
					}
					r.Metadatas = append(r.Metadatas, metas)
				}
			}
		}
	}

	// Parse Embeddings
	if embsRaw, ok := temp["embeddings"]; ok && embsRaw != nil {
		if embsList, ok := embsRaw.([]interface{}); ok {
			r.Embeddings = make([][][]float32, 0, len(embsList))
			for _, embsGroup := range embsList {
				if embsGroup == nil {
					r.Embeddings = append(r.Embeddings, nil)
					continue
				}
				if group, ok := embsGroup.([]interface{}); ok {
					embs := make([][]float32, 0, len(group))
					for _, emb := range group {
						if emb == nil {
							embs = append(embs, nil)
							continue
						}
						if embArr, ok := emb.([]interface{}); ok {
							floats := make([]float32, 0, len(embArr))
							for _, f := range embArr {
								switch fVal := f.(type) {
								case float64:
									floats = append(floats, float32(fVal))
								case json.Number:
									v, err := fVal.Float64()
									if err != nil {
										return errors.Wrapf(err, "invalid embedding value: %v", fVal)
									}
									floats = append(floats, float32(v))
								}
							}
							embs = append(embs, floats)
						}
					}
					r.Embeddings = append(r.Embeddings, embs)
				}
			}
		}
	}

	// Parse Scores
	if scoresRaw, ok := temp["scores"]; ok && scoresRaw != nil {
		if scoresList, ok := scoresRaw.([]interface{}); ok {
			r.Scores = make([][]float64, 0, len(scoresList))
			for _, scoresGroup := range scoresList {
				if scoresGroup == nil {
					r.Scores = append(r.Scores, nil)
					continue
				}
				if group, ok := scoresGroup.([]interface{}); ok {
					scores := make([]float64, 0, len(group))
					for _, score := range group {
						switch scoreVal := score.(type) {
						case float64:
							scores = append(scores, scoreVal)
						case json.Number:
							v, err := scoreVal.Float64()
							if err != nil {
								return errors.Wrapf(err, "invalid score value: %v", scoreVal)
							}
							scores = append(scores, v)
						}
					}
					r.Scores = append(r.Scores, scores)
				}
			}
		}
	}

	return nil
}

// Rows returns the first search group's results for easy iteration.
// For multiple search requests, use RowGroups().
func (r *SearchResultImpl) Rows() []ResultRow {
	if len(r.IDs) == 0 {
		return nil
	}
	return r.buildGroupRows(0)
}

// RowGroups returns all search groups as [][]ResultRow.
func (r *SearchResultImpl) RowGroups() [][]ResultRow {
	if len(r.IDs) == 0 {
		return nil
	}
	groups := make([][]ResultRow, len(r.IDs))
	for g := range r.IDs {
		groups[g] = r.buildGroupRows(g)
	}
	return groups
}

// At returns the result at the given group and index with bounds checking.
// Returns false if either index is out of bounds.
func (r *SearchResultImpl) At(group, index int) (ResultRow, bool) {
	if group < 0 || group >= len(r.IDs) {
		return ResultRow{}, false
	}
	ids := r.IDs[group]
	if index < 0 || index >= len(ids) {
		return ResultRow{}, false
	}
	return r.buildRow(group, index), true
}

func (r *SearchResultImpl) buildGroupRows(g int) []ResultRow {
	ids := r.IDs[g]
	if len(ids) == 0 {
		return nil
	}
	rows := make([]ResultRow, len(ids))
	for i := range ids {
		rows[i] = r.buildRow(g, i)
	}
	return rows
}

func (r *SearchResultImpl) buildRow(g, i int) ResultRow {
	row := ResultRow{
		ID: r.IDs[g][i],
	}
	if g < len(r.Documents) && i < len(r.Documents[g]) {
		row.Document = r.Documents[g][i]
	}
	if g < len(r.Metadatas) && i < len(r.Metadatas[g]) {
		row.Metadata = r.Metadatas[g][i]
	}
	if g < len(r.Embeddings) && i < len(r.Embeddings[g]) {
		row.Embedding = r.Embeddings[g][i]
	}
	if g < len(r.Scores) && i < len(r.Scores[g]) {
		row.Score = r.Scores[g][i]
	}
	return row
}
