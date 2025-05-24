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
