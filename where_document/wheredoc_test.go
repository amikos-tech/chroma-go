//go:build basic

package wheredoc

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func Compare(t *testing.T, actual, expected map[string]interface{}) bool {
	builtExprJSON, _ := json.Marshal(actual)
	expectedJSON, _ := json.Marshal(expected)
	require.Equal(t, string(expectedJSON), string(builtExprJSON))
	return true
}

func TestNewWhereDocumentBuilder(t *testing.T) {
	t.Run("Test contains with invalid value", func(t *testing.T) {
		builder := NewWhereDocumentBuilder().Contains(1)
		_, err := builder.Build()
		if err == nil {
			t.Errorf("Expected error, but got nil")
		}
	})

	t.Run("Test not_contains with invalid value", func(t *testing.T) {
		builder := NewWhereDocumentBuilder().NotContains(1.1)
		_, err := builder.Build()
		if err == nil {
			t.Errorf("Expected error, but got nil")
		}
	})

	t.Run("Test contains", func(t *testing.T) {
		builder := NewWhereDocumentBuilder().Contains("testValue")

		expected := map[string]interface{}{
			"$contains": "testValue",
		}
		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !reflect.DeepEqual(builtExpr, expected) {
			t.Errorf("Expected %v, but got %v", expected, builtExpr)
		}
	})

	t.Run("Test not_contains", func(t *testing.T) {
		builder := NewWhereDocumentBuilder().NotContains("testValue")

		expected := map[string]interface{}{
			"$not_contains": "testValue",
		}
		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !reflect.DeepEqual(builtExpr, expected) {
			t.Errorf("Expected %v, but got %v", expected, builtExpr)
		}
	})

	t.Run("Test and", func(t *testing.T) {
		builder := NewWhereDocumentBuilder().And(
			NewWhereDocumentBuilder().Contains("Noah"),
			NewWhereDocumentBuilder().NotContains("Joana"),
		)

		expected := map[string]interface{}{
			"$and": []interface{}{
				map[string]interface{}{
					"$contains": "Noah",
				},
				map[string]interface{}{
					"$not_contains": "Joana",
				},
			},
		}

		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		Compare(t, builtExpr, expected)
	})

	t.Run("Test or", func(t *testing.T) {
		builder := NewWhereDocumentBuilder().Or(
			NewWhereDocumentBuilder().Contains("Noah"),
			NewWhereDocumentBuilder().NotContains("Joana"),
		)

		expected := map[string]interface{}{
			"$or": []interface{}{
				map[string]interface{}{
					"$contains": "Noah",
				},
				map[string]interface{}{
					"$not_contains": "Joana",
				},
			},
		}

		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		Compare(t, builtExpr, expected)
	})

	t.Run("Test nested and/or", func(t *testing.T) {
		builder := NewWhereDocumentBuilder().And(
			NewWhereDocumentBuilder().Or(
				NewWhereDocumentBuilder().Contains("Noah"),
				NewWhereDocumentBuilder().NotContains("Joana"),
			),
			NewWhereDocumentBuilder().NotContains("Jane"),
		)

		expected := map[string]interface{}{
			"$and": []interface{}{
				map[string]interface{}{
					"$or": []interface{}{
						map[string]interface{}{
							"$contains": "Noah",
						},
						map[string]interface{}{
							"$not_contains": "Joana",
						},
					},
				},
				map[string]interface{}{
					"$not_contains": "Jane",
				},
			},
		}

		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		// print both maps as JSONs
		Compare(t, builtExpr, expected)
	})
}

func TestWhereDocumentBuilderWithOptions(t *testing.T) {
	t.Run("Test Contains", func(t *testing.T) {
		t.Parallel()
		var x, err = WhereDocument(Contains("something"))
		require.NoError(t, err)
		var actual = map[string]interface{}{
			"$contains": "something",
		}
		Compare(t, x, actual)
	})

	t.Run("Test NotContains", func(t *testing.T) {
		t.Parallel()
		var x, err = WhereDocument(NotContains("something"))
		require.NoError(t, err)
		var actual = map[string]interface{}{
			"$not_contains": "something",
		}
		Compare(t, x, actual)
	})

	t.Run("Test And", func(t *testing.T) {
		t.Parallel()
		var x, err = WhereDocument(And(Contains("something"), NotContains("something")))
		require.NoError(t, err)
		var actual = map[string]interface{}{
			"$and": []map[string]interface{}{
				{"$contains": "something"},
				{"$not_contains": "something"},
			},
		}
		Compare(t, x, actual)
	})

	t.Run("Test Or", func(t *testing.T) {
		t.Parallel()
		var x, err = WhereDocument(Or(Contains("something"), NotContains("something")))
		require.NoError(t, err)
		var actual = map[string]interface{}{
			"$or": []map[string]interface{}{
				{"$contains": "something"},
				{"$not_contains": "something"},
			},
		}
		Compare(t, x, actual)
	})
}
