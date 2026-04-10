# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [v0.4.2] - Unreleased

### Changed

- **Search API** - `WithGroupBy(nil)` now returns a validation error instead of silently omitting grouping. Callers that want no grouping should omit `WithGroupBy(...)` entirely.

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
