# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [v0.4.2] - Unreleased

### Added

- **Search API** - Exported `ErrNilFilter`, `ErrNilRank` and `ErrNilGroupBy` sentinels for the nil-option validation errors, so callers can discriminate them with `errors.Is` instead of matching on message text. The sentinels survive the wrapping performed by `Collection.Search`. Error message text is unchanged.

### Fixed

- **Search API** - A typed-nil `WhereClause` (e.g. `var w *WhereClauseString`) no longer panics when passed to `WithFilter`/`WithSearchWhere` or nested inside `And`/`Or`. At the option boundary a typed nil is normalized to a true nil and treated as "no filter"; nested in `And`/`Or` it now produces the `nil clause in $and expression` validation error rather than dereferencing a nil pointer.
- **Search API** - `WithFilter(nil)`/`WithSearchWhere(nil)` no longer allocate an empty `SearchFilter`, so the marshalled request omits the `filter` key entirely instead of sending `{"filter":{}}`. IDs added via `WithIDs` are still preserved regardless of option ordering.

### Changed

- **Search API** - `WithSearchFilter(nil)` and `WithRank(nil)` now return validation errors, matching `WithGroupBy(nil)`'s existing behavior (including typed-nil pointers, e.g. `var kr *KnnRank`, which are also rejected). Callers that want to omit a filter/rank/group-by should omit the option entirely rather than passing nil. This is an intentional divergence from the Python and TypeScript SDKs, which treat nil/None/undefined as a no-op for these options — chroma-go treats an explicit nil as a likely caller bug rather than an omission signal. `WithFilter`/`WithSearchWhere` intentionally keep the SDK-parity no-op behavior (nil means "unfiltered"), since they are the primary, ergonomic filter entry point rather than a low-level struct option.

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
