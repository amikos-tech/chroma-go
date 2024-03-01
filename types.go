package chroma

import (
	"fmt"

	"github.com/amikos-tech/chroma-go/types"
	//nolint:gci
	//nolint:gci
)

type ErrNoDocumentOrEmbedding struct{}

func (e *ErrNoDocumentOrEmbedding) Error() string {
	return "Document or URI or Embedding must be provided"
}

type CollectionQueryBuilder struct {
	QueryTexts      []string
	QueryEmbeddings []*types.Embedding
	Where           map[string]interface{}
	WhereDocument   map[string]interface{}
	NResults        int32
	Include         []types.QueryEnum
	Offset          int32
	Limit           int32
	Ids             []string
}

type CollectionQueryOption func(*CollectionQueryBuilder) error

func WithWhere(where map[string]interface{}) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		// TODO validate where
		c.Where = where
		return nil
	}
}

func WithWhereDocument(where map[string]interface{}) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		// TODO validate where
		c.WhereDocument = where
		return nil
	}
}

func WithNResults(nResults int32) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		if nResults < 1 {
			return fmt.Errorf("nResults must be greater than 0")
		}
		c.NResults = nResults
		return nil
	}
}

func WithQueryText(queryText string) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		if queryText == "" {
			return fmt.Errorf("queryText must not be empty")
		}
		c.QueryTexts = append(c.QueryTexts, queryText)
		return nil
	}
}

func WithQueryTexts(queryTexts []string) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		if len(queryTexts) == 0 {
			return fmt.Errorf("queryTexts must not be empty")
		}
		c.QueryTexts = queryTexts
		return nil
	}
}

func WithQueryEmbeddings(queryEmbeddings []*types.Embedding) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		for _, embedding := range queryEmbeddings {
			if embedding == nil || !embedding.IsDefined() {
				return fmt.Errorf("embedding must not be nil or empty")
			}
		}
		c.QueryEmbeddings = append(c.QueryEmbeddings, queryEmbeddings...)
		return nil
	}
}

func WithQueryEmbedding(queryEmbedding *types.Embedding) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		if queryEmbedding == nil {
			return fmt.Errorf("embedding must not be empty")
		}
		c.QueryEmbeddings = append(c.QueryEmbeddings, queryEmbedding)
		return nil
	}
}

func WithInclude(include ...types.QueryEnum) CollectionQueryOption {
	return func(c *CollectionQueryBuilder) error {
		c.Include = include
		return nil
	}
}

func WithOffset(offset int32) CollectionQueryOption {
	return func(q *CollectionQueryBuilder) error {
		if offset < 0 {
			return fmt.Errorf("offset must be greater than or equal to 0")
		}
		q.Offset = offset
		return nil
	}
}

func WithLimit(limit int32) CollectionQueryOption {
	return func(q *CollectionQueryBuilder) error {
		if limit < 1 {
			return fmt.Errorf("limit must be greater than 0")
		}
		q.Limit = limit
		return nil
	}
}

func WithIds(ids []string) CollectionQueryOption {
	return func(q *CollectionQueryBuilder) error {
		q.Ids = ids
		return nil
	}
}
