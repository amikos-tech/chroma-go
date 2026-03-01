package downloadutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDownloadFile_RejectsHTTPByDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer server.Close()

	targetPath := filepath.Join(t.TempDir(), "artifact.bin")
	err := DownloadFile(targetPath, server.URL, Config{MaxBytes: 1024, DirPerm: 0700})
	require.Error(t, err)
	require.Contains(t, err.Error(), "only HTTPS URLs are supported")
}

func TestDownloadFile_AllowsHTTPWhenConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer server.Close()

	targetPath := filepath.Join(t.TempDir(), "artifact.bin")
	err := DownloadFile(targetPath, server.URL, Config{
		MaxBytes:  1024,
		DirPerm:   0700,
		AllowHTTP: true,
	})
	require.NoError(t, err)

	data, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	require.Equal(t, "payload", string(data))
}

func TestValidateSourceURL_RejectsCredentials(t *testing.T) {
	_, err := validateSourceURL("https://alice:super-secret@example.com/artifact.tgz", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "source URL must not contain credentials")
	require.NotContains(t, err.Error(), "super-secret")
	require.Contains(t, err.Error(), "xxxxx")
}

func TestValidateSourceURL_RejectsRelativeURLAndEmptyHost(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "relative URL", raw: "/artifact.tgz", want: "must be absolute"},
		{name: "empty host", raw: "https:///artifact.tgz", want: "host cannot be empty"},
		{name: "file scheme", raw: "file://localhost/tmp/artifact.tgz", want: "only HTTPS URLs are supported"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateSourceURL(tc.raw, false)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestRejectHTTPSDowngradeRedirect_RedactsCredentials(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://next:s3cr3t@example.com/download", nil)
	require.NoError(t, err)
	via, err := http.NewRequest(http.MethodGet, "https://prev:topsecret@example.com/download", nil)
	require.NoError(t, err)

	err = RejectHTTPSDowngradeRedirect(req, []*http.Request{via})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "s3cr3t")
	require.NotContains(t, err.Error(), "topsecret")
	require.Contains(t, err.Error(), "xxxxx")
}

func TestDownloadFileWithRetry_ReusesSingleHTTPClient(t *testing.T) {
	originalClientFactory := newHTTPClientFunc
	t.Cleanup(func() {
		newHTTPClientFunc = originalClientFactory
	})

	var clientCreations int32
	newHTTPClientFunc = func(cfg Config) *http.Client {
		atomic.AddInt32(&clientCreations, 1)
		return newHTTPClient(cfg)
	}

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch atomic.AddInt32(&requestCount, 1) {
		case 1, 2:
			http.Error(w, "retry", http.StatusInternalServerError)
		default:
			_, _ = w.Write([]byte("payload"))
		}
	}))
	defer server.Close()

	targetPath := filepath.Join(t.TempDir(), "artifact.bin")
	err := DownloadFileWithRetry(targetPath, server.URL, 3, Config{
		MaxBytes:  1024,
		DirPerm:   0700,
		AllowHTTP: true,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, atomic.LoadInt32(&clientCreations))
	require.EqualValues(t, 3, atomic.LoadInt32(&requestCount))
}

func TestDownloadFileWithClient_RejectsIncompleteDownload(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:    http.StatusOK,
				Status:        "200 OK",
				Header:        make(http.Header),
				Body:          io.NopCloser(strings.NewReader("abc")),
				ContentLength: 10,
				Request:       req,
			}, nil
		}),
	}

	targetPath := filepath.Join(t.TempDir(), "artifact.bin")
	parsedURL, err := url.Parse("https://example.com/artifact.bin")
	require.NoError(t, err)

	err = downloadFileWithClient(targetPath, parsedURL, withDefaults(Config{MaxBytes: 1024, DirPerm: 0700}), client)
	require.Error(t, err)
	require.Contains(t, err.Error(), "download incomplete")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
