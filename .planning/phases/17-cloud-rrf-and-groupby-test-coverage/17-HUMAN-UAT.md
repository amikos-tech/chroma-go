---
status: partial
phase: 17-cloud-rrf-and-groupby-test-coverage
source: [17-VERIFICATION.md]
started: 2026-04-02T08:00:00Z
updated: 2026-04-02T08:00:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. RRF Live Cloud Execution
expected: Both subtests pass. Smoke test returns quantum doc (ID 1, 3, or 5) as first result. Weight test produces non-equal score slices.
command: `go test -tags=basicv2,cloud -run "TestCloudClientSearchRRF" -v -timeout=5m ./pkg/api/v2/...`
result: [pending]

### 2. GroupBy Live Cloud Execution
expected: Both subtests pass. MinK and MaxK subtests show per-category counts <= 2 across >= 2 categories.
command: `go test -tags=basicv2,cloud -run "TestCloudClientSearchGroupBy" -v -timeout=5m ./pkg/api/v2/...`
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
