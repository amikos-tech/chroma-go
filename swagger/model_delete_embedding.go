/*
ChromaDB API

This is OpenAPI schema for ChromaDB API.

API version: 1.0.0
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package openapi

import (
	"encoding/json"
)

// checks if the DeleteEmbedding type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &DeleteEmbedding{}

// DeleteEmbedding struct for DeleteEmbedding
type DeleteEmbedding struct {
	Ids           []string               `json:"ids,omitempty"`
	Where         map[string]interface{} `json:"where,omitempty"`
	WhereDocument map[string]interface{} `json:"where_document,omitempty"`
}

// NewDeleteEmbedding instantiates a new DeleteEmbedding object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewDeleteEmbedding() *DeleteEmbedding {
	this := DeleteEmbedding{}
	return &this
}

// NewDeleteEmbeddingWithDefaults instantiates a new DeleteEmbedding object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewDeleteEmbeddingWithDefaults() *DeleteEmbedding {
	this := DeleteEmbedding{}
	return &this
}

// GetIds returns the Ids field value if set, zero value otherwise.
func (o *DeleteEmbedding) GetIds() []string {
	if o == nil || IsNil(o.Ids) {
		var ret []string
		return ret
	}
	return o.Ids
}

// GetIdsOk returns a tuple with the Ids field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DeleteEmbedding) GetIdsOk() ([]string, bool) {
	if o == nil || IsNil(o.Ids) {
		return nil, false
	}
	return o.Ids, true
}

// HasIds returns a boolean if a field has been set.
func (o *DeleteEmbedding) HasIds() bool {
	if o != nil && !IsNil(o.Ids) {
		return true
	}

	return false
}

// SetIds gets a reference to the given []string and assigns it to the Ids field.
func (o *DeleteEmbedding) SetIds(v []string) {
	o.Ids = v
}

// GetWhere returns the Where field value if set, zero value otherwise.
func (o *DeleteEmbedding) GetWhere() map[string]interface{} {
	if o == nil || IsNil(o.Where) {
		var ret map[string]interface{}
		return ret
	}
	return o.Where
}

// GetWhereOk returns a tuple with the Where field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DeleteEmbedding) GetWhereOk() (map[string]interface{}, bool) {
	if o == nil || IsNil(o.Where) {
		return map[string]interface{}{}, false
	}
	return o.Where, true
}

// HasWhere returns a boolean if a field has been set.
func (o *DeleteEmbedding) HasWhere() bool {
	if o != nil && !IsNil(o.Where) {
		return true
	}

	return false
}

// SetWhere gets a reference to the given map[string]interface{} and assigns it to the Where field.
func (o *DeleteEmbedding) SetWhere(v map[string]interface{}) {
	o.Where = v
}

// GetWhereDocument returns the WhereDocument field value if set, zero value otherwise.
func (o *DeleteEmbedding) GetWhereDocument() map[string]interface{} {
	if o == nil || IsNil(o.WhereDocument) {
		var ret map[string]interface{}
		return ret
	}
	return o.WhereDocument
}

// GetWhereDocumentOk returns a tuple with the WhereDocument field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DeleteEmbedding) GetWhereDocumentOk() (map[string]interface{}, bool) {
	if o == nil || IsNil(o.WhereDocument) {
		return map[string]interface{}{}, false
	}
	return o.WhereDocument, true
}

// HasWhereDocument returns a boolean if a field has been set.
func (o *DeleteEmbedding) HasWhereDocument() bool {
	if o != nil && !IsNil(o.WhereDocument) {
		return true
	}

	return false
}

// SetWhereDocument gets a reference to the given map[string]interface{} and assigns it to the WhereDocument field.
func (o *DeleteEmbedding) SetWhereDocument(v map[string]interface{}) {
	o.WhereDocument = v
}

func (o DeleteEmbedding) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o DeleteEmbedding) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.Ids) {
		toSerialize["ids"] = o.Ids
	}
	if !IsNil(o.Where) {
		toSerialize["where"] = o.Where
	}
	if !IsNil(o.WhereDocument) {
		toSerialize["where_document"] = o.WhereDocument
	}
	return toSerialize, nil
}

type NullableDeleteEmbedding struct {
	value *DeleteEmbedding
	isSet bool
}

func (v NullableDeleteEmbedding) Get() *DeleteEmbedding {
	return v.value
}

func (v *NullableDeleteEmbedding) Set(val *DeleteEmbedding) {
	v.value = val
	v.isSet = true
}

func (v NullableDeleteEmbedding) IsSet() bool {
	return v.isSet
}

func (v *NullableDeleteEmbedding) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableDeleteEmbedding(val *DeleteEmbedding) *NullableDeleteEmbedding {
	return &NullableDeleteEmbedding{value: val, isSet: true}
}

func (v NullableDeleteEmbedding) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableDeleteEmbedding) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}