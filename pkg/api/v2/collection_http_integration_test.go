//go:build basicv2

package v2

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcchroma "github.com/testcontainers/testcontainers-go/modules/chroma"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func TestCollectionAddIntegration(t *testing.T) {
	ctx := context.Background()
	var chromaVersion = "0.6.3"
	var chromaImage = "ghcr.io/chroma-core/chroma"
	if os.Getenv("CHROMA_VERSION") != "" {
		chromaVersion = os.Getenv("CHROMA_VERSION")
	}
	if os.Getenv("CHROMA_IMAGE") != "" {
		chromaImage = os.Getenv("CHROMA_IMAGE")
	}
	chromaContainer, err := tcchroma.Run(ctx,
		fmt.Sprintf("%s:%s", chromaImage, chromaVersion),
		testcontainers.WithEnv(map[string]string{"ALLOW_RESET": "true"}),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, chromaContainer.Terminate(ctx))
	})
	endpoint, err := chromaContainer.RESTEndpoint(context.Background())
	require.NoError(t, err)
	chromaURL := os.Getenv("CHROMA_URL")
	if chromaURL == "" {
		chromaURL = endpoint
	}
	c, err := NewHTTPClient(WithBaseURL(chromaURL), WithDebug())
	require.NoError(t, err)

	t.Run("add documents", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		count, err := collection.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 3, count)
	})
	t.Run("get documents", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		res, err := collection.Get(ctx, WithIDsGet("1", "2", "3"))
		require.NoError(t, err)
		require.Equal(t, 3, len(res.GetIDs()))
	})
	t.Run("get documents with limit and offset", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		res, err := collection.Get(ctx, WithLimitGet(1), WithOffsetGet(0))
		require.NoError(t, err)
		require.Equal(t, 1, len(res.GetIDs()))
	})
	t.Run("get documents with where", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("test_document_1", "test_document_2", "test_document_3"),
			WithMetadatas(
				NewDocumentMetadata(NewStringAttribute("test_key", "doc1")),
				NewDocumentMetadata(NewStringAttribute("test_key", "doc2")),
				NewDocumentMetadata(NewStringAttribute("test_key", "doc3")),
			),
		)
		require.NoError(t, err)
		res, err := collection.Get(ctx, WithWhereGet(EqString("test_key", "doc1")))
		require.NoError(t, err)
		require.Equal(t, 1, len(res.GetIDs()))
		require.Equal(t, "test_document_1", res.GetDocuments()[0].ContentString())
	})
	t.Run("count documents", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		count, err := collection.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 3, count)
	})

	t.Run("delete documents", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		err = collection.Delete(ctx, WithIDsDelete("1", "2", "3"))
		require.NoError(t, err)
		count, err := collection.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})
	t.Run("upsert documents", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)

		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		err = collection.Upsert(ctx, WithIDs("1", "2", "3"), WithTexts("test_document_1_updated", "test_document_2_updated", "test_document_3_updated"))
		require.NoError(t, err)
		count, err := collection.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 3, count)
		res, err := collection.Get(ctx, WithIDsGet("1", "2", "3"))
		require.NoError(t, err)
		require.Equal(t, 3, len(res.GetIDs()))
		require.Equal(t, "test_document_1_updated", res.GetDocuments()[0].ContentString())
		require.Equal(t, "test_document_2_updated", res.GetDocuments()[1].ContentString())
		require.Equal(t, "test_document_3_updated", res.GetDocuments()[2].ContentString())
	})

	t.Run("update documents", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		err = collection.Update(ctx, WithIDs("1", "2", "3"), WithTexts("test_document_1_updated", "test_document_2_updated", "test_document_3_updated"))
		require.NoError(t, err)
		count, err := collection.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 3, count)
		res, err := collection.Get(ctx, WithIDsGet("1", "2", "3"))
		require.NoError(t, err)
		require.Equal(t, 3, len(res.GetIDs()))
		require.Equal(t, "test_document_1_updated", res.GetDocuments()[0].ContentString())
		require.Equal(t, "test_document_2_updated", res.GetDocuments()[1].ContentString())
		require.Equal(t, "test_document_3_updated", res.GetDocuments()[2].ContentString())
	})
	t.Run("query documents", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		res, err := collection.Query(ctx, WithQueryTexts("test_document_1"))
		require.NoError(t, err)
		require.Equal(t, 1, len(res.GetIDGroups()))
		require.Equal(t, 3, len(res.GetIDGroups()[0]))
		require.Equal(t, "test_document_1", res.GetDocumentsGroups()[0][0].ContentString())
	})
	t.Run("query documents with where", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(
			NewUUIDGenerator()),
			WithTexts("test_document_1", "test_document_2", "test_document_3"),
			WithMetadatas(
				NewDocumentMetadata(NewStringAttribute("test_key", "doc1")),
				NewDocumentMetadata(NewStringAttribute("test_key", "doc2")),
				NewDocumentMetadata(NewStringAttribute("test_key", "doc3")),
			),
		)
		require.NoError(t, err)
		res, err := collection.Query(ctx, WithQueryTexts("test_document_1"), WithWhereQuery(EqString("test_key", "doc1")))
		require.NoError(t, err)
		require.Equal(t, 1, len(res.GetIDGroups()))
		require.Equal(t, 1, len(res.GetIDGroups()[0]))
		require.Equal(t, "test_document_1", res.GetDocumentsGroups()[0][0].ContentString())
	})
	t.Run("query documents with where document", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		res, err := collection.Query(ctx, WithQueryTexts("test_document_1"), WithWhereDocumentQuery(Contains("test_document_1")))
		require.NoError(t, err)
		require.Equal(t, 1, len(res.GetIDGroups()))
		require.Equal(t, 1, len(res.GetIDGroups()[0]))
		require.Equal(t, "test_document_1", res.GetDocumentsGroups()[0][0].ContentString())
	})
}
