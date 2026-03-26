package v2

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

var errEFClosed = errors.New("embedding function is closed")

var (
	_ embeddings.EmbeddingFunction          = (*closeOnceEF)(nil)
	_ io.Closer                             = (*closeOnceEF)(nil)
	_ embeddings.EmbeddingFunctionUnwrapper = (*closeOnceEF)(nil)
)

var (
	_ embeddings.ContentEmbeddingFunction   = (*closeOnceContentEF)(nil)
	_ io.Closer                             = (*closeOnceContentEF)(nil)
	_ embeddings.EmbeddingFunctionUnwrapper = (*closeOnceContentEF)(nil)
)

type closeOnceEF struct {
	ef       embeddings.EmbeddingFunction
	once     sync.Once
	closed   atomic.Bool
	closeErr error
}

func (w *closeOnceEF) Close() error {
	w.once.Do(func() {
		w.closed.Store(true)
		if closer, ok := w.ef.(io.Closer); ok {
			w.closeErr = closer.Close()
		}
	})
	return w.closeErr
}

func (w *closeOnceEF) EmbedDocuments(ctx context.Context, texts []string) ([]embeddings.Embedding, error) {
	if w.closed.Load() {
		return nil, errEFClosed
	}
	return w.ef.EmbedDocuments(ctx, texts)
}

func (w *closeOnceEF) EmbedQuery(ctx context.Context, text string) (embeddings.Embedding, error) {
	if w.closed.Load() {
		return nil, errEFClosed
	}
	return w.ef.EmbedQuery(ctx, text)
}

func (w *closeOnceEF) Name() string {
	return w.ef.Name()
}

func (w *closeOnceEF) GetConfig() embeddings.EmbeddingFunctionConfig {
	return w.ef.GetConfig()
}

func (w *closeOnceEF) DefaultSpace() embeddings.DistanceMetric {
	return w.ef.DefaultSpace()
}

func (w *closeOnceEF) SupportedSpaces() []embeddings.DistanceMetric {
	return w.ef.SupportedSpaces()
}

func (w *closeOnceEF) UnwrapEmbeddingFunction() embeddings.EmbeddingFunction {
	if unwrapper, ok := w.ef.(embeddings.EmbeddingFunctionUnwrapper); ok {
		return unwrapper.UnwrapEmbeddingFunction()
	}
	return w.ef
}

type closeOnceContentEF struct {
	ef       embeddings.ContentEmbeddingFunction
	once     sync.Once
	closed   atomic.Bool
	closeErr error
}

func (w *closeOnceContentEF) Close() error {
	w.once.Do(func() {
		w.closed.Store(true)
		if closer, ok := w.ef.(io.Closer); ok {
			w.closeErr = closer.Close()
		}
	})
	return w.closeErr
}

func (w *closeOnceContentEF) EmbedContent(ctx context.Context, content embeddings.Content) (embeddings.Embedding, error) {
	if w.closed.Load() {
		return nil, errEFClosed
	}
	return w.ef.EmbedContent(ctx, content)
}

func (w *closeOnceContentEF) EmbedContents(ctx context.Context, contents []embeddings.Content) ([]embeddings.Embedding, error) {
	if w.closed.Load() {
		return nil, errEFClosed
	}
	return w.ef.EmbedContents(ctx, contents)
}

func (w *closeOnceContentEF) UnwrapEmbeddingFunction() embeddings.EmbeddingFunction {
	if unwrapper, ok := w.ef.(embeddings.EmbeddingFunctionUnwrapper); ok {
		return unwrapper.UnwrapEmbeddingFunction()
	}
	return nil
}

func wrapEFCloseOnce(ef embeddings.EmbeddingFunction) embeddings.EmbeddingFunction {
	if ef == nil {
		return nil
	}
	return &closeOnceEF{ef: ef}
}

func wrapContentEFCloseOnce(ef embeddings.ContentEmbeddingFunction) embeddings.ContentEmbeddingFunction {
	if ef == nil {
		return nil
	}
	return &closeOnceContentEF{ef: ef}
}
