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
		libCacheDir = filepath.Join(t.TempDir(), "lib_cache")
		fmt.Println(onnxLibPath)
		err := os.RemoveAll(onnxLibPath)
		require.NoError(t, err)
		err = EnsureOnnxRuntimeSharedLibrary()
		require.NoError(t, err)
	})
	t.Run("Download Tokenizers", func(t *testing.T) {
		libCacheDir = filepath.Join(t.TempDir(), "lib_cache")
		err := os.RemoveAll(libTokenizersLibPath)
		require.NoError(t, err)
		err = EnsureLibTokenizersSharedLibrary()
		require.NoError(t, err)
	})
	t.Run("Download Model", func(t *testing.T) {
		libCacheDir = filepath.Join(t.TempDir(), "lib_cache")
		err := os.RemoveAll(onnxModelCachePath)
		require.NoError(t, err)
		err = EnsureDefaultEmbeddingFunctionModel()
		require.NoError(t, err)
	})
}
