# Persistent Client - Go Examples

> **Reference**: [Original Documentation](https://docs.trychroma.com/docs/run-chroma/persistent-client)

## Overview

`chroma-go` supports a persistent client via `v2.NewPersistentClient(...)`.
It embeds Chroma in your Go process and persists data on disk, similar to Python's `PersistentClient`.

## Requirements

1. `NewPersistentClient` auto-downloads the matching `chroma-go-local` shim library by default.
2. Optional: set `CHROMA_LIB_PATH` (or pass `WithPersistentLibraryPath(...)`) to use a specific local library file instead.

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
	client, err := v2.NewPersistentClient(
		v2.WithPersistentPath("./chroma_data"),
		v2.WithPersistentAllowReset(true),
	)
	if err != nil {
		log.Fatalf("Error creating persistent client: %v", err)
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

## Runnable Example

A concise runnable example is available in the repository:

- [`examples/v2/persistent_client`](https://github.com/amikos-tech/chroma-go/tree/main/examples/v2/persistent_client)

Run it with:

```bash
cd examples/v2/persistent_client
go run .
```

## Persistent Client Options

Runtime options:

- `WithPersistentRuntimeMode(v2.PersistentRuntimeModeEmbedded)` or `WithPersistentRuntimeMode(v2.PersistentRuntimeModeServer)` - choose runtime mode (default: embedded).
- `WithPersistentPath(path)` - persistence directory.
- `WithPersistentPort(port)` - server-mode port (default `8000`; use `0` to auto-select an available port).
- `WithPersistentListenAddress(addr)` - server-mode bind address.
- `WithPersistentAllowReset(bool)` - enable `Reset`.
- `WithPersistentConfigPath(path)` - start runtime from YAML file (defaults to server mode).
- `WithPersistentRawYAML(yaml)` - start runtime from inline YAML (defaults to server mode).
- `WithPersistentLibraryPath(path)` - explicit library path (alternative to `CHROMA_LIB_PATH`).
- `WithPersistentLibraryVersion(tag)` - override auto-download release tag (default `v0.3.1`).
- `WithPersistentLibraryCacheDir(path)` - override local shim cache directory.
- `WithPersistentLibraryAutoDownload(false)` - disable auto-download fallback.

Pass regular `ClientOption`s (logger, tenant/database, headers, auth, etc):

- `WithPersistentClientOption(v2.WithDatabaseAndTenant("db", "tenant"))`
- `WithPersistentClientOptions(...)`

## Starting Server Mode from YAML Config

```go
client, err := v2.NewPersistentClient(
	v2.WithPersistentConfigPath("./chroma.yaml"),
	v2.WithPersistentClientOption(v2.WithLogger(myLogger)),
)
```

Or inline:

```go
client, err := v2.NewPersistentClient(
	v2.WithPersistentRawYAML(`
port: 8010
persist_path: "./chroma_data"
allow_reset: true
`),
)
```

## Notes

- `NewPersistentClient` still uses the same `Client` interface, so collection/query code remains unchanged.
- If you prefer an external server (Docker, CLI, Cloud), continue using `NewHTTPClient` / `NewCloudClient`.
- `WithPersistentConfigPath` and `WithPersistentRawYAML` are mutually exclusive.
