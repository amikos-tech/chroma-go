package api

import (
	"encoding/json"
	"fmt"
)

type CollectionMetadata interface {
	Keys() []string
	GetRaw(key string) (interface{}, bool)
	GetString(key string) (string, bool)
	GetInt(key string) (int, bool)
	GetFloat(key string) (float64, bool)
	GetBool(key string) (bool, bool)
	SetRaw(key string, value interface{})
	SetString(key, value string)
	SetInt(key string, value int)
	SetFloat(key string, value float64)
	SetBool(key string, value bool)
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(b []byte) error
}

type MetadataValue struct {
	Bool        *bool
	Float32     *float64
	Int         *int
	StringValue *string
}

func (mv *MetadataValue) GetInt() (int, bool) {
	if mv.Int == nil {
		return 0, false
	}
	return int(*mv.Int), true
}

func (mv *MetadataValue) String() string {
	if mv.Bool != nil {
		return fmt.Sprintf("%v", *mv.Bool)
	}
	if mv.Float32 != nil {
		return fmt.Sprintf("%v", *mv.Float32)
	}
	if mv.Int != nil {
		return fmt.Sprintf("%v", *mv.Int)
	}
	if mv.StringValue != nil {
		return *mv.StringValue
	}
	return ""
}

func (mv *MetadataValue) GetFloat() (float64, bool) {
	if mv.Float32 == nil {
		return 0, false
	}
	return *mv.Float32, true
}

func (mv *MetadataValue) GetBool() (bool, bool) {
	if mv.Bool == nil {
		return false, false
	}
	return *mv.Bool, true
}

func (mv *MetadataValue) GetString() (string, bool) {
	if mv.StringValue == nil {
		return "", false
	}
	return *mv.StringValue, true
}

func (mv *MetadataValue) GetRaw() (interface{}, bool) {
	if mv.Bool != nil {
		return *mv.Bool, true
	}
	if mv.Float32 != nil {
		return *mv.Float32, true
	}
	if mv.Int != nil {
		return *mv.Int, true
	}
	if mv.StringValue != nil {
		return *mv.StringValue, true
	}
	return nil, false
}

func (mv *MetadataValue) MarshalJSON() ([]byte, error) {
	if mv.Bool != nil {
		return json.Marshal(mv.Bool)
	}
	if mv.Float32 != nil {
		return json.Marshal(mv.Float32)
	}
	if mv.Int != nil {
		return json.Marshal(mv.Int)
	}
	if mv.StringValue != nil {
		return json.Marshal(mv.StringValue)
	}
	return json.Marshal(nil)
}

func (mv *MetadataValue) UnmarshalJSON(b []byte) error {
	var err error
	// try to unmarshal JSON data into Bool
	err = json.Unmarshal(b, &mv.Bool)
	if err == nil {
		jsonBool, _ := json.Marshal(mv.Bool)
		if string(jsonBool) == "{}" { // empty struct
			mv.Bool = nil
		} else {
			return nil // data stored in dst.Bool, return on the first match
		}
	} else {
		mv.Bool = nil
	}

	err = json.Unmarshal(b, &mv.Int)
	if err == nil {
		jsonInt, _ := json.Marshal(mv.Int)
		if string(jsonInt) == "{}" { // empty struct
			mv.Int = nil
		} else {
			return nil // data stored in dst.Bool, return on the first match
		}
	} else {
		mv.Int = nil
	}

	err = json.Unmarshal(b, &mv.Float32)
	if err == nil {
		jsonFloat32, _ := json.Marshal(mv.Float32)
		if string(jsonFloat32) == "{}" { // empty struct
			mv.Float32 = nil
		} else {
			return nil // data stored in dst.Bool, return on the first match
		}
	} else {
		mv.Float32 = nil
	}

	err = json.Unmarshal(b, &mv.StringValue)
	if err == nil {
		jsonString, _ := json.Marshal(mv.StringValue)
		if string(jsonString) == "{}" { // empty struct
			mv.StringValue = nil
		} else {
			return nil // data stored in dst.Bool, return on the first match
		}
	} else {
		mv.StringValue = nil
	}

	return fmt.Errorf("data failed to match schemas in anyOf(Metadata)")
}

// Collection metadata
type CollectionMetadataImpl struct {
	metadata map[string]MetadataValue
}

func NewMetadata() CollectionMetadata {
	return &CollectionMetadataImpl{metadata: map[string]MetadataValue{}}
}

func NewMetadataFromMap(metadata map[string]interface{}) CollectionMetadata {
	if metadata == nil {
		return NewMetadata()
	}

	mv := &CollectionMetadataImpl{metadata: make(map[string]MetadataValue)}

	for k, v := range metadata {
		switch val := v.(type) {
		case bool:
			mv.SetBool(k, val)
		case float32:
			mv.SetFloat(k, float64(val))
		case float64:
			mv.SetFloat(k, val)
		case int:
			mv.SetInt(k, val)
		case int32:
			mv.SetInt(k, int(val))
		case int64:
			mv.SetInt(k, int(val))
		case string:
			mv.SetString(k, val)
		}
	}
	return mv
}

func (cm *CollectionMetadataImpl) Keys() []string {
	keys := make([]string, 0, len(cm.metadata))
	for k := range cm.metadata {
		keys = append(keys, k)
	}
	return keys
}

func (cm *CollectionMetadataImpl) GetRaw(key string) (value interface{}, ok bool) {
	v, ok := cm.metadata[key]
	return v, ok
}

func (cm *CollectionMetadataImpl) GetString(key string) (value string, ok bool) {
	v, ok := cm.metadata[key]
	if !ok {
		return "", false
	}
	str, ok := v.GetString()
	return str, ok
}

func (cm *CollectionMetadataImpl) GetInt(key string) (value int, ok bool) {
	v, ok := cm.metadata[key]
	if !ok {
		return 0, false
	}
	i, ok := v.GetInt()
	return i, ok
}

func (cm *CollectionMetadataImpl) GetFloat(key string) (value float64, ok bool) {
	v, ok := cm.metadata[key]
	if !ok {
		return 0, false
	}
	f, ok := v.GetFloat()
	return f, ok
}

func (cm *CollectionMetadataImpl) GetBool(key string) (value bool, ok bool) {
	v, ok := cm.metadata[key]
	if !ok {
		return false, false
	}
	b, ok := v.GetBool()
	return b, ok
}

func (cm *CollectionMetadataImpl) SetRaw(key string, value interface{}) {
	switch val := value.(type) {
	case bool:
		cm.metadata[key] = MetadataValue{Bool: &val}
	case float32:
		var f64 = float64(val)
		cm.metadata[key] = MetadataValue{Float32: &f64}
	case float64:
		cm.metadata[key] = MetadataValue{Float32: &val}
	case int:
		cm.metadata[key] = MetadataValue{Int: &val}
	case string:
		cm.metadata[key] = MetadataValue{StringValue: &val}
	}
}

func (cm *CollectionMetadataImpl) SetString(key, value string) {
	cm.metadata[key] = MetadataValue{StringValue: &value}
}

func (cm *CollectionMetadataImpl) SetInt(key string, value int) {
	cm.metadata[key] = MetadataValue{Int: &value}
}

func (cm *CollectionMetadataImpl) SetFloat(key string, value float64) {
	cm.metadata[key] = MetadataValue{Float32: &value}
}

func (cm *CollectionMetadataImpl) SetBool(key string, value bool) {
	cm.metadata[key] = MetadataValue{Bool: &value}
}

func (cm *CollectionMetadataImpl) MarshalJSON() ([]byte, error) {
	processed := make(map[string]interface{})
	for k, v := range cm.metadata {
		switch val, _ := v.GetRaw(); val.(type) {
		case bool:
			processed[k], _ = v.GetBool()
		case float32:
			processed[k], _ = v.GetFloat()
		case float64:
			processed[k], _ = v.GetFloat()
		case int:
			processed[k], _ = v.GetInt()
		case string:
			processed[k], _ = v.GetString()
		}
	}
	j, err := json.Marshal(processed)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (cm *CollectionMetadataImpl) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &cm.metadata)
	if err != nil {
		return err
	}
	if cm.metadata == nil {
		cm.metadata = make(map[string]MetadataValue)
	}
	return nil
}
