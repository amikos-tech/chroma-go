# Phase 24: GetOrCreateCollection EF Safety - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-12
**Phase:** 24-getorcreatecollection-ef-safety
**Areas discussed:** Cleanup ownership, Concurrent convergence, EF coverage breadth, Verification depth

---

## Cleanup ownership

| Option | Description | Selected |
|--------|-------------|----------|
| A | Treat caller-provided EFs as borrowed; never close them on failed `GetCollection` / revalidation cleanup. Only SDK-owned auto-wired or default EFs are closable. | ✓ |
| B | Close any EF that was installed on the failed get path, even when it came from the caller. | |
| C | Never close any EF on failed get paths, including SDK-owned temporary EFs. | |

**User's choice:** A
**Notes:** Locked as a provenance-based ownership rule: caller EFs stay borrowed until a verified handoff succeeds; only SDK-owned temporary/default/auto-wired EFs remain eligible for cleanup on failure paths.

---

## Concurrent convergence

| Option | Description | Selected |
|--------|-------------|----------|
| A-strict | Always converge to the winner snapshot when another goroutine wins the miss/create race. | |
| A-conditional | Converge to the winner snapshot when authoritative state is observable, but keep the temporary fallback EF in the already-accepted empty/no-config ambiguity branch. | ✓ |
| B | Keep the loser-local transient override EF even when another goroutine already created the collection. | |
| C | Return an explicit race/ambiguity error instead of converging. | |

**User's choice:** A-conditional
**Notes:** The user accepted convergence as the default outcome but rejected the unconditional variant because it can recreate the Phase 23 nil/unusable-EF handle bug in the empty/no-config branch.

---

## EF coverage breadth

| Option | Description | Selected |
|--------|-------------|----------|
| A | Fix dense EF only. | |
| B | Fix the shared ownership bug class across dense EF, `contentEF`, and dual-interface content EF paths. | ✓ |
| C | Fix dense EF now and defer content/dual-interface ownership parity to a later phase. | |

**User's choice:** B
**Notes:** Although the phase is named after the dense EF symptom, the user explicitly chose breadth across all embedded EF ownership paths that share the same cleanup mechanism.

---

## Verification depth

| Option | Description | Selected |
|--------|-------------|----------|
| A | One deterministic fallback failure-path regression plus one orchestrated concurrent `GetOrCreateCollection` regression intended to pass under `go test -race`. | ✓ |
| B | Those two tests plus repeated stress loops/subtests for extra timing pressure. | |
| C | Deterministic unit coverage only, without a concurrent `-race` regression. | |

**User's choice:** A
**Notes:** The user chose the smallest verification shape that still credibly satisfies `EFL-03` and matches this repo's usual narrow `basicv2` bug-fix testing style.

---

## the agent's Discretion

- Exact state/provenance representation for provisional borrowed EFs versus SDK-owned temporary EFs
- Exact implementation of the conditional convergence gate
- Exact test fixture and synchronization seam used to force the concurrent fallback path

## Deferred Ideas

- Reference counting or lease-based embedded EF lifecycle management
- Broader stress/soak race harnessing for embedded lifecycle paths
- Cross-backend EF ownership contract changes beyond this embedded fallback bug
