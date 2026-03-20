---
status: complete
phase: 01-shared-multimodal-contract
source:
  - .planning/phases/01-shared-multimodal-contract/01-00-SUMMARY.md
  - .planning/phases/01-shared-multimodal-contract/01-01-SUMMARY.md
  - .planning/phases/01-shared-multimodal-contract/01-02-SUMMARY.md
  - .planning/phases/01-shared-multimodal-contract/01-03-SUMMARY.md
started: 2026-03-19T09:23:52Z
updated: 2026-03-19T09:30:53Z
---

## Current Test

[testing complete]

## Tests

### 1. Mixed Modality Content
expected: Creating shared `Content` values with text, image, audio, video, and PDF parts should validate successfully. Text parts carry text only, binary parts carry the matching source kind, and bytes-backed inputs remain unchanged even if the caller mutates the original byte slice after construction.
result: pass

### 2. Ordered Parts and Batches
expected: Mixed `[]Part` order and batched `[]Content` order should remain exactly as provided after validation. The first content item should keep its text-image-audio-video-pdf sequence, and later batch entries should stay intact.
result: pass

### 3. Optional Request Fields
expected: `Intent`, `Dimension`, and `ProviderHints` should be accepted on `Content`, survive validation unchanged, and remain readable by callers without reordering or stripping fields.
result: pass

### 4. Structured Validation Errors
expected: Invalid content, part, source, or empty batch inputs should return `*ValidationError` with stable issue metadata such as `Path` and `Code` for cases like missing parts, unsupported modality, source mismatches, invalid dimensions, and blank intent values.
result: pass

### 5. Legacy ImageInput Bridge
expected: URL, file, and base64 `ImageInput` values should convert into `ModalityImage` parts that preserve the original source kind and payload field only. Missing or conflicting input fields should return typed validation issues on `input`.
result: pass

### 6. Additive Embedding Interface
expected: Existing text-only and legacy multimodal embedding interfaces should remain unchanged, while `ContentEmbeddingFunction` exists as an additive interface for callers who want shared multimodal `Content` input.
result: pass

## Summary

total: 6
passed: 6
issues: 0
pending: 0
skipped: 0

## Gaps

[none yet]
