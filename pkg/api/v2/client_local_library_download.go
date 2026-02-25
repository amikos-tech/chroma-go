package v2

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	defaultLocalLibraryVersion = "v0.2.0"
	localLibraryModulePath     = "github.com/amikos-tech/chroma-go-local"
	localLibraryChecksumsAsset = "chroma-go-shim_SHA256SUMS.txt"
	localLibraryLockFileName   = ".download.lock"
)

var (
	localLibraryReleaseBaseURL       = "https://github.com/amikos-tech/chroma-go-local/releases/download"
	localLibraryDownloadMu           sync.Mutex
	localLibraryDownloadAttempts     = 3
	localLibraryLockWaitTimeout      = 45 * time.Second
	localLibraryLockStaleAfter       = 10 * time.Minute
	localGetenvFunc                  = os.Getenv
	localUserHomeDirFunc             = os.UserHomeDir
	localReadBuildInfoFunc           = debug.ReadBuildInfo
	localDownloadFileFunc            = localDownloadFileWithRetry
	localEnsureLibraryDownloadedFunc = ensureLocalLibraryDownloaded
	localDetectLibraryVersionFunc    = detectLocalLibraryVersion
	localDefaultLibraryCacheDirFunc  = defaultLocalLibraryCacheDir
)

type localLibraryAsset struct {
	platform        string
	archiveName     string
	libraryFileName string
}

func resolveLocalLibraryPath(cfg *localClientConfig) (string, error) {
	if cfg == nil {
		return "", errors.New("local client config cannot be nil")
	}

	if p := strings.TrimSpace(cfg.libraryPath); p != "" {
		return p, nil
	}

	if envPath := strings.TrimSpace(localGetenvFunc("CHROMA_LIB_PATH")); envPath != "" {
		return envPath, nil
	}

	if !cfg.autoDownloadLibrary {
		return "", errors.New("local runtime library path is not configured: set WithLocalLibraryPath(...), CHROMA_LIB_PATH, or enable WithLocalLibraryAutoDownload(true)")
	}

	version := normalizeLocalLibraryTag(cfg.libraryVersion)
	if version == "" {
		version = normalizeLocalLibraryTag(localDetectLibraryVersionFunc())
	}
	if version == "" {
		version = defaultLocalLibraryVersion
	}

	cacheDir := strings.TrimSpace(cfg.libraryCacheDir)
	if cacheDir == "" {
		var err error
		cacheDir, err = localDefaultLibraryCacheDirFunc()
		if err != nil {
			return "", errors.Wrap(err, "failed to determine local library cache dir")
		}
	}

	libPath, err := localEnsureLibraryDownloadedFunc(version, cacheDir)
	if err != nil {
		return "", err
	}
	return libPath, nil
}

func detectLocalLibraryVersion() string {
	buildInfo, ok := localReadBuildInfoFunc()
	if !ok || buildInfo == nil {
		return defaultLocalLibraryVersion
	}

	for _, dep := range buildInfo.Deps {
		if dep == nil || dep.Path != localLibraryModulePath {
			continue
		}
		version := dep.Version
		if dep.Replace != nil && dep.Replace.Version != "" {
			version = dep.Replace.Version
		}
		version = normalizeLocalLibraryTag(version)
		if version != "" {
			return version
		}
	}
	return defaultLocalLibraryVersion
}

func defaultLocalLibraryCacheDir() (string, error) {
	homeDir, err := localUserHomeDirFunc()
	if err != nil {
		return "", errors.Wrap(err, "failed to resolve home directory")
	}
	if strings.TrimSpace(homeDir) == "" {
		return "", errors.New("home directory is empty")
	}
	return filepath.Join(homeDir, ".cache", "chroma", "local_shim"), nil
}

func ensureLocalLibraryDownloaded(version, cacheDir string) (libPath string, retErr error) {
	version = normalizeLocalLibraryTag(version)
	if version == "" {
		return "", errors.New("local library version cannot be empty")
	}
	if strings.TrimSpace(cacheDir) == "" {
		return "", errors.New("local library cache dir cannot be empty")
	}

	asset, err := localLibraryAssetForRuntime(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}

	targetDir := filepath.Join(cacheDir, version, asset.platform)
	targetLibraryPath := filepath.Join(targetDir, asset.libraryFileName)
	exists, err := localFileExistsNonEmpty(targetLibraryPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat local runtime library at %s", targetLibraryPath)
	}
	if exists {
		return targetLibraryPath, nil
	}

	localLibraryDownloadMu.Lock()
	defer localLibraryDownloadMu.Unlock()

	exists, err = localFileExistsNonEmpty(targetLibraryPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat local runtime library at %s", targetLibraryPath)
	}
	if exists {
		return targetLibraryPath, nil
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", errors.Wrap(err, "failed to create local library cache dir")
	}

	lockFile, err := localAcquireDownloadLock(filepath.Join(targetDir, localLibraryLockFileName))
	if err != nil {
		return "", err
	}
	defer func() {
		if releaseErr := localReleaseDownloadLock(lockFile); releaseErr != nil {
			wrapped := errors.Wrapf(releaseErr, "failed to release download lock %s", lockFile.Name())
			if retErr != nil {
				retErr = stderrors.Join(retErr, wrapped)
			} else {
				retErr = wrapped
			}
		}
	}()

	exists, err = localFileExistsNonEmpty(targetLibraryPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat local runtime library at %s", targetLibraryPath)
	}
	if exists {
		return targetLibraryPath, nil
	}

	checksumsPath := filepath.Join(targetDir, localLibraryChecksumsAsset)
	checksumsURL := fmt.Sprintf("%s/%s/%s", localLibraryReleaseBaseURL, version, localLibraryChecksumsAsset)
	if err := localDownloadFileFunc(checksumsPath, checksumsURL); err != nil {
		return "", errors.Wrap(err, "failed to download local library checksums")
	}

	expectedChecksum, err := localChecksumFromSumsFile(checksumsPath, asset.archiveName)
	if err != nil {
		return "", errors.Wrap(err, "failed to resolve local library checksum")
	}

	archivePath := filepath.Join(targetDir, asset.archiveName)
	exists, err = localFileExistsNonEmpty(archivePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat local runtime archive at %s", archivePath)
	}
	if exists {
		if err := localVerifyFileChecksum(archivePath, expectedChecksum); err != nil {
			_ = os.Remove(archivePath)
		}
	}
	exists, err = localFileExistsNonEmpty(archivePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat local runtime archive at %s", archivePath)
	}
	if !exists {
		archiveURL := fmt.Sprintf("%s/%s/%s", localLibraryReleaseBaseURL, version, asset.archiveName)
		if err := localDownloadFileFunc(archivePath, archiveURL); err != nil {
			return "", errors.Wrap(err, "failed to download local library archive")
		}
	}

	if err := localVerifyFileChecksum(archivePath, expectedChecksum); err != nil {
		_ = os.Remove(archivePath)
		return "", errors.Wrap(err, "local library archive checksum verification failed")
	}
	if err := localVerifyTarGzFile(archivePath); err != nil {
		_ = os.Remove(archivePath)
		return "", errors.Wrap(err, "local library archive verification failed")
	}

	tempLibraryPath := targetLibraryPath + ".tmp"
	_ = os.Remove(tempLibraryPath)
	if err := localExtractLibraryFromTarGz(archivePath, asset.libraryFileName, tempLibraryPath); err != nil {
		_ = os.Remove(tempLibraryPath)
		return "", errors.Wrap(err, "failed to extract local runtime library")
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tempLibraryPath, 0755); err != nil {
			_ = os.Remove(tempLibraryPath)
			return "", errors.Wrap(err, "failed to set permissions on local runtime library")
		}
	}
	_ = os.Remove(targetLibraryPath)
	if err := os.Rename(tempLibraryPath, targetLibraryPath); err != nil {
		_ = os.Remove(tempLibraryPath)
		return "", errors.Wrap(err, "failed to finalize local runtime library")
	}
	exists, err = localFileExistsNonEmpty(targetLibraryPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to stat extracted local runtime library at %s", targetLibraryPath)
	}
	if !exists {
		return "", errors.Errorf("local runtime library not found after extraction: %s", targetLibraryPath)
	}

	return targetLibraryPath, nil
}

func localLibraryAssetForRuntime(goos, goarch string) (localLibraryAsset, error) {
	var platformOS string
	switch goos {
	case "linux":
		platformOS = "linux"
	case "darwin":
		platformOS = "macos"
	case "windows":
		platformOS = "windows"
	default:
		return localLibraryAsset{}, errors.Errorf("unsupported OS for local runtime download: %s", goos)
	}

	switch platformOS {
	case "linux", "windows":
		if goarch != "amd64" {
			return localLibraryAsset{}, errors.Errorf("unsupported architecture for %s local runtime download: %s", goos, goarch)
		}
	case "macos":
		if goarch != "arm64" {
			return localLibraryAsset{}, errors.Errorf("unsupported architecture for %s local runtime download: %s", goos, goarch)
		}
	}

	platform := platformOS + "-" + goarch
	asset := localLibraryAsset{
		platform:    platform,
		archiveName: "chroma-go-shim-" + platform + ".tar.gz",
	}
	switch goos {
	case "darwin":
		asset.libraryFileName = "libchroma_go_shim.dylib"
	case "windows":
		asset.libraryFileName = "chroma_go_shim.dll"
	default:
		asset.libraryFileName = "libchroma_go_shim.so"
	}
	return asset, nil
}

func normalizeLocalLibraryTag(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || version == "(devel)" {
		return ""
	}
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

func localAcquireDownloadLock(lockPath string) (*os.File, error) {
	lockDir := filepath.Dir(lockPath)
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return nil, errors.Wrap(err, "failed to create lock directory")
	}

	deadline := time.Now().Add(localLibraryLockWaitTimeout)
	for {
		lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			_, _ = fmt.Fprintf(lockFile, "%d", os.Getpid())
			_ = lockFile.Sync()
			return lockFile, nil
		}
		if !os.IsExist(err) {
			return nil, errors.Wrap(err, "failed to create lock file")
		}

		if info, statErr := os.Stat(lockPath); statErr == nil {
			if time.Since(info.ModTime()) > localLibraryLockStaleAfter {
				if removeErr := os.Remove(lockPath); removeErr != nil && !os.IsNotExist(removeErr) {
					return nil, errors.Wrapf(removeErr, "failed to remove stale lock file %s", lockPath)
				}
				continue
			}
		}

		if time.Now().After(deadline) {
			return nil, errors.Errorf("timeout waiting for lock: %s", lockPath)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func localReleaseDownloadLock(lockFile *os.File) error {
	if lockFile == nil {
		return nil
	}
	lockPath := lockFile.Name()
	var errs []error
	if closeErr := lockFile.Close(); closeErr != nil {
		errs = append(errs, errors.Wrap(closeErr, "failed to close download lock file"))
	}
	if removeErr := os.Remove(lockPath); removeErr != nil && !os.IsNotExist(removeErr) {
		errs = append(errs, errors.Wrapf(removeErr, "failed to remove download lock file %s", lockPath))
	}
	if len(errs) > 0 {
		return stderrors.Join(errs...)
	}
	return nil
}

func localChecksumFromSumsFile(sumsFilePath, assetName string) (string, error) {
	f, err := os.Open(sumsFilePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to open checksum file")
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimRight(scanner.Text(), "\r"))
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[1] == assetName {
			return strings.ToLower(fields[0]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", errors.Wrap(err, "failed to read checksum file")
	}
	return "", errors.Errorf("checksum entry not found for asset %s", assetName)
}

func localVerifyFileChecksum(filePath, expectedChecksum string) error {
	expectedChecksum = strings.TrimSpace(strings.ToLower(expectedChecksum))
	if expectedChecksum == "" {
		return errors.New("expected checksum cannot be empty")
	}

	f, err := os.Open(filePath)
	if err != nil {
		return errors.Wrap(err, "failed to open file for checksum verification")
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return errors.Wrap(err, "failed to hash downloaded file")
	}

	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	if actualChecksum != expectedChecksum {
		return errors.Errorf("checksum mismatch for %s: expected %s, got %s", filePath, expectedChecksum, actualChecksum)
	}
	return nil
}

func localDownloadFileWithRetry(filePath, url string) error {
	var lastErr error
	for attempt := 1; attempt <= localLibraryDownloadAttempts; attempt++ {
		if err := localDownloadFile(filePath, url); err != nil {
			lastErr = err
			if attempt < localLibraryDownloadAttempts {
				time.Sleep(time.Duration(attempt) * time.Second)
			}
			continue
		}
		return nil
	}
	return errors.Wrap(lastErr, "download failed after retries")
}

func localDownloadFile(filePath, url string) error {
	client := &http.Client{
		Timeout: 10 * time.Minute,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return errors.Wrap(err, "failed to make HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected response %s for URL %s", resp.Status, url)
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return errors.Wrap(err, "failed to create destination directory")
	}

	tempPath := filePath + ".download-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	out, err := os.Create(tempPath)
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}

	written, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tempPath)
		return errors.Wrap(copyErr, "failed to copy HTTP response")
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return errors.Wrap(closeErr, "failed to close temp file")
	}
	if resp.ContentLength > 0 && written != resp.ContentLength {
		_ = os.Remove(tempPath)
		return errors.Errorf("download incomplete: expected %d bytes, got %d bytes", resp.ContentLength, written)
	}

	_ = os.Remove(filePath)
	if err := os.Rename(tempPath, filePath); err != nil {
		_ = os.Remove(tempPath)
		return errors.Wrap(err, "failed to finalize downloaded file")
	}
	return nil
}

func localVerifyTarGzFile(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return errors.Wrap(err, "could not open archive for verification")
	}
	defer f.Close()

	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		return errors.Wrap(err, "invalid gzip archive")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	if _, err := tarReader.Next(); err != nil {
		return errors.Wrap(err, "invalid tar archive")
	}
	return nil
}

func localExtractLibraryFromTarGz(archivePath, libraryFileName, destinationPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to open archive")
	}
	defer f.Close()

	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		return errors.Wrap(err, "failed to read gzip archive")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed to read tar entry")
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(header.Name) != libraryFileName {
			continue
		}

		out, err := os.Create(destinationPath)
		if err != nil {
			return errors.Wrap(err, "failed to create extracted library file")
		}

		_, copyErr := io.Copy(out, tarReader)
		syncErr := out.Sync()
		closeErr := out.Close()
		if copyErr != nil {
			_ = os.Remove(destinationPath)
			return errors.Wrap(copyErr, "failed to extract library from archive")
		}
		if syncErr != nil {
			_ = os.Remove(destinationPath)
			return errors.Wrap(syncErr, "failed to sync extracted library")
		}
		if closeErr != nil {
			_ = os.Remove(destinationPath)
			return errors.Wrap(closeErr, "failed to close extracted library")
		}
		return nil
	}
	return errors.Errorf("library %s not found in archive", libraryFileName)
}

func localFileExistsNonEmpty(filePath string) (bool, error) {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !info.IsDir() && info.Size() > 0, nil
}
