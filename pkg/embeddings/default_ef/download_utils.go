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
		return fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
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
		return fmt.Errorf("could not open tar.gz file: %v", err)
	}
	defer f.Close()

	// Create a gzip reader
	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("could not create gzip reader: %v", err)
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
			return fmt.Errorf("could not read tar header: %v", err)
		}

		// Check if this is the file we're looking for
		if header.Name == targetFile {
			// Create the destination file
			outFile, err := os.Create(filepath.Join(destPath, filepath.Base(targetFile)))
			if err != nil {
				return fmt.Errorf("could not create output file: %v", err)
			}
			defer outFile.Close()

			// Copy the file data from the tar archive to the destination file
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("could not copy file data: %v", err)
			}

			fmt.Printf("Successfully extracted %s to %s\n", targetFile, destPath)
			return nil // Successfully extracted the file
		}
		if targetFile == "" {
			// Create the destination file
			outFile, err := os.Create(filepath.Join(destPath, filepath.Base(header.Name)))
			if err != nil {
				return fmt.Errorf("could not create output file: %v", err)
			}
			defer outFile.Close()

			// Copy the file data from the tar archive to the destination file
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("could not copy file data: %v", err)
			}
		}
	}

	if targetFile != "" {
		return fmt.Errorf("file %s not found in the archive", targetFile)
	}
	return nil
}

func EnsureOnnxRuntimeSharedLibrary() error {
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
	if _, err := os.Stat(onnxLibPath); os.IsNotExist(err) {
		downloadAndExtractNeeded = true
		if err := os.MkdirAll(onnxCacheDir, 0755); err != nil {
			return err
		}
	}
	if !downloadAndExtractNeeded {
		return nil
	}
	targetArchive := filepath.Join(onnxCacheDir, "onnxruntime-"+cos+"-"+carch+"-"+LibOnnxRuntimeVersion+".tgz")
	if _, err := os.Stat(onnxLibPath); os.IsNotExist(err) {
		// Download the library
		url := "https://github.com/microsoft/onnxruntime/releases/download/v" + LibOnnxRuntimeVersion + "/onnxruntime-" + cos + "-" + carch + "-" + LibOnnxRuntimeVersion + ".tgz"

		fmt.Println("Downloading onnxruntime from GitHub...")
		// TODO integrity check
		if _, err := os.Stat(targetArchive); os.IsNotExist(err) {
			if err := downloadFile(targetArchive, url); err != nil {
				return err
			}
		}
	}
	targetFile := "onnxruntime-" + cos + "-" + carch + "-" + LibOnnxRuntimeVersion + "/lib/libonnxruntime." + LibOnnxRuntimeVersion + "." + getExtensionForOs()
	if cos == "linux" {
		targetFile = "onnxruntime-" + cos + "-" + carch + "-" + LibOnnxRuntimeVersion + "/lib/libonnxruntime." + getExtensionForOs() + "." + LibOnnxRuntimeVersion
	}
	fmt.Println("Extracting onnxruntime shared library..." + onnxLibPath)
	if err := extractSpecificFile(targetArchive, targetFile, onnxCacheDir); err != nil {
		fmt.Println("Error:", err)
	}

	if cos == "linux" {
		wantedTargetFile := filepath.Join(onnxCacheDir, "libonnxruntime."+LibOnnxRuntimeVersion+"."+getExtensionForOs())
		err := os.Rename(filepath.Join(onnxCacheDir, "libonnxruntime."+getExtensionForOs()+"."+LibOnnxRuntimeVersion), wantedTargetFile)
		if err != nil {
			return err
		}
	}

	err := os.RemoveAll(targetArchive)
	if err != nil {
		return err
	}
	return nil
}

func EnsureLibTokenizersSharedLibrary() error {
	cos, carch := getOSAndArch()
	downloadAndExtractNeeded := false
	if _, err := os.Stat(libTokenizersLibPath); os.IsNotExist(err) {
		downloadAndExtractNeeded = true
		if err := os.MkdirAll(libTokenizersCacheDir, 0755); err != nil {
			return err
		}
	}
	if !downloadAndExtractNeeded {
		return nil
	}
	targetArchive := filepath.Join(libTokenizersCacheDir, "libtokenizers."+cos+"-"+carch+".tar.gz")
	if _, err := os.Stat(libTokenizersLibPath); os.IsNotExist(err) {
		// Download the library
		url := "https://github.com/amikos-tech/tokenizers/releases/download/v" + LibTokenizersVersion + "/libtokenizers." + cos + "-" + carch + ".tar.gz"

		fmt.Println("Downloading libtokenizers from GitHub...")
		// TODO integrity check
		if _, err := os.Stat(targetArchive); os.IsNotExist(err) {
			if err := downloadFile(targetArchive, url); err != nil {
				return err
			}
		}
	}
	targetFile := "libtokenizers." + getExtensionForOs()
	fmt.Println("Extracting libtokenizers shared library..." + onnxLibPath)
	if err := extractSpecificFile(targetArchive, targetFile, libTokenizersCacheDir); err != nil {
		fmt.Println("Error:", err)
	}

	err := os.RemoveAll(targetArchive)
	if err != nil {
		return err
	}
	return nil
}

func EnsureDefaultEmbeddingFunctionModel() error {
	downloadAndExtractNeeded := false
	if _, err := os.Stat(onnxModelCachePath); os.IsNotExist(err) {
		downloadAndExtractNeeded = true
		if err := os.MkdirAll(onnxModelCachePath, 0755); err != nil {
			return err
		}
	}
	if !downloadAndExtractNeeded {
		return nil
	}
	targetArchive := filepath.Join(onnxModelsCachePath, "onnx.tar.gz")
	if _, err := os.Stat(targetArchive); os.IsNotExist(err) {
		fmt.Println("Downloading onnx model from S3...")
		// TODO integrity check
		if err := downloadFile(targetArchive, onnxModelDownloadEndpoint); err != nil {
			return err
		}
	}
	fmt.Println("Extracting onnx model..." + onnxModelCachePath)
	if err := extractSpecificFile(targetArchive, "", onnxModelCachePath); err != nil {
		fmt.Println("Error:", err)
		return err
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
