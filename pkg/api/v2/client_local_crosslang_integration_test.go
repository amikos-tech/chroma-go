//go:build crosslang && !cloud

package v2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

type localPersistenceHarnessResult struct {
	Status          string   `json:"status"`
	Error           string   `json:"error,omitempty"`
	Action          string   `json:"action"`
	Collection      string   `json:"collection"`
	Count           int      `json:"count"`
	TopID           string   `json:"top_id,omitempty"`
	UpdatedID       string   `json:"updated_id,omitempty"`
	UpdatedDocument string   `json:"updated_document,omitempty"`
	IDs             []string `json:"ids,omitempty"`
}

func findRepoRoot(t *testing.T) string {
	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repository root")
		}
		dir = parent
	}
}

func pythonExecutable(t *testing.T, repoRoot string) string {
	venvPython := filepath.Join(repoRoot, ".venv", "bin", "python")
	if _, err := os.Stat(venvPython); err == nil {
		return venvPython
	}
	pythonExec, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not found; skipping cross-language local persistence tests")
	}
	return pythonExec
}

func localLibraryPathIfConfigured(t *testing.T) (string, bool) {
	libPath := strings.TrimSpace(os.Getenv("CHROMA_LIB_PATH"))
	if libPath == "" {
		return "", false
	}
	if _, err := os.Stat(libPath); err != nil {
		t.Skipf("CHROMA_LIB_PATH does not point to a readable file: %v", err)
	}
	return libPath, true
}

func newLocalClientForRoundTrip(t *testing.T, persistPath string) Client {
	opts := []LocalClientOption{
		WithLocalPersistPath(persistPath),
	}
	if libPath, ok := localLibraryPathIfConfigured(t); ok {
		opts = append(opts, WithLocalLibraryPath(libPath))
	} else {
		t.Log("CHROMA_LIB_PATH is not set; using local runtime library auto-download")
	}

	client, err := NewLocalClient(opts...)
	require.NoError(t, err)
	return client
}

func runLocalPersistenceHarness(
	t *testing.T,
	action string,
	persistPath string,
	collectionName string,
) localPersistenceHarnessResult {
	repoRoot := findRepoRoot(t)
	scriptPath := filepath.Join(repoRoot, "scripts", "local_persistence_crosscheck.py")
	require.FileExists(t, scriptPath, "missing Python local persistence harness")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		pythonExecutable(t, repoRoot),
		scriptPath,
		"--action", action,
		"--persist-path", persistPath,
		"--collection", collectionName,
	)
	cmd.Env = os.Environ()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Logf("Python stdout: %s", stdout.String())
		t.Logf("Python stderr: %s", stderr.String())
	}
	require.NoError(
		t,
		err,
		"python harness failed: stdout=%s stderr=%s",
		stdout.String(),
		stderr.String(),
	)

	var result localPersistenceHarnessResult
	err = json.Unmarshal(stdout.Bytes(), &result)
	require.NoError(t, err, "failed to parse python harness output: %s", stdout.String())
	require.Equal(t, "success", result.Status, "python harness reported error: %s", result.Error)
	return result
}

func runMountedDockerChroma(t *testing.T, persistPath string) (string, func()) {
	ctx := context.Background()

	chromaImage := os.Getenv("CHROMA_IMAGE")
	if chromaImage == "" {
		// chroma-go-local v0.2.0 uses Chroma 1.5.x internally.
		chromaImage = "ghcr.io/chroma-core/chroma:1.5.1"
	}

	req := testcontainers.ContainerRequest{
		Image:        chromaImage,
		ExposedPorts: []string{"8000/tcp"},
		Cmd: []string{
			"run",
			"--path", "/data",
			"--host", "0.0.0.0",
			"--port", "8000",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(nat.Port("8000/tcp")),
			wait.ForHTTP("/api/v2/heartbeat").WithPort(nat.Port("8000/tcp")),
		).WithDeadline(90 * time.Second),
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.Mounts = []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: persistPath,
					Target: "/data",
				},
			}
		},
	}

	chromaContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := chromaContainer.Host(ctx)
	require.NoError(t, err)
	port, err := chromaContainer.MappedPort(ctx, "8000")
	require.NoError(t, err)
	baseURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	return baseURL, func() {
		if err := chromaContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate Chroma container: %v", err)
		}
	}
}

func oneHotEmbeddings() []embeddings.Embedding {
	return []embeddings.Embedding{
		embeddings.NewEmbeddingFromFloat32([]float32{1.0, 0.0, 0.0}),
		embeddings.NewEmbeddingFromFloat32([]float32{0.0, 1.0, 0.0}),
		embeddings.NewEmbeddingFromFloat32([]float32{0.0, 0.0, 1.0}),
	}
}

func newCrossLangPersistDir(t *testing.T) string {
	t.Helper()

	persistPath, err := os.MkdirTemp("", "chroma-local-persist-*")
	require.NoError(t, err)

	t.Cleanup(func() {
		if removeErr := os.RemoveAll(persistPath); removeErr != nil {
			t.Logf("best-effort cleanup for local persistence dir failed (%s): %v", persistPath, removeErr)
		}
	})

	return persistPath
}

func TestLocalPersistenceRoundTrip_GoAndPython(t *testing.T) {
	ctx := context.Background()
	persistPath := newCrossLangPersistDir(t)
	goCollectionName := fmt.Sprintf("go_local_py_%d", time.Now().UnixNano())
	pythonCollectionName := fmt.Sprintf("py_local_go_%d", time.Now().UnixNano())

	localClient := newLocalClientForRoundTrip(t, persistPath)
	localClosed := false
	closeLocal := func() {
		if !localClosed {
			require.NoError(t, localClient.Close())
			localClosed = true
		}
	}
	t.Cleanup(closeLocal)

	goCollection, err := localClient.GetOrCreateCollection(
		ctx,
		goCollectionName,
		WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()),
	)
	require.NoError(t, err)

	err = goCollection.Add(
		ctx,
		WithIDs("go-1", "go-2", "go-3"),
		WithTexts("go document 1", "go document 2", "go document 3"),
		WithEmbeddings(oneHotEmbeddings()...),
	)
	require.NoError(t, err)

	count, err := goCollection.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, count)

	closeLocal()

	verifyResult := runLocalPersistenceHarness(t, "verify-go", persistPath, goCollectionName)
	require.Equal(t, goCollectionName, verifyResult.Collection)
	require.Equal(t, 3, verifyResult.Count)
	require.Equal(t, "go-1", verifyResult.TopID)
	require.Equal(t, "go-2", verifyResult.UpdatedID)
	require.Equal(t, "go document 2 updated by python", verifyResult.UpdatedDocument)

	localClient = newLocalClientForRoundTrip(t, persistPath)
	localClosed = false

	goCollection, err = localClient.GetCollection(ctx, goCollectionName)
	require.NoError(t, err)

	getResult, err := goCollection.Get(ctx, WithIDs("go-2"), WithInclude(IncludeDocuments))
	require.NoError(t, err)
	require.Len(t, getResult.GetDocuments(), 1)
	require.Equal(t, "go document 2 updated by python", getResult.GetDocuments()[0].ContentString())

	createResult := runLocalPersistenceHarness(t, "create-python", persistPath, pythonCollectionName)
	require.Equal(t, pythonCollectionName, createResult.Collection)
	require.Equal(t, 3, createResult.Count)
	require.Equal(t, "py-2", createResult.TopID)

	pythonCollection, err := localClient.GetCollection(ctx, pythonCollectionName)
	require.NoError(t, err)

	pyQueryResult, err := pythonCollection.Query(
		ctx,
		WithQueryEmbeddings(embeddings.NewEmbeddingFromFloat32([]float32{0.05, 0.95, 0.0})),
		WithNResults(1),
	)
	require.NoError(t, err)
	require.NotEmpty(t, pyQueryResult.GetIDGroups())
	require.NotEmpty(t, pyQueryResult.GetIDGroups()[0])
	require.Equal(t, DocumentID("py-2"), pyQueryResult.GetIDGroups()[0][0])

	err = pythonCollection.Update(
		ctx,
		WithIDs("py-2"),
		WithTexts("python document 2 updated by go"),
		WithEmbeddings(embeddings.NewEmbeddingFromFloat32([]float32{0.0, 1.0, 0.0})),
	)
	require.NoError(t, err)

	updatedResult, err := pythonCollection.Get(ctx, WithIDs("py-2"), WithInclude(IncludeDocuments))
	require.NoError(t, err)
	require.Len(t, updatedResult.GetDocuments(), 1)
	require.Equal(t, "python document 2 updated by go", updatedResult.GetDocuments()[0].ContentString())
}

func TestLocalPersistenceRoundTrip_GoAndDocker(t *testing.T) {
	ctx := context.Background()
	persistPath := newCrossLangPersistDir(t)
	goCollectionName := fmt.Sprintf("go_local_docker_%d", time.Now().UnixNano())
	dockerCollectionName := fmt.Sprintf("docker_local_go_%d", time.Now().UnixNano())

	localClient := newLocalClientForRoundTrip(t, persistPath)
	localClosed := false
	closeLocal := func() {
		if !localClosed {
			require.NoError(t, localClient.Close())
			localClosed = true
		}
	}
	t.Cleanup(closeLocal)

	goCollection, err := localClient.GetOrCreateCollection(
		ctx,
		goCollectionName,
		WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()),
	)
	require.NoError(t, err)

	err = goCollection.Add(
		ctx,
		WithIDs("go-d1", "go-d2", "go-d3"),
		WithTexts("go docker document 1", "go docker document 2", "go docker document 3"),
		WithEmbeddings(oneHotEmbeddings()...),
	)
	require.NoError(t, err)

	closeLocal()

	baseURL, terminateContainer := runMountedDockerChroma(t, persistPath)
	containerTerminated := false
	terminate := func() {
		if !containerTerminated {
			terminateContainer()
			containerTerminated = true
		}
	}
	t.Cleanup(terminate)

	dockerClient, err := NewHTTPClient(
		WithBaseURL(baseURL),
		WithDatabaseAndTenant(DefaultDatabase, DefaultTenant),
	)
	require.NoError(t, err)
	defer dockerClient.Close()

	goCollectionFromDocker, err := dockerClient.GetCollection(ctx, goCollectionName)
	require.NoError(t, err)

	queryFromDocker, err := goCollectionFromDocker.Query(
		ctx,
		WithQueryEmbeddings(embeddings.NewEmbeddingFromFloat32([]float32{0.95, 0.05, 0.0})),
		WithNResults(1),
	)
	require.NoError(t, err)
	require.NotEmpty(t, queryFromDocker.GetIDGroups())
	require.Equal(t, DocumentID("go-d1"), queryFromDocker.GetIDGroups()[0][0])

	err = goCollectionFromDocker.Update(
		ctx,
		WithIDs("go-d2"),
		WithTexts("go docker document 2 updated by docker"),
		WithEmbeddings(embeddings.NewEmbeddingFromFloat32([]float32{0.0, 1.0, 0.0})),
	)
	require.NoError(t, err)

	dockerCollection, err := dockerClient.GetOrCreateCollection(
		ctx,
		dockerCollectionName,
		WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()),
	)
	require.NoError(t, err)

	err = dockerCollection.Add(
		ctx,
		WithIDs("docker-1", "docker-2", "docker-3"),
		WithTexts("docker document 1", "docker document 2", "docker document 3"),
		WithEmbeddings(oneHotEmbeddings()...),
	)
	require.NoError(t, err)

	dockerCount, err := dockerCollection.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, dockerCount)

	terminate()

	localClient = newLocalClientForRoundTrip(t, persistPath)
	localClosed = false

	dockerCollectionFromLocal, err := localClient.GetCollection(ctx, dockerCollectionName)
	require.NoError(t, err)

	queryFromLocal, err := dockerCollectionFromLocal.Query(
		ctx,
		WithQueryEmbeddings(embeddings.NewEmbeddingFromFloat32([]float32{0.05, 0.95, 0.0})),
		WithNResults(1),
	)
	require.NoError(t, err)
	require.NotEmpty(t, queryFromLocal.GetIDGroups())
	require.Equal(t, DocumentID("docker-2"), queryFromLocal.GetIDGroups()[0][0])

	err = dockerCollectionFromLocal.Update(
		ctx,
		WithIDs("docker-2"),
		WithTexts("docker document 2 updated by go local"),
		WithEmbeddings(embeddings.NewEmbeddingFromFloat32([]float32{0.0, 1.0, 0.0})),
	)
	require.NoError(t, err)

	updatedDockerDoc, err := dockerCollectionFromLocal.Get(ctx, WithIDs("docker-2"), WithInclude(IncludeDocuments))
	require.NoError(t, err)
	require.Len(t, updatedDockerDoc.GetDocuments(), 1)
	require.Equal(t, "docker document 2 updated by go local", updatedDockerDoc.GetDocuments()[0].ContentString())

	goCollectionFromLocal, err := localClient.GetCollection(ctx, goCollectionName)
	require.NoError(t, err)

	updatedGoDoc, err := goCollectionFromLocal.Get(ctx, WithIDs("go-d2"), WithInclude(IncludeDocuments))
	require.NoError(t, err)
	require.Len(t, updatedGoDoc.GetDocuments(), 1)
	require.Equal(t, "go docker document 2 updated by docker", updatedGoDoc.GetDocuments()[0].ContentString())
}
