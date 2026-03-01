package tokenizers

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeTokenizerTag(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		expected  string
		expectErr bool
	}{
		{name: "empty", in: "", expectErr: true},
		{name: "latest", in: "latest", expected: "latest"},
		{name: "semver", in: "0.1.4", expected: "rust-v0.1.4"},
		{name: "go tag", in: "v0.1.4", expected: "rust-v0.1.4"},
		{name: "rust prefix", in: "rust-v0.1.4", expected: "rust-v0.1.4"},
		{name: "bare rust prefix", in: "rust-0.1.4", expected: "rust-v0.1.4"},
		{name: "empty rust suffix", in: "rust-", expected: "latest"},
		{name: "invalid chars", in: "v0.1.4/../../", expectErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeTokenizerTag(tc.in)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestTokenizerReleaseBaseURLsDeduplicates(t *testing.T) {
	originalPrimary := tokenizerReleaseBaseURL
	originalFallback := tokenizerFallbackReleaseBaseURL
	t.Cleanup(func() {
		tokenizerReleaseBaseURL = originalPrimary
		tokenizerFallbackReleaseBaseURL = originalFallback
	})

	tokenizerReleaseBaseURL = "https://releases.amikos.tech/pure-tokenizers/"
	tokenizerFallbackReleaseBaseURL = "https://releases.amikos.tech/pure-tokenizers"

	require.Equal(t, []string{"https://releases.amikos.tech/pure-tokenizers"}, tokenizerReleaseBaseURLs())
}

func TestTokenizerChecksumFromSumsFile(t *testing.T) {
	t.Run("find checksum", func(t *testing.T) {
		dir := t.TempDir()
		sumsPath := filepath.Join(dir, "SHA256SUMS")
		contents := "1111111111111111111111111111111111111111111111111111111111111111  libtokenizers-x86_64-apple-darwin.tar.gz\n"
		require.NoError(t, os.WriteFile(sumsPath, []byte(contents), 0600))

		checksum, err := tokenizerChecksumFromSumsFile(sumsPath, "libtokenizers-x86_64-apple-darwin.tar.gz")
		require.NoError(t, err)
		require.Equal(t, "1111111111111111111111111111111111111111111111111111111111111111", checksum)
	})

	t.Run("missing checksum", func(t *testing.T) {
		dir := t.TempDir()
		sumsPath := filepath.Join(dir, "SHA256SUMS")
		contents := "2222222222222222222222222222222222222222222222222222222222222222  other.tar.gz\n"
		require.NoError(t, os.WriteFile(sumsPath, []byte(contents), 0600))

		_, err := tokenizerChecksumFromSumsFile(sumsPath, "libtokenizers-x86_64-apple-darwin.tar.gz")
		require.Error(t, err)
		require.Contains(t, err.Error(), "checksum entry not found")
	})
}

func TestEnsureTokenizerLibraryDownloadedFromPrimaryWhenFallbackUnavailable(t *testing.T) {
	if os.Getenv("RUN_LIVE_TOKENIZERS_DOWNLOAD_TESTS") != "1" {
		t.Skip("set RUN_LIVE_TOKENIZERS_DOWNLOAD_TESTS=1 to run live download integration test")
	}

	originalPrimary := tokenizerReleaseBaseURL
	originalFallback := tokenizerFallbackReleaseBaseURL
	originalHomeDir := tokenizerUserHomeDirFunc
	originalDownloadFunc := tokenizerDownloadFileFunc
	originalBuildInfo := tokenizerReadBuildInfoFunc
	t.Cleanup(func() {
		tokenizerReleaseBaseURL = originalPrimary
		tokenizerFallbackReleaseBaseURL = originalFallback
		tokenizerUserHomeDirFunc = originalHomeDir
		tokenizerDownloadFileFunc = originalDownloadFunc
		tokenizerReadBuildInfoFunc = originalBuildInfo
	})

	tempHome := t.TempDir()
	tokenizerUserHomeDirFunc = func() (string, error) { return tempHome, nil }
	tokenizerDownloadFileFunc = tokenizerDownloadFileWithRetry
	tokenizerReadBuildInfoFunc = func() (*debug.BuildInfo, bool) { return nil, false }
	tokenizerReleaseBaseURL = defaultTokenizerReleaseBaseURL
	tokenizerFallbackReleaseBaseURL = "https://127.0.0.1.invalid/pure-tokenizers"

	t.Setenv("TOKENIZERS_VERSION", "rust-v0.1.4")

	libPath, err := ensureTokenizerLibraryDownloaded()
	require.NoError(t, err)
	require.FileExists(t, libPath)
	require.Contains(t, libPath, filepath.Join(".cache", "chroma", "pure_tokenizers", "rust-v0.1.4"))
}
