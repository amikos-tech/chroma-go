package v2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	chhttp "github.com/amikos-tech/chroma-go/pkg/commons/http"
)

type APIClientV2 struct {
	BaseAPIClient
	preflightLimits    map[string]interface{}
	preflightCompleted bool
	collectionCache    map[string]Collection
}

func NewHTTPClient(opts ...ClientOption) (Client, error) {
	bc, err := newBaseAPIClient(opts...)
	if err != nil {
		return nil, err
	}
	if bc.BaseURL() == "" {
		bc.SetBaseURL("http://localhost:8080/api/v2")
	} else if !strings.HasSuffix(bc.BaseURL(), "/api/v2") {
		newBasePath, err := url.JoinPath(bc.BaseURL(), "/api/v2")
		if err != nil {
			return nil, err
		}
		bc.SetBaseURL(newBasePath)
	}
	c := &APIClientV2{
		BaseAPIClient:      *bc,
		preflightLimits:    map[string]interface{}{},
		preflightCompleted: false,
		collectionCache:    map[string]Collection{},
	}
	return c, nil
}

func (client *APIClientV2) PreFlight(ctx context.Context) error {
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
			client.preflightLimits[fmt.Sprintf("%s#%s", string(ResourceCollection), string(OperationCreate))] = int(maxBatchSize)
			client.preflightLimits[fmt.Sprintf("%s#%s", string(ResourceCollection), string(OperationGet))] = int(maxBatchSize)
			client.preflightLimits[fmt.Sprintf("%s#%s", string(ResourceCollection), string(OperationQuery))] = int(maxBatchSize)
			client.preflightLimits[fmt.Sprintf("%s#%s", string(ResourceCollection), string(OperationUpdate))] = int(maxBatchSize)
			client.preflightLimits[fmt.Sprintf("%s#%s", string(ResourceCollection), string(OperationDelete))] = int(maxBatchSize)
		}
	}
	client.preflightCompleted = true
	return nil
}

func (client *APIClientV2) GetVersion(ctx context.Context) (string, error) {
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

func (client *APIClientV2) Heartbeat(ctx context.Context) error {
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
		return errors.Errorf("heartbeat failed")
	}
}

func (client *APIClientV2) GetTenant(ctx context.Context, tenant Tenant) (Tenant, error) {
	err := tenant.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "error validating tenant")
	}
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", tenant.Name())
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
	return NewTenantFromJSON(respBody)
}

func (client *APIClientV2) CreateTenant(ctx context.Context, tenant Tenant) (Tenant, error) {
	err := tenant.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "error validating tenant")
	}
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
	_, err = client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return nil, err
	}
	return tenant, nil
}

func (client *APIClientV2) ListDatabases(ctx context.Context, tenant Tenant) ([]Database, error) {
	err := tenant.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "error validating tenant")
	}
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", tenant.Name(), "databases")
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
	var dbs []map[string]interface{}
	if err := json.Unmarshal([]byte(respBody), &dbs); err != nil {
		return nil, errors.Wrap(err, "error decoding response")
	}
	var databases []Database
	for _, db := range dbs {
		database, err := NewDatabaseFromMap(db)
		if err != nil {
			return nil, errors.Wrap(err, "error decoding database")
		}
		databases = append(databases, database)
	}
	return databases, nil
}

func (client *APIClientV2) GetDatabase(ctx context.Context, db Database) (Database, error) {
	err := db.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "error validating database")
	}
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", db.Tenant().Name(), "databases", db.Name())
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
	newDB, err := NewDatabaseFromJSON(respBody)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding response")
	}
	return newDB, nil
}

func (client *APIClientV2) CreateDatabase(ctx context.Context, db Database) (Database, error) {
	err := db.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "error validating database")
	}
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", db.Tenant().Name(), "databases")
	if err != nil {
		return nil, err
	}
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
	return db, nil
}

func (client *APIClientV2) DeleteDatabase(ctx context.Context, db Database) error {
	err := db.Validate()
	if err != nil {
		return errors.Wrap(err, "error validating database")
	}
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", db.Tenant().Name(), "databases", db.Name())
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return err
	}
	_, err = client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return err
	}
	return nil
}

func (client *APIClientV2) Reset(ctx context.Context) error {
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

func (client *APIClientV2) CreateCollection(ctx context.Context, name string, options ...CreateCollectionOption) (Collection, error) {
	newOptions := append([]CreateCollectionOption{WithDatabaseCreate(client.CurrentDatabase())}, options...)
	req, err := NewCreateCollectionOp(name, newOptions...)
	if err != nil {
		return nil, errors.Wrap(err, "error preparing collection create request")
	}
	err = req.PrepareAndValidateCollectionRequest()
	if err != nil {
		return nil, errors.Wrap(err, "error validating collection create request")
	}
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", req.Database.Tenant().Name(), "databases", req.Database.Name(), "collections")
	if err != nil {
		return nil, errors.Wrap(err, "error composing request URL")
	}
	reqJSON, err := req.MarshalJSON()
	if err != nil {
		return nil, errors.Wrap(err, "error marshalling request JSON")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, errors.Wrap(err, "error creating HTTP request")
	}
	resp, err := client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "error sending request")
	}
	var cm CollectionModel
	if err := json.NewDecoder(resp.Body).Decode(&cm); err != nil {
		return nil, errors.Wrap(err, "error decoding response")
	}
	c := &CollectionImpl{
		name:              cm.Name,
		id:                cm.ID,
		tenant:            NewTenant(cm.Tenant),
		database:          NewDatabase(cm.Database, NewTenant(cm.Tenant)),
		metadata:          cm.Metadata,
		client:            client,
		embeddingFunction: req.embeddingFunction,
	}
	client.collectionCache[cm.Name] = c
	return c, nil
}

func (client *APIClientV2) GetOrCreateCollection(ctx context.Context, name string, options ...CreateCollectionOption) (Collection, error) {
	options = append(options, WithIfNotExistsCreate())
	return client.CreateCollection(ctx, name, options...)
}

func (client *APIClientV2) DeleteCollection(ctx context.Context, name string, options ...DeleteCollectionOption) error {
	newOpts := append([]DeleteCollectionOption{WithDatabaseDelete(client.CurrentDatabase())}, options...)
	req, err := NewDeleteCollectionOp(newOpts...)
	if err != nil {
		return errors.Wrap(err, "error preparing collection delete request")
	}
	err = req.PrepareAndValidateCollectionRequest()
	if err != nil {
		return errors.Wrap(err, "error validating collection delete request")
	}
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", req.Database.Tenant().Name(), "databases", req.Database.Name(), "collections", name)
	if err != nil {
		return errors.Wrap(err, "error composing delete request URL")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return errors.Wrap(err, "error creating HTTP request")
	}
	_, err = client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return errors.Wrap(err, "delete request error")
	}
	delete(client.collectionCache, name)
	return nil
}

func (client *APIClientV2) GetCollection(ctx context.Context, name string, opts ...GetCollectionOption) (Collection, error) {
	newOpts := append([]GetCollectionOption{WithCollectionNameGet(name), WithDatabaseGet(client.CurrentDatabase())}, opts...)
	req, err := NewGetCollectionOp(newOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "error preparing collection get request")
	}
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", req.Database.Tenant().Name(), "databases", req.Database.Name(), "collections", name)
	if err != nil {
		return nil, errors.Wrap(err, "error composing request URL")
	}
	err = req.PrepareAndValidateCollectionRequest()
	if err != nil {
		return nil, errors.Wrap(err, "error validating collection get request")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating HTTP request")
	}
	resp, err := client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "error sending request")
	}
	respBody := chhttp.ReadRespBody(resp.Body)
	var cm CollectionModel
	err = json.Unmarshal([]byte(respBody), &cm)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding response")
	}
	c := &CollectionImpl{
		name:              cm.Name,
		id:                cm.ID,
		tenant:            NewTenant(cm.Tenant),
		database:          NewDatabase(cm.Database, NewTenant(cm.Tenant)),
		metadata:          cm.Metadata,
		client:            client,
		embeddingFunction: req.embeddingFunction,
	}
	client.collectionCache[name] = c
	return c, nil
}

func (client *APIClientV2) CountCollections(ctx context.Context, opts ...CountCollectionsOption) (int, error) {
	newOpts := append([]CountCollectionsOption{WithDatabaseCount(client.CurrentDatabase())}, opts...)
	req, err := NewCountCollectionsOp(newOpts...)
	if err != nil {
		return 0, errors.Wrap(err, "error preparing collection count request")
	}
	err = req.PrepareAndValidateCollectionRequest()
	if err != nil {
		return 0, errors.Wrap(err, "error validating collection count request")
	}
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", req.Database.Tenant().Name(), "databases", req.Database.Name(), "collections_count")
	if err != nil {
		return 0, errors.Wrap(err, "error composing request URL")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, errors.Wrap(err, "error creating HTTP request")
	}
	resp, err := client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return 0, errors.Wrap(err, "error sending request")
	}
	respBody := chhttp.ReadRespBody(resp.Body)
	count, err := strconv.Atoi(respBody)
	if err != nil {
		return 0, errors.Wrap(err, "error converting response to int")
	}
	return count, nil
}

func (client *APIClientV2) ListCollections(ctx context.Context, opts ...ListCollectionsOption) ([]Collection, error) {
	newOpts := append([]ListCollectionsOption{WithDatabaseList(client.CurrentDatabase())}, opts...)
	req, err := NewListCollectionsOp(newOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "error preparing collection list request")
	}
	err = req.PrepareAndValidateCollectionRequest()
	if err != nil {
		return nil, errors.Wrap(err, "error validating collection list request")
	}
	reqURL, err := url.JoinPath("tenants", req.Database.Tenant().Name(), "databases", req.Database.Name(), "collections")
	if err != nil {
		return nil, errors.Wrap(err, "error composing request URL")
	}
	queryParams := url.Values{}
	if req.Limit() > 0 {
		queryParams.Set("limit", strconv.Itoa(req.Limit()))
	}
	if req.Offset() > 0 {
		queryParams.Set("offset", strconv.Itoa(req.Offset()))
	}
	reqURL = fmt.Sprintf("%s?%s", reqURL, queryParams.Encode())
	resp, err := client.ExecuteRequest(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error executing request")
	}
	var cols []CollectionModel
	if err := json.Unmarshal(resp, &cols); err != nil {
		return nil, errors.Wrap(err, "error decoding response")
	}

	var apiCollections = make([]Collection, 0)
	if len(cols) > 0 {
		for _, cm := range cols {
			c := &CollectionImpl{
				name:     cm.Name,
				id:       cm.ID,
				tenant:   NewTenant(cm.Tenant),
				database: NewDatabase(cm.Database, NewTenant(cm.Tenant)),
				metadata: cm.Metadata,
				client:   client,
			}
			apiCollections = append(apiCollections, c)
		}
	}
	return apiCollections, nil
}

func (client *APIClientV2) UseTenant(ctx context.Context, tenant Tenant) error {
	t, err := client.GetTenant(ctx, tenant)
	if err != nil {
		return err
	}
	client.BaseAPIClient.SetTenant(t)
	client.BaseAPIClient.SetDatabase(t.Database(DefaultDatabase)) // TODO is this optimal?
	return nil
}

func (client *APIClientV2) UseDatabase(ctx context.Context, database Database) error {
	err := database.Validate()
	if err != nil {
		return errors.Wrap(err, "error validating database")
	}
	d, err := client.GetDatabase(ctx, database)
	if err != nil {
		return err
	}
	client.BaseAPIClient.SetDatabase(d)
	client.BaseAPIClient.SetTenant(d.Tenant())
	return nil
}

func (client *APIClientV2) CurrentTenant() Tenant {
	return client.BaseAPIClient.Tenant()
}

func (client *APIClientV2) CurrentDatabase() Database {
	return client.BaseAPIClient.Database()
}

func (client *APIClientV2) GetPreFlightConditionsRaw() map[string]interface{} {
	return map[string]interface{}{}
}

func (client *APIClientV2) Satisfies(resourceOperation ResourceOperation, metric interface{}, metricName string) error {
	m, ok := client.preflightLimits[fmt.Sprintf("%s#%s", string(resourceOperation.Resource()), string(resourceOperation.Operation()))]
	if !ok {
		return nil
	}

	switch metric.(type) {
	case int, int32:
		if m.(int) <= metric.(int) {
			return errors.Errorf("%s count limit exceeded for %s %s. Expected less than or equal %v but got %v", metricName, string(resourceOperation.Resource()), string(resourceOperation.Operation()), m, metric)
		}
	case float64, float32:
		if m.(float64) <= metric.(float64) {
			return errors.Errorf("%s count limit exceeded for %s %s. Expected less than or equal %v but got %v", metricName, string(resourceOperation.Resource()), string(resourceOperation.Operation()), m, metric)
		}
	}

	return nil
}

func (client *APIClientV2) GetIdentity(ctx context.Context) (Identity, error) {
	var identity Identity
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "auth", "identity")
	if err != nil {
		return identity, errors.Wrap(err, "error composing request URL")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return identity, errors.Wrap(err, "error creating HTTP request")
	}
	resp, err := client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return identity, errors.Wrap(err, "error sending request")
	}
	if err := json.NewDecoder(resp.Body).Decode(&identity); err != nil {
		return identity, errors.Wrap(err, "error decoding response")
	}
	return identity, nil
}

func (client *APIClientV2) Close() error {
	if client.BaseAPIClient.httpClient != nil {
		client.BaseAPIClient.httpClient.CloseIdleConnections()
	}
	var errs []error
	if len(client.collectionCache) > 0 {
		for _, c := range client.collectionCache {
			err := c.Close()
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return errors.Errorf("error closing collections: %v", errs)
	}
	return nil
}
