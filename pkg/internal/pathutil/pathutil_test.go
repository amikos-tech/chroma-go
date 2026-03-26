package pathutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainsDotDotHelper(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"traversal at start", "../etc/passwd", true},
		{"traversal in middle", "foo/../bar", true},
		{"clean absolute path", "/usr/local/file.png", false},
		{"triple dot directory", "/path/to/.../file", false},
		{"dotdot prefix in segment", "/path/..foo/file", false},
		{"empty string", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, containsDotDot(tt.path))
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	t.Run("clean path returns cleaned", func(t *testing.T) {
		cleaned, err := ValidateFilePath("/usr/local/file.png")
		require.NoError(t, err)
		assert.Equal(t, "/usr/local/file.png", cleaned)
	})

	t.Run("traversal path returns error", func(t *testing.T) {
		_, err := ValidateFilePath("../etc/passwd")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "contains path traversal")
	})

	t.Run("relative path gets cleaned", func(t *testing.T) {
		cleaned, err := ValidateFilePath("./some/path/file.txt")
		require.NoError(t, err)
		assert.Equal(t, "some/path/file.txt", cleaned)
	})

	t.Run("multi-level relative traversal returns error", func(t *testing.T) {
		_, err := ValidateFilePath("../../etc/passwd")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "contains path traversal")
	})

	t.Run("empty path returns error", func(t *testing.T) {
		_, err := ValidateFilePath("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file path cannot be empty")
	})

	t.Run("absolute path with embedded traversal gets cleaned", func(t *testing.T) {
		cleaned, err := ValidateFilePath("/usr/local/bin/../../../etc/passwd")
		require.NoError(t, err)
		assert.Equal(t, "/etc/passwd", cleaned)
	})
}

func TestSafePath(t *testing.T) {
	t.Run("valid join stays within dest", func(t *testing.T) {
		result, err := SafePath("/tmp/extract", "model.bin")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/extract/model.bin", result)
	})

	t.Run("malicious filename gets basename only", func(t *testing.T) {
		result, err := SafePath("/tmp/extract", "../../etc/passwd")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/extract/passwd", result)
	})

	t.Run("absolute filename uses only basename", func(t *testing.T) {
		result, err := SafePath("/tmp/extract", "/tmp/extract")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/extract/extract", result)
	})

	t.Run("root destPath with valid filename succeeds", func(t *testing.T) {
		result, err := SafePath("/", "model.bin")
		require.NoError(t, err)
		assert.Equal(t, "/model.bin", result)
	})

	t.Run("empty destPath returns error", func(t *testing.T) {
		_, err := SafePath("", "model.bin")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "destination path cannot be empty")
	})

	t.Run("empty filename returns error", func(t *testing.T) {
		_, err := SafePath("/tmp/extract", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid filename")
	})

	t.Run("dot filename returns error", func(t *testing.T) {
		_, err := SafePath("/tmp/extract", ".")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid filename")
	})
}
