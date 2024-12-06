package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/amikos-tech/chroma-go/pkg/api"
	chhttp "github.com/amikos-tech/chroma-go/pkg/commons/http"
)

type APIClientV1 struct {
	api.BaseAPIClient
	preflightLimits    map[string]interface{}
	preflightCompleted bool
}

func NewClient(opts ...api.ClientOption) (api.Client, error) {
	bc, err := api.NewBaseAPIClient(opts...)
	if err != nil {
		return nil, err
	}
	if bc.BaseURL() == "" {
		bc.SetBaseURL("http://localhost:8080/api/v1")
	} else if !strings.HasSuffix(bc.BaseURL(), "/api/v1") {
		newBasePath, err := url.JoinPath(bc.BaseURL(), "/api/v1")
		if err != nil {
			return nil, err
		}
		bc.SetBaseURL(newBasePath)
	}
	c := &APIClientV1{
		BaseAPIClient:      *bc,
		preflightLimits:    map[string]interface{}{},
		preflightCompleted: false,
	}
	return c, nil
}

func (client *APIClientV1) PreFlight(ctx context.Context) error {
	if client.preflightCompleted {
		return nil
	}
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "pre-flight-checks")
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return err
	}
	var preflightLimits map[string]interface{}
	if json.NewDecoder(resp.Body).Decode(&preflightLimits) != nil {
		return errors.New("error decoding preflight response")
	}
	if mbs, ok := preflightLimits["max_batch_size"]; ok {
		if maxBatchSize, ok := mbs.(float64); ok {
			client.preflightLimits[fmt.Sprintf("%s#%s", string(api.ResourceCollection), string(api.OperationCreate))] = int(maxBatchSize)
			client.preflightLimits[fmt.Sprintf("%s#%s", string(api.ResourceCollection), string(api.OperationGet))] = int(maxBatchSize)
			client.preflightLimits[fmt.Sprintf("%s#%s", string(api.ResourceCollection), string(api.OperationQuery))] = int(maxBatchSize)
			client.preflightLimits[fmt.Sprintf("%s#%s", string(api.ResourceCollection), string(api.OperationUpdate))] = int(maxBatchSize)
			client.preflightLimits[fmt.Sprintf("%s#%s", string(api.ResourceCollection), string(api.OperationDelete))] = int(maxBatchSize)
		}
	}
	client.preflightCompleted = true
	return nil
}

func (client *APIClientV1) GetVersion(ctx context.Context) (string, error) {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "version")
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return "", err
	}
	respBody := chhttp.ReadRespBody(resp.Body)
	version := strings.ReplaceAll(respBody, `"`, "")
	return version, nil
}

func (client *APIClientV1) Heartbeat(ctx context.Context) error {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "heartbeat")
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return err
	}
	respBody := chhttp.ReadRespBody(resp.Body)
	if strings.Contains(respBody, "nanosecond heartbeat") {
		return nil
	} else {
		return fmt.Errorf("heartbeat failed")
	}
}

func (client *APIClientV1) GetTenant(ctx context.Context, tenant string) (api.Tenant, error) {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", tenant)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return nil, err
	}
	respBody := chhttp.ReadRespBody(resp.Body)
	return api.NewTenant(respBody), nil
}

func (client *APIClientV1) CreateTenant(ctx context.Context, tenant api.Tenant) (api.Tenant, error) {
	reqJSON, err := json.Marshal(tenant)
	if err != nil {
		return nil, err
	}
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants")
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, err
	}
	resp, err := client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return nil, err
	}
	respBody := chhttp.ReadRespBody(resp.Body)
	return api.NewTenant(respBody), nil
}

func (client *APIClientV1) GetDatabase(ctx context.Context, tenant, database string) (api.Database, error) {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "databases", database)
	if err != nil {
		return nil, err
	}
	queryParams := url.Values{}
	queryParams.Set("tenant", tenant)
	reqURL = fmt.Sprintf("%s?%s", reqURL, queryParams.Encode())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return nil, err
	}
	respBody := chhttp.ReadRespBody(resp.Body)
	return api.NewDatabase(respBody, api.NewTenant(tenant)), nil
}

func (client *APIClientV1) CreateDatabase(ctx context.Context, tenant, database string) (api.Database, error) {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "databases", database)
	if err != nil {
		return nil, err
	}
	queryParams := url.Values{}
	queryParams.Set("tenant", tenant)
	reqURL = fmt.Sprintf("%s?%s", reqURL, queryParams.Encode())
	db := api.NewDatabase(database, api.NewTenant(tenant))
	reqJSON, err := json.Marshal(db)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, err
	}
	_, err = client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return nil, err
	}
	return api.NewDatabase(database, api.NewTenant(tenant)), nil
}

func (client *APIClientV1) Reset(ctx context.Context) error {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "reset")
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, nil)
	if err != nil {
		return err
	}
	_, err = client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return err
	}
	return nil
}

func (client *APIClientV1) CreateCollection(ctx context.Context, name string, options ...api.CreateCollectionOption) (api.Collection, error) {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "collections")
	if err != nil {
		return nil, err
	}
	req, err := api.NewCreateCollectionOp(name, options...)
	if err != nil {
		return nil, err
	}
	err = req.Validate()
	if err != nil {
		return nil, err
	}
	queryParams := url.Values{}
	var tenant, database string
	if req.Tenant != nil {
		tenant = req.Tenant.Name()
	} else {
		tenant = client.BaseAPIClient.Tenant().Name()
	}
	if req.Database != nil {
		database = req.Database.Name()
	} else {
		database = client.BaseAPIClient.Database().Name()
	}
	queryParams.Set("tenant", tenant)
	queryParams.Set("database", database)
	reqURL = fmt.Sprintf("%s?%s", reqURL, queryParams.Encode())
	reqJSON, err := req.MarshalJSON()
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, err
	}
	resp, err := client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return nil, err
	}
	var cm CollectionModel
	if err := json.NewDecoder(resp.Body).Decode(&cm); err != nil {
		return nil, err
	}
	c := &Collection{
		CollectionBase: api.CollectionBase{
			Name:         cm.Name,
			CollectionID: cm.ID,
			Tenant:       api.NewTenant(tenant),
			Database:     api.NewDatabase(database, api.NewTenant(tenant)),
			Metadata:     cm.Metadata,
		},
		client: client,
	}
	return c, nil
}

func (client *APIClientV1) GetOrCreateCollection(ctx context.Context, name string, options ...api.CreateCollectionOption) (api.Collection, error) {
	options = append(options, api.WithIfNotExistsCreate())
	return client.CreateCollection(ctx, name, options...)
}

func (client *APIClientV1) DeleteCollection(ctx context.Context, name string) error {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "collections", name)
	if err != nil {
		return err
	}
	queryParams := url.Values{}
	queryParams.Set("tenant", client.BaseAPIClient.Tenant().Name())
	queryParams.Set("database", client.BaseAPIClient.Database().Name())
	reqURL = fmt.Sprintf("%s?%s", reqURL, queryParams.Encode())

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return nil
	}
	_, err = client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return err
	}
	return nil
}

func (client *APIClientV1) GetCollection(ctx context.Context, name string, opts ...api.GetCollectionOption) (api.Collection, error) {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "collections", name)
	if err != nil {
		return nil, err
	}
	queryParams := url.Values{}
	queryParams.Set("tenant", client.BaseAPIClient.Tenant().Name())
	queryParams.Set("database", client.BaseAPIClient.Database().Name())
	reqURL = fmt.Sprintf("%s?%s", reqURL, queryParams.Encode())

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return nil, err
	}
	var cm CollectionModel
	if err := json.NewDecoder(resp.Body).Decode(&cm); err != nil {
		return nil, err
	}
	c := &Collection{
		CollectionBase: api.CollectionBase{
			Name:         cm.Name,
			CollectionID: cm.ID,
			Tenant:       client.BaseAPIClient.Tenant(),
			Database:     client.BaseAPIClient.Database(),
			Metadata:     cm.Metadata,
		},
		client: client,
	}
	return c, nil
}

func (client *APIClientV1) CountCollections(ctx context.Context) (int, error) {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "count_collections")
	if err != nil {
		return 0, err
	}
	queryParams := url.Values{}
	queryParams.Set("tenant", client.BaseAPIClient.Tenant().Name())
	queryParams.Set("database", client.BaseAPIClient.Database().Name())

	reqURL = fmt.Sprintf("%s?%s", reqURL, queryParams.Encode())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, nil
	}
	resp, err := client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return 0, err
	}
	respBody := chhttp.ReadRespBody(resp.Body)

	return strconv.Atoi(respBody)
}

func (client *APIClientV1) ListCollections(ctx context.Context, opts ...api.ListCollectionsOption) ([]api.Collection, error) {
	listOpts := &api.ListCollectionOp{}
	for _, opt := range opts {
		err := opt(listOpts)
		if err != nil {
			return nil, err
		}
	}
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "collections")
	if err != nil {
		return nil, err
	}
	queryParams := url.Values{}
	queryParams.Set("tenant", client.BaseAPIClient.Tenant().Name())
	queryParams.Set("database", client.BaseAPIClient.Database().Name())
	if listOpts.Limit() > 0 {
		queryParams.Set("limit", strconv.Itoa(listOpts.Limit()))
	}
	if listOpts.Offset() > 0 {
		queryParams.Set("offset", strconv.Itoa(listOpts.Offset()))
	}
	reqURL = fmt.Sprintf("%s?%s", reqURL, queryParams.Encode())

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return nil, err
	}
	var cols []CollectionModel
	if err := json.NewDecoder(resp.Body).Decode(&cols); err != nil {
		return nil, err
	}

	var apiCollections = make([]api.Collection, 0)
	if len(cols) > 0 {
		for _, cm := range cols {
			c := &Collection{
				CollectionBase: api.CollectionBase{
					Name:         cm.Name,
					CollectionID: cm.ID,
					Tenant:       client.BaseAPIClient.Tenant(),
					Database:     client.BaseAPIClient.Database(),
					Metadata:     cm.Metadata,
				},
				client: client,
			}
			apiCollections = append(apiCollections, c)
		}
	}
	return apiCollections, nil
}

func (client *APIClientV1) UseTenant(ctx context.Context, tenant string) error {
	t, err := client.GetTenant(ctx, tenant)
	if err != nil {
		return err
	}
	client.BaseAPIClient.SetTenant(t)
	return nil
}

func (client *APIClientV1) UseDatabase(ctx context.Context, database string) error {
	d, err := client.GetDatabase(ctx, client.BaseAPIClient.Tenant().Name(), database)
	if err != nil {
		return err
	}
	client.BaseAPIClient.SetDatabase(d)
	return nil
}

func (client *APIClientV1) UseTenantAndDatabase(ctx context.Context, tenant, database string) error {
	db, err := client.GetDatabase(ctx, tenant, database)
	if err != nil {
		return err
	}
	client.BaseAPIClient.SetDatabase(db)
	client.BaseAPIClient.SetTenant(db.Tenant())
	return nil
}

func (client *APIClientV1) GetPreFlightConditionsRaw() map[string]interface{} {
	return map[string]interface{}{}
}

func (client *APIClientV1) Satisfies(resourceOperation api.ResourceOperation, metric interface{}, metricName string) error {
	m, ok := client.preflightLimits[fmt.Sprintf("%s#%s", string(resourceOperation.Resource()), string(resourceOperation.Operation()))]
	if !ok {
		return nil
	}

	switch metric.(type) {
	case int, int32:
		if m.(int) <= metric.(int) {
			return fmt.Errorf("%s count limit exceeded for %s %s. Expected less than or equal %v but got %v", metricName, string(resourceOperation.Resource()), string(resourceOperation.Operation()), m, metric)
		}
	case float64, float32:
		if m.(float64) <= metric.(float64) {
			return fmt.Errorf("%s count limit exceeded for %s %s. Expected less than or equal %v but got %v", metricName, string(resourceOperation.Resource()), string(resourceOperation.Operation()), m, metric)
		}

	}

	return nil
}
