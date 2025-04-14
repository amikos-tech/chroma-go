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

func (client *APIClientV2) GetTenant(ctx context.Context, tenant string) (Tenant, error) {
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
	return NewTenantFromJSON(respBody)
}

func (client *APIClientV2) CreateTenant(ctx context.Context, tenant Tenant) (Tenant, error) {
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

func (client *APIClientV2) ListDatabases(ctx context.Context, tenant string) ([]Database, error) {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", tenant, "databases")
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

func (client *APIClientV2) GetDatabase(ctx context.Context, tenant, database string) (Database, error) {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", tenant, "databases", database)
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
	db, err := NewDatabaseFromJSON(respBody)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding response")
	}
	return db, nil
}

func (client *APIClientV2) CreateDatabase(ctx context.Context, tenant, database string) (Database, error) {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", tenant, "databases")
	if err != nil {
		return nil, err
	}
	db := NewDatabase(database, NewTenant(tenant))
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

func (client *APIClientV2) DeleteDatabase(ctx context.Context, tenant, database string) error {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", tenant, "databases", database)
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
	req, err := NewCreateCollectionOp(name, options...)
	if err != nil {
		return nil, errors.Wrap(err, "error preparing collection create request")
	}
	err = req.PrepareAndValidateCollectionRequest()
	if err != nil {
		return nil, errors.Wrap(err, "error validating collection create request")
	}
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
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", tenant, "databases", database, "collections")
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
	return c, nil
}

func (client *APIClientV2) GetOrCreateCollection(ctx context.Context, name string, options ...CreateCollectionOption) (Collection, error) {
	options = append(options, WithIfNotExistsCreate())
	return client.CreateCollection(ctx, name, options...)
}

func (client *APIClientV2) DeleteCollection(ctx context.Context, name string) error {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", client.BaseAPIClient.Tenant().Name(), "databases", client.BaseAPIClient.Database().Name(), "collections", name)
	if err != nil {
		return errors.Wrap(err, "error composing request URL")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return errors.Wrap(err, "error creating HTTP request")
	}
	_, err = client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return errors.Wrap(err, "error sending request")
	}
	return nil
}

func (client *APIClientV2) GetCollection(ctx context.Context, name string, opts ...GetCollectionOption) (Collection, error) {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", client.BaseAPIClient.Tenant().Name(), "databases", client.BaseAPIClient.Database().Name(), "collections", name)
	if err != nil {
		return nil, errors.Wrap(err, "error composing request URL")
	}
	getOp, err := NewGetCollectionOp(opts...)
	if err != nil {
		return nil, errors.Wrap(err, "error preparing collection get request")
	}
	err = getOp.PrepareAndValidateCollectionRequest()
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
		embeddingFunction: getOp.embeddingFunction,
	}
	return c, nil
}

func (client *APIClientV2) CountCollections(ctx context.Context) (int, error) {
	reqURL, err := url.JoinPath(client.BaseAPIClient.BaseURL(), "tenants", client.BaseAPIClient.Tenant().Name(), "databases", client.BaseAPIClient.Database().Name(), "collections_count")
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
	listOpts := &ListCollectionOp{}
	for _, opt := range opts {
		err := opt(listOpts)
		if err != nil {
			return nil, errors.Wrap(err, "error applying list collection option")
		}
	}
	reqURL, err := url.JoinPath("tenants", client.BaseAPIClient.Tenant().Name(), "databases", client.BaseAPIClient.Database().Name(), "collections")
	if err != nil {
		return nil, errors.Wrap(err, "error composing request URL")
	}
	queryParams := url.Values{}
	if listOpts.Limit() > 0 {
		queryParams.Set("limit", strconv.Itoa(listOpts.Limit()))
	}
	if listOpts.Offset() > 0 {
		queryParams.Set("offset", strconv.Itoa(listOpts.Offset()))
	}
	reqURL = fmt.Sprintf("%s?%s", reqURL, queryParams.Encode())

	resp, err := client.ExecuteRequest(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error executing request")
	}

	// httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "error creating HTTP request")
	// }
	// resp, err := client.BaseAPIClient.SendRequest(httpReq)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "error sending request")
	// }
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

func (client *APIClientV2) UseTenant(ctx context.Context, tenant string) error {
	t, err := client.GetTenant(ctx, tenant)
	if err != nil {
		return err
	}
	client.BaseAPIClient.SetTenant(t)
	return nil
}

func (client *APIClientV2) UseDatabase(ctx context.Context, database string) error {
	d, err := client.GetDatabase(ctx, client.BaseAPIClient.Tenant().Name(), database)
	if err != nil {
		return err
	}
	client.BaseAPIClient.SetDatabase(d)
	return nil
}

func (client *APIClientV2) UseTenantAndDatabase(ctx context.Context, tenant, database string) error {
	db, err := client.GetDatabase(ctx, tenant, database)
	if err != nil {
		return err
	}
	client.BaseAPIClient.SetDatabase(db)
	client.BaseAPIClient.SetTenant(db.Tenant())
	return nil
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
