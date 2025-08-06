package defaultef

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

var libCacheDir = filepath.Join(os.Getenv("HOME"), ChromaCacheDir)
var onnxCacheDir = filepath.Join(libCacheDir, "shared", "onnxruntime")
var onnxLibPath = filepath.Join(onnxCacheDir, "libonnxruntime."+LibOnnxRuntimeVersion+"."+getExtensionForOs())

var libTokenizersCacheDir = filepath.Join(libCacheDir, "shared", "libtokenizers")
var libTokenizersLibPath = filepath.Join(libTokenizersCacheDir, "libtokenizers."+getExtensionForOs())
var onnxModelsCachePath = filepath.Join(libCacheDir, "onnx_models")
var onnxModelCachePath = filepath.Join(onnxModelsCachePath, "all-MiniLM-L6-v2/onnx")
var onnxModelPath = filepath.Join(onnxModelCachePath, "model.onnx")
var onnxModelTokenizerConfigPath = filepath.Join(onnxModelCachePath, "tokenizer.json")

func lockFile(path string) (*os.File, error) {
	lockPath := filepath.Join(path, ".lock")
	err := os.MkdirAll(filepath.Dir(lockPath), 0755)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create directory for lock file: %s", filepath.Dir(lockPath))
	}
	// Try to create lock file with exclusive access
	for i := 0; i < 30; i++ { // Wait up to 30 seconds

		lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			// Write PID to lock file for debugging
			fmt.Fprintf(lockFile, "%d", os.Getpid())
			_ = lockFile.Sync()
			return lockFile, nil
		}

		if !os.IsExist(err) {
			return nil, err
		}

		// Check if the process holding the lock is still alive
		if isLockStale(lockPath) {
			os.Remove(lockPath) // Remove stale lock
			continue
		}

		time.Sleep(1 * time.Second)
	}

	return nil, errors.New("timeout waiting for file lock")
}

func unlockFile(lockFile *os.File) error {
	if lockFile == nil {
		return nil
	}
	lockPath := lockFile.Name()
	lockFile.Close()
	return os.Remove(lockPath)
}

func isLockStale(lockPath string) bool {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return true // If we can't read it, assume stale
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return true
	}

	// Check if process exists (Unix-specific)
	process, err := os.FindProcess(pid)
	if err != nil {
		return true
	}

	// Send signal 0 to check if process is alive
	err = process.Signal(syscall.Signal(0))
	return err != nil
}

func downloadFile(filepath string, url string) error {

	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "failed to make HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected response %s for URL %s", resp.Status, url)
	}

	// Check Content-Length if available
	contentLength := resp.ContentLength
	// if contentLength > 0 {
	//	fmt.Printf("Expected download size: %d bytes\n", contentLength)
	//}

	out, err := os.Create(filepath)
	if err != nil {
		return errors.Wrapf(err, "failed to create file: %s", filepath)
	}
	defer out.Close()

	// Copy directly from response body, don't buffer everything in memory
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to copy file contents: %s", filepath)
	}

	// fmt.Printf("Downloaded %d bytes\n", written)

	// Verify size if we know the expected size
	if contentLength > 0 && written != contentLength {
		return errors.Errorf("download incomplete: expected %d bytes, got %d bytes", contentLength, written)
	}

	// Explicitly sync to disk
	if err := out.Sync(); err != nil {
		return errors.Wrapf(err, "failed to sync file to disk: %s", filepath)
	}

	// Verify file exists and has expected size
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return errors.Wrapf(err, "failed to stat downloaded file: %s", filepath)
	}

	if fileInfo.Size() != written {
		return errors.Errorf("file size mismatch after download: expected %d, got %d", written, fileInfo.Size())
	}

	// fmt.Printf("Download completed and verified: %s (%d bytes)\n", filepath, fileInfo.Size())
	return nil
}

func verifyTarGzFile(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return errors.Wrapf(err, "could not open file for verification: %s", filepath)
	}
	defer file.Close()

	// Try to read the gzip header
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return errors.Wrap(err, "invalid gzip file")
	}
	defer gzipReader.Close()

	// Try to read the tar header
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

func extractSpecificFile(tarGzPath, targetFile, destPath string) error {
	// Open the .tar.gz file
	f, err := os.Open(tarGzPath)
	if err != nil {
		return errors.Wrapf(err, "could not open tar.gz file: %s", tarGzPath)
	}
	defer f.Close()

	// Create a gzip reader
	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		return errors.Wrap(err, "could not create gzip reader")
	}
	defer gzipReader.Close()

	// Create a tar reader
	tarReader := tar.NewReader(gzipReader)

	// Iterate through the files in the tar archive
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break // End of archive
		}

		if err != nil {
			return errors.Wrap(err, "could not read tar header")
		}

		// Check if this is the file we're looking for
		if header.Name == targetFile {
			// Create the destination file
			outFile, err := os.Create(filepath.Join(destPath, filepath.Base(targetFile)))
			if err != nil {
				return errors.Wrapf(err, "could not create output file: %s", filepath.Join(destPath, filepath.Base(targetFile)))
			}
			defer outFile.Close()
			// Copy the file data from the tar archive to the destination file
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return errors.Wrapf(err, "could not copy file data to output file: %s", filepath.Join(destPath, filepath.Base(targetFile)))
			}
			if err := outFile.Sync(); err != nil {
				return errors.Wrapf(err, "could not sync output file to disk: %s", filepath.Join(destPath, filepath.Base(targetFile)))
			}
			return nil // Successfully extracted the file
		}
		if targetFile == "" {
			// Create the destination file
			outFile, err := os.Create(filepath.Join(destPath, filepath.Base(header.Name)))
			if err != nil {
				return errors.Wrapf(err, "could not create output file: %s", filepath.Join(destPath, filepath.Base(header.Name)))
			}
			defer outFile.Close()

			// Copy the file data from the tar archive to the destination file
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return errors.Wrap(err, "could not copy file data")
			}
			if err := outFile.Sync(); err != nil {
				return errors.Wrapf(err, "could not sync output file to disk: %s", filepath.Join(destPath, filepath.Base(targetFile)))
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

var (
	onnxInitErr error
	onnxMu      sync.Mutex
)

func EnsureOnnxRuntimeSharedLibrary() error {
	onnxMu.Lock()
	defer onnxMu.Unlock()
	lockFile, err := lockFile(onnxCacheDir)
	if err != nil {
		return errors.Wrap(err, "failed to acquire lock for onnx download")
	}
	defer func() {
		_ = unlockFile(lockFile)
	}()

	cos, carch := getOSAndArch()
	if carch == "amd64" {
		carch = "x64"
	}
	if cos == "darwin" {
		cos = "osx"
		if carch == "x64" {
			carch = "x86_64"
		}
	}

	downloadAndExtractNeeded := false
	if _, onnxInitErr = os.Stat(onnxLibPath); os.IsNotExist(onnxInitErr) {
		downloadAndExtractNeeded = true
		onnxInitErr = os.MkdirAll(onnxCacheDir, 0755)
		if onnxInitErr != nil {
			return errors.Wrap(onnxInitErr, "failed to create onnx cache")
		}
	}
	if !downloadAndExtractNeeded {
		return nil
	}
	targetArchive := filepath.Join(onnxCacheDir, "onnxruntime-"+cos+"-"+carch+"-"+LibOnnxRuntimeVersion+".tgz")
	if _, onnxInitErr = os.Stat(onnxLibPath); os.IsNotExist(onnxInitErr) {
		// Download the library
		url := "https://github.com/microsoft/onnxruntime/releases/download/v" + LibOnnxRuntimeVersion + "/onnxruntime-" + cos + "-" + carch + "-" + LibOnnxRuntimeVersion + ".tgz"
		// TODO integrity check
		if _, onnxInitErr = os.Stat(targetArchive); os.IsNotExist(onnxInitErr) {
			onnxInitErr = downloadFile(targetArchive, url)
			if onnxInitErr != nil {
				return errors.Wrap(onnxInitErr, "failed to download onnxruntime.tgz")
			}
			if _, err := os.Stat(targetArchive); err != nil {
				return errors.Wrap(err, "downloaded archive not found after download")
			}
			if err := verifyTarGzFile(targetArchive); err != nil {
				return errors.Wrap(err, "failed to verify downloaded onnxruntime archive")
			}
		}
	}
	targetFile := "onnxruntime-" + cos + "-" + carch + "-" + LibOnnxRuntimeVersion + "/lib/libonnxruntime." + LibOnnxRuntimeVersion + "." + getExtensionForOs()
	if cos == "linux" {
		targetFile = "onnxruntime-" + cos + "-" + carch + "-" + LibOnnxRuntimeVersion + "/lib/libonnxruntime." + getExtensionForOs() + "." + LibOnnxRuntimeVersion
	}
	onnxInitErr = extractSpecificFile(targetArchive, targetFile, onnxCacheDir)
	if onnxInitErr != nil {
		return errors.Wrapf(onnxInitErr, "could not extract onnxruntime shared library")
	}

	if cos == "linux" {
		// wantedTargetFile := filepath.Join(onnxCacheDir, "libonnxruntime."+LibOnnxRuntimeVersion+"."+getExtensionForOs())
		onnxInitErr = os.Rename(filepath.Join(onnxCacheDir, "libonnxruntime."+getExtensionForOs()+"."+LibOnnxRuntimeVersion), onnxLibPath)
		if onnxInitErr != nil {
			return errors.Wrapf(onnxInitErr, "could not rename extracted file to %s", onnxLibPath)
		}
	}

	if _, err := os.Stat(onnxLibPath); err != nil {
		return errors.Wrapf(err, "extracted file not found at expected location: %s", onnxLibPath)
	}

	onnxInitErr = os.RemoveAll(targetArchive)
	if onnxInitErr != nil {
		return errors.Wrapf(onnxInitErr, "could not remove temporary archive: %s", targetArchive)
	}

	return onnxInitErr
}

func EnsureLibTokenizersSharedLibrary() error {
	lockFile, err := lockFile(libTokenizersCacheDir)
	if err != nil {
		return errors.Wrap(err, "failed to acquire lock for onnx download")
	}
	defer func() {
		_ = unlockFile(lockFile)
	}()

	cos, carch := getOSAndArch()
	downloadAndExtractNeeded := false
	if _, err := os.Stat(libTokenizersLibPath); os.IsNotExist(err) {
		downloadAndExtractNeeded = true
		if err := os.MkdirAll(libTokenizersCacheDir, 0755); err != nil {
			return errors.Wrap(err, "failed to create libtokenizers cache")
		}
	}
	if !downloadAndExtractNeeded {
		return nil
	}
	targetArchive := filepath.Join(libTokenizersCacheDir, "libtokenizers."+cos+"-"+carch+".tar.gz")
	if _, err := os.Stat(libTokenizersLibPath); os.IsNotExist(err) {
		// Download the library
		url := "https://github.com/amikos-tech/tokenizers/releases/download/v" + LibTokenizersVersion + "/libtokenizers." + cos + "-" + carch + ".tar.gz"

		// fmt.Println("Downloading libtokenizers from GitHub...")
		// TODO integrity check
		if _, err := os.Stat(targetArchive); os.IsNotExist(err) {
			if err := downloadFile(targetArchive, url); err != nil {
				return errors.Wrap(err, "failed to download libtokenizers.tar.gz")
			}
		}
	}
	targetFile := "libtokenizers." + getExtensionForOs()
	// fmt.Println("Extracting libtokenizers shared library..." + onnxLibPath)
	if err := extractSpecificFile(targetArchive, targetFile, libTokenizersCacheDir); err != nil {
		// fmt.Println("Error:", err)
		return errors.Wrapf(err, "could not extract libtokenizers shared library")
	}

	err = os.RemoveAll(targetArchive)
	if err != nil {
		return errors.Wrapf(err, "could not remove temporary archive: %s", targetArchive)
	}
	return nil
}

func EnsureDefaultEmbeddingFunctionModel() error {

	lockFile, err := lockFile(onnxModelsCachePath)
	if err != nil {
		return errors.Wrap(err, "failed to acquire lock for onnx download")
	}
	defer func() {
		_ = unlockFile(lockFile)
	}()

	downloadAndExtractNeeded := false
	if _, err := os.Stat(onnxModelCachePath); os.IsNotExist(err) {
		downloadAndExtractNeeded = true
		if err := os.MkdirAll(onnxModelCachePath, 0755); err != nil {
			return errors.Wrap(err, "failed to create onnx model cache")
		}
	}
	if !downloadAndExtractNeeded {
		return nil
	}
	targetArchive := filepath.Join(onnxModelsCachePath, "onnx.tar.gz")
	if _, err := os.Stat(targetArchive); os.IsNotExist(err) {
		// TODO integrity check
		if err := downloadFile(targetArchive, onnxModelDownloadEndpoint); err != nil {
			return errors.Wrap(err, "failed to download onnx model")
		}
	}
	if err := extractSpecificFile(targetArchive, "", onnxModelCachePath); err != nil {
		return errors.Wrapf(err, "could not extract onnx model")
	}

	// err := os.RemoveAll(targetArchive)
	// if err != nil {
	//	return err
	//}
	return nil
}

// LibOnnxRuntimeVersion is the version of the ONNX Runtime library to download
func getExtensionForOs() string {
	cos := runtime.GOOS
	if cos == "darwin" {
		return "dylib"
	}
	if cos == "windows" {
		return "dll"
	}
	return "so" // assume Linux default
}
