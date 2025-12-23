package embeddings

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshalEmbeddings(t *testing.T) {
	embed := NewEmbeddingFromFloat32([]float32{1.1234567891, 2.4, 3.5})

	bytes, err := json.Marshal(embed)
	require.NoError(t, err)
	require.JSONEq(t, `[1.1234568,2.4,3.5]`, string(bytes))
}

func TestUnmarshalEmbeddings(t *testing.T) {
	var embed Float32Embedding
	jsonStr := `[1.1234568,2.4,3.5]`

	err := json.Unmarshal([]byte(jsonStr), &embed)
	require.NoError(t, err)
	require.Equal(t, 3, embed.Len())
	require.Equal(t, float32(1.1234568), embed.ContentAsFloat32()[0])
	require.Equal(t, float32(2.4), embed.ContentAsFloat32()[1])
	require.Equal(t, float32(3.5), embed.ContentAsFloat32()[2])
}

func TestSparseVectorValidate(t *testing.T) {
	t.Run("valid sparse vector", func(t *testing.T) {
		sv, err := NewSparseVector([]int{1, 5, 10}, []float32{0.5, 0.3, 0.8})
		require.NoError(t, err)
		require.NoError(t, sv.Validate())
	})

	t.Run("mismatched lengths", func(t *testing.T) {
		_, err := NewSparseVector([]int{1, 5, 10}, []float32{0.5})
		require.Error(t, err)
		require.Contains(t, err.Error(), "same length")
	})

	t.Run("nil sparse vector", func(t *testing.T) {
		var sv *SparseVector
		err := sv.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "nil")
	})

	t.Run("empty sparse vector is valid", func(t *testing.T) {
		sv, err := NewSparseVector([]int{}, []float32{})
		require.NoError(t, err)
		require.NoError(t, sv.Validate())
	})

	t.Run("negative index at construction", func(t *testing.T) {
		_, err := NewSparseVector([]int{1, -5, 10}, []float32{0.5, 0.3, 0.8})
		require.Error(t, err)
		require.Contains(t, err.Error(), "negative")
		require.Contains(t, err.Error(), "position 1")
	})

	t.Run("negative index in validate", func(t *testing.T) {
		sv := &SparseVector{
			Indices: []int{0, -1, 2},
			Values:  []float32{0.1, 0.2, 0.3},
		}
		err := sv.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "negative")
	})

	t.Run("duplicate index at construction", func(t *testing.T) {
		_, err := NewSparseVector([]int{1, 5, 1}, []float32{0.1, 0.2, 0.3})
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate index")
	})

	t.Run("duplicate index in validate", func(t *testing.T) {
		sv := &SparseVector{
			Indices: []int{1, 5, 1},
			Values:  []float32{0.1, 0.2, 0.3},
		}
		err := sv.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate index")
	})

	t.Run("NaN value at construction", func(t *testing.T) {
		nan := float32(math.NaN())
		_, err := NewSparseVector([]int{1, 2}, []float32{0.5, nan})
		require.Error(t, err)
		require.Contains(t, err.Error(), "NaN")
	})

	t.Run("NaN value in validate", func(t *testing.T) {
		nan := float32(math.NaN())
		sv := &SparseVector{
			Indices: []int{1, 2},
			Values:  []float32{0.5, nan},
		}
		err := sv.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "NaN")
	})

	t.Run("positive infinity at construction", func(t *testing.T) {
		inf := float32(math.Inf(1))
		_, err := NewSparseVector([]int{1, 2}, []float32{0.5, inf})
		require.Error(t, err)
		require.Contains(t, err.Error(), "infinite")
	})

	t.Run("negative infinity at construction", func(t *testing.T) {
		inf := float32(math.Inf(-1))
		_, err := NewSparseVector([]int{1, 2}, []float32{inf, 0.5})
		require.Error(t, err)
		require.Contains(t, err.Error(), "infinite")
	})

	t.Run("infinity in validate", func(t *testing.T) {
		inf := float32(math.Inf(1))
		sv := &SparseVector{
			Indices: []int{1, 2},
			Values:  []float32{0.5, inf},
		}
		err := sv.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "infinite")
	})
}
