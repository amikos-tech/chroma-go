# Phase 17: Cloud RRF and GroupBy Test Coverage - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-02
**Phase:** 17-cloud-rrf-and-groupby-test-coverage
**Areas discussed:** Sparse embedding setup, Assertion depth, Test file organization

---

## Sparse Embedding Setup

| Option | Description | Selected |
|--------|-------------|----------|
| chromacloudsplade EF | Use existing chromacloudsplade embedding function for real server-side sparse vectors. Already imported in cloud tests, mirrors real-world RRF usage. | ✓ |
| Manual sparse vectors | Inject hand-crafted sparse embedding vectors directly. More predictable for assertions but less realistic. | |

**User's choice:** chromacloudsplade EF (Recommended)
**Notes:** None

---

## Assertion Depth

| Option | Description | Selected |
|--------|-------------|----------|
| Behavioral | Verify ordering changes with different RRF weights, confirm GroupBy enforces per-group MinK/MaxK caps. Meaningful coverage that catches regressions. | ✓ |
| Smoke-level | Just assert no errors and results are non-empty. Fast to write, but won't catch subtle behavioral regressions. | |
| Structural + behavioral | Behavioral plus validate response shape. Most thorough but more brittle against server changes. | |

**User's choice:** Behavioral (Recommended)
**Notes:** None

---

## Test File Organization

| Option | Description | Selected |
|--------|-------------|----------|
| Existing client_cloud_test.go | Keeps all cloud integration tests in one file. Already has setupCloudClient, cleanup, and basic search tests. | ✓ |
| New client_cloud_search_test.go | Separates search-specific cloud tests from CRUD tests. Cleaner separation but new file for same tags/setup. | |

**User's choice:** Existing client_cloud_test.go (Recommended)
**Notes:** None

---

## Claude's Discretion

- Test data content (document texts, metadata values)
- Number of documents per test collection
- Specific RRF weight values

## Deferred Ideas

None
