package defaultef

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
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

func TestEnsureOnnxRuntimeSharedLibrary_NoConfigFieldMutationRace(t *testing.T) {
	setOfflineRuntimePathOrSkip(t)
	resetConfigForTesting()
	cfg := getConfig()

	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 300; i++ {
			if err := EnsureOnnxRuntimeSharedLibrary(); err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		total := 0
		for i := 0; i < 200000; i++ {
			total += len(cfg.OnnxLibPath)
		}
		if total == -1 {
			t.Log(total)
		}
	}()

	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}
}

func TestExtractSpecificFile_ExtractAllSkipsNonRegularTarEntries(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "model.tar.gz")
	require.NoError(t, os.WriteFile(archivePath, buildTarGzWithDirAndFile(t), 0o644))

	destDir := t.TempDir()
	require.NoError(t, extractSpecificFile(archivePath, "", destDir))

	_, err := os.Stat(filepath.Join(destDir, "onnx"))
	require.True(t, os.IsNotExist(err), "directory tar entry should not be materialized as a regular file")

	content, err := os.ReadFile(filepath.Join(destDir, "model.onnx"))
	require.NoError(t, err)
	require.Equal(t, "model-bytes", string(content))
}

func TestDownloadFile_CleansUpPartialDestinationOnFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "32")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("short"))
	}))
	defer ts.Close()

	destDir := t.TempDir()
	destPath := filepath.Join(destDir, "partial.bin")

	err := downloadFile(destPath, ts.URL)
	require.Error(t, err)

	_, statErr := os.Stat(destPath)
	require.True(t, os.IsNotExist(statErr), "partial destination file should be removed on failed download")

	tmpMatches, globErr := filepath.Glob(filepath.Join(destDir, "partial.bin.tmp-*"))
	require.NoError(t, globErr)
	require.Empty(t, tmpMatches, "temporary download files should be cleaned up")
}

func buildTarGzWithDirAndFile(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	require.NoError(t, tarWriter.WriteHeader(&tar.Header{
		Name:     "onnx/",
		Mode:     0o755,
		Typeflag: tar.TypeDir,
	}))

	payload := []byte("model-bytes")
	require.NoError(t, tarWriter.WriteHeader(&tar.Header{
		Name:     "onnx/model.onnx",
		Mode:     0o644,
		Typeflag: tar.TypeReg,
		Size:     int64(len(payload)),
	}))
	_, err := tarWriter.Write(payload)
	require.NoError(t, err)

	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzWriter.Close())
	return buf.Bytes()
}
