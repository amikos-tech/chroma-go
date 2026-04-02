package v2

import (
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func reportClosePanic(recovered any) error {
	err := fmt.Errorf("panic during EF close: %v\nstack: %s", recovered, debug.Stack())
	_, _ = fmt.Fprintf(os.Stderr, "chroma-go: %s\n", err)
	return err
}

// safeCloseEF calls closer.Close() with panic recovery so a panicking
// Close does not prevent subsequent cleanup from executing.
func safeCloseEF(closer io.Closer) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = reportClosePanic(r)
		}
	}()
	return closer.Close()
}

// isDenseEFSharedWithContent reports whether the dense EF and content EF share
// the same underlying resource. When true, closing the content EF is sufficient
// and the dense EF should not be closed separately.
func isDenseEFSharedWithContent(denseEF embeddings.EmbeddingFunction, contentEF embeddings.ContentEmbeddingFunction) bool {
	if denseEF == nil || contentEF == nil {
		return false
	}
	unwrapped := unwrapCloseOnceEF(denseEF)
	if unwrapper, ok := contentEF.(embeddings.EmbeddingFunctionUnwrapper); ok {
		return unwrapper.UnwrapEmbeddingFunction() == unwrapped
	}
	if efFromContent, ok := contentEF.(embeddings.EmbeddingFunction); ok {
		return efFromContent == unwrapped
	}
	return false
}

// closeEmbeddingFunctions closes the content and dense embedding functions,
// skipping the dense EF if it shares the same underlying resource as the content EF.
func closeEmbeddingFunctions(denseEF embeddings.EmbeddingFunction, contentEF embeddings.ContentEmbeddingFunction) error {
	var errs []error
	if contentEF != nil {
		if closer, ok := contentEF.(io.Closer); ok {
			if err := safeCloseEF(closer); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if denseEF != nil && !isDenseEFSharedWithContent(denseEF, contentEF) {
		if closer, ok := denseEF.(io.Closer); ok {
			if err := safeCloseEF(closer); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return stderrors.Join(errs...)
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

func logAutoWireBuildErrorToStderr(collectionName, target string, err error) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintf(
		os.Stderr,
		"chroma-go: failed to auto-wire %s: collection=%s error=%v\n",
		target,
		collectionName,
		err,
	)
}
