# Phase 4: Provider Mapping and Explicit Failures - Research

**Researched:** 2026-03-20
**Domain:** Go interface design, validation extension, opt-in capability contracts
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Intent mapping contract**
- Provider-owned mapping via a standalone `IntentMapper` interface (not embedded in `ContentEmbeddingFunction`)
- `IntentMapper` exposes `MapIntent(Intent) → (string, error)` — each provider owns its own mapping table
- When a provider does not implement `IntentMapper` and receives a non-empty intent, the intent string is passed through as-is to the provider API (not rejected)
- A shared `ValidateContentSupport(content, caps)` helper is available for providers to call before dispatch — validates modality, intent, and dimension against `CapabilityMetadata`

**Failure semantics**
- Unsupported-combination errors reuse the existing `ValidationError` type with new validation codes (`unsupported_intent`, `unsupported_modality`, `unsupported_dimension`)
- Capability validation happens eagerly via the shared pre-check helper before provider I/O
- For batch requests, validation fails on the first unsupported item (consistent with existing `ValidateContents` behavior)
- The shared pre-check validates all three: modality, intent, AND dimension support against `CapabilityMetadata`

**Provider adoption path**
- Phase 4 delivers the contract, helpers, and tests only — no real provider implements `IntentMapper` in this phase
- Real provider adoption happens in Phase 6 (Gemini) and Phase 7 (vLLM/Nemotron)
- Providers declare supported intents in `CapabilityMetadata.Intents`; the shared pre-check rejects unsupported neutral intents before `MapIntent` is called
- When a provider does not implement `CapabilityAware` (e.g., adapted legacy providers), the shared pre-check skips validation entirely and passes through

**Escape hatch behavior**
- `MapIntent` is always called for all intents (both neutral and custom/raw strings) — provider decides its own policy for custom values
- When both a neutral intent and a conflicting provider hint are set, the intent (portable field) wins per Phase 1 decision
- The shared pre-check skips intent validation against `CapabilityMetadata` when the intent is a non-neutral (custom) string — escape hatches bypass capability enforcement
- A public `IsNeutralIntent(Intent) bool` helper identifies whether an intent is one of the 5 shared neutral constants — used by pre-check and available to providers in `MapIntent`

### Claude's Discretion
- Concrete function signatures and parameter ordering for `ValidateContentSupport`
- Exact validation code string values for new unsupported-* codes
- Internal organization of the `IsNeutralIntent` helper (set vs switch)
- Test scaffolding structure for mock IntentMapper implementations

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| MAP-01 | Neutral intents are mapped to provider-native task and input semantics through a defined contract with tests | `IntentMapper` interface definition, `IsNeutralIntent` helper, mapping contract design, test scaffolding with mock mapper |
| MAP-02 | Unsupported modality or intent combinations fail explicitly instead of silently downgrading or guessing | `ValidateContentSupport` helper, three new validation codes, eager pre-check before I/O, batch-fail-on-first behavior |
</phase_requirements>

---

## Summary

Phase 4 is a pure contract-and-helper phase: no real provider wires up `IntentMapper` yet. The work consists of three tightly scoped additions to `pkg/embeddings/`:

1. **`IntentMapper` interface** — standalone opt-in interface (same pattern as `CapabilityAware`, `Closeable`) with `MapIntent(Intent) (string, error)`.
2. **`IsNeutralIntent(Intent) bool` helper** — placed in `multimodal.go` alongside the 5 neutral constants; a switch statement over the known constants is the correct implementation.
3. **`ValidateContentSupport(content Content, caps CapabilityMetadata) error` helper** — placed in `multimodal_validate.go` (or a new `content_validate.go`), using three new unexported validation code constants (`unsupported_modality`, `unsupported_intent`, `unsupported_dimension`).

All three use patterns that already exist in the codebase and will compile without touching any provider package. Tests use a mock `IntentMapper` defined in the test file itself, proving the contract without live providers. The planner should produce exactly 2 plans matching the phase outline: Plan 01 for the interface/helper/codes implementation, Plan 02 for adapter behavior validation and mapping tests.

**Primary recommendation:** Model `IntentMapper` as a standalone opt-in interface next to `CapabilityAware` in `embedding.go`; put `ValidateContentSupport` in `multimodal_validate.go`; put `IsNeutralIntent` in `multimodal.go`. Keep both additions to under 50 lines each.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/stretchr/testify` | already in go.mod | Assertions in tests | Used in every test file in the package |

No new dependencies are needed. Phase 4 is entirely within `pkg/embeddings/` using the standard library and patterns already in place.

### Supporting
No supporting libraries needed. The phase operates entirely on existing types.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| standalone `IntentMapper` interface | method on `ContentEmbeddingFunction` | Embedding it into the base interface would force all existing adapters to implement it; standalone opt-in is lighter and consistent with `CapabilityAware` |
| new validation codes as exported constants | embed message text inline | Unexported constants keep the pattern consistent with the 6 existing codes in `multimodal_validate.go` and allow tests to import them via the internal `embeddings` package test files |

---

## Architecture Patterns

### Recommended File Layout

```
pkg/embeddings/
├── embedding.go              # Add IntentMapper interface here (alongside CapabilityAware)
├── multimodal.go             # Add IsNeutralIntent helper here (alongside intent constants)
├── multimodal_validate.go    # Add ValidateContentSupport + 3 new codes here
├── intent_mapper_test.go     # New file: mock IntentMapper, mapping tests, escape hatch tests
└── content_validate_test.go  # New file (or extend multimodal_validation_test.go): ValidateContentSupport tests
```

Both new test files live in `package embeddings` (same package, no `_test` suffix needed) consistent with the existing test files.

### Pattern 1: Opt-In Interface via Type Assertion
**What:** A new interface is defined at package level; callers check for it with a type assertion before calling.
**When to use:** When behavior is optional and not all providers need to implement it.

```go
// Source: pkg/embeddings/embedding.go (established pattern — CapabilityAware, Closeable, EmbeddingFunctionUnwrapper)

// IntentMapper is implemented by providers that translate neutral intents to native strings.
type IntentMapper interface {
    MapIntent(intent Intent) (string, error)
}
```

Callers check it exactly like `CapabilityAware`:
```go
// Source: pattern derived from existing CapabilityAware usage in pkg/embeddings/
if mapper, ok := ef.(IntentMapper); ok {
    native, err := mapper.MapIntent(content.Intent)
    // ...
}
```

### Pattern 2: Switch-Based Neutral Intent Check
**What:** `IsNeutralIntent` uses a switch statement over the 5 known constants.
**When to use:** Closed set of known values, avoids map allocation on the hot path.

```go
// Source: pkg/embeddings/multimodal.go (alongside the 5 intent constants)

// IsNeutralIntent reports whether the intent is one of the 5 shared neutral constants.
func IsNeutralIntent(intent Intent) bool {
    switch intent {
    case IntentRetrievalQuery,
        IntentRetrievalDocument,
        IntentClassification,
        IntentClustering,
        IntentSemanticSimilarity:
        return true
    default:
        return false
    }
}
```

### Pattern 3: ValidateContentSupport Pre-Check Helper
**What:** A standalone helper that validates all three dimensions (modality, intent, dimension) against `CapabilityMetadata` before provider I/O.
**When to use:** Called by providers in their `EmbedContent`/`EmbedContents` before dispatching to the API.

```go
// Source: pkg/embeddings/multimodal_validate.go (extends existing validation constants)

const (
    // existing codes ...
    validationCodeUnsupportedModality  = "unsupported_modality"
    validationCodeUnsupportedIntent    = "unsupported_intent"
    validationCodeUnsupportedDimension = "unsupported_dimension"
)

// ValidateContentSupport checks content against declared capabilities.
// Returns nil if caps has no Modalities declared (pass-through for unadapted providers).
func ValidateContentSupport(content Content, caps CapabilityMetadata) error {
    validationErr := &ValidationError{}

    // Modality check: validate each part's modality against declared capabilities.
    // Only validates when caps.Modalities is non-empty.
    if len(caps.Modalities) > 0 {
        for i, part := range content.Parts {
            if !caps.SupportsModality(part.Modality) {
                validationErr.addIssue(
                    fmt.Sprintf("parts[%d].modality", i),
                    validationCodeUnsupportedModality,
                    fmt.Sprintf("provider does not support %q modality", part.Modality),
                )
                break // fail on first unsupported item, consistent with batch behavior
            }
        }
    }

    // Intent check: only for neutral intents; custom strings bypass capability enforcement.
    if content.Intent != "" && IsNeutralIntent(content.Intent) && len(caps.Intents) > 0 {
        if !caps.SupportsIntent(content.Intent) {
            validationErr.addIssue(
                "intent",
                validationCodeUnsupportedIntent,
                fmt.Sprintf("provider does not support %q intent", content.Intent),
            )
        }
    }

    // Dimension check: only when caps declares dimension support is absent.
    if content.Dimension != nil && !caps.SupportsRequestOption(RequestOptionDimension) && len(caps.RequestOptions) > 0 {
        validationErr.addIssue(
            "dimension",
            validationCodeUnsupportedDimension,
            "provider does not support output dimension override",
        )
    }

    return validationErr.orNil()
}
```

**Critical detail:** When `caps.Modalities` is empty (provider doesn't implement `CapabilityAware` or adapter inferred no caps), the helper passes through without rejecting anything. This preserves the contract: only capability-aware providers get pre-flight validation.

### Pattern 4: Batch Validation via ValidateContentSupport
**What:** For batch requests, iterate and fail on the first unsupported item.
**When to use:** `EmbedContents` pre-check.

```go
// Source: pattern from validateBatchCompatibility in multimodal_compat.go

func ValidateContentsSupport(contents []Content, caps CapabilityMetadata) error {
    for i, content := range contents {
        if err := ValidateContentSupport(content, caps); err != nil {
            return prefixBatchCompatibilityError(i, err)
        }
    }
    return nil
}
```

### Pattern 5: Mock IntentMapper for Tests
**What:** A test-local struct implementing `IntentMapper` with a configurable mapping table and error injection.
**When to use:** All mapping tests in `intent_mapper_test.go`.

```go
// Source: established pattern from capabilityAwareStubEmbeddingFunction in capabilities_test.go

type stubIntentMapper struct {
    mappings map[Intent]string
    err      map[Intent]error
}

func (m *stubIntentMapper) MapIntent(intent Intent) (string, error) {
    if err, ok := m.err[intent]; ok {
        return "", err
    }
    if native, ok := m.mappings[intent]; ok {
        return native, nil
    }
    return string(intent), nil // pass-through for unmapped intents
}
```

### Anti-Patterns to Avoid
- **Embedding MapIntent in ContentEmbeddingFunction:** Forces all adapters to implement it; breaks zero-friction adoption. Standalone opt-in is correct.
- **Blocking on missing caps:** `ValidateContentSupport` must pass through when `caps.Modalities` is empty — do not reject all requests for providers with no declared caps.
- **Calling MapIntent for empty intent:** Only call `MapIntent` when `content.Intent != ""`.
- **Pre-check rejecting custom intents against CapabilityMetadata:** The shared pre-check skips intent validation when `IsNeutralIntent` returns false — custom values bypass capability enforcement entirely.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Error accumulation | Custom error struct | `ValidationError` + `ValidationIssue` | Already handles path, code, message, multi-issue collection, `orNil()`, `addIssue()` |
| Batch error prefixing | Index-aware error wrapper | `prefixBatchCompatibilityError(i, err)` | Already defined in `multimodal_compat.go`, tested |
| Single-issue error | Inline `ValidationError` literal | `compatibilityError(path, code, message)` | Already defined in `multimodal_compat.go` |
| Validation issue prefix | String manipulation | `prefixValidationIssues(prefix, err)` | Already in `multimodal_validate.go` |

**Key insight:** The error infrastructure for Phase 4 is 100% already built. Phase 4 adds three new validation code strings and one new helper function on top of infrastructure that already exists.

---

## Common Pitfalls

### Pitfall 1: Checking caps.Modalities but not handling empty slice
**What goes wrong:** When `caps.Modalities` is `nil` or empty, `SupportsModality` returns false for everything — if `ValidateContentSupport` doesn't guard this, it rejects all content for providers with no declared capabilities.
**Why it happens:** `CapabilityMetadata.SupportsModality` iterates over `m.Modalities`; empty slice means no iteration, returns false.
**How to avoid:** Always guard with `len(caps.Modalities) > 0` before modality pre-check. Same guard needed for `Intents` and `RequestOptions`.
**Warning signs:** Tests with empty `CapabilityMetadata{}` fail unexpectedly.

### Pitfall 2: Intent pre-check firing for custom strings
**What goes wrong:** A caller passes `Intent("RETRIEVAL_QUERY_V2")` (a Gemini-native string as escape hatch) and the pre-check rejects it because `caps.Intents` doesn't list it.
**Why it happens:** Pre-check does not call `IsNeutralIntent` first.
**How to avoid:** Always guard intent validation with `IsNeutralIntent(content.Intent)` — only neutral intents get capability-checked.
**Warning signs:** Any custom intent value returns `unsupported_intent` error.

### Pitfall 3: Calling MapIntent before ValidateContentSupport
**What goes wrong:** Provider calls `MapIntent` for a modality combination that isn't supported, then fails on the API side instead of with a clean pre-check error.
**Why it happens:** Order of operations matters: validate first, then map.
**How to avoid:** Document the intended call order: `ValidateContentSupport` → `MapIntent` → provider I/O. Phase 4 delivers the helpers; Phases 6-7 enforce the order in real providers.
**Warning signs:** Integration tests in Phases 6-7 get provider-side errors instead of `ValidationError` for unsupported modalities.

### Pitfall 4: MapIntent receiving empty intent
**What goes wrong:** Provider calls `mapper.MapIntent("")` and gets back an empty string or an error, silently corrupting the request.
**Why it happens:** `EmbedContent` calls `MapIntent` without checking `content.Intent != ""`.
**How to avoid:** Gate `MapIntent` calls on non-empty intent: `if content.Intent != ""`. Document this in the interface godoc.
**Warning signs:** Mock mapper tests fail with unexpected calls for content with no intent set.

### Pitfall 5: Confusing validation codes with error messages
**What goes wrong:** Code that compares error messages instead of `ValidationIssue.Code` in assertions.
**Why it happens:** The existing `ValidationError.Error()` string includes the message but not the code.
**How to avoid:** Always use `errors.As` + check `.Issues[0].Code` in tests — see `requireValidationIssue` helper in `capabilities_test.go`.
**Warning signs:** Brittle tests that break when message wording changes.

---

## Code Examples

Verified patterns from existing source:

### Adding a new validation code constant
```go
// Source: pkg/embeddings/multimodal_validate.go lines 10-17 (existing pattern)
const (
    validationCodeForbidden    = "forbidden"
    validationCodeInvalidValue = "invalid_value"
    // ... existing codes ...
    // New codes for Phase 4:
    validationCodeUnsupportedModality  = "unsupported_modality"
    validationCodeUnsupportedIntent    = "unsupported_intent"
    validationCodeUnsupportedDimension = "unsupported_dimension"
)
```

### Interface compile-time check (established pattern)
```go
// Source: pkg/embeddings/multimodal_compat.go lines 13-22 (var _ pattern)

// Ensure stubIntentMapper satisfies IntentMapper at compile time (test file)
var _ IntentMapper = (*stubIntentMapper)(nil)
```

### Using requireValidationIssue in tests
```go
// Source: pkg/embeddings/capabilities_test.go lines 305-316

requireValidationIssue(t, err, "intent", validationCodeUnsupportedIntent, "retrieval_query")
requireValidationIssue(t, err, "parts[0].modality", validationCodeUnsupportedModality, "audio")
requireValidationIssue(t, err, "dimension", validationCodeUnsupportedDimension, "dimension override")
```

### Test helper reuse — batch error prefix
```go
// Source: pkg/embeddings/multimodal_compat.go lines 319-323

// prefixBatchCompatibilityError is already exported-to-package; reuse in ValidateContentsSupport
return prefixBatchCompatibilityError(i, err)
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Intent passed through context.Context (openai, nomic, gemini) | Intent on Content request object | Phase 1 | Intent is now a first-class portable field, not a context side-channel |
| No neutral-intent-to-provider mapping contract | `IntentMapper` interface (Phase 4) | Now | Providers can declare their own native mapping table |
| No pre-flight capability validation | `ValidateContentSupport` helper (Phase 4) | Now | Unsupported combinations fail eagerly instead of silently degrading |
| Validation codes all structural (required, forbidden, etc.) | Add `unsupported_*` capability codes | Now | Callers can distinguish structural vs capability mismatch errors |

**Deprecated/outdated:**
- Using context.Context for intent in new `ContentEmbeddingFunction` implementations: the `Content.Intent` field is the portable path. Context-based intent remains valid in legacy `EmbeddingFunction` providers.

---

## Open Questions

1. **Dimension pre-check semantics when `RequestOptions` is empty**
   - What we know: `CapabilityMetadata.RequestOptions` can be empty for providers that don't declare option support.
   - What's unclear: Should an empty `RequestOptions` slice mean "no options supported" or "options unknown"? The modality pattern is "empty = unknown/pass-through".
   - Recommendation: Use the same guard as modality — `len(caps.RequestOptions) > 0` before checking dimension. Consistent with the CapabilityAware opt-in philosophy.

2. **ValidateContentsSupport as a separate exported function vs inline loop**
   - What we know: `validateBatchCompatibility` in `multimodal_compat.go` is unexported; the batch loop pattern is proven.
   - What's unclear: Should Phase 4 export `ValidateContentsSupport` or keep the batch loop inline in providers?
   - Recommendation: Export `ValidateContentsSupport(contents []Content, caps CapabilityMetadata) error` — mirrors `ValidateContents` and makes it available to Phases 6-7 providers without duplication.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | `testing` + `testify/require` (already in module) |
| Config file | none — standard `go test` |
| Quick run command | `go test ./pkg/embeddings/ -run 'TestIntentMapper\|TestValidateContentSupport\|TestIsNeutralIntent'` |
| Full suite command | `go test ./pkg/embeddings/` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| MAP-01 | `IntentMapper.MapIntent` maps neutral intents to native strings | unit | `go test ./pkg/embeddings/ -run TestIntentMapper` | ❌ Wave 0 |
| MAP-01 | `IsNeutralIntent` identifies the 5 neutral constants | unit | `go test ./pkg/embeddings/ -run TestIsNeutralIntent` | ❌ Wave 0 |
| MAP-01 | Custom/raw intents pass through without capability rejection | unit | `go test ./pkg/embeddings/ -run TestIntentMapperEscapeHatch` | ❌ Wave 0 |
| MAP-02 | `ValidateContentSupport` rejects unsupported modality | unit | `go test ./pkg/embeddings/ -run TestValidateContentSupportModality` | ❌ Wave 0 |
| MAP-02 | `ValidateContentSupport` rejects unsupported neutral intent | unit | `go test ./pkg/embeddings/ -run TestValidateContentSupportIntent` | ❌ Wave 0 |
| MAP-02 | `ValidateContentSupport` rejects unsupported dimension | unit | `go test ./pkg/embeddings/ -run TestValidateContentSupportDimension` | ❌ Wave 0 |
| MAP-02 | `ValidateContentSupport` passes through when caps empty | unit | `go test ./pkg/embeddings/ -run TestValidateContentSupportPassThrough` | ❌ Wave 0 |
| MAP-02 | Batch validation fails on first unsupported item | unit | `go test ./pkg/embeddings/ -run TestValidateContentsSupportBatch` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./pkg/embeddings/ -run 'TestIntentMapper\|TestValidateContentSupport\|TestIsNeutralIntent'`
- **Per wave merge:** `go test ./pkg/embeddings/`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `pkg/embeddings/intent_mapper_test.go` — covers MAP-01 (stub `IntentMapper`, neutral mapping, escape hatch pass-through)
- [ ] `pkg/embeddings/content_validate_test.go` — covers MAP-02 (all `ValidateContentSupport` cases, batch behavior)

*(No new framework install needed — testify already in go.mod)*

---

## Sources

### Primary (HIGH confidence)
- `pkg/embeddings/multimodal_validate.go` — existing validation code constants, `ValidationError`, `addIssue`, `orNil`, `prefixValidationIssues`
- `pkg/embeddings/multimodal_compat.go` — `compatibilityError`, `prefixBatchCompatibilityError`, `validateCompatibleContent` pattern, `validateBatchCompatibility`
- `pkg/embeddings/embedding.go` — `CapabilityAware`, `Closeable`, `EmbeddingFunctionUnwrapper` opt-in interface pattern
- `pkg/embeddings/capabilities.go` — `CapabilityMetadata`, `SupportsModality`, `SupportsIntent`, `SupportsRequestOption`
- `pkg/embeddings/multimodal.go` — `Intent`, 5 neutral constants, `Content`, `Part`
- `pkg/embeddings/capabilities_test.go` — `requireValidationIssue` helper, stub struct patterns
- `pkg/embeddings/gemini/task_type.go` — 8 Gemini task types (mapping targets for Phase 6)
- `pkg/embeddings/nomic/nomic.go` — 4 Nomic task types (mapping targets, reference)
- `.planning/phases/04-provider-mapping-and-explicit-failures/04-CONTEXT.md` — all locked decisions

### Secondary (MEDIUM confidence)
- `.planning/phases/01-shared-multimodal-contract/01-CONTEXT.md` — intent ergonomics, portable field wins, escape hatch design
- `.planning/REQUIREMENTS.md` — MAP-01, MAP-02 requirement text

### Tertiary (LOW confidence)
None — all findings verified directly against source code.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, all patterns verified in source
- Architecture: HIGH — all patterns derived directly from existing `pkg/embeddings/` code
- Pitfalls: HIGH — derived from reading the actual implementation of `validateCompatibleContent`, `SupportsModality`, and existing test patterns

**Research date:** 2026-03-20
**Valid until:** 2026-06-20 (stable internal codebase, no external dependency changes)
