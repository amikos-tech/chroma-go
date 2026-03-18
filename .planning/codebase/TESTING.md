# Testing Patterns

**Analysis Date:** 2026-03-18

## Test Framework

**Runner:**
- `go test` is the underlying runner
- `gotestsum` is the standard wrapper for grouped runs and JUnit output (`Makefile`)

**Assertion Library:**
- `github.com/stretchr/testify/require` is heavily used
- `assert` appears in some provider/config tests
- `gopter` is used for property-based tests in parts of `pkg/api/v2`

**Run Commands:**
```bash
make test                 # V2 API suite (basicv2)
make test-cloud           # Cloud integration suite
make test-ef              # Embedding provider suites
make test-rf              # Reranking suites
make test-crosslang       # Go + Python cross-language checks
make test-local-load-smoke
make lint
```

## Test File Organization

**Location:**
- Tests are colocated with the code they cover (`pkg/api/v2/*_test.go`, `pkg/embeddings/<provider>/*_test.go`)
- Cross-language helper logic lives in `scripts/`
- CI orchestration lives in `.github/workflows/`

**Naming:**
- standard `*_test.go`
- some suites use specialized names like `*_integration_test.go`, `*_perf_helpers_test.go`, `*_crosslang_integration_test.go`

**Structure:**
- package-level unit tests around request validation, serialization, and client behavior
- integration tests for HTTP/cloud/runtime behavior
- provider-specific tests often include both config-roundtrip tests and env-gated live API tests

## Test Structure

**Patterns:**
- `t.Run(...)` subtests are common
- `httptest.NewServer(...)` is used heavily for HTTP client unit tests
- arrange/act/assert is visible even when not labeled
- `t.Setenv(...)` is used for env-sensitive flows

**Build Tags:**
- `basicv2` - core V2 API tests
- `cloud` - cloud-enabled tests
- `ef` - embedding provider tests
- `rf` - reranking provider tests
- `crosslang` - Go/Python parity checks
- `soak` - local runtime performance suites

## Mocking

**Patterns:**
- stdlib `httptest` replaces remote APIs for most client/provider unit tests
- lightweight mock structs are preferred over full mocking frameworks
- testcontainers are used instead of mocks for many integration cases

**What Gets Mocked:**
- HTTP endpoints
- credential providers
- env-var based configuration
- selected local runtime dependencies in focused tests

## Fixtures and Factories

**Test Data:**
- inline fixtures are common in provider tests
- property-based generators are used for some V2 model tests (`gopter`)
- checked-in runtime/model data exists under `data/` for some local flows

## Coverage

**Current Shape:**
- coverage artifacts are produced into repo-root files like `coverage.out`, `coverage-ef.out`, `coverage-rf.out`, `coverage-crosslang.out`
- JUnit XML files are also emitted into the repo root

**CI Coverage Strategy:**
- broad matrix across lint, Windows compile guard, cross-platform smoke, API versions, cloud, EF, RF, cross-language, and perf workflows
- many live provider tests are optional/env-gated rather than always-on

## Test Types

**Unit Tests:**
- validation, serialization, option behavior, registry/config round-trips
- examples: `pkg/api/v2/client_http_test.go`, `pkg/embeddings/registry_test.go`

**Integration Tests:**
- real Chroma containers via testcontainers
- embedded runtime tests for persistent/local client
- live provider tests when credentials are available

**Cross-Language Tests:**
- Go/Python parity and persistence checks via Python helpers in `scripts/`

**Performance Tests:**
- smoke and soak suites around local persistent runtime
- outputs written to `artifacts/perf/*`

## Common Patterns

**Async / External Dependency Gating:**
- `t.Skip(...)` is used frequently when credentials, runtimes, or platform support are missing
- Windows/platform-specific skips exist for filesystem permission and runtime artifact tests

**Compatibility Testing:**
- CI runs against multiple Chroma versions
- tests explicitly branch on server capability/version for some features

**Practical Expectation for New Work:**
- add colocated tests with matching build tags
- update examples/docs if public behavior changes
- for provider/config changes, cover `GetConfig()` and rebuild-from-config paths

---

*Testing analysis: 2026-03-18*
*Update when build tags, CI matrix, or test tooling changes*
