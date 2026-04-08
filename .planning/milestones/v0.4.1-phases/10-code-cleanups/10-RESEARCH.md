# Phase 10: Code Cleanups - Research

**Researched:** 2026-03-26
**Domain:** Go internal package extraction, interface type correction, global state test isolation, MIME inference
**Confidence:** HIGH

## Summary

Phase 10 addresses four independent cleanup items tracked in issues #456, #461, #466, and #469. Each is a self-contained refactoring with clear mechanical steps. The phase creates a new `pkg/internal/pathutil` shared package, fixes a `*context.Context` pointer-to-interface anti-pattern in three providers, adds test-only unregister helpers to the embedding registry, and extends `resolveMIME` to infer MIME types from URL path extensions.

All four items are code-level changes within the existing codebase. No new dependencies, no external services, no migration. The risk is low because each change has a narrow blast radius and the existing test suite provides coverage for correctness validation.

**Primary recommendation:** Implement in two plans -- Plan 1 handles the shared pathutil package + DefaultContext fix (foundational changes), Plan 2 handles registry test cleanup + resolveMIME URL support (independent changes that benefit from Plan 1's patterns being settled).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** `pkg/internal/pathutil` contains only path safety functions: `ContainsDotDot`, `ValidateFilePath`, `SafePath`. No file-reading helpers like `resolveBytes` -- those stay provider-local since they interleave provider-specific MIME logic.
- **D-02:** Gemini, Voyage, and default_ef replace their local path safety implementations with imports from the shared package.
- **D-03:** Change `*context.Context` to `context.Context` directly in Gemini, Nomic, and Mistral provider structs. No deprecation or compatibility shim -- the compiler tells callers exactly what to fix.
- **D-04:** Update all internal usages that take address (`&ctx`) or dereference (`*c.DefaultContext`) to use the value type directly.
- **D-05:** Add unexported unregister helpers (e.g., `unregisterDense(name)`) for use in test cleanup only. Public registry API stays append-only.
- **D-06:** All registry tests that register providers must use `t.Cleanup` with unregister helpers to prevent global state leaks between test runs.
- **D-07:** Gemini and VoyageAI `resolveMIME` functions gain URL path extension extraction (parse URL, get `path.Ext` from URL path) as a fallback when `MIMEType` and `FilePath` are both empty.
- **D-08:** Query strings and fragments in URLs are stripped before extension extraction.

### Claude's Discretion
- Exact function signatures and internal implementation details of the shared pathutil package
- How to handle edge cases in URL MIME resolution (e.g., URLs with no extension)
- Whether to reorder fallback chain in resolveMIME (MIMEType -> FilePath -> URL -> error)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

## Architecture Patterns

### New Package: `pkg/internal/pathutil`

```
pkg/internal/
  cosignutil/         # existing
  downloadutil/       # existing
  pathutil/           # NEW
    pathutil.go       # ContainsDotDot, ValidateFilePath, SafePath
    pathutil_test.go  # unit tests
```

The `internal` directory already exists with two packages. Adding `pathutil` follows the established convention. The `internal` visibility means only code within the module can import it -- this is appropriate since these are safety utilities not meant for external consumers.

### Pattern: Extracting to Internal Package

The extraction is mechanical:
1. Create `pkg/internal/pathutil/pathutil.go` with exported functions
2. Replace local calls in `gemini/content.go`, `voyage/content.go`, and `default_ef/download_utils.go`
3. Remove the now-unused local implementations

**Function mapping:**

| Current Location | Current Name | New Name | Signature Change |
|-----------------|-------------|----------|------------------|
| `gemini/content.go:136` | `containsDotDot(path string) bool` | `pathutil.ContainsDotDot(path string) bool` | Exported |
| `voyage/content.go:144` | `containsDotDot(path string) bool` | `pathutil.ContainsDotDot(path string) bool` | Exported |
| `default_ef/download_utils.go:157` | `safePath(destPath, filename string) (string, error)` | `pathutil.SafePath(destPath, filename string) (string, error)` | Exported |

Note: `ValidateFilePath` is a new convenience function that combines `filepath.Clean` + `ContainsDotDot` check into a single call, replacing the inline pattern used in both `gemini/content.go:111-114` and `voyage/content.go:119-121`.

### Pattern: DefaultContext Type Fix

The `*context.Context` anti-pattern exists in three providers:

| Provider | File | Field | Line |
|----------|------|-------|------|
| Gemini | `gemini/gemini.go` | `Client.DefaultContext` | L52 |
| Nomic | `nomic/nomic.go` | `Client.DefaultContext` | L60 |
| Mistral | `mistral/mistral.go` | `Client.DefaultContext` | L36 |

Each follows the same pattern:
```go
// BEFORE (anti-pattern)
type Client struct {
    DefaultContext *context.Context
}
func applyDefaults(c *Client) error {
    if c.DefaultContext == nil {
        ctx := context.Background()
        c.DefaultContext = &ctx
    }
}
// Usage: *c.DefaultContext

// AFTER (correct)
type Client struct {
    DefaultContext context.Context
}
func applyDefaults(c *Client) error {
    if c.DefaultContext == nil {
        c.DefaultContext = context.Background()
    }
}
// Usage: c.DefaultContext
```

**Gemini-specific concern:** The `applyDefaults` function at L76 uses `*c.DefaultContext` when creating the genai client:
```go
c.Client, err = genai.NewClient(*c.DefaultContext, &genai.ClientConfig{...})
```
After the fix, this becomes `c.DefaultContext` directly.

**No external callers set DefaultContext directly.** The field is only accessed through the functional options pattern or `applyDefaults`. No tests set `DefaultContext` via struct literals. The change is fully internal.

### Pattern: Registry Unregister Helpers

The registry uses four package-level maps protected by a single `sync.RWMutex`:
- `denseFactories`
- `sparseFactories`
- `multimodalFactories`
- `contentFactories`

Two tests already demonstrate the cleanup pattern (L510-514, L538-542):
```go
t.Cleanup(func() {
    mu.Lock()
    delete(contentFactories, name)
    mu.Unlock()
})
```

The decision is to formalize this into unexported helpers:
```go
func unregisterDense(name string) {
    mu.Lock()
    delete(denseFactories, name)
    mu.Unlock()
}
```

Four helpers needed: `unregisterDense`, `unregisterSparse`, `unregisterMultimodal`, `unregisterContent`.

**All 20+ tests in `registry_test.go`** that call `Register*` need `t.Cleanup` with the corresponding unregister helper. Currently only 2 of them have cleanup.

### Pattern: resolveMIME URL Extension Inference

Both Gemini and Voyage `resolveMIME` follow identical logic. The fix adds a URL path extension fallback:

```go
// Current fallback chain: MIMEType -> FilePath ext -> error
// New fallback chain:     MIMEType -> FilePath ext -> URL path ext -> error

func resolveMIME(source *embeddings.BinarySource) (string, error) {
    if source == nil {
        return "", errors.New("source cannot be nil")
    }
    if source.MIMEType != "" {
        return source.MIMEType, nil
    }
    if source.FilePath != "" {
        ext := strings.ToLower(filepath.Ext(source.FilePath))
        if mime, ok := extToMIME[ext]; ok {
            return mime, nil
        }
    }
    // NEW: URL path extension fallback
    if source.URL != "" {
        u, err := url.Parse(source.URL)
        if err == nil {
            ext := strings.ToLower(filepath.Ext(u.Path))
            if mime, ok := extToMIME[ext]; ok {
                return mime, nil
            }
        }
    }
    return "", errors.New("MIME type is required: ...")
}
```

**Key detail:** `url.Parse` strips query strings and fragments from `u.Path` automatically -- `u.Path` is just the path component. Using `filepath.Ext(u.Path)` correctly extracts `.png` from `https://example.com/photo.png?token=xyz#section`. This satisfies D-08 without extra stripping logic.

**Test impact:** The existing test `TestConvertToGenaiContentURLMissingMIME` expects an error when a URL has no extension. After this fix, URLs with known extensions will succeed. The test uses `https://example.com/image-no-ext` (no extension), so it will still fail as expected. New tests should cover URLs with extensions.

### Anti-Patterns to Avoid
- **Moving resolveBytes to shared package:** Decision D-01 explicitly keeps `resolveBytes` provider-local because it interleaves provider-specific logic (Gemini takes a `context.Context` parameter; Voyage does not).
- **Exporting unregister helpers:** Decision D-05 keeps them unexported. The public API stays append-only.
- **Adding a WithContext option function:** The DefaultContext field is internal plumbing. None of the three providers expose a `WithContext` option. Don't add one as part of this cleanup.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| URL parsing for MIME inference | Manual string splitting to extract extension | `net/url.Parse` + `filepath.Ext(u.Path)` | Handles query strings, fragments, encoded paths correctly |
| Path traversal detection | Custom regex patterns | `filepath.Clean` + `strings.Split` + `slices.Contains` | The existing `containsDotDot` implementation is already correct and battle-tested |
| Tar path safety | Manual path joining | `filepath.Clean` + `filepath.Join` + prefix check | The existing `safePath` implementation handles OS path separator edge cases |

## Common Pitfalls

### Pitfall 1: Nil context.Context After Type Change
**What goes wrong:** Changing from `*context.Context` to `context.Context` means the zero value is `nil` (interfaces have nil zero value), not a nil pointer. The `applyDefaults` nil check still works because `context.Context` is an interface and can be compared to nil.
**Why it happens:** `context.Context` is an interface. Its zero value is nil, not a valid context.
**How to avoid:** The existing `if c.DefaultContext == nil` check works for both `*context.Context` and `context.Context`. No logic change needed -- just remove the pointer indirection.
**Warning signs:** Panics at runtime when passing a nil context to HTTP or genai clients.

### Pitfall 2: Registry Test Ordering Dependencies
**What goes wrong:** Tests that register providers with the same name fail because a prior test already registered that name and didn't clean up.
**Why it happens:** Go test execution order within a package is deterministic but can change across runs with `-shuffle` flag.
**How to avoid:** Every `Register*` call in tests must have a corresponding `t.Cleanup(func() { unregister*(name) })` immediately after registration.
**Warning signs:** Tests pass individually but fail when run together (`go test ./...`).

### Pitfall 3: resolveMIME Test Expectations
**What goes wrong:** Existing tests that expect `resolveMIME` to fail on URL sources with file extensions would break.
**Why it happens:** The new URL extension fallback changes the behavior for URLs that have recognizable extensions.
**How to avoid:** Audit all `resolveMIME` test cases. The Gemini test `TestConvertToGenaiContentURLMissingMIME` uses a URL without an extension (`image-no-ext`), so it still errors. But any test with `https://.../*.png` that expects an error would break.
**Warning signs:** Failed assertions in `resolveMIME` tests.

### Pitfall 4: Import Cycles with Internal Package
**What goes wrong:** The new `pathutil` package accidentally imports from `pkg/embeddings`, creating a cycle.
**Why it happens:** If pathutil needs types from the embeddings package.
**How to avoid:** The pathutil package should only use stdlib. Its functions take and return primitive types (string, bool, error). No imports from the project's own packages.
**Warning signs:** Compiler error: `import cycle not allowed`.

## Code Examples

### pathutil Package Implementation

```go
// Source: Extracted from gemini/content.go:136-138 and default_ef/download_utils.go:157-164
package pathutil

import (
    "os"
    "path/filepath"
    "slices"
    "strings"

    "github.com/pkg/errors"
)

// ContainsDotDot reports whether the cleaned path still contains ".." components.
func ContainsDotDot(path string) bool {
    return slices.Contains(strings.Split(filepath.ToSlash(path), "/"), "..")
}

// ValidateFilePath cleans a file path and checks for path traversal.
// Returns the cleaned path or an error if traversal is detected.
func ValidateFilePath(path string) (string, error) {
    cleaned := filepath.Clean(path)
    if ContainsDotDot(cleaned) {
        return "", errors.Errorf("file path %q contains path traversal", path)
    }
    return cleaned, nil
}

// SafePath validates that joining destPath with filename results in a path
// within destPath, preventing path traversal attacks from malicious tar entries.
func SafePath(destPath, filename string) (string, error) {
    destPath = filepath.Clean(destPath)
    targetPath := filepath.Join(destPath, filepath.Base(filename))
    if !strings.HasPrefix(targetPath, destPath+string(os.PathSeparator)) && targetPath != destPath {
        return "", errors.Errorf("invalid path: %q escapes destination directory", filename)
    }
    return targetPath, nil
}
```

### resolveMIME URL Extension Fallback

```go
// Source: New code for gemini/content.go and voyage/content.go
import "net/url"

func resolveMIME(source *embeddings.BinarySource) (string, error) {
    if source == nil {
        return "", errors.New("source cannot be nil")
    }
    if source.MIMEType != "" {
        return source.MIMEType, nil
    }
    if source.FilePath != "" {
        ext := strings.ToLower(filepath.Ext(source.FilePath))
        if mime, ok := extToMIME[ext]; ok {
            return mime, nil
        }
    }
    if source.URL != "" {
        u, err := url.Parse(source.URL)
        if err == nil {
            ext := strings.ToLower(filepath.Ext(u.Path))
            if mime, ok := extToMIME[ext]; ok {
                return mime, nil
            }
        }
    }
    return "", errors.New("MIME type is required: set BinarySource.MIMEType or use a file/URL with a known extension")
}
```

### Registry Unregister Helpers

```go
// Source: New code for pkg/embeddings/registry.go
func unregisterDense(name string) {
    mu.Lock()
    delete(denseFactories, name)
    mu.Unlock()
}

func unregisterSparse(name string) {
    mu.Lock()
    delete(sparseFactories, name)
    mu.Unlock()
}

func unregisterMultimodal(name string) {
    mu.Lock()
    delete(multimodalFactories, name)
    mu.Unlock()
}

func unregisterContent(name string) {
    mu.Lock()
    delete(contentFactories, name)
    mu.Unlock()
}
```

### DefaultContext Fix Pattern

```go
// Source: Applied to gemini/gemini.go, nomic/nomic.go, mistral/mistral.go

// Struct field change:
// BEFORE: DefaultContext *context.Context
// AFTER:  DefaultContext context.Context

// applyDefaults change:
// BEFORE:
//   if c.DefaultContext == nil {
//       ctx := context.Background()
//       c.DefaultContext = &ctx
//   }
// AFTER:
//   if c.DefaultContext == nil {
//       c.DefaultContext = context.Background()
//   }

// Usage change (Gemini only):
// BEFORE: genai.NewClient(*c.DefaultContext, ...)
// AFTER:  genai.NewClient(c.DefaultContext, ...)
```

## Affected Files Inventory

| File | Changes |
|------|---------|
| `pkg/internal/pathutil/pathutil.go` | NEW: shared path safety functions |
| `pkg/internal/pathutil/pathutil_test.go` | NEW: unit tests |
| `pkg/embeddings/gemini/content.go` | Replace `containsDotDot` with `pathutil.ContainsDotDot`, add URL fallback to `resolveMIME` |
| `pkg/embeddings/voyage/content.go` | Replace `containsDotDot` with `pathutil.ContainsDotDot`, add URL fallback to `resolveMIME` |
| `pkg/embeddings/default_ef/download_utils.go` | Replace `safePath` with `pathutil.SafePath` |
| `pkg/embeddings/gemini/gemini.go` | Change `DefaultContext` from `*context.Context` to `context.Context` |
| `pkg/embeddings/nomic/nomic.go` | Change `DefaultContext` from `*context.Context` to `context.Context` |
| `pkg/embeddings/mistral/mistral.go` | Change `DefaultContext` from `*context.Context` to `context.Context` |
| `pkg/embeddings/registry.go` | Add 4 unexported `unregister*` helpers |
| `pkg/embeddings/registry_test.go` | Add `t.Cleanup` with unregister calls to all registration tests |

## Project Constraints (from CLAUDE.md)

- Build tags: `basicv2`, `ef`, `rf` -- pathutil tests need no build tag (stdlib only)
- Always run `make lint` before committing
- Use `testify` for assertions
- Never panic in production code -- pathutil functions must return errors, not panic
- Conventional commits required
- `pkg/errors` package used for error wrapping (not `fmt.Errorf`)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | go test + testify |
| Config file | Makefile targets with build tags |
| Quick run command | `go test ./pkg/internal/pathutil/... ./pkg/embeddings/...` |
| Full suite command | `make test && make test-ef` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SC-1 | pathutil.ContainsDotDot matches existing behavior | unit | `go test ./pkg/internal/pathutil/... -run TestContainsDotDot -v` | Wave 0 |
| SC-2 | pathutil.ValidateFilePath rejects traversal | unit | `go test ./pkg/internal/pathutil/... -run TestValidateFilePath -v` | Wave 0 |
| SC-3 | pathutil.SafePath prevents tar escape | unit | `go test ./pkg/internal/pathutil/... -run TestSafePath -v` | Wave 0 |
| SC-4 | Gemini resolveBytes uses pathutil | unit | `go test ./pkg/embeddings/gemini/... -run TestResolveBytes -v` | Existing |
| SC-5 | Voyage resolveBytes uses pathutil | unit | `go test ./pkg/embeddings/voyage/... -run TestResolveBytes -v` | Existing |
| SC-6 | DefaultContext type change compiles | compile | `go build ./pkg/embeddings/...` | N/A |
| SC-7 | Registry tests isolate state | unit | `go test ./pkg/embeddings/ -run TestRegister -v -count=2` | Existing (needs cleanup) |
| SC-8 | resolveMIME infers from URL extension | unit | `go test ./pkg/embeddings/gemini/... -run TestResolveMIME -v` | Existing (needs URL cases) |
| SC-9 | resolveMIME strips query/fragment | unit | `go test ./pkg/embeddings/gemini/... -run TestResolveMIMEURL -v` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./pkg/internal/pathutil/... ./pkg/embeddings/... -count=1`
- **Per wave merge:** `make test && make test-ef && make lint`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `pkg/internal/pathutil/pathutil_test.go` -- covers SC-1, SC-2, SC-3
- [ ] New resolveMIME URL test cases in gemini and voyage content tests -- covers SC-8, SC-9
- [ ] Framework install: none needed -- Go test already available

## Open Questions

1. **ValidateFilePath signature detail**
   - What we know: Gemini and Voyage both do `filepath.Clean` + `containsDotDot` inline. A combined helper saves duplication.
   - What's unclear: Whether `ValidateFilePath` should also check for empty input or other edge cases.
   - Recommendation: Keep it simple -- Clean + ContainsDotDot only. Empty string handling is caller's responsibility (matches current behavior).

2. **URL extension MIME error message update**
   - What we know: The current error message says "set BinarySource.MIMEType or use a file with a known extension".
   - What's unclear: Whether to update the error message to mention URL extensions too.
   - Recommendation: Update to "set BinarySource.MIMEType or use a file/URL with a known extension" for clarity.

## Sources

### Primary (HIGH confidence)
- Source code analysis of all affected files in the repository
- `pkg/embeddings/gemini/content.go` -- containsDotDot and resolveMIME implementations
- `pkg/embeddings/voyage/content.go` -- identical containsDotDot and resolveMIME implementations
- `pkg/embeddings/default_ef/download_utils.go` -- safePath implementation
- `pkg/embeddings/registry.go` -- registry structure with mutex and factory maps
- `pkg/embeddings/registry_test.go` -- existing test patterns including 2 cleanup examples
- Go stdlib `net/url.Parse` -- `u.Path` does not include query or fragment (verified behavior)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - no new dependencies, purely stdlib + existing project packages
- Architecture: HIGH - new internal package follows existing `pkg/internal/` convention exactly
- Pitfalls: HIGH - all code paths inspected, test impact analyzed from source

**Research date:** 2026-03-26
**Valid until:** 2026-04-26 (stable -- internal refactoring, no external API dependencies)
