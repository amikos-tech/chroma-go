package pathutil

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pkg/errors"
)

// containsDotDot reports whether the cleaned path still contains ".." components.
func containsDotDot(path string) bool {
	return slices.Contains(strings.Split(filepath.ToSlash(path), "/"), "..")
}

// ValidateFilePath cleans a file path and rejects relative ".." traversal.
// It does NOT sandbox absolute paths: an input like "/a/b/../../etc/passwd"
// cleans to "/etc/passwd" and is accepted. Callers needing confinement to a
// specific directory should use SafePath instead.
func ValidateFilePath(path string) (string, error) {
	if path == "" {
		return "", errors.New("file path cannot be empty")
	}
	cleaned := filepath.Clean(path)
	if containsDotDot(cleaned) {
		return "", errors.Errorf("file path %q contains path traversal", path)
	}
	return cleaned, nil
}

// SafePath validates that joining destPath with filename results in a path
// within destPath, preventing path traversal attacks from malicious tar entries.
func SafePath(destPath, filename string) (string, error) {
	if destPath == "" {
		return "", errors.New("destination path cannot be empty")
	}
	base := filepath.Base(filename)
	if base == "." || base == string(os.PathSeparator) {
		return "", errors.Errorf("invalid filename: %q", filename)
	}
	destPath = filepath.Clean(destPath)
	targetPath := filepath.Join(destPath, base)
	if destPath == "/" {
		if !strings.HasPrefix(targetPath, "/") {
			return "", errors.Errorf("invalid path: %q escapes destination directory", filename)
		}
	} else if !strings.HasPrefix(targetPath, destPath+string(os.PathSeparator)) {
		return "", errors.Errorf("invalid path: %q escapes destination directory", filename)
	}
	return targetPath, nil
}
