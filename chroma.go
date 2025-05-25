package chromago

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/Masterminds/semver" //nolint:gci
	"github.com/guiperry/chroma-go_cerebras/collection"
	chhttp "github.com/guiperry/chroma-go_cerebras/pkg/commons/http"
	defaultef "github.com/guiperry/chroma-go_cerebras/pkg/embeddings/default_ef"
	openapiclient "github.com/guiperry/chroma-go_cerebras/swagger"
	"github.com/guiperry/chroma-go_cerebras/types"
)

type ClientConfiguration struct {
	BasePath          string                  `json:"basePath,omitempty"`
	DefaultHeaders    map[string]string       `json:"defaultHeader,omitempty"`
	EmbeddingFunction types.EmbeddingFunction `json:"embeddingFunction,omitempty"`
}

func APIEmbeddingToEmbedding(embedding openapiclient.EmbeddingsInner) *types.Embedding {
	switch {
	case embedding.ArrayOfInt32 != nil:
		return types.NewEmbeddingFromInt32(*embedding.ArrayOfInt32)
	case embedding.ArrayOfFloat32 != nil:
		return types.NewEmbeddingFromFloat32(*embedding.ArrayOfFloat32)
	default:
		return &types.Embedding{}
	}
}

func APIEmbeddingsToEmbeddings(embeddings []openapiclient.EmbeddingsInner) []*types.Embedding {
	result := make([]*types.Embedding, 0)
	for _, v := range embeddings {
		result = append(result, APIEmbeddingToEmbedding(v))
	}
	return result
}

// Client represents the ChromaDB Client
type Client struct {
	ApiClient          *openapiclient.APIClient //nolint
	Tenant             string
	Database           string
	APIVersion         semver.Version
	preFlightConfig    map[string]interface{}
	preFlightCompleted bool
	apiConfiguration   *openapiclient.Configuration
	httpTransport      *http.Transport
	userHTTPClient     *http.Client
	BasePath           string
	activeCollections  []*Collection
	timeout            time.Duration
}

type ClientOption func(p *Client) error

func WithTenant(tenant string) ClientOption {
	return func(c *Client) error {
		// TODO validate here?
		c.Tenant = tenant
		return nil
	}
}

// WithBasePath sets the base path for the client. The base path must be a valid URL.
func WithBasePath(basePath string) ClientOption {
	return func(c *Client) error {
		if basePath == "" {
			return fmt.Errorf("basePath cannot be empty")
		}
		if _, err := url.ParseRequestURI(basePath); err != nil {
			return fmt.Errorf("invalid basePath URL: %s", err)
		}
		c.BasePath = basePath
		return nil
	}
}

func WithDatabase(database string) ClientOption {
	return func(c *Client) error {
		// TODO validate here?
		c.Database = database
		return nil
	}
}

func WithDebug(debug bool) ClientOption {
	return func(c *Client) error {
		if c.apiConfiguration == nil {
			c.apiConfiguration = openapiclient.NewConfiguration()
		}
		c.apiConfiguration.Debug = debug
		return nil
	}
}
func WithDefaultHeaders(headers map[string]string) ClientOption {
	return func(c *Client) error {
		if c.apiConfiguration == nil {
			c.apiConfiguration = openapiclient.NewConfiguration()
		}
		c.apiConfiguration.DefaultHeader = headers
		return nil
	}
}

func WithAuth(provider types.CredentialsProvider) ClientOption {
	return func(c *Client) error {
		if c == nil {
			return fmt.Errorf("client is nil")
		}
		if c.apiConfiguration == nil {
			return fmt.Errorf("api configuration is nil")
		}
		return provider.Authenticate(c.apiConfiguration)
	}
}

// WithSSLCert adds a custom SSL certificate to the client. The certificate must be in PEM format. The Option can be added multiple times to add multiple certificates. The option is mutually exclusive with WithHttpClient.
func WithSSLCert(certPath string) ClientOption {
	return func(c *Client) error {
		if _, err := os.Stat(certPath); certPath == "" || err != nil {
			return fmt.Errorf("invalid cert path %v", err)
		}
		if c.httpTransport == nil {
			c.httpTransport = &http.Transport{}
		}
		cert, err := os.ReadFile(certPath)
		if err != nil {
			return err
		}

		// Create or reuse existing a certificate pool and add the custom certificate
		var certPool *x509.CertPool
		switch {
		case c.httpTransport.TLSClientConfig == nil:
			c.httpTransport.TLSClientConfig = &tls.Config{}
			certPool = x509.NewCertPool()
			c.httpTransport.TLSClientConfig.RootCAs = certPool
		case c.httpTransport.TLSClientConfig.RootCAs == nil:
			certPool = x509.NewCertPool()
			c.httpTransport.TLSClientConfig.RootCAs = certPool
		default:
			certPool = c.httpTransport.TLSClientConfig.RootCAs
		}
		if ok := certPool.AppendCertsFromPEM(cert); !ok {
			return fmt.Errorf("failed to append cert to pool")
		}
		c.httpTransport.TLSClientConfig.RootCAs = certPool
		return nil
	}
}

// WithInsecure disables SSL certificate verification. This option is not recommended for production use. The option is mutually exclusive with WithHttpClient.
func WithInsecure() ClientOption {
	return func(c *Client) error {
		if c.httpTransport == nil {
			c.httpTransport = &http.Transport{}
		}
		if c.httpTransport.TLSClientConfig == nil {
			c.httpTransport.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		} else {
			c.httpTransport.TLSClientConfig.InsecureSkipVerify = true
		}
		return nil
	}
}

// WithHTTPClient sets a custom http.Client for the client. The option is mutually exclusive with WithSSLCert and WithIgnoreSSLCert.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) error {
		if client == nil {
			return fmt.Errorf("client cannot be nil")
		}
		c.userHTTPClient = client
		return nil
	}
}

// WithTimeout sets the timeout for the client
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) error {
		c.timeout = timeout
		return nil
	}
}

func applyOptions(c *Client, options ...ClientOption) error {
	for _, opt := range options {
		if err := opt(c); err != nil {
			return err
		}
	}
	return nil
}

func NewClient(options ...ClientOption) (*Client, error) {
	c := &Client{
		Tenant:            types.DefaultTenant,
		Database:          types.DefaultDatabase,
		apiConfiguration:  openapiclient.NewConfiguration(),
		httpTransport:     &http.Transport{TLSClientConfig: &tls.Config{}},
		BasePath:          "http://localhost:8000",
		activeCollections: make([]*Collection, 0),
		timeout:           types.DefaultTimeout,
	}

	err := applyOptions(c, options...)
	if err != nil {
		return nil, err
	}
	c.apiConfiguration.Servers = openapiclient.ServerConfigurations{
		{
			URL:         c.BasePath,
			Description: "No description provided",
		},
	}
	if c.userHTTPClient != nil {
		c.apiConfiguration.HTTPClient = c.userHTTPClient
	} else {
		c.apiConfiguration.HTTPClient = &http.Client{
			Transport: c.httpTransport,
		}
	}
	c.ApiClient = openapiclient.NewAPIClient(c.apiConfiguration)
	return c, nil
}

func (c *Client) SetTenant(tenant string) {
	c.Tenant = tenant
}

func (c *Client) SetDatabase(database string) {
	c.Database = database
}

func (c *Client) preFlightChecks(ctx context.Context) error {
	if c.preFlightCompleted {
		return nil
	}
	_version, rawResp, err := c.ApiClient.DefaultApi.Version(ctx).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return chErr
	}
	version, err := semver.NewVersion(strings.ReplaceAll(_version, `"`, ""))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	c.APIVersion = *version
	multiTenantAPIVersion, _ := semver.NewConstraint(">=0.4.15")

	if multiTenantAPIVersion.Check(&c.APIVersion) {
		_, err := c.GetTenant(ctx, c.Tenant)
		if err != nil {
			return err
		}
		_, err = c.GetDatabase(ctx, c.Database, &c.Tenant)
		if err != nil {
			return err
		}
		preFlightCfg, err := c.PreflightChecks(ctx)
		if err != nil {
			return err
		}
		c.preFlightConfig = preFlightCfg
	}

	c.preFlightCompleted = true
	return nil
}

// GetCollection returns an instance of a collection object which can be used to interact with the collection data.
func (c *Client) GetCollection(ctx context.Context, collectionName string, embeddingFunction types.EmbeddingFunction) (*Collection, error) {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	err := c.preFlightChecks(ctx)
	if err != nil {
		return nil, err
	}
	tenantName := types.DefaultTenant
	databaseName := types.DefaultDatabase
	col, rawResp, err := c.ApiClient.DefaultApi.GetCollection(ctx, collectionName).Tenant(c.Tenant).Database(c.Database).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return nil, chErr
	}
	if embeddingFunction == nil {
		ef, _, err := defaultef.NewDefaultEmbeddingFunction()
		if err != nil {
			return nil, err
		}
		embeddingFunction = types.NewV2EmbeddingFunctionAdapter(ef)
	}
	return NewCollection(c, col.Id, col.Name, getMetadataFromAPI(col.Metadata), embeddingFunction, tenantName, databaseName), nil
}

// Heartbeat checks whether the Chroma server is up and running returns a map[string]float32 with the current server timestamp
func (c *Client) Heartbeat(ctx context.Context) (map[string]float32, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, rawResp, err := c.ApiClient.DefaultApi.Heartbeat(ctx).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return nil, chErr
	}
	return resp, err
}

func GetStringTypeOfEmbeddingFunction(ef types.EmbeddingFunction) string {
	if ef == nil {
		return ""
	}
	typ := reflect.TypeOf(ef)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem() // Dereference if it's a pointer
	}
	return typ.String()
}

// CreateTenant creates a new tenant with the given name, fails if the tenant already exists
func (c *Client) CreateTenant(ctx context.Context, tenantName string) (*openapiclient.Tenant, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, rawResp, err := c.ApiClient.DefaultApi.CreateTenant(ctx).CreateTenant(openapiclient.CreateTenant{Name: tenantName}).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return nil, chErr
	}
	return resp, err
}

// GetTenant returns the tenant with the given name, fails if the tenant does not exist
func (c *Client) GetTenant(ctx context.Context, tenantName string) (*openapiclient.Tenant, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, rawResp, err := c.ApiClient.DefaultApi.GetTenant(ctx, tenantName).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return nil, chErr
	}
	return resp, err
}

// CreateDatabase creates a new database with the given name, fails if the database already exists
func (c *Client) CreateDatabase(ctx context.Context, databaseName string, tenantName *string) (*openapiclient.Database, error) {
	if tenantName == nil {
		tenantName = &c.Tenant
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, rawResp, err := c.ApiClient.DefaultApi.CreateDatabase(ctx).Tenant(*tenantName).CreateDatabase(openapiclient.CreateDatabase{Name: databaseName}).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return nil, chErr
	}
	return resp, err
}

// GetDatabase returns the database with the given name, fails if the database does not exist
func (c *Client) GetDatabase(ctx context.Context, databaseName string, tenantName *string) (*openapiclient.Database, error) {
	if tenantName == nil {
		tenantName = &c.Tenant
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, rawResp, err := c.ApiClient.DefaultApi.GetDatabase(ctx, databaseName).Tenant(*tenantName).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return nil, chErr
	}
	return resp, err
}

// copyMap returns a new map with the same key-value pairs as the original map, if the original is nil then returns a new empty map
func copyMap(originalMap map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{})
	if originalMap == nil {
		return newMap
	}
	for key, value := range originalMap {
		newMap[key] = value
	}
	return newMap
}

// CreateCollection [legacy] creates a new collection with the given name, metadata, embedding function and distance function
func (c *Client) CreateCollection(ctx context.Context, collectionName string, metadata map[string]interface{}, createOrGet bool, embeddingFunction types.EmbeddingFunction, distanceFunction types.DistanceFunction) (*Collection, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	err := c.preFlightChecks(ctx)
	if err != nil {
		return nil, err
	}
	var _metadata = copyMap(metadata)
	if embeddingFunction == nil {
		ef, _, err := defaultef.NewDefaultEmbeddingFunction()
		if err != nil {
			return nil, err
		}
		embeddingFunction = types.NewV2EmbeddingFunctionAdapter(ef)
	}
	if metadata["embedding_function"] == nil && embeddingFunction != nil {
		_metadata["embedding_function"] = GetStringTypeOfEmbeddingFunction(embeddingFunction)
	}
	var errorEfCloser func()
	if closer, ok := embeddingFunction.(io.Closer); ok {
		errorEfCloser = func() {
			err := closer.Close()
			if err != nil {
				fmt.Printf("error closing embedding function: %v\n", err)
			}
		}
	}
	if distanceFunction == "" {
		_metadata[types.HNSWSpace] = strings.ToLower(string(types.L2))
	} else {
		_metadata[types.HNSWSpace] = strings.ToLower(string(distanceFunction))
	}
	col := openapiclient.CreateCollection{
		Name:        collectionName,
		GetOrCreate: &createOrGet,
		Metadata:    _metadata,
	}
	resp, rawResp, err := c.ApiClient.DefaultApi.CreateCollection(ctx).CreateCollection(col).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		// we defer close the EF if it implements the io.Closer interface
		// TODO is this a good strategy, what if there are other collections that use the EF?
		if errorEfCloser != nil {
			defer errorEfCloser()
		}
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return nil, chErr
	}
	mtd := resp.Metadata
	newCol := NewCollection(c, resp.Id, resp.Name, getMetadataFromAPI(mtd), embeddingFunction, c.Tenant, c.Database)
	c.activeCollections = append(c.activeCollections, newCol)
	return newCol, nil
}

// NewCollection creates a new collection with the given name and options
func (c *Client) NewCollection(ctx context.Context, name string, options ...collection.Option) (*Collection, error) {
	b := &collection.Builder{Metadata: make(map[string]interface{})}
	for _, option := range options {
		if err := option(b); err != nil {
			return nil, err
		}
	}
	if name == "" {
		return nil, fmt.Errorf("collection name cannot be empty")
	}
	b.Name = name
	var distanceFunction types.DistanceFunction
	if df := b.Metadata[types.HNSWSpace]; df == nil {
		b.Metadata[types.HNSWSpace] = types.L2
	} else {
		var derr error
		distanceFunction, derr = types.ToDistanceFunction(df)
		if derr != nil {
			return nil, derr
		}
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.CreateCollection(ctx, b.Name, b.Metadata, b.CreateIfNotExist, b.EmbeddingFunction, distanceFunction)
}

// DeleteCollection deletes the collection with the given name
func (c *Client) DeleteCollection(ctx context.Context, collectionName string) (*Collection, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	err := c.preFlightChecks(ctx)
	if err != nil {
		return nil, err
	}
	col, rawResp, gcerr := c.ApiClient.DefaultApi.GetCollection(ctx, collectionName).Execute()
	if gcerr != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, gcerr)
		return nil, chErr
	}
	deletedCol, _, err := c.ApiClient.DefaultApi.DeleteCollection(ctx, collectionName).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return nil, chErr
	}
	if deletedCol == nil {
		return NewCollection(c, col.Id, col.Name, getMetadataFromAPI(col.Metadata), nil, c.Tenant, c.Database), nil
	} else {
		return NewCollection(c, deletedCol.Id, deletedCol.Name, getMetadataFromAPI(deletedCol.Metadata), nil, c.Tenant, c.Database), nil
	}
}

// Reset deletes all data in the Chroma server if `ALLOW_RESET` is set to true in the environment variables of the server, otherwise fails
func (c *Client) Reset(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, rawResp, err := c.ApiClient.DefaultApi.Reset(ctx).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return false, chErr
	}
	return resp, err
}

// ListCollections returns a list of all collections in the database
func (c *Client) ListCollections(ctx context.Context) ([]*Collection, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	err := c.preFlightChecks(ctx)
	if err != nil {
		return nil, err
	}
	req := c.ApiClient.DefaultApi.ListCollections(ctx)
	resp, rawResp, err := req.Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return nil, chErr
	}
	collections := make([]*Collection, len(resp))
	for i, col := range resp {
		collections[i] = NewCollection(c, col.Id, col.Name, getMetadataFromAPI(col.Metadata), nil, c.Tenant, c.Database)
	}
	return collections, nil
}

// CountCollections returns the number of collections in the database
func (c *Client) CountCollections(ctx context.Context) (int32, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	err := c.preFlightChecks(ctx)
	if err != nil {
		return -1, err
	}
	resp, rawResp, err := c.ApiClient.DefaultApi.CountCollections(ctx).Tenant(c.Tenant).Database(c.Database).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return 0, chErr
	}
	return resp, nil
}

// PreflightChecks returns the preflight checks of the Chroma server, returns a map of the preflight checks. Currently on max_batch_size supported by the server is returned
func (c *Client) PreflightChecks(ctx context.Context) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, rawResp, err := c.ApiClient.DefaultApi.PreFlightChecks(ctx).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return nil, chErr
	}
	return resp, err
}

// Version returns the version of the Chroma server
func (c *Client) Version(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, rawResp, err := c.ApiClient.DefaultApi.Version(ctx).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return "", chErr
	}
	version := strings.ReplaceAll(resp, `"`, "")
	var semVersion *semver.Version
	semVersion, err = semver.NewVersion(version)
	if err != nil {
		return "", err
	}
	c.APIVersion = *semVersion
	return version, err
}

// Close closes the client and all closeable resources
func (c *Client) Close() error {
	errors := make([]error, 0)
	for _, col := range c.activeCollections {
		err := col.close()
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("errors: %v", errors)
	}
	return nil
}

type GetResults struct {
	Ids        []string
	Documents  []string
	Metadatas  []map[string]interface{}
	Embeddings []*types.Embedding
}

type Collection struct {
	Name              string
	EmbeddingFunction types.EmbeddingFunction
	ApiClient         *openapiclient.APIClient //nolint
	Metadata          map[string]interface{}
	ID                string
	Tenant            string
	Database          string
	chromaClient      *Client
}

func (c *Collection) close() error {
	var err error
	if closer, ok := c.EmbeddingFunction.(io.Closer); ok {
		err = closer.Close()
	}
	if c.chromaClient != nil {
		// remove the collection from the active collections
		for i, col := range c.chromaClient.activeCollections {
			if col.ID == c.ID {
				c.chromaClient.activeCollections = append(c.chromaClient.activeCollections[:i], c.chromaClient.activeCollections[i+1:]...)
				break
			}
		}
	}
	return err
}

func (c *Collection) String() string {
	return fmt.Sprintf("Collection{ Name: %s, ID: %s, Tenant: %s, Database: %s, Metadata: %v }",
		c.Name, c.ID, c.Tenant, c.Database, c.Metadata)
}

func NewCollection(chromaClient *Client, id string, name string, metadata *map[string]interface{}, embeddingFunction types.EmbeddingFunction, tenant string, database string) *Collection {
	_metadata := make(map[string]interface{})
	if metadata != nil {
		_metadata = *metadata
	}
	return &Collection{
		Name:              name,
		EmbeddingFunction: embeddingFunction,
		ApiClient:         chromaClient.ApiClient,
		Metadata:          _metadata,
		ID:                id,
		Tenant:            tenant,
		Database:          database,
		chromaClient:      chromaClient,
	}
}

func (c *Collection) Add(ctx context.Context, embeddings []*types.Embedding, metadatas []map[string]interface{}, documents []string, ids []string) (*Collection, error) {
	var _embeddings []openapiclient.EmbeddingsInner

	if len(ids) != len(documents) && len(documents) != len(metadatas) {
		return c, fmt.Errorf("ids and embeddings must have the same length")
	}
	ctx, cancel := context.WithTimeout(ctx, c.chromaClient.timeout)
	defer cancel()
	if len(embeddings) == 0 {
		embds, embErr := c.EmbeddingFunction.EmbedDocuments(ctx, documents)
		if embErr != nil {
			return c, embErr
		}
		_embeddings = types.ToAPIEmbeddings(embds)
	} else {
		_embeddings = types.ToAPIEmbeddings(embeddings)
	}

	if len(ids) == 0 {
		return c, fmt.Errorf("ids cannot be empty")
	}
	var addEmbedding = openapiclient.AddEmbedding{
		Embeddings: _embeddings,
		Metadatas:  metadatas,
		Documents:  documents,
		Ids:        ids,
	}
	_, rawResp, err := c.ApiClient.DefaultApi.Add(ctx, c.ID).AddEmbedding(addEmbedding).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return c, chErr
	}
	return c, nil
}

func (c *Collection) AddRecords(ctx context.Context, recordSet *types.RecordSet) (*Collection, error) {
	return c.Add(ctx, recordSet.GetEmbeddings(), recordSet.GetMetadatas(), recordSet.GetDocuments(), recordSet.GetIDs())
}

func (c *Collection) Upsert(ctx context.Context, embeddings []*types.Embedding, metadatas []map[string]interface{}, documents []string, ids []string) (*Collection, error) {
	var _embeddings []openapiclient.EmbeddingsInner

	if len(ids) != len(documents) && len(documents) != len(metadatas) {
		return c, fmt.Errorf("ids and embeddings must have the same length")
	}
	ctx, cancel := context.WithTimeout(ctx, c.chromaClient.timeout)
	defer cancel()
	if len(embeddings) == 0 {
		embds, embErr := c.EmbeddingFunction.EmbedDocuments(ctx, documents)
		if embErr != nil {
			return c, embErr
		}
		_embeddings = types.ToAPIEmbeddings(embds)
	} else {
		_embeddings = types.ToAPIEmbeddings(embeddings)
	}
	if len(ids) == 0 {
		return c, fmt.Errorf("ids cannot be empty")
	}

	var addEmbedding = openapiclient.AddEmbedding{
		Embeddings: _embeddings,
		Metadatas:  metadatas,
		Documents:  documents,
		Ids:        ids,
	}

	_, rawResp, err := c.ApiClient.DefaultApi.Upsert(ctx, c.ID).AddEmbedding(addEmbedding).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return c, chErr
	}

	return c, nil
}

func (c *Collection) Modify(ctx context.Context, embeddings []*types.Embedding, metadatas []map[string]interface{}, documents []string, ids []string) (*Collection, error) {
	var _embeddings []openapiclient.EmbeddingsInner
	ctx, cancel := context.WithTimeout(ctx, c.chromaClient.timeout)
	defer cancel()
	if len(embeddings) == 0 {
		embds, embErr := c.EmbeddingFunction.EmbedDocuments(ctx, documents)
		if embErr != nil {
			return c, embErr
		}
		_embeddings = types.ToAPIEmbeddings(embds)
	} else {
		_embeddings = types.ToAPIEmbeddings(embeddings)
	}

	var updateEmbedding = openapiclient.UpdateEmbedding{
		Embeddings: _embeddings,
		Metadatas:  metadatas,
		Documents:  documents,
		Ids:        ids,
	}

	_, rawResp, err := c.ApiClient.DefaultApi.Update(ctx, c.ID).UpdateEmbedding(updateEmbedding).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return c, chErr
	}
	return c, nil
}

func (c *Collection) GetWithOptions(ctx context.Context, options ...types.CollectionQueryOption) (*GetResults, error) {
	query := &types.CollectionQueryBuilder{}
	for _, opt := range options {
		err := opt(query)
		if err != nil {
			return nil, err
		}
	}
	if query.Include == nil {
		query.Include = []types.QueryEnum{types.IDocuments, types.IMetadatas}
	}
	inc := make([]openapiclient.IncludeInner, len(query.Include))
	for i, v := range query.Include {
		_v := string(v)
		inc[i] = openapiclient.IncludeInner{
			String: &_v,
		}
	}
	ctx, cancel := context.WithTimeout(ctx, c.chromaClient.timeout)
	defer cancel()
	cd, rawResp, err := c.ApiClient.DefaultApi.Get(ctx, c.ID).GetEmbedding(openapiclient.GetEmbedding{
		Ids:           query.Ids,
		Where:         query.Where,
		WhereDocument: query.WhereDocument,
		Include:       inc,
		Limit:         &query.Limit,
		Offset:        &query.Offset,
	}).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return nil, chErr
	}

	results := &GetResults{
		Ids:        cd.Ids,
		Documents:  cd.Documents,
		Metadatas:  cd.Metadatas,
		Embeddings: APIEmbeddingsToEmbeddings(cd.Embeddings),
	}
	return results, nil
}

func (c *Collection) Get(ctx context.Context, where map[string]interface{}, whereDocuments map[string]interface{}, ids []string, include []types.QueryEnum) (*GetResults, error) {
	return c.GetWithOptions(ctx, types.WithWhereMap(where), types.WithWhereDocumentMap(whereDocuments), types.WithIds(ids), types.WithInclude(include...))
}

type QueryResults struct {
	Documents                     [][]string                 `json:"documents,omitempty"`
	Ids                           [][]string                 `json:"ids,omitempty"`
	Metadatas                     [][]map[string]interface{} `json:"metadatas,omitempty"`
	Distances                     [][]float32                `json:"distances,omitempty"`
	QueryTexts                    []string
	QueryEmbeddings               []*types.Embedding
	QueryTextsGeneratedEmbeddings []*types.Embedding // the generated embeddings from the query texts
}

func getMetadataFromAPI(metadata *map[string]openapiclient.Metadata) *map[string]interface{} {
	if metadata == nil {
		return nil
	}
	result := make(map[string]interface{})
	for key, value := range *metadata {
		switch {
		case value.String != nil:
			result[key] = *value.String
		case value.Bool != nil:
			result[key] = *value.Bool
		case value.Float32 != nil:
			result[key] = *value.Float32
		case value.Int32 != nil:
			result[key] = *value.Int32
		}
	}
	return &result
}

func (c *Collection) Query(ctx context.Context, queryTexts []string, nResults int32, where map[string]interface{}, whereDocuments map[string]interface{}, include []types.QueryEnum) (*QueryResults, error) {
	return c.QueryWithOptions(ctx, types.WithQueryTexts(queryTexts), types.WithNResults(nResults), types.WithWhereMap(where), types.WithWhereDocumentMap(whereDocuments), types.WithInclude(include...))
}
func (c *Collection) QueryWithOptions(ctx context.Context, queryOptions ...types.CollectionQueryOption) (*QueryResults, error) {
	b := &types.CollectionQueryBuilder{
		QueryTexts:      make([]string, 0),
		QueryEmbeddings: make([]*types.Embedding, 0),
		Where:           nil,
		WhereDocument:   nil,
	}
	for _, opt := range queryOptions {
		if err := opt(b); err != nil {
			return nil, err
		}
	}
	var localInclude = b.Include
	if len(b.Include) == 0 {
		localInclude = []types.QueryEnum{types.IDocuments, types.IMetadatas, types.IDistances}
	}
	_includes := make([]openapiclient.IncludeInner, len(localInclude))
	for i, v := range localInclude {
		_v := string(v)
		_includes[i] = openapiclient.IncludeInner{
			String: &_v,
		}
	}
	ctx, cancel := context.WithTimeout(ctx, c.chromaClient.timeout)
	defer cancel()
	if len(b.QueryEmbeddings) == 0 && c.EmbeddingFunction == nil {
		return nil, fmt.Errorf("embedding function is not set. Please configure the embedding function when you get or create the collection, or provide the query embeddings")
	}
	embds, embErr := c.EmbeddingFunction.EmbedDocuments(ctx, b.QueryTexts)
	if embErr != nil {
		return nil, embErr
	}
	var queryEmbeds = make([]openapiclient.EmbeddingsInner, 0)
	queryEmbeds = append(queryEmbeds, types.ToAPIEmbeddings(b.QueryEmbeddings)...)
	queryEmbeds = append(queryEmbeds, types.ToAPIEmbeddings(embds)...)
	qr, rawResp, err := c.ApiClient.DefaultApi.GetNearestNeighbors(ctx, c.ID).QueryEmbedding(openapiclient.QueryEmbedding{
		Where:           b.Where,
		WhereDocument:   b.WhereDocument,
		NResults:        &b.NResults,
		Include:         _includes,
		QueryEmbeddings: queryEmbeds,
	}).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return nil, chErr
	}

	qresults := QueryResults{
		Documents:                     qr.Documents,
		Ids:                           qr.Ids,
		Metadatas:                     qr.Metadatas,
		Distances:                     qr.Distances,
		QueryTexts:                    b.QueryTexts,
		QueryEmbeddings:               b.QueryEmbeddings,
		QueryTextsGeneratedEmbeddings: embds,
	}
	return &qresults, nil
}
func (c *Collection) Count(ctx context.Context) (int32, error) {
	ctx, cancel := context.WithTimeout(ctx, c.chromaClient.timeout)
	defer cancel()
	req := c.ApiClient.DefaultApi.Count(ctx, c.ID)
	cd, rawResp, err := req.Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return 0, chErr
	}

	return cd, nil
}

func (c *Collection) Update(ctx context.Context, newName string, newMetadata *map[string]interface{}) (*Collection, error) {
	ctx, cancel := context.WithTimeout(ctx, c.chromaClient.timeout)
	defer cancel()
	_newMetadata := make(map[string]interface{})
	if newMetadata != nil {
		_newMetadata = *newMetadata
	}
	_, rawResp, err := c.ApiClient.DefaultApi.UpdateCollection(ctx, c.ID).UpdateCollection(openapiclient.UpdateCollection{NewName: &newName, NewMetadata: _newMetadata}).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return c, chErr
	}
	c.Name = newName
	c.Metadata = _newMetadata
	return c, nil
}

func (c *Collection) Delete(ctx context.Context, ids []string, where map[string]interface{}, whereDocuments map[string]interface{}) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.chromaClient.timeout)
	defer cancel()
	dr, rawResp, err := c.ApiClient.DefaultApi.Delete(ctx, c.ID).DeleteEmbedding(openapiclient.DeleteEmbedding{Where: where, WhereDocument: whereDocuments, Ids: ids}).Execute()
	if err != nil || (rawResp.StatusCode >= 400 && rawResp.StatusCode < 599) {
		chErr := chhttp.ChromaErrorFromHTTPResponse(rawResp, err)
		return nil, chErr
	}
	return dr, nil
}
