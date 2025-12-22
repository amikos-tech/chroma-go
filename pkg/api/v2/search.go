package v2

import (
	"context"
	"log"

	"gopkg.in/errgo.v2/errors"
)

type SearchQuery struct {
	Searches []SearchRequest `json:"searches"`
}

type SearchResult interface{}

type ProjectionKey string

func K(key string) ProjectionKey {
	return ProjectionKey(key)
}

const (
	KDocument  ProjectionKey = "#document"
	KEmbedding ProjectionKey = "#embedding"
	KScore     ProjectionKey = "#score"
	KMetadata  ProjectionKey = "#metadata"
	KID        ProjectionKey = "#id"
)

type SelectAll func()

type SearchRequest struct {
	Filter struct {
		QueryIds    []string `json:"query_ids"`
		WhereClause struct {
		} `json:"where_clause"`
	} `json:"filter"`
	Limit struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"limit"`
	Rank struct {
	} `json:"rank"`
	Select struct {
		Keys []string `json:"keys"`
	} `json:"select"`
}

// Search(WithFilter(),WithRank(),WithLimit(),WithSelect()),... // more searches
type SearchCollectionOption func(update *SearchQuery) error

type SearchOption func(req *SearchRequest) error

func WithFilter(where WhereFilter) SearchOption {
	return func(req *SearchRequest) error {
		return nil
	}
}
func WithRank() SearchOption {
	return func(req *SearchRequest) error {
		return nil
	}
}

func WithPage(pageOpts ...PageOpts) SearchOption {
	return func(req *SearchRequest) error {
		return nil
	}
}

type SearchPage struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
type PageOpts func(page *SearchPage) error

func WithLimit(limit int) PageOpts {
	return func(page *SearchPage) error {
		if limit < 1 {
			return errors.New("invalid limit, must be >= 1")
		}
		page.Limit = limit
		return nil
	}
}

func WithOffset(offset int) PageOpts {
	return func(page *SearchPage) error {
		if offset < 0 {
			return errors.New("invalid offset, must be >= 0")
		}
		page.Offset = offset
		return nil
	}
}

func WithSelect(projectionKeys ...ProjectionKey) SearchOption {
	return func(req *SearchRequest) error {
		return nil
	}
}
func WithSelectAll() SearchOption {
	return func(req *SearchRequest) error {
		return nil
	}
}

//type KnnQuery struct {
//	QueryText string    `json:"query_text"`
//	QueryVec  []float32 `json:"query_vector"`
//}

//func WithKnn(query KnnQueryOption, knnOptions ...KnnOption) SearchOption {
//	return func(req *SearchRequest) error {
//		return nil
//	}
//}

func NewSearchRequest(opts ...SearchOption) SearchCollectionOption {
	sq := &SearchQuery{}
	search := &SearchRequest{}
	for _, opt := range opts {
		_ = opt(search)
	}
	sq.Searches = append(sq.Searches, *search)
	return func(update *SearchQuery) error {
		*update = *sq
		return nil
	}
}

func main() {
	client, err := NewHTTPClient()
	if err != nil {
		log.Printf("Error creating client: %s \n", err)
		return
	}
	// Close the client to release any resources such as local embedding functions
	defer func() {
		err = client.Close()
		if err != nil {
			log.Printf("Error closing client: %s \n", err)
			return
		}
	}()

	// Create a new collection with options. We don't provide an embedding function here, so the default embedding function will be used
	col, err := client.GetOrCreateCollection(context.Background(), "col1",
		WithCollectionMetadataCreate(
			NewMetadata(
				NewStringAttribute("str", "hello2"),
				NewIntAttribute("int", 1),
				NewFloatAttribute("float", 1.1),
			),
		),
	)
	_, _ = col.Search(context.Background(),
		NewSearchRequest(
			WithPage( // I dont think I like the nesting here
				WithOffset(1),
				WithLimit(10),
			),
			WithSelectAll(),       // select all standard fields
			WithSelect(K("test")), // select custom metadata field "test"
			WithFilter(
				Or(
					EqInt("int_key", 1),
					EqString("str_key", "hello2"),
				),
			),
			WithKnnRank(
				KnnQueryText("test"),
				WithKnnLimit(10), //n_results
				WithKnnKey(KEmbedding),
				WithKnnDefault(0.1),
				WithKnnReturnRank(),
			),
			WithRffRank(
				WithRffRanks(
					NewKnnRank(
						KnnQueryText("scientific papers"),
						WithKnnLimit(50),
						WithKnnDefault(1000.0),
						WithKnnKey("sparse_embedding"),
					).Multiply(FloatOperand(0.5)).WithWeight(0.5),
					NewKnnRank(
						KnnQueryText("AI research"),
						WithKnnLimit(100),
					).Multiply(FloatOperand(0.5)).WithWeight(0.5),
				),
				WithRffK(100),
				WithRffNormalize(),
			),
		),
	)
}
