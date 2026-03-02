# Local Persistent Client Example

This example demonstrates an end-to-end local persistence workflow using `NewPersistentClient`:

- Start a local persistent client with an explicit persistence path.
- Create/get a collection and upsert sample documents.
- Close and reopen the client from the same path.
- Verify persisted data with count checks and semantic query results.

## Run

```bash
cd examples/v2/persistent_client
go run .
```

## Expected Output (abridged)

```text
=== Phase 1: Write data to local persistent client ===
Persistence path: ./chroma_data_local_persistent
Count after write: 3
Phase 1 top result: id=doc-2, document="Local persistence keeps data between application restarts."
Phase 1 client closed.

=== Phase 2: Reopen client and verify persistence ===
Reopening from path: ./chroma_data_local_persistent
Count after reopen: 3
Phase 2 top result: id=doc-2, document="Local persistence keeps data between application restarts."
Persistence verified: collection "persistent_local_demo" retained 3 documents after client restart.
```

## Troubleshooting

- First run can be slower because `chroma-go-local` and the default embedding model may be downloaded.
- Persistent data is stored under `./chroma_data_local_persistent`.
- To reset local data for a clean run:

```bash
rm -rf ./chroma_data_local_persistent
```
