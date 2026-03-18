# Coding Conventions

**Analysis Date:** 2026-03-18

## Naming Patterns

**Files:**
- lowercase Go filenames by concern (`client.go`, `collection.go`, `configuration.go`)
- provider packages usually use `<provider>.go` plus `option.go` and `*_test.go`
- examples and docs directories are feature-oriented (`examples/v2/search`, `docs/docs/embeddings.md`)

**Functions:**
- Constructors use `New...` (`NewHTTPClient`, `NewPersistentClient`, `NewOpenAIEmbeddingFunction`)
- Options use `With...` consistently across clients, collections, providers, and rerankers
- Validation helpers and request builders are explicit and verb-oriented (`PrepareAndValidateCollectionRequest`, `BuildEmbeddingFunctionFromConfig`)

**Variables:**
- camelCase for locals and fields
- exported constants/types use Go-standard PascalCase / ALL_CAPS where appropriate
- env-var names are explicit string constants in many provider packages

**Types:**
- interfaces for public contracts (`Client`, `Collection`, `EmbeddingFunction`, `RerankingFunction`)
- request/response structs tend to be package-local and JSON-oriented
- op structs capture option state before execution (`CreateCollectionOp`, `GetCollectionOp`)

## Code Style

**Formatting:**
- `gofmt` baseline
- `gci` import ordering via `.golangci.yml`
- standard Go comment and spacing conventions

**Linting:**
- `golangci-lint` is the canonical linter entry point
- repo enables `dupword`, `ginkgolinter`, `gocritic`, `mirror`, and full `staticcheck`
- examples are excluded from linting/formatting enforcement

## Import Organization

**Order:**
1. Standard library
2. Third-party packages
3. Repository-local imports with `github.com/amikos-tech/chroma-go/...`

**Grouping:**
- blank lines between groups
- import order is enforced by `gci`

## Error Handling

**Patterns:**
- return errors instead of panicking in runtime paths
- wrap errors with context using `github.com/pkg/errors`
- validate inputs early, especially in option setters and request-prep methods

**Error Types:**
- most code uses wrapped generic errors rather than custom error types
- validation messages are direct and user-readable
- nil/empty/invalid state checks are common before I/O

## Logging

**Framework:**
- logger abstraction in `pkg/logger`
- Zap bridge support exists for structured logging use cases

**Patterns:**
- logging is injected/configurable rather than hard-coded across the API surface
- tests often use quiet mocks or test loggers instead of production logging

## Comments

**When to Comment:**
- exported API surfaces are documented heavily with Go doc comments and examples
- comments explain behavior, compatibility, or API caveats rather than trivial code mechanics
- deprecations are explicitly documented in comments

**TODO Comments:**
- plain `// TODO ...` comments are used for follow-ups and gaps
- there are also explicit `Deprecated:` doc comments across compatibility layers

## Function Design

**Size:**
- public surface files are broad, but behavior is still decomposed into option setters, helper methods, and provider-specific types

**Parameters:**
- functional options are preferred over large parameter lists
- context is threaded through nearly all I/O-facing methods

**Return Values:**
- constructors usually return `(*Type, error)` or `(Interface, error)`
- collection and provider methods generally return typed results plus `error`

## Module Design

**Exports:**
- public APIs live in top-level package files with exported interfaces/types
- provider packages expose their own constructors/options directly

**Patterns Reused Across Repo:**
- functional options
- interface-driven provider abstraction
- build-tag partitioned tests
- env-var-backed config persistence for remote providers

## Project-Specific Guidance

- Prefer V2 API changes in `pkg/api/v2/`
- Add colocated tests with the correct build tags
- Avoid `Must*` patterns and runtime panics in production code; this is explicitly called out in `CLAUDE.md`
- When adding providers, match existing provider package layout and config round-trip behavior

---

*Convention analysis: 2026-03-18*
*Update when lint rules, option patterns, or public API style changes*
