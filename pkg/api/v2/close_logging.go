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
// the same underlying resource. Callers use this to avoid double-close: the
// dense EF is skipped only when the content side both shares the resource AND
// owns its cleanup (ownContent=true).
func isDenseEFSharedWithContent(denseEF embeddings.EmbeddingFunction, contentEF embeddings.ContentEmbeddingFunction) bool {
	if denseEF == nil || contentEF == nil {
		return false
	}
	unwrapped := unwrapCloseOnceEF(denseEF)
	if unwrapper, ok := contentEF.(embeddings.EmbeddingFunctionUnwrapper); ok {
		return unwrapper.UnwrapEmbeddingFunction() == unwrapped
	}
	if efFromContent, ok := contentEF.(embeddings.EmbeddingFunction); ok {
		return unwrapCloseOnceEF(efFromContent) == unwrapped
	}
	return false
}

// closeEmbeddingFunctions closes the content and dense embedding functions,
// skipping the dense EF if it shares the same underlying resource as the content EF.
func closeEmbeddingFunctions(denseEF embeddings.EmbeddingFunction, contentEF embeddings.ContentEmbeddingFunction) error {
	return closeOwnedEmbeddingFunctions(denseEF, true, contentEF, true)
}

func closeOwnedEmbeddingFunctions(
	denseEF embeddings.EmbeddingFunction,
	ownDense bool,
	contentEF embeddings.ContentEmbeddingFunction,
	ownContent bool,
) error {
	var errs []error
	if ownContent && contentEF != nil {
		if closer, ok := contentEF.(io.Closer); ok {
			if err := safeCloseEF(closer); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if ownDense && denseEF != nil && (!ownContent || !isDenseEFSharedWithContent(denseEF, contentEF)) {
		if closer, ok := denseEF.(io.Closer); ok {
			if err := safeCloseEF(closer); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return stderrors.Join(errs...)
}

func logCloseErrorToStderr(msg, name string, err error) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintf(
		os.Stderr,
		"chroma-go: %s: collection=%s error=%v\n",
		msg,
		name,
		err,
	)
}

func logCollectionCleanupCloseErrorToStderr(name string, err error) {
	logCloseErrorToStderr("failed to close EF during collection cache cleanup", name, err)
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

func logCreateIfNotExistsPreflightErrorToStderr(collectionName string, err error) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintf(
		os.Stderr,
		"chroma-go: create-if-not-exists preflight GetCollection failed: collection=%s error=%v\n",
		collectionName,
		err,
	)
}

func logComparableCollectionMapMarshalErrorToStderr(err error) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintf(
		os.Stderr,
		"chroma-go: unexpected json.Marshal failure comparing collection payloads: error=%v\n",
		err,
	)
}
