//go:build basicv2 && !cloud

package v2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithIDsGet(t *testing.T) {
	opt := WithIDs("id1", "id2", "id3")

	op := &CollectionGetOp{}
	err := opt.ApplyToGet(op)
	require.NoError(t, err)
	require.Equal(t, []DocumentID{"id1", "id2", "id3"}, op.Ids)
}

func TestWithIDsQuery(t *testing.T) {
	opt := WithIDs("id1", "id2")

	op := &CollectionQueryOp{}
	err := opt.ApplyToQuery(op)
	require.NoError(t, err)
	require.Equal(t, []DocumentID{"id1", "id2"}, op.Ids)
}

func TestWithIDsDelete(t *testing.T) {
	opt := WithIDs("id1")

	op := &CollectionDeleteOp{}
	err := opt.ApplyToDelete(op)
	require.NoError(t, err)
	require.Equal(t, []DocumentID{"id1"}, op.Ids)
}

func TestWithIDsAdd(t *testing.T) {
	opt := WithIDs("id1", "id2")

	op := &CollectionAddOp{}
	err := opt.ApplyToAdd(op)
	require.NoError(t, err)
	require.Equal(t, []DocumentID{"id1", "id2"}, op.Ids)
}

func TestWithIDsUpdate(t *testing.T) {
	opt := WithIDs("id1", "id2", "id3", "id4")

	op := &CollectionUpdateOp{}
	err := opt.ApplyToUpdate(op)
	require.NoError(t, err)
	require.Equal(t, []DocumentID{"id1", "id2", "id3", "id4"}, op.Ids)
}

func TestWithIDsSearch(t *testing.T) {
	opt := WithIDs("id1", "id2")

	req := &SearchRequest{}
	err := opt.ApplyToSearchRequest(req)
	require.NoError(t, err)
	require.NotNil(t, req.Filter)
	require.Equal(t, []DocumentID{"id1", "id2"}, req.Filter.IDs)
}

func TestWithIDsAppends(t *testing.T) {
	opt1 := WithIDs("id1", "id2")
	opt2 := WithIDs("id3", "id4")

	op := &CollectionGetOp{}
	require.NoError(t, opt1.ApplyToGet(op))
	require.NoError(t, opt2.ApplyToGet(op))

	require.Equal(t, []DocumentID{"id1", "id2", "id3", "id4"}, op.Ids)
}

func TestWithWhereGet(t *testing.T) {
	filter := EqString("status", "active")
	opt := WithWhere(filter)

	op := &CollectionGetOp{}
	err := opt.ApplyToGet(op)
	require.NoError(t, err)
	require.Equal(t, filter, op.Where)
}

func TestWithWhereQuery(t *testing.T) {
	filter := GtInt("count", 10)
	opt := WithWhere(filter)

	op := &CollectionQueryOp{}
	err := opt.ApplyToQuery(op)
	require.NoError(t, err)
	require.Equal(t, filter, op.Where)
}

func TestWithWhereDelete(t *testing.T) {
	filter := EqString("status", "deleted")
	opt := WithWhere(filter)

	op := &CollectionDeleteOp{}
	err := opt.ApplyToDelete(op)
	require.NoError(t, err)
	require.Equal(t, filter, op.Where)
}

func TestWithWhereDocumentGet(t *testing.T) {
	filter := Contains("machine learning")
	opt := WithWhereDocument(filter)

	op := &CollectionGetOp{}
	err := opt.ApplyToGet(op)
	require.NoError(t, err)
	require.Equal(t, filter, op.WhereDocument)
}

func TestWithWhereDocumentQuery(t *testing.T) {
	filter := NotContains("deprecated")
	opt := WithWhereDocument(filter)

	op := &CollectionQueryOp{}
	err := opt.ApplyToQuery(op)
	require.NoError(t, err)
	require.Equal(t, filter, op.WhereDocument)
}

func TestWithWhereDocumentDelete(t *testing.T) {
	filter := Contains("old data")
	opt := WithWhereDocument(filter)

	op := &CollectionDeleteOp{}
	err := opt.ApplyToDelete(op)
	require.NoError(t, err)
	require.Equal(t, filter, op.WhereDocument)
}

func TestWithSearchWhereSearch(t *testing.T) {
	filter := EqString(K("status"), "published")
	opt := WithSearchWhere(filter)

	req := &SearchRequest{}
	err := opt.ApplyToSearchRequest(req)
	require.NoError(t, err)
	require.NotNil(t, req.Filter)
	require.Equal(t, filter, req.Filter.Where)
}

func TestFilterIDOpAppendIDs(t *testing.T) {
	op := &FilterIDOp{}
	op.AppendIDs("id1", "id2")
	require.Equal(t, []DocumentID{"id1", "id2"}, op.Ids)

	op.AppendIDs("id3")
	require.Equal(t, []DocumentID{"id1", "id2", "id3"}, op.Ids)
}

func TestFilterOpSetWhere(t *testing.T) {
	op := &FilterOp{}
	filter := EqString("key", "value")
	op.SetWhere(filter)
	require.Equal(t, filter, op.Where)
}

func TestFilterOpSetWhereDocument(t *testing.T) {
	op := &FilterOp{}
	filter := Contains("text")
	op.SetWhereDocument(filter)
	require.Equal(t, filter, op.WhereDocument)
}

func TestSearchFilterAppendIDs(t *testing.T) {
	filter := &SearchFilter{}
	filter.AppendIDs("id1", "id2")
	require.Equal(t, []DocumentID{"id1", "id2"}, filter.IDs)

	filter.AppendIDs("id3")
	require.Equal(t, []DocumentID{"id1", "id2", "id3"}, filter.IDs)
}

func TestSearchFilterSetSearchWhere(t *testing.T) {
	filter := &SearchFilter{}
	where := EqString(K("status"), "active")
	filter.SetSearchWhere(where)
	require.Equal(t, where, filter.Where)
}

func TestCombinedOptionsGet(t *testing.T) {
	op := &CollectionGetOp{}
	require.NoError(t, WithIDs("id1", "id2").ApplyToGet(op))
	require.NoError(t, WithWhere(EqString("status", "active")).ApplyToGet(op))
	require.NoError(t, WithWhereDocument(Contains("test")).ApplyToGet(op))

	require.Equal(t, []DocumentID{"id1", "id2"}, op.Ids)
	require.NotNil(t, op.Where)
	require.NotNil(t, op.WhereDocument)
}

func TestCombinedOptionsQuery(t *testing.T) {
	op := &CollectionQueryOp{}
	require.NoError(t, WithIDs("id1").ApplyToQuery(op))
	require.NoError(t, WithWhere(GtFloat("score", 0.5)).ApplyToQuery(op))

	require.Equal(t, []DocumentID{"id1"}, op.Ids)
	require.NotNil(t, op.Where)
}

func TestCombinedOptionsSearch(t *testing.T) {
	req := &SearchRequest{}
	require.NoError(t, WithIDs("id1", "id2").ApplyToSearchRequest(req))
	require.NoError(t, WithSearchWhere(EqString(K("category"), "tech")).ApplyToSearchRequest(req))

	require.NotNil(t, req.Filter)
	require.Equal(t, []DocumentID{"id1", "id2"}, req.Filter.IDs)
	require.NotNil(t, req.Filter.Where)
}

func TestDeprecatedFunctionsStillWork(t *testing.T) {
	t.Run("WithIDsGet", func(t *testing.T) {
		op := &CollectionGetOp{}
		err := WithIDsGet("id1", "id2").ApplyToGet(op)
		require.NoError(t, err)
		require.Equal(t, []DocumentID{"id1", "id2"}, op.Ids)
	})

	t.Run("WithIDsQuery", func(t *testing.T) {
		op := &CollectionQueryOp{}
		err := WithIDsQuery("id1", "id2").ApplyToQuery(op)
		require.NoError(t, err)
		require.Equal(t, []DocumentID{"id1", "id2"}, op.Ids)
	})

	t.Run("WithIDsDelete", func(t *testing.T) {
		op := &CollectionDeleteOp{}
		err := WithIDsDelete("id1", "id2").ApplyToDelete(op)
		require.NoError(t, err)
		require.Equal(t, []DocumentID{"id1", "id2"}, op.Ids)
	})

	t.Run("WithIDsUpdate", func(t *testing.T) {
		op := &CollectionUpdateOp{}
		err := WithIDsUpdate("id1", "id2").ApplyToUpdate(op)
		require.NoError(t, err)
		require.Equal(t, []DocumentID{"id1", "id2"}, op.Ids)
	})

	t.Run("WithWhereGetDeprecated", func(t *testing.T) {
		op := &CollectionGetOp{}
		filter := EqString("key", "value")
		err := WithWhereGet(filter).ApplyToGet(op)
		require.NoError(t, err)
		require.Equal(t, filter, op.Where)
	})

	t.Run("WithWhereQueryDeprecated", func(t *testing.T) {
		op := &CollectionQueryOp{}
		filter := EqString("key", "value")
		err := WithWhereQuery(filter).ApplyToQuery(op)
		require.NoError(t, err)
		require.Equal(t, filter, op.Where)
	})

	t.Run("WithWhereDeleteDeprecated", func(t *testing.T) {
		op := &CollectionDeleteOp{}
		filter := EqString("key", "value")
		err := WithWhereDelete(filter).ApplyToDelete(op)
		require.NoError(t, err)
		require.Equal(t, filter, op.Where)
	})

	t.Run("WithWhereDocumentGetDeprecated", func(t *testing.T) {
		op := &CollectionGetOp{}
		filter := Contains("text")
		err := WithWhereDocumentGet(filter).ApplyToGet(op)
		require.NoError(t, err)
		require.Equal(t, filter, op.WhereDocument)
	})

	t.Run("WithWhereDocumentQueryDeprecated", func(t *testing.T) {
		op := &CollectionQueryOp{}
		filter := Contains("text")
		err := WithWhereDocumentQuery(filter).ApplyToQuery(op)
		require.NoError(t, err)
		require.Equal(t, filter, op.WhereDocument)
	})

	t.Run("WithWhereDocumentDeleteDeprecated", func(t *testing.T) {
		op := &CollectionDeleteOp{}
		filter := Contains("text")
		err := WithWhereDocumentDelete(filter).ApplyToDelete(op)
		require.NoError(t, err)
		require.Equal(t, filter, op.WhereDocument)
	})

	t.Run("WithFilterIDs", func(t *testing.T) {
		req := &SearchRequest{}
		err := WithFilterIDs("id1", "id2").ApplyToSearchRequest(req)
		require.NoError(t, err)
		require.NotNil(t, req.Filter)
		require.Equal(t, []DocumentID{"id1", "id2"}, req.Filter.IDs)
	})
}

func TestUnifiedOptionsInNewCollectionGetOp(t *testing.T) {
	op, err := NewCollectionGetOp(
		WithIDs("id1", "id2"),
		WithWhere(EqString("status", "active")),
		WithWhereDocument(Contains("test")),
		WithInclude(IncludeDocuments, IncludeMetadatas),
		WithLimit(10),
		WithOffset(5),
	)
	require.NoError(t, err)
	require.Equal(t, []DocumentID{"id1", "id2"}, op.Ids)
	require.NotNil(t, op.Where)
	require.NotNil(t, op.WhereDocument)
	require.Equal(t, []Include{IncludeDocuments, IncludeMetadatas}, op.Include)
	require.Equal(t, 10, op.Limit)
	require.Equal(t, 5, op.Offset)
}

func TestUnifiedOptionsInNewCollectionQueryOp(t *testing.T) {
	op, err := NewCollectionQueryOp(
		WithIDs("id1"),
		WithWhere(GtInt("count", 5)),
		WithQueryTexts("hello world"),
		WithNResults(20),
		WithInclude(IncludeEmbeddings),
	)
	require.NoError(t, err)
	require.Equal(t, []DocumentID{"id1"}, op.Ids)
	require.NotNil(t, op.Where)
	require.Equal(t, []string{"hello world"}, op.QueryTexts)
	require.Equal(t, 20, op.NResults)
	require.Equal(t, []Include{IncludeEmbeddings}, op.Include)
}

func TestUnifiedOptionsInNewCollectionDeleteOp(t *testing.T) {
	op, err := NewCollectionDeleteOp(
		WithIDs("id1", "id2"),
		WithWhere(EqString("status", "deleted")),
	)
	require.NoError(t, err)
	require.Equal(t, []DocumentID{"id1", "id2"}, op.Ids)
	require.NotNil(t, op.Where)
}

func TestUnifiedOptionsInNewCollectionAddOp(t *testing.T) {
	op, err := NewCollectionAddOp(
		WithIDs("id1", "id2"),
		WithTexts("doc1", "doc2"),
	)
	require.NoError(t, err)
	require.Equal(t, []DocumentID{"id1", "id2"}, op.Ids)
	require.Len(t, op.Documents, 2)
}

func TestEarlyValidationEmptyIDs(t *testing.T) {
	t.Run("empty IDs for Query returns error", func(t *testing.T) {
		op := &CollectionQueryOp{}
		err := WithIDs().ApplyToQuery(op)
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one id is required")
	})

	t.Run("empty IDs for Delete returns error", func(t *testing.T) {
		op := &CollectionDeleteOp{}
		err := WithIDs().ApplyToDelete(op)
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one id is required")
	})

	t.Run("empty IDs for Add returns error", func(t *testing.T) {
		op := &CollectionAddOp{}
		err := WithIDs().ApplyToAdd(op)
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one id is required")
	})

	t.Run("empty IDs for Update returns error", func(t *testing.T) {
		op := &CollectionUpdateOp{}
		err := WithIDs().ApplyToUpdate(op)
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one id is required")
	})

	t.Run("empty IDs for Get is allowed", func(t *testing.T) {
		op := &CollectionGetOp{}
		err := WithIDs().ApplyToGet(op)
		require.NoError(t, err)
		require.Empty(t, op.Ids)
	})

	t.Run("empty IDs for Search returns error", func(t *testing.T) {
		req := &SearchRequest{}
		err := WithIDs().ApplyToSearchRequest(req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one id is required")
	})
}

func TestEarlyValidationInvalidWhereFilter(t *testing.T) {
	t.Run("invalid where filter for Get returns error", func(t *testing.T) {
		invalidFilter := EqString("", "value")
		op := &CollectionGetOp{}
		err := WithWhere(invalidFilter).ApplyToGet(op)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid key")
	})

	t.Run("invalid where filter for Query returns error", func(t *testing.T) {
		invalidFilter := EqString("", "value")
		op := &CollectionQueryOp{}
		err := WithWhere(invalidFilter).ApplyToQuery(op)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid key")
	})

	t.Run("invalid where filter for Delete returns error", func(t *testing.T) {
		invalidFilter := EqString("", "value")
		op := &CollectionDeleteOp{}
		err := WithWhere(invalidFilter).ApplyToDelete(op)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid key")
	})

	t.Run("nil where filter is allowed", func(t *testing.T) {
		op := &CollectionGetOp{}
		err := WithWhere(nil).ApplyToGet(op)
		require.NoError(t, err)
	})
}

func TestEarlyValidationInvalidWhereDocumentFilter(t *testing.T) {
	t.Run("invalid where document filter for Get returns error", func(t *testing.T) {
		invalidFilter := OrDocument() // empty Or is invalid
		op := &CollectionGetOp{}
		err := WithWhereDocument(invalidFilter).ApplyToGet(op)
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected at least one")
	})

	t.Run("invalid where document filter for Query returns error", func(t *testing.T) {
		invalidFilter := OrDocument() // empty Or is invalid
		op := &CollectionQueryOp{}
		err := WithWhereDocument(invalidFilter).ApplyToQuery(op)
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected at least one")
	})

	t.Run("invalid where document filter for Delete returns error", func(t *testing.T) {
		invalidFilter := OrDocument() // empty Or is invalid
		op := &CollectionDeleteOp{}
		err := WithWhereDocument(invalidFilter).ApplyToDelete(op)
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected at least one")
	})

	t.Run("nil where document filter is allowed", func(t *testing.T) {
		op := &CollectionGetOp{}
		err := WithWhereDocument(nil).ApplyToGet(op)
		require.NoError(t, err)
	})
}

func TestEarlyValidationInvalidSearchWhereFilter(t *testing.T) {
	t.Run("invalid search where filter returns error", func(t *testing.T) {
		invalidFilter := EqString(K(""), "value")
		req := &SearchRequest{}
		err := WithSearchWhere(invalidFilter).ApplyToSearchRequest(req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid key")
	})

	t.Run("nil search where filter is allowed", func(t *testing.T) {
		req := &SearchRequest{}
		err := WithSearchWhere(nil).ApplyToSearchRequest(req)
		require.NoError(t, err)
	})
}
