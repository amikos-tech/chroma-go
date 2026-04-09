//go:build basicv2 && cloud

package v2

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/chromacloud"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/chromacloudsplade"
)

// setupCloudClient creates a cloud client for testing with proper cleanup
func setupCloudClient(t *testing.T) Client {
	t.Helper()

	if os.Getenv("CHROMA_API_KEY") == "" && os.Getenv("CHROMA_DATABASE") == "" && os.Getenv("CHROMA_TENANT") == "" {
		err := godotenv.Load("../../../.env")
		require.NoError(t, err)
	}

	client, err := NewCloudClient(
		WithLogger(testLogger()),
		WithDatabaseAndTenant(os.Getenv("CHROMA_DATABASE"), os.Getenv("CHROMA_TENANT")),
		WithCloudAPIKey(os.Getenv("CHROMA_API_KEY")),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		t.Setenv("CHROMA_TENANT", "")
		t.Setenv("CHROMA_DATABASE", "")
		t.Setenv("CHROMA_API_KEY", "")
	})

	t.Cleanup(func() {
		_ = client.Close()
	})

	return client
}

// TestCloudCleanup is a dedicated cleanup test that runs after all other cloud tests
// It removes all test collections to clean up the shared cloud database
func TestCloudCleanup(t *testing.T) {
	client := setupCloudClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	collections, err := client.ListCollections(ctx)
	if err != nil {
		t.Fatalf("Failed to list collections for cleanup: %v", err)
	}

	var toDelete []string
	for _, collection := range collections {
		name := collection.Name()
		if strings.HasPrefix(name, "test_") {
			toDelete = append(toDelete, name)
		}
	}

	if len(toDelete) == 0 {
		t.Log("No test collections to clean up")
		return
	}

	t.Logf("Cleaning up %d test collections...", len(toDelete))

	var deleted, failed int
	for _, name := range toDelete {
		deleteCtx, deleteCancel := context.WithTimeout(ctx, 10*time.Second)
		if err := client.DeleteCollection(deleteCtx, name); err != nil {
			t.Logf("Warning: failed to delete collection '%s': %v", name, err)
			failed++
		} else {
			deleted++
		}
		deleteCancel()
	}

	t.Logf("Cleanup completed: deleted %d, failed %d", deleted, failed)
}

// TestCloudClientCRUD tests basic CRUD operations
func TestCloudClientCRUD(t *testing.T) {
	client := setupCloudClient(t)

	t.Run("Get Version", func(t *testing.T) {
		ctx := context.Background()
		v, err := client.GetVersion(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, v)
		require.Contains(t, v, "1.0")
	})

	t.Run("List collections", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		collections, err := client.ListCollections(ctx)
		require.NoError(t, err)
		t.Logf("Found %d collections", len(collections))
	})

	t.Run("Count collections", func(t *testing.T) {
		ctx := context.Background()
		collectionCount, err := client.CountCollections(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, collectionCount, 0)
	})

	t.Run("Create collection", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_collection-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name())
	})

	t.Run("Delete collection", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_collection-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name())

		err = client.DeleteCollection(ctx, collectionName)
		require.NoError(t, err)

		collections, err := client.ListCollections(ctx)
		require.NoError(t, err)
		for _, c := range collections {
			require.NotEqual(t, collectionName, c.Name())
		}
	})

	t.Run("Add data to collection", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_collection-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name())

		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("this is document about cats", "123141231", "$@!123115"))
		require.NoError(t, err)
	})

	t.Run("Delete data from collection", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_collection-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name())

		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("this is document about cats", "123141231", "$@!123115"))
		require.NoError(t, err)

		err = collection.Delete(ctx, WithIDs("1", "2"))
		require.NoError(t, err)

		count, err := collection.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("Update and get data in collection", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_collection-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name())

		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("this is document about cats", "123141231", "$@!123115"))
		require.NoError(t, err)

		err = collection.Update(ctx, WithIDs("1", "2"), WithTexts("updated text for 1", "updated text for 2"))
		require.NoError(t, err)

		results, err := collection.Get(ctx, WithIDs("1", "2"))
		require.NoError(t, err)
		require.Equal(t, results.Count(), 2)
		require.Equal(t, "updated text for 1", results.GetDocuments()[0].ContentString())
		require.Equal(t, "updated text for 2", results.GetDocuments()[1].ContentString())
	})

	t.Run("Query data in collection", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_collection-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name())

		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("this is document about cats", "dogs are man's best friends", "lions are big cats"))
		require.NoError(t, err)

		results, err := collection.Query(ctx, WithQueryTexts("tell me about cats"), WithNResults(2))
		require.NoError(t, err)
		require.Contains(t, results.GetDocumentsGroups()[0][0].ContentString(), "cats")
		require.Contains(t, results.GetDocumentsGroups()[0][1].ContentString(), "cats")
	})
}

// TestCloudClientAutoWire tests embedding function auto-wiring
func TestCloudClientAutoWire(t *testing.T) {
	client := setupCloudClient(t)

	t.Run("auto-wire chroma cloud embedding function on GetCollection", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_autowire_cloud_get-" + uuid.New().String()

		ef, err := chromacloud.NewEmbeddingFunction(chromacloud.WithEnvAPIKey())
		require.NoError(t, err)
		createdCol, err := client.CreateCollection(ctx, collectionName, WithEmbeddingFunctionCreate(ef))
		require.NoError(t, err)
		require.NotNil(t, createdCol)
		require.Equal(t, collectionName, createdCol.Name())

		retrievedCol, err := client.GetCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, retrievedCol)

		err = retrievedCol.Add(ctx, WithIDs("doc1", "doc2"), WithTexts("hello world", "goodbye world"))
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		results, err := retrievedCol.Query(ctx, WithQueryTexts("hello"), WithNResults(1))
		require.NoError(t, err)
		require.NotNil(t, results)
		require.NotEmpty(t, results.GetDocumentsGroups())
	})

	t.Run("auto-wire custom embedding function on GetCollection", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_autowire_custom_get-" + uuid.New().String()

		ef := embeddings.NewConsistentHashEmbeddingFunction()
		createdCol, err := client.CreateCollection(ctx, collectionName, WithEmbeddingFunctionCreate(ef))
		require.NoError(t, err)
		require.NotNil(t, createdCol)
		require.Equal(t, collectionName, createdCol.Name())

		retrievedCol, err := client.GetCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, retrievedCol)

		err = retrievedCol.Add(ctx, WithIDs("doc1", "doc2"), WithTexts("hello world", "goodbye world"))
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		results, err := retrievedCol.Query(ctx, WithQueryTexts("hello"), WithNResults(1))
		require.NoError(t, err)
		require.NotNil(t, results)
		require.NotEmpty(t, results.GetDocumentsGroups())
	})

	t.Run("auto-wire embedding function on ListCollections", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_autowire_list-" + uuid.New().String()

		ef := embeddings.NewConsistentHashEmbeddingFunction()
		_, err := client.CreateCollection(ctx, collectionName, WithEmbeddingFunctionCreate(ef))
		require.NoError(t, err)

		collections, err := client.ListCollections(ctx)
		require.NoError(t, err)

		var foundCol Collection
		for _, col := range collections {
			if col.Name() == collectionName {
				foundCol = col
				break
			}
		}
		require.NotNil(t, foundCol, "collection should be found in list")

		err = foundCol.Add(ctx, WithIDs("doc1"), WithTexts("test document"))
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		count, err := foundCol.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("explicit EF overrides auto-wire", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_autowire_override-" + uuid.New().String()

		ef1 := embeddings.NewConsistentHashEmbeddingFunction()
		_, err := client.CreateCollection(ctx, collectionName, WithEmbeddingFunctionCreate(ef1))
		require.NoError(t, err)

		ef2 := embeddings.NewConsistentHashEmbeddingFunction()
		col, err := client.GetCollection(ctx, collectionName, WithEmbeddingFunctionGet(ef2))
		require.NoError(t, err)
		require.NotNil(t, col)

		err = col.Add(ctx, WithIDs("doc1"), WithTexts("test"))
		require.NoError(t, err)
	})

	t.Run("Collection fork", func(t *testing.T) {
		t.Skipf("Skipping fork")
		ctx := context.Background()
		collectionName := "test_collection-" + uuid.New().String()
		forkedCollectionName := "forked_collection-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name())

		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("this is document about cats", "dogs are man's best friends", "lions are big cats"))
		require.NoError(t, err)
		time.Sleep(5 * time.Second)
		forkedCollection, err := collection.Fork(ctx, forkedCollectionName)
		require.NoError(t, err)

		results, err := forkedCollection.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 3, results)
	})

	t.Run("auto-wire sparse embedding function from schema", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_sparse_ef_autowire-" + uuid.New().String()

		sparseEF, err := chromacloudsplade.NewEmbeddingFunction(chromacloudsplade.WithEnvAPIKey())
		require.NoError(t, err)

		schema, err := NewSchema(
			WithDefaultVectorIndex(NewVectorIndexConfig(WithSpace(SpaceL2))),
			WithSparseVectorIndex("sparse_embedding", NewSparseVectorIndexConfig(
				WithSparseEmbeddingFunction(sparseEF),
				WithSparseSourceKey("#document"),
			)),
		)
		require.NoError(t, err)

		createdCol, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
		require.NoError(t, err)
		require.NotNil(t, createdCol)

		retrievedCol, err := client.GetCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, retrievedCol)

		retrievedSchema := retrievedCol.Schema()
		require.NotNil(t, retrievedSchema, "Schema should be present")

		allSparseEFs := retrievedSchema.GetAllSparseEmbeddingFunctions()
		require.Len(t, allSparseEFs, 1, "Should have exactly one sparse EF")
		require.NotNil(t, allSparseEFs["sparse_embedding"], "Sparse EF should be auto-wired from Cloud schema")
		require.Equal(t, "chroma-cloud-splade", allSparseEFs["sparse_embedding"].Name())

		sparseEFByKey := retrievedSchema.GetSparseEmbeddingFunction("sparse_embedding")
		require.NotNil(t, sparseEFByKey, "Should find sparse EF by key")
		require.Equal(t, "chroma-cloud-splade", sparseEFByKey.Name())
	})
}

// TestCloudClientSearch tests Search API functionality
func TestCloudClientSearch(t *testing.T) {
	client := setupCloudClient(t)

	t.Run("Search data in collection", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_collection-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name())

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts("this is document about cats", "dogs are man's best friends", "lions are big cats"),
			WithMetadatas(
				NewDocumentMetadata(NewStringAttribute("category", "pets")),
				NewDocumentMetadata(NewStringAttribute("category", "pets")),
				NewDocumentMetadata(NewStringAttribute("category", "wildlife")),
			),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("tell me about cats"), WithKnnLimit(10)),
				WithPage(PageLimit(2)),
				WithSelect(KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, results)

		searchResult, ok := results.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, searchResult.IDs)
		require.NotEmpty(t, searchResult.Documents)
		require.NotEmpty(t, searchResult.Scores)
	})

	t.Run("Search with pagination", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_collection-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3", "4", "5"),
			WithTexts(
				"cats are fluffy pets",
				"dogs are loyal companions",
				"lions are wild cats",
				"tigers are striped cats",
				"birds can fly high",
			),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("cats"), WithKnnLimit(10)),
				WithPage(PageLimit(2)),
				WithSelect(KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, results)

		searchResult, ok := results.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, searchResult.IDs)
		require.LessOrEqual(t, len(searchResult.IDs[0]), 2)
	})

	t.Run("Search with IDIn filter", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_search_id_in-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts("cats are fluffy", "dogs are loyal", "lions are big cats"),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("cats"), WithKnnLimit(10)),
				WithFilter(IDIn("1", "3")),
				WithPage(PageLimit(5)),
				WithSelect(KID, KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, results)

		sr := results.(*SearchResultImpl)
		require.NotEmpty(t, sr.IDs)
		require.LessOrEqual(t, len(sr.IDs[0]), 2)

		for _, id := range sr.IDs[0] {
			require.True(t, id == "1" || id == "3", "Expected ID 1 or 3, got %s", id)
		}
	})

	t.Run("Search with IDNotIn filter", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_search_id_not_in-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts("cats are fluffy", "dogs are loyal", "lions are big cats"),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("cats"), WithKnnLimit(10)),
				WithFilter(IDNotIn("1")),
				WithPage(PageLimit(5)),
				WithSelect(KID, KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, results)

		sr := results.(*SearchResultImpl)
		require.NotEmpty(t, sr.IDs)

		for _, id := range sr.IDs[0] {
			require.NotEqual(t, DocumentID("1"), id, "ID 1 should be excluded")
		}
	})

	t.Run("Search with IDNotIn combined with metadata filter", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_search_id_not_in_combo-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3", "4"),
			WithTexts("cats are fluffy", "dogs are loyal", "lions are big cats", "tigers are striped"),
			WithMetadatas(
				NewDocumentMetadata(NewStringAttribute("category", "pets")),
				NewDocumentMetadata(NewStringAttribute("category", "pets")),
				NewDocumentMetadata(NewStringAttribute("category", "wildlife")),
				NewDocumentMetadata(NewStringAttribute("category", "wildlife")),
			),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("cats"), WithKnnLimit(10)),
				WithFilter(And(
					EqString(K("category"), "wildlife"),
					IDNotIn("3"),
				)),
				WithPage(PageLimit(5)),
				WithSelect(KID, KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, results)

		sr := results.(*SearchResultImpl)
		require.NotEmpty(t, sr.IDs)

		for _, id := range sr.IDs[0] {
			require.NotEqual(t, DocumentID("3"), id, "ID 3 should be excluded")
			require.True(t, id == "4", "Expected ID 4, got %s", id)
		}
	})

	t.Run("Search with DocumentContains filter", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_search_doc_contains-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts(
				"cats are fluffy pets that purr",
				"dogs are loyal companions that bark",
				"lions are big wild cats in Africa",
			),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("pets"), WithKnnLimit(10)),
				WithFilter(DocumentContains("fluffy")),
				WithPage(PageLimit(5)),
				WithSelect(KID, KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, results)

		sr := results.(*SearchResultImpl)
		require.NotEmpty(t, sr.IDs)
		require.Len(t, sr.IDs[0], 1, "Should only return 1 document containing 'fluffy'")
		require.Equal(t, DocumentID("1"), sr.IDs[0][0])
	})

	t.Run("Search with DocumentNotContains filter", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_search_doc_not_contains-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts(
				"cats are fluffy pets that purr",
				"dogs are loyal companions that bark",
				"lions are big wild cats in Africa",
			),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("animals"), WithKnnLimit(10)),
				WithFilter(DocumentNotContains("cats")),
				WithPage(PageLimit(5)),
				WithSelect(KID, KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, results)

		sr := results.(*SearchResultImpl)
		require.NotEmpty(t, sr.IDs)
		require.Len(t, sr.IDs[0], 1, "Should only return 1 document not containing 'cats'")
		require.Equal(t, DocumentID("2"), sr.IDs[0][0])
	})

	t.Run("Search with metadata projection and Rows iteration", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_search_metadata_rows-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts("cats are fluffy pets", "dogs are loyal companions", "lions are big cats"),
			WithMetadatas(
				NewDocumentMetadata(
					NewStringAttribute("category", "pets"),
					NewIntAttribute("year", 2020),
					NewFloatAttribute("rating", 4.5),
				),
				NewDocumentMetadata(
					NewStringAttribute("category", "pets"),
					NewIntAttribute("year", 2021),
					NewFloatAttribute("rating", 4.8),
				),
				NewDocumentMetadata(
					NewStringAttribute("category", "wildlife"),
					NewIntAttribute("year", 2019),
					NewFloatAttribute("rating", 4.2),
				),
			),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("cats"), WithKnnLimit(10)),
				WithPage(PageLimit(3)),
				WithSelect(KID, KDocument, KScore, KMetadata),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, results)

		sr, ok := results.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, sr.IDs)
		require.NotEmpty(t, sr.Metadatas)
		require.NotNil(t, sr.Metadatas[0], "First group of metadatas should not be nil")

		rows := sr.Rows()
		require.NotEmpty(t, rows, "Rows should not be empty")

		for _, row := range rows {
			require.NotEmpty(t, row.ID, "Row ID should not be empty")
			require.NotEmpty(t, row.Document, "Row Document should not be empty")
			require.NotNil(t, row.Metadata, "Row Metadata should not be nil")

			category, ok := row.Metadata.GetString("category")
			require.True(t, ok, "Should be able to get category")
			require.NotEmpty(t, category)

			year, ok := row.Metadata.GetInt("year")
			require.True(t, ok, "Should be able to get year")
			require.Greater(t, year, int64(2000))

			rating, ok := row.Metadata.GetFloat("rating")
			require.True(t, ok, "Should be able to get rating")
			require.Greater(t, rating, float64(0))

			require.NotZero(t, row.Score, "Score should not be zero")
		}

		row, ok := sr.At(0, 0)
		require.True(t, ok, "At(0, 0) should succeed")
		require.NotEmpty(t, row.ID)
		require.NotNil(t, row.Metadata)

		_, ok = sr.At(0, 100)
		require.False(t, ok, "At(0, 100) should return false")
		_, ok = sr.At(100, 0)
		require.False(t, ok, "At(100, 0) should return false")
	})

	t.Run("Search with ReadLevelIndexAndWAL", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_search_read_level_wal-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts("cats are fluffy pets", "dogs are loyal companions", "lions are big cats"),
		)
		require.NoError(t, err)

		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("animals"), WithKnnLimit(10)),
				WithPage(PageLimit(10)),
				WithSelect(KID, KDocument, KScore),
			),
			WithReadLevel(ReadLevelIndexAndWAL),
		)
		require.NoError(t, err)
		require.NotNil(t, results)

		sr, ok := results.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, sr.IDs)
		require.Len(t, sr.IDs[0], 3, "ReadLevelIndexAndWAL should return all 3 documents from WAL")
	})

	t.Run("Search with ReadLevelIndexOnly", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_search_read_level_index-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts("cats are fluffy pets", "dogs are loyal companions", "lions are big cats"),
		)
		require.NoError(t, err)

		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("animals"), WithKnnLimit(10)),
				WithPage(PageLimit(10)),
				WithSelect(KID, KDocument, KScore),
			),
			WithReadLevel(ReadLevelIndexOnly),
		)
		require.NoError(t, err)
		require.NotNil(t, results)

		sr, ok := results.(*SearchResultImpl)
		require.True(t, ok)
		require.Len(t, sr.IDs, 1)
		require.LessOrEqual(t, len(sr.IDs[0]), 3, "ReadLevelIndexOnly should return 0-3 documents if index may not yet compacted")
	})
}

// TestCloudClientArrayMetadata tests array metadata with $contains/$not_contains queries
func TestCloudClientArrayMetadata(t *testing.T) {
	client := setupCloudClient(t)

	t.Run("array metadata round-trip and contains queries", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_array_meta-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)

		meta1 := NewDocumentMetadata(
			NewStringArrayAttribute("tags", []string{"science", "physics"}),
			NewIntArrayAttribute("scores", []int64{100, 200}),
		)
		meta2 := NewDocumentMetadata(
			NewStringArrayAttribute("tags", []string{"math", "algebra"}),
			NewIntArrayAttribute("scores", []int64{300}),
		)
		meta3 := NewDocumentMetadata(
			NewStringArrayAttribute("tags", []string{"science", "biology"}),
			NewIntArrayAttribute("scores", []int64{400, 500}),
		)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts("doc about physics", "doc about algebra", "doc about biology"),
			WithMetadatas(meta1, meta2, meta3),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		// MetadataContainsString: "science" should match docs 1 and 3
		qr, err := collection.Query(ctx,
			WithQueryTexts("science"),
			WithNResults(10),
			WithWhere(MetadataContainsString(K("tags"), "science")),
		)
		require.NoError(t, err)
		idGroups := qr.GetIDGroups()
		require.Equal(t, 1, len(idGroups))
		require.Equal(t, 2, len(idGroups[0]))
		idSet := map[DocumentID]bool{}
		for _, id := range idGroups[0] {
			idSet[id] = true
		}
		require.True(t, idSet["1"])
		require.True(t, idSet["3"])

		// MetadataNotContainsString: exclude "science" should return doc 2
		qr, err = collection.Query(ctx,
			WithQueryTexts("math"),
			WithNResults(10),
			WithWhere(MetadataNotContainsString(K("tags"), "science")),
		)
		require.NoError(t, err)
		idGroups = qr.GetIDGroups()
		require.Equal(t, 1, len(idGroups))
		require.Equal(t, 1, len(idGroups[0]))
		require.Equal(t, DocumentID("2"), idGroups[0][0])
	})
}

// TestCloudClientSchema tests Schema configuration
func TestCloudClientSchema(t *testing.T) {
	client := setupCloudClient(t)

	t.Run("Create collection with default schema", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_schema_default-" + uuid.New().String()

		schema, err := NewSchemaWithDefaults()
		require.NoError(t, err)

		collection, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name())

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts("cats are fluffy pets", "dogs are loyal companions", "lions are big cats"),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Query(ctx, WithQueryTexts("tell me about cats"), WithNResults(2))
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())
		require.Contains(t, results.GetDocumentsGroups()[0][0].ContentString(), "cats")

		searchResults, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("cats"), WithKnnLimit(10)),
				WithPage(PageLimit(2)),
				WithSelect(KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, searchResults)
		sr, ok := searchResults.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, sr.IDs)
	})

	t.Run("Create collection with cosine space", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_schema_cosine-" + uuid.New().String()

		schema, err := NewSchema(
			WithDefaultVectorIndex(NewVectorIndexConfig(WithSpace(SpaceCosine))),
		)
		require.NoError(t, err)

		collection, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts("cats are fluffy pets", "dogs are loyal companions", "lions are big cats"),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Query(ctx, WithQueryTexts("fluffy pets"), WithNResults(2))
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())
	})

	t.Run("Create collection with inner product space", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_schema_ip-" + uuid.New().String()

		schema, err := NewSchema(
			WithDefaultVectorIndex(NewVectorIndexConfig(WithSpace(SpaceIP))),
		)
		require.NoError(t, err)

		collection, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts("cats are fluffy pets", "dogs are loyal companions", "lions are big cats"),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Query(ctx, WithQueryTexts("fluffy pets"), WithNResults(2))
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())
	})

	t.Run("Create collection with custom HNSW config", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_schema_hnsw-" + uuid.New().String()

		schema, err := NewSchema(
			WithDefaultVectorIndex(NewVectorIndexConfig(
				WithSpace(SpaceL2),
				WithHnsw(NewHnswConfig(
					WithEfConstruction(200),
					WithMaxNeighbors(32),
					WithEfSearch(50),
				)),
			)),
		)
		require.NoError(t, err)

		collection, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts("cats are fluffy pets", "dogs are loyal companions", "lions are big cats"),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Query(ctx, WithQueryTexts("cats"), WithNResults(2))
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())

		searchResults, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("cats"), WithKnnLimit(10)),
				WithPage(PageLimit(2)),
				WithSelect(KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, searchResults)
	})

	t.Run("Create collection with SPANN quantize in user schema should fail", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_schema_spann_quantize-" + uuid.New().String()

		schema, err := NewSchema(
			WithDefaultVectorIndex(NewVectorIndexConfig(
				WithSpace(SpaceL2),
				WithSpann(NewSpannConfig(
					WithSpannQuantize(SpannQuantizationFourBitRabitQWithUSearch),
				)),
			)),
		)
		require.NoError(t, err)

		_, err = client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
		require.Error(t, err)
		errMsg := strings.ToLower(err.Error())
		require.True(t, strings.Contains(errMsg, "quantize"),
			"expected quantize-related error, got: %v", err)
	})

	t.Run("Create collection with WithVectorIndexCreate", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_schema_convenience-" + uuid.New().String()

		collection, err := client.CreateCollection(ctx, collectionName,
			WithVectorIndexCreate(NewVectorIndexConfig(WithSpace(SpaceCosine))),
		)
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts("cats are fluffy pets", "dogs are loyal companions", "lions are big cats"),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Query(ctx, WithQueryTexts("fluffy"), WithNResults(2))
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())
	})

	t.Run("Create collection with FTS index", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_schema_fts-" + uuid.New().String()

		schema, err := NewSchema(
			WithDefaultVectorIndex(NewVectorIndexConfig(WithSpace(SpaceL2))),
			WithDefaultFtsIndex(&FtsIndexConfig{}),
		)
		require.NoError(t, err)

		collection, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts(
				"The quick brown fox jumps over the lazy dog",
				"A journey of a thousand miles begins with a single step",
				"To be or not to be that is the question",
			),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		searchResults, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("quick fox"), WithKnnLimit(10)),
				WithPage(PageLimit(2)),
				WithSelect(KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, searchResults)
		sr, ok := searchResults.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, sr.IDs)
	})

	t.Run("Create collection with metadata indexes", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_schema_metadata-" + uuid.New().String()

		schema, err := NewSchema(
			WithDefaultVectorIndex(NewVectorIndexConfig(WithSpace(SpaceL2))),
			WithStringIndex("category"),
			WithIntIndex("year"),
			WithFloatIndex("rating"),
			WithBoolIndex("available"),
		)
		require.NoError(t, err)

		collection, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3", "4"),
			WithTexts("cats are fluffy pets", "dogs are loyal companions", "lions are big cats", "birds can fly"),
			WithMetadatas(
				NewDocumentMetadata(
					NewStringAttribute("category", "pets"),
					NewIntAttribute("year", 2020),
					NewFloatAttribute("rating", 4.5),
					NewBoolAttribute("available", true),
				),
				NewDocumentMetadata(
					NewStringAttribute("category", "pets"),
					NewIntAttribute("year", 2021),
					NewFloatAttribute("rating", 4.8),
					NewBoolAttribute("available", true),
				),
				NewDocumentMetadata(
					NewStringAttribute("category", "wildlife"),
					NewIntAttribute("year", 2019),
					NewFloatAttribute("rating", 4.2),
					NewBoolAttribute("available", false),
				),
				NewDocumentMetadata(
					NewStringAttribute("category", "wildlife"),
					NewIntAttribute("year", 2022),
					NewFloatAttribute("rating", 3.9),
					NewBoolAttribute("available", true),
				),
			),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Query(ctx,
			WithQueryTexts("animals"),
			WithNResults(10),
			WithWhere(EqString(K("category"), "pets")),
		)
		require.NoError(t, err)
		require.LessOrEqual(t, len(results.GetDocumentsGroups()[0]), 2)

		results, err = collection.Query(ctx,
			WithQueryTexts("animals"),
			WithNResults(10),
			WithWhere(GteInt("year", 2020)),
		)
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())

		results, err = collection.Query(ctx,
			WithQueryTexts("animals"),
			WithNResults(10),
			WithWhere(GtFloat("rating", 4.0)),
		)
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())

		results, err = collection.Query(ctx,
			WithQueryTexts("animals"),
			WithNResults(10),
			WithWhere(EqBool("available", true)),
		)
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())

		searchResults, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("animals"), WithKnnLimit(10)),
				WithPage(PageLimit(5)),
				WithSelect(KDocument, KScore, KMetadata),
			),
		)
		require.NoError(t, err)
		sr, ok := searchResults.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, sr.IDs)
	})

	t.Run("Create collection with disabled indexes", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_schema_disabled-" + uuid.New().String()

		schema, err := NewSchema(
			WithDefaultVectorIndex(NewVectorIndexConfig(WithSpace(SpaceL2))),
			DisableStringIndex("large_text"),
		)
		require.NoError(t, err)

		collection, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2"),
			WithTexts("cats are fluffy pets", "dogs are loyal companions"),
			WithMetadatas(
				NewDocumentMetadata(NewStringAttribute("large_text", "some long text that should not be indexed")),
				NewDocumentMetadata(NewStringAttribute("large_text", "another long text")),
			),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Query(ctx, WithQueryTexts("pets"), WithNResults(2))
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())
	})

	t.Run("Create collection with disabled document FTS", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_schema_disabled_fts-" + uuid.New().String()

		schema, err := NewSchema(
			WithDefaultVectorIndex(NewVectorIndexConfig(WithSpace(SpaceL2))),
			DisableFtsIndex(DocumentKey),
		)
		require.NoError(t, err)

		collection, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
		require.NoError(t, err)
		require.NotNil(t, collection)
		t.Cleanup(func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if deleteErr := client.DeleteCollection(cleanupCtx, collectionName); deleteErr != nil {
				t.Logf("Warning: failed to delete collection '%s': %v", collectionName, deleteErr)
			}
		})

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts(
				"cats are fluffy pets that purr",
				"dogs are loyal companions that bark",
				"lions are big wild cats in Africa",
			),
		)
		require.NoError(t, err)

		// Dense vector queries should still work with FTS disabled once indexing is complete.
		require.Eventually(t, func() bool {
			results, queryErr := collection.Query(ctx, WithQueryTexts("pets"), WithNResults(2))
			if queryErr != nil {
				return false
			}
			return len(results.GetDocumentsGroups()) > 0
		}, 20*time.Second, 500*time.Millisecond, "expected query results after indexing")

		// Document text filters rely on FTS and should fail when it is disabled.
		_, err = collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("pets"), WithKnnLimit(10)),
				WithFilter(DocumentContains("fluffy")),
				WithPage(PageLimit(5)),
				WithSelect(KID, KDocument, KScore),
			),
		)
		require.Error(t, err)
		errMsg := strings.ToLower(err.Error())
		require.True(t, strings.Contains(errMsg, "fts") || strings.Contains(errMsg, "full-text"),
			"expected FTS-related error, got: %v", err)
	})

	t.Run("Comprehensive schema test", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_schema_comprehensive-" + uuid.New().String()

		schema, err := NewSchema(
			WithDefaultVectorIndex(NewVectorIndexConfig(
				WithSpace(SpaceCosine),
				WithHnsw(NewHnswConfig(WithEfConstruction(150))),
			)),
			WithDefaultFtsIndex(&FtsIndexConfig{}),
			WithStringIndex("category"),
			WithIntIndex("year"),
		)
		require.NoError(t, err)

		collection, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts(
				"Machine learning is transforming industries",
				"Deep learning neural networks are powerful",
				"Natural language processing enables chatbots",
			),
			WithMetadatas(
				NewDocumentMetadata(NewStringAttribute("category", "AI"), NewIntAttribute("year", 2023)),
				NewDocumentMetadata(NewStringAttribute("category", "AI"), NewIntAttribute("year", 2022)),
				NewDocumentMetadata(NewStringAttribute("category", "NLP"), NewIntAttribute("year", 2023)),
			),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		searchResults, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("machine learning AI"), WithKnnLimit(10)),
				WithPage(PageLimit(3)),
				WithSelect(KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, searchResults)

		searchResults, err = collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("learning"), WithKnnLimit(10)),
				WithPage(PageLimit(3)),
				WithSelect(KDocument, KScore, KMetadata),
			),
		)
		require.NoError(t, err)
		sr, ok := searchResults.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, sr.IDs)

		results, err := collection.Query(ctx,
			WithQueryTexts("neural networks"),
			WithNResults(2),
		)
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())

		results, err = collection.Query(ctx,
			WithQueryTexts("learning"),
			WithNResults(10),
			WithWhere(EqInt("year", 2023)),
		)
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())
	})
}

// TestCloudClientConfig tests client configuration and validation
func TestCloudClientConfig(t *testing.T) {
	// Note: This test group doesn't use setupCloudClient because it tests client creation itself

	if os.Getenv("CHROMA_API_KEY") == "" && os.Getenv("CHROMA_DATABASE") == "" && os.Getenv("CHROMA_TENANT") == "" {
		err := godotenv.Load("../../../.env")
		require.NoError(t, err)
	}

	// First create a client for the indexing status test
	client, err := NewCloudClient(
		WithLogger(testLogger()),
		WithDatabaseAndTenant(os.Getenv("CHROMA_DATABASE"), os.Getenv("CHROMA_TENANT")),
		WithCloudAPIKey(os.Getenv("CHROMA_API_KEY")),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = client.Close()
	})

	t.Run("indexing status", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_indexing_status-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("doc1", "doc2", "doc3"))
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		status, err := collection.IndexingStatus(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, status.TotalOps, uint64(3))
		require.GreaterOrEqual(t, status.OpIndexingProgress, 0.0)
		require.LessOrEqual(t, status.OpIndexingProgress, 1.0)
	})

	t.Run("Without API Key", func(t *testing.T) {
		t.Setenv("CHROMA_API_KEY", "")
		testClient, err := NewCloudClient(
			WithLogger(testLogger()),
			WithDatabaseAndTenant("test_database", "test_tenant"),
		)
		require.Error(t, err)
		require.Nil(t, testClient)
		require.Contains(t, err.Error(), "api key")
	})

	t.Run("Without Tenant and DB", func(t *testing.T) {
		t.Setenv("CHROMA_TENANT", "")
		t.Setenv("CHROMA_DATABASE", "")
		testClient, err := NewCloudClient(
			WithLogger(testLogger()),
			WithCloudAPIKey("test"),
		)
		require.Error(t, err)
		require.Nil(t, testClient)
		require.Contains(t, err.Error(), "tenant and database must be set for cloud client")
	})

	t.Run("With env tenant and DB", func(t *testing.T) {
		t.Setenv("CHROMA_TENANT", "test_tenant")
		t.Setenv("CHROMA_DATABASE", "test_database")
		testClient, err := NewCloudClient(
			WithLogger(testLogger()),
			WithCloudAPIKey("test"),
		)
		require.NoError(t, err)
		require.NotNil(t, testClient)
		require.Equal(t, NewTenant("test_tenant"), testClient.Tenant())
		require.Equal(t, NewDatabase("test_database", NewTenant("test_tenant")), testClient.Database())
	})

	t.Run("With env API key, tenant and DB", func(t *testing.T) {
		t.Setenv("CHROMA_TENANT", "test_tenant")
		t.Setenv("CHROMA_DATABASE", "test_database")
		t.Setenv("CHROMA_API_KEY", "test")
		testClient, err := NewCloudClient(
			WithLogger(testLogger()),
		)
		require.NoError(t, err)
		require.NotNil(t, testClient)
		require.NotNil(t, testClient.authProvider)
		require.IsType(t, &TokenAuthCredentialsProvider{}, testClient.authProvider)
		p, ok := testClient.authProvider.(*TokenAuthCredentialsProvider)
		require.True(t, ok)
		require.Equal(t, "test", p.Token)
		require.Equal(t, NewTenant("test_tenant"), testClient.Tenant())
		require.Equal(t, NewDatabase("test_database", NewTenant("test_tenant")), testClient.Database())
	})

	t.Run("With options overrides (precedence)", func(t *testing.T) {
		t.Setenv("CHROMA_TENANT", "test_tenant")
		t.Setenv("CHROMA_DATABASE", "test_database")
		t.Setenv("CHROMA_API_KEY", "test")
		testClient, err := NewCloudClient(
			WithLogger(testLogger()),
			WithCloudAPIKey("different_test_key"),
			WithDatabaseAndTenant("other_db", "other_tenant"),
		)
		require.NoError(t, err)
		require.NotNil(t, testClient)
		require.NotNil(t, testClient.authProvider)
		require.IsType(t, &TokenAuthCredentialsProvider{}, testClient.authProvider)
		p, ok := testClient.authProvider.(*TokenAuthCredentialsProvider)
		require.True(t, ok)
		require.Equal(t, "different_test_key", p.Token)
		require.Equal(t, NewTenant("other_tenant"), testClient.Tenant())
		require.Equal(t, NewDatabase("other_db", NewTenant("other_tenant")), testClient.Database())
	})
}

// TestCloudModifyConfiguration tests ModifyConfiguration with HNSW and SPANN parameters
func TestCloudModifyConfiguration(t *testing.T) {
	client := setupCloudClient(t)

	// HNSW modify is tested in integration tests; cloud env defaults to SPANN
	// and does not support HNSW collections.

	t.Run("modify SPANN configuration", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_modify_spann_cfg-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)

		cfg := NewUpdateCollectionConfiguration(
			WithSpannSearchNprobeModify(32),
			WithSpannEfSearchModify(64),
		)
		err = collection.ModifyConfiguration(ctx, cfg)
		require.NoError(t, err)

		updated, err := client.GetCollection(ctx, collectionName)
		require.NoError(t, err)
		spannCfg, ok := updated.Configuration().GetRaw("spann")
		require.True(t, ok)
		spannMap, ok := spannCfg.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(32), spannMap["search_nprobe"])
		assert.Equal(t, float64(64), spannMap["ef_search"])
	})
}

func TestCloudClientSearchRRF(t *testing.T) {
	client := setupCloudClient(t)

	t.Run("RRF smoke with dense and sparse KNN ranks", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_rrf_smoke-" + uuid.New().String()

		sparseEF, err := chromacloudsplade.NewEmbeddingFunction(chromacloudsplade.WithEnvAPIKey())
		require.NoError(t, err)

		schema, err := NewSchema(
			WithDefaultVectorIndex(NewVectorIndexConfig(WithSpace(SpaceL2))),
			WithSparseVectorIndex("sparse_embedding", NewSparseVectorIndexConfig(
				WithSparseEmbeddingFunction(sparseEF),
				WithSparseSourceKey("#document"),
			)),
		)
		require.NoError(t, err)

		collection, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3", "4", "5"),
			WithTexts(
				"quantum computing advances in 2024",
				"classical music theory and harmony",
				"quantum mechanics and particle physics",
				"cooking recipes for beginners",
				"quantum entanglement research papers",
			),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		denseKnn, err := NewKnnRank(KnnQueryText("quantum physics"), WithKnnReturnRank(), WithKnnLimit(10))
		require.NoError(t, err)
		sparseKnn, err := NewKnnRank(KnnQueryText("quantum physics"), WithKnnKey(K("sparse_embedding")), WithKnnReturnRank(), WithKnnLimit(10))
		require.NoError(t, err)

		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithRrfRank(WithRrfRanks(denseKnn.WithWeight(1.0), sparseKnn.WithWeight(1.0))),
				NewPage(Limit(5)),
				WithSelect(KID, KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, results)

		sr, ok := results.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, sr.IDs)
		require.NotEmpty(t, sr.Scores)
		require.Equal(t, len(sr.IDs[0]), len(sr.Scores[0]), "scores slice length should match IDs slice length")

		quantumIDs := map[DocumentID]bool{"1": true, "3": true, "5": true}
		require.True(t, quantumIDs[sr.IDs[0][0]], "first result should be a quantum doc, got %s", sr.IDs[0][0])
	})

	t.Run("RRF with custom k and different weights changes fusion scores", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_rrf_weights-" + uuid.New().String()

		sparseEF, err := chromacloudsplade.NewEmbeddingFunction(chromacloudsplade.WithEnvAPIKey())
		require.NoError(t, err)

		schema, err := NewSchema(
			WithDefaultVectorIndex(NewVectorIndexConfig(WithSpace(SpaceL2))),
			WithSparseVectorIndex("sparse_embedding", NewSparseVectorIndexConfig(
				WithSparseEmbeddingFunction(sparseEF),
				WithSparseSourceKey("#document"),
			)),
		)
		require.NoError(t, err)

		collection, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3", "4", "5"),
			WithTexts(
				"quantum computing advances in 2024",
				"classical music theory and harmony",
				"quantum mechanics and particle physics",
				"cooking recipes for beginners",
				"quantum entanglement research papers",
			),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		// Search A: equal weights, default k
		denseKnnA, err := NewKnnRank(KnnQueryText("quantum physics"), WithKnnReturnRank(), WithKnnLimit(10))
		require.NoError(t, err)
		sparseKnnA, err := NewKnnRank(KnnQueryText("quantum physics"), WithKnnKey(K("sparse_embedding")), WithKnnReturnRank(), WithKnnLimit(10))
		require.NoError(t, err)

		resultsA, err := collection.Search(ctx,
			NewSearchRequest(
				WithRrfRank(WithRrfRanks(denseKnnA.WithWeight(1.0), sparseKnnA.WithWeight(1.0)), WithRrfK(60)),
				NewPage(Limit(5)),
				WithSelect(KID, KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, resultsA)

		srA, ok := resultsA.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, srA.IDs)

		// Search B: heavy dense weight, low sparse weight, custom k
		denseKnnB, err := NewKnnRank(KnnQueryText("quantum physics"), WithKnnReturnRank(), WithKnnLimit(10))
		require.NoError(t, err)
		sparseKnnB, err := NewKnnRank(KnnQueryText("quantum physics"), WithKnnKey(K("sparse_embedding")), WithKnnReturnRank(), WithKnnLimit(10))
		require.NoError(t, err)

		resultsB, err := collection.Search(ctx,
			NewSearchRequest(
				WithRrfRank(WithRrfRanks(denseKnnB.WithWeight(5.0), sparseKnnB.WithWeight(0.1)), WithRrfK(10)),
				NewPage(Limit(5)),
				WithSelect(KID, KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, resultsB)

		srB, ok := resultsB.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, srB.IDs)
		require.NotEmpty(t, srB.Scores)

		t.Logf("Search A IDs: %v, Scores: %v", srA.IDs[0], srA.Scores[0])
		t.Logf("Search B IDs: %v, Scores: %v", srB.IDs[0], srB.Scores[0])
		require.NotEqual(t, srA.Scores, srB.Scores, "different RRF configurations should produce different fusion scores")
	})

	t.Run("RRF with single KNN rank succeeds", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_rrf_single_rank-" + uuid.New().String()

		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts("quantum physics", "classical music", "quantum entanglement"),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		denseKnn, err := NewKnnRank(KnnQueryText("quantum"), WithKnnReturnRank(), WithKnnLimit(10))
		require.NoError(t, err)

		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithRrfRank(WithRrfRanks(denseKnn.WithWeight(1.0))),
				NewPage(Limit(3)),
				WithSelect(KID, KScore),
			),
		)
		require.NoError(t, err)

		sr, ok := results.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, sr.IDs)
		require.NotEmpty(t, sr.Scores)
	})

	t.Run("RRF with zero weight uses default weight behavior", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_rrf_zero_weight-" + uuid.New().String()

		sparseEF, err := chromacloudsplade.NewEmbeddingFunction(chromacloudsplade.WithEnvAPIKey())
		require.NoError(t, err)

		schema, err := NewSchema(
			WithDefaultVectorIndex(NewVectorIndexConfig(WithSpace(SpaceL2))),
			WithSparseVectorIndex("sparse_embedding", NewSparseVectorIndexConfig(
				WithSparseEmbeddingFunction(sparseEF),
				WithSparseSourceKey("#document"),
			)),
		)
		require.NoError(t, err)

		collection, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
		require.NoError(t, err)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts("quantum physics", "classical music", "quantum entanglement"),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		denseKnn, err := NewKnnRank(KnnQueryText("quantum"), WithKnnReturnRank(), WithKnnLimit(10))
		require.NoError(t, err)
		sparseKnn, err := NewKnnRank(KnnQueryText("quantum"), WithKnnKey(K("sparse_embedding")), WithKnnReturnRank(), WithKnnLimit(10))
		require.NoError(t, err)

		// Zero weight currently falls back to the default weight (1.0) during RRF marshaling.
		resultsZero, err := collection.Search(ctx,
			NewSearchRequest(
				WithRrfRank(WithRrfRanks(denseKnn.WithWeight(1.0), sparseKnn.WithWeight(0.0))),
				NewPage(Limit(3)),
				WithSelect(KID, KScore),
			),
		)
		require.NoError(t, err)

		resultsDefault, err := collection.Search(ctx,
			NewSearchRequest(
				WithRrfRank(WithRrfRanks(denseKnn.WithWeight(1.0), sparseKnn.WithWeight(1.0))),
				NewPage(Limit(3)),
				WithSelect(KID, KScore),
			),
		)
		require.NoError(t, err)

		srZero, ok := resultsZero.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, srZero.IDs)
		require.NotEmpty(t, srZero.Scores)

		srDefault, ok := resultsDefault.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, srDefault.IDs)
		require.NotEmpty(t, srDefault.Scores)

		require.Equal(t, srDefault.IDs, srZero.IDs, "weight 0.0 should currently behave like the default weight")
		require.Equal(t, srDefault.Scores, srZero.Scores, "weight 0.0 should currently behave like the default weight")
	})

	t.Run("RRF rejects negative weight", func(t *testing.T) {
		denseKnn, err := NewKnnRank(KnnQueryText("test"), WithKnnReturnRank(), WithKnnLimit(10))
		require.NoError(t, err)

		_, err = NewRrfRank(WithRrfRanks(denseKnn.WithWeight(-1.0)))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "negative weight")
	})

	t.Run("RRF rejects k of zero", func(t *testing.T) {
		denseKnn, err := NewKnnRank(KnnQueryText("test"), WithKnnReturnRank(), WithKnnLimit(10))
		require.NoError(t, err)

		_, err = NewRrfRank(WithRrfRanks(denseKnn.WithWeight(1.0)), WithRrfK(0))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "k must be >= 1")
	})

	t.Run("RRF rejects NaN weight", func(t *testing.T) {
		denseKnn, err := NewKnnRank(KnnQueryText("test"), WithKnnReturnRank(), WithKnnLimit(10))
		require.NoError(t, err)

		_, err = NewRrfRank(WithRrfRanks(denseKnn.WithWeight(math.NaN())))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "NaN and Inf are not allowed")
	})

	t.Run("RRF rejects no ranks", func(t *testing.T) {
		_, err := NewRrfRank()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires at least one rank")
	})
}

func TestCloudClientSearchRRFArithmetic(t *testing.T) {
	client := setupCloudClient(t)

	ctx := context.Background()
	collectionName := "test_rrf_arithmetic-" + uuid.New().String()

	// M5: Register cleanup BEFORE CreateCollection so partial-create panics still fire.
	// M4: Use a fresh background context with timeout so delete survives parent-ctx cancellation.
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = client.DeleteCollection(cleanupCtx, collectionName)
	})

	sparseEF, err := chromacloudsplade.NewEmbeddingFunction(chromacloudsplade.WithEnvAPIKey())
	require.NoError(t, err)

	schema, err := NewSchema(
		WithDefaultVectorIndex(NewVectorIndexConfig(WithSpace(SpaceL2))),
		WithSparseVectorIndex("sparse_embedding", NewSparseVectorIndexConfig(
			WithSparseEmbeddingFunction(sparseEF),
			WithSparseSourceKey("#document"),
		)),
	)
	require.NoError(t, err)

	collection, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
	require.NoError(t, err)
	require.NotNil(t, collection)

	err = collection.Add(ctx,
		WithIDs("1", "2", "3", "4", "5"),
		WithTexts(
			"quantum computing advances in 2024",
			"classical music theory and harmony",
			"quantum mechanics and particle physics",
			"cooking recipes for beginners",
			"quantum entanglement research papers",
		),
	)
	require.NoError(t, err)

	// N1: Indexing readiness gate — bounded poll on Collection.Count (pkg/api/v2/collection.go:141)
	// instead of a fixed time.Sleep. Faster on healthy days, resilient on slow ones.
	require.Eventually(t, func() bool {
		n, cerr := collection.Count(ctx)
		return cerr == nil && n == 5
	}, 10*time.Second, 500*time.Millisecond, "collection indexing did not reach 5 docs within 10s")

	denseKnn, err := NewKnnRank(KnnQueryText("quantum physics"), WithKnnReturnRank(), WithKnnLimit(10))
	require.NoError(t, err)
	sparseKnn, err := NewKnnRank(KnnQueryText("quantum physics"), WithKnnKey(K("sparse_embedding")), WithKnnReturnRank(), WithKnnLimit(10))
	require.NoError(t, err)

	rrf, err := NewRrfRank(
		WithRrfRanks(denseKnn.WithWeight(1.0), sparseKnn.WithWeight(1.0)),
		WithRrfK(60),
	)
	require.NoError(t, err)

	// Baseline: plain RRF via the generic WithRank option.
	// NOTE: We deliberately use WithRank(rrf), NOT WithRrfRank(...). WithRrfRank is a
	// BUILDER that only accepts RrfOption — it cannot wrap pre-built ranks. Arithmetic
	// methods on *RrfRank return Rank (concrete *MulRank, *SubRank, ...), so WithRank is
	// the only option that accepts both the baseline *RrfRank and arithmetic results.
	baseline, err := collection.Search(ctx,
		NewSearchRequest(
			WithRank(rrf),
			NewPage(Limit(5)),
			WithSelect(KID, KDocument, KScore),
		),
	)
	require.NoError(t, err)
	require.NotNil(t, baseline)

	baselineSR, ok := baseline.(*SearchResultImpl)
	require.True(t, ok)
	require.NotEmpty(t, baselineSR.IDs, "baseline: outer IDs slice must not be empty")
	require.NotEmpty(t, baselineSR.Scores, "baseline: outer Scores slice must not be empty")
	require.NotEmpty(t, baselineSR.IDs[0], "baseline: inner IDs slice must not be empty")
	require.NotEmpty(t, baselineSR.Scores[0], "baseline: inner Scores slice must not be empty")
	t.Logf("baseline: IDs=%v Scores=%v", baselineSR.IDs, baselineSR.Scores)

	type bucket string
	const (
		bucketSafe       bucket = "safe"
		bucketSemflip    bucket = "semflip"
		bucketDegenerate bucket = "degenerate"
	)

	rows := []struct {
		name   string
		bucket bucket
		apply  func(r *RrfRank) Rank
	}{
		{name: "Add", bucket: bucketSafe, apply: func(r *RrfRank) Rank { return r.Add(FloatOperand(1.0)) }},
		{name: "Sub", bucket: bucketSafe, apply: func(r *RrfRank) Rank { return r.Sub(FloatOperand(1.0)) }},
		{name: "Multiply", bucket: bucketSafe, apply: func(r *RrfRank) Rank { return r.Multiply(FloatOperand(2.0)) }},
		{name: "Div", bucket: bucketSafe, apply: func(r *RrfRank) Rank { return r.Div(FloatOperand(2.0)) }},
		{name: "Negate", bucket: bucketSemflip, apply: func(r *RrfRank) Rank { return r.Negate() }},
		{name: "Abs", bucket: bucketSemflip, apply: func(r *RrfRank) Rank { return r.Abs() }},
		{name: "Exp", bucket: bucketSemflip, apply: func(r *RrfRank) Rank { return r.Exp() }},
		{name: "Log", bucket: bucketDegenerate, apply: func(r *RrfRank) Rank { return r.Log() }},
		{name: "Max_0", bucket: bucketDegenerate, apply: func(r *RrfRank) Rank { return r.Max(FloatOperand(0.0)) }},
		{name: "Min_0", bucket: bucketDegenerate, apply: func(r *RrfRank) Rank { return r.Min(FloatOperand(0.0)) }},
	}

	for _, tt := range rows {
		t.Run(tt.name, func(t *testing.T) {
			// H1 fix (Pattern A — defer): capture variables that will be populated
			// by the body after the Search call succeeds, and register a deferred
			// logger BEFORE the Search so the `pass1 ...` observation line ALWAYS
			// reaches the user regardless of err/nil/type-assertion-failure state.
			// This is the exact observation Plan 02 Task 0 depends on for the
			// risky semflip/degenerate rows.
			var (
				srIDs     [][]DocumentID
				srScores  [][]float64
				searchErr error
			)
			if tt.bucket == bucketSemflip || tt.bucket == bucketDegenerate {
				defer func() {
					t.Logf("pass1 %s %s: err=%v IDs=%v Scores=%v",
						tt.bucket, tt.name, searchErr, srIDs, srScores)
				}()
			}

			arith := tt.apply(rrf)
			results, err := collection.Search(ctx,
				NewSearchRequest(
					WithRank(arith),
					NewPage(Limit(5)),
					WithSelect(KID, KDocument, KScore),
				),
			)
			searchErr = err

			// Capture slices for the defer closure BEFORE any require.* that might
			// halt the subtest. If results is nil or wrong type, leave the captured
			// slices as nil — the defer prints `IDs=[] Scores=[]` which is still
			// a useful observation (it tells the user the server returned nothing).
			if results != nil {
				if sr, srOk := results.(*SearchResultImpl); srOk {
					srIDs = sr.IDs
					srScores = sr.Scores
				}
			}

			switch tt.bucket {
			case bucketSafe:
				// Safe bucket: strict assertions. require.NoError and require.NotNil
				// are fine here because safe methods are expected to succeed — any
				// failure is a real regression.
				require.NoError(t, err, "method %s: search must not return an error", tt.name)
				require.NotNil(t, results, "method %s: results must not be nil", tt.name)

				sr, ok := results.(*SearchResultImpl)
				require.True(t, ok, "method %s: result must be *SearchResultImpl", tt.name)

				// M3: shape guardrails BEFORE the differential assertion.
				// These catch easy regressions (empty slices, NaN/Inf, cardinality mismatch)
				// that a raw NotEqual would not.
				require.NotEmpty(t, sr.IDs, "method %s: IDs must not be empty", tt.name)
				require.NotEmpty(t, sr.Scores, "method %s: Scores must not be empty", tt.name)
				require.Len(t, sr.IDs, len(baselineSR.IDs),
					"method %s: query count must match baseline", tt.name)
				require.NotEmpty(t, sr.IDs[0],
					"method %s: inner IDs slice must not be empty", tt.name)
				require.NotEmpty(t, sr.Scores[0],
					"method %s: inner Scores slice must not be empty", tt.name)
				require.Len(t, sr.Scores[0], len(baselineSR.Scores[0]),
					"method %s: result cardinality must match baseline", tt.name)
				for _, s := range sr.Scores[0] {
					require.False(t, math.IsNaN(s),
						"method %s: score contains NaN: %v", tt.name, sr.Scores[0])
					require.False(t, math.IsInf(s, 0),
						"method %s: score contains Inf: %v", tt.name, sr.Scores[0])
				}
				require.NotEqual(t, baselineSR.Scores, sr.Scores,
					"method %s: arithmetic wrapping must produce a measurable score change from baseline", tt.name)
				t.Logf("pass1 safe %s: IDs=%v Scores=%v", tt.name, sr.IDs, sr.Scores)

			case bucketSemflip, bucketDegenerate:
				// H1 fix: observe-only, NO require.NoError / require.NotNil calls
				// for these rows. The deferred logger above emits the pass1 line
				// unconditionally. Pass 2 (Plan 02) will tighten these assertions
				// after the user reports empirical behavior per D-13.
				//
				// Intentionally empty: the defer does the observation.
			}
		})
	}

	t.Log("pass 1 scaffolding complete — run `go test -tags=\"basicv2 cloud\" -v -run TestCloudClientSearchRRFArithmetic ./pkg/api/v2/...` and report per-row observations so Plan 02 (Pass 2) can tighten semflip/degenerate assertions")
}

func TestCloudClientSearchGroupBy(t *testing.T) {
	client := setupCloudClient(t)

	t.Run("GroupBy with MinK caps results per group", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_groupby_mink-" + uuid.New().String()

		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3", "4", "5", "6", "7", "8", "9"),
			WithTexts(
				"machine learning basics",
				"deep learning tutorial",
				"neural network guide",
				"python web framework",
				"javascript frontend library",
				"react component design",
				"quantum computing intro",
				"quantum algorithms explained",
				"quantum error correction",
			),
			WithMetadatas(
				NewDocumentMetadata(NewStringAttribute("category", "AI")),
				NewDocumentMetadata(NewStringAttribute("category", "AI")),
				NewDocumentMetadata(NewStringAttribute("category", "AI")),
				NewDocumentMetadata(NewStringAttribute("category", "web")),
				NewDocumentMetadata(NewStringAttribute("category", "web")),
				NewDocumentMetadata(NewStringAttribute("category", "web")),
				NewDocumentMetadata(NewStringAttribute("category", "quantum")),
				NewDocumentMetadata(NewStringAttribute("category", "quantum")),
				NewDocumentMetadata(NewStringAttribute("category", "quantum")),
			),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("technology"), WithKnnLimit(50)),
				WithGroupBy(NewGroupBy(NewMinK(2, KScore), K("category"))),
				NewPage(Limit(20)),
				WithSelect(KID, KDocument, KScore, KMetadata),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, results)

		sr, ok := results.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, sr.IDs)

		categoryCounts := map[string]int{}
		for _, group := range sr.RowGroups() {
			for _, row := range group {
				cat, ok := row.Metadata.GetString("category")
				require.True(t, ok)
				categoryCounts[cat]++
			}
		}

		require.Equal(t, 3, len(categoryCounts), "GroupBy should return all three categories")
		for cat, count := range categoryCounts {
			assert.LessOrEqual(t, count, 2, "MinK(2) should cap category %q to at most 2 results, got %d", cat, count)
		}
	})

	t.Run("GroupBy with MaxK selects top k per group", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_groupby_maxk-" + uuid.New().String()

		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3", "4", "5", "6", "7", "8", "9"),
			WithTexts(
				"machine learning basics",
				"deep learning tutorial",
				"neural network guide",
				"python web framework",
				"javascript frontend library",
				"react component design",
				"quantum computing intro",
				"quantum algorithms explained",
				"quantum error correction",
			),
			WithMetadatas(
				NewDocumentMetadata(NewStringAttribute("category", "AI"), NewIntAttribute("priority", 10)),
				NewDocumentMetadata(NewStringAttribute("category", "AI"), NewIntAttribute("priority", 20)),
				NewDocumentMetadata(NewStringAttribute("category", "AI"), NewIntAttribute("priority", 30)),
				NewDocumentMetadata(NewStringAttribute("category", "web"), NewIntAttribute("priority", 15)),
				NewDocumentMetadata(NewStringAttribute("category", "web"), NewIntAttribute("priority", 25)),
				NewDocumentMetadata(NewStringAttribute("category", "web"), NewIntAttribute("priority", 35)),
				NewDocumentMetadata(NewStringAttribute("category", "quantum"), NewIntAttribute("priority", 5)),
				NewDocumentMetadata(NewStringAttribute("category", "quantum"), NewIntAttribute("priority", 50)),
				NewDocumentMetadata(NewStringAttribute("category", "quantum"), NewIntAttribute("priority", 100)),
			),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("technology"), WithKnnLimit(50)),
				WithGroupBy(NewGroupBy(NewMaxK(2, KScore), K("category"))),
				NewPage(Limit(20)),
				WithSelect(KID, KDocument, KScore, KMetadata),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, results)

		sr, ok := results.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, sr.IDs)

		categoryCounts := map[string]int{}
		for _, group := range sr.RowGroups() {
			for _, row := range group {
				cat, ok := row.Metadata.GetString("category")
				require.True(t, ok)
				categoryCounts[cat]++
			}
		}

		require.Equal(t, 3, len(categoryCounts), "GroupBy should return all three categories")
		for cat, count := range categoryCounts {
			assert.LessOrEqual(t, count, 2, "MaxK(2) should cap category %q to at most 2 results, got %d", cat, count)
		}
	})

	t.Run("GroupBy with k=1 returns at most 1 per group", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_groupby_k1-" + uuid.New().String()

		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)

		err = collection.Add(ctx,
			WithIDs("1", "2", "3", "4", "5", "6"),
			WithTexts(
				"machine learning basics",
				"deep learning tutorial",
				"python web framework",
				"javascript frontend library",
				"quantum computing intro",
				"quantum algorithms explained",
			),
			WithMetadatas(
				NewDocumentMetadata(NewStringAttribute("category", "AI")),
				NewDocumentMetadata(NewStringAttribute("category", "AI")),
				NewDocumentMetadata(NewStringAttribute("category", "web")),
				NewDocumentMetadata(NewStringAttribute("category", "web")),
				NewDocumentMetadata(NewStringAttribute("category", "quantum")),
				NewDocumentMetadata(NewStringAttribute("category", "quantum")),
			),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("technology"), WithKnnLimit(50)),
				WithGroupBy(NewGroupBy(NewMinK(1, KScore), K("category"))),
				NewPage(Limit(20)),
				WithSelect(KID, KDocument, KScore, KMetadata),
			),
		)
		require.NoError(t, err)

		sr, ok := results.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, sr.IDs)

		categoryCounts := map[string]int{}
		for _, group := range sr.RowGroups() {
			for _, row := range group {
				cat, ok := row.Metadata.GetString("category")
				require.True(t, ok)
				categoryCounts[cat]++
			}
		}

		require.Equal(t, 3, len(categoryCounts), "GroupBy should return all three categories")
		for cat, count := range categoryCounts {
			assert.LessOrEqual(t, count, 1, "MinK(1) should cap category %q to at most 1 result, got %d", cat, count)
		}
	})

	t.Run("GroupBy rejects k of zero", func(t *testing.T) {
		groupBy := NewGroupBy(NewMinK(0, KScore), K("category"))
		err := groupBy.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "k must be >= 1")
	})

	t.Run("GroupBy rejects nil aggregate", func(t *testing.T) {
		groupBy := NewGroupBy(nil, K("category"))
		err := groupBy.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "aggregate is required")
	})

	t.Run("GroupBy rejects no keys", func(t *testing.T) {
		groupBy := NewGroupBy(NewMinK(2, KScore))
		err := groupBy.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one key is required")
	})
}
