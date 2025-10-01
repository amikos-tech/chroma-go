package v2

import "fmt"

// MetadataBuilder provides a fluent interface for building metadata
type MetadataBuilder struct {
	metadata CollectionMetadata
	errors   []error
}

// Builder creates a new MetadataBuilder
func Builder() *MetadataBuilder {
	return &MetadataBuilder{
		metadata: NewMetadata(),
	}
}

// String adds a string value to the metadata.
// Empty keys will cause Build() to return an error.
func (b *MetadataBuilder) String(key, value string) *MetadataBuilder {
	if key == "" {
		b.errors = append(b.errors, fmt.Errorf("empty key provided for string value: %q", value))
		return b
	}
	b.metadata.SetString(key, value)
	return b
}

// Int adds an integer value to the metadata.
// Empty keys will cause Build() to return an error.
func (b *MetadataBuilder) Int(key string, value int64) *MetadataBuilder {
	if key == "" {
		b.errors = append(b.errors, fmt.Errorf("empty key provided for int value: %d", value))
		return b
	}
	b.metadata.SetInt(key, value)
	return b
}

// Float adds a float value to the metadata.
// Empty keys will cause Build() to return an error.
func (b *MetadataBuilder) Float(key string, value float64) *MetadataBuilder {
	if key == "" {
		b.errors = append(b.errors, fmt.Errorf("empty key provided for float value: %f", value))
		return b
	}
	b.metadata.SetFloat(key, value)
	return b
}

// Bool adds a boolean value to the metadata.
// Empty keys will cause Build() to return an error.
func (b *MetadataBuilder) Bool(key string, value bool) *MetadataBuilder {
	if key == "" {
		b.errors = append(b.errors, fmt.Errorf("empty key provided for bool value: %t", value))
		return b
	}
	b.metadata.SetBool(key, value)
	return b
}

// Build returns the constructed metadata or an error if validation failed
func (b *MetadataBuilder) Build() (CollectionMetadata, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("metadata validation failed with %d error(s): %v", len(b.errors), b.errors)
	}
	return b.metadata, nil
}

// DocumentMetadataBuilder provides a fluent interface for building document metadata
type DocumentMetadataBuilder struct {
	metadata DocumentMetadata
	errors   []error
}

// DocumentBuilder creates a new DocumentMetadataBuilder
func DocumentBuilder() *DocumentMetadataBuilder {
	return &DocumentMetadataBuilder{
		metadata: NewDocumentMetadata(),
	}
}

// String adds a string value to the document metadata.
// Empty keys will cause Build() to return an error.
func (b *DocumentMetadataBuilder) String(key, value string) *DocumentMetadataBuilder {
	if key == "" {
		b.errors = append(b.errors, fmt.Errorf("empty key provided for string value: %q", value))
		return b
	}
	b.metadata.SetString(key, value)
	return b
}

// Int adds an integer value to the document metadata.
// Empty keys will cause Build() to return an error.
func (b *DocumentMetadataBuilder) Int(key string, value int64) *DocumentMetadataBuilder {
	if key == "" {
		b.errors = append(b.errors, fmt.Errorf("empty key provided for int value: %d", value))
		return b
	}
	b.metadata.SetInt(key, value)
	return b
}

// Float adds a float value to the document metadata.
// Empty keys will cause Build() to return an error.
func (b *DocumentMetadataBuilder) Float(key string, value float64) *DocumentMetadataBuilder {
	if key == "" {
		b.errors = append(b.errors, fmt.Errorf("empty key provided for float value: %f", value))
		return b
	}
	b.metadata.SetFloat(key, value)
	return b
}

// Bool adds a boolean value to the document metadata.
// Empty keys will cause Build() to return an error.
func (b *DocumentMetadataBuilder) Bool(key string, value bool) *DocumentMetadataBuilder {
	if key == "" {
		b.errors = append(b.errors, fmt.Errorf("empty key provided for bool value: %t", value))
		return b
	}
	b.metadata.SetBool(key, value)
	return b
}

// Build returns the constructed document metadata or an error if validation failed
func (b *DocumentMetadataBuilder) Build() (DocumentMetadata, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("document metadata validation failed with %d error(s): %v", len(b.errors), b.errors)
	}
	return b.metadata, nil
}
