package tokenizers

import (
	"sync"
	"testing"
)

var tokenizerTestHooksMu sync.Mutex

func lockTokenizerTestHooks(t *testing.T) {
	t.Helper()
	tokenizerTestHooksMu.Lock()
	t.Cleanup(func() {
		tokenizerTestHooksMu.Unlock()
	})
}
