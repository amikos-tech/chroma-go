# Performance Testing

This page documents the local/persistent Chroma soak/load validation harness.

## Goals

- Detect memory/goroutine/file-descriptor leaks in local runtime usage.
- Detect major query/write latency regressions.
- Verify persistence durability across local runtime restarts.
- Track persistence-store behavior over time (directory size + WAL growth).

## Harness Location

- `pkg/api/v2/client_local_perf_test.go`
- `pkg/api/v2/client_local_perf_helpers_test.go`

Build tags:

- `//go:build soak && !cloud`

## Profiles

### Smoke profile

Smoke is used for PR gating with strict threshold enforcement.
In CI, this workflow is triggered only on relevant local-runtime/perf path changes.

Default scenarios:

1. `embedded_synthetic_smoke` (90s)
2. `server_synthetic_smoke` (90s)
3. `embedded_churn_smoke` (35 create/close cycles)
4. `server_churn_smoke` (35 create/close cycles)

Run:

```bash
make test-local-load-smoke
```

### Soak profile

Soak is intended for nightly endurance runs in report-only mode.

Default scenarios:

1. `embedded_synthetic_soak` (20m)
2. `server_synthetic_soak` (20m)
3. `embedded_default_ef_soak` (10m, enabled by default in soak)
4. `server_default_ef_soak` (10m, enabled by default in soak)

Run:

```bash
make test-local-soak-nightly
```

## Workload Shape

The synthetic workload uses a single write lane plus read workers.

- Read operations: `Query` + `Get` (`~70% Query`, `~30% Get` across the read lane)
- Write operations: `Upsert` (and optional `Delete+reinsert`)
- Seed phase: batched `Add`
- Churn workload: repeated `NewPersistentClient`/`Close` lifecycle cycles with heartbeat checks
- Sampling period: 5s by default (configurable per scenario)

## Thresholds

### Hard-fail thresholds (smoke when `CHROMA_PERF_ENFORCE=true`)

1. Error rate must be 0
2. `Query` p95 <= 750ms
3. `Get` p95 <= 750ms
4. write p95 <= 1500ms
5. post-GC heap growth <= `max(30% of baseline heap, 64MiB)`
6. goroutine growth <= +8
7. FD growth <= +16 (when measurable)
8. durability check must pass for scenarios that require restart verification

### Report-only alerts (soak by default)

1. Heap slope alert:
   - synthetic: >3 MiB/min
   - `default_ef`: >8 MiB/min
2. Goroutine slope alert: >0.2/min
3. WAL anomaly: >=90% non-decreasing WAL samples and final WAL >4x median without record-count growth
4. Throughput drift: last quartile throughput >20% lower than first quartile

## Reports

Each scenario writes a JSON summary:

- `perf-summary-<profile>-<scenario>.json`

A profile-level Markdown summary is also generated:

- `perf-summary-<profile>.md`

The CI workflow publishes these artifacts and appends the Markdown summary to job output.

## Environment Variables

- `CHROMA_PERF_PROFILE` - `smoke` or `soak` (default: `smoke`)
- `CHROMA_PERF_ENFORCE` - `true`/`false` (default: `true` for smoke, `false` for soak)
- `CHROMA_PERF_INCLUDE_DEFAULT_EF` - include `default_ef` scenarios (default: `false` for smoke, `true` for soak)
- `CHROMA_PERF_REPORT_DIR` - directory for JSON/Markdown reports
- `CHROMA_PERF_ENABLE_DELETE_REINSERT` - enable delete+reinsert writes (default: `false`)

## Current Runtime Caveat

`Delete+reinsert` operations are disabled by default in this harness.

Reason: current local runtime builds can assert/abort under delete-heavy vector
index mutation (`hnswalg.h` integrity assertion). The harness keeps this path
behind `CHROMA_PERF_ENABLE_DELETE_REINSERT=true` so teams can opt in for
focused investigation without destabilizing baseline smoke/soak gates.
