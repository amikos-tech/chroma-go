# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [v0.4.2] - Unreleased

### Added

- **Search API** - Exported `ErrNilFilter`, `ErrNilRank` and `ErrNilGroupBy` sentinels for the nil-option validation errors, so callers can discriminate them with `errors.Is` instead of matching on message text. The sentinels survive the wrapping performed by `Collection.Search`. Error message text is unchanged.

### Fixed

- **Get/Query/Delete** - A typed-nil clause nested inside `AndDocument`/`OrDocument` now produces a `nil clause in $and expression` validation error instead of panicking, matching the `And`/`Or` behavior for metadata clauses.
- **Search API** - Marshalling an `And`/`Or` clause directly now validates first, matching `AndDocument`/`OrDocument`. Previously a typed-nil operand bypassed the `Validate()` guard and panicked inside the nested clause's own `MarshalJSON`.
- **Search API** - A typed-nil `WhereClause` (e.g. `var w *WhereClauseString`) no longer panics when passed to `WithFilter`/`WithSearchWhere` or nested inside `And`/`Or`. At the option boundary a typed nil is normalized to a true nil and treated as "no filter"; nested in `And`/`Or` it now produces the `nil clause in $and expression` validation error rather than dereferencing a nil pointer.
- **Search API** - `WithFilter(nil)`/`WithSearchWhere(nil)` no longer allocate an empty `SearchFilter`, so the marshalled request omits the `filter` key entirely instead of sending `{"filter":{}}`. IDs added via `WithIDs` are still preserved regardless of option ordering.
- **Search API** - Nil detection now covers every nillable kind, not just pointers. A caller-supplied `Rank` or `WhereClause` implementation backed by a nil map, slice, func or channel previously slipped past the guard — `WithRank` accepted it and serialized `"rank":null` instead of returning `ErrNilRank`.
- **Search API** - Typed nils no longer panic on the paths that bypass the option constructors. `SearchFilter.Where` and `SearchRequest.Rank` are exported fields, so a struct-literal request skipped `WithFilter`/`WithRank` validation entirely: `SearchFilter.MarshalJSON` panicked inside `Validate()`, `SearchRequest.MarshalJSON` panicked calling `MarshalJSON` on a typed-nil `Rank`, and `cloneRank` panicked dereferencing a typed nil matched by its type switch. `SetSearchWhere` now normalizes a typed nil to a true nil, and marshalling, rank cloning and text-query embedding all guard against one.
- **Get/Query/Delete** - `WithWhere` and `WithWhereDocument` no longer panic on a typed-nil filter. The `ApplyToGet`/`ApplyToQuery`/`ApplyToDelete` paths guarded with a plain `!= nil` before calling `Validate()`; a typed nil is now treated as "no filter" and normalized away.
- **Get/Query/Delete** - `PrepareAndValidate` no longer panics when `FilterOp.Where` or `FilterOp.WhereDocument` holds a typed nil, which is reachable by assembling an op struct directly instead of through the option constructors. `SetWhere`/`SetWhereDocument` normalize a typed nil on assignment. `Delete` additionally no longer miscounts a typed-nil filter as a supplied one — it now correctly reports "at least one filter is required" rather than panicking.

### Changed

- **Minimum Go version** - The module now requires Go 1.25 (`go 1.25.0` in `go.mod`), up from Go 1.24. The bump is driven by the testcontainers-go v0.43.0 upgrade, which removed the deprecated `github.com/docker/docker` dependency from the module graph in favor of `github.com/moby/moby`. Go 1.24 is outside the Go project's support window; builders with `GOTOOLCHAIN=auto` (the default) are unaffected, while `GOTOOLCHAIN=local` builds need a Go 1.25+ toolchain installed.
- **Embedding model defaults** - Cohere now defaults to `embed-english-v3.0` after the v2 model retirement. This changes the embedding dimension from 4096 to 1024 for callers relying on the default rather than pinning a model explicitly — existing collections built with the old default need to be re-embedded, as the two are not vector-compatible. Morph defaults to `morph-embedding-v3` for compatible custom endpoints; its hosted live test is disabled because Morph retired the hosted embedding API.
- **Search API** - `WithSearchFilter(nil)` and `WithRank(nil)` now return validation errors, matching `WithGroupBy(nil)`'s existing behavior (including typed-nil pointers, e.g. `var kr *KnnRank`, which are also rejected). Callers that want to omit a filter/rank/group-by should omit the option entirely rather than passing nil. This is an intentional divergence from the Python and TypeScript SDKs, which treat nil/None/undefined as a no-op for these options — chroma-go treats an explicit nil as a likely caller bug rather than an omission signal. `WithFilter`/`WithSearchWhere` intentionally keep the SDK-parity no-op behavior (nil means "unfiltered"), since they are the primary, ergonomic filter entry point rather than a low-level struct option.
- **Rank API** - Arithmetic methods now reject explicit nil operands during marshaling with the matchable `ErrNilRank` error instead of silently substituting `Val(0)`. Untyped nil and typed-nil Rank operands use the same error path; callers should omit the arithmetic operation rather than pass nil.
- **Rank API** - `NewKnnRank` and `WithKnnRank` now reject a nil query, an empty key, a non-positive limit, and an empty dense-vector query before assigning the rank to a search request. `KnnRank.MarshalJSON` enforces the same requirements, so manually constructed `KnnRank` values that previously serialized with a nil query, empty key, or zero limit now return an error; use `NewKnnRank` to apply the valid defaults.

## [v0.4.1] - 2026-03-23

### Added

- **Content API** - Portable multimodal embedding interface with `Content`, `Part`, and `BinarySource` types for embedding text, images, audio, video, and PDF content through a unified API (`EmbedContent`/`EmbedContents`).
- **Portable Intents** - Five provider-neutral intent constants (`IntentRetrievalQuery`, `IntentRetrievalDocument`, `IntentClassification`, `IntentClustering`, `IntentSemanticSimilarity`) that map to provider-specific task types.
- **Per-request Options** - `Dimension` and `ProviderHints` fields on `Content` for per-request configuration without mutating provider-wide settings.
- **Capability Metadata** - `CapabilityAware` interface for providers to declare supported modalities, intents, and request options. Callers can inspect capabilities without provider-specific type assertions.
- **Compatibility Adapters** - Automatic bridging between the Content API and legacy `EmbedDocuments`/`EmbedQuery` interfaces through the registry's `BuildContent` fallback chain.
- **Intent Mapping** - `IntentMapper` interface for providers to translate neutral intents to provider-native semantics with explicit errors for unsupported combinations.
- **Gemini Multimodal** - Gemini embedding function implements `ContentEmbeddingFunction`, `CapabilityAware`, and `IntentMapper` for text, image, audio, video, and PDF modalities. Default model updated to `gemini-embedding-2-preview`.
- **VoyageAI Multimodal** - VoyageAI embedding function implements `ContentEmbeddingFunction`, `CapabilityAware`, and `IntentMapper` for text, image, and video modalities via the `voyage-multimodal-3.5` model.
- **Registry Integration** - Content embedding functions can be built from stored configuration via `BuildContent`/`BuildContentCloseable` with full config round-trip support.
