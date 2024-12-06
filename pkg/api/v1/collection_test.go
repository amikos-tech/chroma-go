package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/amikos-tech/chroma-go/pkg/api"
	chhttp "github.com/amikos-tech/chroma-go/pkg/commons/http"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestCollectionAdd(t *testing.T) {
	var rx = regexp.MustCompile(`^/api/v1/collections/[a-zA-Z0-9\-]+/add$`)
	var tests = []struct {
		name                        string
		matchURL                    func(r *http.Request) bool
		validateRequestWithResponse func(w http.ResponseWriter, r *http.Request)
		sendRequest                 func(client api.Client)
		limits                      string
	}{
		{
			name: "with IDs and docs",
			matchURL: func(r *http.Request) bool {
				return r.Method == http.MethodPost && rx.Match([]byte(r.URL.Path))
			},
			validateRequestWithResponse: func(w http.ResponseWriter, r *http.Request) {
				respBody := chhttp.ReadRespBody(r.Body)
				respMap := make(map[string]any)
				err := json.Unmarshal([]byte(respBody), &respMap)
				require.NoError(t, err)
				w.WriteHeader(http.StatusOK)
				_, err = w.Write([]byte(`true`))
				require.NoError(t, err)
			},
			sendRequest: func(client api.Client) {
				collection := &Collection{
					CollectionBase: api.CollectionBase{
						Name:         "test",
						CollectionID: "8ecf0f7e-e806-47f8-96a1-4732ef42359e",
						Tenant:       api.NewDefaultTenant(),
						Database:     api.NewDefaultDatabase(),
						Metadata:     api.NewMetadata(),
					},
					client: client.(*APIClientV1),
				}

				require.NotNil(t, collection)
				err := collection.Add(context.Background(), api.WithIDs("1", "2", "3"), api.WithTexts("doc1", "doc2", "doc3"))
				require.NoError(t, err)
			},
			limits: `{"max_batch_size":100}`,
		},
		{
			name: "exceeding max batch size",
			matchURL: func(r *http.Request) bool {
				return r.Method == http.MethodPost && rx.Match([]byte(r.URL.Path))
			},
			validateRequestWithResponse: func(w http.ResponseWriter, r *http.Request) {
				respBody := chhttp.ReadRespBody(r.Body)
				respMap := make(map[string]any)
				err := json.Unmarshal([]byte(respBody), &respMap)
				require.NoError(t, err)
				w.WriteHeader(http.StatusOK)
				_, err = w.Write([]byte(`true`))
				require.NoError(t, err)
			},
			sendRequest: func(client api.Client) {
				collection := &Collection{
					CollectionBase: api.CollectionBase{
						Name:         "test",
						CollectionID: "8ecf0f7e-e806-47f8-96a1-4732ef42359e",
						Tenant:       api.NewDefaultTenant(),
						Database:     api.NewDefaultDatabase(),
						Metadata:     api.NewMetadata(),
					},
					client: client.(*APIClientV1),
				}

				require.NotNil(t, collection)
				err := collection.Add(context.Background(), api.WithIDs("1", "2", "3"), api.WithTexts("doc1", "doc2", "doc3"))
				require.Error(t, err)
				fmt.Println(err)
				require.Contains(t, err.Error(), "limit exceeded")
			},
			limits: `{"max_batch_size":2}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Logf("Request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/pre-flight-checks":
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(tt.limits))
					require.NoError(t, err)
				case tt.matchURL(r):
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
