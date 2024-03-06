package chroma

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/Masterminds/semver" //nolint:gci
	"github.com/amikos-tech/chroma-go/collection"
	openapiclient "github.com/amikos-tech/chroma-go/swagger"
	"github.com/amikos-tech/chroma-go/types"
)

type ClientConfiguration struct {
	BasePath          string                  `json:"basePath,omitempty"`
	DefaultHeaders    map[string]string       `json:"defaultHeader,omitempty"`
	EmbeddingFunction types.EmbeddingFunction `json:"embeddingFunction,omitempty"`
}

func APIMetadataToMap(metadata map[string]openapiclient.MetadatasInnerValue) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range metadata {
		result[k] = v
	}
	return result
}

func APIMetadatasToMaps(metadatas []map[string]openapiclient.MetadatasInnerValue) []map[string]interface{} {
	result := make([]map[string]interface{}, len(metadatas))
	for i, v := range metadatas {
		result[i] = APIMetadataToMap(v)
	}
	return result
}

func APIEmbeddingToEmbedding(embedding openapiclient.EmbeddingsInner) []interface{} {
	switch {
	case embedding.ArrayOfInt32 != nil:
		result := make([]interface{}, len(*embedding.ArrayOfInt32))
		for i, v := range *embedding.ArrayOfInt32 {
			result[i] = v
		}
		return result
	case embedding.ArrayOfFloat32 != nil:
		result := make([]interface{}, len(*embedding.ArrayOfFloat32))
		for i, v := range *embedding.ArrayOfFloat32 {
			result[i] = v
		}
		return result
	default:
		return nil
	}
}

func EmbeddingToAPIEmbedding(embedding []interface{}) openapiclient.EmbeddingsInner {
	if len(embedding) == 0 {
		return openapiclient.EmbeddingsInner{ArrayOfInt32: nil, ArrayOfFloat32: nil}
	}
	switch embedding[0].(type) {
	case int32:
		result := make([]int32, len(embedding))
		for i, v := range embedding {
			result[i] = v.(int32)
		}
		return openapiclient.EmbeddingsInner{ArrayOfInt32: &result, ArrayOfFloat32: nil}
	case float32:
		result := make([]float32, len(embedding))
		for i, v := range embedding {
			result[i] = v.(float32)
		}
		return openapiclient.EmbeddingsInner{ArrayOfInt32: nil, ArrayOfFloat32: &result}
	default:
		return openapiclient.EmbeddingsInner{ArrayOfInt32: nil, ArrayOfFloat32: nil}
	}
}

func EmbeddingsToAPIEmbeddings(embeddings *[][]interface{}, embeddingsF32 *[][]float32) []openapiclient.EmbeddingsInner {
	switch {
	case embeddings != nil:
		result := make([]openapiclient.EmbeddingsInner, len(*embeddings))
		for i, v := range *embeddings {
			result[i] = EmbeddingToAPIEmbedding(v)
		}
		return result
	case embeddingsF32 != nil:
		result := make([]openapiclient.EmbeddingsInner, len(*embeddingsF32))
		for i, v := range *embeddingsF32 {
			copyOfV := make([]float32, len(v)) // Create a new slice
			copy(copyOfV, v)                   // Copy the data
			result[i] = openapiclient.EmbeddingsInner{ArrayOfInt32: nil, ArrayOfFloat32: &copyOfV}
		}
		return result
	default:
		return nil
	}
}

func APIEmbeddingsToEmbeddings(embeddings []openapiclient.EmbeddingsInner) [][]interface{} {
	result := make([][]interface{}, len(embeddings))
	for i, v := range embeddings {
		result[i] = APIEmbeddingToEmbedding(v)
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
}

type ClientOption func(p *Client) error

func WithTenant(tenant string) ClientOption {
	return func(c *Client) error {
		// TODO validate here?
		c.Tenant = tenant
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

func applyOptions(c *Client, options ...ClientOption) error {
	for _, opt := range options {
		if err := opt(c); err != nil {
			return err
		}
	}
	return nil
}

func NewClient(basePath string, options ...ClientOption) (*Client, error) {
	c := &Client{
		Tenant:           types.DefaultTenant,
		Database:         types.DefaultDatabase,
		apiConfiguration: openapiclient.NewConfiguration(),
	}

	c.apiConfiguration.Servers = openapiclient.ServerConfigurations{
		{
			URL:         basePath,
			Description: "No description provided",
		},
	}
	err := applyOptions(c, options...)
	if err != nil {
		return nil, err
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
	_version, _, err := c.ApiClient.DefaultApi.Version(ctx).Execute()
	if err != nil {
		return err
	}
	version, err := semver.NewVersion(strings.ReplaceAll(_version, `"`, ""))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
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

func (c *Client) GetCollection(ctx context.Context, collectionName string, embeddingFunction types.EmbeddingFunction) (*Collection, error) {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	err := c.preFlightChecks(ctx)
	if err != nil {
		return nil, err
	}
	tenantName := types.DefaultTenant
	databaseName := types.DefaultDatabase
	col, httpResp, err := c.ApiClient.DefaultApi.GetCollection(ctx, collectionName).Tenant(c.Tenant).Database(c.Database).Execute()
	if err != nil {
		return nil, err
	}
	if httpResp.StatusCode != 200 {
		return nil, fmt.Errorf("error getting collection: %v", httpResp)
	}
	return NewCollection(c.ApiClient, col.Id, col.Name, col.Metadata, embeddingFunction, tenantName, databaseName), nil
}

func (c *Client) Heartbeat(ctx context.Context) (map[string]float32, error) {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	resp, _, err := c.ApiClient.DefaultApi.Heartbeat(ctx).Execute()
	return resp, err
}

func GetStringTypeOfEmbeddingFunction(ef types.EmbeddingFunction) string {
	typ := reflect.TypeOf(ef)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem() // Dereference if it's a pointer
	}
	return typ.String()
}

func (c *Client) CreateTenant(ctx context.Context, tenantName string) (*openapiclient.Tenant, error) {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	resp, _, err := c.ApiClient.DefaultApi.CreateTenant(ctx).CreateTenant(openapiclient.CreateTenant{Name: tenantName}).Execute()
	return resp, err
}

func (c *Client) GetTenant(ctx context.Context, tenantName string) (*openapiclient.Tenant, error) {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	resp, _, err := c.ApiClient.DefaultApi.GetTenant(ctx, tenantName).Execute()
	return resp, err
}

func (c *Client) CreateDatabase(ctx context.Context, databaseName string, tenantName *string) (*openapiclient.Database, error) {
	if tenantName == nil {
		tenantName = &c.Tenant
	}
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	resp, _, err := c.ApiClient.DefaultApi.CreateDatabase(ctx).Tenant(*tenantName).CreateDatabase(openapiclient.CreateDatabase{Name: databaseName}).Execute()
	return resp, err
}

func (c *Client) GetDatabase(ctx context.Context, databaseName string, tenantName *string) (*openapiclient.Database, error) {
	if tenantName == nil {
		tenantName = &c.Tenant
	}
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	resp, _, err := c.ApiClient.DefaultApi.GetDatabase(ctx, databaseName).Tenant(*tenantName).Execute()
	return resp, err
}

func (c *Client) CreateCollection(ctx context.Context, collectionName string, metadata map[string]interface{}, createOrGet bool, embeddingFunction types.EmbeddingFunction, distanceFunction types.DistanceFunction) (*Collection, error) {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	err := c.preFlightChecks(ctx)
	if err != nil {
		return nil, err
	}
	_metadata := metadata
	if metadata["embedding_function"] == nil {
		_metadata["embedding_function"] = GetStringTypeOfEmbeddingFunction(embeddingFunction)
	}
	if distanceFunction == "" {
		_metadata["hnsw:space"] = strings.ToLower(string(types.L2))
	} else {
		_metadata["hnsw:space"] = strings.ToLower(string(distanceFunction))
	}

	col := openapiclient.CreateCollection{
		Name:        collectionName,
		GetOrCreate: &createOrGet,
		Metadata:    _metadata,
	}
	resp, _, err := c.ApiClient.DefaultApi.CreateCollection(ctx).CreateCollection(col).Execute()
	if err != nil {
		return nil, err
	}
	mtd := resp.Metadata
	return NewCollection(c.ApiClient, resp.Id, resp.Name, mtd, embeddingFunction, c.Tenant, c.Database), nil
}

func (c *Client) NewCollection(ctx context.Context, options ...collection.Option) (*Collection, error) {
	b := &collection.Builder{}
	for _, option := range options {
		if err := option(b); err != nil {
			return nil, err
		}
	}
	if b.Name == "" {
		return nil, fmt.Errorf("collection name cannot be empty")
	}

	if b.Metadata["hnsw:space"] == nil {
		b.Metadata["hnsw:space"] = types.L2
	}
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	return c.CreateCollection(ctx, b.Name, b.Metadata, b.CreateIfNotExist, b.EmbeddingFunction, b.Metadata["hnsw:space"].(types.DistanceFunction))
}

func (c *Client) DeleteCollection(ctx context.Context, collectionName string) (*Collection, error) {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	err := c.preFlightChecks(ctx)
	if err != nil {
		return nil, err
	}
	col, _, gcerr := c.ApiClient.DefaultApi.GetCollection(ctx, collectionName).Execute()
	if gcerr != nil {
		return nil, gcerr
	}
	deletedCol, _, err := c.ApiClient.DefaultApi.DeleteCollection(ctx, collectionName).Execute()
	if err != nil {
		return nil, err
	}
	if deletedCol == nil {
		return NewCollection(c.ApiClient, col.Id, col.Name, col.Metadata, nil, c.Tenant, c.Database), nil
	} else {
		return NewCollection(c.ApiClient, deletedCol.Id, deletedCol.Name, deletedCol.Metadata, nil, c.Tenant, c.Database), nil
	}
}

func (c *Client) Reset(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	resp, _, err := c.ApiClient.DefaultApi.Reset(ctx).Execute()
	return resp, err
}

func (c *Client) ListCollections(ctx context.Context) ([]*Collection, error) {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	err := c.preFlightChecks(ctx)
	if err != nil {
		return nil, err
	}
	req := c.ApiClient.DefaultApi.ListCollections(ctx)
	resp, _, err := req.Execute()
	if err != nil {
		return nil, err
	}
	collections := make([]*Collection, len(resp))
	for i, col := range resp {
		collections[i] = NewCollection(c.ApiClient, col.Id, col.Name, col.Metadata, nil, c.Tenant, c.Database)
	}
	return collections, nil
}

func (c *Client) CountCollections(ctx context.Context) (int32, error) {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	err := c.preFlightChecks(ctx)
	if err != nil {
		return -1, err
	}
	resp, _, err := c.ApiClient.DefaultApi.CountCollections(ctx).Tenant(c.Tenant).Database(c.Database).Execute()
	return resp, err
}

func (c *Client) PreflightChecks(ctx context.Context) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	resp, _, err := c.ApiClient.DefaultApi.PreFlightChecks(ctx).Execute()
	return resp, err
}

func (c *Client) Version(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	resp, _, err := c.ApiClient.DefaultApi.Version(ctx).Execute()
	version := strings.ReplaceAll(resp, `"`, "")
	return version, err
}

type GetResults struct {
	Ids        []string
	Documents  []string
	Metadatas  []map[string]interface{}
	Embeddings [][]interface{}
}

type Collection struct {
	Name              string
	EmbeddingFunction types.EmbeddingFunction
	ApiClient         *openapiclient.APIClient //nolint
	Metadata          map[string]interface{}
	ID                string
	Tenant            string
	Database          string
}

func (c *Collection) String() string {
	return fmt.Sprintf("Collection{ Name: %s, ID: %s, Tenant: %s, Database: %s, Metadata: %v }",
		c.Name, c.ID, c.Tenant, c.Database, c.Metadata)
}

func NewCollection(apiClient *openapiclient.APIClient, id string, name string, metadata map[string]interface{}, embeddingFunction types.EmbeddingFunction, tenant string, database string) *Collection {
	return &Collection{
		Name:              name,
		EmbeddingFunction: embeddingFunction,
		ApiClient:         apiClient,
		Metadata:          metadata,
		ID:                id,
		Tenant:            tenant,
		Database:          database,
	}
}

func (c *Collection) Add(ctx context.Context, embeddings []*types.Embedding, metadatas []map[string]interface{}, documents []string, ids []string) (*Collection, error) {
	var _embeddings []openapiclient.EmbeddingsInner

	if len(ids) != len(documents) && len(documents) != len(metadatas) {
		return c, fmt.Errorf("ids and embeddings must have the same length")
	}
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
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
	_, _, err := c.ApiClient.DefaultApi.Add(ctx, c.ID).AddEmbedding(addEmbedding).Execute()
	if err != nil {
		return c, err
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
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
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

	_, _, err := c.ApiClient.DefaultApi.Upsert(ctx, c.ID).AddEmbedding(addEmbedding).Execute()

	if err != nil {
		return c, err
	}
	return c, nil
}

func (c *Collection) Modify(ctx context.Context, embeddings []*types.Embedding, metadatas []map[string]interface{}, documents []string, ids []string) (*Collection, error) {
	var _embeddings []openapiclient.EmbeddingsInner
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
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

	_, _, err := c.ApiClient.DefaultApi.Update(ctx, c.ID).UpdateEmbedding(updateEmbedding).Execute()

	if err != nil {
		return c, err
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
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	cd, _, err := c.ApiClient.DefaultApi.Get(ctx, c.ID).GetEmbedding(openapiclient.GetEmbedding{
		Ids:           query.Ids,
		Where:         query.Where,
		WhereDocument: query.WhereDocument,
		Include:       inc,
		Limit:         &query.Limit,
		Offset:        &query.Offset,
	}).Execute()
	if err != nil {
		return nil, err
	}

	results := &GetResults{
		Ids:        cd.Ids,
		Documents:  cd.Documents,
		Metadatas:  APIMetadatasToMaps(cd.Metadatas),
		Embeddings: APIEmbeddingsToEmbeddings(cd.Embeddings),
	}
	return results, nil
}

func (c *Collection) Get(ctx context.Context, where map[string]interface{}, whereDocuments map[string]interface{}, ids []string, include []types.QueryEnum) (*GetResults, error) {
	return c.GetWithOptions(ctx, types.WithWhere(where), types.WithWhereDocument(whereDocuments), types.WithIds(ids), types.WithInclude(include...))
}

type QueryResults struct {
	Documents [][]string                 `json:"documents,omitempty"`
	Ids       [][]string                 `json:"ids,omitempty"`
	Metadatas [][]map[string]interface{} `json:"metadatas,omitempty"`
	Distances [][]float32                `json:"distances,omitempty"`
}

func getMetadatasFromAPI(metadatas [][]map[string]openapiclient.MetadatasInnerValue) ([][]map[string]interface{}, error) {
	// Initialize the result slice
	result := make([][]map[string]interface{}, len(metadatas))

	// Iterate over the outer slice
	for i, outerItem := range metadatas {
		result[i] = make([]map[string]interface{}, len(outerItem))

		// Iterate over the inner map
		for j, metadataMap := range outerItem {
			resultMap := make(map[string]interface{})
			for key, value := range metadataMap {
				// Convert MetadatasInnerValue to interface{}
				var rawValue interface{}
				b, e := value.MarshalJSON()
				if e != nil {
					return nil, e
				}
				rawValue = b
				// Store in the result map
				resultMap[key] = rawValue
			}
			result[i][j] = resultMap
		}
	}

	return result, nil
}
func ConvertEmbeds(embeds [][]float32) []interface{} {
	_embeddings := make([]interface{}, len(embeds))
	for i, v := range embeds {
		_embeddings[i] = v
	}
	return _embeddings
}
func (c *Collection) Query(ctx context.Context, queryTexts []string, nResults int32, where map[string]interface{}, whereDocuments map[string]interface{}, include []types.QueryEnum) (*QueryResults, error) {
	return c.QueryWithOptions(ctx, types.WithQueryTexts(queryTexts), types.WithNResults(nResults), types.WithWhere(where), types.WithWhereDocument(whereDocuments), types.WithInclude(include...))
}
func (c *Collection) QueryWithOptions(ctx context.Context, queryOptions ...types.CollectionQueryOption) (*QueryResults, error) {
	b := &types.CollectionQueryBuilder{
		QueryTexts:      make([]string, 0),
		QueryEmbeddings: make([]*types.Embedding, 0),
		Where:           make(map[string]interface{}),
		WhereDocument:   make(map[string]interface{}),
	}
	for _, opt := range queryOptions {
		if err := opt(b); err != nil {
			return nil, err
		}
	}
	var localInclude = b.Include
	if len(b.Include) == 0 {
		localInclude = []types.QueryEnum{types.IDocuments, types.IMetadatas}
	}
	_includes := make([]openapiclient.IncludeInner, len(localInclude))
	for i, v := range localInclude {
		_v := string(v)
		_includes[i] = openapiclient.IncludeInner{
			String: &_v,
		}
	}
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	embds, embErr := c.EmbeddingFunction.EmbedDocuments(ctx, b.QueryTexts)
	if embErr != nil {
		return nil, embErr
	}

	fmt.Printf("QueryEmbeddings %v\n", embds[0].ArrayOfFloat32)

	var queryEmbeds = make([]openapiclient.EmbeddingsInner, 0)
	queryEmbeds = append(queryEmbeds, types.ToAPIEmbeddings(b.QueryEmbeddings)...)
	queryEmbeds = append(queryEmbeds, types.ToAPIEmbeddings(embds)...)
	qr, _, err := c.ApiClient.DefaultApi.GetNearestNeighbors(ctx, c.ID).QueryEmbedding(openapiclient.QueryEmbedding{
		Where:           b.Where,
		WhereDocument:   b.WhereDocument,
		NResults:        &b.NResults,
		Include:         _includes,
		QueryEmbeddings: queryEmbeds,
	}).Execute()

	if err != nil {
		return nil, err
	}

	metadatas, err := getMetadatasFromAPI(qr.Metadatas)
	if err != nil {
		return nil, err
	}
	qresults := QueryResults{
		Documents: qr.Documents,
		Ids:       qr.Ids,
		Metadatas: metadatas,
		Distances: qr.Distances,
	}
	return &qresults, nil
}
func (c *Collection) Count(ctx context.Context) (int32, error) {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	req := c.ApiClient.DefaultApi.Count(ctx, c.ID)
	cd, _, err := req.Execute()

	if err != nil {
		return -1, err
	}

	return cd, nil
}

func (c *Collection) Update(ctx context.Context, newName string, newMetadata map[string]interface{}) (*Collection, error) {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	_, _, err := c.ApiClient.DefaultApi.UpdateCollection(ctx, c.ID).UpdateCollection(openapiclient.UpdateCollection{NewName: &newName, NewMetadata: newMetadata}).Execute()
	if err != nil {
		return c, err
	}
	c.Name = newName
	c.Metadata = newMetadata
	return c, nil
}

func (c *Collection) Delete(ctx context.Context, ids []string, where map[string]interface{}, whereDocuments map[string]interface{}) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultTimeout)
	defer cancel()
	dr, _, err := c.ApiClient.DefaultApi.Delete(ctx, c.ID).DeleteEmbedding(openapiclient.DeleteEmbedding{Where: where, WhereDocument: whereDocuments, Ids: ids}).Execute()
	if err != nil {
		return nil, err
	}
	return dr, nil
}
