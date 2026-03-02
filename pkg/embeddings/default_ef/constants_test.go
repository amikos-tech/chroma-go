package defaultef

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveHomeDirNeverFallsBackToCurrentDirectory(t *testing.T) {
	t.Setenv("HOME", "")
	homeDir := resolveHomeDir()

	require.NotEmpty(t, strings.TrimSpace(homeDir))
	require.NotEqual(t, ".", strings.TrimSpace(homeDir))
}
