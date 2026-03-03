package defaultef

import (
	"sync"
	"testing"
)

var defaultEFTestHooksMu sync.Mutex

func lockDefaultEFTestHooks(t *testing.T) {
	t.Helper()
	defaultEFTestHooksMu.Lock()
	t.Cleanup(func() {
		defaultEFTestHooksMu.Unlock()
	})
}
