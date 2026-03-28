---
phase: 13-collection-forkcount
plan: 01
subsystem: collection-api
tags: [fork-count, collection, v2-api]
dependency_graph:
  requires: []
  provides: [ForkCount-interface-method, ForkCount-http-impl, ForkCount-embedded-stub]
  affects: [pkg/api/v2/collection.go, pkg/api/v2/collection_http.go, pkg/api/v2/client_local_embedded.go]
tech_stack:
  added: []
  patterns: [GET+JSON-decode, unsupported-embedded-stub]
key_files:
  created: []
  modified:
    - pkg/api/v2/collection.go
    - pkg/api/v2/collection_http.go
    - pkg/api/v2/client_local_embedded.go
    - pkg/api/v2/collection_http_test.go
    - pkg/api/v2/client_local_test.go
decisions:
  - Follow IndexingStatus GET+JSON pattern for HTTP implementation
  - Return int (not int32) matching Count() signature convention
  - Embedded mode returns explicit unsupported error matching Fork() pattern
metrics:
  duration: 2min
  completed: "2026-03-28T15:04:36Z"
---

# Phase 13 Plan 01: Collection ForkCount Interface and Implementations Summary

ForkCount method added to Collection interface with HTTP GET /fork_count JSON decode and embedded unsupported stub, plus three unit tests.

## What Was Done

### Task 1: Add ForkCount to interface, HTTP implementation, and embedded stub

Added `ForkCount(ctx context.Context) (int, error)` to the `Collection` interface in `collection.go`. HTTP implementation in `collection_http.go` follows the IndexingStatus pattern: builds URL path with `/fork_count` suffix, issues GET via `ExecuteRequest`, and decodes the `{"count": n}` JSON response using a typed struct. Embedded implementation in `client_local_embedded.go` returns `"fork count is not supported in embedded local mode"`.

**Commit:** 1986caf

### Task 2: Add HTTP and embedded ForkCount unit tests

Added three tests:
- `TestCollectionForkCount`: httptest server returns `{"count":5}`, verifies result == 5
- `TestCollectionForkCountServerError`: httptest server returns 500, verifies error returned
- `TestEmbeddedCollection_ForkCountNotSupported`: zero-value embeddedCollection, verifies unsupported error message

**Commit:** cbed9e3

## Deviations from Plan

None - plan executed exactly as written.

## Verification Results

- `go build -tags=basicv2 ./pkg/api/v2/...` -- passed
- `go test -tags=basicv2 -run "ForkCount" ./pkg/api/v2/... -v` -- all 3 tests passed
- `make lint` -- 0 issues

## Known Stubs

None.

## Self-Check: PASSED
