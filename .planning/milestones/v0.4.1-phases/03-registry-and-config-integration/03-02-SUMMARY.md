---
phase: 03-registry-and-config-integration
plan: 02
subsystem: api/v2
tags: [config, auto-wiring, content-ef, multimodal, registry]
dependency_graph:
  requires: [03-01]
  provides: [BuildContentEFFromConfig, WithContentEmbeddingFunctionGet, contentEmbeddingFunction]
  affects: [pkg/api/v2/configuration.go, pkg/api/v2/client.go, pkg/api/v2/collection_http.go, pkg/api/v2/client_http.go]
tech_stack:
  added: []
  patterns: [functional-options, auto-wiring, registry-fallback-chain]
key_files:
  created: []
  modified:
    - pkg/api/v2/configuration.go
    - pkg/api/v2/client.go
    - pkg/api/v2/collection_http.go
    - pkg/api/v2/client_http.go
decisions:
  - "Derive dense EF from content EF at GetCollection time when content implements EmbeddingFunction, avoiding double initialization"
  - "Close contentEF first in CollectionImpl.Close() to avoid double-close when contentEF wraps denseEF (adapter case)"
  - "Add embeddings import to client_http.go as a Rule 3 auto-fix (missing import blocked build)"
metrics:
  duration: 2min
  completed_date: "2026-03-20"
  tasks: 2
  files: 4
---

# Phase 03 Plan 02: Config Build Chain and Content EF Auto-Wiring Summary

**One-liner:** Extended config build chain with multimodal fallback and added ContentEmbeddingFunction auto-wiring field on Collection using registry BuildContent fallback chain.

## What Was Built

Extended four files in pkg/api/v2 to complete REG-01 (build from config) and REG-02 (auto-wiring stability):

1. **configuration.go** — Three additions:
   - `BuildEmbeddingFunctionFromConfig` now tries dense, then multimodal (multimodal providers implementing EmbeddingFunction such as Roboflow can now be auto-wired as dense EF)
   - New `BuildContentEFFromConfig` delegates to `embeddings.BuildContent` which follows the full fallback chain: content factory -> multimodal+adapt -> dense+adapt
   - New `SetContentEmbeddingFunction` method type-asserts content EF to EmbeddingFunction before calling SetEmbeddingFunction for config persistence

2. **client.go** — Two additions:
   - `GetCollectionOp` gains `contentEmbeddingFunction embeddings.ContentEmbeddingFunction` field
   - `WithContentEmbeddingFunctionGet` option function for explicit content EF on GetCollection calls

3. **collection_http.go** — Three changes:
   - `CollectionImpl` gains `contentEmbeddingFunction embeddings.ContentEmbeddingFunction` field
   - `Close()` closes contentEF first, then denseEF (avoids double-close when contentEF wraps denseEF as an adapter)
   - `Fork()` propagates `contentEmbeddingFunction` to the forked collection

4. **client_http.go** — Extended GetCollection auto-wiring:
   - Added content EF auto-wiring block after dense EF block (explicit option > auto-wired)
   - When contentEF is set and implements EmbeddingFunction, derives dense EF from it (avoids double initialization)
   - Added missing `embeddings` import (auto-fix Rule 3)
   - `CollectionImpl` literal now sets `contentEmbeddingFunction: contentEF`

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Extend config build chain (configuration.go + client.go) | 66bfafc | configuration.go, client.go |
| 2 | Wire collection contentEF field and auto-wiring (collection_http.go + client_http.go) | 00560bd | collection_http.go, client_http.go |

## Acceptance Criteria Verification

- [x] configuration.go contains `func BuildContentEFFromConfig(cfg *CollectionConfigurationImpl) (embeddings.ContentEmbeddingFunction, error)`
- [x] configuration.go contains `func (c *CollectionConfigurationImpl) SetContentEmbeddingFunction(ef embeddings.ContentEmbeddingFunction)`
- [x] configuration.go BuildEmbeddingFunctionFromConfig body contains `embeddings.HasMultimodal(efInfo.Name)`
- [x] client.go GetCollectionOp struct contains `contentEmbeddingFunction embeddings.ContentEmbeddingFunction`
- [x] client.go contains `func WithContentEmbeddingFunctionGet(ef embeddings.ContentEmbeddingFunction) GetCollectionOption`
- [x] collection_http.go CollectionImpl struct contains `contentEmbeddingFunction embeddings.ContentEmbeddingFunction`
- [x] collection_http.go Close() checks contentEmbeddingFunction for io.Closer before checking embeddingFunction
- [x] collection_http.go Fork method propagates contentEmbeddingFunction to forked collection
- [x] client_http.go GetCollection contains `BuildContentEFFromConfig(configuration)`
- [x] client_http.go GetCollection assigns `contentEmbeddingFunction: contentEF` in CollectionImpl literal
- [x] client_http.go contains `if contentEF != nil && ef == nil` with type assertion to derive dense EF
- [x] `go build ./pkg/api/v2/...` exits 0
- [x] `go vet ./pkg/api/v2/...` exits 0
- [x] `make lint` reports 0 issues

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added missing embeddings import to client_http.go**
- **Found during:** Task 2
- **Issue:** client_http.go used `embeddings.EmbeddingFunction` type assertion but did not import the embeddings package
- **Fix:** Added `"github.com/amikos-tech/chroma-go/pkg/embeddings"` to import block
- **Files modified:** pkg/api/v2/client_http.go
- **Commit:** 00560bd

## Self-Check: PASSED

All 4 modified files exist on disk. Both task commits (66bfafc, 00560bd) verified in git log.
