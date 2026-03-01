package downloadutil

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
