package defaultef

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Download_Onnx(t *testing.T) {
	fmt.Println(onnxLibPath)
	err := os.RemoveAll(onnxLibPath)
	require.NoError(t, err)
	err = EnsureOnnxRuntimeSharedLibrary()
	require.NoError(t, err)
}

func Test_Download_Tokenizers(t *testing.T) {
	err := os.RemoveAll(libTokenizersLibPath)
	require.NoError(t, err)
	err = EnsureLibTokenizersSharedLibrary()
	require.NoError(t, err)
}

func Test_Download_Model(t *testing.T) {
	err := os.RemoveAll(onnxModelCachePath)
	require.NoError(t, err)
	err = EnsureDefaultEmbeddingFunctionModel()
	require.NoError(t, err)
}
