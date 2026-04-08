# Phase 12: SDK Auto-Wiring Research - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-28
**Phase:** 12-sdk-auto-wiring-research
**Areas discussed:** Research methodology, Deliverable format, Scope of tracing, Action on differences

---

## Research Methodology

| Option | Description | Selected |
|--------|-------------|----------|
| Source reading | Read latest stable source of Python, JS, Rust SDKs on GitHub | ✓ |
| Source + docs | Read source AND official Chroma docs to cross-reference | |
| Source + live tests | Read source AND run actual SDK calls against local server | |

**User's choice:** Source reading only
**Notes:** Fast, precise, no setup required

| Option | Description | Selected |
|--------|-------------|----------|
| Latest stable | Read latest stable release tags | ✓ |
| Latest + main branch | Read stable AND tip of main | |
| Match chroma-go compat range | Read versions matching 0.6.3–1.5.5 | |

**User's choice:** Latest stable
**Notes:** Matches what most users run today

| Option | Description | Selected |
|--------|-------------|----------|
| Python + JS only | Stick to what issue #455 asks for | |
| Add Rust SDK | Also trace the Rust client | ✓ |

**User's choice:** Add Rust SDK
**Notes:** Three comparison points total plus chroma-go

---

## Deliverable Format

| Option | Description | Selected |
|--------|-------------|----------|
| Comparison matrix | Markdown table: rows=operations, columns=SDKs | ✓ |
| ADR document | Architecture Decision Record format | |
| Narrative writeup | Prose-based analysis | |

**User's choice:** Comparison matrix
**Notes:** Compact, scannable, easy to reference

| Option | Description | Selected |
|--------|-------------|----------|
| Phase directory | As 12-RESEARCH.md in .planning/phases/12-*/ | ✓ |
| Docs directory | As docs/docs/sdk-auto-wiring.md | |
| Both | Planning artifact + polished docs version | |

**User's choice:** Phase directory
**Notes:** Follows GSD conventions, stays with phase artifacts

---

## Scope of Tracing

| Option | Description | Selected |
|--------|-------------|----------|
| Config persistence | How each SDK stores/retrieves EF config | ✓ |
| Close/cleanup lifecycle | How each SDK handles EF resource cleanup | ✓ |
| Factory/registry patterns | How stored config maps back to EF instances | ✓ |
| EF-only strict scope | Only auto-wiring in get/list/create | |

**User's choice:** Full scope (all three extended areas + core auto-wiring)
**Notes:** Initially selected all four including contradictory "EF-only" — clarified to full scope

---

## Action on Differences

| Option | Description | Selected |
|--------|-------------|----------|
| Document + recommend | Document differences AND include recommendations section | ✓ |
| Document only | Strictly document, no opinions | |
| Document + file issues | Document AND create GitHub issues per change | |

**User's choice:** Document + recommend
**Notes:** Gives downstream phases clear action list without implementing

| Option | Description | Selected |
|--------|-------------|----------|
| Keep enhancements | chroma-go extras are Go-specific enhancements unless buggy | ✓ |
| Strict parity | Flag any deviation as something to fix | |
| Case by case | Evaluate each difference individually | |

**User's choice:** Keep enhancements
**Notes:** Default posture is to keep Go-specific enhancements; only flag for removal if causing bugs or confusion

---

## Claude's Discretion

- Exact structure of comparison matrix subsections
- Level of code snippet inclusion
- Whether to include summary table or per-area tables

## Deferred Ideas

None — discussion stayed within phase scope
