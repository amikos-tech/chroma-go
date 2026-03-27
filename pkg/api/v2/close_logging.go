package v2

import (
	"fmt"
	"os"
	"runtime/debug"
)

func reportClosePanic(recovered any) error {
	err := fmt.Errorf("panic during EF close: %v\nstack: %s", recovered, debug.Stack())
	_, _ = fmt.Fprintf(os.Stderr, "chroma-go: %s\n", err)
	return err
}

func logCollectionCleanupCloseErrorToStderr(name string, err error) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintf(
		os.Stderr,
		"chroma-go: failed to close EF during collection cache cleanup: collection=%s error=%v\n",
		name,
		err,
	)
}
