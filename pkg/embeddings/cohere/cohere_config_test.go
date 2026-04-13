package cohere

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewCohereEmbeddingFunction_UsesLegacyDefaultModel(t *testing.T) {
	t.Parallel()

	ef, err := NewCohereEmbeddingFunction(WithAPIKey("test-key"))
	require.NoError(t, err)
	require.Equal(t, ModelEmbedEnglishV20, ef.DefaultModel)
	require.Equal(t, string(ModelEmbedEnglishV20), ef.GetConfig()["model_name"])
}
