package defaultef

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func setOfflineRuntimePathOrSkip(t *testing.T) {
	t.Helper()

	if p := strings.TrimSpace(os.Getenv("CHROMAGO_ONNX_RUNTIME_PATH")); p != "" {
		if info, err := os.Stat(p); err == nil && !info.IsDir() && info.Size() > 0 {
			return
		}
	}

	home := resolveHomeDir()
	patterns := []string{
		filepath.Join(home, ".cache", "chroma", "shared", "onnxruntime", "libonnxruntime.*"),
		filepath.Join(home, ".cache", "chroma", "shared", "onnxruntime", "onnxruntime-*", "lib", "libonnxruntime*.dylib"),
		filepath.Join(home, ".cache", "chroma", "shared", "onnxruntime", "onnxruntime-*", "lib", "libonnxruntime*.so"),
		filepath.Join(home, ".cache", "chroma", "shared", "onnxruntime", "onnxruntime-*", "lib", "onnxruntime*.dll"),
		filepath.Join(home, "Library", "Caches", "onnx-purego", "onnxruntime", "onnxruntime-*", "lib", "libonnxruntime*.dylib"),
		filepath.Join(home, "Library", "Caches", "onnx-purego", "onnxruntime", "onnxruntime-*", "lib", "libonnxruntime*.so"),
		filepath.Join(home, "Library", "Caches", "onnx-purego", "onnxruntime", "onnxruntime-*", "lib", "onnxruntime*.dll"),
		filepath.Join(home, ".cache", "onnx-purego", "onnxruntime", "onnxruntime-*", "lib", "libonnxruntime*.dylib"),
		filepath.Join(home, ".cache", "onnx-purego", "onnxruntime", "onnxruntime-*", "lib", "libonnxruntime*.so"),
		filepath.Join(home, ".cache", "onnx-purego", "onnxruntime", "onnxruntime-*", "lib", "onnxruntime*.dll"),
	}

	candidates := make([]string, 0, 16)
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil || info.IsDir() || info.Size() == 0 {
				continue
			}
			candidates = append(candidates, match)
		}
	}

	if len(candidates) == 0 {
		t.Skip("no local ONNX Runtime library found; set CHROMAGO_ONNX_RUNTIME_PATH or provide GH_TOKEN/GITHUB_TOKEN for bootstrap metadata lookups")
		return
	}

	sort.Strings(candidates)
	t.Setenv("CHROMAGO_ONNX_RUNTIME_PATH", candidates[len(candidates)-1])
	resetConfigForTesting()
}
