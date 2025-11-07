package v2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCollectionConfiguration(t *testing.T) {
	schema := NewSchema()
	config := NewCollectionConfiguration(schema)

	assert.NotNil(t, config)
	assert.Equal(t, schema, config.GetSchema())
	assert.NotNil(t, config.raw)
}

func TestNewCollectionConfigurationFromMap(t *testing.T) {
	rawData := map[string]interface{}{
		"foo": "bar",
		"baz": 123,
	}

	config := NewCollectionConfigurationFromMap(rawData)
	assert.NotNil(t, config)

	val, ok := config.GetRaw("foo")
	assert.True(t, ok)
	assert.Equal(t, "bar", val)

	val, ok = config.GetRaw("baz")
	assert.True(t, ok)
	assert.Equal(t, 123, val)
}

func TestCollectionConfiguration_GetSetRaw(t *testing.T) {
	config := NewCollectionConfiguration(nil)

	// Test SetRaw and GetRaw
	config.SetRaw("key1", "value1")
	val, ok := config.GetRaw("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	// Test non-existent key
	val, ok = config.GetRaw("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestCollectionConfiguration_GetSetSchema(t *testing.T) {
	config := NewCollectionConfiguration(nil)
	assert.Nil(t, config.GetSchema())

	schema := NewSchemaWithDefaults()
	config.SetSchema(schema)

	assert.Equal(t, schema, config.GetSchema())
	// Schema should also be in raw map
	val, ok := config.GetRaw("schema")
	assert.True(t, ok)
	assert.Equal(t, schema, val)
}

func TestCollectionConfiguration_Keys(t *testing.T) {
	config := NewCollectionConfiguration(nil)

	// Initially empty
	keys := config.Keys()
	assert.Equal(t, 0, len(keys))

	// Add some values
	config.SetRaw("key1", "value1")
	config.SetRaw("key2", "value2")

	keys = config.Keys()
	assert.Equal(t, 2, len(keys))
	assert.Contains(t, keys, "key1")
	assert.Contains(t, keys, "key2")
}

func TestCollectionConfiguration_MarshalJSON(t *testing.T) {
	config := NewCollectionConfiguration(NewSchemaWithDefaults())
	config.SetRaw("custom_key", "custom_value")

	data, err := json.Marshal(config)
	require.NoError(t, err)
	assert.NotNil(t, data)

	// Verify JSON structure
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)
	assert.Contains(t, result, "schema")
	assert.Contains(t, result, "custom_key")
	assert.Equal(t, "custom_value", result["custom_key"])
}

func TestCollectionConfiguration_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"custom_key": "custom_value",
		"another_key": 42
	}`

	config := &CollectionConfigurationImpl{}
	err := json.Unmarshal([]byte(jsonData), config)
	require.NoError(t, err)

	val, ok := config.GetRaw("custom_key")
	assert.True(t, ok)
	assert.Equal(t, "custom_value", val)

	val, ok = config.GetRaw("another_key")
	assert.True(t, ok)
	// JSON numbers are decoded as float64
	assert.Equal(t, float64(42), val)
}
