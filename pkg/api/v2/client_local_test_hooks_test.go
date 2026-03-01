//go:build basicv2 && !cloud
// +build basicv2,!cloud

package v2

import (
	"sync"
	"testing"

	"github.com/pkg/errors"

	downloadutil "github.com/amikos-tech/chroma-go/pkg/internal/downloadutil"
)

var localTestHooksMu sync.Mutex

// lockLocalTestHooks serializes tests that mutate package-level hook vars.
func lockLocalTestHooks(t *testing.T) {
	t.Helper()
	localTestHooksMu.Lock()
	originalDownloadFileFunc := localDownloadFileFunc
	localDownloadFileFunc = func(filePath, url string) error {
		return errors.WithStack(downloadutil.DownloadFileWithRetry(
			filePath,
			url,
			localLibraryDownloadAttempts,
			downloadutil.Config{
				MaxBytes:  localLibraryMaxArtifactBytes,
				DirPerm:   localLibraryCacheDirPerm,
				AllowHTTP: true,
			},
		))
	}
	t.Cleanup(func() {
		localDownloadFileFunc = originalDownloadFileFunc
		localTestHooksMu.Unlock()
	})
}
