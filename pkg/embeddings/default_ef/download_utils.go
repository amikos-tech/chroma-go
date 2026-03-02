package defaultef

import (
	"archive/tar"
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
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	ort "github.com/amikos-tech/pure-onnx/ort"
)

// Known SHA256 checksum for the ONNX model archive.
// This ensures the downloaded model has not been tampered with.
// To update: download the file and run `shasum -a 256 onnx.tar.gz`
const onnxModelSHA256 = "913d7300ceae3b2dbc2c50d1de4baacab4be7b9380491c27fab7418616a16ec3"

const (
	defaultEFDownloadLockWaitTimeout       = 45 * time.Second
	defaultEFDownloadLockStaleAfter        = 10 * time.Minute
	defaultEFDownloadLockHeartbeatInterval = 30 * time.Second
)

func verifyFileChecksum(filepath string, expectedChecksum string) error {
	if expectedChecksum == "" {
		return nil
	}

	file, err := os.Open(filepath)
	if err != nil {
		return errors.Wrapf(err, "failed to open file for checksum verification: %s", filepath)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return errors.Wrapf(err, "failed to compute checksum for: %s", filepath)
	}

	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	if actualChecksum != expectedChecksum {
		return errors.Errorf("checksum mismatch for %s: expected %s, got %s", filepath, expectedChecksum, actualChecksum)
	}

	return nil
}

func downloadFile(filepath string, url string) error {
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

	contentLength := resp.ContentLength

	out, err := os.Create(filepath)
	if err != nil {
		return errors.Wrapf(err, "failed to create file: %s", filepath)
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to copy file contents: %s", filepath)
	}

	if contentLength > 0 && written != contentLength {
		return errors.Errorf("download incomplete: expected %d bytes, got %d bytes", contentLength, written)
	}

	if err := out.Sync(); err != nil {
		return errors.Wrapf(err, "failed to sync file to disk: %s", filepath)
	}

	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return errors.Wrapf(err, "failed to stat downloaded file: %s", filepath)
	}

	if fileInfo.Size() != written {
		return errors.Errorf("file size mismatch after download: expected %d, got %d", written, fileInfo.Size())
	}

	return nil
}

func verifyTarGzFile(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return errors.Wrapf(err, "could not open file for verification: %s", filepath)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return errors.Wrap(err, "invalid gzip file")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	_, err = tarReader.Next()
	if err != nil {
		return errors.Wrap(err, "invalid tar file or corrupt archive")
	}

	return nil
}

func getOSAndArch() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}

// safePath validates that joining destPath with filename results in a path
// within destPath, preventing path traversal attacks from malicious tar entries.
func safePath(destPath, filename string) (string, error) {
	destPath = filepath.Clean(destPath)
	targetPath := filepath.Join(destPath, filepath.Base(filename))
	if !strings.HasPrefix(targetPath, destPath+string(os.PathSeparator)) && targetPath != destPath {
		return "", errors.Errorf("invalid path: %q escapes destination directory", filename)
	}
	return targetPath, nil
}

func extractSpecificFile(tarGzPath, targetFile, destPath string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return errors.Wrapf(err, "could not open tar.gz file: %s", tarGzPath)
	}
	defer f.Close()

	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		return errors.Wrap(err, "could not create gzip reader")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return errors.Wrap(err, "could not read tar header")
		}

		if header.Name == targetFile {
			outPath, err := safePath(destPath, targetFile)
			if err != nil {
				return err
			}
			outFile, err := os.Create(outPath)
			if err != nil {
				return errors.Wrapf(err, "could not create output file: %s", outPath)
			}
			defer outFile.Close()
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return errors.Wrapf(err, "could not copy file data to output file: %s", outPath)
			}
			if err := outFile.Sync(); err != nil {
				return errors.Wrapf(err, "could not sync output file to disk: %s", outPath)
			}
			return nil
		}
		if targetFile == "" {
			outPath, err := safePath(destPath, header.Name)
			if err != nil {
				return err
			}
			outFile, err := os.Create(outPath)
			if err != nil {
				return errors.Wrapf(err, "could not create output file: %s", outPath)
			}
			defer outFile.Close()
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return errors.Wrap(err, "could not copy file data")
			}
			if err := outFile.Sync(); err != nil {
				return errors.Wrapf(err, "could not sync output file to disk: %s", outPath)
			}
		}
	}

	if targetFile != "" {
		expectedPath := filepath.Join(destPath, filepath.Base(targetFile))
		if _, err := os.Stat(expectedPath); err != nil {
			return errors.Wrapf(err, "extracted file not found at expected location: %s", expectedPath)
		}
	}
	return nil
}

var onnxMu sync.Mutex

func EnsureOnnxRuntimeSharedLibrary() error {
	cfg := getConfig()

	onnxMu.Lock()
	defer onnxMu.Unlock()

	if err := os.MkdirAll(cfg.OnnxCacheDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create onnx cache")
	}

	lockFile, err := defaultEFAcquireDownloadLock(filepath.Join(cfg.OnnxCacheDir, ".download.lock"))
	if err != nil {
		return errors.Wrap(err, "failed to acquire lock for onnx bootstrap")
	}
	defer func() {
		_ = defaultEFReleaseDownloadLock(lockFile)
	}()
	stopHeartbeat := defaultEFStartDownloadLockHeartbeat(lockFile)
	defer func() {
		_ = stopHeartbeat()
	}()

	bootstrapOpts := []ort.BootstrapOption{
		ort.WithBootstrapCacheDir(cfg.OnnxCacheDir),
	}
	if cfg.LibOnnxRuntimeVersion == "custom" {
		bootstrapOpts = append(bootstrapOpts, ort.WithBootstrapLibraryPath(cfg.OnnxLibPath))
	} else {
		bootstrapOpts = append(bootstrapOpts, ort.WithBootstrapVersion(cfg.LibOnnxRuntimeVersion))
	}

	resolvedPath, err := ort.EnsureOnnxRuntimeSharedLibrary(bootstrapOpts...)
	if err != nil {
		return errors.Wrap(err, "failed to resolve onnxruntime shared library via bootstrap")
	}

	cfg.OnnxLibPath = resolvedPath
	return nil
}

func EnsureDefaultEmbeddingFunctionModel() error {
	cfg := getConfig()

	lockFile, err := defaultEFAcquireDownloadLock(filepath.Join(cfg.OnnxModelsCachePath, ".download.lock"))
	if err != nil {
		return errors.Wrap(err, "failed to acquire lock for onnx model download")
	}
	defer func() {
		_ = defaultEFReleaseDownloadLock(lockFile)
	}()
	stopHeartbeat := defaultEFStartDownloadLockHeartbeat(lockFile)
	defer func() {
		_ = stopHeartbeat()
	}()

	modelExists, err := defaultEFFileExistsNonEmpty(cfg.OnnxModelPath)
	if err != nil {
		return errors.Wrap(err, "failed to check onnx model file")
	}
	tokenizerExists, err := defaultEFFileExistsNonEmpty(cfg.OnnxModelTokenizerConfigPath)
	if err != nil {
		return errors.Wrap(err, "failed to check tokenizer config file")
	}
	if modelExists && tokenizerExists {
		return nil
	}

	if err := os.MkdirAll(cfg.OnnxModelCachePath, 0755); err != nil {
		return errors.Wrap(err, "failed to create onnx model cache")
	}

	targetArchive := filepath.Join(cfg.OnnxModelsCachePath, "onnx.tar.gz")
	archiveExists, err := defaultEFFileExistsNonEmpty(targetArchive)
	if err != nil {
		return errors.Wrap(err, "failed to check onnx model archive")
	}
	if archiveExists {
		if err := verifyFileChecksum(targetArchive, onnxModelSHA256); err != nil {
			_ = os.Remove(targetArchive)
			archiveExists = false
		}
	}
	if !archiveExists {
		if err := downloadFile(targetArchive, onnxModelDownloadEndpoint); err != nil {
			return errors.Wrap(err, "failed to download onnx model")
		}
		if err := verifyFileChecksum(targetArchive, onnxModelSHA256); err != nil {
			_ = os.Remove(targetArchive)
			return errors.Wrap(err, "onnx model integrity check failed")
		}
	}
	if err := extractSpecificFile(targetArchive, "", cfg.OnnxModelCachePath); err != nil {
		return errors.Wrapf(err, "could not extract onnx model")
	}

	modelExists, err = defaultEFFileExistsNonEmpty(cfg.OnnxModelPath)
	if err != nil {
		return errors.Wrap(err, "failed to verify extracted onnx model")
	}
	tokenizerExists, err = defaultEFFileExistsNonEmpty(cfg.OnnxModelTokenizerConfigPath)
	if err != nil {
		return errors.Wrap(err, "failed to verify extracted tokenizer config")
	}
	if !modelExists || !tokenizerExists {
		return errors.Errorf(
			"onnx model extraction incomplete: model_exists=%t tokenizer_exists=%t",
			modelExists,
			tokenizerExists,
		)
	}

	return nil
}

func defaultEFShouldEvictStaleLock(lockPath string, initialInfo os.FileInfo) (bool, error) {
	if initialInfo == nil || time.Since(initialInfo.ModTime()) <= defaultEFDownloadLockStaleAfter {
		return false, nil
	}

	currentInfo, err := os.Stat(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrapf(err, "failed to stat stale lock file %s", lockPath)
	}

	if !os.SameFile(initialInfo, currentInfo) {
		return false, nil
	}
	if !currentInfo.ModTime().Equal(initialInfo.ModTime()) || currentInfo.Size() != initialInfo.Size() {
		return false, nil
	}
	if time.Since(currentInfo.ModTime()) <= defaultEFDownloadLockStaleAfter {
		return false, nil
	}
	return true, nil
}

func defaultEFAcquireDownloadLock(lockPath string) (*os.File, error) {
	lockDir := filepath.Dir(lockPath)
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return nil, errors.Wrap(err, "failed to create lock directory")
	}

	deadline := time.Now().Add(defaultEFDownloadLockWaitTimeout)
	for {
		lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			_, _ = fmt.Fprintf(lockFile, "%d", os.Getpid())
			_ = lockFile.Sync()
			return lockFile, nil
		}
		if !os.IsExist(err) {
			return nil, errors.Wrap(err, "failed to create lock file")
		}

		if info, statErr := os.Stat(lockPath); statErr == nil {
			evictStale, staleErr := defaultEFShouldEvictStaleLock(lockPath, info)
			if staleErr != nil {
				return nil, staleErr
			}
			if evictStale {
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

func defaultEFReleaseDownloadLock(lockFile *os.File) error {
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

func defaultEFStartDownloadLockHeartbeat(lockFile *os.File) func() error {
	if lockFile == nil || defaultEFDownloadLockHeartbeatInterval <= 0 {
		return func() error { return nil }
	}

	lockPath := lockFile.Name()
	stopCh := make(chan struct{})
	doneCh := make(chan error, 1)
	var stopOnce sync.Once
	var stopErr error
	go func() {
		ticker := time.NewTicker(defaultEFDownloadLockHeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				now := time.Now()
				if err := os.Chtimes(lockPath, now, now); err != nil {
					if os.IsNotExist(err) {
						doneCh <- nil
					} else {
						doneCh <- errors.Wrapf(err, "failed to refresh download lock file %s", lockPath)
					}
					return
				}
			case <-stopCh:
				doneCh <- nil
				return
			}
		}
	}()

	return func() error {
		stopOnce.Do(func() {
			close(stopCh)
			stopErr = <-doneCh
		})
		return stopErr
	}
}

func defaultEFFileExistsNonEmpty(filePath string) (bool, error) {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !info.IsDir() && info.Size() > 0, nil
}
