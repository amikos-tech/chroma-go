package api

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMetadataBasicOperations(t *testing.T) {
	md := NewMetadata()

	// Test setting and getting different types
	md.SetString("str", "test")
	if val, ok := md.GetString("str"); !ok || val != "test" {
		t.Errorf("GetString failed, got: %v, want: test", val)
	}

	md.SetInt("int", 42)
	if val, ok := md.GetInt("int"); !ok || val != 42 {
		t.Errorf("GetInt failed, got: %v, want: 42", val)
	}

	md.SetFloat("float", 3.14)
	if val, ok := md.GetFloat("float"); !ok || val != 3.14 {
		t.Errorf("GetFloat failed, got: %v, want: 3.14", val)
	}

	md.SetBool("bool", true)
	if val, ok := md.GetBool("bool"); !ok || !val {
		t.Errorf("GetBool failed, got: %v, want: true", val)
	}
}

func TestMetadataFromMap(t *testing.T) {
	input := map[string]interface{}{
		"str":     "value",
		"int":     42,
		"float":   3.14,
		"bool":    true,
		"int64":   int64(100),
		"float64": float64(2.718),
	}

	md := NewMetadataFromMap(input)

	// Verify all values were correctly converted and stored
	if str, ok := md.GetString("str"); !ok || str != "value" {
		t.Errorf("String conversion failed")
	}
	if i, ok := md.GetInt("int"); !ok || i != 42 {
		t.Errorf("Int conversion failed")
	}
	if f, ok := md.GetFloat("float"); !ok || f != 3.14 {
		t.Errorf("Float conversion failed")
	}
	if b, ok := md.GetBool("bool"); !ok || !b {
		t.Errorf("Bool conversion failed")
	}
}

func TestMetadataJSON(t *testing.T) {
	md := NewMetadata()
	md.SetString("str", "test")
	md.SetInt("int", 42)
	md.SetFloat("float", 3.14)
	md.SetBool("bool", true)

	// Test marshaling
	bytes, err := json.Marshal(md)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Test unmarshaling
	newMd := NewMetadata()
	err = json.Unmarshal(bytes, newMd)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify all values were preserved
	if str, ok := newMd.GetString("str"); !ok || str != "test" {
		t.Errorf("JSON roundtrip failed for string")
	}
	if i, ok := newMd.GetInt("int"); !ok || i != 42 {
		t.Errorf("JSON roundtrip failed for int")
	}
	if f, ok := newMd.GetFloat("float"); !ok || f != 3.14 {
		t.Errorf("JSON roundtrip failed for float")
	}
	if b, ok := newMd.GetBool("bool"); !ok || !b {
		t.Errorf("JSON roundtrip failed for bool")
	}
}

func TestMetadataKeys(t *testing.T) {
	md := NewMetadata()
	md.SetString("str", "test")
	md.SetInt("int", 42)

	keys := md.Keys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}

	// Check if all keys are present
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}
	if !keyMap["str"] || !keyMap["int"] {
		t.Error("Keys() didn't return all expected keys")
	}
}

func TestMetadataNonExistentKeys(t *testing.T) {
	md := NewMetadata()

	if _, ok := md.GetString("nonexistent"); ok {
		t.Error("GetString should return false for nonexistent key")
	}
	if _, ok := md.GetInt("nonexistent"); ok {
		t.Error("GetInt should return false for nonexistent key")
	}
	if _, ok := md.GetFloat("nonexistent"); ok {
		t.Error("GetFloat should return false for nonexistent key")
	}
	if _, ok := md.GetBool("nonexistent"); ok {
		t.Error("GetBool should return false for nonexistent key")
	}
}

func TestMetadataSerialization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name:  "basic types",
			input: `{"str":"test","int":42,"float":3.14,"bool":true}`,
			expected: map[string]interface{}{
				"str":   "test",
				"int":   float64(42),   // Changed from int to float64
				"float": float64(3.14), // Changed from float32 to float64
				"bool":  true,
			},
		},
		{
			name:     "empty object",
			input:    `{}`,
			expected: map[string]interface{}{},
		},
		{
			name:  "null values",
			input: `{"null_field":null,"str":"test"}`,
			expected: map[string]interface{}{
				"str": "test",
			},
		},
		{
			name:    "invalid json",
			input:   `{"broken":}`,
			wantErr: true,
		},
		{
			name:  "number precision",
			input: `{"float":3.141592653589793}`,
			expected: map[string]interface{}{
				"float": 3.141592653589793, // Changed from float32 to float64
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := NewMetadata()
			err := json.Unmarshal([]byte(tt.input), md)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Test serialization roundtrip
			bytes, err := json.Marshal(md)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var result map[string]interface{}
			err = json.Unmarshal(bytes, &result)
			if err != nil {
				t.Fatalf("Unmarshal of result failed: %v", err)
			}

			// Compare result with expected
			for k, expected := range tt.expected {
				got := result[k]
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("key %s: got %v (%T), want %v (%T)", k, got, got, expected, expected)
				}
			}
		})
	}
}
