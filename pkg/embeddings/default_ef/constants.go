package defaultef

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const (
	defaultLibOnnxRuntimeVersion = "1.23.1"
	onnxModelDownloadEndpoint    = "https://chroma-onnx-models.s3.amazonaws.com/all-MiniLM-L6-v2/onnx.tar.gz"
	ChromaCacheDir               = ".cache/chroma/"
)

// Config holds all computed configuration paths for ONNX Runtime
type Config struct {
	LibOnnxRuntimeVersion string
	LibCacheDir           string
	OnnxCacheDir          string
	OnnxLibPath           string

	OnnxModelsCachePath          string
	OnnxModelCachePath           string
	OnnxModelPath                string
	OnnxModelTokenizerConfigPath string
}

var (
	configOnce sync.Once
	config     *Config
)

// getConfig returns the singleton configuration instance,
// initializing it on first call using environment variables
func getConfig() *Config {
	configOnce.Do(func() {
		config = initializeConfig()
	})
	return config
}

// initializeConfig creates a new Config by reading environment variables
// and computing all derived paths
func initializeConfig() *Config {
	homeDir := resolveHomeDir()
	libCacheDir := filepath.Join(homeDir, ChromaCacheDir)
	onnxModelsCachePath := filepath.Join(libCacheDir, "onnx_models")
	onnxModelCachePath := filepath.Join(onnxModelsCachePath, "all-MiniLM-L6-v2/onnx")
	onnxModelPath := filepath.Join(onnxModelCachePath, "model.onnx")
	onnxModelTokenizerConfigPath := filepath.Join(onnxModelCachePath, "tokenizer.json")

	// Priority 1: Check for explicit path to ONNX Runtime library
	if customPath := strings.TrimSpace(os.Getenv("CHROMAGO_ONNX_RUNTIME_PATH")); customPath != "" {
		// User provided explicit path to library file
		// We still need to compute model paths, but use custom lib path

		return &Config{
			LibOnnxRuntimeVersion:        "custom", // marker for custom path
			LibCacheDir:                  libCacheDir,
			OnnxCacheDir:                 filepath.Dir(customPath), // not used for custom path
			OnnxLibPath:                  customPath,
			OnnxModelsCachePath:          onnxModelsCachePath,
			OnnxModelCachePath:           onnxModelCachePath,
			OnnxModelPath:                onnxModelPath,
			OnnxModelTokenizerConfigPath: onnxModelTokenizerConfigPath,
		}
	}

	// Priority 2: Use version-based auto-download
	version := defaultLibOnnxRuntimeVersion
	if v := os.Getenv("CHROMAGO_ONNX_RUNTIME_VERSION"); v != "" {
		// Basic validation: non-empty and reasonable length
		if len(v) > 0 && len(v) < 100 {
			version = v
		}
	}

	// Compute all paths based on the version
	onnxCacheDir := filepath.Join(libCacheDir, "shared", "onnxruntime")
	onnxLibPath := filepath.Join(onnxCacheDir, "libonnxruntime."+getExtensionForOs())

	return &Config{
		LibOnnxRuntimeVersion:        version,
		LibCacheDir:                  libCacheDir,
		OnnxCacheDir:                 onnxCacheDir,
		OnnxLibPath:                  onnxLibPath,
		OnnxModelsCachePath:          onnxModelsCachePath,
		OnnxModelCachePath:           onnxModelCachePath,
		OnnxModelPath:                onnxModelPath,
		OnnxModelTokenizerConfigPath: onnxModelTokenizerConfigPath,
	}
}

// getExtensionForOs returns the shared library extension for the current OS
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

func resolveHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(homeDir) != "" {
		return homeDir
	}
	if envHome := strings.TrimSpace(os.Getenv("HOME")); envHome != "" {
		return envHome
	}
	if tmpDir := strings.TrimSpace(os.TempDir()); tmpDir != "" {
		return tmpDir
	}
	return string(os.PathSeparator)
}
