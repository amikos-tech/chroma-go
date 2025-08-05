//go:build basicv2

package v2

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

func TestCloudClientHTTPIntegration(t *testing.T) {
	if os.Getenv("CHROMA_API_KEY") == "" && os.Getenv("CHROMA_DATABASE") == "" && os.Getenv("CHROMA_TENANT") == "" {
		err := godotenv.Load("../../../.env")
		require.NoError(t, err)
	}
	client, err := NewCloudAPIClient(
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

		t.Cleanup(func() {
			err := client.DeleteCollection(context.Background(), collectionName)
			require.NoError(t, err)
		})
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

		t.Cleanup(func() {
			err := client.DeleteCollection(context.Background(), collectionName)
			require.NoError(t, err)
		})
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

		t.Cleanup(func() {
			err := client.DeleteCollection(context.Background(), collectionName)
			require.NoError(t, err)
		})
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

		t.Cleanup(func() {
			err := client.DeleteCollection(context.Background(), collectionName)
			require.NoError(t, err)
		})
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

		t.Cleanup(func() {
			err := client.DeleteCollection(context.Background(), collectionName)
			require.NoError(t, err)
		})
	})

}
