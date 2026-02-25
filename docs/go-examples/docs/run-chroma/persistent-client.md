# Persistent Client - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/run-chroma/persistent-client)

## Overview

`chroma-go` supports a local persistent client via `v2.NewLocalClient(...)`.
It embeds Chroma in your Go process and persists data on disk, similar to Python's `PersistentClient`.

## Requirements

1. `NewLocalClient` auto-downloads the matching `chroma-go-local` shim library by default.
2. Optional: set `CHROMA_LIB_PATH` (or pass `WithLocalLibraryPath(...)`) to use a specific local library file instead.

Override example:

```bash
export CHROMA_LIB_PATH=/absolute/path/to/libchroma_go_shim.dylib
```

## Python vs Go Client

{% codetabs group="lang" %}
{% codetab label="Python" %}
```python
import chromadb

client = chromadb.PersistentClient(path="./chroma_data")
```
{% /codetab %}
{% codetab label="Go" %}
```go
package main

import (
	"context"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	client, err := v2.NewLocalClient(
		v2.WithLocalPersistPath("./chroma_data"),
		v2.WithLocalAllowReset(true),
	)
	if err != nil {
		log.Fatalf("Error creating local client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	col, err := client.GetOrCreateCollection(ctx, "persistent_collection")
	if err != nil {
		log.Fatalf("Error creating collection: %v", err)
	}

	err = col.Add(ctx,
		v2.WithIDs("doc1", "doc2"),
		v2.WithTexts("First document content", "Second document content"),
	)
	if err != nil {
		log.Fatalf("Error adding documents: %v", err)
	}
}
```
{% /codetab %}
{% /codetabs %}

## Local Client Options

Runtime options:

- `WithLocalRuntimeMode(v2.LocalRuntimeModeEmbedded|v2.LocalRuntimeModeServer)` - choose runtime mode (default: embedded).
- `WithLocalPersistPath(path)` - persistence directory.
- `WithLocalPort(port)` - server-mode port (default `8000`; use `0` to auto-select an available port).
- `WithLocalListenAddress(addr)` - server-mode bind address.
- `WithLocalAllowReset(bool)` - enable `Reset`.
- `WithLocalConfigPath(path)` - start runtime from YAML file (defaults to server mode).
- `WithLocalRawYAML(yaml)` - start runtime from inline YAML (defaults to server mode).
- `WithLocalLibraryPath(path)` - explicit library path (alternative to `CHROMA_LIB_PATH`).
- `WithLocalLibraryVersion(tag)` - override auto-download release tag (default `v0.2.0`).
- `WithLocalLibraryCacheDir(path)` - override local shim cache directory.
- `WithLocalLibraryAutoDownload(false)` - disable auto-download fallback.

Pass regular `ClientOption`s (logger, tenant/database, headers, auth, etc):

- `WithLocalClientOption(v2.WithDatabaseAndTenant("db", "tenant"))`
- `WithLocalClientOptions(...)`

## Starting Server Mode from YAML Config

```go
client, err := v2.NewLocalClient(
	v2.WithLocalConfigPath("./chroma.yaml"),
	v2.WithLocalClientOption(v2.WithLogger(myLogger)),
)
```

Or inline:

```go
client, err := v2.NewLocalClient(
	v2.WithLocalRawYAML(`
port: 8010
persist_path: "./chroma_data"
allow_reset: true
`),
)
```

## Notes

- `NewLocalClient` still uses the same `Client` interface, so collection/query code remains unchanged.
- If you prefer an external server (Docker, CLI, Cloud), continue using `NewHTTPClient` / `NewCloudClient`.
- `WithLocalConfigPath` and `WithLocalRawYAML` are mutually exclusive.
