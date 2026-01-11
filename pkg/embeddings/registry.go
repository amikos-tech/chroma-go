package embeddings

import (
	"sync"

	"github.com/pkg/errors"
)

// EmbeddingFunctionFactory creates an EmbeddingFunction from config.
type EmbeddingFunctionFactory func(config EmbeddingFunctionConfig) (EmbeddingFunction, error)

// SparseEmbeddingFunctionFactory creates a SparseEmbeddingFunction from config.
type SparseEmbeddingFunctionFactory func(config EmbeddingFunctionConfig) (SparseEmbeddingFunction, error)

var (
	denseFactories  = make(map[string]EmbeddingFunctionFactory)
	sparseFactories = make(map[string]SparseEmbeddingFunctionFactory)
	mu              sync.RWMutex
)

// RegisterDense registers a dense embedding function factory by name.
func RegisterDense(name string, factory EmbeddingFunctionFactory) {
	mu.Lock()
	defer mu.Unlock()
	denseFactories[name] = factory
}

// RegisterSparse registers a sparse embedding function factory by name.
func RegisterSparse(name string, factory SparseEmbeddingFunctionFactory) {
	mu.Lock()
	defer mu.Unlock()
	sparseFactories[name] = factory
}

// BuildDense creates a dense EmbeddingFunction from name and config.
func BuildDense(name string, config EmbeddingFunctionConfig) (EmbeddingFunction, error) {
	mu.RLock()
	factory, ok := denseFactories[name]
	mu.RUnlock()
	if !ok {
		return nil, errors.Errorf("unknown embedding function: %s", name)
	}
	return factory(config)
}

// BuildSparse creates a SparseEmbeddingFunction from name and config.
func BuildSparse(name string, config EmbeddingFunctionConfig) (SparseEmbeddingFunction, error) {
	mu.RLock()
	factory, ok := sparseFactories[name]
	mu.RUnlock()
	if !ok {
		return nil, errors.Errorf("unknown sparse embedding function: %s", name)
	}
	return factory(config)
}

// ListDense returns all registered dense embedding function names.
func ListDense() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(denseFactories))
	for name := range denseFactories {
		names = append(names, name)
	}
	return names
}

// ListSparse returns all registered sparse embedding function names.
func ListSparse() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(sparseFactories))
	for name := range sparseFactories {
		names = append(names, name)
	}
	return names
}

// HasDense checks if a dense embedding function is registered.
func HasDense(name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := denseFactories[name]
	return ok
}

// HasSparse checks if a sparse embedding function is registered.
func HasSparse(name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := sparseFactories[name]
	return ok
}
