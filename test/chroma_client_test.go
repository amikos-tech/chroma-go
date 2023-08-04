/*
Testing Chroma Client
*/

package test

import (
	"fmt"
	chroma "github.com/amikos-tech/chroma-go"
	openai "github.com/amikos-tech/chroma-go/openai"
	godotenv "github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"regexp"
	"testing"
)

func Test_chroma_client(t *testing.T) {

	client := chroma.NewClient("http://localhost:8000")

	t.Run("Test Heartbeat", func(t *testing.T) {
		resp, err := client.Heartbeat()

		require.Nil(t, err)
		require.NotNil(t, resp)
		assert.Truef(t, resp["nanosecond heartbeat"] > 0, "Heartbeat should be greater than 0")
	})

	t.Run("Test CreateCollection", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]string{}
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			err := godotenv.Load("../.env")
			if err != nil {
				assert.Failf(t, "Error loading .env file", "%s", err)
			}
		}
		embeddingFunction := openai.NewOpenAIEmbeddingFunction(apiKey)
		distanceFunction := chroma.L2
		_, errRest := client.Reset()
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		resp, err := client.CreateCollection(collectionName, metadata, true, embeddingFunction, distanceFunction)
		require.Nil(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, collectionName, resp.Name)
		fmt.Printf("resp: %v\n", resp.Metadata)
		assert.Equal(t, 2, len(resp.Metadata))
		//assert the metadata contains key embedding_function
		assert.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), resp.Metadata["embedding_function"])
	})

	t.Run("Test Add Documents", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]string{}
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			err := godotenv.Load("../.env")
			if err != nil {
				assert.Failf(t, "Error loading .env file", "%s", err)
			}
		}
		embeddingFunction := openai.NewOpenAIEmbeddingFunction(apiKey)
		distanceFunction := chroma.L2
		_, errRest := client.Reset()
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		resp, err := client.CreateCollection(collectionName, metadata, true, embeddingFunction, distanceFunction)
		require.Nil(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, collectionName, resp.Name)
		fmt.Printf("resp: %v\n", resp.Metadata)
		assert.Equal(t, 2, len(resp.Metadata))
		//assert the metadata contains key embedding_function
		assert.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), resp.Metadata["embedding_function"])
		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
		}
		ids := []string{
			"ID1",
			"ID2",
		}

		metadatas := []map[string]string{
			{"key1": "value1"},
			{"key2": "value2"},
		}
		//_, _ := embeddingFunction.CreateEmbedding(documents)
		_, addError := resp.Add(nil, metadatas, documents, ids)
		require.Nil(t, addError)
	})

	t.Run("Test Get Collection Documents", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]string{}
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			err := godotenv.Load("../.env")
			if err != nil {
				assert.Failf(t, "Error loading .env file", "%s", err)
			}
		}
		embeddingFunction := openai.NewOpenAIEmbeddingFunction(apiKey)
		distanceFunction := chroma.L2
		_, errRest := client.Reset()
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		resp, err := client.CreateCollection(collectionName, metadata, true, embeddingFunction, distanceFunction)
		require.Nil(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, collectionName, resp.Name)
		assert.Equal(t, 2, len(resp.Metadata))
		//assert the metadata contains key embedding_function
		assert.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), resp.Metadata["embedding_function"])
		documents := []string{
			"Document 1 content here",
			"Document 2 content here",
		}
		ids := []string{
			"ID1",
			"ID2",
		}

		metadatas := []map[string]string{
			{"key1": "value1"},
			{"key2": "value2"},
		}
		col, addError := resp.Add(nil, metadatas, documents, ids)
		require.Nil(t, addError)

		col, geterr := col.Get(nil, nil, nil)
		require.Nil(t, geterr)
		assert.Equal(t, 2, len(col.CollectionData.Ids))
		assert.Contains(t, col.CollectionData.Ids, "ID1")
		assert.Contains(t, col.CollectionData.Ids, "ID2")
	})

	t.Run("Test Query Collection Documents", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]string{}
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			err := godotenv.Load("../.env")
			if err != nil {
				assert.Failf(t, "Error loading .env file", "%s", err)
			}
		}
		embeddingFunction := openai.NewOpenAIEmbeddingFunction(apiKey)
		distanceFunction := chroma.L2
		_, errRest := client.Reset()
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		resp, err := client.CreateCollection(collectionName, metadata, true, embeddingFunction, distanceFunction)
		require.Nil(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, collectionName, resp.Name)
		assert.Equal(t, 2, len(resp.Metadata))
		//assert the metadata contains key embedding_function
		assert.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), resp.Metadata["embedding_function"])
		documents := []string{
			"This is a document about cats. Cats are great.",
			"this is a document about dogs. Dogs are great.",
		}
		ids := []string{
			"ID1",
			"ID2",
		}

		metadatas := []map[string]string{
			{"key1": "value1"},
			{"key2": "value2"},
		}
		col, addError := resp.Add(nil, metadatas, documents, ids)
		require.Nil(t, addError)

		qr, qrerr := col.Query([]string{"I love dogs"}, 5, nil, nil, nil)
		require.Nil(t, qrerr)
		fmt.Printf("qr: %v\n", qr)
		assert.Equal(t, 2, len(qr.Documents[0]))
		assert.Equal(t, documents[1], qr.Documents[0][0]) //ensure that the first document is the one about dogs
	})

	t.Run("Test Count Collection Documents", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]string{}
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			err := godotenv.Load("../.env")
			if err != nil {
				assert.Failf(t, "Error loading .env file", "%s", err)
			}
		}
		embeddingFunction := openai.NewOpenAIEmbeddingFunction(apiKey)
		distanceFunction := chroma.L2
		_, errRest := client.Reset()
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		resp, err := client.CreateCollection(collectionName, metadata, true, embeddingFunction, distanceFunction)
		require.Nil(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, collectionName, resp.Name)
		assert.Equal(t, 2, len(resp.Metadata))
		//assert the metadata contains key embedding_function
		assert.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), resp.Metadata["embedding_function"])
		documents := []string{
			"This is a document about cats. Cats are great.",
			"this is a document about dogs. Dogs are great.",
		}
		ids := []string{
			"ID1",
			"ID2",
		}

		metadatas := []map[string]string{
			{"key1": "value1"},
			{"key2": "value2"},
		}
		col, addError := resp.Add(nil, metadatas, documents, ids)
		require.Nil(t, addError)

		countDocs, qrerr := col.Count()
		require.Nil(t, qrerr)
		fmt.Printf("qr: %v\n", countDocs)
		assert.Equal(t, int32(2), countDocs)
	})

	t.Run("Test List Collections", func(t *testing.T) {
		collectionName1 := "test-collection1"
		collectionName2 := "test-collection2"
		metadata := map[string]string{}
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			err := godotenv.Load("../.env")
			if err != nil {
				assert.Failf(t, "Error loading .env file", "%s", err)
			}
		}
		embeddingFunction := openai.NewOpenAIEmbeddingFunction(apiKey)
		distanceFunction := chroma.L2
		_, errRest := client.Reset()
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		_, _ = client.CreateCollection(collectionName1, metadata, true, embeddingFunction, distanceFunction)
		_, _ = client.CreateCollection(collectionName2, metadata, true, embeddingFunction, distanceFunction)
		collections, gcerr := client.ListCollections()
		require.Nil(t, gcerr)
		assert.Equal(t, 2, len(collections))
		names := make([]string, len(collections))
		for i, person := range collections {
			names[i] = person.Name
		}
		assert.Contains(t, names, collectionName1)
		assert.Contains(t, names, collectionName2)
	})

	t.Run("Test Get Chroma Version", func(t *testing.T) {
		version, verr := client.Version()
		require.Nil(t, verr)
		require.NotNil(t, version)
		//semver expression
		pattern := `^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
		match, err := regexp.MatchString(pattern, version)
		if err != nil {
			assert.Fail(t, fmt.Sprintf("Error matching version: %s", err))
			return
		}
		assert.True(t, match, "Version does not match pattern")
	})

	t.Run("Test Delete Collection", func(t *testing.T) {
		collectionName1 := "test-collection1"
		collectionName2 := "test-collection2"
		metadata := map[string]string{}
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			err := godotenv.Load("../.env")
			if err != nil {
				assert.Failf(t, "Error loading .env file", "%s", err)
			}
		}
		embeddingFunction := openai.NewOpenAIEmbeddingFunction(apiKey)
		distanceFunction := chroma.L2
		_, errRest := client.Reset()
		if errRest != nil {
			assert.Fail(t, fmt.Sprintf("Error resetting database: %s", errRest))
		}
		_, _ = client.CreateCollection(collectionName1, metadata, true, embeddingFunction, distanceFunction)
		_, _ = client.CreateCollection(collectionName2, metadata, true, embeddingFunction, distanceFunction)
		collections, gcerr := client.ListCollections()
		require.Nil(t, gcerr)
		assert.Equal(t, 2, len(collections))
		names := make([]string, len(collections))
		for i, person := range collections {
			names[i] = person.Name
		}
		assert.Contains(t, names, collectionName1)
		assert.Contains(t, names, collectionName2)

		//delete collection
		ocol, derr := client.DeleteCollection(collectionName1)
		require.Nil(t, derr)
		assert.Equal(t, collectionName1, ocol.Name)

		//list collections
		collections, gcerr = client.ListCollections()
		require.Nil(t, gcerr)
		assert.Equal(t, 1, len(collections))
	})

}
