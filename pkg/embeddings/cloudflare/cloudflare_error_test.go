//go:build ef

package cloudflare

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateEmbeddingPreservesStructuredErrorsWhileSanitizingRawTail(t *testing.T) {
	t.Parallel()

	longTail := strings.Repeat("0123456789", 80)
	responseBody := fmt.Sprintf(
		`{"success":false,"messages":[],"errors":[{"code":"bad_request","message":"structured provider error"}],"raw":"%s"}`,
		longTail,
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer server.Close()

	client, err := NewCloudflareClient(
		WithAPIToken("test-token"),
		WithGatewayEndpoint(server.URL),
		WithHTTPClient(server.Client()),
		WithDefaultModel("test-model"),
		WithInsecure(),
	)
	require.NoError(t, err)

	_, err = client.CreateEmbedding(context.Background(), &CreateEmbeddingRequest{
		Text: []string{"test document"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), `"code":"bad_request"`)
	require.Contains(t, err.Error(), `"message":"structured provider error"`)
	require.Contains(t, err.Error(), `"raw":"0123456789`)
	require.Contains(t, err.Error(), "[truncated]")
	require.NotContains(t, err.Error(), responseBody)
	require.NotContains(t, err.Error(), longTail)
}

func TestCreateEmbeddingSanitizesStructuredErrorMessages(t *testing.T) {
	t.Parallel()

	longMessage := strings.Repeat("structured provider error ", 40)
	responseBody := fmt.Sprintf(
		`{"success":false,"messages":[],"errors":[{"code":"bad_request","message":"%s"}]}`,
		longMessage,
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer server.Close()

	client, err := NewCloudflareClient(
		WithAPIToken("test-token"),
		WithGatewayEndpoint(server.URL),
		WithHTTPClient(server.Client()),
		WithDefaultModel("test-model"),
		WithInsecure(),
	)
	require.NoError(t, err)

	_, err = client.CreateEmbedding(context.Background(), &CreateEmbeddingRequest{
		Text: []string{"test document"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "structured provider error")
	require.Contains(t, err.Error(), "[truncated]")
	require.NotContains(t, err.Error(), longMessage)
	require.NotContains(t, err.Error(), responseBody)
}

func TestCreateEmbeddingSanitizesNonJSONErrorBody(t *testing.T) {
	t.Parallel()

	longBody := strings.Repeat("plain-text-error-", 80)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(longBody))
	}))
	defer server.Close()

	client, err := NewCloudflareClient(
		WithAPIToken("test-token"),
		WithGatewayEndpoint(server.URL),
		WithHTTPClient(server.Client()),
		WithDefaultModel("test-model"),
		WithInsecure(),
	)
	require.NoError(t, err)

	_, err = client.CreateEmbedding(context.Background(), &CreateEmbeddingRequest{
		Text: []string{"test document"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected code [502 Bad Gateway]")
	require.Contains(t, err.Error(), "plain-text-error-plain-text-error-")
	require.Contains(t, err.Error(), "[truncated]")
	require.NotContains(t, err.Error(), longBody)
	require.NotContains(t, err.Error(), "failed to unmarshal response body")
}
