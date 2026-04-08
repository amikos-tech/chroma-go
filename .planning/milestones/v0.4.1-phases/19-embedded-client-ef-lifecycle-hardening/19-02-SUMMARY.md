---
phase: 19-embedded-client-ef-lifecycle-hardening
plan: 02
subsystem: embedded-client-ef-lifecycle
tags: [structured-logging, observability, embedded-client, logger]
dependency_graph:
  requires: [19-01]
  provides: [structured-logger-for-ef-lifecycle]
  affects: [pkg/api/v2/client_local.go, pkg/api/v2/client_local_embedded.go]
tech_stack:
  added: []
  patterns: [nil-logger-fallback, dual-propagation]
key_files:
  created: []
  modified:
    - pkg/api/v2/client_local.go
    - pkg/api/v2/client_local_embedded.go
    - pkg/api/v2/client_local_embedded_test.go
decisions:
  - embeddedLocalClient.logger is nil by default (NOT NoopLogger) to enable stderr fallback
  - WithPersistentLogger propagates to both embedded client (direct field) and state client (via WithLogger)
  - Auto-wire build errors use Warn level, close/cleanup errors use Error level
metrics:
  duration: 9min
  completed: "2026-04-06T10:02:00Z"
  tasks: 2
  files: 3
---

# Phase 19 Plan 02: Structured Logger for Embedded Client Summary

WithPersistentLogger option injects structured logger into embedded client for auto-wire and close error observability, with nil-logger fallback to stderr preserving backward compatibility.

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | dbcf252 | feat(19-02): add WithPersistentLogger option and wire structured logger into embedded client |
| 2 | b29e343 | test(19-02): add tests for structured logger receiving errors, stderr fallback, and propagation |

## Changes Made

### Task 1: WithPersistentLogger Option and Logger Wiring

**client_local.go:**
- Added `logger logger.Logger` field to `localClientConfig` struct
- Added `WithPersistentLogger(l logger.Logger) PersistentClientOption` that sets both `cfg.logger` and appends `WithLogger(l)` to `cfg.clientOptions` (dual propagation)

**client_local_embedded.go:**
- Added `logger logger.Logger` field to `embeddedLocalClient` struct (nil by default)
- `newEmbeddedLocalClient` passes `cfg.logger` to the struct literal
- Updated 4 logging callsites with `if client.logger != nil` guards:
  - GetCollection content EF auto-wire error: `logger.Warn("failed to auto-wire content embedding function")`
  - GetCollection dense EF auto-wire error: `logger.Warn("failed to auto-wire embedding function")`
  - Close() loop: `logger.Error("failed to close EF during client shutdown")`
  - deleteCollectionState: `logger.Error("failed to close EF during collection state cleanup")`
- All 4 sites fall back to stderr helpers when logger is nil

### Task 2: Three Logger Tests

1. **TestEmbeddedClient_LoggerReceivesErrors** with two subtests:
   - "auto-wire errors route to logger at Warn level": Registers a failing content factory, seeds a collection with that provider, calls GetCollection, asserts `warnCount >= 1`
   - "close errors route to logger at Error level": Injects a failing-close EF into collectionState, calls deleteCollectionState, asserts `errorCount >= 1`

2. **TestEmbeddedClient_NoLoggerFallsBackToStderr**: Creates client with nil logger, triggers close error via deleteCollectionState, captures stderr and asserts it contains `"chroma-go: failed to close EF"`

3. **TestWithPersistentLogger_PropagatesToStateClient**: Applies `WithPersistentLogger` to a real `localClientConfig`, verifies `cfg.logger` is set, verifies `cfg.clientOptions` contains at least one entry, creates a `BaseAPIClient` from those options and verifies the logger propagated

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Auto-wire test factory registration**
- **Found during:** Task 2 test execution
- **Issue:** Plan assumed nonexistent provider names would trigger build errors, but `BuildContentEFFromConfig` returns `nil, nil` for unregistered providers (no error to log)
- **Fix:** Registered a test content factory with `embeddingspkg.RegisterContent` that returns an intentional error, making the auto-wire path actually produce an error for the logger to capture
- **Files modified:** pkg/api/v2/client_local_embedded_test.go
- **Commit:** b29e343

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| embeddedLocalClient.logger nil by default | NoopLogger would swallow errors silently; nil enables stderr fallback for backward compatibility |
| WithPersistentLogger dual propagation | One call sets both embedded client logger and state client logger (via WithLogger in clientOptions) |
| Warn for auto-wire, Error for close | Auto-wire failures are non-fatal (collection works without EF); close failures indicate resource leaks |
| Register test factory for auto-wire test | Nonexistent providers return nil,nil not errors; registered failing factory triggers the actual error path |

## Verification

- `go build ./pkg/api/v2/...` - PASS
- `go test -tags=basicv2 -race -run "TestEmbeddedClient_Logger|TestWithPersistentLogger" -count=1 ./pkg/api/v2/...` - PASS
- `go test -tags=basicv2 -race -count=1 ./pkg/api/v2/...` - PASS (58s)
- `make lint` - PASS (0 issues)

## Self-Check: PASSED

- All 3 modified files exist on disk
- Both commits (dbcf252, b29e343) found in git log
- SUMMARY.md created at correct path
