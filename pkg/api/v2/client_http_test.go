//go:build basicv2 && !cloud

package v2

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/require"

	chhttp "github.com/amikos-tech/chroma-go/pkg/commons/http"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

type failingCredentialsProvider struct {
	err error
}

func (f failingCredentialsProvider) Authenticate() (map[string]string, error) {
	return nil, f.err
}

func MetadataModel() gopter.Gen {
	return gen.SliceOf(
		gen.Struct(reflect.TypeOf(struct {
			Key   string
			Value interface{}
		}{}), map[string]gopter.Gen{
			"Key":   gen.Identifier(),
			"Value": gen.OneGenOf(gen.Int64(), gen.Float64(), gen.AlphaString(), gen.Bool()),
		}),
	).Map(func(entries *gopter.GenResult) CollectionMetadata {
		result := make(map[string]interface{})
		for _, entry := range entries.Result.([]struct {
			Key   string
			Value interface{}
		}) {
			result[entry.Key] = entry.Value
		}
		return NewMetadataFromMap(result)
	})
}

// CollectionIDStrategy generates random UUIDs as a gopter generator.
func CollectionIDStrategy() gopter.Gen {
	return func(params *gopter.GenParameters) *gopter.GenResult {
		id := uuid.New() // Generates a new random UUID
		return gopter.NewGenResult(id.String(), gopter.NoShrinker)
	}
}

func TenantStrategy() gopter.Gen {
	return gen.OneGenOf(func(params *gopter.GenParameters) *gopter.GenResult {
		id := uuid.New() // Generates a new random UUID
		return gopter.NewGenResult(id.String(), gopter.NoShrinker)
	}, func(params *gopter.GenParameters) *gopter.GenResult {
		return gopter.NewGenResult(DefaultTenant, gopter.NoShrinker)
	})
}

func DatabaseStrategy() gopter.Gen {
	return gen.OneGenOf(func(params *gopter.GenParameters) *gopter.GenResult {
		id := uuid.New() // Generates a new random UUID
		return gopter.NewGenResult(id.String(), gopter.NoShrinker)
	}, func(params *gopter.GenParameters) *gopter.GenResult {
		return gopter.NewGenResult(DefaultDatabase, gopter.NoShrinker)
	})
}

func CollectionModelStrategy() gopter.Gen {
	return gen.Struct(reflect.TypeOf(CollectionModel{}), map[string]gopter.Gen{
		"ID":       CollectionIDStrategy(),
		"Name":     gen.AlphaString(),
		"Tenant":   TenantStrategy(),
		"Database": DatabaseStrategy(),
		"Metadata": MetadataModel(),
	})
}

// Property-based test for creating collections
func TestCreateCollectionProperty(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)

	properties.Property("CreateCollection handles different names and metadata", prop.ForAll(
		func(name string, col CollectionModel) bool {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				respBody := chhttp.ReadRespBody(r.Body)
				var op CreateCollectionOp
				err := json.Unmarshal([]byte(respBody), &op)
				require.NoError(t, err)
				require.Equal(t, name, op.Name)
				// Configuration is now included with EF info
				require.NotNil(t, op.Configuration)
				cm := CollectionModel{
					ID:       col.ID,
					Name:     col.Name,
					Tenant:   col.Tenant,
					Database: col.Database,
					Metadata: col.Metadata,
				}
				w.WriteHeader(http.StatusOK)
				err = json.NewEncoder(w).Encode(&cm)
				require.NoError(t, err)
			}))
			defer server.Close()

			client, err := NewHTTPClient(WithBaseURL(server.URL), WithDatabaseAndTenant(col.Database, col.Tenant))

			require.NoError(t, err)

			// Call API with random data
			c, err := client.CreateCollection(context.Background(), name, WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
			require.NoError(t, err)
			require.NotNil(t, c)
			require.Equal(t, col.ID, c.ID())
			require.Equal(t, col.Name, c.Name())
			require.Equal(t, col.Tenant, c.Tenant().Name())
			require.Equal(t, col.Database, c.Database().Name())
			require.ElementsMatch(t, col.Metadata.Keys(), c.Metadata().Keys())
			for _, k := range col.Metadata.Keys() {
				val1, ok1 := col.Metadata.GetRaw(k)
				require.True(t, ok1)
				metadataValue1, ok11 := val1.(MetadataValue)
				require.True(t, ok11)
				val2, ok2 := c.Metadata().GetRaw(k)
				require.True(t, ok2)
				metadataValue2, ok22 := val2.(MetadataValue)
				require.True(t, ok22)
				r1, _ := metadataValue1.GetRaw()
				r2, _ := metadataValue2.GetRaw()
				if !metadataValue1.Equal(&metadataValue2) {
					fmt.Println(col.Metadata.GetRaw(k))
					fmt.Println(c.Metadata().GetRaw(k))
					fmt.Printf("%T != %T\n", r1, r2)
					fmt.Println(k, r1, r2, metadataValue1.Equal(&metadataValue2))
				}
				require.Truef(t, metadataValue1.Equal(&metadataValue2), "metadata values are not equal: %v != %v", metadataValue1, metadataValue2)
			}
			return true
		},
		gen.AlphaString().SuchThat(func(v interface{}) bool {
			return len(v.(string)) > 0
		}), // Random collection name
		CollectionModelStrategy(), // Random collection
	))

	properties.TestingRun(t)
}

func TestAPIClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		respBody := chhttp.ReadRespBody(r.Body)
		t.Logf("Body: %s", respBody)
		switch {
		case r.URL.Path == "/api/v2/version" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`0.6.3`))
			require.NoError(t, err)
		case r.URL.Path == "/api/v2/heartbeat" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"nanosecond heartbeat":1732127707371421353}`))
			require.NoError(t, err)
		case r.URL.Path == "/api/v2/tenants/default_tenant" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"name":"default_tenant"}`))
			require.NoError(t, err)
		case r.URL.Path == "/api/v2/tenants" && r.Method == http.MethodPost:
			require.JSONEq(t, `{"name":"test_tenant"}`, respBody)
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{}`))
			require.NoError(t, err)
		// create database
		case r.URL.Path == "/api/v2/tenants/test_tenant/databases" && r.Method == http.MethodPost:
			require.JSONEq(t, `{"name":"test_db"}`, respBody)
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{}`))
			require.NoError(t, err)
		// get database
		case r.URL.Path == "/api/v2/tenants/test_tenant/databases/test_db" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{
  "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "name": "test_db",
  "tenant": "test_tenant"
}`))
			require.NoError(t, err)
		case r.URL.Path == "/api/v2/tenants/test_tenant/databases" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`[
{
  "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "name": "test_db1",
  "tenant": "test_tenant"
},
{
  "id": "2fa85f64-5717-4562-b3fc-2c963f66afa6",
  "name": "test_db2",
  "tenant": "test_tenant"
}
]`))
			require.NoError(t, err)
		// Delete database
		case r.URL.Path == "/api/v2/tenants/test_tenant/databases/test_db" && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{}`))
			require.NoError(t, err)
		case r.URL.Path == "/api/v2/tenants/default_tenant/databases/default_database/collections_count" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`100`))
			require.NoError(t, err)
		case r.URL.Path == "/api/v2/tenants/default_tenant/databases/default_database/collections" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`[
  {
    "id": "8ecf0f7e-e806-47f8-96a1-4732ef42359e",
    "configuration_json": {
      "hnsw_configuration": {
        "space": "l2",
        "ef_construction": 100,
        "ef_search": 10,
        "num_threads": 14,
        "M": 16,
        "resize_factor": 1.2,
        "batch_size": 100,
        "sync_threshold": 1000,
        "_type": "HNSWConfigurationInternal"
      },
      "_type": "CollectionConfigurationInternal"
    },
    "database": "default_database",
    "dimension": 384,
    "log_position": 0,
    "metadata": {
      "t": 1
    },
    "name": "testcoll",
    "tenant": "default_tenant",
    "version": 0
  }
]`))
			require.NoError(t, err)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client, err := NewHTTPClient(WithBaseURL(server.URL))
	require.NoError(t, err)

	t.Run("GetVersion", func(t *testing.T) {
		ver, err := client.GetVersion(context.Background())
		require.NoError(t, err)
		require.NotNil(t, ver)
		require.Equal(t, "0.6.3", ver)
	})
	t.Run("Hearbeat", func(t *testing.T) {
		err := client.Heartbeat(context.Background())
		require.NoError(t, err)
	})

	t.Run("GetTenant", func(t *testing.T) {
		tenant, err := client.GetTenant(context.Background(), NewDefaultTenant())
		require.NoError(t, err)
		require.NotNil(t, tenant)
		require.Equal(t, "default_tenant", tenant.Name())
	})

	t.Run("CreateTenant", func(t *testing.T) {
		tenant, err := client.CreateTenant(context.Background(), NewTenant("test_tenant"))
		require.NoError(t, err)
		require.NotNil(t, tenant)
		require.Equal(t, "test_tenant", tenant.Name())
	})

	t.Run("CreateDatabase", func(t *testing.T) {
		db, err := client.CreateDatabase(context.Background(), NewTenant("test_tenant").Database("test_db"))
		require.NoError(t, err)
		require.NotNil(t, db)
		require.Equal(t, "test_db", db.Name())
	})

	t.Run("ListDatabases", func(t *testing.T) {
		dbs, err := client.ListDatabases(context.Background(), NewTenant("test_tenant"))
		require.NoError(t, err)
		require.NotNil(t, dbs)
		require.Len(t, dbs, 2)
		require.Equal(t, "test_db1", dbs[0].Name())
		require.Equal(t, "test_tenant", dbs[0].Tenant().Name())
		require.Equal(t, "test_db2", dbs[1].Name())
		require.Equal(t, "test_tenant", dbs[1].Tenant().Name())
	})

	t.Run("GetDatabase", func(t *testing.T) {
		db, err := client.GetDatabase(context.Background(), NewTenant("test_tenant").Database("test_db"))
		require.NoError(t, err)
		require.NotNil(t, db)
		require.Equal(t, "test_db", db.Name())
		require.Equal(t, "test_tenant", db.Tenant().Name())
		require.Equal(t, "3fa85f64-5717-4562-b3fc-2c963f66afa6", db.ID())
	})

	t.Run("DeleteDatabase", func(t *testing.T) {
		err := client.DeleteDatabase(context.Background(), NewTenant("test_tenant").Database("test_db"))
		require.NoError(t, err)
	})

	t.Run("CountCollections", func(t *testing.T) {
		count, err := client.CountCollections(context.Background())
		require.NoError(t, err)
		require.Equal(t, 100, count)
	})

	t.Run("ListCollections", func(t *testing.T) {
		cols, err := client.ListCollections(context.Background())
		require.NoError(t, err)
		require.NotNil(t, cols)
		require.Len(t, cols, 1)
		c := cols[0]
		require.Equal(t, "8ecf0f7e-e806-47f8-96a1-4732ef42359e", c.ID())
		require.Equal(t, 384, c.Dimension())
		require.Equal(t, "testcoll", c.Name())
		require.Equal(t, NewDefaultTenant(), c.Tenant())
		require.Equal(t, NewDefaultDatabase(), c.Database())
		require.NotNil(t, c.Metadata())
		vi, ok := c.Metadata().GetInt("t")
		require.True(t, ok)
		require.Equal(t, int64(1), vi)
	})

	t.Run("CreateCollection", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("Request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
			respBody := chhttp.ReadRespBody(r.Body)
			t.Logf("Body: %s", respBody)

			switch {
			case r.URL.Path == "/api/v2/tenants/default_tenant/databases/default_database/collections" && r.Method == http.MethodPost:
				w.WriteHeader(http.StatusOK)
				var op CreateCollectionOp
				err := json.Unmarshal([]byte(respBody), &op)
				require.NoError(t, err)
				require.Equal(t, "test", op.Name)
				require.NotNil(t, op.Configuration) // Configuration now includes EF info
				values, err := url.ParseQuery(r.URL.RawQuery)
				require.NoError(t, err)
				cm := CollectionModel{
					ID:       "8ecf0f7e-e806-47f8-96a1-4732ef42359e",
					Name:     op.Name,
					Tenant:   values.Get("tenant"),
					Database: values.Get("database"),
					Metadata: op.Metadata,
				}
				err = json.NewEncoder(w).Encode(&cm)
				require.NoError(t, err)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()
		innerClient, err := NewHTTPClient(WithBaseURL(server.URL))
		require.NoError(t, err)
		c, err := innerClient.CreateCollection(context.Background(), "test", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		require.NotNil(t, c)
	})

	t.Run("GetOrCreateCollection", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("Request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
			respBody := chhttp.ReadRespBody(r.Body)
			t.Logf("Body: %s", respBody)

			switch {
			case r.URL.Path == "/api/v2/tenants/default_tenant/databases/default_database/collections" && r.Method == http.MethodPost:
				w.WriteHeader(http.StatusOK)
				var reqBody map[string]interface{}
				require.NoError(t, json.Unmarshal([]byte(respBody), &reqBody))
				require.Equal(t, "test", reqBody["name"])
				require.Equal(t, true, reqBody["get_or_create"])
				values, err := url.ParseQuery(r.URL.RawQuery)
				require.NoError(t, err)
				var op CreateCollectionOp
				err = json.Unmarshal([]byte(respBody), &op)
				require.NoError(t, err)
				cm := CollectionModel{
					ID:        "8ecf0f7e-e806-47f8-96a1-4732ef42359e",
					Name:      op.Name,
					Tenant:    values.Get("tenant"),
					Database:  values.Get("database"),
					Metadata:  op.Metadata,
					Dimension: 9001,
				}
				err = json.NewEncoder(w).Encode(&cm)
				require.NoError(t, err)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()
		innerClient, err := NewHTTPClient(WithBaseURL(server.URL))
		require.NoError(t, err)
		c, err := innerClient.GetOrCreateCollection(context.Background(), "test", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		require.NotNil(t, c)
		require.Equal(t, "8ecf0f7e-e806-47f8-96a1-4732ef42359e", c.ID())
		require.Equal(t, "test", c.Name())
		require.Equal(t, 9001, c.Dimension())
	})

	t.Run("GetCollection", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("Request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)

			switch {
			case r.URL.Path == "/api/v2/tenants/default_tenant/databases/default_database/collections/test" && r.Method == http.MethodGet:
				w.WriteHeader(http.StatusOK)
				require.NoError(t, err)
				cm := CollectionModel{
					ID:        "8ecf0f7e-e806-47f8-96a1-4732ef42359e",
					Name:      "test",
					Tenant:    "default_tenant",
					Database:  "default_database",
					Metadata:  NewMetadataFromMap(map[string]any{"t": 1}),
					Dimension: 9001,
				}
				err = json.NewEncoder(w).Encode(&cm)
				require.NoError(t, err)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()
		innerClient, err := NewHTTPClient(WithBaseURL(server.URL))
		require.NoError(t, err)
		c, err := innerClient.GetCollection(context.Background(), "test", WithEmbeddingFunctionGet(embeddings.NewConsistentHashEmbeddingFunction()))
		// TODO also test with tenant and database and EF
		require.NoError(t, err)
		require.NotNil(t, c)
		require.Equal(t, "8ecf0f7e-e806-47f8-96a1-4732ef42359e", c.ID())
		require.Equal(t, "test", c.Name())
		require.Equal(t, NewDefaultTenant(), c.Tenant())
		require.Equal(t, NewDefaultDatabase(), c.Database())
		require.NotNil(t, c.Metadata())
		require.Equal(t, 9001, c.Dimension())
		vi, ok := c.Metadata().GetInt("t")
		require.True(t, ok)
		require.Equal(t, int64(1), vi)
	})
}

func TestGetCollection_ParsesSpannQuantizeFromSchemaResponse(t *testing.T) {
	schema, err := NewSchema(
		WithDefaultVectorIndex(NewVectorIndexConfig(
			WithSpace(SpaceL2),
			WithSpann(NewSpannConfig(
				WithSpannQuantize(SpannQuantizationFourBitRabitQWithUSearch),
			)),
		)),
	)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/tenants/default_tenant/databases/default_database/collections/test" &&
			r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			cm := CollectionModel{
				ID:       "8ecf0f7e-e806-47f8-96a1-4732ef42359e",
				Name:     "test",
				Tenant:   DefaultTenant,
				Database: DefaultDatabase,
				Schema:   schema,
			}
			require.NoError(t, json.NewEncoder(w).Encode(&cm))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewHTTPClient(WithBaseURL(server.URL), WithLogger(testLogger()))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, client.Close())
	}()

	collection, err := client.GetCollection(context.Background(), "test")
	require.NoError(t, err)
	require.NotNil(t, collection)
	require.NotNil(t, collection.Schema())

	embeddingVT, ok := collection.Schema().GetKey(EmbeddingKey)
	require.True(t, ok)
	require.NotNil(t, embeddingVT.FloatList)
	require.NotNil(t, embeddingVT.FloatList.VectorIndex)
	require.NotNil(t, embeddingVT.FloatList.VectorIndex.Config)
	require.NotNil(t, embeddingVT.FloatList.VectorIndex.Config.Spann)
	require.Equal(t, SpannQuantizationFourBitRabitQWithUSearch, embeddingVT.FloatList.VectorIndex.Config.Spann.Quantize)
}

func TestListCollections_ParsesSpannQuantizeFromSchemaResponse(t *testing.T) {
	schema, err := NewSchema(
		WithDefaultVectorIndex(NewVectorIndexConfig(
			WithSpace(SpaceL2),
			WithSpann(NewSpannConfig(
				WithSpannQuantize(SpannQuantizationFourBitRabitQWithUSearch),
			)),
		)),
	)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/tenants/default_tenant/databases/default_database/collections" &&
			r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			collections := []CollectionModel{
				{
					ID:       "8ecf0f7e-e806-47f8-96a1-4732ef42359e",
					Name:     "test",
					Tenant:   DefaultTenant,
					Database: DefaultDatabase,
					Schema:   schema,
				},
			}
			require.NoError(t, json.NewEncoder(w).Encode(&collections))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewHTTPClient(WithBaseURL(server.URL), WithLogger(testLogger()))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, client.Close())
	}()

	collections, err := client.ListCollections(context.Background())
	require.NoError(t, err)
	require.Len(t, collections, 1)
	require.NotNil(t, collections[0].Schema())

	embeddingVT, ok := collections[0].Schema().GetKey(EmbeddingKey)
	require.True(t, ok)
	require.NotNil(t, embeddingVT.FloatList)
	require.NotNil(t, embeddingVT.FloatList.VectorIndex)
	require.NotNil(t, embeddingVT.FloatList.VectorIndex.Config)
	require.NotNil(t, embeddingVT.FloatList.VectorIndex.Config.Spann)
	require.Equal(t, SpannQuantizationFourBitRabitQWithUSearch, embeddingVT.FloatList.VectorIndex.Config.Spann.Quantize)
}

func TestGetCollection_BuildErrorGuard(t *testing.T) {
	configuration := NewCollectionConfiguration()
	configuration.SetEmbeddingFunctionInfo(&EmbeddingFunctionInfo{
		Type:   "known",
		Name:   "nonexistent_provider_xyz",
		Config: map[string]any{},
	})
	configurationMap, err := marshalToMap(configuration)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/tenants/default_tenant/databases/default_database/collections/test" &&
			r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			require.NoError(t, json.NewEncoder(w).Encode(&CollectionModel{
				ID:                "8ecf0f7e-e806-47f8-96a1-4732ef42359e",
				Name:              "test",
				Tenant:            DefaultTenant,
				Database:          DefaultDatabase,
				ConfigurationJSON: configurationMap,
			}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewHTTPClient(WithBaseURL(server.URL), WithLogger(testLogger()))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, client.Close())
	}()

	collection, err := client.GetCollection(context.Background(), "test")
	require.NoError(t, err)

	impl, ok := collection.(*CollectionImpl)
	require.True(t, ok)
	require.Nil(t, impl.embeddingFunction, "EF must stay nil when auto-wire build fails")
	require.Nil(t, impl.contentEmbeddingFunction, "content EF must stay nil when auto-wire build fails")
}

func TestListCollections_BuildErrorGuard(t *testing.T) {
	configuration := NewCollectionConfiguration()
	configuration.SetEmbeddingFunctionInfo(&EmbeddingFunctionInfo{
		Type:   "known",
		Name:   "nonexistent_provider_xyz",
		Config: map[string]any{},
	})
	configurationMap, err := marshalToMap(configuration)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/tenants/default_tenant/databases/default_database/collections" &&
			r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			require.NoError(t, json.NewEncoder(w).Encode([]CollectionModel{
				{
					ID:                "8ecf0f7e-e806-47f8-96a1-4732ef42359e",
					Name:              "test",
					Tenant:            DefaultTenant,
					Database:          DefaultDatabase,
					ConfigurationJSON: configurationMap,
				},
			}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewHTTPClient(WithBaseURL(server.URL), WithLogger(testLogger()))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, client.Close())
	}()

	collections, err := client.ListCollections(context.Background())
	require.NoError(t, err)
	require.Len(t, collections, 1)

	impl, ok := collections[0].(*CollectionImpl)
	require.True(t, ok)
	require.Nil(t, impl.embeddingFunction, "EF must stay nil when auto-wire build fails")
	require.Nil(t, impl.contentEmbeddingFunction, "content EF must stay nil when auto-wire build fails")
}

func TestCreateCollection(t *testing.T) {
	var tests = []struct {
		name                        string
		validateRequestWithResponse func(w http.ResponseWriter, r *http.Request)
		sendRequest                 func(client Client)
	}{
		{
			name: "with name only",
			validateRequestWithResponse: func(w http.ResponseWriter, r *http.Request) {
				respBody := chhttp.ReadRespBody(r.Body)
				respMap := make(map[string]any)
				err := json.Unmarshal([]byte(respBody), &respMap)
				require.NoError(t, err)
				require.Equal(t, "test", respMap["name"])
				w.WriteHeader(http.StatusOK)
				_, err = w.Write([]byte(`{"id":"8ecf0f7e-e806-47f8-96a1-4732ef42359e","name":"test"}`))
				require.NoError(t, err)
			},
			sendRequest: func(client Client) {
				collection, err := client.CreateCollection(context.Background(), "test", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
				require.NoError(t, err)
				require.NotNil(t, collection)
				require.Equal(t, "8ecf0f7e-e806-47f8-96a1-4732ef42359e", collection.ID())
				require.Equal(t, "test", collection.Name())
			},
		},
		{
			name: "with metadata",
			validateRequestWithResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				respBody := chhttp.ReadRespBody(r.Body)
				var op CreateCollectionOp
				err := json.Unmarshal([]byte(respBody), &op)
				require.NoError(t, err)
				v, ok := op.Metadata.GetInt("int")
				require.True(t, ok)
				require.Equal(t, int64(1), v)
				vf, ok := op.Metadata.GetFloat("float")
				require.True(t, ok)
				require.Equal(t, 1.1, vf)
				vs, ok := op.Metadata.GetString("string")
				require.True(t, ok)
				require.Equal(t, "test", vs)
				vb, ok := op.Metadata.GetBool("bool")
				require.True(t, ok)
				require.True(t, vb)
				cm := CollectionModel{
					ID:       "8ecf0f7e-e806-47f8-96a1-4732ef42359e",
					Name:     op.Name,
					Tenant:   "default_tenant",
					Database: "default_database",
					Metadata: op.Metadata,
				}
				err = json.NewEncoder(w).Encode(&cm)
				require.NoError(t, err)
			},
			sendRequest: func(client Client) {
				collection, err := client.CreateCollection(context.Background(), "test",
					WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()),
					WithCollectionMetadataCreate(
						NewMetadataFromMap(map[string]any{"int": 1, "float": 1.1, "string": "test", "bool": true})),
				)
				require.NoError(t, err)
				require.NotNil(t, collection)
				require.Equal(t, "8ecf0f7e-e806-47f8-96a1-4732ef42359e", collection.ID())
				require.Equal(t, "test", collection.Name())
				vf, ok := collection.Metadata().GetFloat("float")
				require.True(t, ok)
				require.Equal(t, 1.1, vf)
				vs, ok := collection.Metadata().GetString("string")
				require.True(t, ok)
				require.Equal(t, "test", vs)
				vb, ok := collection.Metadata().GetBool("bool")
				require.True(t, ok)
				require.True(t, vb)
				vi, ok := collection.Metadata().GetInt("int")
				require.True(t, ok)
				require.Equal(t, int64(1), vi)
				require.Equal(t, NewDefaultTenant(), collection.Tenant())
				require.Equal(t, NewDefaultDatabase(), collection.Database())
			},
		},
		{
			name: "with HNSW params",
			validateRequestWithResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				respBody := chhttp.ReadRespBody(r.Body)
				var op CreateCollectionOp
				err := json.Unmarshal([]byte(respBody), &op)
				require.NoError(t, err)
				var vi int64
				var vs string
				var vf float64
				var ok bool
				vs, ok = op.Metadata.GetString(HNSWSpace)
				require.True(t, ok)
				require.Equal(t, string(embeddings.L2), vs)
				vi, ok = op.Metadata.GetInt(HNSWNumThreads)
				require.True(t, ok)
				require.Equal(t, int64(14), vi)
				vf, ok = op.Metadata.GetFloat(HNSWResizeFactor)
				require.True(t, ok)
				require.Equal(t, 1.2, vf)
				vi, ok = op.Metadata.GetInt(HNSWBatchSize)
				require.True(t, ok)
				require.Equal(t, int64(2000), vi)
				vi, ok = op.Metadata.GetInt(HNSWSyncThreshold)
				require.True(t, ok)
				require.Equal(t, int64(10000), vi)
				vi, ok = op.Metadata.GetInt(HNSWConstructionEF)
				require.True(t, ok)
				require.Equal(t, int64(100), vi)
				vi, ok = op.Metadata.GetInt(HNSWSearchEF)
				require.True(t, ok)
				require.Equal(t, int64(999), vi)
				cm := CollectionModel{
					ID:       "8ecf0f7e-e806-47f8-96a1-4732ef42359e",
					Name:     op.Name,
					Tenant:   DefaultTenant,
					Database: DefaultDatabase,
					Metadata: op.Metadata,
				}
				err = json.NewEncoder(w).Encode(&cm)
				require.NoError(t, err)
			},
			sendRequest: func(client Client) {
				collection, err := client.CreateCollection(
					context.Background(),
					"test",
					WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()),
					WithHNSWSpaceCreate(embeddings.L2),
					WithHNSWMCreate(100),
					WithHNSWNumThreadsCreate(14),
					WithHNSWResizeFactorCreate(1.2),
					WithHNSWBatchSizeCreate(2000),
					WithHNSWSyncThresholdCreate(10000),
					WithHNSWConstructionEfCreate(100),
					WithHNSWSearchEfCreate(999),
				)
				require.NoError(t, err)
				require.NotNil(t, collection)
				require.Equal(t, "8ecf0f7e-e806-47f8-96a1-4732ef42359e", collection.ID())
				require.Equal(t, "test", collection.Name())
				hnswSpace, ok := collection.Metadata().GetString(HNSWSpace)
				require.True(t, ok)
				require.Equal(t, string(embeddings.L2), hnswSpace)
				hnswNumThreads, ok := collection.Metadata().GetInt(HNSWNumThreads)
				require.True(t, ok)
				require.Equal(t, int64(14), hnswNumThreads)
				hnswResizeFactor, ok := collection.Metadata().GetFloat(HNSWResizeFactor)
				require.True(t, ok)
				require.Equal(t, 1.2, hnswResizeFactor)
				hnswBatchSize, ok := collection.Metadata().GetInt(HNSWBatchSize)
				require.True(t, ok)
				require.Equal(t, int64(2000), hnswBatchSize)
				hnswSyncThreshold, ok := collection.Metadata().GetInt(HNSWSyncThreshold)
				require.True(t, ok)
				require.Equal(t, int64(10000), hnswSyncThreshold)
				hnswConstructionEf, ok := collection.Metadata().GetInt(HNSWConstructionEF)
				require.True(t, ok)
				require.Equal(t, int64(100), hnswConstructionEf)
				hnswSearchEf, ok := collection.Metadata().GetInt(HNSWSearchEF)
				require.True(t, ok)
				require.Equal(t, int64(999), hnswSearchEf)
			},
		},
		{
			name: "with tenant and database",
			validateRequestWithResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				respBody := chhttp.ReadRespBody(r.Body)
				var op CreateCollectionOp
				err := json.Unmarshal([]byte(respBody), &op)
				require.NoError(t, err)
				require.Contains(t, "mytenant", r.URL.RawQuery)
				require.Contains(t, "mydb", r.URL.RawQuery)
				cm := CollectionModel{
					ID:       "8ecf0f7e-e806-47f8-96a1-4732ef42359e",
					Name:     op.Name,
					Tenant:   "mytenant",
					Database: "mydb",
					Metadata: op.Metadata,
				}
				err = json.NewEncoder(w).Encode(&cm)
				require.NoError(t, err)
			},
			sendRequest: func(client Client) {
				collection, err := client.CreateCollection(
					context.Background(),
					"test",
					WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()),
					WithDatabaseCreate(NewTenant("mytenant").Database("mydb")),
				)
				require.NoError(t, err)
				require.NotNil(t, collection)
				require.Equal(t, "8ecf0f7e-e806-47f8-96a1-4732ef42359e", collection.ID())
				require.Equal(t, "test", collection.Name())
				require.Equal(t, NewTenant("mytenant"), collection.Tenant())
				require.Equal(t, NewDatabase("mydb", NewTenant("mytenant")), collection.Database())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Logf("Request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				matched, err := regexp.MatchString(`/api/v2/tenants/[^/]+/databases/[^/]+/collections`, r.URL.Path)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				switch {
				case matched && r.Method == http.MethodPost:
					tt.validateRequestWithResponse(w, r)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			client, err := NewHTTPClient(WithBaseURL(server.URL), WithLogger(testLogger()))
			require.NoError(t, err)
			tt.sendRequest(client)
			err = client.Close()
			require.NoError(t, err)
		})
	}
}

func TestClientClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		matched, err := regexp.MatchString(`/api/v2/tenants/[^/]+/databases/[^/]+/collections`, r.URL.Path)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		switch {
		case matched && r.Method == http.MethodPost:
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	client, err := NewHTTPClient(WithBaseURL(server.URL), WithLogger(testLogger()))
	require.NoError(t, err)
	err = client.Close()
	require.NoError(t, err)

}

func TestWithCollectionMetadataMapCreateStrict(t *testing.T) {
	t.Run("returns deferred error when option is applied", func(t *testing.T) {
		opt := WithCollectionMetadataMapCreateStrict(map[string]interface{}{
			"tags": []interface{}{"a", 1},
		})

		_, err := NewCreateCollectionOp("test", opt)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error converting metadata map")
		require.Contains(t, err.Error(), `invalid array metadata for key "tags"`)
	})

	t.Run("sets metadata when map is valid", func(t *testing.T) {
		op, err := NewCreateCollectionOp("test", WithCollectionMetadataMapCreateStrict(map[string]interface{}{
			"title": "hello",
			"tags":  []interface{}{"a", "b"},
		}))
		require.NoError(t, err)

		title, ok := op.Metadata.GetString("title")
		require.True(t, ok)
		require.Equal(t, "hello", title)

		tags, ok := op.Metadata.GetStringArray("tags")
		require.True(t, ok)
		require.Equal(t, []string{"a", "b"}, tags)
	})
}

func TestCreateCollectionWithCollectionMetadataMapCreateStrictDeferredError(t *testing.T) {
	var reqCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewHTTPClient(WithBaseURL(server.URL), WithLogger(testLogger()))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, client.Close())
	}()

	_, err = client.CreateCollection(
		context.Background(),
		"test",
		WithCollectionMetadataMapCreateStrict(map[string]interface{}{
			"tags": []interface{}{"a", 1},
		}),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error preparing collection create request")
	require.Contains(t, err.Error(), "error converting metadata map")
	require.Equal(t, int32(0), reqCount.Load())
}

func TestGetOrCreateCollectionWithCollectionMetadataMapCreateStrictDeferredError(t *testing.T) {
	var reqCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewHTTPClient(WithBaseURL(server.URL), WithLogger(testLogger()))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, client.Close())
	}()

	_, err = client.GetOrCreateCollection(
		context.Background(),
		"test",
		WithCollectionMetadataMapCreateStrict(map[string]interface{}{
			"tags": []interface{}{"a", 1},
		}),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error preparing collection create request")
	require.Contains(t, err.Error(), "error converting metadata map")
	require.Equal(t, int32(0), reqCount.Load())
}

// runConcurrencyTest spins up concurrent writers and readers that verify
// TenantAndDatabase consistency. Each writer calls its function `iterations`
// times; 4 reader goroutines validate that snapshots are always consistent.
func runConcurrencyTest(t *testing.T, snapshot func() (Tenant, Database), iterations int, writers ...func(i int) error) {
	t.Helper()
	errCh := make(chan error, len(writers)+4)
	var wg sync.WaitGroup
	start := make(chan struct{})

	for _, fn := range writers {
		fn := fn
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < iterations; i++ {
				if err := fn(i); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < iterations; i++ {
				tenant, database := snapshot()
				if tenant == nil || database == nil || database.Tenant() == nil {
					errCh <- fmt.Errorf("nil value at iteration %d", i)
					return
				}
				if database.Tenant().Name() != tenant.Name() {
					errCh <- fmt.Errorf("inconsistent pair at iteration %d: tenant=%q db.tenant=%q", i, tenant.Name(), database.Tenant().Name())
					return
				}
			}
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}
}

func TestBaseAPIClientConcurrentTenantDatabaseAccess(t *testing.T) {
	stateClient, err := NewHTTPClient(WithBaseURL("http://localhost:8080"))
	require.NoError(t, err)
	defer func() { require.NoError(t, stateClient.Close()) }()
	client := stateClient.(*APIClientV2)

	pairWriter := func(prefix string) func(int) error {
		return func(i int) error {
			tenant := NewTenant(fmt.Sprintf("%s-tenant-%d", prefix, i%32))
			client.SetTenantAndDatabase(tenant, NewDatabase(fmt.Sprintf("%s-db-%d", prefix, i%32), tenant))
			return nil
		}
	}
	runConcurrencyTest(t, client.TenantAndDatabase, 1000,
		pairWriter("a"), pairWriter("b"), pairWriter("c"))

	require.NotNil(t, client.CurrentTenant())
	require.NotNil(t, client.CurrentDatabase())
}

func TestAPIClientV2ConcurrentUseTenantUseDatabase(t *testing.T) {
	tenantPath := regexp.MustCompile(`^/api/v2/tenants/([^/]+)$`)
	databasePath := regexp.MustCompile(`^/api/v2/tenants/([^/]+)/databases/([^/]+)$`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && tenantPath.MatchString(r.URL.Path):
			match := tenantPath.FindStringSubmatch(r.URL.Path)
			if len(match) != 2 {
				http.Error(w, "invalid tenant path", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(fmt.Sprintf(`{"name":"%s"}`, match[1])))
		case r.Method == http.MethodGet && databasePath.MatchString(r.URL.Path):
			match := databasePath.FindStringSubmatch(r.URL.Path)
			if len(match) != 3 {
				http.Error(w, "invalid database path", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(fmt.Sprintf(`{"name":"%s","tenant":"%s"}`, match[2], match[1])))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	stateClient, err := NewHTTPClient(WithBaseURL(server.URL))
	require.NoError(t, err)
	defer func() { require.NoError(t, stateClient.Close()) }()
	client := stateClient.(*APIClientV2)

	runConcurrencyTest(t, client.TenantAndDatabase, 300,
		func(i int) error {
			return client.UseTenant(context.Background(), NewTenant(fmt.Sprintf("tenant-%d", i%16)))
		},
		func(i int) error {
			tenant := NewTenant(fmt.Sprintf("tenant-%d", i%16))
			return client.UseDatabase(context.Background(), NewDatabase(fmt.Sprintf("db-%d", i%16), tenant))
		},
	)
}

func TestAPIClientV2ConcurrentUseTenantDatabase(t *testing.T) {
	tenantPath := regexp.MustCompile(`^/api/v2/tenants/([^/]+)$`)
	databasePath := regexp.MustCompile(`^/api/v2/tenants/([^/]+)/databases/([^/]+)$`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && tenantPath.MatchString(r.URL.Path):
			match := tenantPath.FindStringSubmatch(r.URL.Path)
			if len(match) != 2 {
				http.Error(w, "invalid tenant path", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(fmt.Sprintf(`{"name":"%s"}`, match[1])))
		case r.Method == http.MethodGet && databasePath.MatchString(r.URL.Path):
			match := databasePath.FindStringSubmatch(r.URL.Path)
			if len(match) != 3 {
				http.Error(w, "invalid database path", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(fmt.Sprintf(`{"name":"%s","tenant":"%s"}`, match[2], match[1])))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	stateClient, err := NewHTTPClient(WithBaseURL(server.URL))
	require.NoError(t, err)
	defer func() { require.NoError(t, stateClient.Close()) }()
	client := stateClient.(*APIClientV2)

	runConcurrencyTest(t, client.TenantAndDatabase, 300,
		func(i int) error {
			tenant := NewTenant(fmt.Sprintf("tenant-%d", i%16))
			database := NewDatabase(fmt.Sprintf("db-%d", i%16), tenant)
			return client.UseTenantDatabase(context.Background(), tenant, database)
		},
		func(i int) error {
			return client.UseTenantDatabase(context.Background(), NewTenant(fmt.Sprintf("tenant-%d", (i+5)%16)), nil)
		},
	)
}

func TestAPIClientV2UseTenantDatabase_NilDatabaseDefaults(t *testing.T) {
	tenantPath := regexp.MustCompile(`^/api/v2/tenants/([^/]+)$`)
	databasePath := regexp.MustCompile(`^/api/v2/tenants/([^/]+)/databases/([^/]+)$`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && tenantPath.MatchString(r.URL.Path):
			match := tenantPath.FindStringSubmatch(r.URL.Path)
			if len(match) != 2 {
				http.Error(w, "invalid tenant path", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(fmt.Sprintf(`{"name":"%s"}`, match[1])))
		case r.Method == http.MethodGet && databasePath.MatchString(r.URL.Path):
			match := databasePath.FindStringSubmatch(r.URL.Path)
			if len(match) != 3 {
				http.Error(w, "invalid database path", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(fmt.Sprintf(`{"name":"%s","tenant":"%s"}`, match[2], match[1])))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	clientRaw, err := NewHTTPClient(WithBaseURL(server.URL))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, clientRaw.Close())
	}()

	client, ok := clientRaw.(*APIClientV2)
	require.True(t, ok)

	err = client.UseTenantDatabase(context.Background(), NewTenant("tenant-default-db"), nil)
	require.NoError(t, err)

	tenant, database := client.TenantAndDatabase()
	require.NotNil(t, tenant)
	require.Equal(t, "tenant-default-db", tenant.Name())
	require.NotNil(t, database)
	require.Equal(t, DefaultDatabase, database.Name())
	require.NotNil(t, database.Tenant())
	require.Equal(t, tenant.Name(), database.Tenant().Name())
}

func TestAPIClientV2UseTenantDatabase_NilTenantReturnsError(t *testing.T) {
	clientRaw, err := NewHTTPClient(WithBaseURL("http://localhost:8080"))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, clientRaw.Close())
	}()

	client, ok := clientRaw.(*APIClientV2)
	require.True(t, ok)

	err = client.UseTenantDatabase(context.Background(), nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tenant cannot be nil")
}

func TestAPIClientV2UseTenant_NilTenantReturnsError(t *testing.T) {
	clientRaw, err := NewHTTPClient(WithBaseURL("http://localhost:8080"))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, clientRaw.Close())
	}()

	client, ok := clientRaw.(*APIClientV2)
	require.True(t, ok)

	err = client.UseTenant(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tenant cannot be nil")
}

func TestBaseAPIClientZeroValueTenantDatabaseDefaults(t *testing.T) {
	var client BaseAPIClient

	tenant, database := client.TenantAndDatabase()
	require.NotNil(t, tenant)
	require.Equal(t, DefaultTenant, tenant.Name())
	require.NotNil(t, database)
	require.Equal(t, DefaultDatabase, database.Name())
	require.NotNil(t, database.Tenant())
	require.Equal(t, tenant.Name(), database.Tenant().Name())

	client.SetTenantAndDatabase(nil, nil)
	tenant, database = client.TenantAndDatabase()
	require.NotNil(t, tenant)
	require.Equal(t, DefaultTenant, tenant.Name())
	require.NotNil(t, database)
	require.Equal(t, DefaultDatabase, database.Name())
	require.NotNil(t, database.Tenant())
	require.Equal(t, tenant.Name(), database.Tenant().Name())
}

func TestBaseAPIClientNormalizeTenantAndDatabase_FromDatabaseTenant(t *testing.T) {
	var client BaseAPIClient

	tenantFromDB := NewTenant("tenant-from-db")
	client.SetTenantAndDatabase(nil, NewDatabase("db-from-db", tenantFromDB))

	tenant, database := client.TenantAndDatabase()
	require.NotNil(t, tenant)
	require.Equal(t, "tenant-from-db", tenant.Name())
	require.NotNil(t, database)
	require.Equal(t, "db-from-db", database.Name())
	require.NotNil(t, database.Tenant())
	require.Equal(t, "tenant-from-db", database.Tenant().Name())
}

func TestBaseAPIClientNormalizeTenantAndDatabase_RewritesMismatchedDatabaseTenant(t *testing.T) {
	var client BaseAPIClient

	client.SetTenantAndDatabase(
		NewTenant("tenant-target"),
		NewDatabase("db-mismatch", NewTenant("tenant-other")),
	)

	tenant, database := client.TenantAndDatabase()
	require.NotNil(t, tenant)
	require.Equal(t, "tenant-target", tenant.Name())
	require.NotNil(t, database)
	require.Equal(t, "db-mismatch", database.Name())
	require.NotNil(t, database.Tenant())
	require.Equal(t, "tenant-target", database.Tenant().Name())
}

func TestBaseAPIClientNormalizeTenantAndDatabase_EmptyDatabaseNameDefaults(t *testing.T) {
	var client BaseAPIClient

	client.SetTenantAndDatabase(NewTenant("tenant-empty-db"), NewDatabase("", NewTenant("tenant-other")))

	tenant, database := client.TenantAndDatabase()
	require.NotNil(t, tenant)
	require.Equal(t, "tenant-empty-db", tenant.Name())
	require.NotNil(t, database)
	require.Equal(t, DefaultDatabase, database.Name())
	require.NotNil(t, database.Tenant())
	require.Equal(t, "tenant-empty-db", database.Tenant().Name())
}

func TestNewHTTPClientReturnsErrorWhenAuthProviderFailsDuringConstruction(t *testing.T) {
	_, err := NewHTTPClient(WithAuth(failingCredentialsProvider{err: stderrors.New("auth setup failed")}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "error applying auth credentials")
	require.Contains(t, err.Error(), "auth setup failed")
}

func TestBaseAPIClientDefaultHeadersReturnsDefensiveCopy(t *testing.T) {
	clientRaw, err := NewHTTPClient(
		WithBaseURL("http://localhost:8080"),
		WithDefaultHeaders(map[string]string{"X-Test": "1"}),
	)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, clientRaw.Close())
	}()

	client, ok := clientRaw.(*APIClientV2)
	require.True(t, ok)

	headers := client.DefaultHeaders()
	headers["X-Test"] = "modified"
	headers["X-New"] = "2"

	headersAfter := client.DefaultHeaders()
	require.Equal(t, "1", headersAfter["X-Test"])
	_, exists := headersAfter["X-New"]
	require.False(t, exists)
}

func TestCloudClientDefaultHeadersIncludeAuthFromOptionAndEnv(t *testing.T) {
	t.Run("from option", func(t *testing.T) {
		client, err := NewCloudClient(
			WithDatabaseAndTenant("db-option", "tenant-option"),
			WithCloudAPIKey("option-token"),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
		defer func() {
			require.NoError(t, client.Close())
		}()

		headers := client.DefaultHeaders()
		require.Equal(t, "option-token", headers[string(XChromaTokenHeader)])
	})

	t.Run("from env fallback", func(t *testing.T) {
		t.Setenv("CHROMA_API_KEY", "env-token")
		client, err := NewCloudClient(
			WithDatabaseAndTenant("db-env", "tenant-env"),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
		defer func() {
			require.NoError(t, client.Close())
		}()

		headers := client.DefaultHeaders()
		require.Equal(t, "env-token", headers[string(XChromaTokenHeader)])
	})
}

func TestNewHTTPClientUsesDedicatedHTTPClientAndTransport(t *testing.T) {
	clientRaw, err := NewHTTPClient()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, clientRaw.Close())
	}()

	client, ok := clientRaw.(*APIClientV2)
	require.True(t, ok)
	require.NotSame(t, http.DefaultClient, client.HTTPClient())
	require.NotSame(t, http.DefaultTransport, client.HTTPClient().Transport)
}

func TestClientSetup(t *testing.T) {
	t.Run("With default tenant and database", func(t *testing.T) {
		client, err := NewHTTPClient(WithBaseURL("http://localhost:8080"), WithLogger(testLogger()))
		require.NoError(t, err)
		require.NotNil(t, client)
		require.Equal(t, NewDefaultTenant(), client.CurrentTenant())
		require.Equal(t, NewDefaultDatabase(), client.CurrentDatabase())
	})

	t.Run("With env tenant and database", func(t *testing.T) {
		t.Setenv("CHROMA_TENANT", "test_tenant")
		t.Setenv("CHROMA_DATABASE", "test_db")
		client, err := NewHTTPClient(WithBaseURL("http://localhost:8080"), WithLogger(testLogger()))
		require.NoError(t, err)
		require.NotNil(t, client)
		require.Equal(t, NewTenant("test_tenant"), client.CurrentTenant())
		require.Equal(t, NewDatabase("test_db", NewTenant("test_tenant")), client.CurrentDatabase())
	})

	t.Run("With env database only", func(t *testing.T) {
		t.Setenv("CHROMA_TENANT", "")
		t.Setenv("CHROMA_DATABASE", "test_db")
		client, err := NewHTTPClient(WithBaseURL("http://localhost:8080"), WithLogger(testLogger()))
		require.NoError(t, err)
		require.NotNil(t, client)

		tenant := client.CurrentTenant()
		require.NotNil(t, tenant)
		require.Equal(t, DefaultTenant, tenant.Name())

		database := client.CurrentDatabase()
		require.NotNil(t, database)
		require.Equal(t, "test_db", database.Name())
		require.NotNil(t, database.Tenant())
		require.Equal(t, DefaultTenant, database.Tenant().Name())
	})

	t.Run("WithHTTPClient and WithInsecure are mutually exclusive", func(t *testing.T) {
		httpClient := &http.Client{Transport: &http.Transport{}}
		_, err := NewHTTPClient(
			WithBaseURL("http://localhost:8080"),
			WithHTTPClient(httpClient),
			WithInsecure(),
			WithLogger(testLogger()),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be combined")
	})

	t.Run("WithInsecure then WithHTTPClient are mutually exclusive", func(t *testing.T) {
		httpClient := &http.Client{Transport: &http.Transport{}}
		_, err := NewHTTPClient(
			WithBaseURL("http://localhost:8080"),
			WithInsecure(),
			WithHTTPClient(httpClient),
			WithLogger(testLogger()),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be combined")
	})

	t.Run("WithHTTPClient and WithTransport are mutually exclusive", func(t *testing.T) {
		httpClient := &http.Client{Transport: &http.Transport{}}
		_, err := NewHTTPClient(
			WithBaseURL("http://localhost:8080"),
			WithHTTPClient(httpClient),
			WithTransport(&http.Transport{}),
			WithLogger(testLogger()),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be combined")
	})

	t.Run("WithTransport then WithHTTPClient are mutually exclusive", func(t *testing.T) {
		httpClient := &http.Client{Transport: &http.Transport{}}
		_, err := NewHTTPClient(
			WithBaseURL("http://localhost:8080"),
			WithTransport(&http.Transport{}),
			WithHTTPClient(httpClient),
			WithLogger(testLogger()),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be combined")
	})

	t.Run("WithHTTPClient and WithSSLCert are mutually exclusive", func(t *testing.T) {
		httpClient := &http.Client{Transport: &http.Transport{}}
		_, err := NewHTTPClient(
			WithBaseURL("http://localhost:8080"),
			WithHTTPClient(httpClient),
			WithSSLCert("/path/unused-when-conflict-is-detected.pem"),
			WithLogger(testLogger()),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be combined")
	})

	t.Run("WithHTTPClient synchronizes internal transport reference", func(t *testing.T) {
		customTransport := &http.Transport{}
		httpClient := &http.Client{Transport: customTransport}
		clientRaw, err := NewHTTPClient(
			WithBaseURL("http://localhost:8080"),
			WithHTTPClient(httpClient),
			WithLogger(testLogger()),
		)
		require.NoError(t, err)
		apiClient, ok := clientRaw.(*APIClientV2)
		require.True(t, ok)
		require.Same(t, httpClient, apiClient.httpClient)
		require.Same(t, customTransport, apiClient.httpTransport)
	})
}

func TestCreateCollectionWithContentEF(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		cm := CollectionModel{
			ID:       "test-id-001",
			Name:     "test-content-ef",
			Tenant:   "default_tenant",
			Database: "default_database",
		}
		_ = json.NewEncoder(w).Encode(&cm)
	}))
	defer server.Close()

	client, err := NewHTTPClient(WithBaseURL(server.URL))
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	contentEF := &mockCloseableContentEF{}
	col, err := client.CreateCollection(context.Background(), "test-content-ef",
		WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()),
		WithContentEmbeddingFunctionCreate(contentEF),
	)
	require.NoError(t, err)
	impl := col.(*CollectionImpl)
	require.NotNil(t, impl.contentEmbeddingFunction, "contentEmbeddingFunction must be set on CollectionImpl")
}

func TestGetOrCreateCollectionWithContentEF(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		cm := CollectionModel{
			ID:       "test-id-002",
			Name:     "test-getorcreate-ef",
			Tenant:   "default_tenant",
			Database: "default_database",
		}
		_ = json.NewEncoder(w).Encode(&cm)
	}))
	defer server.Close()

	client, err := NewHTTPClient(WithBaseURL(server.URL))
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	contentEF := &mockCloseableContentEF{}
	col, err := client.GetOrCreateCollection(context.Background(), "test-getorcreate-ef",
		WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()),
		WithContentEmbeddingFunctionCreate(contentEF),
	)
	require.NoError(t, err)
	impl := col.(*CollectionImpl)
	require.NotNil(t, impl.contentEmbeddingFunction, "contentEmbeddingFunction must be set via GetOrCreateCollection")
}

func TestWithContentEmbeddingFunctionCreateNil(t *testing.T) {
	_, err := NewCreateCollectionOp("test", WithContentEmbeddingFunctionCreate(nil))
	require.Error(t, err)
	require.Contains(t, err.Error(), "content embedding function cannot be nil")
}

func TestCreateCollectionOpUnmarshalJSON_InitializesDefaultDenseEFFactory(t *testing.T) {
	var op CreateCollectionOp

	err := json.Unmarshal([]byte(`{"name":"test-json-default"}`), &op)
	require.NoError(t, err)
	require.NotNil(t, op.defaultDenseEFFactory)
}

func TestCreateCollectionOpEnsureDefaultDenseEFFactory_ZeroValue(t *testing.T) {
	var op CreateCollectionOp

	require.Nil(t, op.defaultDenseEFFactory)
	op.ensureDefaultDenseEFFactory()
	require.NotNil(t, op.defaultDenseEFFactory)
}

func TestCreateCollectionOpCloseSDKOwnedDefaultDenseEF_NonClosableKeepsTrackedReference(t *testing.T) {
	ef := &mockNonCloseableEF{}
	op := &CreateCollectionOp{
		embeddingFunction:      ef,
		sdkOwnedDefaultDenseEF: ef,
	}

	err := op.closeSDKOwnedDefaultDenseEF("unused")
	require.EqualError(t, err, "sdk-owned default embedding function is not closable")
	require.Same(t, ef, op.sdkOwnedDefaultDenseEF)
}

func TestCreateCollectionOpCloseSDKOwnedDefaultDenseEF_RecoversPanic(t *testing.T) {
	ef := &mockPanickingCloseEF{}
	op := &CreateCollectionOp{
		embeddingFunction:      ef,
		sdkOwnedDefaultDenseEF: ef,
	}

	var err error
	require.NotPanics(t, func() {
		err = op.closeSDKOwnedDefaultDenseEF("error closing default embedding function")
	})
	require.Error(t, err, "panicking Close must surface as an error, not a panic")
	require.Contains(t, err.Error(), "error closing default embedding function")
	require.Contains(t, err.Error(), "panic during EF close")
}

// TestCreateCollectionOpCloseSDKOwnedDefaultDenseEF_ClearsOnAnyCloseAttempt pins
// the contract that prevents the outer-defer double-close identified by the PR
// #504 review (item 2): once Close() has been attempted on the SDK-owned default
// EF -- success, error, or panic -- the tracking field must be nilled out. A
// subsequent invocation from an outer defer must be a no-op so the underlying
// resource is not closed a second time.
func TestCreateCollectionOpCloseSDKOwnedDefaultDenseEF_ClearsOnAnyCloseAttempt(t *testing.T) {
	t.Run("close returning error clears tracking", func(t *testing.T) {
		ef := &mockFailingCloseEF{closeErr: stderrors.New("boom")}
		op := &CreateCollectionOp{
			embeddingFunction:      ef,
			sdkOwnedDefaultDenseEF: ef,
		}

		err := op.closeSDKOwnedDefaultDenseEF("first-call")
		require.Error(t, err)
		require.Contains(t, err.Error(), "boom")
		require.Nil(t, op.sdkOwnedDefaultDenseEF,
			"sdkOwnedDefaultDenseEF must be cleared after any Close attempt to prevent outer defer double-close")
		require.Equal(t, int32(1), ef.closeCount.Load())

		// Simulate the outer defer invoked by embeddedLocalClient.CreateCollection.
		err2 := op.closeSDKOwnedDefaultDenseEF("second-call")
		require.NoError(t, err2, "outer defer re-entry must be a no-op")
		require.Equal(t, int32(1), ef.closeCount.Load(),
			"Close must not run a second time via outer defer")
	})

	t.Run("close panic clears tracking", func(t *testing.T) {
		ef := &mockPanickingCloseEF{}
		op := &CreateCollectionOp{
			embeddingFunction:      ef,
			sdkOwnedDefaultDenseEF: ef,
		}

		err := op.closeSDKOwnedDefaultDenseEF("first-call")
		require.Error(t, err)
		require.Contains(t, err.Error(), "panic during EF close")
		require.Nil(t, op.sdkOwnedDefaultDenseEF,
			"panicking Close must also clear tracking so outer defer does not re-enter")

		err2 := op.closeSDKOwnedDefaultDenseEF("second-call")
		require.NoError(t, err2, "outer defer re-entry must be a no-op")
	})
}

func TestCreateCollectionOpCloseSDKOwnedDefaultDenseEF_ClearsAfterSuccessfulClose(t *testing.T) {
	ef := &mockCloseableEF{}
	op := &CreateCollectionOp{
		embeddingFunction:      ef,
		sdkOwnedDefaultDenseEF: ef,
	}

	err := op.closeSDKOwnedDefaultDenseEF("unused")
	require.NoError(t, err)
	require.Nil(t, op.sdkOwnedDefaultDenseEF,
		"sdkOwnedDefaultDenseEF must be cleared after a successful close")
	require.Equal(t, int32(1), ef.closeCount.Load())
}

func TestWithDefaultDenseEFFactoryCreate_RejectsNil(t *testing.T) {
	_, err := NewCreateCollectionOp("test-nil-default-factory", withDefaultDenseEFFactoryCreate(nil))
	require.EqualError(t, err, "default dense EF factory cannot be nil")
}

func TestPrepareAndValidateCollectionRequest_ContentEFConfigPersistence(t *testing.T) {
	t.Run("dual-interface contentEF persists config", func(t *testing.T) {
		dualEF := &mockDualEF{}
		op, err := NewCreateCollectionOp("test-dual",
			WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()),
			WithContentEmbeddingFunctionCreate(dualEF),
		)
		require.NoError(t, err)
		err = op.PrepareAndValidateCollectionRequest()
		require.NoError(t, err)
		require.NotNil(t, op.Configuration, "Configuration must be set")
		efInfo, ok := op.Configuration.GetEmbeddingFunctionInfo()
		require.True(t, ok, "EF info must be present in config")
		require.NotNil(t, efInfo, "EF info must not be nil")
	})

	t.Run("content-only contentEF leaves denseEF config intact", func(t *testing.T) {
		contentOnlyEF := &mockCloseableContentEF{}
		op, err := NewCreateCollectionOp("test-content-only",
			WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()),
			WithContentEmbeddingFunctionCreate(contentOnlyEF),
		)
		require.NoError(t, err)
		err = op.PrepareAndValidateCollectionRequest()
		require.NoError(t, err)
		require.NotNil(t, op.Configuration, "Configuration must be set from denseEF")
		efInfo, ok := op.Configuration.GetEmbeddingFunctionInfo()
		require.True(t, ok, "EF info must be present from denseEF")
		require.NotNil(t, efInfo)
	})

	t.Run("dual contentEF promoted to runtime denseEF when no explicit denseEF", func(t *testing.T) {
		dualEF := &mockDualEF{}
		op, err := NewCreateCollectionOp("test-promote",
			WithContentEmbeddingFunctionCreate(dualEF),
		)
		require.NoError(t, err)
		err = op.PrepareAndValidateCollectionRequest()
		require.NoError(t, err)
		require.Equal(t, dualEF.Name(), op.embeddingFunction.(embeddings.EmbeddingFunction).Name(),
			"runtime denseEF must be the dual contentEF, not default ORT")
	})

	t.Run("dual contentEF does not replace explicit denseEF at runtime", func(t *testing.T) {
		explicitEF := embeddings.NewConsistentHashEmbeddingFunction()
		dualEF := &mockDualEF{}
		op, err := NewCreateCollectionOp("test-no-promote",
			WithEmbeddingFunctionCreate(explicitEF),
			WithContentEmbeddingFunctionCreate(dualEF),
		)
		require.NoError(t, err)
		err = op.PrepareAndValidateCollectionRequest()
		require.NoError(t, err)
		require.Equal(t, explicitEF.Name(), op.embeddingFunction.(embeddings.EmbeddingFunction).Name(),
			"runtime denseEF must remain the user-provided EF")
	})

	t.Run("non-closable sdk-owned default EF returns error during promotion", func(t *testing.T) {
		dualEF := &mockDualEF{}
		op, err := NewCreateCollectionOp("test-promote-non-closable",
			WithContentEmbeddingFunctionCreate(dualEF),
			withDefaultDenseEFFactoryCreate(func() (embeddings.EmbeddingFunction, func() error, error) {
				return &mockNonCloseableEF{}, func() error { return nil }, nil
			}),
		)
		require.NoError(t, err)
		err = op.PrepareAndValidateCollectionRequest()
		require.EqualError(t, err, "sdk-owned default embedding function is not closable")
	})

	t.Run("default dense factory error surfaces", func(t *testing.T) {
		op, err := NewCreateCollectionOp("test-factory-error",
			withDefaultDenseEFFactoryCreate(func() (embeddings.EmbeddingFunction, func() error, error) {
				return nil, nil, stderrors.New("factory boom")
			}),
		)
		require.NoError(t, err)

		err = op.PrepareAndValidateCollectionRequest()
		require.EqualError(t, err, "error creating default embedding function: factory boom")
	})

	t.Run("dual contentEF promotion closes sdk-owned default EF once", func(t *testing.T) {
		dualEF := &mockDualEF{}
		temporaryDefaultEF := &mockCloseableEF{}
		op, err := NewCreateCollectionOp("test-promote-close-count",
			WithContentEmbeddingFunctionCreate(dualEF),
			withDefaultDenseEFFactoryCreate(func() (embeddings.EmbeddingFunction, func() error, error) {
				return temporaryDefaultEF, func() error { return nil }, nil
			}),
		)
		require.NoError(t, err)

		err = op.PrepareAndValidateCollectionRequest()
		require.NoError(t, err)
		require.Same(t, dualEF, op.embeddingFunction)
		require.Nil(t, op.sdkOwnedDefaultDenseEF)
		require.Equal(t, int32(1), temporaryDefaultEF.closeCount.Load())
	})

	t.Run("wrapped content-only EF does not persist empty config", func(t *testing.T) {
		contentOnlyEF := &mockCloseableContentEF{}
		wrapped := wrapContentEFCloseOnce(contentOnlyEF)
		// closeOnceContentEF always satisfies EmbeddingFunction; if wrapping happened
		// before PrepareAndValidate, the type assertion would be a false positive.
		_, isDense := wrapped.(embeddings.EmbeddingFunction)
		require.True(t, isDense, "wrapped content-only EF must satisfy EmbeddingFunction interface")
		// Verify the wrapper returns empty metadata for the non-dual inner EF.
		denseView := wrapped.(embeddings.EmbeddingFunction)
		require.Empty(t, denseView.Name(), "wrapper Name() must be empty for content-only inner EF")
		require.Empty(t, denseView.GetConfig(), "wrapper GetConfig() must be empty for content-only inner EF")
	})
}

// TestAPIClientV2_CreateCollection_ClosesTemporaryDefaultEFOnError pins the PR
// #504 review item 1 fix: when the HTTP CreateCollection path allocates an
// SDK-owned temporary default EF via PrepareAndValidateCollectionRequest and
// then fails during SendRequest or response decoding, the temporary EF must be
// closed rather than orphaned.
func TestAPIClientV2_CreateCollection_ClosesTemporaryDefaultEFOnError(t *testing.T) {
	t.Run("server error closes temporary default EF", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"boom"}`))
		}))
		defer server.Close()

		client, err := NewHTTPClient(WithBaseURL(server.URL), WithLogger(testLogger()))
		require.NoError(t, err)
		defer func() { _ = client.Close() }()

		temporaryDefaultEF := &mockCloseableEF{}
		_, err = client.CreateCollection(
			context.Background(),
			"http-create-fails",
			withDefaultDenseEFFactoryCreate(func() (embeddings.EmbeddingFunction, func() error, error) {
				return temporaryDefaultEF, func() error { return nil }, nil
			}),
		)
		require.Error(t, err, "CreateCollection must surface the server error")
		require.Equal(t, int32(1), temporaryDefaultEF.closeCount.Load(),
			"temporary default EF must be closed when HTTP CreateCollection fails")
	})

	t.Run("response decode error closes temporary default EF", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`not-json`))
		}))
		defer server.Close()

		client, err := NewHTTPClient(WithBaseURL(server.URL), WithLogger(testLogger()))
		require.NoError(t, err)
		defer func() { _ = client.Close() }()

		temporaryDefaultEF := &mockCloseableEF{}
		_, err = client.CreateCollection(
			context.Background(),
			"http-create-decode-fails",
			withDefaultDenseEFFactoryCreate(func() (embeddings.EmbeddingFunction, func() error, error) {
				return temporaryDefaultEF, func() error { return nil }, nil
			}),
		)
		require.Error(t, err, "CreateCollection must surface the decode error")
		require.Contains(t, err.Error(), "error decoding response")
		require.Equal(t, int32(1), temporaryDefaultEF.closeCount.Load(),
			"temporary default EF must be closed when response decode fails")
	})
}

func TestPrepareAndValidateCollectionRequest_ContentEFSchemaPath(t *testing.T) {
	t.Run("dual-interface contentEF overrides schema EF", func(t *testing.T) {
		schema, err := NewSchemaWithDefaults()
		require.NoError(t, err)
		dualEF := &mockDualEF{}
		op, err := NewCreateCollectionOp("test-schema-dual",
			WithSchemaCreate(schema),
			WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()),
			WithContentEmbeddingFunctionCreate(dualEF),
		)
		require.NoError(t, err)
		err = op.PrepareAndValidateCollectionRequest()
		require.NoError(t, err)
		// contentEF (dual-interface) should have overridden the denseEF in schema
		got := op.Schema.GetEmbeddingFunction()
		require.NotNil(t, got, "schema must have EF set")
		require.Equal(t, dualEF.Name(), got.Name(), "schema EF must be the dual contentEF")
	})

	t.Run("content-only contentEF leaves schema denseEF intact", func(t *testing.T) {
		schema, err := NewSchemaWithDefaults()
		require.NoError(t, err)
		denseEF := embeddings.NewConsistentHashEmbeddingFunction()
		contentOnlyEF := &mockCloseableContentEF{}
		op, err := NewCreateCollectionOp("test-schema-content-only",
			WithSchemaCreate(schema),
			WithEmbeddingFunctionCreate(denseEF),
			WithContentEmbeddingFunctionCreate(contentOnlyEF),
		)
		require.NoError(t, err)
		err = op.PrepareAndValidateCollectionRequest()
		require.NoError(t, err)
		// content-only EF cannot override schema; original denseEF stays
		got := op.Schema.GetEmbeddingFunction()
		require.NotNil(t, got, "schema must have EF set from denseEF")
		require.Equal(t, denseEF.Name(), got.Name(), "schema EF must be the original denseEF")
	})

	t.Run("dual contentEF promoted to runtime denseEF with schema and no explicit denseEF", func(t *testing.T) {
		schema, err := NewSchemaWithDefaults()
		require.NoError(t, err)
		dualEF := &mockDualEF{}
		op, err := NewCreateCollectionOp("test-schema-promote",
			WithSchemaCreate(schema),
			WithContentEmbeddingFunctionCreate(dualEF),
		)
		require.NoError(t, err)
		err = op.PrepareAndValidateCollectionRequest()
		require.NoError(t, err)
		require.Equal(t, dualEF.Name(), op.embeddingFunction.(embeddings.EmbeddingFunction).Name(),
			"runtime denseEF must be the dual contentEF, not default ORT")
		got := op.Schema.GetEmbeddingFunction()
		require.NotNil(t, got, "schema must have EF set")
		require.Equal(t, dualEF.Name(), got.Name(), "schema EF must be the dual contentEF")
	})
}

func TestCreateCollectionWithContentEF_CloseLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		cm := CollectionModel{
			ID:       "test-id-close",
			Name:     "test-close-ef",
			Tenant:   "default_tenant",
			Database: "default_database",
		}
		_ = json.NewEncoder(w).Encode(&cm)
	}))
	defer server.Close()

	client, err := NewHTTPClient(WithBaseURL(server.URL))
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	contentEF := &mockCloseableContentEF{}
	col, err := client.CreateCollection(context.Background(), "test-close-ef",
		WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()),
		WithContentEmbeddingFunctionCreate(contentEF),
	)
	require.NoError(t, err)
	impl := col.(*CollectionImpl)

	err = impl.Close()
	require.NoError(t, err)
	require.Equal(t, int32(1), contentEF.closeCount.Load(), "contentEF must be closed exactly once")

	err = impl.Close()
	require.NoError(t, err)
	require.Equal(t, int32(1), contentEF.closeCount.Load(), "contentEF must not be closed again")
}
