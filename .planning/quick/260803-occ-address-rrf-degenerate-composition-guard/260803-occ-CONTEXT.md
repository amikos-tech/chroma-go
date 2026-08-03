# Quick Task 260803-occ: address RRF degenerate composition guards (#497 #498 #500 #501) - Context

**Gathered:** 2026-08-03
**Status:** Ready for planning

<domain>
## Task Boundary

Validate and address the client-side defects in issues #497, #498, #500 (and #501,
which has identical scope to the client-side remedy for #497/#498).

All four collapse into two client-side changes in `pkg/api/v2`:

1. **Build-time guard** — reject provably degenerate `*RrfRank` arithmetic
   compositions before a query is ever serialized. Covers #497, #498, #501.
2. **Read-time guard** — detect result-shape degeneration on `Search` responses.
   Covers #500.

Out of scope: server-side changes, upstream chroma-core reports, V1 API.

</domain>

<validation>
## Pre-Planning Validation Findings

Root cause (confirmed at `pkg/api/v2/rank.go:1517`): `RrfRank.marshalJSONWithDepth`
negates the reciprocal-rank sum (`result := rrfSum.Negate()`) to match Chroma's
lower-is-better ordering convention. Every arithmetic method on `*RrfRank`
therefore composes against a value that is **always <= 0** on a non-empty corpus.

| Issue | Original claim | Verdict |
|-------|----------------|---------|
| #497 | "Server-behavior defect — Log degenerates silently" | **Misframed.** `f32::ln(x<=0) = NaN`; the server drops NaN rows. Deterministic IEEE 754 behavior, not a server bug. #500's body already self-corrects this framing. Only the client-side remedy is actionable here. |
| #498 | "`Max(Val(0))` collapses to all-zero; add client guard" | **Valid.** `max(x, 0) == 0` for all x <= 0. The client has full static visibility into the composition and can reject it. |
| #500 | "Client does not detect result-shape mismatch" | **Valid, with a caveat.** The naive check (`IDs[0]` non-empty + `Scores[0]` empty) would fire on every ordinary query, because `Scores` is only populated when `KScore` is present in `WithSelect`. The guard MUST be gated on the request's `Select`. |
| #501 | "Reject meaningless compositions at build time" | **Valid and identical** to the client-side remedy for #497/#498. Shipped by the same change. |

Enabling fact for the read-time guard: `CollectionImpl.Search`
(`pkg/api/v2/collection_http.go:426`) still holds the parsed `*SearchQuery`
when it decodes the response, so per-request `Select` is available at
validation time without any plumbing changes.

### Upstream cross-check (chroma-core @ d1921129)

Checked `chromadb/execution/expression/operator.py` and
`rust/types/src/execution/operator.rs` before planning:

- Python `Rrf` is a plain `Rank` subclass — inherits `.log()`, `.max()` with
  **zero** guards. `Rrf.to_dict()` validates only ranks/k/weights, then returns
  `(-rrf_sum).to_dict()`.
- Rust `rrf()` (`operator.rs:2868`) is `Ok(-sum)`. `RankExpr::Logarithm`
  evaluates to a bare `f32::ln` (`rust/worker/src/execution/operators/rank.rs:115`).
  No guards anywhere.
- Upstream's stated position on the log footgun is a doc comment:
  *"Logarithm - compresses range (add constant to avoid log(0))"*.

**This does NOT change the guard decisions.** See "Consistency principle" below.

### The real bug — null misalignment (VERIFIED, not described in any issue)

The wire contract models scores as `Vec<Option<Vec<Option<f32>>>>` and documents
as `Vec<Option<Vec<Option<String>>>>` (`rust/types/src/api_types.rs:2493-2499`),
so a per-element `null` is **legal, intended protocol** — not an anomaly.

`SearchResultImpl.UnmarshalJSON` type-switches each element with no `null` case:
- Scores: `pkg/api/v2/search.go:930-940` — `case float64` / `case json.Number` only.
- Documents: `pkg/api/v2/search.go:840` — `if docStr, ok := doc.(string); ok`.

A JSON `null` matches neither branch and is **silently skipped instead of
appended**, so every later element shifts left one position while `IDs` stays
put. `Metadatas` (`search.go:862`) and `Embeddings` (`search.go:893`) handle nil
correctly — only Scores and Documents are broken.

Empirically confirmed against `{"ids":[["a","b","c"]],"scores":[[1.5,null,3.5]],"documents":[["da",null,"dc"]]}`:

```
row id=a  score=1.5  doc="da"     correct
row id=b  score=3.5  doc="dc"     WRONG — b receives c's score and c's document
row id=c  score=0    doc=""       WRONG — real score silently becomes 0.0
```

This is silent data corruption binding scores/documents to the wrong document.
It fires on ANY query returning a null score or document — no RRF involved. It
is the actual mechanism behind #500's "result-shape mismatch" symptom; the issue
diagnosed RRF degeneration as the cause, which is wrong.

**Ordering constraint:** the alignment fix MUST land with the read-time guard.
Without it, `len(Scores[g]) != len(IDs[g])` is caused by ordinary nulls, so the
guard would false-positive on legitimate responses.

</validation>

<consistency_principle>
## Consistency Principle (governs the guard decisions)

The deciding invariant is **the Go client's own contract**, not parity with the
Python/Rust SDKs. `pkg/api/v2` already rejects, with typed errors, many things
upstream accepts silently:

| Go behavior | Upstream equivalent |
|-------------|---------------------|
| `MaxExpressionDepth = 100` (#530) | no limit |
| `MaxRrfRanks = 100` | no limit |
| `MaxExpressionTerms = 1000` | no limit |
| NaN/Inf weights rejected (`rank.go:1392`) | not checked |
| `ErrNilRank` on nil operands (#531) | n/a |
| eager option validation (#532) | validated at request time |

That divergence is a deliberate, sustained house style: **reject invalid or
degenerate input eagerly with an exported sentinel error**. Going permissive on
RRF degeneration to match Python would make the client inconsistent with itself,
which is the worse failure.

Scope limit: strictness applies to **caller mistakes**, not to legal server
responses. Per-element `null` in scores/documents is explicitly modeled in the
wire protocol, so it must be handled correctly — never errored on.

</consistency_principle>

<decisions>
## Implementation Decisions

### Build-time guard scope — LOCKED
Reject **only** `rrf.Log()` and `rrf.Max(Val(0))`.

Explicitly left legal:
- `rrf.Abs()` — reverses ordering, but well-defined; the doc comment at
  `rank.go:1321` already warns.
- `rrf.Negate()` — same; legitimate when the caller genuinely wants ascending order.
- `rrf.Min(Val(0))` — a harmless no-op (`min(x,0) == x` for x <= 0). Do **not**
  silently rewrite it away; silently mutating the user's expression tree is its
  own surprise.

Rationale: lowest false-positive risk. Only provably-degenerate cases are blocked.

### Guard mechanism — LOCKED
Deferred-error pattern, per the sketch in #501 and the existing `*UnknownRank`
precedent at `rank.go:85-105`: the arithmetic method returns a `Rank` whose
`MarshalJSON` fails with a descriptive error. This keeps the fluent API
ergonomic (no `(Rank, error)` return-type break) while making the query
impossible to send.

`Max` must only reject when the operand is a **literal zero constant** that is
statically knowable at build time. A non-constant operand, or a non-zero
constant, stays legal.

### Null-element alignment fix — LOCKED (new, highest severity)
`SearchResultImpl.UnmarshalJSON` must **preserve position** for `null` elements
in `scores` and `documents` instead of skipping them. Add an explicit nil case to
both type switches so index `i` always corresponds to `IDs[g][i]`.

Match the nil-handling already used for Metadatas (`search.go:862`) and
Embeddings (`search.go:893`) — this makes all four field parsers consistent.

### Absence representation — LOCKED
`ResultRow` gains **additive** boolean companions; existing fields are unchanged
so this is backward compatible:

```go
type ResultRow struct {
    ID          DocumentID
    Document    string
    HasDocument bool // false when the server sent null / field not selected
    Metadata    DocumentMetadata
    Embedding   []float32
    Score       float64
    HasScore    bool // false when the server sent null / field not selected
}
```

Rejected: changing `Score` to `*float64` — cleaner model, but a breaking change
to a public struct field.

Note `ResultRow` is shared by Search, Query, and Get (`pkg/api/v2/results.go:14`).
Set the new booleans correctly on **all** construction paths, not just Search, or
the field means different things depending on which call produced the row.

### Read-time guard behavior — LOCKED
`Search` returns an **error** when, for any query group `g`:
- the corresponding request selected `KScore`, AND
- `len(IDs[g]) > 0`, AND
- `len(Scores[g]) != len(IDs[g])`

Use an **exported sentinel** so callers can `errors.Is` it, matching `ErrNilRank`
(`pkg/api/v2/options.go:936`). Return `(nil, err)` — same shape as every other
guard in this package.

Only meaningful after the alignment fix: once nulls preserve position, a
cardinality mismatch can only come from genuine server-side degeneration.

Rejected alternatives:
- Returning result-plus-sentinel-error: Go callers routinely drop the result on
  non-nil err, so the extra payload is wasted, and it deviates from how every
  other guard in this package returns.
- A `Degenerate() bool` predicate: opt-in, so most callers never call it — which
  is precisely the silent-failure mode #500 exists to close.
- Gating behind a client option for one release: leaves the silent failure as the
  default for a release; #531/#532 shipped their guards without a deprecation
  window.

### Issue disposition — LOCKED
The PR closes **all four**: #497, #498, #500, #501.

Note in the PR body that #497's "server-behavior defect" framing is incorrect —
the behavior is deterministic IEEE 754 math and upstream has no guard; the Go
client addresses it client-side per the consistency principle above.

### Claude's Discretion
- Exact error variable names, wording, and whether the sentinel errors are
  exported (prefer exported sentinels so callers can `errors.Is` them).
- Placement of the read-time check (inline in `Search` vs. a helper on
  `SearchResultImpl`) — but it must run inside `Search` so the error surfaces
  without caller opt-in.
- Test structure and table shape.

</decisions>

<specifics>
## Specific Ideas

- Existing cloud test `TestCloudClientSearchRRFArithmetic` in
  `pkg/api/v2/client_cloud_test.go` currently **pins the degenerate behavior**
  as a regression assertion for both the `Log` and `Max_0` cases. Those pins
  must flip to assert the new error paths.
- Per CLAUDE.md: no panics in production code, no `Must*` functions, run
  `make lint` before committing, conventional commits.
- Unit tests are the primary bar here since the guards are client-side and
  need no server round-trip; the cloud test pins still need updating so
  `make test-cloud` stays green.

</specifics>

<canonical_refs>
## Canonical References

- `pkg/api/v2/rank.go:1305-1328` — existing doc comment enumerating the
  degenerate transforms (Log, Max(0), Abs, Negate).
- `pkg/api/v2/rank.go:85-105` — `*UnknownRank`, the deferred-error precedent.
- `pkg/api/v2/rank.go:1517` — the auto-negation that causes all of this.
- `pkg/api/v2/collection_http.go:426-457` — `Search`, where the read-time guard lands.
- `pkg/api/v2/search.go:773-793` — `SearchResultImpl` field shape.
- `pkg/api/v2/search.go:177` — `KScore`.
- Chroma docs: results ordered by score ascending (lower is better).

</canonical_refs>
