//go:build basicv2 && !cloud

package v2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// --- MetadataValue array getters ---

func TestMetadataValueGetStringArray(t *testing.T) {
	mv := MetadataValue{StringArray: []string{"a", "b", "c"}}
	arr, ok := mv.GetStringArray()
	require.True(t, ok)
	require.Equal(t, []string{"a", "b", "c"}, arr)
}

func TestMetadataValueGetStringArrayNil(t *testing.T) {
	mv := MetadataValue{}
	_, ok := mv.GetStringArray()
	require.False(t, ok)
}

func TestMetadataValueGetIntArray(t *testing.T) {
	mv := MetadataValue{IntArray: []int64{1, 2, 3}}
	arr, ok := mv.GetIntArray()
	require.True(t, ok)
	require.Equal(t, []int64{1, 2, 3}, arr)
}

func TestMetadataValueGetIntArrayNil(t *testing.T) {
	mv := MetadataValue{}
	_, ok := mv.GetIntArray()
	require.False(t, ok)
}

func TestMetadataValueGetFloatArray(t *testing.T) {
	mv := MetadataValue{FloatArray: []float64{1.1, 2.2}}
	arr, ok := mv.GetFloatArray()
	require.True(t, ok)
	require.Equal(t, []float64{1.1, 2.2}, arr)
}

func TestMetadataValueGetFloatArrayNil(t *testing.T) {
	mv := MetadataValue{}
	_, ok := mv.GetFloatArray()
	require.False(t, ok)
}

func TestMetadataValueGetBoolArray(t *testing.T) {
	mv := MetadataValue{BoolArray: []bool{true, false, true}}
	arr, ok := mv.GetBoolArray()
	require.True(t, ok)
	require.Equal(t, []bool{true, false, true}, arr)
}

func TestMetadataValueGetBoolArrayNil(t *testing.T) {
	mv := MetadataValue{}
	_, ok := mv.GetBoolArray()
	require.False(t, ok)
}

// --- MetadataValue GetRaw for arrays ---

func TestMetadataValueGetRawStringArray(t *testing.T) {
	mv := MetadataValue{StringArray: []string{"x", "y"}}
	raw, ok := mv.GetRaw()
	require.True(t, ok)
	require.Equal(t, []string{"x", "y"}, raw)
}

func TestMetadataValueGetRawIntArray(t *testing.T) {
	mv := MetadataValue{IntArray: []int64{10, 20}}
	raw, ok := mv.GetRaw()
	require.True(t, ok)
	require.Equal(t, []int64{10, 20}, raw)
}

func TestMetadataValueGetRawFloatArray(t *testing.T) {
	mv := MetadataValue{FloatArray: []float64{1.5, 2.5}}
	raw, ok := mv.GetRaw()
	require.True(t, ok)
	require.Equal(t, []float64{1.5, 2.5}, raw)
}

func TestMetadataValueGetRawBoolArray(t *testing.T) {
	mv := MetadataValue{BoolArray: []bool{true}}
	raw, ok := mv.GetRaw()
	require.True(t, ok)
	require.Equal(t, []bool{true}, raw)
}

// --- MetadataValue.Equal for arrays ---

func TestMetadataValueEqualArrays(t *testing.T) {
	tests := []struct {
		name  string
		a, b  MetadataValue
		equal bool
	}{
		{
			name:  "equal string arrays",
			a:     MetadataValue{StringArray: []string{"a", "b"}},
			b:     MetadataValue{StringArray: []string{"a", "b"}},
			equal: true,
		},
		{
			name:  "unequal string arrays",
			a:     MetadataValue{StringArray: []string{"a", "b"}},
			b:     MetadataValue{StringArray: []string{"a", "c"}},
			equal: false,
		},
		{
			name:  "equal int arrays",
			a:     MetadataValue{IntArray: []int64{1, 2}},
			b:     MetadataValue{IntArray: []int64{1, 2}},
			equal: true,
		},
		{
			name:  "equal float arrays",
			a:     MetadataValue{FloatArray: []float64{1.1, 2.2}},
			b:     MetadataValue{FloatArray: []float64{1.1, 2.2}},
			equal: true,
		},
		{
			name:  "equal bool arrays",
			a:     MetadataValue{BoolArray: []bool{true, false}},
			b:     MetadataValue{BoolArray: []bool{true, false}},
			equal: true,
		},
		{
			name:  "string array vs int array",
			a:     MetadataValue{StringArray: []string{"1"}},
			b:     MetadataValue{IntArray: []int64{1}},
			equal: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.equal, tt.a.Equal(&tt.b))
		})
	}
}

// --- MetadataValue.String for arrays ---

func TestMetadataValueStringArrays(t *testing.T) {
	require.Equal(t, "[a b c]", (&MetadataValue{StringArray: []string{"a", "b", "c"}}).String())
	require.Equal(t, "[1 2 3]", (&MetadataValue{IntArray: []int64{1, 2, 3}}).String())
	require.Equal(t, "[true false]", (&MetadataValue{BoolArray: []bool{true, false}}).String())
}

// --- MetadataValue JSON marshal/unmarshal for arrays ---

func TestMetadataValueMarshalStringArray(t *testing.T) {
	mv := MetadataValue{StringArray: []string{"hello", "world"}}
	b, err := json.Marshal(&mv)
	require.NoError(t, err)
	require.JSONEq(t, `["hello","world"]`, string(b))
}

func TestMetadataValueMarshalIntArray(t *testing.T) {
	mv := MetadataValue{IntArray: []int64{1, 2, 3}}
	b, err := json.Marshal(&mv)
	require.NoError(t, err)
	require.JSONEq(t, `[1,2,3]`, string(b))
}

func TestMetadataValueMarshalFloatArray(t *testing.T) {
	mv := MetadataValue{FloatArray: []float64{1.5, 2.5}}
	b, err := json.Marshal(&mv)
	require.NoError(t, err)
	require.JSONEq(t, `[1.5,2.5]`, string(b))
}

func TestMetadataValueMarshalBoolArray(t *testing.T) {
	mv := MetadataValue{BoolArray: []bool{true, false}}
	b, err := json.Marshal(&mv)
	require.NoError(t, err)
	require.JSONEq(t, `[true,false]`, string(b))
}

func TestMetadataValueUnmarshalStringArray(t *testing.T) {
	var mv MetadataValue
	err := json.Unmarshal([]byte(`["a","b","c"]`), &mv)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b", "c"}, mv.StringArray)
	require.Nil(t, mv.IntArray)
}

func TestMetadataValueUnmarshalIntArray(t *testing.T) {
	var mv MetadataValue
	err := json.Unmarshal([]byte(`[1,2,3]`), &mv)
	require.NoError(t, err)
	require.Equal(t, []int64{1, 2, 3}, mv.IntArray)
	require.Nil(t, mv.FloatArray)
}

func TestMetadataValueUnmarshalFloatArray(t *testing.T) {
	var mv MetadataValue
	err := json.Unmarshal([]byte(`[1.5,2.5]`), &mv)
	require.NoError(t, err)
	require.Equal(t, []float64{1.5, 2.5}, mv.FloatArray)
	require.Nil(t, mv.IntArray)
}

func TestMetadataValueUnmarshalBoolArray(t *testing.T) {
	var mv MetadataValue
	err := json.Unmarshal([]byte(`[true,false,true]`), &mv)
	require.NoError(t, err)
	require.Equal(t, []bool{true, false, true}, mv.BoolArray)
}

func TestMetadataValueUnmarshalEmptyArray(t *testing.T) {
	var mv MetadataValue
	err := json.Unmarshal([]byte(`[]`), &mv)
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-empty")
}

func TestMetadataValueUnmarshalMixedNumericArray(t *testing.T) {
	// Array with mixed int and float should produce float array
	var mv MetadataValue
	err := json.Unmarshal([]byte(`[1, 2.5, 3]`), &mv)
	require.NoError(t, err)
	require.Equal(t, []float64{1.0, 2.5, 3.0}, mv.FloatArray)
}

// --- Array metadata roundtrip (marshal then unmarshal) ---

func TestMetadataValueArrayRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		mv   MetadataValue
	}{
		{"string array", MetadataValue{StringArray: []string{"a", "b"}}},
		{"int array", MetadataValue{IntArray: []int64{10, 20, 30}}},
		{"float array", MetadataValue{FloatArray: []float64{1.1, 2.2}}},
		{"bool array", MetadataValue{BoolArray: []bool{true, false}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(&tt.mv)
			require.NoError(t, err)

			var result MetadataValue
			err = json.Unmarshal(b, &result)
			require.NoError(t, err)

			require.True(t, tt.mv.Equal(&result), "roundtrip failed: got %+v", result)
		})
	}
}

// --- Attribute builders ---

func TestNewStringArrayAttribute(t *testing.T) {
	attr := NewStringArrayAttribute("tags", []string{"a", "b"})
	require.Equal(t, "tags", attr.key)
	arr, ok := attr.value.GetStringArray()
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, arr)
}

func TestNewIntArrayAttribute(t *testing.T) {
	attr := NewIntArrayAttribute("nums", []int64{1, 2, 3})
	require.Equal(t, "nums", attr.key)
	arr, ok := attr.value.GetIntArray()
	require.True(t, ok)
	require.Equal(t, []int64{1, 2, 3}, arr)
}

func TestNewFloatArrayAttribute(t *testing.T) {
	attr := NewFloatArrayAttribute("scores", []float64{1.5, 2.5})
	require.Equal(t, "scores", attr.key)
	arr, ok := attr.value.GetFloatArray()
	require.True(t, ok)
	require.Equal(t, []float64{1.5, 2.5}, arr)
}

func TestNewBoolArrayAttribute(t *testing.T) {
	attr := NewBoolArrayAttribute("flags", []bool{true, false})
	require.Equal(t, "flags", attr.key)
	arr, ok := attr.value.GetBoolArray()
	require.True(t, ok)
	require.Equal(t, []bool{true, false}, arr)
}

// --- CollectionMetadata array getters/setters ---

func TestCollectionMetadataArrayOperations(t *testing.T) {
	md := NewMetadata(
		NewStringArrayAttribute("tags", []string{"science", "math"}),
		NewIntArrayAttribute("years", []int64{2020, 2021}),
		NewFloatArrayAttribute("scores", []float64{9.5, 8.3}),
		NewBoolArrayAttribute("flags", []bool{true, false}),
	)

	tags, ok := md.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"science", "math"}, tags)

	years, ok := md.GetIntArray("years")
	require.True(t, ok)
	require.Equal(t, []int64{2020, 2021}, years)

	scores, ok := md.GetFloatArray("scores")
	require.True(t, ok)
	require.Equal(t, []float64{9.5, 8.3}, scores)

	flags, ok := md.GetBoolArray("flags")
	require.True(t, ok)
	require.Equal(t, []bool{true, false}, flags)

	// Nonexistent keys
	_, ok = md.GetStringArray("nonexistent")
	require.False(t, ok)
}

func TestCollectionMetadataSetArrays(t *testing.T) {
	md := NewEmptyMetadata()
	md.SetStringArray("tags", []string{"a"})
	md.SetIntArray("nums", []int64{1})
	md.SetFloatArray("scores", []float64{1.0})
	md.SetBoolArray("flags", []bool{true})

	tags, ok := md.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"a"}, tags)

	nums, ok := md.GetIntArray("nums")
	require.True(t, ok)
	require.Equal(t, []int64{1}, nums)

	scores, ok := md.GetFloatArray("scores")
	require.True(t, ok)
	require.Equal(t, []float64{1.0}, scores)

	flags, ok := md.GetBoolArray("flags")
	require.True(t, ok)
	require.Equal(t, []bool{true}, flags)
}

// --- DocumentMetadata array getters/setters ---

func TestDocumentMetadataArrayOperations(t *testing.T) {
	md := NewDocumentMetadata(
		NewStringArrayAttribute("tags", []string{"physics", "biology"}),
		NewIntArrayAttribute("pages", []int64{10, 20, 30}),
	)

	tags, ok := md.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"physics", "biology"}, tags)

	pages, ok := md.GetIntArray("pages")
	require.True(t, ok)
	require.Equal(t, []int64{10, 20, 30}, pages)
}

func TestDocumentMetadataSetArrays(t *testing.T) {
	md := NewDocumentMetadata()
	md.SetStringArray("tags", []string{"a", "b"})
	md.SetIntArray("nums", []int64{1, 2})
	md.SetFloatArray("scores", []float64{1.1, 2.2})
	md.SetBoolArray("flags", []bool{true})

	tags, ok := md.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, tags)
}

func TestDocumentMetadataSetRawArrays(t *testing.T) {
	md := NewDocumentMetadata()
	md.SetRaw("tags", []string{"x", "y"})
	md.SetRaw("nums", []int64{5, 6})
	md.SetRaw("scores", []float64{1.0, 2.0})
	md.SetRaw("flags", []bool{false})

	tags, ok := md.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"x", "y"}, tags)

	nums, ok := md.GetIntArray("nums")
	require.True(t, ok)
	require.Equal(t, []int64{5, 6}, nums)
}

// --- DocumentMetadata JSON roundtrip with arrays ---

func TestDocumentMetadataArrayJSONRoundtrip(t *testing.T) {
	md := NewDocumentMetadata(
		NewStringAttribute("title", "test"),
		NewStringArrayAttribute("tags", []string{"a", "b"}),
		NewIntArrayAttribute("years", []int64{2020, 2021}),
		NewFloatArrayAttribute("scores", []float64{9.5, 8.3}),
		NewBoolArrayAttribute("flags", []bool{true, false}),
	)

	b, err := json.Marshal(md)
	require.NoError(t, err)

	// Unmarshal into a new metadata
	newMd := &DocumentMetadataImpl{}
	err = json.Unmarshal(b, newMd)
	require.NoError(t, err)

	title, ok := newMd.GetString("title")
	require.True(t, ok)
	require.Equal(t, "test", title)

	tags, ok := newMd.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, tags)

	years, ok := newMd.GetIntArray("years")
	require.True(t, ok)
	require.Equal(t, []int64{2020, 2021}, years)

	scores, ok := newMd.GetFloatArray("scores")
	require.True(t, ok)
	require.Equal(t, []float64{9.5, 8.3}, scores)

	flags, ok := newMd.GetBoolArray("flags")
	require.True(t, ok)
	require.Equal(t, []bool{true, false}, flags)
}

// --- NewMetadataFromMap with arrays ---

func TestNewMetadataFromMapArrays(t *testing.T) {
	input := map[string]interface{}{
		"tags":   []string{"a", "b"},
		"nums":   []int64{1, 2, 3},
		"scores": []float64{1.5, 2.5},
		"flags":  []bool{true, false},
		"str":    "value",
	}
	md := NewMetadataFromMap(input)

	tags, ok := md.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, tags)

	nums, ok := md.GetIntArray("nums")
	require.True(t, ok)
	require.Equal(t, []int64{1, 2, 3}, nums)

	scores, ok := md.GetFloatArray("scores")
	require.True(t, ok)
	require.Equal(t, []float64{1.5, 2.5}, scores)

	flags, ok := md.GetBoolArray("flags")
	require.True(t, ok)
	require.Equal(t, []bool{true, false}, flags)

	str, ok := md.GetString("str")
	require.True(t, ok)
	require.Equal(t, "value", str)
}

func TestNewMetadataFromMapInterfaceSlice(t *testing.T) {
	// This tests the []interface{} path (as produced by json.Unmarshal)
	input := map[string]interface{}{
		"tags": []interface{}{"a", "b"},
	}
	md := NewMetadataFromMap(input)

	tags, ok := md.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, tags)
}

// --- NewDocumentMetadataFromMap with arrays ---

func TestNewDocumentMetadataFromMapArrays(t *testing.T) {
	input := map[string]interface{}{
		"tags":   []string{"x", "y"},
		"scores": []float64{7.0, 8.0},
	}
	md, err := NewDocumentMetadataFromMap(input)
	require.NoError(t, err)

	tags, ok := md.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"x", "y"}, tags)

	scores, ok := md.GetFloatArray("scores")
	require.True(t, ok)
	require.Equal(t, []float64{7.0, 8.0}, scores)
}

func TestNewDocumentMetadataFromMapInterfaceSlice(t *testing.T) {
	input := map[string]interface{}{
		"tags": []interface{}{"a", "b", "c"},
		"nums": []interface{}{float64(1), float64(2)},
	}
	md, err := NewDocumentMetadataFromMap(input)
	require.NoError(t, err)

	tags, ok := md.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"a", "b", "c"}, tags)

	nums, ok := md.GetFloatArray("nums")
	require.True(t, ok)
	require.Equal(t, []float64{1, 2}, nums)
}

func TestNewDocumentMetadataFromMapEmptyArray(t *testing.T) {
	input := map[string]interface{}{
		"tags": []interface{}{},
	}
	_, err := NewDocumentMetadataFromMap(input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-empty")
}

func TestNewDocumentMetadataFromMapMixedTypeArray(t *testing.T) {
	input := map[string]interface{}{
		"tags": []interface{}{"a", 42},
	}
	_, err := NewDocumentMetadataFromMap(input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mixed types")
}

// --- ValidateArrayMetadata ---

func TestValidateArrayMetadataNonEmpty(t *testing.T) {
	tests := []struct {
		name    string
		mv      MetadataValue
		wantErr bool
	}{
		{"valid string array", MetadataValue{StringArray: []string{"a"}}, false},
		{"valid int array", MetadataValue{IntArray: []int64{1}}, false},
		{"valid float array", MetadataValue{FloatArray: []float64{1.0}}, false},
		{"valid bool array", MetadataValue{BoolArray: []bool{true}}, false},
		{"empty string array", MetadataValue{StringArray: []string{}}, true},
		{"empty int array", MetadataValue{IntArray: []int64{}}, true},
		{"empty float array", MetadataValue{FloatArray: []float64{}}, true},
		{"empty bool array", MetadataValue{BoolArray: []bool{}}, true},
		{"scalar value", MetadataValue{StringValue: strPtr("hello")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateArrayMetadata(&tt.mv)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// --- CollectionMetadata JSON roundtrip with arrays ---

func TestCollectionMetadataArrayJSONRoundtrip(t *testing.T) {
	md := NewMetadata(
		NewStringAttribute("title", "test"),
		NewStringArrayAttribute("tags", []string{"a", "b"}),
		NewIntArrayAttribute("years", []int64{2020, 2021}),
	)

	b, err := json.Marshal(md)
	require.NoError(t, err)

	newMd := NewMetadata()
	err = newMd.UnmarshalJSON(b)
	require.NoError(t, err)

	title, ok := newMd.GetString("title")
	require.True(t, ok)
	require.Equal(t, "test", title)

	tags, ok := newMd.GetStringArray("tags")
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, tags)

	years, ok := newMd.GetIntArray("years")
	require.True(t, ok)
	require.Equal(t, []int64{2020, 2021}, years)
}

// --- convertInterfaceSliceToMetadataValue ---

func TestConvertInterfaceSliceStringArray(t *testing.T) {
	mv, err := convertInterfaceSliceToMetadataValue([]interface{}{"a", "b"})
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, mv.StringArray)
}

func TestConvertInterfaceSliceBoolArray(t *testing.T) {
	mv, err := convertInterfaceSliceToMetadataValue([]interface{}{true, false})
	require.NoError(t, err)
	require.Equal(t, []bool{true, false}, mv.BoolArray)
}

func TestConvertInterfaceSliceFloat64Array(t *testing.T) {
	mv, err := convertInterfaceSliceToMetadataValue([]interface{}{1.5, 2.5})
	require.NoError(t, err)
	require.Equal(t, []float64{1.5, 2.5}, mv.FloatArray)
}

func TestConvertInterfaceSliceEmpty(t *testing.T) {
	_, err := convertInterfaceSliceToMetadataValue([]interface{}{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-empty")
}

func TestConvertInterfaceSliceMixedTypes(t *testing.T) {
	_, err := convertInterfaceSliceToMetadataValue([]interface{}{"a", 42})
	require.Error(t, err)
	require.Contains(t, err.Error(), "mixed types")
}

func TestConvertInterfaceSliceUnsupportedType(t *testing.T) {
	_, err := convertInterfaceSliceToMetadataValue([]interface{}{struct{}{}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported")
}

// helper
func strPtr(s string) *string {
	return &s
}
