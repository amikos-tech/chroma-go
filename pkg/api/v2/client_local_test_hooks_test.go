//go:build basicv2 && !cloud
// +build basicv2,!cloud

package v2

import (
	"sync"
	"testing"
)

var localTestHooksMu sync.Mutex

// lockLocalTestHooks serializes tests that mutate package-level hook vars.
func lockLocalTestHooks(t *testing.T) {
	t.Helper()
	localTestHooksMu.Lock()
	t.Cleanup(func() {
		localTestHooksMu.Unlock()
	})
}
