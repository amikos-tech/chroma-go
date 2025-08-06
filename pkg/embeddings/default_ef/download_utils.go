package defaultef

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"

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

func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "failed to make HTTP request")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected response %s for URL %s: %v", resp.Status, url, string(respBody))
	}

	out, err := os.Create(filepath)
	if err != nil {
		return errors.Wrapf(err, "failed to create file: %s", filepath)
	}
	defer out.Close()

	_, err = io.Copy(out, bytes.NewReader(respBody))
	if err != nil {
		return errors.Wrapf(err, "failed to copy file contents: %s", filepath)
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
				return errors.Wrap(err, "could not copy file data")
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
		}
	}

	if targetFile != "" {
		return errors.Errorf("file %s not found in the archive", targetFile)
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

		fmt.Printf("Downloading onnxruntime from GitHub: %s\n", url)
		// TODO integrity check
		if _, onnxInitErr = os.Stat(targetArchive); os.IsNotExist(onnxInitErr) {
			onnxInitErr = downloadFile(targetArchive, url)
			if onnxInitErr != nil {
				return errors.Wrap(onnxInitErr, "failed to download onnxruntime.tgz")
			}
		}
	}
	targetFile := "onnxruntime-" + cos + "-" + carch + "-" + LibOnnxRuntimeVersion + "/lib/libonnxruntime." + LibOnnxRuntimeVersion + "." + getExtensionForOs()
	if cos == "linux" {
		targetFile = "onnxruntime-" + cos + "-" + carch + "-" + LibOnnxRuntimeVersion + "/lib/libonnxruntime." + getExtensionForOs() + "." + LibOnnxRuntimeVersion
	}
	// fmt.Println("Extracting onnxruntime shared library..." + onnxLibPath)
	onnxInitErr = extractSpecificFile(targetArchive, targetFile, onnxCacheDir)
	if onnxInitErr != nil {
		return errors.Wrapf(onnxInitErr, "could not extract onnxruntime shared library")
	}

	if cos == "linux" {
		wantedTargetFile := filepath.Join(onnxCacheDir, "libonnxruntime."+LibOnnxRuntimeVersion+"."+getExtensionForOs())
		onnxInitErr = os.Rename(filepath.Join(onnxCacheDir, "libonnxruntime."+getExtensionForOs()+"."+LibOnnxRuntimeVersion), wantedTargetFile)
		if onnxInitErr != nil {
			return errors.Wrapf(onnxInitErr, "could not rename extracted file to %s", wantedTargetFile)
		}
	}

	onnxInitErr = os.RemoveAll(targetArchive)
	if onnxInitErr != nil {
		return errors.Wrapf(onnxInitErr, "could not remove temporary archive: %s", targetArchive)
	}

	return onnxInitErr
}

func EnsureLibTokenizersSharedLibrary() error {
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

	err := os.RemoveAll(targetArchive)
	if err != nil {
		return errors.Wrapf(err, "could not remove temporary archive: %s", targetArchive)
	}
	return nil
}

func EnsureDefaultEmbeddingFunctionModel() error {
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
		// fmt.Println("Downloading onnx model from S3...")
		// TODO integrity check
		if err := downloadFile(targetArchive, onnxModelDownloadEndpoint); err != nil {
			return errors.Wrap(err, "failed to download onnx model")
		}
	}
	// fmt.Println("Extracting onnx model..." + onnxModelCachePath)
	if err := extractSpecificFile(targetArchive, "", onnxModelCachePath); err != nil {
		// fmt.Println("Error:", err)
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
