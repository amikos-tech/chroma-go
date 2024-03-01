package collection

import (
	"fmt"

	"github.com/amikos-tech/chroma-go/metadata"
	"github.com/amikos-tech/chroma-go/types"
)

type Builder struct {
	Tenant            string
	Database          string
	Name              string
	Metadata          map[string]interface{}
	CreateIfNotExist  bool
	EmbeddingFunction types.EmbeddingFunction
	IDGenerator       types.IDGenerator
}

type Option func(*Builder) error

func WithName(name string) Option {
	return func(c *Builder) error {
		if name == "" {
			return fmt.Errorf("collection name cannot be empty")
		}
		c.Name = name
		return nil
	}
}

func WithEmbeddingFunction(embeddingFunction types.EmbeddingFunction) Option {
	return func(c *Builder) error {
		c.EmbeddingFunction = embeddingFunction
		return nil
	}
}

func WithIDGenerator(idGenerator types.IDGenerator) Option {
	return func(c *Builder) error {
		c.IDGenerator = idGenerator
		return nil
	}
}

func WithCreateIfNotExist(create bool) Option {
	return func(c *Builder) error {
		c.CreateIfNotExist = create
		return nil
	}
}

func WithHNSWDistanceFunction(distanceFunction types.DistanceFunction) Option {
	return func(b *Builder) error {
		if distanceFunction != types.L2 && distanceFunction != types.IP && distanceFunction != types.COSINE {
			return fmt.Errorf("invalid distance function, must be one of l2, ip, or cosine")
		}
		return WithMetadata(types.HNSWSpace, distanceFunction)(b)
	}
}

func WithHNSWBatchSize(batchSize int32) Option {
	return func(b *Builder) error {
		if batchSize < 1 {
			return fmt.Errorf("batch size must be greater than 0")
		}
		return WithMetadata(types.HNSWBatchSize, batchSize)(b)
	}
}

func WithHNSWSyncThreshold(syncThreshold int32) Option {
	return func(b *Builder) error {
		if syncThreshold < 1 {
			return fmt.Errorf("sync threshold must be greater than 0")
		}
		return WithMetadata(types.HNSWSyncThreshold, syncThreshold)(b)
	}
}

func WithHNSWM(m int32) Option {
	return func(b *Builder) error {
		if m < 1 {
			return fmt.Errorf("m must be greater than 0")
		}
		return WithMetadata(types.HNSWM, m)(b)
	}
}

func WithHNSWConstructionEf(efConstruction int32) Option {
	return func(b *Builder) error {
		if efConstruction < 1 {
			return fmt.Errorf("efConstruction must be greater than 0")
		}
		return WithMetadata(types.HNSWConstructionEF, efConstruction)(b)
	}
}

// WithMetadatas adds metadata to the collection. If the metadata key already exists, the value is overwritten.
func WithMetadatas(metadata map[string]interface{}) Option {
	return func(b *Builder) error {
		if b.Metadata == nil {
			b.Metadata = make(map[string]interface{})
		}
		for k, v := range metadata {
			err := WithMetadata(k, v)(b)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func WithHNSWSearchEf(efSearch int32) Option {
	return func(b *Builder) error {
		if efSearch < 1 {
			return fmt.Errorf("efSearch must be greater than 0")
		}
		return WithMetadata(types.HNSWSearchEF, efSearch)(b)
	}
}

func WithHNSWNumThreads(numThreads int32) Option {
	return func(b *Builder) error {
		if numThreads < 1 {
			return fmt.Errorf("numThreads must be greater than 0")
		}
		return WithMetadata(types.HNSWNumThreads, numThreads)(b)
	}
}

func WithHNSWResizeFactor(resizeFactor float32) Option {
	return func(b *Builder) error {
		if resizeFactor < 0 {
			return fmt.Errorf("resizeFactor must be greater than or equal to 0")
		}
		return WithMetadata(types.HNSWResizeFactor, resizeFactor)(b)
	}
}

func WithMetadata(key string, value interface{}) Option {
	return func(b *Builder) error {
		if b.Metadata == nil {
			b.Metadata = make(map[string]interface{})
		}
		err := metadata.WithMetadata(key, value)(metadata.NewMetadataBuilder(&b.Metadata))
		if err != nil {
			return err
		}
		return nil
	}
}

func WithTenant(tenant string) Option {
	return func(c *Builder) error {
		if tenant == "" {
			return fmt.Errorf("tenant cannot be empty")
		}
		c.Tenant = tenant
		return nil
	}
}

func WithDatabase(database string) Option {
	return func(c *Builder) error {
		if database == "" {
			return fmt.Errorf("database cannot be empty")
		}
		c.Database = database
		return nil
	}
}
