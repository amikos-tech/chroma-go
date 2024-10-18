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
