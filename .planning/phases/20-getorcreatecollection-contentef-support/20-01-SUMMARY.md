---
phase: 20-getorcreatecollection-contentef-support
plan: 01
subsystem: api
tags: [contentEF, embedding-function, collection-lifecycle, close-once, config-persistence]

requires:
  - phase: 19-embedded-client-ef-lifecycle-hardening
    provides: close-once wrappers, state map lifecycle, buildEmbeddedCollection contentEF parameter
provides:
  - CreateCollectionOp.contentEmbeddingFunction field and WithContentEmbeddingFunctionCreate option
  - HTTP CreateCollection contentEF wiring with close-once wrapping
  - Embedded CreateCollection contentEF state storage for new collections
  - Embedded GetOrCreateCollection contentEF forwarding to GetCollection
  - PrepareAndValidateCollectionRequest contentEF config persistence (after denseEF)
affects: [20-02, collection-tests, content-embedding-function]

tech-stack:
  added: []
  patterns: [contentEF config persistence ordering after denseEF for precedence]

key-files:
  created: []
  modified:
    - pkg/api/v2/client.go
    - pkg/api/v2/client_http.go
    - pkg/api/v2/client_local_embedded.go

key-decisions:
  - "contentEF config persists after denseEF so contentEF takes precedence when it also implements EmbeddingFunction"
  - "HTTP GetOrCreateCollection delegates to CreateCollection (no separate GetCollection call)"

patterns-established:
  - "contentEF config persistence ordering: denseEF first, contentEF second for precedence semantics"
  - "Embedded CreateCollection ignores user-provided contentEF for existing collections (isNewCreation=false)"

requirements-completed: [SC-1, SC-2, SC-4, SC-5]

duration: 2min
completed: 2026-04-07
---

# Phase 20 Plan 01: GetOrCreateCollection contentEF Support Summary

**contentEmbeddingFunction field added to CreateCollectionOp with config persistence, HTTP close-once wiring, and embedded state/forwarding**

## Performance

- **Duration:** 2 min
- **Started:** 2026-04-07T15:40:43Z
- **Completed:** 2026-04-07T15:42:34Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Added contentEmbeddingFunction field to CreateCollectionOp and WithContentEmbeddingFunctionCreate option function
- Extended PrepareAndValidateCollectionRequest to persist contentEF config after denseEF through both Schema and Configuration paths
- Wired contentEF through HTTP CreateCollection with close-once wrapping
- Wired contentEF through embedded CreateCollection with state storage and GetOrCreateCollection forwarding

## Task Commits

Each task was committed atomically:

1. **Task 1: Add contentEF field to CreateCollectionOp and WithContentEmbeddingFunctionCreate option** - `628dced` (feat)
2. **Task 2: Wire contentEF through HTTP and embedded client paths** - `893afeb` (feat)

## Files Created/Modified
- `pkg/api/v2/client.go` - CreateCollectionOp struct extension, WithContentEmbeddingFunctionCreate option, PrepareAndValidateCollectionRequest contentEF config persistence
- `pkg/api/v2/client_http.go` - HTTP CreateCollection contentEF wiring with wrapContentEFCloseOnce
- `pkg/api/v2/client_local_embedded.go` - Embedded CreateCollection contentEF state storage, GetOrCreateCollection contentEF forwarding

## Decisions Made
- contentEF config persistence block placed after denseEF block so contentEF takes precedence when it also implements EmbeddingFunction (overrides default ORT fallback in persisted config)
- HTTP GetOrCreateCollection delegates to CreateCollection without a separate GetCollection call -- contentEF flows through the existing delegation pattern
- No redundant disableEFConfigStorage guard on contentEF block since the early return already gates both blocks

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All collection creation paths now accept contentEF
- Ready for Plan 02: test coverage for contentEF in CreateCollection and GetOrCreateCollection

## Self-Check: PASSED

- All 3 modified files exist on disk
- Both task commits verified: 628dced, 893afeb
- SUMMARY.md created at expected path

---
*Phase: 20-getorcreatecollection-contentef-support*
*Completed: 2026-04-07*
