package v2

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// SearchQuery holds one or more search requests to execute as a batch.
type SearchQuery struct {
	Searches []SearchRequest `json:"searches"`
}

// SearchResult represents the result of a search operation.
type SearchResult interface{}

// ProjectionKey identifies a field to include in search results.
// Use standard keys ([KDocument], [KEmbedding], etc.) or create custom ones with [K].
type ProjectionKey string

// K creates a projection key for a custom metadata field.
//
// Example:
//
//	WithSelect(KDocument, KScore, K("title"), K("author"))
func K(key string) ProjectionKey {
	return ProjectionKey(key)
}

// Standard projection keys for search result fields.
const (
	KDocument  ProjectionKey = "#document"  // The document text
	KEmbedding ProjectionKey = "#embedding" // The vector embedding
	KScore     ProjectionKey = "#score"     // The ranking score
	KMetadata  ProjectionKey = "#metadata"  // All metadata fields
	KID        ProjectionKey = "#id"        // The document ID
)

// SearchFilter specifies which documents to include in search results.
type SearchFilter struct {
	IDs           []DocumentID        `json:"ids,omitempty"`
	Where         WhereClause         `json:"where,omitempty"`
	WhereDocument WhereDocumentFilter `json:"where_document,omitempty"`
}

func (f *SearchFilter) MarshalJSON() ([]byte, error) {
	result := make(map[string]interface{})
	if len(f.IDs) > 0 {
		result["ids"] = f.IDs
	}
	if f.Where != nil {
		result["where"] = f.Where
	}
	if f.WhereDocument != nil {
		result["where_document"] = f.WhereDocument
	}
	if len(result) == 0 {
		return nil, nil
	}
	return json.Marshal(result)
}

// SearchSelect specifies which fields to include in search results.
type SearchSelect struct {
	Keys []ProjectionKey `json:"keys,omitempty"`
}

// SearchRequest represents a single search operation with filter, ranking, pagination, and projection.
type SearchRequest struct {
	Filter *SearchFilter `json:"filter,omitempty"`
	Limit  *SearchPage   `json:"limit,omitempty"`
	Rank   Rank          `json:"rank,omitempty"`
	Select *SearchSelect `json:"select,omitempty"`
}

func (r *SearchRequest) MarshalJSON() ([]byte, error) {
	result := make(map[string]interface{})

	if r.Filter != nil {
		filterData, err := r.Filter.MarshalJSON()
		if err != nil {
			return nil, err
		}
		if filterData != nil {
			var filterMap map[string]interface{}
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
		var rankMap interface{}
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

	return json.Marshal(result)
}

// SearchCollectionOption configures a [SearchQuery] for the collection's Search method.
type SearchCollectionOption func(update *SearchQuery) error

// SearchOption configures a [SearchRequest].
type SearchOption func(req *SearchRequest) error

// WithSearchFilter sets a complete filter on the search request.
func WithSearchFilter(filter *SearchFilter) SearchOption {
	return func(req *SearchRequest) error {
		req.Filter = filter
		return nil
	}
}

// WithFilter adds a metadata filter to the search.
//
// Example:
//
//	WithFilter(And(EqString("status", "published"), GtInt("views", 100)))
func WithFilter(where WhereClause) SearchOption {
	return func(req *SearchRequest) error {
		if req.Filter == nil {
			req.Filter = &SearchFilter{}
		}
		req.Filter.Where = where
		return nil
	}
}

// WithFilterIDs restricts search to specific document IDs.
func WithFilterIDs(ids ...DocumentID) SearchOption {
	return func(req *SearchRequest) error {
		if req.Filter == nil {
			req.Filter = &SearchFilter{}
		}
		req.Filter.IDs = ids
		return nil
	}
}

// WithFilterDocument adds a document content filter to the search.
func WithFilterDocument(whereDoc WhereDocumentFilter) SearchOption {
	return func(req *SearchRequest) error {
		if req.Filter == nil {
			req.Filter = &SearchFilter{}
		}
		req.Filter.WhereDocument = whereDoc
		return nil
	}
}

// SearchPage specifies pagination for search results.
type SearchPage struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// PageOpts configures pagination options.
type PageOpts func(page *SearchPage) error

// WithLimit sets the maximum number of results to return.
func WithLimit(limit int) PageOpts {
	return func(page *SearchPage) error {
		if limit < 1 {
			return errors.New("invalid limit, must be >= 1")
		}
		page.Limit = limit
		return nil
	}
}

// WithOffset sets the number of results to skip (for pagination).
func WithOffset(offset int) PageOpts {
	return func(page *SearchPage) error {
		if offset < 0 {
			return errors.New("invalid offset, must be >= 0")
		}
		page.Offset = offset
		return nil
	}
}

// WithPage adds pagination to the search request.
//
// Example:
//
//	WithPage(WithLimit(20), WithOffset(40))  // Page 3 of 20 results per page
func WithPage(pageOpts ...PageOpts) SearchOption {
	return func(req *SearchRequest) error {
		page := &SearchPage{}
		for _, opt := range pageOpts {
			if err := opt(page); err != nil {
				return err
			}
		}
		req.Limit = page
		return nil
	}
}

// WithSelect specifies which fields to include in search results.
//
// Example:
//
//	WithSelect(KDocument, KScore, K("title"), K("author"))
func WithSelect(projectionKeys ...ProjectionKey) SearchOption {
	return func(req *SearchRequest) error {
		if req.Select == nil {
			req.Select = &SearchSelect{}
		}
		req.Select.Keys = append(req.Select.Keys, projectionKeys...)
		return nil
	}
}

// WithSelectAll includes all standard fields in search results.
func WithSelectAll() SearchOption {
	return func(req *SearchRequest) error {
		req.Select = &SearchSelect{
			Keys: []ProjectionKey{KID, KDocument, KEmbedding, KMetadata, KScore},
		}
		return nil
	}
}

// NewSearchRequest creates a search request and adds it to the query.
//
// Example:
//
//	result, err := collection.Search(ctx,
//	    NewSearchRequest(
//	        WithKnnRank(KnnQueryText("machine learning"), WithKnnLimit(50)),
//	        WithFilter(EqString("status", "published")),
//	        WithPage(WithLimit(10)),
//	        WithSelect(KDocument, KScore),
//	    ),
//	)
func NewSearchRequest(opts ...SearchOption) SearchCollectionOption {
	return func(update *SearchQuery) error {
		search := &SearchRequest{}
		for _, opt := range opts {
			if err := opt(search); err != nil {
				return err
			}
		}
		update.Searches = append(update.Searches, *search)
		return nil
	}
}

// SearchResultImpl holds the results of a search operation.
type SearchResultImpl struct {
	IDs        [][]DocumentID         `json:"ids,omitempty"`
	Documents  [][]string             `json:"documents,omitempty"`
	Metadatas  [][]CollectionMetadata `json:"metadatas,omitempty"`
	Embeddings [][][]float32          `json:"embeddings,omitempty"`
	Scores     [][]float64            `json:"scores,omitempty"`
}
