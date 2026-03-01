package tokenizers

import (
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func resetTokenizerLibraryInitState(t *testing.T) {
	t.Helper()
	lockTokenizerTestHooks(t)

	origFn := ensureTokenizerLibraryDownloadedFn
	origErr := libraryInitErr
	origPath := libraryInitPath
	origReady := libraryInitReady

	t.Cleanup(func() {
		ensureTokenizerLibraryDownloadedFn = origFn
		libraryInitErr = origErr
		libraryInitPath = origPath
		libraryInitReady = origReady
	})

	ensureTokenizerLibraryDownloadedFn = ensureTokenizerLibraryDownloaded
	libraryInitErr = nil
	libraryInitPath = ""
	libraryInitReady = false
}

func TestEnsureTokenizerLibraryReady_RetriesAfterFailure(t *testing.T) {
	resetTokenizerLibraryInitState(t)
	t.Setenv("TOKENIZERS_LIB_PATH", "")

	var attempts int
	ensureTokenizerLibraryDownloadedFn = func() (string, error) {
		attempts++
		if attempts == 1 {
			return "", stderrors.New("transient download error")
		}
		return "/tmp/libtokenizers.so", nil
	}

	err := ensureTokenizerLibraryReady()
	require.Error(t, err)
	require.Equal(t, 1, attempts)
	require.False(t, libraryInitReady)

	err = ensureTokenizerLibraryReady()
	require.NoError(t, err)
	require.Equal(t, 2, attempts)
	require.True(t, libraryInitReady)
	require.Equal(t, "/tmp/libtokenizers.so", libraryInitPath)

	err = ensureTokenizerLibraryReady()
	require.NoError(t, err)
	require.Equal(t, 2, attempts)
}

func TestEnsureTokenizerLibraryReady_UsesEnvPath(t *testing.T) {
	resetTokenizerLibraryInitState(t)
	t.Setenv("TOKENIZERS_LIB_PATH", "/opt/custom/libtokenizers.so")

	var attempts int
	ensureTokenizerLibraryDownloadedFn = func() (string, error) {
		attempts++
		return "", stderrors.New("should not be called when TOKENIZERS_LIB_PATH is set")
	}

	err := ensureTokenizerLibraryReady()
	require.NoError(t, err)
	require.Equal(t, 0, attempts)
	require.True(t, libraryInitReady)
	require.Equal(t, "/opt/custom/libtokenizers.so", libraryInitPath)
}

func TestEnsureTokenizerLibraryReady_CachesSuccessfulInit(t *testing.T) {
	resetTokenizerLibraryInitState(t)
	t.Setenv("TOKENIZERS_LIB_PATH", "")

	var attempts int
	ensureTokenizerLibraryDownloadedFn = func() (string, error) {
		attempts++
		return "/tmp/libtokenizers.so", nil
	}

	require.NoError(t, ensureTokenizerLibraryReady())
	require.NoError(t, ensureTokenizerLibraryReady())
	require.Equal(t, 1, attempts)
}
