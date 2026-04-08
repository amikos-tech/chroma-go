# Phase 18: Embedded Client contentEmbeddingFunction Parity - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-02
**Phase:** 18-embedded-client-contentembeddingfunction-parity
**Areas discussed:** Fork stub handling, Auto-wiring scope, Close() sharing detection

---

## Fork stub handling

| Option | Description | Selected |
|--------|-------------|----------|
| Skip it (Recommended) | Fork returns error before any EF logic runs. Adding dead contentEF propagation code is noise. If Fork ever becomes supported, contentEF wiring would be part of that feature work. | ✓ |
| Add defensive code | Add contentEF close-once wrapping to the Fork stub even though it returns error. Future-proofs against someone enabling Fork without remembering contentEF. | |
| You decide | Claude picks the approach based on codebase patterns and prior decisions. | |

**User's choice:** Skip it (Recommended)
**Notes:** Fork() returns unsupported error in embedded mode — no contentEF propagation needed.

---

## Auto-wiring scope

| Option | Description | Selected |
|--------|-------------|----------|
| Full auto-wiring (Recommended) | Mirror HTTP path: call BuildContentEFFromConfig when no explicit contentEF provided. True feature parity — users get the same behavior regardless of client type. | ✓ |
| Explicit only | Only wire contentEF when WithContentEmbeddingFunctionGet is passed. Simpler, but creates a behavioral gap between HTTP and embedded clients. | |
| You decide | Claude picks the approach based on what makes sense for the embedded path. | |

**User's choice:** Full auto-wiring (Recommended)
**Notes:** None

---

## Close() sharing detection

| Option | Description | Selected |
|--------|-------------|----------|
| Full mirror (Recommended) | Same unwrapper + identity check as HTTP. With auto-wiring enabled, the adapter case (contentEF wrapping denseEF) can absolutely happen. Prevents double-close bugs. | ✓ |
| Simplified | Only close contentEF, skip denseEF entirely when contentEF is present. Simpler but may leak denseEF resources when they are independent. | |
| You decide | Claude picks based on the auto-wiring decision above. | |

**User's choice:** Full mirror (Recommended)
**Notes:** None

## Claude's Discretion

- embeddedCollectionState struct field naming and initialization details
- Test structure and file organization
- Whether to add contentEmbeddingFunction to CreateCollectionOp or keep it GetCollection-only

## Deferred Ideas

None — discussion stayed within phase scope
