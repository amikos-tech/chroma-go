package defaultef

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultEF_BootstrapSmoke(t *testing.T) {
	if os.Getenv("RUN_DEFAULT_EF_BOOTSTRAP_SMOKE") != "1" {
		t.Skip("set RUN_DEFAULT_EF_BOOTSTRAP_SMOKE=1 to run default_ef bootstrap smoke test")
	}

	// Isolate cache/model writes for CI runs.
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)
	require.NoError(t, configureOfflineBundleForBootstrapSmoke(t, tempHome))
	resetConfigForTesting()
	t.Cleanup(resetConfigForTesting)

	ef, closeEF, err := NewDefaultEmbeddingFunction()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = closeEF()
	})

	embeddings, err := ef.EmbedDocuments(context.Background(), []string{"default_ef runtime smoke"})
	require.NoError(t, err)
	require.Len(t, embeddings, 1)
	require.Equal(t, 384, embeddings[0].Len())
}

func configureOfflineBundleForBootstrapSmoke(t *testing.T, home string) error {
	bundleHome := strings.TrimSpace(os.Getenv("CHROMA_OFFLINE_BUNDLE_HOME"))
	if bundleHome == "" {
		return nil
	}

	if _, err := os.Stat(bundleHome); err != nil {
		return fmt.Errorf("invalid CHROMA_OFFLINE_BUNDLE_HOME: %w", err)
	}

	if err := setBundledEnvIfUnset(t, "CHROMA_LIB_PATH", func() (string, error) {
		return bundledLocalShimLibraryPath(bundleHome)
	}); err != nil {
		return err
	}

	if err := setBundledEnvIfUnset(t, "TOKENIZERS_LIB_PATH", func() (string, error) {
		return bundledTokenizerLibraryPath(bundleHome)
	}); err != nil {
		return err
	}

	if err := setBundledEnvIfUnset(t, "CHROMAGO_ONNX_RUNTIME_PATH", func() (string, error) {
		return bundledOnnxRuntimeLibraryPath(bundleHome)
	}); err != nil {
		return err
	}

	return copyBundledOnnxModel(bundleHome, home)
}

func setBundledEnvIfUnset(t *testing.T, name string, resolveValue func() (string, error)) error {
	if strings.TrimSpace(os.Getenv(name)) != "" {
		return nil
	}

	value, err := resolveValue()
	if err != nil {
		return err
	}
	if value == "" {
		return fmt.Errorf("CHROMA_OFFLINE_BUNDLE_HOME did not provide value for %s", name)
	}
	if err := ensureExistingBundleFile(name, value); err != nil {
		return err
	}

	t.Setenv(name, value)
	return nil
}

func ensureExistingBundleFile(name, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s path %s is unavailable: %w", name, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s path %s is not a file", name, path)
	}
	if info.Size() <= 0 {
		return fmt.Errorf("%s path %s is empty", name, path)
	}
	return nil
}

func bundledLocalShimLibraryPath(bundleHome string) (string, error) {
	switch runtime.GOOS {
	case "linux":
		if runtime.GOARCH != "amd64" {
			return "", fmt.Errorf("unsupported linux architecture for bundled local shim: %s", runtime.GOARCH)
		}
		return filepath.Join(bundleHome, "local-shim", "linux-amd64", "libchroma_shim.so"), nil
	case "darwin":
		if runtime.GOARCH != "arm64" {
			return "", fmt.Errorf("unsupported darwin architecture for bundled local shim: %s", runtime.GOARCH)
		}
		return filepath.Join(bundleHome, "local-shim", "darwin-arm64", "libchroma_shim.dylib"), nil
	case "windows":
		if runtime.GOARCH != "amd64" {
			return "", fmt.Errorf("unsupported windows architecture for bundled local shim: %s", runtime.GOARCH)
		}
		return filepath.Join(bundleHome, "local-shim", "windows-amd64", "chroma_shim.dll"), nil
	default:
		return "", fmt.Errorf("unsupported os for bundled local shim: %s", runtime.GOOS)
	}
}

func bundledTokenizerLibraryPath(bundleHome string) (string, error) {
	switch runtime.GOOS {
	case "linux":
		return filepath.Join(bundleHome, "tokenizers", fmt.Sprintf("linux-%s", runtime.GOARCH), "libtokenizers.so"), nil
	case "darwin":
		if runtime.GOARCH != "arm64" {
			return "", fmt.Errorf("unsupported darwin architecture for bundled tokenizers: %s", runtime.GOARCH)
		}
		return filepath.Join(bundleHome, "tokenizers", "darwin-arm64", "libtokenizers.dylib"), nil
	case "windows":
		if runtime.GOARCH != "amd64" {
			return "", fmt.Errorf("unsupported windows architecture for bundled tokenizers: %s", runtime.GOARCH)
		}
		return filepath.Join(bundleHome, "tokenizers", "windows-amd64", "tokenizers.dll"), nil
	default:
		return "", fmt.Errorf("unsupported os for bundled tokenizers: %s", runtime.GOOS)
	}
}

func bundledOnnxRuntimeLibraryPath(bundleHome string) (string, error) {
	dir := filepath.Join(bundleHome, "onnx-runtime", runtime.GOOS+"-"+runtime.GOARCH)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("invalid bundled ONNX runtime directory: %w", err)
	}

	candidates := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if !strings.Contains(name, "onnx") {
			continue
		}
		switch runtime.GOOS {
		case "darwin":
			if strings.HasSuffix(name, ".dylib") {
				candidates = append(candidates, filepath.Join(dir, entry.Name()))
			}
		case "windows":
			if strings.HasSuffix(name, ".dll") {
				candidates = append(candidates, filepath.Join(dir, entry.Name()))
			}
		default:
			if strings.Contains(name, ".so") {
				candidates = append(candidates, filepath.Join(dir, entry.Name()))
			}
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no ONNX runtime library found in %s", dir)
	}
	sort.Strings(candidates)
	return candidates[len(candidates)-1], nil
}

func copyBundledOnnxModel(bundleHome, home string) error {
	source := filepath.Join(bundleHome, "onnx-models", "all-MiniLM-L6-v2", "onnx")
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("offline bundle model path not found at %s: %w", source, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("offline bundle model path %s is not a directory", source)
	}

	target := filepath.Join(home, ".cache", "chroma", "onnx_models", "all-MiniLM-L6-v2", "onnx")
	return copyDirectoryContents(source, target)
}

func copyDirectoryContents(sourceDir, targetDir string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return err
	}
	for _, entry := range entries {
		source := filepath.Join(sourceDir, entry.Name())
		target := filepath.Join(targetDir, entry.Name())
		if entry.IsDir() {
			if err := copyDirectoryContents(source, target); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(source, target); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(sourcePath, targetPath string) error {
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}
	if sourceInfo.IsDir() {
		return fmt.Errorf("expected file while copying bundled model: %s", sourcePath)
	}

	in, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		return err
	}
	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
