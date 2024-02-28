package utils

import (
	"encoding/json"
	"reflect"
	"testing"
)

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
func TestNewMetadataBuilder(t *testing.T) {
	t.Run("Test invalid metadata", func(t *testing.T) {
		builder := NewMetadataBuilder().ForValue("testKey", map[string]interface{}{"invalid": "value"})
		_, err := builder.Build()
		if err == nil {
			t.Errorf("Expected error, but got nil")
		}
	})

	t.Run("Test int", func(t *testing.T) {
		builder := NewMetadataBuilder().ForValue("testKey", 1)

		expected := map[string]interface{}{
			"testKey": 1,
		}
		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !reflect.DeepEqual(builtExpr, expected) {
			t.Errorf("Expected %v, but got %v", expected, builtExpr)
		}
	})
	t.Run("Test float32", func(t *testing.T) {
		builder := NewMetadataBuilder().ForValue("testKey", float32(1.1))

		expected := map[string]interface{}{
			"testKey": float32(1.1),
		}
		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !reflect.DeepEqual(builtExpr, expected) {
			t.Errorf("Expected %v, but got %v", expected, builtExpr)
		}
	})

	t.Run("Test bool", func(t *testing.T) {
		builder := NewMetadataBuilder().ForValue("testKey", true)

		expected := map[string]interface{}{
			"testKey": true,
		}
		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !reflect.DeepEqual(builtExpr, expected) {
			t.Errorf("Expected %v, but got %v", expected, builtExpr)
		}
	})

	t.Run("Test string", func(t *testing.T) {
		builder := NewMetadataBuilder().ForValue("testKey", "value")

		expected := map[string]interface{}{
			"testKey": "value",
		}
		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !reflect.DeepEqual(builtExpr, expected) {
			t.Errorf("Expected %v, but got %v", expected, builtExpr)
		}
	})

	t.Run("Test multiple KV", func(t *testing.T) {
		builder := NewMetadataBuilder().ForValue("testKey", "value").ForValue("testKey2", 1).ForValue("testKey3", true).ForValue("testKey4", float32(1.1))

		expected := map[string]interface{}{
			"testKey":  "value",
			"testKey2": 1,
			"testKey3": true,
			"testKey4": float32(1.1),
		}
		builtExpr, err := builder.Build()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !reflect.DeepEqual(builtExpr, expected) {
			t.Errorf("Expected %v, but got %v", expected, builtExpr)
		}
	})
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
		// print both maps as JSONs
		builtExprJSON, _ := json.Marshal(builtExpr)
		expectedJSON, _ := json.Marshal(expected)

		if string(builtExprJSON) != string(expectedJSON) {
			t.Errorf("Expected %v, but got %v", string(expectedJSON), string(builtExprJSON))
		}
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
		// print both maps as JSONs
		builtExprJSON, _ := json.Marshal(builtExpr)
		expectedJSON, _ := json.Marshal(expected)

		if string(builtExprJSON) != string(expectedJSON) {
			t.Errorf("Expected %v, but got %v", string(expectedJSON), string(builtExprJSON))
		}
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
		builtExprJSON, _ := json.Marshal(builtExpr)
		expectedJSON, _ := json.Marshal(expected)

		if string(builtExprJSON) != string(expectedJSON) {
			t.Errorf("Expected %v, but got %v", string(expectedJSON), string(builtExprJSON))
		}
	})
}
