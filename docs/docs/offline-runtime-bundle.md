# Offline Runtime Dependencies

Use this flow to pre-download and cache all native assets needed for Chroma’s default embedding/runtime path,
so smoke tests can run without network downloads.

It prepares:

- `local-shim` (`chroma-go-local` shared library)
- `onnx-runtime` (ONNX Runtime shared library)
- `tokenizers` (pure-tokenizers shared library)
- `onnx-models/all-MiniLM-L6-v2/onnx` (cached model and tokenizer)

## Prepare dependencies

```bash
./scripts/fetch_runtime_deps.sh
```

Optional overrides:

- `--output-dir`
- `--goos` (must match the host platform for now)
- `--goarch` (must match the host platform for now)
- `--local-shim-version`
- `--tokenizers-version`
- `--onnx-runtime-version`
- `--help`

## Use

The script writes an env helper file at `artifacts/runtime-deps/runtime-env.sh`.

```bash
. artifacts/runtime-deps/runtime-env.sh
RUN_DEFAULT_EF_BOOTSTRAP_SMOKE=1 \
go test -v -count=1 -run '^TestDefaultEF_BootstrapSmoke$' ./pkg/embeddings/default_ef
```

It can also be executed via Make:

```bash
make offline-smoke
```

### Custom output directory

```bash
OFFLINE_RUNTIME_DEPS_DIR=/path/to/deps ./scripts/fetch_runtime_deps.sh
```

## Make targets

- `make offline-runtime-deps`: run `./scripts/fetch_runtime_deps.sh` into `$(OFFLINE_RUNTIME_DEPS_DIR)` (defaults to
  `./artifacts/runtime-deps`).
- `make offline-smoke`: prepare deps and run `TestDefaultEF_BootstrapSmoke` using the generated env.
