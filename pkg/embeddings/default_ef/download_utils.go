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
	"sync"
	"time"

	ort "github.com/amikos-tech/pure-onnx/ort"
	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/internal/pathutil"
)

// Known SHA256 checksum for the ONNX model archive.
// This ensures the downloaded model has not been tampered with.
// To update: download the file and run `shasum -a 256 onnx.tar.gz`
const onnxModelSHA256 = "913d7300ceae3b2dbc2c50d1de4baacab4be7b9380491c27fab7418616a16ec3"

const (
	defaultEFDownloadLockWaitTimeout       = 2 * time.Minute
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

func downloadFile(destinationPath string, url string) (err error) {
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

	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
		return errors.Wrapf(err, "failed to create parent directory for: %s", destinationPath)
	}

	out, err := os.CreateTemp(filepath.Dir(destinationPath), filepath.Base(destinationPath)+".tmp-*")
	if err != nil {
		return errors.Wrapf(err, "failed to create temporary file for: %s", destinationPath)
	}
	tempPath := out.Name()
	closed := false
	committed := false
	defer func() {
		if !closed {
			closeErr := out.Close()
			if closeErr != nil && err == nil {
				err = errors.Wrapf(closeErr, "failed to close temporary file: %s", tempPath)
			}
		}
		if !committed {
			_ = os.Remove(tempPath)
		}
	}()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to copy file contents to temporary file: %s", tempPath)
	}

	if contentLength > 0 && written != contentLength {
		return errors.Errorf("download incomplete: expected %d bytes, got %d bytes", contentLength, written)
	}

	if err := out.Sync(); err != nil {
		return errors.Wrapf(err, "failed to sync temporary file to disk: %s", tempPath)
	}

	fileInfo, err := os.Stat(tempPath)
	if err != nil {
		return errors.Wrapf(err, "failed to stat temporary downloaded file: %s", tempPath)
	}

	if fileInfo.Size() != written {
		return errors.Errorf("file size mismatch after download: expected %d, got %d", written, fileInfo.Size())
	}

	if err := out.Close(); err != nil {
		closed = true
		return errors.Wrapf(err, "failed to close temporary file before rename: %s", tempPath)
	}
	closed = true

	if err := os.Rename(tempPath, destinationPath); err != nil {
		// On Windows os.Rename fails if destination exists; retry after removing it.
		if removeErr := os.Remove(destinationPath); removeErr != nil && !os.IsNotExist(removeErr) {
			return stderrors.Join(
				errors.Wrapf(err, "failed to replace destination file: %s", destinationPath),
				errors.Wrapf(removeErr, "failed to remove existing destination file: %s", destinationPath),
			)
		}
		if err := os.Rename(tempPath, destinationPath); err != nil {
			return errors.Wrapf(err, "failed to move temporary file %s to %s", tempPath, destinationPath)
		}
	}
	committed = true

	return nil
}

func getOSAndArch() (string, string) {
	return runtime.GOOS, runtime.GOARCH
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
			if !isRegularTarFile(header) {
				return errors.Errorf("tar entry %q is not a regular file", targetFile)
			}
			outPath, err := pathutil.SafePath(destPath, targetFile)
			if err != nil {
				return err
			}
			if err := writeTarEntryToFile(outPath, tarReader); err != nil {
				return err
			}
			return nil
		}
		if targetFile == "" {
			if !isRegularTarFile(header) {
				continue
			}
			outPath, err := pathutil.SafePath(destPath, header.Name)
			if err != nil {
				return err
			}
			if err := writeTarEntryToFile(outPath, tarReader); err != nil {
				return err
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

func isRegularTarFile(header *tar.Header) bool {
	return header.Typeflag == tar.TypeReg || header.Typeflag == 0
}

func writeTarEntryToFile(outPath string, tarReader *tar.Reader) (err error) {
	outFile, err := os.Create(outPath)
	if err != nil {
		return errors.Wrapf(err, "could not create output file: %s", outPath)
	}
	defer func() {
		closeErr := outFile.Close()
		if closeErr != nil && err == nil {
			err = errors.Wrapf(closeErr, "could not close output file: %s", outPath)
		}
	}()

	if _, err := io.Copy(outFile, tarReader); err != nil {
		return errors.Wrapf(err, "could not copy file data to output file: %s", outPath)
	}
	if err := outFile.Sync(); err != nil {
		return errors.Wrapf(err, "could not sync output file to disk: %s", outPath)
	}
	return nil
}

var onnxMu sync.Mutex

func EnsureOnnxRuntimeSharedLibrary() error {
	cfg := getConfig()

	onnxMu.Lock()
	defer onnxMu.Unlock()

	bootstrapOpts := make([]ort.BootstrapOption, 0, 2)
	if cfg.LibOnnxRuntimeVersion == "custom" {
		// Custom shared library paths do not need cache directory writes.
		bootstrapOpts = append(bootstrapOpts, ort.WithBootstrapLibraryPath(cfg.OnnxLibPath))
	} else {
		if err := os.MkdirAll(cfg.OnnxCacheDir, 0755); err != nil {
			return errors.Wrap(err, "failed to create onnx cache")
		}
		bootstrapOpts = append(
			bootstrapOpts,
			ort.WithBootstrapCacheDir(cfg.OnnxCacheDir),
			ort.WithBootstrapVersion(cfg.LibOnnxRuntimeVersion),
		)
	}

	if _, err := ort.EnsureOnnxRuntimeSharedLibrary(bootstrapOpts...); err != nil {
		return errors.Wrap(err, "failed to resolve onnxruntime shared library via bootstrap")
	}
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
