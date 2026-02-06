//go:build basicv2 && !cloud

package v2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetadataContainsStringWhere(t *testing.T) {
	clause := MetadataContainsString(K("tags"), "science")
	require.NoError(t, clause.Validate())

	b, err := json.Marshal(clause)
	require.NoError(t, err)
	require.JSONEq(t, `{"tags":{"$contains":"science"}}`, string(b))
}

func TestMetadataNotContainsStringWhere(t *testing.T) {
	clause := MetadataNotContainsString(K("tags"), "old")
	require.NoError(t, clause.Validate())

	b, err := json.Marshal(clause)
	require.NoError(t, err)
	require.JSONEq(t, `{"tags":{"$not_contains":"old"}}`, string(b))
}

func TestMetadataContainsIntWhere(t *testing.T) {
	clause := MetadataContainsInt(K("scores"), 100)
	require.NoError(t, clause.Validate())

	b, err := json.Marshal(clause)
	require.NoError(t, err)
	require.JSONEq(t, `{"scores":{"$contains":100}}`, string(b))
}

func TestMetadataNotContainsIntWhere(t *testing.T) {
	clause := MetadataNotContainsInt(K("scores"), 50)
	require.NoError(t, clause.Validate())

	b, err := json.Marshal(clause)
	require.NoError(t, err)
	require.JSONEq(t, `{"scores":{"$not_contains":50}}`, string(b))
}

func TestMetadataContainsFloatWhere(t *testing.T) {
	clause := MetadataContainsFloat(K("values"), 1.5)
	require.NoError(t, clause.Validate())

	b, err := json.Marshal(clause)
	require.NoError(t, err)
	require.JSONEq(t, `{"values":{"$contains":1.5}}`, string(b))
}

func TestMetadataNotContainsFloatWhere(t *testing.T) {
	clause := MetadataNotContainsFloat(K("values"), 2.5)
	require.NoError(t, clause.Validate())

	b, err := json.Marshal(clause)
	require.NoError(t, err)
	require.JSONEq(t, `{"values":{"$not_contains":2.5}}`, string(b))
}

func TestMetadataContainsBoolWhere(t *testing.T) {
	clause := MetadataContainsBool(K("flags"), true)
	require.NoError(t, clause.Validate())

	b, err := json.Marshal(clause)
	require.NoError(t, err)
	require.JSONEq(t, `{"flags":{"$contains":true}}`, string(b))
}

func TestMetadataNotContainsBoolWhere(t *testing.T) {
	clause := MetadataNotContainsBool(K("flags"), false)
	require.NoError(t, clause.Validate())

	b, err := json.Marshal(clause)
	require.NoError(t, err)
	require.JSONEq(t, `{"flags":{"$not_contains":false}}`, string(b))
}

func TestMetadataContainsStringEmptyKeyValidation(t *testing.T) {
	clause := MetadataContainsString("", "value")
	err := clause.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid key")
}

func TestMetadataContainsStringEmptyValueValidation(t *testing.T) {
	clause := MetadataContainsString(K("tags"), "")
	err := clause.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-empty string")
}

func TestMetadataContainsWithAndCombination(t *testing.T) {
	clause := And(
		MetadataContainsString(K("tags"), "science"),
		MetadataContainsInt(K("scores"), 100),
	)
	require.NoError(t, clause.Validate())

	b, err := json.Marshal(clause)
	require.NoError(t, err)

	// Just ensure it marshals without error - the exact JSON structure is complex
	require.Contains(t, string(b), "$and")
	require.Contains(t, string(b), "$contains")
}
