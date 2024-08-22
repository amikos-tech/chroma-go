package defaultef

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Default_EF(t *testing.T) {
	ef, closeEf, err := NewDefaultEmbeddingFunction()
	require.NoError(t, err)
	t.Cleanup(closeEf)
	require.NotNil(t, ef)
	embeddings, err := ef.EmbedDocuments(context.TODO(), []string{"test"})
	require.NoError(t, err)
	require.NotNil(t, embeddings)
	require.Len(t, embeddings, 1)
	require.Equal(t, embeddings[0].Len(), 384)
}
