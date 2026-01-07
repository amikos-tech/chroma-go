package v2

import (
	"encoding/json"

	"github.com/pkg/errors"
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
