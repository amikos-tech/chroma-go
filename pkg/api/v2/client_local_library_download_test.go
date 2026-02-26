//go:build basicv2 && !cloud
// +build basicv2,!cloud

package v2

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResolveLocalLibraryPath_PrefersExplicitPathThenEnvThenDownload(t *testing.T) {
	lockLocalTestHooks(t)

	origGetenv := localGetenvFunc
	origEnsure := localEnsureLibraryDownloadedFunc
	origDetect := localDetectLibraryVersionFunc
	origCache := localDefaultLibraryCacheDirFunc
	t.Cleanup(func() {
		localGetenvFunc = origGetenv
		localEnsureLibraryDownloadedFunc = origEnsure
		localDetectLibraryVersionFunc = origDetect
		localDefaultLibraryCacheDirFunc = origCache
	})

	localGetenvFunc = func(key string) string {
		if key == "CHROMA_LIB_PATH" {
			return "/env/libchroma_go_shim.so"
		}
		return ""
	}
	localDetectLibraryVersionFunc = func() (string, error) { return "v9.9.9", nil }
	localDefaultLibraryCacheDirFunc = func() (string, error) { return "/tmp/cache", nil }
	localEnsureLibraryDownloadedFunc = func(version, cacheDir string) (string, error) {
		return "/downloaded/libchroma_go_shim.so", nil
	}

	cfg := &localClientConfig{
		libraryPath:         "/explicit/libchroma_go_shim.so",
		autoDownloadLibrary: true,
	}
	path, err := resolveLocalLibraryPath(cfg)
	require.NoError(t, err)
	require.Equal(t, "/explicit/libchroma_go_shim.so", path)

	cfg.libraryPath = ""
	path, err = resolveLocalLibraryPath(cfg)
	require.NoError(t, err)
	require.Equal(t, "/env/libchroma_go_shim.so", path)

	localGetenvFunc = func(string) string { return "" }
	path, err = resolveLocalLibraryPath(cfg)
	require.NoError(t, err)
	require.Equal(t, "/downloaded/libchroma_go_shim.so", path)
}

func TestResolveLocalLibraryPath_AutoDownloadDisabled(t *testing.T) {
	lockLocalTestHooks(t)

	origGetenv := localGetenvFunc
	origEnsure := localEnsureLibraryDownloadedFunc
	t.Cleanup(func() {
		localGetenvFunc = origGetenv
		localEnsureLibraryDownloadedFunc = origEnsure
	})

	localGetenvFunc = func(string) string { return "" }
	localEnsureLibraryDownloadedFunc = func(version, cacheDir string) (string, error) {
		t.Fatal("download should not be attempted when auto-download is disabled")
		return "", nil
	}

	cfg := &localClientConfig{
		autoDownloadLibrary: false,
	}
	path, err := resolveLocalLibraryPath(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "local runtime library path is not configured")
	require.Equal(t, "", path)
}

func TestResolveLocalLibraryPath_RejectsInvalidDetectedVersion(t *testing.T) {
	lockLocalTestHooks(t)

	origGetenv := localGetenvFunc
	origDetect := localDetectLibraryVersionFunc
	origEnsure := localEnsureLibraryDownloadedFunc
	t.Cleanup(func() {
		localGetenvFunc = origGetenv
		localDetectLibraryVersionFunc = origDetect
		localEnsureLibraryDownloadedFunc = origEnsure
	})

	localGetenvFunc = func(string) string { return "" }
	localDetectLibraryVersionFunc = func() (string, error) { return "../malicious", nil }
	localEnsureLibraryDownloadedFunc = func(version, cacheDir string) (string, error) {
		t.Fatal("download should not be attempted for invalid detected version")
		return "", nil
	}

	_, err := resolveLocalLibraryPath(&localClientConfig{
		autoDownloadLibrary: true,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid detected local library version")
}

func TestEnsureLocalLibraryDownloaded_PropagatesLibraryStatErrors(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission checks are not reliable on windows")
	}

	asset, err := localLibraryAssetForRuntime(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("runtime not supported for local runtime artifacts: %v", err)
	}

	cacheDir := t.TempDir()
	targetDir := filepath.Join(cacheDir, "v9.9.9", asset.platform)
	require.NoError(t, os.MkdirAll(targetDir, 0755))
	require.NoError(t, os.Chmod(targetDir, 0000))
	t.Cleanup(func() {
		_ = os.Chmod(targetDir, 0755)
	})

	_, err = ensureLocalLibraryDownloaded("v9.9.9", cacheDir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to stat local runtime library")
}

func TestLocalAcquireDownloadLock_ReportsStaleLockRemovalError(t *testing.T) {
	lockLocalTestHooks(t)

	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission checks are not reliable on windows")
	}

	origWaitTimeout := localLibraryLockWaitTimeout
	origStaleAfter := localLibraryLockStaleAfter
	localLibraryLockWaitTimeout = 2 * time.Second
	localLibraryLockStaleAfter = -1 * time.Second
	t.Cleanup(func() {
		localLibraryLockWaitTimeout = origWaitTimeout
		localLibraryLockStaleAfter = origStaleAfter
	})

	lockDir := t.TempDir()
	lockPath := filepath.Join(lockDir, localLibraryLockFileName)
	require.NoError(t, os.WriteFile(lockPath, []byte("123"), 0644))
	require.NoError(t, os.Chmod(lockDir, 0555))
	t.Cleanup(func() {
		_ = os.Chmod(lockDir, 0755)
	})

	lockFile, err := localAcquireDownloadLock(lockPath)
	require.Nil(t, lockFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to remove stale lock file")
}

func TestLocalReleaseDownloadLock_ReportsCloseError(t *testing.T) {
	lockDir := t.TempDir()
	lockPath := filepath.Join(lockDir, localLibraryLockFileName)

	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	require.NoError(t, err)
	require.NoError(t, lockFile.Close())

	err = localReleaseDownloadLock(lockFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to close download lock file")
}

func TestLocalReleaseDownloadLock_ReportsRemoveError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission checks are not reliable on windows")
	}

	lockDir := t.TempDir()
	lockPath := filepath.Join(lockDir, localLibraryLockFileName)
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	require.NoError(t, err)

	require.NoError(t, os.Chmod(lockDir, 0555))
	t.Cleanup(func() {
		_ = os.Chmod(lockDir, 0755)
	})

	err = localReleaseDownloadLock(lockFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to remove download lock file")
}

func TestLocalAcquireDownloadLock_DoesNotExpireActiveLockWithHeartbeat(t *testing.T) {
	lockLocalTestHooks(t)

	origWaitTimeout := localLibraryLockWaitTimeout
	origStaleAfter := localLibraryLockStaleAfter
	origHeartbeat := localLibraryLockHeartbeatInterval
	localLibraryLockWaitTimeout = 700 * time.Millisecond
	localLibraryLockStaleAfter = 1500 * time.Millisecond
	localLibraryLockHeartbeatInterval = 100 * time.Millisecond
	t.Cleanup(func() {
		localLibraryLockWaitTimeout = origWaitTimeout
		localLibraryLockStaleAfter = origStaleAfter
		localLibraryLockHeartbeatInterval = origHeartbeat
	})

	lockDir := t.TempDir()
	lockPath := filepath.Join(lockDir, localLibraryLockFileName)
	lockFile, err := localAcquireDownloadLock(lockPath)
	require.NoError(t, err)

	stopHeartbeat := localStartDownloadLockHeartbeat(lockFile)
	t.Cleanup(func() {
		require.NoError(t, stopHeartbeat())
		require.NoError(t, localReleaseDownloadLock(lockFile))
	})

	old := time.Now().Add(-10 * time.Second)
	require.NoError(t, os.Chtimes(lockPath, old, old))
	time.Sleep(3 * localLibraryLockHeartbeatInterval)

	secondLock, err := localAcquireDownloadLock(lockPath)
	require.Nil(t, secondLock)
	require.Error(t, err)
	require.Contains(t, err.Error(), "timeout waiting for lock")
}

func TestLocalStartDownloadLockHeartbeat_StopIsIdempotent(t *testing.T) {
	lockLocalTestHooks(t)

	origHeartbeat := localLibraryLockHeartbeatInterval
	localLibraryLockHeartbeatInterval = 50 * time.Millisecond
	t.Cleanup(func() {
		localLibraryLockHeartbeatInterval = origHeartbeat
	})

	lockDir := t.TempDir()
	lockPath := filepath.Join(lockDir, localLibraryLockFileName)
	lockFile, err := localAcquireDownloadLock(lockPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, localReleaseDownloadLock(lockFile))
	})

	stopHeartbeat := localStartDownloadLockHeartbeat(lockFile)
	require.NoError(t, stopHeartbeat())
	require.NoError(t, stopHeartbeat())
}

func TestEnsureLocalLibraryDownloaded_DownloadsAndExtracts(t *testing.T) {
	lockLocalTestHooks(t)

	asset, err := localLibraryAssetForRuntime(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("runtime not supported for local runtime artifacts: %v", err)
	}

	archiveBytes := newTarGzWithLibrary(t, asset.libraryFileName, []byte("local-shim-bytes"))
	checksum := sha256.Sum256(archiveBytes)
	checksumHex := hex.EncodeToString(checksum[:])
	sumsBody := []byte(checksumHex + "  " + asset.archiveName + "\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v9.9.9/" + localLibraryChecksumsAsset:
			_, _ = w.Write(sumsBody)
		case "/v9.9.9/" + asset.archiveName:
			_, _ = w.Write(archiveBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	origBaseURL := localLibraryReleaseBaseURL
	origAttempts := localLibraryDownloadAttempts
	t.Cleanup(func() {
		localLibraryReleaseBaseURL = origBaseURL
		localLibraryDownloadAttempts = origAttempts
	})
	localLibraryReleaseBaseURL = server.URL
	localLibraryDownloadAttempts = 1

	cacheDir := t.TempDir()
	libPath, err := ensureLocalLibraryDownloaded("v9.9.9", cacheDir)
	require.NoError(t, err)
	require.FileExists(t, libPath)

	content, err := os.ReadFile(libPath)
	require.NoError(t, err)
	require.Equal(t, "local-shim-bytes", string(content))
}

func TestEnsureLocalLibraryDownloaded_FailsOnChecksumMismatch(t *testing.T) {
	lockLocalTestHooks(t)

	asset, err := localLibraryAssetForRuntime(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("runtime not supported for local runtime artifacts: %v", err)
	}

	archiveBytes := newTarGzWithLibrary(t, asset.libraryFileName, []byte("local-shim-bytes"))
	sumsBody := []byte("deadbeef  " + asset.archiveName + "\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v9.9.9/" + localLibraryChecksumsAsset:
			_, _ = w.Write(sumsBody)
		case "/v9.9.9/" + asset.archiveName:
			_, _ = w.Write(archiveBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	origBaseURL := localLibraryReleaseBaseURL
	origAttempts := localLibraryDownloadAttempts
	t.Cleanup(func() {
		localLibraryReleaseBaseURL = origBaseURL
		localLibraryDownloadAttempts = origAttempts
	})
	localLibraryReleaseBaseURL = server.URL
	localLibraryDownloadAttempts = 1

	cacheDir := t.TempDir()
	_, err = ensureLocalLibraryDownloaded("v9.9.9", cacheDir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "checksum verification failed")
}

func TestLocalDownloadFile_RejectsOversizedArtifact(t *testing.T) {
	lockLocalTestHooks(t)

	origMaxSize := localLibraryMaxArtifactBytes
	localLibraryMaxArtifactBytes = 16
	t.Cleanup(func() {
		localLibraryMaxArtifactBytes = origMaxSize
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("this payload is definitely larger than sixteen bytes"))
	}))
	defer server.Close()

	targetPath := filepath.Join(t.TempDir(), "artifact.bin")
	err := localDownloadFile(targetPath, server.URL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "too large")
}

func TestLocalExtractLibraryFromTarGz_RejectsOversizedLibrary(t *testing.T) {
	lockLocalTestHooks(t)

	origMaxSize := localLibraryMaxArtifactBytes
	localLibraryMaxArtifactBytes = 16
	t.Cleanup(func() {
		localLibraryMaxArtifactBytes = origMaxSize
	})

	archivePath := filepath.Join(t.TempDir(), "artifact.tar.gz")
	archiveBytes := newTarGzWithLibrary(t, "libchroma_go_shim.so", []byte("this payload is too large"))
	require.NoError(t, os.WriteFile(archivePath, archiveBytes, 0644))

	targetPath := filepath.Join(t.TempDir(), "libchroma_go_shim.so")
	err := localExtractLibraryFromTarGz(archivePath, "libchroma_go_shim.so", targetPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds max allowed size")
}

func TestLocalRejectHTTPSDowngradeRedirect(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com/next", nil)
	require.NoError(t, err)
	via, err := http.NewRequest(http.MethodGet, "https://example.com/start", nil)
	require.NoError(t, err)

	err = localRejectHTTPSDowngradeRedirect(req, []*http.Request{via})
	require.Error(t, err)
	require.Contains(t, err.Error(), "redirect from HTTPS to HTTP is not allowed")

	secureReq, err := http.NewRequest(http.MethodGet, "https://example.com/next", nil)
	require.NoError(t, err)
	require.NoError(t, localRejectHTTPSDowngradeRedirect(secureReq, []*http.Request{via}))

	longRedirectChain := make([]*http.Request, 10)
	for i := range longRedirectChain {
		longRedirectChain[i], err = http.NewRequest(http.MethodGet, "https://example.com/redirect", nil)
		require.NoError(t, err)
	}
	err = localRejectHTTPSDowngradeRedirect(secureReq, longRedirectChain)
	require.Error(t, err)
	require.Contains(t, err.Error(), "stopped after 10 redirects")
}

func newTarGzWithLibrary(t *testing.T, fileName string, content []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	header := &tar.Header{
		Name: "./" + fileName,
		Mode: 0644,
		Size: int64(len(content)),
	}
	require.NoError(t, tarWriter.WriteHeader(header))
	_, err := tarWriter.Write(content)
	require.NoError(t, err)
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzWriter.Close())

	return buf.Bytes()
}
