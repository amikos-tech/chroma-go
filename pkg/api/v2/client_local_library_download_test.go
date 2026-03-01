//go:build basicv2 && !cloud
// +build basicv2,!cloud

package v2

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func localLibraryArchiveName(version, platform string) string {
	archiveNames := localLibraryArchiveNames(version, platform)
	if len(archiveNames) == 0 {
		return ""
	}
	return archiveNames[0]
}

func localChecksumFromSumsFile(sumsFilePath, assetName string) (string, error) {
	_, checksum, err := localChecksumFromSumsFileAny(sumsFilePath, []string{assetName})
	if err != nil {
		return "", err
	}
	return checksum, nil
}

func bypassLocalCosignChainVerification(t *testing.T) {
	t.Helper()
	orig := localVerifyCosignCertificateChainFunc
	localVerifyCosignCertificateChainFunc = func(*x509.Certificate) error { return nil }
	t.Cleanup(func() {
		localVerifyCosignCertificateChainFunc = orig
	})
}

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
			return "/env/libchroma_shim.so"
		}
		return ""
	}
	localDetectLibraryVersionFunc = func() (string, error) { return "v9.9.9", nil }
	localDefaultLibraryCacheDirFunc = func() (string, error) { return "/tmp/cache", nil }
	localEnsureLibraryDownloadedFunc = func(version, cacheDir string) (string, error) {
		return "/downloaded/libchroma_shim.so", nil
	}

	cfg := &localClientConfig{
		libraryPath:         "/explicit/libchroma_shim.so",
		autoDownloadLibrary: true,
	}
	path, err := resolveLocalLibraryPath(cfg)
	require.NoError(t, err)
	require.Equal(t, "/explicit/libchroma_shim.so", path)

	cfg.libraryPath = ""
	path, err = resolveLocalLibraryPath(cfg)
	require.NoError(t, err)
	require.Equal(t, "/env/libchroma_shim.so", path)

	localGetenvFunc = func(string) string { return "" }
	path, err = resolveLocalLibraryPath(cfg)
	require.NoError(t, err)
	require.Equal(t, "/downloaded/libchroma_shim.so", path)
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

func TestNormalizeLocalLibraryTag_AllowlistsSafeCharacters(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        string
		wantErrPart string
	}{
		{name: "empty", input: "", want: ""},
		{name: "devel", input: "(devel)", want: ""},
		{name: "plain semver", input: "0.2.0", want: "v0.2.0"},
		{name: "prefixed semver", input: "v0.2.0", want: "v0.2.0"},
		{name: "pre-release", input: "v0.2.0-rc.1", want: "v0.2.0-rc.1"},
		{name: "underscore", input: "v0_2_0", want: "v0_2_0"},
		{name: "reject slash", input: "v0.2.0/next", wantErrPart: "only ASCII letters"},
		{name: "reject query", input: "v0.2.0?x=y", wantErrPart: "only ASCII letters"},
		{name: "reject fragment", input: "v0.2.0#frag", wantErrPart: "only ASCII letters"},
		{name: "reject percent", input: "v0.2.0%2f", wantErrPart: "only ASCII letters"},
		{name: "reject backslash", input: "v0.2.0\\win", wantErrPart: "only ASCII letters"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeLocalLibraryTag(tc.input)
			if tc.wantErrPart != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrPart)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
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
	bypassLocalCosignChainVerification(t)

	asset, err := localLibraryAssetForRuntime(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("runtime not supported for local runtime artifacts: %v", err)
	}

	archiveBytes := newTarGzWithLibrary(t, asset.libraryFileName, []byte("local-shim-bytes"))
	archiveName := localLibraryArchiveName("v9.9.9", asset.platform)
	checksum := sha256.Sum256(archiveBytes)
	checksumHex := hex.EncodeToString(checksum[:])
	sumsBody := []byte(checksumHex + "  " + archiveName + "\n")
	sumsSignatureBody, sumsCertificatePEM := newSignedChecksumArtifacts(t, "v9.9.9", sumsBody)
	// Mirror currently serves .pem files as base64-encoded PEM payloads.
	sumsCertificateBody := []byte(base64.StdEncoding.EncodeToString(sumsCertificatePEM))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v9.9.9/" + localLibraryChecksumsAsset:
			_, _ = w.Write(sumsBody)
		case "/v9.9.9/" + localLibraryChecksumsSignatureAsset:
			_, _ = w.Write(sumsSignatureBody)
		case "/v9.9.9/" + localLibraryChecksumsCertificateAsset:
			_, _ = w.Write(sumsCertificateBody)
		case "/v9.9.9/" + archiveName:
			_, _ = w.Write(archiveBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	origBaseURL := localLibraryReleaseBaseURL
	origFallbackURL := localLibraryReleaseFallbackBaseURL
	origAttempts := localLibraryDownloadAttempts
	t.Cleanup(func() {
		localLibraryReleaseBaseURL = origBaseURL
		localLibraryReleaseFallbackBaseURL = origFallbackURL
		localLibraryDownloadAttempts = origAttempts
	})
	localLibraryReleaseBaseURL = server.URL
	localLibraryReleaseFallbackBaseURL = ""
	localLibraryDownloadAttempts = 1

	cacheDir := t.TempDir()
	libPath, err := ensureLocalLibraryDownloaded("v9.9.9", cacheDir)
	require.NoError(t, err)
	require.FileExists(t, libPath)

	content, err := os.ReadFile(libPath)
	require.NoError(t, err)
	require.Equal(t, "local-shim-bytes", string(content))

	if runtime.GOOS != "windows" {
		libInfo, err := os.Stat(libPath)
		require.NoError(t, err)
		require.Equal(t, localLibraryArtifactFilePerm, libInfo.Mode().Perm())

		dirInfo, err := os.Stat(filepath.Dir(libPath))
		require.NoError(t, err)
		require.Equal(t, localLibraryCacheDirPerm, dirInfo.Mode().Perm())
	}
}

func TestEnsureLocalLibraryDownloaded_DownloadsAndExtractsLocalChromaArchive(t *testing.T) {
	lockLocalTestHooks(t)
	bypassLocalCosignChainVerification(t)

	asset, err := localLibraryAssetForRuntime(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("runtime not supported for local runtime artifacts: %v", err)
	}

	archiveBytes := newTarGzWithLibrary(t, asset.libraryFileName, []byte("local-shim-bytes"))
	archiveNames := localLibraryArchiveNames("v9.9.9", asset.platform)
	require.GreaterOrEqual(t, len(archiveNames), 2)
	var localChromaArchiveName string
	for _, archiveName := range archiveNames {
		if strings.Contains(archiveName, localLibraryArchivePrefixLocalChroma+"-") {
			localChromaArchiveName = archiveName
			break
		}
	}
	require.NotEmpty(t, localChromaArchiveName)
	checksum := sha256.Sum256(archiveBytes)
	checksumHex := hex.EncodeToString(checksum[:])
	sumsBody := []byte(checksumHex + "  " + localChromaArchiveName + "\n")
	sumsSignatureBody, sumsCertificatePEM := newSignedChecksumArtifacts(t, "v9.9.9", sumsBody)
	sumsCertificateBody := []byte(base64.StdEncoding.EncodeToString(sumsCertificatePEM))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v9.9.9/" + localLibraryChecksumsAsset:
			_, _ = w.Write(sumsBody)
		case "/v9.9.9/" + localLibraryChecksumsSignatureAsset:
			_, _ = w.Write(sumsSignatureBody)
		case "/v9.9.9/" + localLibraryChecksumsCertificateAsset:
			_, _ = w.Write(sumsCertificateBody)
		case "/v9.9.9/" + localChromaArchiveName:
			_, _ = w.Write(archiveBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	origBaseURL := localLibraryReleaseBaseURL
	origFallbackURL := localLibraryReleaseFallbackBaseURL
	origAttempts := localLibraryDownloadAttempts
	t.Cleanup(func() {
		localLibraryReleaseBaseURL = origBaseURL
		localLibraryReleaseFallbackBaseURL = origFallbackURL
		localLibraryDownloadAttempts = origAttempts
	})
	localLibraryReleaseBaseURL = server.URL
	localLibraryReleaseFallbackBaseURL = ""
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
	bypassLocalCosignChainVerification(t)

	asset, err := localLibraryAssetForRuntime(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("runtime not supported for local runtime artifacts: %v", err)
	}

	archiveBytes := newTarGzWithLibrary(t, asset.libraryFileName, []byte("local-shim-bytes"))
	archiveName := localLibraryArchiveName("v9.9.9", asset.platform)
	sumsBody := []byte("deadbeef  " + archiveName + "\n")
	sumsSignatureBody, sumsCertificateBody := newSignedChecksumArtifacts(t, "v9.9.9", sumsBody)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v9.9.9/" + localLibraryChecksumsAsset:
			_, _ = w.Write(sumsBody)
		case "/v9.9.9/" + localLibraryChecksumsSignatureAsset:
			_, _ = w.Write(sumsSignatureBody)
		case "/v9.9.9/" + localLibraryChecksumsCertificateAsset:
			_, _ = w.Write(sumsCertificateBody)
		case "/v9.9.9/" + archiveName:
			_, _ = w.Write(archiveBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	origBaseURL := localLibraryReleaseBaseURL
	origFallbackURL := localLibraryReleaseFallbackBaseURL
	origAttempts := localLibraryDownloadAttempts
	t.Cleanup(func() {
		localLibraryReleaseBaseURL = origBaseURL
		localLibraryReleaseFallbackBaseURL = origFallbackURL
		localLibraryDownloadAttempts = origAttempts
	})
	localLibraryReleaseBaseURL = server.URL
	localLibraryReleaseFallbackBaseURL = ""
	localLibraryDownloadAttempts = 1

	cacheDir := t.TempDir()
	_, err = ensureLocalLibraryDownloaded("v9.9.9", cacheDir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "checksum verification failed")
}

func TestEnsureLocalLibraryDownloaded_FailsOnSignedChecksumsVerification(t *testing.T) {
	lockLocalTestHooks(t)
	bypassLocalCosignChainVerification(t)

	asset, err := localLibraryAssetForRuntime(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("runtime not supported for local runtime artifacts: %v", err)
	}

	archiveBytes := newTarGzWithLibrary(t, asset.libraryFileName, []byte("local-shim-bytes"))
	archiveName := localLibraryArchiveName("v9.9.9", asset.platform)
	checksum := sha256.Sum256(archiveBytes)
	checksumHex := hex.EncodeToString(checksum[:])
	sumsBody := []byte(checksumHex + "  " + archiveName + "\n")
	_, sumsCertificateBody := newSignedChecksumArtifacts(t, "v9.9.9", sumsBody)
	invalidSignatureBody := []byte(base64.StdEncoding.EncodeToString([]byte("invalid-signature")))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v9.9.9/" + localLibraryChecksumsAsset:
			_, _ = w.Write(sumsBody)
		case "/v9.9.9/" + localLibraryChecksumsSignatureAsset:
			_, _ = w.Write(invalidSignatureBody)
		case "/v9.9.9/" + localLibraryChecksumsCertificateAsset:
			_, _ = w.Write(sumsCertificateBody)
		case "/v9.9.9/" + archiveName:
			_, _ = w.Write(archiveBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	origBaseURL := localLibraryReleaseBaseURL
	origFallbackURL := localLibraryReleaseFallbackBaseURL
	origAttempts := localLibraryDownloadAttempts
	t.Cleanup(func() {
		localLibraryReleaseBaseURL = origBaseURL
		localLibraryReleaseFallbackBaseURL = origFallbackURL
		localLibraryDownloadAttempts = origAttempts
	})
	localLibraryReleaseBaseURL = server.URL
	localLibraryReleaseFallbackBaseURL = ""
	localLibraryDownloadAttempts = 1

	cacheDir := t.TempDir()
	_, err = ensureLocalLibraryDownloaded("v9.9.9", cacheDir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "checksums signature")
}

func TestEnsureLocalLibraryDownloaded_VerifyFailure_JoinsRemoveError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission checks are not reliable on windows")
	}

	tests := []struct {
		name             string
		setupArchive     func(t *testing.T, asset localLibraryAsset, targetDir, archiveName string) (archiveBytes []byte, sumsBody []byte, chmodTrigger string)
		expectedMessages []string
	}{
		{
			name: "existing archive checksum mismatch",
			setupArchive: func(t *testing.T, asset localLibraryAsset, targetDir, archiveName string) ([]byte, []byte, string) {
				t.Helper()
				require.NoError(t, os.MkdirAll(targetDir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(targetDir, archiveName), []byte("existing-corrupt-archive"), 0644))
				return nil, []byte("deadbeef  " + archiveName + "\n"), localLibraryChecksumsCertificateAsset
			},
			expectedMessages: []string{"existing local runtime archive checksum verification failed", "failed to remove corrupted local runtime archive"},
		},
		{
			name: "downloaded archive checksum mismatch",
			setupArchive: func(t *testing.T, asset localLibraryAsset, targetDir, archiveName string) ([]byte, []byte, string) {
				t.Helper()
				archiveBytes := newTarGzWithLibrary(t, asset.libraryFileName, []byte("local-shim-bytes"))
				return archiveBytes, []byte("deadbeef  " + archiveName + "\n"), archiveName
			},
			expectedMessages: []string{"local library archive checksum verification failed", "failed to remove corrupted local runtime archive"},
		},
		{
			name: "downloaded archive tar.gz verification failure",
			setupArchive: func(t *testing.T, asset localLibraryAsset, targetDir, archiveName string) ([]byte, []byte, string) {
				t.Helper()
				archiveBytes := []byte("definitely-not-a-tar-gz")
				checksum := sha256.Sum256(archiveBytes)
				checksumHex := hex.EncodeToString(checksum[:])
				return archiveBytes, []byte(checksumHex + "  " + archiveName + "\n"), archiveName
			},
			expectedMessages: []string{"local library archive verification failed", "failed to remove corrupted local runtime archive"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lockLocalTestHooks(t)
			bypassLocalCosignChainVerification(t)

			asset, err := localLibraryAssetForRuntime(runtime.GOOS, runtime.GOARCH)
			if err != nil {
				t.Skipf("runtime not supported for local runtime artifacts: %v", err)
			}

			cacheDir := t.TempDir()
			targetDir := filepath.Join(cacheDir, "v9.9.9", asset.platform)
			archiveName := localLibraryArchiveName("v9.9.9", asset.platform)
			archiveBytes, sumsBody, chmodTrigger := tc.setupArchive(t, asset, targetDir, archiveName)
			sumsSignatureBody, sumsCertificateBody := newSignedChecksumArtifacts(t, "v9.9.9", sumsBody)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v9.9.9/" + localLibraryChecksumsAsset:
					_, _ = w.Write(sumsBody)
				case "/v9.9.9/" + localLibraryChecksumsSignatureAsset:
					_, _ = w.Write(sumsSignatureBody)
				case "/v9.9.9/" + localLibraryChecksumsCertificateAsset:
					_, _ = w.Write(sumsCertificateBody)
				case "/v9.9.9/" + archiveName:
					if archiveBytes != nil {
						_, _ = w.Write(archiveBytes)
					} else {
						http.NotFound(w, r)
					}
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			origBaseURL := localLibraryReleaseBaseURL
			origFallbackURL := localLibraryReleaseFallbackBaseURL
			origAttempts := localLibraryDownloadAttempts
			origDownloadFile := localDownloadFileFunc
			t.Cleanup(func() {
				localLibraryReleaseBaseURL = origBaseURL
				localLibraryReleaseFallbackBaseURL = origFallbackURL
				localLibraryDownloadAttempts = origAttempts
				localDownloadFileFunc = origDownloadFile
				_ = os.Chmod(targetDir, 0755)
			})
			localLibraryReleaseBaseURL = server.URL
			localLibraryReleaseFallbackBaseURL = ""
			localLibraryDownloadAttempts = 1
			localDownloadFileFunc = func(filePath, url string) error {
				if err := origDownloadFile(filePath, url); err != nil {
					return err
				}
				if strings.HasSuffix(url, "/"+chmodTrigger) {
					return os.Chmod(targetDir, 0555)
				}
				return nil
			}

			_, err = ensureLocalLibraryDownloaded("v9.9.9", cacheDir)
			require.Error(t, err)
			for _, msg := range tc.expectedMessages {
				require.Contains(t, err.Error(), msg)
			}
		})
	}
}

func TestEnsureLocalLibraryDownloaded_FallsBackToSecondaryReleaseURL(t *testing.T) {
	lockLocalTestHooks(t)
	bypassLocalCosignChainVerification(t)

	asset, err := localLibraryAssetForRuntime(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("runtime not supported for local runtime artifacts: %v", err)
	}

	archiveBytes := newTarGzWithLibrary(t, asset.libraryFileName, []byte("local-shim-bytes"))
	archiveName := localLibraryArchiveName("v9.9.9", asset.platform)
	checksum := sha256.Sum256(archiveBytes)
	checksumHex := hex.EncodeToString(checksum[:])
	sumsBody := []byte(checksumHex + "  " + archiveName + "\n")
	sumsSignatureBody, sumsCertificateBody := newSignedChecksumArtifacts(t, "v9.9.9", sumsBody)
	var primaryArchiveRequests int32

	primaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v9.9.9/" + localLibraryChecksumsAsset:
			_, _ = w.Write(sumsBody)
		case "/v9.9.9/" + archiveName:
			atomic.AddInt32(&primaryArchiveRequests, 1)
			http.Error(w, "primary archive should not be requested", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer primaryServer.Close()

	fallbackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v9.9.9/" + localLibraryChecksumsAsset:
			_, _ = w.Write(sumsBody)
		case "/v9.9.9/" + localLibraryChecksumsSignatureAsset:
			_, _ = w.Write(sumsSignatureBody)
		case "/v9.9.9/" + localLibraryChecksumsCertificateAsset:
			_, _ = w.Write(sumsCertificateBody)
		case "/v9.9.9/" + archiveName:
			_, _ = w.Write(archiveBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer fallbackServer.Close()

	origBaseURL := localLibraryReleaseBaseURL
	origFallbackURL := localLibraryReleaseFallbackBaseURL
	origAttempts := localLibraryDownloadAttempts
	t.Cleanup(func() {
		localLibraryReleaseBaseURL = origBaseURL
		localLibraryReleaseFallbackBaseURL = origFallbackURL
		localLibraryDownloadAttempts = origAttempts
	})
	localLibraryReleaseBaseURL = primaryServer.URL
	localLibraryReleaseFallbackBaseURL = fallbackServer.URL
	localLibraryDownloadAttempts = 1

	cacheDir := t.TempDir()
	libPath, err := ensureLocalLibraryDownloaded("v9.9.9", cacheDir)
	require.NoError(t, err)
	require.FileExists(t, libPath)
	require.EqualValues(t, 0, atomic.LoadInt32(&primaryArchiveRequests))
}

func TestLocalValidateCosignCertificate_AllowsExpiredFulcioCertificate(t *testing.T) {
	lockLocalTestHooks(t)
	bypassLocalCosignChainVerification(t)

	certificate := newCosignCertificate(t, localTestCosignCertificateOptions{
		identity:    fmt.Sprintf(localLibraryCosignIdentityTemplate, "v9.9.9"),
		oidcIssuer:  localLibraryCosignOIDCIssuer,
		notBefore:   time.Now().Add(-2 * time.Hour),
		notAfter:    time.Now().Add(-1 * time.Hour),
		extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	})

	err := localValidateCosignCertificate(
		certificate,
		fmt.Sprintf(localLibraryCosignIdentityTemplate, "v9.9.9"),
		localLibraryCosignOIDCIssuer,
	)
	require.NoError(t, err)
}

func TestLocalValidateCosignCertificate_AllowsRawOIDCIssuerExtension(t *testing.T) {
	lockLocalTestHooks(t)
	bypassLocalCosignChainVerification(t)

	certificate := newCosignCertificate(t, localTestCosignCertificateOptions{
		identity:           fmt.Sprintf(localLibraryCosignIdentityTemplate, "v9.9.9"),
		oidcIssuer:         localLibraryCosignOIDCIssuer,
		oidcIssuerRawValue: true,
		notBefore:          time.Now().Add(-1 * time.Minute),
		notAfter:           time.Now().Add(10 * time.Minute),
		extKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	})

	err := localValidateCosignCertificate(
		certificate,
		fmt.Sprintf(localLibraryCosignIdentityTemplate, "v9.9.9"),
		localLibraryCosignOIDCIssuer,
	)
	require.NoError(t, err)
}

func TestLocalValidateCosignCertificate_RejectsFutureNotBefore(t *testing.T) {
	lockLocalTestHooks(t)
	bypassLocalCosignChainVerification(t)

	certificate := newCosignCertificate(t, localTestCosignCertificateOptions{
		identity:    fmt.Sprintf(localLibraryCosignIdentityTemplate, "v9.9.9"),
		oidcIssuer:  localLibraryCosignOIDCIssuer,
		notBefore:   time.Now().Add(10 * time.Minute),
		notAfter:    time.Now().Add(20 * time.Minute),
		extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	})

	err := localValidateCosignCertificate(
		certificate,
		fmt.Sprintf(localLibraryCosignIdentityTemplate, "v9.9.9"),
		localLibraryCosignOIDCIssuer,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not yet valid")
}

func TestLocalValidateCosignCertificate_RejectsMissingCodeSigningUsage(t *testing.T) {
	lockLocalTestHooks(t)
	bypassLocalCosignChainVerification(t)

	certificate := newCosignCertificate(t, localTestCosignCertificateOptions{
		identity:    fmt.Sprintf(localLibraryCosignIdentityTemplate, "v9.9.9"),
		oidcIssuer:  localLibraryCosignOIDCIssuer,
		notBefore:   time.Now().Add(-1 * time.Minute),
		notAfter:    time.Now().Add(10 * time.Minute),
		extKeyUsage: nil,
	})

	err := localValidateCosignCertificate(
		certificate,
		fmt.Sprintf(localLibraryCosignIdentityTemplate, "v9.9.9"),
		localLibraryCosignOIDCIssuer,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "code signing")
}

func TestLocalValidateCosignCertificate_RejectsWrongIdentityAndIssuer(t *testing.T) {
	lockLocalTestHooks(t)
	bypassLocalCosignChainVerification(t)

	certificate := newCosignCertificate(t, localTestCosignCertificateOptions{
		identity:    fmt.Sprintf(localLibraryCosignIdentityTemplate, "v0.0.1"),
		oidcIssuer:  "https://example.invalid/issuer",
		notBefore:   time.Now().Add(-1 * time.Minute),
		notAfter:    time.Now().Add(10 * time.Minute),
		extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	})

	err := localValidateCosignCertificate(
		certificate,
		fmt.Sprintf(localLibraryCosignIdentityTemplate, "v9.9.9"),
		localLibraryCosignOIDCIssuer,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "identity")
}

func TestLocalValidateCosignCertificate_RejectsWrongOIDCIssuer(t *testing.T) {
	lockLocalTestHooks(t)
	bypassLocalCosignChainVerification(t)

	certificate := newCosignCertificate(t, localTestCosignCertificateOptions{
		identity:    fmt.Sprintf(localLibraryCosignIdentityTemplate, "v9.9.9"),
		oidcIssuer:  "https://example.invalid/issuer",
		notBefore:   time.Now().Add(-1 * time.Minute),
		notAfter:    time.Now().Add(10 * time.Minute),
		extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	})

	err := localValidateCosignCertificate(
		certificate,
		fmt.Sprintf(localLibraryCosignIdentityTemplate, "v9.9.9"),
		localLibraryCosignOIDCIssuer,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "OIDC issuer mismatch")
}

func TestLocalValidateCosignCertificate_RejectsUntrustedChain(t *testing.T) {
	lockLocalTestHooks(t)

	certificate := newCosignCertificate(t, localTestCosignCertificateOptions{
		identity:    fmt.Sprintf(localLibraryCosignIdentityTemplate, "v9.9.9"),
		oidcIssuer:  localLibraryCosignOIDCIssuer,
		notBefore:   time.Now().Add(-1 * time.Minute),
		notAfter:    time.Now().Add(10 * time.Minute),
		extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	})

	err := localValidateCosignCertificate(
		certificate,
		fmt.Sprintf(localLibraryCosignIdentityTemplate, "v9.9.9"),
		localLibraryCosignOIDCIssuer,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "certificate chain verification failed")
}

func TestLocalReadBase64EncodedFile_DecodePaths(t *testing.T) {
	dir := t.TempDir()
	rawPath := filepath.Join(dir, "raw.txt")
	invalidPath := filepath.Join(dir, "invalid.txt")

	payload := []byte("hello-base64")
	require.NoError(t, os.WriteFile(rawPath, []byte(base64.RawStdEncoding.EncodeToString(payload)), 0644))
	require.NoError(t, os.WriteFile(invalidPath, []byte("%%%not-base64%%%"), 0644))

	decoded, err := localReadBase64EncodedFile(rawPath)
	require.NoError(t, err)
	require.Equal(t, payload, decoded)

	_, err = localReadBase64EncodedFile(invalidPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid base64 payload")
}

func TestLocalReadCosignCertificate_RejectsInvalidAndTrailingPEM(t *testing.T) {
	dir := t.TempDir()
	invalidPath := filepath.Join(dir, "invalid.pem")
	trailingPath := filepath.Join(dir, "trailing.pem")
	require.NoError(t, os.WriteFile(invalidPath, []byte("definitely-not-a-cert"), 0644))

	validPEM := newCosignCertificatePEM(t, localTestCosignCertificateOptions{
		identity:    fmt.Sprintf(localLibraryCosignIdentityTemplate, "v9.9.9"),
		oidcIssuer:  localLibraryCosignOIDCIssuer,
		notBefore:   time.Now().Add(-1 * time.Minute),
		notAfter:    time.Now().Add(10 * time.Minute),
		extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	})
	require.NoError(t, os.WriteFile(trailingPath, append(validPEM, []byte("\nTRAILING")...), 0644))

	_, err := localReadCosignCertificate(invalidPath)
	require.Error(t, err)

	_, err = localReadCosignCertificate(trailingPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "trailing data")
}

func TestLocalReadCosignCertificate_ReadErrorIncludesPath(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.pem")
	_, err := localReadCosignCertificate(missingPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), missingPath)
}

func TestLocalCertificateExtensionValue_ReportsASN1DecodeError(t *testing.T) {
	certificate := &x509.Certificate{
		Extensions: []pkix.Extension{
			{
				Id:    localLibraryCosignOIDCIssuerExtensionOID,
				Value: []byte{0xff, 0xfe},
			},
		},
	}

	_, _, err := localCertificateExtensionValue(certificate, localLibraryCosignOIDCIssuerExtensionOID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode certificate extension")
	require.Contains(t, err.Error(), "asn1")
}

func TestLocalVerifyBlobSignature_SupportsRSAAndEd25519(t *testing.T) {
	payload := []byte("blob-for-signature-verification")
	digest := sha256.Sum256(payload)

	rsaPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	rsaCertificate := newSelfSignedCertificate(t, rsaPrivateKey.Public(), rsaPrivateKey)
	rsaSignature, err := rsa.SignPKCS1v15(rand.Reader, rsaPrivateKey, crypto.SHA256, digest[:])
	require.NoError(t, err)
	require.NoError(t, localVerifyBlobSignature(rsaCertificate, payload, rsaSignature))

	edPublicKey, edPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	edCertificate := newSelfSignedCertificate(t, edPublicKey, edPrivateKey)
	edSignature := ed25519.Sign(edPrivateKey, payload)
	require.NoError(t, localVerifyBlobSignature(edCertificate, payload, edSignature))
}

func TestLocalChecksumFromSumsFile_SupportsBSDFileMarker(t *testing.T) {
	assetName := "chroma-go-local-v9.9.9-linux-amd64.tar.gz"
	sumsPath := filepath.Join(t.TempDir(), "SHA256SUMS.txt")
	require.NoError(t, os.WriteFile(sumsPath, []byte("DEADBEEF  *"+assetName+"\n"), 0644))

	checksum, err := localChecksumFromSumsFile(sumsPath, assetName)
	require.NoError(t, err)
	require.Equal(t, "deadbeef", checksum)
}

func TestLocalChecksumFromSumsFileAny_SupportsPrefixedAssetPaths(t *testing.T) {
	legacyAssetName := "chroma-go-local-v9.9.9-linux-amd64.tar.gz"
	localChromaAssetName := "local-chroma-v9.9.9-linux-amd64.tar.gz"
	sumsPath := filepath.Join(t.TempDir(), "SHA256SUMS.txt")
	line := "DEADBEEF  chroma-go-local/v9.9.9/" + localChromaAssetName + "\n"
	require.NoError(t, os.WriteFile(sumsPath, []byte(line), 0644))

	assetName, checksum, err := localChecksumFromSumsFileAny(sumsPath, []string{legacyAssetName, localChromaAssetName})
	require.NoError(t, err)
	require.Equal(t, localChromaAssetName, assetName)
	require.Equal(t, "deadbeef", checksum)
}

func TestLocalChecksumFromSumsFileAny_ReturnsOriginalCandidateName(t *testing.T) {
	localChromaAssetName := "local-chroma-v9.9.9-linux-amd64.tar.gz"
	candidateName := "./" + localChromaAssetName
	sumsPath := filepath.Join(t.TempDir(), "SHA256SUMS.txt")
	require.NoError(t, os.WriteFile(sumsPath, []byte("DEADBEEF  "+localChromaAssetName+"\n"), 0644))

	assetName, checksum, err := localChecksumFromSumsFileAny(sumsPath, []string{candidateName})
	require.NoError(t, err)
	require.Equal(t, candidateName, assetName)
	require.Equal(t, "deadbeef", checksum)
}

func TestLocalChecksumFromSumsFileAny_ReturnsNotFound(t *testing.T) {
	sumsPath := filepath.Join(t.TempDir(), "SHA256SUMS.txt")
	require.NoError(t, os.WriteFile(sumsPath, []byte("DEADBEEF  other-asset.tar.gz\n"), 0644))

	_, _, err := localChecksumFromSumsFileAny(sumsPath, []string{
		"chroma-go-local-v9.9.9-linux-amd64.tar.gz",
		"local-chroma-v9.9.9-linux-amd64.tar.gz",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "checksum entry not found for assets")
}

func TestLocalChecksumFromSumsFileAny_RejectsEmptyAssetNames(t *testing.T) {
	sumsPath := filepath.Join(t.TempDir(), "SHA256SUMS.txt")
	require.NoError(t, os.WriteFile(sumsPath, []byte("DEADBEEF  local-chroma-v9.9.9-linux-amd64.tar.gz\n"), 0644))

	tests := []struct {
		name       string
		assetNames []string
	}{
		{name: "nil", assetNames: nil},
		{name: "empty slice", assetNames: []string{}},
		{name: "blank values", assetNames: []string{"", "   ", "\n\t"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := localChecksumFromSumsFileAny(sumsPath, tc.assetNames)
			require.Error(t, err)
			require.Contains(t, err.Error(), "asset names cannot be empty")
		})
	}
}

func TestLocalChecksumFromSumsFileAny_PrefersFirstMatchingFileLine(t *testing.T) {
	legacyAssetName := "chroma-go-local-v9.9.9-linux-amd64.tar.gz"
	localChromaAssetName := "local-chroma-v9.9.9-linux-amd64.tar.gz"
	sumsPath := filepath.Join(t.TempDir(), "SHA256SUMS.txt")
	sumsBody := strings.Join([]string{
		"1111  " + localChromaAssetName,
		"2222  " + legacyAssetName,
	}, "\n") + "\n"
	require.NoError(t, os.WriteFile(sumsPath, []byte(sumsBody), 0644))

	assetName, checksum, err := localChecksumFromSumsFileAny(sumsPath, []string{legacyAssetName, localChromaAssetName})
	require.NoError(t, err)
	require.Equal(t, localChromaAssetName, assetName)
	require.Equal(t, "1111", checksum)
}

func TestLocalNormalizedChecksumAssetName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "bsd marker", input: "*local-chroma-v0.4.0-linux-amd64.tar.gz", want: "local-chroma-v0.4.0-linux-amd64.tar.gz"},
		{name: "relative path", input: "./local-chroma-v0.4.0-linux-amd64.tar.gz", want: "local-chroma-v0.4.0-linux-amd64.tar.gz"},
		{name: "windows path", input: `chroma-go-local\v0.4.0\local-chroma-v0.4.0-linux-amd64.tar.gz`, want: "local-chroma-v0.4.0-linux-amd64.tar.gz"},
		{name: "deep path", input: "prefix/nested/path/chroma-go-local-v0.4.0-linux-amd64.tar.gz", want: "chroma-go-local-v0.4.0-linux-amd64.tar.gz"},
		{name: "blank", input: "   ", want: ""},
		{name: "parent directory", input: "..", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, localNormalizedChecksumAssetName(tc.input))
		})
	}
}

func TestLocalLibraryArchiveNames_IncludeLegacyAndLocalChroma(t *testing.T) {
	archiveNames := localLibraryArchiveNames("v9.9.9", "linux-amd64")
	require.Equal(t, []string{
		"chroma-go-local-v9.9.9-linux-amd64.tar.gz",
		"local-chroma-v9.9.9-linux-amd64.tar.gz",
	}, archiveNames)
}

func TestLocalRemoveCorruptedArchive_ReportsRemoveError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission checks are not reliable on windows")
	}

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "bad-archive.tar.gz")
	require.NoError(t, os.WriteFile(archivePath, []byte("bad"), 0644))
	require.NoError(t, os.Chmod(dir, 0555))
	t.Cleanup(func() {
		_ = os.Chmod(dir, 0755)
	})

	err := localRemoveCorruptedArchive(archivePath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to remove corrupted local runtime archive")
}

func TestLocalRemoveCorruptedArchive_MissingFileReturnsNil(t *testing.T) {
	err := localRemoveCorruptedArchive(filepath.Join(t.TempDir(), "missing.tar.gz"))
	require.NoError(t, err)
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
	archiveBytes := newTarGzWithLibrary(t, "libchroma_shim.so", []byte("this payload is too large"))
	require.NoError(t, os.WriteFile(archivePath, archiveBytes, 0644))

	targetPath := filepath.Join(t.TempDir(), "libchroma_shim.so")
	err := localExtractLibraryFromTarGz(archivePath, "libchroma_shim.so", targetPath)
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

type localTestCosignCertificateOptions struct {
	identity           string
	oidcIssuer         string
	oidcIssuerRawValue bool
	notBefore          time.Time
	notAfter           time.Time
	extKeyUsage        []x509.ExtKeyUsage
}

func newCosignCertificatePEM(t *testing.T, opts localTestCosignCertificateOptions) []byte {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	certificate := newCosignCertificateWithKey(t, opts, &privateKey.PublicKey, privateKey)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw})
}

func newCosignCertificate(t *testing.T, opts localTestCosignCertificateOptions) *x509.Certificate {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return newCosignCertificateWithKey(t, opts, &privateKey.PublicKey, privateKey)
}

func newCosignCertificateWithKey(t *testing.T, opts localTestCosignCertificateOptions, publicKey any, signer any) *x509.Certificate {
	t.Helper()

	identityURI, err := url.Parse(opts.identity)
	require.NoError(t, err)
	oidcIssuerValue := []byte(opts.oidcIssuer)
	if !opts.oidcIssuerRawValue {
		oidcIssuerValue, err = asn1.Marshal(opts.oidcIssuer)
		require.NoError(t, err)
	}

	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		NotBefore:    opts.notBefore,
		NotAfter:     opts.notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  opts.extKeyUsage,
		URIs:         []*url.URL{identityURI},
		ExtraExtensions: []pkix.Extension{
			{
				Id:    localLibraryCosignOIDCIssuerExtensionOID,
				Value: oidcIssuerValue,
			},
		},
	}

	certificateDER, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, publicKey, signer)
	require.NoError(t, err)
	certificate, err := x509.ParseCertificate(certificateDER)
	require.NoError(t, err)
	return certificate
}

func newSelfSignedCertificate(t *testing.T, publicKey any, signer any) *x509.Certificate {
	t.Helper()

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		NotBefore:    time.Now().Add(-1 * time.Minute),
		NotAfter:     time.Now().Add(10 * time.Minute),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, signer)
	require.NoError(t, err)
	certificate, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return certificate
}

func newSignedChecksumArtifacts(t *testing.T, version string, checksumBody []byte) ([]byte, []byte) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	expectedIdentity := fmt.Sprintf(localLibraryCosignIdentityTemplate, version)
	identityURI, err := url.Parse(expectedIdentity)
	require.NoError(t, err)

	oidcIssuerValue, err := asn1.Marshal(localLibraryCosignOIDCIssuer)
	require.NoError(t, err)

	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-1 * time.Minute),
		NotAfter:     time.Now().Add(10 * time.Minute),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		URIs:         []*url.URL{identityURI},
		ExtraExtensions: []pkix.Extension{
			{
				Id:    localLibraryCosignOIDCIssuerExtensionOID,
				Value: oidcIssuerValue,
			},
		},
	}

	certificateDER, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)
	certificatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certificateDER,
	})
	require.NotEmpty(t, certificatePEM)

	digest := sha256.Sum256(checksumBody)
	signature, err := ecdsa.SignASN1(rand.Reader, privateKey, digest[:])
	require.NoError(t, err)

	return []byte(base64.StdEncoding.EncodeToString(signature)), certificatePEM
}
