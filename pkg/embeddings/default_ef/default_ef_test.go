package defaultef

import (
	"context"
	stderrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amikos-tech/pure-onnx/embeddings/minilm"
	ort "github.com/amikos-tech/pure-onnx/ort"
	"github.com/stretchr/testify/require"
)

const (
	testEventuallyTimeout = 2 * time.Second
	testEventuallyTick    = 10 * time.Millisecond
)

func requireEventuallySignal(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	require.Eventually(t, func() bool {
		select {
		case <-ch:
			return true
		default:
			return false
		}
	}, testEventuallyTimeout, testEventuallyTick, msg)
}

func requireEventuallyError(t *testing.T, ch <-chan error, msg string) error {
	t.Helper()
	var (
		got bool
		err error
	)
	require.Eventually(t, func() bool {
		if got {
			return true
		}
		select {
		case err = <-ch:
			got = true
			return true
		default:
			return false
		}
	}, testEventuallyTimeout, testEventuallyTick, msg)
	return err
}

func testDefaultEFConfig() *Config {
	return &Config{
		LibOnnxRuntimeVersion:        "custom",
		OnnxCacheDir:                 "test-cache",
		OnnxLibPath:                  "test-lib",
		OnnxModelPath:                "test-model",
		OnnxModelTokenizerConfigPath: "test-tokenizer",
	}
}

type fakeEmbedder struct {
	embedDocumentsFn func([]string) ([][]float32, error)
	embedQueryFn     func(string) ([]float32, error)
	closeFn          func() error
}

func (f *fakeEmbedder) EmbedDocuments(documents []string) ([][]float32, error) {
	if f.embedDocumentsFn == nil {
		return nil, nil
	}
	return f.embedDocumentsFn(documents)
}

func (f *fakeEmbedder) EmbedQuery(document string) ([]float32, error) {
	if f.embedQueryFn == nil {
		return nil, nil
	}
	return f.embedQueryFn(document)
}

func (f *fakeEmbedder) Close() error {
	if f.closeFn == nil {
		return nil
	}
	return f.closeFn()
}

// Guardrail for concurrency-sensitive tests:
// keep dependency overrides local to each test via defaultEFDeps injection.
// Do not mutate shared package state after starting goroutines.
func testDefaultEFDeps() defaultEFDeps {
	return defaultEFDeps{
		ensureOnnxRuntimeSharedLibrary: func() error {
			return nil
		},
		ensureDefaultEmbeddingModel: func() error {
			return nil
		},
		initializeEnvironmentWithBootstrap: func(...ort.BootstrapOption) error {
			return nil
		},
		newEmbedder: func(string, string, ...minilm.Option) (defaultEFEmbedder, error) {
			return &fakeEmbedder{}, nil
		},
		destroyEnvironment: func() error {
			return nil
		},
	}
}

func TestNewDefaultEmbeddingFunctionWithDepsRejectsMissingDependencies(t *testing.T) {
	_, _, err := newDefaultEmbeddingFunctionWithDeps(testDefaultEFConfig(), defaultEFDeps{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid default embedding function dependencies")
	require.Contains(t, err.Error(), "ensureOnnxRuntimeSharedLibrary dependency is nil")
}

func TestNewDefaultEmbeddingFunctionWithDepsRejectsNilConfig(t *testing.T) {
	_, _, err := newDefaultEmbeddingFunctionWithDeps(nil, testDefaultEFDeps())
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid default embedding function config: nil")
}

func Test_Default_EF(t *testing.T) {
	setOfflineRuntimePathOrSkip(t)
	ef, closeEf, err := NewDefaultEmbeddingFunction()
	require.NoError(t, err)
	t.Cleanup(func() {
		err := closeEf()
		if err != nil {
			t.Logf("error while closing embedding function: %v", err)
		}
	})
	require.NotNil(t, ef)
	embeddings, err := ef.EmbedDocuments(context.TODO(), []string{"Hello Chroma!", "Hello world!"})
	require.NoError(t, err)
	require.NotNil(t, embeddings)
	require.Len(t, embeddings, 2)
	for _, embedding := range embeddings {
		require.Equal(t, embedding.Len(), 384)
	}
}

func TestClose(t *testing.T) {
	setOfflineRuntimePathOrSkip(t)
	ef, closeEf, err := NewDefaultEmbeddingFunction()
	require.NoError(t, err)
	require.NotNil(t, ef)
	err = closeEf()
	require.NoError(t, err)
	_, err = ef.EmbedQuery(context.TODO(), "Hello Chroma!")
	require.Error(t, err)
	require.Contains(t, err.Error(), "embedding function is closed")
}
func TestCloseClosed(t *testing.T) {
	ef := &DefaultEmbeddingFunction{}
	err := ef.Close()
	require.NoError(t, err)
}

func TestCustomOnnxRuntimeVersion(t *testing.T) {
	// Test that CHROMAGO_ONNX_RUNTIME_VERSION env var correctly sets the version
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Test with custom version
	customVersion := "1.21.0"
	t.Setenv("CHROMAGO_ONNX_RUNTIME_VERSION", customVersion)

	// Reset config to pick up the new env var
	resetConfigForTesting()

	cfg := getConfig()
	require.NotNil(t, cfg)
	require.Equal(t, customVersion, cfg.LibOnnxRuntimeVersion, "Config should use custom ONNX Runtime version from env var")
	require.Contains(t, cfg.OnnxCacheDir, "onnxruntime", "ONNX cache dir should target runtime cache")
}

func TestCustomOnnxRuntimePath(t *testing.T) {
	// This test downloads a specific ONNX Runtime version from GitHub
	// and tests using CHROMAGO_ONNX_RUNTIME_PATH
	// Set RUN_SLOW_TESTS=1 to enable this test
	if os.Getenv("RUN_SLOW_TESTS") != "1" {
		t.Skip("This test requires downloading ~33MB from GitHub and takes time - set RUN_SLOW_TESTS=1 to run")
	}

	// Set up temp directory
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Get platform info
	cos, carch := getOSAndArch()
	if cos == "windows" {
		t.Skip("slow custom runtime path test currently targets Unix ONNX tarball artifacts")
	}
	if carch == "amd64" {
		carch = "x64"
	}
	if cos == "darwin" {
		cos = "osx"
		if carch == "x64" {
			carch = "x86_64"
		}
	}

	// Download ONNX Runtime 1.23.0 (different from default 1.23.1)
	version := "1.23.0"
	url := "https://github.com/microsoft/onnxruntime/releases/download/v" + version + "/onnxruntime-" + cos + "-" + carch + "-" + version + ".tgz"

	t.Logf("Downloading ONNX Runtime %s for %s-%s from GitHub", version, cos, carch)
	targetArchive := tempDir + "/onnxruntime-" + version + ".tgz"
	err := downloadFile(targetArchive, url)
	require.NoError(t, err, "Failed to download ONNX Runtime from GitHub")

	// Extract the library file
	extractDir := tempDir + "/extracted"
	err = os.MkdirAll(extractDir, 0755)
	require.NoError(t, err)

	// Determine the library filename pattern in the archive
	// Note: tar archives have a leading "./" in the path
	var targetFile string
	if cos == "linux" {
		targetFile = "./onnxruntime-" + cos + "-" + carch + "-" + version + "/lib/libonnxruntime." + getExtensionForOs() + "." + version
	} else {
		targetFile = "./onnxruntime-" + cos + "-" + carch + "-" + version + "/lib/libonnxruntime." + version + "." + getExtensionForOs()
	}

	t.Logf("Extracting %s from archive", targetFile)
	err = extractSpecificFile(targetArchive, targetFile, extractDir)
	require.NoError(t, err, "Failed to extract library from archive")

	// Get the extracted library path - extractSpecificFile uses filepath.Base
	// so we need to construct the path with just the filename
	libFilename := ""
	if cos == "linux" {
		libFilename = "libonnxruntime." + getExtensionForOs() + "." + version
	} else {
		libFilename = "libonnxruntime." + version + "." + getExtensionForOs()
	}
	libPath := filepath.Join(extractDir, libFilename)

	// Verify the library file exists
	_, err = os.Stat(libPath)
	require.NoError(t, err, "Extracted library not found at %s", libPath)

	t.Logf("Using custom ONNX Runtime library at: %s", libPath)

	// Set the custom path environment variable
	t.Setenv("CHROMAGO_ONNX_RUNTIME_PATH", libPath)

	// Reset config to pick up the new environment variable
	resetConfigForTesting()

	// Create embedding function - should use the custom library
	ef, closeEf, err := NewDefaultEmbeddingFunction()
	require.NoError(t, err, "Failed to create embedding function with custom ONNX Runtime path")
	t.Cleanup(func() {
		err := closeEf()
		if err != nil {
			t.Logf("error while closing embedding function: %v", err)
		}
	})
	require.NotNil(t, ef)

	// Test that embeddings work with the custom library
	embeddings, err := ef.EmbedDocuments(context.TODO(), []string{"Testing custom ONNX Runtime path"})
	require.NoError(t, err, "Failed to generate embeddings with custom library")
	require.NotNil(t, embeddings)
	require.Len(t, embeddings, 1)
	require.Equal(t, 384, embeddings[0].Len(), "Expected 384-dimensional embeddings")

	t.Logf("✓ Successfully used ONNX Runtime %s from custom path", version)
}

func TestConcurrentInitModelEnsureDoesNotBlockClose(t *testing.T) {
	modelEnsureStarted := make(chan struct{})
	releaseModelEnsure := make(chan struct{})
	initDone := make(chan error, 1)

	deps := testDefaultEFDeps()
	deps.ensureDefaultEmbeddingModel = func() error {
		close(modelEnsureStarted)
		<-releaseModelEnsure
		return stderrors.New("simulated model ensure stall")
	}
	deps.initializeEnvironmentWithBootstrap = func(...ort.BootstrapOption) error {
		return stderrors.New("unexpected environment initialization call")
	}

	go func() {
		_, _, err := newDefaultEmbeddingFunctionWithDeps(testDefaultEFConfig(), deps)
		initDone <- err
	}()

	requireEventuallySignal(t, modelEnsureStarted, "initializer did not enter model ensure stage")

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- (&DefaultEmbeddingFunction{
			destroyEnvironment: func() error { return nil },
		}).Close()
	}()

	require.NoError(t, requireEventuallyError(t, closeDone, "close blocked while another initializer was waiting on model ensure"))

	close(releaseModelEnsure)

	initErr := requireEventuallyError(t, initDone, "initializer did not unblock after model ensure release")
	require.Error(t, initErr)
	require.Contains(t, initErr.Error(), "failed to ensure default embedding function model")
}

func TestConcurrentInitOnnxEnsureDoesNotBlockClose(t *testing.T) {
	onnxEnsureStarted := make(chan struct{})
	releaseOnnxEnsure := make(chan struct{})
	modelEnsureCalled := make(chan struct{}, 1)
	initDone := make(chan error, 1)

	deps := testDefaultEFDeps()
	deps.ensureOnnxRuntimeSharedLibrary = func() error {
		close(onnxEnsureStarted)
		<-releaseOnnxEnsure
		return stderrors.New("simulated onnx ensure stall")
	}
	deps.ensureDefaultEmbeddingModel = func() error {
		select {
		case modelEnsureCalled <- struct{}{}:
		default:
		}
		return nil
	}
	deps.initializeEnvironmentWithBootstrap = func(...ort.BootstrapOption) error {
		return stderrors.New("unexpected environment initialization call")
	}

	go func() {
		_, _, err := newDefaultEmbeddingFunctionWithDeps(testDefaultEFConfig(), deps)
		initDone <- err
	}()

	requireEventuallySignal(t, onnxEnsureStarted, "initializer did not enter onnx ensure stage")

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- (&DefaultEmbeddingFunction{
			destroyEnvironment: func() error { return nil },
		}).Close()
	}()

	require.NoError(t, requireEventuallyError(t, closeDone, "close blocked while another initializer was waiting on onnx ensure"))

	close(releaseOnnxEnsure)

	initErr := requireEventuallyError(t, initDone, "initializer did not unblock after onnx ensure release")
	require.Error(t, initErr)
	require.Contains(t, initErr.Error(), "failed to ensure onnx runtime shared library")

	select {
	case <-modelEnsureCalled:
		t.Fatal("model ensure should not run when onnx ensure fails")
	default:
	}
}

func TestOptionFailureCleanupAggregatesDestroyError(t *testing.T) {
	deps := testDefaultEFDeps()
	deps.destroyEnvironment = func() error {
		return stderrors.New("destroy failed")
	}

	done := make(chan error, 1)
	go func() {
		_, _, err := newDefaultEmbeddingFunctionWithDeps(testDefaultEFConfig(), deps, func(*DefaultEmbeddingFunction) error {
			return stderrors.New("option failed")
		})
		done <- err
	}()

	err := requireEventuallyError(t, done, "option failure path deadlocked")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to apply default embedding function option")
	require.Contains(t, err.Error(), "failed to destroy onnx runtime environment after option error")
}

func TestEmbedderCreationFailureAggregatesDestroyError(t *testing.T) {
	deps := testDefaultEFDeps()
	deps.newEmbedder = func(string, string, ...minilm.Option) (defaultEFEmbedder, error) {
		return nil, stderrors.New("embedder setup failed")
	}
	deps.destroyEnvironment = func() error {
		return stderrors.New("destroy failed")
	}

	_, _, err := newDefaultEmbeddingFunctionWithDeps(testDefaultEFConfig(), deps)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create ONNX embedder")
	require.Contains(t, err.Error(), "failed to destroy onnx runtime environment after embedder setup error")
}

func TestOptionFailureCleanupAggregatesCloseError(t *testing.T) {
	deps := testDefaultEFDeps()
	deps.newEmbedder = func(string, string, ...minilm.Option) (defaultEFEmbedder, error) {
		return &fakeEmbedder{
			closeFn: func() error {
				return stderrors.New("close failed")
			},
		}, nil
	}

	done := make(chan error, 1)
	go func() {
		_, _, err := newDefaultEmbeddingFunctionWithDeps(testDefaultEFConfig(), deps, func(*DefaultEmbeddingFunction) error {
			return stderrors.New("option failed")
		})
		done <- err
	}()

	err := requireEventuallyError(t, done, "option failure close-error path deadlocked")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to apply default embedding function option")
	require.Contains(t, err.Error(), "failed to close embedder after option error")
	require.NotContains(t, err.Error(), "failed to destroy onnx runtime environment after option error")
}

func TestOptionFailureReturnsOnlyOptionErrorWhenCleanupSucceeds(t *testing.T) {
	deps := testDefaultEFDeps()
	deps.newEmbedder = func(string, string, ...minilm.Option) (defaultEFEmbedder, error) {
		return &fakeEmbedder{}, nil
	}

	_, _, err := newDefaultEmbeddingFunctionWithDeps(testDefaultEFConfig(), deps, func(*DefaultEmbeddingFunction) error {
		return stderrors.New("option failed")
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to apply default embedding function option")
	require.NotContains(t, err.Error(), "failed to close embedder after option error")
	require.NotContains(t, err.Error(), "failed to destroy onnx runtime environment after option error")
}

func TestEmbedderCreationFailureReturnsOnlyEmbedderErrorWhenDestroySucceeds(t *testing.T) {
	deps := testDefaultEFDeps()
	deps.newEmbedder = func(string, string, ...minilm.Option) (defaultEFEmbedder, error) {
		return nil, stderrors.New("embedder setup failed")
	}
	deps.destroyEnvironment = func() error {
		return nil
	}

	_, _, err := newDefaultEmbeddingFunctionWithDeps(testDefaultEFConfig(), deps)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create ONNX embedder")
	require.NotContains(t, err.Error(), "failed to destroy onnx runtime environment after embedder setup error")
}

func TestCloseReturnsEmbedderCloseError(t *testing.T) {
	ef := &DefaultEmbeddingFunction{
		embedder: &fakeEmbedder{
			closeFn: func() error {
				return stderrors.New("close failed")
			},
		},
		destroyEnvironment: func() error {
			return nil
		},
	}
	err := ef.Close()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to close embedder")
}

func TestOptionFailureCleanupAggregatesCloseAndDestroyErrors(t *testing.T) {
	deps := testDefaultEFDeps()
	deps.newEmbedder = func(string, string, ...minilm.Option) (defaultEFEmbedder, error) {
		return &fakeEmbedder{
			closeFn: func() error {
				return stderrors.New("close failed")
			},
		}, nil
	}
	deps.destroyEnvironment = func() error {
		return stderrors.New("destroy failed")
	}

	_, _, err := newDefaultEmbeddingFunctionWithDeps(testDefaultEFConfig(), deps, func(*DefaultEmbeddingFunction) error {
		return stderrors.New("option failed")
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to apply default embedding function option")
	require.Contains(t, err.Error(), "failed to close embedder after option error")
	require.Contains(t, err.Error(), "failed to destroy onnx runtime environment after option error")
}

func TestOptionFailureSkipsEmbedderCloseWhenOptionNilOutEmbedder(t *testing.T) {
	deps := testDefaultEFDeps()
	closeCalled := make(chan struct{}, 1)
	deps.newEmbedder = func(string, string, ...minilm.Option) (defaultEFEmbedder, error) {
		return &fakeEmbedder{
			closeFn: func() error {
				select {
				case closeCalled <- struct{}{}:
				default:
				}
				return nil
			},
		}, nil
	}

	_, _, err := newDefaultEmbeddingFunctionWithDeps(testDefaultEFConfig(), deps, func(ef *DefaultEmbeddingFunction) error {
		ef.embedder = nil
		return stderrors.New("option failed")
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to apply default embedding function option")
	require.NotContains(t, err.Error(), "failed to close embedder after option error")
	require.NotContains(t, err.Error(), "failed to destroy onnx runtime environment after option error")

	select {
	case <-closeCalled:
		t.Fatal("embedder close should not be called when option sets embedder to nil")
	default:
	}
}

func TestOptionFailureLeakedReferenceCloseDoesNotDestroyEnvironmentTwice(t *testing.T) {
	deps := testDefaultEFDeps()

	var destroyCalls int32
	deps.destroyEnvironment = func() error {
		atomic.AddInt32(&destroyCalls, 1)
		return nil
	}

	leakedEF := (*DefaultEmbeddingFunction)(nil)
	_, _, err := newDefaultEmbeddingFunctionWithDeps(testDefaultEFConfig(), deps, func(ef *DefaultEmbeddingFunction) error {
		leakedEF = ef
		return stderrors.New("option failed")
	})
	require.Error(t, err)
	require.NotNil(t, leakedEF)

	require.NoError(t, leakedEF.Close())
	require.NoError(t, leakedEF.Close())
	require.Equal(t, int32(1), atomic.LoadInt32(&destroyCalls))
}

func TestConcurrentInitCloseUse(t *testing.T) {
	setOfflineRuntimePathOrSkip(t)
	const numGoroutines = 10
	const numOperations = 5

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				ef, closeEf, err := NewDefaultEmbeddingFunction()
				if err != nil {
					// ORT init errors are expected when rapidly destroying/re-initializing
					// The test verifies no race conditions (via -race flag) and no deadlocks
					continue
				}

				_, _ = ef.EmbedDocuments(context.TODO(), []string{"test document"})
				_ = closeEf()
			}
		}()
	}

	wg.Wait()
	// Test passes if no deadlock, no panic, and no race detected (via -race flag)
}

func TestConcurrentCloseWhileEmbedding(t *testing.T) {
	setOfflineRuntimePathOrSkip(t)
	ef1, closeEf1, err := NewDefaultEmbeddingFunction()
	if err != nil {
		t.Skipf("Skipping test due to ORT init error: %v", err)
	}

	_, closeEf2, err := NewDefaultEmbeddingFunction()
	if err != nil {
		_ = closeEf1()
		t.Skipf("Skipping test due to ORT init error: %v", err)
	}

	var wg sync.WaitGroup

	// Goroutine 1: repeatedly embed with ef1
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			_, _ = ef1.EmbedDocuments(context.TODO(), []string{"document from ef1"})
		}
	}()

	// Goroutine 2: close ef2 while ef1 is embedding
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = closeEf2()
	}()

	wg.Wait()

	// Clean up ef1
	_ = closeEf1()
	// Test passes if no deadlock, no panic, and no race detected (via -race flag)
}

func TestConcurrentEmbeddings(t *testing.T) {
	setOfflineRuntimePathOrSkip(t)
	ef, closeEf, err := NewDefaultEmbeddingFunction()
	if err != nil {
		t.Skipf("Skipping test due to ORT init error: %v", err)
	}
	t.Cleanup(func() { _ = closeEf() })

	const numGoroutines = 5
	var wg sync.WaitGroup
	results := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			embeddings, err := ef.EmbedDocuments(context.TODO(), []string{"test document for goroutine"})
			if err != nil {
				t.Errorf("goroutine %d: embedding failed: %v", id, err)
				return
			}
			if len(embeddings) != 1 {
				t.Errorf("goroutine %d: expected 1 embedding, got %d", id, len(embeddings))
				return
			}
			results <- id
		}(i)
	}

	wg.Wait()
	close(results)

	completed := 0
	for range results {
		completed++
	}
	require.Equal(t, numGoroutines, completed, "All goroutines should complete successfully")
}

func TestConcurrentSessionReuse(t *testing.T) {
	setOfflineRuntimePathOrSkip(t)
	ef, closeEf, err := NewDefaultEmbeddingFunction()
	if err != nil {
		t.Skipf("Skipping test due to ORT init error: %v", err)
	}
	t.Cleanup(func() { _ = closeEf() })

	const numGoroutines = 10
	const numIterations = 5
	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines*numIterations)

	// Test concurrent access to single session with varying input sizes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				// Vary the number of documents to test different tensor shapes
				numDocs := (id % 3) + 1
				docs := make([]string, numDocs)
				for k := 0; k < numDocs; k++ {
					docs[k] = fmt.Sprintf("document %d from goroutine %d iteration %d", k, id, j)
				}

				embeddings, err := ef.EmbedDocuments(context.TODO(), docs)
				if err != nil {
					errCh <- fmt.Errorf("goroutine %d iter %d: %w", id, j, err)
					return
				}
				if len(embeddings) != numDocs {
					errCh <- fmt.Errorf("goroutine %d iter %d: expected %d embeddings, got %d", id, j, numDocs, len(embeddings))
					return
				}
				for k, emb := range embeddings {
					if emb.Len() != 384 {
						errCh <- fmt.Errorf("goroutine %d iter %d doc %d: expected 384 dims, got %d", id, j, k, emb.Len())
						return
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	require.Empty(t, errs, "Expected no errors during concurrent session reuse")
}

func TestMultipleInstancesConcurrent(t *testing.T) {
	setOfflineRuntimePathOrSkip(t)
	const numInstances = 3
	efs := make([]*DefaultEmbeddingFunction, numInstances)
	closers := make([]func() error, numInstances)

	// Create multiple instances
	for i := 0; i < numInstances; i++ {
		ef, closeEf, err := NewDefaultEmbeddingFunction()
		if err != nil {
			// Clean up already created instances
			for j := 0; j < i; j++ {
				_ = closers[j]()
			}
			t.Skipf("Skipping test due to ORT init error: %v", err)
		}
		efs[i] = ef
		closers[i] = closeEf
	}
	t.Cleanup(func() {
		for _, closer := range closers {
			_ = closer()
		}
	})

	var wg sync.WaitGroup
	errCh := make(chan error, numInstances*5)

	// Run concurrent embeddings on different instances
	for i := 0; i < numInstances; i++ {
		wg.Add(1)
		go func(id int, ef *DefaultEmbeddingFunction) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				embeddings, err := ef.EmbedDocuments(context.TODO(), []string{
					fmt.Sprintf("doc from instance %d", id),
				})
				if err != nil {
					errCh <- fmt.Errorf("instance %d iter %d: %w", id, j, err)
					return
				}
				if len(embeddings) != 1 || embeddings[0].Len() != 384 {
					errCh <- fmt.Errorf("instance %d iter %d: invalid embedding", id, j)
					return
				}
			}
		}(i, efs[i])
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	require.Empty(t, errs, "Expected no errors with multiple concurrent instances")
}
