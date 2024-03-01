/*
Testing Chroma Client
*/

package test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chroma "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/cohere"
	"github.com/amikos-tech/chroma-go/hf"
	"github.com/amikos-tech/chroma-go/openai"
	"github.com/amikos-tech/chroma-go/types"
)

func getAPIKey() string {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		err := godotenv.Load("../.env")
		if err != nil {
			fmt.Println("Error loading .env file")
		}
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	return apiKey
}

func Test_chroma_client(t *testing.T) {
	t.Parallel()
	chromaURL := os.Getenv("CHROMA_URL")
	if chromaURL == "" {
		chromaURL = "http://localhost:8000"
	}
	client := chroma.NewClient(chromaURL)

	t.Run("Test client with default tenant", func(t *testing.T) {
		tenant := types.DefaultTenant
		clientWithTenant := chroma.NewClient(chromaURL, chroma.WithTenant(tenant))
		require.NotNil(t, clientWithTenant)
		assert.Equal(t, tenant, clientWithTenant.Tenant)
		_, err := clientWithTenant.Reset(context.Background())
		require.NoError(t, err)
		_, err = clientWithTenant.ListCollections(context.Background())
		require.NoError(t, err)
	})

	t.Run("Test client with default tenant and db", func(t *testing.T) {
		tenant := types.DefaultTenant
		database := types.DefaultDatabase
		clientWithTenant := chroma.NewClient(chromaURL, chroma.WithTenant(tenant), chroma.WithDatabase(database))
		require.NotNil(t, clientWithTenant)
		assert.Equal(t, tenant, clientWithTenant.Tenant)
		_, err := clientWithTenant.Reset(context.Background())
		require.NoError(t, err)
		_, err = clientWithTenant.ListCollections(context.Background())
		require.NoError(t, err)
	})

	t.Run("Test Heartbeat", func(t *testing.T) { //nolint:paralleltest
		resp, err := client.Heartbeat(context.Background())
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp["nanosecond heartbeat"] > 0, "Heartbeat should be greater than 0")
	})

	t.Run("Test Create Tenant", func(t *testing.T) {
		_, err := client.Reset(context.Background())
		require.NoError(t, err)
		_, err = client.CreateTenant(context.Background(), "test-tenant")
		require.NoError(t, err)
	})

	t.Run("Test Get Tenant", func(t *testing.T) {
		_, err := client.Reset(context.Background())
		require.NoError(t, err)
		_, err = client.CreateTenant(context.Background(), "test-tenant")
		require.NoError(t, err)
		resp, err := client.GetTenant(context.Background(), "test-tenant")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "test-tenant", *resp.Name, "Tenant name should be test-tenant")
	})

	t.Run("Test create database", func(t *testing.T) {
		_, err := client.Reset(context.Background())
		require.NoError(t, err)
		_, err = client.CreateDatabase(context.Background(), "test db", nil)
		require.NoError(t, err)
	})

	t.Run("Test get database", func(t *testing.T) {
		_, err := client.Reset(context.Background())
		require.NoError(t, err)
		_, err = client.CreateDatabase(context.Background(), "test db", nil)
		require.NoError(t, err)
		resp, err := client.GetDatabase(context.Background(), "test db", nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "test db", *resp.Name, "Database name should be test db")
		require.NotNil(t, *resp.Id, "Database id should not be nil")
		require.Equal(t, "default_tenant", *resp.Tenant, "Database tenant should be default-tenant")
	})

	t.Run("Test create database with custom tenant", func(t *testing.T) {
		_, err := client.Reset(context.Background())
		require.NoError(t, err)
		var tenant = "test-tenant"
		_, err = client.CreateTenant(context.Background(), tenant)
		require.NoError(t, err)
		_, err = client.CreateDatabase(context.Background(), "test db", &tenant)
		require.NoError(t, err)
		resp, err := client.GetDatabase(context.Background(), "test db", &tenant)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "test db", *resp.Name, "Database name should be test db")
		require.NotNil(t, *resp.Id, "Database id should not be nil")
		require.Equal(t, tenant, *resp.Tenant, "Database tenant should be default-tenant")
	})

	t.Run("Test CreateCollection", func(t *testing.T) {
		collectionName := "test-collection"
		var metadata = map[string]interface{}{}
		apiKey := getAPIKey()
		embeddingFunction := openai.NewOpenAIEmbeddingFunction(apiKey)
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		resp, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, collectionName, resp.Name)
		require.Len(t, resp.Metadata, 2)
		// assert the metadata contains key embedding_function
		require.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), resp.Metadata["embedding_function"])
	})

	t.Run("Test Count Collections", func(t *testing.T) {
		collectionName := "test-collection"
		var metadata = map[string]interface{}{}
		apiKey := getAPIKey()
		embeddingFunction := openai.NewOpenAIEmbeddingFunction(apiKey)
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		resp, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, collectionName, resp.Name)
		require.Len(t, resp.Metadata, 2)
		colCount, err := client.CountCollections(context.Background())
		require.NoError(t, err)
		require.Equal(t, int32(1), colCount)
		// assert the metadata contains key embedding_function
		require.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), resp.Metadata["embedding_function"])
	})

	t.Run("Test Add Documents", func(t *testing.T) {
		collectionName := "test-collection"
		var metadata = map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		resp, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, collectionName, resp.Name)
		require.Len(t, resp.Metadata, 2)
		// assert the metadata contains key embedding_function
		require.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), resp.Metadata["embedding_function"])
		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
		}
		ids := []string{
			"ID1",
			"ID2",
		}

		metadatas := []map[string]interface{}{
			{"key1": "value1"},
			{"key2": "value2"},
		}
		// _, _ := embeddingFunction.CreateEmbedding(documents)
		_, addError := resp.Add(context.Background(), nil, metadatas, documents, ids)
		require.NoError(t, addError)
	})

	t.Run("Test Upsert Documents", func(t *testing.T) {
		collectionName := "test-collection"
		var metadata = map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		collection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name)
		require.Equal(t, 2, len(collection.Metadata))

		// assert the metadata contains key embedding_function
		require.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), collection.Metadata["embedding_function"])
		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
		}
		ids := []string{
			"ID1",
			"ID2",
		}

		metadatas := []map[string]interface{}{
			{"key1": "value1"},
			{"key2": "value2"},
		}
		_, addError := collection.Add(context.Background(), nil, metadatas, documents, ids)
		require.NoError(t, addError)

		documentsNew := []string{
			"Document 1 content here",
			"Document 2 content here",
		}
		idsNew := []string{
			"ID1",
			"ID5",
		}

		metadatasNew := []map[string]interface{}{
			{"key1": "value1"},
			{"key2": "value2"},
		}
		_, upError := collection.Upsert(context.Background(), nil, metadatasNew, documentsNew, idsNew)
		require.NoError(t, upError)
		getCollection, getError := collection.Get(context.Background(), nil, nil, nil, nil)
		require.NoError(t, getError)
		require.NotNil(t, getCollection)
		require.Equal(t, 3, len(getCollection.Documents))
		require.Equal(t, []string{"ID1", "ID2", "ID5"}, getCollection.Ids)
	})

	t.Run("Test Modify Collection Documents", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		collection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name)
		require.Equal(t, 2, len(collection.Metadata))

		// assert the metadata contains key embedding_function
		require.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), collection.Metadata["embedding_function"])
		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
		}
		ids := []string{
			"ID1",
			"ID2",
		}

		metadatas := []map[string]interface{}{
			{"key1": "value1"},
			{"key2": "value2"},
		}
		_, addError := collection.Add(context.Background(), nil, metadatas, documents, ids)
		require.Nil(t, addError)

		documentsNew := []string{
			"Document 1 updated content",
		}
		idsNew := []string{
			"ID1",
		}

		metadatasNew := []map[string]interface{}{
			{"key1": "updated1"},
		}
		_, upError := collection.Modify(context.Background(), nil, metadatasNew, documentsNew, idsNew)
		require.NoError(t, upError)
		getCollection, getError := collection.Get(context.Background(), nil, nil, nil, nil)
		require.NoError(t, getError)
		require.NotNil(t, getCollection)
		require.Equal(t, 2, len(getCollection.Documents))
		require.Equal(t, []string{"ID1", "ID2"}, getCollection.Ids)
		require.Equal(t, []string{"Document 1 updated content", "Document 2 content here"}, getCollection.Documents)
		if data, ok := getCollection.Metadatas[0]["key1"].([]uint8); ok {
			str := string(data)
			str = strings.ReplaceAll(str, `"`, "")
			assert.Equal(t, "updated1", str)
		} else {
			fmt.Println("Value is not a []uint8")
		}
	})

	t.Run("Test Get Collection Documents", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		resp, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, collectionName, resp.Name)
		require.Equal(t, 2, len(resp.Metadata))
		// assert the metadata contains key embedding_function
		require.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), resp.Metadata["embedding_function"])
		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
		}
		ids := []string{
			"ID1",
			"ID2",
		}

		metadatas := []map[string]interface{}{
			{"key1": "value1"},
			{"key2": "value2"},
		}
		col, addError := resp.Add(context.Background(), nil, metadatas, documents, ids)
		require.NoError(t, addError)

		res, geterr := col.Get(context.Background(), nil, nil, nil, nil)
		require.NoError(t, geterr)
		require.Equal(t, 2, len(res.Ids))
		require.Contains(t, res.Ids, "ID1")
		require.Contains(t, res.Ids, "ID2")
	})

	t.Run("Test Query Collection Documents", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		collection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name)
		require.Equal(t, 2, len(collection.Metadata))
		// assert the metadata contains key embedding_function
		require.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), collection.Metadata["embedding_function"])
		documents := []string{
			"This is a document about cats. Cats are great.",
			"this is a document about dogs. Dogs are great.",
		}
		ids := []string{
			"ID1",
			"ID2",
		}

		metadatas := []map[string]interface{}{
			{"key1": "value1"},
			{"key2": "value2"},
		}
		col, addError := collection.Add(context.Background(), nil, metadatas, documents, ids)
		require.NoError(t, addError)

		colGet, getErr := col.Count(context.Background())
		require.NoError(t, getErr)
		assert.Equal(t, int32(2), colGet)

		qr, qrerr := col.Query(context.Background(), []string{"Dogs are my favorite animals"}, 5, nil, nil, nil)
		require.NoError(t, qrerr)
		require.Equal(t, 2, len(qr.Documents[0]))
		require.Equal(t, documents[1], qr.Documents[0][0]) // ensure that the first document is the one about dogs
	})

	t.Run("Test Count Collection Documents", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		resp, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, collectionName, resp.Name)
		require.Equal(t, 2, len(resp.Metadata))
		// assert the metadata contains key embedding_function
		require.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), resp.Metadata["embedding_function"])
		documents := []string{
			"This is a document about cats. Cats are great.",
			"this is a document about dogs. Dogs are great.",
		}
		ids := []string{
			"ID1",
			"ID2",
		}

		metadatas := []map[string]interface{}{
			{"key1": "value1"},
			{"key2": "value2"},
		}
		col, addError := resp.Add(context.Background(), nil, metadatas, documents, ids)
		require.NoError(t, addError)

		countDocs, qrerr := col.Count(context.Background())
		require.NoError(t, qrerr)
		require.Equal(t, int32(2), countDocs)
	})

	t.Run("Test List Collections", func(t *testing.T) {
		collectionName1 := "test-collection1"
		collectionName2 := "test-collection2"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		_, _ = client.CreateCollection(context.Background(), collectionName1, metadata, true, embeddingFunction, types.L2)
		_, _ = client.CreateCollection(context.Background(), collectionName2, metadata, true, embeddingFunction, types.L2)
		collections, gcerr := client.ListCollections(context.Background())
		require.NoError(t, gcerr)
		require.Len(t, collections, 2)
		names := make([]string, len(collections))
		for i, person := range collections {
			names[i] = person.Name
		}
		require.Contains(t, names, collectionName1)
		require.Contains(t, names, collectionName2)
	})

	t.Run("Test Get Chroma Version", func(t *testing.T) {
		version, verr := client.Version(context.Background())
		require.NoError(t, verr)
		require.NotNil(t, version)
		// semver expression
		pattern := `^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
		match, err := regexp.MatchString(pattern, version)
		if err != nil {
			assert.Fail(t, fmt.Sprintf("Error matching version: %s", err))
			return
		}
		require.True(t, match, "Version does not match pattern")
	})

	t.Run("Test Delete Collection", func(t *testing.T) {
		collectionName1 := "test-collection1"
		collectionName2 := "test-collection2"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		_, _ = client.CreateCollection(context.Background(), collectionName1, metadata, true, embeddingFunction, types.L2)
		_, _ = client.CreateCollection(context.Background(), collectionName2, metadata, true, embeddingFunction, types.L2)
		collections, gcerr := client.ListCollections(context.Background())
		require.NoError(t, gcerr)
		require.Len(t, collections, 2)
		names := make([]string, len(collections))
		for i, person := range collections {
			names[i] = person.Name
		}
		require.Contains(t, names, collectionName1)
		require.Contains(t, names, collectionName2)

		// delete collection
		ocol, derr := client.DeleteCollection(context.Background(), collectionName1)
		require.NoError(t, derr)
		require.Equal(t, collectionName1, ocol.Name)

		// list collections
		collections, gcerr = client.ListCollections(context.Background())
		require.NoError(t, gcerr)
		require.Equal(t, 1, len(collections))
	})

	t.Run("Test Update Collection Name and Metadata", func(t *testing.T) {
		collectionName1 := "test-collection1"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		col, ccerr := client.CreateCollection(context.Background(), collectionName1, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, ccerr)
		// update collection
		newMetadata := map[string]interface{}{"new": "metadata"}

		updatedCol, uerr := col.Update(context.Background(), "new-name", newMetadata)
		require.NoError(t, uerr)
		require.Equal(t, "new-name", updatedCol.Name)
	})

	t.Run("Test Delete Embeddings by ID", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		collection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name)
		require.Equal(t, 2, len(collection.Metadata))
		docs, ids, docMetadata, embeds := GetTestDocumentTest()
		_, addError := collection.Add(context.Background(), embeds, docMetadata, docs, ids)
		require.NoError(t, addError)
		deletedIds, dellErr := collection.Delete(context.Background(), []string{"ID1"}, nil, nil)
		require.NoError(t, dellErr)
		require.Equal(t, 1, len(deletedIds))
		require.Equal(t, "ID1", deletedIds[0])
	})

	t.Run("Test Delete Embeddings by Where", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		collection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name)
		require.Equal(t, 2, len(collection.Metadata))
		docs, ids, docMetadata, embeds := GetTestDocumentTest()
		_, addError := collection.Add(context.Background(), embeds, docMetadata, docs, ids)
		require.NoError(t, addError)
		deletedIds, dellErr := collection.Delete(context.Background(), nil, map[string]interface{}{"key2": "value2"}, nil)
		require.NoError(t, dellErr)
		require.Equal(t, 1, len(deletedIds))
		require.Equal(t, "ID2", deletedIds[0])
	})

	t.Run("Test Delete Embeddings by Where Document Contains", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		collection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name)
		require.Equal(t, 2, len(collection.Metadata))
		docs, ids, docMetadata, embeds := GetTestDocumentTest()
		_, addError := collection.Add(context.Background(), embeds, docMetadata, docs, ids)
		require.NoError(t, addError)
		deletedIds, dellErr := collection.Delete(context.Background(), nil, nil, map[string]interface{}{"$contains": "Document 1"})
		require.NoError(t, dellErr)
		require.Equal(t, 1, len(deletedIds))
		require.Equal(t, "ID1", deletedIds[0])
	})

	t.Run("Test Add Documents with Cohere EF", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		apiKey := os.Getenv("COHERE_API_KEY")
		if apiKey == "" {
			err := godotenv.Load("../.env")
			if err != nil {
				assert.Failf(t, "Error loading .env file", "%s", err)
			}
			apiKey = os.Getenv("COHERE_API_KEY")
		}
		embeddingFunction := cohere.NewCohereEmbeddingFunction(apiKey)
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		collection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.COSINE)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name)
		require.Equal(t, 2, len(collection.Metadata))
		// assert the metadata contains key embedding_function
		assert.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), collection.Metadata["embedding_function"])
		docs, ids, docMetadata, embeds := GetTestDocumentTest()
		_, addError := collection.Add(context.Background(), embeds, docMetadata, docs, ids)
		require.Nil(t, addError)
	})

	t.Run("Test Add Documents with Hugging Face EF", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		apiKey := os.Getenv("HF_API_KEY")
		if apiKey == "" {
			err := godotenv.Load("../.env")
			if err != nil {
				assert.Failf(t, "Error loading .env file", "%s", err)
			}
			apiKey = os.Getenv("HF_API_KEY")
		}
		embeddingFunction := hf.NewHuggingFaceEmbeddingFunction(apiKey, "sentence-transformers/paraphrase-MiniLM-L6-v2")
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		collection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.IP)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name)
		require.Equal(t, 2, len(collection.Metadata))
		require.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), collection.Metadata["embedding_function"])
		docs, ids, docMetadata, embeds := GetTestDocumentTest()
		_, addError := collection.Add(context.Background(), embeds, docMetadata, docs, ids)
		require.Nil(t, addError)
	})

	t.Run("Test Collection Get Include Embeddings", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		collection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name)
		require.Equal(t, 2, len(collection.Metadata))
		docs, ids, docMetadata, embeds := GetTestDocumentTest()
		_, addError := collection.Add(context.Background(), embeds, docMetadata, docs, ids)
		require.NoError(t, addError)
		getEmbeddings, dellErr := collection.GetWithOptions(context.Background(), chroma.WithInclude(types.IEmbeddings))
		require.NoError(t, dellErr)
		require.Len(t, getEmbeddings.Ids, 2)
		require.Len(t, getEmbeddings.Embeddings, 2)
		require.Len(t, getEmbeddings.Documents, 0)
		require.Len(t, getEmbeddings.Metadatas, 0)
	})

	t.Run("Test Collection Get Include Documents", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		collection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name)
		require.Equal(t, 2, len(collection.Metadata))
		docs, ids, docMetadata, embeds := GetTestDocumentTest()
		_, addError := collection.Add(context.Background(), embeds, docMetadata, docs, ids)
		require.NoError(t, addError)
		getEmbeddings, dellErr := collection.GetWithOptions(context.Background(), chroma.WithInclude(types.IDocuments))
		require.NoError(t, dellErr)
		require.Len(t, getEmbeddings.Ids, 2)
		require.Len(t, getEmbeddings.Embeddings, 0)
		require.Len(t, getEmbeddings.Documents, 2)
		require.Len(t, getEmbeddings.Metadatas, 0)
	})

	t.Run("Test Collection Get Include Metadatas", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		collection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, collection)
		require.Equal(t, collectionName, collection.Name)
		require.Equal(t, 2, len(collection.Metadata))
		docs, ids, docMetadata, embeds := GetTestDocumentTest()
		_, addError := collection.Add(context.Background(), embeds, docMetadata, docs, ids)
		require.NoError(t, addError)
		getEmbeddings, dellErr := collection.GetWithOptions(context.Background(), chroma.WithInclude(types.IMetadatas))
		require.NoError(t, dellErr)
		require.Len(t, getEmbeddings.Ids, 2)
		require.Len(t, getEmbeddings.Embeddings, 0)
		require.Len(t, getEmbeddings.Documents, 0)
		require.Len(t, getEmbeddings.Metadatas, 2)
	})
}
