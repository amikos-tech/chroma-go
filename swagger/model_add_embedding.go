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

// checks if the AddEmbedding type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &AddEmbedding{}

// AddEmbedding struct for AddEmbedding
type AddEmbedding struct {
	Embeddings []EmbeddingsInner        `json:"embeddings,omitempty"`
	Metadatas  []map[string]interface{} `json:"metadatas,omitempty"`
	Documents  []string                 `json:"documents,omitempty"`
	Ids        []string                 `json:"ids"`
}

// NewAddEmbedding instantiates a new AddEmbedding object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewAddEmbedding(ids []string) *AddEmbedding {
	this := AddEmbedding{}
	this.Ids = ids
	return &this
}

// NewAddEmbeddingWithDefaults instantiates a new AddEmbedding object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewAddEmbeddingWithDefaults() *AddEmbedding {
	this := AddEmbedding{}
	return &this
}

// GetEmbeddings returns the Embeddings field value if set, zero value otherwise.
func (o *AddEmbedding) GetEmbeddings() []EmbeddingsInner {
	if o == nil || IsNil(o.Embeddings) {
		var ret []EmbeddingsInner
		return ret
	}
	return o.Embeddings
}

// GetEmbeddingsOk returns a tuple with the Embeddings field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AddEmbedding) GetEmbeddingsOk() ([]EmbeddingsInner, bool) {
	if o == nil || IsNil(o.Embeddings) {
		return nil, false
	}
	return o.Embeddings, true
}

// HasEmbeddings returns a boolean if a field has been set.
func (o *AddEmbedding) HasEmbeddings() bool {
	if o != nil && !IsNil(o.Embeddings) {
		return true
	}

	return false
}

// SetEmbeddings gets a reference to the given []EmbeddingsInner and assigns it to the Embeddings field.
func (o *AddEmbedding) SetEmbeddings(v []EmbeddingsInner) {
	o.Embeddings = v
}

// GetMetadatas returns the Metadatas field value if set, zero value otherwise.
func (o *AddEmbedding) GetMetadatas() []map[string]interface{} {
	if o == nil || IsNil(o.Metadatas) {
		var ret []map[string]interface{}
		return ret
	}
	return o.Metadatas
}

// GetMetadatasOk returns a tuple with the Metadatas field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AddEmbedding) GetMetadatasOk() ([]map[string]interface{}, bool) {
	if o == nil || IsNil(o.Metadatas) {
		return nil, false
	}
	return o.Metadatas, true
}

// HasMetadatas returns a boolean if a field has been set.
func (o *AddEmbedding) HasMetadatas() bool {
	if o != nil && !IsNil(o.Metadatas) {
		return true
	}

	return false
}

// SetMetadatas gets a reference to the given []map[string]interface{} and assigns it to the Metadatas field.
func (o *AddEmbedding) SetMetadatas(v []map[string]interface{}) {
	o.Metadatas = v
}

// GetDocuments returns the Documents field value if set, zero value otherwise.
func (o *AddEmbedding) GetDocuments() []string {
	if o == nil || IsNil(o.Documents) {
		var ret []string
		return ret
	}
	return o.Documents
}

// GetDocumentsOk returns a tuple with the Documents field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AddEmbedding) GetDocumentsOk() ([]string, bool) {
	if o == nil || IsNil(o.Documents) {
		return nil, false
	}
	return o.Documents, true
}

// HasDocuments returns a boolean if a field has been set.
func (o *AddEmbedding) HasDocuments() bool {
	if o != nil && !IsNil(o.Documents) {
		return true
	}

	return false
}

// SetDocuments gets a reference to the given []string and assigns it to the Documents field.
func (o *AddEmbedding) SetDocuments(v []string) {
	o.Documents = v
}

// GetIds returns the Ids field value
func (o *AddEmbedding) GetIds() []string {
	if o == nil {
		var ret []string
		return ret
	}

	return o.Ids
}

// GetIdsOk returns a tuple with the Ids field value
// and a boolean to check if the value has been set.
func (o *AddEmbedding) GetIdsOk() ([]string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Ids, true
}

// SetIds sets field value
func (o *AddEmbedding) SetIds(v []string) {
	o.Ids = v
}

func (o AddEmbedding) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o AddEmbedding) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.Embeddings) {
		toSerialize["embeddings"] = o.Embeddings
	}
	if !IsNil(o.Metadatas) {
		toSerialize["metadatas"] = o.Metadatas
	}
	if !IsNil(o.Documents) {
		toSerialize["documents"] = o.Documents
	}
	toSerialize["ids"] = o.Ids
	return toSerialize, nil
}

type NullableAddEmbedding struct {
	value *AddEmbedding
	isSet bool
}

func (v NullableAddEmbedding) Get() *AddEmbedding {
	return v.value
}

func (v *NullableAddEmbedding) Set(val *AddEmbedding) {
	v.value = val
	v.isSet = true
}

func (v NullableAddEmbedding) IsSet() bool {
	return v.isSet
}

func (v *NullableAddEmbedding) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableAddEmbedding(val *AddEmbedding) *NullableAddEmbedding {
	return &NullableAddEmbedding{value: val, isSet: true}
}

func (v NullableAddEmbedding) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableAddEmbedding) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
