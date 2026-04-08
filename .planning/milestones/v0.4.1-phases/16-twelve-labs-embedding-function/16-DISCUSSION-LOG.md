# Phase 16: Twelve Labs Embedding Function - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-01
**Phase:** 16-twelve-labs-embedding-function
**Areas discussed:** API endpoint strategy, Modality & content handling, Authentication & config, Test approach

---

## API Endpoint Strategy

### Unified vs split request path

| Option | Description | Selected |
|--------|-------------|----------|
| Single unified path | One EmbedContent method sends all modalities to /v1/embed. Follows Gemini pattern. | ✓ |
| Separate text vs multimodal | Text-only calls use lighter format, multimodal uses full structure. Follows Voyage pattern. | |

**User's choice:** Single unified path
**Notes:** Matches the Twelve Labs API design — single endpoint accepts all modalities.

### Default model

| Option | Description | Selected |
|--------|-------------|----------|
| Configurable, default marengo3.0 | WithModel option lets users override. Follows Gemini/Voyage pattern. | ✓ |
| Hardcoded marengo3.0 | Simpler but requires code changes for new models. | |

**User's choice:** Configurable with marengo3.0 default
**Notes:** User requested research first — confirmed Marengo 2.7 was sunset March 30, 2026. marengo3.0 is the only viable model.

### Sync vs async endpoints

| Option | Description | Selected |
|--------|-------------|----------|
| Sync only | Covers text + image + short audio/video. No polling needed. | ✓ |
| Both sync and async | Full coverage including long video/audio. Adds task creation + polling. | |

**User's choice:** Sync only for this phase
**Notes:** User explicitly requested: "Let's implement sync in this phase but then let's add a GH issue for the async support."

---

## Modality & Content Handling

### Advertised modalities

| Option | Description | Selected |
|--------|-------------|----------|
| Text, image, audio | Matches original roadmap scope. | |
| Text, image, audio, video | Sync endpoint supports video < 10 min. All four modalities. | ✓ |
| Text and image only | Minimal start. | |

**User's choice:** Text, image, audio, video

### Mixed-part support

| Option | Description | Selected |
|--------|-------------|----------|
| Single-modality only | Each Content maps to one API call. SupportsMixedPart: false. | ✓ |
| Mixed-part with fused embeddings | Combine modalities via fused_embedding option. | |

**User's choice:** Single-modality only

### Input mapping

| Option | Description | Selected |
|--------|-------------|----------|
| URL + base64 | Map SourceKindURL → url, others → base_64_string. No asset_id. | ✓ |
| URL + base64 + asset_id | Same plus provider-specific option for pre-uploaded assets. | |

**User's choice:** URL + base64

---

## Authentication & Config

### Environment variable

| Option | Description | Selected |
|--------|-------------|----------|
| TWELVE_LABS_API_KEY | Matches brand name spacing. | ✓ |
| TWELVELABS_API_KEY | Matches Python SDK package name. | |

**User's choice:** TWELVE_LABS_API_KEY

### Audio embedding option

| Option | Description | Selected |
|--------|-------------|----------|
| Provider option with default | WithAudioEmbeddingOption functional option. Default: "audio". | ✓ |
| You decide | Claude picks. | |

**User's choice:** Provider option with "audio" default

---

## Test Approach

### Test structure

| Option | Description | Selected |
|--------|-------------|----------|
| httptest mocks + build tag | Unit tests with ef build tag. No real API calls. | |
| httptest mocks + integration test | Mocks plus separate integration test hitting real API. | ✓ |

**User's choice:** httptest mocks + integration test

### Interface coverage

| Option | Description | Selected |
|--------|-------------|----------|
| Test both interfaces | ContentEmbeddingFunction and EmbeddingFunction via dual registration. | ✓ |
| Content API only | Only test new ContentEmbeddingFunction. | |

**User's choice:** Test both interfaces

---

## Claude's Discretion

- Request body construction details
- Config round-trip key naming
- Provider-side byte resolution implementation
- Error mapping from Twelve Labs API errors

## Deferred Ideas

- Async embedding support (POST /v1.3/embed-v2/tasks) for audio/video up to 4 hours — GitHub issue to be created
