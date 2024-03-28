//go:build test

package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/test"
)

func TestWithMetadata(t *testing.T) {
	t.Run("Test invalid Metadata", func(t *testing.T) {
		builder := NewMetadataBuilder(nil)
		err := WithMetadata("testKey", map[string]interface{}{"invalid": "value"})(builder)
		require.Error(t, err)
	})

	t.Run("Test int", func(t *testing.T) {
		actual := make(map[string]interface{})
		builder := NewMetadataBuilder(&actual)
		err := WithMetadata("testKey", 1)(builder)
		require.NoError(t, err)
		expected := map[string]interface{}{
			"testKey": 1,
		}
		test.Compare(t, actual, expected)
	})
	t.Run("Test float32", func(t *testing.T) {
		actual := make(map[string]interface{})
		builder := NewMetadataBuilder(&actual)
		err := WithMetadata("testKey", float32(1.1))(builder)
		require.NoError(t, err)
		expected := map[string]interface{}{
			"testKey": float32(1.1),
		}
		test.Compare(t, actual, expected)
	})

	t.Run("Test bool", func(t *testing.T) {
		actual := make(map[string]interface{})
		builder := NewMetadataBuilder(&actual)
		err := WithMetadata("testKey", true)(builder)
		require.NoError(t, err)
		expected := map[string]interface{}{
			"testKey": true,
		}
		test.Compare(t, builder.Metadata, expected)
	})

	t.Run("Test string", func(t *testing.T) {
		actual := make(map[string]interface{})
		builder := NewMetadataBuilder(&actual)
		err := WithMetadata("testKey", "value")(builder)
		require.NoError(t, err)
		expected := map[string]interface{}{
			"testKey": "value",
		}
		test.Compare(t, actual, expected)
	})

	t.Run("Test all types", func(t *testing.T) {
		actual := make(map[string]interface{})
		builder := NewMetadataBuilder(&actual)
		err := WithMetadata("testKey", "value")(builder)
		require.NoError(t, err)
		err = WithMetadata("testKey2", 1)(builder)
		require.NoError(t, err)
		err = WithMetadata("testKey3", true)(builder)
		require.NoError(t, err)
		err = WithMetadata("testKey4", float32(1.1))(builder)
		require.NoError(t, err)
		expected := map[string]interface{}{
			"testKey":  "value",
			"testKey2": 1,
			"testKey3": true,
			"testKey4": float32(1.1),
		}
		test.Compare(t, actual, expected)
	})
}
