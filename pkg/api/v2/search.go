package v2

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// SearchQuery contains multiple search requests
type SearchQuery struct {
	Searches []SearchRequest `json:"searches"`
}

// SearchResult represents the result of a search operation
type SearchResult interface{}

// ProjectionKey represents a field to project in search results
type ProjectionKey string

// K creates a custom projection key for metadata fields
func K(key string) ProjectionKey {
	return ProjectionKey(key)
}

// Standard projection keys
const (
	KDocument  ProjectionKey = "#document"
	KEmbedding ProjectionKey = "#embedding"
	KScore     ProjectionKey = "#score"
	KMetadata  ProjectionKey = "#metadata"
	KID        ProjectionKey = "#id"
)

// SearchFilter represents filter criteria for search
type SearchFilter struct {
	IDs           []DocumentID        `json:"ids,omitempty"`
	Where         WhereClause         `json:"where,omitempty"`
	WhereDocument WhereDocumentFilter `json:"where_document,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for SearchFilter
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

// SearchSelect represents fields to select in search results
type SearchSelect struct {
	Keys []ProjectionKey `json:"keys,omitempty"`
}

// SearchRequest represents a single search operation
type SearchRequest struct {
	Filter *SearchFilter `json:"filter,omitempty"`
	Limit  *SearchPage   `json:"limit,omitempty"`
	Rank   Rank          `json:"rank,omitempty"`
	Select *SearchSelect `json:"select,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for SearchRequest
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

// SearchCollectionOption is an option for the Search method on a collection
type SearchCollectionOption func(update *SearchQuery) error

// SearchOption is an option for building a SearchRequest
type SearchOption func(req *SearchRequest) error

// WithSearchFilter adds a filter to the search request
func WithSearchFilter(filter *SearchFilter) SearchOption {
	return func(req *SearchRequest) error {
		req.Filter = filter
		return nil
	}
}

// WithFilter adds a where filter to the search request
func WithFilter(where WhereClause) SearchOption {
	return func(req *SearchRequest) error {
		if req.Filter == nil {
			req.Filter = &SearchFilter{}
		}
		req.Filter.Where = where
		return nil
	}
}

// WithFilterIDs adds ID filter to the search request
func WithFilterIDs(ids ...DocumentID) SearchOption {
	return func(req *SearchRequest) error {
		if req.Filter == nil {
			req.Filter = &SearchFilter{}
		}
		req.Filter.IDs = ids
		return nil
	}
}

// WithFilterDocument adds a document filter to the search request
func WithFilterDocument(whereDoc WhereDocumentFilter) SearchOption {
	return func(req *SearchRequest) error {
		if req.Filter == nil {
			req.Filter = &SearchFilter{}
		}
		req.Filter.WhereDocument = whereDoc
		return nil
	}
}

// SearchPage represents pagination options
type SearchPage struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// PageOpts is an option for configuring pagination
type PageOpts func(page *SearchPage) error

// WithLimit sets the limit for pagination
func WithLimit(limit int) PageOpts {
	return func(page *SearchPage) error {
		if limit < 1 {
			return errors.New("invalid limit, must be >= 1")
		}
		page.Limit = limit
		return nil
	}
}

// WithOffset sets the offset for pagination
func WithOffset(offset int) PageOpts {
	return func(page *SearchPage) error {
		if offset < 0 {
			return errors.New("invalid offset, must be >= 0")
		}
		page.Offset = offset
		return nil
	}
}

// WithPage adds pagination to the search request
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

// WithSelect adds projection keys to the search request
func WithSelect(projectionKeys ...ProjectionKey) SearchOption {
	return func(req *SearchRequest) error {
		if req.Select == nil {
			req.Select = &SearchSelect{}
		}
		req.Select.Keys = append(req.Select.Keys, projectionKeys...)
		return nil
	}
}

// WithSelectAll adds all standard projection keys to the search request
func WithSelectAll() SearchOption {
	return func(req *SearchRequest) error {
		req.Select = &SearchSelect{
			Keys: []ProjectionKey{KID, KDocument, KEmbedding, KMetadata, KScore},
		}
		return nil
	}
}

// NewSearchRequest creates a new search request with the given options
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

// SearchResultImpl is the concrete implementation of SearchResult
type SearchResultImpl struct {
	IDs        [][]DocumentID         `json:"ids,omitempty"`
	Documents  [][]string             `json:"documents,omitempty"`
	Metadatas  [][]CollectionMetadata `json:"metadatas,omitempty"`
	Embeddings [][][]float32          `json:"embeddings,omitempty"`
	Scores     [][]float64            `json:"scores,omitempty"`
}
