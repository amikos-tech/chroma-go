package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/amikos-tech/chroma-go/pkg/api"
	chhttp "github.com/amikos-tech/chroma-go/pkg/commons/http"
)

type CollectionModel struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	ConfigurationJSON map[string]interface{} `json:"configuration_json,omitempty"`
	Metadata          api.CollectionMetadata `json:"metadata,omitempty"`
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
		Metadata api.CollectionMetadata `json:"metadata,omitempty"`
	}{Alias: (*Alias)(op), Metadata: api.NewMetadata()}
	err := json.Unmarshal(b, aux)
	if err != nil {
		return err
	}
	op.Metadata = aux.Metadata
	return nil
}

type Collection struct {
	api.CollectionBase
	client *APIClientV1
}

type Option func(*Collection) error

func (c *Collection) Name() string {
	return c.CollectionBase.Name
}

func (c *Collection) ID() string {
	return c.CollectionBase.CollectionID
}

func (c *Collection) Tenant() api.Tenant {
	return c.CollectionBase.Tenant
}

func (c *Collection) Database() api.Database {
	return c.CollectionBase.Database
}

func (c *Collection) Configuration() api.CollectionConfiguration {
	return c.CollectionBase.Configuration
}

func (c *Collection) Add(ctx context.Context, opts ...api.CollectionUpdateOption) error {
	err := c.client.PreFlight(ctx)
	if err != nil {
		return err
	}
	addObject, err := api.NewCollectionUpdateOp(opts...)
	if err != nil {
		return err
	}
	err = addObject.PrepareAndValidate()
	if err != nil {
		return err
	}
	err = c.client.Satisfies(addObject, len(addObject.Ids), "documents")
	if err != nil {
		return err
	}
	reqURL, err := url.JoinPath(c.client.BaseURL(), "collections", c.ID(), "add")
	if err != nil {
		return err
	}

	reqJSON, err := addObject.MarshalJSON()
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(reqJSON))
	if err != nil {
		return err
	}
	_, err = c.client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return err
	}
	return nil
}

func (c *Collection) Upsert(ctx context.Context, opts ...api.CollectionUpdateOption) error {
	err := c.client.PreFlight(ctx)
	if err != nil {
		return err
	}
	upsertObject, err := api.NewCollectionUpdateOp(opts...)
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
	reqURL, err := url.JoinPath(c.client.BaseURL(), "collections", c.ID(), "upsert")
	if err != nil {
		return err
	}
	reqJSON, err := upsertObject.MarshalJSON()
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(reqJSON))
	if err != nil {
		return err
	}
	_, err = c.client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return err
	}
	return nil
}
func (c *Collection) Update(ctx context.Context, opts ...api.CollectionUpdateOption) error {
	err := c.client.PreFlight(ctx)
	if err != nil {
		return err
	}
	updateObject, err := api.NewCollectionUpdateOp(opts...)
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
	reqURL, err := url.JoinPath(c.client.BaseURL(), "collections", c.ID(), "update")
	if err != nil {
		return err
	}
	reqJSON, err := updateObject.MarshalJSON()
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(reqJSON))
	if err != nil {
		return err
	}
	_, err = c.client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return err
	}
	return nil
}
func (c *Collection) Delete(ctx context.Context, opts ...api.CollectionDeleteOption) error {
	err := c.client.PreFlight(ctx)
	if err != nil {
		return err
	}
	deleteObject, err := api.NewCollectionDeleteOp(opts...)
	if err != nil {
		return err
	}
	err = deleteObject.Validate()
	if err != nil {
		return err
	}
	err = c.client.Satisfies(deleteObject, len(deleteObject.Ids), "documents")
	if err != nil {
		return err
	}
	reqURL, err := url.JoinPath(c.client.BaseURL(), "collections", c.ID(), "delete")
	if err != nil {
		return err
	}
	reqJSON, err := deleteObject.MarshalJSON()
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(reqJSON))
	if err != nil {
		return err
	}
	_, err = c.client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return err
	}
	return nil
}
func (c *Collection) Count(ctx context.Context) (int, error) {
	reqURL, err := url.JoinPath(c.client.BaseURL(), "collections", c.ID(), "count")
	if err != nil {
		return 0, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := c.client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return 0, err
	}
	respBody := chhttp.ReadRespBody(resp.Body)

	return strconv.Atoi(respBody)
}
func (c *Collection) ModifyName(ctx context.Context, newName string) error {
	if newName == "" {
		return errors.New("newName cannot be empty")
	}

	reqURL, err := url.JoinPath(c.client.BaseURL(), "collections", c.ID())
	if err != nil {
		return err
	}

	reqJSON, err := json.Marshal(map[string]string{"new_name": newName})
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, bytes.NewReader(reqJSON))
	if err != nil {
		return err
	}
	_, err = c.client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return err
	}

	return nil
}
func (c *Collection) ModifyMetadata(ctx context.Context, newMetadata api.CollectionMetadata) error {
	if newMetadata == nil {
		return errors.New("newMetadata cannot be nil")
	}
	reqURL, err := url.JoinPath(c.client.BaseURL(), "collections", c.ID())
	if err != nil {
		return err
	}
	reqJSON, err := json.Marshal(map[string]interface{}{"new_metadata": newMetadata})
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, bytes.NewReader(reqJSON))
	if err != nil {
		return err
	}
	_, err = c.client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return err
	}
	return nil
}
func (c *Collection) Get(ctx context.Context, opts ...api.CollectionGetOption) (api.GetResult, error) {
	getObject, err := api.NewCollectionGetOp(opts...)
	if err != nil {
		return nil, err
	}
	err = getObject.Validate()
	if err != nil {
		return nil, err
	}
	reqURL, err := url.JoinPath(c.client.BaseURL(), "collections", c.ID())
	if err != nil {
		return nil, err
	}
	reqJSON, err := getObject.MarshalJSON()
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, err
	}
	_, err = c.client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return nil, err
	}
	// TODO: Implement GetResult
	return nil, nil
}
func (c *Collection) Query(ctx context.Context, opts ...api.CollectionQueryOption) (api.QueryResult, error) {
	querybject, err := api.NewCollectionQueryOp(opts...)
	if err != nil {
		return nil, err
	}
	err = querybject.Validate()
	if err != nil {
		return nil, err
	}
	reqURL, err := url.JoinPath(c.client.BaseURL(), "collections", c.ID(), "query")
	if err != nil {
		return nil, err
	}
	reqJSON, err := querybject.MarshalJSON()
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, err
	}
	_, err = c.client.BaseAPIClient.SendRequest(httpReq)
	if err != nil {
		return nil, err
	}
	// TODO: Implement QueryResult
	return nil, nil
}

func (c *Collection) ModifyConfiguration(ctx context.Context, newConfig api.CollectionConfiguration) error {
	return errors.New("not supported")
}

func (c *Collection) Metadata() api.CollectionMetadata {
	return c.CollectionBase.Metadata
}
