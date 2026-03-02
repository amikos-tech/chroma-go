//go:build basicv2 && !cloud
// +build basicv2,!cloud

package v2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalClientOptions_InvalidInputs_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		option  PersistentClientOption
		wantErr string
	}{
		{name: "runtime mode invalid", option: WithPersistentRuntimeMode(PersistentRuntimeMode("bad")), wantErr: "unsupported local runtime mode"},
		{name: "library path empty", option: WithPersistentLibraryPath(""), wantErr: "local library path cannot be empty"},
		{name: "library version empty", option: WithPersistentLibraryVersion(""), wantErr: "local library version cannot be empty"},
		{name: "library version slash", option: WithPersistentLibraryVersion("v1/evil"), wantErr: "only ASCII letters"},
		{name: "library version backslash", option: WithPersistentLibraryVersion("v1\\evil"), wantErr: "only ASCII letters"},
		{name: "library version parent path", option: WithPersistentLibraryVersion("../evil"), wantErr: "only ASCII letters"},
		{name: "library cache dir empty", option: WithPersistentLibraryCacheDir(""), wantErr: "local library cache dir cannot be empty"},
		{name: "config path empty", option: WithPersistentConfigPath(""), wantErr: "local config path cannot be empty"},
		{name: "raw yaml empty", option: WithPersistentRawYAML(""), wantErr: "local raw YAML cannot be empty"},
		{name: "persist path empty", option: WithPersistentPath(""), wantErr: "local persist path cannot be empty"},
		{name: "listen address empty", option: WithPersistentListenAddress(""), wantErr: "local listen address cannot be empty"},
		{name: "port negative", option: WithPersistentPort(-1), wantErr: "between 0 and 65535"},
		{name: "port too high", option: WithPersistentPort(70000), wantErr: "between 0 and 65535"},
		{name: "local client option nil", option: WithPersistentClientOption(nil), wantErr: "local client option cannot be nil"},
		{name: "local client options has nil", option: WithPersistentClientOptions(WithBaseURL("http://localhost:8000"), nil), wantErr: "index 1 cannot be nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultLocalClientConfig()
			err := tt.option(cfg)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestLocalClientOptions_ValidInputsMutateConfig_TableDriven(t *testing.T) {
	tests := []struct {
		name   string
		option PersistentClientOption
		assert func(t *testing.T, cfg *localClientConfig)
	}{
		{
			name:   "runtime mode server",
			option: WithPersistentRuntimeMode(PersistentRuntimeModeServer),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, PersistentRuntimeModeServer, cfg.runtimeMode)
			},
		},
		{
			name:   "library path set",
			option: WithPersistentLibraryPath("/tmp/libchroma_go_shim.so"),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, "/tmp/libchroma_go_shim.so", cfg.libraryPath)
			},
		},
		{
			name:   "library version set",
			option: WithPersistentLibraryVersion("0.2.0"),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, "0.2.0", cfg.libraryVersion)
			},
		},
		{
			name:   "library cache dir set",
			option: WithPersistentLibraryCacheDir("/tmp/chroma-cache"),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, "/tmp/chroma-cache", cfg.libraryCacheDir)
			},
		},
		{
			name:   "auto download disabled",
			option: WithPersistentLibraryAutoDownload(false),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.False(t, cfg.autoDownloadLibrary)
			},
		},
		{
			name:   "config path forces server mode",
			option: WithPersistentConfigPath("/tmp/chroma.yaml"),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, "/tmp/chroma.yaml", cfg.configPath)
				require.Equal(t, PersistentRuntimeModeServer, cfg.runtimeMode)
			},
		},
		{
			name:   "raw yaml forces server mode",
			option: WithPersistentRawYAML("port: 8001"),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, "port: 8001", cfg.rawYAML)
				require.Equal(t, PersistentRuntimeModeServer, cfg.runtimeMode)
			},
		},
		{
			name:   "persist path set",
			option: WithPersistentPath("/tmp/chroma-data"),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, "/tmp/chroma-data", cfg.persistPath)
			},
		},
		{
			name:   "listen address forces server mode",
			option: WithPersistentListenAddress("127.0.0.1"),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, "127.0.0.1", cfg.listenAddress)
				require.Equal(t, PersistentRuntimeModeServer, cfg.runtimeMode)
			},
		},
		{
			name:   "port set and forces server mode",
			option: WithPersistentPort(8001),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, 8001, cfg.port)
				require.Equal(t, PersistentRuntimeModeServer, cfg.runtimeMode)
			},
		},
		{
			name:   "allow reset set",
			option: WithPersistentAllowReset(true),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.True(t, cfg.allowReset)
			},
		},
		{
			name:   "single client option appended",
			option: WithPersistentClientOption(WithBaseURL("http://localhost:9001")),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Len(t, cfg.clientOptions, 1)
			},
		},
		{
			name:   "multiple client options appended",
			option: WithPersistentClientOptions(WithBaseURL("http://localhost:9001"), WithTimeout(0)),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Len(t, cfg.clientOptions, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultLocalClientConfig()
			err := tt.option(cfg)
			require.NoError(t, err)
			tt.assert(t, cfg)
		})
	}
}
