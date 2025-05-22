package v2

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/pkg/errors"
)

type CollectionModel struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	ConfigurationJSON map[string]interface{} `json:"configuration_json,omitempty"`
	Metadata          CollectionMetadata     `json:"metadata,omitempty"`
	Dimension         int                    `json:"dimension,omitempty"`
	Tenant            string                 `json:"tenant,omitempty"`
	Database          string                 `json:"database,omitempty"`
	Version           int                    `json:"version,omitempty"`
	LogPosition       int                    `json:"log_position,omitempty"`
}

func (op *CollectionModel) MarshalJSON() ([]byte, error) {
	type Alias CollectionModel
	return json.Marshal(struct{ *Alias }{Alias: (*Alias)(op)})
}

func (op *CollectionModel) UnmarshalJSON(b []byte) error {
	type Alias CollectionModel
	aux := &struct {
		*Alias
		Metadata CollectionMetadata `json:"metadata,omitempty"`
	}{Alias: (*Alias)(op), Metadata: NewMetadata()}
	err := json.Unmarshal(b, aux)
	if err != nil {
		return err
	}
	op.Metadata = aux.Metadata
	return nil
}

type CollectionImpl struct {
	name              string
	id                string
	tenant            Tenant
	database          Database
	metadata          CollectionMetadata
	dimension         int
	configuration     CollectionConfiguration
	client            *APIClientV2
	embeddingFunction embeddings.EmbeddingFunction
}

type Option func(*CollectionImpl) error

func (c *CollectionImpl) Name() string {
	return c.name
}

func (c *CollectionImpl) ID() string {
	return c.id
}

func (c *CollectionImpl) Tenant() Tenant {
	return c.tenant
}

func (c *CollectionImpl) Database() Database {
	return c.database
}

func (c *CollectionImpl) Dimension() int {
	return c.dimension
}

func (c *CollectionImpl) Configuration() CollectionConfiguration {
	return c.configuration
}

func (c *CollectionImpl) Add(ctx context.Context, opts ...CollectionAddOption) error {
	err := c.client.PreFlight(ctx)
	if err != nil {
		return errors.Wrap(err, "preflight failed")
	}
	addObject, err := NewCollectionAddOp(opts...)
	if err != nil {
		return errors.Wrap(err, "failed to create new collection update operation")
	}
	err = addObject.PrepareAndValidate()
	if err != nil {
		return errors.Wrap(err, "failed to prepare and validate collection update operation")
	}
	err = c.client.Satisfies(addObject, len(addObject.Ids), "documents")
	if err != nil {
		return errors.Wrap(err, "failed to satisfy collection update operation")
	}
	err = addObject.EmbedData(ctx, c.embeddingFunction)
	if err != nil {
		return errors.Wrap(err, "failed to embed data")
	}
	reqURL, err := url.JoinPath("tenants", c.Tenant().Name(), "databases", c.Database().Name(), "collections", c.ID(), "add")
	if err != nil {
		return errors.Wrap(err, "error composing request URL")
	}
	_, err = c.client.ExecuteRequest(ctx, http.MethodPost, reqURL, addObject)
	if err != nil {
		return errors.Wrap(err, "error sending request")
	}
	return nil
}

func (c *CollectionImpl) Upsert(ctx context.Context, opts ...CollectionAddOption) error {
	err := c.client.PreFlight(ctx)
	if err != nil {
		return err
	}
	upsertObject, err := NewCollectionAddOp(opts...)
	if err != nil {
		return err
	}
	err = upsertObject.PrepareAndValidate()
	if err != nil {
		return err
	}
	err = c.client.Satisfies(upsertObject, len(upsertObject.Ids), "documents")
	if err != nil {
		return err
	}
	err = upsertObject.EmbedData(ctx, c.embeddingFunction)
	if err != nil {
		return errors.Wrap(err, "failed to embed data")
	}
	reqURL, err := url.JoinPath("tenants", c.Tenant().Name(), "databases", c.Database().Name(), "collections", c.ID(), "upsert")
	if err != nil {
		return err
	}
	_, err = c.client.ExecuteRequest(ctx, http.MethodPost, reqURL, upsertObject)
	if err != nil {
		return err
	}
	return nil
}
func (c *CollectionImpl) Update(ctx context.Context, opts ...CollectionUpdateOption) error {
	err := c.client.PreFlight(ctx)
	if err != nil {
		return err
	}
	updateObject, err := NewCollectionUpdateOp(opts...)
	if err != nil {
		return err
	}
	err = updateObject.PrepareAndValidate()
	if err != nil {
		return err
	}
	err = c.client.Satisfies(updateObject, len(updateObject.Ids), "documents")
	if err != nil {
		return err
	}
	err = updateObject.EmbedData(ctx, c.embeddingFunction)
	if err != nil {
		return errors.Wrap(err, "failed to embed data")
	}
	reqURL, err := url.JoinPath("tenants", c.Tenant().Name(), "databases", c.Database().Name(), "collections", c.ID(), "update")
	if err != nil {
		return err
	}
	_, err = c.client.ExecuteRequest(ctx, http.MethodPost, reqURL, updateObject)
	if err != nil {
		return err
	}
	return nil
}
func (c *CollectionImpl) Delete(ctx context.Context, opts ...CollectionDeleteOption) error {
	err := c.client.PreFlight(ctx)
	if err != nil {
		return err
	}
	deleteObject, err := NewCollectionDeleteOp(opts...)
	if err != nil {
		return err
	}
	err = deleteObject.PrepareAndValidate()
	if err != nil {
		return err
	}
	err = c.client.Satisfies(deleteObject, len(deleteObject.Ids), "documents")
	if err != nil {
		return err
	}
	reqURL, err := url.JoinPath("tenants", c.Tenant().Name(), "databases", c.Database().Name(), "collections", c.ID(), "delete")
	if err != nil {
		return err
	}
	_, err = c.client.ExecuteRequest(ctx, http.MethodPost, reqURL, deleteObject)
	if err != nil {
		return err
	}
	return nil
}
func (c *CollectionImpl) Count(ctx context.Context) (int, error) {
	reqURL, err := url.JoinPath("tenants", c.Tenant().Name(), "databases", c.Database().Name(), "collections", c.ID(), "count")
	if err != nil {
		return 0, errors.Wrap(err, "error composing request URL")
	}
	respBody, err := c.client.ExecuteRequest(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, errors.Wrap(err, "error getting collection count")
	}
	return strconv.Atoi(string(respBody))
}
func (c *CollectionImpl) ModifyName(ctx context.Context, newName string) error {
	// TODO better name validation
	if newName == "" {
		return errors.New("newName cannot be empty")
	}
	reqURL, err := url.JoinPath("tenants", c.Tenant().Name(), "databases", c.Database().Name(), "collections", c.ID())

	if err != nil {
		return errors.Wrap(err, "error composing request URL")
	}
	_, err = c.client.ExecuteRequest(ctx, http.MethodPut, reqURL, map[string]string{"new_name": newName})
	if err != nil {
		return errors.Wrap(err, "error modifying collection name")
	}
	return nil
}
func (c *CollectionImpl) ModifyMetadata(ctx context.Context, newMetadata CollectionMetadata) error {
	if newMetadata == nil {
		return errors.New("newMetadata cannot be nil")
	}
	reqURL, err := url.JoinPath("tenants", c.Tenant().Name(), "databases", c.Database().Name(), "collections", c.ID())
	if err != nil {
		return err
	}
	_, err = c.client.ExecuteRequest(ctx, http.MethodPut, reqURL, map[string]interface{}{"new_metadata": newMetadata})
	if err != nil {
		return err
	}
	return nil
}
func (c *CollectionImpl) Get(ctx context.Context, opts ...CollectionGetOption) (GetResult, error) {
	getObject, err := NewCollectionGetOp(opts...)
	if err != nil {
		return nil, err
	}
	err = getObject.PrepareAndValidate()
	if err != nil {
		return nil, err
	}
	reqURL, err := url.JoinPath("tenants", c.Tenant().Name(), "databases", c.Database().Name(), "collections", c.ID(), "get")
	if err != nil {
		return nil, err
	}
	respBody, err := c.client.ExecuteRequest(ctx, http.MethodPost, reqURL, getObject)
	if err != nil {
		return nil, errors.Wrap(err, "error getting collection")
	}
	getResult := &GetResultImpl{}
	err = json.Unmarshal(respBody, getResult)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshalling get result")
	}
	return getResult, nil
}
func (c *CollectionImpl) Query(ctx context.Context, opts ...CollectionQueryOption) (QueryResult, error) {
	querybject, err := NewCollectionQueryOp(opts...)
	if err != nil {
		return nil, errors.Wrap(err, "error creating new collection query operation")
	}
	err = querybject.PrepareAndValidate()
	if err != nil {
		return nil, errors.Wrap(err, "error validating query object")
	}
	err = querybject.EmbedData(ctx, c.embeddingFunction)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to embed data")
	}
	reqURL, err := url.JoinPath("tenants", c.Tenant().Name(), "databases", c.Database().Name(), "collections", c.ID(), "query")
	if err != nil {
		return nil, errors.Wrap(err, "error building query url")
	}
	respBody, err := c.client.ExecuteRequest(ctx, http.MethodPost, reqURL, querybject)
	if err != nil {
		return nil, errors.Wrap(err, "error sending query request")
	}
	queryResult := &QueryResultImpl{}
	err = json.Unmarshal(respBody, queryResult)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshalling query result")
	}
	return queryResult, nil
}

func (c *CollectionImpl) ModifyConfiguration(ctx context.Context, newConfig CollectionConfiguration) error {
	return errors.New("not yet supported")
}

func (c *CollectionImpl) Metadata() CollectionMetadata {
	return c.metadata
}

func (c *CollectionImpl) Close() error {
	if c.embeddingFunction != nil {
		if closer, ok := c.embeddingFunction.(io.Closer); ok {
			return closer.Close()
		}
	}
	return nil
}

// TODO add utility methods for metadata lookups
