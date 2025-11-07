package v2

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// IndexConfig is the interface for all index configuration types
type IndexConfig interface {
	// IndexType returns the type of index (e.g., "HNSW", "FTS", "InvertedIndex")
	IndexType() string
	// ValueType returns the value type this index operates on (e.g., "VectorValue", "StringValue")
	ValueType() string
}

// VectorIndexConfig represents configuration for vector indexing
type VectorIndexConfig struct {
	Space             embeddings.DistanceMetric `json:"space,omitempty"`
	EmbeddingFunction string                    `json:"embedding_function,omitempty"`
	HnswConfig        *HnswIndexConfig          `json:"hnsw,omitempty"`
	SpannConfig       *SpannIndexConfig         `json:"spann,omitempty"`
}

func (c *VectorIndexConfig) IndexType() string {
	return "VectorIndex"
}

func (c *VectorIndexConfig) ValueType() string {
	return "VectorValue"
}

// HnswIndexConfig represents HNSW (Hierarchical Navigable Small World) algorithm configuration
type HnswIndexConfig struct {
	ConstructionEF int     `json:"construction_ef,omitempty"`
	SearchEF       int     `json:"search_ef,omitempty"`
	M              int     `json:"M,omitempty"`
	NumThreads     int     `json:"num_threads,omitempty"`
	ResizeFactor   float64 `json:"resize_factor,omitempty"`
}

func (c *HnswIndexConfig) IndexType() string {
	return "HNSW"
}

func (c *HnswIndexConfig) ValueType() string {
	return "VectorValue"
}

// SpannIndexConfig represents SPANN (Scalable Approximate Nearest Neighbor) algorithm configuration
type SpannIndexConfig struct {
	// TODO: Add SPANN-specific configuration when available
}

func (c *SpannIndexConfig) IndexType() string {
	return "SPANN"
}

func (c *SpannIndexConfig) ValueType() string {
	return "VectorValue"
}

// FtsIndexConfig represents Full-Text Search index configuration
type FtsIndexConfig struct {
	Tokenizer string `json:"tokenizer,omitempty"`
}

func (c *FtsIndexConfig) IndexType() string {
	return "FTS"
}

func (c *FtsIndexConfig) ValueType() string {
	return "DocumentValue"
}

// SparseVectorIndexConfig represents configuration for sparse vector indexing
type SparseVectorIndexConfig struct {
	// TODO: Add sparse vector specific configuration when available
}

func (c *SparseVectorIndexConfig) IndexType() string {
	return "SparseVectorIndex"
}

func (c *SparseVectorIndexConfig) ValueType() string {
	return "SparseVectorValue"
}

// StringInvertedIndexConfig represents configuration for string metadata indexing
type StringInvertedIndexConfig struct{}

func (c *StringInvertedIndexConfig) IndexType() string {
	return "InvertedIndex"
}

func (c *StringInvertedIndexConfig) ValueType() string {
	return "StringValue"
}

// IntInvertedIndexConfig represents configuration for integer metadata indexing
type IntInvertedIndexConfig struct{}

func (c *IntInvertedIndexConfig) IndexType() string {
	return "InvertedIndex"
}

func (c *IntInvertedIndexConfig) ValueType() string {
	return "IntValue"
}

// FloatInvertedIndexConfig represents configuration for float metadata indexing
type FloatInvertedIndexConfig struct{}

func (c *FloatInvertedIndexConfig) IndexType() string {
	return "InvertedIndex"
}

func (c *FloatInvertedIndexConfig) ValueType() string {
	return "FloatValue"
}

// BoolInvertedIndexConfig represents configuration for boolean metadata indexing
type BoolInvertedIndexConfig struct{}

func (c *BoolInvertedIndexConfig) IndexType() string {
	return "InvertedIndex"
}

func (c *BoolInvertedIndexConfig) ValueType() string {
	return "BoolValue"
}

// IndexEntry represents an index configuration with an enabled flag
type IndexEntry struct {
	Config  IndexConfig `json:"config"`
	Enabled bool        `json:"enabled"`
}

// Schema manages index configurations for a collection
// It supports both default index configurations and per-key overrides
type Schema struct {
	// defaults maps value type to default index configurations
	// e.g., "VectorValue" -> VectorIndexConfig
	defaults map[string]IndexConfig

	// keyOverrides maps key -> value_type -> index_type -> config
	// e.g., "my_field" -> "StringValue" -> "InvertedIndex" -> StringInvertedIndexConfig
	keyOverrides map[string]map[string]map[string]IndexConfig
}

// NewSchema creates a new empty Schema
func NewSchema() *Schema {
	return &Schema{
		defaults:     make(map[string]IndexConfig),
		keyOverrides: make(map[string]map[string]map[string]IndexConfig),
	}
}

// NewSchemaWithDefaults creates a new Schema with default configurations
func NewSchemaWithDefaults() *Schema {
	schema := NewSchema()

	// Set default vector index with HNSW
	schema.defaults["VectorValue"] = &VectorIndexConfig{
		Space: embeddings.L2,
		HnswConfig: &HnswIndexConfig{
			ConstructionEF: 100,
			SearchEF:       10,
			M:              16,
		},
	}

	// Set default FTS index
	schema.defaults["DocumentValue"] = &FtsIndexConfig{
		Tokenizer: "default",
	}

	return schema
}

// SetDefault sets a default index configuration for a value type
func (s *Schema) SetDefault(valueType string, config IndexConfig) error {
	if valueType == "" {
		return errors.New("value type cannot be empty")
	}
	if config == nil {
		return errors.New("config cannot be nil")
	}
	s.defaults[valueType] = config
	return nil
}

// GetDefault returns the default index configuration for a value type
func (s *Schema) GetDefault(valueType string) (IndexConfig, bool) {
	config, ok := s.defaults[valueType]
	return config, ok
}

// CreateIndex enables an index for a specific key with the given configuration
// If config is nil, it enables all default indexes for the key
func (s *Schema) CreateIndex(key string, config IndexConfig) error {
	if key == "" {
		return errors.New("key cannot be empty")
	}
	if config == nil {
		return errors.New("config cannot be nil")
	}

	valueType := config.ValueType()
	indexType := config.IndexType()

	if s.keyOverrides[key] == nil {
		s.keyOverrides[key] = make(map[string]map[string]IndexConfig)
	}
	if s.keyOverrides[key][valueType] == nil {
		s.keyOverrides[key][valueType] = make(map[string]IndexConfig)
	}

	s.keyOverrides[key][valueType][indexType] = config
	return nil
}

// DeleteIndex removes an index for a specific key, value type, and index type
func (s *Schema) DeleteIndex(key string, valueType string, indexType string) error {
	if key == "" {
		return errors.New("key cannot be empty")
	}
	if valueType == "" {
		return errors.New("value type cannot be empty")
	}
	if indexType == "" {
		return errors.New("index type cannot be empty")
	}

	if s.keyOverrides[key] == nil {
		return errors.Errorf("no overrides found for key: %s", key)
	}
	if s.keyOverrides[key][valueType] == nil {
		return errors.Errorf("no overrides found for key: %s, value type: %s", key, valueType)
	}

	delete(s.keyOverrides[key][valueType], indexType)

	// Clean up empty maps
	if len(s.keyOverrides[key][valueType]) == 0 {
		delete(s.keyOverrides[key], valueType)
	}
	if len(s.keyOverrides[key]) == 0 {
		delete(s.keyOverrides, key)
	}

	return nil
}

// GetIndexForKey returns the index configuration for a specific key, value type, and index type
func (s *Schema) GetIndexForKey(key string, valueType string, indexType string) (IndexConfig, bool) {
	if s.keyOverrides[key] == nil {
		return nil, false
	}
	if s.keyOverrides[key][valueType] == nil {
		return nil, false
	}
	config, ok := s.keyOverrides[key][valueType][indexType]
	return config, ok
}

// GetAllIndexesForKey returns all index configurations for a specific key
func (s *Schema) GetAllIndexesForKey(key string) map[string]map[string]IndexConfig {
	if s.keyOverrides[key] == nil {
		return nil
	}
	return s.keyOverrides[key]
}

// Keys returns all keys that have index overrides
func (s *Schema) Keys() []string {
	keys := make([]string, 0, len(s.keyOverrides))
	for k := range s.keyOverrides {
		keys = append(keys, k)
	}
	return keys
}

// MarshalJSON serializes the Schema to JSON
func (s *Schema) MarshalJSON() ([]byte, error) {
	type schemaJSON struct {
		Defaults     map[string]interface{}                       `json:"defaults,omitempty"`
		KeyOverrides map[string]map[string]map[string]interface{} `json:"key_overrides,omitempty"`
	}

	result := schemaJSON{
		Defaults:     make(map[string]interface{}),
		KeyOverrides: make(map[string]map[string]map[string]interface{}),
	}

	// Marshal defaults
	for k, v := range s.defaults {
		result.Defaults[k] = v
	}

	// Marshal key overrides
	for key, valueTypes := range s.keyOverrides {
		result.KeyOverrides[key] = make(map[string]map[string]interface{})
		for valueType, indexTypes := range valueTypes {
			result.KeyOverrides[key][valueType] = make(map[string]interface{})
			for indexType, config := range indexTypes {
				result.KeyOverrides[key][valueType][indexType] = config
			}
		}
	}

	return json.Marshal(result)
}

// UnmarshalJSON deserializes the Schema from JSON
func (s *Schema) UnmarshalJSON(data []byte) error {
	// For now, we'll unmarshal into raw map structure
	// Full implementation would need to handle type detection for different IndexConfig types
	type schemaJSON struct {
		Defaults     map[string]json.RawMessage                       `json:"defaults,omitempty"`
		KeyOverrides map[string]map[string]map[string]json.RawMessage `json:"key_overrides,omitempty"`
	}

	var raw schemaJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return errors.Wrap(err, "failed to unmarshal schema")
	}

	s.defaults = make(map[string]IndexConfig)
	s.keyOverrides = make(map[string]map[string]map[string]IndexConfig)

	// TODO: Implement proper type detection and unmarshaling for different IndexConfig types
	// For now, this is a placeholder that would need to be enhanced based on actual API responses

	return nil
}
