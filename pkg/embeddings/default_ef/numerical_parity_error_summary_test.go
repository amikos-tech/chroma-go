package defaultef

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSummarizePythonError(t *testing.T) {
	t.Run("empty stderr", func(t *testing.T) {
		require.Equal(t, "unknown python error", summarizePythonError(""))
		require.Equal(t, "unknown python error", summarizePythonError("  \n\t"))
	})

	t.Run("traceback returns last non-empty line", func(t *testing.T) {
		stderr := strings.Join([]string{
			"Traceback (most recent call last):",
			`  File "<stdin>", line 1, in <module>`,
			"ValueError: broken tokenizer config",
			"",
		}, "\n")
		require.Equal(t, "ValueError: broken tokenizer config", summarizePythonError(stderr))
	})

	t.Run("long last line is truncated", func(t *testing.T) {
		longLine := "RuntimeError: " + strings.Repeat("x", 260)
		got := summarizePythonError("Traceback\n" + longLine)

		require.True(t, strings.HasPrefix(got, "RuntimeError: "))
		require.True(t, strings.HasSuffix(got, "..."))
		require.Len(t, got, 223) // 220 chars + "..."
	})
}
