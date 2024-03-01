package metadata

import (
	"github.com/amikos-tech/chroma-go/types"
)

type MetadataBuilder struct {
	Metadata map[string]interface{}
}

func NewMetadataBuilder(metadata *map[string]interface{}) *MetadataBuilder {
	if metadata != nil {
		return &MetadataBuilder{Metadata: *metadata}
	}
	return &MetadataBuilder{Metadata: make(map[string]interface{})}
}

type Option func(*MetadataBuilder) error

func WithMetadata(key string, value interface{}) Option {
	return func(b *MetadataBuilder) error {
		switch value.(type) {
		case string, int, float32, bool, int32, uint32, int64, uint64:
			b.Metadata[key] = value
		case types.DistanceFunction:
			b.Metadata[key] = value
		default:
			return &types.InvalidMetadataValueError{Key: key, Value: value}
		}
		return nil
	}
}

func WithMetadatas(metadata map[string]interface{}) Option {
	return func(b *MetadataBuilder) error {
		for k, v := range metadata {
			switch v.(type) {
			case string, int, float32, bool:
				b.Metadata[k] = v
			default:
				return &types.InvalidMetadataValueError{Key: k, Value: v}
			}
		}
		return nil
	}
}
