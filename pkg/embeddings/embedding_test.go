package embeddings

import (
	"encoding/json"
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
}
