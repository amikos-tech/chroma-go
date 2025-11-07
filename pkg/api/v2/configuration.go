package v2

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// CollectionConfigurationImpl is the concrete implementation of CollectionConfiguration
type CollectionConfigurationImpl struct {
	schema *Schema
	raw    map[string]interface{}
}

// NewCollectionConfiguration creates a new CollectionConfigurationImpl with the given schema
func NewCollectionConfiguration(schema *Schema) *CollectionConfigurationImpl {
	return &CollectionConfigurationImpl{
		schema: schema,
		raw:    make(map[string]interface{}),
	}
}

// NewCollectionConfigurationFromMap creates a CollectionConfigurationImpl from a raw map
// This is useful when deserializing from API responses
func NewCollectionConfigurationFromMap(raw map[string]interface{}) *CollectionConfigurationImpl {
	config := &CollectionConfigurationImpl{
		raw: raw,
	}

	// Try to extract schema if present
	if schemaData, ok := raw["schema"]; ok {
		if schemaMap, ok := schemaData.(map[string]interface{}); ok {
			schema := NewSchema()
			// TODO: Properly deserialize schema from map
			// For now, store it in raw format
			_ = schemaMap
			config.schema = schema
		}
	}

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

// GetSchema returns the schema associated with this configuration
func (c *CollectionConfigurationImpl) GetSchema() *Schema {
	return c.schema
}

// SetSchema sets the schema for this configuration
func (c *CollectionConfigurationImpl) SetSchema(schema *Schema) {
	c.schema = schema
	if schema != nil && c.raw != nil {
		c.raw["schema"] = schema
	}
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

	// Include schema if present
	if c.schema != nil {
		c.raw["schema"] = c.schema
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

	// Try to extract and parse schema if present
	if schemaData, ok := c.raw["schema"]; ok {
		schemaBytes, err := json.Marshal(schemaData)
		if err != nil {
			return errors.Wrap(err, "failed to marshal schema data")
		}

		schema := NewSchema()
		if err := json.Unmarshal(schemaBytes, schema); err != nil {
			return errors.Wrap(err, "failed to unmarshal schema")
		}
		c.schema = schema
	}

	return nil
}
