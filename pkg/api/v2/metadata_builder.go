package v2

// MetadataBuilder provides a fluent interface for building metadata
type MetadataBuilder struct {
	metadata CollectionMetadata
}

// Builder creates a new MetadataBuilder
func Builder() *MetadataBuilder {
	return &MetadataBuilder{
		metadata: NewMetadata(),
	}
}

// String adds a string value to the metadata
func (b *MetadataBuilder) String(key, value string) *MetadataBuilder {
	b.metadata.SetString(key, value)
	return b
}

// Int adds an integer value to the metadata
func (b *MetadataBuilder) Int(key string, value int64) *MetadataBuilder {
	b.metadata.SetInt(key, value)
	return b
}

// Float adds a float value to the metadata
func (b *MetadataBuilder) Float(key string, value float64) *MetadataBuilder {
	b.metadata.SetFloat(key, value)
	return b
}

// Bool adds a boolean value to the metadata
func (b *MetadataBuilder) Bool(key string, value bool) *MetadataBuilder {
	b.metadata.SetBool(key, value)
	return b
}

// Build returns the constructed metadata
func (b *MetadataBuilder) Build() CollectionMetadata {
	return b.metadata
}

// DocumentMetadataBuilder provides a fluent interface for building document metadata
type DocumentMetadataBuilder struct {
	metadata DocumentMetadata
}

// DocumentBuilder creates a new DocumentMetadataBuilder
func DocumentBuilder() *DocumentMetadataBuilder {
	return &DocumentMetadataBuilder{
		metadata: NewDocumentMetadata(),
	}
}

// String adds a string value to the document metadata
func (b *DocumentMetadataBuilder) String(key, value string) *DocumentMetadataBuilder {
	b.metadata.SetString(key, value)
	return b
}

// Int adds an integer value to the document metadata
func (b *DocumentMetadataBuilder) Int(key string, value int64) *DocumentMetadataBuilder {
	b.metadata.SetInt(key, value)
	return b
}

// Float adds a float value to the document metadata
func (b *DocumentMetadataBuilder) Float(key string, value float64) *DocumentMetadataBuilder {
	b.metadata.SetFloat(key, value)
	return b
}

// Bool adds a boolean value to the document metadata
func (b *DocumentMetadataBuilder) Bool(key string, value bool) *DocumentMetadataBuilder {
	b.metadata.SetBool(key, value)
	return b
}

// Build returns the constructed document metadata
func (b *DocumentMetadataBuilder) Build() DocumentMetadata {
	return b.metadata
}
