---
phase: quick-260731-n7b
plan: 01
subsystem: api
tags: [go, godoc, rank, v2-api, nil-safety]

requires:
  - phase: quick-260731-fvo
    provides: typed-nil Rank guard work that established the nil-handling doc precedent on the Rank interface
provides:
  - "UnknownRank godoc documenting the nil-receiver panic on promoted methods, its cause, and why it is unsupported usage rather than a defect"
affects: [rank, v2-search, nil-handling-docs]

tech-stack:
  added: []
  patterns:
    - "Nil-receiver boundaries on unconstructable sentinel types are documented, not defended, when the public API cannot produce a nil instance"

key-files:
  created: []
  modified:
    - pkg/api/v2/rank.go

key-decisions:
  - "Documented the nil *UnknownRank panic instead of switching the embedded ValRank to *ValRank — issue #526 rejected the type rework since UnknownRank has no exported constructor and operandToRank only ever returns a non-nil &UnknownRank{}."
  - "Matched the existing Rank interface nil-pointer caveat wording (unsupported usage) so both boundaries read as one policy rather than two unrelated warnings."

patterns-established:
  - "Godoc link brackets ([Rank]) used when cross-referencing the interface-level nil contract from concrete type docs"

requirements-completed: [GH-526]

duration: 6min
completed: 2026-07-31
---

# Quick Task 260731-n7b: Document UnknownRank nil-receiver panic Summary

**UnknownRank's godoc now explains that promoted methods (including the empty-bodied IsOperand) panic on a nil receiver because ValRank is embedded by value, and frames it as unsupported usage unreachable through the public API.**

## Performance

- **Duration:** ~6 min
- **Started:** 2026-07-31T11:34:00Z
- **Completed:** 2026-07-31T11:40:00Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments

- Appended a caveat paragraph to the `UnknownRank` doc comment covering all three required points: that promoted methods panic on a nil `*UnknownRank`, that value embedding of `ValRank` makes the call convention dereference the receiver before the method body runs, and that `IsOperand` panics despite an empty body.
- Linked `[Rank]` with godoc brackets so the caveat reads as an extension of the existing interface-level nil-pointer contract (rank.go:52-55) rather than a separate warning.
- Stated that `UnknownRank` has no exported constructor and is only ever handed out as a non-nil value from internal conversion, preempting the "why not just fix it" question for the next reader.
- Kept the change strictly comment-only — the plan's mechanical gate confirmed 9 changed lines, 0 of them non-comment.

## Task Commits

1. **Task 1: Document the nil-receiver panic boundary on UnknownRank** - `3c0aa73` (docs)

## Files Created/Modified

- `pkg/api/v2/rank.go` - Added a 9-line caveat paragraph to the `UnknownRank` doc comment (lines 90-98); the struct, its `MarshalJSON`, and all `ValRank` methods are byte-identical.

## Verification

| Check | Result |
|-------|--------|
| `go build ./...` | Clean |
| `go vet ./pkg/api/v2/...` | No diagnostics |
| Comment-only diff gate | `COMMENT-ONLY DIFF CONFIRMED (9 lines)` — 9 changed, 0 non-comment |
| `git diff --stat` | `pkg/api/v2/rank.go` only, 1 file / 9 insertions |
| `make lint` | `0 issues.` |
| `go test ./pkg/api/v2/...` | `ok` (0.475s) — unchanged from pre-change |
| `go doc ...v2.UnknownRank` | Caveat renders as a separate paragraph; `[Rank]` resolves as a doc link |

## Decisions Made

- **Kept `ValRank` embedded by value.** Issue #526 evaluated switching to `*ValRank` and rejected it. Changing the embedding would alter the zero value of an exported type and every promoted method's behavior for a case that is unreachable: `UnknownRank` has no exported constructor, and its only construction site (`operandToRank`, rank.go:1304) returns `&UnknownRank{}`. Documenting the boundary is cheaper and carries no compatibility risk.
- **No nil-receiver guards or `recover()` added.** CLAUDE.md's panic-prevention rules target paths a user can actually reach through the public API. Adding defensive checks to `*ValRank` methods to protect an unconstructable type would add cost to every rank operation for zero real-world benefit, and the plan's threat register accepted T-n7b-01 on exactly that basis.
- **Mirrored the existing interface-level phrasing.** The `Rank` doc already says nil concrete-pointer `MarshalJSON` calls "are unsupported and may panic"; reusing "unsupported usage" makes both caveats one consistent policy.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. Two Bash invocations chaining `&&` with a `git diff | grep` pipeline were rejected by the worktree-isolation guard; each verification step was re-run as a separate plain command with identical results.

## Threat Flags

None. Comment-only change; no trust boundary, parsing path, or I/O surface added or modified.

## Next Phase Readiness

- Issue #526 is resolvable on the strength of this doc alone. The orchestrator closes it; `gh issue close` was deliberately not run here.
- No follow-up work implied. If a future change ever adds an exported `UnknownRank` constructor, this caveat becomes stale and the nil-guard question should be reopened.

## Self-Check: PASSED

- `pkg/api/v2/rank.go` — FOUND
- Commit `3c0aa73` — FOUND

---
*Quick task: 260731-n7b*
*Completed: 2026-07-31*
