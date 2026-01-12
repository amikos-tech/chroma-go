package v2

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

const (
	embeddingFunctionKey     = "embedding_function"
	embeddingFunctionTypeKey = "type"
	embeddingFunctionNameKey = "name"
	embeddingFunctionCfgKey  = "config"
	efTypeKnown              = "known"
)

// CollectionConfigurationImpl is the concrete implementation of CollectionConfiguration
type CollectionConfigurationImpl struct {
	raw map[string]interface{}
}

// NewCollectionConfiguration creates a new CollectionConfigurationImpl with the given schema
func NewCollectionConfiguration() *CollectionConfigurationImpl {
	return &CollectionConfigurationImpl{
		raw: make(map[string]interface{}),
	}
}

// NewCollectionConfigurationFromMap creates a CollectionConfigurationImpl from a raw map
// This is useful when deserializing from API responses
func NewCollectionConfigurationFromMap(raw map[string]interface{}) *CollectionConfigurationImpl {
	config := &CollectionConfigurationImpl{
		raw: raw,
	}

	// Try to extract schema if present
	return config
}

// GetRaw returns the raw value for a given key
func (c *CollectionConfigurationImpl) GetRaw(key string) (interface{}, bool) {
	if c.raw == nil {
		return nil, false
	}
	val, ok := c.raw[key]
	return val, ok
}

// SetRaw sets a raw value for a given key
func (c *CollectionConfigurationImpl) SetRaw(key string, value interface{}) {
	if c.raw == nil {
		c.raw = make(map[string]interface{})
	}
	c.raw[key] = value
}

// Keys returns all keys in the configuration
func (c *CollectionConfigurationImpl) Keys() []string {
	if c.raw == nil {
		return []string{}
	}
	keys := make([]string, 0, len(c.raw))
	for k := range c.raw {
		keys = append(keys, k)
	}
	return keys
}

// MarshalJSON serializes the configuration to JSON
func (c *CollectionConfigurationImpl) MarshalJSON() ([]byte, error) {
	if c.raw == nil {
		c.raw = make(map[string]interface{})
	}

	return json.Marshal(c.raw)
}

// UnmarshalJSON deserializes the configuration from JSON
func (c *CollectionConfigurationImpl) UnmarshalJSON(data []byte) error {
	if c.raw == nil {
		c.raw = make(map[string]interface{})
	}

	if err := json.Unmarshal(data, &c.raw); err != nil {
		return errors.Wrap(err, "failed to unmarshal configuration")
	}

	return nil
}

// EmbeddingFunctionInfo represents the embedding function configuration stored in collection configuration
type EmbeddingFunctionInfo struct {
	Type   string                 `json:"type"`
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config"`
}

// IsKnown returns true if the embedding function type is "known" and can be reconstructed
func (e *EmbeddingFunctionInfo) IsKnown() bool {
	return e != nil && e.Type == efTypeKnown
}

// GetEmbeddingFunctionInfo extracts the embedding function configuration from the collection configuration
func (c *CollectionConfigurationImpl) GetEmbeddingFunctionInfo() (*EmbeddingFunctionInfo, bool) {
	if c.raw == nil {
		return nil, false
	}
	efRaw, ok := c.raw[embeddingFunctionKey]
	if !ok {
		return nil, false
	}
	efMap, ok := efRaw.(map[string]interface{})
	if !ok {
		return nil, false
	}

	info := &EmbeddingFunctionInfo{}
	if t, ok := efMap[embeddingFunctionTypeKey].(string); ok {
		info.Type = t
	}
	if n, ok := efMap[embeddingFunctionNameKey].(string); ok {
		info.Name = n
	}
	if cfg, ok := efMap[embeddingFunctionCfgKey].(map[string]interface{}); ok {
		info.Config = cfg
	}

	return info, true
}

// SetEmbeddingFunctionInfo sets the embedding function configuration in the collection configuration
func (c *CollectionConfigurationImpl) SetEmbeddingFunctionInfo(info *EmbeddingFunctionInfo) {
	if c.raw == nil {
		c.raw = make(map[string]interface{})
	}
	if info == nil {
		return
	}
	c.raw[embeddingFunctionKey] = map[string]interface{}{
		embeddingFunctionTypeKey: info.Type,
		embeddingFunctionNameKey: info.Name,
		embeddingFunctionCfgKey:  info.Config,
	}
}

// SetEmbeddingFunction creates an EmbeddingFunctionInfo from an EmbeddingFunction and stores it
func (c *CollectionConfigurationImpl) SetEmbeddingFunction(ef embeddings.EmbeddingFunction) {
	if ef == nil {
		return
	}
	c.SetEmbeddingFunctionInfo(&EmbeddingFunctionInfo{
		Type:   efTypeKnown,
		Name:   ef.Name(),
		Config: ef.GetConfig(),
	})
}

// BuildEmbeddingFunctionFromConfig attempts to reconstruct an embedding function from the configuration.
// Returns nil without error if:
// - Configuration is nil
// - No embedding_function in config
// - Type is not "known"
// - Name not registered in the dense registry
// Returns error if the factory fails to build the embedding function.
func BuildEmbeddingFunctionFromConfig(cfg *CollectionConfigurationImpl) (embeddings.EmbeddingFunction, error) {
	if cfg == nil {
		return nil, nil
	}

	efInfo, ok := cfg.GetEmbeddingFunctionInfo()
	if !ok || efInfo == nil {
		return nil, nil
	}

	if !efInfo.IsKnown() {
		return nil, nil
	}

	if !embeddings.HasDense(efInfo.Name) {
		return nil, nil
	}

	return embeddings.BuildDense(efInfo.Name, efInfo.Config)
}
