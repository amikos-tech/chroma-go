//go:build basic

package test

import (
	"context"
	"errors"
	"fmt"
	"github.com/Masterminds/semver"
	chroma "github.com/amikos-tech/chroma-go"
	chhttp "github.com/amikos-tech/chroma-go/pkg/commons/http"
	"github.com/amikos-tech/chroma-go/types"
	"github.com/stretchr/testify/require"
	tcchroma "github.com/testcontainers/testcontainers-go/modules/chroma"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAPIErrorHandling(t *testing.T) {
	ctx := context.Background()
	var chromaVersion = "0.6.2"
	var chromaImage = "ghcr.io/chroma-core/chroma"
	if os.Getenv("CHROMA_VERSION") != "" {
		chromaVersion = os.Getenv("CHROMA_VERSION")
	}
	if os.Getenv("CHROMA_IMAGE") != "" {
		chromaImage = os.Getenv("CHROMA_IMAGE")
	}
	chromaContainer, err := tcchroma.Run(ctx,
		fmt.Sprintf("%s:%s", chromaImage, chromaVersion),
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

	t.Run("Test API Error handling Version", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(`{"error":"InvalidArgumentError","message": "bad argument"}`))
			require.NoError(t, err)
		}))
		defer server.Close()
		clientWithTenant, err := chroma.NewClient(chroma.WithBasePath(server.URL))
		require.NoError(t, err)
		require.NotNil(t, clientWithTenant)
		_, err = clientWithTenant.Version(context.Background())
		require.Error(t, err)
		chromaError := err.(*chhttp.ChromaError)
		require.Equal(t, "InvalidArgumentError", chromaError.ErrorID)
		require.Equal(t, "bad argument", chromaError.Message)
		require.Equal(t, 400, chromaError.ErrorCode)
	})
	t.Run("Test API Error handling Heartbeat", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(`{"error":"InvalidArgumentError","message": "bad argument"}`))
			require.NoError(t, err)
		}))
		defer server.Close()
		clientWithTenant, err := chroma.NewClient(chroma.WithBasePath(server.URL))
		require.NoError(t, err)
		require.NotNil(t, clientWithTenant)
		_, err = clientWithTenant.Heartbeat(context.Background())
		require.Error(t, err)
		chromaError := err.(*chhttp.ChromaError)
		require.Equal(t, "InvalidArgumentError", chromaError.ErrorID)
		require.Equal(t, "bad argument", chromaError.Message)
		require.Equal(t, 400, chromaError.ErrorCode)
	})
	t.Run("Test API Error handling GetTenant", func(t *testing.T) {

		_, err = client.Version(ctx)
		require.NoError(t, err)
		if client.APIVersion.LessThan(semver.MustParse("0.4.15")) {
			t.Skipf("Skipping test for API version %s", client.APIVersion.String())
		}
		_, err = client.GetTenant(context.Background(), "dummy")
		require.Error(t, err)
		chromaError := err.(*chhttp.ChromaError)
		if client.APIVersion.GreaterThan(semver.MustParse("0.5.5")) { // >0.5.5
			require.Equal(t, "NotFoundError", chromaError.ErrorID)
			require.Equal(t, "Tenant dummy not found", chromaError.Message)
			require.Equal(t, 404, chromaError.ErrorCode)
		} else {
			require.Contains(t, chromaError.Error(), "NotFoundError")
			require.Contains(t, chromaError.Error(), "Tenant dummy not found")
			require.Equal(t, 500, chromaError.ErrorCode)
		}
	})

	t.Run("Test API Error handling CreateTenant", func(t *testing.T) {
		_, err = client.Version(ctx)
		require.NoError(t, err)
		if client.APIVersion.LessThan(semver.MustParse("0.4.15")) {
			t.Skipf("Skipping test for API version %s", client.APIVersion.String())
		}
		_, err = client.CreateTenant(context.Background(), types.DefaultTenant)
		require.Error(t, err)
		chromaError := err.(*chhttp.ChromaError)
		if client.APIVersion.GreaterThan(semver.MustParse("0.5.5")) { // >0.5.5
			require.Equal(t, "UniqueConstraintError", chromaError.ErrorID)
			require.Equal(t, "Tenant default_tenant already exists", chromaError.Message)
			require.Equal(t, 409, chromaError.ErrorCode)
		} else {
			require.Contains(t, chromaError.Error(), "UniqueConstraintError")
			require.Contains(t, chromaError.Error(), "Tenant default_tenant already exists")
			require.Equal(t, 500, chromaError.ErrorCode)
		}
	})

	t.Run("Test API Error handling GetDatabase", func(t *testing.T) {
		_, err = client.Version(ctx)
		require.NoError(t, err)
		if client.APIVersion.LessThan(semver.MustParse("0.4.15")) {
			t.Skipf("Skipping test for API version %s", client.APIVersion.String())
		}
		var defaultTenant = types.DefaultTenant
		_, err = client.GetDatabase(context.Background(), "dummy", &defaultTenant)
		require.Error(t, err)
		chromaError := err.(*chhttp.ChromaError)
		if client.APIVersion.GreaterThan(semver.MustParse("0.5.5")) { // >0.5.5
			require.Equal(t, "NotFoundError", chromaError.ErrorID)
			require.Equal(t, "Database dummy not found for tenant default_tenant. Are you sure it exists?", chromaError.Message)
			require.Equal(t, 404, chromaError.ErrorCode)
		} else {
			require.Contains(t, chromaError.Error(), "NotFoundError")
			require.Contains(t, chromaError.Error(), "Database dummy not found for tenant default_tenant")
			require.Equal(t, 500, chromaError.ErrorCode)
		}
	})

	t.Run("Test API Error handling CreateDatabase", func(t *testing.T) {
		_, err = client.Version(ctx)
		require.NoError(t, err)
		if client.APIVersion.LessThan(semver.MustParse("0.4.15")) {
			t.Skipf("Skipping test for API version %s", client.APIVersion.String())
		}
		var defaultTenant = types.DefaultTenant
		_, err = client.CreateDatabase(context.Background(), types.DefaultDatabase, &defaultTenant)
		require.Error(t, err)
		chromaError := err.(*chhttp.ChromaError)
		if client.APIVersion.GreaterThan(semver.MustParse("0.5.5")) { // >0.5.5
			require.Equal(t, "UniqueConstraintError", chromaError.ErrorID)
			require.Equal(t, "Database default_database already exists for tenant default_tenant", chromaError.Message)
			require.Equal(t, 409, chromaError.ErrorCode)
		} else {
			require.Contains(t, chromaError.Error(), "UniqueConstraintError")
			require.Contains(t, chromaError.Error(), "Database default_database already exists for tenant default_tenant")
			require.Equal(t, 500, chromaError.ErrorCode)
		}
	})

	t.Run("Test API Error handling NewCollection", func(t *testing.T) {
		_, err = client.Version(ctx)
		require.NoError(t, err)
		_, err = client.NewCollection(context.Background(), "test_collection")
		require.NoError(t, err)
		_, err = client.NewCollection(context.Background(), "test_collection")
		require.Error(t, err)
		chromaError := err.(*chhttp.ChromaError)
		if client.APIVersion.GreaterThan(semver.MustParse("0.5.5")) { // >0.5.5
			require.Equal(t, "UniqueConstraintError", chromaError.ErrorID)
			require.Equal(t, "Collection test_collection already exists", chromaError.Message)
			require.Equal(t, 409, chromaError.ErrorCode)
		} else {
			require.Contains(t, chromaError.Error(), "Collection test_collection already exists")
			require.Equal(t, 500, chromaError.ErrorCode)
		}
	})

	t.Run("Test API Error handling DeleteCollection", func(t *testing.T) {
		_, err = client.Version(ctx)
		require.NoError(t, err)
		_, err = client.DeleteCollection(context.Background(), "test_collection_for_delete")
		require.Error(t, err)
		chromaError := err.(*chhttp.ChromaError)
		if client.APIVersion.GreaterThan(semver.MustParse("0.5.5")) { // >0.5.5
			require.Equal(t, "InvalidCollection", chromaError.ErrorID)
			require.Equal(t, "Collection test_collection_for_delete does not exist.", chromaError.Message)
			require.Equal(t, 400, chromaError.ErrorCode)
		} else {
			require.Contains(t, chromaError.Error(), "ValueError")
			require.Contains(t, chromaError.Error(), "Collection test_collection_for_delete does not exist.")
			require.Equal(t, 500, chromaError.ErrorCode)
		}
	})

	t.Run("Test API Error handling Reset", func(t *testing.T) {
		_, err = client.Version(ctx)
		require.NoError(t, err)
		_, err = client.Reset(context.Background())
		require.Error(t, err)
		chromaError := err.(*chhttp.ChromaError)
		if client.APIVersion.GreaterThan(semver.MustParse("0.5.5")) { // >0.5.5
			require.Equal(t, "InvalidArgumentError", chromaError.ErrorID)
			require.Contains(t, chromaError.Message, "Resetting is not allowed by this configuration")
			require.Equal(t, 400, chromaError.ErrorCode)
		} else {
			require.Contains(t, chromaError.Error(), "ValueError")
			require.Contains(t, chromaError.Error(), "Resetting is not allowed by this configuration")
			require.Equal(t, 500, chromaError.ErrorCode)

		}
	})

	t.Run("Test API Error handling ListCollections", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "pre-flight-checks") {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"max_batch_size": 41666}`))
				require.NoError(t, err)
				return
			}
			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "version") {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`0.5.17`))
				require.NoError(t, err)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte(`{"error":"InternalServerError","message": "something went wrong"}`))
			require.NoError(t, err)
		}))
		defer server.Close()
		clientWithTenant, err := chroma.NewClient(chroma.WithBasePath(server.URL))
		require.NoError(t, err)
		require.NotNil(t, clientWithTenant)
		_, err = clientWithTenant.ListCollections(context.Background())
		require.Error(t, err)
		var chromaError *chhttp.ChromaError
		errors.As(err, &chromaError)
		require.Equal(t, "InternalServerError", chromaError.ErrorID)
		require.Equal(t, "something went wrong", chromaError.Message)
		require.Equal(t, http.StatusInternalServerError, chromaError.ErrorCode)
	})

	t.Run("Test API Error handling CountCollections", func(t *testing.T) {
		server := getMockServer(t)
		defer server.Close()
		clientWithTenant, err := chroma.NewClient(chroma.WithBasePath(server.URL))
		require.NoError(t, err)
		require.NotNil(t, clientWithTenant)
		_, err = clientWithTenant.CountCollections(context.Background())
		require.Error(t, err)
		var chromaError *chhttp.ChromaError
		errors.As(err, &chromaError)
		require.Equal(t, "InternalServerError", chromaError.ErrorID)
		require.Equal(t, "something went wrong", chromaError.Message)
		require.Equal(t, http.StatusInternalServerError, chromaError.ErrorCode)
	})

	t.Run("Test API Error handling Add", func(t *testing.T) {
		server := getMockServer(t)
		defer server.Close()
		clientWithTenant, err := chroma.NewClient(chroma.WithBasePath(server.URL))
		require.NoError(t, err)
		require.NotNil(t, clientWithTenant)
		col, err := clientWithTenant.GetCollection(context.Background(), "test_collection", nil)
		require.NoError(t, err)
		_, err = col.Add(context.Background(), nil, nil, []string{"test"}, []string{"test"})
		require.Error(t, err)
		var chromaError *chhttp.ChromaError
		errors.As(err, &chromaError)
		require.Equal(t, "InternalServerError", chromaError.ErrorID)
		require.Equal(t, "something went wrong", chromaError.Message)
		require.Equal(t, http.StatusInternalServerError, chromaError.ErrorCode)
	})

	t.Run("Test API Error handling Delete", func(t *testing.T) {
		server := getMockServer(t)
		defer server.Close()
		clientWithTenant, err := chroma.NewClient(chroma.WithBasePath(server.URL))
		require.NoError(t, err)
		require.NotNil(t, clientWithTenant)
		col, err := clientWithTenant.GetCollection(context.Background(), "test_collection", nil)
		require.NoError(t, err)
		_, err = col.Delete(context.Background(), []string{"test"}, nil, nil)
		require.Error(t, err)
		var chromaError *chhttp.ChromaError
		errors.As(err, &chromaError)
		require.Equal(t, "InternalServerError", chromaError.ErrorID)
		require.Equal(t, "something went wrong", chromaError.Message)
		require.Equal(t, http.StatusInternalServerError, chromaError.ErrorCode)
	})

	t.Run("Test API Error handling Modify", func(t *testing.T) {
		server := getMockServer(t)
		defer server.Close()
		clientWithTenant, err := chroma.NewClient(chroma.WithBasePath(server.URL))
		require.NoError(t, err)
		require.NotNil(t, clientWithTenant)
		col, err := clientWithTenant.GetCollection(context.Background(), "test_collection", nil)
		require.NoError(t, err)
		_, err = col.Modify(context.Background(), nil, nil, []string{"test"}, []string{"test"})
		require.Error(t, err)
		var chromaError *chhttp.ChromaError
		errors.As(err, &chromaError)
		require.Equal(t, "InternalServerError", chromaError.ErrorID)
		require.Equal(t, "something went wrong", chromaError.Message)
		require.Equal(t, http.StatusInternalServerError, chromaError.ErrorCode)
	})
	t.Run("Test API Error handling Upsert", func(t *testing.T) {
		server := getMockServer(t)
		defer server.Close()
		clientWithTenant, err := chroma.NewClient(chroma.WithBasePath(server.URL))
		require.NoError(t, err)
		require.NotNil(t, clientWithTenant)
		col, err := clientWithTenant.GetCollection(context.Background(), "test_collection", nil)
		require.NoError(t, err)
		_, err = col.Upsert(context.Background(), nil, nil, []string{"test"}, []string{"test"})
		require.Error(t, err)
		var chromaError *chhttp.ChromaError
		errors.As(err, &chromaError)
		require.Equal(t, "InternalServerError", chromaError.ErrorID)
		require.Equal(t, "something went wrong", chromaError.Message)
		require.Equal(t, http.StatusInternalServerError, chromaError.ErrorCode)
	})

	t.Run("Test API Error handling Count", func(t *testing.T) {
		server := getMockServer(t)
		defer server.Close()
		clientWithTenant, err := chroma.NewClient(chroma.WithBasePath(server.URL))
		require.NoError(t, err)
		require.NotNil(t, clientWithTenant)
		col, err := clientWithTenant.GetCollection(context.Background(), "test_collection", nil)
		require.NoError(t, err)
		_, err = col.Count(context.Background())
		require.Error(t, err)
		var chromaError *chhttp.ChromaError
		errors.As(err, &chromaError)
		require.Equal(t, "InternalServerError", chromaError.ErrorID)
		require.Equal(t, "something went wrong", chromaError.Message)
		require.Equal(t, http.StatusInternalServerError, chromaError.ErrorCode)
	})
}

func TestTimeout(t *testing.T) {
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
}
