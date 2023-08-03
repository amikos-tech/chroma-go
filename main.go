package chroma_go

import (
	"context"
	"encoding/json"
	"fmt"
	openapiclient "github.com/amikos-tech/chroma-go/swagger"
	"reflect"
	"strings"
)

// function to create collection
// Path: main.go
func main() {

}

type ClientConfiguration struct {
	BasePath          string            `json:"basePath,omitempty"`
	DefaultHeaders    map[string]string `json:"defaultHeader,omitempty"`
	EmbeddingFunction EmbeddingFunction `json:"embeddingFunction,omitempty"`
}

type EmbeddingFunction interface {
	CreateEmbedding(documents []string) ([][]float32, error)
	CreateEmbeddingWithModel(documents []string, model string) ([][]float32, error)
}

// Client represents the ChromaDB Client
type Client struct {
	ApiClient *openapiclient.APIClient
}

func NewClient(basePath string) *Client {
	configuration := openapiclient.NewConfiguration()
	configuration.BasePath = basePath
	apiClient := openapiclient.NewAPIClient(configuration)
	return &Client{
		ApiClient: apiClient,
	}
}

func (c *Client) GetCollection(collectionName string, embeddingFunction EmbeddingFunction) (*Collection, error) {
	// Implementation here
	return nil, nil
}

func (c *Client) Heartbeat() (map[string]float64, error) {
	resp, httpResp, err := c.ApiClient.DefaultApi.Heartbeat(context.Background())
	fmt.Printf("Heartbeat: %v\n", httpResp)
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

func (c *Client) CreateCollection(collectionName string, metadata map[string]string, createOrGet bool, embeddingFunction EmbeddingFunction, distanceFunction DistanceFunction) (*Collection, error) {
	_metadata := metadata

	if _metadata == nil || len(_metadata) == 0 {
		_metadata = make(map[string]string)
	}
	if _metadata["embedding_function"] == "" {
		_metadata["embedding_function"] = GetStringTypeOfEmbeddingFunction(embeddingFunction)
	}
	if distanceFunction == "" {
		_metadata["hnsw:space"] = strings.ToLower(string(L2))
	} else {
		_metadata["hnsw:space"] = strings.ToLower(string(distanceFunction))
	}

	col := openapiclient.CreateCollection{
		Name:        collectionName,
		GetOrCreate: createOrGet,
		Metadata:    _metadata,
	}
	resp, httpResp, err := c.ApiClient.DefaultApi.CreateCollection(context.Background(), col)
	if err != nil {
		return nil, err
	}
	fmt.Printf("CreateCollection: %v\n", httpResp.Body)
	respJSON, _ := json.Marshal(resp)
	fmt.Println(string(respJSON))
	mtd := *resp.Metadata
	return NewCollection(c.ApiClient, resp.Id, resp.Name, mtd, embeddingFunction), nil
}

func (c *Client) DeleteCollection(collectionName string) (*Collection, error) {
	// Implementation here
	return nil, nil
}

func (c *Client) Upsert(collectionName string, ef EmbeddingFunction) (*Collection, error) {
	// Implementation here
	return nil, nil
}

func (c *Client) Reset() (bool, error) {
	resp, httpResp, err := c.ApiClient.DefaultApi.Reset(context.Background())
	fmt.Printf("Reset: %v\n", httpResp)
	return resp, err
}

func (c *Client) ListCollections() ([]*Collection, error) {
	// Implementation here
	return nil, nil
}

func (c *Client) Version() (string, error) {
	// Implementation here
	return "", nil
}

type CollectionData struct {
	Ids       []string            `json:"ids,omitempty"`
	Documents []string            `json:"documents,omitempty"`
	Metadatas []map[string]string `json:"metadatas,omitempty"`
}

type Collection struct {
	Name              string
	EmbeddingFunction EmbeddingFunction
	ApiClient         *openapiclient.APIClient
	Metadata          map[string]string
	id                string
	CollectionData    *CollectionData
}

func NewCollection(apiClient *openapiclient.APIClient, id string, name string, metadata map[string]string, embeddingFunction EmbeddingFunction) *Collection {
	return &Collection{
		Name:              name,
		EmbeddingFunction: embeddingFunction,
		ApiClient:         apiClient,
		Metadata:          metadata,
		id:                id,
	}
}

func (c *Collection) Add(embeddings [][]float32, metadatas []map[string]string, documents []string, ids []string) (*Collection, error) {
	req := openapiclient.AddEmbedding{
		Embeddings: embeddings,
		Metadatas:  metadatas,
		Documents:  documents,
		Ids:        ids,
	}

	if len(embeddings) == 0 {
		embds, embErr := c.EmbeddingFunction.CreateEmbedding(documents)
		if embErr != nil {
			return c, embErr
		}
		req.Embeddings = embds
	}

	_, httpResp, err := c.ApiClient.DefaultApi.Add(context.Background(), req, c.id)

	if err != nil {
		return c, err
	}
	fmt.Printf("Add: %v\n", httpResp)
	return c, nil
}

func (c *Collection) Get(where map[string]string, whereDocuments map[string]string, ids []string) (*Collection, error) {
	req := openapiclient.GetEmbedding{
		Ids:           ids,
		Where:         where,
		WhereDocument: whereDocuments,
	}

	cd, httpResp, err := c.ApiClient.DefaultApi.Get(context.Background(), req, c.id)

	if err != nil {
		return c, err
	}
	cdata := CollectionData{
		Ids:       cd.Ids,
		Documents: cd.Documents,
		Metadatas: cd.Metadatas,
	}
	c.CollectionData = &cdata
	fmt.Printf("Add: %v\n", httpResp)
	return c, nil
}
