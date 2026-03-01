package downloadutil

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
