package defaultef

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDownload(t *testing.T) {
	t.Run("Resolve runtime from custom path", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("HOME", tempDir)
		customLibPath := filepath.Join(tempDir, "custom", "libonnxruntime.dylib")
		require.NoError(t, os.MkdirAll(filepath.Dir(customLibPath), 0o755))
		require.NoError(t, os.WriteFile(customLibPath, []byte("dummy"), 0o644))
		t.Setenv("CHROMAGO_ONNX_RUNTIME_PATH", customLibPath)

		resetConfigForTesting()

		cfg := getConfig()
		err := EnsureOnnxRuntimeSharedLibrary()
		require.NoError(t, err)
		require.Equal(t, customLibPath, cfg.OnnxLibPath)
		_, err = os.Stat(cfg.OnnxLibPath)
		require.NoError(t, err)
	})

	t.Run("Custom path does not require write permissions in library directory", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("directory permission enforcement differs on windows")
		}

		tempDir := t.TempDir()
		t.Setenv("HOME", tempDir)

		readOnlyDir := filepath.Join(tempDir, "readonly")
		require.NoError(t, os.MkdirAll(readOnlyDir, 0o755))

		customLibPath := filepath.Join(readOnlyDir, "libonnxruntime.dylib")
		require.NoError(t, os.WriteFile(customLibPath, []byte("dummy"), 0o644))
		require.NoError(t, os.Chmod(readOnlyDir, 0o555))
		t.Cleanup(func() {
			_ = os.Chmod(readOnlyDir, 0o755)
		})

		t.Setenv("CHROMAGO_ONNX_RUNTIME_PATH", customLibPath)
		resetConfigForTesting()

		cfg := getConfig()
		err := EnsureOnnxRuntimeSharedLibrary()
		require.NoError(t, err)
		require.Equal(t, customLibPath, cfg.OnnxLibPath)

		lockPath := filepath.Join(readOnlyDir, ".download.lock")
		_, statErr := os.Stat(lockPath)
		require.True(t, os.IsNotExist(statErr), "custom runtime path should not create lockfiles in runtime directory")
	})

	t.Run("Model already cached", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("HOME", tempDir)

		resetConfigForTesting()

		cfg := getConfig()
		require.NoError(t, os.MkdirAll(cfg.OnnxModelCachePath, 0o755))
		require.NoError(t, os.WriteFile(cfg.OnnxModelPath, []byte("dummy-onnx"), 0o644))
		require.NoError(t, os.WriteFile(cfg.OnnxModelTokenizerConfigPath, []byte("{}"), 0o644))

		err := EnsureDefaultEmbeddingFunctionModel()
		require.NoError(t, err)
		_, err = os.Stat(cfg.OnnxModelPath)
		require.NoError(t, err)
		_, err = os.Stat(cfg.OnnxModelTokenizerConfigPath)
		require.NoError(t, err)
	})
}
