# Phase 1: Shared Multimodal Contract - Research

**Researched:** 2026-03-18
**Domain:** Go shared embedding contracts, multimodal request modeling, and pre-I/O validation
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

### Public request shape
- The core embeddable unit is a single canonical `Content` item containing an ordered list of `Parts`.
- A mixed-part `Content` produces one aggregated embedding.
- Part order must be preserved exactly as provided by the caller.
- Phase 1 should expose both a single-item API and a batch API over the same canonical item shape:
  - `EmbedContent(content)`
  - `EmbedContents(contents)`
- Batch support is a companion API over repeated `Content` items, not the primary shape of one multimodal item.

### Media source model
- Binary modalities should share one common source pattern across image, audio, video, and PDF.
- The public contract must carry explicit modality, not infer it from MIME or source details.
- URL is a first-class source alongside file path, base64/inline bytes, and similar direct data forms.
- Source kind must remain explicit in the public model so provenance is preserved.
- Canonical execution may normalize to bytes when necessary, but the shared contract must preserve source provenance first.

### Intent ergonomics
- `Intent` should be a string-backed custom type, not a closed enum.
- The SDK should provide a small shared core of neutral intent constants, starting with:
  - `retrieval_query`
  - `retrieval_document`
  - `classification`
  - `clustering`
  - `semantic_similarity`
- Intent is optional. Omission means no explicit intent was set.
- Neutral intent names should be user-facing and domain-style, not provider-style spellings.
- Callers may still pass raw/custom string values in the same `Intent` field as an escape hatch.
- Neutral constants are the portable contract; raw/custom values are allowed but may be provider-specific.
- `retrieval_query` and `retrieval_document` remain distinct neutral intents, even for mixed multimodal content.
- Unsupported intents must fail explicitly.

### Request-time overrides
- Portable request-time overrides belong on the `Content` request object itself, not hidden in `context.Context`.
- Phase 1 portable overrides are limited to:
  - `intent`
  - output `dimension`
- Provider-specific request hints should live in a separate optional provider-hints map on the request.
- If a provider-hint conflicts with a portable field, the portable field wins.

### Validation and loading semantics
- Text parts stay minimal in Phase 1 and should only carry the text payload.
- Shared contract validation should be strict before provider I/O.
- Invalid shared shape should fail early, including empty content, empty parts, and invalid source combinations.
- File and URL sources remain lazy references until embedding time.
- For this version, URLs should be passed through directly to providers when they support URL-based inputs.
- Introducing an SDK-managed security boundary that fetches remote content into bytes is a later explicit decision, not implicit in Phase 1.
- `dimension` is an optional shared request hint.
- If a provider cannot support the requested dimension, the call should hard fail with a clear unsupported-dimension error rather than warn and ignore it.

### Claude's Discretion
- Concrete type names and helper constructor names, as long as they preserve the decisions above.
- Exact field naming for provider-hints and source-kind fields.
- Internal adapter layout for turning file/base64/URL helpers into the canonical public part model.
- Whether compatibility shims rely on helper constructors, wrapper methods, or small adapter types, as long as the Phase 1 public contract stays consistent with these decisions.

### Deferred Ideas (OUT OF SCOPE)
- Broader neutral intent catalog beyond the initial shared core — future expansion once more provider mappings are understood
- Additional text-part metadata such as language, MIME, labels, or annotations — defer until a concrete provider need emerges
- SDK-managed remote fetching/security boundary for URL sources — separate future decision once provider behavior and threat boundaries are clearer
- Stronger universal guarantees around dimension support or a wider shared override surface — revisit after provider capability work
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| MMOD-01 | Caller can describe a multimodal embedding request as an ordered set of parts containing text, image, audio, video, or PDF content | Add a canonical `Content` item with ordered `[]Part`, explicit `Modality`, and a shared binary `Source` model in `pkg/embeddings`. |
| MMOD-02 | Caller can submit mixed-part multimodal requests without losing the original part ordering | Keep parts and batch items as slices only; avoid any map-based representation; validate but never reorder. |
| MMOD-03 | Caller can set a provider-neutral intent for a multimodal request using shared semantics such as retrieval query, retrieval document, classification, clustering, or semantic similarity | Use a string-backed `Intent` type with shared constants plus custom raw values, and validate unsupported intents explicitly. |
| MMOD-04 | Caller can set per-request options such as target output dimensionality and provider-specific hints without mutating provider-wide configuration | Put `Intent`, `Dimension`, and `ProviderHints` on the request object; do not use `context.Context` or mutate provider defaults. |
| MMOD-05 | Invalid request shapes are rejected before provider I/O with explicit validation errors | Introduce `Validate()` on content/part/source shapes and return a typed validation error that can surface multiple issues. |
</phase_requirements>

## Summary

Phase 1 is best planned as a pure shared-contract phase inside `pkg/embeddings`, not a provider migration and not a collection-layer rewrite. The current public contract is fragmented: `EmbeddingFunction` is text-only, `MultimodalEmbeddingFunction` is image-only, and per-request overrides for model/task/dimension are currently hidden in `context.Context` for providers like Gemini, OpenAI, and Nomic. That makes `pkg/embeddings` the correct implementation seam for a new additive `Content`/`Part`/`Intent` request model, but it also means Phase 1 should avoid dragging registry/config or V2 collection changes forward.

The main compatibility constraint is that the current V2 client and collection layers still store, auto-wire, and execute only `embeddings.EmbeddingFunction` instances. `pkg/api/v2/configuration.go` rebuilds dense embedding functions only, and collection operations embed `[]string` documents through `EmbeddingFunction`. That means the new multimodal request types must be additive foundations for later phases, not replacements for existing interfaces. Keep `ImageInput` and `MultimodalEmbeddingFunction` intact, add new shared types and validation now, and let Phase 2 own compatibility adapters and Phase 3 own registry/config integration.

The biggest planning risk is overreach. If Phase 1 tries to solve provider capability negotiation, collection auto-wiring, remote URL fetching, or provider-specific mapping semantics, it will either break compatibility or force premature abstractions. The safe plan is: add canonical request types, add strict shape validation, add helper constructors and typed errors, preserve old APIs, and write focused unit tests that lock down modality coverage, ordering, request options, and pre-I/O failure behavior.

**Primary recommendation:** Add additive `Content`, `Part`, `Source`, `Intent`, and typed validation primitives in `pkg/embeddings`, preserve legacy image/text interfaces unchanged, and keep request-time options on the request object instead of `context.Context`.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.24.11 | Language/toolchain | Pinned in `go.mod`; the repo already targets Go 1.24.x. |
| `pkg/embeddings` | repo-local | Shared public embedding contracts | All dense, sparse, and multimodal abstractions already live here. |
| `github.com/pkg/errors` | v0.9.1 | Wrapped explicit errors | The repo consistently uses wrapped errors at validation and request boundaries. |
| `github.com/go-playground/validator/v10` via `embeddings.NewValidator()` | v10.30.1 | Constructor/struct validation | Existing providers already use this pattern, including `Secret`-aware validation. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/stretchr/testify` | v1.11.1 | Assertions in unit tests | Use for new contract and validation tests in `pkg/embeddings`. |
| `pkg/api/v2` | repo-local | Compatibility guardrail | Read it to avoid pulling config/collection changes into Phase 1. |
| `google.golang.org/genai` | v1.45.0 | Existing in-repo reference for `Content`/`Part` mental model | Useful only as a naming/shape reference; do not couple the shared contract to the SDK. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Explicit request object fields for `Intent`, `Dimension`, and `ProviderHints` | `context.Context` overrides | Already inconsistent today; hidden state fails MMOD-04 and complicates testing. |
| Additive new multimodal interfaces and types | Replacing `ImageInput` and `MultimodalEmbeddingFunction` immediately | Too risky; breaks the compatibility work explicitly deferred to Phase 2. |
| Typed validation error implementing `error` | Plain string errors only | Simpler, but weaker support for the “explicit validation results” requirement. |

**Installation:**
```bash
# No new dependency is recommended for Phase 1.
# Use the existing module stack in go.mod.
```

**Version verification:** Versions above were verified from `go.mod` on 2026-03-18. Phase 1 should not introduce new modules unless planning uncovers a hard blocker, which current evidence does not support.

## Architecture Patterns

### Recommended Project Structure
```text
pkg/embeddings/
├── embedding.go              # Existing legacy interfaces and ImageInput stay in place
├── multimodal.go             # New Content/Part/Source/Intent/request-option types
├── multimodal_validate.go    # Validation helpers and typed validation errors
├── multimodal_compat.go      # Optional ImageInput -> Part helpers only if needed
└── multimodal_test.go        # Unit tests for shape, ordering, and errors
```

### Pattern 1: Canonical Content Item With Ordered Parts
**What:** Model one embeddable semantic unit as a single `Content` item containing ordered `[]Part`; batch embedding is `[]Content` outside that shape.

**When to use:** For every new multimodal caller-facing API introduced by this phase.

**Recommended sketch (inference from current code and locked decisions, HIGH confidence on shape / MEDIUM on exact names):**
```go
type Content struct {
	Parts         []Part
	Intent        Intent
	Dimension     *int
	ProviderHints map[string]any
}

type Part struct {
	Modality Modality
	Text     string
	Source   *BinarySource
}
```

**Why this fits the repo:** It mirrors the existing split between “single item” and “batch of items,” keeps order via slices, and matches the phase decision that one mixed-part content item yields one aggregated embedding.

### Pattern 2: Explicit Binary Source With Preserved Provenance
**What:** Use one binary source abstraction across image, audio, video, and PDF, with explicit `SourceKind` and one source value active at a time.

**When to use:** For any non-text part.

**Recommended sketch (inference from `ImageInput` plus phase decisions, HIGH confidence on behavior / MEDIUM on exact fields):**
```go
type SourceKind string

const (
	SourceKindURL    SourceKind = "url"
	SourceKindFile   SourceKind = "file"
	SourceKindBase64 SourceKind = "base64"
	SourceKindBytes  SourceKind = "bytes"
)

type BinarySource struct {
	Kind     SourceKind
	URL      string
	FilePath string
	Base64   string
	Bytes    []byte
	MIMEType string // optional, see Open Questions
}
```

**Why this fits the repo:** `ImageInput` already encodes a one-of source model, but it infers source kind from populated fields and only handles images. Phase 1 needs the same idea generalized without losing provenance.

### Pattern 3: Validate Shapes Early, Return Typed Errors
**What:** Keep Go-style `Validate() error`, but make the returned error a typed validation error containing structured issues.

**When to use:** On `Content`, `Part`, and `BinarySource` prior to any provider adapter or I/O.

**Recommended sketch (inference from current validation style, MEDIUM confidence on exact type names):**
```go
type ValidationIssue struct {
	Path    string
	Code    string
	Message string
}

type ValidationError struct {
	Issues []ValidationIssue
}

func (e *ValidationError) Error() string { /* summarize issues */ return "" }
```

**Why this fits the repo:** The repo already validates early and returns explicit errors. A typed validation error is additive and gives planners a clean seam for MMOD-05 without breaking Go ergonomics.

### Pattern 4: Keep Legacy Interfaces Untouched In This Phase
**What:** Add new request and interface types beside existing ones. Do not rewrite or remove `ImageInput`, `EmbeddingFunction`, or `MultimodalEmbeddingFunction` in Phase 1.

**When to use:** Throughout planning and task breakdown for this phase.

**Evidence:** `ImageInput` and `MultimodalEmbeddingFunction` are public today, Roboflow implements them, and V2 collection/config flows still depend on `EmbeddingFunction`.

### Anti-Patterns to Avoid
- **Putting portable request options in `context.Context`:** Current providers already do this for task/dimension/model overrides, but that is exactly the behavior Phase 1 should replace at the shared contract level.
- **Replacing legacy interfaces now:** `ImageInput` and image-only multimodal behavior are public compatibility surface, not cleanup.
- **Requiring file reads or network fetches in `Validate()`:** Locked decisions say file/URL sources remain lazy until embedding time.
- **Using `map[string]any` or interfaces to represent parts:** That would lose compile-time shape guarantees and make validation much harder.
- **Making batch the primary shape:** The canonical unit is one `Content`; batching is just `[]Content`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Request-time override transport | Hidden `context.Context` keys for portable fields | Explicit request fields on `Content` | The phase requires portable, inspectable request options. |
| Generic multimodal payload unions | Reflection-heavy `map[string]any` payload trees | Explicit Go structs with one-of validation | Keeps the API readable, stable, and testable. |
| Remote URL ingestion/security boundary | SDK-managed fetch-and-convert pipeline | Lazy URL/file refs; let providers consume URL directly when supported | The phase explicitly defers remote fetching and associated security policy. |
| Registry/config persistence changes | New config schema for multimodal request objects | Leave config/registry to later phases | Current V2 reconstruction only knows dense `EmbeddingFunction`. |
| Universal MIME sniffing/file parsing | Cross-format probing and content decoding in the shared layer | Minimal shape validation only | Audio/video/PDF handling is provider-specific and easy to overfit. |

**Key insight:** Phase 1 should validate structure, not capability and not content bytes. Capability checks belong in later phases once providers declare what they actually support.

## Common Pitfalls

### Pitfall 1: Breaking the Existing Image-Only Contract
**What goes wrong:** A planner treats Phase 1 as permission to replace `ImageInput` or rename `MultimodalEmbeddingFunction`.
**Why it happens:** The current multimodal surface is limited, so it is tempting to “clean it up” while adding richer types.
**How to avoid:** Keep legacy types unchanged and add adapters/helpers only.
**Warning signs:** Tasks mention deleting `ImageInput`, renaming `EmbedImage(s)`, or changing Roboflow signatures.

### Pitfall 2: Pulling Config/Collection Work Forward
**What goes wrong:** The phase plan starts editing config auto-wiring, collection add/query semantics, or stored schema formats.
**Why it happens:** The shared contract sits near registry/config code, and the roadmap mentions future persistence work.
**How to avoid:** Treat `pkg/api/v2/configuration.go` and collection ops as constraints only. Phase 1 adds shared types and validation, not reconstruction or collection usage.
**Warning signs:** Tasks mention `BuildEmbeddingFunctionFromConfig`, `WithEmbeddingFunctionCreate`, `Collection.Add`, or schema serialization changes.

### Pitfall 3: Reintroducing Hidden Request State
**What goes wrong:** New APIs still rely on context keys for intent or dimension instead of storing them on the request itself.
**Why it happens:** Current Gemini/OpenAI/Nomic implementations already support context-based overrides.
**How to avoid:** Make the request object the single source of truth for shared portable options; any provider adapter can translate from request fields to existing context keys later.
**Warning signs:** New helper functions are named `ContextWithIntent` or `ContextWithDimension` in `pkg/embeddings`.

### Pitfall 4: Overvalidating Lazy Sources
**What goes wrong:** Validation tries to fetch URLs, open files, or fully decode base64 payloads.
**Why it happens:** `ImageInput.ToBase64()` currently reads local files and validates image extensions, which is easy to copy too literally.
**How to avoid:** Validate only structural rules in the shared contract: required fields, one-of source selection, non-empty references, and value type/range checks.
**Warning signs:** `Validate()` calls `os.Open`, `http.Get`, `io.ReadAll`, or provider-specific parsers.

### Pitfall 5: Losing Part Ordering In Adapters
**What goes wrong:** Part slices get flattened, regrouped by modality, or converted through unordered intermediate structures.
**Why it happens:** Providers often accept modality-specific top-level fields or separate arrays.
**How to avoid:** Preserve the caller’s `[]Part` order in the shared type and keep any later provider mapping order-aware.
**Warning signs:** Intermediate representations keyed by modality or tasks that “split parts into text/images” before validation.

### Pitfall 6: Treating All Custom Intents As Portable
**What goes wrong:** Any raw string in `Intent` is treated as implicitly supported.
**Why it happens:** The phase allows raw/custom strings as an escape hatch.
**How to avoid:** Keep neutral constants as the portable set and require explicit unsupported-intent failures for provider mappings later.
**Warning signs:** Validation or interface comments imply “any string intent is fine everywhere.”

## Code Examples

Verified patterns from current code and official docs:

### One-Of Source Validation
```go
// Source: pkg/embeddings/embedding.go
func (i ImageInput) Validate() error {
	count := 0
	if i.Base64 != "" {
		count++
	}
	if i.URL != "" {
		count++
	}
	if i.FilePath != "" {
		count++
	}
	if count == 0 {
		return errors.New("image input must have exactly one of Base64, URL, or FilePath set")
	}
	if count > 1 {
		return errors.New("image input must have exactly one of Base64, URL, or FilePath set, got multiple")
	}
	return nil
}
```

### Early Numeric Override Validation
```go
// Source: pkg/embeddings/gemini/gemini.go
func intToInt32Ptr(v int) (*int32, error) {
	if v <= 0 {
		return nil, errors.New("dimension must be greater than 0")
	}
	if v > math.MaxInt32 {
		return nil, errors.Errorf("dimension must be <= %d", math.MaxInt32)
	}
	conv := int32(v)
	return &conv, nil
}
```

### Dense-Only Config Reconstruction Boundary
```go
// Source: pkg/api/v2/configuration.go
func BuildEmbeddingFunctionFromConfig(cfg *CollectionConfigurationImpl) (embeddings.EmbeddingFunction, error) {
	if cfg == nil {
		return nil, nil
	}
	efInfo, ok := cfg.GetEmbeddingFunctionInfo()
	if ok && efInfo != nil && efInfo.IsKnown() && embeddings.HasDense(efInfo.Name) {
		return embeddings.BuildDense(efInfo.Name, efInfo.Config)
	}
	// schema fallback omitted
	return nil, nil
}
```

### Ordered Part Mental Model For Mixed Media
```go
// Source: inferred from Google Gemini official docs:
// https://ai.google.dev/api/generate-content
parts := []Part{
	NewTextPart("Give me a summary of this document:"),
	NewPDFPartFromURI(fileURI /* plus MIME when available */),
}
content := NewContent(parts...)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Text-only `EmbeddingFunction` plus image-only `MultimodalEmbeddingFunction` | Additive canonical `Content` with ordered `Parts` | Planned in Phase 1 | Shared contract can finally represent mixed multimodal inputs. |
| Provider-specific task names and defaults | Neutral string-backed `Intent` constants plus raw escape hatch | Planned in Phase 1 | Portable semantics become possible without forcing one provider’s vocabulary. |
| Request overrides hidden in `context.Context` | Request-scoped fields on the content object | Planned in Phase 1 | Request behavior becomes explicit, serializable, and testable. |
| Source kind inferred from whichever field is populated | Explicit `SourceKind` retained in the public model | Planned in Phase 1 | Provenance is preserved for later provider adapters and security decisions. |

**Deprecated/outdated:**
- Image-only `ImageInput` as the shared multimodal foundation: too narrow for audio/video/PDF and mixed ordered parts, but still must remain for compatibility.
- `context.Context` as the portable request-option API: workable for provider internals, not suitable for the new shared contract.

## Open Questions

1. **Should binary sources carry optional MIME type in Phase 1?**
   - What we know: The phase requires explicit modality and source provenance. Google’s current file/URI multimodal docs pair URI with MIME type for image/video/PDF requests.
   - What’s unclear: Whether Phase 1 can omit MIME without immediately forcing provider-specific hints for common file/URI cases.
   - Recommendation: Add an optional `MIMEType` field now if it can be purely additive; if not, reserve the field name in comments and keep provider hints as the fallback.

2. **How should “explicit validation results” be surfaced idiomatically in Go?**
   - What we know: Existing repo code prefers `Validate() error`, but MMOD-05 asks for explicit validation results rather than opaque failures.
   - What’s unclear: Whether callers need first-error behavior only or structured issue lists.
   - Recommendation: Return a typed `*ValidationError` that implements `error` and carries `[]ValidationIssue`.

3. **What should the new additive interface be called?**
   - What we know: `MultimodalEmbeddingFunction` is already taken by the current image-only interface. Phase 1 must add new APIs without breaking that name.
   - What’s unclear: Whether the clearest additive name is `ContentEmbeddingFunction`, `RichMultimodalEmbeddingFunction`, or similar.
   - Recommendation: Choose a new name in `pkg/embeddings` and keep it obviously additive. Do not overload the existing `MultimodalEmbeddingFunction` name in this phase.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` + `github.com/stretchr/testify` v1.11.1 |
| Config file | none |
| Quick run command | `go test ./pkg/embeddings -run 'TestMultimodalContentSupportsAllModalities|TestMultimodalContentPreservesOrder|TestMultimodalRequestOptions|TestMultimodalIntentValidation|TestMultimodalValidationErrors|TestNewImagePartFromImageInput' && go test -tags=basicv2 ./pkg/api/v2 -run '^TestBuildEmbeddingFunctionFromConfig$'` |
| Full suite command | `make test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| MMOD-01 | Construct valid text/image/audio/video/PDF parts and content items | unit | `go test ./pkg/embeddings -run '^TestMultimodalContentSupportsAllModalities$'` | ❌ Wave 0 |
| MMOD-02 | Preserve `Parts` ordering and `[]Content` batch ordering exactly | unit | `go test ./pkg/embeddings -run '^TestMultimodalContentPreservesOrder$'` | ❌ Wave 0 |
| MMOD-03 | Validate neutral intent constants and raw/custom intent behavior | unit | `go test ./pkg/embeddings -run '^TestMultimodalIntentValidation$'` | ❌ Wave 0 |
| MMOD-04 | Keep `Intent`, `Dimension`, and `ProviderHints` request-scoped without provider mutation | unit | `go test ./pkg/embeddings -run '^TestMultimodalRequestOptions$'` | ❌ Wave 0 |
| MMOD-05 | Reject empty content, empty parts, invalid source combinations, and conflicting shapes before I/O | unit | `go test ./pkg/embeddings -run '^TestMultimodalValidationErrors$'` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./pkg/embeddings -run 'TestMultimodalContentSupportsAllModalities|TestMultimodalContentPreservesOrder|TestMultimodalRequestOptions|TestMultimodalIntentValidation|TestMultimodalValidationErrors|TestNewImagePartFromImageInput' && go test -tags=basicv2 ./pkg/api/v2 -run '^TestBuildEmbeddingFunctionFromConfig$'`
- **Per wave merge:** `go test ./pkg/embeddings -run 'TestMultimodalContentSupportsAllModalities|TestMultimodalContentPreservesOrder|TestMultimodalRequestOptions|TestMultimodalIntentValidation|TestMultimodalValidationErrors|TestNewImagePartFromImageInput' && go test -tags=basicv2 ./pkg/api/v2 -run '^TestBuildEmbeddingFunctionFromConfig$' && make test`
- **Phase gate:** `make test`

### Wave 0 Gaps
- [ ] `pkg/embeddings/multimodal_test.go` — covers MMOD-01 and MMOD-02
- [ ] `pkg/embeddings/multimodal_validation_test.go` — covers MMOD-03, MMOD-04, and MMOD-05
- [ ] Typed validation error assertions — new tests should assert issue paths/codes, not just error strings

## Sources

### Primary (HIGH confidence)
- `pkg/embeddings/embedding.go` — current `EmbeddingFunction`, `ImageInput`, and `MultimodalEmbeddingFunction` definitions; validation style and current multimodal limitation
- `pkg/embeddings/registry.go` — dense/sparse/multimodal registry split
- `pkg/embeddings/roboflow/roboflow.go` — current public image-only multimodal implementation and config registration
- `pkg/embeddings/gemini/gemini.go` — current request-time task/dimension override pattern and early validation
- `pkg/embeddings/openai/openai.go` — current request-time dimension override pattern
- `pkg/embeddings/nomic/nomic.go` — current request-time task/dimensionality override pattern
- `pkg/api/v2/configuration.go` — dense-only config reconstruction boundary
- `pkg/api/v2/client.go` — collection create/get only accepts `embeddings.EmbeddingFunction`
- `pkg/api/v2/collection.go` — collection embedding path still embeds text documents only
- `docs/docs/embeddings.md` — current public Roboflow text/image examples
- `go.mod` — verified module/toolchain versions
- `Makefile` — standard test commands

### Secondary (MEDIUM confidence)
- `https://ai.google.dev/gemini-api/docs/embeddings` — current Gemini embeddings docs; confirms task/dimension concepts and `content.parts` request shape
- `https://ai.google.dev/api/generate-content` — current Gemini multimodal content docs; confirms ordered content-part model across image/video/PDF. Used as a design reference, not as evidence that Gemini embeddings already support all modalities.

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all recommendations are existing repo tools and dependencies
- Architecture: HIGH - driven directly by current code boundaries in `pkg/embeddings`, `pkg/api/v2`, and docs
- Pitfalls: HIGH - each risk is visible in the current public API or current provider patterns

**Research date:** 2026-03-18
**Valid until:** 2026-04-17
