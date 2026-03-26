//go:build basicv2

package v2

import (
	"context"
	"io"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// mockCloseableEF implements embeddings.EmbeddingFunction and io.Closer.
type mockCloseableEF struct {
	closeCount atomic.Int32
}

var _ embeddings.EmbeddingFunction = (*mockCloseableEF)(nil)
var _ io.Closer = (*mockCloseableEF)(nil)

func (m *mockCloseableEF) EmbedDocuments(_ context.Context, _ []string) ([]embeddings.Embedding, error) {
	return []embeddings.Embedding{embeddings.NewEmbeddingFromFloat32([]float32{1, 2, 3})}, nil
}

func (m *mockCloseableEF) EmbedQuery(_ context.Context, _ string) (embeddings.Embedding, error) {
	return embeddings.NewEmbeddingFromFloat32([]float32{1, 2, 3}), nil
}

func (m *mockCloseableEF) Name() string { return "mock" }

func (m *mockCloseableEF) GetConfig() embeddings.EmbeddingFunctionConfig {
	return embeddings.EmbeddingFunctionConfig{"name": "mock"}
}

func (m *mockCloseableEF) DefaultSpace() embeddings.DistanceMetric { return embeddings.L2 }

func (m *mockCloseableEF) SupportedSpaces() []embeddings.DistanceMetric {
	return []embeddings.DistanceMetric{embeddings.L2}
}

func (m *mockCloseableEF) Close() error {
	m.closeCount.Add(1)
	return nil
}

// mockCloseableContentEF implements embeddings.ContentEmbeddingFunction and io.Closer.
type mockCloseableContentEF struct {
	closeCount atomic.Int32
}

var _ embeddings.ContentEmbeddingFunction = (*mockCloseableContentEF)(nil)
var _ io.Closer = (*mockCloseableContentEF)(nil)

func (m *mockCloseableContentEF) EmbedContent(_ context.Context, _ embeddings.Content) (embeddings.Embedding, error) {
	return embeddings.NewEmbeddingFromFloat32([]float32{1, 2, 3}), nil
}

func (m *mockCloseableContentEF) EmbedContents(_ context.Context, _ []embeddings.Content) ([]embeddings.Embedding, error) {
	return []embeddings.Embedding{embeddings.NewEmbeddingFromFloat32([]float32{1, 2, 3})}, nil
}

func (m *mockCloseableContentEF) Close() error {
	m.closeCount.Add(1)
	return nil
}

// mockUnwrappableEF wraps a mockCloseableEF and implements EmbeddingFunctionUnwrapper.
type mockUnwrappableEF struct {
	mockCloseableEF
	inner embeddings.EmbeddingFunction
}

var _ embeddings.EmbeddingFunctionUnwrapper = (*mockUnwrappableEF)(nil)

func (m *mockUnwrappableEF) UnwrapEmbeddingFunction() embeddings.EmbeddingFunction {
	return m.inner
}

// mockNonCloseableEF implements embeddings.EmbeddingFunction but NOT io.Closer.
type mockNonCloseableEF struct{}

var _ embeddings.EmbeddingFunction = (*mockNonCloseableEF)(nil)

func (m *mockNonCloseableEF) EmbedDocuments(_ context.Context, _ []string) ([]embeddings.Embedding, error) {
	return []embeddings.Embedding{embeddings.NewEmbeddingFromFloat32([]float32{1, 2, 3})}, nil
}

func (m *mockNonCloseableEF) EmbedQuery(_ context.Context, _ string) (embeddings.Embedding, error) {
	return embeddings.NewEmbeddingFromFloat32([]float32{1, 2, 3}), nil
}

func (m *mockNonCloseableEF) Name() string { return "mock-no-close" }

func (m *mockNonCloseableEF) GetConfig() embeddings.EmbeddingFunctionConfig {
	return embeddings.EmbeddingFunctionConfig{"name": "mock-no-close"}
}

func (m *mockNonCloseableEF) DefaultSpace() embeddings.DistanceMetric { return embeddings.L2 }

func (m *mockNonCloseableEF) SupportedSpaces() []embeddings.DistanceMetric {
	return []embeddings.DistanceMetric{embeddings.L2}
}

func TestCloseOnceEF_IdempotentClose(t *testing.T) {
	inner := &mockCloseableEF{}
	wrapped := wrapEFCloseOnce(inner)

	closer, ok := wrapped.(io.Closer)
	require.True(t, ok, "wrapped EF should implement io.Closer")

	err := closer.Close()
	assert.NoError(t, err)
	assert.Equal(t, int32(1), inner.closeCount.Load())

	err = closer.Close()
	assert.NoError(t, err)
	assert.Equal(t, int32(1), inner.closeCount.Load(), "inner Close() should only be called once")
}

func TestCloseOnceEF_UseAfterClose(t *testing.T) {
	inner := &mockCloseableEF{}
	wrapped := wrapEFCloseOnce(inner)

	closer := wrapped.(io.Closer)
	err := closer.Close()
	require.NoError(t, err)

	ctx := context.Background()

	_, err = wrapped.EmbedDocuments(ctx, []string{"hello"})
	assert.ErrorIs(t, err, errEFClosed)

	_, err = wrapped.EmbedQuery(ctx, "hello")
	assert.ErrorIs(t, err, errEFClosed)
}

func TestCloseOnceEF_DelegatesBeforeClose(t *testing.T) {
	inner := &mockCloseableEF{}
	wrapped := wrapEFCloseOnce(inner)

	ctx := context.Background()

	docs, err := wrapped.EmbedDocuments(ctx, []string{"hello"})
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, []float32{1, 2, 3}, docs[0].ContentAsFloat32())

	query, err := wrapped.EmbedQuery(ctx, "hello")
	require.NoError(t, err)
	assert.Equal(t, []float32{1, 2, 3}, query.ContentAsFloat32())

	assert.Equal(t, "mock", wrapped.Name())
	assert.Equal(t, embeddings.EmbeddingFunctionConfig{"name": "mock"}, wrapped.GetConfig())
	assert.Equal(t, embeddings.L2, wrapped.DefaultSpace())
	assert.Equal(t, []embeddings.DistanceMetric{embeddings.L2}, wrapped.SupportedSpaces())
}

func TestCloseOnceEF_NonCloseableInner(t *testing.T) {
	inner := &mockNonCloseableEF{}
	wrapped := wrapEFCloseOnce(inner)

	closer, ok := wrapped.(io.Closer)
	require.True(t, ok, "wrapper always implements io.Closer")

	err := closer.Close()
	assert.NoError(t, err, "closing a non-closeable inner should return nil")
}

func TestCloseOnceEF_UnwrapperDelegation(t *testing.T) {
	realInner := &mockCloseableEF{}
	unwrappable := &mockUnwrappableEF{inner: realInner}
	wrapped := wrapEFCloseOnce(unwrappable)

	unwrapper, ok := wrapped.(embeddings.EmbeddingFunctionUnwrapper)
	require.True(t, ok, "wrapped EF should implement EmbeddingFunctionUnwrapper")

	unwrapped := unwrapper.UnwrapEmbeddingFunction()
	assert.Same(t, realInner, unwrapped, "should delegate to inner's UnwrapEmbeddingFunction")
}

func TestCloseOnceEF_UnwrapperNonUnwrapper(t *testing.T) {
	inner := &mockCloseableEF{}
	wrapped := wrapEFCloseOnce(inner)

	unwrapper, ok := wrapped.(embeddings.EmbeddingFunctionUnwrapper)
	require.True(t, ok)

	unwrapped := unwrapper.UnwrapEmbeddingFunction()
	assert.Same(t, inner, unwrapped, "should return inner EF when it does not implement EmbeddingFunctionUnwrapper")
}

func TestCloseOnceContentEF_IdempotentClose(t *testing.T) {
	inner := &mockCloseableContentEF{}
	wrapped := wrapContentEFCloseOnce(inner)

	closer, ok := wrapped.(io.Closer)
	require.True(t, ok, "wrapped content EF should implement io.Closer")

	err := closer.Close()
	assert.NoError(t, err)
	assert.Equal(t, int32(1), inner.closeCount.Load())

	err = closer.Close()
	assert.NoError(t, err)
	assert.Equal(t, int32(1), inner.closeCount.Load(), "inner Close() should only be called once")
}

func TestCloseOnceContentEF_UseAfterClose(t *testing.T) {
	inner := &mockCloseableContentEF{}
	wrapped := wrapContentEFCloseOnce(inner)

	closer := wrapped.(io.Closer)
	err := closer.Close()
	require.NoError(t, err)

	ctx := context.Background()
	content := embeddings.Content{}

	_, err = wrapped.EmbedContent(ctx, content)
	assert.ErrorIs(t, err, errEFClosed)

	_, err = wrapped.EmbedContents(ctx, []embeddings.Content{content})
	assert.ErrorIs(t, err, errEFClosed)
}

func TestWrapEFCloseOnce_NilReturnsNil(t *testing.T) {
	assert.Nil(t, wrapEFCloseOnce(nil))
	assert.Nil(t, wrapContentEFCloseOnce(nil))
}

func TestCollectionImpl_ForkOwnsEF(t *testing.T) {
	inner := &mockCloseableEF{}

	nonOwner := &CollectionImpl{
		embeddingFunction: inner,
		ownsEF:            false,
	}
	err := nonOwner.Close()
	assert.NoError(t, err)
	assert.Equal(t, int32(0), inner.closeCount.Load(), "non-owner Close() should not close EF")

	owner := &CollectionImpl{
		embeddingFunction: inner,
		ownsEF:            true,
	}
	err = owner.Close()
	assert.NoError(t, err)
	assert.Equal(t, int32(1), inner.closeCount.Load(), "owner Close() should close EF")
}

func TestEmbeddedCollection_ForkOwnsEF(t *testing.T) {
	inner := &mockCloseableEF{}

	nonOwner := &embeddedCollection{
		embeddingFunction: inner,
		ownsEF:            false,
	}
	err := nonOwner.Close()
	assert.NoError(t, err)
	assert.Equal(t, int32(0), inner.closeCount.Load(), "non-owner Close() should not close EF")

	owner := &embeddedCollection{
		embeddingFunction: inner,
		ownsEF:            true,
	}
	err = owner.Close()
	assert.NoError(t, err)
	assert.Equal(t, int32(1), inner.closeCount.Load(), "owner Close() should close EF")
}
