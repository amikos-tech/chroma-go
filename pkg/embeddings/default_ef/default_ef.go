package defaultef

import (
	"context"
	stderrors "errors"
	"sync"
	"sync/atomic"

	"github.com/amikos-tech/pure-onnx/embeddings/minilm"
	ort "github.com/amikos-tech/pure-onnx/ort"
	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

type Option func(p *DefaultEmbeddingFunction) error

var (
	_ embeddings.EmbeddingFunction = (*DefaultEmbeddingFunction)(nil)
	_ embeddings.Closeable         = (*DefaultEmbeddingFunction)(nil)
)

type DefaultEmbeddingFunction struct {
	embedder  *minilm.Embedder
	closed    int32
	closeOnce sync.Once
}

var initLock sync.RWMutex

var (
	// These function variables are test seams. Tests mutate them only while holding
	// defaultEFTestHooksMu from test_hooks_test.go to avoid cross-test races.
	ensureOnnxRuntimeSharedLibraryFn     = EnsureOnnxRuntimeSharedLibrary
	ensureDefaultEmbeddingModelFn        = EnsureDefaultEmbeddingFunctionModel
	initializeEnvironmentWithBootstrapFn = ort.InitializeEnvironmentWithBootstrap
	destroyEnvironmentFn                 = ort.DestroyEnvironment
)

func NewDefaultEmbeddingFunction(opts ...Option) (*DefaultEmbeddingFunction, func() error, error) {
	cfg := getConfig()

	if err := ensureOnnxRuntimeSharedLibraryFn(); err != nil {
		return nil, nil, errors.Wrap(err, "failed to ensure onnx runtime shared library")
	}
	if err := ensureDefaultEmbeddingModelFn(); err != nil {
		return nil, nil, errors.Wrap(err, "failed to ensure default embedding function model")
	}

	initLock.Lock()
	defer initLock.Unlock()

	bootstrapOpts := []ort.BootstrapOption{
		ort.WithBootstrapCacheDir(cfg.OnnxCacheDir),
	}
	if cfg.LibOnnxRuntimeVersion == "custom" {
		bootstrapOpts = append(bootstrapOpts, ort.WithBootstrapLibraryPath(cfg.OnnxLibPath))
	} else {
		bootstrapOpts = append(bootstrapOpts, ort.WithBootstrapVersion(cfg.LibOnnxRuntimeVersion))
	}
	if err := initializeEnvironmentWithBootstrapFn(bootstrapOpts...); err != nil {
		return nil, nil, errors.Wrap(err, "failed to initialize onnx runtime environment")
	}

	embedder, err := minilm.NewEmbedder(
		cfg.OnnxModelPath,
		cfg.OnnxModelTokenizerConfigPath,
		minilm.WithMeanPooling(),
		minilm.WithL2Normalization(),
	)
	if err != nil {
		embedderErr := errors.Wrap(err, "failed to create ONNX embedder")
		if cleanupErr := destroyEnvironmentFn(); cleanupErr != nil {
			return nil, nil, stderrors.Join(
				embedderErr,
				errors.Wrap(cleanupErr, "failed to destroy onnx runtime environment after embedder setup error"),
			)
		}
		return nil, nil, embedderErr
	}

	ef := &DefaultEmbeddingFunction{embedder: embedder}
	for _, opt := range opts {
		if err := opt(ef); err != nil {
			optionErr := errors.Wrap(err, "failed to apply default embedding function option")
			if closeErr := ef.Close(); closeErr != nil {
				return nil, nil, stderrors.Join(
					optionErr,
					errors.Wrap(closeErr, "failed to cleanup default embedding function after option error"),
				)
			}
			return nil, nil, optionErr
		}
	}

	return ef, ef.Close, nil
}

func (e *DefaultEmbeddingFunction) EmbedDocuments(_ context.Context, documents []string) ([]embeddings.Embedding, error) {
	if atomic.LoadInt32(&e.closed) == 1 {
		return nil, errors.New("embedding function is closed")
	}
	initLock.RLock()
	defer initLock.RUnlock()
	if atomic.LoadInt32(&e.closed) == 1 {
		return nil, errors.New("embedding function is closed")
	}

	if e.embedder == nil {
		return nil, errors.New("embedding function is not initialized")
	}

	vectors, err := e.embedder.EmbedDocuments(documents)
	if err != nil {
		return nil, errors.Wrap(err, "failed to embed documents")
	}
	ebmds, err := embeddings.NewEmbeddingsFromFloat32(vectors)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert embeddings")
	}
	if len(ebmds) == 0 {
		return embeddings.NewEmptyEmbeddings(), nil
	}
	if len(ebmds) != len(documents) {
		return nil, errors.Errorf("number of embeddings %d does not match number of documents %d", len(ebmds), len(documents))
	}
	return ebmds, nil
}

func (e *DefaultEmbeddingFunction) EmbedQuery(_ context.Context, document string) (embeddings.Embedding, error) {
	if atomic.LoadInt32(&e.closed) == 1 {
		return nil, errors.New("embedding function is closed")
	}
	initLock.RLock()
	defer initLock.RUnlock()
	if atomic.LoadInt32(&e.closed) == 1 {
		return nil, errors.New("embedding function is closed")
	}

	if e.embedder == nil {
		return nil, errors.New("embedding function is not initialized")
	}

	vector, err := e.embedder.EmbedQuery(document)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode query")
	}
	embds, err := embeddings.NewEmbeddingsFromFloat32([][]float32{vector})
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert query embedding")
	}
	if len(embds) == 0 {
		return embeddings.NewEmptyEmbedding(), nil
	}

	return embds[0], nil
}

func (e *DefaultEmbeddingFunction) Close() error {
	if atomic.LoadInt32(&e.closed) == 1 {
		return nil
	}
	initLock.Lock()
	defer initLock.Unlock()

	var closeErr error
	e.closeOnce.Do(func() {
		atomic.StoreInt32(&e.closed, 1)

		var errs []error

		if e.embedder != nil {
			if err := e.embedder.Close(); err != nil {
				errs = append(errs, errors.Wrap(err, "failed to close embedder"))
			}
			e.embedder = nil
		}

		if err := destroyEnvironmentFn(); err != nil {
			errs = append(errs, errors.Wrap(err, "failed to destroy onnx runtime environment"))
		}

		if len(errs) > 0 {
			closeErr = stderrors.Join(errs...)
		}
	})
	return closeErr
}

func (e *DefaultEmbeddingFunction) Name() string {
	return "default"
}

func (e *DefaultEmbeddingFunction) GetConfig() embeddings.EmbeddingFunctionConfig {
	return embeddings.EmbeddingFunctionConfig{}
}

func (e *DefaultEmbeddingFunction) DefaultSpace() embeddings.DistanceMetric {
	return embeddings.L2
}

func (e *DefaultEmbeddingFunction) SupportedSpaces() []embeddings.DistanceMetric {
	return []embeddings.DistanceMetric{embeddings.L2, embeddings.COSINE, embeddings.IP}
}

// NewDefaultEmbeddingFunctionFromConfig creates a default embedding function from a config map.
// The returned EmbeddingFunction implements Closeable; callers should type-assert
// and call Close() when done to release ONNX runtime and tokenizer resources.
func NewDefaultEmbeddingFunctionFromConfig(_ embeddings.EmbeddingFunctionConfig) (*DefaultEmbeddingFunction, error) {
	ef, _, err := NewDefaultEmbeddingFunction()
	return ef, err
}

func init() {
	factory := func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.EmbeddingFunction, error) {
		return NewDefaultEmbeddingFunctionFromConfig(cfg)
	}
	// Register as "default" to match Python client naming
	if err := embeddings.RegisterDense("default", factory); err != nil {
		panic(err)
	}
	// Register alias for backward compatibility with existing Go-created collections
	if err := embeddings.RegisterDense("onnx_mini_lm_l6_v2", factory); err != nil {
		panic(err)
	}
}
