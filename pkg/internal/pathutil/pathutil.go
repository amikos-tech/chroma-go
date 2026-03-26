package pathutil

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pkg/errors"
)

// ContainsDotDot reports whether the cleaned path still contains ".." components.
func ContainsDotDot(path string) bool {
	return slices.Contains(strings.Split(filepath.ToSlash(path), "/"), "..")
}

// ValidateFilePath cleans a file path and checks for path traversal.
// Returns the cleaned path or an error if traversal is detected.
func ValidateFilePath(path string) (string, error) {
	cleaned := filepath.Clean(path)
	if ContainsDotDot(cleaned) {
		return "", errors.Errorf("file path %q contains path traversal", path)
	}
	return cleaned, nil
}

// SafePath validates that joining destPath with filename results in a path
// within destPath, preventing path traversal attacks from malicious tar entries.
func SafePath(destPath, filename string) (string, error) {
	destPath = filepath.Clean(destPath)
	targetPath := filepath.Join(destPath, filepath.Base(filename))
	if !strings.HasPrefix(targetPath, destPath+string(os.PathSeparator)) && targetPath != destPath {
		return "", errors.Errorf("invalid path: %q escapes destination directory", filename)
	}
	return targetPath, nil
}
