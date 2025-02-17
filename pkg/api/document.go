package api

import "encoding/json"

type DocumentMetadata interface {
	GetRaw(key string) (interface{}, bool)
	GetString(key string) (string, bool)
	GetInt(key string) (int64, bool)
	GetFloat(key string) (float64, bool)
	GetBool(key string) (bool, bool)
	SetRaw(key string, value interface{})
	SetString(key, value string)
	SetInt(key string, value int64)
	SetFloat(key string, value float64)
	SetBool(key string, value bool)
}

type DocumentID string

type Document interface {
	ContentRaw() []byte
	ContentString() string
}

type TextDocument struct {
	Content string
}

func NewTextDocument(content string) *TextDocument {
	return &TextDocument{Content: content}
}

func (d *TextDocument) ContentRaw() []byte {
	return []byte(d.Content)
}

func (d *TextDocument) ContentString() string {
	return d.Content
}

func (d *TextDocument) UnmarshalJSON(data []byte) error {
	d.Content = string(data)
	return nil
}

func (d *TextDocument) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.Content + `"`), nil
}

type DocumentMetadataImpl struct {
	metadata map[string]MetadataValue
}

func NewDocumentMetadata(metadata map[string]interface{}) DocumentMetadata {
	if metadata == nil {
		return NewMetadata()
	}

	mv := &DocumentMetadataImpl{metadata: make(map[string]MetadataValue)}

	for k, v := range metadata {
		switch val := v.(type) {
		case bool:
			mv.SetBool(k, val)
		case float32:
			mv.SetFloat(k, float64(val))
		case float64:
			mv.SetFloat(k, val)
		case int:
			mv.SetInt(k, int64(val))
		case int32:
			mv.SetInt(k, int64(val))
		case int64:
			mv.SetInt(k, val)
		case string:
			mv.SetString(k, val)
		}
	}
	return mv
}

func (cm *DocumentMetadataImpl) Keys() []string {
	keys := make([]string, 0, len(cm.metadata))
	for k := range cm.metadata {
		keys = append(keys, k)
	}
	return keys
}

func (cm *DocumentMetadataImpl) GetRaw(key string) (value interface{}, ok bool) {
	v, ok := cm.metadata[key]
	return v, ok
}

func (cm *DocumentMetadataImpl) GetString(key string) (value string, ok bool) {
	v, ok := cm.metadata[key]
	if !ok {
		return "", false
	}
	str, ok := v.GetString()
	return str, ok
}

func (cm *DocumentMetadataImpl) GetInt(key string) (value int64, ok bool) {
	v, ok := cm.metadata[key]
	if !ok {
		return 0, false
	}
	i, ok := v.GetInt()
	return i, ok
}

func (cm *DocumentMetadataImpl) GetFloat(key string) (value float64, ok bool) {
	v, ok := cm.metadata[key]
	if !ok {
		return 0, false
	}
	f, ok := v.GetFloat()
	return f, ok
}

func (cm *DocumentMetadataImpl) GetBool(key string) (value bool, ok bool) {
	v, ok := cm.metadata[key]
	if !ok {
		return false, false
	}
	b, ok := v.GetBool()
	return b, ok
}

func (cm *DocumentMetadataImpl) SetRaw(key string, value interface{}) {
	switch val := value.(type) {
	case bool:
		cm.metadata[key] = MetadataValue{Bool: &val}
	case float32:
		var f64 = float64(val)
		cm.metadata[key] = MetadataValue{Float64: &f64}
	case float64:
		cm.metadata[key] = MetadataValue{Float64: &val}
	case int:
		tv := int64(val)
		cm.metadata[key] = MetadataValue{Int: &tv}
	case int32:
		tv := int64(val)
		cm.metadata[key] = MetadataValue{Int: &tv}
	case int64:
		cm.metadata[key] = MetadataValue{Int: &val}
	case string:
		cm.metadata[key] = MetadataValue{StringValue: &val}
	}
}

func (cm *DocumentMetadataImpl) SetString(key, value string) {
	cm.metadata[key] = MetadataValue{StringValue: &value}
}

func (cm *DocumentMetadataImpl) SetInt(key string, value int64) {
	cm.metadata[key] = MetadataValue{Int: &value}
}

func (cm *DocumentMetadataImpl) SetFloat(key string, value float64) {
	cm.metadata[key] = MetadataValue{Float64: &value}
}

func (cm *DocumentMetadataImpl) SetBool(key string, value bool) {
	cm.metadata[key] = MetadataValue{Bool: &value}
}

func (cm *DocumentMetadataImpl) MarshalJSON() ([]byte, error) {
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

func (cm *DocumentMetadataImpl) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &cm.metadata)
	if err != nil {
		return err
	}
	if cm.metadata == nil {
		cm.metadata = make(map[string]MetadataValue)
	}
	return nil
}
