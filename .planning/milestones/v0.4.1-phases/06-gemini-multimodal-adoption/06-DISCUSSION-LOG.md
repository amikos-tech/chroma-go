# Phase 6: Gemini Multimodal Adoption - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-20
**Phase:** 06-gemini-multimodal-adoption
**Areas discussed:** Content part handling, Intent mapping, Backward compat strategy, Registry & config

---

## Content Part Handling

### Non-text part encoding

| Option | Description | Selected |
|--------|-------------|----------|
| Inline blobs | Convert BinarySource to genai.Blob with MIME type and inline data via NewPartFromBytes | :heavy_check_mark: |
| URI references | Use genai.FileData with URI for URL-backed sources, inline blobs for others | |

**User's choice:** Inline blobs
**Notes:** User requested verification that inline blobs work with Gemini embedding API. Research confirmed: genai SDK's NewPartFromBytes handles encoding, embedding endpoint doesn't support URI references.

### MIME type resolution

| Option | Description | Selected |
|--------|-------------|----------|
| BinarySource.MIMEType field | Use MIMEType field, infer from extension for files, fail if empty and can't infer | :heavy_check_mark: |
| Modality-based defaults | Derive MIME from modality + source kind with default MIME per modality | |

**User's choice:** BinarySource.MIMEType field
**Notes:** User flagged potential security concerns with MIME resolution. Decision to add MIME-modality consistency validation as a pre-flight security check.

### Mixed-part support

**User's choice:** Yes, native mixed-part support
**Notes:** User requested grounded research before deciding. Gemini docs confirm: gemini-embedding-2-preview natively supports multiple parts in one Content, producing one aggregated embedding. SupportsMixedPart: true.

### Model selection

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-select by content | Use embedding-001 for text, auto-upgrade to embedding-2 for multimodal | |
| User chooses model explicitly | Fail if legacy model gets multimodal content | |
| Default to embedding-2-preview | Change default model to gemini-embedding-2-preview for all new instances | :heavy_check_mark: |

**User's choice:** Default to embedding-2-preview
**Notes:** User wants explicit failure propagation when legacy model receives multimodal content, plus a negative test case demonstrating the failure mode.

### Byte data resolution

| Option | Description | Selected |
|--------|-------------|----------|
| Provider-side resolution | Gemini resolves all BinarySource kinds (file, base64, bytes, URL) at call time | :heavy_check_mark: |
| Require pre-resolved bytes | Only accept SourceKindBytes, callers must pre-resolve | |

**User's choice:** Provider-side resolution
**Notes:** User asked for clarification on trade-offs and whether API payload is always base64. Clarified that genai SDK handles base64 encoding internally — provider just provides raw bytes. Provider-side resolution maintains consistent BinarySource contract across providers.

---

## Intent Mapping

### Neutral intent mapping

| Option | Description | Selected |
|--------|-------------|----------|
| Direct 1:1 mapping | Map 5 neutral intents to obvious Gemini equivalents, Gemini-only types via ProviderHints | :heavy_check_mark: |
| Extended neutral intents | Add new neutral intents for Gemini-only task types | |
| Custom intent passthrough | Let callers pass raw Gemini task type strings as Intent values | |

**User's choice:** Direct 1:1 mapping
**Notes:** None

### Gemini-only task type access

| Option | Description | Selected |
|--------|-------------|----------|
| task_type hint key | Use ProviderHints["task_type"] to access CODE_RETRIEVAL_QUERY, QUESTION_ANSWERING, FACT_VERIFICATION | :heavy_check_mark: |
| Custom intent string | Pass raw task type string as Intent, bypasses capability enforcement | |

**User's choice:** task_type hint key
**Notes:** None

### Empty intent behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Pass empty to API | Return empty string, API uses default behavior | :heavy_check_mark: |
| Default to RETRIEVAL_DOCUMENT | Default to RETRIEVAL_DOCUMENT when no intent set | |

**User's choice:** Pass empty to API
**Notes:** User noted "assuming that Google genai SDK accepts it." Verified: existing buildEmbedContentConfig already returns nil config when taskType is empty, which the SDK accepts.

---

## Backward Compat Strategy

### EmbedDocuments/EmbedQuery relation to EmbedContent/EmbedContents

| Option | Description | Selected |
|--------|-------------|----------|
| Shared core, text wrappers | One unified core, legacy methods become thin wrappers | |
| Shared helpers, separate entry | Separate entry points, shared config/response helpers | :heavy_check_mark: |
| Fully separate paths | Completely independent code paths | |

**User's choice:** Shared helpers, separate entry
**Notes:** User requested detailed trade-off analysis before deciding. Discussion covered: risk to legacy callers, code duplication, maintenance burden. Shared helpers approach gives zero-risk legacy path + no duplicated config logic + clean separation.

---

## Registry & Config

### Registration strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Dual registration | Keep RegisterDense + add RegisterContent, same name "google_genai" | :heavy_check_mark: |
| Content-only registration | Remove RegisterDense, only RegisterContent | |

**User's choice:** Dual registration
**Notes:** None

### Config fields

**User's choice:** Same fields, no additions
**Notes:** User directed research into Chroma's upstream `google_genai.json` schema. Schema has `additionalProperties: false` — only `model_name`, `task_type`, `dimension`, `api_key_env_var`, `vertexai`, `project`, `location` are valid. Config schema enforced by cross-language compatibility.

---

## Claude's Discretion

- Internal helper names (resolveBytes, resolveMIME, convertToGenaiContent)
- ValidateContentSupport placement within EmbedContent/EmbedContents
- Test scaffolding structure
- Error message wording
- Vertex AI support timing

## Deferred Ideas

- Vertex AI backend support (config fields exist in schema but not yet implemented)
- Gemini Files API for large media uploads
