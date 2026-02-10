//go:build basicv2 && !cloud

package v2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetadataValueGetStringArray(t *testing.T) {
	mv := MetadataValue{StringArray: []string{"a", "b", "c"}}
	arr, ok := mv.GetStringArray()
	require.True(t, ok)
	require.Equal(t, []string{"a", "b", "c"}, arr)
}

func TestMetadataValueGetIntArray(t *testing.T) {
	mv := MetadataValue{IntArray: []int64{1, 2, 3}}
	arr, ok := mv.GetIntArray()
	require.True(t, ok)
	require.Equal(t, []int64{1, 2, 3}, arr)
}

func TestMetadataValueGetFloatArray(t *testing.T) {
	mv := MetadataValue{FloatArray: []float64{1.1, 2.2, 3.3}}
	arr, ok := mv.GetFloatArray()
	require.True(t, ok)
	require.Equal(t, []float64{1.1, 2.2, 3.3}, arr)
}

func TestMetadataValueGetBoolArray(t *testing.T) {
	mv := MetadataValue{BoolArray: []bool{true, false, true}}
	arr, ok := mv.GetBoolArray()
	require.True(t, ok)
	require.Equal(t, []bool{true, false, true}, arr)
}

func TestMetadataValueGetStringArrayMissing(t *testing.T) {
	mv := MetadataValue{}
	_, ok := mv.GetStringArray()
	require.False(t, ok)
}

func TestMetadataValueGetIntArrayMissing(t *testing.T) {
	mv := MetadataValue{}
	_, ok := mv.GetIntArray()
	require.False(t, ok)
}

func TestMetadataValueGetFloatArrayMissing(t *testing.T) {
	mv := MetadataValue{}
	_, ok := mv.GetFloatArray()
	require.False(t, ok)
}

func TestMetadataValueGetBoolArrayMissing(t *testing.T) {
	mv := MetadataValue{}
	_, ok := mv.GetBoolArray()
	require.False(t, ok)
}

func TestMetadataValueGetRawArray(t *testing.T) {
	tests := []struct {
		name     string
		mv       MetadataValue
		expected interface{}
	}{
		{"string array", MetadataValue{StringArray: []string{"a"}}, []string{"a"}},
		{"int array", MetadataValue{IntArray: []int64{1}}, []int64{1}},
		{"float array", MetadataValue{FloatArray: []float64{1.5}}, []float64{1.5}},
		{"bool array", MetadataValue{BoolArray: []bool{true}}, []bool{true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, ok := tt.mv.GetRaw()
			require.True(t, ok)
			require.Equal(t, tt.expected, raw)
		})
	}
}

func TestMetadataValueEqualArrays(t *testing.T) {
	tests := []struct {
		name  string
		a, b  MetadataValue
		equal bool
	}{
		{"same string arrays", MetadataValue{StringArray: []string{"a", "b"}}, MetadataValue{StringArray: []string{"a", "b"}}, true},
		{"different string arrays", MetadataValue{StringArray: []string{"a"}}, MetadataValue{StringArray: []string{"b"}}, false},
		{"same int arrays", MetadataValue{IntArray: []int64{1, 2}}, MetadataValue{IntArray: []int64{1, 2}}, true},
		{"different int arrays", MetadataValue{IntArray: []int64{1}}, MetadataValue{IntArray: []int64{2}}, false},
		{"same float arrays", MetadataValue{FloatArray: []float64{1.1}}, MetadataValue{FloatArray: []float64{1.1}}, true},
		{"different float arrays", MetadataValue{FloatArray: []float64{1.1}}, MetadataValue{FloatArray: []float64{2.2}}, false},
		{"same bool arrays", MetadataValue{BoolArray: []bool{true, false}}, MetadataValue{BoolArray: []bool{true, false}}, true},
		{"different bool arrays", MetadataValue{BoolArray: []bool{true}}, MetadataValue{BoolArray: []bool{false}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.equal, tt.a.Equal(&tt.b))
		})
	}
}

func TestMetadataValueStringRepresentationArrays(t *testing.T) {
	require.Equal(t, "[a b c]", (&MetadataValue{StringArray: []string{"a", "b", "c"}}).String())
	require.Equal(t, "[1 2 3]", (&MetadataValue{IntArray: []int64{1, 2, 3}}).String())
	require.Equal(t, "[1.1 2.2]", (&MetadataValue{FloatArray: []float64{1.1, 2.2}}).String())
	require.Equal(t, "[true false]", (&MetadataValue{BoolArray: []bool{true, false}}).String())
}

func TestMetadataValueJSONMarshalArrays(t *testing.T) {
	tests := []struct {
		name     string
		mv       MetadataValue
		expected string
	}{
		{"string array", MetadataValue{StringArray: []string{"a", "b"}}, `["a","b"]`},
		{"int array", MetadataValue{IntArray: []int64{1, 2}}, `[1,2]`},
		{"float array", MetadataValue{FloatArray: []float64{1.5, 2.5}}, `[1.5,2.5]`},
		{"bool array", MetadataValue{BoolArray: []bool{true, false}}, `[true,false]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(&tt.mv)
			require.NoError(t, err)
			require.JSONEq(t, tt.expected, string(b))
		})
	}
}

func TestMetadataValueJSONUnmarshalArrays(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, mv MetadataValue)
	}{
		{"string array", `["hello","world"]`, func(t *testing.T, mv MetadataValue) {
			arr, ok := mv.GetStringArray()
			require.True(t, ok)
			require.Equal(t, []string{"hello", "world"}, arr)
		}},
		{"int array", `[1,2,3]`, func(t *testing.T, mv MetadataValue) {
			arr, ok := mv.GetIntArray()
			require.True(t, ok)
			require.Equal(t, []int64{1, 2, 3}, arr)
		}},
		{"float array", `[1.5,2.5]`, func(t *testing.T, mv MetadataValue) {
			arr, ok := mv.GetFloatArray()
			require.True(t, ok)
			require.Equal(t, []float64{1.5, 2.5}, arr)
		}},
		{"bool array", `[true,false,true]`, func(t *testing.T, mv MetadataValue) {
			arr, ok := mv.GetBoolArray()
			require.True(t, ok)
			require.Equal(t, []bool{true, false, true}, arr)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mv MetadataValue
			err := json.Unmarshal([]byte(tt.input), &mv)
			require.NoError(t, err)
			tt.check(t, mv)
		})
	}
}

func TestMetadataValueJSONRoundtripArrays(t *testing.T) {
	original := MetadataValue{StringArray: []string{"x", "y", "z"}}
	b, err := json.Marshal(&original)
	require.NoError(t, err)
	var decoded MetadataValue
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)
	require.True(t, original.Equal(&decoded))
}

func TestMetadataValueUnmarshalNestedArrayReject(t *testing.T) {
	var mv MetadataValue
	err := json.Unmarshal([]byte(`[["nested"]]`), &mv)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nested arrays are not supported")
}

func TestMetadataValueUnmarshalObjectArrayReject(t *testing.T) {
	var mv MetadataValue
	err := json.Unmarshal([]byte(`[{"key":"val"}]`), &mv)
	require.Error(t, err)
	require.Contains(t, err.Error(), "arrays of objects are not supported")
}

func TestMetadataValueUnmarshalEmptyArrayReject(t *testing.T) {
	var mv MetadataValue
	err := json.Unmarshal([]byte(`[]`), &mv)
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-empty")
}

func TestValidateArrayMetadataEmpty(t *testing.T) {
	tests := []struct {
		name string
		mv   MetadataValue
	}{
		{"empty string array", MetadataValue{StringArray: []string{}}},
		{"empty int array", MetadataValue{IntArray: []int64{}}},
		{"empty float array", MetadataValue{FloatArray: []float64{}}},
		{"empty bool array", MetadataValue{BoolArray: []bool{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateArrayMetadata(&tt.mv)
			require.Error(t, err)
			require.Contains(t, err.Error(), "non-empty")
		})
	}
}

func TestValidateArrayMetadataValid(t *testing.T) {
	mv := MetadataValue{StringArray: []string{"a"}}
	require.NoError(t, ValidateArrayMetadata(&mv))
}

func TestValidateArrayMetadataScalar(t *testing.T) {
	s := "hello"
	mv := MetadataValue{StringValue: &s}
	require.NoError(t, ValidateArrayMetadata(&mv))
}

func TestNewMetadataFromMapWithArrays(t *testing.T) {
	m := NewMetadataFromMap(map[string]interface{}{
		"tags":   []string{"a", "b"},
		"scores": []int64{1, 2},
		"ratios": []float64{0.5, 1.5},
		"flags":  []bool{true, false},
	})
	arr, ok := m.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, arr)

	intArr, ok := m.GetIntArray("scores")
	require.True(t, ok)
	require.Equal(t, []int64{1, 2}, intArr)

	floatArr, ok := m.GetFloatArray("ratios")
	require.True(t, ok)
	require.Equal(t, []float64{0.5, 1.5}, floatArr)

	boolArr, ok := m.GetBoolArray("flags")
	require.True(t, ok)
	require.Equal(t, []bool{true, false}, boolArr)
}

func TestNewMetadataFromMapWithInterfaceSlice(t *testing.T) {
	m := NewMetadataFromMap(map[string]interface{}{
		"tags": []interface{}{"a", "b"},
	})
	arr, ok := m.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, arr)
}

func TestNewDocumentMetadataFromMapWithArrays(t *testing.T) {
	md, err := NewDocumentMetadataFromMap(map[string]interface{}{
		"tags":   []string{"x", "y"},
		"scores": []int64{10, 20},
		"ratios": []float64{0.1, 0.2},
		"flags":  []bool{true},
	})
	require.NoError(t, err)

	arr, ok := md.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"x", "y"}, arr)

	intArr, ok := md.GetIntArray("scores")
	require.True(t, ok)
	require.Equal(t, []int64{10, 20}, intArr)

	floatArr, ok := md.GetFloatArray("ratios")
	require.True(t, ok)
	require.Equal(t, []float64{0.1, 0.2}, floatArr)

	boolArr, ok := md.GetBoolArray("flags")
	require.True(t, ok)
	require.Equal(t, []bool{true}, boolArr)
}

func TestNewDocumentMetadataFromMapWithInterfaceSlice(t *testing.T) {
	md, err := NewDocumentMetadataFromMap(map[string]interface{}{
		"tags": []interface{}{"a", "b"},
	})
	require.NoError(t, err)
	arr, ok := md.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, arr)
}

func TestConvertInterfaceSliceMixedTypes(t *testing.T) {
	_, err := convertInterfaceSliceToMetadataValue([]interface{}{"a", 1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "mixed types")
}

func TestConvertInterfaceSliceEmpty(t *testing.T) {
	_, err := convertInterfaceSliceToMetadataValue([]interface{}{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-empty")
}

func TestConvertInterfaceSliceNestedArray(t *testing.T) {
	_, err := convertInterfaceSliceToMetadataValue([]interface{}{[]interface{}{"nested"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "nested arrays")
}

func TestCollectionMetadataImplArraySettersGetters(t *testing.T) {
	cm := NewEmptyMetadata()
	cm.SetStringArray("tags", []string{"a", "b"})
	cm.SetIntArray("scores", []int64{1, 2})
	cm.SetFloatArray("ratios", []float64{0.5})
	cm.SetBoolArray("flags", []bool{true})

	arr, ok := cm.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, arr)

	intArr, ok := cm.GetIntArray("scores")
	require.True(t, ok)
	require.Equal(t, []int64{1, 2}, intArr)

	floatArr, ok := cm.GetFloatArray("ratios")
	require.True(t, ok)
	require.Equal(t, []float64{0.5}, floatArr)

	boolArr, ok := cm.GetBoolArray("flags")
	require.True(t, ok)
	require.Equal(t, []bool{true}, boolArr)
}

func TestDocumentMetadataImplArraySettersGetters(t *testing.T) {
	dm := NewDocumentMetadata()
	dm.SetStringArray("tags", []string{"x"})
	dm.SetIntArray("nums", []int64{42})
	dm.SetFloatArray("vals", []float64{3.14})
	dm.SetBoolArray("bools", []bool{false, true})

	arr, ok := dm.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"x"}, arr)

	intArr, ok := dm.GetIntArray("nums")
	require.True(t, ok)
	require.Equal(t, []int64{42}, intArr)

	floatArr, ok := dm.GetFloatArray("vals")
	require.True(t, ok)
	require.Equal(t, []float64{3.14}, floatArr)

	boolArr, ok := dm.GetBoolArray("bools")
	require.True(t, ok)
	require.Equal(t, []bool{false, true}, boolArr)
}

func TestDocumentMetadataImplSetRawArrays(t *testing.T) {
	dm := NewDocumentMetadata()
	dm.SetRaw("tags", []string{"a", "b"})
	dm.SetRaw("nums", []int64{1, 2})
	dm.SetRaw("vals", []float64{1.1})
	dm.SetRaw("bools", []bool{true})

	arr, ok := dm.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, arr)

	intArr, ok := dm.GetIntArray("nums")
	require.True(t, ok)
	require.Equal(t, []int64{1, 2}, intArr)

	floatArr, ok := dm.GetFloatArray("vals")
	require.True(t, ok)
	require.Equal(t, []float64{1.1}, floatArr)

	boolArr, ok := dm.GetBoolArray("bools")
	require.True(t, ok)
	require.Equal(t, []bool{true}, boolArr)
}

func TestCollectionMetadataImplSetRawArrays(t *testing.T) {
	cm := NewEmptyMetadata()
	cm.SetRaw("tags", []string{"a"})
	cm.SetRaw("nums", []int64{1})
	cm.SetRaw("vals", []float64{1.1})
	cm.SetRaw("bools", []bool{true})

	arr, ok := cm.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"a"}, arr)

	intArr, ok := cm.GetIntArray("nums")
	require.True(t, ok)
	require.Equal(t, []int64{1}, intArr)

	floatArr, ok := cm.GetFloatArray("vals")
	require.True(t, ok)
	require.Equal(t, []float64{1.1}, floatArr)

	boolArr, ok := cm.GetBoolArray("bools")
	require.True(t, ok)
	require.Equal(t, []bool{true}, boolArr)
}

func TestCollectionMetadataMarshalJSONWithArrays(t *testing.T) {
	cm := NewEmptyMetadata()
	cm.SetStringArray("tags", []string{"a", "b"})
	cm.SetIntArray("nums", []int64{1, 2})

	b, err := cm.MarshalJSON()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(b, &result)
	require.NoError(t, err)
	require.Contains(t, result, "tags")
	require.Contains(t, result, "nums")
}

func TestDocumentMetadataMarshalJSONWithArrays(t *testing.T) {
	dm := NewDocumentMetadata()
	dm.SetStringArray("tags", []string{"x", "y"})
	dm.SetBoolArray("flags", []bool{true})

	impl := dm.(*DocumentMetadataImpl)
	b, err := impl.MarshalJSON()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(b, &result)
	require.NoError(t, err)
	require.Contains(t, result, "tags")
	require.Contains(t, result, "flags")
}

func TestDocumentMetadataUnmarshalJSONWithArrays(t *testing.T) {
	input := `{"tags":["a","b"],"scores":[1,2],"ratios":[1.5,2.5],"flags":[true,false]}`
	impl := &DocumentMetadataImpl{}
	err := impl.UnmarshalJSON([]byte(input))
	require.NoError(t, err)

	arr, ok := impl.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, arr)

	intArr, ok := impl.GetIntArray("scores")
	require.True(t, ok)
	require.Equal(t, []int64{1, 2}, intArr)

	floatArr, ok := impl.GetFloatArray("ratios")
	require.True(t, ok)
	require.Equal(t, []float64{1.5, 2.5}, floatArr)

	boolArr, ok := impl.GetBoolArray("flags")
	require.True(t, ok)
	require.Equal(t, []bool{true, false}, boolArr)
}

func TestCollectionMetadataUnmarshalJSONWithArrays(t *testing.T) {
	input := `{"tags":["hello","world"],"nums":[10,20]}`
	impl := &CollectionMetadataImpl{}
	err := impl.UnmarshalJSON([]byte(input))
	require.NoError(t, err)

	arr, ok := impl.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"hello", "world"}, arr)

	intArr, ok := impl.GetIntArray("nums")
	require.True(t, ok)
	require.Equal(t, []int64{10, 20}, intArr)
}

func TestNewStringArrayAttribute(t *testing.T) {
	attr := NewStringArrayAttribute("tags", []string{"a", "b"})
	require.Equal(t, "tags", attr.key)
	arr, ok := attr.value.GetStringArray()
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, arr)
}

func TestNewIntArrayAttribute(t *testing.T) {
	attr := NewIntArrayAttribute("nums", []int64{1, 2})
	require.Equal(t, "nums", attr.key)
	arr, ok := attr.value.GetIntArray()
	require.True(t, ok)
	require.Equal(t, []int64{1, 2}, arr)
}

func TestNewFloatArrayAttribute(t *testing.T) {
	attr := NewFloatArrayAttribute("vals", []float64{1.5})
	require.Equal(t, "vals", attr.key)
	arr, ok := attr.value.GetFloatArray()
	require.True(t, ok)
	require.Equal(t, []float64{1.5}, arr)
}

func TestNewBoolArrayAttribute(t *testing.T) {
	attr := NewBoolArrayAttribute("flags", []bool{true, false})
	require.Equal(t, "flags", attr.key)
	arr, ok := attr.value.GetBoolArray()
	require.True(t, ok)
	require.Equal(t, []bool{true, false}, arr)
}

func TestNewArrayAttributeEmptyReturnsNil(t *testing.T) {
	require.Nil(t, NewStringArrayAttribute("k", []string{}))
	require.Nil(t, NewIntArrayAttribute("k", []int64{}))
	require.Nil(t, NewFloatArrayAttribute("k", []float64{}))
	require.Nil(t, NewBoolArrayAttribute("k", []bool{}))
}

func TestNewMetadataSkipsNilAttributes(t *testing.T) {
	md := NewMetadata(
		NewStringAttribute("name", "test"),
		NewStringArrayAttribute("empty", []string{}), // nil, should be skipped
		NewStringArrayAttribute("tags", []string{"a"}),
	)
	_, ok := md.GetStringArray("empty")
	require.False(t, ok)
	arr, ok := md.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"a"}, arr)
}

func TestUnmarshalArrayMixedTypesError(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"bool then string", `[true, "hello"]`},
		{"string then number", `["hello", 42]`},
		{"number then bool", `[42, true]`},
		{"string then bool", `["hello", false]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mv MetadataValue
			err := json.Unmarshal([]byte(tt.input), &mv)
			require.Error(t, err)
			require.Contains(t, err.Error(), "mixed types")
		})
	}
}

func TestUnmarshalArrayNullRejected(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"null first element", `[null, "hello"]`},
		{"null in string array", `["hello", null]`},
		{"null in bool array", `[true, null]`},
		{"null in number array", `[42, null]`},
		{"only null", `[null]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mv MetadataValue
			err := json.Unmarshal([]byte(tt.input), &mv)
			require.Error(t, err)
			require.Contains(t, err.Error(), "null")
		})
	}
}
