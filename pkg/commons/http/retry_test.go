package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoWithRetryReturnsReadableResponseWhenRetriesExhausted(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"rate limited"}`))
	}))
	defer server.Close()

	strategy, err := NewSimpleRetryStrategy(
		WithMaxRetries(2),
		WithFixedDelay(1),
		WithRetryableStatusCodes(http.StatusTooManyRequests),
	)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, strings.NewReader(`{"q":"hello"}`))
	require.NoError(t, err)

	resp, err := strategy.DoWithRetry(server.Client(), req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, int32(2), requests.Load())
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	body, readErr := io.ReadAll(resp.Body)
	require.NoError(t, readErr, "response body must be readable after retries are exhausted")
	assert.Contains(t, string(body), "rate limited")
}

func TestDoWithRetrySucceedsAfterRetryableResponse(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"rate limited"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	strategy, err := NewSimpleRetryStrategy(
		WithMaxRetries(3),
		WithFixedDelay(1),
		WithRetryableStatusCodes(http.StatusTooManyRequests),
	)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, strings.NewReader(`{"q":"hello"}`))
	require.NoError(t, err)

	resp, err := strategy.DoWithRetry(server.Client(), req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, int32(2), requests.Load())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, readErr := io.ReadAll(resp.Body)
	require.NoError(t, readErr)
	assert.Contains(t, string(body), `"ok":true`)
}
