package api

type DocumentMetadata interface {
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
