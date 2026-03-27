//go:build basicv2

package v2

import (
	"context"
	"fmt"
	"io"
	"sync"
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

// mockFailingCloseEF implements embeddings.EmbeddingFunction and io.Closer, returning an error on Close.
type mockFailingCloseEF struct {
	mockCloseableEF
	closeErr error
}

func (m *mockFailingCloseEF) Close() error {
	m.closeCount.Add(1)
	return m.closeErr
}

// mockFailingCloseContentEF implements ContentEmbeddingFunction and io.Closer, returning an error on Close.
type mockFailingCloseContentEF struct {
	mockCloseableContentEF
	closeErr error
}

func (m *mockFailingCloseContentEF) Close() error {
	m.closeCount.Add(1)
	return m.closeErr
}

// mockCloseableContentEFWithUnwrap implements ContentEmbeddingFunction, io.Closer, and EmbeddingFunctionUnwrapper.
type mockCloseableContentEFWithUnwrap struct {
	mockCloseableContentEF
	inner embeddings.EmbeddingFunction
}

var _ embeddings.EmbeddingFunctionUnwrapper = (*mockCloseableContentEFWithUnwrap)(nil)

func (m *mockCloseableContentEFWithUnwrap) UnwrapEmbeddingFunction() embeddings.EmbeddingFunction {
	return m.inner
}

// mockDualEF implements both EmbeddingFunction and ContentEmbeddingFunction.
type mockDualEF struct {
	mockCloseableEF
}

var _ embeddings.ContentEmbeddingFunction = (*mockDualEF)(nil)

func (m *mockDualEF) EmbedContent(_ context.Context, _ embeddings.Content) (embeddings.Embedding, error) {
	return embeddings.NewEmbeddingFromFloat32([]float32{1, 2, 3}), nil
}

func (m *mockDualEF) EmbedContents(_ context.Context, _ []embeddings.Content) ([]embeddings.Embedding, error) {
	return []embeddings.Embedding{embeddings.NewEmbeddingFromFloat32([]float32{1, 2, 3})}, nil
}

// mockNonCloseableContentEF implements ContentEmbeddingFunction but NOT io.Closer.
type mockNonCloseableContentEF struct{}

var _ embeddings.ContentEmbeddingFunction = (*mockNonCloseableContentEF)(nil)

func (m *mockNonCloseableContentEF) EmbedContent(_ context.Context, _ embeddings.Content) (embeddings.Embedding, error) {
	return embeddings.NewEmbeddingFromFloat32([]float32{1, 2, 3}), nil
}

func (m *mockNonCloseableContentEF) EmbedContents(_ context.Context, _ []embeddings.Content) ([]embeddings.Embedding, error) {
	return []embeddings.Embedding{embeddings.NewEmbeddingFromFloat32([]float32{1, 2, 3})}, nil
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

func TestCloseOnceEF_ConcurrentClose(t *testing.T) {
	inner := &mockCloseableEF{}
	wrapped := wrapEFCloseOnce(inner)
	closer := wrapped.(io.Closer)

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = closer.Close()
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), inner.closeCount.Load(), "inner Close() must be called exactly once under contention")
}

func TestCloseOnceContentEF_ConcurrentClose(t *testing.T) {
	inner := &mockCloseableContentEF{}
	wrapped := wrapContentEFCloseOnce(inner)
	closer := wrapped.(io.Closer)

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = closer.Close()
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), inner.closeCount.Load(), "inner Close() must be called exactly once under contention")
}

func TestForkCloseLifecycle_CollectionImpl(t *testing.T) {
	inner := &mockCloseableEF{}

	parent := &CollectionImpl{
		embeddingFunction: inner,
		ownsEF:            true,
	}

	// Simulate Fork wiring: fork gets wrapped EF and ownsEF=false
	forked := &CollectionImpl{
		embeddingFunction: wrapEFCloseOnce(parent.embeddingFunction),
		ownsEF:            false,
	}

	ctx := context.Background()

	// Fork close does not close inner
	err := forked.Close()
	require.NoError(t, err)
	assert.Equal(t, int32(0), inner.closeCount.Load(), "closing fork must not close parent's EF")

	// Parent EF still usable after fork close
	_, err = inner.EmbedQuery(ctx, "test")
	assert.NoError(t, err, "parent's EF must remain usable after fork close")

	// Parent close closes inner
	err = parent.Close()
	require.NoError(t, err)
	assert.Equal(t, int32(1), inner.closeCount.Load(), "closing parent must close EF")
}

func TestForkCloseLifecycle_EmbeddedCollection(t *testing.T) {
	inner := &mockCloseableEF{}

	parent := &embeddedCollection{
		embeddingFunction: inner,
		ownsEF:            true,
	}

	forked := &embeddedCollection{
		embeddingFunction: wrapEFCloseOnce(parent.embeddingFunction),
		ownsEF:            false,
	}

	ctx := context.Background()

	err := forked.Close()
	require.NoError(t, err)
	assert.Equal(t, int32(0), inner.closeCount.Load(), "closing fork must not close parent's EF")

	_, err = inner.EmbedQuery(ctx, "test")
	assert.NoError(t, err, "parent's EF must remain usable after fork close")

	err = parent.Close()
	require.NoError(t, err)
	assert.Equal(t, int32(1), inner.closeCount.Load(), "closing parent must close EF")
}

func TestCollectionImpl_OwnerDoubleClose(t *testing.T) {
	inner := &mockCloseableEF{}
	owner := &CollectionImpl{
		embeddingFunction: inner,
		ownsEF:            true,
	}

	err := owner.Close()
	assert.NoError(t, err)
	assert.Equal(t, int32(1), inner.closeCount.Load())

	err = owner.Close()
	assert.NoError(t, err)
	assert.Equal(t, int32(1), inner.closeCount.Load(), "owner double-Close must not close EF twice")
}

func TestEmbeddedCollection_OwnerDoubleClose(t *testing.T) {
	inner := &mockCloseableEF{}
	owner := &embeddedCollection{
		embeddingFunction: inner,
		ownsEF:            true,
	}

	err := owner.Close()
	assert.NoError(t, err)
	assert.Equal(t, int32(1), inner.closeCount.Load())

	err = owner.Close()
	assert.NoError(t, err)
	assert.Equal(t, int32(1), inner.closeCount.Load(), "owner double-Close must not close EF twice")
}

func TestCloseOnceEF_ErrorPropagation(t *testing.T) {
	closeErr := fmt.Errorf("close failed")
	inner := &mockFailingCloseEF{closeErr: closeErr}
	wrapped := wrapEFCloseOnce(inner)
	closer := wrapped.(io.Closer)

	err := closer.Close()
	assert.ErrorIs(t, err, closeErr, "first Close must propagate error")

	err = closer.Close()
	assert.ErrorIs(t, err, closeErr, "subsequent Close must return same error")
}

func TestWrapEFCloseOnce_AlreadyWrapped(t *testing.T) {
	inner := &mockCloseableEF{}
	wrapped := wrapEFCloseOnce(inner)
	doubleWrapped := wrapEFCloseOnce(wrapped)
	assert.Same(t, wrapped, doubleWrapped, "wrapping an already-wrapped EF must return the same instance")
}

func TestWrapContentEFCloseOnce_AlreadyWrapped(t *testing.T) {
	inner := &mockCloseableContentEF{}
	wrapped := wrapContentEFCloseOnce(inner)
	doubleWrapped := wrapContentEFCloseOnce(wrapped)
	assert.Same(t, wrapped, doubleWrapped, "wrapping an already-wrapped content EF must return the same instance")
}

func TestCloseOnceContentEF_DelegatesBeforeClose(t *testing.T) {
	inner := &mockCloseableContentEF{}
	wrapped := wrapContentEFCloseOnce(inner)

	ctx := context.Background()

	result, err := wrapped.EmbedContent(ctx, embeddings.Content{})
	require.NoError(t, err)
	assert.Equal(t, []float32{1, 2, 3}, result.ContentAsFloat32())

	results, err := wrapped.EmbedContents(ctx, []embeddings.Content{{}})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, []float32{1, 2, 3}, results[0].ContentAsFloat32())
}

func TestCloseOnceContentEF_NonCloseableInner(t *testing.T) {
	inner := &mockNonCloseableContentEF{}
	wrapped := wrapContentEFCloseOnce(inner)

	closer, ok := wrapped.(io.Closer)
	require.True(t, ok, "wrapper always implements io.Closer")

	err := closer.Close()
	assert.NoError(t, err, "closing a non-closeable inner content EF should return nil")
}

func TestCloseOnceContentEF_UnwrapperDelegation(t *testing.T) {
	realInner := &mockCloseableEF{}
	unwrappable := &mockCloseableContentEFWithUnwrap{inner: realInner}
	wrapped := wrapContentEFCloseOnce(unwrappable)

	unwrapper, ok := wrapped.(embeddings.EmbeddingFunctionUnwrapper)
	require.True(t, ok)

	unwrapped := unwrapper.UnwrapEmbeddingFunction()
	assert.Same(t, realInner, unwrapped, "should delegate to inner's UnwrapEmbeddingFunction")
}

func TestCloseOnceContentEF_UnwrapperFallbackDualEF(t *testing.T) {
	dualEF := &mockDualEF{}
	wrapped := wrapContentEFCloseOnce(dualEF)

	unwrapper, ok := wrapped.(embeddings.EmbeddingFunctionUnwrapper)
	require.True(t, ok)

	unwrapped := unwrapper.UnwrapEmbeddingFunction()
	assert.Same(t, dualEF, unwrapped, "should return inner when it implements EmbeddingFunction but not EmbeddingFunctionUnwrapper")
}

func TestCloseOnceContentEF_UnwrapperNilFallback(t *testing.T) {
	inner := &mockCloseableContentEF{}
	wrapped := wrapContentEFCloseOnce(inner)

	unwrapper, ok := wrapped.(embeddings.EmbeddingFunctionUnwrapper)
	require.True(t, ok)

	unwrapped := unwrapper.UnwrapEmbeddingFunction()
	assert.Nil(t, unwrapped, "should return nil when inner implements neither EmbeddingFunctionUnwrapper nor EmbeddingFunction")
}

func TestCloseOnceContentEF_ErrorPropagation(t *testing.T) {
	closeErr := fmt.Errorf("content close failed")
	inner := &mockFailingCloseContentEF{closeErr: closeErr}
	wrapped := wrapContentEFCloseOnce(inner)
	closer := wrapped.(io.Closer)

	err := closer.Close()
	assert.ErrorIs(t, err, closeErr, "first Close must propagate error")

	err = closer.Close()
	assert.ErrorIs(t, err, closeErr, "subsequent Close must return same error")
}

func TestForkCloseLifecycle_CollectionImpl_WithContentEF(t *testing.T) {
	innerEF := &mockCloseableEF{}
	innerContentEF := &mockCloseableContentEF{}

	parent := &CollectionImpl{
		embeddingFunction:        innerEF,
		contentEmbeddingFunction: innerContentEF,
		ownsEF:                   true,
	}

	forked := &CollectionImpl{
		embeddingFunction:        wrapEFCloseOnce(parent.embeddingFunction),
		contentEmbeddingFunction: wrapContentEFCloseOnce(parent.contentEmbeddingFunction),
		ownsEF:                   false,
	}

	err := forked.Close()
	require.NoError(t, err)
	assert.Equal(t, int32(0), innerEF.closeCount.Load(), "fork must not close parent's dense EF")
	assert.Equal(t, int32(0), innerContentEF.closeCount.Load(), "fork must not close parent's content EF")

	ctx := context.Background()
	_, err = innerEF.EmbedQuery(ctx, "test")
	assert.NoError(t, err, "parent's dense EF must remain usable after fork close")
	_, err = innerContentEF.EmbedContent(ctx, embeddings.Content{})
	assert.NoError(t, err, "parent's content EF must remain usable after fork close")

	err = parent.Close()
	require.NoError(t, err)
	assert.Equal(t, int32(1), innerEF.closeCount.Load(), "parent must close dense EF exactly once")
	assert.Equal(t, int32(1), innerContentEF.closeCount.Load(), "parent must close content EF exactly once")
}

func TestCollectionImpl_Close_DualError(t *testing.T) {
	efErr := fmt.Errorf("dense EF close failed")
	contentErr := fmt.Errorf("content EF close failed")

	owner := &CollectionImpl{
		embeddingFunction:        &mockFailingCloseEF{closeErr: efErr},
		contentEmbeddingFunction: &mockFailingCloseContentEF{closeErr: contentErr},
		ownsEF:                   true,
	}

	err := owner.Close()
	require.Error(t, err)
	assert.ErrorIs(t, err, efErr, "must include dense EF error")
	assert.ErrorIs(t, err, contentErr, "must include content EF error")
}

func TestCloseOnceEF_ConcurrentEmbedAndClose(t *testing.T) {
	inner := &mockCloseableEF{}
	wrapped := wrapEFCloseOnce(inner)
	closer := wrapped.(io.Closer)

	const goroutines = 50
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := wrapped.EmbedDocuments(context.Background(), []string{"test"})
			if err != nil {
				assert.ErrorIs(t, err, errEFClosed)
			}
		}()
	}

	_ = closer.Close()
	wg.Wait()

	_, err := wrapped.EmbedDocuments(context.Background(), []string{"test"})
	assert.ErrorIs(t, err, errEFClosed, "embed after close must return errEFClosed")
	assert.Equal(t, int32(1), inner.closeCount.Load(), "inner must be closed exactly once")
}

type mockPanickingCloseEF struct {
	mockCloseableEF
}

func (m *mockPanickingCloseEF) Close() error {
	panic("close exploded")
}

func TestCloseOnceEF_ClosePanicCaptured(t *testing.T) {
	inner := &mockPanickingCloseEF{}
	wrapped := wrapEFCloseOnce(inner)
	closer := wrapped.(io.Closer)

	err := closer.Close()
	require.Error(t, err, "panic during close must be captured as an error")
	assert.Contains(t, err.Error(), "panic during EF close")

	err2 := closer.Close()
	assert.Equal(t, err, err2, "subsequent Close must return the same captured error")
}

func TestCloseOnceContentEF_ClosePanicCaptured(t *testing.T) {
	inner := &mockCloseableContentEF{}
	wrapped := wrapContentEFCloseOnce(inner)

	// Replace inner's close with a panicking one via a wrapper.
	panicking := &mockPanickingCloseContentEF{mockCloseableContentEF: *inner}
	wrappedPanic := wrapContentEFCloseOnce(panicking)
	closer := wrappedPanic.(io.Closer)

	err := closer.Close()
	require.Error(t, err, "panic during close must be captured as an error")
	assert.Contains(t, err.Error(), "panic during EF close")

	// Original wrapped is independent — verify it still works.
	err2 := wrapped.(io.Closer).Close()
	assert.NoError(t, err2)
}

type mockPanickingCloseContentEF struct {
	mockCloseableContentEF
}

func (m *mockPanickingCloseContentEF) Close() error {
	panic("content close exploded")
}
