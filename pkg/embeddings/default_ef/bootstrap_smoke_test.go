package defaultef

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultEF_BootstrapSmoke(t *testing.T) {
	if os.Getenv("RUN_DEFAULT_EF_BOOTSTRAP_SMOKE") != "1" {
		t.Skip("set RUN_DEFAULT_EF_BOOTSTRAP_SMOKE=1 to run default_ef bootstrap smoke test")
	}

	// Isolate cache/model writes for CI runs.
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)
	resetConfigForTesting()
	t.Cleanup(resetConfigForTesting)

	ef, closeEF, err := NewDefaultEmbeddingFunction()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = closeEF()
	})

	embeddings, err := ef.EmbedDocuments(context.Background(), []string{"default_ef runtime smoke"})
	require.NoError(t, err)
	require.Len(t, embeddings, 1)
	require.Equal(t, 384, embeddings[0].Len())
}
