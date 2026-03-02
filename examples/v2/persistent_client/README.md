# Local Persistent Client Example

This example is a concise copy/paste starter for `NewPersistentClient`:

- Start a local persistent client with an explicit persistence path.
- Create/get a collection.
- Upsert sample documents.
- Run a semantic query.

## Run

```bash
cd examples/v2/persistent_client
go run .
```

## Expected Output (abridged)

```text
Persistence path: ./chroma_data_local_persistent
Collection: persistent_local_demo
Existing docs before upsert: 3
Docs after upsert: 3
Top query result: id=doc-2, document="Local persistence keeps data between application restarts."
Tip: run `go run .` again. If "Existing docs before upsert" is > 0, local persistence is working.
```

## Troubleshooting

- First run can be slower because `chroma-go-local` and the default embedding model may be downloaded.
- Persistent data is stored under `./chroma_data_local_persistent`.
- To reset local data for a clean run:

```bash
rm -rf ./chroma_data_local_persistent
```
