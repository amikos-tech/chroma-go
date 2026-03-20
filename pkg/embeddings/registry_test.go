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

type mockCloseableEmbeddingFunction struct {
	name   string
	closed bool
}

func (m *mockCloseableEmbeddingFunction) EmbedDocuments(_ context.Context, _ []string) ([]Embedding, error) {
	return nil, nil
}

func (m *mockCloseableEmbeddingFunction) EmbedQuery(_ context.Context, _ string) (Embedding, error) {
	return nil, nil
}

func (m *mockCloseableEmbeddingFunction) Name() string {
	return m.name
}

func (m *mockCloseableEmbeddingFunction) GetConfig() EmbeddingFunctionConfig {
	return EmbeddingFunctionConfig{"name": m.name}
}

func (m *mockCloseableEmbeddingFunction) DefaultSpace() DistanceMetric {
	return COSINE
}

func (m *mockCloseableEmbeddingFunction) SupportedSpaces() []DistanceMetric {
	return []DistanceMetric{COSINE, L2, IP}
}

func (m *mockCloseableEmbeddingFunction) Close() error {
	m.closed = true
	return nil
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
	err := RegisterDense(name, func(_ EmbeddingFunctionConfig) (EmbeddingFunction, error) {
		return &mockEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)

	assert.True(t, HasDense(name))

	ef, err := BuildDense(name, nil)
	require.NoError(t, err)
	assert.Equal(t, name, ef.Name())
}

func TestRegisterAndBuildSparse(t *testing.T) {
	name := "test_sparse_ef"
	err := RegisterSparse(name, func(_ EmbeddingFunctionConfig) (SparseEmbeddingFunction, error) {
		return &mockSparseEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)

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
	err := RegisterDense(name, func(_ EmbeddingFunctionConfig) (EmbeddingFunction, error) {
		return &mockEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)

	names := ListDense()
	assert.Contains(t, names, name)
}

func TestListSparse(t *testing.T) {
	name := "test_list_sparse"
	err := RegisterSparse(name, func(_ EmbeddingFunctionConfig) (SparseEmbeddingFunction, error) {
		return &mockSparseEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)

	names := ListSparse()
	assert.Contains(t, names, name)
}

func TestHasDense(t *testing.T) {
	name := "test_has_dense"
	assert.False(t, HasDense(name))

	err := RegisterDense(name, func(_ EmbeddingFunctionConfig) (EmbeddingFunction, error) {
		return &mockEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)

	assert.True(t, HasDense(name))
}

func TestHasSparse(t *testing.T) {
	name := "test_has_sparse"
	assert.False(t, HasSparse(name))

	err := RegisterSparse(name, func(_ EmbeddingFunctionConfig) (SparseEmbeddingFunction, error) {
		return &mockSparseEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)

	assert.True(t, HasSparse(name))
}

func TestRegisterDenseDuplicate(t *testing.T) {
	name := "test_dense_duplicate"
	err := RegisterDense(name, func(_ EmbeddingFunctionConfig) (EmbeddingFunction, error) {
		return &mockEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)

	err = RegisterDense(name, func(_ EmbeddingFunctionConfig) (EmbeddingFunction, error) {
		return &mockEmbeddingFunction{name: name}, nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestRegisterSparseDuplicate(t *testing.T) {
	name := "test_sparse_duplicate"
	err := RegisterSparse(name, func(_ EmbeddingFunctionConfig) (SparseEmbeddingFunction, error) {
		return &mockSparseEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)

	err = RegisterSparse(name, func(_ EmbeddingFunctionConfig) (SparseEmbeddingFunction, error) {
		return &mockSparseEmbeddingFunction{name: name}, nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestBuildDenseCloseableWithCloseable(t *testing.T) {
	name := "test_dense_closeable"
	mockEf := &mockCloseableEmbeddingFunction{name: name}
	err := RegisterDense(name, func(_ EmbeddingFunctionConfig) (EmbeddingFunction, error) {
		return mockEf, nil
	})
	require.NoError(t, err)

	ef, closer, err := BuildDenseCloseable(name, nil)
	require.NoError(t, err)
	require.NotNil(t, ef)
	require.NotNil(t, closer)
	assert.Equal(t, name, ef.Name())
	assert.False(t, mockEf.closed)

	err = closer()
	require.NoError(t, err)
	assert.True(t, mockEf.closed)
}

func TestBuildDenseCloseableWithoutCloseable(t *testing.T) {
	name := "test_dense_no_closeable"
	err := RegisterDense(name, func(_ EmbeddingFunctionConfig) (EmbeddingFunction, error) {
		return &mockEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)

	ef, closer, err := BuildDenseCloseable(name, nil)
	require.NoError(t, err)
	require.NotNil(t, ef)
	require.NotNil(t, closer)
	assert.Equal(t, name, ef.Name())

	// closer should be a no-op for non-closeable EFs
	err = closer()
	require.NoError(t, err)
}

func TestBuildDenseCloseableUnknown(t *testing.T) {
	_, _, err := BuildDenseCloseable("nonexistent_closeable", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown embedding function")
}

func TestBuildSparseCloseableWithoutCloseable(t *testing.T) {
	name := "test_sparse_closeable"
	err := RegisterSparse(name, func(_ EmbeddingFunctionConfig) (SparseEmbeddingFunction, error) {
		return &mockSparseEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)

	ef, closer, err := BuildSparseCloseable(name, nil)
	require.NoError(t, err)
	require.NotNil(t, ef)
	require.NotNil(t, closer)
	assert.Equal(t, name, ef.Name())

	// closer should be a no-op for non-closeable EFs
	err = closer()
	require.NoError(t, err)
}

func TestBuildSparseCloseableUnknown(t *testing.T) {
	_, _, err := BuildSparseCloseable("nonexistent_sparse_closeable", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown sparse embedding function")
}

// mockContentEmbeddingFunction is a minimal ContentEmbeddingFunction for testing.
type mockContentEmbeddingFunction struct {
	name string
}

func (m *mockContentEmbeddingFunction) EmbedContent(_ context.Context, _ Content) (Embedding, error) {
	return nil, nil
}

func (m *mockContentEmbeddingFunction) EmbedContents(_ context.Context, _ []Content) ([]Embedding, error) {
	return nil, nil
}

// mockMultimodalEmbeddingFunction implements MultimodalEmbeddingFunction for testing.
type mockMultimodalEmbeddingFunction struct {
	mockEmbeddingFunction
}

func (m *mockMultimodalEmbeddingFunction) EmbedImages(_ context.Context, _ []ImageInput) ([]Embedding, error) {
	return nil, nil
}

func (m *mockMultimodalEmbeddingFunction) EmbedImage(_ context.Context, _ ImageInput) (Embedding, error) {
	return nil, nil
}

// mockCapabilityAwareEmbeddingFunction implements EmbeddingFunction + CapabilityAware for testing.
type mockCapabilityAwareEmbeddingFunction struct {
	mockEmbeddingFunction
	caps CapabilityMetadata
}

func (m *mockCapabilityAwareEmbeddingFunction) Capabilities() CapabilityMetadata {
	return m.caps
}

func TestRegisterAndBuildContent(t *testing.T) {
	name := "test_content_ef"
	mock := &mockContentEmbeddingFunction{name: name}
	err := RegisterContent(name, func(_ EmbeddingFunctionConfig) (ContentEmbeddingFunction, error) {
		return mock, nil
	})
	require.NoError(t, err)

	assert.True(t, HasContent(name))

	ef, err := BuildContent(name, nil)
	require.NoError(t, err)
	require.NotNil(t, ef)
}

func TestBuildContentFallbackMultimodal(t *testing.T) {
	name := "test_content_mm_fallback"
	err := RegisterMultimodal(name, func(_ EmbeddingFunctionConfig) (MultimodalEmbeddingFunction, error) {
		return &mockMultimodalEmbeddingFunction{mockEmbeddingFunction: mockEmbeddingFunction{name: name}}, nil
	})
	require.NoError(t, err)

	ef, err := BuildContent(name, nil)
	require.NoError(t, err)
	require.NotNil(t, ef)
	// The adapter implements CapabilityAware
	_, ok := ef.(CapabilityAware)
	assert.True(t, ok)
}

func TestBuildContentFallbackDense(t *testing.T) {
	name := "test_content_dense_fallback"
	err := RegisterDense(name, func(_ EmbeddingFunctionConfig) (EmbeddingFunction, error) {
		return &mockEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)

	ef, err := BuildContent(name, nil)
	require.NoError(t, err)
	require.NotNil(t, ef)
}

func TestBuildContentUnknown(t *testing.T) {
	_, err := BuildContent("nonexistent_content", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown content embedding function")
}

func TestBuildContentCloseableWithCloseable(t *testing.T) {
	name := "test_content_closeable"
	mockEf := &mockCloseableEmbeddingFunction{name: name}
	err := RegisterDense(name, func(_ EmbeddingFunctionConfig) (EmbeddingFunction, error) {
		return mockEf, nil
	})
	require.NoError(t, err)

	ef, closer, err := BuildContentCloseable(name, nil)
	require.NoError(t, err)
	require.NotNil(t, ef)
	require.NotNil(t, closer)
	assert.False(t, mockEf.closed)

	err = closer()
	require.NoError(t, err)
	assert.True(t, mockEf.closed)
}

func TestBuildContentCloseableWithoutCloseable(t *testing.T) {
	name := "test_content_no_closeable"
	err := RegisterDense(name, func(_ EmbeddingFunctionConfig) (EmbeddingFunction, error) {
		return &mockEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)

	ef, closer, err := BuildContentCloseable(name, nil)
	require.NoError(t, err)
	require.NotNil(t, ef)
	require.NotNil(t, closer)

	err = closer()
	require.NoError(t, err)
}

func TestBuildContentFallbackCapabilityAware(t *testing.T) {
	name := "test_content_capaware"
	customCaps := CapabilityMetadata{
		Modalities:    []Modality{ModalityText, ModalityImage},
		SupportsBatch: true,
	}
	err := RegisterDense(name, func(_ EmbeddingFunctionConfig) (EmbeddingFunction, error) {
		return &mockCapabilityAwareEmbeddingFunction{
			mockEmbeddingFunction: mockEmbeddingFunction{name: name},
			caps:                  customCaps,
		}, nil
	})
	require.NoError(t, err)

	ef, err := BuildContent(name, nil)
	require.NoError(t, err)
	require.NotNil(t, ef)

	ca, ok := ef.(CapabilityAware)
	require.True(t, ok)
	assert.Equal(t, customCaps, ca.Capabilities())
}

func TestListContent(t *testing.T) {
	name := "test_list_content"
	err := RegisterContent(name, func(_ EmbeddingFunctionConfig) (ContentEmbeddingFunction, error) {
		return &mockContentEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)

	names := ListContent()
	assert.Contains(t, names, name)
}

func TestHasContent(t *testing.T) {
	name := "test_has_content"
	assert.False(t, HasContent(name))

	err := RegisterContent(name, func(_ EmbeddingFunctionConfig) (ContentEmbeddingFunction, error) {
		return &mockContentEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)

	assert.True(t, HasContent(name))
}

func TestRegisterContentDuplicate(t *testing.T) {
	name := "test_content_duplicate"
	err := RegisterContent(name, func(_ EmbeddingFunctionConfig) (ContentEmbeddingFunction, error) {
		return &mockContentEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)

	err = RegisterContent(name, func(_ EmbeddingFunctionConfig) (ContentEmbeddingFunction, error) {
		return &mockContentEmbeddingFunction{name: name}, nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}
