package ort

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func TestNewOrtEmbeddingFunctionResolves(t *testing.T) {
	ef, closeEF, err := NewOrtEmbeddingFunction()
	if err != nil {
		t.Skipf("ORT embedding function unavailable in this environment: %v", err)
	}
	defer closeEF()

	assert.Equal(t, "default", ef.Name())
}

func TestBuildDenseAliasesForOrt(t *testing.T) {
	names := []string{"default", "ort", "onnx_mini_lm_l6_v2"}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			ef, closeFn, err := embeddings.BuildDenseCloseable(name, embeddings.EmbeddingFunctionConfig{})
			if err != nil {
				t.Skipf("BuildDense(%s) unavailable in this environment: %v", name, err)
			}
			require.NotNil(t, ef)
			require.NotNil(t, closeFn)
			require.NoError(t, closeFn())
		})
	}
}

