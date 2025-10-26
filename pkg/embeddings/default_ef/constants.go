package defaultef

import (
	"os"
	"path/filepath"
)

const (
	defaultLibOnnxRuntimeVersion = "1.21.0"
	onnxModelDownloadEndpoint    = "https://chroma-onnx-models.s3.amazonaws.com/all-MiniLM-L6-v2/onnx.tar.gz"
	ChromaCacheDir               = ".cache/chroma/"
)

var (
	libOnnxRuntimeVersion string
	libCacheDir           string
	onnxCacheDir          string
	onnxLibPath           string

	onnxModelsCachePath          string
	onnxModelCachePath           string
	onnxModelPath                string
	onnxModelTokenizerConfigPath string
)

func init() {
	if v, ok := os.LookupEnv("CHROMAGO_ONNX_RUNTIME_VERSION"); ok {
		libOnnxRuntimeVersion = v
	} else {
		libOnnxRuntimeVersion = defaultLibOnnxRuntimeVersion
	}

	libCacheDir = filepath.Join(os.Getenv("HOME"), ChromaCacheDir)
	onnxCacheDir = filepath.Join(libCacheDir, "shared", "onnxruntime")
	onnxLibPath = filepath.Join(onnxCacheDir, "libonnxruntime."+libOnnxRuntimeVersion+"."+getExtensionForOs())

	onnxModelsCachePath = filepath.Join(libCacheDir, "onnx_models")
	onnxModelCachePath = filepath.Join(onnxModelsCachePath, "all-MiniLM-L6-v2/onnx")
	onnxModelPath = filepath.Join(onnxModelCachePath, "model.onnx")
	onnxModelTokenizerConfigPath = filepath.Join(onnxModelCachePath, "tokenizer.json")
}
