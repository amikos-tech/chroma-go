//go:build basicv2 && cloud

package v2

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

func TestCloudClientHTTPIntegration(t *testing.T) {
	t.Cleanup(func() {
		t.Setenv("CHROMA_TENANT", "")
		t.Setenv("CHROMA_DATABASE", "")
		t.Setenv("CHROMA_API_KEY", "")
	})
	if os.Getenv("CHROMA_API_KEY") == "" && os.Getenv("CHROMA_DATABASE") == "" && os.Getenv("CHROMA_TENANT") == "" {
		err := godotenv.Load("../../../.env")
		require.NoError(t, err)
	}
	client, err := NewCloudClient(
		WithDebug(),
		WithDatabaseAndTenant(os.Getenv("CHROMA_DATABASE"), os.Getenv("CHROMA_TENANT")),
		WithCloudAPIKey(os.Getenv("CHROMA_API_KEY")),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		err := client.Close()
		require.NoError(t, err)
	})

	t.Run("Get Version", func(t *testing.T) {
		ctx := context.Background()
		v, err := client.GetVersion(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, v)
		require.Contains(t, v, "1.0")
	})

	t.Run("List collections", func(t *testing.T) {
		ctx := context.Background()
		collections, err := client.ListCollections(ctx)
		require.NoError(t, err)
		fmt.Println(collections)

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

		// Verify deletion
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

		// Add data to the collection
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

		// Add data to the collection
		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("this is document about cats", "123141231", "$@!123115"))
		require.NoError(t, err)

		err = collection.Delete(ctx, WithIDsDelete("1", "2"))
		require.NoError(t, err)

		// Verify deletion
		count, err := collection.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, count) // Only one document should remain

	})

	t.Run("Update and get data in collection", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_collection-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name())

		// Add data to the collection
		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("this is document about cats", "123141231", "$@!123115"))
		require.NoError(t, err)

		err = collection.Update(ctx, WithIDsUpdate("1", "2"), WithTextsUpdate("updated text for 1", "updated text for 2"))
		require.NoError(t, err)

		// Verify update

		results, err := collection.Get(ctx, WithIDsGet("1", "2"))
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

		// Add data to the collection
		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("this is document about cats", "dogs are man's best friends", "lions are big cats"))
		require.NoError(t, err)

		results, err := collection.Query(ctx, WithQueryTexts("tell me about cats"), WithNResults(2))
		require.NoError(t, err)
		require.Contains(t, results.GetDocumentsGroups()[0][0].ContentString(), "cats")
		require.Contains(t, results.GetDocumentsGroups()[0][1].ContentString(), "cats")

	})

	t.Run("Collection fork", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_collection-" + uuid.New().String()
		forkedCollectionName := "forked_collection-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name())

		// Add data to the collection
		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("this is document about cats", "dogs are man's best friends", "lions are big cats"))
		require.NoError(t, err)
		time.Sleep(5 * time.Second) // Wait for the data to be indexed
		forkedCollection, err := collection.Fork(ctx, forkedCollectionName)
		require.NoError(t, err)

		results, err := forkedCollection.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 3, results)

	})

	t.Run("Search data in collection", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_collection-" + uuid.New().String()
		collection, err := client.CreateCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name())

		// Add data to the collection
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
		time.Sleep(2 * time.Second) // Wait for indexing

		// Basic KNN search
		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("tell me about cats"), WithKnnLimit(10)),
				WithPage(WithLimit(2)),
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

		// Add data
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

		// Search with pagination
		results, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("cats"), WithKnnLimit(10)),
				WithPage(WithLimit(2)),
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

	t.Cleanup(func() {
		collections, err := client.ListCollections(context.Background())
		require.NoError(t, err)
		for _, collection := range collections {
			if collection.Name() != "chroma" && collection.Name() != "default" {
				err := client.DeleteCollection(context.Background(), collection.Name())
				require.NoError(t, err)
			}
		}
		fmt.Println("Cleanup completed")
		time.Sleep(1 * time.Second) // Wait for cleanup to complete
	})

	t.Run("Without API Key", func(t *testing.T) {
		t.Setenv("CHROMA_API_KEY", "")
		client, err := NewCloudClient(
			WithDebug(),
			WithDatabaseAndTenant("test_database", "test_tenant"),
		)
		require.Error(t, err)
		require.Nil(t, client)
		require.Contains(t, err.Error(), "api key")
	})

	t.Run("Without Tenant and DB", func(t *testing.T) {
		t.Setenv("CHROMA_TENANT", "")
		t.Setenv("CHROMA_DATABASE", "")
		client, err := NewCloudClient(
			WithDebug(),
			WithCloudAPIKey("test"),
		)
		require.Error(t, err)
		require.Nil(t, client)
		require.Contains(t, err.Error(), "tenant and database must be set for cloud client")
	})
	t.Run("With env tenant and DB", func(t *testing.T) {
		t.Setenv("CHROMA_TENANT", "test_tenant")
		t.Setenv("CHROMA_DATABASE", "test_database")
		client, err := NewCloudClient(
			WithDebug(),
			WithCloudAPIKey("test"),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
		require.Equal(t, NewTenant("test_tenant"), client.Tenant())
		require.Equal(t, NewDatabase("test_database", NewTenant("test_tenant")), client.Database())
	})

	t.Run("With env API key, tenant and DB", func(t *testing.T) {
		t.Setenv("CHROMA_TENANT", "test_tenant")
		t.Setenv("CHROMA_DATABASE", "test_database")
		t.Setenv("CHROMA_API_KEY", "test")
		client, err := NewCloudClient(
			WithDebug(),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
		require.NotNil(t, client.authProvider)
		require.IsType(t, &TokenAuthCredentialsProvider{}, client.authProvider)
		p, ok := client.authProvider.(*TokenAuthCredentialsProvider)
		require.True(t, ok)
		require.Equal(t, "test", p.Token)
		require.Equal(t, NewTenant("test_tenant"), client.Tenant())
		require.Equal(t, NewDatabase("test_database", NewTenant("test_tenant")), client.Database())
	})

	t.Run("With options overrides (precedence)", func(t *testing.T) {
		t.Setenv("CHROMA_TENANT", "test_tenant")
		t.Setenv("CHROMA_DATABASE", "test_database")
		t.Setenv("CHROMA_API_KEY", "test")
		client, err := NewCloudClient(
			WithDebug(),
			WithCloudAPIKey("different_test_key"),
			WithDatabaseAndTenant("other_db", "other_tenant"),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
		require.NotNil(t, client.authProvider)
		require.IsType(t, &TokenAuthCredentialsProvider{}, client.authProvider)
		p, ok := client.authProvider.(*TokenAuthCredentialsProvider)
		require.True(t, ok)
		require.Equal(t, "different_test_key", p.Token)
		require.Equal(t, NewTenant("other_tenant"), client.Tenant())
		require.Equal(t, NewDatabase("other_db", NewTenant("other_tenant")), client.Database())
	})

}
