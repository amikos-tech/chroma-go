//go:build basicv2

package v2

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcchroma "github.com/testcontainers/testcontainers-go/modules/chroma"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func TestClientHTTPIntegration(t *testing.T) {
	ctx := context.Background()
	var chromaVersion = "1.0.5"
	var chromaImage = "ghcr.io/chroma-core/chroma"
	if os.Getenv("CHROMA_VERSION") != "" {
		chromaVersion = os.Getenv("CHROMA_VERSION")
	}
	if os.Getenv("CHROMA_IMAGE") != "" {
		chromaImage = os.Getenv("CHROMA_IMAGE")
	}
	cwd, err := os.Getwd()
	require.NoError(t, err)
	mounts := []HostMount{
		{
			Source: filepath.Join(cwd, "v1-config.yaml"),
			Target: "/config.yaml",
		},
	}

	req := testcontainers.ContainerRequest{
		Image:        fmt.Sprintf("%s:%s", chromaImage, chromaVersion),
		ExposedPorts: []string{"8000/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("8000/tcp"),
			wait.ForHTTP("/api/v2/heartbeat").WithStatusCodeMatcher(func(status int) bool {
				return status == 200
			}),
		),
		Env: map[string]string{
			"ALLOW_RESET": "true", // this does not work with 1.0.x
		},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			dockerMounts := make([]mount.Mount, 0)
			for _, mnt := range mounts {
				dockerMounts = append(dockerMounts, mount.Mount{
					Type:   mount.TypeBind,
					Source: mnt.Source,
					Target: mnt.Target,
				})
			}
			hostConfig.Mounts = dockerMounts
		},
	}
	chromaContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, chromaContainer.Terminate(ctx))
	})

	ip, err := chromaContainer.Host(ctx)
	require.NoError(t, err)
	port, err := chromaContainer.MappedPort(ctx, "8000")
	require.NoError(t, err)
	endpoint := fmt.Sprintf("http://%s:%s", ip, port.Port())

	chromaURL := os.Getenv("CHROMA_URL")
	if chromaURL == "" {
		chromaURL = endpoint
	}
	c, err := NewHTTPClient(WithBaseURL(chromaURL), WithDebug())
	require.NoError(t, err)

	t.Run("get version", func(t *testing.T) {
		v, err := c.GetVersion(ctx)
		require.NoError(t, err)
		if strings.HasPrefix(chromaVersion, "1.0") {
			require.Contains(t, v, "1.")
		} else {
			require.Equal(t, chromaVersion, v)
		}
	})
	t.Run("heartbeat", func(t *testing.T) {
		err := c.Heartbeat(ctx)
		require.NoError(t, err)
	})
	t.Run("get identity", func(t *testing.T) {
		id, err := c.GetIdentity(ctx)
		require.NoError(t, err)
		require.Equal(t, NewDefaultTenant().Name(), id.Tenant)
		require.Equal(t, 1, len(id.Databases))
		require.Equal(t, NewDefaultDatabase().Name(), id.Databases[0])
	})
	t.Run("get tenant", func(t *testing.T) {
		tenant, err := c.GetTenant(ctx, NewDefaultTenant().Name())
		require.NoError(t, err)
		require.Equal(t, NewDefaultTenant().Name(), tenant.Name())
	})
	t.Run("create tenant", func(t *testing.T) {
		tenant, err := c.CreateTenant(ctx, NewTenant("test"))
		require.NoError(t, err)
		require.Equal(t, "test", tenant.Name())
	})
	t.Run("list databases", func(t *testing.T) {
		databases, err := c.ListDatabases(ctx, NewDefaultTenant().Name())
		require.NoError(t, err)
		require.Equal(t, 1, len(databases))
		require.Equal(t, NewDefaultDatabase().Name(), databases[0].Name())
	})

	t.Run("get database", func(t *testing.T) {
		db, err := c.GetDatabase(ctx, NewDefaultTenant().Name(), NewDefaultDatabase().Name())
		require.NoError(t, err)
		require.Equal(t, NewDefaultDatabase().Name(), db.Name())
	})
	t.Run("create database", func(t *testing.T) {
		db, err := c.CreateDatabase(ctx, NewDefaultTenant().Name(), "test_database")
		require.NoError(t, err)
		require.Equal(t, "test_database", db.Name())
	})
	t.Run("delete database", func(t *testing.T) {
		_, err := c.CreateDatabase(ctx, NewDefaultTenant().Name(), "testdb_to_delete")
		require.NoError(t, err)
		err = c.DeleteDatabase(ctx, NewDefaultTenant().Name(), "testdb_to_delete")
		require.NoError(t, err)
	})

	t.Run("create collection", func(t *testing.T) {
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		require.Equal(t, "test_collection", collection.Name())
	})
	t.Run("get collection", func(t *testing.T) {
		newCollection, err := c.CreateCollection(ctx, "test_collection_2", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		collection, err := c.GetCollection(ctx, newCollection.Name(), WithEmbeddingFunctionGet(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		require.Equal(t, newCollection.Name(), collection.Name())
	})
	t.Run("list collections", func(t *testing.T) {
		_, err := c.CreateCollection(ctx, "test_collection_3", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		collections, err := c.ListCollections(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(collections), 1)
		collectionNames := make([]string, 0)
		for _, collection := range collections {
			collectionNames = append(collectionNames, collection.Name())
		}
		require.Contains(t, collectionNames, "test_collection_3")
	})
	t.Run("delete collection", func(t *testing.T) {
		newCollection, err := c.CreateCollection(ctx, "test_collection_4", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = c.DeleteCollection(ctx, newCollection.Name())
		require.NoError(t, err)
	})
	t.Run("count collections", func(t *testing.T) {
		_, err := c.CreateCollection(ctx, "test_collection_5", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		count, err := c.CountCollections(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, count, 1)
	})

	t.Run("reset", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
	})

	t.Run("create tenant, db and collection", func(t *testing.T) {
		tenant, err := c.CreateTenant(ctx, NewTenant("test"))
		require.NoError(t, err)
		require.Equal(t, "test", tenant.Name())
		db, err := c.CreateDatabase(ctx, tenant.Name(), "test_db")
		require.NoError(t, err)
		require.Equal(t, "test_db", db.Name())
		err = c.UseTenantAndDatabase(ctx, tenant.Name(), db.Name())
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		require.Equal(t, "test_collection", collection.Name())
		require.Equal(t, tenant.Name(), collection.Tenant().Name())
		require.Equal(t, db.Name(), collection.Database().Name())
	})
}

type HostMount struct {
	Source string
	Target string
}

func TestClientHTTPIntegrationWithBasicAuth(t *testing.T) {
	ctx := context.Background()
	var chromaVersion = "0.6.3"
	var chromaImage = "ghcr.io/chroma-core/chroma"
	if os.Getenv("CHROMA_VERSION") != "" {
		chromaVersion = os.Getenv("CHROMA_VERSION")
	}
	if os.Getenv("CHROMA_IMAGE") != "" {
		chromaImage = os.Getenv("CHROMA_IMAGE")
	}
	cwd, err := os.Getwd()
	require.NoError(t, err)
	mounts := []HostMount{
		{
			Source: filepath.Join(cwd, "server.htpasswd"),
			Target: "/chroma/chroma/server.htpasswd",
		},
	}

	req := testcontainers.ContainerRequest{
		Image:        fmt.Sprintf("%s:%s", chromaImage, chromaVersion),
		ExposedPorts: []string{"8000/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("8000/tcp"),
			wait.ForHTTP("/api/v2/heartbeat").WithStatusCodeMatcher(func(status int) bool {
				return status == 200
			}),
		),
		Env: map[string]string{
			"ALLOW_RESET":                          "true",
			"CHROMA_SERVER_AUTHN_CREDENTIALS_FILE": "/chroma/chroma/server.htpasswd",
			"CHROMA_SERVER_AUTHN_PROVIDER":         "chromadb.auth.basic_authn.BasicAuthenticationServerProvider",
		},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			dockerMounts := make([]mount.Mount, 0)
			for _, mnt := range mounts {
				dockerMounts = append(dockerMounts, mount.Mount{
					Type:   mount.TypeBind,
					Source: mnt.Source,
					Target: mnt.Target,
				})
			}
			hostConfig.Mounts = dockerMounts
		},
	}
	chromaContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, chromaContainer.Terminate(ctx))
	})

	ip, err := chromaContainer.Host(ctx)
	require.NoError(t, err)
	port, err := chromaContainer.MappedPort(ctx, "8000")
	require.NoError(t, err)
	endpoint := fmt.Sprintf("http://%s:%s", ip, port.Port())
	chromaURL := os.Getenv("CHROMA_URL")
	if chromaURL == "" {
		chromaURL = endpoint
	}
	t.Run("success auth", func(t *testing.T) {
		c, err := NewHTTPClient(WithBaseURL(chromaURL), WithDebug(), WithAuth(NewBasicAuthCredentialsProvider("admin", "password123")))
		require.NoError(t, err)
		require.NotNil(t, c)
		collections, err := c.ListCollections(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, len(collections))
	})
	t.Run("wrong auth", func(t *testing.T) {
		wrongAuthClient, err := NewHTTPClient(WithBaseURL(chromaURL), WithDebug(), WithAuth(NewBasicAuthCredentialsProvider("admin", "wrong_password")))
		require.NoError(t, err)
		_, err = wrongAuthClient.ListCollections(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "403")
	})
}

func TestClientHTTPIntegrationWithBearerAuthorizationHeaderAuth(t *testing.T) {
	ctx := context.Background()
	var chromaVersion = "0.6.3"
	var chromaImage = "ghcr.io/chroma-core/chroma"
	if os.Getenv("CHROMA_VERSION") != "" {
		chromaVersion = os.Getenv("CHROMA_VERSION")
	}
	if os.Getenv("CHROMA_IMAGE") != "" {
		chromaImage = os.Getenv("CHROMA_IMAGE")
	}
	token := "chr0ma-t0k3n"

	req := testcontainers.ContainerRequest{
		Image:        fmt.Sprintf("%s:%s", chromaImage, chromaVersion),
		ExposedPorts: []string{"8000/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("8000/tcp"),
			wait.ForHTTP("/api/v2/heartbeat").WithStatusCodeMatcher(func(status int) bool {
				return status == 200
			}),
		),
		Env: map[string]string{
			"ALLOW_RESET":                        "true",
			"CHROMA_SERVER_AUTHN_CREDENTIALS":    token,
			"CHROMA_SERVER_AUTHN_PROVIDER":       "chromadb.auth.token_authn.TokenAuthenticationServerProvider",
			"CHROMA_AUTH_TOKEN_TRANSPORT_HEADER": "Authorization",
		},
	}
	chromaContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, chromaContainer.Terminate(ctx))
	})

	ip, err := chromaContainer.Host(ctx)
	require.NoError(t, err)
	port, err := chromaContainer.MappedPort(ctx, "8000")
	require.NoError(t, err)
	endpoint := fmt.Sprintf("http://%s:%s", ip, port.Port())
	//
	//chromaContainer, err := tcchroma.Run(ctx,
	//	fmt.Sprintf("%s:%s", chromaImage, chromaVersion),
	//	testcontainers.WithEnv(map[string]string{"ALLOW_RESET": "true"}),
	//	testcontainers.WithEnv(map[string]string{"CHROMA_SERVER_AUTHN_CREDENTIALS": token}),
	//	testcontainers.WithEnv(map[string]string{"CHROMA_SERVER_AUTHN_PROVIDER": "chromadb.auth.token_authn.TokenAuthenticationServerProvider"}),
	//	testcontainers.WithEnv(map[string]string{"CHROMA_AUTH_TOKEN_TRANSPORT_HEADER": "Authorization"}),
	//)
	//require.NoError(t, err)
	//t.Cleanup(func() {
	//	require.NoError(t, chromaContainer.Terminate(ctx))
	//})
	//endpoint, err := chromaContainer.RESTEndpoint(context.Background())
	//require.NoError(t, err)
	chromaURL := os.Getenv("CHROMA_URL")
	if chromaURL == "" {
		chromaURL = endpoint
	}
	t.Run("success auth", func(t *testing.T) {
		c, err := NewHTTPClient(WithBaseURL(chromaURL), WithDebug(), WithAuth(NewTokenAuthCredentialsProvider(token, AuthorizationTokenHeader)))
		require.NoError(t, err)
		require.NotNil(t, c)
		collections, err := c.ListCollections(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, len(collections))
	})
	t.Run("wrong auth", func(t *testing.T) {
		wrongAuthClient, err := NewHTTPClient(WithBaseURL(chromaURL), WithDebug(), WithAuth(NewTokenAuthCredentialsProvider("wrong_token", AuthorizationTokenHeader)))
		require.NoError(t, err)
		_, err = wrongAuthClient.ListCollections(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "403")
	})
}

func TestClientHTTPIntegrationWithBearerXChromaTokenHeaderAuth(t *testing.T) {
	ctx := context.Background()
	var chromaVersion = "0.6.3"
	var chromaImage = "ghcr.io/chroma-core/chroma"
	if os.Getenv("CHROMA_VERSION") != "" {
		chromaVersion = os.Getenv("CHROMA_VERSION")
	}
	if strings.HasPrefix(chromaVersion, "1.0") || chromaVersion == "latest" {
		t.Skip("Not supported by Chroma 1.0.x")
	}
	if os.Getenv("CHROMA_IMAGE") != "" {
		chromaImage = os.Getenv("CHROMA_IMAGE")
	}
	token := "chr0ma-t0k3n"

	req := testcontainers.ContainerRequest{
		Image:        fmt.Sprintf("%s:%s", chromaImage, chromaVersion),
		ExposedPorts: []string{"8000/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("8000/tcp"),
			wait.ForHTTP("/api/v2/heartbeat").WithStatusCodeMatcher(func(status int) bool {
				return status == 200
			}),
		),
		Env: map[string]string{
			"ALLOW_RESET":                        "true",
			"CHROMA_SERVER_AUTHN_CREDENTIALS":    token,
			"CHROMA_SERVER_AUTHN_PROVIDER":       "chromadb.auth.token_authn.TokenAuthenticationServerProvider",
			"CHROMA_AUTH_TOKEN_TRANSPORT_HEADER": "X-Chroma-Token",
		},
	}
	chromaContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, chromaContainer.Terminate(ctx))
	})

	ip, err := chromaContainer.Host(ctx)
	require.NoError(t, err)
	port, err := chromaContainer.MappedPort(ctx, "8000")
	require.NoError(t, err)
	endpoint := fmt.Sprintf("http://%s:%s", ip, port.Port())
	//
	//chromaContainer, err := tcchroma.Run(ctx,
	//	fmt.Sprintf("%s:%s", chromaImage, chromaVersion),
	//	testcontainers.WithEnv(map[string]string{"ALLOW_RESET": "true"}),
	//	testcontainers.WithEnv(map[string]string{"CHROMA_SERVER_AUTHN_CREDENTIALS": token}),
	//	testcontainers.WithEnv(map[string]string{"CHROMA_SERVER_AUTHN_PROVIDER": "chromadb.auth.token_authn.TokenAuthenticationServerProvider"}),
	//	testcontainers.WithEnv(map[string]string{"CHROMA_AUTH_TOKEN_TRANSPORT_HEADER": "X-Chroma-Token"}),
	//)
	//require.NoError(t, err)
	//t.Cleanup(func() {
	//	require.NoError(t, chromaContainer.Terminate(ctx))
	//})
	//endpoint, err := chromaContainer.RESTEndpoint(context.Background())
	//require.NoError(t, err)
	chromaURL := os.Getenv("CHROMA_URL")
	if chromaURL == "" {
		chromaURL = endpoint
	}
	t.Run("success auth", func(t *testing.T) {
		c, err := NewHTTPClient(WithBaseURL(chromaURL), WithDebug(), WithAuth(NewTokenAuthCredentialsProvider(token, XChromaTokenHeader)))
		require.NoError(t, err)
		require.NotNil(t, c)
		collections, err := c.ListCollections(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, len(collections))
	})
	t.Run("wrong auth", func(t *testing.T) {
		wrongAuthClient, err := NewHTTPClient(WithBaseURL(chromaURL), WithDebug(), WithAuth(NewTokenAuthCredentialsProvider("wrong_token", XChromaTokenHeader)))
		require.NoError(t, err)
		_, err = wrongAuthClient.ListCollections(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "403")
	})
}

func TestClientHTTPIntegrationWithSSL(t *testing.T) {

	ctx := context.Background()
	var chromaImage = "ghcr.io/chroma-core/chroma"
	var chromaVersion = "latest"
	if os.Getenv("CHROMA_VERSION") != "" {
		chromaVersion = os.Getenv("CHROMA_VERSION")
	}
	if strings.HasPrefix(chromaVersion, "1.0") || chromaVersion == "latest" {
		t.Skip("Not supported by Chroma 1.0.x")
	}

	if os.Getenv("CHROMA_IMAGE") != "" {
		chromaImage = os.Getenv("CHROMA_IMAGE")
	}
	tempDir := t.TempDir()
	certPath := fmt.Sprintf("%s/server.crt", tempDir)
	keyPath := fmt.Sprintf("%s/server.key", tempDir)
	containerCertPath := "/chroma/server.crt"
	containerKeyPath := "/chroma/server.key"

	cmd := []string{"--workers", "1",
		"--host", "0.0.0.0",
		"--port", "8000",
		"--proxy-headers",
		"--log-config", "/chroma/chromadb/log_config.yml",
		"--timeout-keep-alive", "30",
		"--ssl-certfile", containerCertPath,
		"--ssl-keyfile", containerKeyPath,
	}
	entrypoint := []string{}
	if chromaVersion != "latest" {
		cv := semver.MustParse(chromaVersion)
		if cv.LessThan(semver.MustParse("0.4.11")) {
			entrypoint = append(entrypoint, "/bin/bash", "-c")
			cmd = []string{fmt.Sprintf("pip install --force-reinstall --no-cache-dir chroma-hnswlib numpy==1.26.4 && ln -s /chroma/log_config.yml /chroma/chromadb/log_config.yml && uvicorn chromadb.app:app %s", strings.Join(cmd, " "))}
		} else if cv.LessThan(semver.MustParse("0.4.23")) {
			cmd = append([]string{"uvicorn", "chromadb.app:app"}, cmd...)
		}
	}

	CreateSelfSignedCert(certPath, keyPath)
	chromaContainer, err := tcchroma.Run(ctx,
		fmt.Sprintf("%s:%s", chromaImage, chromaVersion),
		testcontainers.WithEnv(map[string]string{"ALLOW_RESET": "true"}),
		testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				WaitingFor: wait.ForAll(
					wait.ForListeningPort("8000/tcp"),
				),
				Entrypoint: entrypoint,
				HostConfigModifier: func(hostConfig *container.HostConfig) {
					hostConfig.Mounts = []mount.Mount{
						{
							Type:   mount.TypeBind,
							Source: certPath,
							Target: containerCertPath,
						},
						{
							Type:   mount.TypeBind,
							Source: keyPath,
							Target: containerKeyPath,
						},
					}
				},
				Cmd: cmd,
			},
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, chromaContainer.Terminate(ctx))
	})
	endpoint, err := chromaContainer.RESTEndpoint(context.Background())
	require.NoError(t, err)
	chromaURL := os.Getenv("CHROMA_URL")
	if chromaURL == "" {
		chromaURL = endpoint
	}
	chromaURL = strings.ReplaceAll(endpoint, "http://", "https://")

	t.Run("Test with insecure client", func(t *testing.T) {
		client, err := NewHTTPClient(WithBaseURL(chromaURL), WithInsecure(), WithDebug())
		require.NoError(t, err)
		version, err := client.GetVersion(ctx)
		require.NoError(t, err)
		require.NotNil(t, version)
	})

	t.Run("Test without SSL", func(t *testing.T) {
		client, err := NewHTTPClient(WithBaseURL(chromaURL), WithDebug())
		require.NoError(t, err)
		_, err = client.GetVersion(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "x509: certificate signed by unknown authority")
	})

	t.Run("Test with SSL", func(t *testing.T) {
		client, err := NewHTTPClient(WithBaseURL(chromaURL), WithSSLCert(certPath), WithDebug())
		require.NoError(t, err)
		version, err := client.GetVersion(ctx)
		require.NoError(t, err)
		require.NotNil(t, version)
	})
}
