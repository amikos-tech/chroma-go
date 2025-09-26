package v2

import (
	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// Simplified Collection Options - These can be used across multiple operations
// where applicable, reducing the cognitive load of remembering operation-specific suffixes.

// WithIDsUnified sets document IDs for Add, Update, Delete, Get, and Query operations.
// This replaces WithIDsGet, WithIDsDelete, WithIDsQuery, WithIDs, WithIDsUpdate.
func WithIDsUnified(ids ...DocumentID) interface{} {
	return multiOption{
		addOpt: func(op *CollectionAddOp) error {
			op.Ids = append(op.Ids, ids...)
			return nil
		},
		updateOpt: func(op *CollectionUpdateOp) error {
			op.Ids = append(op.Ids, ids...)
			return nil
		},
		deleteOpt: func(op *CollectionDeleteOp) error {
			op.Ids = append(op.Ids, ids...)
			return nil
		},
		getOpt: func(op *CollectionGetOp) error {
			op.Ids = append(op.Ids, ids...)
			return nil
		},
		queryOpt: func(op *CollectionQueryOp) error {
			if len(ids) == 0 {
				return errors.New("at least one id is required")
			}
			if op.Ids == nil {
				op.Ids = make([]DocumentID, 0)
			}
			op.Ids = append(op.Ids, ids...)
			return nil
		},
	}
}

// WithDocuments sets documents for Add and Update operations.
// This replaces WithTexts and WithTextsUpdate for better naming consistency.
// For now, use WithTexts for Add operations and WithTextsUpdate for Update operations.
func WithDocuments(documents ...string) interface{} {
	return multiOption{
		addOpt: func(op *CollectionAddOp) error {
			if len(documents) == 0 {
				return errors.New("at least one document is required")
			}
			if op.Documents == nil {
				op.Documents = make([]Document, 0)
			}
			for _, text := range documents {
				op.Documents = append(op.Documents, NewTextDocument(text))
			}
			return nil
		},
		updateOpt: func(op *CollectionUpdateOp) error {
			if len(documents) == 0 {
				return errors.New("at least one document is required")
			}
			if op.Documents == nil {
				op.Documents = make([]Document, 0)
			}
			for _, text := range documents {
				op.Documents = append(op.Documents, NewTextDocument(text))
			}
			return nil
		},
	}
}

// WithWhere sets where filters for Get, Query, and Delete operations.
// This replaces WithWhereGet, WithWhereQuery, and WithWhereDelete.
func WithWhere(where WhereFilter) interface{} {
	return multiOption{
		getOpt: func(op *CollectionGetOp) error {
			op.Where = where
			return nil
		},
		queryOpt: func(op *CollectionQueryOp) error {
			op.Where = where
			return nil
		},
		deleteOpt: func(op *CollectionDeleteOp) error {
			op.Where = where
			return nil
		},
	}
}

// WithWhereDocument sets document filters for Get, Query, and Delete operations.
// This replaces WithWhereDocumentGet, WithWhereDocumentQuery, and WithWhereDocumentDelete.
func WithWhereDocument(whereDocument WhereDocumentFilter) interface{} {
	return multiOption{
		getOpt: func(op *CollectionGetOp) error {
			op.WhereDocument = whereDocument
			return nil
		},
		queryOpt: func(op *CollectionQueryOp) error {
			op.WhereDocument = whereDocument
			return nil
		},
		deleteOpt: func(op *CollectionDeleteOp) error {
			op.WhereDocument = whereDocument
			return nil
		},
	}
}

// WithInclude sets what to include in Get and Query results.
// This replaces WithIncludeGet and WithIncludeQuery.
func WithInclude(include ...Include) interface{} {
	return multiOption{
		getOpt: func(op *CollectionGetOp) error {
			op.Include = include
			return nil
		},
		queryOpt: func(op *CollectionQueryOp) error {
			op.Include = include
			return nil
		},
	}
}

// WithMetadatasUnified sets document metadata for Add and Update operations.
// This replaces WithMetadatas and WithMetadatasUpdate.
func WithMetadatasUnified(metadatas ...DocumentMetadata) interface{} {
	return multiOption{
		addOpt: func(op *CollectionAddOp) error {
			op.Metadatas = metadatas
			return nil
		},
		updateOpt: func(op *CollectionUpdateOp) error {
			op.Metadatas = metadatas
			return nil
		},
	}
}

// WithEmbeddingsUnified sets embeddings for Add and Update operations.
// This replaces WithEmbeddings and WithEmbeddingsUpdate.
func WithEmbeddingsUnified(embeddings ...embeddings.Embedding) interface{} {
	return multiOption{
		addOpt: func(op *CollectionAddOp) error {
			if len(embeddings) == 0 {
				return errors.New("at least one embedding is required")
			}
			embds := make([]any, 0)
			for _, e := range embeddings {
				embds = append(embds, e)
			}
			op.Embeddings = embds
			return nil
		},
		updateOpt: func(op *CollectionUpdateOp) error {
			if len(embeddings) == 0 {
				return errors.New("at least one embedding is required")
			}
			embds := make([]any, 0)
			for _, e := range embeddings {
				embds = append(embds, e)
			}
			op.Embeddings = embds
			return nil
		},
	}
}

// Collection Creation Simplified Options

// WithMetadata sets metadata when creating a collection.
// This replaces WithCollectionMetadataCreate.
func WithMetadata(metadata CollectionMetadata) CreateCollectionOption {
	return WithCollectionMetadataCreate(metadata)
}

// WithEmbeddingFunction sets the embedding function for collection operations.
// This replaces WithEmbeddingFunctionCreate and WithEmbeddingFunctionGet.
func WithEmbeddingFunction(ef embeddings.EmbeddingFunction) interface{} {
	return multiOption{
		createOpt: func(op *CreateCollectionOp) error {
			if ef == nil {
				return errors.New("embeddingFunction cannot be nil")
			}
			op.embeddingFunction = ef
			return nil
		},
		getCollectionOpt: func(op *GetCollectionOp) error {
			if ef == nil {
				return errors.New("embedding function cannot be nil")
			}
			op.embeddingFunction = ef
			return nil
		},
	}
}

// WithDatabase sets the database for collection operations.
// This replaces WithDatabaseCreate, WithDatabaseGet, WithDatabaseDelete, etc.
func WithDatabase(database Database) interface{} {
	return multiOption{
		createOpt: func(op *CreateCollectionOp) error {
			if database == nil {
				return errors.New("database cannot be nil")
			}
			err := database.Validate()
			if err != nil {
				return errors.Wrap(err, "error validating database")
			}
			op.Database = database
			return nil
		},
		getCollectionOpt: func(op *GetCollectionOp) error {
			if database == nil {
				return errors.New("database cannot be nil")
			}
			err := database.Validate()
			if err != nil {
				return errors.Wrap(err, "error validating database")
			}
			op.Database = database
			return nil
		},
		deleteCollectionOpt: func(op *DeleteCollectionOp) error {
			if database == nil {
				return errors.New("database cannot be nil")
			}
			err := database.Validate()
			if err != nil {
				return errors.Wrap(err, "error validating database")
			}
			op.Database = database
			return nil
		},
		listOpt: func(op *ListCollectionOp) error {
			if database == nil {
				return errors.New("database cannot be nil")
			}
			err := database.Validate()
			if err != nil {
				return errors.Wrap(err, "error validating database")
			}
			op.Database = database
			return nil
		},
		countOpt: func(op *CountCollectionsOp) error {
			if database == nil {
				return errors.New("database cannot be nil")
			}
			err := database.Validate()
			if err != nil {
				return errors.Wrap(err, "error validating database")
			}
			op.Database = database
			return nil
		},
	}
}

// WithCreateIfNotExists enables get-or-create behavior when creating a collection.
// This replaces WithIfNotExistsCreate.
func WithCreateIfNotExists() CreateCollectionOption {
	return WithIfNotExistsCreate()
}

// Query Simplified Options

// WithLimit sets the result limit for Query operations.
// This replaces WithNResults for better naming consistency.
func WithLimit(limit int) CollectionQueryOption {
	return WithNResults(limit)
}

// WithQueryText sets query texts for similarity search.
// Simplified version of WithQueryTexts for single text queries.
func WithQueryText(text string) CollectionQueryOption {
	return WithQueryTexts(text)
}

// WithQueryEmbedding sets a single query embedding for similarity search.
// Simplified version of WithQueryEmbeddings for single embedding queries.
func WithQueryEmbedding(embedding embeddings.Embedding) CollectionQueryOption {
	return WithQueryEmbeddings(embedding)
}

// multiOption allows a single option to work with multiple operation types
type multiOption struct {
	addOpt              CollectionAddOption
	updateOpt           CollectionUpdateOption
	deleteOpt           CollectionDeleteOption
	getOpt              CollectionGetOption
	queryOpt            CollectionQueryOption
	createOpt           CreateCollectionOption
	getCollectionOpt    GetCollectionOption
	deleteCollectionOpt DeleteCollectionOption
	listOpt             ListCollectionsOption
	countOpt            CountCollectionsOption
}

// Implement all option interfaces
func (m multiOption) applyAdd(op *CollectionAddOp) error {
	if m.addOpt != nil {
		return m.addOpt(op)
	}
	return nil
}

func (m multiOption) applyUpdate(op *CollectionUpdateOp) error {
	if m.updateOpt != nil {
		return m.updateOpt(op)
	}
	return nil
}

func (m multiOption) applyDelete(op *CollectionDeleteOp) error {
	if m.deleteOpt != nil {
		return m.deleteOpt(op)
	}
	return nil
}

func (m multiOption) applyGet(op *CollectionGetOp) error {
	if m.getOpt != nil {
		return m.getOpt(op)
	}
	return nil
}

func (m multiOption) applyQuery(op *CollectionQueryOp) error {
	if m.queryOpt != nil {
		return m.queryOpt(op)
	}
	return nil
}

func (m multiOption) applyCreate(op *CreateCollectionOp) error {
	if m.createOpt != nil {
		return m.createOpt(op)
	}
	return nil
}

func (m multiOption) applyGetCollection(op *GetCollectionOp) error {
	if m.getCollectionOpt != nil {
		return m.getCollectionOpt(op)
	}
	return nil
}

func (m multiOption) applyDeleteCollection(op *DeleteCollectionOp) error {
	if m.deleteCollectionOpt != nil {
		return m.deleteCollectionOpt(op)
	}
	return nil
}

func (m multiOption) applyList(op *ListCollectionOp) error {
	if m.listOpt != nil {
		return m.listOpt(op)
	}
	return nil
}

func (m multiOption) applyCount(op *CountCollectionsOp) error {
	if m.countOpt != nil {
		return m.countOpt(op)
	}
	return nil
}
