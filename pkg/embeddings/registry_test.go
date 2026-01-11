package embeddings

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEmbeddingFunction struct {
	name string
}

func (m *mockEmbeddingFunction) EmbedDocuments(_ context.Context, _ []string) ([]Embedding, error) {
	return nil, nil
}

func (m *mockEmbeddingFunction) EmbedQuery(_ context.Context, _ string) (Embedding, error) {
	return nil, nil
}

func (m *mockEmbeddingFunction) Name() string {
	return m.name
}

func (m *mockEmbeddingFunction) GetConfig() EmbeddingFunctionConfig {
	return EmbeddingFunctionConfig{"name": m.name}
}

func (m *mockEmbeddingFunction) DefaultSpace() DistanceMetric {
	return COSINE
}

func (m *mockEmbeddingFunction) SupportedSpaces() []DistanceMetric {
	return []DistanceMetric{COSINE, L2, IP}
}

type mockSparseEmbeddingFunction struct {
	name string
}

func (m *mockSparseEmbeddingFunction) EmbedDocumentsSparse(_ context.Context, _ []string) ([]*SparseVector, error) {
	return nil, nil
}

func (m *mockSparseEmbeddingFunction) EmbedQuerySparse(_ context.Context, _ string) (*SparseVector, error) {
	return nil, nil
}

func (m *mockSparseEmbeddingFunction) Name() string {
	return m.name
}

func (m *mockSparseEmbeddingFunction) GetConfig() EmbeddingFunctionConfig {
	return EmbeddingFunctionConfig{"name": m.name}
}

func TestRegisterAndBuildDense(t *testing.T) {
	name := "test_dense_ef"
	RegisterDense(name, func(_ EmbeddingFunctionConfig) (EmbeddingFunction, error) {
		return &mockEmbeddingFunction{name: name}, nil
	})

	assert.True(t, HasDense(name))

	ef, err := BuildDense(name, nil)
	require.NoError(t, err)
	assert.Equal(t, name, ef.Name())
}

func TestRegisterAndBuildSparse(t *testing.T) {
	name := "test_sparse_ef"
	RegisterSparse(name, func(_ EmbeddingFunctionConfig) (SparseEmbeddingFunction, error) {
		return &mockSparseEmbeddingFunction{name: name}, nil
	})

	assert.True(t, HasSparse(name))

	ef, err := BuildSparse(name, nil)
	require.NoError(t, err)
	assert.Equal(t, name, ef.Name())
}

func TestBuildDenseUnknown(t *testing.T) {
	_, err := BuildDense("nonexistent_dense", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown embedding function")
}

func TestBuildSparseUnknown(t *testing.T) {
	_, err := BuildSparse("nonexistent_sparse", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown sparse embedding function")
}

func TestListDense(t *testing.T) {
	name := "test_list_dense"
	RegisterDense(name, func(_ EmbeddingFunctionConfig) (EmbeddingFunction, error) {
		return &mockEmbeddingFunction{name: name}, nil
	})

	names := ListDense()
	assert.Contains(t, names, name)
}

func TestListSparse(t *testing.T) {
	name := "test_list_sparse"
	RegisterSparse(name, func(_ EmbeddingFunctionConfig) (SparseEmbeddingFunction, error) {
		return &mockSparseEmbeddingFunction{name: name}, nil
	})

	names := ListSparse()
	assert.Contains(t, names, name)
}

func TestHasDense(t *testing.T) {
	name := "test_has_dense"
	assert.False(t, HasDense(name))

	RegisterDense(name, func(_ EmbeddingFunctionConfig) (EmbeddingFunction, error) {
		return &mockEmbeddingFunction{name: name}, nil
	})

	assert.True(t, HasDense(name))
}

func TestHasSparse(t *testing.T) {
	name := "test_has_sparse"
	assert.False(t, HasSparse(name))

	RegisterSparse(name, func(_ EmbeddingFunctionConfig) (SparseEmbeddingFunction, error) {
		return &mockSparseEmbeddingFunction{name: name}, nil
	})

	assert.True(t, HasSparse(name))
}
