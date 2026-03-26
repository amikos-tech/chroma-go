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
	t.Cleanup(func() { unregisterDense(name) })

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
	t.Cleanup(func() { unregisterSparse(name) })

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
	t.Cleanup(func() { unregisterDense(name) })

	names := ListDense()
	assert.Contains(t, names, name)
}

func TestListSparse(t *testing.T) {
	name := "test_list_sparse"
	err := RegisterSparse(name, func(_ EmbeddingFunctionConfig) (SparseEmbeddingFunction, error) {
		return &mockSparseEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { unregisterSparse(name) })

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
	t.Cleanup(func() { unregisterDense(name) })

	assert.True(t, HasDense(name))
}

func TestHasSparse(t *testing.T) {
	name := "test_has_sparse"
	assert.False(t, HasSparse(name))

	err := RegisterSparse(name, func(_ EmbeddingFunctionConfig) (SparseEmbeddingFunction, error) {
		return &mockSparseEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { unregisterSparse(name) })

	assert.True(t, HasSparse(name))
}

func TestRegisterDenseDuplicate(t *testing.T) {
	name := "test_dense_duplicate"
	err := RegisterDense(name, func(_ EmbeddingFunctionConfig) (EmbeddingFunction, error) {
		return &mockEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { unregisterDense(name) })

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
	t.Cleanup(func() { unregisterSparse(name) })

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
	t.Cleanup(func() { unregisterDense(name) })

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
	t.Cleanup(func() { unregisterDense(name) })

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
	t.Cleanup(func() { unregisterSparse(name) })

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
	t.Cleanup(func() { unregisterContent(name) })

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
	t.Cleanup(func() { unregisterMultimodal(name) })

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
	t.Cleanup(func() { unregisterDense(name) })

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
	t.Cleanup(func() { unregisterDense(name) })

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
	t.Cleanup(func() { unregisterDense(name) })

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
	t.Cleanup(func() { unregisterDense(name) })

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
	t.Cleanup(func() { unregisterContent(name) })

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
	t.Cleanup(func() { unregisterContent(name) })

	assert.True(t, HasContent(name))
}

func TestRegisterContentDuplicate(t *testing.T) {
	name := "test_content_duplicate"
	err := RegisterContent(name, func(_ EmbeddingFunctionConfig) (ContentEmbeddingFunction, error) {
		return &mockContentEmbeddingFunction{name: name}, nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { unregisterContent(name) })

	err = RegisterContent(name, func(_ EmbeddingFunctionConfig) (ContentEmbeddingFunction, error) {
		return &mockContentEmbeddingFunction{name: name}, nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

// mockContentEFWithResult returns a deterministic embedding from EmbedContent/EmbedContents.
type mockContentEFWithResult struct {
	name string
}

func (m *mockContentEFWithResult) EmbedContent(_ context.Context, _ Content) (Embedding, error) {
	return NewEmbeddingFromFloat32([]float32{1.0, 2.0, 3.0}), nil
}

func (m *mockContentEFWithResult) EmbedContents(_ context.Context, contents []Content) ([]Embedding, error) {
	result := make([]Embedding, len(contents))
	for i := range contents {
		result[i] = NewEmbeddingFromFloat32([]float32{1.0, 2.0, 3.0})
	}
	return result, nil
}

// mockDenseEFWithResult returns a deterministic embedding from EmbedDocuments/EmbedQuery.
type mockDenseEFWithResult struct {
	name string
}

func (m *mockDenseEFWithResult) EmbedDocuments(_ context.Context, texts []string) ([]Embedding, error) {
	result := make([]Embedding, len(texts))
	for i := range texts {
		result[i] = NewEmbeddingFromFloat32([]float32{4.0, 5.0, 6.0})
	}
	return result, nil
}

func (m *mockDenseEFWithResult) EmbedQuery(_ context.Context, _ string) (Embedding, error) {
	return NewEmbeddingFromFloat32([]float32{4.0, 5.0, 6.0}), nil
}

func (m *mockDenseEFWithResult) Name() string {
	return m.name
}

func (m *mockDenseEFWithResult) GetConfig() EmbeddingFunctionConfig {
	return EmbeddingFunctionConfig{"name": m.name}
}

func (m *mockDenseEFWithResult) DefaultSpace() DistanceMetric { return COSINE }

func (m *mockDenseEFWithResult) SupportedSpaces() []DistanceMetric {
	return []DistanceMetric{COSINE}
}

// TestBuildContentEmbedContentRoundTrip closes the DOCS-02 criterion 3 gap: proves that
// RegisterContent -> BuildContent -> EmbedContent dispatches end-to-end for the native content path.
func TestBuildContentEmbedContentRoundTrip(t *testing.T) {
	name := "test_content_embed_roundtrip"
	err := RegisterContent(name, func(_ EmbeddingFunctionConfig) (ContentEmbeddingFunction, error) {
		return &mockContentEFWithResult{name: name}, nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { unregisterContent(name) })

	ef, err := BuildContent(name, nil)
	require.NoError(t, err)
	require.NotNil(t, ef)

	content := Content{
		Parts: []Part{NewTextPart("test text")},
	}
	embedding, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, embedding)
	assert.Equal(t, 3, embedding.Len())
	assert.Equal(t, []float32{1.0, 2.0, 3.0}, embedding.ContentAsFloat32())
}

// TestBuildContentAdapterEmbedContentRoundTrip closes the DOCS-02 criterion 3 gap: proves that
// RegisterDense -> BuildContent (adapter fallback) -> EmbedContent dispatches end-to-end.
func TestBuildContentAdapterEmbedContentRoundTrip(t *testing.T) {
	name := "test_content_adapter_roundtrip"
	err := RegisterDense(name, func(_ EmbeddingFunctionConfig) (EmbeddingFunction, error) {
		return &mockDenseEFWithResult{name: name}, nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { unregisterDense(name) })

	ef, err := BuildContent(name, nil)
	require.NoError(t, err)
	require.NotNil(t, ef)

	content := Content{
		Parts: []Part{NewTextPart("test text")},
	}
	embedding, err := ef.EmbedContent(context.Background(), content)
	require.NoError(t, err)
	require.NotNil(t, embedding)
	assert.Equal(t, 3, embedding.Len())
	assert.Equal(t, []float32{4.0, 5.0, 6.0}, embedding.ContentAsFloat32())
}
