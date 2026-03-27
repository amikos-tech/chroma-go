package v2

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"

	chhttp "github.com/amikos-tech/chroma-go/pkg/commons/http"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/amikos-tech/chroma-go/pkg/logger"
)

type APIClientV2 struct {
	BaseAPIClient
	preflightConditionsRaw map[string]interface{}
	preflightLimits        map[string]interface{}
	preflightCompleted     bool
	preflightMu            sync.RWMutex
	collectionCache        map[string]Collection
	collectionMu           sync.RWMutex
}

func NewHTTPClient(opts ...ClientOption) (Client, error) {
	updatedOpts := make([]ClientOption, 0)
	updatedOpts = append(updatedOpts, WithDatabaseAndTenantFromEnv()) // prepend env vars as first default
	for _, option := range opts {
		if option != nil {
			updatedOpts = append(updatedOpts, option)
		}
	}
	updatedOpts = append(updatedOpts, WithDefaultDatabaseAndTenant())
	bc, err := newBaseAPIClient(updatedOpts...)
	if err != nil {
		return nil, err
	}
	if bc.BaseURL() == "" {
		bc.setBaseURL("http://localhost:8000/api/v2")
	} else if !strings.HasSuffix(bc.BaseURL(), "/api/v2") {
		newBasePath, err := url.JoinPath(bc.BaseURL(), "/api/v2")
		if err != nil {
			return nil, err
		}
		bc.setBaseURL(newBasePath)
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
	client.preflightMu.Lock()
	defer client.preflightMu.Unlock()

	if client.preflightCompleted {
		return nil
	}

	reqURL, err := url.JoinPath(client.BaseURL(), "pre-flight-checks")
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.SendRequest(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	var preflightLimits map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&preflightLimits); err != nil {
		return errors.Wrap(err, "error decoding preflight response")
	}
	client.preflightConditionsRaw = preflightLimits
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
	reqURL, err := url.JoinPath(client.BaseURL(), "version")
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.SendRequest(httpReq)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody := chhttp.ReadRespBody(resp.Body)
	version := strings.ReplaceAll(respBody, `"`, "")
	return version, nil
}

func (client *APIClientV2) Heartbeat(ctx context.Context) error {
	reqURL, err := url.JoinPath(client.BaseURL(), "heartbeat")
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.SendRequest(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
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
	reqURL, err := url.JoinPath(client.BaseURL(), "tenants", tenant.Name())
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.SendRequest(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
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
	reqURL, err := url.JoinPath(client.BaseURL(), "tenants")
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, err
	}
	resp, err := client.SendRequest(httpReq)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating tenant %s", tenant.Name())
	}
	defer func() { _ = resp.Body.Close() }()
	return tenant, nil
}

func (client *APIClientV2) ListDatabases(ctx context.Context, tenant Tenant) ([]Database, error) {
	err := tenant.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "error validating tenant")
	}
	reqURL, err := url.JoinPath(client.BaseURL(), "tenants", tenant.Name(), "databases")
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.SendRequest(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
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
	reqURL, err := url.JoinPath(client.BaseURL(), "tenants", db.Tenant().Name(), "databases", db.Name())
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.SendRequest(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
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
	reqURL, err := url.JoinPath(client.BaseURL(), "tenants", db.Tenant().Name(), "databases")
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
	resp, err := client.SendRequest(httpReq)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating database %s", db.Name())
	}
	defer func() { _ = resp.Body.Close() }()
	return db, nil
}

func (client *APIClientV2) DeleteDatabase(ctx context.Context, db Database) error {
	err := db.Validate()
	if err != nil {
		return errors.Wrap(err, "error validating database")
	}
	reqURL, err := url.JoinPath(client.BaseURL(), "tenants", db.Tenant().Name(), "databases", db.Name())
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.SendRequest(httpReq)
	if err != nil {
		return errors.Wrapf(err, "error deleting database %s", db.Name())
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

func (client *APIClientV2) Reset(ctx context.Context) error {
	reqURL, err := url.JoinPath(client.BaseURL(), "reset")
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.SendRequest(httpReq)
	if err != nil {
		return errors.Wrap(err, "error resetting server")
	}
	defer func() { _ = resp.Body.Close() }()
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
	reqURL, err := url.JoinPath(client.BaseURL(), "tenants", req.Database.Tenant().Name(), "databases", req.Database.Name(), "collections")
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
	resp, err := client.SendRequest(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "error sending request")
	}
	defer func() { _ = resp.Body.Close() }()
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
		schema:            cm.Schema,
		configuration:     NewCollectionConfigurationFromMap(cm.ConfigurationJSON),
		client:            client,
		embeddingFunction: wrapEFCloseOnce(req.embeddingFunction),
		dimension:         cm.Dimension,
	}
	c.ownsEF.Store(true)
	client.addCollectionToCache(c)
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
	reqURL, err := url.JoinPath(client.BaseURL(), "tenants", req.Database.Tenant().Name(), "databases", req.Database.Name(), "collections", name)
	if err != nil {
		return errors.Wrap(err, "error composing delete request URL")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return errors.Wrap(err, "error creating HTTP request")
	}
	resp, err := client.SendRequest(httpReq)
	if err != nil {
		return errors.Wrap(err, "delete request error")
	}
	defer func() { _ = resp.Body.Close() }()
	client.deleteCollectionFromCache(name)
	return nil
}

func (client *APIClientV2) GetCollection(ctx context.Context, name string, opts ...GetCollectionOption) (Collection, error) {
	newOpts := append([]GetCollectionOption{WithCollectionNameGet(name), WithDatabaseGet(client.CurrentDatabase())}, opts...)
	req, err := NewGetCollectionOp(newOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "error preparing collection get request")
	}
	reqURL, err := url.JoinPath(client.BaseURL(), "tenants", req.Database.Tenant().Name(), "databases", req.Database.Name(), "collections", name)
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
	resp, err := client.SendRequest(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "error sending request")
	}
	defer func() { _ = resp.Body.Close() }()
	respBody := chhttp.ReadRespBody(resp.Body)
	var cm CollectionModel
	err = json.Unmarshal([]byte(respBody), &cm)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding response")
	}
	configuration := NewCollectionConfigurationFromMap(cm.ConfigurationJSON)
	// Auto-wire content EF first to avoid double factory instantiation
	contentEF := req.contentEmbeddingFunction
	if contentEF == nil {
		autoWiredContentEF, buildErr := BuildContentEFFromConfig(configuration)
		if buildErr != nil {
			client.logger.Warn("failed to auto-wire content embedding function", logger.ErrorField("error", buildErr))
		}
		contentEF = autoWiredContentEF
	}
	// Auto-wire dense EF: try unwrapping from content adapter first, then build from config
	ef := req.embeddingFunction
	if ef == nil {
		if unwrapper, ok := contentEF.(embeddings.EmbeddingFunctionUnwrapper); ok {
			ef = unwrapper.UnwrapEmbeddingFunction()
		} else if denseFromContent, ok := contentEF.(embeddings.EmbeddingFunction); ok {
			ef = denseFromContent
		}
		if ef == nil {
			autoWiredEF, buildErr := BuildEmbeddingFunctionFromConfig(configuration)
			if buildErr != nil {
				client.logger.Warn("failed to auto-wire embedding function", logger.ErrorField("error", buildErr))
			}
			ef = autoWiredEF
		}
	}
	c := &CollectionImpl{
		name:                     cm.Name,
		id:                       cm.ID,
		tenant:                   NewTenant(cm.Tenant),
		database:                 NewDatabase(cm.Database, NewTenant(cm.Tenant)),
		metadata:                 cm.Metadata,
		schema:                   cm.Schema,
		configuration:            configuration,
		client:                   client,
		dimension:                cm.Dimension,
		embeddingFunction:        wrapEFCloseOnce(ef),
		contentEmbeddingFunction: wrapContentEFCloseOnce(contentEF),
	}
	c.ownsEF.Store(true)
	client.addCollectionToCache(c)
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
	reqURL, err := url.JoinPath(client.BaseURL(), "tenants", req.Database.Tenant().Name(), "databases", req.Database.Name(), "collections_count")
	if err != nil {
		return 0, errors.Wrap(err, "error composing request URL")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, errors.Wrap(err, "error creating HTTP request")
	}
	resp, err := client.SendRequest(httpReq)
	if err != nil {
		return 0, errors.Wrap(err, "error sending request")
	}
	defer func() { _ = resp.Body.Close() }()
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
			configuration := NewCollectionConfigurationFromMap(cm.ConfigurationJSON)
			// Auto-wire EF from configuration
			ef, buildErr := BuildEmbeddingFunctionFromConfig(configuration)
			if buildErr != nil {
				client.logger.Warn("failed to auto-wire embedding function for collection",
					logger.String("collection", cm.Name),
					logger.ErrorField("error", buildErr))
			}
			c := &CollectionImpl{
				name:              cm.Name,
				id:                cm.ID,
				tenant:            NewTenant(cm.Tenant),
				database:          NewDatabase(cm.Database, NewTenant(cm.Tenant)),
				metadata:          cm.Metadata,
				schema:            cm.Schema,
				configuration:     configuration,
				dimension:         cm.Dimension,
				client:            client,
				embeddingFunction: ef,
			}
			c.ownsEF.Store(true)
			apiCollections = append(apiCollections, c)
		}
	}
	return apiCollections, nil
}

// Deprecated: Use UseTenantDatabase on a concrete client (*APIClientV2) to
// validate remote state and atomically update the local tenant/database pair.
// The generic Client interface does not expose UseTenantDatabase.
func (client *APIClientV2) UseTenant(ctx context.Context, tenant Tenant) error {
	if tenant == nil {
		return errors.New("tenant cannot be nil")
	}
	t, err := client.GetTenant(ctx, tenant)
	if err != nil {
		return err
	}
	client.SetTenantAndDatabase(t, t.Database(DefaultDatabase))
	return nil
}

// UseDatabase validates and switches the active database. The tenant is derived
// from the database object itself.
func (client *APIClientV2) UseDatabase(ctx context.Context, database Database) error {
	if database == nil {
		return errors.New("database cannot be nil")
	}
	err := database.Validate()
	if err != nil {
		return errors.Wrap(err, "error validating database")
	}
	d, err := client.GetDatabase(ctx, database)
	if err != nil {
		return errors.Wrap(err, "error getting database")
	}
	client.SetTenantAndDatabase(d.Tenant(), d)
	return nil
}

// UseTenantDatabase validates tenant/database against the server, then updates
// the local active tenant/database pair under a single lock acquisition.
func (client *APIClientV2) UseTenantDatabase(ctx context.Context, tenant Tenant, database Database) error {
	if tenant == nil {
		return errors.New("tenant cannot be nil")
	}
	t, err := client.GetTenant(ctx, tenant)
	if err != nil {
		return errors.Wrap(err, "error getting tenant")
	}
	var db Database
	if database == nil {
		db = NewDatabase(DefaultDatabase, t)
	} else {
		db = NewDatabase(database.Name(), t)
	}
	if err := db.Validate(); err != nil {
		return errors.Wrap(err, "error validating database")
	}
	d, err := client.GetDatabase(ctx, db)
	if err != nil {
		return errors.Wrap(err, "error getting database")
	}
	client.SetTenantAndDatabase(d.Tenant(), d)
	return nil
}

func (client *APIClientV2) CurrentTenant() Tenant {
	return client.Tenant()
}

func (client *APIClientV2) CurrentDatabase() Database {
	return client.Database()
}

func (client *APIClientV2) getPreFlightConditionsRaw() map[string]interface{} {
	client.preflightMu.RLock()
	defer client.preflightMu.RUnlock()

	cp := make(map[string]interface{}, len(client.preflightConditionsRaw))
	for k, v := range client.preflightConditionsRaw {
		cp[k] = v
	}
	return cp
}

func (client *APIClientV2) satisfies(resourceOperation ResourceOperation, metric interface{}, metricName string) error {
	client.preflightMu.RLock()
	m, ok := client.preflightLimits[fmt.Sprintf("%s#%s", string(resourceOperation.Resource()), string(resourceOperation.Operation()))]
	client.preflightMu.RUnlock()
	if !ok {
		return nil
	}

	// preflightLimits always stores int values, use comma-ok idiom to avoid panics
	limit, ok := m.(int)
	if !ok {
		return nil
	}

	// Convert metric to int for comparison
	var metricVal int
	switch v := metric.(type) {
	case int:
		metricVal = v
	case int32:
		metricVal = int(v)
	case int64:
		metricVal = int(v)
	case float64:
		metricVal = int(v)
	case float32:
		metricVal = int(v)
	default:
		return nil
	}

	if limit < metricVal {
		return errors.Errorf("%s count limit exceeded for %s %s. Maximum allowed is %v but got %v", metricName, string(resourceOperation.Resource()), string(resourceOperation.Operation()), limit, metricVal)
	}

	return nil
}

func (client *APIClientV2) GetIdentity(ctx context.Context) (Identity, error) {
	var identity Identity
	reqURL, err := url.JoinPath(client.BaseURL(), "auth", "identity")
	if err != nil {
		return identity, errors.Wrap(err, "error composing request URL")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return identity, errors.Wrap(err, "error creating HTTP request")
	}
	resp, err := client.SendRequest(httpReq)
	if err != nil {
		return identity, errors.Wrap(err, "error sending request")
	}
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(&identity); err != nil {
		return identity, errors.Wrap(err, "error decoding response")
	}
	return identity, nil
}

func (client *APIClientV2) Close() error {
	if client.httpClient != nil {
		client.httpClient.CloseIdleConnections()
	}
	var errs []error
	// Copy collections while holding lock to avoid race conditions
	client.collectionMu.RLock()
	collections := make([]Collection, 0, len(client.collectionCache))
	for _, c := range client.collectionCache {
		collections = append(collections, c)
	}
	client.collectionMu.RUnlock()
	// Close collections without holding the lock to avoid deadlocks
	for _, c := range collections {
		err := c.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}
	// Sync the logger to flush any buffered log entries
	if client.logger != nil {
		if err := client.logger.Sync(); err != nil {
			// Ignore sync errors for stderr/stdout which are common in tests
			// These occur when the underlying file descriptor is invalid (e.g., in tests)
			// See: https://github.com/uber-go/zap/issues/991
			if !strings.Contains(err.Error(), "bad file descriptor") &&
				!strings.Contains(err.Error(), "/dev/stderr") &&
				!strings.Contains(err.Error(), "/dev/stdout") {
				errs = append(errs, errors.Wrap(err, "error syncing logger"))
			}
		}
	}
	if len(errs) > 0 {
		return stderrors.Join(errs...)
	}
	return nil
}

func (client *APIClientV2) localSetPreflightLimit(maxBatchSize int) {
	if maxBatchSize <= 0 {
		return
	}
	client.preflightMu.Lock()
	defer client.preflightMu.Unlock()

	if client.preflightCompleted {
		return
	}

	client.preflightConditionsRaw = map[string]interface{}{
		"max_batch_size": maxBatchSize,
	}
	if client.preflightLimits == nil {
		client.preflightLimits = map[string]interface{}{}
	}
	for _, op := range []OperationType{OperationCreate, OperationGet, OperationQuery, OperationUpdate, OperationDelete} {
		key := fmt.Sprintf("%s#%s", string(ResourceCollection), string(op))
		client.preflightLimits[key] = maxBatchSize
	}
	client.preflightCompleted = true
}

func (client *APIClientV2) localCollectionByName(name string) Collection {
	client.collectionMu.RLock()
	defer client.collectionMu.RUnlock()
	return client.collectionCache[name]
}

func (client *APIClientV2) localAddCollectionToCache(c Collection) {
	if c == nil {
		return
	}
	client.collectionMu.Lock()
	defer client.collectionMu.Unlock()
	if client.collectionCache == nil {
		client.collectionCache = map[string]Collection{}
	}
	client.collectionCache[c.Name()] = c
}

func (client *APIClientV2) localDeleteCollectionFromCache(name string) {
	if name == "" {
		return
	}
	// Determine what to close under the lock, then close outside the lock
	// to avoid blocking concurrent cache operations during slow EF teardown.
	var toClose Collection
	client.collectionMu.Lock()
	deleted, exists := client.collectionCache[name]
	delete(client.collectionCache, name)
	if exists {
		impl, ok := deleted.(*CollectionImpl)
		if ok && impl.ownsEF.Load() {
			// Owner being removed — transfer EF ownership to a fork that
			// shares the same underlying EF, or close it if no such fork
			// exists. EFs are wrapped in closeOnce wrappers at creation
			// time, so parent and fork share the same wrapper; the
			// wrapper's sync.Once ensures Close() runs at most once.
			transferred := false
			for _, c := range client.collectionCache {
				other, ok := c.(*CollectionImpl)
				if !ok || other.ownsEF.Load() {
					continue
				}
				if collectionsShareEF(impl, other) {
					impl.ownsEF.Store(false)
					other.ownsEF.Store(true)
					transferred = true
					break
				}
			}
			if !transferred {
				toClose = deleted
			}
		}
	}
	client.collectionMu.Unlock()
	if toClose == nil {
		return
	}
	if err := toClose.Close(); err != nil {
		if client.logger != nil {
			client.logger.Error("failed to close EF during collection cache cleanup",
				logger.String("collection", name),
				logger.ErrorField("error", err))
			return
		}
		logCollectionCleanupCloseErrorToStderr(name, err)
	}
}

func (client *APIClientV2) localRenameCollectionInCache(oldName string, collection Collection) {
	if collection == nil {
		return
	}
	client.collectionMu.Lock()
	defer client.collectionMu.Unlock()
	if client.collectionCache == nil {
		client.collectionCache = map[string]Collection{}
	}
	if oldName != "" {
		delete(client.collectionCache, oldName)
	}
	client.collectionCache[collection.Name()] = collection
}

func (client *APIClientV2) addCollectionToCache(c Collection) {
	client.localAddCollectionToCache(c)
}

func (client *APIClientV2) deleteCollectionFromCache(name string) {
	client.localDeleteCollectionFromCache(name)
}

// collectionsShareEF reports whether two CollectionImpl instances reference the
// same underlying embedding function (after unwrapping closeOnce wrappers).
func collectionsShareEF(a, b *CollectionImpl) bool {
	if a.embeddingFunction != nil && b.embeddingFunction != nil {
		if unwrapCloseOnceEF(a.embeddingFunction) == unwrapCloseOnceEF(b.embeddingFunction) {
			return true
		}
	}
	if a.contentEmbeddingFunction != nil && b.contentEmbeddingFunction != nil {
		if unwrapCloseOnceContentEF(a.contentEmbeddingFunction) == unwrapCloseOnceContentEF(b.contentEmbeddingFunction) {
			return true
		}
	}
	return false
}

func unwrapCloseOnceEF(ef embeddings.EmbeddingFunction) embeddings.EmbeddingFunction {
	if w, ok := ef.(*closeOnceEF); ok {
		return w.ef
	}
	if w, ok := ef.(*closeOnceContentEF); ok {
		if inner, ok := w.ef.(embeddings.EmbeddingFunction); ok {
			return inner
		}
	}
	return ef
}

func unwrapCloseOnceContentEF(ef embeddings.ContentEmbeddingFunction) embeddings.ContentEmbeddingFunction {
	if w, ok := ef.(*closeOnceContentEF); ok {
		return w.ef
	}
	return ef
}
