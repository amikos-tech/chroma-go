# Research Summary: v0.4.2 Bug Fixes and Robustness

## Executive Summary

v0.4.2 is a focused robustness milestone: 5 bugs, 1 enhancement (async embedding), 1 refactor, 1 test fix. Every bug has been traced to a code location with a confirmed root cause. No new dependencies required.

## Key Stack Findings

No new dependencies. One shared utility addition: `SanitizeErrorBody` in `pkg/commons/http/utils.go`. Critical: do not modify `MaxResponseBodySize` (200 MB transport safety cap).

## Feature Landscape

| # | Issue | Type | Complexity | Risk |
|---|-------|------|-----------|------|
| #481 | RrfRank arithmetic no-ops | Bug | Low | Low |
| #482 | WithGroupBy(nil) silent skip | Bug | Low | Low |
| #494 | ORT EF leak in CreateCollection | Bug | Medium | Medium |
| #493 | Closed EFs in GetOrCreateCollection | Bug | Medium | High |
| #478 | Error body truncation | Cleanup | Low (breadth) | Low |
| #479 | Twelve Labs async embedding | Enhancement | High | Medium |
| #412 | Download stack consolidation | Refactor | Medium | Low |
| #465 | Morph EF test 404 | Test fix | Low | Low |

## Suggested Build Order

1. #481 RrfRank — smallest, establishes test pattern
2. #482 WithGroupBy — one-liner, clean CI
3. #494 ORT EF leak — simpler lifecycle bug
4. #493 GetOrCreateCollection EF — follow-on to #494
5. #478 Error truncation — shared utility before new provider work
6. #479 Twelve Labs async — largest new surface
7. #412 Download consolidation — pure refactor, can slip
8. #465 Morph test — blocked on upstream URL

## Top Pitfalls

1. RrfRank: ErrorRank sentinel vs real expression nodes — decide before coding
2. EF lifecycle: all wrapping must go through wrapEFCloseOnce, never direct construction
3. Async polling: every HTTP request must use caller's context; time.Sleep needs select on ctx.Done
4. Error truncation at error-message sites, NOT at ReadLimitedBody transport layer
5. Download refactor must preserve package-level var injection points for tests

## Open Design Decisions

1. **RrfRank fix shape**: ErrorRank sentinel (safer) vs expression node wrapping (correct algebra)
2. **#493 fix approach**: retry-get vs hold-lock over full get-then-create
3. **Morph URL**: permanently moved or temporarily down?
