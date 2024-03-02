package chroma

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"

	openapiclient "github.com/amikos-tech/chroma-go/swagger"
)

type ClientConfiguration struct {
	BasePath          string            `json:"basePath,omitempty"`
	DefaultHeaders    map[string]string `json:"defaultHeader,omitempty"`
	EmbeddingFunction EmbeddingFunction `json:"embeddingFunction,omitempty"`
}

type EmbeddingFunction interface {
	// EmbedDocuments returns a vector for each text.
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
	// EmbedQuery embeds a single text.
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
}

func MapToAPI(inmap map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range inmap {
		result[k] = v
	}
	return result
}

func MapListToAPI(inmap []map[string]string) []map[string]interface{} {
	result := make([]map[string]interface{}, len(inmap))
	for i, v := range inmap {
		result[i] = MapToAPI(v)
	}
	return result
}

// func MapFromApi(inmap map[string]interface{}) map[string]string {
//	result := make(map[string]string)
//	for k, v := range inmap {
//		result[k] = v.(string)
//	}
//	return result
// }

// func MapListFromApi(inmap []map[string]interface{}) []map[string]string {
//	result := make([]map[string]string, len(inmap))
//	for i, v := range inmap {
//		result[i] = MapFromApi(v)
//	}
//	return result
// }

// Client represents the ChromaDB Client
type Client struct {
	ApiClient *openapiclient.APIClient //nolint
}

type AuthType string

const (
	BASIC              AuthType = "basic"
	TokenAuthorization AuthType = "authorization"
	TokenXChromaToken  AuthType = "xchromatoken"
)

type AuthMethod interface {
	GetCredentials() map[string]string
	GetType() AuthType
}

type BasicAuth struct {
	Username string
	Password string
}

func (b BasicAuth) GetCredentials() map[string]string {
	return map[string]string{
		"username": b.Username,
		"password": b.Password,
	}
}

func (b BasicAuth) GetType() AuthType {
	return BASIC
}

func NewBasicAuth(username string, password string) ClientAuthCredentials {
	return ClientAuthCredentials{
		AuthMethod: BasicAuth{
			Username: username,
			Password: password,
		},
	}
}

type AuthorizationTokenAuth struct {
	Token string
}

func (t AuthorizationTokenAuth) GetType() AuthType {
	return TokenAuthorization
}

func (t AuthorizationTokenAuth) GetCredentials() map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + t.Token,
	}
}

type XChromaTokenAuth struct {
	Token string
}

func (t XChromaTokenAuth) GetType() AuthType {
	return TokenXChromaToken
}

func (t XChromaTokenAuth) GetCredentials() map[string]string {
	return map[string]string{
		"X-Chroma-Token": t.Token,
	}
}

type ClientAuthCredentials struct {
	AuthMethod AuthMethod
}

func NewTokenAuth(token string, authType AuthType) ClientAuthCredentials {
	switch {
	case authType == TokenAuthorization:
		return ClientAuthCredentials{
			AuthMethod: AuthorizationTokenAuth{
				Token: token,
			},
		}
	case authType == TokenXChromaToken:
		return ClientAuthCredentials{
			AuthMethod: XChromaTokenAuth{
				Token: token,
			},
		}
	default:
		panic("Invalid auth type")
	}
}

type ClientConfig struct {
	BasePath              string
	DefaultHeaders        *map[string]string
	ClientAuthCredentials *ClientAuthCredentials
}

func NewClientConfig(basePath string, defaultHeaders *map[string]string, clientAuthCredentials *ClientAuthCredentials) ClientConfig {
	return ClientConfig{
		BasePath:              basePath,
		DefaultHeaders:        defaultHeaders,
		ClientAuthCredentials: clientAuthCredentials,
	}
}

func NewClient(config ClientConfig) *Client {
	configuration := openapiclient.NewConfiguration()
	if config.ClientAuthCredentials != nil {
		// combine config.DefaultHeaders and config.AuthMethod.GetCredentials() maps
		var headers = make(map[string]string)
		if config.DefaultHeaders != nil {
			for k, v := range *config.DefaultHeaders {
				headers[k] = v
			}
		}
		for k, v := range config.ClientAuthCredentials.AuthMethod.GetCredentials() {
			headers[k] = v
		}
		configuration.DefaultHeader = headers
	} else if config.DefaultHeaders != nil {
		configuration.DefaultHeader = *config.DefaultHeaders
	}
	configuration.Servers = openapiclient.ServerConfigurations{
		{
			URL:         config.BasePath,
			Description: "No description provided",
		},
	}
	configuration.Debug = true
	apiClient := openapiclient.NewAPIClient(configuration)
	return &Client{
		ApiClient: apiClient,
	}
}

func (c *Client) GetCollection(ctx context.Context, collectionName string, embeddingFunction EmbeddingFunction) (*Collection, error) {
	col, httpResp, err := c.ApiClient.DefaultApi.GetCollection(ctx, collectionName).Execute()
	if err != nil {
		return nil, err
	}
	if httpResp.StatusCode != 200 {
		return nil, fmt.Errorf("error getting collection: %v", httpResp)
	}
	return NewCollection(c.ApiClient, col.Id, col.Name, col.Metadata, embeddingFunction), nil
}

func (c *Client) Heartbeat(ctx context.Context) (map[string]float32, error) {
	resp, _, err := c.ApiClient.DefaultApi.Heartbeat(ctx).Execute()
	return resp, err
}

type DistanceFunction string

const (
	L2     DistanceFunction = "l2"
	COSINE DistanceFunction = "cosine"
	IP     DistanceFunction = "ip"
)

func GetStringTypeOfEmbeddingFunction(ef EmbeddingFunction) string {
	typ := reflect.TypeOf(ef)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem() // Dereference if it's a pointer
	}
	return typ.String()
}

func (c *Client) CreateCollection(ctx context.Context, collectionName string, metadata map[string]interface{}, createOrGet bool, embeddingFunction EmbeddingFunction, distanceFunction DistanceFunction) (*Collection, error) {
	_metadata := metadata
	if metadata["embedding_function"] == nil {
		_metadata["embedding_function"] = GetStringTypeOfEmbeddingFunction(embeddingFunction)
	}
	if distanceFunction == "" {
		_metadata["hnsw:space"] = strings.ToLower(string(L2))
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
	return NewCollection(c.ApiClient, resp.Id, resp.Name, mtd, embeddingFunction), nil
}

func (c *Client) DeleteCollection(ctx context.Context, collectionName string) (*Collection, error) {
	col, httpResp, gcerr := c.ApiClient.DefaultApi.GetCollection(ctx, collectionName).Execute()
	if gcerr != nil {
		log.Fatal(httpResp, gcerr)
		return nil, gcerr
	}
	deletedCol, httpResp, err := c.ApiClient.DefaultApi.DeleteCollection(ctx, collectionName).Execute()
	if err != nil {
		log.Fatal(httpResp, err)
		return nil, err
	}
	if deletedCol == nil {
		return NewCollection(c.ApiClient, col.Id, col.Name, col.Metadata, nil), nil
	} else {
		return NewCollection(c.ApiClient, deletedCol.Id, deletedCol.Name, deletedCol.Metadata, nil), nil
	}
}

func (c *Client) Reset(ctx context.Context) (bool, error) {
	resp, _, err := c.ApiClient.DefaultApi.Reset(ctx).Execute()
	return resp, err
}

func (c *Client) ListCollections(ctx context.Context) ([]*Collection, error) {
	req := c.ApiClient.DefaultApi.ListCollections(ctx)
	resp, _, err := req.Execute()
	if err != nil {
		return nil, err
	}
	collections := make([]*Collection, len(resp))
	for i, col := range resp {
		collections[i] = NewCollection(c.ApiClient, col.Id, col.Name, col.Metadata, nil)
	}
	return collections, nil
}

func (c *Client) Version(ctx context.Context) (string, error) {
	resp, _, err := c.ApiClient.DefaultApi.Version(ctx).Execute()
	version := strings.ReplaceAll(resp, `"`, "")
	return version, err
}

type CollectionData struct {
	Ids       []string                 `json:"ids,omitempty"`
	Documents []string                 `json:"documents,omitempty"`
	Metadatas []map[string]interface{} `json:"metadatas,omitempty"`
}

type Collection struct {
	Name              string
	EmbeddingFunction EmbeddingFunction
	ApiClient         *openapiclient.APIClient //nolint
	Metadata          map[string]interface{}
	id                string
	CollectionData    *CollectionData
}

func NewCollection(apiClient *openapiclient.APIClient, id string, name string, metadata map[string]interface{}, embeddingFunction EmbeddingFunction) *Collection {
	return &Collection{
		Name:              name,
		EmbeddingFunction: embeddingFunction,
		ApiClient:         apiClient,
		Metadata:          metadata,
		id:                id,
	}
}

func (c *Collection) Add(ctx context.Context, embeddings [][]float32, metadatas []map[string]interface{}, documents []string, ids []string) (*Collection, error) {
	var _embeddings []interface{}

	if len(embeddings) == 0 {
		embds, embErr := c.EmbeddingFunction.EmbedDocuments(ctx, documents)
		if embErr != nil {
			return c, embErr
		}
		_embeddings = ConvertEmbeds(embds)
	} else {
		_embeddings = ConvertEmbeds(embeddings)
	}
	var addEmbedding = openapiclient.AddEmbedding{
		Embeddings: _embeddings,
		Metadatas:  metadatas,
		Documents:  documents,
		Ids:        ids,
	}
	_, _, err := c.ApiClient.DefaultApi.Add(ctx, c.id).AddEmbedding(addEmbedding).Execute()
	if err != nil {
		return c, err
	}
	return c, nil
}

func (c *Collection) Upsert(ctx context.Context, embeddings [][]float32, metadatas []map[string]interface{}, documents []string, ids []string) (*Collection, error) {
	var _embeddings []interface{}

	if len(embeddings) == 0 {
		embds, embErr := c.EmbeddingFunction.EmbedDocuments(ctx, documents)
		if embErr != nil {
			return c, embErr
		}
		_embeddings = ConvertEmbeds(embds)
	} else {
		_embeddings = ConvertEmbeds(embeddings)
	}

	var addEmbedding = openapiclient.AddEmbedding{
		Embeddings: _embeddings,
		Metadatas:  metadatas,
		Documents:  documents,
		Ids:        ids,
	}

	_, _, err := c.ApiClient.DefaultApi.Upsert(ctx, c.id).AddEmbedding(addEmbedding).Execute()

	if err != nil {
		return c, err
	}
	return c, nil
}

func (c *Collection) Modify(ctx context.Context, embeddings [][]float32, metadatas []map[string]interface{}, documents []string, ids []string) (*Collection, error) {
	var _embeddings []interface{}

	if len(embeddings) == 0 {
		embds, embErr := c.EmbeddingFunction.EmbedDocuments(ctx, documents)
		if embErr != nil {
			return c, embErr
		}
		_embeddings = ConvertEmbeds(embds)
	} else {
		_embeddings = ConvertEmbeds(embeddings)
	}

	var updateEmbedding = openapiclient.UpdateEmbedding{
		Embeddings: _embeddings,
		Metadatas:  metadatas,
		Documents:  documents,
		Ids:        ids,
	}

	_, _, err := c.ApiClient.DefaultApi.Update(ctx, c.id).UpdateEmbedding(updateEmbedding).Execute()

	if err != nil {
		return c, err
	}
	return c, nil
}

func (c *Collection) Get(ctx context.Context, where map[string]interface{}, whereDocuments map[string]interface{}, ids []string) (*Collection, error) {
	cd, _, err := c.ApiClient.DefaultApi.Get(ctx, c.id).GetEmbedding(openapiclient.GetEmbedding{
		Ids:           ids,
		Where:         where,
		WhereDocument: whereDocuments,
	}).Execute()
	if err != nil {
		return c, err
	}

	metadatas, err := getMetadatasListFromAPI(cd.Metadatas)
	if err != nil {
		return c, err
	}

	cdata := CollectionData{
		Ids:       cd.Ids,
		Documents: cd.Documents,
		Metadatas: metadatas,
	}
	c.CollectionData = &cdata
	return c, nil
}

type QueryEnum string

const (
	documents  QueryEnum = "documents"
	embeddings QueryEnum = "embeddings"
	metadatas  QueryEnum = "metadatas"
	distances  QueryEnum = "distances"
)

type QueryResults struct {
	Documents [][]string                 `json:"documents,omitempty"`
	Ids       [][]string                 `json:"ids,omitempty"`
	Metadatas [][]map[string]interface{} `json:"metadatas,omitempty"`
	Distances [][]float32                `json:"distances,omitempty"`
}

func getMetadatasListFromAPI(metadatas []map[string]openapiclient.MetadatasInnerValue) ([]map[string]interface{}, error) {
	// Initialize the result slice
	result := make([]map[string]interface{}, len(metadatas))
	// Iterate over the inner map
	for j, metadataMap := range metadatas {
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
		result[j] = resultMap
	}

	return result, nil
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
func (c *Collection) Query(ctx context.Context, queryTexts []string, nResults int32, where map[string]interface{}, whereDocuments map[string]interface{}, include []QueryEnum) (*QueryResults, error) {
	var localInclude = include
	if len(include) == 0 {
		localInclude = []QueryEnum{documents, metadatas, distances}
	}
	_includes := make([]openapiclient.IncludeInner, len(localInclude))
	for i, v := range localInclude {
		_v := string(v)
		_includes[i] = openapiclient.IncludeInner{
			String: &_v,
		}
	}

	embds, embErr := c.EmbeddingFunction.EmbedDocuments(ctx, queryTexts)
	if embErr != nil {
		return nil, embErr
	}
	qr, _, err := c.ApiClient.DefaultApi.GetNearestNeighbors(ctx, c.id).QueryEmbedding(openapiclient.QueryEmbedding{
		Where:           where,
		WhereDocument:   whereDocuments,
		NResults:        &nResults,
		Include:         _includes,
		QueryEmbeddings: ConvertEmbeds(embds),
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
	req := c.ApiClient.DefaultApi.Count(ctx, c.id)
	cd, _, err := req.Execute()

	if err != nil {
		return -1, err
	}

	return cd, nil
}

func (c *Collection) Update(ctx context.Context, newName string, newMetadata map[string]interface{}) (*Collection, error) {
	_, httpResp, err := c.ApiClient.DefaultApi.UpdateCollection(ctx, c.id).UpdateCollection(openapiclient.UpdateCollection{NewName: &newName, NewMetadata: newMetadata}).Execute()
	if err != nil {
		log.Fatal(httpResp, err)
		return c, err
	}
	c.Name = newName
	c.Metadata = newMetadata
	return c, nil
}

func (c *Collection) Delete(ctx context.Context, ids []string, where map[string]interface{}, whereDocuments map[string]interface{}) ([]string, error) {
	dr, httpResp, err := c.ApiClient.DefaultApi.Delete(ctx, c.id).DeleteEmbedding(openapiclient.DeleteEmbedding{Where: where, WhereDocument: whereDocuments, Ids: ids}).Execute()
	if err != nil {
		log.Fatal(httpResp, err)
		return nil, err
	}
	return dr, nil
}
