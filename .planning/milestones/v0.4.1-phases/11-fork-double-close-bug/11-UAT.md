---
status: complete
phase: 11-fork-double-close-bug
source: [11-01-SUMMARY.md, 11-02-SUMMARY.md]
started: 2026-03-26T18:10:00Z
updated: 2026-03-26T18:15:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Close-once wrapper unit tests pass
expected: Run `go test -tags=basicv2 -v -run TestCloseOnce ./pkg/api/v2/...` — all tests pass covering idempotent close, use-after-close errors, delegation, unwrapper passthrough, nil safety, and ownership gating.
result: pass

### 2. Full V2 test suite passes (no regression)
expected: Run `make test` — all existing V2 tests pass. The fork double-close fix and ownership flag additions do not regress any existing collection or client behavior.
result: pass

### 3. Forked collection Close() does not close shared EF
expected: When a collection is forked via Fork(), the forked collection's Close() must NOT close the underlying embedding function. Only the parent (owning) collection's Close() should tear down the EF. Verify via: ownsEF is false on forked collections, close-once wrapper prevents double-close.
result: pass

### 4. Parent collection Close() still closes EF
expected: A collection obtained via CreateCollection or GetCollection has ownsEF=true. Calling Close() on it properly tears down the embedding function resources.
result: pass

### 5. Lint passes
expected: Run `make lint` — no new lint warnings or errors from the changes in ef_close_once.go, collection_http.go, client_local_embedded.go, or client_http.go.
result: pass

## Summary

total: 5
passed: 5
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none yet]
