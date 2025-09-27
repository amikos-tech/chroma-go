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

// Convenience functions for common metadata patterns

// QuickMetadata creates metadata from key-value pairs using a more intuitive API.
// Usage: QuickMetadata("key1", "value1", "key2", 42, "key3", true)
// Returns an empty metadata object if invalid arguments are provided.
func QuickMetadata(keysAndValues ...interface{}) (metadata CollectionMetadata) {
	defer func() {
		if r := recover(); r != nil {
			// Return empty metadata on panic
			metadata = NewMetadata()
		}
	}()

	if len(keysAndValues)%2 != 0 {
		// Return empty metadata for invalid argument count
		return NewMetadata()
	}

	metadata = NewMetadata()
	for i := 0; i < len(keysAndValues); i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			// Skip invalid keys silently
			continue
		}

		switch v := keysAndValues[i+1].(type) {
		case string:
			metadata.SetString(key, v)
		case int:
			metadata.SetInt(key, int64(v))
		case int64:
			metadata.SetInt(key, v)
		case float64:
			metadata.SetFloat(key, v)
		case float32:
			metadata.SetFloat(key, float64(v))
		case bool:
			metadata.SetBool(key, v)
		default:
			metadata.SetRaw(key, v)
		}
	}
	return metadata
}

// QuickDocumentMetadata creates document metadata from key-value pairs.
// Usage: QuickDocumentMetadata("key1", "value1", "key2", 42, "key3", true)
// Returns an empty metadata object if invalid arguments are provided.
func QuickDocumentMetadata(keysAndValues ...interface{}) (metadata DocumentMetadata) {
	defer func() {
		if r := recover(); r != nil {
			// Return empty metadata on panic
			metadata = NewDocumentMetadata()
		}
	}()

	if len(keysAndValues)%2 != 0 {
		// Return empty metadata for invalid argument count
		return NewDocumentMetadata()
	}

	metadata = NewDocumentMetadata()
	for i := 0; i < len(keysAndValues); i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			// Skip invalid keys silently
			continue
		}

		switch v := keysAndValues[i+1].(type) {
		case string:
			metadata.SetString(key, v)
		case int:
			metadata.SetInt(key, int64(v))
		case int64:
			metadata.SetInt(key, v)
		case float64:
			metadata.SetFloat(key, v)
		case float32:
			metadata.SetFloat(key, float64(v))
		case bool:
			metadata.SetBool(key, v)
		default:
			metadata.SetRaw(key, v)
		}
	}
	return metadata
}
