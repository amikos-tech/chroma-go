package v2

import (
	"context"
	stderrors "errors"
	"strings"
	"time"

	"github.com/pkg/errors"

	localchroma "github.com/amikos-tech/chroma-go-local"
)

// LocalRuntimeMode controls how [NewLocalClient] hosts Chroma locally.
type LocalRuntimeMode string

const (
	// LocalRuntimeModeEmbedded runs Chroma in-process via the embedded Rust shim.
	LocalRuntimeModeEmbedded LocalRuntimeMode = "embedded"
	// LocalRuntimeModeServer runs Chroma in-process but serves the HTTP API locally.
	LocalRuntimeModeServer LocalRuntimeMode = "server"
)

type localServer interface {
	URL() string
	Close() error
}

type localEmbeddedRuntime interface {
	Heartbeat() (uint64, error)
	Healthcheck() (*localchroma.EmbeddedHealthCheckResponse, error)
	MaxBatchSize() (uint32, error)
	CreateTenant(request localchroma.EmbeddedCreateTenantRequest) error
	GetTenant(request localchroma.EmbeddedGetTenantRequest) (*localchroma.EmbeddedTenant, error)
	UpdateTenant(request localchroma.EmbeddedUpdateTenantRequest) error
	CreateDatabase(request localchroma.EmbeddedCreateDatabaseRequest) error
	ListDatabases(request localchroma.EmbeddedListDatabasesRequest) ([]localchroma.EmbeddedDatabase, error)
	GetDatabase(request localchroma.EmbeddedGetDatabaseRequest) (*localchroma.EmbeddedDatabase, error)
	DeleteDatabase(request localchroma.EmbeddedDeleteDatabaseRequest) error
	CreateCollection(request localchroma.EmbeddedCreateCollectionRequest) (*localchroma.EmbeddedCollection, error)
	GetCollection(request localchroma.EmbeddedGetCollectionRequest) (*localchroma.EmbeddedCollection, error)
	DeleteCollection(request localchroma.EmbeddedDeleteCollectionRequest) error
	ListCollections(request localchroma.EmbeddedListCollectionsRequest) ([]localchroma.EmbeddedCollection, error)
	CountCollections(request localchroma.EmbeddedCountCollectionsRequest) (uint32, error)
	UpdateCollection(request localchroma.EmbeddedUpdateCollectionRequest) error
	ForkCollection(request localchroma.EmbeddedForkCollectionRequest) (*localchroma.EmbeddedCollection, error)
	Add(request localchroma.EmbeddedAddRequest) error
	UpsertRecords(request localchroma.EmbeddedUpsertRecordsRequest) error
	UpdateRecords(request localchroma.EmbeddedUpdateRecordsRequest) error
	DeleteRecords(request localchroma.EmbeddedDeleteRecordsRequest) error
	GetRecords(request localchroma.EmbeddedGetRecordsRequest) (*localchroma.EmbeddedGetRecordsResponse, error)
	CountRecords(request localchroma.EmbeddedCountRecordsRequest) (uint32, error)
	Query(request localchroma.EmbeddedQueryRequest) (*localchroma.EmbeddedQueryResponse, error)
	IndexingStatus(request localchroma.EmbeddedIndexingStatusRequest) (*localchroma.EmbeddedIndexingStatusResponse, error)
	Reset() error
	Close() error
}

const (
	localClientStartupTimeout      = 15 * time.Second
	localClientStartupPollInterval = 100 * time.Millisecond
)

var (
	localInitFunc      = localchroma.Init
	localNewServerFunc = func(opts ...localchroma.ServerOption) (localServer, error) {
		return localchroma.NewServer(opts...)
	}
	localStartServerFunc = func(config localchroma.StartServerConfig) (localServer, error) {
		return localchroma.StartServer(config)
	}
	localNewEmbeddedFunc = func(opts ...localchroma.EmbeddedOption) (localEmbeddedRuntime, error) {
		return localchroma.NewEmbedded(opts...)
	}
	localStartEmbeddedFunc = func(config localchroma.StartEmbeddedConfig) (localEmbeddedRuntime, error) {
		return localchroma.StartEmbedded(config)
	}
	localVersionWithErrorFunc   = localchroma.VersionWithError
	localWaitReadyFunc          = waitForLocalServerReady
	localWaitEmbeddedReadyFunc  = waitForLocalEmbeddedReady
	localResolveLibraryPathFunc = resolveLocalLibraryPath
)

// LocalClient hosts Chroma in-process using chroma-go-local.
//
// It delegates to either embedded mode (default) or local HTTP server mode,
// depending on runtime configuration.
type LocalClient struct {
	Client
	mode   LocalRuntimeMode
	server localServer
}

// LocalClientOption configures a [LocalClient].
type LocalClientOption func(*localClientConfig) error

type localClientConfig struct {
	runtimeMode LocalRuntimeMode

	libraryPath         string
	libraryVersion      string
	libraryCacheDir     string
	autoDownloadLibrary bool
	configPath          string
	rawYAML             string

	persistPath   string
	listenAddress string
	port          int
	allowReset    bool

	clientOptions []ClientOption
}

func defaultLocalClientConfig() *localClientConfig {
	serverDefaults := localchroma.DefaultServerConfig()
	embeddedDefaults := localchroma.DefaultEmbeddedConfig()
	return &localClientConfig{
		runtimeMode:         LocalRuntimeModeEmbedded,
		libraryVersion:      defaultLocalLibraryVersion,
		autoDownloadLibrary: true,
		persistPath:         embeddedDefaults.PersistPath,
		listenAddress:       serverDefaults.ListenAddress,
		port:                serverDefaults.Port,
		allowReset:          embeddedDefaults.AllowReset,
		clientOptions:       make([]ClientOption, 0),
	}
}

// NewLocalClient creates a Chroma client that starts and manages an in-process local Chroma runtime.
//
// Embedded mode is used by default. Use [WithLocalRuntimeMode] (or server-specific options such as [WithLocalPort])
// to run a local HTTP server mode instead.
func NewLocalClient(opts ...LocalClientOption) (Client, error) {
	cfg := defaultLocalClientConfig()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}
	if err := validateLocalConfigSource(cfg.configPath, cfg.rawYAML); err != nil {
		return nil, err
	}

	libraryPath, err := localResolveLibraryPathFunc(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "error resolving local chroma runtime library path")
	}

	if err := localInitFunc(libraryPath); err != nil {
		return nil, errors.Wrap(err, "error initializing local chroma runtime")
	}

	switch cfg.runtimeMode {
	case LocalRuntimeModeEmbedded:
		embedded, err := startLocalEmbedded(cfg)
		if err != nil {
			return nil, errors.Wrap(err, "error starting local chroma embedded runtime")
		}
		embeddedClient, err := newEmbeddedLocalClient(cfg, embedded)
		if err != nil {
			_ = embedded.Close()
			return nil, errors.Wrap(err, "error creating embedded local client")
		}
		return &LocalClient{Client: embeddedClient, mode: LocalRuntimeModeEmbedded}, nil

	case LocalRuntimeModeServer:
		server, err := startLocalServer(cfg)
		if err != nil {
			return nil, errors.Wrap(err, "error starting local chroma runtime")
		}

		clientOptions := make([]ClientOption, 0, len(cfg.clientOptions)+1)
		clientOptions = append(clientOptions, cfg.clientOptions...)
		clientOptions = append(clientOptions, WithBaseURL(server.URL()))

		httpClient, err := NewHTTPClient(clientOptions...)
		if err != nil {
			_ = server.Close()
			return nil, errors.Wrap(err, "error creating wrapped HTTP client for local runtime")
		}

		apiClient, ok := httpClient.(*APIClientV2)
		if !ok {
			_ = httpClient.Close()
			_ = server.Close()
			return nil, errors.New("unexpected client type returned by NewHTTPClient")
		}

		if err := localWaitReadyFunc(apiClient); err != nil {
			_ = apiClient.Close()
			_ = server.Close()
			return nil, errors.Wrap(err, "local runtime server failed readiness checks")
		}

		return &LocalClient{Client: apiClient, mode: LocalRuntimeModeServer, server: server}, nil

	default:
		return nil, errors.Errorf("unsupported local runtime mode: %s", cfg.runtimeMode)
	}
}

// BaseURL returns the local server URL when running in server mode.
// Embedded mode returns an empty string.
func (client *LocalClient) BaseURL() string {
	if client == nil || client.Client == nil {
		return ""
	}
	type baseURLProvider interface {
		BaseURL() string
	}
	if provider, ok := client.Client.(baseURLProvider); ok {
		return provider.BaseURL()
	}
	return ""
}

func waitForLocalServerReady(client *APIClientV2) error {
	if client == nil {
		return errors.New("client cannot be nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), localClientStartupTimeout)
	defer cancel()

	var lastErr error
	for {
		if err := client.Heartbeat(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if ctx.Err() != nil {
			break
		}
		timer := time.NewTimer(localClientStartupPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
	if lastErr == nil {
		lastErr = ctx.Err()
	}
	return lastErr
}

func waitForLocalEmbeddedReady(embedded localEmbeddedRuntime) error {
	if embedded == nil {
		return errors.New("embedded runtime cannot be nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), localClientStartupTimeout)
	defer cancel()

	var lastErr error
	for {
		health, err := embedded.Healthcheck()
		if err == nil && health != nil {
			if health.IsExecutorReady && health.IsLogClientReady {
				return nil
			}
			lastErr = errors.Errorf(
				"embedded runtime not ready: executor_ready=%t log_client_ready=%t",
				health.IsExecutorReady,
				health.IsLogClientReady,
			)
		} else {
			if _, hbErr := embedded.Heartbeat(); hbErr == nil {
				return nil
			} else if err != nil {
				lastErr = stderrors.Join(err, hbErr)
			} else {
				lastErr = hbErr
			}
		}

		if ctx.Err() != nil {
			break
		}
		timer := time.NewTimer(localClientStartupPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
	if lastErr == nil {
		lastErr = ctx.Err()
	}
	return lastErr
}

func validateLocalConfigSource(configPath, rawYAML string) error {
	if strings.TrimSpace(configPath) != "" && strings.TrimSpace(rawYAML) != "" {
		return errors.New("WithLocalConfigPath and WithLocalRawYAML are mutually exclusive")
	}
	return nil
}

func startLocalEmbedded(cfg *localClientConfig) (localEmbeddedRuntime, error) {
	if cfg == nil {
		return nil, errors.New("local client config cannot be nil")
	}
	if err := validateLocalConfigSource(cfg.configPath, cfg.rawYAML); err != nil {
		return nil, err
	}
	if cfg.configPath != "" {
		return localStartEmbeddedFunc(localchroma.StartEmbeddedConfig{ConfigPath: cfg.configPath})
	}
	if cfg.rawYAML != "" {
		return localStartEmbeddedFunc(localchroma.StartEmbeddedConfig{ConfigString: cfg.rawYAML})
	}

	opts := []localchroma.EmbeddedOption{
		localchroma.WithEmbeddedPersistPath(cfg.persistPath),
		localchroma.WithEmbeddedAllowReset(cfg.allowReset),
	}
	return localNewEmbeddedFunc(opts...)
}

func startLocalServer(cfg *localClientConfig) (localServer, error) {
	if cfg == nil {
		return nil, errors.New("local client config cannot be nil")
	}
	if err := validateLocalConfigSource(cfg.configPath, cfg.rawYAML); err != nil {
		return nil, err
	}
	if cfg.configPath != "" {
		return localStartServerFunc(localchroma.StartServerConfig{ConfigPath: cfg.configPath})
	}
	if cfg.rawYAML != "" {
		return localStartServerFunc(localchroma.StartServerConfig{ConfigString: cfg.rawYAML})
	}

	opts := []localchroma.ServerOption{
		// Let the runtime bind port 0 directly to avoid TOCTOU between probing and binding.
		localchroma.WithPort(cfg.port),
		localchroma.WithListenAddress(cfg.listenAddress),
		localchroma.WithPersistPath(cfg.persistPath),
		localchroma.WithAllowReset(cfg.allowReset),
	}
	return localNewServerFunc(opts...)
}

// Close shuts down the local runtime and any wrapped client resources.
func (client *LocalClient) Close() error {
	if client == nil {
		return nil
	}

	var errs []error

	if client.Client != nil {
		if err := client.Client.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if client.server != nil {
		if err := client.server.Close(); err != nil {
			errs = append(errs, errors.Wrap(err, "error closing local runtime server"))
		}
	}
	if len(errs) > 0 {
		return stderrors.Join(errs...)
	}
	return nil
}

func normalizeLocalRuntimeMode(mode LocalRuntimeMode) (LocalRuntimeMode, error) {
	value := strings.ToLower(strings.TrimSpace(string(mode)))
	switch value {
	case string(LocalRuntimeModeEmbedded):
		return LocalRuntimeModeEmbedded, nil
	case string(LocalRuntimeModeServer):
		return LocalRuntimeModeServer, nil
	default:
		return "", errors.Errorf("unsupported local runtime mode: %s", mode)
	}
}

// WithLocalRuntimeMode selects how [NewLocalClient] runs local Chroma.
func WithLocalRuntimeMode(mode LocalRuntimeMode) LocalClientOption {
	return func(cfg *localClientConfig) error {
		normalizedMode, err := normalizeLocalRuntimeMode(mode)
		if err != nil {
			return err
		}
		cfg.runtimeMode = normalizedMode
		return nil
	}
}

// WithLocalLibraryPath sets the path to the local runtime shared library.
//
// Resolution order in [NewLocalClient]:
// 1. [WithLocalLibraryPath]
// 2. `CHROMA_LIB_PATH`
// 3. Auto-download from `chroma-go-local` releases (when enabled)
func WithLocalLibraryPath(path string) LocalClientOption {
	return func(cfg *localClientConfig) error {
		if strings.TrimSpace(path) == "" {
			return errors.New("local library path cannot be empty")
		}
		cfg.libraryPath = strings.TrimSpace(path)
		return nil
	}
}

// WithLocalLibraryVersion sets the chroma-go-local release tag used for auto-downloading the runtime library.
//
// Examples: "v0.2.0", "0.2.0"
func WithLocalLibraryVersion(version string) LocalClientOption {
	return func(cfg *localClientConfig) error {
		version = strings.TrimSpace(version)
		if version == "" {
			return errors.New("local library version cannot be empty")
		}
		if err := validateLocalLibraryTag(version); err != nil {
			return err
		}
		cfg.libraryVersion = version
		return nil
	}
}

// WithLocalLibraryCacheDir sets the cache directory used for downloaded local runtime libraries.
func WithLocalLibraryCacheDir(path string) LocalClientOption {
	return func(cfg *localClientConfig) error {
		if strings.TrimSpace(path) == "" {
			return errors.New("local library cache dir cannot be empty")
		}
		cfg.libraryCacheDir = strings.TrimSpace(path)
		return nil
	}
}

// WithLocalLibraryAutoDownload enables/disables automatic library download when no explicit path is provided.
//
// Resolution order:
// 1. WithLocalLibraryPath(...)
// 2. CHROMA_LIB_PATH
// 3. auto-download (when enabled)
func WithLocalLibraryAutoDownload(enabled bool) LocalClientOption {
	return func(cfg *localClientConfig) error {
		cfg.autoDownloadLibrary = enabled
		return nil
	}
}

// WithLocalConfigPath starts local runtime from a YAML config file path.
//
// This option is mutually exclusive with [WithLocalRawYAML].
// It selects server runtime mode because YAML config startup is server-based.
func WithLocalConfigPath(path string) LocalClientOption {
	return func(cfg *localClientConfig) error {
		if strings.TrimSpace(path) == "" {
			return errors.New("local config path cannot be empty")
		}
		cfg.configPath = strings.TrimSpace(path)
		cfg.runtimeMode = LocalRuntimeModeServer
		return nil
	}
}

// WithLocalRawYAML starts local runtime from an inline YAML config string.
//
// This option is mutually exclusive with [WithLocalConfigPath].
// It selects server runtime mode because YAML config startup is server-based.
func WithLocalRawYAML(yaml string) LocalClientOption {
	return func(cfg *localClientConfig) error {
		if strings.TrimSpace(yaml) == "" {
			return errors.New("local raw YAML cannot be empty")
		}
		cfg.rawYAML = yaml
		cfg.runtimeMode = LocalRuntimeModeServer
		return nil
	}
}

// WithLocalPersistPath sets the local persistence directory.
func WithLocalPersistPath(path string) LocalClientOption {
	return func(cfg *localClientConfig) error {
		if strings.TrimSpace(path) == "" {
			return errors.New("local persist path cannot be empty")
		}
		cfg.persistPath = strings.TrimSpace(path)
		return nil
	}
}

// WithLocalListenAddress sets the local server listen address.
//
// This option selects server runtime mode.
func WithLocalListenAddress(address string) LocalClientOption {
	return func(cfg *localClientConfig) error {
		if strings.TrimSpace(address) == "" {
			return errors.New("local listen address cannot be empty")
		}
		cfg.listenAddress = strings.TrimSpace(address)
		cfg.runtimeMode = LocalRuntimeModeServer
		return nil
	}
}

// WithLocalPort sets the local server port.
//
// Use `0` to auto-select an available port.
// If unset, server mode defaults to port `8000`.
// This option selects server runtime mode.
func WithLocalPort(port int) LocalClientOption {
	return func(cfg *localClientConfig) error {
		if port < 0 || port > 65535 {
			return errors.New("local port must be between 0 and 65535")
		}
		cfg.port = port
		cfg.runtimeMode = LocalRuntimeModeServer
		return nil
	}
}

// WithLocalAllowReset enables/disables local reset behavior.
func WithLocalAllowReset(allowReset bool) LocalClientOption {
	return func(cfg *localClientConfig) error {
		cfg.allowReset = allowReset
		return nil
	}
}

// WithLocalClientOption adds one standard [ClientOption] to the wrapped local client state.
func WithLocalClientOption(option ClientOption) LocalClientOption {
	return func(cfg *localClientConfig) error {
		if option == nil {
			return errors.New("local client option cannot be nil")
		}
		cfg.clientOptions = append(cfg.clientOptions, option)
		return nil
	}
}

// WithLocalClientOptions adds multiple standard [ClientOption] values to the wrapped local client state.
func WithLocalClientOptions(options ...ClientOption) LocalClientOption {
	return func(cfg *localClientConfig) error {
		for i, option := range options {
			if option == nil {
				return errors.Errorf("local client option at index %d cannot be nil", i)
			}
			cfg.clientOptions = append(cfg.clientOptions, option)
		}
		return nil
	}
}
