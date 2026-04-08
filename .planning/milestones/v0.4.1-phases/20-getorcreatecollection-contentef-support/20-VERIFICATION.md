---
phase: 20-getorcreatecollection-contentef-support
verified: 2026-04-07T20:00:00Z
status: passed
score: 5/5 must-haves verified
overrides_applied: 0
re_verification: null
---

# Phase 20: GetOrCreateCollection contentEF Support Verification Report

**Phase Goal:** Add contentEmbeddingFunction support to GetOrCreateCollection by extending CreateCollectionOp with a contentEF field, adding WithContentEmbeddingFunctionCreate option, and forwarding contentEF to GetCollection in both HTTP and embedded client paths.
**Verified:** 2026-04-07T20:00:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (Success Criteria)

| #  | Truth (SC)                                                                                  | Status     | Evidence                                                                                                                         |
|----|---------------------------------------------------------------------------------------------|------------|----------------------------------------------------------------------------------------------------------------------------------|
| 1  | `CreateCollectionOp` includes a `contentEmbeddingFunction` field (SC-1)                    | VERIFIED   | `pkg/api/v2/client.go` line 248: `contentEmbeddingFunction embeddings.ContentEmbeddingFunction \`json:"-"\``                    |
| 2  | `WithContentEmbeddingFunctionCreate` option available for CreateCollection/GetOrCreateCollection (SC-2) | VERIFIED | `pkg/api/v2/client.go` line 493: `func WithContentEmbeddingFunctionCreate(ef embeddings.ContentEmbeddingFunction) CreateCollectionOption` with nil guard |
| 3  | `GetOrCreateCollection` forwards contentEF to `GetCollection` via `WithContentEmbeddingFunctionGet` (SC-3) | VERIFIED | `pkg/api/v2/client_local_embedded.go` lines 442-444: `if req.contentEmbeddingFunction != nil { getOptions = append(getOptions, WithContentEmbeddingFunctionGet(req.contentEmbeddingFunction)) }` |
| 4  | Both HTTP and embedded client paths handle the new option (SC-4)                            | VERIFIED   | HTTP: `client_http.go` line 354 `contentEmbeddingFunction: wrapContentEFCloseOnce(req.contentEmbeddingFunction)`. Embedded: `client_local_embedded.go` lines 390-412 store in state for new collections, nil for existing. |
| 5  | Tests cover GetOrCreateCollection with explicit contentEF (SC-5)                            | VERIFIED   | 9 tests all pass: 5 HTTP (`TestCreateCollectionWithContentEF`, `TestGetOrCreateCollectionWithContentEF`, `TestWithContentEmbeddingFunctionCreateNil`, `TestPrepareAndValidateCollectionRequest_ContentEFConfigPersistence`, `TestCreateCollectionWithContentEF_CloseLifecycle`) + 4 embedded (`TestEmbeddedCreateCollection_ContentEF_NewCollection`, `TestEmbeddedCreateCollection_ContentEF_ExistingCollection`, `TestEmbeddedGetOrCreateCollection_ContentEF_ForwardedToGetCollection`, `TestEmbeddedGetOrCreateCollection_ContentEF_VerifyViaSubsequentGetCollection`) |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact                                        | Expected                                                         | Status     | Details                                                                                       |
|-------------------------------------------------|------------------------------------------------------------------|------------|-----------------------------------------------------------------------------------------------|
| `pkg/api/v2/client.go`                          | CreateCollectionOp.contentEmbeddingFunction field + option       | VERIFIED   | Field at line 248, `WithContentEmbeddingFunctionCreate` at line 493, config persistence at lines 302-316 |
| `pkg/api/v2/client_http.go`                     | HTTP CreateCollection contentEF wiring with close-once wrapping  | VERIFIED   | `wrapContentEFCloseOnce(req.contentEmbeddingFunction)` at line 354                            |
| `pkg/api/v2/client_local_embedded.go`           | Embedded CreateCollection state + GetOrCreateCollection forward  | VERIFIED   | State storage at lines 390-410, forwarding at lines 442-444                                   |
| `pkg/api/v2/client_http_test.go`                | HTTP contentEF tests (5 functions)                               | VERIFIED   | All 5 test functions present and passing                                                      |
| `pkg/api/v2/client_local_embedded_test.go`      | Embedded contentEF tests (4 functions)                           | VERIFIED   | All 4 test functions present and passing                                                      |

### Key Link Verification

| From                             | To                                | Via                                                            | Status     | Details                                                                 |
|----------------------------------|-----------------------------------|----------------------------------------------------------------|------------|-------------------------------------------------------------------------|
| `pkg/api/v2/client.go`           | `pkg/api/v2/client_http.go`       | `CreateCollectionOp.contentEmbeddingFunction` consumed by HTTP | VERIFIED   | `req.contentEmbeddingFunction` at line 354 of client_http.go            |
| `pkg/api/v2/client.go`           | `pkg/api/v2/client_local_embedded.go` | `CreateCollectionOp.contentEmbeddingFunction` consumed by embedded | VERIFIED | `req.contentEmbeddingFunction` at lines 390-396 of client_local_embedded.go |
| `pkg/api/v2/client_local_embedded.go` | `pkg/api/v2/client.go`       | GetOrCreateCollection forwards via `WithContentEmbeddingFunctionGet` | VERIFIED | `WithContentEmbeddingFunctionGet(req.contentEmbeddingFunction)` at line 443 |

### Data-Flow Trace (Level 4)

| Artifact                         | Data Variable              | Source                                         | Produces Real Data | Status     |
|----------------------------------|----------------------------|------------------------------------------------|--------------------|------------|
| `client_http.go` CreateCollection | `contentEmbeddingFunction` | `req.contentEmbeddingFunction` from caller     | Yes (caller-provided, wrapped with close-once) | FLOWING |
| `client_local_embedded.go` CreateCollection | `overrideContentEF` | `req.contentEmbeddingFunction` from caller, stored in state for new collections | Yes | FLOWING |
| `client_local_embedded.go` GetOrCreateCollection | `contentEmbeddingFunction` | Forwarded via `WithContentEmbeddingFunctionGet` to `GetCollection` | Yes | FLOWING |

### Behavioral Spot-Checks

| Behavior                                                    | Command                                                                                                       | Result                                      | Status  |
|-------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------|---------------------------------------------|---------|
| HTTP CreateCollection with contentEF sets non-nil field     | `go test -tags=basicv2 -run TestCreateCollectionWithContentEF ./pkg/api/v2/...`                              | PASS                                        | PASS    |
| HTTP GetOrCreateCollection delegates contentEF              | `go test -tags=basicv2 -run TestGetOrCreateCollectionWithContentEF ./pkg/api/v2/...`                         | PASS                                        | PASS    |
| nil contentEF is rejected                                   | `go test -tags=basicv2 -run TestWithContentEmbeddingFunctionCreateNil ./pkg/api/v2/...`                      | PASS                                        | PASS    |
| Config persistence for dual-interface and content-only EFs  | `go test -tags=basicv2 -run TestPrepareAndValidateCollectionRequest_ContentEFConfigPersistence ./pkg/api/v2/...` | PASS (2 sub-tests)                       | PASS    |
| Close lifecycle (close-once idempotency)                    | `go test -tags=basicv2 -run TestCreateCollectionWithContentEF_CloseLifecycle ./pkg/api/v2/...`               | PASS                                        | PASS    |
| Embedded CreateCollection stores contentEF for new collection | `go test -tags=basicv2 -run TestEmbeddedCreateCollection_ContentEF_NewCollection ./pkg/api/v2/...`          | PASS                                        | PASS    |
| Embedded CreateCollection ignores contentEF for existing collection | `go test -tags=basicv2 -run TestEmbeddedCreateCollection_ContentEF_ExistingCollection ./pkg/api/v2/...` | PASS                                    | PASS    |
| Embedded GetOrCreateCollection forwards contentEF via GetCollection | `go test -tags=basicv2 -run TestEmbeddedGetOrCreateCollection_ContentEF_ForwardedToGetCollection ./pkg/api/v2/...` | PASS                               | PASS    |
| State carry-forward to subsequent GetCollection             | `go test -tags=basicv2 -run TestEmbeddedGetOrCreateCollection_ContentEF_VerifyViaSubsequentGetCollection ./pkg/api/v2/...` | PASS                         | PASS    |

All 9 phase-20 tests pass (run combined: 0.468s).

### Requirements Coverage

| Requirement | Source Plan | Description                                                  | Status    | Evidence                                                |
|-------------|------------|--------------------------------------------------------------|-----------|---------------------------------------------------------|
| SC-1        | 20-01, 20-02 | CreateCollectionOp includes contentEmbeddingFunction field | SATISFIED | `client.go` line 248                                    |
| SC-2        | 20-01, 20-02 | WithContentEmbeddingFunctionCreate option available          | SATISFIED | `client.go` line 493                                    |
| SC-3        | 20-01, 20-02 | GetOrCreateCollection forwards contentEF via WithContentEmbeddingFunctionGet | SATISFIED | `client_local_embedded.go` lines 442-444       |
| SC-4        | 20-01, 20-02 | Both HTTP and embedded paths handle new option               | SATISFIED | HTTP line 354, embedded lines 390-412                   |
| SC-5        | 20-01, 20-02 | Tests cover GetOrCreateCollection with explicit contentEF    | SATISFIED | 9 passing tests across HTTP and embedded paths          |

Note: SC-1 through SC-5 are phase-level success criteria from ROADMAP.md, not IDs in REQUIREMENTS.md. No REQUIREMENTS.md IDs (e.g. MMOD-XX, CAPS-XX) are mapped to phase 20 in the traceability table — this phase introduces new functionality outside the existing requirement IDs.

### Anti-Patterns Found

No anti-patterns detected in the three modified production files (`client.go`, `client_http.go`, `client_local_embedded.go`). No TODO, FIXME, placeholder comments, or empty return stubs found.

### Human Verification Required

None. All success criteria are verifiable programmatically and all tests pass.

### Gaps Summary

No gaps. All five success criteria are met with evidence in the actual codebase. Build passes, 9 new tests pass, no anti-patterns found.

---

_Verified: 2026-04-07T20:00:00Z_
_Verifier: Claude (gsd-verifier)_
