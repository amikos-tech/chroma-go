//go:build ef

package chromacloudsplade

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmbedTruncatesParsedErrorField(t *testing.T) {
	t.Parallel()

	const tailMarker = "splade-tail-marker"
	longError := strings.Repeat("sparse failure detail ", 80) + tailMarker
	responseBody := fmt.Sprintf(`{"error":"%s"}`, longError)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer server.Close()

	client, err := NewClient(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithInsecure(),
	)
	require.NoError(t, err)

	_, err = client.embed(context.Background(), []string{"document"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "API error [status 200]")
	require.Contains(t, err.Error(), "sparse failure detail sparse failure detail")
	require.Contains(t, err.Error(), "[truncated]")
	require.NotContains(t, err.Error(), tailMarker)
}
