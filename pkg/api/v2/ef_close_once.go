package v2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

var (
	errEFClosed       = errors.New("embedding function is closed; ensure Close() is not called while operations are in flight")
	errEFNotSupported = errors.New("content embedding function does not support this operation")
)

// Close-once wrappers are one layer in a three-layer defense against
// double-close bugs in forked collections:
//
//  1. ownsEF flag — fork's Close() skips EF cleanup entirely.
//  2. closeOnce wrapper — guards direct Close() calls on the fork's wrapped EF reference as defense-in-depth.
//  3. Collection-level sync.Once — makes the owner's Close() idempotent.

var (
	_ embeddings.EmbeddingFunction          = (*closeOnceEF)(nil)
	_ io.Closer                             = (*closeOnceEF)(nil)
	_ embeddings.EmbeddingFunctionUnwrapper = (*closeOnceEF)(nil)
)

var (
	_ embeddings.ContentEmbeddingFunction   = (*closeOnceContentEF)(nil)
	_ embeddings.EmbeddingFunction          = (*closeOnceContentEF)(nil)
	_ io.Closer                             = (*closeOnceContentEF)(nil)
	_ embeddings.EmbeddingFunctionUnwrapper = (*closeOnceContentEF)(nil)
)

type closeOnceState struct {
	once     sync.Once
	closed   atomic.Bool
	closeErr error
}

// Best-effort use-after-close guard: the check is not atomic with the subsequent
// delegate call (TOCTOU). Callers must not call Close() while operations are in flight.
func (s *closeOnceState) isClosed() bool {
	return s.closed.Load()
}

func (s *closeOnceState) doClose(fn func() error) error {
	s.once.Do(func() {
		s.closed.Store(true)
		defer func() {
			if r := recover(); r != nil {
				s.closeErr = reportClosePanic(r)
			}
		}()
		s.closeErr = fn()
	})
	return s.closeErr // safe: sync.Once guarantees happens-before for closeErr write
}

type closeOnceEF struct {
	ef embeddings.EmbeddingFunction
	closeOnceState
}

func (w *closeOnceEF) Close() error {
	return w.doClose(func() error {
		if closer, ok := w.ef.(io.Closer); ok {
			return closer.Close()
		}
		return nil
	})
}

func (w *closeOnceEF) EmbedDocuments(ctx context.Context, texts []string) ([]embeddings.Embedding, error) {
	if w.isClosed() {
		return nil, errEFClosed
	}
	return w.ef.EmbedDocuments(ctx, texts)
}

func (w *closeOnceEF) EmbedQuery(ctx context.Context, text string) (embeddings.Embedding, error) {
	if w.isClosed() {
		return nil, errEFClosed
	}
	return w.ef.EmbedQuery(ctx, text)
}

// Metadata methods delegate unconditionally — they read static configuration,
// not mutable resources, so they are safe to call after Close.

func (w *closeOnceEF) Name() string                                  { return w.ef.Name() }
func (w *closeOnceEF) GetConfig() embeddings.EmbeddingFunctionConfig { return w.ef.GetConfig() }
func (w *closeOnceEF) DefaultSpace() embeddings.DistanceMetric       { return w.ef.DefaultSpace() }
func (w *closeOnceEF) SupportedSpaces() []embeddings.DistanceMetric  { return w.ef.SupportedSpaces() }

func (w *closeOnceEF) UnwrapEmbeddingFunction() embeddings.EmbeddingFunction {
	if unwrapper, ok := w.ef.(embeddings.EmbeddingFunctionUnwrapper); ok {
		return unwrapper.UnwrapEmbeddingFunction()
	}
	return w.ef
}

type closeOnceContentEF struct {
	ef embeddings.ContentEmbeddingFunction
	closeOnceState
}

func (w *closeOnceContentEF) Close() error {
	return w.doClose(func() error {
		if closer, ok := w.ef.(io.Closer); ok {
			return closer.Close()
		}
		return nil
	})
}

func (w *closeOnceContentEF) EmbedContent(ctx context.Context, content embeddings.Content) (embeddings.Embedding, error) {
	if w.isClosed() {
		return nil, errEFClosed
	}
	return w.ef.EmbedContent(ctx, content)
}

func (w *closeOnceContentEF) EmbedContents(ctx context.Context, contents []embeddings.Content) ([]embeddings.Embedding, error) {
	if w.isClosed() {
		return nil, errEFClosed
	}
	return w.ef.EmbedContents(ctx, contents)
}

// EmbeddingFunction methods delegate to the inner type when it implements
// EmbeddingFunction (dual-interface types). This preserves decorator
// transparency so type assertions like ef.(EmbeddingFunction) succeed on the
// wrapper when they would succeed on the unwrapped type.

func (w *closeOnceContentEF) EmbedDocuments(ctx context.Context, texts []string) ([]embeddings.Embedding, error) {
	if w.isClosed() {
		return nil, errEFClosed
	}
	if ef, ok := w.ef.(embeddings.EmbeddingFunction); ok {
		return ef.EmbedDocuments(ctx, texts)
	}
	return nil, fmt.Errorf("EmbedDocuments: %w", errEFNotSupported)
}

func (w *closeOnceContentEF) EmbedQuery(ctx context.Context, text string) (embeddings.Embedding, error) {
	if w.isClosed() {
		return nil, errEFClosed
	}
	if ef, ok := w.ef.(embeddings.EmbeddingFunction); ok {
		return ef.EmbedQuery(ctx, text)
	}
	return nil, fmt.Errorf("EmbedQuery: %w", errEFNotSupported)
}

func (w *closeOnceContentEF) Name() string {
	if ef, ok := w.ef.(embeddings.EmbeddingFunction); ok {
		return ef.Name()
	}
	return ""
}

func (w *closeOnceContentEF) GetConfig() embeddings.EmbeddingFunctionConfig {
	if ef, ok := w.ef.(embeddings.EmbeddingFunction); ok {
		return ef.GetConfig()
	}
	return embeddings.EmbeddingFunctionConfig{}
}

func (w *closeOnceContentEF) DefaultSpace() embeddings.DistanceMetric {
	if ef, ok := w.ef.(embeddings.EmbeddingFunction); ok {
		return ef.DefaultSpace()
	}
	return ""
}

func (w *closeOnceContentEF) SupportedSpaces() []embeddings.DistanceMetric {
	if ef, ok := w.ef.(embeddings.EmbeddingFunction); ok {
		return ef.SupportedSpaces()
	}
	return nil
}

func (w *closeOnceContentEF) UnwrapEmbeddingFunction() embeddings.EmbeddingFunction {
	if unwrapper, ok := w.ef.(embeddings.EmbeddingFunctionUnwrapper); ok {
		return unwrapper.UnwrapEmbeddingFunction()
	}
	if ef, ok := w.ef.(embeddings.EmbeddingFunction); ok {
		return ef
	}
	return nil
}

func wrapEFCloseOnce(ef embeddings.EmbeddingFunction) embeddings.EmbeddingFunction {
	if ef == nil {
		return nil
	}
	if _, ok := ef.(*closeOnceEF); ok {
		return ef
	}
	if _, ok := ef.(*closeOnceContentEF); ok {
		return ef
	}
	return &closeOnceEF{ef: ef}
}

func wrapContentEFCloseOnce(ef embeddings.ContentEmbeddingFunction) embeddings.ContentEmbeddingFunction {
	if ef == nil {
		return nil
	}
	if _, ok := ef.(*closeOnceContentEF); ok {
		return ef
	}
	return &closeOnceContentEF{ef: ef}
}
