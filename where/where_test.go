//go:build basic

package where

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func Compare(t *testing.T, actual, expected map[string]interface{}) bool {
	builtExprJSON, _ := json.Marshal(actual)
	expectedJSON, _ := json.Marshal(expected)
	require.Equal(t, string(expectedJSON), string(builtExprJSON))
	return true
}

func TestWhereBuilder(t *testing.T) {
	t.Run("Test eq with invalid value", func(t *testing.T) {
		builder := NewWhereBuilder().Eq("testKey", map[string]interface{}{"invalid": "value"})
		_, err := builder.Build()
		if err == nil {
			t.Errorf("Expected error, but got nil")
		}
	})

	t.Run("Test eq", func(t *testing.T) {
		builder := NewWhereBuilder().Eq("testKey", "testValue")

		expected := map[string]interface{}{
			"testKey": map[string]interface{}{
				"$eq": "testValue",
			},
		}
		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		builtExprJSON, _ := json.Marshal(builtExpr)
		expectedJSON, _ := json.Marshal(expected)

		if string(builtExprJSON) != string(expectedJSON) {
			t.Errorf("Expected %v, but got %v", string(expectedJSON), string(builtExprJSON))
		}
	})

	t.Run("Test ne", func(t *testing.T) {
		builder := NewWhereBuilder().Ne("testKey", "testValue")

		expected := map[string]interface{}{
			"testKey": map[string]interface{}{
				"$ne": "testValue",
			},
		}

		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		builtExprJSON, _ := json.Marshal(builtExpr)
		expectedJSON, _ := json.Marshal(expected)

		if string(builtExprJSON) != string(expectedJSON) {
			t.Errorf("Expected %v, but got %v", string(expectedJSON), string(builtExprJSON))
		}
	})

	t.Run("Test gt", func(t *testing.T) {
		builder := NewWhereBuilder().Gt("testKey", 1)

		expected := map[string]interface{}{
			"testKey": map[string]interface{}{
				"$gt": 1,
			},
		}

		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		builtExprJSON, _ := json.Marshal(builtExpr)
		expectedJSON, _ := json.Marshal(expected)

		if string(builtExprJSON) != string(expectedJSON) {
			t.Errorf("Expected %v, but got %v", string(expectedJSON), string(builtExprJSON))
		}
	})

	t.Run("Test gte", func(t *testing.T) {
		builder := NewWhereBuilder().Gte("testKey", 1)

		expected := map[string]interface{}{
			"testKey": map[string]interface{}{
				"$gte": 1,
			},
		}

		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		builtExprJSON, _ := json.Marshal(builtExpr)
		expectedJSON, _ := json.Marshal(expected)

		if string(builtExprJSON) != string(expectedJSON) {
			t.Errorf("Expected %v, but got %v", string(expectedJSON), string(builtExprJSON))
		}
	})

	t.Run("Test lt", func(t *testing.T) {
		builder := NewWhereBuilder().Lt("testKey", 1)

		expected := map[string]interface{}{
			"testKey": map[string]interface{}{
				"$lt": 1,
			},
		}

		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		builtExprJSON, _ := json.Marshal(builtExpr)
		expectedJSON, _ := json.Marshal(expected)

		if string(builtExprJSON) != string(expectedJSON) {
			t.Errorf("Expected %v, but got %v", string(expectedJSON), string(builtExprJSON))
		}
	})

	t.Run("Test lte", func(t *testing.T) {
		builder := NewWhereBuilder().Lte("testKey", 1)

		expected := map[string]interface{}{
			"testKey": map[string]interface{}{
				"$lte": 1,
			},
		}

		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		builtExprJSON, _ := json.Marshal(builtExpr)
		expectedJSON, _ := json.Marshal(expected)

		if string(builtExprJSON) != string(expectedJSON) {
			t.Errorf("Expected %v, but got %v", string(expectedJSON), string(builtExprJSON))
		}
	})

	t.Run("Test lte floats", func(t *testing.T) {
		builder := NewWhereBuilder().Lte("testKey", float32(1.1))

		expected := map[string]interface{}{
			"testKey": map[string]interface{}{
				"$lte": 1.1,
			},
		}

		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		builtExprJSON, _ := json.Marshal(builtExpr)
		expectedJSON, _ := json.Marshal(expected)

		if string(builtExprJSON) != string(expectedJSON) {
			t.Errorf("Expected %v, but got %v", string(expectedJSON), string(builtExprJSON))
		}
	})

	t.Run("Test in", func(t *testing.T) {
		builder := NewWhereBuilder().In("testKey", []interface{}{1, 2, 3})

		expected := map[string]interface{}{
			"testKey": map[string]interface{}{
				"$in": []interface{}{1, 2, 3},
			},
		}

		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		builtExprJSON, _ := json.Marshal(builtExpr)
		expectedJSON, _ := json.Marshal(expected)

		if string(builtExprJSON) != string(expectedJSON) {
			t.Errorf("Expected %v, but got %v", string(expectedJSON), string(builtExprJSON))
		}
	})

	t.Run("Test nin", func(t *testing.T) {
		builder := NewWhereBuilder().Nin("testKey", []interface{}{1, 2, 3})

		expected := map[string]interface{}{
			"testKey": map[string]interface{}{
				"$nin": []interface{}{1, 2, 3},
			},
		}

		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		builtExprJSON, _ := json.Marshal(builtExpr)
		expectedJSON, _ := json.Marshal(expected)

		if string(builtExprJSON) != string(expectedJSON) {
			t.Errorf("Expected %v, but got %v", string(expectedJSON), string(builtExprJSON))
		}
	})

	t.Run("Test and", func(t *testing.T) {
		builder := NewWhereBuilder().And(NewWhereBuilder().Eq("name", "Noah"), NewWhereBuilder().Eq("age", 25))

		expected := map[string]interface{}{
			"$and": []interface{}{
				map[string]interface{}{
					"name": map[string]interface{}{
						"$eq": "Noah",
					},
				},
				map[string]interface{}{
					"age": map[string]interface{}{
						"$eq": 25,
					},
				},
			},
		}

		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		// print both maps as JSONs
		builtExprJSON, _ := json.Marshal(builtExpr)
		expectedJSON, _ := json.Marshal(expected)

		if string(builtExprJSON) != string(expectedJSON) {
			t.Errorf("Expected %v, but got %v", string(expectedJSON), string(builtExprJSON))
		}
	})

	t.Run("Test or", func(t *testing.T) {
		builder := NewWhereBuilder().Or(NewWhereBuilder().Eq("name", "Noah"), NewWhereBuilder().Eq("age", 25))

		expected := map[string]interface{}{
			"$or": []interface{}{
				map[string]interface{}{
					"name": map[string]interface{}{
						"$eq": "Noah",
					},
				},
				map[string]interface{}{
					"age": map[string]interface{}{
						"$eq": 25,
					},
				},
			},
		}

		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		// print both maps as JSONs
		builtExprJSON, _ := json.Marshal(builtExpr)
		expectedJSON, _ := json.Marshal(expected)

		if string(builtExprJSON) != string(expectedJSON) {
			t.Errorf("Expected %v, but got %v", string(expectedJSON), string(builtExprJSON))
		}
	})

	t.Run("Test nested and/or", func(t *testing.T) {
		builder := NewWhereBuilder().And(NewWhereBuilder().Or(NewWhereBuilder().Eq("name", "Noah"), NewWhereBuilder().Eq("name", "Joana")), NewWhereBuilder().Eq("age", 25))

		expected := map[string]interface{}{
			"$and": []interface{}{
				map[string]interface{}{
					"$or": []interface{}{
						map[string]interface{}{
							"name": map[string]interface{}{
								"$eq": "Noah",
							},
						},
						map[string]interface{}{
							"name": map[string]interface{}{
								"$eq": "Joana",
							},
						},
					},
				},
				map[string]interface{}{
					"age": map[string]interface{}{
						"$eq": 25,
					},
				},
			},
		}

		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		// print both maps as JSONs
		builtExprJSON, _ := json.Marshal(builtExpr)
		expectedJSON, _ := json.Marshal(expected)

		if string(builtExprJSON) != string(expectedJSON) {
			t.Errorf("Expected %v, but got %v", string(expectedJSON), string(builtExprJSON))
		}
	})
}

func TestWhereBuilderWithOptions(t *testing.T) {
	t.Run("Eq", func(t *testing.T) {
		t.Parallel()
		var x, err = Where(Eq("a", 1))
		require.NoError(t, err)
		var actual = map[string]interface{}{
			"a": map[string]interface{}{"$eq": 1},
		}
		Compare(t, x, actual)
	})

	t.Run("Ne", func(t *testing.T) {
		t.Parallel()
		var x, err = Where(Ne("a", 1))
		require.NoError(t, err)
		var actual = map[string]interface{}{
			"a": map[string]interface{}{"$ne": 1},
		}
		Compare(t, x, actual)
	})

	t.Run("Gt", func(t *testing.T) {
		t.Parallel()
		var x, err = Where(Gt("a", 1))
		require.NoError(t, err)
		var actual = map[string]interface{}{
			"a": map[string]interface{}{"$gt": 1},
		}
		Compare(t, x, actual)
	})

	t.Run("Gte", func(t *testing.T) {
		t.Parallel()
		var x, err = Where(Gte("a", 1))
		require.NoError(t, err)
		var actual = map[string]interface{}{
			"a": map[string]interface{}{"$gte": 1},
		}
		Compare(t, x, actual)
	})

	t.Run("Lt", func(t *testing.T) {
		t.Parallel()
		var x, err = Where(Lt("a", 1))
		require.NoError(t, err)
		var actual = map[string]interface{}{
			"a": map[string]interface{}{"$lt": 1},
		}
		Compare(t, x, actual)
	})

	t.Run("Lte", func(t *testing.T) {
		t.Parallel()
		var x, err = Where(Lte("a", 1))
		require.NoError(t, err)
		var actual = map[string]interface{}{
			"a": map[string]interface{}{"$lte": 1},
		}
		Compare(t, x, actual)
	})

	t.Run("In", func(t *testing.T) {
		t.Parallel()
		var x, err = Where(In("a", []interface{}{1, 2, 3}))
		require.NoError(t, err)
		var actual = map[string]interface{}{
			"a": map[string]interface{}{"$in": []interface{}{1, 2, 3}},
		}
		Compare(t, x, actual)
	})

	t.Run("Nin", func(t *testing.T) {
		t.Parallel()
		var x, err = Where(Nin("a", []interface{}{1, 2, 3}))
		require.NoError(t, err)
		var actual = map[string]interface{}{
			"a": map[string]interface{}{"$nin": []interface{}{1, 2, 3}},
		}
		Compare(t, x, actual)
	})

	t.Run("And", func(t *testing.T) {
		t.Parallel()
		var x, err = Where(And(Eq("a", 1), Ne("b", 2)))
		require.NoError(t, err)
		var actual = map[string]interface{}{
			"$and": []map[string]interface{}{
				{
					"a": map[string]interface{}{"$eq": 1},
				},
				{
					"b": map[string]interface{}{"$ne": 2},
				},
			},
		}
		Compare(t, x, actual)
	})

	t.Run("Or", func(t *testing.T) {
		t.Parallel()
		var x, err = Where(Or(Eq("a", 1), Ne("b", 2)))
		require.NoError(t, err)
		var actual = map[string]interface{}{
			"$or": []map[string]interface{}{
				{
					"a": map[string]interface{}{"$eq": 1},
				},
				{
					"b": map[string]interface{}{"$ne": 2},
				},
			},
		}
		Compare(t, x, actual)
	})
	t.Run("Test nested where", func(t *testing.T) {
		t.Parallel()
		var x, err = Where(
			And(
				Eq("a", 1),
				Or(
					Ne("b", -1),
					Gt("c", 3),
				),
			),
		)
		require.NoError(t, err)
		var actual = map[string]interface{}{
			"$and": []map[string]interface{}{
				{
					"a": map[string]interface{}{"$eq": 1},
				},
				{
					"$or": []map[string]interface{}{
						{
							"b": map[string]interface{}{"$ne": -1},
						},
						{
							"c": map[string]interface{}{"$gt": 3},
						},
					},
				},
			},
		}
		Compare(t, x, actual)
	})
}
