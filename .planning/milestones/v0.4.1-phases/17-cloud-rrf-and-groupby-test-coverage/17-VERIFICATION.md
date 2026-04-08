---
phase: 17-cloud-rrf-and-groupby-test-coverage
verified: 2026-04-02T08:00:00Z
status: passed
score: 5/5 must-haves verified
re_verification: false
gaps: []
human_verification:
  - test: "Run TestCloudClientSearchRRF against live Chroma Cloud"
    expected: "RRF smoke subtest returns quantum docs (IDs 1, 3, or 5) as the first result; weighted subtest produces non-equal score slices for equal-weight vs heavy-dense configurations"
    why_human: "Requires live CHROMA_API_KEY, CHROMA_DATABASE, CHROMA_TENANT credentials and a running Cloud instance; cannot be verified without external service"
  - test: "Run TestCloudClientSearchGroupBy against live Chroma Cloud"
    expected: "MinK subtest: per-category counts <= 2 across >=2 categories; MaxK subtest: same per-group cap with numeric priority metadata present"
    why_human: "Same Cloud credential dependency; RowGroups() grouping behavior only observable at runtime against a real index"
---

# Phase 17: Cloud RRF and GroupBy Test Coverage Verification Report

**Phase Goal:** Add end-to-end cloud integration tests that exercise Search API RRF and GroupBy primitives against live Chroma Cloud.
**Verified:** 2026-04-02
**Status:** passed (automated) / human_needed (runtime behavior)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|---------|
| 1 | RRF search with dense + sparse KNN ranks returns ranked results against live Cloud | VERIFIED | `TestCloudClientSearchRRF` subtest "RRF smoke" at line 1460 creates both dense and sparse KNN ranks, executes `WithRrfRank(WithRrfRanks(...))`, asserts `require.NotEmpty(sr.IDs)`, `require.NotEmpty(sr.Scores)`, and checks `sr.IDs[0][0]` is in quantum set |
| 2 | RRF weight changes produce observable ordering differences in results | VERIFIED | Subtest "RRF with custom k and different weights" at line 1518 runs two searches with different weight/k configs and asserts `assert.NotEqual(t, srA.Scores, srB.Scores)` |
| 3 | GroupBy MinK caps per-group result count to k | VERIFIED | `TestCloudClientSearchGroupBy` subtest "MinK caps results per group" at line 1600 iterates `sr.RowGroups()`, counts per-category occurrences, and asserts `assert.LessOrEqual(t, count, 2)` for each category |
| 4 | GroupBy MaxK caps per-group result count to k | VERIFIED | Subtest "MaxK selects top k per group" at line 1666 uses same RowGroups iteration pattern with `NewMaxK(2, KScore)` and same `assert.LessOrEqual(t, count, 2)` per category |
| 5 | All new tests use cloud build tags and existing setupCloudClient infrastructure | VERIFIED | File-level `//go:build basicv2 && cloud` at line 1 applies to all functions; both new test functions call `setupCloudClient(t)` as their first statement |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|---------|--------|---------|
| `pkg/api/v2/client_cloud_test.go` | Contains `func TestCloudClientSearchRRF` | VERIFIED | Function exists at line 1457 |
| `pkg/api/v2/client_cloud_test.go` | Contains `func TestCloudClientSearchGroupBy` | VERIFIED | Function exists at line 1597 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `client_cloud_test.go` | `pkg/api/v2/rank.go` | `NewKnnRank`, `WithRrfRank`, `WithKnnReturnRank`, `WithRrfRanks` | WIRED | Lines 1493-1495, 1500, 1552-1554, 1559, 1572-1574, 1579 |
| `client_cloud_test.go` | `pkg/api/v2/rank.go` | `WithKnnKey(K("sparse_embedding"))` | WIRED | Lines 1495, 1554, 1574 — sparse KNN rank uses key |
| `client_cloud_test.go` | `pkg/api/v2/rank.go` | `WithRrfK(` | WIRED | Lines 1559 (`WithRrfK(60)`), 1579 (`WithRrfK(10)`) |
| `client_cloud_test.go` | `pkg/api/v2/groupby.go` | `NewGroupBy` | WIRED | Lines 1639, 1705 |
| `client_cloud_test.go` | `pkg/api/v2/aggregate.go` | `NewMinK`, `NewMaxK` | WIRED | Line 1639 (`NewMinK(2, KScore)`), line 1705 (`NewMaxK(2, KScore)`) |
| `client_cloud_test.go` | `pkg/api/v2/search.go` | `RowGroups()` (not `Rows()`) | WIRED | Lines 1652, 1718 — both GroupBy subtests use `sr.RowGroups()` |

### Data-Flow Trace (Level 4)

Not applicable. This phase produces test code only. Test functions exercise existing API surface; there is no rendering pipeline to trace. The assertions are behavioral: they check live Cloud responses at test runtime.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| File compiles and lints | `make lint` | `0 issues.` | PASS |
| 3+ Search test functions defined | `grep -c 'func TestCloudClientSearch' ...` | 3 (TestCloudClientSearch, TestCloudClientSearchRRF, TestCloudClientSearchGroupBy) | PASS |
| Commits documented in SUMMARY exist in git | `git log --oneline fdc2fbb 4159460` | Both commits present | PASS |
| RowGroups used (not Rows) in GroupBy tests | grep pattern | Both GroupBy subtests use `sr.RowGroups()` at lines 1652 and 1718 | PASS |
| Live Cloud execution | Requires credentials | SKIP — external service |

### Requirements Coverage

The PLAN declares `requirements: [SC-01, SC-02, SC-03, SC-04]`. These IDs are not defined in REQUIREMENTS.md — they are phase-local codes that map to the 4 Success Criteria in the ROADMAP Phase 17 entry. REQUIREMENTS.md's traceability table does not include Phase 17, and no global requirement IDs (e.g., MMOD-*, CAPS-*, etc.) are assigned to this phase. This is expected: Phase 17 is a testing/coverage hardening phase rather than a feature phase.

| Requirement (Phase-local) | ROADMAP Success Criterion | Status | Evidence |
|--------------------------|--------------------------|--------|---------|
| SC-01 | RRF smoke test using dense + sparse KNN ranks with `WithKnnReturnRank` | SATISFIED | Smoke subtest lines 1460-1516 |
| SC-02 | RRF weighted/custom-k test proves request acceptance and ordering changes | SATISFIED | Weights subtest lines 1518-1594, `assert.NotEqual(srA.Scores, srB.Scores)` |
| SC-03 | GroupBy MinK/MaxK tests assert per-group caps and flattened limits | SATISFIED | Both GroupBy subtests iterate `RowGroups()` and assert `LessOrEqual(count, 2)` |
| SC-04 | All tests tagged `cloud` and use existing cloud test infrastructure | SATISFIED | File-level build tag `//go:build basicv2 && cloud`; `setupCloudClient(t)` in both functions |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | — | — | No anti-patterns detected in added code |

Checked for: TODO/FIXME comments, empty return stubs, hardcoded empty data flowing to assertions, props hardcoded at call site. None found in the new test functions (lines 1457-1731).

### Human Verification Required

#### 1. RRF Live Cloud Execution

**Test:** `go test -tags=basicv2,cloud -run "TestCloudClientSearchRRF" -v -timeout=5m ./pkg/api/v2/...` with valid CHROMA_API_KEY, CHROMA_DATABASE, CHROMA_TENANT set.
**Expected:** Both subtests pass. Smoke test reports first result as one of IDs 1, 3, or 5 (quantum docs). Weight test reports `srA.Scores != srB.Scores`.
**Why human:** Requires live Chroma Cloud credentials. Cannot be executed without external service access.

#### 2. GroupBy Live Cloud Execution

**Test:** `go test -tags=basicv2,cloud -run "TestCloudClientSearchGroupBy" -v -timeout=5m ./pkg/api/v2/...` with valid credentials.
**Expected:** Both subtests pass. MinK subtest shows `categoryCounts` has >= 2 categories, each with count <= 2. MaxK subtest shows same constraint with priority metadata populated.
**Why human:** Same Cloud credential dependency. GroupBy grouping correctness and `RowGroups()` multi-group iteration are only observable against a real indexed collection.

### Gaps Summary

No gaps. All automated checks pass:
- Both required test functions exist in `pkg/api/v2/client_cloud_test.go` with correct implementations.
- All 4 subtests are present and match the plan specifications exactly.
- All key links to `rank.go`, `groupby.go`, `aggregate.go`, and `search.go` are verified.
- The critical correctness constraint (use `RowGroups()` not `Rows()` for GroupBy iteration) is correctly implemented in both GroupBy subtests.
- `chromacloudsplade.NewEmbeddingFunction` used per D-01 for RRF sparse embeddings.
- `assert.NotEqual(t, srA.Scores, srB.Scores)` full score slice comparison per D-02.
- Build tag `//go:build basicv2 && cloud` inherited file-level per D-03.
- `make lint` exits 0.
- Both task commits (`fdc2fbb`, `4159460`) verified present in git history.

Runtime behavior against live Chroma Cloud requires human execution with cloud credentials.

---

_Verified: 2026-04-02_
_Verifier: Claude (gsd-verifier)_
