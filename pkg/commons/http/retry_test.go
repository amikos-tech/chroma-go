package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRetryStrategyWithExponentialBackOff(t *testing.T) {
	client := &http.Client{}

	retryableStatusCodes := []int{http.StatusInternalServerError}

	// Create a new RetryStrategy with exponential backoff enabled
	retryStrategy, err := NewRetryStrategy(client,
		WithMaxRetries(3),
		WithFixedDelay(100*time.Millisecond),
		WithRetryableStatusCodes(retryableStatusCodes),
		WithExponentialBackOff(),
	)
	require.NoError(t, err, "error setting up strategy: %v", err)
	var serverRetries = 0
	// Create a test server that always returns a 500 status code
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverRetries++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err, "unexpected error: %v", err)

	startTime := time.Now()

	_, err = retryStrategy.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Calculate the total elapsed time
	elapsedTime := time.Since(startTime)
	// Since we have exponential backoff with delays 100ms, 200ms, 400ms, the total delay should be at least 700ms
	expectedMinDelay := 100*time.Millisecond + 200*time.Millisecond + 400*time.Millisecond
	require.Less(t, expectedMinDelay, elapsedTime, "expected total delay to be at least %v, but got %v", expectedMinDelay, elapsedTime)
	require.Equal(t, 3, serverRetries)
}
