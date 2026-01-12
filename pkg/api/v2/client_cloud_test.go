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

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/amikos-tech/chroma-go/pkg/embeddings/chromacloud"
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

	t.Run("auto-wire chroma cloud embedding function on GetCollection", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_autowire_cloud_get-" + uuid.New().String()

		// Create collection WITH Chroma Cloud embedding function (the default for cloud)
		ef, err := chromacloud.NewEmbeddingFunction(chromacloud.WithEnvAPIKey())
		require.NoError(t, err)
		createdCol, err := client.CreateCollection(ctx, collectionName, WithEmbeddingFunctionCreate(ef))
		require.NoError(t, err)
		require.NotNil(t, createdCol)
		require.Equal(t, collectionName, createdCol.Name())

		// Get collection WITHOUT specifying embedding function - should auto-wire
		retrievedCol, err := client.GetCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, retrievedCol)

		// Verify the collection can be used for embedding operations
		err = retrievedCol.Add(ctx, WithIDs("doc1", "doc2"), WithTexts("hello world", "goodbye world"))
		require.NoError(t, err)

		time.Sleep(2 * time.Second) // Wait for indexing

		// Query using text (requires EF to be wired)
		results, err := retrievedCol.Query(ctx, WithQueryTexts("hello"), WithNResults(1))
		require.NoError(t, err)
		require.NotNil(t, results)
		require.NotEmpty(t, results.GetDocumentsGroups())
	})

	t.Run("auto-wire custom embedding function on GetCollection", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_autowire_custom_get-" + uuid.New().String()

		// Create collection WITH custom embedding function (consistent_hash)
		ef := embeddings.NewConsistentHashEmbeddingFunction()
		createdCol, err := client.CreateCollection(ctx, collectionName, WithEmbeddingFunctionCreate(ef))
		require.NoError(t, err)
		require.NotNil(t, createdCol)
		require.Equal(t, collectionName, createdCol.Name())

		// Get collection WITHOUT specifying embedding function - should auto-wire
		retrievedCol, err := client.GetCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, retrievedCol)

		// Verify the collection can be used for embedding operations
		err = retrievedCol.Add(ctx, WithIDs("doc1", "doc2"), WithTexts("hello world", "goodbye world"))
		require.NoError(t, err)

		time.Sleep(2 * time.Second) // Wait for indexing

		// Query using text (requires EF to be wired)
		results, err := retrievedCol.Query(ctx, WithQueryTexts("hello"), WithNResults(1))
		require.NoError(t, err)
		require.NotNil(t, results)
		require.NotEmpty(t, results.GetDocumentsGroups())
	})

	t.Run("auto-wire embedding function on ListCollections", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_autowire_list-" + uuid.New().String()

		// Create collection with EF
		ef := embeddings.NewConsistentHashEmbeddingFunction()
		_, err := client.CreateCollection(ctx, collectionName, WithEmbeddingFunctionCreate(ef))
		require.NoError(t, err)

		// List collections - should auto-wire EF
		collections, err := client.ListCollections(ctx)
		require.NoError(t, err)

		// Find our collection
		var foundCol Collection
		for _, col := range collections {
			if col.Name() == collectionName {
				foundCol = col
				break
			}
		}
		require.NotNil(t, foundCol, "collection should be found in list")

		// Verify the collection can be used for embedding operations
		err = foundCol.Add(ctx, WithIDs("doc1"), WithTexts("test document"))
		require.NoError(t, err)

		time.Sleep(2 * time.Second) // Wait for indexing

		count, err := foundCol.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("explicit EF overrides auto-wire", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_autowire_override-" + uuid.New().String()

		// Create collection with one EF
		ef1 := embeddings.NewConsistentHashEmbeddingFunction()
		_, err := client.CreateCollection(ctx, collectionName, WithEmbeddingFunctionCreate(ef1))
		require.NoError(t, err)

		// Get with explicit EF - should use the explicit one
		ef2 := embeddings.NewConsistentHashEmbeddingFunction()
		col, err := client.GetCollection(ctx, collectionName, WithEmbeddingFunctionGet(ef2))
		require.NoError(t, err)
		require.NotNil(t, col)

		// Verify it works
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

	// Schema Integration Tests

	t.Run("Schema: Create collection with default schema", func(t *testing.T) {
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

		// Verify Query API
		results, err := collection.Query(ctx, WithQueryTexts("tell me about cats"), WithNResults(2))
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())
		require.Contains(t, results.GetDocumentsGroups()[0][0].ContentString(), "cats")

		// Verify Search API
		searchResults, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("cats"), WithKnnLimit(10)),
				WithPage(WithLimit(2)),
				WithSelect(KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, searchResults)
		sr, ok := searchResults.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, sr.IDs)
	})

	t.Run("Schema: Create collection with cosine space", func(t *testing.T) {
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

	t.Run("Schema: Create collection with inner product space", func(t *testing.T) {
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

	t.Run("Schema: Create collection with custom HNSW config", func(t *testing.T) {
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

		// Verify Query
		results, err := collection.Query(ctx, WithQueryTexts("cats"), WithNResults(2))
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())

		// Verify Search
		searchResults, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("cats"), WithKnnLimit(10)),
				WithPage(WithLimit(2)),
				WithSelect(KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, searchResults)
	})

	t.Run("Schema: Create collection with WithVectorIndexCreate", func(t *testing.T) {
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

	t.Run("Schema: Create collection with FTS index", func(t *testing.T) {
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

		// Test Search with FTS
		searchResults, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("quick fox"), WithKnnLimit(10)),
				WithPage(WithLimit(2)),
				WithSelect(KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, searchResults)
		sr, ok := searchResults.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, sr.IDs)
	})

	t.Run("Schema: Create collection with metadata indexes", func(t *testing.T) {
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

		// Test string filter
		results, err := collection.Query(ctx,
			WithQueryTexts("animals"),
			WithNResults(10),
			WithWhereQuery(EqString("category", "pets")),
		)
		require.NoError(t, err)
		require.LessOrEqual(t, len(results.GetDocumentsGroups()[0]), 2)

		// Test int filter (year >= 2020)
		results, err = collection.Query(ctx,
			WithQueryTexts("animals"),
			WithNResults(10),
			WithWhereQuery(GteInt("year", 2020)),
		)
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())

		// Test float filter (rating > 4.0)
		results, err = collection.Query(ctx,
			WithQueryTexts("animals"),
			WithNResults(10),
			WithWhereQuery(GtFloat("rating", 4.0)),
		)
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())

		// Test bool filter
		results, err = collection.Query(ctx,
			WithQueryTexts("animals"),
			WithNResults(10),
			WithWhereQuery(EqBool("available", true)),
		)
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())

		// Test Search API with metadata selection
		searchResults, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("animals"), WithKnnLimit(10)),
				WithPage(WithLimit(5)),
				WithSelect(KDocument, KScore, KMetadata),
			),
		)
		require.NoError(t, err)
		sr, ok := searchResults.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, sr.IDs)
	})

	t.Run("Schema: Create collection with disabled indexes", func(t *testing.T) {
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

		// Collection should still work for vector search
		results, err := collection.Query(ctx, WithQueryTexts("pets"), WithNResults(2))
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())
	})

	t.Run("Schema: Comprehensive schema test", func(t *testing.T) {
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

		// Test KNN Search
		searchResults, err := collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("machine learning AI"), WithKnnLimit(10)),
				WithPage(WithLimit(3)),
				WithSelect(KDocument, KScore),
			),
		)
		require.NoError(t, err)
		require.NotNil(t, searchResults)

		// Test Search API with metadata selection
		searchResults, err = collection.Search(ctx,
			NewSearchRequest(
				WithKnnRank(KnnQueryText("learning"), WithKnnLimit(10)),
				WithPage(WithLimit(3)),
				WithSelect(KDocument, KScore, KMetadata),
			),
		)
		require.NoError(t, err)
		sr, ok := searchResults.(*SearchResultImpl)
		require.True(t, ok)
		require.NotEmpty(t, sr.IDs)

		// Test Query API
		results, err := collection.Query(ctx,
			WithQueryTexts("neural networks"),
			WithNResults(2),
		)
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())

		// Test Query with where clause
		results, err = collection.Query(ctx,
			WithQueryTexts("learning"),
			WithNResults(10),
			WithWhereQuery(EqInt("year", 2023)),
		)
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())
	})

	t.Run("Schema: Verify EF auto-wire from schema on GetCollection", func(t *testing.T) {
		ctx := context.Background()
		collectionName := "test_schema_ef_autowire-" + uuid.New().String()

		// Create schema - EF will be injected automatically
		schema, err := NewSchemaWithDefaults()
		require.NoError(t, err)

		// Create the collection with schema
		collection, err := client.CreateCollection(ctx, collectionName, WithSchemaCreate(schema))
		require.NoError(t, err)
		require.NotNil(t, collection)

		// Verify created collection has EF
		createdImpl, ok := collection.(*CollectionImpl)
		require.True(t, ok)
		require.NotNil(t, createdImpl.embeddingFunction, "Created collection should have EF")

		// Retrieve the collection WITHOUT providing an EF - should auto-wire from server config
		retrieved, err := client.GetCollection(ctx, collectionName)
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		// Verify EF was auto-wired
		retrievedImpl, ok := retrieved.(*CollectionImpl)
		require.True(t, ok)
		require.NotNil(t, retrievedImpl.embeddingFunction, "Retrieved collection should have auto-wired EF")
		require.Equal(t, createdImpl.embeddingFunction.Name(), retrievedImpl.embeddingFunction.Name())

		// Verify the collection is functional with the auto-wired EF
		err = retrieved.Add(ctx,
			WithIDs("1", "2"),
			WithTexts("test document one", "test document two"),
		)
		require.NoError(t, err)
		time.Sleep(2 * time.Second)

		results, err := retrieved.Query(ctx, WithQueryTexts("test"), WithNResults(2))
		require.NoError(t, err)
		require.NotEmpty(t, results.GetDocumentsGroups())
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
