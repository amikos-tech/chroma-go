package v2

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"math"
	"strings"
	"sync"

	"github.com/pkg/errors"

	localchroma "github.com/amikos-tech/chroma-go-local"
	embeddingspkg "github.com/amikos-tech/chroma-go/pkg/embeddings"
)

type embeddedCollectionState struct {
	embeddingFunction embeddingspkg.EmbeddingFunction
	metadata          CollectionMetadata
	configuration     CollectionConfiguration
	schema            *Schema
	dimension         int
}

type localClientState interface {
	Close() error
	CurrentTenant() Tenant
	CurrentDatabase() Database
	SetTenant(tenant Tenant)
	SetDatabase(database Database)
	Satisfies(resourceOperation ResourceOperation, metric interface{}, metricName string) error
	localSetPreflightLimit(maxBatchSize int)
	localCollectionByName(name string) Collection
	localAddCollectionToCache(collection Collection)
	localDeleteCollectionFromCache(name string)
	localRenameCollectionInCache(oldName string, collection Collection)
}

type embeddedLocalClient struct {
	state    localClientState
	embedded localEmbeddedRuntime

	collectionStateMu sync.RWMutex
	collectionState   map[string]*embeddedCollectionState
}

func newEmbeddedLocalClient(cfg *localClientConfig, embedded localEmbeddedRuntime) (Client, error) {
	if cfg == nil {
		return nil, errors.New("local client config cannot be nil")
	}
	if embedded == nil {
		return nil, errors.New("embedded runtime cannot be nil")
	}

	stateClient, err := newEmbeddedLocalStateClient(cfg.clientOptions...)
	if err != nil {
		return nil, errors.Wrap(err, "error creating local client state")
	}

	if err := localWaitEmbeddedReadyFunc(embedded); err != nil {
		_ = stateClient.Close()
		return nil, errors.Wrap(err, "embedded runtime failed readiness checks")
	}

	return &embeddedLocalClient{
		state:           stateClient,
		embedded:        embedded,
		collectionState: map[string]*embeddedCollectionState{},
	}, nil
}

func newEmbeddedLocalStateClient(options ...ClientOption) (localClientState, error) {
	updatedOptions := make([]ClientOption, 0, len(options)+1)
	for _, option := range options {
		if option != nil {
			updatedOptions = append(updatedOptions, option)
		}
	}
	// Local runtime should not implicitly inherit CHROMA_DATABASE/CHROMA_TENANT.
	updatedOptions = append(updatedOptions, WithDefaultDatabaseAndTenant())

	baseClient, err := newBaseAPIClient(updatedOptions...)
	if err != nil {
		return nil, err
	}

	return &APIClientV2{
		BaseAPIClient:      *baseClient,
		preflightLimits:    map[string]interface{}{},
		preflightCompleted: false,
		collectionCache:    map[string]Collection{},
	}, nil
}

func (client *embeddedLocalClient) PreFlight(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	maxBatchSize, err := client.embedded.MaxBatchSize()
	if err != nil {
		return errors.Wrap(err, "error retrieving embedded max batch size")
	}
	client.state.localSetPreflightLimit(int(maxBatchSize))
	return nil
}

func (client *embeddedLocalClient) Heartbeat(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := client.embedded.Heartbeat()
	if err != nil {
		return errors.Wrap(err, "embedded heartbeat failed")
	}
	return nil
}

func (client *embeddedLocalClient) GetVersion(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	version, err := localVersionWithErrorFunc()
	if err != nil {
		return "", errors.Wrap(err, "error reading embedded runtime version")
	}
	return version, nil
}

func (client *embeddedLocalClient) GetIdentity(ctx context.Context) (Identity, error) {
	if err := ctx.Err(); err != nil {
		return Identity{}, err
	}
	identity := Identity{UserID: "local_embedded"}
	if tenant := client.CurrentTenant(); tenant != nil {
		identity.Tenant = tenant.Name()
	}
	if database := client.CurrentDatabase(); database != nil {
		identity.Databases = []string{database.Name()}
	}
	return identity, nil
}

func (client *embeddedLocalClient) GetTenant(ctx context.Context, tenant Tenant) (Tenant, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, errors.New("tenant cannot be nil")
	}
	if err := tenant.Validate(); err != nil {
		return nil, errors.Wrap(err, "error validating tenant")
	}

	t, err := client.embedded.GetTenant(localchroma.EmbeddedGetTenantRequest{Name: tenant.Name()})
	if err != nil {
		return nil, errors.Wrapf(err, "error getting tenant %s", tenant.Name())
	}
	return NewTenant(t.Name), nil
}

func (client *embeddedLocalClient) UseTenant(ctx context.Context, tenant Tenant) error {
	t, err := client.GetTenant(ctx, tenant)
	if err != nil {
		return err
	}
	client.state.SetTenant(t)
	client.state.SetDatabase(t.Database(DefaultDatabase))
	return nil
}

func (client *embeddedLocalClient) UseDatabase(ctx context.Context, database Database) error {
	if database == nil {
		return errors.New("database cannot be nil")
	}
	if err := database.Validate(); err != nil {
		return errors.Wrap(err, "error validating database")
	}
	db, err := client.GetDatabase(ctx, database)
	if err != nil {
		return err
	}
	client.state.SetDatabase(db)
	client.state.SetTenant(db.Tenant())
	return nil
}

func (client *embeddedLocalClient) CreateTenant(ctx context.Context, tenant Tenant) (Tenant, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, errors.New("tenant cannot be nil")
	}
	if err := tenant.Validate(); err != nil {
		return nil, errors.Wrap(err, "error validating tenant")
	}
	if err := client.embedded.CreateTenant(localchroma.EmbeddedCreateTenantRequest{Name: tenant.Name()}); err != nil {
		return nil, errors.Wrapf(err, "error creating tenant %s", tenant.Name())
	}
	return tenant, nil
}

func (client *embeddedLocalClient) ListDatabases(ctx context.Context, tenant Tenant) ([]Database, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, errors.New("tenant cannot be nil")
	}
	if err := tenant.Validate(); err != nil {
		return nil, errors.Wrap(err, "error validating tenant")
	}
	dbs, err := client.embedded.ListDatabases(localchroma.EmbeddedListDatabasesRequest{TenantID: tenant.Name()})
	if err != nil {
		return nil, errors.Wrapf(err, "error listing databases for tenant %s", tenant.Name())
	}

	result := make([]Database, 0, len(dbs))
	for _, db := range dbs {
		tenantName := db.Tenant
		if tenantName == "" {
			tenantName = tenant.Name()
		}
		result = append(result, NewDatabase(db.Name, NewTenant(tenantName)))
	}
	return result, nil
}

func (client *embeddedLocalClient) GetDatabase(ctx context.Context, db Database) (Database, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if db == nil {
		return nil, errors.New("database cannot be nil")
	}
	if err := db.Validate(); err != nil {
		return nil, errors.Wrap(err, "error validating database")
	}

	response, err := client.embedded.GetDatabase(localchroma.EmbeddedGetDatabaseRequest{
		Name:     db.Name(),
		TenantID: db.Tenant().Name(),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "error getting database %s", db.Name())
	}
	tenantName := response.Tenant
	if tenantName == "" {
		tenantName = db.Tenant().Name()
	}
	return NewDatabase(response.Name, NewTenant(tenantName)), nil
}

func (client *embeddedLocalClient) CreateDatabase(ctx context.Context, db Database) (Database, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if db == nil {
		return nil, errors.New("database cannot be nil")
	}
	if err := db.Validate(); err != nil {
		return nil, errors.Wrap(err, "error validating database")
	}
	if err := client.embedded.CreateDatabase(localchroma.EmbeddedCreateDatabaseRequest{
		Name:     db.Name(),
		TenantID: db.Tenant().Name(),
	}); err != nil {
		return nil, errors.Wrapf(err, "error creating database %s", db.Name())
	}
	return db, nil
}

func (client *embeddedLocalClient) DeleteDatabase(ctx context.Context, db Database) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if db == nil {
		return errors.New("database cannot be nil")
	}
	if err := db.Validate(); err != nil {
		return errors.Wrap(err, "error validating database")
	}
	return client.embedded.DeleteDatabase(localchroma.EmbeddedDeleteDatabaseRequest{
		Name:     db.Name(),
		TenantID: db.Tenant().Name(),
	})
}

func (client *embeddedLocalClient) CurrentTenant() Tenant {
	return client.state.CurrentTenant()
}

func (client *embeddedLocalClient) CurrentDatabase() Database {
	return client.state.CurrentDatabase()
}

func (client *embeddedLocalClient) Reset(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return client.embedded.Reset()
}

func (client *embeddedLocalClient) CreateCollection(ctx context.Context, name string, options ...CreateCollectionOption) (Collection, error) {
	newOptions := append([]CreateCollectionOption{WithDatabaseCreate(client.CurrentDatabase())}, options...)
	req, err := NewCreateCollectionOp(name, newOptions...)
	if err != nil {
		return nil, errors.Wrap(err, "error preparing collection create request")
	}
	if err := req.PrepareAndValidateCollectionRequest(); err != nil {
		return nil, errors.Wrap(err, "error validating collection create request")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	isNewCreation := true
	if req.CreateIfNotExists {
		existingCollection, lookupErr := client.embedded.GetCollection(localchroma.EmbeddedGetCollectionRequest{
			Name:         req.Name,
			TenantID:     req.Database.Tenant().Name(),
			DatabaseName: req.Database.Name(),
		})
		isNewCreation = lookupErr != nil || existingCollection == nil
	}

	model, err := client.embedded.CreateCollection(localchroma.EmbeddedCreateCollectionRequest{
		Name:         req.Name,
		TenantID:     req.Database.Tenant().Name(),
		DatabaseName: req.Database.Name(),
		GetOrCreate:  req.CreateIfNotExists,
	})
	if err != nil {
		return nil, errors.Wrap(err, "error creating collection")
	}

	overrideEF := req.embeddingFunction
	if isNewCreation {
		client.upsertCollectionState(model.ID, func(state *embeddedCollectionState) {
			state.embeddingFunction = req.embeddingFunction
			if req.Metadata != nil {
				state.metadata = req.Metadata
			}
			if req.Configuration != nil {
				state.configuration = req.Configuration
			}
			if req.Schema != nil {
				state.schema = req.Schema
			}
		})
	} else {
		overrideEF = nil
	}

	collection := client.buildEmbeddedCollection(*model, req.Database, overrideEF)
	return collection, nil
}

func (client *embeddedLocalClient) GetOrCreateCollection(ctx context.Context, name string, options ...CreateCollectionOption) (Collection, error) {
	newOptions := append([]CreateCollectionOption{WithDatabaseCreate(client.CurrentDatabase())}, options...)
	req, err := NewCreateCollectionOp(name, newOptions...)
	if err != nil {
		return nil, errors.Wrap(err, "error preparing collection get-or-create request")
	}
	if req.Name == "" {
		return nil, errors.New("collection name cannot be empty")
	}
	if req.Database == nil {
		return nil, errors.New("database cannot be nil")
	}
	if err := req.Database.Validate(); err != nil {
		return nil, errors.Wrap(err, "error validating database")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	getOptions := []GetCollectionOption{WithDatabaseGet(req.Database)}
	if req.embeddingFunction != nil {
		getOptions = append(getOptions, WithEmbeddingFunctionGet(req.embeddingFunction))
	}
	collection, getErr := client.GetCollection(ctx, req.Name, getOptions...)
	if getErr == nil {
		return collection, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	createOptions := append([]CreateCollectionOption{}, options...)
	createOptions = append(createOptions, WithIfNotExistsCreate())
	collection, createErr := client.CreateCollection(ctx, req.Name, createOptions...)
	if createErr != nil {
		return nil, errors.Wrap(stderrors.Join(getErr, createErr), "error get-or-creating collection")
	}
	return collection, nil
}

func (client *embeddedLocalClient) DeleteCollection(ctx context.Context, name string, options ...DeleteCollectionOption) error {
	newOpts := append([]DeleteCollectionOption{WithDatabaseDelete(client.CurrentDatabase())}, options...)
	req, err := NewDeleteCollectionOp(newOpts...)
	if err != nil {
		return errors.Wrap(err, "error preparing collection delete request")
	}
	if err := req.PrepareAndValidateCollectionRequest(); err != nil {
		return errors.Wrap(err, "error validating collection delete request")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	targetCollectionID := ""
	targetCollection, lookupErr := client.embedded.GetCollection(localchroma.EmbeddedGetCollectionRequest{
		Name:         name,
		TenantID:     req.Database.Tenant().Name(),
		DatabaseName: req.Database.Name(),
	})
	if lookupErr == nil && targetCollection != nil {
		targetCollectionID = targetCollection.ID
	} else {
		cached := client.cachedCollectionByName(name)
		if cached != nil && cached.Database() != nil && cached.Database().Tenant() != nil &&
			cached.Database().Name() == req.Database.Name() &&
			cached.Database().Tenant().Name() == req.Database.Tenant().Name() {
			targetCollectionID = cached.ID()
		}
	}

	err = client.embedded.DeleteCollection(localchroma.EmbeddedDeleteCollectionRequest{
		Name:         name,
		TenantID:     req.Database.Tenant().Name(),
		DatabaseName: req.Database.Name(),
	})
	if err != nil {
		return errors.Wrapf(err, "error deleting collection %s", name)
	}
	if targetCollectionID != "" {
		client.deleteCollectionState(targetCollectionID)
	}
	client.state.localDeleteCollectionFromCache(name)
	return nil
}

func (client *embeddedLocalClient) GetCollection(ctx context.Context, name string, opts ...GetCollectionOption) (Collection, error) {
	newOpts := append([]GetCollectionOption{WithCollectionNameGet(name), WithDatabaseGet(client.CurrentDatabase())}, opts...)
	req, err := NewGetCollectionOp(newOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "error preparing collection get request")
	}
	if err := req.PrepareAndValidateCollectionRequest(); err != nil {
		return nil, errors.Wrap(err, "error validating collection get request")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	model, err := client.embedded.GetCollection(localchroma.EmbeddedGetCollectionRequest{
		Name:         req.name,
		TenantID:     req.Database.Tenant().Name(),
		DatabaseName: req.Database.Name(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error getting collection")
	}

	if req.embeddingFunction != nil {
		client.upsertCollectionState(model.ID, func(state *embeddedCollectionState) {
			state.embeddingFunction = req.embeddingFunction
		})
	}
	collection := client.buildEmbeddedCollection(*model, req.Database, req.embeddingFunction)
	return collection, nil
}

func (client *embeddedLocalClient) CountCollections(ctx context.Context, opts ...CountCollectionsOption) (int, error) {
	newOpts := append([]CountCollectionsOption{WithDatabaseCount(client.CurrentDatabase())}, opts...)
	req, err := NewCountCollectionsOp(newOpts...)
	if err != nil {
		return 0, errors.Wrap(err, "error preparing collection count request")
	}
	if err := req.PrepareAndValidateCollectionRequest(); err != nil {
		return 0, errors.Wrap(err, "error validating collection count request")
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	count, err := client.embedded.CountCollections(localchroma.EmbeddedCountCollectionsRequest{
		TenantID:     req.Database.Tenant().Name(),
		DatabaseName: req.Database.Name(),
	})
	if err != nil {
		return 0, errors.Wrap(err, "error counting collections")
	}
	return int(count), nil
}

func intToUint32(value int, fieldName string) (uint32, error) {
	if value < 0 {
		return 0, errors.Errorf("%s must be greater than or equal to 0", fieldName)
	}
	if uint64(value) > uint64(math.MaxUint32) {
		return 0, errors.Errorf("%s cannot exceed %d", fieldName, uint64(math.MaxUint32))
	}
	return uint32(value), nil
}

func (client *embeddedLocalClient) ListCollections(ctx context.Context, opts ...ListCollectionsOption) ([]Collection, error) {
	newOpts := append([]ListCollectionsOption{WithDatabaseList(client.CurrentDatabase())}, opts...)
	req, err := NewListCollectionsOp(newOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "error preparing collection list request")
	}
	if err := req.PrepareAndValidateCollectionRequest(); err != nil {
		return nil, errors.Wrap(err, "error validating collection list request")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	limit, err := intToUint32(req.Limit(), "limit")
	if err != nil {
		return nil, err
	}
	offset, err := intToUint32(req.Offset(), "offset")
	if err != nil {
		return nil, err
	}

	models, err := client.embedded.ListCollections(localchroma.EmbeddedListCollectionsRequest{
		TenantID:     req.Database.Tenant().Name(),
		DatabaseName: req.Database.Name(),
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		return nil, errors.Wrap(err, "error listing collections")
	}

	collections := make([]Collection, 0, len(models))
	for _, model := range models {
		collections = append(collections, client.buildEmbeddedCollection(model, req.Database, nil))
	}
	return collections, nil
}

func (client *embeddedLocalClient) Close() error {
	var errs []error

	if client.state != nil {
		if err := client.state.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if client.embedded != nil {
		if err := client.embedded.Close(); err != nil {
			errs = append(errs, errors.Wrap(err, "error closing embedded local runtime"))
		}
	}
	if len(errs) > 0 {
		return stderrors.Join(errs...)
	}
	return nil
}

func (client *embeddedLocalClient) cachedCollectionByName(name string) Collection {
	if client == nil {
		return nil
	}
	return client.state.localCollectionByName(name)
}

func (client *embeddedLocalClient) renameCollectionInCache(oldName string, collection Collection) {
	if client == nil || collection == nil {
		return
	}
	client.state.localRenameCollectionInCache(oldName, collection)
}

func (client *embeddedLocalClient) deleteCollectionState(collectionID string) {
	if collectionID == "" {
		return
	}
	client.collectionStateMu.Lock()
	defer client.collectionStateMu.Unlock()
	delete(client.collectionState, collectionID)
}

func (client *embeddedLocalClient) upsertCollectionState(collectionID string, update func(state *embeddedCollectionState)) embeddedCollectionState {
	client.collectionStateMu.Lock()
	defer client.collectionStateMu.Unlock()

	state, ok := client.collectionState[collectionID]
	if !ok || state == nil {
		state = &embeddedCollectionState{}
		client.collectionState[collectionID] = state
	}
	if state.metadata == nil {
		state.metadata = NewEmptyMetadata()
	}
	if state.configuration == nil {
		state.configuration = NewCollectionConfiguration()
	}
	if update != nil {
		update(state)
	}
	if state.metadata == nil {
		state.metadata = NewEmptyMetadata()
	}
	if state.configuration == nil {
		state.configuration = NewCollectionConfiguration()
	}

	return embeddedCollectionState{
		embeddingFunction: state.embeddingFunction,
		metadata:          state.metadata,
		configuration:     state.configuration,
		schema:            state.schema,
		dimension:         state.dimension,
	}
}

func (client *embeddedLocalClient) setCollectionDimension(collectionID string, dimension int) {
	if collectionID == "" || dimension <= 0 {
		return
	}
	client.upsertCollectionState(collectionID, func(state *embeddedCollectionState) {
		state.dimension = dimension
	})
}

func (client *embeddedLocalClient) collectionDimension(collectionID string, fallback int) int {
	if collectionID == "" {
		return fallback
	}
	client.collectionStateMu.RLock()
	defer client.collectionStateMu.RUnlock()

	state, ok := client.collectionState[collectionID]
	if !ok || state == nil || state.dimension <= 0 {
		return fallback
	}
	return state.dimension
}

func (client *embeddedLocalClient) buildEmbeddedCollection(model localchroma.EmbeddedCollection, fallbackDB Database, overrideEF embeddingspkg.EmbeddingFunction) *embeddedCollection {
	tenantName := model.Tenant
	databaseName := model.Database

	if fallbackDB != nil {
		if tenantName == "" && fallbackDB.Tenant() != nil {
			tenantName = fallbackDB.Tenant().Name()
		}
		if databaseName == "" {
			databaseName = fallbackDB.Name()
		}
	}
	if tenantName == "" {
		if tenant := client.CurrentTenant(); tenant != nil {
			tenantName = tenant.Name()
		} else {
			tenantName = DefaultTenant
		}
	}
	if databaseName == "" {
		if db := client.CurrentDatabase(); db != nil {
			databaseName = db.Name()
		} else {
			databaseName = DefaultDatabase
		}
	}

	snapshot := client.upsertCollectionState(model.ID, func(state *embeddedCollectionState) {
		if overrideEF != nil {
			state.embeddingFunction = overrideEF
		}
	})
	if snapshot.embeddingFunction == nil {
		snapshot.embeddingFunction = overrideEF
	}

	tenant := NewTenant(tenantName)
	database := NewDatabase(databaseName, tenant)
	collection := &embeddedCollection{
		name:              model.Name,
		id:                model.ID,
		tenant:            tenant,
		database:          database,
		metadata:          snapshot.metadata,
		configuration:     snapshot.configuration,
		schema:            snapshot.schema,
		dimension:         snapshot.dimension,
		embeddingFunction: snapshot.embeddingFunction,
		client:            client,
	}
	client.state.localAddCollectionToCache(collection)
	return collection
}

type embeddedCollection struct {
	mu sync.RWMutex

	name          string
	id            string
	tenant        Tenant
	database      Database
	metadata      CollectionMetadata
	configuration CollectionConfiguration
	schema        *Schema
	dimension     int

	embeddingFunction embeddingspkg.EmbeddingFunction
	client            *embeddedLocalClient
}

func (c *embeddedCollection) Name() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.name
}

func (c *embeddedCollection) ID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.id
}

func (c *embeddedCollection) Tenant() Tenant {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tenant
}

func (c *embeddedCollection) Database() Database {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.database
}

func (c *embeddedCollection) Metadata() CollectionMetadata {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.metadata == nil {
		return NewEmptyMetadata()
	}
	return c.metadata
}

func (c *embeddedCollection) Dimension() int {
	c.mu.RLock()
	collectionID := c.id
	fallback := c.dimension
	c.mu.RUnlock()
	return c.client.collectionDimension(collectionID, fallback)
}

func (c *embeddedCollection) Configuration() CollectionConfiguration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.configuration == nil {
		return NewCollectionConfiguration()
	}
	return c.configuration
}

func (c *embeddedCollection) Schema() *Schema {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.schema
}

func (c *embeddedCollection) embeddingFunctionSnapshot() embeddingspkg.EmbeddingFunction {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.embeddingFunction
}

func (c *embeddedCollection) Add(ctx context.Context, opts ...CollectionAddOption) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.client.PreFlight(ctx); err != nil {
		return errors.Wrap(err, "preflight failed")
	}

	addObject, err := NewCollectionAddOp(opts...)
	if err != nil {
		return errors.Wrap(err, "failed to create add operation")
	}
	if err := addObject.PrepareAndValidate(); err != nil {
		return errors.Wrap(err, "failed to validate add operation")
	}
	if err := c.client.state.Satisfies(addObject, len(addObject.Ids), "documents"); err != nil {
		return errors.Wrap(err, "failed to satisfy add operation")
	}
	embeddingFunction := c.embeddingFunctionSnapshot()
	if err := addObject.EmbedData(ctx, embeddingFunction); err != nil {
		return errors.Wrap(err, "failed to embed data")
	}

	vectors, err := embeddingsAnyToFloat32Matrix(addObject.Embeddings)
	if err != nil {
		return errors.Wrap(err, "failed to convert embeddings")
	}
	metadatas, err := documentMetadatasToMaps(addObject.Metadatas)
	if err != nil {
		return errors.Wrap(err, "failed to convert metadatas")
	}
	if err := c.client.embedded.Add(localchroma.EmbeddedAddRequest{
		CollectionID: c.id,
		IDs:          documentIDsToStrings(addObject.Ids),
		Embeddings:   vectors,
		Documents:    documentsToStrings(addObject.Documents),
		Metadatas:    metadatas,
		TenantID:     c.tenant.Name(),
		DatabaseName: c.database.Name(),
	}); err != nil {
		return errors.Wrap(err, "error adding records")
	}

	if len(vectors) > 0 && c.Dimension() == 0 {
		dimension := len(vectors[0])
		if dimension > 0 {
			c.client.setCollectionDimension(c.id, dimension)
		}
	}
	return nil
}

func (c *embeddedCollection) Upsert(ctx context.Context, opts ...CollectionAddOption) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.client.PreFlight(ctx); err != nil {
		return errors.Wrap(err, "preflight failed")
	}

	upsertObject, err := NewCollectionAddOp(opts...)
	if err != nil {
		return errors.Wrap(err, "failed to create upsert operation")
	}
	if err := upsertObject.PrepareAndValidate(); err != nil {
		return errors.Wrap(err, "failed to validate upsert operation")
	}
	if err := c.client.state.Satisfies(upsertObject, len(upsertObject.Ids), "documents"); err != nil {
		return errors.Wrap(err, "failed to satisfy upsert operation")
	}
	embeddingFunction := c.embeddingFunctionSnapshot()
	if err := upsertObject.EmbedData(ctx, embeddingFunction); err != nil {
		return errors.Wrap(err, "failed to embed data")
	}

	vectors, err := embeddingsAnyToFloat32Matrix(upsertObject.Embeddings)
	if err != nil {
		return errors.Wrap(err, "failed to convert embeddings")
	}
	metadatas, err := documentMetadatasToMaps(upsertObject.Metadatas)
	if err != nil {
		return errors.Wrap(err, "failed to convert metadatas")
	}
	if err := c.client.embedded.UpsertRecords(localchroma.EmbeddedUpsertRecordsRequest{
		CollectionID: c.id,
		IDs:          documentIDsToStrings(upsertObject.Ids),
		Embeddings:   vectors,
		Documents:    documentsToStrings(upsertObject.Documents),
		Metadatas:    metadatas,
		TenantID:     c.tenant.Name(),
		DatabaseName: c.database.Name(),
	}); err != nil {
		return errors.Wrap(err, "error upserting records")
	}

	if len(vectors) > 0 && c.Dimension() == 0 {
		dimension := len(vectors[0])
		if dimension > 0 {
			c.client.setCollectionDimension(c.id, dimension)
		}
	}
	return nil
}

func (c *embeddedCollection) Update(ctx context.Context, opts ...CollectionUpdateOption) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.client.PreFlight(ctx); err != nil {
		return errors.Wrap(err, "preflight failed")
	}

	updateObject, err := NewCollectionUpdateOp(opts...)
	if err != nil {
		return errors.Wrap(err, "failed to create update operation")
	}
	if err := updateObject.PrepareAndValidate(); err != nil {
		return errors.Wrap(err, "failed to validate update operation")
	}
	if err := c.client.state.Satisfies(updateObject, len(updateObject.Ids), "documents"); err != nil {
		return errors.Wrap(err, "failed to satisfy update operation")
	}
	embeddingFunction := c.embeddingFunctionSnapshot()
	if err := updateObject.EmbedData(ctx, embeddingFunction); err != nil {
		return errors.Wrap(err, "failed to embed data")
	}

	vectors, err := embeddingsAnyToFloat32Matrix(updateObject.Embeddings)
	if err != nil {
		return errors.Wrap(err, "failed to convert embeddings")
	}
	metadatas, err := documentMetadatasToMaps(updateObject.Metadatas)
	if err != nil {
		return errors.Wrap(err, "failed to convert metadatas")
	}
	if err := c.client.embedded.UpdateRecords(localchroma.EmbeddedUpdateRecordsRequest{
		CollectionID: c.id,
		IDs:          documentIDsToStrings(updateObject.Ids),
		Embeddings:   vectors,
		Documents:    documentsToStrings(updateObject.Documents),
		Metadatas:    metadatas,
		TenantID:     c.tenant.Name(),
		DatabaseName: c.database.Name(),
	}); err != nil {
		return errors.Wrap(err, "error updating records")
	}

	if len(vectors) > 0 && c.Dimension() == 0 {
		dimension := len(vectors[0])
		if dimension > 0 {
			c.client.setCollectionDimension(c.id, dimension)
		}
	}
	return nil
}

func (c *embeddedCollection) Delete(ctx context.Context, opts ...CollectionDeleteOption) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.client.PreFlight(ctx); err != nil {
		return errors.Wrap(err, "preflight failed")
	}

	deleteObject, err := NewCollectionDeleteOp(opts...)
	if err != nil {
		return errors.Wrap(err, "failed to create delete operation")
	}
	if err := deleteObject.PrepareAndValidate(); err != nil {
		return errors.Wrap(err, "failed to validate delete operation")
	}
	if err := c.client.state.Satisfies(deleteObject, len(deleteObject.Ids), "documents"); err != nil {
		return errors.Wrap(err, "failed to satisfy delete operation")
	}

	where, err := whereFilterToMap(deleteObject.Where)
	if err != nil {
		return errors.Wrap(err, "failed to convert where filter")
	}
	whereDocument, err := whereDocumentFilterToMap(deleteObject.WhereDocument)
	if err != nil {
		return errors.Wrap(err, "failed to convert whereDocument filter")
	}

	return c.client.embedded.DeleteRecords(localchroma.EmbeddedDeleteRecordsRequest{
		CollectionID:  c.id,
		IDs:           documentIDsToStrings(deleteObject.Ids),
		Where:         where,
		WhereDocument: whereDocument,
		TenantID:      c.tenant.Name(),
		DatabaseName:  c.database.Name(),
	})
}

func (c *embeddedCollection) Count(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	count, err := c.client.embedded.CountRecords(localchroma.EmbeddedCountRecordsRequest{
		CollectionID: c.id,
		TenantID:     c.tenant.Name(),
		DatabaseName: c.database.Name(),
	})
	if err != nil {
		return 0, errors.Wrap(err, "error counting records")
	}
	return int(count), nil
}

func (c *embeddedCollection) ModifyName(ctx context.Context, newName string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return errors.New("newName cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	c.mu.Lock()
	oldName := c.name
	if err := c.client.embedded.UpdateCollection(localchroma.EmbeddedUpdateCollectionRequest{
		CollectionID: c.id,
		NewName:      newName,
		DatabaseName: c.database.Name(),
	}); err != nil {
		c.mu.Unlock()
		return errors.Wrap(err, "error renaming collection")
	}
	c.name = newName
	c.mu.Unlock()
	c.client.renameCollectionInCache(oldName, c)
	return nil
}

func (c *embeddedCollection) ModifyMetadata(ctx context.Context, newMetadata CollectionMetadata) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if newMetadata == nil {
		return errors.New("newMetadata cannot be nil")
	}
	return errors.New("embedded local mode does not support persisting collection metadata updates")
}

func (c *embeddedCollection) ModifyConfiguration(_ context.Context, newConfig *UpdateCollectionConfiguration) error {
	if newConfig == nil {
		return errors.New("newConfig cannot be nil")
	}
	if err := newConfig.Validate(); err != nil {
		return err
	}
	return errors.New("embedded local mode does not support persisting collection configuration updates")
}

func (c *embeddedCollection) Get(ctx context.Context, opts ...CollectionGetOption) (GetResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	getObject, err := NewCollectionGetOp(opts...)
	if err != nil {
		return nil, err
	}
	if err := getObject.PrepareAndValidate(); err != nil {
		return nil, err
	}

	where, err := whereFilterToMap(getObject.Where)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert where filter")
	}
	whereDocument, err := whereDocumentFilterToMap(getObject.WhereDocument)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert whereDocument filter")
	}
	limit, err := intToUint32(getObject.Limit, "limit")
	if err != nil {
		return nil, err
	}
	offset, err := intToUint32(getObject.Offset, "offset")
	if err != nil {
		return nil, err
	}

	response, err := c.client.embedded.GetRecords(localchroma.EmbeddedGetRecordsRequest{
		CollectionID:  c.id,
		IDs:           documentIDsToStrings(getObject.Ids),
		Where:         where,
		WhereDocument: whereDocument,
		Limit:         limit,
		Offset:        offset,
		Include:       sanitizeEmbeddedGetIncludes(getObject.Include),
		TenantID:      c.tenant.Name(),
		DatabaseName:  c.database.Name(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error getting records")
	}
	return embeddedGetRecordsToGetResult(response, getObject.Include)
}

func (c *embeddedCollection) Query(ctx context.Context, opts ...CollectionQueryOption) (QueryResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	queryObject, err := NewCollectionQueryOp(opts...)
	if err != nil {
		return nil, errors.Wrap(err, "error creating query operation")
	}
	if err := queryObject.PrepareAndValidate(); err != nil {
		return nil, errors.Wrap(err, "error validating query operation")
	}
	embeddingFunction := c.embeddingFunctionSnapshot()
	if err := queryObject.EmbedData(ctx, embeddingFunction); err != nil {
		return nil, errors.Wrap(err, "failed to embed data")
	}

	where, err := whereFilterToMap(queryObject.Where)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert where filter")
	}
	whereDocument, err := whereDocumentFilterToMap(queryObject.WhereDocument)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert whereDocument filter")
	}

	queryEmbeddings, err := embeddingsToFloat32Matrix(queryObject.QueryEmbeddings)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert query embeddings")
	}
	nResults, err := intToUint32(queryObject.NResults, "nResults")
	if err != nil {
		return nil, err
	}

	queryResponse, err := c.client.embedded.Query(localchroma.EmbeddedQueryRequest{
		CollectionID:    c.id,
		QueryEmbeddings: queryEmbeddings,
		NResults:        nResults,
		IDs:             documentIDsToStrings(queryObject.Ids),
		Where:           where,
		WhereDocument:   whereDocument,
		Include:         sanitizeEmbeddedQueryIncludes(queryObject.Include),
		TenantID:        c.tenant.Name(),
		DatabaseName:    c.database.Name(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	effectiveIncludes := queryObject.Include
	if len(effectiveIncludes) == 0 {
		effectiveIncludes = []Include{IncludeDocuments, IncludeMetadatas, IncludeDistances}
	}

	result := &QueryResultImpl{
		IDLists: stringGroupsToDocumentIDGroups(queryResponse.IDs),
		Include: effectiveIncludes,
	}

	includes := includeSet(effectiveIncludes)
	needDocs := includes[IncludeDocuments]
	needMetadatas := includes[IncludeMetadatas]
	needEmbeddings := includes[IncludeEmbeddings]
	needDistances := includes[IncludeDistances]
	if !needDocs && !needMetadatas && !needEmbeddings && !needDistances {
		return result, nil
	}

	recordInclude := make([]Include, 0, 3)
	if needDocs {
		recordInclude = append(recordInclude, IncludeDocuments)
	}
	if needMetadatas {
		recordInclude = append(recordInclude, IncludeMetadatas)
	}
	if needEmbeddings || needDistances {
		recordInclude = append(recordInclude, IncludeEmbeddings)
	}

	if needDocs {
		result.DocumentsLists = make([]Documents, 0, len(queryResponse.IDs))
	}
	if needMetadatas {
		result.MetadatasLists = make([]DocumentMetadatas, 0, len(queryResponse.IDs))
	}
	if needEmbeddings {
		result.EmbeddingsLists = make([]embeddingspkg.Embeddings, 0, len(queryResponse.IDs))
	}
	if needDistances {
		result.DistancesLists = make([]embeddingspkg.Distances, 0, len(queryResponse.IDs))
		if len(queryResponse.IDs) > len(queryEmbeddings) {
			return nil, errors.Errorf(
				"query response returned %d id groups but only %d query embeddings were provided",
				len(queryResponse.IDs),
				len(queryEmbeddings),
			)
		}
	}

	distanceMetric := c.queryDistanceMetric()

	for groupIdx, group := range queryResponse.IDs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(group) == 0 {
			if needDocs {
				result.DocumentsLists = append(result.DocumentsLists, Documents{})
			}
			if needMetadatas {
				result.MetadatasLists = append(result.MetadatasLists, DocumentMetadatas{})
			}
			if needEmbeddings {
				result.EmbeddingsLists = append(result.EmbeddingsLists, embeddingspkg.Embeddings{})
			}
			if needDistances {
				result.DistancesLists = append(result.DistancesLists, embeddingspkg.Distances{})
			}
			continue
		}

		records, err := c.client.embedded.GetRecords(localchroma.EmbeddedGetRecordsRequest{
			CollectionID: c.id,
			IDs:          group,
			Include:      sanitizeEmbeddedGetIncludes(recordInclude),
			TenantID:     c.tenant.Name(),
			DatabaseName: c.database.Name(),
		})
		if err != nil {
			return nil, errors.Wrap(err, "error loading query projections")
		}

		index := make(map[string]int, len(records.IDs))
		for i, id := range records.IDs {
			index[id] = i
		}

		if needDocs {
			docs := make(Documents, len(group))
			for i, id := range group {
				recordIdx, ok := index[id]
				if !ok || recordIdx >= len(records.Documents) || records.Documents[recordIdx] == nil {
					continue
				}
				doc := NewTextDocument(*records.Documents[recordIdx])
				docs[i] = doc
			}
			result.DocumentsLists = append(result.DocumentsLists, docs)
		}

		if needMetadatas {
			metadatas := make(DocumentMetadatas, len(group))
			for i, id := range group {
				recordIdx, ok := index[id]
				if !ok || recordIdx >= len(records.Metadatas) || records.Metadatas[recordIdx] == nil {
					continue
				}
				metadata, err := NewDocumentMetadataFromMap(records.Metadatas[recordIdx])
				if err != nil {
					return nil, errors.Wrap(err, "error decoding query metadata")
				}
				metadatas[i] = metadata
			}
			result.MetadatasLists = append(result.MetadatasLists, metadatas)
		}

		if needEmbeddings {
			embeddingsGroup := make(embeddingspkg.Embeddings, len(group))
			for i, id := range group {
				recordIdx, ok := index[id]
				if !ok || recordIdx >= len(records.Embeddings) {
					continue
				}
				embeddingsGroup[i] = embeddingspkg.NewEmbeddingFromFloat32(records.Embeddings[recordIdx])
			}
			result.EmbeddingsLists = append(result.EmbeddingsLists, embeddingsGroup)
		}

		if needDistances {
			distances := make(embeddingspkg.Distances, len(group))
			queryVector := queryEmbeddings[groupIdx]
			for i, id := range group {
				recordIdx, ok := index[id]
				if !ok || recordIdx >= len(records.Embeddings) {
					continue
				}
				distance, err := computeEmbeddedDistance(distanceMetric, queryVector, records.Embeddings[recordIdx])
				if err != nil {
					return nil, errors.Wrap(err, "error computing query distances")
				}
				distances[i] = distance
			}
			result.DistancesLists = append(result.DistancesLists, distances)
		}
	}

	return result, nil
}

func (c *embeddedCollection) Search(_ context.Context, _ ...SearchCollectionOption) (SearchResult, error) {
	return nil, errors.New("search is not supported in embedded local mode")
}

func (c *embeddedCollection) Fork(ctx context.Context, newName string) (Collection, error) {
	if strings.TrimSpace(newName) == "" {
		return nil, errors.New("newName cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	forked, err := c.client.embedded.ForkCollection(localchroma.EmbeddedForkCollectionRequest{
		SourceCollectionID:   c.id,
		TargetCollectionName: strings.TrimSpace(newName),
		TenantID:             c.tenant.Name(),
		DatabaseName:         c.database.Name(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error forking collection")
	}
	c.mu.RLock()
	embeddingFunction := c.embeddingFunction
	metadata := c.metadata
	configuration := c.configuration
	schema := c.schema
	database := c.database
	c.mu.RUnlock()

	c.client.upsertCollectionState(forked.ID, func(state *embeddedCollectionState) {
		state.embeddingFunction = embeddingFunction
		state.metadata = metadata
		state.configuration = configuration
		state.schema = schema
		state.dimension = c.Dimension()
	})

	forkedCollection := c.client.buildEmbeddedCollection(*forked, database, embeddingFunction)
	return forkedCollection, nil
}

func (c *embeddedCollection) IndexingStatus(ctx context.Context) (*IndexingStatus, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	status, err := c.client.embedded.IndexingStatus(localchroma.EmbeddedIndexingStatusRequest{
		CollectionID: c.id,
		DatabaseName: c.database.Name(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error getting indexing status")
	}
	return &IndexingStatus{
		NumIndexedOps:      status.NumIndexedOps,
		NumUnindexedOps:    status.NumUnindexedOps,
		TotalOps:           status.TotalOps,
		OpIndexingProgress: float64(status.OpIndexingProgress),
	}, nil
}

func (c *embeddedCollection) Close() error {
	embeddingFunction := c.embeddingFunctionSnapshot()
	if embeddingFunction != nil {
		if closer, ok := embeddingFunction.(io.Closer); ok {
			return closer.Close()
		}
	}
	return nil
}

func documentIDsToStrings(ids []DocumentID) []string {
	if len(ids) == 0 {
		return nil
	}
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = string(id)
	}
	return result
}

func stringSliceToDocumentIDs(ids []string) DocumentIDs {
	if len(ids) == 0 {
		return nil
	}
	result := make(DocumentIDs, len(ids))
	for i, id := range ids {
		result[i] = DocumentID(id)
	}
	return result
}

func stringGroupsToDocumentIDGroups(groups [][]string) []DocumentIDs {
	if len(groups) == 0 {
		return nil
	}
	result := make([]DocumentIDs, len(groups))
	for i, group := range groups {
		result[i] = stringSliceToDocumentIDs(group)
	}
	return result
}

func documentsToStrings(documents []Document) []string {
	if len(documents) == 0 {
		return nil
	}
	result := make([]string, len(documents))
	for i, document := range documents {
		if document == nil {
			continue
		}
		result[i] = document.ContentString()
	}
	return result
}

func documentMetadatasToMaps(metadatas []DocumentMetadata) ([]map[string]any, error) {
	if len(metadatas) == 0 {
		return nil, nil
	}
	result := make([]map[string]any, len(metadatas))
	for i, metadata := range metadatas {
		if metadata == nil {
			continue
		}
		payload, err := json.Marshal(metadata)
		if err != nil {
			return nil, errors.Wrap(err, "error marshaling metadata")
		}
		var decoded map[string]any
		if err := json.Unmarshal(payload, &decoded); err != nil {
			return nil, errors.Wrap(err, "error unmarshaling metadata")
		}
		result[i] = decoded
	}
	return result, nil
}

func embeddingsToFloat32Matrix(input []embeddingspkg.Embedding) ([][]float32, error) {
	if len(input) == 0 {
		return nil, nil
	}
	result := make([][]float32, len(input))
	for i, embedding := range input {
		if embedding == nil {
			return nil, errors.Errorf("embedding at index %d cannot be nil", i)
		}
		content := embedding.ContentAsFloat32()
		if len(content) == 0 {
			return nil, errors.Errorf("embedding at index %d cannot be empty", i)
		}
		copied := make([]float32, len(content))
		copy(copied, content)
		result[i] = copied
	}
	return result, nil
}

func embeddingsAnyToFloat32Matrix(values []any) ([][]float32, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make([][]float32, len(values))
	for i, value := range values {
		vector, err := anyToFloat32Slice(value)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid embedding at index %d", i)
		}
		result[i] = vector
	}
	return result, nil
}

func anyToFloat32Slice(value any) ([]float32, error) {
	switch typed := value.(type) {
	case nil:
		return nil, errors.New("embedding cannot be nil")
	case embeddingspkg.Embedding:
		content := typed.ContentAsFloat32()
		copied := make([]float32, len(content))
		copy(copied, content)
		return copied, nil
	case []float32:
		copied := make([]float32, len(typed))
		copy(copied, typed)
		return copied, nil
	case []float64:
		result := make([]float32, len(typed))
		for i, value := range typed {
			result[i] = float32(value)
		}
		return result, nil
	case []int:
		result := make([]float32, len(typed))
		for i, value := range typed {
			result[i] = float32(value)
		}
		return result, nil
	case []int32:
		result := make([]float32, len(typed))
		for i, value := range typed {
			result[i] = float32(value)
		}
		return result, nil
	case []int64:
		result := make([]float32, len(typed))
		for i, value := range typed {
			result[i] = float32(value)
		}
		return result, nil
	case []any:
		result := make([]float32, len(typed))
		for i, element := range typed {
			switch v := element.(type) {
			case float32:
				result[i] = v
			case float64:
				result[i] = float32(v)
			case int:
				result[i] = float32(v)
			case int32:
				result[i] = float32(v)
			case int64:
				result[i] = float32(v)
			case json.Number:
				f, err := v.Float64()
				if err != nil {
					return nil, errors.Wrap(err, "invalid numeric embedding value")
				}
				result[i] = float32(f)
			default:
				return nil, errors.Errorf("unsupported embedding element type %T", element)
			}
		}
		return result, nil
	default:
		return nil, errors.Errorf("unsupported embedding type %T", value)
	}
}

func whereFilterToMap(where WhereFilter) (map[string]any, error) {
	if where == nil {
		return nil, nil
	}
	payload, err := where.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func whereDocumentFilterToMap(whereDocument WhereDocumentFilter) (map[string]any, error) {
	if whereDocument == nil {
		return nil, nil
	}
	payload, err := whereDocument.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func includeSet(includes []Include) map[Include]bool {
	set := make(map[Include]bool, len(includes))
	for _, include := range includes {
		set[include] = true
	}
	return set
}

func sanitizeEmbeddedGetIncludes(includes []Include) []string {
	if len(includes) == 0 {
		return nil
	}
	filtered := make([]string, 0, len(includes))
	for _, include := range includes {
		switch include {
		case IncludeDocuments, IncludeMetadatas, IncludeEmbeddings, IncludeURIs:
			filtered = append(filtered, string(include))
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func sanitizeEmbeddedQueryIncludes(includes []Include) []string {
	if len(includes) == 0 {
		return nil
	}
	filtered := make([]string, 0, len(includes))
	for _, include := range includes {
		switch include {
		case IncludeDocuments, IncludeMetadatas, IncludeEmbeddings, IncludeURIs, IncludeDistances:
			filtered = append(filtered, string(include))
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func (c *embeddedCollection) queryDistanceMetric() embeddingspkg.DistanceMetric {
	if c == nil {
		return embeddingspkg.L2
	}
	c.mu.RLock()
	metadata := c.metadata
	embeddingFunction := c.embeddingFunction
	c.mu.RUnlock()
	if metadata != nil {
		if value, ok := metadata.GetString(HNSWSpace); ok {
			switch metric := embeddingspkg.DistanceMetric(strings.ToLower(strings.TrimSpace(value))); metric {
			case embeddingspkg.L2, embeddingspkg.COSINE, embeddingspkg.IP:
				return metric
			}
		}
	}
	if embeddingFunction != nil {
		switch metric := embeddingFunction.DefaultSpace(); metric {
		case embeddingspkg.L2, embeddingspkg.COSINE, embeddingspkg.IP:
			return metric
		}
	}
	return embeddingspkg.L2
}

func computeEmbeddedDistance(metric embeddingspkg.DistanceMetric, queryVector []float32, resultVector []float32) (embeddingspkg.Distance, error) {
	if len(queryVector) == 0 || len(resultVector) == 0 {
		return 0, nil
	}
	if len(queryVector) != len(resultVector) {
		return 0, errors.Errorf(
			"embedding dimension mismatch: query=%d result=%d",
			len(queryVector),
			len(resultVector),
		)
	}

	switch metric {
	case embeddingspkg.COSINE:
		var dot, queryNorm, resultNorm float64
		for i := range queryVector {
			q := float64(queryVector[i])
			r := float64(resultVector[i])
			dot += q * r
			queryNorm += q * q
			resultNorm += r * r
		}
		if queryNorm == 0 || resultNorm == 0 {
			return embeddingspkg.Distance(1.0), nil
		}
		return embeddingspkg.Distance(1.0 - dot/(math.Sqrt(queryNorm)*math.Sqrt(resultNorm))), nil

	case embeddingspkg.IP:
		var dot float64
		for i := range queryVector {
			dot += float64(queryVector[i]) * float64(resultVector[i])
		}
		return embeddingspkg.Distance(1.0 - dot), nil

	default:
		var sum float64
		for i := range queryVector {
			delta := float64(queryVector[i] - resultVector[i])
			sum += delta * delta
		}
		return embeddingspkg.Distance(sum), nil
	}
}

func embeddedGetRecordsToGetResult(response *localchroma.EmbeddedGetRecordsResponse, include []Include) (GetResult, error) {
	if response == nil {
		return nil, errors.New("embedded get response cannot be nil")
	}

	result := &GetResultImpl{
		Ids:     stringSliceToDocumentIDs(response.IDs),
		Include: include,
	}

	if len(response.Documents) > 0 {
		documents := make(Documents, len(response.Documents))
		for i, document := range response.Documents {
			if document == nil {
				continue
			}
			documents[i] = NewTextDocument(*document)
		}
		result.Documents = documents
	}

	if len(response.Metadatas) > 0 {
		metadatas := make(DocumentMetadatas, len(response.Metadatas))
		for i, metadata := range response.Metadatas {
			if metadata == nil {
				continue
			}
			decoded, err := NewDocumentMetadataFromMap(metadata)
			if err != nil {
				return nil, errors.Wrap(err, "error decoding metadata")
			}
			metadatas[i] = decoded
		}
		result.Metadatas = metadatas
	}

	if len(response.Embeddings) > 0 {
		embeddingsList, err := embeddingspkg.NewEmbeddingsFromFloat32(response.Embeddings)
		if err != nil {
			return nil, errors.Wrap(err, "error decoding embeddings")
		}
		result.Embeddings = embeddingsList
	}

	return result, nil
}
