//go:build basic

/*
Testing Chroma Client
*/

package test

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	defaultef "github.com/amikos-tech/chroma-go/pkg/embeddings/default_ef"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	testcontainers "github.com/testcontainers/testcontainers-go"
	tcchroma "github.com/testcontainers/testcontainers-go/modules/chroma"
	"github.com/testcontainers/testcontainers-go/wait"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/semver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chroma "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/collection"
	"github.com/amikos-tech/chroma-go/types"
	"github.com/amikos-tech/chroma-go/where"
	wheredoc "github.com/amikos-tech/chroma-go/where_document"
)

func TestChromaClient(t *testing.T) {
	ctx := context.Background()
	var chromaVersion = "latest"
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
	client, err := chroma.NewClient(chroma.WithBasePath(chromaURL), chroma.WithDebug(true))
	require.NoError(t, err)

	t.Run("Test client with default tenant", func(t *testing.T) {
		tenant := types.DefaultTenant
		clientWithTenant, err := chroma.NewClient(chroma.WithBasePath(chromaURL), chroma.WithTenant(tenant))
		require.NoError(t, err)
		require.NotNil(t, clientWithTenant)
		assert.Equal(t, tenant, clientWithTenant.Tenant)
		_, err = clientWithTenant.Reset(context.Background())
		require.NoError(t, err)
		_, err = clientWithTenant.ListCollections(context.Background())
		require.NoError(t, err)
	})

	t.Run("Test client with default tenant and db", func(t *testing.T) {
		tenant := types.DefaultTenant
		database := types.DefaultDatabase
		clientWithTenant, err := chroma.NewClient(chroma.WithBasePath(chromaURL), chroma.WithTenant(tenant), chroma.WithDatabase(database))
		require.NoError(t, err)
		require.NotNil(t, clientWithTenant)
		assert.Equal(t, tenant, clientWithTenant.Tenant)
		_, err = clientWithTenant.Reset(context.Background())
		require.NoError(t, err)
		_, err = clientWithTenant.ListCollections(context.Background())
		require.NoError(t, err)
	})

	t.Run("Test client with default headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			require.Equal(t, r.Header.Get("Authorization"), "Bearer my-custom-token")
		}))
		defer server.Close()
		var defaultHeaders = map[string]string{"Authorization": "Bearer my-custom-token"}
		clientWithTenant, err := chroma.NewClient(chroma.WithBasePath(server.URL), chroma.WithDefaultHeaders(defaultHeaders))
		require.NoError(t, err)
		require.NotNil(t, clientWithTenant)
		_, err = clientWithTenant.Heartbeat(context.Background())
		if err != nil {
			return
		}
	})

	t.Run("Test with basic auth", func(t *testing.T) {
		var user = "username"
		var password = "password"
		auth := user + ":" + password
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			require.Equal(t, r.Header.Get("Authorization"), "Basic "+encodedAuth)
		}))
		defer server.Close()
		clientWithAuth, err := chroma.NewClient(chroma.WithBasePath(server.URL), chroma.WithAuth(types.NewBasicAuthCredentialsProvider(user, password)))
		require.NoError(t, err)
		require.NotNil(t, clientWithAuth)
		_, err = clientWithAuth.Heartbeat(context.Background())
		if err != nil {
			return
		}
	})

	t.Run("Test with token auth - Authorization Header", func(t *testing.T) {
		var token = "mytoken"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			require.Equal(t, r.Header.Get(string(types.AuthorizationTokenHeader)), "Bearer "+token)
		}))
		defer server.Close()
		clientWithAuth, err := chroma.NewClient(chroma.WithBasePath(server.URL), chroma.WithAuth(types.NewTokenAuthCredentialsProvider(token, types.AuthorizationTokenHeader)))
		require.NoError(t, err)
		require.NotNil(t, clientWithAuth)
		_, err = clientWithAuth.Heartbeat(context.Background())
		if err != nil {
			return
		}
	})

	t.Run("Test with token auth - X-Chroma-Token Header", func(t *testing.T) {
		var token = "mytoken"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			require.Equal(t, r.Header.Get(string(types.XChromaTokenHeader)), token)
		}))
		defer server.Close()
		clientWithAuth, err := chroma.NewClient(chroma.WithBasePath(server.URL), chroma.WithAuth(types.NewTokenAuthCredentialsProvider(token, types.XChromaTokenHeader)))
		require.NoError(t, err)
		require.NotNil(t, clientWithAuth)
		_, err = clientWithAuth.Heartbeat(context.Background())
		if err != nil {
			return
		}
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
		multiTenantAPIVersion, err := semver.NewConstraint(">=0.4.15")
		require.NoError(t, err)
		_version, err := client.Version(context.Background())
		require.NoError(t, err)
		version, err := semver.NewVersion(strings.ReplaceAll(_version, `"`, ""))
		require.NoError(t, err)
		if !multiTenantAPIVersion.Check(version) {
			t.Skipf("Skipping test for version %s", version.String())
		}
		_, err = client.CreateTenant(context.Background(), "test-tenant")
		require.NoError(t, err)
	})

	t.Run("Test Get Tenant", func(t *testing.T) {
		_, err := client.Reset(context.Background())
		require.NoError(t, err)
		multiTenantAPIVersion, err := semver.NewConstraint(">=0.4.15")
		require.NoError(t, err)
		_version, err := client.Version(context.Background())
		require.NoError(t, err)
		version, err := semver.NewVersion(strings.ReplaceAll(_version, `"`, ""))
		require.NoError(t, err)
		if !multiTenantAPIVersion.Check(version) {
			t.Skipf("Skipping test for version %s", version.String())
		}
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
		multiTenantAPIVersion, err := semver.NewConstraint(">=0.4.15")
		require.NoError(t, err)
		_version, err := client.Version(context.Background())
		require.NoError(t, err)
		version, err := semver.NewVersion(strings.ReplaceAll(_version, `"`, ""))
		require.NoError(t, err)
		if !multiTenantAPIVersion.Check(version) {
			t.Skipf("Skipping test for version %s", version.String())
		}
		_, err = client.CreateDatabase(context.Background(), "test db", nil)
		require.NoError(t, err)
	})

	t.Run("Test get database", func(t *testing.T) {
		_, err := client.Reset(context.Background())
		require.NoError(t, err)
		multiTenantAPIVersion, err := semver.NewConstraint(">=0.4.15")
		require.NoError(t, err)
		_version, err := client.Version(context.Background())
		require.NoError(t, err)
		version, err := semver.NewVersion(strings.ReplaceAll(_version, `"`, ""))
		require.NoError(t, err)
		if !multiTenantAPIVersion.Check(version) {
			t.Skipf("Skipping test for version %s", version.String())
		}
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
		multiTenantAPIVersion, err := semver.NewConstraint(">=0.4.15")
		require.NoError(t, err)
		_version, err := client.Version(context.Background())
		require.NoError(t, err)
		version, err := semver.NewVersion(strings.ReplaceAll(_version, `"`, ""))
		require.NoError(t, err)
		if !multiTenantAPIVersion.Check(version) {
			t.Skipf("Skipping test for version %s", version.String())
		}
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
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
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
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		supportedVersion, err := semver.NewConstraint(">=0.4.20")
		require.NoError(t, err)
		_version, err := client.Version(context.Background())
		require.NoError(t, err)
		version, err := semver.NewVersion(strings.ReplaceAll(_version, `"`, ""))
		require.NoError(t, err)
		if !supportedVersion.Check(version) {
			t.Skipf("Skipping test for version %s", version.String())
		}
		require.NoError(t, errRest)
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
		require.NoError(t, errRest)
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
		require.NoError(t, errRest)
		newCollection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, newCollection)
		require.Equal(t, collectionName, newCollection.Name)
		require.Equal(t, 2, len(newCollection.Metadata))

		// assert the metadata contains key embedding_function
		require.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), newCollection.Metadata["embedding_function"])
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
		_, addError := newCollection.Add(context.Background(), nil, metadatas, documents, ids)
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
		_, upError := newCollection.Upsert(context.Background(), nil, metadatasNew, documentsNew, idsNew)
		require.NoError(t, upError)
		getCollection, getError := newCollection.Get(context.Background(), nil, nil, nil, nil)
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
		require.NoError(t, errRest)
		newCollection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, newCollection)
		require.Equal(t, collectionName, newCollection.Name)
		require.Equal(t, 2, len(newCollection.Metadata))

		// assert the metadata contains key embedding_function
		require.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), newCollection.Metadata["embedding_function"])
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
		_, addError := newCollection.Add(context.Background(), nil, metadatas, documents, ids)
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
		_, upError := newCollection.Modify(context.Background(), nil, metadatasNew, documentsNew, idsNew)
		require.NoError(t, upError)
		getCollection, getError := newCollection.Get(context.Background(), nil, nil, nil, nil)
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
		require.NoError(t, errRest)
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

	t.Run("Test Query Collection Documents - with Default includes", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		newCollection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, newCollection)
		require.Equal(t, collectionName, newCollection.Name)
		require.Equal(t, 2, len(newCollection.Metadata))
		// assert the metadata contains key embedding_function
		require.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), newCollection.Metadata["embedding_function"])
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
		col, addError := newCollection.Add(context.Background(), nil, metadatas, documents, ids)
		require.NoError(t, addError)

		colGet, getErr := col.Count(context.Background())
		require.NoError(t, getErr)
		assert.Equal(t, int32(2), colGet)

		qr, err := col.Query(context.Background(), []string{"Dogs are my favorite animals"}, 5, nil, nil, nil)
		require.NoError(t, err)
		require.Equal(t, 2, len(qr.Documents[0]))
		require.Equal(t, 2, len(qr.Metadatas[0]))
		require.Equal(t, 2, len(qr.Distances[0]))
		require.Equal(t, documents[1], qr.Documents[0][0]) // ensure that the first document is the one about dogs
	})

	t.Run("Test Query Collection Documents - With Options", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		newCollection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, newCollection)
		require.Equal(t, collectionName, newCollection.Name)
		require.Equal(t, 2, len(newCollection.Metadata))
		// assert the metadata contains key embedding_function
		require.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), newCollection.Metadata["embedding_function"])
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
		col, addError := newCollection.Add(context.Background(), nil, metadatas, documents, ids)
		require.NoError(t, addError)

		colGet, getErr := col.Count(context.Background())
		require.NoError(t, getErr)
		assert.Equal(t, int32(2), colGet)

		queryText, err := embeddingFunction.EmbedQuery(context.Background(), "Dogs are my favorite animals")
		require.NoError(t, err)
		qr, err := col.QueryWithOptions(context.Background(), types.WithQueryEmbedding(queryText), types.WithNResults(5))
		require.NoError(t, err)
		require.Equal(t, 2, len(qr.Documents[0]))
		require.Equal(t, 2, len(qr.Metadatas[0]))
		require.Equal(t, 2, len(qr.Distances[0]))
		require.Equal(t, documents[1], qr.Documents[0][0]) // ensure that the first document is the one about dogs
	})

	t.Run("Test Query Collection - No Embedding Function", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		newCollection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, newCollection)
		require.Equal(t, collectionName, newCollection.Name)
		require.Equal(t, 2, len(newCollection.Metadata))
		_, addError := newCollection.Add(context.Background(), nil, nil, []string{"test"}, []string{"ID1"})
		require.NoError(t, addError)

		// assert the metadata contains key embedding_function
		newCollection, err = client.GetCollection(context.Background(), collectionName, nil)
		require.NoError(t, err)
		_, err = newCollection.Query(context.Background(), []string{"Dogs are my favorite animals"}, 5, nil, nil, []types.QueryEnum{types.IDocuments})
		require.Error(t, err, "embedding function mismatch error expected")
	})

	t.Run("Test Query Collection Documents - with document only includes", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		newCollection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, newCollection)
		require.Equal(t, collectionName, newCollection.Name)
		require.Equal(t, 2, len(newCollection.Metadata))
		// assert the metadata contains key embedding_function
		require.Contains(t, chroma.GetStringTypeOfEmbeddingFunction(embeddingFunction), newCollection.Metadata["embedding_function"])
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
		col, addError := newCollection.Add(context.Background(), nil, metadatas, documents, ids)
		require.NoError(t, addError)

		colGet, getErr := col.Count(context.Background())
		require.NoError(t, getErr)
		assert.Equal(t, int32(2), colGet)

		qr, err := col.Query(context.Background(), []string{"Dogs are my favorite animals"}, 5, nil, nil, []types.QueryEnum{types.IDocuments})
		require.NoError(t, err)
		require.Equal(t, 2, len(qr.Documents[0]))
		require.Equal(t, 0, len(qr.Metadatas))
		require.Equal(t, 0, len(qr.Distances))
		require.Equal(t, documents[1], qr.Documents[0][0]) // ensure that the first document is the one about dogs
	})

	t.Run("Test Count Collection Documents", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
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
		require.NoError(t, errRest)
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

	t.Run("Test List Collections no meta", func(t *testing.T) {
		collectionName1 := "test-collection1"
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		_, _ = client.CreateCollection(context.Background(), collectionName1, nil, true, nil, types.L2)
		collections, gcerr := client.ListCollections(context.Background())
		require.NoError(t, gcerr)
		require.Len(t, collections, 1)
		names := make([]string, len(collections))
		for i, person := range collections {
			names[i] = person.Name
		}
		require.Contains(t, names, collectionName1)
		require.Contains(t, collections[0].Metadata, "hnsw:space")
	})

	t.Run("Test List Collections created by other clients", func(t *testing.T) {
		collectionName1 := "test-collection1"
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		data := []byte(`{"name":"` + collectionName1 + `"}`)
		resp, err := http.Post(client.ApiClient.GetConfig().Servers[0].URL+"/api/v1/collections", "application/json", bytes.NewBuffer(data))
		require.NoError(t, err)
		require.NotNil(t, resp)
		collections, gcerr := client.ListCollections(context.Background())
		require.NoError(t, gcerr)
		require.Len(t, collections, 1)
		names := make([]string, len(collections))
		for i, person := range collections {
			names[i] = person.Name
		}
		require.Contains(t, names, collectionName1)
		_respMetadata := collections[0].Metadata
		require.Len(t, _respMetadata, 0)
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
		require.NoError(t, errRest)
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
		require.NoError(t, errRest)
		col, ccerr := client.CreateCollection(context.Background(), collectionName1, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, ccerr)
		// update collection
		newMetadata := map[string]interface{}{"new": "metadata"}

		updatedCol, uerr := col.Update(context.Background(), "new-name", &newMetadata)
		require.NoError(t, uerr)
		require.Equal(t, "new-name", updatedCol.Name)
	})

	t.Run("Test Delete Embeddings by ID", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		newCollection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, newCollection)
		require.Equal(t, collectionName, newCollection.Name)
		require.Equal(t, 2, len(newCollection.Metadata))
		docs, ids, docMetadata, embeds := GetTestDocumentTest()
		_, addError := newCollection.Add(context.Background(), embeds, docMetadata, docs, ids)
		require.NoError(t, addError)
		_, err = newCollection.Delete(context.Background(), []string{"ID1"}, nil, nil)
		require.NoError(t, err)
	})

	t.Run("Test Delete Embeddings by Where", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		newCollection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, newCollection)
		require.Equal(t, collectionName, newCollection.Name)
		require.Equal(t, 2, len(newCollection.Metadata))
		docs, ids, docMetadata, embeds := GetTestDocumentTest()
		_, addError := newCollection.Add(context.Background(), embeds, docMetadata, docs, ids)
		require.NoError(t, addError)
		_, err = newCollection.Delete(context.Background(), nil, map[string]interface{}{"key2": "value2"}, nil)
		require.NoError(t, err)
	})

	t.Run("Test Delete Embeddings by Where Document Contains", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		newCollection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, newCollection)
		require.Equal(t, collectionName, newCollection.Name)
		require.Equal(t, 2, len(newCollection.Metadata))
		docs, ids, docMetadata, embeds := GetTestDocumentTest()
		_, addError := newCollection.Add(context.Background(), embeds, docMetadata, docs, ids)
		require.NoError(t, addError)
		_, err = newCollection.Delete(context.Background(), nil, nil, map[string]interface{}{"$contains": "Document 1"})
		require.NoError(t, err)
	})

	t.Run("Test Collection Get Include Embeddings", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		newCollection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, newCollection)
		require.Equal(t, collectionName, newCollection.Name)
		require.Equal(t, 2, len(newCollection.Metadata))
		docs, ids, docMetadata, embeds := GetTestDocumentTest()
		_, addError := newCollection.Add(context.Background(), embeds, docMetadata, docs, ids)
		require.NoError(t, addError)
		getEmbeddings, dellErr := newCollection.GetWithOptions(context.Background(), types.WithInclude(types.IEmbeddings))
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
		require.NoError(t, errRest)
		newCollection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, newCollection)
		require.Equal(t, collectionName, newCollection.Name)
		require.Equal(t, 2, len(newCollection.Metadata))
		docs, ids, docMetadata, embeds := GetTestDocumentTest()
		_, addError := newCollection.Add(context.Background(), embeds, docMetadata, docs, ids)
		require.NoError(t, addError)
		getEmbeddings, dellErr := newCollection.GetWithOptions(context.Background(), types.WithInclude(types.IDocuments))
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
		require.NoError(t, errRest)
		newCollection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, newCollection)
		require.Equal(t, collectionName, newCollection.Name)
		require.Equal(t, 2, len(newCollection.Metadata))
		docs, ids, docMetadata, embeds := GetTestDocumentTest()
		_, addError := newCollection.Add(context.Background(), embeds, docMetadata, docs, ids)
		require.NoError(t, addError)
		getEmbeddings, dellErr := newCollection.GetWithOptions(context.Background(), types.WithInclude(types.IMetadatas))
		require.NoError(t, dellErr)
		require.Len(t, getEmbeddings.Ids, 2)
		require.Len(t, getEmbeddings.Embeddings, 0)
		require.Len(t, getEmbeddings.Documents, 0)
		require.Len(t, getEmbeddings.Metadatas, 2)
	})

	t.Run("Test With Collection Builder", func(t *testing.T) {
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		newCollection, err := client.NewCollection(
			context.Background(),
			"test-collection",
			collection.WithMetadata("key1", "value1"),
			collection.WithEmbeddingFunction(types.NewConsistentHashEmbeddingFunction()),
			collection.WithHNSWDistanceFunction(types.L2),
		)
		require.NoError(t, err)
		// let's create a record set
		rs, rerr := types.NewRecordSet(
			types.WithEmbeddingFunction(types.NewConsistentHashEmbeddingFunction()),
			types.WithIDGenerator(types.NewULIDGenerator()),
		)
		require.NoError(t, rerr)
		// you can loop here to add multiple records
		rs.WithRecord(types.WithDocument("Document 1 content"), types.WithMetadata("key1", "value1"))
		rs.WithRecord(types.WithDocument("Document 2 content"), types.WithMetadata("key2", "value2"))
		_, err = rs.BuildAndValidate(context.Background())
		require.NoError(t, err)
		_, err = newCollection.AddRecords(context.Background(), rs)
		require.NoError(t, err)
		require.NotNil(t, newCollection)
		require.Equal(t, "test-collection", newCollection.Name)
		require.Equal(t, "value1", newCollection.Metadata["key1"])
		require.Equal(t, string(types.L2), newCollection.Metadata[types.HNSWSpace])
	})

	t.Run("Test With Collection Builder - no DF", func(t *testing.T) {
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		newCollection, err := client.NewCollection(
			context.Background(),
			"test-collection",
			collection.WithMetadata("key1", "value1"),
			collection.WithEmbeddingFunction(types.NewConsistentHashEmbeddingFunction()),
		)
		require.NoError(t, err)
		// let's create a record set
		rs, rerr := types.NewRecordSet(
			types.WithEmbeddingFunction(types.NewConsistentHashEmbeddingFunction()),
			types.WithIDGenerator(types.NewULIDGenerator()),
		)
		require.NoError(t, rerr)
		// you can loop here to add multiple records
		rs.WithRecord(types.WithDocument("Document 1 content"), types.WithMetadata("key1", "value1"))
		rs.WithRecord(types.WithDocument("Document 2 content"), types.WithMetadata("key2", "value2"))
		_, err = rs.BuildAndValidate(context.Background())
		require.NoError(t, err)
		_, err = newCollection.AddRecords(context.Background(), rs)
		require.NoError(t, err)
		require.NotNil(t, newCollection)
		require.Equal(t, "test-collection", newCollection.Name)
		require.Equal(t, "value1", newCollection.Metadata["key1"])
		require.Equal(t, string(types.L2), newCollection.Metadata[types.HNSWSpace])
	})

	t.Run("[Err] Test With Collection Builder - no embedding function", func(t *testing.T) {
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		_, err := client.NewCollection(
			context.Background(),
			"test-collection",
			collection.WithMetadata("key1", "value1"),
		)
		require.NoError(t, err)
		// let's create a record set
		rs, rerr := types.NewRecordSet(
			types.WithIDGenerator(types.NewULIDGenerator()),
		)
		require.NoError(t, rerr)
		// you can loop here to add multiple records
		rs.WithRecord(types.WithDocument("Document 1 content"), types.WithMetadata("key1", "value1"))
		rs.WithRecord(types.WithDocument("Document 2 content"), types.WithMetadata("key2", "value2"))
		_, err = rs.BuildAndValidate(context.Background())
		require.Error(t, err)
		require.Contains(t, err.Error(), "embedding function is required")
	})

	t.Run("[Err] Test With Collection Builder - empty collection name", func(t *testing.T) {
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		_, err := client.NewCollection(
			context.Background(),
			"",
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "collection name cannot be empty")
	})

	t.Run("Test Get With Where Option", func(t *testing.T) {
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		newCollection, err := client.NewCollection(
			context.Background(),
			"test-collection",
			collection.WithMetadata("key1", "value1"),
			collection.WithEmbeddingFunction(types.NewConsistentHashEmbeddingFunction()),
			collection.WithHNSWDistanceFunction(types.L2),
		)
		require.NoError(t, err)
		// let's create a record set
		rs, rerr := types.NewRecordSet(
			types.WithEmbeddingFunction(types.NewConsistentHashEmbeddingFunction()),
			types.WithIDGenerator(types.NewULIDGenerator()),
		)
		require.NoError(t, rerr)
		// you can loop here to add multiple records
		rs.WithRecord(types.WithDocument("Document 1 content"), types.WithMetadata("key1", "value1"))
		rs.WithRecord(types.WithDocument("Document 2 content"), types.WithMetadata("key2", "value2"))
		rs.WithRecord(types.WithDocument("Document 3 content"), types.WithMetadata("key3", "value3"))
		_, err = rs.BuildAndValidate(context.Background())
		require.NoError(t, err)
		newCollection, err = newCollection.AddRecords(context.Background(), rs)
		require.NoError(t, err)

		result, err := newCollection.GetWithOptions(context.Background(), types.WithWhere(where.Or(where.Eq("key1", "value1"), where.Eq("key2", "value2"))))
		require.NoError(t, err)
		require.Len(t, result.Ids, 2)
		require.Contains(t, result.Documents, "Document 1 content")
		require.Contains(t, result.Documents, "Document 2 content")
		require.NotContains(t, result.Documents, "Document 3 content")
	})

	t.Run("Test Get With WhereDocument Option", func(t *testing.T) {
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		newCollection, err := client.NewCollection(
			context.Background(),
			"test-collection",
			collection.WithMetadata("key1", "value1"),
			collection.WithEmbeddingFunction(types.NewConsistentHashEmbeddingFunction()),
			collection.WithHNSWDistanceFunction(types.L2),
		)
		require.NoError(t, err)
		// let's create a record set
		rs, rerr := types.NewRecordSet(
			types.WithEmbeddingFunction(types.NewConsistentHashEmbeddingFunction()),
			types.WithIDGenerator(types.NewULIDGenerator()),
		)
		require.NoError(t, rerr)
		// you can loop here to add multiple records
		rs.WithRecord(types.WithDocument("Document 1 content"), types.WithMetadata("key1", "value1"))
		rs.WithRecord(types.WithDocument("Document 2 content"), types.WithMetadata("key2", "value2"))
		rs.WithRecord(types.WithDocument("Document 3 content"), types.WithMetadata("key3", "value3"))
		_, err = rs.BuildAndValidate(context.Background())
		require.NoError(t, err)
		newCollection, err = newCollection.AddRecords(context.Background(), rs)
		require.NoError(t, err)

		result, err := newCollection.GetWithOptions(context.Background(), types.WithWhereDocument(wheredoc.Or(wheredoc.Contains("Document 1"), wheredoc.Contains("Document 2"))))
		require.NoError(t, err)
		require.Len(t, result.Ids, 2)
		require.Contains(t, result.Documents, "Document 1 content")
		require.Contains(t, result.Documents, "Document 2 content")
		require.NotContains(t, result.Documents, "Document 3 content")
	})

	t.Run("Test create collection with nil EF", func(t *testing.T) {
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		_, err := client.CreateCollection(context.Background(), "test-collection", nil, true, nil, types.L2)
		require.NoError(t, err)
	})

	t.Run("Test get collection with nil EF", func(t *testing.T) {
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		_, err := client.CreateCollection(context.Background(), "test-collection", nil, true, nil, types.L2)
		require.NoError(t, err)
		resp, err := client.GetCollection(context.Background(), "test-collection", nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "test-collection", resp.Name, "Collection name should be test-collection")
		require.NotNil(t, resp.ID, "Collection id should not be nil")
	})

	t.Run("Test query texts and generated query embeddings are in the result", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		newCollection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, newCollection)

		qr, err := newCollection.Query(context.Background(), []string{"Dogs are my favorite animals"}, 5, nil, nil, nil)
		require.NoError(t, err)
		require.Len(t, qr.QueryTexts, 1)
		require.Len(t, qr.QueryTextsGeneratedEmbeddings, 1)
		require.Len(t, qr.QueryEmbeddings, 0)
	})

	t.Run("Test query embeddings are in the result", func(t *testing.T) {
		collectionName := "test-collection"
		metadata := map[string]interface{}{}
		embeddingFunction := types.NewConsistentHashEmbeddingFunction()
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		newCollection, err := client.CreateCollection(context.Background(), collectionName, metadata, true, embeddingFunction, types.L2)
		require.NoError(t, err)
		require.NotNil(t, newCollection)
		queryText, err := embeddingFunction.EmbedQuery(context.Background(), "Dogs are my favorite animals")
		require.NoError(t, err, "failed to embed query")
		qr, err := newCollection.QueryWithOptions(context.Background(), types.WithQueryEmbedding(queryText), types.WithNResults(5))
		require.NoError(t, err)
		require.Len(t, qr.QueryEmbeddings, 1)
		require.Len(t, qr.QueryTextsGeneratedEmbeddings, 0)
		require.Len(t, qr.QueryTexts, 0)
	})

	t.Run("Test basePath empty error", func(t *testing.T) {
		_, err := chroma.NewClient(chroma.WithBasePath(""))
		require.Error(t, err)
		require.Contains(t, err.Error(), "basePath cannot be empty")
	})

	t.Run("Test basePath invalid URL error", func(t *testing.T) {
		_, err := chroma.NewClient(chroma.WithBasePath("this is not a valid URL"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid basePath URL")
	})

	t.Run("Test basePath valid URL", func(t *testing.T) {
		_, err := chroma.NewClient(chroma.WithBasePath("http://localhost:8000"))
		require.NoError(t, err)
	})

	t.Run("Test with user-provided HTTP client", func(t *testing.T) {
		httpClient := &http.Client{
			Timeout: 10 * time.Second,
		}
		client, err := chroma.NewClient(chroma.WithBasePath("http://localhost:8000"), chroma.WithHTTPClient(httpClient))
		require.NoError(t, err)
		require.Equal(t, httpClient, client.ApiClient.GetConfig().HTTPClient)
	})

	t.Run("Test Query With Where", func(t *testing.T) {
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		newCollection, err := client.NewCollection(
			context.Background(),
			"test-collection",
			collection.WithMetadata("key1", "value1"),
			collection.WithEmbeddingFunction(types.NewConsistentHashEmbeddingFunction()),
			collection.WithHNSWDistanceFunction(types.L2),
		)
		require.NoError(t, err)
		rs, rerr := types.NewRecordSet(
			types.WithEmbeddingFunction(types.NewConsistentHashEmbeddingFunction()),
			types.WithIDGenerator(types.NewULIDGenerator()),
		)
		require.NoError(t, rerr)
		rs.WithRecord(types.WithDocument("Document 1 content"), types.WithMetadata("key1", "value1"))
		rs.WithRecord(types.WithDocument("Document 2 content"), types.WithMetadata("key2", "value2"))
		rs.WithRecord(types.WithDocument("Document 3 content"), types.WithMetadata("key3", "value3"))
		_, err = rs.BuildAndValidate(context.Background())
		require.NoError(t, err)
		newCollection, err = newCollection.AddRecords(context.Background(), rs)
		require.NoError(t, err)

		whereClause := map[string]interface{}{
			"key1": map[string]interface{}{
				"$eq": "value1",
			},
		}
		result, err := newCollection.Query(context.Background(), []string{"Document 1"}, 10, whereClause, nil, nil)
		require.NoError(t, err)
		require.Len(t, result.Ids[0], 1)
		require.Contains(t, result.Documents[0], "Document 1 content")
		result, err = newCollection.QueryWithOptions(context.Background(), types.WithNResults(10), types.WithQueryText("Document 1"), types.WithWhere(where.Eq("key1", "value1")))
		require.NoError(t, err)
		require.Len(t, result.Ids[0], 1)
		require.Contains(t, result.Documents[0], "Document 1 content")
	})

	t.Run("Test Client close with no closable resources", func(t *testing.T) {
		client, err := chroma.NewClient(chroma.WithBasePath(chromaURL), chroma.WithDebug(true))
		require.NoError(t, err)
		require.NoError(t, client.Close())
	})

	t.Run("Test Client close with no closable resources", func(t *testing.T) {
		client, err := chroma.NewClient(chroma.WithBasePath(chromaURL), chroma.WithDebug(true))
		require.NoError(t, err)
		ef, _, err := defaultef.NewDefaultEmbeddingFunction()
		require.NoError(t, err)
		_, err = client.NewCollection(context.Background(), "test", collection.WithEmbeddingFunction(ef))
		require.NoError(t, err)
		require.NoError(t, client.Close())
	})

	t.Run("Test Create Collection with Default EF", func(t *testing.T) {
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		col, err := client.NewCollection(context.Background(), "test")
		require.NoError(t, err)
		require.NotNil(t, col.EmbeddingFunction)
		_, err = col.Add(context.Background(), nil, nil, []string{"test"}, []string{"test"})
		require.NoError(t, err, "failed to add document")
		results, err := col.GetWithOptions(context.Background(), types.WithInclude(types.IEmbeddings, types.IDocuments), types.WithIds([]string{"test"}))
		require.NoError(t, err, "failed to get document")
		require.Len(t, results.Embeddings, 1)
		require.Len(t, results.Documents, 1)
		require.Equal(t, "test", results.Ids[0])
		require.Equal(t, "test", results.Documents[0])
		require.Equal(t, 384, results.Embeddings[0].Len())
		require.NoError(t, client.Close())
	})

	t.Run("Test Get Collection with Default EF", func(t *testing.T) {
		_, errRest := client.Reset(context.Background())
		require.NoError(t, errRest)
		col, err := client.NewCollection(context.Background(), "test")
		require.NoError(t, err)
		require.NotNil(t, col.EmbeddingFunction)
		_, err = col.Add(context.Background(), nil, nil, []string{"test"}, []string{"test"})
		require.NoError(t, err, "failed to add document")

		getCollection, err := client.GetCollection(context.Background(), "test", nil)
		require.NoError(t, err, "failed to get collection")
		require.NotNil(t, getCollection.EmbeddingFunction)
		_, err = getCollection.Add(context.Background(), nil, nil, []string{"test"}, []string{"test"})
		require.NoError(t, err, "failed to add documents after get")
	})

	t.Run("Test with timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Second) // Simulate a delay
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Hello, client"))
		}))
		defer server.Close()

		c, err := chroma.NewClient(chroma.WithBasePath(server.URL), chroma.WithTimeout(5*time.Second))
		require.NoError(t, err)
		_, err = c.Version(context.Background())
		require.Error(t, err)
		require.Contains(t, err.Error(), "context deadline exceeded")
	})
}

func TestClientSecurity(t *testing.T) {
	ctx := context.Background()
	var chromaImage = "ghcr.io/chroma-core/chroma"
	var chromaVersion = "latest"
	if os.Getenv("CHROMA_VERSION") != "" {
		chromaVersion = os.Getenv("CHROMA_VERSION")
	}

	if os.Getenv("CHROMA_IMAGE") != "" {
		chromaImage = os.Getenv("CHROMA_IMAGE")
	}
	tempDir := t.TempDir()
	certPath := fmt.Sprintf("%s/server.crt", tempDir)
	keyPath := fmt.Sprintf("%s/server.key", tempDir)
	containerCertPath := "/chroma/server.crt"
	containerKeyPath := "/chroma/server.key"

	cmd := []string{"--workers", "1",
		"--host", "0.0.0.0",
		"--port", "8000",
		"--proxy-headers",
		"--log-config", "/chroma/chromadb/log_config.yml",
		"--timeout-keep-alive", "30",
		"--ssl-certfile", containerCertPath,
		"--ssl-keyfile", containerKeyPath,
	}
	entrypoint := []string{}
	if chromaVersion != "latest" {
		cv := semver.MustParse(chromaVersion)
		if cv.LessThan(semver.MustParse("0.4.11")) {
			entrypoint = append(entrypoint, "/bin/bash", "-c")
			cmd = append([]string{fmt.Sprintf("pip install --force-reinstall --no-cache-dir chroma-hnswlib numpy==1.26.4 && ln -s /chroma/log_config.yml /chroma/chromadb/log_config.yml && uvicorn chromadb.app:app %s", strings.Join(cmd, " "))})
		} else if cv.LessThan(semver.MustParse("0.4.23")) {
			cmd = append([]string{"uvicorn", "chromadb.app:app"}, cmd...)
		}
	}

	CreateSelfSignedCert(certPath, keyPath)
	chromaContainer, err := tcchroma.Run(ctx,
		fmt.Sprintf("%s:%s", chromaImage, chromaVersion),
		testcontainers.WithEnv(map[string]string{"ALLOW_RESET": "true"}),
		testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				WaitingFor: wait.ForAll(
					wait.ForListeningPort("8000/tcp"),
				),
				Entrypoint: entrypoint,
				HostConfigModifier: func(hostConfig *container.HostConfig) {
					hostConfig.Mounts = []mount.Mount{
						{
							Type:   mount.TypeBind,
							Source: certPath,
							Target: containerCertPath,
						},
						{
							Type:   mount.TypeBind,
							Source: keyPath,
							Target: containerKeyPath,
						},
					}
				},
				Cmd: cmd,
			},
		}),
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
	chromaURL = strings.ReplaceAll(endpoint, "http://", "https://")
	t.Run("Test with insecure client", func(t *testing.T) {
		client, err := chroma.NewClient(chroma.WithBasePath(chromaURL), chroma.WithInsecure())
		require.NoError(t, err)
		version, err := client.Version(ctx)
		require.NoError(t, err)
		require.NotNil(t, version)
	})

	t.Run("Test with self-signed failure", func(t *testing.T) {
		client, err := chroma.NewClient(chroma.WithBasePath(chromaURL))
		require.NoError(t, err)
		_, err = client.Version(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "x509: certificate signed by unknown authority")
	})

	t.Run("Test with self-signed cert in transport", func(t *testing.T) {
		client, err := chroma.NewClient(chroma.WithBasePath(chromaURL), chroma.WithSSLCert(certPath))
		require.NoError(t, err)
		version, err := client.Version(ctx)
		require.NoError(t, err)
		require.NotNil(t, version)
	})
}
