# Requirements: Chroma Go

**Defined:** 2026-04-08
**Core Value:** Go applications can use Chroma and embedding providers through a stable, portable API that minimizes provider-specific friction.

## v1 Requirements

### API Bugs

- [ ] **RANK-01**: RrfRank arithmetic methods (Multiply, Sub, Add, Div, Negate) compute correct composite rank expressions instead of returning self
- [ ] **RANK-02**: RrfRank arithmetic results produce valid JSON when marshaled
- [ ] **GRP-01**: WithGroupBy(nil) returns a validation error instead of silently skipping grouping

### Embedded Client Lifecycle

- [ ] **EFL-01**: Default ORT EF created by PrepareAndValidateCollectionRequest is closed when CreateCollection finds an existing collection
- [ ] **EFL-02**: GetOrCreateCollection does not pass closed EFs to CreateCollection fallback when GetCollection fails mid-build
- [ ] **EFL-03**: Tests cover EF lifecycle under `-race` flag for concurrent GetOrCreateCollection calls

### Error Handling

- [ ] **ERR-01**: Shared `SanitizeErrorBody` utility truncates HTTP error bodies to a safe display length with `[truncated]` suffix
- [ ] **ERR-02**: All embedding providers use `SanitizeErrorBody` for error message construction instead of raw `string(respData)`

### Provider Enhancement

- [ ] **TLA-01**: Twelve Labs provider detects async task responses from the sync endpoint and enters a polling loop
- [ ] **TLA-02**: Async polling respects caller context for cancellation and timeout
- [ ] **TLA-03**: Async polling handles terminal states (ready, failed) with appropriate error messages
- [ ] **TLA-04**: Tests cover async task creation, polling, completion, failure, and context cancellation

### Internal Cleanup

- [ ] **DL-01**: `default_ef/download_utils.go` uses `downloadutil.DownloadFile` instead of its own HTTP download implementation
- [ ] **DL-02**: Existing download tests pass unchanged after consolidation
- [ ] **MORPH-01**: Morph EF integration test handles upstream 404 gracefully (skip or updated URL)

## v2 Requirements

### Provider Adoption

- **PROV-01**: Additional providers beyond the current multimodal baseline adopt the richer shared multimodal contract
- **PROV-02**: End-to-end examples cover concrete audio, video, or PDF provider implementations once they exist

### Advanced Discovery

- **DISC-01**: Capability discovery can be derived from remote or generated provider metadata instead of only static code declarations
- **DISC-02**: Shared batching helpers optimize multimodal requests where providers expose compatible batch semantics

## Out of Scope

| Feature | Reason |
|---------|--------|
| Full download stack unification (cosign, mirrors, lock files) | Bounded to `default_ef` only; broader unification deferred to v0.4.3+ |
| RrfRank real algebraic expression nodes | ErrorRank sentinel is safer for existing callers; can revisit if use cases emerge |
| Twelve Labs async as a separate interface method | Option-based approach fits existing EmbedContent pattern |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| RANK-01 | - | Pending |
| RANK-02 | - | Pending |
| GRP-01 | - | Pending |
| EFL-01 | - | Pending |
| EFL-02 | - | Pending |
| EFL-03 | - | Pending |
| ERR-01 | - | Pending |
| ERR-02 | - | Pending |
| TLA-01 | - | Pending |
| TLA-02 | - | Pending |
| TLA-03 | - | Pending |
| TLA-04 | - | Pending |
| DL-01 | - | Pending |
| DL-02 | - | Pending |
| MORPH-01 | - | Pending |

**Coverage:**
- v1 requirements: 15 total
- Mapped to phases: 0
- Unmapped: 15

---
*Requirements defined: 2026-04-08*
*Last updated: 2026-04-08 after initial definition*
