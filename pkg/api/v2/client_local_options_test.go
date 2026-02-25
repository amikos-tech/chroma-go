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
		option  LocalClientOption
		wantErr string
	}{
		{name: "runtime mode invalid", option: WithLocalRuntimeMode(LocalRuntimeMode("bad")), wantErr: "unsupported local runtime mode"},
		{name: "library path empty", option: WithLocalLibraryPath(""), wantErr: "local library path cannot be empty"},
		{name: "library version empty", option: WithLocalLibraryVersion(""), wantErr: "local library version cannot be empty"},
		{name: "library version slash", option: WithLocalLibraryVersion("v1/evil"), wantErr: "path separators are not allowed"},
		{name: "library version backslash", option: WithLocalLibraryVersion("v1\\evil"), wantErr: "path separators are not allowed"},
		{name: "library version parent path", option: WithLocalLibraryVersion("../evil"), wantErr: "path separators are not allowed"},
		{name: "library cache dir empty", option: WithLocalLibraryCacheDir(""), wantErr: "local library cache dir cannot be empty"},
		{name: "config path empty", option: WithLocalConfigPath(""), wantErr: "local config path cannot be empty"},
		{name: "raw yaml empty", option: WithLocalRawYAML(""), wantErr: "local raw YAML cannot be empty"},
		{name: "persist path empty", option: WithLocalPersistPath(""), wantErr: "local persist path cannot be empty"},
		{name: "listen address empty", option: WithLocalListenAddress(""), wantErr: "local listen address cannot be empty"},
		{name: "port negative", option: WithLocalPort(-1), wantErr: "between 0 and 65535"},
		{name: "port too high", option: WithLocalPort(70000), wantErr: "between 0 and 65535"},
		{name: "local client option nil", option: WithLocalClientOption(nil), wantErr: "local client option cannot be nil"},
		{name: "local client options has nil", option: WithLocalClientOptions(WithBaseURL("http://localhost:8000"), nil), wantErr: "index 1 cannot be nil"},
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
		option LocalClientOption
		assert func(t *testing.T, cfg *localClientConfig)
	}{
		{
			name:   "runtime mode server",
			option: WithLocalRuntimeMode(LocalRuntimeModeServer),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, LocalRuntimeModeServer, cfg.runtimeMode)
			},
		},
		{
			name:   "library path set",
			option: WithLocalLibraryPath("/tmp/libchroma_go_shim.so"),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, "/tmp/libchroma_go_shim.so", cfg.libraryPath)
			},
		},
		{
			name:   "library version set",
			option: WithLocalLibraryVersion("0.2.0"),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, "0.2.0", cfg.libraryVersion)
			},
		},
		{
			name:   "library cache dir set",
			option: WithLocalLibraryCacheDir("/tmp/chroma-cache"),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, "/tmp/chroma-cache", cfg.libraryCacheDir)
			},
		},
		{
			name:   "auto download disabled",
			option: WithLocalLibraryAutoDownload(false),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.False(t, cfg.autoDownloadLibrary)
			},
		},
		{
			name:   "config path forces server mode",
			option: WithLocalConfigPath("/tmp/chroma.yaml"),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, "/tmp/chroma.yaml", cfg.configPath)
				require.Equal(t, LocalRuntimeModeServer, cfg.runtimeMode)
			},
		},
		{
			name:   "raw yaml forces server mode",
			option: WithLocalRawYAML("port: 8001"),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, "port: 8001", cfg.rawYAML)
				require.Equal(t, LocalRuntimeModeServer, cfg.runtimeMode)
			},
		},
		{
			name:   "persist path set",
			option: WithLocalPersistPath("/tmp/chroma-data"),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, "/tmp/chroma-data", cfg.persistPath)
			},
		},
		{
			name:   "listen address forces server mode",
			option: WithLocalListenAddress("127.0.0.1"),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, "127.0.0.1", cfg.listenAddress)
				require.Equal(t, LocalRuntimeModeServer, cfg.runtimeMode)
			},
		},
		{
			name:   "port set and forces server mode",
			option: WithLocalPort(8001),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Equal(t, 8001, cfg.port)
				require.Equal(t, LocalRuntimeModeServer, cfg.runtimeMode)
			},
		},
		{
			name:   "allow reset set",
			option: WithLocalAllowReset(true),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.True(t, cfg.allowReset)
			},
		},
		{
			name:   "single client option appended",
			option: WithLocalClientOption(WithBaseURL("http://localhost:9001")),
			assert: func(t *testing.T, cfg *localClientConfig) {
				require.Len(t, cfg.clientOptions, 1)
			},
		},
		{
			name:   "multiple client options appended",
			option: WithLocalClientOptions(WithBaseURL("http://localhost:9001"), WithTimeout(0)),
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
