---
status: complete
phase: 14-delete-with-limit
source: [14-01-SUMMARY.md, 14-02-SUMMARY.md]
started: 2026-03-30T09:00:00Z
updated: 2026-03-30T09:05:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Unit tests pass
expected: Run `go test -tags=basicv2 -run TestDeleteWithLimit ./pkg/api/v2/...` — all 8 subtests pass covering option application, validation edge cases, and JSON marshaling.
result: pass

### 2. HTTP integration test passes
expected: Run `go test -tags=basicv2 -run TestCollectionDelete ./pkg/api/v2/...` — the "with where and limit" test case passes, proving limit round-trips through HTTP transport.
result: pass

### 3. WithLimit applies to Delete
expected: Calling `collection.Delete(ctx, WithLimit(10), WithWhere(...))` compiles and the limit value propagates to CollectionDeleteOp.Limit field.
result: pass

### 4. Validation rejects limit without filter
expected: `collection.Delete(ctx, WithLimit(10))` (no where/where_document) returns an error requiring a filter when limit is specified.
result: pass

### 5. Validation rejects limit <= 0
expected: `collection.Delete(ctx, WithLimit(0))` and `WithLimit(-1)` both return validation errors.
result: pass

### 6. Lint passes
expected: Run `make lint` — no new warnings or errors from the modified files (options.go, collection.go, client_local_embedded.go).
result: pass
note: gci alignment auto-fixed before verification (FilterOp/FilterIDOp padding in CollectionDeleteOp struct)

## Summary

total: 6
passed: 6
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none]
