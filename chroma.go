package chroma_go

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
	CreateEmbedding(documents []string) ([][]float32, error)
	CreateEmbeddingWithModel(documents []string, model string) ([][]float32, error)
}

func MapToApi(inmap map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range inmap {
		result[k] = v
	}
	return result
}

func MapListToApi(inmap []map[string]string) []map[string]interface{} {
	result := make([]map[string]interface{}, len(inmap))
	for i, v := range inmap {
		result[i] = MapToApi(v)
	}
	return result
}

func MapFromApi(inmap map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range inmap {
		result[k] = v.(string)
	}
	return result
}

func MapListFromApi(inmap []map[string]interface{}) []map[string]string {
	result := make([]map[string]string, len(inmap))
	for i, v := range inmap {
		result[i] = MapFromApi(v)
	}
	return result
}

// Client represents the ChromaDB Client
type Client struct {
	ApiClient *openapiclient.APIClient
}

func NewClient(basePath string) *Client {
	configuration := openapiclient.NewConfiguration()
	configuration.Servers = openapiclient.ServerConfigurations{
		{
			URL:         basePath,
			Description: "No description provided",
		},
	}
	configuration.Debug = true
	apiClient := openapiclient.NewAPIClient(configuration)
	return &Client{
		ApiClient: apiClient,
	}
}

func (c *Client) GetCollection(collectionName string, embeddingFunction EmbeddingFunction) (*Collection, error) {
	col, httpResp, err := c.ApiClient.DefaultApi.GetCollection(context.Background(), collectionName).Execute()
	if err != nil {
		return nil, err
	}
	if httpResp.StatusCode != 200 {
		return nil, fmt.Errorf("error getting collection: %v", httpResp)
	}
	return NewCollection(c.ApiClient, col.Id, col.Name, col.Metadata, embeddingFunction), nil
}

func (c *Client) Heartbeat() (map[string]float32, error) {
	resp, _, err := c.ApiClient.DefaultApi.Heartbeat(context.Background()).Execute()
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

func (c *Client) CreateCollection(collectionName string, metadata map[string]interface{}, createOrGet bool, embeddingFunction EmbeddingFunction, distanceFunction DistanceFunction) (*Collection, error) {
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
	resp, _, err := c.ApiClient.DefaultApi.CreateCollection(context.Background()).CreateCollection(col).Execute()
	if err != nil {
		return nil, err
	}
	mtd := resp.Metadata
	return NewCollection(c.ApiClient, resp.Id, resp.Name, mtd, embeddingFunction), nil
}

func (c *Client) DeleteCollection(collectionName string) (*Collection, error) {
	col, httpResp, gcerr := c.ApiClient.DefaultApi.GetCollection(context.Background(), collectionName).Execute()
	if gcerr != nil {
		log.Fatal(httpResp, gcerr)
		return nil, gcerr
	}
	deletedCol, httpResp, err := c.ApiClient.DefaultApi.DeleteCollection(context.Background(), collectionName).Execute()
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

func (c *Client) Reset() (bool, error) {
	resp, _, err := c.ApiClient.DefaultApi.Reset(context.Background()).Execute()
	return resp, err
}

func (c *Client) ListCollections() ([]*Collection, error) {
	req := c.ApiClient.DefaultApi.ListCollections(context.Background())
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

func (c *Client) Version() (string, error) {
	resp, _, err := c.ApiClient.DefaultApi.Version(context.Background()).Execute()
	version := strings.Replace(resp, `"`, "", -1)
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
	ApiClient         *openapiclient.APIClient
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

func (c *Collection) Add(embeddings [][]float32, metadatas []map[string]interface{}, documents []string, ids []string) (*Collection, error) {

	var _embeddings []interface{}

	if len(embeddings) == 0 {
		embds, embErr := c.EmbeddingFunction.CreateEmbedding(documents)
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
	_, _, err := c.ApiClient.DefaultApi.Add(context.Background(), c.id).AddEmbedding(addEmbedding).Execute()
	if err != nil {
		return c, err
	}
	return c, nil
}

func (c *Collection) Upsert(embeddings [][]float32, metadatas []map[string]interface{}, documents []string, ids []string) (*Collection, error) {
	var _embeddings []interface{}

	if len(embeddings) == 0 {
		embds, embErr := c.EmbeddingFunction.CreateEmbedding(documents)
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

	_, _, err := c.ApiClient.DefaultApi.Upsert(context.Background(), c.id).AddEmbedding(addEmbedding).Execute()

	if err != nil {
		return c, err
	}
	return c, nil
}

func (c *Collection) Modify(embeddings [][]float32, metadatas []map[string]interface{}, documents []string, ids []string) (*Collection, error) {
	var _embeddings []interface{}

	if len(embeddings) == 0 {
		embds, embErr := c.EmbeddingFunction.CreateEmbedding(documents)
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

	_, _, err := c.ApiClient.DefaultApi.Update(context.Background(), c.id).UpdateEmbedding(updateEmbedding).Execute()

	if err != nil {
		return c, err
	}
	return c, nil
}

func (c *Collection) Get(where map[string]interface{}, whereDocuments map[string]interface{}, ids []string) (*Collection, error) {
	cd, _, err := c.ApiClient.DefaultApi.Get(context.Background(), c.id).GetEmbedding(openapiclient.GetEmbedding{
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
func (c *Collection) Query(queryTexts []string, nResults int32, where map[string]interface{}, whereDocuments map[string]interface{}, include []QueryEnum) (*QueryResults, error) {
	var _local_include []QueryEnum = include
	if len(include) == 0 {
		_local_include = []QueryEnum{documents, metadatas, distances}
	}
	_includes := make([]openapiclient.IncludeInner, len(_local_include))
	for i, v := range _local_include {
		_v := string(v)
		_includes[i] = openapiclient.IncludeInner{
			String: &_v,
		}
	}

	embds, embErr := c.EmbeddingFunction.CreateEmbedding(queryTexts)
	if embErr != nil {
		return nil, embErr
	}
	qr, _, err := c.ApiClient.DefaultApi.GetNearestNeighbors(context.Background(), c.id).QueryEmbedding(openapiclient.QueryEmbedding{
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

func (c *Collection) Count() (int32, error) {
	req := c.ApiClient.DefaultApi.Count(context.Background(), c.id)

	cd, _, err := req.Execute()

	if err != nil {
		return -1, err
	}

	return cd, nil
}

func (c *Collection) Update(newName string, newMetadata map[string]interface{}) (*Collection, error) {

	_, httpResp, err := c.ApiClient.DefaultApi.UpdateCollection(context.Background(), c.id).UpdateCollection(openapiclient.UpdateCollection{NewName: &newName, NewMetadata: newMetadata}).Execute()
	if err != nil {
		log.Fatal(httpResp, err)
		return c, err
	}
	c.Name = newName
	c.Metadata = newMetadata
	return c, nil
}

func (c *Collection) Delete(ids []string, where map[string]interface{}, whereDocuments map[string]interface{}) ([]string, error) {

	dr, httpResp, err := c.ApiClient.DefaultApi.Delete(context.Background(), c.id).DeleteEmbedding(openapiclient.DeleteEmbedding{Where: where, WhereDocument: whereDocuments, Ids: ids}).Execute()
	if err != nil {
		log.Fatal(httpResp, err)
		return nil, err
	}
	return dr, nil

}
