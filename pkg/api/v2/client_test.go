package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/api"
	chhttp "github.com/amikos-tech/chroma-go/pkg/commons/http"
)

var sampleCollectionListJSON = `[{
    "id": "8ecf0f7e-e806-47f8-96a1-4732ef42359e",
    "name": "testcoll",
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
    "metadata": {
      "t": 1
    },
    "dimension": null,
    "tenant": "default_tenant",
    "database": "default_database",
    "version": 0,
    "log_position": 0
  }]`

func MetadataModel() gopter.Gen {
	return gen.SliceOf(
		gen.Struct(reflect.TypeOf(struct {
			Key   string
			Value interface{}
		}{}), map[string]gopter.Gen{
			"Key":   gen.Identifier(),
			"Value": gen.OneGenOf(gen.Int64(), gen.Float64(), gen.AlphaString(), gen.Bool()),
		}),
	).Map(func(entries *gopter.GenResult) api.CollectionMetadata {
		result := make(map[string]interface{})
		for _, entry := range entries.Result.([]struct {
			Key   string
			Value interface{}
		}) {
			result[entry.Key] = entry.Value
		}
		return api.NewMetadataFromMap(result)
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
		return gopter.NewGenResult(api.DefaultTenant, gopter.NoShrinker)
	})
}

func DatabaseStrategy() gopter.Gen {
	return gen.OneGenOf(func(params *gopter.GenParameters) *gopter.GenResult {
		id := uuid.New() // Generates a new random UUID
		return gopter.NewGenResult(id.String(), gopter.NoShrinker)
	}, func(params *gopter.GenParameters) *gopter.GenResult {
		return gopter.NewGenResult(api.DefaultDatabase, gopter.NoShrinker)
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
				require.JSONEq(t, `{"name":"`+name+`"}`, respBody)
				var op api.CreateCollectionOp
				err := json.Unmarshal([]byte(respBody), &op)
				require.NoError(t, err)
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

			client, err := NewClient(api.WithBaseURL(server.URL), api.WithDatabaseAndTenant(col.Database, col.Tenant))

			require.NoError(t, err)

			// Call API with random data
			c, err := client.CreateCollection(context.Background(), name)
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
				metadataValue1, ok11 := val1.(api.MetadataValue)
				require.True(t, ok11)
				val2, ok2 := c.Metadata().GetRaw(k)
				require.True(t, ok2)
				metadataValue2, ok22 := val2.(api.MetadataValue)
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
		case r.URL.Path == "/api/v2/tenants/default_tenant/databases/default_database/count_collections" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`100`))
			require.NoError(t, err)
		case r.URL.Path == "/api/v2/tenants/default_tenant/databases/default_database/collections" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(sampleCollectionListJSON))
			require.NoError(t, err)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client, err := NewClient(api.WithBaseURL(server.URL))
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
		require.Equal(t, "testcoll", c.Name())
		require.Equal(t, api.NewDefaultTenant(), c.Tenant())
		require.Equal(t, api.NewDefaultDatabase(), c.Database())
		require.NotNil(t, c.Metadata())
		vi, ok := c.Metadata().GetInt("t")
		require.True(t, ok)
		require.Equal(t, 1, vi)
	})

	t.Run("CreateCollection", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("Request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
			respBody := chhttp.ReadRespBody(r.Body)
			t.Logf("Body: %s", respBody)

			switch {
			case r.URL.Path == "/api/v2/tenants/default_tenant/databases/default_database/collections" && r.Method == http.MethodPost:
				w.WriteHeader(http.StatusOK)
				require.JSONEq(t, `{"name":"test"}`, respBody)
				values, err := url.ParseQuery(r.URL.RawQuery)
				require.NoError(t, err)
				var op api.CreateCollectionOp
				err = json.Unmarshal([]byte(respBody), &op)
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
		innerClient, err := NewClient(api.WithBaseURL(server.URL))
		require.NoError(t, err)
		c, err := innerClient.CreateCollection(context.Background(), "test")
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
				require.JSONEq(t, `{"get_or_create":true, "name":"test"}`, respBody)
				values, err := url.ParseQuery(r.URL.RawQuery)
				require.NoError(t, err)
				var op api.CreateCollectionOp
				err = json.Unmarshal([]byte(respBody), &op)
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
		innerClient, err := NewClient(api.WithBaseURL(server.URL))
		require.NoError(t, err)
		c, err := innerClient.GetOrCreateCollection(context.Background(), "test")
		require.NoError(t, err)
		require.NotNil(t, c)
	})

	t.Run("GetCollection", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("Request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)

			switch {
			case r.URL.Path == "/api/v2/tenants/default_tenant/databases/default_database/collections/test" && r.Method == http.MethodGet:
				w.WriteHeader(http.StatusOK)
				values, err := url.ParseQuery(r.URL.RawQuery)
				require.NoError(t, err)
				cm := CollectionModel{
					ID:       "8ecf0f7e-e806-47f8-96a1-4732ef42359e",
					Name:     "test",
					Tenant:   values.Get("tenant"),
					Database: values.Get("database"),
					Metadata: api.NewMetadataFromMap(map[string]any{"t": 1}),
				}
				err = json.NewEncoder(w).Encode(&cm)
				require.NoError(t, err)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()
		innerClient, err := NewClient(api.WithBaseURL(server.URL))
		require.NoError(t, err)
		c, err := innerClient.GetCollection(context.Background(), "test")
		require.NoError(t, err)
		require.NotNil(t, c)
		require.Equal(t, "8ecf0f7e-e806-47f8-96a1-4732ef42359e", c.ID())
		require.Equal(t, "test", c.Name())
		require.Equal(t, api.NewDefaultTenant(), c.Tenant())
		require.Equal(t, api.NewDefaultDatabase(), c.Database())
		require.NotNil(t, c.Metadata())
		vi, ok := c.Metadata().GetInt("t")
		require.True(t, ok)
		require.Equal(t, 1, vi)
	})
}

func TestCreateCollection(t *testing.T) {
	var tests = []struct {
		name                        string
		validateRequestWithResponse func(w http.ResponseWriter, r *http.Request)
		sendRequest                 func(client api.Client)
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
			sendRequest: func(client api.Client) {
				collection, err := client.CreateCollection(context.Background(), "test")
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
				var op api.CreateCollectionOp
				err := json.Unmarshal([]byte(respBody), &op)
				require.NoError(t, err)
				values, err := url.ParseQuery(r.URL.RawQuery)
				require.NoError(t, err)
				v, ok := op.Metadata.GetInt("int")
				require.True(t, ok)
				require.Equal(t, 1, v)
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
					Tenant:   values.Get("tenant"),
					Database: values.Get("database"),
					Metadata: op.Metadata,
				}
				err = json.NewEncoder(w).Encode(&cm)
				require.NoError(t, err)
			},
			sendRequest: func(client api.Client) {
				collection, err := client.CreateCollection(context.Background(), "test", api.WithCollectionMetadataCreate(
					api.NewMetadataFromMap(map[string]any{"int": 1, "float": 1.1, "string": "test", "bool": true})),
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
				require.Equal(t, 1, vi)
				require.Equal(t, api.NewDefaultTenant(), collection.Tenant())
				require.Equal(t, api.NewDefaultDatabase(), collection.Database())
			},
		},
		{
			name: "with HNSW params",
			validateRequestWithResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				respBody := chhttp.ReadRespBody(r.Body)
				var op api.CreateCollectionOp
				err := json.Unmarshal([]byte(respBody), &op)
				require.NoError(t, err)
				var vi int64
				var vs string
				var vf float64
				var ok bool
				vs, ok = op.Metadata.GetString(api.HNSWSpace)
				require.True(t, ok)
				require.Equal(t, string(api.L2), vs)
				vi, ok = op.Metadata.GetInt(api.HNSWNumThreads)
				require.True(t, ok)
				require.Equal(t, 14, vi)
				vf, ok = op.Metadata.GetFloat(api.HNSWResizeFactor)
				require.True(t, ok)
				require.Equal(t, 1.2, vf)
				vi, ok = op.Metadata.GetInt(api.HNSWBatchSize)
				require.True(t, ok)
				require.Equal(t, 2000, vi)
				vi, ok = op.Metadata.GetInt(api.HNSWSyncThreshold)
				require.True(t, ok)
				require.Equal(t, 10000, vi)
				vi, ok = op.Metadata.GetInt(api.HNSWConstructionEF)
				require.True(t, ok)
				require.Equal(t, 100, vi)
				vi, ok = op.Metadata.GetInt(api.HNSWSearchEF)
				require.True(t, ok)
				require.Equal(t, 999, vi)
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
			},
			sendRequest: func(client api.Client) {
				collection, err := client.CreateCollection(
					context.Background(),
					"test",
					api.WithHNSWSpaceCreate(api.L2),
					api.WithHNSWMCreate(100),
					api.WithHNSWNumThreadsCreate(14),
					api.WithHNSWResizeFactorCreate(1.2),
					api.WithHNSWBatchSizeCreate(2000),
					api.WithHNSWSyncThresholdCreate(10000),
					api.WithHNSWConstructionEfCreate(100),
					api.WithHNSWSearchEfCreate(999),
				)
				require.NoError(t, err)
				require.NotNil(t, collection)
				require.Equal(t, "8ecf0f7e-e806-47f8-96a1-4732ef42359e", collection.ID())
				require.Equal(t, "test", collection.Name())
				hnswSpace, ok := collection.Metadata().GetString(api.HNSWSpace)
				require.True(t, ok)
				require.Equal(t, string(api.L2), hnswSpace)
				hnswNumThreads, ok := collection.Metadata().GetInt(api.HNSWNumThreads)
				require.True(t, ok)
				require.Equal(t, 14, hnswNumThreads)
				hnswResizeFactor, ok := collection.Metadata().GetFloat(api.HNSWResizeFactor)
				require.True(t, ok)
				require.Equal(t, 1.2, hnswResizeFactor)
				hnswBatchSize, ok := collection.Metadata().GetInt(api.HNSWBatchSize)
				require.True(t, ok)
				require.Equal(t, 2000, hnswBatchSize)
				hnswSyncThreshold, ok := collection.Metadata().GetInt(api.HNSWSyncThreshold)
				require.True(t, ok)
				require.Equal(t, 10000, hnswSyncThreshold)
				hnswConstructionEf, ok := collection.Metadata().GetInt(api.HNSWConstructionEF)
				require.True(t, ok)
				require.Equal(t, 100, hnswConstructionEf)
				hnswSearchEf, ok := collection.Metadata().GetInt(api.HNSWSearchEF)
				require.True(t, ok)
				require.Equal(t, 999, hnswSearchEf)
			},
		},

		{
			name: "with tenant and database",
			validateRequestWithResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				respBody := chhttp.ReadRespBody(r.Body)
				var op api.CreateCollectionOp
				err := json.Unmarshal([]byte(respBody), &op)
				require.NoError(t, err)
				require.Contains(t, "mytenant", r.URL.RawQuery)
				require.Contains(t, "mydb", r.URL.RawQuery)
				_, err = w.Write([]byte(`{"id":"8ecf0f7e-e806-47f8-96a1-4732ef42359e","name":"test"}`))
				require.NoError(t, err)
			},
			sendRequest: func(client api.Client) {
				collection, err := client.CreateCollection(
					context.Background(),
					"test",
					api.WithTenantCreate("mytenant"),
					api.WithDatabaseCreate("mydb"),
				)
				require.NoError(t, err)
				require.NotNil(t, collection)
				require.Equal(t, "8ecf0f7e-e806-47f8-96a1-4732ef42359e", collection.ID())
				require.Equal(t, "test", collection.Name())
				require.Equal(t, api.NewTenant("mytenant"), collection.Tenant())
				require.Equal(t, api.NewDatabase("mydb", api.NewTenant("mytenant")), collection.Database())
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
			client, err := NewClient(api.WithBaseURL(server.URL), api.WithDebug())
			require.NoError(t, err)
			tt.sendRequest(client)
		})
	}
}
