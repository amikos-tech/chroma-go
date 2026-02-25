//go:build basicv2 && !cloud
// +build basicv2,!cloud

package v2

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	localchroma "github.com/amikos-tech/chroma-go-local"
)

type stubLocalServer struct {
	url        string
	closeCount int
	closeErr   error
}

func (s *stubLocalServer) URL() string {
	return s.url
}

func (s *stubLocalServer) Close() error {
	s.closeCount++
	return s.closeErr
}

type stubEmbeddedRuntime struct {
	closeCount int
	closeErr   error
}

func (s *stubEmbeddedRuntime) Heartbeat() (uint64, error) {
	return 1, nil
}

func (s *stubEmbeddedRuntime) Healthcheck() (*localchroma.EmbeddedHealthCheckResponse, error) {
	return &localchroma.EmbeddedHealthCheckResponse{
		IsExecutorReady:  true,
		IsLogClientReady: true,
	}, nil
}

func (s *stubEmbeddedRuntime) MaxBatchSize() (uint32, error) {
	return 128, nil
}

func (s *stubEmbeddedRuntime) CreateTenant(localchroma.EmbeddedCreateTenantRequest) error {
	return nil
}

func (s *stubEmbeddedRuntime) GetTenant(localchroma.EmbeddedGetTenantRequest) (*localchroma.EmbeddedTenant, error) {
	return &localchroma.EmbeddedTenant{Name: DefaultTenant}, nil
}

func (s *stubEmbeddedRuntime) UpdateTenant(localchroma.EmbeddedUpdateTenantRequest) error {
	return nil
}

func (s *stubEmbeddedRuntime) CreateDatabase(localchroma.EmbeddedCreateDatabaseRequest) error {
	return nil
}

func (s *stubEmbeddedRuntime) ListDatabases(localchroma.EmbeddedListDatabasesRequest) ([]localchroma.EmbeddedDatabase, error) {
	return nil, nil
}

func (s *stubEmbeddedRuntime) GetDatabase(localchroma.EmbeddedGetDatabaseRequest) (*localchroma.EmbeddedDatabase, error) {
	return &localchroma.EmbeddedDatabase{Name: DefaultDatabase, Tenant: DefaultTenant}, nil
}

func (s *stubEmbeddedRuntime) DeleteDatabase(localchroma.EmbeddedDeleteDatabaseRequest) error {
	return nil
}

func (s *stubEmbeddedRuntime) CreateCollection(localchroma.EmbeddedCreateCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	return &localchroma.EmbeddedCollection{ID: "stub-id", Name: "stub", Tenant: DefaultTenant, Database: DefaultDatabase}, nil
}

func (s *stubEmbeddedRuntime) GetCollection(localchroma.EmbeddedGetCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	return &localchroma.EmbeddedCollection{ID: "stub-id", Name: "stub", Tenant: DefaultTenant, Database: DefaultDatabase}, nil
}

func (s *stubEmbeddedRuntime) DeleteCollection(localchroma.EmbeddedDeleteCollectionRequest) error {
	return nil
}

func (s *stubEmbeddedRuntime) ListCollections(localchroma.EmbeddedListCollectionsRequest) ([]localchroma.EmbeddedCollection, error) {
	return nil, nil
}

func (s *stubEmbeddedRuntime) CountCollections(localchroma.EmbeddedCountCollectionsRequest) (uint32, error) {
	return 0, nil
}

func (s *stubEmbeddedRuntime) UpdateCollection(localchroma.EmbeddedUpdateCollectionRequest) error {
	return nil
}

func (s *stubEmbeddedRuntime) ForkCollection(localchroma.EmbeddedForkCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	return &localchroma.EmbeddedCollection{ID: "fork-id", Name: "fork", Tenant: DefaultTenant, Database: DefaultDatabase}, nil
}

func (s *stubEmbeddedRuntime) Add(localchroma.EmbeddedAddRequest) error {
	return nil
}

func (s *stubEmbeddedRuntime) UpsertRecords(localchroma.EmbeddedUpsertRecordsRequest) error {
	return nil
}

func (s *stubEmbeddedRuntime) UpdateRecords(localchroma.EmbeddedUpdateRecordsRequest) error {
	return nil
}

func (s *stubEmbeddedRuntime) DeleteRecords(localchroma.EmbeddedDeleteRecordsRequest) error {
	return nil
}

func (s *stubEmbeddedRuntime) GetRecords(localchroma.EmbeddedGetRecordsRequest) (*localchroma.EmbeddedGetRecordsResponse, error) {
	return &localchroma.EmbeddedGetRecordsResponse{}, nil
}

func (s *stubEmbeddedRuntime) CountRecords(localchroma.EmbeddedCountRecordsRequest) (uint32, error) {
	return 0, nil
}

func (s *stubEmbeddedRuntime) Query(localchroma.EmbeddedQueryRequest) (*localchroma.EmbeddedQueryResponse, error) {
	return &localchroma.EmbeddedQueryResponse{}, nil
}

func (s *stubEmbeddedRuntime) IndexingStatus(localchroma.EmbeddedIndexingStatusRequest) (*localchroma.EmbeddedIndexingStatusResponse, error) {
	return &localchroma.EmbeddedIndexingStatusResponse{}, nil
}

func (s *stubEmbeddedRuntime) Reset() error {
	return nil
}

func (s *stubEmbeddedRuntime) Close() error {
	s.closeCount++
	return s.closeErr
}

func TestDefaultLocalClientConfig_UsesRequestedDefaults(t *testing.T) {
	cfg := defaultLocalClientConfig()

	require.Equal(t, LocalRuntimeModeEmbedded, cfg.runtimeMode)
	require.Equal(t, defaultLocalLibraryVersion, cfg.libraryVersion)
	require.Equal(t, localchroma.DefaultServerConfig().Port, cfg.port)
}

func TestNewLocalClient_DefaultsToEmbeddedRuntime(t *testing.T) {
	lockLocalTestHooks(t)

	origInit := localInitFunc
	origResolve := localResolveLibraryPathFunc
	origNewEmbedded := localNewEmbeddedFunc
	origStartEmbedded := localStartEmbeddedFunc
	origWaitEmbedded := localWaitEmbeddedReadyFunc
	origNewServer := localNewServerFunc
	origStartServer := localStartServerFunc
	t.Cleanup(func() {
		localInitFunc = origInit
		localResolveLibraryPathFunc = origResolve
		localNewEmbeddedFunc = origNewEmbedded
		localStartEmbeddedFunc = origStartEmbedded
		localWaitEmbeddedReadyFunc = origWaitEmbedded
		localNewServerFunc = origNewServer
		localStartServerFunc = origStartServer
	})

	localResolveLibraryPathFunc = func(cfg *localClientConfig) (string, error) {
		return cfg.libraryPath, nil
	}

	var gotInitPath string
	localInitFunc = func(path string) error {
		gotInitPath = path
		return nil
	}

	stubRuntime := &stubEmbeddedRuntime{}
	var embeddedCfg *localchroma.EmbeddedConfig
	localNewEmbeddedFunc = func(opts ...localchroma.EmbeddedOption) (localEmbeddedRuntime, error) {
		cfg := localchroma.DefaultEmbeddedConfig()
		for _, opt := range opts {
			opt(cfg)
		}
		embeddedCfg = cfg
		return stubRuntime, nil
	}
	localStartEmbeddedFunc = func(config localchroma.StartEmbeddedConfig) (localEmbeddedRuntime, error) {
		t.Fatalf("did not expect StartEmbedded path, got: %+v", config)
		return nil, nil
	}
	localWaitEmbeddedReadyFunc = func(embedded localEmbeddedRuntime) error { return nil }

	localNewServerFunc = func(opts ...localchroma.ServerOption) (localServer, error) {
		t.Fatal("server runtime should not start by default")
		return nil, nil
	}
	localStartServerFunc = func(config localchroma.StartServerConfig) (localServer, error) {
		t.Fatal("server runtime should not start by default")
		return nil, nil
	}

	client, err := NewLocalClient(
		WithLocalLibraryPath("/tmp/chroma-go-shim/libchroma_go_shim.so"),
		WithLocalPersistPath("/tmp/chroma-embedded"),
		WithLocalAllowReset(true),
	)
	require.NoError(t, err)

	localClient, ok := client.(*LocalClient)
	require.True(t, ok)
	require.Equal(t, LocalRuntimeModeEmbedded, localClient.mode)
	require.Equal(t, "", localClient.BaseURL())
	require.Equal(t, "/tmp/chroma-go-shim/libchroma_go_shim.so", gotInitPath)
	require.NotNil(t, embeddedCfg)
	require.Equal(t, "/tmp/chroma-embedded", embeddedCfg.PersistPath)
	require.Equal(t, localchroma.DefaultEmbeddedConfig().SQLiteFilename, embeddedCfg.SQLiteFilename)
	require.True(t, embeddedCfg.AllowReset)

	require.NoError(t, localClient.Close())
	require.Equal(t, 1, stubRuntime.closeCount)
}

func TestNewLocalClient_UsesBuilderOptionsAndWrapsHTTPClient(t *testing.T) {
	lockLocalTestHooks(t)

	origInit := localInitFunc
	origNew := localNewServerFunc
	origStart := localStartServerFunc
	origWait := localWaitReadyFunc
	origResolve := localResolveLibraryPathFunc
	t.Cleanup(func() {
		localInitFunc = origInit
		localNewServerFunc = origNew
		localStartServerFunc = origStart
		localWaitReadyFunc = origWait
		localResolveLibraryPathFunc = origResolve
	})
	localWaitReadyFunc = func(client *APIClientV2) error { return nil }
	localResolveLibraryPathFunc = func(cfg *localClientConfig) (string, error) {
		return cfg.libraryPath, nil
	}

	var gotInitPath string
	localInitFunc = func(path string) error {
		gotInitPath = path
		return nil
	}

	var capturedConfig *localchroma.ServerConfig
	server := &stubLocalServer{url: "http://127.0.0.1:8877"}
	localNewServerFunc = func(opts ...localchroma.ServerOption) (localServer, error) {
		cfg := localchroma.DefaultServerConfig()
		for _, opt := range opts {
			opt(cfg)
		}
		capturedConfig = cfg
		return server, nil
	}
	localStartServerFunc = func(config localchroma.StartServerConfig) (localServer, error) {
		t.Fatalf("did not expect StartServer path, got: %+v", config)
		return nil, nil
	}

	client, err := NewLocalClient(
		WithLocalLibraryPath("/tmp/chroma-go-shim/libchroma_go_shim.so"),
		WithLocalPersistPath("/tmp/chroma-data"),
		WithLocalListenAddress("127.0.0.1"),
		WithLocalPort(8877),
		WithLocalAllowReset(true),
		WithLocalClientOptions(
			WithDatabaseAndTenant("db_local", "tenant_local"),
			WithDefaultHeaders(map[string]string{"X-Test": "1"}),
		),
	)
	require.NoError(t, err)

	localClient, ok := client.(*LocalClient)
	require.True(t, ok)
	require.Equal(t, "/tmp/chroma-go-shim/libchroma_go_shim.so", gotInitPath)
	require.NotNil(t, capturedConfig)
	require.Equal(t, "/tmp/chroma-data", capturedConfig.PersistPath)
	require.Equal(t, localchroma.DefaultServerConfig().SQLiteFilename, capturedConfig.SQLiteFilename)
	require.Equal(t, "127.0.0.1", capturedConfig.ListenAddress)
	require.Equal(t, 8877, capturedConfig.Port)
	require.True(t, capturedConfig.AllowReset)

	require.Equal(t, "db_local", localClient.CurrentDatabase().Name())
	require.Equal(t, "tenant_local", localClient.CurrentTenant().Name())
	require.Equal(t, "http://127.0.0.1:8877/api/v2", localClient.BaseURL())

	require.NoError(t, localClient.Close())
	require.Equal(t, 1, server.closeCount)
}

func TestNewLocalClient_ServerModeBaseURLCannotBeOverridden(t *testing.T) {
	lockLocalTestHooks(t)

	origInit := localInitFunc
	origNew := localNewServerFunc
	origStart := localStartServerFunc
	origWait := localWaitReadyFunc
	origResolve := localResolveLibraryPathFunc
	t.Cleanup(func() {
		localInitFunc = origInit
		localNewServerFunc = origNew
		localStartServerFunc = origStart
		localWaitReadyFunc = origWait
		localResolveLibraryPathFunc = origResolve
	})
	localWaitReadyFunc = func(client *APIClientV2) error { return nil }
	localResolveLibraryPathFunc = func(cfg *localClientConfig) (string, error) {
		return cfg.libraryPath, nil
	}
	localInitFunc = func(path string) error { return nil }

	server := &stubLocalServer{url: "http://127.0.0.1:8878"}
	localNewServerFunc = func(_ ...localchroma.ServerOption) (localServer, error) {
		return server, nil
	}
	localStartServerFunc = func(config localchroma.StartServerConfig) (localServer, error) {
		t.Fatalf("did not expect StartServer path, got: %+v", config)
		return nil, nil
	}

	client, err := NewLocalClient(
		WithLocalRuntimeMode(LocalRuntimeModeServer),
		WithLocalLibraryPath("/tmp/chroma-go-shim/libchroma_go_shim.so"),
		WithLocalClientOption(WithBaseURL("https://example.invalid/api/v2")),
	)
	require.NoError(t, err)

	localClient, ok := client.(*LocalClient)
	require.True(t, ok)
	require.Equal(t, "http://127.0.0.1:8878/api/v2", localClient.BaseURL())

	require.NoError(t, localClient.Close())
	require.Equal(t, 1, server.closeCount)
}

func TestNewLocalClient_UsesRawYAMLStartPath(t *testing.T) {
	lockLocalTestHooks(t)

	origInit := localInitFunc
	origNew := localNewServerFunc
	origStart := localStartServerFunc
	origWait := localWaitReadyFunc
	origResolve := localResolveLibraryPathFunc
	t.Cleanup(func() {
		localInitFunc = origInit
		localNewServerFunc = origNew
		localStartServerFunc = origStart
		localWaitReadyFunc = origWait
		localResolveLibraryPathFunc = origResolve
	})
	localWaitReadyFunc = func(client *APIClientV2) error { return nil }
	localResolveLibraryPathFunc = func(cfg *localClientConfig) (string, error) {
		return cfg.libraryPath, nil
	}

	localInitFunc = func(path string) error { return nil }

	server := &stubLocalServer{url: "http://127.0.0.1:9900"}
	localNewServerFunc = func(_ ...localchroma.ServerOption) (localServer, error) {
		t.Fatal("did not expect NewServer builder path")
		return nil, nil
	}

	var capturedStartConfig localchroma.StartServerConfig
	localStartServerFunc = func(config localchroma.StartServerConfig) (localServer, error) {
		capturedStartConfig = config
		return server, nil
	}

	client, err := NewLocalClient(
		WithLocalRawYAML("port: 9900\npersist_path: \"/tmp/chroma\""),
	)
	require.NoError(t, err)
	require.Equal(t, "", capturedStartConfig.ConfigPath)
	require.Contains(t, capturedStartConfig.ConfigString, "port: 9900")
	require.NoError(t, client.Close())
	require.Equal(t, 1, server.closeCount)
}

func TestNewLocalClient_ClosesServerWhenWrappedHTTPClientCreationFails(t *testing.T) {
	lockLocalTestHooks(t)

	origInit := localInitFunc
	origNew := localNewServerFunc
	origStart := localStartServerFunc
	origWait := localWaitReadyFunc
	origResolve := localResolveLibraryPathFunc
	t.Cleanup(func() {
		localInitFunc = origInit
		localNewServerFunc = origNew
		localStartServerFunc = origStart
		localWaitReadyFunc = origWait
		localResolveLibraryPathFunc = origResolve
	})
	localWaitReadyFunc = func(client *APIClientV2) error { return nil }
	localResolveLibraryPathFunc = func(cfg *localClientConfig) (string, error) {
		return cfg.libraryPath, nil
	}

	localInitFunc = func(path string) error { return nil }

	server := &stubLocalServer{url: "http://127.0.0.1:8800"}
	localNewServerFunc = func(_ ...localchroma.ServerOption) (localServer, error) {
		return server, nil
	}
	localStartServerFunc = func(config localchroma.StartServerConfig) (localServer, error) {
		t.Fatalf("did not expect StartServer path, got: %+v", config)
		return nil, nil
	}

	// Force wrapped HTTP client creation failure via invalid client option.
	_, err := NewLocalClient(
		WithLocalRuntimeMode(LocalRuntimeModeServer),
		WithLocalClientOption(WithBaseURL("")),
	)
	require.Error(t, err)
	require.Equal(t, 1, server.closeCount)
}

func TestNewLocalClient_ClosesServerWhenReadinessCheckFails(t *testing.T) {
	lockLocalTestHooks(t)

	origInit := localInitFunc
	origNew := localNewServerFunc
	origStart := localStartServerFunc
	origWait := localWaitReadyFunc
	origResolve := localResolveLibraryPathFunc
	t.Cleanup(func() {
		localInitFunc = origInit
		localNewServerFunc = origNew
		localStartServerFunc = origStart
		localWaitReadyFunc = origWait
		localResolveLibraryPathFunc = origResolve
	})
	localResolveLibraryPathFunc = func(cfg *localClientConfig) (string, error) {
		return cfg.libraryPath, nil
	}

	localInitFunc = func(path string) error { return nil }
	server := &stubLocalServer{url: "http://127.0.0.1:8810"}
	localNewServerFunc = func(_ ...localchroma.ServerOption) (localServer, error) {
		return server, nil
	}
	localStartServerFunc = func(config localchroma.StartServerConfig) (localServer, error) {
		t.Fatalf("did not expect StartServer path, got: %+v", config)
		return nil, nil
	}
	localWaitReadyFunc = func(client *APIClientV2) error {
		return errors.New("not ready")
	}

	_, err := NewLocalClient(WithLocalRuntimeMode(LocalRuntimeModeServer))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed readiness checks")
	require.Equal(t, 1, server.closeCount)
}

func TestNewLocalClient_PropagatesInitError(t *testing.T) {
	lockLocalTestHooks(t)

	origInit := localInitFunc
	origNew := localNewServerFunc
	origStart := localStartServerFunc
	origWait := localWaitReadyFunc
	origResolve := localResolveLibraryPathFunc
	t.Cleanup(func() {
		localInitFunc = origInit
		localNewServerFunc = origNew
		localStartServerFunc = origStart
		localWaitReadyFunc = origWait
		localResolveLibraryPathFunc = origResolve
	})
	localWaitReadyFunc = func(client *APIClientV2) error { return nil }
	localResolveLibraryPathFunc = func(cfg *localClientConfig) (string, error) {
		return cfg.libraryPath, nil
	}

	expectedErr := errors.New("init failed")
	localInitFunc = func(path string) error { return expectedErr }
	localNewServerFunc = func(_ ...localchroma.ServerOption) (localServer, error) {
		t.Fatal("server should not start when init fails")
		return nil, nil
	}
	localStartServerFunc = func(config localchroma.StartServerConfig) (localServer, error) {
		t.Fatalf("did not expect StartServer path, got: %+v", config)
		return nil, nil
	}

	_, err := NewLocalClient()
	require.Error(t, err)
	require.Contains(t, err.Error(), "error initializing local chroma runtime")
}

func TestStartLocalServer_RejectsMutuallyExclusiveConfigSources(t *testing.T) {
	_, err := startLocalServer(&localClientConfig{
		configPath: "/tmp/chroma.yaml",
		rawYAML:    "port: 8801",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "mutually exclusive")
}

func TestNewLocalClient_PropagatesLibraryResolveError(t *testing.T) {
	lockLocalTestHooks(t)

	origResolve := localResolveLibraryPathFunc
	t.Cleanup(func() {
		localResolveLibraryPathFunc = origResolve
	})

	localResolveLibraryPathFunc = func(cfg *localClientConfig) (string, error) {
		return "", errors.New("resolve failed")
	}

	_, err := NewLocalClient()
	require.Error(t, err)
	require.Contains(t, err.Error(), "error resolving local chroma runtime library path")
}
