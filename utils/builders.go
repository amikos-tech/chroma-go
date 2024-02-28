package utils

import (
	"context"
	"fmt"

	"github.com/amikos-tech/chroma-go"
)

type MetadataBuilder struct {
	metadata map[string]interface{}
	err      error
}

func (m *MetadataBuilder) Build() (map[string]interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.metadata, nil
}

type InvalidMetadataValueError struct {
	key   string
	value interface{}
}

func (e *InvalidMetadataValueError) Error() string {
	return fmt.Sprintf("Invalid metadata value for key %s: %v", e.key, e.value)
}

type InvalidWhereValueError struct {
	key   string
	value interface{}
}

func (e *InvalidWhereValueError) Error() string {
	return fmt.Sprintf("Invalid value for where clause for key %s: %v. Allowed values are string, int, float, bool", e.key, e.value)
}

type InvalidWhereDocumentValueError struct {
	value interface{}
}

func (e *InvalidWhereDocumentValueError) Error() string {
	return fmt.Sprintf("Invalid value for where document clause for value %v. Allowed values are string", e.value)
}

func (m *MetadataBuilder) ForValue(key string, value interface{}) *MetadataBuilder {
	if m.err != nil {
		return m
	}
	switch value.(type) {
	case string, int, float32, bool:
		m.metadata[key] = value
	default:
		m.err = &InvalidMetadataValueError{key, value}
	}
	return m
}

func NewMetadataBuilder() *MetadataBuilder {
	return &MetadataBuilder{metadata: make(map[string]interface{})}
}

type WhereBuilder struct {
	WhereClause map[string]interface{}
	err         error
}

func NewWhereBuilder() *WhereBuilder {
	return &WhereBuilder{WhereClause: make(map[string]interface{})}
}

func (w *WhereBuilder) operation(operation string, field string, value interface{}) *WhereBuilder {
	if w.err != nil {
		return w
	}
	inner := make(map[string]interface{})
	if v, ok := value.([]interface{}); ok {
		inner[operation] = v
	} else {
		switch value.(type) {
		case string, int, float32, bool:
		default:
			w.err = &InvalidWhereValueError{field, value}
			return w
		}
		inner[operation] = value
	}
	w.WhereClause[field] = inner
	return w
}
func (w *WhereBuilder) Build() (map[string]interface{}, error) {
	if w.err != nil {
		return nil, w.err
	}
	return w.WhereClause, nil
}
func (w *WhereBuilder) Eq(key string, value interface{}) *WhereBuilder {
	return w.operation("$eq", key, value)
}
func (w *WhereBuilder) Ne(key string, value interface{}) *WhereBuilder {
	return w.operation("$ne", key, value)
}
func (w *WhereBuilder) Gt(key string, value interface{}) *WhereBuilder {
	return w.operation("$gt", key, value)
}
func (w *WhereBuilder) Gte(key string, value interface{}) *WhereBuilder {
	return w.operation("$gte", key, value)
}
func (w *WhereBuilder) Lt(key string, value interface{}) *WhereBuilder {
	return w.operation("$lt", key, value)
}
func (w *WhereBuilder) Lte(key string, value interface{}) *WhereBuilder {
	return w.operation("$lte", key, value)
}
func (w *WhereBuilder) In(key string, value []interface{}) *WhereBuilder {
	return w.operation("$in", key, value)
}
func (w *WhereBuilder) Nin(key string, value []interface{}) *WhereBuilder {
	return w.operation("$nin", key, value)
}
func (w *WhereBuilder) And(builders ...*WhereBuilder) *WhereBuilder {
	if w.err != nil {
		return w
	}
	var andClause []map[string]interface{}
	for _, b := range builders {
		buildExpr, err := b.Build()
		if err != nil {
			w.err = err
			return w
		}
		andClause = append(andClause, buildExpr)
	}
	w.WhereClause["$and"] = andClause
	return w
}
func (w *WhereBuilder) Or(builders ...*WhereBuilder) *WhereBuilder {
	if w.err != nil {
		return w
	}
	var orClause []map[string]interface{}
	for _, b := range builders {
		buildExpr, err := b.Build()
		if err != nil {
			w.err = err
			return w
		}
		orClause = append(orClause, buildExpr)
	}
	w.WhereClause["$or"] = orClause
	return w
}

type WhereDocumentBuilder struct {
	WhereClause map[string]interface{}
	err         error
}

func NewWhereDocumentBuilder() *WhereDocumentBuilder {
	return &WhereDocumentBuilder{WhereClause: make(map[string]interface{})}
}

func (w *WhereDocumentBuilder) operation(operation string, value interface{}) *WhereDocumentBuilder {
	if w.err != nil {
		return w
	}
	inner := make(map[string]interface{})

	switch value.(type) {
	case string:
	default:
		w.err = &InvalidWhereDocumentValueError{value}
		return w
	}
	inner[operation] = value
	w.WhereClause[operation] = value
	return w
}

func (w *WhereDocumentBuilder) Contains(value interface{}) *WhereDocumentBuilder {
	return w.operation("$contains", value)
}

func (w *WhereDocumentBuilder) NotContains(value interface{}) *WhereDocumentBuilder {
	return w.operation("$not_contains", value)
}

func (w *WhereDocumentBuilder) And(builders ...*WhereDocumentBuilder) *WhereDocumentBuilder {
	if w.err != nil {
		return w
	}
	var andClause []map[string]interface{}
	for _, b := range builders {
		buildExpr, err := b.Build()
		if err != nil {
			w.err = err
			return w
		}
		andClause = append(andClause, buildExpr)
	}
	w.WhereClause["$and"] = andClause
	return w
}

func (w *WhereDocumentBuilder) Or(builders ...*WhereDocumentBuilder) *WhereDocumentBuilder {
	if w.err != nil {
		return w
	}
	var orClause []map[string]interface{}
	for _, b := range builders {
		buildExpr, err := b.Build()
		if err != nil {
			w.err = err
			return w
		}
		orClause = append(orClause, buildExpr)
	}
	w.WhereClause["$or"] = orClause
	return w
}

func (w *WhereDocumentBuilder) Build() (map[string]interface{}, error) {
	if w.err != nil {
		return nil, w.err
	}
	return w.WhereClause, nil
}

type CollectionBuilder struct {
	Tenant            string
	Database          string
	Name              string
	Metadata          map[string]interface{}
	CreateIfNotExist  bool
	EmbeddingFunction chroma.EmbeddingFunction
	Client            *chroma.Client
	IDGenerator       chroma.IDGenerator
	err               error
}

func NewCollectionBuilder(name string, client *chroma.Client) *CollectionBuilder {
	return &CollectionBuilder{
		Tenant:   chroma.DefaultTenant,
		Database: chroma.DefaultDatabase,
		Metadata: make(map[string]interface{}),
		Name:     name,
		Client:   client,
	}
}

func (c *CollectionBuilder) WithName(name string) *CollectionBuilder {
	if c.err != nil {
		return c
	}
	c.Name = name
	return c
}

func (c *CollectionBuilder) WithEmbeddingFunction(embeddingFunction chroma.EmbeddingFunction) *CollectionBuilder {
	if c.err != nil {
		return c
	}
	c.EmbeddingFunction = embeddingFunction
	return c
}

func (c *CollectionBuilder) WithIDGenerator(idGenerator chroma.IDGenerator) *CollectionBuilder {
	if c.err != nil {
		return c
	}
	c.IDGenerator = idGenerator
	return c
}

func (c *CollectionBuilder) WithCreateIfNotExist(create bool) *CollectionBuilder {
	if c.err != nil {
		return c
	}
	c.CreateIfNotExist = create
	return c
}

func (c *CollectionBuilder) WithHNSWDistanceFunction(df chroma.DistanceFunction) *CollectionBuilder {
	if c.err != nil {
		return c
	}
	c.WithMetadata("hnsw:space", df)
	return c
}

func (c *CollectionBuilder) WithHNSWBatchSize(batchSize int32) *CollectionBuilder {
	if c.err != nil {
		return c
	}
	c.WithMetadata("hnsw:batch_size", batchSize)
	return c
}

func (c *CollectionBuilder) WithMetadatas(metadata map[string]interface{}) *CollectionBuilder {
	if c.err != nil {
		return c
	}
	c.Metadata = metadata
	return c
}

func (c *CollectionBuilder) WithMetadata(key string, value interface{}) *CollectionBuilder {
	if c.err != nil {
		return c
	}
	if c.Metadata == nil {
		c.Metadata = make(map[string]interface{})
	}
	switch value.(type) {
	case string, int, float32, bool:
		c.Metadata[key] = value
	default:
		c.err = &InvalidMetadataValueError{key, value}
	}
	return c
}

func (c *CollectionBuilder) WithTenant(tenant string) *CollectionBuilder {
	if c.err != nil {
		return c
	}
	c.Tenant = tenant
	return c
}

func (c *CollectionBuilder) WithDatabase(database string) *CollectionBuilder {
	if c.err != nil {
		return c
	}
	c.Database = database
	return c
}

func (c *CollectionBuilder) HasError() (bool, error) {
	if c.err != nil {
		return true, c.err
	}
	return false, nil
}

func (c *CollectionBuilder) Create(ctx context.Context) (*chroma.Collection, error) {
	if c.err != nil {
		return nil, c.err
	}
	if c.Metadata["hnsw:space"] == nil {
		c.WithHNSWDistanceFunction(chroma.L2)
	}
	return c.Client.CreateCollection(ctx, c.Name, c.Metadata, c.CreateIfNotExist, c.EmbeddingFunction, c.Metadata["hnsw:space"].(chroma.DistanceFunction))
}

type CollectionQueryBuilder struct {
	Collection      *chroma.Collection
	QueryTexts      []string
	QueryEmbeddings [][]interface{}
	Where           map[string]interface{}
	WhereDoc        map[string]interface{}
	NResults        int32
	Include         []chroma.QueryEnum
	err             error
}

func NewCollectionQueryBuilder(collection *chroma.Collection) *CollectionQueryBuilder {
	return &CollectionQueryBuilder{
		Collection:      collection,
		QueryTexts:      make([]string, 0),
		QueryEmbeddings: make([][]interface{}, 0),
		Where:           make(map[string]interface{}),
		WhereDoc:        make(map[string]interface{}),
	}
}

func (c *CollectionQueryBuilder) WithWhere(where map[string]interface{}) *CollectionQueryBuilder {
	if c.err != nil {
		return c
	}
	c.Where = where
	return c
}

func (c *CollectionQueryBuilder) WithWhereDocument(where map[string]interface{}) *CollectionQueryBuilder {
	if c.err != nil {
		return c
	}
	c.WhereDoc = where
	return c
}

func (c *CollectionQueryBuilder) HasError() (bool, error) {
	if c.err != nil {
		return true, c.err
	}
	return false, nil
}

func (c *CollectionQueryBuilder) WithNResults(nResults int32) *CollectionQueryBuilder {
	if c.err != nil {
		return c
	}
	if nResults < 1 {
		c.err = fmt.Errorf("nResults must be greater than 0")
		return c
	}
	c.NResults = nResults
	return c
}

func (c *CollectionQueryBuilder) WithQueryText(queryText string) *CollectionQueryBuilder {
	if c.err != nil {
		return c
	}
	c.QueryTexts = append(c.QueryTexts, queryText)
	return c
}

func (c *CollectionQueryBuilder) WithQueryTexts(queryTexts []string) *CollectionQueryBuilder {
	if c.err != nil {
		return c
	}
	if len(c.QueryTexts) == 0 {
		c.QueryTexts = queryTexts
	} else {
		c.QueryTexts = append(c.QueryTexts, queryTexts...)
	}

	return c
}

func (c *CollectionQueryBuilder) WithQueryEmbeddings(queryEmbeddings [][]interface{}) *CollectionQueryBuilder {
	if c.err != nil {
		return c
	}
	for _, embedding := range queryEmbeddings {
		if len(embedding) == 0 {
			c.err = fmt.Errorf("embedding must not be empty")
			return c
		}
		for _, v := range embedding {
			switch v.(type) {
			case int, float32:
			default:
				c.err = fmt.Errorf("embedding must be a list of int or float32")
				return c
			}
		}
	}
	for _, embedding := range queryEmbeddings {
		c.WithQueryEmbedding(embedding)
	}
	return c
}

func (c *CollectionQueryBuilder) WithQueryEmbedding(queryEmbedding []interface{}) *CollectionQueryBuilder {
	if c.err != nil {
		return c
	}
	if len(queryEmbedding) == 0 {
		c.err = fmt.Errorf("embedding must not be empty")
		return c
	}
	for _, v := range queryEmbedding {
		switch v.(type) {
		case int, float32:
		default:
			c.err = fmt.Errorf("embedding must be a list of int or float32")
			return c
		}
	}
	c.QueryEmbeddings = append(c.QueryEmbeddings, queryEmbedding)
	return c
}

func (c *CollectionQueryBuilder) WithInclude(include ...chroma.QueryEnum) *CollectionQueryBuilder {
	if c.err != nil {
		return c
	}
	c.Include = include
	return c
}

func (c *CollectionQueryBuilder) Query(ctx context.Context) (*chroma.QueryResults, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.Collection.Query(ctx, c.QueryTexts, c.NResults, c.Where, c.WhereDoc, c.Include)
}
