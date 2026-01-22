//go:build basicv2 && !cloud

package v2

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// TestDeprecatedOptions tests backward compatibility of deprecated option functions.
// These tests ensure the deprecated functions still work while users migrate to unified options.
// This file can be deleted when the deprecated functions are removed.

func TestDeprecatedIDsOptions(t *testing.T) {
	t.Run("WithIDsGet delegates to WithIDs", func(t *testing.T) {
		op := &CollectionGetOp{}
		err := WithIDsGet("id1", "id2").ApplyToGet(op)
		require.NoError(t, err)
		require.Equal(t, []DocumentID{"id1", "id2"}, op.Ids)
	})

	t.Run("WithIDsQuery delegates to WithIDs", func(t *testing.T) {
		op := &CollectionQueryOp{}
		err := WithIDsQuery("id1", "id2").ApplyToQuery(op)
		require.NoError(t, err)
		require.Equal(t, []DocumentID{"id1", "id2"}, op.Ids)
	})

	t.Run("WithIDsDelete delegates to WithIDs", func(t *testing.T) {
		op := &CollectionDeleteOp{}
		err := WithIDsDelete("id1", "id2").ApplyToDelete(op)
		require.NoError(t, err)
		require.Equal(t, []DocumentID{"id1", "id2"}, op.Ids)
	})

	t.Run("WithIDsUpdate delegates to WithIDs", func(t *testing.T) {
		op := &CollectionUpdateOp{}
		err := WithIDsUpdate("id1", "id2").ApplyToUpdate(op)
		require.NoError(t, err)
		require.Equal(t, []DocumentID{"id1", "id2"}, op.Ids)
	})

	t.Run("WithFilterIDs delegates to WithIDs", func(t *testing.T) {
		req := &SearchRequest{}
		err := WithFilterIDs("id1", "id2").ApplyToSearchRequest(req)
		require.NoError(t, err)
		require.NotNil(t, req.Filter)
		require.Equal(t, []DocumentID{"id1", "id2"}, req.Filter.IDs)
	})
}

func TestDeprecatedWhereOptions(t *testing.T) {
	where := EqString(K("key"), "value")

	t.Run("WithWhereGet delegates to WithWhere", func(t *testing.T) {
		op := &CollectionGetOp{}
		err := WithWhereGet(where).ApplyToGet(op)
		require.NoError(t, err)
		require.NotNil(t, op.Where)
	})

	t.Run("WithWhereQuery delegates to WithWhere", func(t *testing.T) {
		op := &CollectionQueryOp{}
		err := WithWhereQuery(where).ApplyToQuery(op)
		require.NoError(t, err)
		require.NotNil(t, op.Where)
	})

	t.Run("WithWhereDelete delegates to WithWhere", func(t *testing.T) {
		op := &CollectionDeleteOp{}
		err := WithWhereDelete(where).ApplyToDelete(op)
		require.NoError(t, err)
		require.NotNil(t, op.Where)
	})
}

func TestDeprecatedWhereDocumentOptions(t *testing.T) {
	whereDoc := Contains("test")

	t.Run("WithWhereDocumentGet delegates to WithWhereDocument", func(t *testing.T) {
		op := &CollectionGetOp{}
		err := WithWhereDocumentGet(whereDoc).ApplyToGet(op)
		require.NoError(t, err)
		require.NotNil(t, op.WhereDocument)
	})

	t.Run("WithWhereDocumentQuery delegates to WithWhereDocument", func(t *testing.T) {
		op := &CollectionQueryOp{}
		err := WithWhereDocumentQuery(whereDoc).ApplyToQuery(op)
		require.NoError(t, err)
		require.NotNil(t, op.WhereDocument)
	})

	t.Run("WithWhereDocumentDelete delegates to WithWhereDocument", func(t *testing.T) {
		op := &CollectionDeleteOp{}
		err := WithWhereDocumentDelete(whereDoc).ApplyToDelete(op)
		require.NoError(t, err)
		require.NotNil(t, op.WhereDocument)
	})
}

func TestDeprecatedIncludeOptions(t *testing.T) {
	t.Run("WithIncludeGet delegates to WithInclude", func(t *testing.T) {
		op := &CollectionGetOp{}
		err := WithIncludeGet(IncludeDocuments, IncludeMetadatas).ApplyToGet(op)
		require.NoError(t, err)
		require.Equal(t, []Include{IncludeDocuments, IncludeMetadatas}, op.Include)
	})

	t.Run("WithIncludeQuery delegates to WithInclude", func(t *testing.T) {
		op := &CollectionQueryOp{}
		err := WithIncludeQuery(IncludeDocuments, IncludeMetadatas).ApplyToQuery(op)
		require.NoError(t, err)
		require.Equal(t, []Include{IncludeDocuments, IncludeMetadatas}, op.Include)
	})
}

func TestDeprecatedLimitOffsetOptions(t *testing.T) {
	t.Run("WithLimitGet delegates to WithLimit", func(t *testing.T) {
		op := &CollectionGetOp{}
		err := WithLimitGet(100).ApplyToGet(op)
		require.NoError(t, err)
		require.Equal(t, 100, op.Limit)
	})

	t.Run("WithOffsetGet delegates to WithOffset", func(t *testing.T) {
		op := &CollectionGetOp{}
		err := WithOffsetGet(50).ApplyToGet(op)
		require.NoError(t, err)
		require.Equal(t, 50, op.Offset)
	})
}

func TestDeprecatedUpdateOptions(t *testing.T) {
	t.Run("WithTextsUpdate delegates to WithTexts", func(t *testing.T) {
		op := &CollectionUpdateOp{}
		err := WithTextsUpdate("doc1", "doc2").ApplyToUpdate(op)
		require.NoError(t, err)
		require.Len(t, op.Documents, 2)
	})

	t.Run("WithMetadatasUpdate delegates to WithMetadatas", func(t *testing.T) {
		op := &CollectionUpdateOp{}
		meta := NewDocumentMetadata(NewStringAttribute("key", "value"))
		err := WithMetadatasUpdate(meta).ApplyToUpdate(op)
		require.NoError(t, err)
		require.Len(t, op.Metadatas, 1)
	})

	t.Run("WithEmbeddingsUpdate delegates to WithEmbeddings", func(t *testing.T) {
		op := &CollectionUpdateOp{}
		emb := embeddings.NewEmbeddingFromFloat32([]float32{1.0, 2.0, 3.0})
		err := WithEmbeddingsUpdate(emb).ApplyToUpdate(op)
		require.NoError(t, err)
		require.Len(t, op.Embeddings, 1)
	})
}
