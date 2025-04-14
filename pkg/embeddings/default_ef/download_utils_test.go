package defaultef

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDownload(t *testing.T) {
	t.Run("Download", func(t *testing.T) {
		onnxLibPath = filepath.Join(t.TempDir(), "libonnxruntime."+LibOnnxRuntimeVersion+"."+getExtensionForOs())
		fmt.Println(onnxLibPath)
		err := os.RemoveAll(onnxLibPath)
		require.NoError(t, err)
		err = EnsureOnnxRuntimeSharedLibrary()
		require.NoError(t, err)
	})
	t.Run("Download Tokenizers", func(t *testing.T) {
		libTokenizersLibPath = filepath.Join(t.TempDir(), "libtokenizers."+LibTokenizersVersion+"."+getExtensionForOs())
		err := os.RemoveAll(libTokenizersLibPath)
		require.NoError(t, err)
		err = EnsureLibTokenizersSharedLibrary()
		require.NoError(t, err)
	})
	t.Run("Download Model", func(t *testing.T) {
		onnxModelCachePath = filepath.Join(t.TempDir(), "onnx_model")
		err := os.RemoveAll(onnxModelCachePath)
		require.NoError(t, err)
		err = EnsureDefaultEmbeddingFunctionModel()
		require.NoError(t, err)
	})
}
