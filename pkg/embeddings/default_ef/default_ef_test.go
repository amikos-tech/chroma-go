package defaultef

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Default_EF(t *testing.T) {
	ef, closeEf, err := NewDefaultEmbeddingFunction()
	require.NoError(t, err)
	t.Cleanup(func() {
		err := closeEf()
		if err != nil {
			t.Logf("error while closing embedding function: %v", err)
		}
	})
	require.NotNil(t, ef)
	embeddings, err := ef.EmbedDocuments(context.TODO(), []string{"Hello Chroma!", "Hello world!"})
	require.NoError(t, err)
	require.NotNil(t, embeddings)
	require.Len(t, embeddings, 2)
	for _, embedding := range embeddings {
		require.Equal(t, embedding.Len(), 384)
	}
}

func TestClose(t *testing.T) {
	ef, closeEf, err := NewDefaultEmbeddingFunction()
	require.NoError(t, err)
	require.NotNil(t, ef)
	err = closeEf()
	require.NoError(t, err)
	_, err = ef.EmbedQuery(context.TODO(), "Hello Chroma!")
	require.Error(t, err)
	require.Contains(t, err.Error(), "embedding function is closed")
}
func TestCloseClosed(t *testing.T) {
	ef := &DefaultEmbeddingFunction{}
	err := ef.Close()
	require.NoError(t, err)
}
