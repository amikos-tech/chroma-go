//go:build basicv2

package v2

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/amikos-tech/chroma-go/pkg/logger"
)

type mockSharedContentAdapter struct {
	inner      *mockCloseableEF
	closeCount int
}

var _ embeddings.ContentEmbeddingFunction = (*mockSharedContentAdapter)(nil)
var _ embeddings.EmbeddingFunctionUnwrapper = (*mockSharedContentAdapter)(nil)
var _ io.Closer = (*mockSharedContentAdapter)(nil)

func (m *mockSharedContentAdapter) EmbedContent(_ context.Context, _ embeddings.Content) (embeddings.Embedding, error) {
	return embeddings.NewEmbeddingFromFloat32([]float32{1, 2, 3}), nil
}

func (m *mockSharedContentAdapter) EmbedContents(_ context.Context, _ []embeddings.Content) ([]embeddings.Embedding, error) {
	return []embeddings.Embedding{embeddings.NewEmbeddingFromFloat32([]float32{1, 2, 3})}, nil
}

func (m *mockSharedContentAdapter) Close() error {
	m.closeCount++
	return m.inner.Close()
}

func (m *mockSharedContentAdapter) UnwrapEmbeddingFunction() embeddings.EmbeddingFunction {
	return m.inner
}

type capturingLogger struct {
	warnCount  int
	errorCount int
	lastMsg    string
}

var _ logger.Logger = (*capturingLogger)(nil)

func (l *capturingLogger) Debug(string, ...logger.Field) {}
func (l *capturingLogger) Info(string, ...logger.Field)  {}

func (l *capturingLogger) Warn(msg string, _ ...logger.Field) {
	l.warnCount++
	l.lastMsg = msg
}

func (l *capturingLogger) Error(msg string, _ ...logger.Field) {
	l.errorCount++
	l.lastMsg = msg
}

func (l *capturingLogger) DebugWithContext(context.Context, string, ...logger.Field) {}
func (l *capturingLogger) InfoWithContext(context.Context, string, ...logger.Field)  {}
func (l *capturingLogger) WarnWithContext(context.Context, string, ...logger.Field)  {}
func (l *capturingLogger) ErrorWithContext(context.Context, string, ...logger.Field) {}

func (l *capturingLogger) With(...logger.Field) logger.Logger {
	return l
}

func (l *capturingLogger) IsDebugEnabled() bool { return false }
func (l *capturingLogger) Sync() error          { return nil }

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	outputCh := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outputCh <- buf.String()
	}()

	defer func() {
		os.Stderr = oldStderr
	}()

	fn()

	require.NoError(t, w.Close())
	output := <-outputCh
	require.NoError(t, r.Close())
	return output
}

func TestCloseOnceEF_ClosePanicLogsToStderr(t *testing.T) {
	inner := &mockPanickingCloseEF{}
	wrapped := wrapEFCloseOnce(inner)

	output := captureStderr(t, func() {
		err := wrapped.(io.Closer).Close()
		require.Error(t, err)
	})

	assert.Contains(t, output, "panic during EF close")
	assert.Contains(t, output, "close exploded")
}

func TestEmbeddedCollection_ClosePanicCaptured(t *testing.T) {
	owner := &embeddedCollection{
		embeddingFunction: &mockPanickingCloseEF{},
	}
	owner.ownsEF.Store(true)

	output := captureStderr(t, func() {
		err := owner.Close()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "panic during EF close")
		assert.Contains(t, err.Error(), "stack:")
	})

	assert.Contains(t, output, "panic during EF close")
}

func TestCollectionImpl_Close_SkipsSharedDenseCloseViaUnwrapper(t *testing.T) {
	inner := &mockCloseableEF{}
	contentAdapter := &mockSharedContentAdapter{inner: inner}
	owner := &CollectionImpl{
		embeddingFunction:        inner,
		contentEmbeddingFunction: contentAdapter,
	}
	owner.ownsEF.Store(true)

	err := owner.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, contentAdapter.closeCount, "content adapter must be closed exactly once")
	assert.Equal(t, int32(1), inner.closeCount.Load(), "shared dense EF must not be double-closed")
}

func TestCollectionImpl_Close_SkipsSharedDenseCloseViaDualInterfaceEF(t *testing.T) {
	dual := &mockDualEF{}
	owner := &CollectionImpl{
		embeddingFunction:        dual,
		contentEmbeddingFunction: dual,
	}
	owner.ownsEF.Store(true)

	err := owner.Close()
	require.NoError(t, err)
	assert.Equal(t, int32(1), dual.closeCount.Load(), "shared dual-interface EF must only be closed once")
}

func TestDeleteCollectionFromCache_TransfersOwnershipWithContentOnlySharedEF(t *testing.T) {
	sharedContent := &mockCloseableContentEF{}
	parent := &CollectionImpl{
		name:                     "parent",
		contentEmbeddingFunction: sharedContent,
	}
	parent.ownsEF.Store(true)
	fork := &CollectionImpl{
		name:                     "fork",
		contentEmbeddingFunction: wrapContentEFCloseOnce(parent.contentEmbeddingFunction),
	}

	client := &APIClientV2{
		collectionCache: map[string]Collection{
			"parent": parent,
			"fork":   fork,
		},
	}

	client.localDeleteCollectionFromCache("parent")

	assert.True(t, fork.ownsEF.Load(), "fork must receive ownership when content EF is the shared resource")
	assert.Equal(t, int32(0), sharedContent.closeCount.Load(), "shared content EF must stay open on ownership transfer")

	err := fork.Close()
	require.NoError(t, err)
	assert.Equal(t, int32(1), sharedContent.closeCount.Load(), "new owner must close shared content EF exactly once")
}

func TestDeleteCollectionFromCache_CloseErrorLogsAtErrorLevel(t *testing.T) {
	log := &capturingLogger{}
	parent := &CollectionImpl{
		name:              "parent",
		embeddingFunction: &mockFailingCloseEF{closeErr: assert.AnError},
	}
	parent.ownsEF.Store(true)

	client := &APIClientV2{
		BaseAPIClient: BaseAPIClient{logger: log},
		collectionCache: map[string]Collection{
			"parent": parent,
		},
	}

	client.localDeleteCollectionFromCache("parent")

	assert.Equal(t, 1, log.errorCount)
	assert.Zero(t, log.warnCount)
	assert.Equal(t, "failed to close EF during collection cache cleanup", log.lastMsg)
}

func TestDeleteCollectionFromCache_CloseErrorFallsBackToStderr(t *testing.T) {
	parent := &CollectionImpl{
		name:              "parent",
		embeddingFunction: &mockFailingCloseEF{closeErr: assert.AnError},
	}
	parent.ownsEF.Store(true)

	client := &APIClientV2{
		collectionCache: map[string]Collection{
			"parent": parent,
		},
	}

	output := captureStderr(t, func() {
		client.localDeleteCollectionFromCache("parent")
	})

	assert.Contains(t, output, "failed to close EF during collection cache cleanup")
	assert.Contains(t, output, "collection=parent")
	assert.Contains(t, output, assert.AnError.Error())
}

func TestCloseOnceContentEF_ConcurrentContentEmbedAndClose(t *testing.T) {
	inner := &mockCloseableContentEF{}
	wrapped := wrapContentEFCloseOnce(inner)
	closer := wrapped.(io.Closer)

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(index int) {
			defer wg.Done()
			if index%2 == 0 {
				_, err := wrapped.EmbedContent(context.Background(), embeddings.Content{})
				if err != nil {
					assert.ErrorIs(t, err, errEFClosed)
				}
				return
			}
			_, err := wrapped.EmbedContents(context.Background(), []embeddings.Content{{}})
			if err != nil {
				assert.ErrorIs(t, err, errEFClosed)
			}
		}(i)
	}

	_ = closer.Close()
	wg.Wait()

	_, err := wrapped.EmbedContent(context.Background(), embeddings.Content{})
	assert.ErrorIs(t, err, errEFClosed)
	assert.Equal(t, int32(1), inner.closeCount.Load(), "inner content EF must be closed exactly once")
}

func TestWrapEFCloseOnce_RejectsCrossTypeDoubleWrap(t *testing.T) {
	dual := &mockDualEF{}
	contentWrapped := wrapContentEFCloseOnce(dual)

	ef, ok := contentWrapped.(embeddings.EmbeddingFunction)
	require.True(t, ok)

	assert.Same(t, contentWrapped, wrapEFCloseOnce(ef))
}

func TestUnwrapCloseOnceEF_HandlesCloseOnceContentEF(t *testing.T) {
	dual := &mockDualEF{}
	contentWrapped := wrapContentEFCloseOnce(dual)
	ef := contentWrapped.(embeddings.EmbeddingFunction)

	unwrapped := unwrapCloseOnceEF(ef)
	assert.Same(t, dual, unwrapped, "unwrapCloseOnceEF must unwrap *closeOnceContentEF to its inner EmbeddingFunction")
}

func TestCollectionImpl_Close_SharedDetectionWithWrappedEFs(t *testing.T) {
	innerEF := &mockCloseableEF{}
	contentAdapter := &mockSharedContentAdapter{inner: innerEF}

	// Simulate fork with ownership transfer: both EFs are wrapped
	fork := &CollectionImpl{
		embeddingFunction:        wrapEFCloseOnce(innerEF),
		contentEmbeddingFunction: wrapContentEFCloseOnce(contentAdapter),
	}
	fork.ownsEF.Store(true)

	err := fork.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, contentAdapter.closeCount, "content adapter must be closed exactly once")
	assert.Equal(t, int32(1), innerEF.closeCount.Load(), "shared dense EF must not be double-closed via both wrappers")
}

func TestCollectionImpl_Close_ContentPanicStillClosesDenseEF(t *testing.T) {
	denseEF := &mockCloseableEF{}
	owner := &CollectionImpl{
		embeddingFunction:        denseEF,
		contentEmbeddingFunction: &mockPanickingCloseContentEF{},
	}
	owner.ownsEF.Store(true)

	output := captureStderr(t, func() {
		err := owner.Close()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "panic during EF close")
	})

	assert.Contains(t, output, "content close exploded", "panic must be reported to stderr")
	assert.Equal(t, int32(1), denseEF.closeCount.Load(), "dense EF must still be closed after content EF panic")
}

func TestEmbeddedCollection_Close_SkipsSharedDenseCloseViaUnwrapper(t *testing.T) {
	denseEF := &mockCloseableEF{}
	contentAdapter := &mockSharedContentAdapter{inner: denseEF}
	owner := &embeddedCollection{
		embeddingFunction:        denseEF,
		contentEmbeddingFunction: contentAdapter,
	}
	owner.ownsEF.Store(true)

	err := owner.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, contentAdapter.closeCount, "content adapter must be closed exactly once")
	assert.Equal(t, int32(1), denseEF.closeCount.Load(), "shared dense EF must not be double-closed")
}

func TestEmbeddedCollection_Close_SkipsSharedDenseCloseViaDualInterfaceEF(t *testing.T) {
	dual := &mockDualEF{}
	owner := &embeddedCollection{
		embeddingFunction:        dual,
		contentEmbeddingFunction: dual,
	}
	owner.ownsEF.Store(true)

	err := owner.Close()
	require.NoError(t, err)
	assert.Equal(t, int32(1), dual.closeCount.Load(), "shared dual-interface EF must only be closed once")
}

func TestEmbeddedCollection_Close_DualError(t *testing.T) {
	denseErr := io.ErrClosedPipe
	contentErr := assert.AnError
	owner := &embeddedCollection{
		embeddingFunction:        &mockFailingCloseEF{closeErr: denseErr},
		contentEmbeddingFunction: &mockFailingCloseContentEF{closeErr: contentErr},
	}
	owner.ownsEF.Store(true)

	err := owner.Close()
	require.Error(t, err)
	assert.ErrorIs(t, err, contentErr)
	assert.ErrorIs(t, err, denseErr)
}

func TestEmbeddedCollection_Close_ClosesIndependentContentAndDenseEFs(t *testing.T) {
	denseEF := &mockCloseableEF{}
	contentEF := &mockCloseableContentEF{}
	owner := &embeddedCollection{
		embeddingFunction:        denseEF,
		contentEmbeddingFunction: contentEF,
	}
	owner.ownsEF.Store(true)

	err := owner.Close()
	require.NoError(t, err)
	assert.Equal(t, int32(1), contentEF.closeCount.Load(), "independent content EF must be closed")
	assert.Equal(t, int32(1), denseEF.closeCount.Load(), "independent dense EF must be closed")
}

func TestEmbeddedCollection_Close_NonCloseableContentEFStillClosesDenseEF(t *testing.T) {
	denseEF := &mockCloseableEF{}
	nonCloseable := &mockNonCloseableContentEF{}
	owner := &embeddedCollection{
		embeddingFunction:        denseEF,
		contentEmbeddingFunction: nonCloseable,
	}
	owner.ownsEF.Store(true)

	err := owner.Close()
	require.NoError(t, err)
	assert.Equal(t, int32(1), denseEF.closeCount.Load(), "dense EF must still be closed when content EF is not closeable")
}

func TestEmbeddedCollection_Close_ContentPanicStillClosesDenseEF(t *testing.T) {
	denseEF := &mockCloseableEF{}
	owner := &embeddedCollection{
		embeddingFunction:        denseEF,
		contentEmbeddingFunction: &mockPanickingCloseContentEF{},
	}
	owner.ownsEF.Store(true)

	output := captureStderr(t, func() {
		err := owner.Close()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "panic during EF close")
	})

	assert.Contains(t, output, "content close exploded", "panic must be reported to stderr")
	assert.Equal(t, int32(1), denseEF.closeCount.Load(), "dense EF must still be closed after content EF panic")
}

func TestIsDenseEFSharedWithContent_SymmetricUnwrap(t *testing.T) {
	base := &mockCloseableEF{}
	adapter := &mockSharedContentAdapter{inner: base}

	// Both wrapped in close-once
	wrappedDense := wrapEFCloseOnce(base)
	wrappedContent := wrapContentEFCloseOnce(adapter)

	assert.True(t, isDenseEFSharedWithContent(wrappedDense, wrappedContent),
		"wrapped shared EFs must be detected as shared via symmetric unwrapping")

	// Different base EFs wrapped
	otherBase := &mockCloseableEF{}
	wrappedOtherDense := wrapEFCloseOnce(otherBase)
	assert.False(t, isDenseEFSharedWithContent(wrappedOtherDense, wrappedContent),
		"different underlying EFs must not be detected as shared")

	// Unwrapped EFs (backward compatibility)
	assert.True(t, isDenseEFSharedWithContent(base, adapter),
		"unwrapped shared EFs must still be detected as shared")

	otherBase2 := &mockCloseableEF{}
	assert.False(t, isDenseEFSharedWithContent(otherBase2, adapter),
		"unwrapped different EFs must not be detected as shared")
}

func TestDeleteCollectionFromCache_EmbeddedCollection(t *testing.T) {
	mockEF := &mockCloseableEF{}
	mockContentEF := &mockCloseableContentEF{}

	ec := &embeddedCollection{
		name:                     "embedded-test",
		embeddingFunction:        wrapEFCloseOnce(mockEF),
		contentEmbeddingFunction: wrapContentEFCloseOnce(mockContentEF),
	}
	ec.ownsEF.Store(true)

	client := &APIClientV2{
		collectionCache: map[string]Collection{
			"embedded-test": ec,
		},
	}

	client.localDeleteCollectionFromCache("embedded-test")

	require.Equal(t, int32(1), mockEF.closeCount.Load(), "dense EF must be closed")
	require.Equal(t, int32(1), mockContentEF.closeCount.Load(), "content EF must be closed")
	require.Nil(t, client.collectionCache["embedded-test"], "cache entry must be removed")
}
