---
phase: 2
slug: capability-metadata-and-compatibility
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-19
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + github.com/stretchr/testify |
| **Config file** | none |
| **Quick run command** | `go test ./pkg/embeddings -run 'TestCapabilityMetadata|TestLegacyTextCompatibility|TestLegacyImageCompatibility|TestCompatibilityAdapterRejectsUnsupportedContent|TestMultimodalInterface' && go test -tags=basicv2 ./pkg/api/v2 -run '^TestBuildEmbeddingFunctionFromConfig$|^TestCollectionConfiguration_(Get|Set)EmbeddingFunction'` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~25 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./pkg/embeddings -run 'TestCapabilityMetadata|TestLegacyTextCompatibility|TestLegacyImageCompatibility|TestCompatibilityAdapterRejectsUnsupportedContent|TestMultimodalInterface' && go test -tags=basicv2 ./pkg/api/v2 -run '^TestBuildEmbeddingFunctionFromConfig$|^TestCollectionConfiguration_(Get|Set)EmbeddingFunction'`
- **After every plan wave:** Run `go test ./pkg/embeddings -run 'TestCapabilityMetadata|TestLegacyTextCompatibility|TestLegacyImageCompatibility|TestCompatibilityAdapterRejectsUnsupportedContent|TestMultimodalInterface' && go test -tags=basicv2 ./pkg/api/v2 -run '^TestBuildEmbeddingFunctionFromConfig$|^TestCollectionConfiguration_(Get|Set)EmbeddingFunction' && make test`
- **Before `$gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 25 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | CAPS-01, CAPS-02 | unit | `rg -n "type CapabilityMetadata struct|type CapabilityAware interface|RequestOption" pkg/embeddings && go test ./pkg/embeddings -run 'TestCapabilityMetadata'` | ❌ planned in 02-01 | ⬜ pending |
| 02-02-01 | 02 | 2 | COMP-01, COMP-02 | unit | `rg -n "ContentEmbeddingFunction|unsupported|Capabilities\\(" pkg/embeddings pkg/embeddings/roboflow && go test ./pkg/embeddings -run 'TestLegacyTextCompatibility|TestLegacyImageCompatibility|TestCompatibilityAdapterRejectsUnsupportedContent'` | ❌ planned in 02-02 | ⬜ pending |
| 02-03-01 | 03 | 3 | CAPS-02, COMP-01, COMP-02 | unit + regression | `rg -n "TestCapabilityMetadata|TestLegacyTextCompatibility|TestLegacyImageCompatibility|TestMultimodalInterface" pkg/embeddings && go test ./pkg/embeddings -run 'TestCapabilityMetadata|TestLegacyTextCompatibility|TestLegacyImageCompatibility|TestCompatibilityAdapterRejectsUnsupportedContent|TestMultimodalInterface' && go test -tags=basicv2 ./pkg/api/v2 -run '^TestBuildEmbeddingFunctionFromConfig$|^TestCollectionConfiguration_(Get|Set)EmbeddingFunction'` | ❌ planned in 02-03 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

All phase behaviors should be covered with automated verification.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 25s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
