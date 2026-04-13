package cohere

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCohereEmbeddingFunction_UsesLegacyDefaultModel(t *testing.T) {
	t.Parallel()

	ef, err := NewCohereEmbeddingFunction(WithAPIKey("test-key"))
	require.NoError(t, err)
	require.Equal(t, ModelEmbedEnglishV20, ef.DefaultModel)
	require.Equal(t, string(ModelEmbedEnglishV20), ef.GetConfig()["model_name"])
}

func TestCreateEmbeddingSanitizesErrorBody(t *testing.T) {
	t.Parallel()

	longBody := strings.Repeat("cohere-error-", 80)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(longBody))
	}))
	defer server.Close()

	ef, err := NewCohereEmbeddingFunction(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
		WithInsecure(),
	)
	require.NoError(t, err)

	_, err = ef.EmbedQuery(context.Background(), "hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "[truncated]")
	assert.NotContains(t, err.Error(), longBody)
}
