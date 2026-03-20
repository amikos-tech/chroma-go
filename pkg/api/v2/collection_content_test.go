package v2

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// mockDualEmbeddingFunction implements both ContentEmbeddingFunction and EmbeddingFunction.
type mockDualEmbeddingFunction struct {
	name   string
	config embeddings.EmbeddingFunctionConfig
}

func (m *mockDualEmbeddingFunction) EmbedContent(
	_ context.Context, _ embeddings.Content,
) (embeddings.Embedding, error) {
	return nil, nil
}

func (m *mockDualEmbeddingFunction) EmbedContents(
	_ context.Context, _ []embeddings.Content,
) ([]embeddings.Embedding, error) {
	return nil, nil
}

func (m *mockDualEmbeddingFunction) EmbedDocuments(
	_ context.Context, _ []string,
) ([]embeddings.Embedding, error) {
	return nil, nil
}

func (m *mockDualEmbeddingFunction) EmbedQuery(
	_ context.Context, _ string,
) (embeddings.Embedding, error) {
	return nil, nil
}

func (m *mockDualEmbeddingFunction) Name() string { return m.name }

func (m *mockDualEmbeddingFunction) GetConfig() embeddings.EmbeddingFunctionConfig {
	return m.config
}

func (m *mockDualEmbeddingFunction) DefaultSpace() embeddings.DistanceMetric {
	return embeddings.COSINE
}

func (m *mockDualEmbeddingFunction) SupportedSpaces() []embeddings.DistanceMetric {
	return []embeddings.DistanceMetric{embeddings.COSINE}
}

// mockContentOnlyEF implements only ContentEmbeddingFunction, not EmbeddingFunction.
type mockContentOnlyEF struct{}

func (m *mockContentOnlyEF) EmbedContent(
	_ context.Context, _ embeddings.Content,
) (embeddings.Embedding, error) {
	return nil, nil
}

func (m *mockContentOnlyEF) EmbedContents(
	_ context.Context, _ []embeddings.Content,
) ([]embeddings.Embedding, error) {
	return nil, nil
}

// deriveEFFromContent applies the same logic as client_http.go:
// if contentEF is non-nil and ef is nil, derive ef from contentEF when possible.
func deriveEFFromContent(
	ef embeddings.EmbeddingFunction, contentEF embeddings.ContentEmbeddingFunction,
) embeddings.EmbeddingFunction {
	if contentEF != nil && ef == nil {
		if denseFromContent, ok := contentEF.(embeddings.EmbeddingFunction); ok {
			return denseFromContent
		}
	}
	return ef
}

func TestAutoWiring_ContentEFPopulated(t *testing.T) {
	config := NewCollectionConfiguration()
	config.SetEmbeddingFunctionInfo(&EmbeddingFunctionInfo{
		Type:   "known",
		Name:   "consistent_hash",
		Config: map[string]interface{}{"dim": float64(128)},
	})
	contentEF, err := BuildContentEFFromConfig(config)
	require.NoError(t, err)
	require.NotNil(t, contentEF, "auto-wiring should populate content EF for known dense provider")
}

func TestAutoWiring_ContentEFNilForUnknown(t *testing.T) {
	config := NewCollectionConfiguration()
	config.SetEmbeddingFunctionInfo(&EmbeddingFunctionInfo{
		Type: "known",
		Name: "nonexistent_provider_xyz",
	})
	contentEF, err := BuildContentEFFromConfig(config)
	require.NoError(t, err)
	assert.Nil(t, contentEF, "auto-wiring should return nil for unknown provider without error")
}

func TestAutoWiring_DenseEFDerivedFromContentEF(t *testing.T) {
	dualEF := &mockDualEmbeddingFunction{name: "dual_test", config: embeddings.EmbeddingFunctionConfig{}}

	// Verify that type assertion from ContentEmbeddingFunction to EmbeddingFunction succeeds
	// for types that implement both interfaces.
	var contentEF embeddings.ContentEmbeddingFunction = dualEF
	ef := deriveEFFromContent(nil, contentEF)

	require.NotNil(t, ef, "dense EF should be derived from content EF when content implements EmbeddingFunction")
	assert.Equal(t, "dual_test", ef.Name())
}

func TestWithContentEmbeddingFunction_ExplicitOverride(t *testing.T) {
	explicitEF := &mockContentOnlyEF{}
	opt := WithContentEmbeddingFunctionGet(explicitEF)

	op := &GetCollectionOp{}
	err := opt(op)
	require.NoError(t, err)
	assert.Equal(t, explicitEF, op.contentEmbeddingFunction,
		"explicit option should set contentEmbeddingFunction on GetCollectionOp")
}

func TestWithContentEmbeddingFunction_NilReturnsError(t *testing.T) {
	opt := WithContentEmbeddingFunctionGet(nil)

	op := &GetCollectionOp{}
	err := opt(op)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestAutoWiring_ExplicitDenseEFNotOverridden(t *testing.T) {
	explicitDenseEF := &mockEmbeddingFunction{name: "explicit_dense"}
	dualContentEF := &mockDualEmbeddingFunction{name: "content_derived"}

	// When ef is already set, deriveEFFromContent should leave it unchanged.
	var contentEF embeddings.ContentEmbeddingFunction = dualContentEF
	ef := deriveEFFromContent(explicitDenseEF, contentEF)

	assert.Equal(t, "explicit_dense", ef.Name(),
		"explicit dense EF should not be overridden by content EF derive logic")
}
